package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/services"
)

// GetWorkflowFiles returns all files for a workflow.
// GET /api/osa/workflows/:id/files
func (h *OSAWorkflowsHandler) GetWorkflowFiles(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workflowID := c.Param("id")

	// Query workflow metadata (contains all files)
	query := `
		SELECT ga.id, ga.name, ga.osa_workflow_id, ga.metadata, ga.created_at, ga.updated_at
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE (ga.id = $1 OR ga.osa_workflow_id LIKE $2)
		  AND w.user_id = $3
	`

	searchID, searchPrefix := resolveWorkflowSearch(workflowID)

	var appID uuid.UUID
	var appName string
	var osaWorkflowID string
	var metadataJSON []byte
	var createdAt, updatedAt time.Time

	err := h.pool.QueryRow(c.Request.Context(), query, searchID, searchPrefix, userID).Scan(
		&appID, &appName, &osaWorkflowID, &metadataJSON, &createdAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files", "details": err.Error()})
		}
		return
	}

	// Parse metadata to extract file contents
	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse metadata"})
		return
	}

	// Structure file response with deterministic IDs
	files := []map[string]interface{}{}

	for _, fileType := range workflowFileTypes {
		if content, ok := metadata[fileType].(string); ok && content != "" {
			// Special handling for "code" type - parse multi-file bundle
			if fileType == "code" {
				bundledFiles := services.ParseFileBundle(content)
				for _, bundledFile := range bundledFiles {
					// Generate deterministic UUID for each bundled file
					fileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":code:"+bundledFile.Path))
					ext := services.GetFileExtension(bundledFile.Path)
					fileCategory := services.CategorizeFileType(bundledFile.Path)

					files = append(files, map[string]interface{}{
						"id":         fileID.String(),
						"name":       bundledFile.Path,
						"type":       fileCategory,
						"size":       len(bundledFile.Content),
						"language":   services.GetLanguageFromExtension(ext),
						"created_at": createdAt,
						"updated_at": updatedAt,
					})
				}
			} else {
				// Regular metadata files (analysis, architecture, etc.)
				fileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":"+fileType))
				fileName := fileType + ".md"

				files = append(files, map[string]interface{}{
					"id":         fileID.String(),
					"name":       fileName,
					"type":       "documentation",
					"size":       len(content),
					"language":   "markdown",
					"created_at": createdAt,
					"updated_at": updatedAt,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"workflow_id": osaWorkflowID,
		"files":       files,
		"count":       len(files),
	})
}

// GetFileContent returns the content of a specific file by type name.
// GET /api/osa/workflows/:id/files/:type
func (h *OSAWorkflowsHandler) GetFileContent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	workflowID := c.Param("id")
	fileType := c.Param("type")

	// Validate file type
	validTypes := map[string]bool{
		"analysis": true, "architecture": true, "code": true, "quality": true,
		"deployment": true, "monitoring": true, "strategy": true, "recommendations": true,
	}
	if !validTypes[fileType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	// Query workflow metadata
	query := `
		SELECT ga.metadata
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE (ga.id = $1 OR ga.osa_workflow_id LIKE $2)
		  AND w.user_id = $3
	`

	searchID, searchPrefix := resolveWorkflowSearch(workflowID)

	var metadataJSON []byte
	err := h.pool.QueryRow(c.Request.Context(), query, searchID, searchPrefix, userID).Scan(&metadataJSON)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file", "details": err.Error()})
		}
		return
	}

	// Parse metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse metadata"})
		return
	}

	// Extract file content
	content, ok := metadata[fileType].(string)
	if !ok || content == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"type":    fileType,
		"content": content,
		"size":    len(content),
	})
}

// GetFileContentByID returns file content by deterministic file UUID.
// GET /api/osa/files/:id/content
func (h *OSAWorkflowsHandler) GetFileContentByID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	fileID := c.Param("id")
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	// Query all workflows to find the one containing this file
	query := `
		SELECT ga.id, ga.name, ga.metadata, ga.created_at, ga.updated_at
		FROM osa_module_instances ga
		JOIN osa_workspaces w ON ga.workspace_id = w.id
		WHERE w.user_id = $1
	`

	rows, err := h.pool.Query(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search files", "details": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var appID uuid.UUID
		var appName string
		var metadataJSON []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&appID, &appName, &metadataJSON, &createdAt, &updatedAt); err != nil {
			continue
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			continue
		}

		// Check each file type to see if it matches the requested file ID
		for _, ft := range workflowFileTypes {
			if content, ok := metadata[ft].(string); ok && content != "" {
				// For "code" type, check bundled files
				if ft == "code" {
					bundledFiles := services.ParseFileBundle(content)
					for _, bundledFile := range bundledFiles {
						expectedFileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":code:"+bundledFile.Path))
						if expectedFileID == fileUUID {
							ext := services.GetFileExtension(bundledFile.Path)
							fileCategory := services.CategorizeFileType(bundledFile.Path)

							c.JSON(http.StatusOK, gin.H{
								"content": bundledFile.Content,
								"file": map[string]interface{}{
									"id":         fileID,
									"name":       bundledFile.Path,
									"type":       fileCategory,
									"size":       len(bundledFile.Content),
									"language":   services.GetLanguageFromExtension(ext),
									"created_at": createdAt,
									"updated_at": updatedAt,
								},
							})
							return
						}
					}
				} else {
					// Regular metadata files
					expectedFileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":"+ft))
					if expectedFileID == fileUUID {
						c.JSON(http.StatusOK, gin.H{
							"content": content,
							"file": map[string]interface{}{
								"id":         fileID,
								"name":       ft + ".md",
								"type":       "documentation",
								"size":       len(content),
								"language":   "markdown",
								"created_at": createdAt,
								"updated_at": updatedAt,
							},
						})
						return
					}
				}
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
}
