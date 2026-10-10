package handlers

import (
	"log/slog"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/sorx"
)

// SetOSADeploymentHandler sets the OSA deployment handler (can work independently)
func (h *Handlers) SetOSADeploymentHandler(deploymentHandler *OSADeploymentHandler) {
	h.osaDeploymentHandler = deploymentHandler
}

// SetOSAFileServices sets the OSA file sync and workflow services
func (h *Handlers) SetOSAFileServices(fileSyncService *services.OSAFileSyncService, workspaceInit *services.OSAWorkspaceInitService, workflowsHandler *OSAWorkflowsHandler, webhooksHandler *OSAWebhooksHandler, eventBus *services.BuildEventBus, streamingHandler *OSAStreamingHandler, deploymentHandler *OSADeploymentHandler) {
	h.osaFileSyncService = fileSyncService
	h.osaWorkspaceInit = workspaceInit
	h.osaWorkflowsHandler = workflowsHandler
	h.osaWebhooksHandler = webhooksHandler
	h.osaBuildEventBus = eventBus
	h.osaStreamingHandler = streamingHandler
	h.osaDeploymentHandler = deploymentHandler

	// Initialize OSA apps handler (ISR-4)
	if h.pool != nil {
		queries := sqlc.New(h.pool)
		h.osaAppsHandler = NewOSAAppsHandler(queries, h.pool, slog.Default())

		// Initialize OSA prompt builder
		promptBuilder, err := services.NewOSAPromptBuilder(h.pool, slog.Default())
		if err != nil {
			slog.Error("failed to initialize OSA prompt builder", "error", err)
		} else {
			h.osaPromptBuilder = promptBuilder
			slog.Info("OSA prompt builder initialized successfully")
		}

		// Inject prompt builder into apps handler
		if h.osaAppsHandler != nil && h.osaPromptBuilder != nil {
			h.osaAppsHandler.SetPromptBuilder(h.osaPromptBuilder)
			slog.Info("OSA prompt builder injected into apps handler")
		}

		// Inject event bus into apps handler for SSE streaming
		if h.osaAppsHandler != nil && eventBus != nil {
			h.osaAppsHandler.SetEventBus(eventBus)
			slog.Info("OSA event bus injected into apps handler for SSE streaming")
		}

		// Initialize and inject snapshot diff service (PEDRO-6)
		if h.osaAppsHandler != nil {
			diffService := services.NewSnapshotDiffService(queries, slog.Default())
			h.osaAppsHandler.SetDiffService(diffService)
			slog.Info("snapshot diff service initialized and injected into apps handler")
		}
	}
}

// SetOSAQueueWorker injects the queue worker into the apps handler for immediate notification
func (h *Handlers) SetOSAQueueWorker(worker *services.OSAQueueWorker) {
	if h.osaAppsHandler != nil && worker != nil {
		h.osaAppsHandler.SetQueueWorker(worker)
		slog.Info("OSA queue worker injected into apps handler for immediate notification")
	}
}

// SetOSAPromptBuilder sets the OSA prompt builder (for template-based app generation)
func (h *Handlers) SetOSAPromptBuilder(pb *services.OSAPromptBuilder) {
	h.osaPromptBuilder = pb
}

// SetSorxEngine sets the SORX skill execution engine (for EXECUTE mode routing)
func (h *Handlers) SetSorxEngine(engine *sorx.Engine) {
	h.sorxEngine = engine
}

// SetProtectionService sets the module protection service (for BUILD mode safety)
func (h *Handlers) SetProtectionService(svc *services.ModuleProtectionService) {
	h.protectionService = svc
}
