package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// UpdateDashboardLayout updates only the layout of a dashboard (used by agent)
func (h *DashboardCRUDHandler) UpdateDashboardLayout(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	var req struct {
		Layout json.RawMessage `json:"layout" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.UpdateDashboardLayout(c.Request.Context(), sqlc.UpdateDashboardLayoutParams{
		ID:     pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID: user.ID,
		Layout: req.Layout,
	})
	if err != nil {
		slog.Default().Error("Failed to update dashboard layout", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "update dashboard layout", err)
		return
	}

	// Broadcast SSE event for real-time sync across tabs/devices
	if h.notificationService != nil {
		h.notificationService.SSE().SendToUser(user.ID, services.SSEEvent{
			Type: "dashboard.updated",
			Data: map[string]interface{}{
				"dashboard_id": dashboardID.String(),
			},
		})
	}

	c.JSON(http.StatusOK, transformDashboard(dashboard))
}

// ShareUserDashboard updates sharing settings and generates a share token if needed
func (h *DashboardCRUDHandler) ShareUserDashboard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dashboardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "dashboard ID")
		return
	}

	var req struct {
		Visibility string `json:"visibility" binding:"required"` // private, workspace, public_link
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate visibility value
	if req.Visibility != "private" && req.Visibility != "workspace" && req.Visibility != "public_link" {
		utils.RespondBadRequest(c, slog.Default(), "Invalid visibility value. Must be private, workspace, or public_link")
		return
	}

	// Generate share token for public links
	var shareToken *string
	if req.Visibility == "public_link" {
		token := generateShareToken()
		shareToken = &token
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.UpdateShareToken(c.Request.Context(), sqlc.UpdateShareTokenParams{
		ID:         pgtype.UUID{Bytes: dashboardID, Valid: true},
		UserID:     user.ID,
		Visibility: &req.Visibility,
		ShareToken: shareToken,
	})
	if err != nil {
		slog.Default().Error("Failed to update sharing settings", "error", err, "dashboard_id", dashboardID.String())
		utils.RespondInternalError(c, slog.Default(), "update sharing settings", err)
		return
	}

	c.JSON(http.StatusOK, transformDashboard(dashboard))
}

func (h *DashboardCRUDHandler) GetSharedDashboard(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.RespondBadRequest(c, slog.Default(), "Share token required")
		return
	}

	queries := sqlc.New(h.pool)
	dashboard, err := queries.GetDashboardByShareToken(c.Request.Context(), &token)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Dashboard (invalid or expired share token)")
		return
	}

	c.JSON(http.StatusOK, transformDashboard(dashboard))
}
