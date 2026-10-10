package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerPersonasRoutes wires the per-workspace buyer personas module.
func (h *Handlers) registerPersonasRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ph := NewPersonasHandler(h.pool)
	g := api.Group("/personas")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ph.ListPersonas)
		g.POST("", ph.CreatePersona)
		g.PUT("/:id", ph.UpdatePersona)
		g.DELETE("/:id", ph.DeletePersona)
	}
	slog.Info("Personas routes registered at /api/personas/*")
}
