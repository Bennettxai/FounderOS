package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MoveTaskTool moves a task to a different status/column
type MoveTaskTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *MoveTaskTool) Name() string { return "move_task" }
func (t *MoveTaskTool) Description() string {
	return "Move a task to a different status (kanban column)"
}
func (t *MoveTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{"type": "string", "description": "Task UUID"},
			"status":  map[string]interface{}{"type": "string", "enum": []string{"todo", "in_progress", "done", "cancelled"}},
		},
		"required": []string{"task_id", "status"},
	}
}
func (t *MoveTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	taskUUID, err := uuid.Parse(params.TaskID)
	if err != nil {
		return "", fmt.Errorf("invalid task ID")
	}

	query := `UPDATE tasks SET status = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`
	_, err = t.pool.Exec(ctx, query, params.Status, taskUUID, t.userID)
	if err != nil {
		return "", fmt.Errorf("failed to move task: %w", err)
	}

	statusIcon := "📋"
	switch params.Status {
	case "in_progress":
		statusIcon = "🔄"
	case "done":
		statusIcon = "✅"
	case "cancelled":
		statusIcon = "❌"
	}

	return fmt.Sprintf("%s Task moved to **%s**", statusIcon, params.Status), nil
}

// AssignTaskTool assigns a task to a team member
type AssignTaskTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *AssignTaskTool) Name() string { return "assign_task" }
func (t *AssignTaskTool) Description() string {
	return "Assign a task to a team member"
}
func (t *AssignTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id":     map[string]interface{}{"type": "string", "description": "Task UUID"},
			"assignee_id": map[string]interface{}{"type": "string", "description": "Team member UUID to assign"},
		},
		"required": []string{"task_id", "assignee_id"},
	}
}
func (t *AssignTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		TaskID     string `json:"task_id"`
		AssigneeID string `json:"assignee_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	taskUUID, err := uuid.Parse(params.TaskID)
	if err != nil {
		return "", fmt.Errorf("invalid task ID")
	}
	assigneeUUID, err := uuid.Parse(params.AssigneeID)
	if err != nil {
		return "", fmt.Errorf("invalid assignee ID")
	}

	query := `UPDATE tasks SET assignee_id = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`
	_, err = t.pool.Exec(ctx, query, assigneeUUID, taskUUID, t.userID)
	if err != nil {
		return "", fmt.Errorf("failed to assign task: %w", err)
	}

	return "👤 Task assigned successfully", nil
}
