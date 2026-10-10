package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerWorkspaceRoutes wires up all workspace-related routes:
// /api/workspaces (core CRUD + members/invites/audit/versions),
// /api/workspaces/:id/* (apps, app-versions, workspace-memory, template-recommendations),
// /api/app-templates.
func (h *Handlers) registerWorkspaceRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Core workspace domain (CRUD, members, invites, audit, versions)
	RegisterWorkspaceRoutes(api, NewWorkspaceHandler(
		h.pool,
		h.workspaceService,
		h.workspaceVersionService,
		h.roleContextService,
		h.inviteService,
		h.auditService,
	), auth)

	// Workspace-scoped routes owned by other domains (apps, versioning, memory,
	// template recommendations).
	{
		appH := NewAppHandler(h.pool, h.workspaceService)
		workspacesScopedOther := api.Group("/workspaces/:id")
		workspacesScopedOther.Use(auth, middleware.RequireAuth())
		workspacesScopedOther.Use(middleware.InjectRoleContext(h.pool, h.roleContextService))
		{
			desktopSpacesH := NewDesktopSpacesHandler(h.pool)
			presenceH := newDesktopPresenceHandler(h.pool)
			workspacesScopedOther.GET("/desktop-spaces", desktopSpacesH.List)
			workspacesScopedOther.PUT("/desktop-spaces", desktopSpacesH.Save)
			workspacesScopedOther.DELETE("/desktop-spaces/:desktopSpaceId", desktopSpacesH.Delete)
			workspacesScopedOther.GET("/desktop-spaces/:desktopSpaceId/presence/ws", presenceH.handleWebSocket)

			// Workspace memory routes - CUS-25
			memoryHandler := NewWorkspaceMemoryHandlers(h.pool)
			RegisterWorkspaceMemoryRoutes(workspacesScopedOther, memoryHandler)

			// User generated apps routes - post-onboarding app generation
			workspacesScopedOther.GET("/apps", appH.ListUserApps)
			workspacesScopedOther.POST("/apps", appH.CreateUserAppFromTemplate)
			workspacesScopedOther.GET("/apps/:appId", appH.GetUserApp)
			workspacesScopedOther.PATCH("/apps/:appId", appH.UpdateUserApp)
			workspacesScopedOther.DELETE("/apps/:appId", appH.DeleteUserApp)
			workspacesScopedOther.POST("/apps/:appId/access", appH.IncrementAppAccessCount)

			// OSA queue-based generation endpoint
			if h.osaAppsHandler != nil {
				workspacesScopedOther.POST("/apps/generate-osa", h.osaAppsHandler.GenerateOSAApp)
			}

			// App versioning routes (per-app version management)
			workspacesScopedOther.GET("/apps/:appId/versions", appH.ListAppVersions)
			workspacesScopedOther.GET("/apps/:appId/versions/latest", appH.GetLatestAppVersion)
			workspacesScopedOther.GET("/apps/:appId/versions/stats", appH.GetAppVersionStats)
			workspacesScopedOther.GET("/apps/:appId/versions/:versionNumber", appH.GetAppVersion)
			workspacesScopedOther.POST("/apps/:appId/versions", appH.CreateAppSnapshot)
			workspacesScopedOther.POST("/apps/:appId/restore/:versionNumber", appH.RestoreAppVersion)
			workspacesScopedOther.DELETE("/apps/:appId/versions/cleanup", appH.DeleteOldAppVersions)

			// Template recommendations (workspace-scoped)
			workspacesScopedOther.GET("/template-recommendations", appH.GetTemplateRecommendations)
		}
	}

	// App Templates routes - /api/app-templates
	{
		appTplH := NewAppHandler(h.pool, h.workspaceService)
		appTemplates := api.Group("/app-templates")
		appTemplates.Use(auth, middleware.RequireAuth())
		{
			appTemplates.GET("", appTplH.ListAppTemplates)
			appTemplates.GET("/builtin", appTplH.GetBuiltInTemplates)
			appTemplates.GET("/:id", appTplH.GetAppTemplate)
			appTemplates.POST("/:id/generate", appTplH.GenerateFromTemplate)
		}
	}
}
