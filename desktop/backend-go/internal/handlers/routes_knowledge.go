package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerKnowledgeRoutes wires the engine-backed Knowledge view: the active
// workspace's Optimal Engine knowledge base, gated by active membership of the
// X-Workspace-ID workspace.
//
// This sits alongside RegisterKnowledgeRoutes (the markdown file tree); it adds
// the engine-canonical surface under the same /knowledge prefix without
// touching that registration.
//
// Wire from routes.go with:  h.registerKnowledgeRoutes(api, auth)
func (h *Handlers) registerKnowledgeRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	keh := NewKnowledgeEngineHandler(h.pool)
	g := api.Group("/knowledge")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("/engine", keh.GetEngineKnowledge)
	}
	slog.Info("Knowledge engine routes registered at /api/knowledge/engine")
}
