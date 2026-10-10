package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerWorkspaceAgentsRoutes wires the new Agents module: per-workspace AI
// agents (workers) and their run history. Mounted at /workspace-agents to avoid
// colliding with the legacy /agents routes.
func (h *Handlers) registerWorkspaceAgentsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	wah := NewWorkspaceAgentsHandler(h.pool, h.cfg)
	g := api.Group("/workspace-agents")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", wah.ListAgents)
		g.POST("", wah.CreateAgent)
		g.PUT("/:id", wah.UpdateAgent)
		g.DELETE("/:id", wah.DeleteAgent)
		g.POST("/:id/run", wah.RunAgent)
		g.GET("/:id/runs", wah.ListRuns)
	}
	slog.Info("Workspace Agents routes registered at /api/workspace-agents/*")
}
