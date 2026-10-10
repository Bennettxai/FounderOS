package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerOSARoutes wires up all OSA (21-Agent Orchestration System) routes:
// /api/osa (authenticated), /api/internal/osa (HMAC-protected internal),
// /api/osa/webhooks (public HMAC-verified), /api/sandbox.
func (h *Handlers) registerOSARoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	if h.osaClient != nil {
		slog.Info("Registering OSA authenticated routes")

		// OSA authenticated routes - /api/osa (auth required)
		// Note: /api/osa/health is registered publicly in main.go (no auth)
		osaApiH := NewOSAAPIHandler(h.osaClient, h.pool)
		osaSwarmH := NewOSASwarmHandler(h.osaClient)
		osaAuth := api.Group("/osa")
		osaAuth.Use(auth, middleware.RequireAuth())
		{
			// Generation endpoints
			osaAuth.POST("/generate", osaApiH.HandleGenerateApp)
			osaAuth.GET("/status/:app_id", osaApiH.HandleGetAppStatus)
			osaAuth.GET("/workspaces", osaApiH.HandleListWorkspaces)

			// OSA Template routes - /api/osa/templates
			if h.osaPromptBuilder != nil {
				osaTplH := NewOSATemplateHandler(h.osaClient, h.osaPromptBuilder)
				osaTemplates := osaAuth.Group("/templates")
				{
					osaTemplates.GET("", osaTplH.ListOSATemplates)
					osaTemplates.GET("/:name", osaTplH.GetOSATemplate)
					osaTemplates.POST("/:name/generate", osaTplH.GenerateFromOSATemplate)
					osaTemplates.POST("/:name/preview", osaTplH.PreviewTemplatePrompt)
				}
			} else {
				slog.Info("OSA template routes skipped: osaPromptBuilder not initialized")
			}

			// Swarm endpoints
			osaAuth.POST("/swarm/launch", osaSwarmH.HandleLaunchSwarm)
			osaAuth.GET("/swarm", osaSwarmH.HandleListSwarms)
			osaAuth.GET("/swarm/:id", osaSwarmH.HandleGetSwarm)
			osaAuth.DELETE("/swarm/:id", osaSwarmH.HandleCancelSwarm)

			// Fleet dispatch endpoint
			osaAuth.POST("/fleet/:agent_id/dispatch", osaSwarmH.HandleDispatchInstruction)

			// Tool endpoints
			osaAuth.GET("/tools", osaSwarmH.HandleListTools)
			osaAuth.POST("/tools/:name/execute", osaSwarmH.HandleExecuteTool)

			// SSE streaming for app generation (RESTful path: /generate/:id/stream)
			if h.osaStreamingHandler != nil {
				osaAuth.GET("/generate/:app_id/stream", h.osaStreamingHandler.HandleGenerateAppStream)
			} else {
				slog.Info("OSA generate stream route skipped: osaStreamingHandler not initialized")
			}

			// Workflow endpoints
			if h.osaWorkflowsHandler != nil {
				osaAuth.GET("/workflows", h.osaWorkflowsHandler.ListWorkflows)
				osaAuth.GET("/workflows/:id", h.osaWorkflowsHandler.GetWorkflow)
				osaAuth.GET("/workflows/:id/files", h.osaWorkflowsHandler.GetWorkflowFiles)
				osaAuth.GET("/workflows/:id/files/:type", h.osaWorkflowsHandler.GetFileContent)
				osaAuth.GET("/files/:id/content", h.osaWorkflowsHandler.GetFileContentByID)
				osaAuth.POST("/modules/install", h.osaWorkflowsHandler.InstallModule)
				osaAuth.POST("/sync/trigger", h.osaWorkflowsHandler.TriggerSync)
			} else {
				slog.Info("OSA workflow routes skipped: osaWorkflowsHandler not initialized")
			}

			// Webhook management endpoints
			if h.osaWebhooksHandler != nil {
				osaAuth.GET("/webhooks", h.osaWebhooksHandler.ListWebhooks)
				osaAuth.POST("/webhooks/register", h.osaWebhooksHandler.RegisterWebhook)
			} else {
				slog.Info("OSA webhook management routes skipped: osaWebhooksHandler not initialized")
			}

			// SSE streaming endpoints
			if h.osaStreamingHandler != nil {
				osaAuth.GET("/stream/build/:app_id", h.osaStreamingHandler.StreamBuildProgress)
				osaAuth.GET("/stream/stats", h.osaStreamingHandler.GetStreamStats)
				osaAuth.GET("/stream/stats/:app_id", h.osaStreamingHandler.GetAppStreamStats)
			} else {
				slog.Info("OSA SSE streaming routes skipped: osaStreamingHandler not initialized")
			}

			// App deployment endpoints
			// NOTE: Routes use /deployment namespace to avoid conflicts with /apps/:id wildcard
			if h.osaDeploymentHandler != nil {
				osaAuth.POST("/deployment/:app_id/deploy", h.osaDeploymentHandler.DeployApp)
				osaAuth.POST("/deployment/:app_id/stop", h.osaDeploymentHandler.StopApp)
				osaAuth.GET("/deployment/:app_id/status", h.osaDeploymentHandler.GetAppStatus)
				osaAuth.GET("/deployments", h.osaDeploymentHandler.ListDeployedApps)
			} else {
				slog.Info("OSA deployment routes skipped: osaDeploymentHandler not initialized")
			}

			// App management endpoints (ISR-4)
			if h.osaAppsHandler != nil {
				osaAuth.GET("/module-instances", h.osaAppsHandler.ListApps)
				osaAuth.GET("/module-instances/:id", h.osaAppsHandler.GetApp)
				osaAuth.DELETE("/module-instances/:id", h.osaAppsHandler.DeleteApp)
				osaAuth.PATCH("/module-instances/:id", h.osaAppsHandler.UpdateApp)
				osaAuth.GET("/module-instances/:id/logs", h.osaAppsHandler.GetAppLogs)
				osaAuth.GET("/module-instances/:id/files", h.osaAppsHandler.GetAppFiles)
				osaAuth.PUT("/module-instances/:id/files", h.osaAppsHandler.SaveAppFile)
				osaAuth.GET("/module-instances/:id/download", h.osaAppsHandler.DownloadApp)
				osaAuth.GET("/module-instances/:id/generated-files", h.osaAppsHandler.GetAppGeneratedFiles)

				// Snapshot endpoints (PEDRO-6)
				osaAuth.GET("/module-instances/:id/snapshots", h.osaAppsHandler.ListSnapshots)
				osaAuth.GET("/module-instances/:id/snapshots/:snapshotId1/diff/:snapshotId2", h.osaAppsHandler.GetSnapshotDiff)

				// Queue item status endpoint (polling fallback for SSE)
				osaAuth.GET("/module-instances/queue/:queue_item_id/status", h.osaAppsHandler.GetQueueItemStatus)

				// SSE streaming endpoint for app generation progress
				osaAuth.GET("/module-instances/generate/:queue_item_id/stream", h.osaAppsHandler.StreamAppGeneration)
			} else {
				slog.Info("OSA app management routes skipped: osaAppsHandler not initialized")
			}

			// Prompt templates endpoints
			if h.osaPromptBuilder != nil {
				promptTemplatesHandler := NewOSAPromptTemplatesHandler(h.osaPromptBuilder, h.pool)
				osaAuth.GET("/prompt-templates/system", promptTemplatesHandler.ListSystemTemplates)
				osaAuth.GET("/prompt-templates/health", promptTemplatesHandler.TemplateHealthCheck)
			} else {
				slog.Info("OSA prompt template routes skipped: osaPromptBuilder not initialized")
			}
		}
		slog.Info("OSA authenticated API routes registered at /api/osa/*")

		// OSA Internal routes - /api/internal/osa (for terminal containers)
		// SECURITY: Internal auth middleware validates HMAC signature or IP allowlist
		// This prevents authentication bypass via header spoofing (CVE-like vulnerability)
		osaInternalH := NewOSAInternalHandler(h.osaClient, h.pool)
		internalAuthCfg := middleware.NewInternalAuthConfigFromEnv()
		osaInternal := api.Group("/internal/osa")
		osaInternal.Use(middleware.InternalAuthMiddleware(internalAuthCfg))
		{
			osaInternal.POST("/generate", osaInternalH.HandleInternalGenerateApp)
			osaInternal.GET("/status/:app_id", osaInternalH.HandleInternalGetAppStatus)
			osaInternal.GET("/workspaces", osaInternalH.HandleInternalListWorkspaces)
			osaInternal.GET("/health", osaInternalH.HandleInternalOSAHealth)
		}
		slog.Info("OSA internal API routes registered at /api/internal/osa/* (with HMAC auth)")
	} else {
		slog.Info("OSA routes disabled; osaClient not initialized")
	}

	// OSA App Deployment routes - works independently of full OSA integration
	// NOTE: Only register these if OSA client is NOT initialized (to avoid duplicate routes)
	if h.osaDeploymentHandler != nil && h.osaClient == nil {
		osaDeployRoutes := api.Group("/osa")
		osaDeployRoutes.Use(auth, middleware.RequireAuth())
		{
			osaDeployRoutes.POST("/deployment/:app_id/deploy", h.osaDeploymentHandler.DeployApp)
			osaDeployRoutes.POST("/deployment/:app_id/stop", h.osaDeploymentHandler.StopApp)
			osaDeployRoutes.GET("/deployment/:app_id/status", h.osaDeploymentHandler.GetAppStatus)
			osaDeployRoutes.GET("/deployments", h.osaDeploymentHandler.ListDeployedApps)
		}
		slog.Info("OSA deployment routes registered independently at /api/osa/deployment/* and /api/osa/deployments")
	}

	// OSA Webhook receiver routes - /api/osa/webhooks (OSA-5 → BusinessOS)
	// Public endpoints with HMAC verification (no JWT auth)
	if h.osaWebhooksHandler != nil {
		osaWebhooks := api.Group("/osa/webhooks")
		{
			osaWebhooks.POST("/workflow-complete", h.osaWebhooksHandler.HandleWorkflowComplete)
			osaWebhooks.POST("/build-event", h.osaWebhooksHandler.HandleBuildEvent)
		}
		slog.Info("[Router] OSA webhook receiver routes registered", "path", "/api/osa/webhooks/*")
	} else {
		slog.Info("OSA webhook receiver routes disabled; osaWebhooksHandler not initialized")
	}

	// Sandbox routes - /api/sandbox (App Deployment & Management)
	if h.sandboxHandler != nil {
		sandbox := api.Group("/sandbox")
		sandbox.Use(auth, middleware.RequireAuth())
		{
			sandbox.POST("/deploy", h.sandboxHandler.DeploySandbox)
			sandbox.GET("/:app_id", h.sandboxHandler.GetSandboxInfo)
			sandbox.POST("/:app_id/stop", h.sandboxHandler.StopSandbox)
			sandbox.POST("/:app_id/restart", h.sandboxHandler.RestartSandbox)
			sandbox.DELETE("/:app_id", h.sandboxHandler.RemoveSandbox)
			sandbox.GET("/:app_id/logs", h.sandboxHandler.GetSandboxLogs)
			sandbox.GET("", h.sandboxHandler.ListUserSandboxes)
			sandbox.GET("/stats", h.sandboxHandler.GetSandboxStats)
		}
		slog.Info("Sandbox routes registered at /api/v1/sandbox/*")
	} else {
		slog.Info("Sandbox routes disabled; sandboxHandler not initialized")
	}

	// Sandbox Edit routes - /api/sandbox/edit (Module Edit Lifecycle)
	if h.sandboxEditHandler != nil {
		sandboxEdit := api.Group("/sandbox/edit")
		sandboxEdit.Use(auth, middleware.RequireAuth())
		h.sandboxEditHandler.RegisterRoutes(sandboxEdit)
		slog.Info("Sandbox edit routes registered at /api/v1/sandbox/edit/*")
	} else {
		slog.Info("Sandbox edit routes skipped: sandboxEditHandler not initialized")
	}
}
