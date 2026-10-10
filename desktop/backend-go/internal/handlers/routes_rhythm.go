package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerRhythmRoutes wires the per-workspace operating rhythm module.
func (h *Handlers) registerRhythmRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	rh := NewRhythmHandler(h.pool)
	g := api.Group("/rhythm")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", rh.ListEntries)
		g.POST("", rh.CreateEntry)
		g.PUT("/:id", rh.UpdateEntry)
		g.DELETE("/:id", rh.DeleteEntry)
	}
	slog.Info("Rhythm routes registered at /api/rhythm/*")
}
