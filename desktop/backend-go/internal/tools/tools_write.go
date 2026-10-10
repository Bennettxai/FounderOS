package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ========== WRITE TOOLS ==========

// CreateTaskTool creates a new task
type CreateTaskTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *CreateTaskTool) Name() string { return "create_task" }
func (t *CreateTaskTool) Description() string {
	return "Create a new task. Use this when the user asks to create, add, or make a new task."
}
func (t *CreateTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "The title of the task",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Detailed description of the task (optional)",
			},
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "The project UUID to assign the task to (optional)",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"low", "medium", "high", "critical"},
				"description": "Task priority (default: medium)",
			},
			"due_date": map[string]interface{}{
				"type":        "string",
				"description": "Due date in YYYY-MM-DD format (optional)",
			},
		},
		"required": []string{"title"},
	}
}
func (t *CreateTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ProjectID   string `json:"project_id"`
		Priority    string `json:"priority"`
		DueDate     string `json:"due_date"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if params.Priority == "" {
		params.Priority = "medium"
	}

	// Generate UUID for task
	taskID := uuid.New()

	// Build query dynamically
	query := `INSERT INTO tasks (id, user_id, title, description, status, priority, project_id, due_date, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, 'todo', $5, $6, $7, NOW(), NOW())`

	var projectID interface{} = nil
	if params.ProjectID != "" {
		if parsed, err := uuid.Parse(params.ProjectID); err == nil {
			projectID = parsed
		}
	}

	var dueDate interface{} = nil
	if params.DueDate != "" {
		if parsed, err := time.Parse("2006-01-02", params.DueDate); err == nil {
			dueDate = parsed
		}
	}

	var description interface{} = nil
	if params.Description != "" {
		description = params.Description
	}

	_, err := t.pool.Exec(ctx, query, taskID, t.userID, params.Title, description, params.Priority, projectID, dueDate)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	return fmt.Sprintf("✅ Task created successfully!\n\n**Title:** %s\n**ID:** %s\n**Priority:** %s\n**Status:** todo",
		params.Title, taskID.String(), params.Priority), nil
}

// UpdateTaskTool updates an existing task
type UpdateTaskTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *UpdateTaskTool) Name() string { return "update_task" }
func (t *UpdateTaskTool) Description() string {
	return "Update an existing task's status, priority, or other fields. Use this to mark tasks as done, change priority, etc."
}
func (t *UpdateTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The UUID of the task to update",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"todo", "in_progress", "done"},
				"description": "New status (optional)",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"low", "medium", "high", "critical"},
				"description": "New priority (optional)",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "New title (optional)",
			},
		},
		"required": []string{"task_id"},
	}
}
func (t *UpdateTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		TaskID   string `json:"task_id"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
		Title    string `json:"title"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	taskUUID, err := uuid.Parse(params.TaskID)
	if err != nil {
		return "", fmt.Errorf("invalid task ID")
	}

	// Build dynamic UPDATE query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argNum := 1

	if params.Title != "" {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argNum))
		args = append(args, params.Title)
		argNum++
	}
	if params.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argNum))
		args = append(args, params.Status)
		argNum++
		if params.Status == "done" {
			setClauses = append(setClauses, "completed_at = NOW()")
		}
	}
	if params.Priority != "" {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argNum))
		args = append(args, params.Priority)
		argNum++
	}

	query := fmt.Sprintf(`UPDATE tasks SET %s WHERE id = $%d AND user_id = $%d RETURNING title, status, priority`,
		joinStrings(setClauses, ", "), argNum, argNum+1)
	args = append(args, taskUUID, t.userID)

	var title, status, priority string
	err = t.pool.QueryRow(ctx, query, args...).Scan(&title, &status, &priority)
	if err != nil {
		return "", fmt.Errorf("failed to update task: %w", err)
	}

	statusIcon := "⬜"
	switch status {
	case "done":
		statusIcon = "✅"
	case "in_progress":
		statusIcon = "🔄"
	}

	return fmt.Sprintf("%s Task updated successfully!\n\n**Title:** %s\n**Status:** %s\n**Priority:** %s",
		statusIcon, title, status, priority), nil
}
