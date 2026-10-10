package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerCampaignsRoutes wires the per-workspace campaigns module.
func (h *Handlers) registerCampaignsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ch := NewCampaignsHandler(h.pool)
	g := api.Group("/campaigns")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", ch.ListCampaigns)
		g.POST("", ch.CreateCampaign)
		g.PUT("/:id", ch.UpdateCampaign)
		g.DELETE("/:id", ch.DeleteCampaign)
	}
	slog.Info("Campaigns routes registered at /api/campaigns/*")
}
