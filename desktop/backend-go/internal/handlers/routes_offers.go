package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerOffersRoutes wires the per-workspace offers catalog module.
func (h *Handlers) registerOffersRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	oh := NewOffersHandler(h.pool)
	g := api.Group("/offers")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", oh.ListOffers)
		g.POST("", oh.CreateOffer)
		g.PUT("/:id", oh.UpdateOffer)
		g.DELETE("/:id", oh.DeleteOffer)
	}
	slog.Info("Offers routes registered at /api/offers/*")
}
