package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// EmbeddingHandler handles embedding-related HTTP requests
type EmbeddingHandler struct {
	embeddingService *services.EmbeddingService
	contextBuilder   *services.ContextBuilder
}

// NewEmbeddingHandler creates a new embedding handler
func NewEmbeddingHandler(embeddingService *services.EmbeddingService, contextBuilder *services.ContextBuilder) *EmbeddingHandler {
	return &EmbeddingHandler{
		embeddingService: embeddingService,
		contextBuilder:   contextBuilder,
	}
}

// IndexDocument indexes a document's blocks for semantic search
// POST /api/embeddings/index/:id
func (h *EmbeddingHandler) IndexDocument(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	contextID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid context ID"})
		return
	}

	var req struct {
		Blocks []services.Block `json:"blocks" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	if len(req.Blocks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No blocks provided"})
		return
	}

	if err := h.embeddingService.IndexDocument(c.Request.Context(), contextID, req.Blocks); err != nil {
		RespondInternalErr(c, "index document", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "indexed",
		"blocks_count": len(req.Blocks),
	})
}

// SemanticSearch performs a semantic search across the user's documents
// POST /api/embeddings/search
func (h *EmbeddingHandler) SemanticSearch(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	results, err := h.embeddingService.SimilaritySearch(c.Request.Context(), req.Query, req.Limit, user.ID)
	if err != nil {
		RespondInternalErr(c, "semantic search", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}

// BuildAIContext builds hierarchical context for AI queries
// POST /api/embeddings/context
func (h *EmbeddingHandler) BuildAIContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	hc, err := h.contextBuilder.BuildContext(c.Request.Context(), req.Query, user.ID, req.Limit)
	if err != nil {
		RespondInternalErr(c, "build AI context", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"context":   hc,
		"formatted": hc.FormatForAI(),
	})
}

// GetDocumentContext builds context for a specific document
// GET /api/embeddings/context/:id
func (h *EmbeddingHandler) GetDocumentContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	contextID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid context ID"})
		return
	}

	hc, err := h.contextBuilder.BuildContextForDocument(c.Request.Context(), contextID, user.ID)
	if err != nil {
		RespondInternalErr(c, "build document context", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"context":   hc,
		"formatted": hc.FormatForAI(),
	})
}

// GetStats returns embedding statistics for the user
// GET /api/embeddings/stats
func (h *EmbeddingHandler) GetStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	stats, err := h.embeddingService.GetEmbeddingStats(c.Request.Context(), user.ID)
	if err != nil {
		RespondInternalErr(c, "get embedding stats", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// HealthCheck checks if the embedding service is available
// GET /api/embeddings/health
func (h *EmbeddingHandler) HealthCheck(c *gin.Context) {
	healthy := h.embeddingService.HealthCheck(c.Request.Context())

	status := "healthy"
	httpStatus := http.StatusOK
	if !healthy {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"service": "embedding",
		"model":   "nomic-embed-text",
	})
}
