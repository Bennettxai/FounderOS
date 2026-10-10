package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerAppsRoutes wires the per-workspace Apps module.
func (h *Handlers) registerAppsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ah := NewAppsHandler(h.pool)
	g := api.Group("/apps")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("/catalog", ah.ListCatalog)
		g.POST("/catalog/:id/install", ah.InstallCatalogApp)
		g.GET("", ah.ListApps)
		g.POST("", ah.CreateApp)
		g.PUT("/:id", ah.UpdateApp)
		g.DELETE("/:id", ah.DeleteApp)
	}
	slog.Info("Apps routes registered at /api/apps/*")
}
