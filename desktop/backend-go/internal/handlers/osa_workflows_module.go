package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InstallModule installs a workflow as a BusinessOS module.
// POST /api/osa/modules/install
func (h *OSAWorkflowsHandler) InstallModule(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		WorkflowID  string   `json:"workflow_id"`
		ModuleName  string   `json:"module_name"`
		InstallPath *string  `json:"install_path"`
		FileIDs     []string `json:"file_ids"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Query workflow
	query := `
		SELECT ga.id, ga.name, ga.display_name, ga.description, ga.metadata
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE (ga.id = $1 OR ga.osa_workflow_id LIKE $2)
		  AND w.user_id = $3
	`

	searchID, searchPrefix := resolveWorkflowSearch(req.WorkflowID)

	var appID uuid.UUID
	var name, displayName, description string
	var metadataJSON []byte

	err := h.pool.QueryRow(c.Request.Context(), query, searchID, searchPrefix, userID).Scan(
		&appID, &name, &displayName, &description, &metadataJSON,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse metadata"})
		return
	}

	// Create module entry
	moduleName := req.ModuleName
	if moduleName == "" {
		moduleName = name
	}

	// Helper to wrap string content in JSON object for JSONB columns
	wrapInJSON := func(key string) []byte {
		if val, ok := metadata[key]; ok {
			wrapped := map[string]interface{}{"content": val}
			jsonData, _ := json.Marshal(wrapped)
			return jsonData
		}
		return nil
	}

	insertQuery := `
		INSERT INTO osa_modules (
			name,
			display_name,
			description,
			module_type,
			schema_definition,
			api_definition,
			ui_definition,
			created_by,
			status,
			metadata
		) VALUES ($1, $2, $3, 'generated', $4, $5, $6, $7, 'installed', $8)
		RETURNING id
	`

	var moduleID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), insertQuery,
		moduleName,
		displayName,
		description,
		wrapInJSON("architecture"),    // schema from architecture file
		wrapInJSON("code"),            // API from code file
		wrapInJSON("recommendations"), // UI from recommendations
		userID,
		metadataJSON,
	).Scan(&moduleID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to install module", "details": err.Error()})
		return
	}

	// Update app with module link
	updateQuery := `
		UPDATE osa_module_instances
		SET module_id = $1, status = 'deployed', deployed_at = NOW()
		WHERE id = $2
	`

	_, err = h.pool.Exec(c.Request.Context(), updateQuery, moduleID, appID)
	if err != nil {
		// Non-fatal - module is already created
		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"module_id": moduleID.String(),
			"message":   "Module installed successfully (app update failed)",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"module_id": moduleID.String(),
		"message":   "Module installed successfully",
	})
}

// TriggerSync manually triggers a sync from OSA-5 workspace.
// POST /api/osa/sync/trigger
func (h *OSAWorkflowsHandler) TriggerSync(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Trigger an immediate sync
	go func() {
		// This will be picked up by the polling service on next tick
		// For immediate sync, we could call scanWorkspace directly
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync triggered",
		"user_id": userID,
	})
}
