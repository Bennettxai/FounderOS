// Package sorx provides agent integration for the skill execution engine.
package sorx

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/carrier"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// AgentBridge connects Sorx skills to BusinessOS agents.
// This allows skills to invoke AI agents for reasoning, analysis, and decision-making.
type AgentBridge struct {
	pool     *pgxpool.Pool
	config   *config.Config
	registry *agents.AgentRegistry
}

// NewAgentBridge creates a new agent bridge.
func NewAgentBridge(pool *pgxpool.Pool, cfg *config.Config, embeddingService *services.EmbeddingService, promptPersonalizer *services.PromptPersonalizer) *AgentBridge {
	return &AgentBridge{
		pool:     pool,
		config:   cfg,
		registry: agents.NewAgentRegistry(pool, cfg, embeddingService, nil, nil),
	}
}

// AgentInvocation represents a request to invoke an agent from a skill.
type AgentInvocation struct {
	AgentType      string                 `json:"agent_type"`      // orchestrator, analyst, document, task, project, client
	Task           string                 `json:"task"`            // The task/prompt for the agent
	Context        map[string]interface{} `json:"context"`         // Additional context
	UserID         string                 `json:"user_id"`         // User context
	UserName       string                 `json:"user_name"`       // User name for personalization
	ConversationID *uuid.UUID             `json:"conversation_id"` // Optional conversation context
	WaitForResult  bool                   `json:"wait_for_result"` // Whether to wait for the full result
	Tier           SkillTier              `json:"tier,omitempty"`  // Skill tier for CARRIER routing (0 = unset, uses local)
}

// AgentResult holds the result of an agent invocation.
type AgentResult struct {
	Success   bool                   `json:"success"`
	Output    string                 `json:"output"`
	AgentType string                 `json:"agent_type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// InvokeAgent invokes a BusinessOS agent and returns the result.
//
// For Tier 3-4 invocations, it first attempts to route through CARRIER to
// SorxMain. If CARRIER is unavailable or signals a fallback condition, it
// degrades to the local AgentRegistry transparently.
func (b *AgentBridge) InvokeAgent(ctx context.Context, inv AgentInvocation) (*AgentResult, error) {
	// Tier 3-4: attempt CARRIER routing before local agent.
	if inv.Tier >= TierReasoningAI {
		if result, ok := b.tryCarrierAgent(ctx, inv); ok {
			return result, nil
		}
	}

	// Map agent type string to AgentType
	agentType := agents.AgentTypeFromString(inv.AgentType)

	// Get the agent from registry
	agent := b.registry.GetAgent(agentType, inv.UserID, inv.UserName, inv.ConversationID, nil)

	// Build input messages
	messages := []services.ChatMessage{
		{
			Role:    "user",
			Content: inv.Task,
		},
	}

	// Add context as a system message if provided
	if len(inv.Context) > 0 {
		contextStr := formatContextForAgent(inv.Context)
		if contextStr != "" {
			messages = append([]services.ChatMessage{
				{
					Role:    "system",
					Content: contextStr,
				},
			}, messages...)
		}
	}

	// Create agent input
	input := agents.AgentInput{
		Messages:  messages,
		UserID:    inv.UserID,
		UserName:  inv.UserName,
		FocusMode: "general",
	}
	if inv.ConversationID != nil {
		input.ConversationID = *inv.ConversationID
	}

	// Run the agent
	events, errs := agent.Run(ctx, input)

	// Collect the output
	var output strings.Builder
	result := &AgentResult{
		Success:   true,
		AgentType: string(agentType),
		Metadata:  make(map[string]interface{}),
	}

	for {
		select {
		case event, ok := <-events:
			if !ok {
				result.Output = output.String()
				return result, nil
			}
			if event.Type == streaming.EventTypeToken {
				if content, ok := event.Data.(string); ok {
					output.WriteString(content)
				}
			}
		case err := <-errs:
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				result.Output = output.String()
				return result, err
			}
		case <-ctx.Done():
			result.Success = false
			result.Error = "context cancelled"
			result.Output = output.String()
			return result, ctx.Err()
		}
	}
}

// tryCarrierAgent attempts to route an agent invocation through CARRIER.
// It returns (result, true) on success, or (nil, false) when CARRIER is
// unavailable or signals a fallback — in which case the caller should fall
// through to the local agent.
func (b *AgentBridge) tryCarrierAgent(ctx context.Context, inv AgentInvocation) (*AgentResult, bool) {
	c := getEngineCarrier()
	if c == nil || !c.IsConnected() {
		return nil, false
	}

	params := map[string]any{
		"agent_type": inv.AgentType,
		"task":       inv.Task,
		"tier":       int(inv.Tier),
		"model":      TierToModel(inv.Tier),
	}
	if len(inv.Context) > 0 {
		params["context"] = formatContextForCarrier(inv.Context)
	}
	if inv.UserName != "" {
		params["user_name"] = inv.UserName
	}

	resp, err := c.Send(ctx, carrier.Request{
		Method:     "deliberate",
		RoutingKey: tierToRoutingKey(inv.Tier, "agent_invoke"),
		Priority:   tierToPriority(inv.Tier),
		Params:     params,
		Context:    buildMessageContext(inv),
	})

	if err != nil {
		if !carrier.IsFallback(err) {
			// Hard SorxMain error — still fall through so local agent can
			// attempt to serve the request rather than hard-failing the skill.
			b.logger(ctx).Warn("carrier agent invocation failed with hard error; falling back to local agent",
				"agent_type", inv.AgentType,
				"tier", inv.Tier,
				"error", err,
			)
		}
		return nil, false
	}

	output, extractErr := extractCarrierResult(resp)
	if extractErr != nil {
		b.logger(ctx).Warn("carrier agent result extraction failed; falling back to local agent",
			"agent_type", inv.AgentType,
			"error", extractErr,
		)
		return nil, false
	}

	return &AgentResult{
		Success:   true,
		Output:    output,
		AgentType: inv.AgentType,
		Metadata: map[string]interface{}{
			"routed_via":  "carrier",
			"duration_ms": resp.DurationMS,
		},
	}, true
}

// logger returns a *slog.Logger scoped to the bridge. The ctx parameter is
// reserved for future trace-ID injection.
func (b *AgentBridge) logger(_ context.Context) *slog.Logger {
	return slog.Default().With("component", "agent_bridge")
}

// buildMessageContext constructs a carrier.MessageContext from an AgentInvocation.
// Fields are populated from inv.UserID, inv.Task, inv.ConversationID, and
// well-known keys within inv.Context. All type assertions use ok-checks so
// missing or mis-typed fields are silently skipped — no panics.
func buildMessageContext(inv AgentInvocation) carrier.MessageContext {
	mc := carrier.MessageContext{
		UserID:      inv.UserID,
		UserMessage: inv.Task,
	}

	if inv.ConversationID != nil {
		mc.ConversationID = inv.ConversationID.String()
	}

	if len(inv.Context) == 0 {
		return mc
	}

	if v, ok := inv.Context["workspace_id"].(string); ok {
		mc.WorkspaceID = v
	}
	if v, ok := inv.Context["mode"].(string); ok {
		mc.Mode = v
	}
	if v, ok := inv.Context["temperature"].(string); ok {
		mc.Temperature = v
	}
	if v, ok := inv.Context["user_role"].(string); ok {
		mc.UserRole = v
	}
	if v, ok := inv.Context["selected_project_id"].(string); ok {
		mc.SelectedProjectID = v
	}

	// InstalledModules: accept []string or []interface{}.
	switch v := inv.Context["installed_modules"].(type) {
	case []string:
		mc.InstalledModules = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				mc.InstalledModules = append(mc.InstalledModules, s)
			}
		}
	}

	// ConnectedIntegrations: accept []string or []interface{}.
	switch v := inv.Context["connected_integrations"].(type) {
	case []string:
		mc.ConnectedIntegrations = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				mc.ConnectedIntegrations = append(mc.ConnectedIntegrations, s)
			}
		}
	}

	// ConversationHistory: accept []carrier.ConversationMessage or []interface{}
	// (the latter typically comes from JSON-decoded map values).
	switch v := inv.Context["conversation_history"].(type) {
	case []carrier.ConversationMessage:
		mc.ConversationHistory = v
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				if role != "" && content != "" {
					mc.ConversationHistory = append(mc.ConversationHistory,
						carrier.ConversationMessage{Role: role, Content: content})
				}
			}
		}
	}

	return mc
}

// formatContextForCarrier converts skill context to a compact string suitable
// for inclusion in a CARRIER request param.
func formatContextForCarrier(skillCtx map[string]interface{}) string {
	if len(skillCtx) == 0 {
		return ""
	}
	return formatContextForAgent(skillCtx)
}

// formatContextForAgent formats skill context for agent consumption.
func formatContextForAgent(ctx map[string]interface{}) string {
	if len(ctx) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Skill Context\n\n")

	for key, value := range ctx {
		switch v := value.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("**%s:** %s\n", key, v))
		case []interface{}:
			sb.WriteString(fmt.Sprintf("**%s:**\n", key))
			for _, item := range v {
				sb.WriteString(fmt.Sprintf("  - %v\n", item))
			}
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("**%s:**\n", key))
			for k, val := range v {
				sb.WriteString(fmt.Sprintf("  - %s: %v\n", k, val))
			}
		default:
			sb.WriteString(fmt.Sprintf("**%s:** %v\n", key, v))
		}
	}

	return sb.String()
}

// RegisterAgentActions registers agent-related action handlers in the Sorx engine.
func RegisterAgentActions(bridge *AgentBridge) {
	// Agent invocation actions
	RegisterAction("agent.orchestrator", createAgentAction(bridge, "orchestrator"))
	RegisterAction("agent.analyst", createAgentAction(bridge, "analyst"))
	RegisterAction("agent.document", createAgentAction(bridge, "document"))
	RegisterAction("agent.task", createAgentAction(bridge, "task"))
	RegisterAction("agent.project", createAgentAction(bridge, "project"))
	RegisterAction("agent.client", createAgentAction(bridge, "client"))

	// Generic agent invocation
	RegisterAction("agent.invoke", func(ctx context.Context, ac ActionContext) (interface{}, error) {
		agentType, _ := ac.Params["agent_type"].(string)
		if agentType == "" {
			agentType = "orchestrator"
		}

		task, _ := ac.Params["task"].(string)
		if task == "" {
			return nil, fmt.Errorf("task is required for agent invocation")
		}

		result, err := bridge.InvokeAgent(ctx, AgentInvocation{
			AgentType:     agentType,
			Task:          task,
			Context:       ac.Execution.Context,
			UserID:        ac.Execution.UserID,
			WaitForResult: true,
			Tier:          skillTierFromContext(ac),
		})
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success":    result.Success,
			"output":     result.Output,
			"agent_type": result.AgentType,
			"metadata":   result.Metadata,
		}, nil
	})
}

// createAgentAction creates an action handler for a specific agent type.
func createAgentAction(bridge *AgentBridge, agentType string) ActionHandler {
	return func(ctx context.Context, ac ActionContext) (interface{}, error) {
		task, _ := ac.Params["task"].(string)
		if task == "" {
			// Try to get task from step results or context
			if fromStep, ok := ac.Params["from"].(string); ok {
				if stepResult, ok := ac.Execution.StepResults[fromStep]; ok {
					task = fmt.Sprintf("Process this data and provide analysis:\n%v", stepResult)
				}
			}
		}

		if task == "" {
			return nil, fmt.Errorf("task is required for %s agent", agentType)
		}

		result, err := bridge.InvokeAgent(ctx, AgentInvocation{
			AgentType:     agentType,
			Task:          task,
			Context:       ac.Execution.Context,
			UserID:        ac.Execution.UserID,
			WaitForResult: true,
			Tier:          skillTierFromContext(ac),
		})
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success":    result.Success,
			"output":     result.Output,
			"agent_type": result.AgentType,
		}, nil
	}
}

// SkillTrigger represents a request to trigger a skill from an agent or command.
type SkillTrigger struct {
	SkillID string                 `json:"skill_id"`
	UserID  string                 `json:"user_id"`
	Params  map[string]interface{} `json:"params"`
	Async   bool                   `json:"async"` // If true, return immediately with execution ID
}

// SkillTriggerResult holds the result of triggering a skill.
type SkillTriggerResult struct {
	ExecutionID uuid.UUID   `json:"execution_id"`
	Status      string      `json:"status"`
	Result      interface{} `json:"result,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// TriggerSkill allows agents and commands to trigger Sorx skills.
// This is used when an agent determines that a skill should be executed.
func TriggerSkill(ctx context.Context, engine *Engine, trigger SkillTrigger) (*SkillTriggerResult, error) {
	exec, err := engine.ExecuteSkill(ctx, ExecuteRequest{
		SkillID: trigger.SkillID,
		UserID:  trigger.UserID,
		Params:  trigger.Params,
	})
	if err != nil {
		return &SkillTriggerResult{
			Status: StatusFailed,
			Error:  err.Error(),
		}, err
	}

	result := &SkillTriggerResult{
		ExecutionID: exec.ID,
		Status:      exec.Status,
	}

	// If not async, wait for completion
	if !trigger.Async {
		// Poll for completion (simplified - in production use channels)
		for exec.Status == StatusPending || exec.Status == StatusRunning {
			updatedExec, ok := engine.GetExecution(exec.ID)
			if !ok {
				break
			}
			exec = updatedExec
			if exec.Status == StatusComplete || exec.Status == StatusFailed {
				break
			}
		}
		result.Status = exec.Status
		result.Result = exec.Result
		if exec.Error != "" {
			result.Error = exec.Error
		}
	}

	return result, nil
}
