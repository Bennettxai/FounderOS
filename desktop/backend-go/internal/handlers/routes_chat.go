package handlers

import "github.com/gin-gonic/gin"

// registerChatRoutes wires up all chat-adjacent routes:
// /api/chat, /api/artifacts, /api/contexts, /api/daily-logs,
// /api/thinking, /api/focus.
func (h *Handlers) registerChatRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Chat routes - /api/chat (extracted handler)
	chatHandler := NewChatHandler(
		h.pool,
		h.cfg,
		h.tieredContextService,
		h.contextTracker,
		h.autoLearningTriggers,
		h.blockMapper,
		h.roleContextService,
	).withOptionalDeps(
		h.embeddingService,
		h.promptPersonalizer,
		h.memoryHierarchyService,
		h.skillsLoader,
		h.sessionHealthSvc,
		h.osaClient,
		h.signalHints,
		h.subconsciousObserver,
	)
	chatHandler.SetEngineSync(h.engineSync)
	RegisterChatRoutes(api, chatHandler, auth)

	// Artifacts routes - /api/artifacts
	RegisterArtifactRoutes(api, NewArtifactHandler(h.pool), auth)

	// Contexts routes - /api/contexts
	RegisterContextRoutes(api, NewContextHandler(h.pool, h.queryCache, h.engineSync), auth)

	// Daily log routes - /api/daily-logs
	dailyLogHandler := NewDailyLogHandler(h.pool)
	dailyLogHandler.SetEngineSync(h.engineSync)
	RegisterDailyLogRoutes(api, dailyLogHandler, auth)

	// Thinking/COT + Reasoning templates routes - /api/thinking, /api/reasoning
	RegisterThinkingRoutes(api, NewThinkingHandler(h.pool), auth)

	// Focus Mode routes - /api/focus
	RegisterFocusRoutes(api, NewFocusHandler(h.pool), auth)
}
