package agents

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/signal"
	"github.com/rhl/businessos-backend/internal/streaming"
)

// AgentType identifies the type of agent in the new architecture
type AgentType string

const (
	AgentTypeOrchestrator AgentType = "orchestrator"
	AgentTypeDocument     AgentType = "document"
	AgentTypeProject      AgentType = "project"
	AgentTypeTask         AgentType = "task"
	AgentTypeClient       AgentType = "client"
	AgentTypeAnalyst      AgentType = "analyst"
)

// Agent defines the interface for the new agent architecture
type Agent interface {
	// Identity
	Type() AgentType
	Name() string
	Description() string

	// Configuration
	GetSystemPrompt() string
	GetContextRequirements() ContextRequirements

	// Execution - returns streaming events instead of raw strings
	Run(ctx context.Context, input AgentInput) (<-chan streaming.StreamEvent, <-chan error)
	RunWithTools(ctx context.Context, input AgentInput) (<-chan streaming.StreamEvent, <-chan error)

	// Options
	SetModel(model string)
	SetOptions(opts services.LLMOptions)
	SetCustomSystemPrompt(prompt string)          // For custom agents with user-defined prompts
	SetFocusModePrompt(prompt string)             // For focus mode specific prompt prefix
	SetOutputStylePrompt(prompt string)           // For output style specific instructions
	SetRoleContextPrompt(prompt string)           // For role-based permission context (Feature 1)
	SetMemoryContext(context string)              // For workspace memory injection (Feature: Memory Hierarchy)
	SetSkillsPrompt(prompt string)                // For agent skills system context injection
	SetProfileContext(context string)             // For user onboarding profile context (for personalization)
	SetGenreContext(context string)               // For genre-aware composition (Signal Theory L3)
	SetTieredContext(ctx *services.TieredContext) // For authoritative TieredContext in system prompt (L4)
}

// ContextRequirements declares what context an agent needs
type ContextRequirements struct {
	NeedsProjects    bool
	NeedsTasks       bool
	NeedsClients     bool
	NeedsTeam        bool
	NeedsKnowledge   bool
	NeedsMetrics     bool
	NeedsFullHistory bool
	MaxContextTokens int
	PrioritySections []string
}

// AgentInput contains everything an agent needs to process a request
type AgentInput struct {
	Messages       []services.ChatMessage
	Context        *services.TieredContext
	Selections     UserSelections
	FocusMode      string
	FocusModeOpts  map[string]string
	ConversationID uuid.UUID
	UserID         string
	UserName       string
	MemoryContext  string                 // Workspace memory context to inject into agents
	RoleContext    string                 // Role-based context to inject into agents
	SignalEnvelope *signal.SignalEnvelope // L1 fast-path classification result
}

// UserSelections represents what the user selected in the context bar
type UserSelections struct {
	ProjectID  *uuid.UUID
	ContextIDs []uuid.UUID
	NodeID     *uuid.UUID
	ClientID   *uuid.UUID
}

// AgentContext holds context information for agent initialization
type AgentContext struct {
	Pool               *pgxpool.Pool
	Config             *config.Config
	UserID             string
	UserName           string
	ConversationID     *uuid.UUID
	TieredContext      *services.TieredContext
	EmbeddingService   *services.EmbeddingService
	PromptPersonalizer *services.PromptPersonalizer
	SignalHints        feedback.SignalHintProvider // Homeostatic feedback → prompt corrections
}

// Intent represents the classified intent of a user message
type Intent struct {
	Category       string // "document", "project", "client", "analysis", "general"
	ShouldDelegate bool
	TargetAgent    AgentType
	Confidence     float64
	Reasoning      string
}

// AgentRegistry manages agent creation and retrieval for the new architecture
type AgentRegistry struct {
	pool               *pgxpool.Pool
	config             *config.Config
	embeddingService   *services.EmbeddingService
	promptPersonalizer *services.PromptPersonalizer
	signalHints        feedback.SignalHintProvider // Homeostatic feedback → prompt corrections
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry(
	pool *pgxpool.Pool,
	cfg *config.Config,
	embeddingService *services.EmbeddingService,
	promptPersonalizer *services.PromptPersonalizer,
	signalHints feedback.SignalHintProvider,
) *AgentRegistry {
	return &AgentRegistry{
		pool:               pool,
		config:             cfg,
		embeddingService:   embeddingService,
		promptPersonalizer: promptPersonalizer,
		signalHints:        signalHints,
	}
}

// GetAgent creates an agent of the specified type
func (r *AgentRegistry) GetAgent(
	agentType AgentType,
	userID string,
	userName string,
	conversationID *uuid.UUID,
	tieredContext *services.TieredContext,
) Agent {
	ctx := &AgentContext{
		Pool:               r.pool,
		Config:             r.config,
		UserID:             userID,
		UserName:           userName,
		ConversationID:     conversationID,
		TieredContext:      tieredContext,
		EmbeddingService:   r.embeddingService,
		PromptPersonalizer: r.promptPersonalizer,
		SignalHints:        r.signalHints,
	}

	switch agentType {
	case AgentTypeOrchestrator:
		return NewOrchestrator(ctx)
	case AgentTypeDocument:
		return NewDocumentAgent(ctx)
	case AgentTypeProject:
		return NewProjectAgent(ctx)
	case AgentTypeTask:
		return NewTaskAgent(ctx)
	case AgentTypeClient:
		return NewClientAgent(ctx)
	case AgentTypeAnalyst:
		return NewAnalystAgent(ctx)
	default:
		return NewOrchestrator(ctx)
	}
}

// GetAgentForFocusMode maps focus modes to appropriate agents
func GetAgentForFocusMode(focusMode string) AgentType {
	switch focusMode {
	case "write":
		return AgentTypeDocument
	case "analyze", "research":
		return AgentTypeAnalyst
	case "plan", "build":
		return AgentTypeProject
	default:
		return AgentTypeOrchestrator
	}
}

// AgentTypeFromString converts a string to AgentType
func AgentTypeFromString(s string) AgentType {
	switch s {
	case "orchestrator":
		return AgentTypeOrchestrator
	case "document":
		return AgentTypeDocument
	case "project":
		return AgentTypeProject
	case "task":
		return AgentTypeTask
	case "client":
		return AgentTypeClient
	case "analyst", "analysis":
		return AgentTypeAnalyst
	default:
		return AgentTypeOrchestrator
	}
}
