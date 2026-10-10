package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OSAInternalHandler handles internal OSA API endpoints for terminal containers.
type OSAInternalHandler struct {
	osaClient *osa.ResilientClient
	pool      *pgxpool.Pool
}

// NewOSAInternalHandler creates a new OSAInternalHandler.
func NewOSAInternalHandler(osaClient *osa.ResilientClient, pool *pgxpool.Pool) *OSAInternalHandler {
	return &OSAInternalHandler{osaClient: osaClient, pool: pool}
}

// Internal OSA API endpoints for terminal containers
// These endpoints accept X-User-ID header for authentication from trusted internal sources

// HandleInternalGenerateApp - POST /api/internal/osa/generate
// Internal endpoint for terminal containers to generate apps
func (h *OSAInternalHandler) HandleInternalGenerateApp(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	// Get user ID from header (set by container environment)
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-User-ID format"})
		return
	}

	var req GenerateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse workspace ID
	var workspaceID uuid.UUID
	if req.WorkspaceID != "" {
		workspaceID, err = uuid.Parse(req.WorkspaceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace_id"})
			return
		}
	} else {
		// Get user's default workspace from BusinessOS workspaces table
		// Query directly from pool since we don't have access to queries here
		var wsID uuid.UUID
		err := h.pool.QueryRow(c.Request.Context(), `
			SELECT w.id
			FROM workspaces w
			JOIN workspace_members wm ON w.id = wm.workspace_id
			WHERE wm.user_id = $1 AND wm.status = 'active'
			ORDER BY w.owner_id = $1 DESC, w.created_at ASC
			LIMIT 1
		`, userIDStr).Scan(&wsID)

		if err != nil {
			// If no workspace exists, create a fallback workspace ID
			slog.Warn("No default workspace found for user, using generated workspace ID",
				"user_id", userID,
				"error", err)
			workspaceID = uuid.New()
		} else {
			workspaceID = wsID
		}
	}

	// Call OSA client
	osaReq := &osa.AppGenerationRequest{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Parameters:  req.Parameters,
	}

	// Default type if not specified
	if osaReq.Type == "" {
		osaReq.Type = "full-stack"
	}

	resp, err := h.osaClient.GenerateApp(c.Request.Context(), osaReq)
	if err != nil {
		slog.Error("[OSA Internal] App generation failed", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "App generation failed",
			"details": err.Error(),
		})
		return
	}

	slog.Info("[OSA Internal] App generation started", "user_id", userID, "app_id", resp.AppID)
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"app_id":       resp.AppID,
		"appId":        resp.AppID, // Include both formats for compatibility
		"workspace_id": resp.WorkspaceID,
		"status":       resp.Status,
		"message":      "App generation started",
	})
}

// HandleInternalGetAppStatus - GET /api/internal/osa/status/:app_id
func (h *OSAInternalHandler) HandleInternalGetAppStatus(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	// Get user ID from header
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-User-ID format"})
		return
	}

	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}

	status, err := h.osaClient.GetAppStatus(c.Request.Context(), appID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "App not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleInternalListWorkspaces - GET /api/internal/osa/workspaces
func (h *OSAInternalHandler) HandleInternalListWorkspaces(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	// Get user ID from header
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid X-User-ID format"})
		return
	}

	workspaces, err := h.osaClient.GetWorkspaces(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch workspaces",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspaces": workspaces,
	})
}

// HandleInternalOSAHealth - GET /api/internal/osa/health
func (h *OSAInternalHandler) HandleInternalOSAHealth(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"enabled": false,
			"status":  "disabled",
		})
		return
	}

	health, err := h.osaClient.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"enabled": true,
			"status":  "unhealthy",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": true,
		"status":  health.Status,
		"version": health.Version,
	})
}
