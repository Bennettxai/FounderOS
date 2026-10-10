package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerBuildersRoutes wires the per-workspace Builders registry: the tools
// for creating forms, flows, automations, apps, and sites.
func (h *Handlers) registerBuildersRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	bh := NewBuildersHandler(h.pool)
	g := api.Group("/builders")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", bh.ListBuilders)
		g.POST("", bh.CreateBuilder)
		g.PUT("/:id", bh.UpdateBuilder)
		g.DELETE("/:id", bh.DeleteBuilder)
	}
	slog.Info("Builders routes registered at /api/builders/*")
}
