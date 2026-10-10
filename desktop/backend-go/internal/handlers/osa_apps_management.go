package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// OSAAppsHandler handles app management endpoints
type OSAAppsHandler struct {
	queries       *sqlc.Queries
	pool          *pgxpool.Pool
	eventBus      *services.BuildEventBus  // For SSE streaming
	queueWorker   *services.OSAQueueWorker // For immediate item notification
	logger        *slog.Logger
	promptBuilder *services.OSAPromptBuilder    // Optional - for template-based prompts
	diffService   *services.SnapshotDiffService // For snapshot diff computation
}

// NewOSAAppsHandler creates a new OSA apps handler
func NewOSAAppsHandler(queries *sqlc.Queries, pool *pgxpool.Pool, logger *slog.Logger) *OSAAppsHandler {
	return &OSAAppsHandler{
		queries: queries,
		pool:    pool,
		logger:  logger,
	}
}

// SetPromptBuilder sets the OSA prompt builder for template-based generation
func (h *OSAAppsHandler) SetPromptBuilder(promptBuilder *services.OSAPromptBuilder) {
	h.promptBuilder = promptBuilder
}

// SetEventBus sets the build event bus for SSE streaming
func (h *OSAAppsHandler) SetEventBus(eventBus *services.BuildEventBus) {
	h.eventBus = eventBus
}

// SetQueueWorker sets the queue worker for immediate item notification
func (h *OSAAppsHandler) SetQueueWorker(worker *services.OSAQueueWorker) {
	h.queueWorker = worker
}

// SetDiffService sets the snapshot diff service
func (h *OSAAppsHandler) SetDiffService(diffService *services.SnapshotDiffService) {
	h.diffService = diffService
}

// AppListResponse represents a paginated list of apps
type AppListResponse struct {
	Apps       []AppDetail `json:"apps"`
	TotalCount int64       `json:"total_count"`
	Limit      int32       `json:"limit"`
	Offset     int32       `json:"offset"`
}

// AppDetail represents detailed app information
type AppDetail struct {
	ID               uuid.UUID      `json:"id"`
	WorkspaceID      uuid.UUID      `json:"workspace_id"`
	ModuleID         *uuid.UUID     `json:"module_id,omitempty"`
	Name             string         `json:"name"`
	DisplayName      string         `json:"display_name"`
	Description      *string        `json:"description,omitempty"`
	OSAWorkflowID    *string        `json:"osa_workflow_id,omitempty"`
	OSASandboxID     *string        `json:"osa_sandbox_id,omitempty"`
	CodeRepository   *string        `json:"code_repository,omitempty"`
	DeploymentURL    *string        `json:"deployment_url,omitempty"`
	Status           *string        `json:"status,omitempty"`
	FilesCreated     *int32         `json:"files_created,omitempty"`
	TestsPassed      *bool          `json:"tests_passed,omitempty"`
	BuildStatus      *string        `json:"build_status,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	ErrorMessage     *string        `json:"error_message,omitempty"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
	GeneratedAt      *string        `json:"generated_at,omitempty"`
	DeployedAt       *string        `json:"deployed_at,omitempty"`
	LastBuildAt      *string        `json:"last_build_at,omitempty"`
	BuildEventsCount int64          `json:"build_events_count"`
}

// UpdateAppMetadataRequest represents the update request
type UpdateAppMetadataRequest struct {
	DisplayName *string         `json:"display_name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    *map[string]any `json:"metadata,omitempty"`
}

// ListApps - GET /api/osa/apps
func (h *OSAAppsHandler) ListApps(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse query parameters
	var workspaceID uuid.UUID
	workspaceIDStr := c.Query("workspace_id")
	var workspaceIDPtr *uuid.UUID
	if workspaceIDStr != "" {
		wID, err := uuid.Parse(workspaceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace_id"})
			return
		}
		workspaceID = wID
		workspaceIDPtr = &workspaceID
	}

	statusFilter := c.Query("status")
	var statusPtr *string
	if statusFilter != "" {
		statusPtr = &statusFilter
	}

	pg := ParsePagination(c)
	limit := pg.Limit
	offset := pg.Offset

	// Convert to pgtype.UUID for sqlc compatibility
	var pgWorkspaceID pgtype.UUID
	if workspaceIDPtr != nil {
		pgWorkspaceID = pgtype.UUID{
			Bytes: *workspaceIDPtr,
			Valid: true,
		}
	}

	statusStr := ""
	if statusPtr != nil {
		statusStr = *statusPtr
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	// Get apps
	apps, err := h.queries.ListOSAModuleInstancesByUser(c.Request.Context(), sqlc.ListOSAModuleInstancesByUserParams{
		UserID:  pgUserID,
		Column2: pgWorkspaceID,
		Column3: statusStr,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		h.logger.Error("failed to list apps", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list apps"})
		return
	}

	// Get total count
	totalCount, err := h.queries.CountOSAModuleInstancesByUser(c.Request.Context(), sqlc.CountOSAModuleInstancesByUserParams{
		UserID:  pgUserID,
		Column2: pgWorkspaceID,
		Column3: statusStr,
	})
	if err != nil {
		h.logger.Error("failed to count apps", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count apps"})
		return
	}

	// Convert to response format
	appDetails := make([]AppDetail, len(apps))
	for i, app := range apps {
		appDetails[i] = convertToAppDetail(app)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(appDetails, totalCount, pg))
}

// GetApp - GET /api/osa/apps/:id
func (h *OSAAppsHandler) GetApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Get app with ownership check
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID2 := pgtype.UUID{Bytes: userID, Valid: true}
	app, err := h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID2,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("app not found or access denied", "app_id", appID, "user_id", userID)
			c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
			return
		}
		h.logger.Error("failed to get app", "error", err, "app_id", appID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get app"})
		return
	}

	// Get build events count
	buildEventsCount, err := h.queries.GetOSAAppLogs(c.Request.Context(), sqlc.GetOSAAppLogsParams{
		ModuleInstanceID: pgAppID,
		Column2:          "", // No filter
		Limit:            100,
		Offset:           0,
	})
	if err != nil {
		h.logger.Warn("failed to get build events count", "error", err, "app_id", appID)
	}

	response := convertToAppDetailFromRow(app)
	response.BuildEventsCount = int64(len(buildEventsCount))

	c.JSON(http.StatusOK, response)
}

// DeleteApp - DELETE /api/osa/apps/:id
func (h *OSAAppsHandler) DeleteApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership first
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("app not found or access denied for deletion", "app_id", appID, "user_id", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	// Delete app (cascade deletes build events)
	err = h.queries.DeleteOSAModuleInstance(c.Request.Context(), pgAppID)
	if err != nil {
		h.logger.Error("failed to delete app", "error", err, "app_id", appID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app"})
		return
	}

	h.logger.Info("app deleted successfully", "app_id", appID, "user_id", userID)
	c.Status(http.StatusNoContent)
}

// UpdateApp - PATCH /api/osa/apps/:id
func (h *OSAAppsHandler) UpdateApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("app not found or access denied for update", "app_id", appID, "user_id", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	// Parse request body
	var req UpdateAppMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Prepare update parameters - sqlc uses string/*string for COALESCE nullable text
	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}

	var description *string
	if req.Description != nil {
		description = req.Description
	}

	var metadata []byte
	if req.Metadata != nil {
		// Marshal metadata to JSON bytes
		metadataBytes, marshalErr := json.Marshal(req.Metadata)
		if marshalErr == nil {
			metadata = metadataBytes
		}
	}

	// Update app
	updatedApp, err := h.queries.UpdateOSAModuleInstanceMetadata(c.Request.Context(), sqlc.UpdateOSAModuleInstanceMetadataParams{
		ID:          pgAppID,
		DisplayName: displayName,
		Description: description,
		Metadata:    metadata,
	})
	if err != nil {
		h.logger.Error("failed to update app", "error", err, "app_id", appID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update app"})
		return
	}

	h.logger.Info("app updated successfully", "app_id", appID, "user_id", userID)
	response := convertToAppDetailFromGeneratedApp(updatedApp)
	c.JSON(http.StatusOK, response)
}

// GetAppLogs - GET /api/osa/apps/:id/logs
func (h *OSAAppsHandler) GetAppLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("app not found or access denied for logs", "app_id", appID, "user_id", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	// Parse query parameters
	eventTypeFilter := c.Query("level") // Using 'level' as per spec

	// Pagination
	limit := int32(50)
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseIntParam(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := int32(0)
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := parseIntParam(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Get logs - Column2 is the event_type filter (empty string = no filter)
	logs, err := h.queries.GetOSAAppLogs(c.Request.Context(), sqlc.GetOSAAppLogsParams{
		ModuleInstanceID: pgAppID,
		Column2:          eventTypeFilter,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		h.logger.Error("failed to get app logs", "error", err, "app_id", appID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"total":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}
