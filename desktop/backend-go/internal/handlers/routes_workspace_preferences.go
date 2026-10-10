package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerWorkspacePreferencesRoutes wires the workspace-level settings
// primitive (business defaults every module reads + calendar-specific config).
func (h *Handlers) registerWorkspacePreferencesRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ph := NewWorkspacePreferencesHandler(h.pool)
	g := api.Group("/workspace/preferences")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ph.GetPreferences)
		g.PUT("", ph.UpdatePreferences)
	}
	slog.Info("Workspace preferences routes registered at /api/workspace/preferences")
}
