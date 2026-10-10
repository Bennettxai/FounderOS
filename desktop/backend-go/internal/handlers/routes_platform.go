package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// registerPlatformRoutes wires up platform-level and team routes:
// /api/dashboard (items), /api/team, /api/settings, /api/usage,
// /api/desktop3d, /api/signal/health, /api/sorx,
// and the Pedro-owned task routes (documents, learning, app-profiler, intelligence).
func (h *Handlers) registerPlatformRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Dashboard routes - /api/dashboard (aggregate dashboard items)
	dashboardHandler := NewDashboardItemHandler(h.pool, h.queryCache, h.notificationTriggers)
	dashboardHandler.SetEngineSync(h.engineSync)
	RegisterDashboardItemRoutes(api, dashboardHandler, auth)

	// Team routes - /api/team
	teamHandler := NewTeamHandler(h.pool, h.queryCache)
	teamHandler.SetEngineSync(h.engineSync)
	RegisterTeamRoutes(api, teamHandler, auth)

	// Settings routes - /api/settings
	RegisterSettingsRoutes(api, NewSettingsHandler(h.pool, h.cfg), auth)

	// Desktop 3D Layout routes - /api/desktop3d
	RegisterDesktop3DRoutes(api, NewDesktop3DHandler(h.pool), auth)
	slog.Info("Desktop 3D layout routes registered")

	// Signal health route - /api/signal/health
	signal := api.Group("/signal")
	signal.Use(auth, middleware.RequireAuth())
	{
		signal.GET("/health", h.GetSignalHealth)
	}
	slog.Info("Signal health route registered at /api/signal/health")

	// Sorx Integration Module - /api/sorx
	sorxService := services.NewSorxService(h.pool, h.cfg)
	sorxLogger := slog.Default().With("component", "sorx")
	sorxHandler := NewSorxHandler(sorxService, h.pool, sorxLogger)

	// Credential rate limit: 10 requests per minute per user (OWASP A07)
	credentialRateLimit := middleware.CredentialRateLimitMiddleware()
	strictRateLimit := middleware.StrictRateLimitMiddleware()

	sorxRoutes := api.Group("/sorx")
	{
		// Callback endpoint (for skill execution callbacks) — validates signed ticket internally
		sorxRoutes.POST("/callback", strictRateLimit, sorxHandler.HandleCallback)

		// Public skill catalog (browse available skills)
		sorxRoutes.GET("/skills", sorxHandler.ListSkills)
		sorxRoutes.GET("/skills/:id", sorxHandler.GetSkill)

		// Public skill commands catalog
		sorxRoutes.GET("/commands", sorxHandler.ListSkillCommands)

		// Protected endpoints (require user auth)
		sorxProtected := sorxRoutes.Group("")
		sorxProtected.Use(auth, middleware.RequireAuth())
		{
			// Credential endpoints — require authenticated user to prevent IDOR
			sorxProtected.POST("/credential-ticket", credentialRateLimit, sorxHandler.RequestCredentialTicket)
			sorxProtected.POST("/redeem-credential", credentialRateLimit, sorxHandler.RedeemCredential)
			// Decision endpoints (human-in-the-loop)
			sorxProtected.GET("/decisions", sorxHandler.GetPendingDecisions)
			sorxProtected.GET("/decisions/:id", sorxHandler.GetDecision)
			sorxProtected.POST("/decisions/:id/respond", sorxHandler.RespondToDecision)
			// Skill execution
			sorxProtected.POST("/execute", sorxHandler.TriggerSkill)
			sorxProtected.GET("/executions/:id", sorxHandler.GetSkillExecution)
			// Skill command execution (slash commands that trigger skills)
			sorxProtected.POST("/commands/execute", sorxHandler.ExecuteSkillCommand)
		}
	}

	// Pedro Tasks - Document Processing routes - /api/documents
	if h.documentProcessor != nil {
		documentHandler := NewDocumentHandler(h.documentProcessor)
		protectedDocs := api.Group("")
		protectedDocs.Use(auth, middleware.RequireAuth())
		RegisterDocumentRoutes(protectedDocs, documentHandler)
	} else {
		slog.Info("Document processing routes skipped: documentProcessor not initialized")
	}

	// Pedro Tasks - Learning/Personalization routes - /api/learning
	if h.learningService != nil {
		learningHandler := NewLearningHandler(h.learningService)
		protectedLearning := api.Group("")
		protectedLearning.Use(auth, middleware.RequireAuth())
		RegisterLearningRoutes(protectedLearning, learningHandler)
	} else {
		slog.Info("Learning routes skipped: learningService not initialized")
	}

	// Pedro Tasks - App Profiler routes - /api/app-profiles
	if h.appProfilerService != nil {
		appProfilerHandler := NewAppProfilerHandler(h.appProfilerService)
		protectedProfiler := api.Group("")
		protectedProfiler.Use(auth, middleware.RequireAuth())
		RegisterAppProfilerRoutes(protectedProfiler, appProfilerHandler)
	} else {
		slog.Info("App profiler routes skipped: appProfilerService not initialized")
	}

	// Pedro Tasks - Conversation Intelligence routes - /api/intelligence
	if h.conversationIntelligence != nil && h.memoryExtractor != nil {
		intelligenceHandler := NewConversationIntelligenceHandler(h.conversationIntelligence, h.memoryExtractor)
		protectedIntel := api.Group("")
		protectedIntel.Use(auth, middleware.RequireAuth())
		RegisterConversationIntelligenceRoutes(protectedIntel, intelligenceHandler)
	} else {
		slog.Info("Conversation intelligence routes skipped: conversationIntelligence or memoryExtractor not initialized")
	}

	// Billing metering routes - /api/usage/metering
	// Separate from the token-analytics UsageHandler; tracks plan limits and daily AI call counts.
	RegisterMeteringRoutes(api, NewMeteringHandler(h.pool), auth)
	slog.Info("Billing metering routes registered at /api/usage/metering")
}
