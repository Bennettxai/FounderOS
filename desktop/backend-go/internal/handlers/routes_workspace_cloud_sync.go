package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerWorkspaceCloudSyncRoutes wires the owner-triggered local→cloud sync.
func (h *Handlers) registerWorkspaceCloudSyncRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	sh := NewWorkspaceCloudSyncHandler(h.pool)
	g := api.Group("/workspace")
	g.Use(auth, middleware.RequireAuth())
	{
		g.POST("/sync-to-cloud", sh.SyncToCloud)
	}
	slog.Info("Workspace cloud-sync route registered at /api/workspace/sync-to-cloud")
}
