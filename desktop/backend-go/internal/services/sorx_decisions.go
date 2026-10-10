// Package services provides business logic for BusinessOS.
package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GetPendingDecisions returns all pending decisions for a user.
func (s *SorxService) GetPendingDecisions(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, execution_id, skill_id, step_id,
		       question, options, input_fields, context,
		       priority, status, created_at, expires_at
		FROM pending_decisions
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY
			CASE priority
				WHEN 'urgent' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
			END,
			created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []map[string]interface{}
	for rows.Next() {
		var d struct {
			ID          uuid.UUID
			ExecutionID string
			SkillID     string
			StepID      string
			Question    string
			Options     []string
			InputFields interface{}
			Context     interface{}
			Priority    string
			Status      string
			CreatedAt   time.Time
			ExpiresAt   *time.Time
		}
		if err := rows.Scan(&d.ID, &d.ExecutionID, &d.SkillID, &d.StepID,
			&d.Question, &d.Options, &d.InputFields, &d.Context,
			&d.Priority, &d.Status, &d.CreatedAt, &d.ExpiresAt); err != nil {
			return nil, err
		}

		decisions = append(decisions, map[string]interface{}{
			"id":           d.ID,
			"execution_id": d.ExecutionID,
			"skill_id":     d.SkillID,
			"step_id":      d.StepID,
			"question":     d.Question,
			"options":      d.Options,
			"input_fields": d.InputFields,
			"context":      d.Context,
			"priority":     d.Priority,
			"status":       d.Status,
			"created_at":   d.CreatedAt,
			"expires_at":   d.ExpiresAt,
		})
	}

	return decisions, nil
}

// GetPendingDecision returns a single pending decision.
func (s *SorxService) GetPendingDecision(ctx context.Context, decisionID uuid.UUID) (map[string]interface{}, error) {
	var d struct {
		ID          uuid.UUID
		ExecutionID string
		SkillID     string
		StepID      string
		UserID      string
		Question    string
		Options     []string
		InputFields interface{}
		Context     interface{}
		Priority    string
		Status      string
		Decision    *string
		Inputs      interface{}
		DecidedBy   *string
		DecidedAt   *time.Time
		CreatedAt   time.Time
		ExpiresAt   *time.Time
	}

	err := s.pool.QueryRow(ctx, `
		SELECT id, execution_id, skill_id, step_id, user_id,
		       question, options, input_fields, context,
		       priority, status, decision, decision_inputs,
		       decided_by, decided_at, created_at, expires_at
		FROM pending_decisions
		WHERE id = $1
	`, decisionID).Scan(&d.ID, &d.ExecutionID, &d.SkillID, &d.StepID, &d.UserID,
		&d.Question, &d.Options, &d.InputFields, &d.Context,
		&d.Priority, &d.Status, &d.Decision, &d.Inputs,
		&d.DecidedBy, &d.DecidedAt, &d.CreatedAt, &d.ExpiresAt)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":              d.ID,
		"execution_id":    d.ExecutionID,
		"skill_id":        d.SkillID,
		"step_id":         d.StepID,
		"user_id":         d.UserID,
		"question":        d.Question,
		"options":         d.Options,
		"input_fields":    d.InputFields,
		"context":         d.Context,
		"priority":        d.Priority,
		"status":          d.Status,
		"decision":        d.Decision,
		"decision_inputs": d.Inputs,
		"decided_by":      d.DecidedBy,
		"decided_at":      d.DecidedAt,
		"created_at":      d.CreatedAt,
		"expires_at":      d.ExpiresAt,
	}, nil
}

// RespondToDecision records a human's decision.
func (s *SorxService) RespondToDecision(ctx context.Context, decisionID uuid.UUID, userID string, decision string, inputs map[string]interface{}) error {
	inputsJSON, _ := json.Marshal(inputs)

	_, err := s.pool.Exec(ctx, `
		UPDATE pending_decisions SET
			status = 'decided',
			decision = $2,
			decision_inputs = $3,
			decided_by = $4,
			decided_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, decisionID, decision, inputsJSON, userID)

	return err
}
