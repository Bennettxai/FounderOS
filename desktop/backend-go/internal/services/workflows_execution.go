package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ExecuteWorkflow starts a workflow execution
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflow *CommandWorkflow, userID string, input string, conversationID *uuid.UUID) (*WorkflowExecution, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	// Create execution record
	exec := &WorkflowExecution{
		ID:             uuid.New(),
		WorkflowID:     workflow.ID,
		UserID:         userID,
		ConversationID: conversationID,
		InitialInput:   input,
		Context:        make(map[string]any),
		Status:         StatusRunning,
		CreatedAt:      time.Now(),
	}
	now := time.Now()
	exec.StartedAt = &now

	// Store initial input in context
	exec.Context["input"] = input
	exec.Context["workflow_name"] = workflow.Name
	exec.Context["started_at"] = now.Format(time.RFC3339)

	contextJSON, _ := json.Marshal(exec.Context)
	resultJSON, _ := json.Marshal(exec.Result)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_executions (
			id, workflow_id, user_id, conversation_id, initial_input,
			context, status, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, exec.ID, exec.WorkflowID, exec.UserID, exec.ConversationID,
		exec.InitialInput, contextJSON, exec.Status, exec.StartedAt)
	if err != nil {
		return nil, err
	}

	// Execute steps based on mode
	switch workflow.ExecutionMode {
	case ExecutionModeSequential:
		err = s.executeSequential(ctx, workflow, exec)
	case ExecutionModeParallel:
		err = s.executeParallel(ctx, workflow, exec)
	case ExecutionModeSmart:
		err = s.executeSmart(ctx, workflow, exec)
	default:
		err = s.executeSequential(ctx, workflow, exec)
	}

	// Update final status
	completedAt := time.Now()
	exec.CompletedAt = &completedAt

	if err != nil {
		exec.Status = StatusFailed
		errMsg := err.Error()
		exec.ErrorMessage = &errMsg
	} else {
		exec.Status = StatusCompleted
	}

	resultJSON, _ = json.Marshal(exec.Result)
	contextJSON, _ = json.Marshal(exec.Context)

	_, _ = s.pool.Exec(ctx, `
		UPDATE workflow_executions
		SET status = $1, result = $2, context = $3, error_message = $4, completed_at = $5
		WHERE id = $6
	`, exec.Status, resultJSON, contextJSON, exec.ErrorMessage, exec.CompletedAt, exec.ID)

	return exec, err
}

// executeSequential runs steps one after another
func (s *WorkflowService) executeSequential(ctx context.Context, workflow *CommandWorkflow, exec *WorkflowExecution) error {
	for i, step := range workflow.Steps {
		// Update current step
		exec.CurrentStepID = &step.ID
		_, _ = s.pool.Exec(ctx, `
			UPDATE workflow_executions SET current_step_id = $1 WHERE id = $2
		`, step.ID, exec.ID)

		// Execute step
		stepExec, err := s.executeStep(ctx, &step, exec)
		if err != nil {
			slog.Error("Step failed", "step", step.Name, "error", err)

			switch step.OnFailure {
			case FailureActionStop:
				if workflow.StopOnFailure {
					return fmt.Errorf("step %d (%s) failed: %w", i+1, step.Name, err)
				}
			case FailureActionContinue:
				continue
			case FailureActionSkip:
				continue
			case FailureActionRetry:
				// Simple retry logic
				for attempt := 1; attempt <= step.MaxRetries; attempt++ {
					time.Sleep(time.Duration(step.RetryDelaySeconds) * time.Second)
					stepExec, err = s.executeStep(ctx, &step, exec)
					stepExec.AttemptNumber = attempt + 1
					if err == nil {
						break
					}
				}
				if err != nil && workflow.StopOnFailure {
					return fmt.Errorf("step %d (%s) failed after %d retries: %w",
						i+1, step.Name, step.MaxRetries, err)
				}
			}
		}

		// Store step output in context
		if step.OutputKey != nil && *step.OutputKey != "" && stepExec != nil {
			exec.Context[*step.OutputKey] = stepExec.Output
		}
	}

	return nil
}

// executeParallel runs all steps concurrently (respecting dependencies)
func (s *WorkflowService) executeParallel(ctx context.Context, workflow *CommandWorkflow, exec *WorkflowExecution) error {
	// For simple parallel execution without dependencies
	type result struct {
		step *WorkflowStep
		exec *StepExecution
		err  error
	}

	results := make(chan result, len(workflow.Steps))

	for i := range workflow.Steps {
		step := &workflow.Steps[i]
		go func(step *WorkflowStep) {
			stepExec, err := s.executeStep(ctx, step, exec)
			results <- result{step: step, exec: stepExec, err: err}
		}(step)
	}

	var firstError error
	for i := 0; i < len(workflow.Steps); i++ {
		r := <-results
		if r.err != nil && firstError == nil {
			firstError = fmt.Errorf("step %s failed: %w", r.step.Name, r.err)
		}
		if r.step.OutputKey != nil && *r.step.OutputKey != "" && r.exec != nil {
			exec.Context[*r.step.OutputKey] = r.exec.Output
		}
	}

	if firstError != nil && workflow.StopOnFailure {
		return firstError
	}
	return nil
}

// executeSmart analyzes dependencies and runs optimally
func (s *WorkflowService) executeSmart(ctx context.Context, workflow *CommandWorkflow, exec *WorkflowExecution) error {
	// Build dependency graph and execute in waves
	completed := make(map[uuid.UUID]bool)

	for len(completed) < len(workflow.Steps) {
		var wave []*WorkflowStep

		// Find steps that can run (all dependencies met)
		for i := range workflow.Steps {
			step := &workflow.Steps[i]
			if completed[step.ID] {
				continue
			}

			canRun := true
			for _, depID := range step.DependsOn {
				if !completed[depID] {
					canRun = false
					break
				}
			}

			if canRun {
				wave = append(wave, step)
			}
		}

		if len(wave) == 0 {
			return fmt.Errorf("circular dependency detected or no runnable steps")
		}

		// Execute wave in parallel
		type result struct {
			step *WorkflowStep
			exec *StepExecution
			err  error
		}
		results := make(chan result, len(wave))

		for _, step := range wave {
			go func(step *WorkflowStep) {
				stepExec, err := s.executeStep(ctx, step, exec)
				results <- result{step: step, exec: stepExec, err: err}
			}(step)
		}

		for i := 0; i < len(wave); i++ {
			r := <-results
			completed[r.step.ID] = true

			if r.err != nil && workflow.StopOnFailure {
				return fmt.Errorf("step %s failed: %w", r.step.Name, r.err)
			}

			if r.step.OutputKey != nil && *r.step.OutputKey != "" && r.exec != nil {
				exec.Context[*r.step.OutputKey] = r.exec.Output
			}
		}
	}

	return nil
}
