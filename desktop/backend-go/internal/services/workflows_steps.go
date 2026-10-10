package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// executeStep executes a single step
func (s *WorkflowService) executeStep(ctx context.Context, step *WorkflowStep, exec *WorkflowExecution) (*StepExecution, error) {
	stepExec := &StepExecution{
		ID:            uuid.New(),
		ExecutionID:   exec.ID,
		StepID:        step.ID,
		Status:        StatusRunning,
		AttemptNumber: 1,
		Input:         make(map[string]any),
		Output:        make(map[string]any),
		CreatedAt:     time.Now(),
	}
	now := time.Now()
	stepExec.StartedAt = &now

	// Map input from context using input_mapping
	for outputKey, inputPath := range step.InputMapping {
		if val, ok := exec.Context[inputPath]; ok {
			stepExec.Input[outputKey] = val
		}
	}

	// Save step execution start
	inputJSON, _ := json.Marshal(stepExec.Input)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO workflow_step_executions (
			id, execution_id, step_id, status, attempt_number, input, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, stepExec.ID, stepExec.ExecutionID, stepExec.StepID, stepExec.Status,
		stepExec.AttemptNumber, inputJSON, stepExec.StartedAt)

	var err error

	switch step.ActionType {
	case StepActionCommand:
		err = s.executeCommandStep(ctx, step, exec, stepExec)
	case StepActionAgent:
		err = s.executeAgentStep(ctx, step, exec, stepExec)
	case StepActionTool:
		err = s.executeToolStep(ctx, step, exec, stepExec)
	case StepActionCondition:
		err = s.executeConditionStep(ctx, step, exec, stepExec)
	case StepActionWait:
		time.Sleep(time.Duration(step.WaitSeconds) * time.Second)
		stepExec.Output["waited_seconds"] = step.WaitSeconds
	default:
		err = fmt.Errorf("unknown action type: %s", step.ActionType)
	}

	// Update step execution
	completedAt := time.Now()
	stepExec.CompletedAt = &completedAt
	duration := float64(completedAt.Sub(*stepExec.StartedAt).Milliseconds())
	stepExec.DurationMs = &duration

	if err != nil {
		stepExec.Status = StatusFailed
		errMsg := err.Error()
		stepExec.ErrorMessage = &errMsg
	} else {
		stepExec.Status = StatusCompleted
	}

	outputJSON, _ := json.Marshal(stepExec.Output)
	_, _ = s.pool.Exec(ctx, `
		UPDATE workflow_step_executions
		SET status = $1, output = $2, error_message = $3, completed_at = $4, duration_ms = $5
		WHERE id = $6
	`, stepExec.Status, outputJSON, stepExec.ErrorMessage, stepExec.CompletedAt,
		stepExec.DurationMs, stepExec.ID)

	return stepExec, err
}

// executeCommandStep runs a command step
func (s *WorkflowService) executeCommandStep(ctx context.Context, step *WorkflowStep, exec *WorkflowExecution, stepExec *StepExecution) error {
	if step.CommandTrigger == nil {
		return fmt.Errorf("command step missing trigger")
	}

	args := ""
	if step.CommandArgs != nil {
		args = s.interpolate(*step.CommandArgs, exec.Context)
	}

	result, err := s.commandService.ExecuteCommand(ctx, exec.UserID, *step.CommandTrigger, args, nil)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("command failed: %s", result.Error)
	}

	stepExec.Output["command"] = *step.CommandTrigger
	stepExec.Output["processed_prompt"] = result.ProcessedPrompt
	stepExec.Output["success"] = result.Success

	return nil
}

// executeAgentStep delegates to an agent
func (s *WorkflowService) executeAgentStep(ctx context.Context, step *WorkflowStep, exec *WorkflowExecution, stepExec *StepExecution) error {
	prompt := ""
	if step.PromptTemplate != nil {
		prompt = s.interpolate(*step.PromptTemplate, exec.Context)
	}

	stepExec.Output["prompt"] = prompt
	stepExec.Output["agent_id"] = step.TargetAgentID
	// Note: Actual agent execution would be done by the caller
	// This just prepares the step for execution

	return nil
}

// executeToolStep executes a tool
func (s *WorkflowService) executeToolStep(ctx context.Context, step *WorkflowStep, exec *WorkflowExecution, stepExec *StepExecution) error {
	if step.ToolName == nil {
		return fmt.Errorf("tool step missing tool name")
	}

	// Interpolate tool params
	params := make(map[string]any)
	for k, v := range step.ToolParams {
		if str, ok := v.(string); ok {
			params[k] = s.interpolate(str, exec.Context)
		} else {
			params[k] = v
		}
	}

	stepExec.Output["tool"] = *step.ToolName
	stepExec.Output["params"] = params
	// Note: Actual tool execution would be done by the caller

	return nil
}

// executeConditionStep evaluates a condition
func (s *WorkflowService) executeConditionStep(ctx context.Context, step *WorkflowStep, exec *WorkflowExecution, stepExec *StepExecution) error {
	if step.ConditionExpression == nil {
		return fmt.Errorf("condition step missing expression")
	}

	result := s.evaluateCondition(*step.ConditionExpression, exec.Context)
	stepExec.Output["condition"] = *step.ConditionExpression
	stepExec.Output["result"] = result

	if result {
		stepExec.Output["next_step"] = step.OnTrueStep
	} else {
		stepExec.Output["next_step"] = step.OnFalseStep
	}

	return nil
}

// interpolate replaces {{variable}} with values from context
func (s *WorkflowService) interpolate(template string, context map[string]any) string {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := context[key]; ok {
			switch v := val.(type) {
			case string:
				return v
			default:
				jsonBytes, _ := json.Marshal(v)
				return string(jsonBytes)
			}
		}
		return match
	})
}

// evaluateCondition evaluates a simple condition expression
func (s *WorkflowService) evaluateCondition(expr string, context map[string]any) bool {
	// Simple evaluation: {{var}} == value or {{var}} != value
	expr = strings.TrimSpace(expr)

	// Replace variables
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	evaluated := re.ReplaceAllStringFunc(expr, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := context[key]; ok {
			switch v := val.(type) {
			case bool:
				if v {
					return "true"
				}
				return "false"
			case string:
				return v
			default:
				jsonBytes, _ := json.Marshal(v)
				return string(jsonBytes)
			}
		}
		return "null"
	})

	// Simple comparisons
	if strings.Contains(evaluated, "==") {
		parts := strings.SplitN(evaluated, "==", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
		}
	}
	if strings.Contains(evaluated, "!=") {
		parts := strings.SplitN(evaluated, "!=", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
		}
	}

	// Truthy check
	return evaluated == "true" || evaluated == "1"
}
