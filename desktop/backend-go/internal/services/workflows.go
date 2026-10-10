package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkflowExecutionMode defines how steps are executed
type WorkflowExecutionMode string

const (
	ExecutionModeSequential WorkflowExecutionMode = "sequential"
	ExecutionModeParallel   WorkflowExecutionMode = "parallel"
	ExecutionModeSmart      WorkflowExecutionMode = "smart" // Analyzes dependencies
)

// StepActionType defines the type of action a step performs
type StepActionType string

const (
	StepActionCommand   StepActionType = "command"
	StepActionAgent     StepActionType = "agent"
	StepActionTool      StepActionType = "tool"
	StepActionCondition StepActionType = "condition"
	StepActionWait      StepActionType = "wait"
)

// StepFailureAction defines what to do when a step fails
type StepFailureAction string

const (
	FailureActionStop     StepFailureAction = "stop"
	FailureActionContinue StepFailureAction = "continue"
	FailureActionRetry    StepFailureAction = "retry"
	FailureActionSkip     StepFailureAction = "skip"
)

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusSkipped   ExecutionStatus = "skipped"
)

// CommandWorkflow represents a multi-step workflow
type CommandWorkflow struct {
	ID             uuid.UUID             `json:"id"`
	UserID         string                `json:"user_id"`
	Name           string                `json:"name"`
	DisplayName    string                `json:"display_name"`
	Description    string                `json:"description,omitempty"`
	Trigger        string                `json:"trigger"`
	ExecutionMode  WorkflowExecutionMode `json:"execution_mode"`
	StopOnFailure  bool                  `json:"stop_on_failure"`
	TimeoutSeconds int                   `json:"timeout_seconds"`
	IsActive       bool                  `json:"is_active"`
	IsSystem       bool                  `json:"is_system"`
	Steps          []WorkflowStep        `json:"steps,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID                  uuid.UUID         `json:"id"`
	WorkflowID          uuid.UUID         `json:"workflow_id"`
	Name                string            `json:"name"`
	Description         string            `json:"description,omitempty"`
	StepOrder           int               `json:"step_order"`
	ActionType          StepActionType    `json:"action_type"`
	CommandTrigger      *string           `json:"command_trigger,omitempty"`
	CommandArgs         *string           `json:"command_args,omitempty"`
	TargetAgentID       *uuid.UUID        `json:"target_agent_id,omitempty"`
	PromptTemplate      *string           `json:"prompt_template,omitempty"`
	ToolName            *string           `json:"tool_name,omitempty"`
	ToolParams          map[string]any    `json:"tool_params,omitempty"`
	ConditionExpression *string           `json:"condition_expression,omitempty"`
	OnTrueStep          *uuid.UUID        `json:"on_true_step,omitempty"`
	OnFalseStep         *uuid.UUID        `json:"on_false_step,omitempty"`
	WaitSeconds         int               `json:"wait_seconds,omitempty"`
	DependsOn           []uuid.UUID       `json:"depends_on,omitempty"`
	CanParallel         bool              `json:"can_parallel"`
	OnFailure           StepFailureAction `json:"on_failure"`
	MaxRetries          int               `json:"max_retries"`
	RetryDelaySeconds   int               `json:"retry_delay_seconds"`
	InputMapping        map[string]string `json:"input_mapping,omitempty"`
	OutputKey           *string           `json:"output_key,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// WorkflowExecution represents a running or completed workflow execution
type WorkflowExecution struct {
	ID             uuid.UUID       `json:"id"`
	WorkflowID     uuid.UUID       `json:"workflow_id"`
	UserID         string          `json:"user_id"`
	ConversationID *uuid.UUID      `json:"conversation_id,omitempty"`
	InitialInput   string          `json:"initial_input,omitempty"`
	Context        map[string]any  `json:"context"`
	Status         ExecutionStatus `json:"status"`
	CurrentStepID  *uuid.UUID      `json:"current_step_id,omitempty"`
	Result         map[string]any  `json:"result,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// StepExecution represents a step execution within a workflow
type StepExecution struct {
	ID            uuid.UUID       `json:"id"`
	ExecutionID   uuid.UUID       `json:"execution_id"`
	StepID        uuid.UUID       `json:"step_id"`
	Status        ExecutionStatus `json:"status"`
	AttemptNumber int             `json:"attempt_number"`
	Input         map[string]any  `json:"input,omitempty"`
	Output        map[string]any  `json:"output,omitempty"`
	ErrorMessage  *string         `json:"error_message,omitempty"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	DurationMs    *float64        `json:"duration_ms,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// WorkflowService handles workflow operations
type WorkflowService struct {
	pool           *pgxpool.Pool
	commandService *CommandService
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(pool *pgxpool.Pool) *WorkflowService {
	return &WorkflowService{
		pool:           pool,
		commandService: NewCommandService(pool),
	}
}
