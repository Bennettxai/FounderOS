package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerEnginesRoutes wires the Engines module: internal automation /
// multi-agent workflows that can be defined, listed, and run.
func (h *Handlers) registerEnginesRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	eh := NewEnginesHandler(h.pool, h.cfg)
	g := api.Group("/engines")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", eh.ListWorkflows)
		g.POST("", eh.CreateWorkflow)
		g.PUT("/:id", eh.UpdateWorkflow)
		g.DELETE("/:id", eh.DeleteWorkflow)
		g.POST("/:id/run", eh.RunWorkflow)
		g.GET("/:id/runs", eh.ListRuns)
	}
	slog.Info("Engines routes registered at /api/engines/*")
}
