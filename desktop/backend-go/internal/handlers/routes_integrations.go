package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// registerIntegrationRoutes wires up all integration and module management routes:
// /api/integrations (provider-based router + user integration management),
// /api/modules (browse by module), /api/modules (custom modules CRUD),
// /api/onboarding.
//
// Returns the IntegrationRouter so that dependent routes (e.g. onboarding) can
// extract services from it.
func (h *Handlers) registerIntegrationRoutes(api *gin.RouterGroup, auth gin.HandlerFunc, optionalAuth gin.HandlerFunc) {
	// Initialize the IntegrationRouter which manages all integration providers
	// (Google, Slack, Notion) with their OAuth flows, data sync, and API handlers.
	integrationRouter := NewIntegrationRouter(h.pool)
	h.integrationRouter = integrationRouter

	// Wire calendar event creation into EngineSync so events appear in
	// the Pages graph view alongside everything else. Each integration
	// handler exposes an OnEventCreated callback we attach a closure to.
	if gh := integrationRouter.GetGoogleHandler(); gh != nil {
		gh.OnEventCreated = newGoogleCalendarEngineHook(h.engineSync)
	}
	if gth := integrationRouter.GetGoogleCalendarToolHandler(); gth != nil {
		gth.OnEventCreated = newGoogleCalendarEngineHook(h.engineSync)
	}
	if mh := integrationRouter.GetMicrosoftHandler(); mh != nil {
		mh.OnEventCreated = newMicrosoftCalendarEngineHook(h.engineSync)
	}

	// Wire email and message saves into EngineSync so inbound comms
	// flow into the OptimalEngine knowledge graph as Signals — agents
	// can then RAG over inbox + channels alongside everything else.
	// See docs/COMMUNICATIONS_ENGINE_SYNC.md.
	//
	// Hooks attach to three Gmail service instances (and two Slack
	// MessageService instances) because the legacy unified handler,
	// the tool-OAuth handler, and the connector-adapter pull path each
	// keep their own. The IntegrationRouter exposes the singleton used
	// by the adapter; the per-handler accessors expose the rest.
	gmailHook := newGmailEngineHook(h.engineSync, h.pool)
	slackMsgHook := newSlackMessageEngineHook(h.engineSync, h.pool)

	if gh := integrationRouter.GetGoogleHandler(); gh != nil {
		if svc := gh.Gmail(); svc != nil {
			svc.OnEmailSaved = gmailHook
		}
	}
	if gth := integrationRouter.GetGoogleGmailToolHandler(); gth != nil {
		if svc := gth.Gmail(); svc != nil {
			svc.OnEmailSaved = gmailHook
		}
	}
	if gmailSvc := integrationRouter.GetGoogleGmailService(); gmailSvc != nil {
		gmailSvc.OnEmailSaved = gmailHook
	}
	slog.Info("engine_sync: gmail OnEmailSaved hook wired")

	if sh := integrationRouter.GetSlackHandler(); sh != nil {
		if svc := sh.Messages(); svc != nil {
			svc.OnMessageSaved = slackMsgHook
		}
	}
	if slackMsgSvc := integrationRouter.GetSlackMessageService(); slackMsgSvc != nil {
		slackMsgSvc.OnMessageSaved = slackMsgHook
	}
	slog.Info("engine_sync: slack OnMessageSaved hook wired")

	// Outlook engine sync — same module (ModuleEmail) as Gmail; the
	// metadata.provider="outlook" discriminator distinguishes them in
	// engine queries. See newOutlookEngineHook for the mapping.
	outlookHook := newOutlookEngineHook(h.engineSync, h.pool)
	if mh := integrationRouter.GetMicrosoftHandler(); mh != nil {
		if svc := mh.Outlook(); svc != nil {
			svc.OnEmailSaved = outlookHook
		}
	}
	slog.Info("engine_sync: outlook OnEmailSaved hook wired")

	// Teams engine sync — same module (ModuleMessage) as Slack; the
	// metadata.provider="teams" discriminator distinguishes them.
	teamsMsgHook := newTeamsMessageEngineHook(h.engineSync, h.pool)
	if mh := integrationRouter.GetMicrosoftHandler(); mh != nil {
		if svc := mh.Teams(); svc != nil {
			svc.OnMessageSaved = teamsMsgHook
		}
	}
	slog.Info("engine_sync: teams OnMessageSaved hook wired")

	// Register provider-based integration routes - /api/integrations/{provider}/*
	integrationsGroup := api.Group("/integrations")
	integrationRouter.RegisterRoutes(integrationsGroup, auth)

	// Wave 3 — provider webhooks (UNAUTH; verified via signature/clientState).
	// Mounted on the same /integrations/{provider} prefix as the OAuth
	// callbacks so external configs see one consistent base URL.
	if msgSvc := integrationRouter.GetSlackMessageService(); msgSvc != nil {
		secret := ""
		if h.cfg != nil {
			secret = h.cfg.SlackSigningSecret
		}
		webhook := newSlackWebhookHandler(h.pool, secret, msgSvc, h.commsBroadcaster)
		// Sits next to /integrations/slack/callback (also unauth).
		integrationsGroup.POST("/slack/events", webhook.HandleEvents)
		slog.Info("slack webhook registered: POST /api/integrations/slack/events",
			"signing_secret_configured", secret != "")
	}

	// Gmail Pub/Sub webhook + watch-renewal loop. Notification arrives
	// signed with a Google OIDC token verified against
	// GOOGLE_GMAIL_PUBSUB_AUDIENCE.
	if gmailSvc := integrationRouter.GetGoogleGmailService(); gmailSvc != nil {
		audience := ""
		if h.cfg != nil {
			audience = h.cfg.GmailPubSubAudience
		}
		gmailWebhook := newGmailWebhookHandler(h.pool, gmailSvc, h.commsBroadcaster, audience)
		integrationsGroup.POST("/google_gmail/webhook", gmailWebhook.HandleNotification)

		// Renewal loop runs forever. Hourly tick covers any watch with
		// <24h to expiry (Google caps at 7 days). Same goroutine
		// pattern as the engineSync worker.
		go gmailSvc.WatchRenewalLoop(context.Background())

		slog.Info("gmail webhook registered: POST /api/integrations/google_gmail/webhook (renewal loop running)",
			"audience_configured", audience != "")
	}

	// Microsoft Graph webhook + subscription renewal loop. Single endpoint
	// for both validation handshake (?validationToken=...) and notifications.
	if mh := integrationRouter.GetMicrosoftHandler(); mh != nil {
		msProvider := integrationRouter.GetMicrosoftProvider()
		if msProvider != nil {
			subsSvc := microsoft.NewSubscriptionService(msProvider)
			msWebhook := newMicrosoftWebhookHandler(
				h.pool,
				subsSvc,
				mh.Outlook(),
				mh.Teams(),
				h.commsBroadcaster,
			)
			// Graph validation arrives as POST in production, GET in
			// some docs examples — accept both.
			integrationsGroup.POST("/microsoft/webhook", msWebhook.HandleNotification)
			integrationsGroup.GET("/microsoft/webhook", msWebhook.HandleNotification)

			// Renewal loop runs forever in the background. Hourly tick
			// covers anything within 2h of expiry. Goroutine inherits
			// the server's lifetime; no explicit shutdown today since
			// the existing engineSync worker uses the same pattern.
			go subsSvc.RenewalLoop(context.Background())

			slog.Info("microsoft webhook registered: POST/GET /api/integrations/microsoft/webhook (renewal loop running)")
		}
	}

	// Communications module routes - /api/comms/*
	// Unified inbox + contact search (Wave 2), unified channels +
	// messages (Wave 2 chunk D4), realtime SSE stream (Wave 3 chunk 1).
	commsHandler := newCommsHandler(h.pool, h.commsBroadcaster, h.engineSync)
	commsHandler.RegisterRoutes(api, auth)
	slog.Info("comms routes registered: /api/comms/{inbox,contacts/search,channels,workspaces,stream}")

	// Integrations Module - /api/integrations (user integration management)
	integrationsHandler := NewIntegrationsHandler(h.pool, integrationRouter)
	integrationsModule := api.Group("/integrations")

	// Public endpoints - provider catalog (no auth required)
	integrationsModule.GET("/providers", integrationsHandler.GetProviders)
	integrationsModule.GET("/providers/:id", integrationsHandler.GetProvider)

	// Protected endpoints - user-specific data (auth required)
	integrationsProtected := integrationsModule.Group("")
	integrationsProtected.Use(auth, middleware.RequireAuth())
	{
		// Aggregated status (must be before :id to avoid matching)
		integrationsProtected.GET("/status", integrationsHandler.GetAllIntegrationsStatus)
		// User's connected integrations
		integrationsProtected.GET("/connected", integrationsHandler.GetConnectedIntegrations)
		integrationsProtected.GET("/:id", integrationsHandler.GetIntegration)
		integrationsProtected.PATCH("/:id/settings", integrationsHandler.UpdateIntegrationSettings)
		integrationsProtected.DELETE("/:id", integrationsHandler.DisconnectIntegration)
		integrationsProtected.POST("/:id/sync", integrationsHandler.TriggerSync)
		// AI Model preferences
		integrationsProtected.GET("/ai/preferences", integrationsHandler.GetModelPreferences)
		integrationsProtected.PUT("/ai/preferences", integrationsHandler.UpdateModelPreferences)
	}

	// MCP Server management - /api/integrations/mcp/connectors
	mcpServersHandler := NewMCPServersHandler(h.pool)
	mcpConnectors := integrationsModule.Group("/mcp/connectors")
	mcpConnectors.Use(auth, middleware.RequireAuth())
	{
		mcpConnectors.GET("", mcpServersHandler.ListMCPServers)
		mcpConnectors.POST("", mcpServersHandler.CreateMCPServer)
		mcpConnectors.GET("/:id", mcpServersHandler.GetMCPServer)
		mcpConnectors.PUT("/:id", mcpServersHandler.UpdateMCPServer)
		mcpConnectors.DELETE("/:id", mcpServersHandler.DeleteMCPServer)
		mcpConnectors.POST("/:id/test", mcpServersHandler.TestMCPServer)
		mcpConnectors.POST("/:id/discover", mcpServersHandler.DiscoverMCPTools)
	}
	slog.Info("MCP server routes registered at /api/integrations/mcp/connectors/*")

	// Module-specific integration endpoints - /api/modules/:id/integrations
	modules := api.Group("/modules")
	modules.Use(optionalAuth) // Optional auth for browsing available integrations
	{
		modules.GET("/:id/integrations", integrationsHandler.GetModuleIntegrations)
	}

	// Custom Modules routes - /api/modules (Feature: Custom Module System)
	customModulesHandler := NewCustomModulesHandler(h.pool, slog.Default())
	customModules := api.Group("/modules")
	customModules.Use(auth, middleware.RequireAuth())
	{
		// Module CRUD
		customModules.POST("", customModulesHandler.CreateModule)
		customModules.GET("", customModulesHandler.ListModules)
		customModules.GET("/:id", customModulesHandler.GetModule)
		customModules.PUT("/:id", customModulesHandler.UpdateModule)
		customModules.DELETE("/:id", customModulesHandler.DeleteModule)

		// Module operations
		customModules.POST("/:id/publish", customModulesHandler.PublishModule)
		customModules.POST("/:id/install", customModulesHandler.InstallModule)
		customModules.POST("/:id/share", customModulesHandler.ShareModule)
		customModules.PATCH("/:id/share-scope", customModulesHandler.SetShareScope)

		// Module data
		customModules.GET("/installed", customModulesHandler.ListInstalledModules)
		customModules.GET("/stats", customModulesHandler.GetModuleStats)
		customModules.GET("/popular", customModulesHandler.GetPopularModules)

		// Module import/export
		customModules.GET("/export/:id", customModulesHandler.ExportModule)
		customModules.POST("/import", customModulesHandler.ImportModule)

		// Module records - workspace-scoped record store for "kind: records" modules
		moduleRecordsHandler := NewModuleRecordsHandler(h.pool)
		customModules.GET("/:id/records", moduleRecordsHandler.ListRecords)
		customModules.POST("/:id/records", moduleRecordsHandler.CreateRecord)
		customModules.PUT("/:id/records/:recordId", moduleRecordsHandler.UpdateRecord)
		customModules.DELETE("/:id/records/:recordId", moduleRecordsHandler.DeleteRecord)
	}
	slog.Info("Custom modules routes registered at /api/modules/*")

	// Boards routes - /api/boards (Views & Boards composition layer)
	registerBoardRoutes(api, auth, NewBoardHandler(h.pool))
	slog.Info("Boards routes registered at /api/boards/*")

	// Onboarding routes - /api/onboarding
	// Conversational AI onboarding with workspace setup
	onboardingAIService := services.NewOnboardingAIService()
	gmailService := integrationRouter.GetGoogleGmailService()
	onboardingService := services.NewOnboardingService(h.pool, onboardingAIService, gmailService, h.osaSyncService)
	onboardingHandler := NewOnboardingHandler(onboardingService)
	onboardingHandler.RegisterOnboardingRoutes(api, auth)
	slog.Info("Onboarding routes registered", "ai_provider", onboardingAIService.GetProvider())
}
