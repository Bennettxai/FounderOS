package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
)

// RAGHandler handles RAG (Retrieval Augmented Generation) operations
type RAGHandler struct {
	hybridSearchService *services.HybridSearchService
	rerankerService     *services.ReRankerService
	agenticRAGService   *services.AgenticRAGService
	memoryService       *services.MemoryService
}

// NewRAGHandler creates a new RAGHandler
func NewRAGHandler(
	hybridSearch *services.HybridSearchService,
	reranker *services.ReRankerService,
	agenticRAG *services.AgenticRAGService,
	memory *services.MemoryService,
) *RAGHandler {
	return &RAGHandler{
		hybridSearchService: hybridSearch,
		rerankerService:     reranker,
		agenticRAGService:   agenticRAG,
		memoryService:       memory,
	}
}

// HybridSearchRequest represents a request for hybrid search
type HybridSearchRequest struct {
	Query          string  `json:"query" binding:"required"`
	SemanticWeight float64 `json:"semantic_weight,omitempty"`
	KeywordWeight  float64 `json:"keyword_weight,omitempty"`
	MaxResults     int     `json:"max_results,omitempty"`
	MinSimilarity  float64 `json:"min_similarity,omitempty"`
}

// AgenticRAGRequest represents a request for agentic RAG retrieval
type AgenticRAGRequest struct {
	Query              string     `json:"query" binding:"required"`
	MaxResults         int        `json:"max_results,omitempty"`
	MinQualityScore    float64    `json:"min_quality_score,omitempty"`
	ProjectID          *uuid.UUID `json:"project_id,omitempty"`
	TaskID             *uuid.UUID `json:"task_id,omitempty"`
	UsePersonalization bool       `json:"use_personalization,omitempty"`
}

// HybridSearch performs hybrid search combining semantic and keyword approaches
// POST /api/rag/search/hybrid
func (h *RAGHandler) HybridSearch(c *gin.Context) {
	if h.hybridSearchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Hybrid search service not available",
		})
		return
	}

	var req HybridSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from auth context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Set default options or use provided values
	opts := services.DefaultHybridSearchOptions()
	if req.SemanticWeight > 0 || req.KeywordWeight > 0 {
		opts.SemanticWeight = req.SemanticWeight
		opts.KeywordWeight = req.KeywordWeight
	}
	if req.MaxResults > 0 {
		opts.MaxResults = req.MaxResults
	}
	if req.MinSimilarity > 0 {
		opts.MinSimilarity = req.MinSimilarity
	}

	// Perform search
	results, err := h.hybridSearchService.Search(c.Request.Context(), req.Query, userID.(string), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
		"options": gin.H{
			"semantic_weight": opts.SemanticWeight,
			"keyword_weight":  opts.KeywordWeight,
			"max_results":     opts.MaxResults,
		},
	})
}

// HybridSearchExplain provides detailed explanation of hybrid search results
// POST /api/rag/search/hybrid/explain
func (h *RAGHandler) HybridSearchExplain(c *gin.Context) {
	if h.hybridSearchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Hybrid search service not available",
		})
		return
	}

	var req HybridSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	opts := services.DefaultHybridSearchOptions()
	if req.SemanticWeight > 0 || req.KeywordWeight > 0 {
		opts.SemanticWeight = req.SemanticWeight
		opts.KeywordWeight = req.KeywordWeight
	}
	if req.MaxResults > 0 {
		opts.MaxResults = req.MaxResults
	}

	// Get explanation
	explanation, err := h.hybridSearchService.ExplainSearch(c.Request.Context(), req.Query, userID.(string), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, explanation)
}

// AgenticRAGRetrieve performs intelligent adaptive retrieval
// POST /api/rag/retrieve
func (h *RAGHandler) AgenticRAGRetrieve(c *gin.Context) {
	if h.agenticRAGService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Agentic RAG service not available",
		})
		return
	}

	var req AgenticRAGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Build agentic RAG request
	ragReq := services.AgenticRAGRequest{
		Query:              req.Query,
		UserID:             userID.(string),
		MaxResults:         req.MaxResults,
		MinQualityScore:    req.MinQualityScore,
		ProjectContext:     req.ProjectID,
		TaskContext:        req.TaskID,
		UsePersonalization: req.UsePersonalization,
	}

	// Set defaults
	if ragReq.MaxResults == 0 {
		ragReq.MaxResults = 10
	}
	if ragReq.MinQualityScore == 0 {
		ragReq.MinQualityScore = 0.5
	}

	// Perform retrieval
	response, err := h.agenticRAGService.Retrieve(c.Request.Context(), ragReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// MemoryList lists memories for the authenticated user
// GET /api/rag/memories
func (h *RAGHandler) MemoryList(c *gin.Context) {
	if h.memoryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory service not available",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Query parameters
	memoryType := c.Query("type")
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if _, err := fmt.Sscanf(limitParam, "%d", &limit); err == nil && limit > 0 && limit <= 100 {
			// Valid limit
		} else {
			limit = 50
		}
	}

	var memoryTypePtr *string
	if memoryType != "" {
		memoryTypePtr = &memoryType
	}

	memories, err := h.memoryService.ListMemories(c.Request.Context(), userID.(string), memoryTypePtr, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
		"count":    len(memories),
	})
}

// MemoryGet retrieves a specific memory
// GET /api/rag/memories/:id
func (h *RAGHandler) MemoryGet(c *gin.Context) {
	if h.memoryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory service not available",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	memoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memory ID"})
		return
	}

	memory, err := h.memoryService.GetMemory(c.Request.Context(), userID.(string), memoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
		return
	}

	c.JSON(http.StatusOK, memory)
}

// MemoryCreateRequest represents a request to create a memory
type MemoryCreateRequest struct {
	Title           string     `json:"title" binding:"required"`
	Summary         string     `json:"summary" binding:"required"`
	Content         string     `json:"content" binding:"required"`
	MemoryType      string     `json:"memory_type" binding:"required"`
	Category        string     `json:"category,omitempty"`
	SourceType      string     `json:"source_type,omitempty"`
	SourceID        string     `json:"source_id,omitempty"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	NodeID          *uuid.UUID `json:"node_id,omitempty"`
	ImportanceScore float64    `json:"importance_score,omitempty"`
	Tags            []string   `json:"tags,omitempty"`
}

// MemoryCreate creates a new memory
// POST /api/rag/memories
func (h *RAGHandler) MemoryCreate(c *gin.Context) {
	if h.memoryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory service not available",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req MemoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse SourceID if provided
	var sourceID *uuid.UUID
	if req.SourceID != "" {
		if parsed, err := uuid.Parse(req.SourceID); err == nil {
			sourceID = &parsed
		}
	}

	// Build memory object
	memory := &services.Memory{
		UserID:          userID.(string),
		Title:           req.Title,
		Summary:         req.Summary,
		Content:         req.Content,
		MemoryType:      req.MemoryType,
		Category:        req.Category,
		SourceType:      req.SourceType,
		SourceID:        sourceID,
		ProjectID:       req.ProjectID,
		NodeID:          req.NodeID,
		ImportanceScore: req.ImportanceScore,
		Tags:            req.Tags,
	}

	// Set default importance if not provided
	if memory.ImportanceScore == 0 {
		memory.ImportanceScore = 0.5
	}

	// Create memory
	if err := h.memoryService.CreateMemory(c.Request.Context(), memory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, memory)
}

// ReRankRequest represents a request to re-rank search results
type ReRankRequest struct {
	Query             string                        `json:"query" binding:"required"`
	Results           []services.HybridSearchResult `json:"results" binding:"required"`
	RecencyWeight     float64                       `json:"recency_weight,omitempty"`
	QualityWeight     float64                       `json:"quality_weight,omitempty"`
	InteractionWeight float64                       `json:"interaction_weight,omitempty"`
	ContextRelevance  float64                       `json:"context_relevance,omitempty"`
	CurrentProjectID  *uuid.UUID                    `json:"current_project_id,omitempty"`
}

// ReRankResults re-ranks existing search results using multiple signals
// POST /api/rag/search/rerank
func (h *RAGHandler) ReRankResults(c *gin.Context) {
	if h.rerankerService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Re-ranking service not available",
		})
		return
	}

	var req ReRankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Build re-ranking options with defaults
	opts := services.DefaultReRankingOptions()

	// Apply custom weights if provided
	if req.RecencyWeight > 0 || req.QualityWeight > 0 || req.InteractionWeight > 0 || req.ContextRelevance > 0 {
		opts.RecencyWeight = req.RecencyWeight
		opts.QualityWeight = req.QualityWeight
		opts.InteractionWeight = req.InteractionWeight
		opts.ContextRelevance = req.ContextRelevance
		opts.SemanticWeight = 1.0 - (req.RecencyWeight + req.QualityWeight + req.InteractionWeight + req.ContextRelevance)

		// Ensure semantic weight is valid
		if opts.SemanticWeight < 0 {
			opts.SemanticWeight = 0
		}
	}

	if req.CurrentProjectID != nil {
		opts.CurrentProjectID = req.CurrentProjectID
	}

	// Perform re-ranking
	rerankedResults, err := h.rerankerService.ReRank(c.Request.Context(), req.Query, userID.(string), req.Results, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   req.Query,
		"results": rerankedResults,
		"count":   len(rerankedResults),
		"options": gin.H{
			"recency_weight":     opts.RecencyWeight,
			"quality_weight":     opts.QualityWeight,
			"interaction_weight": opts.InteractionWeight,
			"context_relevance":  opts.ContextRelevance,
			"semantic_weight":    opts.SemanticWeight,
		},
	})
}

// ReRankExplain provides detailed explanation of re-ranking decisions
// POST /api/rag/search/rerank/explain
func (h *RAGHandler) ReRankExplain(c *gin.Context) {
	if h.rerankerService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Re-ranking service not available",
		})
		return
	}

	var req ReRankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Build re-ranking options with defaults
	opts := services.DefaultReRankingOptions()
	if req.RecencyWeight > 0 || req.QualityWeight > 0 || req.InteractionWeight > 0 || req.ContextRelevance > 0 {
		opts.RecencyWeight = req.RecencyWeight
		opts.QualityWeight = req.QualityWeight
		opts.InteractionWeight = req.InteractionWeight
		opts.ContextRelevance = req.ContextRelevance
		opts.SemanticWeight = 1.0 - (req.RecencyWeight + req.QualityWeight + req.InteractionWeight + req.ContextRelevance)
		if opts.SemanticWeight < 0 {
			opts.SemanticWeight = 0
		}
	}
	if req.CurrentProjectID != nil {
		opts.CurrentProjectID = req.CurrentProjectID
	}

	// Get explanation
	explanation, err := h.rerankerService.ExplainReRanking(c.Request.Context(), req.Query, userID.(string), req.Results, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, explanation)
}

// NOTE: MultiModalSearch is handled by multimodal_search.go handler
// Routes are registered via RegisterMultiModalRoutes in handlers.go
// Duplicate removed to avoid conflicts

// SearchExplain provides debug information about search strategy
// GET /api/rag/search/explain
func (h *RAGHandler) SearchExplain(c *gin.Context) {
	query := c.Query("query")
	strategy := c.Query("strategy") // "hybrid", "semantic", "keyword"

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Default to hybrid search
	if strategy == "" {
		strategy = "hybrid"
	}

	// Only handle hybrid search explanation here
	// Multimodal search has its own explain endpoint via GetSupportedModalities
	if strategy == "hybrid" {
		if h.hybridSearchService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Hybrid search service not available",
			})
			return
		}

		opts := services.DefaultHybridSearchOptions()
		explanation, err := h.hybridSearchService.ExplainSearch(c.Request.Context(), query, userID.(string), opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, explanation)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":                "Invalid strategy",
			"supported_strategies": []string{"hybrid"},
			"note":                 "For multimodal search info, use /api/search/modalities",
		})
	}
}

// NOTE: decodeBase64Image is defined in multimodal_search.go
// Duplicate removed to avoid conflicts
