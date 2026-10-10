package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// WorkflowHandler handles workflow CRUD and execution
type WorkflowHandler struct {
	pool *pgxpool.Pool
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(pool *pgxpool.Pool) *WorkflowHandler {
	return &WorkflowHandler{pool: pool}
}

// CreateWorkflowRequest represents a request to create a workflow
type CreateWorkflowRequest struct {
	Name           string                         `json:"name" binding:"required"`
	DisplayName    string                         `json:"display_name" binding:"required"`
	Description    string                         `json:"description"`
	Trigger        string                         `json:"trigger" binding:"required"`
	ExecutionMode  services.WorkflowExecutionMode `json:"execution_mode"`
	StopOnFailure  bool                           `json:"stop_on_failure"`
	TimeoutSeconds int                            `json:"timeout_seconds"`
	Steps          []CreateWorkflowStepRequest    `json:"steps"`
}

// CreateWorkflowStepRequest represents a step in a workflow creation request
type CreateWorkflowStepRequest struct {
	Name                string                     `json:"name" binding:"required"`
	Description         string                     `json:"description"`
	StepOrder           int                        `json:"step_order" binding:"required"`
	ActionType          services.StepActionType    `json:"action_type" binding:"required"`
	CommandTrigger      *string                    `json:"command_trigger,omitempty"`
	CommandArgs         *string                    `json:"command_args,omitempty"`
	TargetAgentID       *uuid.UUID                 `json:"target_agent_id,omitempty"`
	PromptTemplate      *string                    `json:"prompt_template,omitempty"`
	ToolName            *string                    `json:"tool_name,omitempty"`
	ToolParams          map[string]any             `json:"tool_params,omitempty"`
	ConditionExpression *string                    `json:"condition_expression,omitempty"`
	OnTrueStep          *uuid.UUID                 `json:"on_true_step,omitempty"`
	OnFalseStep         *uuid.UUID                 `json:"on_false_step,omitempty"`
	WaitSeconds         int                        `json:"wait_seconds,omitempty"`
	DependsOn           []uuid.UUID                `json:"depends_on,omitempty"`
	CanParallel         bool                       `json:"can_parallel"`
	OnFailure           services.StepFailureAction `json:"on_failure"`
	MaxRetries          int                        `json:"max_retries"`
	RetryDelaySeconds   int                        `json:"retry_delay_seconds"`
	InputMapping        map[string]string          `json:"input_mapping,omitempty"`
	OutputKey           *string                    `json:"output_key,omitempty"`
}

// ExecuteWorkflowRequest represents a request to execute a workflow
type ExecuteWorkflowRequest struct {
	Input          string     `json:"input"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
}

// ListWorkflows lists all workflows for the user
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	workflowService := services.NewWorkflowService(h.pool)
	workflows, err := workflowService.ListWorkflows(c.Request.Context(), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to list workflows", "user_id", userID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "list workflows", err)
		return
	}

	total := int64(len(workflows))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(workflows) {
		start = len(workflows)
	}
	if end > len(workflows) {
		end = len(workflows)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(workflows[start:end], total, pg))
}

// GetWorkflow gets a workflow by ID
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow ID format", "id", c.Param("id"), "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid workflow ID")
		return
	}

	workflowService := services.NewWorkflowService(h.pool)
	workflow, err := workflowService.GetWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Workflow not found", "workflow_id", workflowID, "error", err)
		utils.RespondNotFound(c, slog.Default(), "workflow not found")
		return
	}

	// Check ownership
	if workflow.UserID != userID && workflow.UserID != "SYSTEM" {
		slog.WarnContext(c.Request.Context(), "Access denied to workflow", "workflow_id", workflowID, "user_id", userID, "owner_id", workflow.UserID)
		utils.RespondForbidden(c, slog.Default(), "access denied")
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CreateWorkflow creates a new workflow
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow creation request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid request")
		return
	}

	// Set defaults
	if req.ExecutionMode == "" {
		req.ExecutionMode = services.ExecutionModeSequential
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 300
	}

	workflow := &services.CommandWorkflow{
		UserID:         userID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Trigger:        req.Trigger,
		ExecutionMode:  req.ExecutionMode,
		StopOnFailure:  req.StopOnFailure,
		TimeoutSeconds: req.TimeoutSeconds,
		IsActive:       true,
		IsSystem:       false,
	}

	workflowService := services.NewWorkflowService(h.pool)

	// Create workflow
	err := workflowService.CreateWorkflow(c.Request.Context(), workflow)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to create workflow", "user_id", userID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "create workflow", err)
		return
	}

	// Add steps
	for _, stepReq := range req.Steps {
		step := &services.WorkflowStep{
			WorkflowID:          workflow.ID,
			Name:                stepReq.Name,
			Description:         stepReq.Description,
			StepOrder:           stepReq.StepOrder,
			ActionType:          stepReq.ActionType,
			CommandTrigger:      stepReq.CommandTrigger,
			CommandArgs:         stepReq.CommandArgs,
			TargetAgentID:       stepReq.TargetAgentID,
			PromptTemplate:      stepReq.PromptTemplate,
			ToolName:            stepReq.ToolName,
			ToolParams:          stepReq.ToolParams,
			ConditionExpression: stepReq.ConditionExpression,
			OnTrueStep:          stepReq.OnTrueStep,
			OnFalseStep:         stepReq.OnFalseStep,
			WaitSeconds:         stepReq.WaitSeconds,
			DependsOn:           stepReq.DependsOn,
			CanParallel:         stepReq.CanParallel,
			OnFailure:           stepReq.OnFailure,
			MaxRetries:          stepReq.MaxRetries,
			RetryDelaySeconds:   stepReq.RetryDelaySeconds,
			InputMapping:        stepReq.InputMapping,
			OutputKey:           stepReq.OutputKey,
		}

		if step.OnFailure == "" {
			step.OnFailure = services.FailureActionStop
		}

		err := workflowService.AddStep(c.Request.Context(), step)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "Failed to add workflow step",
				"workflow_id", workflow.ID,
				"step_name", stepReq.Name,
				"error", err)
			utils.RespondInternalError(c, slog.Default(), "add step", err)
			return
		}
	}

	// Fetch complete workflow with steps
	complete, _ := workflowService.GetWorkflow(c.Request.Context(), workflow.ID)
	if complete != nil {
		workflow = complete
	}

	c.JSON(http.StatusCreated, workflow)
}

// DeleteWorkflow deletes a workflow
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow ID format", "id", c.Param("id"), "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid workflow ID")
		return
	}

	// Delete with ownership check
	result, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM command_workflows WHERE id = $1 AND user_id = $2 AND is_system = FALSE
	`, workflowID, userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to delete workflow", "workflow_id", workflowID, "user_id", userID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "delete workflow", err)
		return
	}

	if result.RowsAffected() == 0 {
		slog.WarnContext(c.Request.Context(), "Workflow not found or cannot be deleted", "workflow_id", workflowID, "user_id", userID)
		utils.RespondNotFound(c, slog.Default(), "workflow not found or cannot be deleted")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"})
}

// ExecuteWorkflow executes a workflow
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow ID format", "id", c.Param("id"), "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid workflow ID")
		return
	}

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow execution request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid request")
		return
	}

	workflowService := services.NewWorkflowService(h.pool)

	workflow, err := workflowService.GetWorkflow(c.Request.Context(), workflowID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Workflow not found", "workflow_id", workflowID, "error", err)
		utils.RespondNotFound(c, slog.Default(), "workflow not found")
		return
	}

	// Check access
	if workflow.UserID != userID && workflow.UserID != "SYSTEM" {
		slog.WarnContext(c.Request.Context(), "Access denied to workflow execution", "workflow_id", workflowID, "user_id", userID, "owner_id", workflow.UserID)
		utils.RespondForbidden(c, slog.Default(), "access denied")
		return
	}

	execution, err := workflowService.ExecuteWorkflow(c.Request.Context(), workflow, userID, req.Input, req.ConversationID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Workflow execution failed",
			"workflow_id", workflowID,
			"user_id", userID,
			"error", err)
		utils.RespondInternalError(c, slog.Default(), "execute workflow", err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ExecuteWorkflowByTrigger executes a workflow by trigger
func (h *WorkflowHandler) ExecuteWorkflowByTrigger(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	trigger := c.Param("trigger")
	if trigger == "" {
		utils.RespondBadRequest(c, slog.Default(), "trigger is required")
		return
	}

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid workflow execution request", "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid request")
		return
	}

	workflowService := services.NewWorkflowService(h.pool)

	workflow, err := workflowService.ResolveWorkflow(c.Request.Context(), userID, trigger)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Workflow not found for trigger", "trigger", trigger, "user_id", userID, "error", err)
		utils.RespondNotFound(c, slog.Default(), "workflow not found")
		return
	}

	execution, err := workflowService.ExecuteWorkflow(c.Request.Context(), workflow, userID, req.Input, req.ConversationID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Workflow execution failed",
			"trigger", trigger,
			"user_id", userID,
			"error", err)
		utils.RespondInternalError(c, slog.Default(), "execute workflow", err)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// ListWorkflowExecutions lists workflow executions for the user
func (h *WorkflowHandler) ListWorkflowExecutions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	workflowService := services.NewWorkflowService(h.pool)
	executions, err := workflowService.ListExecutions(c.Request.Context(), userID, int(pg.Limit), int(pg.Offset))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Failed to list workflow executions", "user_id", userID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "list executions", err)
		return
	}

	total := int64(pg.Offset) + int64(len(executions))
	if int64(len(executions)) == int64(pg.Limit) {
		total++
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(executions, total, pg))
}

// GetWorkflowExecution gets a specific execution
func (h *WorkflowHandler) GetWorkflowExecution(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	executionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Invalid execution ID format", "id", c.Param("id"), "error", err)
		utils.RespondBadRequest(c, slog.Default(), "invalid execution ID")
		return
	}

	workflowService := services.NewWorkflowService(h.pool)
	execution, err := workflowService.GetExecution(c.Request.Context(), executionID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "Execution not found", "execution_id", executionID, "error", err)
		utils.RespondNotFound(c, slog.Default(), "execution not found")
		return
	}

	// Check ownership
	if execution.UserID != userID {
		slog.WarnContext(c.Request.Context(), "Access denied to workflow execution", "execution_id", executionID, "user_id", userID, "owner_id", execution.UserID)
		utils.RespondForbidden(c, slog.Default(), "access denied")
		return
	}

	c.JSON(http.StatusOK, execution)
}
