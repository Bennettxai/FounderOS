package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListWorkflows returns all workflows for the current user.
// GET /api/osa/workflows
func (h *OSAWorkflowsHandler) ListWorkflows(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	pg := ParsePagination(c)

	// Count query for accurate pagination metadata
	countQuery := `
		SELECT COUNT(*)
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE w.user_id = $1
	`
	var total int64
	if err := h.pool.QueryRow(c.Request.Context(), countQuery, userID).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count workflows", "details": err.Error()})
		return
	}

	// Paginated query
	query := `
		SELECT
			ga.id,
			ga.name,
			ga.display_name,
			ga.description,
			ga.osa_workflow_id,
			ga.status,
			ga.files_created,
			ga.build_status,
			ga.created_at,
			ga.generated_at,
			ga.deployed_at,
			w.name as workspace_name
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE w.user_id = $1
		ORDER BY ga.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := h.pool.Query(c.Request.Context(), query, userID, pg.Limit, pg.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workflows", "details": err.Error()})
		return
	}
	defer rows.Close()

	type WorkflowListItem struct {
		ID            uuid.UUID  `json:"id"`
		Name          string     `json:"name"`
		DisplayName   string     `json:"display_name"`
		Description   string     `json:"description"`
		WorkflowID    string     `json:"workflow_id"`
		Status        string     `json:"status"`
		FilesCreated  int        `json:"files_created"`
		BuildStatus   *string    `json:"build_status"`
		CreatedAt     time.Time  `json:"created_at"`
		GeneratedAt   *time.Time `json:"generated_at"`
		DeployedAt    *time.Time `json:"deployed_at"`
		WorkspaceName string     `json:"workspace_name"`
	}

	workflows := []WorkflowListItem{}
	for rows.Next() {
		var item WorkflowListItem
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.DisplayName,
			&item.Description,
			&item.WorkflowID,
			&item.Status,
			&item.FilesCreated,
			&item.BuildStatus,
			&item.CreatedAt,
			&item.GeneratedAt,
			&item.DeployedAt,
			&item.WorkspaceName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan workflow", "details": err.Error()})
			return
		}
		workflows = append(workflows, item)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(workflows, total, pg))
}

// GetWorkflow returns details for a specific workflow.
// GET /api/osa/workflows/:id
func (h *OSAWorkflowsHandler) GetWorkflow(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workflowID := c.Param("id")

	// Query workflow details
	query := `
		SELECT
			ga.id,
			ga.name,
			ga.display_name,
			ga.description,
			ga.osa_workflow_id,
			ga.status,
			ga.files_created,
			ga.build_status,
			ga.metadata,
			ga.error_message,
			ga.error_stack,
			ga.created_at,
			ga.generated_at,
			ga.deployed_at,
			w.name as workspace_name,
			w.id as workspace_id
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE (ga.id = $1 OR ga.osa_workflow_id LIKE $2)
		  AND w.user_id = $3
	`

	var (
		id            uuid.UUID
		name          string
		displayName   string
		description   string
		osaWorkflowID string
		status        string
		filesCreated  int
		buildStatus   *string
		metadataJSON  []byte
		errorMessage  *string
		errorStack    *string
		createdAt     time.Time
		generatedAt   *time.Time
		deployedAt    *time.Time
		workspaceName string
		workspaceID   uuid.UUID
	)

	// Try to parse as UUID, otherwise use as workflow ID prefix
	searchID, searchPrefix := resolveWorkflowSearch(workflowID)

	err := h.pool.QueryRow(c.Request.Context(), query, searchID, searchPrefix, userID).Scan(
		&id,
		&name,
		&displayName,
		&description,
		&osaWorkflowID,
		&status,
		&filesCreated,
		&buildStatus,
		&metadataJSON,
		&errorMessage,
		&errorStack,
		&createdAt,
		&generatedAt,
		&deployedAt,
		&workspaceName,
		&workspaceID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workflow", "details": err.Error()})
		}
		return
	}

	// Parse metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		metadata = make(map[string]interface{})
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             id,
		"name":           name,
		"display_name":   displayName,
		"description":    description,
		"workflow_id":    osaWorkflowID,
		"status":         status,
		"files_created":  filesCreated,
		"build_status":   buildStatus,
		"metadata":       metadata,
		"error_message":  errorMessage,
		"error_stack":    errorStack,
		"created_at":     createdAt,
		"generated_at":   generatedAt,
		"deployed_at":    deployedAt,
		"workspace_name": workspaceName,
		"workspace_id":   workspaceID,
	})
}
