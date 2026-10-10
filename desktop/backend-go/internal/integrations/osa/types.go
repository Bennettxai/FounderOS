package osa

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AppGenerationRequest represents a request to generate an application
type AppGenerationRequest struct {
	UserID      uuid.UUID              `json:"user_id"`
	WorkspaceID uuid.UUID              `json:"workspace_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // "full-stack", "module", "tool"
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// AppGenerationResponse represents the response from app generation
type AppGenerationResponse struct {
	AppID       string                 `json:"app_id"`
	Status      string                 `json:"status"` // "pending", "processing", "completed", "failed"
	WorkspaceID string                 `json:"workspace_id"`
	Message     string                 `json:"message,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// AppStatusResponse represents the status of an app generation
type AppStatusResponse struct {
	AppID       string                 `json:"app_id"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"` // 0.0 to 1.0
	CurrentStep string                 `json:"current_step,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OrchestrateRequest represents a request to run the full 21-agent orchestration
type OrchestrateRequest struct {
	UserID      uuid.UUID              `json:"user_id"`
	Input       string                 `json:"input"`
	SessionID   string                 `json:"session_id,omitempty"` // Correlates with Stream() for real-time events
	Phase       string                 `json:"phase,omitempty"`      // "analysis", "strategy", "development", etc.
	Context     map[string]interface{} `json:"context,omitempty"`
	WorkspaceID uuid.UUID              `json:"workspace_id,omitempty"`
}

// OrchestrateResponse represents the response from orchestration
type OrchestrateResponse struct {
	Success       bool                   `json:"success"`
	Output        string                 `json:"output"`
	AgentsUsed    []string               `json:"agents_used,omitempty"`
	ExecutionTime int64                  `json:"execution_ms"`
	Data          map[string]interface{} `json:"data,omitempty"`
	NextStep      string                 `json:"next_step,omitempty"`
}

// Workspace represents an OSA workspace
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspacesResponse represents the list of workspaces
type WorkspacesResponse struct {
	Workspaces []Workspace `json:"workspaces"`
	Total      int         `json:"total"`
}

// HealthResponse represents OSA health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ErrorResponse represents an error from OSA API
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

const (
	EventThinking       = "thinking"
	EventResponse       = "response"
	EventSkillStarted   = "skill_started"
	EventSkillCompleted = "skill_completed"
	EventSkillFailed    = "skill_failed"
	EventError          = "error"
	EventConnected      = "connected"
	EventSignal         = "signal"
)

// Event represents a streaming event emitted by the OSA runtime.
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// SwarmRequest is the payload for launching a multi-agent swarm in OSA.
type SwarmRequest struct {
	Pattern   string                 `json:"pattern"`
	Task      string                 `json:"task"`
	Config    map[string]interface{} `json:"config,omitempty"`
	MaxAgents int                    `json:"max_agents,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// SwarmResponse is returned after launching a swarm.
type SwarmResponse struct {
	SwarmID   string `json:"swarm_id"`
	Status    string `json:"status"`
	SessionID string `json:"session_id,omitempty"`
}

// SwarmStatus represents the current state of a swarm.
type SwarmStatus struct {
	SwarmID    string                 `json:"swarm_id"`
	Status     string                 `json:"status"`
	Pattern    string                 `json:"pattern,omitempty"`
	AgentCount int                    `json:"agent_count,omitempty"`
	Progress   float64                `json:"progress,omitempty"`
	Results    map[string]interface{} `json:"results,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// ToolDefinition describes an OSA tool exposed by the runtime.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Source      string                 `json:"source,omitempty"`
}

// ToolResult is returned after executing an OSA tool.
type ToolResult struct {
	Success bool                   `json:"success"`
	Output  interface{}            `json:"output,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// Instruction is a CloudEvent-like dispatch payload sent to an OSA fleet agent.
type Instruction struct {
	SpecVersion string                 `json:"specversion,omitempty"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	ID          string                 `json:"id"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// APIError preserves HTTP status information from OSA runtime failures so the
// resilience layer can decide whether an operation is retryable.
type APIError struct {
	StatusCode int
	ErrorCode  string
	Details    string
}

func (e *APIError) Error() string {
	message := e.Details
	if message == "" {
		message = e.ErrorCode
	}
	if e.Details != "" {
		return fmt.Sprintf("status %d: %s", e.StatusCode, message)
	}
	if message != "" {
		return fmt.Sprintf("status %d: %s", e.StatusCode, message)
	}
	return fmt.Sprintf("status %d: OSA runtime request failed", e.StatusCode)
}
