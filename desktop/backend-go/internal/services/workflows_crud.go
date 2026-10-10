package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateWorkflow creates a new workflow
func (s *WorkflowService) CreateWorkflow(ctx context.Context, workflow *CommandWorkflow) error {
	if s.pool == nil {
		return fmt.Errorf("no database connection")
	}

	// Ensure trigger starts with /
	if !strings.HasPrefix(workflow.Trigger, "/") {
		workflow.Trigger = "/" + workflow.Trigger
	}

	workflow.ID = uuid.New()
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO command_workflows (
			id, user_id, name, display_name, description, trigger,
			execution_mode, stop_on_failure, timeout_seconds, is_active, is_system
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, workflow.ID, workflow.UserID, workflow.Name, workflow.DisplayName,
		workflow.Description, workflow.Trigger, workflow.ExecutionMode,
		workflow.StopOnFailure, workflow.TimeoutSeconds, workflow.IsActive, workflow.IsSystem)

	return err
}

// AddStep adds a step to a workflow
func (s *WorkflowService) AddStep(ctx context.Context, step *WorkflowStep) error {
	if s.pool == nil {
		return fmt.Errorf("no database connection")
	}

	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()

	toolParamsJSON, _ := json.Marshal(step.ToolParams)
	inputMappingJSON, _ := json.Marshal(step.InputMapping)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_steps (
			id, workflow_id, name, description, step_order, action_type,
			command_trigger, command_args, target_agent_id, prompt_template,
			tool_name, tool_params, condition_expression, on_true_step, on_false_step,
			wait_seconds, depends_on, can_parallel, on_failure, max_retries,
			retry_delay_seconds, input_mapping, output_key
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
	`, step.ID, step.WorkflowID, step.Name, step.Description, step.StepOrder,
		step.ActionType, step.CommandTrigger, step.CommandArgs, step.TargetAgentID,
		step.PromptTemplate, step.ToolName, toolParamsJSON, step.ConditionExpression,
		step.OnTrueStep, step.OnFalseStep, step.WaitSeconds, step.DependsOn,
		step.CanParallel, step.OnFailure, step.MaxRetries, step.RetryDelaySeconds,
		inputMappingJSON, step.OutputKey)

	return err
}

// GetWorkflow gets a workflow by ID with its steps
func (s *WorkflowService) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*CommandWorkflow, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var workflow CommandWorkflow
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, name, display_name, description, trigger,
			execution_mode, stop_on_failure, timeout_seconds, is_active, is_system,
			created_at, updated_at
		FROM command_workflows WHERE id = $1
	`, workflowID).Scan(
		&workflow.ID, &workflow.UserID, &workflow.Name, &workflow.DisplayName,
		&workflow.Description, &workflow.Trigger, &workflow.ExecutionMode,
		&workflow.StopOnFailure, &workflow.TimeoutSeconds, &workflow.IsActive,
		&workflow.IsSystem, &workflow.CreatedAt, &workflow.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Get steps
	rows, err := s.pool.Query(ctx, `
		SELECT id, workflow_id, name, description, step_order, action_type,
			command_trigger, command_args, target_agent_id, prompt_template,
			tool_name, tool_params, condition_expression, on_true_step, on_false_step,
			wait_seconds, depends_on, can_parallel, on_failure, max_retries,
			retry_delay_seconds, input_mapping, output_key, created_at, updated_at
		FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_order
	`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var step WorkflowStep
		var toolParamsJSON, inputMappingJSON []byte
		err := rows.Scan(
			&step.ID, &step.WorkflowID, &step.Name, &step.Description, &step.StepOrder,
			&step.ActionType, &step.CommandTrigger, &step.CommandArgs, &step.TargetAgentID,
			&step.PromptTemplate, &step.ToolName, &toolParamsJSON, &step.ConditionExpression,
			&step.OnTrueStep, &step.OnFalseStep, &step.WaitSeconds, &step.DependsOn,
			&step.CanParallel, &step.OnFailure, &step.MaxRetries, &step.RetryDelaySeconds,
			&inputMappingJSON, &step.OutputKey, &step.CreatedAt, &step.UpdatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal(toolParamsJSON, &step.ToolParams)
		json.Unmarshal(inputMappingJSON, &step.InputMapping)
		workflow.Steps = append(workflow.Steps, step)
	}

	return &workflow, nil
}

// ResolveWorkflow finds a workflow by trigger
func (s *WorkflowService) ResolveWorkflow(ctx context.Context, userID string, trigger string) (*CommandWorkflow, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	if !strings.HasPrefix(trigger, "/") {
		trigger = "/" + trigger
	}
	trigger = strings.ToLower(trigger)

	var workflowID uuid.UUID

	// Try user's workflows first
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM command_workflows
		WHERE user_id = $1 AND trigger = $2 AND is_active = TRUE
	`, userID, trigger).Scan(&workflowID)
	if err != nil {
		// Try system workflows
		err = s.pool.QueryRow(ctx, `
			SELECT id FROM command_workflows
			WHERE user_id = 'SYSTEM' AND trigger = $1 AND is_active = TRUE
		`, trigger).Scan(&workflowID)
		if err != nil {
			return nil, fmt.Errorf("workflow not found: %s", trigger)
		}
	}

	return s.GetWorkflow(ctx, workflowID)
}

// ListWorkflows lists all workflows for a user
func (s *WorkflowService) ListWorkflows(ctx context.Context, userID string) ([]CommandWorkflow, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, display_name, description, trigger,
			execution_mode, stop_on_failure, timeout_seconds, is_active, is_system,
			created_at, updated_at
		FROM command_workflows
		WHERE user_id = $1 OR user_id = 'SYSTEM'
		ORDER BY is_system DESC, name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []CommandWorkflow
	for rows.Next() {
		var w CommandWorkflow
		err := rows.Scan(
			&w.ID, &w.UserID, &w.Name, &w.DisplayName, &w.Description, &w.Trigger,
			&w.ExecutionMode, &w.StopOnFailure, &w.TimeoutSeconds, &w.IsActive,
			&w.IsSystem, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			continue
		}
		workflows = append(workflows, w)
	}

	return workflows, nil
}
