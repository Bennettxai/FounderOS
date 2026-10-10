package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// RegisterRoutes is the top-level route registration entry point.
// It sets up shared auth middleware and delegates each domain to its own
// register* method, keeping this file under 50 lines.
func (h *Handlers) RegisterRoutes(api *gin.RouterGroup) {
	// Auth middleware for protected routes - uses Redis cache if available
	var auth gin.HandlerFunc
	if h.sessionCache != nil {
		// Use Redis-cached auth for horizontal scaling
		auth = middleware.CachedAuthMiddleware(h.pool, h.sessionCache)
	} else {
		// Fallback to direct DB auth (single instance mode)
		auth = middleware.AuthMiddleware(h.pool)
	}
	optionalAuth := middleware.OptionalAuthMiddleware(h.pool) // For dev-friendly routes

	h.registerChatRoutes(api, auth)
	h.registerProjectRoutes(api, auth)
	h.registerWorkspaceRoutes(api, auth)
	h.registerOrganizationRoutes(api, auth)
	h.registerTeamRoutes(api, auth)
	h.registerCRMRoutes(api, auth)
	h.registerRelationsRoutes(api, auth)
	h.registerGlossaryRoutes(api, auth)
	h.registerKnowledgeRoutes(api, auth)
	h.registerWorkspacePreferencesRoutes(api, auth)
	h.registerWorkspaceCloudSyncRoutes(api, auth)
	h.registerRhythmRoutes(api, auth)
	h.registerCampaignsRoutes(api, auth)
	h.registerOffersRoutes(api, auth)
	h.registerAssetsRoutes(api, auth)
	h.registerDeliverablesRoutes(api, auth)
	h.registerIntelligenceRoutes(api, auth)
	h.registerDriveRoutes(api, auth)
	h.registerWorkspaceAgentsRoutes(api, auth)
	h.registerBuildersRoutes(api, auth)
	h.registerEnginesRoutes(api, auth)
	h.registerManageAnalyticsRoutes(api, auth)
	h.registerManageDataRoutes(api, auth)
	h.registerResourcesRoutes(api, auth)
	h.registerPersonasRoutes(api, auth)
	h.registerContentRoutes(api, auth)
	h.registerAppsRoutes(api, auth)
	h.registerSitesRoutes(api, auth)
	h.registerAIRoutes(api, auth)
	h.registerSearchRoutes(api, auth)
	h.registerMemoryRoutes(api, auth)
	h.registerVoiceRoutes(api, auth)
	h.registerNotificationRoutes(api, auth)
	h.registerUserRoutes(api, auth)
	h.registerInfraRoutes(api, auth)
	h.registerAuthRoutes(api, auth)
	h.registerIntegrationRoutes(api, auth, optionalAuth)
	h.registerPlatformRoutes(api, auth)
	h.registerOSARoutes(api, auth)
	h.registerOptimalRoutes(api, auth)
	h.registerOptimalEngineRoutes(api, auth)
	h.registerFounderosRoutes(api, auth)
}
