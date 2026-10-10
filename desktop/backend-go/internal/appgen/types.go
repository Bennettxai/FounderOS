package appgen

import (
	"context"
	"time"
)

// AgentType represents specialized agent roles
type AgentType string

const (
	AgentFrontend AgentType = "frontend"
	AgentBackend  AgentType = "backend"
	AgentDatabase AgentType = "database"
	AgentTest     AgentType = "test"
)

// AppRequest represents an app generation request
type AppRequest struct {
	AppName     string   `json:"app_name"`
	Description string   `json:"description"`
	Features    []string `json:"features,omitempty"`
}

// Plan is the architectural plan from orchestrator
type Plan struct {
	Architecture string    `json:"architecture"`
	Tasks        []Task    `json:"tasks"`
	CreatedAt    time.Time `json:"created_at"`
}

// Task is a single agent task
type Task struct {
	ID          string    `json:"id"`
	Type        AgentType `json:"type"`
	Description string    `json:"description"`
	Workspace   string    `json:"workspace"`
	Priority    int       `json:"priority"`
}

// ProgressEvent for SSE streaming
type ProgressEvent struct {
	TaskID    string    `json:"task_id"`
	AgentType AgentType `json:"agent_type"`
	Status    string    `json:"status"` // starting, in_progress, completed, failed
	Message   string    `json:"message"`
	Progress  int       `json:"progress"` // 0-100
	Timestamp time.Time `json:"timestamp"`
}

// AgentResult from worker execution
type AgentResult struct {
	TaskID       string            `json:"task_id"`
	AgentType    AgentType         `json:"agent_type"`
	Success      bool              `json:"success"`
	Output       string            `json:"output"`
	FilesCreated []string          `json:"files_created"`
	CodeBlocks   map[string]string `json:"code_blocks,omitempty"` // filepath -> content
	Error        string            `json:"error,omitempty"`
	Duration     time.Duration     `json:"duration"`
}

// GeneratedApp is the final result
type GeneratedApp struct {
	AppName       string        `json:"app_name"`
	Results       []AgentResult `json:"results"`
	Success       bool          `json:"success"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	WorkspacePath string        `json:"workspace_path,omitempty"`
	TotalDuration time.Duration `json:"total_duration"`
	CreatedAt     time.Time     `json:"created_at"`
}

// ProgressCallback for real-time updates
type ProgressCallback func(event ProgressEvent)

// Orchestrator coordinates 4 agents in parallel
type Orchestrator interface {
	CreatePlan(ctx context.Context, req AppRequest) (*Plan, error)
	Execute(ctx context.Context, plan *Plan) (*GeneratedApp, error)
	SetProgressCallback(callback ProgressCallback)
	Shutdown() error
	GetCircuitBreakerMetrics() map[string]interface{}
}

// Worker executes a specific agent task
type Worker interface {
	Execute(ctx context.Context, task Task) (*AgentResult, error)
	Type() AgentType
}
