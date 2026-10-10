package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerGlossaryRoutes wires the per-workspace business glossary module.
func (h *Handlers) registerGlossaryRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	gh := NewGlossaryHandler(h.pool)
	g := api.Group("/glossary")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", gh.ListTerms)
		g.POST("", gh.CreateTerm)
		g.PUT("/:id", gh.UpdateTerm)
		g.DELETE("/:id", gh.DeleteTerm)
	}
	slog.Info("Glossary routes registered at /api/glossary/*")
}
