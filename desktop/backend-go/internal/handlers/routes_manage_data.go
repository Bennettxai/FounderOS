package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerManageDataRoutes wires the Manage > Data module: a read-only inventory
// of workspace data entities and their record counts. Mounted under
// /manage-data to avoid colliding with any existing /data route group.
func (h *Handlers) registerManageDataRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	mh := NewManageDataHandler(h.pool)
	g := api.Group("/manage-data")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("/summary", mh.GetSummary)
	}
	slog.Info("Manage data routes registered at /api/manage-data/*")
}
