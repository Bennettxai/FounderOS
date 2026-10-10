package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerContentRoutes wires the per-workspace content pipeline module.
func (h *Handlers) registerContentRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ch := NewContentHandler(h.pool)
	g := api.Group("/content")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ch.ListItems)
		g.POST("", ch.CreateItem)
		g.PUT("/:id", ch.UpdateItem)
		g.DELETE("/:id", ch.DeleteItem)
	}
	slog.Info("Content routes registered at /api/content/*")
}
