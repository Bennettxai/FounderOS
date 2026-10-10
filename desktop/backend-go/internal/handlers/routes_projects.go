package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerProjectRoutes wires up project and node routes:
// /api/projects, /api/projects/templates, /api/nodes.
func (h *Handlers) registerProjectRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Projects routes - /api/projects (extracted handler)
	projectHandler := NewProjectHandler(h.pool, h.queryCache, h.notificationTriggers, h.projectAccessService)
	projectHandler.SetEngineSync(h.engineSync)
	RegisterProjectRoutes(api, projectHandler, auth)

	// Project templates - /api/projects/templates (reusable delivery blueprints).
	// Registered as its own group so it stays out of the /:id matcher above.
	templateHandler := NewProjectTemplateHandler(h.pool)
	templates := api.Group("/projects/templates")
	templates.Use(auth, middleware.RequireAuth())
	{
		templates.GET("", templateHandler.ListTemplates)
		templates.POST("/:key/use", templateHandler.CreateFromTemplate)
	}

	// Nodes routes - /api/nodes
	RegisterNodeRoutes(api, NewNodeHandler(h.pool), auth)
}
