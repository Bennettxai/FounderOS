package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OSAOrchestrator defines the minimal interface needed for OSA escalation.
// Both osa.Client and osa.ResilientClient satisfy this.
type OSAOrchestrator interface {
	Orchestrate(ctx context.Context, req *osa.OrchestrateRequest) (*osa.OrchestrateResponse, error)
}

// EscalateToOSATool triggers async OSA orchestration for complex tasks
// that require multi-agent reasoning, app generation, or deep analysis.
// The Orchestrator LLM calls this tool when it determines the request
// exceeds what the reactive agent layer can handle directly.
type EscalateToOSATool struct {
	osaClient   OSAOrchestrator
	userID      string
	workspaceID *uuid.UUID
}

// NewEscalateToOSATool creates a new escalation tool.
// osaClient can be nil — Execute will return a message explaining OSA is unavailable.
func NewEscalateToOSATool(osaClient OSAOrchestrator, userID string, workspaceID *uuid.UUID) *EscalateToOSATool {
	return &EscalateToOSATool{
		osaClient:   osaClient,
		userID:      userID,
		workspaceID: workspaceID,
	}
}

func (t *EscalateToOSATool) Name() string { return "escalate_to_osa" }

func (t *EscalateToOSATool) Description() string {
	return "Escalate a complex task to the OSA orchestration engine for multi-agent reasoning, " +
		"app generation, strategic planning with MCTS, or deep analysis that requires the full " +
		"21-agent swarm. Use this when the task requires building software, generating applications, " +
		"multi-step autonomous execution, or reasoning that exceeds standard chat capabilities. " +
		"OSA processes asynchronously and can call back into BOS reactive agents as needed."
}

func (t *EscalateToOSATool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_description": map[string]interface{}{
				"type":        "string",
				"description": "Clear description of what OSA should accomplish. Be specific about the goal, constraints, and expected output.",
			},
			"phase": map[string]interface{}{
				"type":        "string",
				"description": "Optional execution phase hint: 'analysis', 'strategy', 'development', 'review'",
				"enum":        []string{"analysis", "strategy", "development", "review"},
			},
		},
		"required": []string{"task_description"},
	}
}

func (t *EscalateToOSATool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.osaClient == nil {
		return "OSA orchestration engine is not currently available. I'll handle this request " +
			"directly using the tools I have. If this requires capabilities beyond what I can " +
			"provide, let me know and we can revisit when OSA is online.", nil
	}

	var params struct {
		TaskDescription string `json:"task_description"`
		Phase           string `json:"phase,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid parameters: %w", err)
	}

	userUUID, err := uuid.Parse(t.userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	sessionID := uuid.NewString()

	osaReq := &osa.OrchestrateRequest{
		UserID:    userUUID,
		Input:     params.TaskDescription,
		SessionID: sessionID,
		Phase:     params.Phase,
	}
	if t.workspaceID != nil {
		osaReq.WorkspaceID = *t.workspaceID
	}

	// Fire-and-forget: OSA processes asynchronously. Results come back
	// via CARRIER proactive_consumer (execute_action, request_decision,
	// proactive_signal) or through the OSA event stream.
	go func() {
		bgCtx := context.Background()
		resp, err := t.osaClient.Orchestrate(bgCtx, osaReq)
		if err != nil {
			slog.Error("OSA escalation failed", "error", err, "session_id", sessionID)
			return
		}
		slog.Info("OSA escalation accepted",
			"session_id", sessionID,
			"agents_used", resp.AgentsUsed,
			"execution_ms", resp.ExecutionTime,
		)
	}()

	slog.Info("escalated to OSA",
		"session_id", sessionID,
		"user_id", t.userID,
		"phase", params.Phase,
		"task_len", len(params.TaskDescription),
	)

	return fmt.Sprintf("Task escalated to OSA orchestration engine (session: %s). "+
		"OSA's 21-agent swarm is processing this asynchronously. I'll continue working "+
		"on what I can handle directly. OSA will callback with results via the proactive "+
		"signal channel when ready.", sessionID), nil
}
