package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// registerAIRoutes wires up all AI configuration and agent routes under /api/ai:
// LLM providers, models, output styles, agent presets, custom agents,
// slash commands, delegation, intent routing, and workflows.
func (h *Handlers) registerAIRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	ai := api.Group("/ai")
	ai.Use(auth, middleware.RequireAuth())
	{
		// AI config (LLM providers, models, system info)
		aiCfgH := NewAIConfigHandler(h.pool, h.cfg)
		ai.GET("/providers", aiCfgH.GetLLMProviders)
		ai.GET("/models", aiCfgH.GetAllModels)
		ai.GET("/models/local", aiCfgH.GetLocalModels)
		ai.POST("/models/pull", aiCfgH.PullModel)
		ai.POST("/models/warmup", aiCfgH.WarmupModel)
		ai.GET("/system", aiCfgH.GetSystemInfo)
		ai.POST("/api-key", aiCfgH.SaveAPIKey)
		ai.POST("/api-keys", aiCfgH.SaveAPIKey)
		ai.PUT("/provider", aiCfgH.UpdateAIProvider)

		// Output styles & preferences
		RegisterOutputStylesRoutes(ai, NewOutputStylesHandler(h.pool))

		// Agent handler (presets, prompts, custom agents)
		agentH := NewAgentHandler(h.pool, h.cfg)
		ai.GET("/agents/presets", agentH.ListAgentPresets)
		ai.GET("/agents/presets/:id", agentH.GetAgentPreset)
		// Agent prompts (built-in) — on AIConfigHandler
		ai.GET("/agents", aiCfgH.GetAgentPrompts)
		ai.GET("/agents/:id", aiCfgH.GetAgentPrompt)
		ai.GET("/custom-agents", agentH.ListCustomAgents)
		ai.POST("/custom-agents", agentH.CreateCustomAgent)
		ai.POST("/custom-agents/sandbox", agentH.TestCustomAgent)
		ai.GET("/custom-agents/category/:category", agentH.ListCustomAgentsByCategory)
		ai.POST("/custom-agents/from-preset/:presetId", agentH.CreateAgentFromPreset)
		ai.GET("/custom-agents/:id", agentH.GetCustomAgent)
		ai.PUT("/custom-agents/:id", agentH.UpdateCustomAgent)
		ai.DELETE("/custom-agents/:id", agentH.DeleteCustomAgent)
		ai.POST("/custom-agents/:id/test", agentH.TestCustomAgent)

		// Slash commands (built-in + custom)
		cmdH := NewCommandHandler(h.pool, h.cfg)
		ai.GET("/commands", cmdH.ListCommands)
		ai.POST("/commands", cmdH.CreateUserCommand)
		ai.GET("/commands/:id", cmdH.GetUserCommand)
		ai.PUT("/commands/:id", cmdH.UpdateUserCommand)
		ai.DELETE("/commands/:id", cmdH.DeleteUserCommand)

		// Agent runs (execution history for the Agents module, workspace-scoped)
		ai.GET("/agent-runs", h.listAgentRuns)
		// Delegate: turn a Dalya request into a task + an agent run (awaiting approval)
		ai.POST("/delegate", h.delegateTask)

		// Agent delegation routes
		delegationHandler := NewDelegationHandler(services.NewDelegationService(h.pool))
		ai.GET("/delegation/agents", delegationHandler.ListAgents)
		ai.GET("/delegation/resolve/:mention", delegationHandler.ResolveAgentMention)
		ai.POST("/delegation/mentions", delegationHandler.ExtractMentions)
		ai.POST("/delegation/delegate", delegationHandler.Delegate)

		// Intent classification / routing
		routerHandler := NewRouterHandler(h.pool)
		routerHandler.RegisterRoutes(ai)

		// Workflow routes
		wfH := NewWorkflowHandler(h.pool)
		ai.GET("/workflows", wfH.ListWorkflows)
		ai.POST("/workflows", wfH.CreateWorkflow)
		ai.GET("/workflows/:id", wfH.GetWorkflow)
		ai.DELETE("/workflows/:id", wfH.DeleteWorkflow)
		ai.POST("/workflows/:id/execute", wfH.ExecuteWorkflow)
		ai.POST("/workflows/trigger/:trigger", wfH.ExecuteWorkflowByTrigger)
		ai.GET("/workflows/executions", wfH.ListWorkflowExecutions)
		ai.GET("/workflows/executions/:id", wfH.GetWorkflowExecution)
	}
}
