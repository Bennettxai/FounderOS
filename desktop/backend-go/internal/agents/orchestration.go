package agents

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// ========== CHAIN OF THOUGHT (COT) SYSTEM ==========

// ThoughtStep represents a single step in the chain of thought
type ThoughtStep struct {
	ID          string        `json:"id"`
	Agent       AgentType     `json:"agent"`
	Action      string        `json:"action"`     // "analyze", "delegate", "execute", "synthesize"
	Input       string        `json:"input"`      // What this step received
	Output      string        `json:"output"`     // What this step produced
	Reasoning   string        `json:"reasoning"`  // Why this step was taken
	Confidence  float64       `json:"confidence"` // 0-1 confidence in this step
	Duration    time.Duration `json:"duration"`   // How long this step took
	Children    []string      `json:"children"`   // IDs of child steps (for parallel execution)
	Status      string        `json:"status"`     // "pending", "running", "completed", "failed"
	Error       string        `json:"error,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
}

// ChainOfThought tracks the full reasoning chain
type ChainOfThought struct {
	ID             string         `json:"id"`
	UserMessage    string         `json:"user_message"`
	Steps          []*ThoughtStep `json:"steps"`
	FinalOutput    string         `json:"final_output"`
	TotalDuration  time.Duration  `json:"total_duration"`
	AgentsInvolved []AgentType    `json:"agents_involved"`
	Status         string         `json:"status"` // "planning", "executing", "synthesizing", "completed"
	mu             sync.RWMutex
}

// NewChainOfThought creates a new COT tracker
func NewChainOfThought(userMessage string) *ChainOfThought {
	return &ChainOfThought{
		ID:          uuid.New().String(),
		UserMessage: userMessage,
		Steps:       make([]*ThoughtStep, 0),
		Status:      "planning",
	}
}

// ========== AGENT MESSAGE SYSTEM ==========

// AgentMessage represents a message passed between agents
type AgentMessage struct {
	ID        string            `json:"id"`
	From      AgentType         `json:"from"`
	To        AgentType         `json:"to"`
	Type      AgentMessageType  `json:"type"`
	Content   string            `json:"content"`
	Context   map[string]string `json:"context,omitempty"`
	RequestID string            `json:"request_id"` // Links related messages
	InReplyTo string            `json:"in_reply_to,omitempty"`
	Priority  int               `json:"priority"` // 1=low, 2=normal, 3=high, 4=urgent
	CreatedAt time.Time         `json:"created_at"`
}

// AgentMessageType defines the type of inter-agent message
type AgentMessageType string

const (
	MsgTypeRequest    AgentMessageType = "request"    // Asking another agent to do something
	MsgTypeResponse   AgentMessageType = "response"   // Response to a request
	MsgTypeDelegate   AgentMessageType = "delegate"   // Delegating a task
	MsgTypeInform     AgentMessageType = "inform"     // Sharing information
	MsgTypeQuery      AgentMessageType = "query"      // Asking for information
	MsgTypeConfirm    AgentMessageType = "confirm"    // Confirming receipt/action
	MsgTypeSynthesize AgentMessageType = "synthesize" // Request to combine outputs
)

// NewAgentMessage creates a new inter-agent message
func NewAgentMessage(from, to AgentType, msgType AgentMessageType, content string) *AgentMessage {
	return &AgentMessage{
		ID:        uuid.New().String(),
		From:      from,
		To:        to,
		Type:      msgType,
		Content:   content,
		Context:   make(map[string]string),
		Priority:  2, // Normal priority
		CreatedAt: time.Now(),
	}
}

// ========== ORCHESTRATOR COT ENGINE ==========

// OrchestratorCOT manages the chain of thought execution
type OrchestratorCOT struct {
	pool     *pgxpool.Pool
	config   *config.Config
	registry *AgentRegistry
	router   *SmartIntentRouter

	// Message bus for inter-agent communication
	messageBus chan *AgentMessage

	// Active chains
	activeChains map[string]*ChainOfThought
	chainsMu     sync.RWMutex
}

// NewOrchestratorCOT creates a new COT orchestrator
func NewOrchestratorCOT(pool *pgxpool.Pool, cfg *config.Config, registry *AgentRegistry) *OrchestratorCOT {
	return &OrchestratorCOT{
		pool:         pool,
		config:       cfg,
		registry:     registry,
		router:       NewSmartIntentRouter(pool, cfg),
		messageBus:   make(chan *AgentMessage, 100),
		activeChains: make(map[string]*ChainOfThought),
	}
}

// ExecutionPlan represents the orchestrator's plan for handling a request
type ExecutionPlan struct {
	Strategy     string        `json:"strategy"` // "direct", "delegate", "multi-agent", "sequential"
	PrimaryAgent AgentType     `json:"primary_agent"`
	Steps        []PlannedStep `json:"steps"`
	Reasoning    string        `json:"reasoning"`
	Confidence   float64       `json:"confidence"`
}

// PlannedStep represents a planned execution step
type PlannedStep struct {
	Order         int            `json:"order"`
	Agent         AgentType      `json:"agent"`
	Task          string         `json:"task"`
	DependsOn     []int          `json:"depends_on,omitempty"` // Order numbers of dependencies
	CanParallel   bool           `json:"can_parallel"`
	Priority      int            `json:"priority"`
	NeedsSearch   bool           `json:"needs_search"`             // Whether this step needs web search
	NeedsTools    bool           `json:"needs_tools"`              // Whether this step needs tool access
	ModelOverride string         `json:"model_override,omitempty"` // Optional model override
	Context       map[string]any `json:"context,omitempty"`        // Context passed from previous steps
}

// AgentResult holds result from an agent execution
type AgentResult struct {
	AgentType AgentType
	Output    string
	Error     error
}

// truncateOutput truncates output for display
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
