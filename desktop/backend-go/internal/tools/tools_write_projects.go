package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateNoteTool creates a project note
type CreateNoteTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *CreateNoteTool) Name() string { return "create_note" }
func (t *CreateNoteTool) Description() string {
	return "Create a note for a project. Use this to add notes, comments, or updates to a project."
}
func (t *CreateNoteTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "The project UUID to add the note to",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The note content",
			},
		},
		"required": []string{"project_id", "content"},
	}
}
func (t *CreateNoteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	projectUUID, err := uuid.Parse(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("invalid project ID")
	}

	query := `INSERT INTO project_notes (project_id, content, created_at) VALUES ($1, $2, NOW()) RETURNING id`
	var noteID uuid.UUID
	err = t.pool.QueryRow(ctx, query, projectUUID, params.Content).Scan(&noteID)
	if err != nil {
		return "", fmt.Errorf("failed to create note: %w", err)
	}

	return fmt.Sprintf("📝 Note added to project!\n\n**Content:** %s", params.Content), nil
}

// CreateProjectTool creates a new project
type CreateProjectTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *CreateProjectTool) Name() string { return "create_project" }
func (t *CreateProjectTool) Description() string {
	return "Create a new project with name, description, and optional settings"
}
func (t *CreateProjectTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":        map[string]interface{}{"type": "string", "description": "Project name"},
			"description": map[string]interface{}{"type": "string", "description": "Project description"},
			"status":      map[string]interface{}{"type": "string", "enum": []string{"ACTIVE", "PAUSED", "COMPLETED", "ARCHIVED"}, "description": "Project status (default: ACTIVE)"},
			"priority":    map[string]interface{}{"type": "string", "enum": []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}, "description": "Project priority (default: MEDIUM)"},
		},
		"required": []string{"name"},
	}
}
func (t *CreateProjectTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}
	if params.Status == "" {
		params.Status = "ACTIVE"
	}
	if params.Priority == "" {
		params.Priority = "MEDIUM"
	}

	// Generate UUID for project
	projectID := uuid.New()
	query := `INSERT INTO projects (id, user_id, name, description, status, priority, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`
	_, err := t.pool.Exec(ctx, query, projectID, t.userID, params.Name, params.Description, params.Status, params.Priority)
	if err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	return fmt.Sprintf("✅ Project created!\n\n**Name:** %s\n**ID:** %s\n**Status:** %s\n**Priority:** %s", params.Name, projectID, params.Status, params.Priority), nil
}

// UpdateProjectTool updates an existing project
type UpdateProjectTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *UpdateProjectTool) Name() string { return "update_project" }
func (t *UpdateProjectTool) Description() string {
	return "Update an existing project's name, description, or status"
}
func (t *UpdateProjectTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id":  map[string]interface{}{"type": "string", "description": "Project UUID"},
			"name":        map[string]interface{}{"type": "string", "description": "New project name"},
			"description": map[string]interface{}{"type": "string", "description": "New description"},
			"status":      map[string]interface{}{"type": "string", "enum": []string{"active", "planning", "on_hold", "completed"}},
		},
		"required": []string{"project_id"},
	}
}
func (t *UpdateProjectTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID   string `json:"project_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	projectUUID, err := uuid.Parse(params.ProjectID)
	if err != nil {
		return "", fmt.Errorf("invalid project ID")
	}

	// Build dynamic update
	updates := []string{}
	args := []interface{}{projectUUID, t.userID}
	argNum := 3

	if params.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, params.Name)
		argNum++
	}
	if params.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, params.Description)
		argNum++
	}
	if params.Status != "" {
		updates = append(updates, fmt.Sprintf("status = $%d", argNum))
		args = append(args, params.Status)
		argNum++
	}

	if len(updates) == 0 {
		return "No updates provided", nil
	}

	query := fmt.Sprintf("UPDATE projects SET %s, updated_at = NOW() WHERE id = $1 AND user_id = $2", joinStrings(updates, ", "))
	_, err = t.pool.Exec(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to update project: %w", err)
	}

	return "✅ Project updated successfully", nil
}

// BulkCreateTasksTool creates multiple tasks at once
type BulkCreateTasksTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *BulkCreateTasksTool) Name() string { return "bulk_create_tasks" }
func (t *BulkCreateTasksTool) Description() string {
	return "Create multiple tasks at once for a project"
}
func (t *BulkCreateTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string", "description": "Project UUID"},
			"tasks": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title":       map[string]interface{}{"type": "string"},
						"description": map[string]interface{}{"type": "string"},
						"priority":    map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high", "critical"}},
					},
					"required": []string{"title"},
				},
			},
		},
		"required": []string{"tasks"},
	}
}
func (t *BulkCreateTasksTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Tasks     []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	var projectUUID *uuid.UUID
	if params.ProjectID != "" {
		parsed, err := uuid.Parse(params.ProjectID)
		if err == nil {
			projectUUID = &parsed
		}
	}

	created := 0
	for _, task := range params.Tasks {
		priority := task.Priority
		if priority == "" {
			priority = "medium"
		}

		// Generate UUID for each task
		taskID := uuid.New()
		query := `INSERT INTO tasks (id, user_id, project_id, title, description, priority, status, created_at, updated_at)
		          VALUES ($1, $2, $3, $4, $5, $6, 'todo', NOW(), NOW())`
		_, err := t.pool.Exec(ctx, query, taskID, t.userID, projectUUID, task.Title, task.Description, priority)
		if err == nil {
			created++
		}
	}

	return fmt.Sprintf("✅ Created %d/%d tasks", created, len(params.Tasks)), nil
}
