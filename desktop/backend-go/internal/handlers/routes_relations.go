package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerRelationsRoutes wires the generic relations layer at /api/relations.
// This is the one primitive that lets any module relate its core entity to any
// other module's entity by ID (entity_links), scoped to the active workspace.
func (h *Handlers) registerRelationsRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	relationsHandler := NewEntityRelationsHandler(h.pool)

	relations := api.Group("/relations")
	relations.Use(auth, middleware.RequireAuth())
	{
		relations.GET("", relationsHandler.ListLinks)   // ?type=client&id=<uuid>
		relations.POST("", relationsHandler.CreateLink) // body: source/target/link_type
		relations.DELETE("/:id", relationsHandler.DeleteLink)
	}
	slog.Info("Relations routes registered at /api/relations/*")
}
