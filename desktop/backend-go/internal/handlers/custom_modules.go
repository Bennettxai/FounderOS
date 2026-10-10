package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// CustomModulesHandler handles custom module operations
type CustomModulesHandler struct {
	pool    *pgxpool.Pool
	service *services.CustomModuleService
	logger  *slog.Logger
}

// NewCustomModulesHandler creates a new custom modules handler
func NewCustomModulesHandler(pool *pgxpool.Pool, logger *slog.Logger) *CustomModulesHandler {
	return &CustomModulesHandler{
		pool:    pool,
		service: services.NewCustomModuleService(pool, logger),
		logger:  logger,
	}
}

// CreateModule creates a new custom module
// POST /api/modules
func (h *CustomModulesHandler) CreateModule(c *gin.Context) {
	var req services.CreateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	userID, workspaceID, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in CreateModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	module, err := h.service.CreateModule(c.Request.Context(), workspaceID, userID, req)
	if err != nil {
		RespondInternalErr(c, "create module", err)
		return
	}

	c.JSON(http.StatusCreated, module)
}

// GetModule retrieves a single module
// GET /api/modules/:id
func (h *CustomModulesHandler) GetModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	module, err := h.service.GetModule(c.Request.Context(), moduleID)
	if err != nil {
		h.logger.Error("Failed to get module", "error", err, "module_id", moduleID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// SECURITY (CRIT-11): Enforce ownership — only allow access to public modules
	// or modules belonging to a workspace the authenticated user is a member of.
	if !module.IsPublic {
		userID, workspaceID, authErr := h.getStringAuthContext(c)
		if authErr != nil {
			h.logger.Error("auth context missing in GetModule", "error", authErr)
			RespondUnauthorizedErr(c, "Authentication required")
			return
		}
		_ = userID // authContext validates the user; workspace membership is the relevant check
		if module.WorkspaceID != workspaceID {
			// Verify the user is a member of the module's workspace via DB
			var memberExists bool
			dbErr := h.pool.QueryRow(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2 AND status = 'active')`,
				module.WorkspaceID, userID,
			).Scan(&memberExists)
			if dbErr != nil || !memberExists {
				c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, module)
}

// ListModules lists modules in a workspace with pagination and filters
// GET /api/modules?page=1&page_size=20&category=utility
func (h *CustomModulesHandler) ListModules(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	// Get workspace ID: try context, query param, then DB lookup
	workspaceID := middleware.GetWorkspaceID(c)
	if workspaceID == nil {
		if wsStr := c.Query("workspace_id"); wsStr != "" {
			if parsed, err := uuid.Parse(wsStr); err == nil {
				// SECURITY: workspace_id here is caller-controlled. Verify the
				// user is a member before listing that workspace's modules,
				// else any user could read another workspace's private modules.
				if !h.userInWorkspace(c, parsed, user.ID) {
					c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
					return
				}
				workspaceID = &parsed
			}
		}
	}
	if workspaceID == nil {
		var wsID uuid.UUID
		err := h.pool.QueryRow(c.Request.Context(),
			`SELECT workspace_id FROM workspace_members WHERE user_id = $1 LIMIT 1`,
			user.ID,
		).Scan(&wsID)
		if err != nil {
			h.logger.Error("no workspace found for user", "user_id", user.ID, "error", err)
			RespondUnauthorizedErr(c, "No workspace found")
			return
		}
		workspaceID = &wsID
	}

	pg := ParsePagination(c)

	modules, err := h.service.ListModules(c.Request.Context(), *workspaceID, int(pg.Limit), int(pg.Offset))
	if err != nil {
		h.logger.Error("Failed to list modules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list modules"})
		return
	}

	total := int64(pg.Offset) + int64(len(modules))
	if int64(len(modules)) == int64(pg.Limit) {
		total++
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(modules, total, pg))
}

// UpdateModule updates an existing module
// PUT /api/modules/:id
func (h *CustomModulesHandler) UpdateModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	var req services.UpdateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	userID, _, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in UpdateModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	module, err := h.service.UpdateModule(c.Request.Context(), moduleID, userID, req)
	if err != nil {
		RespondInternalErr(c, "update module", err)
		return
	}

	c.JSON(http.StatusOK, module)
}

// DeleteModule deletes a module
// DELETE /api/modules/:id
func (h *CustomModulesHandler) DeleteModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	userID, _, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in DeleteModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	err = h.service.DeleteModule(c.Request.Context(), moduleID, userID)
	if err != nil {
		RespondInternalErr(c, "delete module", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Module deleted successfully"})
}

// PublishModule publishes a module to the marketplace
// POST /api/modules/:id/publish
func (h *CustomModulesHandler) PublishModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	userID, _, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in PublishModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	err = h.service.PublishModule(c.Request.Context(), moduleID, userID)
	if err != nil {
		RespondInternalErr(c, "publish module", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Module published successfully"})
}

// InstallModule installs a module to the current workspace
// POST /api/modules/:id/install
func (h *CustomModulesHandler) InstallModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	userID, workspaceID, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in InstallModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	// Get module to install
	module, err := h.service.GetModule(c.Request.Context(), moduleID)
	if err != nil {
		RespondNotFoundErr(c, "module")
		return
	}

	// Create installation record
	_, err = h.service.CreateInstallation(c.Request.Context(), moduleID, workspaceID, userID, module.Version)
	if err != nil {
		RespondInternalErr(c, "install module", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Module installed successfully"})
}

// SetShareScope sets the share_scope of a module ('workspace' or 'organization').
// Only the module owner (created_by) may change the scope.
// PATCH /api/modules/:id/share-scope
func (h *CustomModulesHandler) SetShareScope(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	var req struct {
		ShareScope string `json:"share_scope" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "share_scope is required")
		return
	}
	if req.ShareScope != "workspace" && req.ShareScope != "organization" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "share_scope must be 'workspace' or 'organization'"})
		return
	}

	userID, _, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in SetShareScope", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	// Reuse UpdateModule which already enforces ownership
	scope := req.ShareScope
	module, err := h.service.UpdateModule(c.Request.Context(), moduleID, userID, services.UpdateModuleRequest{
		ShareScope: &scope,
	})
	if err != nil {
		RespondInternalErr(c, "set share scope", err)
		return
	}

	c.JSON(http.StatusOK, module)
}

// ShareModule shares a module with another user/workspace
// POST /api/modules/:id/share
func (h *CustomModulesHandler) ShareModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	var req struct {
		ShareWithUserID      *string `json:"share_with_user_id"`
		ShareWithWorkspaceID *string `json:"share_with_workspace_id"`
		ShareWithEmail       *string `json:"share_with_email"`
		CanView              bool    `json:"can_view"`
		CanInstall           bool    `json:"can_install"`
		CanModify            bool    `json:"can_modify"`
		CanReshare           bool    `json:"can_reshare"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, _, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in ShareModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	// Parse UUIDs if provided
	var shareWithUserID, shareWithWorkspaceID *uuid.UUID
	if req.ShareWithUserID != nil {
		uid, err := uuid.Parse(*req.ShareWithUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		shareWithUserID = &uid
	}
	if req.ShareWithWorkspaceID != nil {
		wid, err := uuid.Parse(*req.ShareWithWorkspaceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID"})
			return
		}
		shareWithWorkspaceID = &wid
	}

	share, err := h.service.CreateShare(c.Request.Context(), services.CreateShareRequest{
		ModuleID:              moduleID,
		SharedWithUserID:      shareWithUserID,
		SharedWithWorkspaceID: shareWithWorkspaceID,
		SharedWithEmail:       req.ShareWithEmail,
		CanView:               req.CanView,
		CanInstall:            req.CanInstall,
		CanModify:             req.CanModify,
		CanReshare:            req.CanReshare,
		SharedBy:              userID,
	})

	if err != nil {
		RespondInternalErr(c, "share module", err)
		return
	}

	c.JSON(http.StatusCreated, share)
}

// ListInstalledModules lists all installed modules in the workspace
// GET /api/modules/installed
func (h *CustomModulesHandler) ListInstalledModules(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	workspaceID, err := h.getWorkspaceID(c, user.ID)
	if err != nil {
		h.logger.Error("auth context missing in ListInstalledModules", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	installations, err := h.service.ListInstallations(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.Error("Failed to list installations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list installed modules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"installations": installations,
		"count":         len(installations),
	})
}

// GetModuleStats returns stats about a module (install count, etc.)
// GET /api/modules/stats?id=uuid
func (h *CustomModulesHandler) GetModuleStats(c *gin.Context) {
	moduleIDStr := c.Query("id")
	if moduleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module ID required"})
		return
	}

	moduleID, err := uuid.Parse(moduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	module, err := h.service.GetModule(c.Request.Context(), moduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// SECURITY: don't expose stats for a private module unless the caller is a
	// member of its workspace (mirrors GetModule / CRIT-11).
	if !module.IsPublic {
		userID, _, authErr := h.getStringAuthContext(c)
		if authErr != nil || !h.userInWorkspace(c, module.WorkspaceID, userID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"module_id":     module.ID,
		"install_count": module.InstallCount,
		"star_count":    module.StarCount,
		"version":       module.Version,
		"is_published":  module.IsPublished,
	})
}

// GetPopularModules returns popular public modules
// GET /api/modules/popular?limit=10
func (h *CustomModulesHandler) GetPopularModules(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	modules, err := h.service.SearchModules(c.Request.Context(), "", limit, 0)
	if err != nil {
		h.logger.Error("Failed to get popular modules", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular modules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"modules": modules,
		"count":   len(modules),
	})
}

// ExportModule exports a module as a ZIP file
// GET /api/modules/export/:id
func (h *CustomModulesHandler) ExportModule(c *gin.Context) {
	moduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	module, err := h.service.GetModule(c.Request.Context(), moduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// SECURITY (CRIT-11): Enforce ownership — exports are only allowed for modules
	// the authenticated user has access to (own workspace or public modules).
	if !module.IsPublic {
		userID, workspaceID, authErr := h.getStringAuthContext(c)
		if authErr != nil {
			h.logger.Error("auth context missing in ExportModule", "error", authErr)
			RespondUnauthorizedErr(c, "Authentication required")
			return
		}
		_ = userID
		if module.WorkspaceID != workspaceID {
			var memberExists bool
			dbErr := h.pool.QueryRow(c.Request.Context(),
				`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2 AND status = 'active')`,
				module.WorkspaceID, userID,
			).Scan(&memberExists)
			if dbErr != nil || !memberExists {
				c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
				return
			}
		}
	}

	// Create ZIP file in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add manifest.json
	manifestFile, err := zipWriter.Create("manifest.json")
	if err != nil {
		h.logger.Error("Failed to create manifest file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export module"})
		return
	}
	manifestJSON, _ := json.MarshalIndent(module.Manifest, "", "  ")
	manifestFile.Write(manifestJSON)

	// Add config.json
	configFile, err := zipWriter.Create("config.json")
	if err != nil {
		h.logger.Error("Failed to create config file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export module"})
		return
	}
	configJSON, _ := json.MarshalIndent(module.Config, "", "  ")
	configFile.Write(configJSON)

	// Add metadata.json
	metadataFile, err := zipWriter.Create("metadata.json")
	if err != nil {
		h.logger.Error("Failed to create metadata file", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export module"})
		return
	}
	metadata := map[string]interface{}{
		"name":        module.Name,
		"slug":        module.Slug,
		"description": module.Description,
		"category":    module.Category,
		"version":     module.Version,
		"icon":        module.Icon,
		"tags":        module.Tags,
		"keywords":    module.Keywords,
	}
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	metadataFile.Write(metadataJSON)

	zipWriter.Close()

	// Set headers and return ZIP
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", module.Slug))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// ImportModule imports a module from a ZIP file
// POST /api/modules/import
func (h *CustomModulesHandler) ImportModule(c *gin.Context) {
	userID, workspaceID, err := h.getAuthContext(c)
	if err != nil {
		h.logger.Error("auth context missing in ImportModule", "error", err)
		RespondUnauthorizedErr(c, "Authentication required")
		return
	}

	// Parse multipart form
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Read file into memory
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Open ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(fileData), int64(len(fileData)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ZIP file"})
		return
	}

	// Extract files
	var manifestData, configData, metadataData []byte

	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		switch file.Name {
		case "manifest.json":
			manifestData = data
		case "config.json":
			configData = data
		case "metadata.json":
			metadataData = data
		}
	}

	if manifestData == nil || metadataData == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ZIP: missing required files"})
		return
	}

	// Parse JSON
	var manifest, config, metadata map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid manifest JSON"})
		return
	}
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata JSON"})
		return
	}
	if configData != nil {
		json.Unmarshal(configData, &config)
	}

	// Create module
	req := services.CreateModuleRequest{
		Name:        metadata["name"].(string),
		Description: metadata["description"].(string),
		Category:    metadata["category"].(string),
		Manifest:    manifest,
		Config:      config,
		Icon:        metadata["icon"].(string),
		Tags:        []string{},
		Keywords:    []string{},
	}

	// Parse tags and keywords if present
	if tags, ok := metadata["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				req.Tags = append(req.Tags, tagStr)
			}
		}
	}
	if keywords, ok := metadata["keywords"].([]interface{}); ok {
		for _, keyword := range keywords {
			if kwStr, ok := keyword.(string); ok {
				req.Keywords = append(req.Keywords, kwStr)
			}
		}
	}

	module, err := h.service.CreateModule(c.Request.Context(), workspaceID, userID, req)
	if err != nil {
		RespondInternalErr(c, "import module", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Module imported successfully",
		"module":  module,
	})
}

func (h *CustomModulesHandler) getWorkspaceID(c *gin.Context, userID string) (uuid.UUID, error) {
	workspaceIDVal, exists := c.Get("workspace_id")
	if exists {
		switch v := workspaceIDVal.(type) {
		case string:
			workspaceID, err := uuid.Parse(v)
			if err != nil {
				return uuid.Nil, fmt.Errorf("invalid workspace ID")
			}
			return workspaceID, nil
		case uuid.UUID:
			return v, nil
		default:
			return uuid.Nil, fmt.Errorf("invalid workspace ID type")
		}
	}

	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr != "" {
		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid workspace ID")
		}
		// SECURITY: workspace_id from the query param is caller-controlled.
		// Verify the user is an active member before trusting it, else a user
		// could read/mutate another workspace's modules by passing its id.
		if !h.userInWorkspace(c, workspaceID, userID) {
			return uuid.Nil, fmt.Errorf("forbidden: not a member of workspace")
		}
		return workspaceID, nil
	}

	var workspaceID uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT workspace_id FROM workspace_members WHERE user_id = $1 LIMIT 1`,
		userID,
	).Scan(&workspaceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("workspace ID not found")
	}

	return workspaceID, nil
}

// userInWorkspace reports whether the user is an active member of the workspace.
// Used to gate any code path where workspace_id comes from a caller-controlled
// source (query param) rather than trusted middleware context.
func (h *CustomModulesHandler) userInWorkspace(c *gin.Context, workspaceID uuid.UUID, userID string) bool {
	var exists bool
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2 AND status = 'active')`,
		workspaceID, userID,
	).Scan(&exists)
	return err == nil && exists
}

// getStringAuthContext resolves Better Auth's opaque string user ID together
// with the active workspace. Better Auth IDs are not UUIDs and must remain
// strings across authorization and audit boundaries.
func (h *CustomModulesHandler) getStringAuthContext(c *gin.Context) (userID string, workspaceID uuid.UUID, err error) {
	user := middleware.GetCurrentUser(c)
	if user != nil {
		userID = user.ID
	} else {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			return "", uuid.Nil, fmt.Errorf("unauthorized: user ID not found")
		}
		switch v := userIDVal.(type) {
		case string:
			userID = v
		case uuid.UUID:
			userID = v.String()
		default:
			return "", uuid.Nil, fmt.Errorf("invalid user ID type")
		}
	}

	if userID == "" {
		return "", uuid.Nil, fmt.Errorf("unauthorized: empty user ID")
	}

	workspaceIDVal, exists := c.Get("workspace_id")
	if exists {
		switch v := workspaceIDVal.(type) {
		case string:
			workspaceID, err = uuid.Parse(v)
		case uuid.UUID:
			workspaceID = v
		default:
			err = fmt.Errorf("invalid workspace ID type")
		}
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("invalid workspace ID")
		}
		return userID, workspaceID, nil
	}

	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr != "" {
		workspaceID, err = uuid.Parse(workspaceIDStr)
		if err != nil {
			return "", uuid.Nil, fmt.Errorf("invalid workspace ID")
		}
		if !h.userInWorkspace(c, workspaceID, userID) {
			return "", uuid.Nil, fmt.Errorf("forbidden: not a member of workspace")
		}
		return userID, workspaceID, nil
	}

	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT workspace_id FROM workspace_members WHERE user_id = $1 AND status = 'active' LIMIT 1`,
		userID,
	).Scan(&workspaceID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("workspace ID not found")
	}

	return userID, workspaceID, nil
}

// Helper function to get auth context
func (h *CustomModulesHandler) getAuthContext(c *gin.Context) (userID uuid.UUID, workspaceID uuid.UUID, err error) {
	// Try middleware-standard "user" context key first (GetCurrentUser)
	user := middleware.GetCurrentUser(c)
	if user != nil {
		userID, err = uuid.Parse(user.ID)
		if err != nil {
			return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user ID")
		}
	} else {
		// Fallback: try "user_id" string key
		userIDVal, exists := c.Get("user_id")
		if !exists {
			return uuid.Nil, uuid.Nil, fmt.Errorf("unauthorized: user ID not found")
		}
		switch v := userIDVal.(type) {
		case string:
			userID, err = uuid.Parse(v)
			if err != nil {
				return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user ID")
			}
		case uuid.UUID:
			userID = v
		default:
			return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user ID type")
		}
	}

	// Try middleware-standard "workspace_id" context key
	workspaceIDVal, exists := c.Get("workspace_id")
	if exists {
		switch v := workspaceIDVal.(type) {
		case string:
			workspaceID, err = uuid.Parse(v)
			if err != nil {
				return uuid.Nil, uuid.Nil, fmt.Errorf("invalid workspace ID")
			}
		case uuid.UUID:
			workspaceID = v
		default:
			return uuid.Nil, uuid.Nil, fmt.Errorf("invalid workspace ID type")
		}
	} else {
		// Fallback: query param
		workspaceIDStr := c.Query("workspace_id")
		if workspaceIDStr != "" {
			workspaceID, err = uuid.Parse(workspaceIDStr)
			if err != nil {
				return uuid.Nil, uuid.Nil, fmt.Errorf("invalid workspace ID")
			}
			// SECURITY: caller-controlled workspace_id — verify membership.
			if !h.userInWorkspace(c, workspaceID, userID.String()) {
				return uuid.Nil, uuid.Nil, fmt.Errorf("forbidden: not a member of workspace")
			}
		} else {
			// Last resort: look up user's first workspace from DB
			var wsID uuid.UUID
			lookupErr := h.pool.QueryRow(c.Request.Context(),
				`SELECT workspace_id FROM workspace_members WHERE user_id = $1 LIMIT 1`,
				userID.String(),
			).Scan(&wsID)
			if lookupErr != nil {
				return uuid.Nil, uuid.Nil, fmt.Errorf("workspace ID not found")
			}
			workspaceID = wsID
		}
	}

	return userID, workspaceID, nil
}
