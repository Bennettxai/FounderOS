package services

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// GetExecution retrieves an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, executionID uuid.UUID) (*WorkflowExecution, error) {
	var exec WorkflowExecution
	var contextJSON, resultJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, workflow_id, user_id, conversation_id, initial_input,
			context, status, current_step_id, result, error_message,
			started_at, completed_at, created_at
		FROM workflow_executions WHERE id = $1
	`, executionID).Scan(
		&exec.ID, &exec.WorkflowID, &exec.UserID, &exec.ConversationID,
		&exec.InitialInput, &contextJSON, &exec.Status, &exec.CurrentStepID,
		&resultJSON, &exec.ErrorMessage, &exec.StartedAt, &exec.CompletedAt,
		&exec.CreatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(contextJSON, &exec.Context)
	json.Unmarshal(resultJSON, &exec.Result)

	return &exec, nil
}

// ListExecutions lists executions for a user
func (s *WorkflowService) ListExecutions(ctx context.Context, userID string, limit, offset int) ([]WorkflowExecution, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workflow_id, user_id, conversation_id, initial_input,
			status, current_step_id, error_message, started_at, completed_at, created_at
		FROM workflow_executions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []WorkflowExecution
	for rows.Next() {
		var exec WorkflowExecution
		err := rows.Scan(
			&exec.ID, &exec.WorkflowID, &exec.UserID, &exec.ConversationID,
			&exec.InitialInput, &exec.Status, &exec.CurrentStepID,
			&exec.ErrorMessage, &exec.StartedAt, &exec.CompletedAt, &exec.CreatedAt)
		if err != nil {
			continue
		}
		executions = append(executions, exec)
	}

	return executions, nil
}
