package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerSitesRoutes wires the per-workspace sites module.
func (h *Handlers) registerSitesRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	sh := NewSitesHandler(h.pool)
	g := api.Group("/sites")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", sh.ListSites)
		g.POST("", sh.CreateSite)
		g.PUT("/:id", sh.UpdateSite)
		g.DELETE("/:id", sh.DeleteSite)
	}
	slog.Info("Sites routes registered at /api/sites/*")
}
