package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ========== READ TOOLS ==========

// GetProjectTool retrieves project details
type GetProjectTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *GetProjectTool) Name() string { return "get_project" }
func (t *GetProjectTool) Description() string {
	return "Get detailed information about a specific project including its tasks, notes, and status"
}
func (t *GetProjectTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "The UUID of the project",
			},
		},
		"required": []string{"project_id"},
	}
}
func (t *GetProjectTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	tool := NewGetEntityContextTool(t.pool, t.userID)
	result := tool.Execute(ctx, GetEntityContextInput{
		EntityType: EntityTypeProject,
		EntityID:   params.ProjectID,
	})

	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}
	return result.Content, nil
}

// GetTaskTool retrieves task details
type GetTaskTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *GetTaskTool) Name() string { return "get_task" }
func (t *GetTaskTool) Description() string {
	return "Get detailed information about a specific task"
}
func (t *GetTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "The UUID of the task",
			},
		},
		"required": []string{"task_id"},
	}
}
func (t *GetTaskTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	tool := NewGetEntityContextTool(t.pool, t.userID)
	result := tool.Execute(ctx, GetEntityContextInput{
		EntityType: EntityTypeTask,
		EntityID:   params.TaskID,
	})

	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}
	return result.Content, nil
}

// GetClientTool retrieves client details
type GetClientTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *GetClientTool) Name() string { return "get_client" }
func (t *GetClientTool) Description() string {
	return "Get detailed information about a client including contacts and interaction history"
}
func (t *GetClientTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"client_id": map[string]interface{}{
				"type":        "string",
				"description": "The UUID of the client",
			},
		},
		"required": []string{"client_id"},
	}
}
func (t *GetClientTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	tool := NewGetEntityContextTool(t.pool, t.userID)
	result := tool.Execute(ctx, GetEntityContextInput{
		EntityType: EntityTypeClient,
		EntityID:   params.ClientID,
	})

	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}
	return result.Content, nil
}

// ListTasksTool lists tasks with filters
type ListTasksTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *ListTasksTool) Name() string { return "list_tasks" }
func (t *ListTasksTool) Description() string {
	return "List tasks with optional filters by project, status, or priority. Use this to see what tasks exist."
}
func (t *ListTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "Filter by project UUID (optional)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"pending", "in_progress", "done"},
				"description": "Filter by status (optional)",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"low", "medium", "high", "critical"},
				"description": "Filter by priority (optional)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of tasks to return (default 20)",
			},
		},
	}
}
func (t *ListTasksTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID string `json:"project_id"`
		Status    string `json:"status"`
		Priority  string `json:"priority"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}

	query := `SELECT t.id, t.title, t.status::text, COALESCE(t.priority, 'medium'),
	          COALESCE(p.name, 'No Project') as project_name
	          FROM tasks t
	          LEFT JOIN projects p ON t.project_id = p.id
	          WHERE t.user_id = $1`
	args := []interface{}{t.userID}
	argNum := 2

	if params.ProjectID != "" {
		if projectUUID, err := uuid.Parse(params.ProjectID); err == nil {
			query += fmt.Sprintf(" AND t.project_id = $%d", argNum)
			args = append(args, projectUUID)
			argNum++
		}
	}
	if params.Status != "" {
		query += fmt.Sprintf(" AND t.status = $%d", argNum)
		args = append(args, params.Status)
		argNum++
	}
	if params.Priority != "" {
		query += fmt.Sprintf(" AND t.priority = $%d", argNum)
		args = append(args, params.Priority)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY CASE t.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END LIMIT $%d", argNum)
	args = append(args, params.Limit)

	rows, err := t.pool.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var result string
	result = "## Tasks\n\n"
	count := 0
	for rows.Next() {
		var id uuid.UUID
		var title, status, priority, projectName string
		if err := rows.Scan(&id, &title, &status, &priority, &projectName); err == nil {
			statusIcon := "⬜"
			switch status {
			case "done":
				statusIcon = "✅"
			case "in_progress":
				statusIcon = "🔄"
			}
			result += fmt.Sprintf("- %s **%s** [%s] - %s (ID: %s)\n", statusIcon, title, priority, projectName, id.String())
			count++
		}
	}

	if count == 0 {
		result += "No tasks found matching the criteria.\n"
	}

	return result, nil
}

// ListProjectsTool lists projects
type ListProjectsTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *ListProjectsTool) Name() string { return "list_projects" }
func (t *ListProjectsTool) Description() string {
	return "List all projects with their status. Use this to see available projects."
}
func (t *ListProjectsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"active", "completed", "on_hold", "cancelled"},
				"description": "Filter by status (optional)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of projects to return (default 20)",
			},
		},
	}
}
func (t *ListProjectsTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Status string `json:"status"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	if params.Limit <= 0 {
		params.Limit = 20
	}

	query := `SELECT id, name, status::text, COALESCE(description, ''),
	          (SELECT COUNT(*) FROM tasks WHERE project_id = projects.id AND status != 'done') as pending_tasks
	          FROM projects WHERE user_id = $1`
	args := []interface{}{t.userID}

	if params.Status != "" {
		query += " AND status = $2"
		args = append(args, params.Status)
	}

	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT %d", params.Limit)

	rows, err := t.pool.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var result string
	result = "## Projects\n\n"
	count := 0
	for rows.Next() {
		var id uuid.UUID
		var name, status, description string
		var pendingTasks int
		if err := rows.Scan(&id, &name, &status, &description, &pendingTasks); err == nil {
			result += fmt.Sprintf("- **%s** [%s] - %d pending tasks (ID: %s)\n", name, status, pendingTasks, id.String())
			if description != "" {
				result += fmt.Sprintf("  %s\n", description)
			}
			count++
		}
	}

	if count == 0 {
		result += "No projects found.\n"
	}

	return result, nil
}

// SearchDocumentsTool searches documents/contexts
type SearchDocumentsTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *SearchDocumentsTool) Name() string { return "search_documents" }
func (t *SearchDocumentsTool) Description() string {
	return "Search through documents and contexts by name or content"
}
func (t *SearchDocumentsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query to find in document names or content",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"document", "profile", "knowledge_base"},
				"description": "Filter by document type (optional)",
			},
		},
		"required": []string{"query"},
	}
}
func (t *SearchDocumentsTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Query string `json:"query"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}

	query := `SELECT id, name, type::text, COALESCE(LEFT(content, 200), '') as preview
	          FROM contexts
	          WHERE user_id = $1 AND is_archived = false
	          AND (name ILIKE $2 OR content ILIKE $2)`
	args := []interface{}{t.userID, "%" + params.Query + "%"}

	if params.Type != "" {
		query += " AND type = $3"
		args = append(args, params.Type)
	}

	query += " ORDER BY updated_at DESC LIMIT 10"

	rows, err := t.pool.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var result string
	result = fmt.Sprintf("## Search Results for \"%s\"\n\n", params.Query)
	count := 0
	for rows.Next() {
		var id uuid.UUID
		var name, docType, preview string
		if err := rows.Scan(&id, &name, &docType, &preview); err == nil {
			result += fmt.Sprintf("- **%s** [%s] (ID: %s)\n", name, docType, id.String())
			if preview != "" {
				result += fmt.Sprintf("  Preview: %s...\n", preview)
			}
			count++
		}
	}

	if count == 0 {
		result += "No documents found matching the query.\n"
	}

	return result, nil
}

// GetTeamCapacityTool gets team workload and capacity
type GetTeamCapacityTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *GetTeamCapacityTool) Name() string { return "get_team_capacity" }
func (t *GetTeamCapacityTool) Description() string {
	return "Get team members' current workload and capacity"
}
func (t *GetTeamCapacityTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}
func (t *GetTeamCapacityTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	// Get team members with their task counts
	query := `
		SELECT
			tm.id, tm.name, tm.role,
			COUNT(CASE WHEN t.status = 'in_progress' THEN 1 END) as active_tasks,
			COUNT(CASE WHEN t.status = 'todo' THEN 1 END) as pending_tasks
		FROM team_members tm
		LEFT JOIN tasks t ON t.assignee_id = tm.id AND t.status IN ('todo', 'in_progress')
		WHERE tm.user_id = $1
		GROUP BY tm.id, tm.name, tm.role
		ORDER BY active_tasks DESC`

	rows, err := t.pool.Query(ctx, query, t.userID)
	if err != nil {
		return "", fmt.Errorf("failed to get team capacity: %w", err)
	}
	defer rows.Close()

	var result string
	result = "## Team Capacity\n\n"
	result += "| Member | Role | Active | Pending |\n"
	result += "|--------|------|--------|--------|\n"

	for rows.Next() {
		var id uuid.UUID
		var name, role string
		var activeTasks, pendingTasks int
		if err := rows.Scan(&id, &name, &role, &activeTasks, &pendingTasks); err != nil {
			continue
		}
		result += fmt.Sprintf("| %s | %s | %d | %d |\n", name, role, activeTasks, pendingTasks)
	}

	return result, nil
}

// QueryMetricsTool queries business metrics for analysis
type QueryMetricsTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *QueryMetricsTool) Name() string { return "query_metrics" }
func (t *QueryMetricsTool) Description() string {
	return "Query business metrics like task completion rates, project progress, client pipeline"
}
func (t *QueryMetricsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"metric_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"tasks", "projects", "clients", "overview"},
				"description": "Type of metrics to query",
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"week", "month", "quarter", "year"},
				"description": "Time range for metrics",
			},
		},
		"required": []string{"metric_type"},
	}
}
func (t *QueryMetricsTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		MetricType string `json:"metric_type"`
		TimeRange  string `json:"time_range"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", err
	}
	if params.TimeRange == "" {
		params.TimeRange = "month"
	}

	var interval string
	switch params.TimeRange {
	case "week":
		interval = "7 days"
	case "month":
		interval = "30 days"
	case "quarter":
		interval = "90 days"
	case "year":
		interval = "365 days"
	default:
		interval = "30 days"
	}

	var result string

	switch params.MetricType {
	case "tasks":
		query := `
			SELECT
				COUNT(*) FILTER (WHERE status = 'done') as completed,
				COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
				COUNT(*) FILTER (WHERE status = 'todo') as todo,
				COUNT(*) as total
			FROM tasks
			WHERE user_id = $1 AND created_at > NOW() - $2::interval`
		var completed, inProgress, todo, total int
		err := t.pool.QueryRow(ctx, query, t.userID, interval).Scan(&completed, &inProgress, &todo, &total)
		if err != nil {
			return "", err
		}
		completionRate := 0.0
		if total > 0 {
			completionRate = float64(completed) / float64(total) * 100
		}
		result = fmt.Sprintf("## Task Metrics (%s)\n\n", params.TimeRange)
		result += fmt.Sprintf("- **Total Tasks:** %d\n", total)
		result += fmt.Sprintf("- **Completed:** %d (%.1f%%)\n", completed, completionRate)
		result += fmt.Sprintf("- **In Progress:** %d\n", inProgress)
		result += fmt.Sprintf("- **Todo:** %d\n", todo)

	case "projects":
		query := `
			SELECT status, COUNT(*)
			FROM projects
			WHERE user_id = $1
			GROUP BY status`
		rows, err := t.pool.Query(ctx, query, t.userID)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		result = "## Project Metrics\n\n"
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			result += fmt.Sprintf("- **%s:** %d\n", status, count)
		}

	case "clients":
		query := `
			SELECT pipeline_stage, COUNT(*)
			FROM clients
			WHERE user_id = $1
			GROUP BY pipeline_stage
			ORDER BY COUNT(*) DESC`
		rows, err := t.pool.Query(ctx, query, t.userID)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		result = "## Client Pipeline\n\n"
		for rows.Next() {
			var stage string
			var count int
			rows.Scan(&stage, &count)
			result += fmt.Sprintf("- **%s:** %d\n", stage, count)
		}

	case "overview":
		result = "## Business Overview\n\n"
		// Tasks
		var taskCount int
		t.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tasks WHERE user_id = $1", t.userID).Scan(&taskCount)
		result += fmt.Sprintf("- **Total Tasks:** %d\n", taskCount)
		// Projects
		var projectCount int
		t.pool.QueryRow(ctx, "SELECT COUNT(*) FROM projects WHERE user_id = $1", t.userID).Scan(&projectCount)
		result += fmt.Sprintf("- **Total Projects:** %d\n", projectCount)
		// Clients
		var clientCount int
		t.pool.QueryRow(ctx, "SELECT COUNT(*) FROM clients WHERE user_id = $1", t.userID).Scan(&clientCount)
		result += fmt.Sprintf("- **Total Clients:** %d\n", clientCount)
	}

	return result, nil
}
