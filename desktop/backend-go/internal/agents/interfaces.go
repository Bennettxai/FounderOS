package agents

import (
	"context"
	"time"
)

// AgentCapability represents a specific capability an agent can perform.
type AgentCapability string

const (
	CapabilityChat     AgentCapability = "chat"
	CapabilityCode     AgentCapability = "code"
	CapabilityAnalysis AgentCapability = "analysis"
	CapabilitySearch   AgentCapability = "search"
	CapabilityWrite    AgentCapability = "write"
	CapabilityReason   AgentCapability = "reason"
)

// AgentStatus represents the current status of an agent.
type AgentStatus string

const (
	AgentStatusIdle    AgentStatus = "idle"
	AgentStatusRunning AgentStatus = "running"
	AgentStatusError   AgentStatus = "error"
)

// Tool represents a tool available to an agent.
type Tool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
}

// TaskInput is the input payload for an agent task.
type TaskInput struct {
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TaskResult is the output of an agent task.
type TaskResult struct {
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	TokensUsed int               `json:"tokens_used,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// AgentTask is a unit of work dispatched to an agent.
type AgentTask struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Description string    `json:"description"`
	Input       TaskInput `json:"input"`
	CreatedAt   time.Time `json:"created_at"`
}

// AgentEvaluation records a quality score for an agent's output.
type AgentEvaluation struct {
	AgentID     string    `json:"agent_id"`
	TaskID      string    `json:"task_id"`
	Score       float64   `json:"score"` // 0-10
	Feedback    string    `json:"feedback"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

// AgentWorkflow is a sequence of agent tasks.
type AgentWorkflow struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Steps       []AgentTask `json:"steps"`
	TenantID    string      `json:"tenant_id"`
}

// RunnableAgent is the interface all registry-managed agents must satisfy.
// It is distinct from the legacy Agent interface defined in agents.go so that
// both can coexist in the same package without modification to existing files.
type RunnableAgent interface {
	// AgentID returns the unique identifier of this agent.
	AgentID() string
	// AgentName returns the human-readable name of this agent.
	AgentName() string
	// Capabilities returns the set of capabilities this agent supports.
	Capabilities() []AgentCapability
	// AgentTools returns the tools available to this agent.
	AgentTools() []Tool
	// Execute performs the given task and returns a result.
	Execute(ctx context.Context, task AgentTask) (*TaskResult, error)
	// AgentStatus returns the current status of the agent.
	AgentStatus() AgentStatus
}

// AgentEvaluator evaluates agent task results.
type AgentEvaluator interface {
	Evaluate(ctx context.Context, task AgentTask, result TaskResult) (*AgentEvaluation, error)
}

// WorkflowRunner executes multi-step agent workflows.
type WorkflowRunner interface {
	Run(ctx context.Context, workflow AgentWorkflow) (*TaskResult, error)
}
