package sorx

import (
	"context"
)

func (e *Engine) persistExecution(ctx context.Context, exec *Execution) error {
	_, err := e.pool.Exec(ctx, `
		INSERT INTO skill_executions (
			id, skill_id, user_id, status, params, context, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, exec.ID, exec.SkillID, exec.UserID, exec.Status, exec.Params, exec.Context, exec.StartedAt)
	return err
}

func (e *Engine) updateExecution(ctx context.Context, exec *Execution) error {
	_, err := e.pool.Exec(ctx, `
		UPDATE skill_executions SET
			status = $2,
			current_step = $3,
			step_results = $4,
			result = $5,
			error = $6,
			completed_at = $7
		WHERE id = $1
	`, exec.ID, exec.Status, exec.CurrentStep, exec.StepResults, exec.Result, exec.Error, exec.CompletedAt)
	return err
}
