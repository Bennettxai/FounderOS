package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerSearchRoutes wires up all search, embeddings, and RAG routes:
// /api/search (web + hybrid + multimodal), /api/embeddings, /api/rag.
func (h *Handlers) registerSearchRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Web Search routes - /api/search
	search := api.Group("/search")
	search.Use(auth, middleware.RequireAuth())
	RegisterWebSearchRoutes(search, NewWebSearchHandler(h.pool))

	// Hybrid / multimodal search routes (appended to /api/search)
	if h.hybridSearchService != nil {
		ragH := NewRAGHandler(h.hybridSearchService, h.rerankerService, h.agenticRAGService, h.memoryService)
		search.POST("/hybrid", ragH.HybridSearch)
		search.POST("/rerank", ragH.HybridSearch)
		search.GET("/explain", ragH.HybridSearchExplain)
	} else {
		slog.Info("Hybrid search routes skipped: hybridSearchService not initialized")
	}
	if h.multiModalHandler != nil {
		search.POST("/multimodal", h.multiModalHandler.SearchWithImage)
	} else {
		slog.Info("Multimodal search route skipped: multiModalHandler not initialized")
	}

	// Embeddings routes - /api/embeddings (for RAG and semantic search)
	if h.embeddingService != nil && h.contextBuilder != nil {
		embeddingHandler := NewEmbeddingHandler(h.embeddingService, h.contextBuilder)
		embeddings := api.Group("/embeddings")
		embeddings.Use(auth, middleware.RequireAuth())
		{
			embeddings.POST("/index/:id", embeddingHandler.IndexDocument)
			embeddings.POST("/search", embeddingHandler.SemanticSearch)
			embeddings.POST("/context", embeddingHandler.BuildAIContext)
			embeddings.GET("/context/:id", embeddingHandler.GetDocumentContext)
			embeddings.GET("/stats", embeddingHandler.GetStats)
			embeddings.GET("/health", embeddingHandler.HealthCheck)
		}
	} else {
		slog.Info("Embeddings routes skipped: embeddingService or contextBuilder not initialized")
	}

	// RAG routes - /api/rag (Advanced retrieval with hybrid search + agentic RAG)
	if h.hybridSearchService != nil || h.agenticRAGService != nil || h.memoryService != nil || h.rerankerService != nil || h.multiModalHandler != nil {
		ragH := NewRAGHandler(h.hybridSearchService, h.rerankerService, h.agenticRAGService, h.memoryService)
		rag := api.Group("/rag")
		rag.Use(auth, middleware.RequireAuth())
		{
			// Hybrid search endpoints
			if h.hybridSearchService != nil {
				rag.POST("/search/hybrid", ragH.HybridSearch)
				rag.POST("/search/hybrid/explain", ragH.HybridSearchExplain)
				rag.GET("/search/explain", ragH.SearchExplain)
			} else {
				slog.Info("RAG hybrid search routes skipped: hybridSearchService not initialized")
			}

			// Re-ranking endpoints
			if h.rerankerService != nil {
				rag.POST("/search/rerank", ragH.ReRankResults)
				rag.POST("/search/rerank/explain", ragH.ReRankExplain)
			} else {
				slog.Info("RAG rerank routes skipped: rerankerService not initialized")
			}

			// Agentic RAG endpoint
			if h.agenticRAGService != nil {
				rag.POST("/retrieve", ragH.AgenticRAGRetrieve)
			} else {
				slog.Info("RAG agentic retrieve route skipped: agenticRAGService not initialized")
			}

			// Memory endpoints
			if h.memoryService != nil {
				rag.GET("/memories", ragH.MemoryList)
				rag.GET("/memories/:id", ragH.MemoryGet)
				rag.POST("/memories", ragH.MemoryCreate)
			} else {
				slog.Info("RAG memory routes skipped: memoryService not initialized")
			}
		}
	} else {
		slog.Info("RAG routes skipped: no RAG services initialized (hybridSearchService, agenticRAGService, memoryService, rerankerService, multiModalHandler all nil)")
	}

	// Multi-modal Search routes - /api/images (Feature 7: Multi-modal Embeddings)
	if h.multiModalHandler != nil {
		RegisterMultiModalRoutes(api, h.multiModalHandler)
	} else {
		slog.Info("Multi-modal image routes skipped: multiModalHandler not initialized")
	}
}
