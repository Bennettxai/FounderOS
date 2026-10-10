package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

func (h *Handlers) registerDeliverablesRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	dh := NewDeliverablesHandler(h.pool)
	g := api.Group("/deliverables")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("", dh.ListDeliverables)
		g.POST("", dh.CreateDeliverable)
		g.PUT("/:id", dh.UpdateDeliverable)
		g.DELETE("/:id", dh.DeleteDeliverable)
	}
	slog.Info("Deliverables routes registered at /api/deliverables/*")
}
