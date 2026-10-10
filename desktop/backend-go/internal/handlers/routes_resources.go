package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerResourcesRoutes wires the per-workspace resource library: links, docs,
// SOPs, and tools the team relies on.
func (h *Handlers) registerResourcesRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	rh := NewResourcesHandler(h.pool)
	g := api.Group("/resources")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", rh.ListResources)
		g.POST("", rh.CreateResource)
		g.PUT("/:id", rh.UpdateResource)
		g.DELETE("/:id", rh.DeleteResource)
	}
	slog.Info("Resources routes registered at /api/resources/*")
}
