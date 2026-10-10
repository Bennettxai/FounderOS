package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerIntelligenceRoutes wires the Intelligence module: deterministic
// signals + on-demand AI synthesis.
func (h *Handlers) registerIntelligenceRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ih := NewIntelligenceHandler(h.pool, h.cfg)
	g := api.Group("/intelligence")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ih.GetIntelligence)
		g.POST("/synthesize", ih.Synthesize)
	}
	slog.Info("Intelligence routes registered at /api/intelligence/*")
}
