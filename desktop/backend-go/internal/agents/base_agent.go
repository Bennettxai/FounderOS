package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/tools"
)

// Sentinel errors for agent operations
var (
	// ErrToolRegistryNotAvailable indicates that the tool registry is not initialized
	ErrToolRegistryNotAvailable = errors.New("tool registry not available")
	// ErrToolNotEnabled indicates that a tool is not enabled for the agent
	ErrToolNotEnabled = errors.New("tool not enabled for this agent")
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	pool               *pgxpool.Pool
	cfg                *config.Config
	userID             string
	userName           string
	conversationID     *uuid.UUID
	model              string
	agentType          AgentType
	agentName          string
	description        string
	systemPrompt       string
	focusModePrompt    string                  // Focus mode specific prompt prefix
	outputStylePrompt  string                  // Output style specific instructions
	roleContextPrompt  string                  // Role-based permissions context (Feature 1)
	memoryContext      string                  // Workspace memory context (Feature: Memory Hierarchy)
	skillsPrompt       string                  // Available skills context (Agent Skills System)
	profileContext     string                  // User onboarding profile context (for personalization)
	genreContext       string                  // Genre-aware composition context (Signal Theory L3)
	tieredContext      *services.TieredContext // Authoritative TieredContext for system prompt (L4)
	contextReqs        ContextRequirements
	llmOptions         services.LLMOptions
	toolRegistry       *tools.AgentToolRegistry
	enabledTools       []string                     // Tool names this agent can use
	promptPersonalizer *services.PromptPersonalizer // For personalizing prompts with user data
	lastUserMessage    string                       // Last user message (for semantic personalization)
	signalHints        feedback.SignalHintProvider  // Homeostatic feedback → prompt corrections (Signal Theory)
}

// BaseAgentConfig holds configuration for creating a BaseAgent
type BaseAgentConfig struct {
	Pool               *pgxpool.Pool
	Config             *config.Config
	UserID             string
	UserName           string
	ConversationID     *uuid.UUID
	Model              string
	AgentType          AgentType
	AgentName          string
	Description        string
	SystemPrompt       string
	ContextReqs        ContextRequirements
	EnabledTools       []string                     // Tool names this agent can use
	EmbeddingService   *services.EmbeddingService   // For context tools (tree_search, browse_tree, load_context)
	PromptPersonalizer *services.PromptPersonalizer // For personalizing prompts with user data
	SignalHints        feedback.SignalHintProvider  // Homeostatic feedback → prompt corrections
}

// NewBaseAgent creates a new base agent with the given configuration
func NewBaseAgent(cfg BaseAgentConfig) *BaseAgent {
	model := cfg.Model
	if model == "" && cfg.Config != nil {
		model = cfg.Config.GetActiveModel()
	}

	// Create tool registry if pool is available.
	// When an EmbeddingService is provided we build a ContextService and use the
	// delegating registry so tree_search, browse_tree, and load_context are enabled
	// and benefit from voice-note and conversation-summary search automatically.
	var toolRegistry *tools.AgentToolRegistry
	if cfg.Pool != nil && cfg.UserID != "" {
		if cfg.EmbeddingService != nil {
			contextSvc := services.NewContextService(cfg.Pool, cfg.EmbeddingService)
			toolRegistry = tools.NewAgentToolRegistryWithContext(cfg.Pool, cfg.UserID, contextSvc)
		} else {
			toolRegistry = tools.NewAgentToolRegistry(cfg.Pool, cfg.UserID)
		}
		// Load user's MCP server tools into the agent registry
		toolRegistry.RegisterMCPTools(context.Background())
	}

	return &BaseAgent{
		pool:               cfg.Pool,
		cfg:                cfg.Config,
		userID:             cfg.UserID,
		userName:           cfg.UserName,
		conversationID:     cfg.ConversationID,
		model:              model,
		agentType:          cfg.AgentType,
		agentName:          cfg.AgentName,
		description:        cfg.Description,
		systemPrompt:       cfg.SystemPrompt,
		contextReqs:        cfg.ContextReqs,
		llmOptions:         services.DefaultLLMOptions(),
		toolRegistry:       toolRegistry,
		enabledTools:       cfg.EnabledTools,
		promptPersonalizer: cfg.PromptPersonalizer,
		signalHints:        cfg.SignalHints,
	}
}

// Type returns the agent type
func (a *BaseAgent) Type() AgentType {
	return a.agentType
}

// Name returns the agent name
func (a *BaseAgent) Name() string {
	return a.agentName
}

// Description returns the agent description
func (a *BaseAgent) Description() string {
	return a.description
}

// GetSystemPrompt returns the system prompt
func (a *BaseAgent) GetSystemPrompt() string {
	return a.systemPrompt
}

// GetContextRequirements returns what context the agent needs
func (a *BaseAgent) GetContextRequirements() ContextRequirements {
	return a.contextReqs
}

// SetModel sets the model to use
func (a *BaseAgent) SetModel(model string) {
	a.model = model
}

// SetOptions sets the LLM options
func (a *BaseAgent) SetOptions(opts services.LLMOptions) {
	a.llmOptions = opts
}

// GetOptions returns the current LLM options
func (a *BaseAgent) GetOptions() services.LLMOptions {
	return a.llmOptions
}

// SetCustomSystemPrompt overrides the system prompt with a custom one (for custom agents)
func (a *BaseAgent) SetCustomSystemPrompt(prompt string) {
	slog.Debug("agent SetCustomSystemPrompt called", "prompt_len", len(prompt))
	if prompt != "" {
		a.systemPrompt = prompt
		slog.Debug("agent custom systemPrompt set", "len", len(a.systemPrompt))
	}
}

// SetFocusModePrompt sets a focus mode specific prompt prefix
func (a *BaseAgent) SetFocusModePrompt(prompt string) {
	a.focusModePrompt = prompt
}

// SetOutputStylePrompt sets an output style specific prompt section
func (a *BaseAgent) SetOutputStylePrompt(prompt string) {
	a.outputStylePrompt = prompt
}

// SetRoleContextPrompt sets role-based permission context (Feature 1)
func (a *BaseAgent) SetRoleContextPrompt(prompt string) {
	a.roleContextPrompt = prompt
	slog.Default().Debug("[Agent] SetRoleContextPrompt called", "chars", len(prompt))
}

// SetMemoryContext sets workspace memory context (Feature: Memory Hierarchy)
func (a *BaseAgent) SetMemoryContext(context string) {
	a.memoryContext = context
}

// SetSkillsPrompt sets the available skills context (Agent Skills System)
func (a *BaseAgent) SetSkillsPrompt(prompt string) {
	a.skillsPrompt = prompt
}

// SetProfileContext sets the user profile context for personalization
func (a *BaseAgent) SetProfileContext(context string) {
	a.profileContext = context
	slog.Default().Debug("[Agent] SetProfileContext called", "chars", len(context))
}

// SetGenreContext sets the genre-aware composition context (Signal Theory L3)
func (a *BaseAgent) SetGenreContext(context string) {
	a.genreContext = context
}

// SetTieredContext sets the authoritative TieredContext for system prompt injection (L4)
func (a *BaseAgent) SetTieredContext(ctx *services.TieredContext) {
	a.tieredContext = ctx
}

// SetLastUserMessage stores the last user message for personalization
func (a *BaseAgent) SetLastUserMessage(message string) {
	a.lastUserMessage = message
}

// RegisterExternalTool adds a tool to the registry and enables it for this agent.
// Used for tools that require dependencies injected at runtime (e.g., OSA client).
func (a *BaseAgent) RegisterExternalTool(tool tools.AgentTool) {
	if a.toolRegistry != nil {
		a.toolRegistry.RegisterTool(tool)
		a.enabledTools = append(a.enabledTools, tool.Name())
	}
}

// GetEnabledTools returns the list of tools this agent can use
func (a *BaseAgent) GetEnabledTools() []string {
	return a.enabledTools
}

// GetToolDefinitions returns tool definitions for LLM function calling
func (a *BaseAgent) GetToolDefinitions() []map[string]interface{} {
	if a.toolRegistry == nil || len(a.enabledTools) == 0 {
		return nil
	}

	defs := make([]map[string]interface{}, 0)
	for _, toolName := range a.enabledTools {
		if tool, ok := a.toolRegistry.GetTool(toolName); ok {
			defs = append(defs, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name(),
					"description": tool.Description(),
					"parameters":  tool.InputSchema(),
				},
			})
		}
	}
	return defs
}

// ExecuteTool executes a tool by name with the given input
func (a *BaseAgent) ExecuteTool(ctx context.Context, toolName string, input json.RawMessage) (string, error) {
	if a.toolRegistry == nil {
		return "", ErrToolRegistryNotAvailable
	}

	// Check if tool is enabled for this agent
	enabled := false
	for _, t := range a.enabledTools {
		if t == toolName {
			enabled = true
			break
		}
	}
	if !enabled {
		return "", fmt.Errorf("%w: %s", ErrToolNotEnabled, toolName)
	}

	return a.toolRegistry.ExecuteTool(ctx, toolName, input)
}

// Pool returns the database pool
func (a *BaseAgent) Pool() *pgxpool.Pool {
	return a.pool
}

// Config returns the configuration
func (a *BaseAgent) Config() *config.Config {
	return a.cfg
}

// UserID returns the user ID
func (a *BaseAgent) UserID() string {
	return a.userID
}

// UserName returns the user name
func (a *BaseAgent) UserName() string {
	return a.userName
}

// ConversationID returns the conversation ID
func (a *BaseAgent) ConversationID() *uuid.UUID {
	return a.conversationID
}

// Model returns the current model
func (a *BaseAgent) Model() string {
	return a.model
}
