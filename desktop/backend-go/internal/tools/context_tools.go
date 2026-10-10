package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EntityType represents types of entities that can be fetched on-demand
type EntityType string

const (
	EntityTypeProject    EntityType = "project"
	EntityTypeContext    EntityType = "context"
	EntityTypeTask       EntityType = "task"
	EntityTypeClient     EntityType = "client"
	EntityTypeTeamMember EntityType = "team_member"
	EntityTypeNode       EntityType = "node"
)

// GetEntityContextTool provides on-demand context fetching for AI
// Used when AI needs more details about an entity it only has awareness of (Level 3)
type GetEntityContextTool struct {
	pool   *pgxpool.Pool
	userID string
}

// GetEntityContextInput represents the input for the tool
type GetEntityContextInput struct {
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
}

// GetEntityContextOutput represents the output from the tool
type GetEntityContextOutput struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// NewGetEntityContextTool creates a new GetEntityContextTool
func NewGetEntityContextTool(pool *pgxpool.Pool, userID string) *GetEntityContextTool {
	return &GetEntityContextTool{
		pool:   pool,
		userID: userID,
	}
}

// ToolDefinition returns the tool definition for the AI to understand how to use it
func (t *GetEntityContextTool) ToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "get_entity_context",
		"description": "Retrieve full details for an entity when you need more information than the summary. Use this when the user mentions something you only have awareness of (like 'that other project' or 'the client').",
		"input_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"entity_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"project", "context", "task", "client", "team_member", "node"},
					"description": "The type of entity to fetch details for",
				},
				"entity_id": map[string]interface{}{
					"type":        "string",
					"format":      "uuid",
					"description": "The UUID of the entity to fetch",
				},
			},
			"required": []string{"entity_type", "entity_id"},
		},
	}
}

// Execute retrieves full context for an entity
func (t *GetEntityContextTool) Execute(ctx context.Context, input GetEntityContextInput) GetEntityContextOutput {
	entityID, err := uuid.Parse(input.EntityID)
	if err != nil {
		return GetEntityContextOutput{
			Success: false,
			Error:   "Invalid entity ID format",
		}
	}

	var content string

	switch input.EntityType {
	case EntityTypeProject:
		content, err = t.getProjectContext(ctx, entityID)
	case EntityTypeContext:
		content, err = t.getDocumentContext(ctx, entityID)
	case EntityTypeTask:
		content, err = t.getTaskContext(ctx, entityID)
	case EntityTypeClient:
		content, err = t.getClientContext(ctx, entityID)
	case EntityTypeTeamMember:
		content, err = t.getTeamMemberContext(ctx, entityID)
	case EntityTypeNode:
		content, err = t.getNodeContext(ctx, entityID)
	default:
		return GetEntityContextOutput{
			Success: false,
			Error:   fmt.Sprintf("Unknown entity type: %s", input.EntityType),
		}
	}

	if err != nil {
		return GetEntityContextOutput{
			Success: false,
			Error:   err.Error(),
		}
	}

	return GetEntityContextOutput{
		Success: true,
		Content: content,
	}
}

// getProjectContext retrieves full project details
func (t *GetEntityContextTool) getProjectContext(ctx context.Context, projectID uuid.UUID) (string, error) {
	// Get project
	var name, description, status, priority, clientName string
	query := `SELECT name, COALESCE(description, ''), status, COALESCE(priority, 'MEDIUM'), COALESCE(client_name, '')
	          FROM projects WHERE id = $1 AND user_id = $2`
	err := t.pool.QueryRow(ctx, query, projectID, t.userID).Scan(&name, &description, &status, &priority, &clientName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("project not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch project: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Project: %s\n\n", name))

	if description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", description))
	}

	sb.WriteString(fmt.Sprintf("**Status:** %s\n", status))
	sb.WriteString(fmt.Sprintf("**Priority:** %s\n", priority))

	if clientName != "" {
		sb.WriteString(fmt.Sprintf("**Client:** %s\n", clientName))
	}

	// Get project tasks
	tasksQuery := `SELECT title, status, COALESCE(priority, 'MEDIUM'), COALESCE(description, '')
	               FROM tasks WHERE project_id = $1 AND user_id = $2
	               ORDER BY CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END
	               LIMIT 20`
	rows, err := t.pool.Query(ctx, tasksQuery, projectID, t.userID)
	if err == nil {
		defer rows.Close()
		sb.WriteString("\n### Tasks\n")
		hasTask := false
		for rows.Next() {
			var title, taskStatus, taskPriority, taskDesc string
			if err := rows.Scan(&title, &taskStatus, &taskPriority, &taskDesc); err == nil {
				hasTask = true
				statusIcon := "⬜"
				if taskStatus == "done" {
					statusIcon = "✅"
				}
				sb.WriteString(fmt.Sprintf("- %s %s [%s]\n", statusIcon, title, taskPriority))
			}
		}
		if !hasTask {
			sb.WriteString("- No tasks\n")
		}
	}

	// Get project notes
	notesQuery := `SELECT content FROM project_notes WHERE project_id = $1 ORDER BY created_at DESC LIMIT 10`
	noteRows, err := t.pool.Query(ctx, notesQuery, projectID)
	if err == nil {
		defer noteRows.Close()
		hasNotes := false
		for noteRows.Next() {
			if !hasNotes {
				sb.WriteString("\n### Notes\n")
				hasNotes = true
			}
			var content string
			if err := noteRows.Scan(&content); err == nil {
				sb.WriteString(fmt.Sprintf("- %s\n", content))
			}
		}
	}

	return sb.String(), nil
}

// getDocumentContext retrieves full document/context details
func (t *GetEntityContextTool) getDocumentContext(ctx context.Context, contextID uuid.UUID) (string, error) {
	var name, docType, systemPrompt, content string
	query := `SELECT name, type::text, COALESCE(system_prompt_template, ''), COALESCE(content, '')
	          FROM contexts WHERE id = $1 AND user_id = $2 AND is_archived = false`
	err := t.pool.QueryRow(ctx, query, contextID, t.userID).Scan(&name, &docType, &systemPrompt, &content)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("document not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch document: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Document: %s\n\n", name))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n\n", docType))

	if systemPrompt != "" {
		sb.WriteString(fmt.Sprintf("**System Prompt:** %s\n\n", systemPrompt))
	}

	if content != "" {
		sb.WriteString("### Content\n")
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// getTaskContext retrieves full task details
func (t *GetEntityContextTool) getTaskContext(ctx context.Context, taskID uuid.UUID) (string, error) {
	var title, status, priority, description, dueDate string
	query := `SELECT title, status::text, COALESCE(priority, 'MEDIUM'),
	          COALESCE(description, ''), COALESCE(to_char(due_date, 'YYYY-MM-DD'), '')
	          FROM tasks WHERE id = $1 AND user_id = $2`
	err := t.pool.QueryRow(ctx, query, taskID, t.userID).Scan(&title, &status, &priority, &description, &dueDate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("task not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch task: %w", err)
	}

	var sb strings.Builder
	statusDisplay := "⬜ Pending"
	if status == "done" {
		statusDisplay = "✅ Completed"
	} else if status == "in_progress" {
		statusDisplay = "🔄 In Progress"
	}
	sb.WriteString(fmt.Sprintf("## Task: %s\n\n", title))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", statusDisplay))
	sb.WriteString(fmt.Sprintf("**Priority:** %s\n", priority))

	if dueDate != "" {
		sb.WriteString(fmt.Sprintf("**Due Date:** %s\n", dueDate))
	}

	if description != "" {
		sb.WriteString(fmt.Sprintf("\n**Description:**\n%s\n", description))
	}

	return sb.String(), nil
}

// getClientContext retrieves full client details
func (t *GetEntityContextTool) getClientContext(ctx context.Context, clientID uuid.UUID) (string, error) {
	var name, status, industry, website, notes string
	query := `SELECT name, status::text, COALESCE(industry, ''), COALESCE(website, ''), COALESCE(notes, '')
	          FROM clients WHERE id = $1 AND user_id = $2`
	err := t.pool.QueryRow(ctx, query, clientID, t.userID).Scan(&name, &status, &industry, &website, &notes)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("client not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch client: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Client: %s\n\n", name))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", status))

	if industry != "" {
		sb.WriteString(fmt.Sprintf("**Industry:** %s\n", industry))
	}

	if website != "" {
		sb.WriteString(fmt.Sprintf("**Website:** %s\n", website))
	}

	if notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes:**\n%s\n", notes))
	}

	// Get contacts
	contactsQuery := `SELECT name, COALESCE(role, ''), COALESCE(email, '')
	                  FROM client_contacts WHERE client_id = $1 ORDER BY is_primary DESC, name ASC LIMIT 10`
	rows, err := t.pool.Query(ctx, contactsQuery, clientID)
	if err == nil {
		defer rows.Close()
		hasContacts := false
		for rows.Next() {
			if !hasContacts {
				sb.WriteString("\n### Contacts\n")
				hasContacts = true
			}
			var contactName, role, email string
			if err := rows.Scan(&contactName, &role, &email); err == nil {
				sb.WriteString(fmt.Sprintf("- **%s**", contactName))
				if role != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", role))
				}
				if email != "" {
					sb.WriteString(fmt.Sprintf(" - %s", email))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String(), nil
}

// getTeamMemberContext retrieves full team member details
func (t *GetEntityContextTool) getTeamMemberContext(ctx context.Context, memberID uuid.UUID) (string, error) {
	var name, role, status, email, department, skills, notes string
	query := `SELECT name, role, status::text, COALESCE(email, ''), COALESCE(department, ''),
	          COALESCE(skills, ''), COALESCE(notes, '')
	          FROM team_members WHERE id = $1 AND user_id = $2`
	err := t.pool.QueryRow(ctx, query, memberID, t.userID).Scan(
		&name, &role, &status, &email, &department, &skills, &notes)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("team member not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch team member: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Team Member: %s\n\n", name))
	sb.WriteString(fmt.Sprintf("**Role:** %s\n", role))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", status))

	if email != "" {
		sb.WriteString(fmt.Sprintf("**Email:** %s\n", email))
	}

	if department != "" {
		sb.WriteString(fmt.Sprintf("**Department:** %s\n", department))
	}

	if skills != "" {
		sb.WriteString(fmt.Sprintf("**Skills:** %s\n", skills))
	}

	if notes != "" {
		sb.WriteString(fmt.Sprintf("\n**Notes:**\n%s\n", notes))
	}

	return sb.String(), nil
}

// getNodeContext retrieves full business node details
func (t *GetEntityContextTool) getNodeContext(ctx context.Context, nodeID uuid.UUID) (string, error) {
	var name, nodeType, description string
	var isActive bool
	query := `SELECT name, type::text, COALESCE(description, ''), is_active
	          FROM nodes WHERE id = $1 AND user_id = $2 AND is_archived = false`
	err := t.pool.QueryRow(ctx, query, nodeID, t.userID).Scan(&name, &nodeType, &description, &isActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("business node not found or access denied")
		}
		return "", fmt.Errorf("failed to fetch node: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Business Node: %s\n\n", name))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n", nodeType))

	if isActive {
		sb.WriteString("**Status:** Active\n")
	}

	if description != "" {
		sb.WriteString(fmt.Sprintf("\n**Description:**\n%s\n", description))
	}

	// Note: Projects and Contexts don't currently have a direct node_id relationship.
	// The tiered context system works through other relationships (parent_id, project selection, etc.)
	// Future enhancement: Add node_id to projects and contexts for direct querying

	// Get user's recent projects as context
	projectsQuery := `SELECT name, status::text, COALESCE(description, '')
	                  FROM projects WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 5`
	rows, err := t.pool.Query(ctx, projectsQuery, t.userID)
	if err == nil {
		defer rows.Close()
		hasProjects := false
		for rows.Next() {
			if !hasProjects {
				sb.WriteString("\n### Recent Projects\n")
				hasProjects = true
			}
			var pName, pStatus, pDesc string
			if err := rows.Scan(&pName, &pStatus, &pDesc); err == nil {
				sb.WriteString(fmt.Sprintf("- **%s** [%s]\n", pName, pStatus))
			}
		}
	}

	return sb.String(), nil
}
