package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// MeteringHandler serves the billing/plan metering endpoints:
//
//	GET /api/usage/metering         – current-period usage summary
//	GET /api/usage/metering/limits  – plan limits for the workspace
//	GET /api/usage/metering/history – raw usage_meters rows
//
// It is intentionally separate from the existing UsageHandler (token analytics)
// so neither handler needs to be modified.
type MeteringHandler struct {
	usageService *services.UsageMeteringService
	pool         *pgxpool.Pool
	logger       *slog.Logger
}

// NewMeteringHandler constructs a MeteringHandler.
func NewMeteringHandler(pool *pgxpool.Pool) *MeteringHandler {
	return &MeteringHandler{
		usageService: services.NewUsageMeteringService(pool),
		pool:         pool,
		logger:       slog.Default().With("handler", "metering"),
	}
}

// RegisterMeteringRoutes mounts all metering routes under /usage/metering.
// All routes require authentication.
func RegisterMeteringRoutes(api *gin.RouterGroup, h *MeteringHandler, auth gin.HandlerFunc) {
	g := api.Group("/usage/metering")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", h.GetMeteringSummary)
		g.GET("/limits", h.GetMeteringLimits)
		g.GET("/history", h.GetMeteringHistory)
	}
}

// GetMeteringSummary returns the current-period metering summary for the workspace.
//
// GET /api/usage/metering
// Query param: workspace_id (optional)
func (h *MeteringHandler) GetMeteringSummary(c *gin.Context) {
	workspaceID, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}

	summary, err := h.usageService.GetUsageSummary(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "metering: get summary failed",
			"workspace_id", workspaceID, "error", err)
		utils.RespondInternalError(c, h.logger, "get metering summary", err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetMeteringLimits returns the plan limits for the workspace.
//
// GET /api/usage/metering/limits
// Query param: workspace_id (optional)
func (h *MeteringHandler) GetMeteringLimits(c *gin.Context) {
	workspaceID, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}

	limits, err := h.usageService.GetPlanLimits(c.Request.Context(), workspaceID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "metering: get limits failed",
			"workspace_id", workspaceID, "error", err)
		utils.RespondInternalError(c, h.logger, "get plan limits", err)
		return
	}

	c.JSON(http.StatusOK, limits)
}

// GetMeteringHistory returns raw usage_meters rows for the workspace, newest first.
//
// GET /api/usage/metering/history
// Query params:
//
//	workspace_id (optional)
//	limit        (optional, 1–500, default 90)
func (h *MeteringHandler) GetMeteringHistory(c *gin.Context) {
	workspaceID, ok := h.resolveWorkspace(c)
	if !ok {
		return
	}

	limit := 90
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}

	history, err := h.usageService.GetUsageHistory(c.Request.Context(), workspaceID, limit)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "metering: get history failed",
			"workspace_id", workspaceID, "error", err)
		utils.RespondInternalError(c, h.logger, "get metering history", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": history,
		"count":   len(history),
	})
}

// resolveWorkspace resolves the workspace UUID for the request in priority order:
// 1. ?workspace_id= query param
// 2. workspace_id injected by upstream workspace middleware
// 3. First workspace belonging to the authenticated user
func (h *MeteringHandler) resolveWorkspace(c *gin.Context) (uuid.UUID, bool) {
	if raw := c.Query("workspace_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace_id"})
			return uuid.Nil, false
		}
		// SECURITY: workspace_id is caller-controlled. Verify the authenticated
		// user is an active member before returning that workspace's usage /
		// billing data, else any user could read another tenant's metering.
		user := middleware.GetCurrentUser(c)
		if user == nil {
			utils.RespondUnauthorized(c, h.logger)
			return uuid.Nil, false
		}
		var isMember bool
		if err := h.pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2 AND status = 'active')`,
			id, user.ID,
		).Scan(&isMember); err != nil || !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
			return uuid.Nil, false
		}
		return id, true
	}

	if wsID := middleware.GetWorkspaceID(c); wsID != nil {
		return *wsID, true
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, h.logger)
		return uuid.Nil, false
	}

	const q = `SELECT id FROM workspaces WHERE owner_id = $1 ORDER BY created_at LIMIT 1`
	var id uuid.UUID
	if err := h.pool.QueryRow(c.Request.Context(), q, user.ID).Scan(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no workspace found for user"})
		return uuid.Nil, false
	}
	return id, true
}
