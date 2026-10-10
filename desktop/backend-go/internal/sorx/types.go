// Package sorx defines types for the Sorx 2.0 skill execution engine.
// Sorx 2.0 is a Universal Skill-Based Integration Framework where AI agents
// learn skills to connect with any system.
package sorx

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Status Constants
// ============================================================================

const (
	StatusPending         = "pending"
	StatusRunning         = "running"
	StatusWaitingCallback = "waiting_callback"
	StatusWaitingDecision = "waiting_decision"
	StatusComplete        = "complete"
	StatusFailed          = "failed"
	StatusCancelled       = "cancelled"
)

// ============================================================================
// Skill Reliability Tiers
// ============================================================================

// SkillTier represents the reliability tier of a skill.
// Higher tiers require more AI involvement but have lower reliability.
type SkillTier int

const (
	// TierDeterministic (Tier 1): 100% uptime when API is up.
	// Pure code execution, no AI involved.
	// Model: NONE
	TierDeterministic SkillTier = 1

	// TierStructuredAI (Tier 2): 95-99% uptime.
	// AI for parameter extraction or simple decisions, core execution deterministic.
	// Model: HAIKU
	TierStructuredAI SkillTier = 2

	// TierReasoningAI (Tier 3): 80-95% uptime.
	// Complex reasoning, multiple steps, may need human-in-the-loop.
	// Model: SONNET
	TierReasoningAI SkillTier = 3

	// TierGenerativeAI (Tier 4): Variable uptime.
	// New skill generation, novel situations, highest risk.
	// Model: OPUS
	TierGenerativeAI SkillTier = 4
)

// TierToModel returns the appropriate model for a skill tier.
func TierToModel(tier SkillTier) string {
	switch tier {
	case TierDeterministic:
		return "" // No model needed
	case TierStructuredAI:
		return "claude-3-haiku-20240307"
	case TierReasoningAI:
		return "claude-sonnet-4-20250514"
	case TierGenerativeAI:
		return "claude-opus-4-1-20250805"
	default:
		return "claude-sonnet-4-20250514"
	}
}

// ============================================================================
// Role Types - Skills belong to roles like people in a company
// ============================================================================

// Role represents a functional role that owns skills.
// Think of roles like job functions in a company.
type Role string

const (
	RoleAny        Role = "any"        // Universal skills
	RoleSales      Role = "sales"      // CRM, outreach, deals
	RoleSupport    Role = "support"    // Tickets, customer service
	RoleFinance    Role = "finance"    // Invoicing, payments, accounting
	RoleOperations Role = "ops"        // Projects, tasks, workflows
	RoleMarketing  Role = "marketing"  // Campaigns, content, analytics
	RoleExecutive  Role = "executive"  // Reports, summaries, decisions
	RoleAnalyst    Role = "analyst"    // Data analysis, insights
	RoleDocument   Role = "document"   // Writing, documentation
)

// ============================================================================
// Step Types
// ============================================================================

const (
	StepTypeAction    = "action"
	StepTypeDecision  = "decision"
	StepTypeCondition = "condition"
	StepTypeLoop      = "loop"
	StepTypeParallel  = "parallel"
	StepTypeAgent     = "agent" // Invoke a BusinessOS agent
)

// ============================================================================
// Event Types
// ============================================================================

const (
	EventDecisionMade            = "decision_made"
	EventIntegrationConnected    = "integration_connected"
	EventIntegrationDisconnected = "integration_disconnected"
	EventSkillStarted            = "skill_started"
	EventSkillCompleted          = "skill_completed"
	EventSkillFailed             = "skill_failed"
	EventStepCompleted           = "step_completed"
	EventHumanInputRequired      = "human_input_required"
)

// ============================================================================
// Temperature Control - Human autonomy dial
// ============================================================================

// Temperature represents the level of autonomy granted to the system.
// Humans control this dial to determine how much the system can do autonomously.
type Temperature string

const (
	// TemperatureCold: Full human control - every action requires approval
	TemperatureCold Temperature = "cold"

	// TemperatureWarm: Balanced - routine actions auto-execute, complex wait
	TemperatureWarm Temperature = "warm"

	// TemperatureHot: High autonomy - most actions auto-execute, only critical escalate
	TemperatureHot Temperature = "hot"
)

// ============================================================================
// Core Request/Response Types
// ============================================================================

// ExecuteRequest represents a request to execute a skill.
type ExecuteRequest struct {
	SkillID     string                 `json:"skill_id"`
	UserID      string                 `json:"user_id"`
	Params      map[string]interface{} `json:"params"`
	Temperature Temperature            `json:"temperature,omitempty"` // Override user's default
	Async       bool                   `json:"async,omitempty"`       // Return immediately
}

// Execution represents a running or completed skill execution.
type Execution struct {
	ID          uuid.UUID              `json:"id"`
	SkillID     string                 `json:"skill_id"`
	UserID      string                 `json:"user_id"`
	Status      string                 `json:"status"`
	CurrentStep int                    `json:"current_step"`
	TotalSteps  int                    `json:"total_steps"`
	Params      map[string]interface{} `json:"params"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Context     map[string]interface{} `json:"context"`
	StepResults map[string]interface{} `json:"step_results"`
	Metrics     *ExecutionMetrics      `json:"metrics,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ExecutionMetrics tracks performance metrics for an execution.
type ExecutionMetrics struct {
	TotalDuration   time.Duration `json:"total_duration_ms"`
	StepDurations   []int64       `json:"step_durations_ms"`
	ModelCalls      int           `json:"model_calls"`
	TokensUsed      int           `json:"tokens_used"`
	EstimatedCost   float64       `json:"estimated_cost"`
	SuccessfulSteps int           `json:"successful_steps"`
	FailedSteps     int           `json:"failed_steps"`
}

// ============================================================================
// Skill Definition
// ============================================================================

// SkillDefinition defines a skill that can be executed.
type SkillDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version,omitempty"`
	Author      string `json:"author,omitempty"`

	// Tier and Model Selection
	Tier  SkillTier `json:"tier"`
	Model string    `json:"model,omitempty"` // Override automatic model selection

	// Role Affinity - which roles can use this skill
	RoleAffinity []Role `json:"role_affinity"`

	// Integration Requirements
	RequiredIntegrations []string `json:"required_integrations"`

	// Data Connectors - systems this skill connects to
	DataConnectors []DataConnector `json:"data_connectors,omitempty"`

	// Objective Data Points - what this skill can satisfy
	DataPointsSatisfied []string `json:"data_points_satisfied,omitempty"`

	// Steps define the workflow
	Steps []Step `json:"steps"`

	// Health Metrics
	SuccessRate    float64 `json:"success_rate,omitempty"`
	AvgExecutionMs int64   `json:"avg_execution_ms,omitempty"`
	TotalExecutions int    `json:"total_executions,omitempty"`

	// Temperature thresholds - when to require approval
	RequiresApprovalAt Temperature `json:"requires_approval_at,omitempty"`
}

// DataConnector defines a data source a skill connects to.
type DataConnector struct {
	ID           string `json:"id"`
	Type         string `json:"type"`         // api, database, file, desktop
	Integration  string `json:"integration"`  // hubspot, gmail, postgres, etc.
	Description  string `json:"description"`
	DataProvided string `json:"data_provided"` // What data this connector provides
}

// Step represents a single step in a skill.
type Step struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"` // action, decision, condition, loop, parallel, agent
	Action           string                 `json:"action,omitempty"`
	Integration      string                 `json:"integration,omitempty"`
	Params           map[string]interface{} `json:"params,omitempty"`
	Condition        string                 `json:"condition,omitempty"`
	RequiresDecision bool                   `json:"requires_decision,omitempty"`
	DecisionQuestion string                 `json:"decision_question,omitempty"`
	DecisionOptions  []string               `json:"decision_options,omitempty"`
	InputFields      map[string]InputField  `json:"input_fields,omitempty"`
	Priority         string                 `json:"priority,omitempty"`
	Substeps         []Step                 `json:"substeps,omitempty"`
	OnError          string                 `json:"on_error,omitempty"` // continue, stop, retry
	Timeout          time.Duration          `json:"timeout,omitempty"`

	// Agent invocation (for StepTypeAgent)
	AgentType   string `json:"agent_type,omitempty"`   // orchestrator, analyst, document, etc.
	AgentPrompt string `json:"agent_prompt,omitempty"` // The task for the agent
}

// InputField defines a field for human input during decision steps.
type InputField struct {
	Type        string   `json:"type"`                  // text, select, multiselect, number, date
	Label       string   `json:"label"`
	Required    bool     `json:"required,omitempty"`
	Options     []string `json:"options,omitempty"`     // For select/multiselect
	Default     string   `json:"default,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Validation  string   `json:"validation,omitempty"`  // Regex or rule
}

// ============================================================================
// Credentials
// ============================================================================

// Credentials holds encrypted credentials for an integration.
type Credentials struct {
	Provider              string
	AccessTokenEncrypted  []byte
	RefreshTokenEncrypted []byte
	ExpiresAt             *time.Time
	Scopes                []string
}

// ============================================================================
// Events
// ============================================================================

// Event represents an event in the Sorx system.
type Event struct {
	Type        string
	ExecutionID uuid.UUID
	StepID      string
	Data        interface{}
	Timestamp   time.Time
}

// ============================================================================
// Action Context
// ============================================================================

// ActionContext provides context for action execution.
type ActionContext struct {
	Execution   *Execution
	Step        *Step
	Credentials *Credentials
	Params      map[string]interface{}
}

// ActionHandler is a function that executes an action.
type ActionHandler func(ctx context.Context, ac ActionContext) (interface{}, error)

// actionHandlers registry for action implementations.
var actionHandlers = make(map[string]ActionHandler)

// RegisterAction registers an action handler.
func RegisterAction(name string, handler ActionHandler) {
	actionHandlers[name] = handler
}

// GetActionHandler retrieves an action handler by name.
func GetActionHandler(name string) (ActionHandler, bool) {
	handler, ok := actionHandlers[name]
	return handler, ok
}

// UnregisterAction removes an action handler by name.
func UnregisterAction(name string) {
	delete(actionHandlers, name)
}

// ============================================================================
// Decision Response
// ============================================================================

// DecisionResponse represents a human's response to a decision.
type DecisionResponse struct {
	DecisionID uuid.UUID              `json:"decision_id"`
	Decision   string                 `json:"decision"`
	Inputs     map[string]interface{} `json:"inputs,omitempty"`
	Comment    string                 `json:"comment,omitempty"`
	DecidedBy  string                 `json:"decided_by"`
	DecidedAt  time.Time              `json:"decided_at"`
}

// ============================================================================
// Skill Templates
// ============================================================================

// SkillTemplate represents a customizable skill template.
type SkillTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	BaseSkillID  string                 `json:"base_skill_id"`
	Customizable []string               `json:"customizable"` // which params can be customized
	Defaults     map[string]interface{} `json:"defaults"`
	UserID       string                 `json:"user_id,omitempty"` // if user-specific
}

// ============================================================================
// Skill Run History
// ============================================================================

// SkillRun represents a historical skill execution record.
type SkillRun struct {
	ID           uuid.UUID              `json:"id"`
	SkillID      string                 `json:"skill_id"`
	SkillName    string                 `json:"skill_name"`
	UserID       string                 `json:"user_id"`
	Status       string                 `json:"status"`
	Duration     time.Duration          `json:"duration"`
	NodesCreated int                    `json:"nodes_created"`
	Params       map[string]interface{} `json:"params"`
	Result       map[string]interface{} `json:"result,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// ============================================================================
// Skill Health Monitoring
// ============================================================================

// SkillHealth represents the health status of a skill.
type SkillHealth struct {
	SkillID         string    `json:"skill_id"`
	Status          string    `json:"status"` // healthy, degraded, needs_split, critical
	SuccessRate     float64   `json:"success_rate"`
	ErrorRate       float64   `json:"error_rate"`
	AvgLatencyMs    int64     `json:"avg_latency_ms"`
	TotalExecutions int       `json:"total_executions"`
	RecentFailures  int       `json:"recent_failures"`
	LastChecked     time.Time `json:"last_checked"`
	NeedsSplit      bool      `json:"needs_split"`
	SplitReason     string    `json:"split_reason,omitempty"`
}

// ============================================================================
// Objective Database - Data Points that define task completion
// ============================================================================

// ObjectiveDataPoint represents a condition that must be true for task completion.
type ObjectiveDataPoint struct {
	ID          string `json:"id"`
	Entity      string `json:"entity"`    // deal, client, project, task, etc.
	Field       string `json:"field"`     // status, signed, paid, etc.
	Condition   string `json:"condition"` // equals, true, greater_than, etc.
	Value       string `json:"value"`     // The expected value
	Description string `json:"description"`
}

// TaskObjective represents a task with its required data points.
type TaskObjective struct {
	TaskID     string               `json:"task_id"`
	TaskName   string               `json:"task_name"`
	DataPoints []ObjectiveDataPoint `json:"data_points"`
	Satisfied  []bool               `json:"satisfied"` // Which data points are satisfied
	Progress   float64              `json:"progress"`  // 0.0 to 1.0
}

// ============================================================================
// User Preferences
// ============================================================================

// UserSkillPreferences stores user-specific skill settings.
type UserSkillPreferences struct {
	UserID         string      `json:"user_id"`
	Temperature    Temperature `json:"temperature"`
	AutoApprove    []string    `json:"auto_approve"`    // Skill IDs that auto-approve
	AlwaysApprove  []string    `json:"always_approve"`  // Skill IDs that always need approval
	DisabledSkills []string    `json:"disabled_skills"` // Skills the user has disabled
}
