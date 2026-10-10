package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerManageAnalyticsRoutes wires the Manage > Analytics module: read-only
// workspace counts + simple trends. Mounted under /manage-analytics to avoid
// colliding with any existing /analytics route group.
func (h *Handlers) registerManageAnalyticsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	mh := NewManageAnalyticsHandler(h.pool)
	g := api.Group("/manage-analytics")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("/summary", mh.GetSummary)
	}
	slog.Info("Manage analytics routes registered at /api/manage-analytics/*")
}
