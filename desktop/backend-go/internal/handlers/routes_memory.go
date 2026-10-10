package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerMemoryRoutes wires up episodic memory, user facts, and context-tree routes:
// /api/memories, /api/user-facts, /api/context-tree.
func (h *Handlers) registerMemoryRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	memoryHandler := NewMemoryHandler(h.pool, h.embeddingService)

	// Memory routes - /api/memories (episodic memory system)
	memories := api.Group("/memories")
	memories.Use(auth, middleware.RequireAuth())
	memories.Use(middleware.AuditSensitiveAccess("memory"))
	{
		memories.GET("", memoryHandler.ListMemories)
		memories.POST("", memoryHandler.CreateMemory)
		memories.GET("/stats", memoryHandler.GetMemoryStats)
		memories.POST("/search", memoryHandler.SearchMemories)
		memories.POST("/relevant", memoryHandler.GetRelevantMemories)
		memories.GET("/project/:projectId", memoryHandler.GetProjectMemories)
		memories.GET("/node/:nodeId", memoryHandler.GetNodeMemories)
		memories.GET("/:id", memoryHandler.GetMemory)
		memories.PUT("/:id", memoryHandler.UpdateMemory)
		memories.DELETE("/:id", memoryHandler.DeleteMemory)
		memories.POST("/:id/pin", memoryHandler.PinMemory)
	}

	// User Facts routes - /api/user-facts
	userFacts := api.Group("/user-facts")
	userFacts.Use(auth, middleware.RequireAuth())
	{
		userFacts.GET("", memoryHandler.ListUserFacts)
		userFacts.PUT("/:key", memoryHandler.UpdateUserFact)
		userFacts.POST("/:key/confirm", memoryHandler.ConfirmUserFact)
		userFacts.POST("/:key/reject", memoryHandler.RejectUserFact)
		userFacts.DELETE("/:key", memoryHandler.DeleteUserFact)
	}

	// Context Tree routes - /api/context-tree (hierarchical context for agents)
	contextTreeHandler := NewContextTreeHandler(h.pool, h.embeddingService)
	contextTree := api.Group("/context-tree")
	contextTree.Use(auth, middleware.RequireAuth())
	{
		// Tree retrieval
		contextTree.GET("/:entityType/:entityId", contextTreeHandler.GetContextTree)
		contextTree.POST("/search", contextTreeHandler.SearchContextTree)
		contextTree.POST("/load", contextTreeHandler.LoadContextItem)
		contextTree.GET("/stats", contextTreeHandler.GetContextStats)
		// Loading rules
		contextTree.GET("/rules/:entityType/:entityId", contextTreeHandler.GetLoadingRules)
		// Context sessions
		contextTree.POST("/session", contextTreeHandler.CreateContextSession)
		contextTree.GET("/session/:sessionId", contextTreeHandler.GetContextSession)
		contextTree.PUT("/session/:sessionId", contextTreeHandler.UpdateContextSession)
		contextTree.DELETE("/session/:sessionId", contextTreeHandler.EndContextSession)
	}
}
