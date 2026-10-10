package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// SandboxDeploymentServiceInterface defines the methods used by SandboxHandler.
// Using an interface allows test mocks to be injected without a live Docker daemon.
type SandboxDeploymentServiceInterface interface {
	Deploy(ctx context.Context, req services.SandboxDeploymentRequest) (*services.SandboxInfo, error)
	Stop(ctx context.Context, appID uuid.UUID) error
	Restart(ctx context.Context, appID uuid.UUID) (*services.SandboxInfo, error)
	Remove(ctx context.Context, appID uuid.UUID) error
	GetSandboxInfo(ctx context.Context, appID uuid.UUID) (*services.SandboxInfo, error)
	GetSandboxLogs(ctx context.Context, appID uuid.UUID, tail string, since string) (string, error)
	ListUserSandboxes(ctx context.Context, userID uuid.UUID) ([]*services.SandboxInfo, error)
	GetStats() map[string]interface{}
}

// SandboxHandler handles sandbox-related HTTP requests
type SandboxHandler struct {
	deploymentService SandboxDeploymentServiceInterface
	logger            *slog.Logger
	pool              *pgxpool.Pool
}

// respondError sends a JSON error response.
// Internal error details are logged server-side but never exposed to clients.
func respondError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		slog.Error("request error", "status", status, "message", message, "error", err)
	}
	c.JSON(status, gin.H{"error": message})
}

// NewSandboxHandler creates a new sandbox handler
func NewSandboxHandler(deploymentService *services.SandboxDeploymentService, logger *slog.Logger, pool *pgxpool.Pool) *SandboxHandler {
	return &SandboxHandler{
		deploymentService: deploymentService,
		logger:            logger.With("handler", "sandbox"),
		pool:              pool,
	}
}

// DeploySandbox handles POST /api/v1/sandbox/deploy
// @Summary Deploy a new sandbox
// @Description Deploys a new sandbox container for an app
// @Tags sandbox
// @Accept json
// @Produce json
// @Param request body services.SandboxDeploymentRequest true "Deployment request"
// @Success 201 {object} services.SandboxInfo "Sandbox deployed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Sandbox already running"
// @Failure 500 {object} map[string]string "Deployment failed"
// @Router /api/v1/sandbox/deploy [post]
func (h *SandboxHandler) DeploySandbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req services.SandboxDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid deployment request", "error", err)
		respondError(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Set user ID from authenticated user
	userUUID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "user_id", user.ID, "error", err)
		respondError(c, http.StatusInternalServerError, "invalid user ID", err)
		return
	}
	req.UserID = userUUID

	// Validate required fields
	if req.AppID == uuid.Nil {
		respondError(c, http.StatusBadRequest, "app_id is required", nil)
		return
	}
	if req.AppName == "" {
		respondError(c, http.StatusBadRequest, "app_name is required", nil)
		return
	}

	// Auto-detect image from app_type if not provided
	if req.Image == "" {
		appInfo, err := h.deploymentService.GetSandboxInfo(c.Request.Context(), req.AppID)
		if err == nil && appInfo != nil && appInfo.AppType != "" {
			req.Image = getDefaultImageForAppType(appInfo.AppType)
			h.logger.Info("auto-detected image from app_type", "app_id", req.AppID, "app_type", appInfo.AppType, "image", req.Image)
		}
		// If still empty, use default node image
		if req.Image == "" {
			req.Image = "node:18-alpine"
			h.logger.Info("using default image (no app_type found)", "app_id", req.AppID, "image", req.Image)
		}
	}

	h.logger.Info("deploying sandbox",
		"app_id", req.AppID,
		"app_name", req.AppName,
		"user_id", req.UserID,
		"image", req.Image)

	info, err := h.deploymentService.Deploy(c.Request.Context(), req)
	if err != nil {
		switch err {
		case services.ErrSandboxAlreadyRunning:
			respondError(c, http.StatusConflict, "sandbox already running", err)
		case services.ErrMaxSandboxesReached:
			respondError(c, http.StatusForbidden, "maximum number of sandboxes reached", err)
		case services.ErrInvalidAppID:
			respondError(c, http.StatusBadRequest, "invalid app ID", err)
		case services.ErrDeploymentFailed:
			respondError(c, http.StatusInternalServerError, "deployment failed", err)
		default:
			h.logger.Error("failed to deploy sandbox", "error", err)
			respondError(c, http.StatusInternalServerError, "deployment failed", err)
		}
		return
	}

	c.JSON(http.StatusCreated, info)
}

// StopSandbox handles POST /api/v1/sandbox/:app_id/stop
// @Summary Stop a sandbox
// @Description Stops a running sandbox container
// @Tags sandbox
// @Produce json
// @Param app_id path string true "App ID (UUID)"
// @Success 200 {object} map[string]string "Sandbox stopped successfully"
// @Failure 400 {object} map[string]string "Invalid app ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Sandbox not found"
// @Failure 500 {object} map[string]string "Failed to stop sandbox"
// @Router /api/v1/sandbox/{app_id}/stop [post]
func (h *SandboxHandler) StopSandbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid app ID", err)
		return
	}

	h.logger.Info("stopping sandbox", "app_id", appID, "user_id", user.ID)

	if err := h.deploymentService.Stop(c.Request.Context(), appID); err != nil {
		switch err {
		case services.ErrSandboxNotFound:
			respondError(c, http.StatusNotFound, "sandbox not found", err)
		case services.ErrSandboxNotRunning:
			respondError(c, http.StatusBadRequest, "sandbox is not running", err)
		default:
			h.logger.Error("failed to stop sandbox", "app_id", appID, "error", err)
			respondError(c, http.StatusInternalServerError, "failed to stop sandbox", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sandbox stopped successfully"})
}

// RestartSandbox handles POST /api/v1/sandbox/:app_id/restart
// @Summary Restart a sandbox
// @Description Restarts a sandbox container
// @Tags sandbox
// @Produce json
// @Param app_id path string true "App ID (UUID)"
// @Success 200 {object} services.SandboxInfo "Sandbox restarted successfully"
// @Failure 400 {object} map[string]string "Invalid app ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Sandbox not found"
// @Failure 500 {object} map[string]string "Failed to restart sandbox"
// @Router /api/v1/sandbox/{app_id}/restart [post]
func (h *SandboxHandler) RestartSandbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid app ID", err)
		return
	}

	h.logger.Info("restarting sandbox", "app_id", appID, "user_id", user.ID)

	info, err := h.deploymentService.Restart(c.Request.Context(), appID)
	if err != nil {
		switch err {
		case services.ErrSandboxNotFound:
			respondError(c, http.StatusNotFound, "sandbox not found", err)
		default:
			h.logger.Error("failed to restart sandbox", "app_id", appID, "error", err)
			respondError(c, http.StatusInternalServerError, "failed to restart sandbox", err)
		}
		return
	}

	c.JSON(http.StatusOK, info)
}

// RemoveSandbox handles DELETE /api/v1/sandbox/:app_id
// @Summary Remove a sandbox
// @Description Removes a sandbox container and releases resources
// @Tags sandbox
// @Produce json
// @Param app_id path string true "App ID (UUID)"
// @Success 200 {object} map[string]string "Sandbox removed successfully"
// @Failure 400 {object} map[string]string "Invalid app ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to remove sandbox"
// @Router /api/v1/sandbox/{app_id} [delete]
func (h *SandboxHandler) RemoveSandbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid app ID", err)
		return
	}

	h.logger.Info("removing sandbox", "app_id", appID, "user_id", user.ID)

	if err := h.deploymentService.Remove(c.Request.Context(), appID); err != nil {
		h.logger.Error("failed to remove sandbox", "app_id", appID, "error", err)
		respondError(c, http.StatusInternalServerError, "failed to remove sandbox", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sandbox removed successfully"})
}

// GetSandboxInfo handles GET /api/v1/sandbox/:app_id
// @Summary Get sandbox info
// @Description Retrieves information about a sandbox
// @Tags sandbox
// @Produce json
// @Param app_id path string true "App ID (UUID)"
// @Success 200 {object} services.SandboxInfo "Sandbox information"
// @Failure 400 {object} map[string]string "Invalid app ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Sandbox not found"
// @Router /api/v1/sandbox/{app_id} [get]
func (h *SandboxHandler) GetSandboxInfo(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid app ID", err)
		return
	}

	info, err := h.deploymentService.GetSandboxInfo(c.Request.Context(), appID)
	if err != nil {
		switch err {
		case services.ErrSandboxNotFound:
			respondError(c, http.StatusNotFound, "sandbox not found", err)
		default:
			h.logger.Error("failed to get sandbox info", "app_id", appID, "error", err)
			respondError(c, http.StatusInternalServerError, "failed to get sandbox info", err)
		}
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetSandboxLogs handles GET /api/v1/sandbox/:app_id/logs
// @Summary Get sandbox logs
// @Description Retrieves logs from a sandbox container
// @Tags sandbox
// @Produce json
// @Param app_id path string true "App ID (UUID)"
// @Param tail query string false "Number of lines to tail (default: all)"
// @Param since query string false "Show logs since timestamp (RFC3339) or duration (e.g., 10m, 1h)"
// @Success 200 {object} map[string]string "Sandbox logs"
// @Failure 400 {object} map[string]string "Invalid app ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Sandbox not found or not running"
// @Failure 500 {object} map[string]string "Failed to get logs"
// @Router /api/v1/sandbox/{app_id}/logs [get]
func (h *SandboxHandler) GetSandboxLogs(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	appIDStr := c.Param("app_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid app ID", err)
		return
	}

	tail := c.DefaultQuery("tail", "all")
	since := c.DefaultQuery("since", "")

	h.logger.Debug("getting sandbox logs", "app_id", appID, "tail", tail, "since", since)

	logs, err := h.deploymentService.GetSandboxLogs(c.Request.Context(), appID, tail, since)
	if err != nil {
		switch err {
		case services.ErrSandboxNotFound:
			respondError(c, http.StatusNotFound, "sandbox not found", err)
		case services.ErrSandboxNotRunning:
			respondError(c, http.StatusNotFound, "sandbox container not running", err)
		default:
			h.logger.Error("failed to get sandbox logs", "app_id", appID, "error", err)
			respondError(c, http.StatusInternalServerError, "failed to get logs", err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"app_id": appID,
		"logs":   logs,
	})
}

// ListUserSandboxes handles GET /api/v1/sandboxes
// @Summary List user's sandboxes
// @Description Lists all sandboxes for the authenticated user
// @Tags sandbox
// @Produce json
// @Success 200 {array} services.SandboxInfo "List of sandboxes"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to list sandboxes"
// @Router /api/v1/sandboxes [get]
func (h *SandboxHandler) ListUserSandboxes(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	userUUID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "user_id", user.ID, "error", err)
		respondError(c, http.StatusInternalServerError, "invalid user ID", err)
		return
	}

	sandboxes, err := h.deploymentService.ListUserSandboxes(c.Request.Context(), userUUID)
	if err != nil {
		h.logger.Error("failed to list user sandboxes", "user_id", user.ID, "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list sandboxes", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sandboxes": sandboxes,
		"count":     len(sandboxes),
	})
}

// GetSandboxStats handles GET /api/v1/sandbox/stats
// @Summary Get sandbox statistics (admin)
// @Description Retrieves deployment service statistics
// @Tags sandbox
// @Produce json
// @Success 200 {object} map[string]interface{} "Sandbox statistics"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/v1/sandbox/stats [get]
func (h *SandboxHandler) GetSandboxStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	// Require workspace owner or admin (hierarchy_level <= 2: owner=1, admin=2).
	// This endpoint exposes system-wide deployment stats and must not be
	// accessible to regular workspace members.
	if h.pool != nil {
		var hierarchyLevel int
		err := h.pool.QueryRow(c.Request.Context(), `
			SELECT COALESCE(MIN(wr.hierarchy_level), 999)
			FROM workspace_members wm
			JOIN workspace_roles wr ON wr.name = wm.role_name AND wr.workspace_id = wm.workspace_id
			WHERE wm.user_id = $1 AND wm.status = 'active'
		`, user.ID).Scan(&hierarchyLevel)
		if err != nil || hierarchyLevel > 2 {
			respondError(c, http.StatusForbidden, "forbidden: admin or owner role required", err)
			return
		}
	}

	stats := h.deploymentService.GetStats()
	c.JSON(http.StatusOK, stats)
}

// SandboxInfoResponse is a response wrapper for sandbox info (used for JSON serialization)
type SandboxInfoResponse struct {
	AppID        string  `json:"app_id"`
	AppName      string  `json:"app_name"`
	UserID       string  `json:"user_id"`
	ContainerID  string  `json:"container_id"`
	Status       string  `json:"status"`
	Port         int     `json:"port"`
	URL          string  `json:"url"`
	Image        string  `json:"image"`
	CreatedAt    string  `json:"created_at"`
	StartedAt    *string `json:"started_at,omitempty"`
	HealthStatus string  `json:"health_status"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// SandboxDeploymentRequestPayload is the request payload for deploying a sandbox
type SandboxDeploymentRequestPayload struct {
	AppID         string            `json:"app_id" binding:"required"`
	AppName       string            `json:"app_name" binding:"required"`
	Image         string            `json:"image" binding:"required"`
	ContainerPort int               `json:"container_port"`
	WorkspacePath string            `json:"workspace_path"`
	Environment   map[string]string `json:"environment"`
	StartCommand  []string          `json:"start_command"`
	WorkingDir    string            `json:"working_dir"`
	MemoryLimit   int64             `json:"memory_limit"`
	CPUQuota      int64             `json:"cpu_quota"`
}

// ValidateSandboxDeploymentRequest validates the deployment request
func ValidateSandboxDeploymentRequest(req *SandboxDeploymentRequestPayload) error {
	if req.AppID == "" {
		return fmt.Errorf("app_id is required")
	}
	if req.AppName == "" {
		return fmt.Errorf("app_name is required")
	}
	if req.Image == "" {
		return fmt.Errorf("image is required")
	}
	if req.ContainerPort < 0 || req.ContainerPort > 65535 {
		return fmt.Errorf("container_port must be between 0 and 65535")
	}
	return nil
}

// ParseQueryInt parses an integer query parameter with a default value
func ParseQueryInt(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.DefaultQuery(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getDefaultImageForAppType returns the default Docker image for a given app type.
// This allows frontend to omit the image field and have it auto-detected.
func getDefaultImageForAppType(appType string) string {
	switch appType {
	case "svelte", "sveltekit":
		return "node:18-alpine"
	case "react", "nextjs", "next":
		return "node:18-alpine"
	case "vue", "nuxt":
		return "node:18-alpine"
	case "go", "golang":
		return "golang:1.21-alpine"
	case "python", "flask", "django", "fastapi":
		return "python:3.11-slim"
	case "rust":
		return "rust:1.74-alpine"
	case "static", "html":
		return "nginx:alpine"
	default:
		return "node:18-alpine" // Default to Node.js
	}
}
