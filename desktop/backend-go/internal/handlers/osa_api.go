package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// OSAAPIHandler handles OSA app generation and status API endpoints.
type OSAAPIHandler struct {
	osaClient *osa.ResilientClient
	pool      *pgxpool.Pool
}

// NewOSAAPIHandler creates a new OSAAPIHandler.
func NewOSAAPIHandler(osaClient *osa.ResilientClient, pool *pgxpool.Pool) *OSAAPIHandler {
	return &OSAAPIHandler{osaClient: osaClient, pool: pool}
}

// GenerateAppRequest matches frontend expectations
type GenerateAppRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description" binding:"required"`
	Type        string                 `json:"type"` // "full-stack", "module", "tool"
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	WorkspaceID string                 `json:"workspace_id"`
}

// HandleGenerateApp - POST /api/osa/generate
func (h *OSAAPIHandler) HandleGenerateApp(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	var req GenerateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse workspace ID
	var workspaceID uuid.UUID
	if req.WorkspaceID != "" {
		var parseErr error
		workspaceID, parseErr = uuid.Parse(req.WorkspaceID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace_id"})
			return
		}
	} else {
		// Get user's default workspace from database
		queries := sqlc.New(h.pool)
		defaultWorkspace, err := queries.GetUserDefaultWorkspace(c.Request.Context(), user.ID)
		if err != nil {
			slog.Error("failed to get user's default workspace",
				"user_id", user.ID,
				"error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get default workspace",
				"details": "Please specify a workspace_id or ensure you have at least one workspace",
			})
			return
		}
		workspaceID = defaultWorkspace.ID.Bytes
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "App generation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"app_id":       resp.AppID,
		"workspace_id": resp.WorkspaceID,
		"status":       resp.Status,
		"message":      "App generation started. Use /api/osa/status/:app_id to track progress.",
	})
}

// HandleGetAppStatus - GET /api/osa/status/:app_id
func (h *OSAAPIHandler) HandleGetAppStatus(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
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

// HandleListWorkspaces - GET /api/osa/workspaces
func (h *OSAAPIHandler) HandleListWorkspaces(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA integration not enabled",
		})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
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

// HandleOSAHealth - GET /api/osa/health
func (h *OSAAPIHandler) HandleOSAHealth(c *gin.Context) {
	if h.osaClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"status":  "disabled",
		})
		return
	}

	health, err := h.osaClient.HealthCheck(c.Request.Context())
	if err != nil {
		slog.Warn("OSA health check failed", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"enabled": true,
			"status":  "unhealthy",
			"error":   err.Error(),
		})
		return
	}

	// Fetch model info directly from OSA (SDK HealthStatus doesn't include model)
	model := ""
	if osaBaseURL := os.Getenv("OSA_BASE_URL"); osaBaseURL != "" {
		type rawHealth struct {
			Model string `json:"model"`
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, "GET", osaBaseURL+"/health", nil)
		if req != nil {
			if resp, err := http.DefaultClient.Do(req); err == nil {
				defer resp.Body.Close()
				var rh rawHealth
				if json.NewDecoder(resp.Body).Decode(&rh) == nil {
					model = rh.Model
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":  true,
		"status":   health.Status,
		"version":  health.Version,
		"model":    model,
		"provider": health.Provider,
	})
}

// HandleOSAModes - GET /api/osa/modes
func (h *OSAAPIHandler) HandleOSAModes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"modes": []gin.H{
			{"mode": "BUILD", "label": "Build", "description": "Create modules and features"},
			{"mode": "ASSIST", "label": "Assist", "description": "Help with tasks"},
			{"mode": "ANALYZE", "label": "Analyze", "description": "Surface insights"},
			{"mode": "EXECUTE", "label": "Execute", "description": "Run actions and workflows"},
			{"mode": "MAINTAIN", "label": "Maintain", "description": "Monitor systems"},
		},
	})
}

// OSAConfigRequest represents a request to change OSA model/provider config.
type OSAConfigRequest struct {
	Provider string `json:"provider" binding:"required"`
	Model    string `json:"model" binding:"required"`
	URL      string `json:"url,omitempty"`
}

// HandleOSAConfig - POST /api/osa/config
// Forwards model/provider config to the running OSA instance.
func (h *OSAAPIHandler) HandleOSAConfig(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OSAConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Forward config to OSA's /api/v1/config endpoint
	osaBaseURL := os.Getenv("OSA_BASE_URL")
	if osaBaseURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OSA not configured"})
		return
	}

	configBody := map[string]string{
		"provider": req.Provider,
		"model":    req.Model,
	}
	if req.URL != "" {
		configBody["url"] = req.URL
	}

	bodyBytes, _ := json.Marshal(configBody)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", osaBaseURL+"/api/v1/config", bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Add shared secret auth
	if secret := os.Getenv("OSA_SHARED_SECRET"); secret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Warn("OSA config update failed", "error", err)
		// Still return success — local state was already updated on frontend
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"applied": false,
			"message": "Config saved locally but OSA instance not reachable",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("OSA config rejected", "status", resp.StatusCode)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"applied": false,
			"message": "Config saved locally but OSA rejected the change",
		})
		return
	}

	slog.Info("OSA config updated", "provider", req.Provider, "model", req.Model)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"applied": true,
	})
}
