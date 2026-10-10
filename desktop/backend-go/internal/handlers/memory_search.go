package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ================================================
// SEMANTIC SEARCH HANDLERS
// ================================================

// SearchMemories performs semantic search on memories using vector similarity.
// POST /api/memories/search
func (h *MemoryHandler) SearchMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req MemorySearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	if h.embeddingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Embedding service not available"})
		return
	}

	queryEmbedding, err := h.embeddingService.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "generate embedding", nil)
		return
	}

	embeddingJSON, _ := json.Marshal(queryEmbedding)

	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at,
		       1 - (embedding <=> $1::vector) as similarity
		FROM memories
		WHERE user_id = $2 AND is_active = true AND embedding IS NOT NULL
	`
	args := []interface{}{string(embeddingJSON), user.ID}
	argIdx := 3

	if req.MemoryType != nil {
		query += ` AND memory_type = $` + strconv.Itoa(argIdx)
		args = append(args, *req.MemoryType)
		argIdx++
	}
	if req.ProjectID != nil {
		query += ` AND project_id = $` + strconv.Itoa(argIdx)
		args = append(args, *req.ProjectID)
		argIdx++
	}
	if req.NodeID != nil {
		query += ` AND node_id = $` + strconv.Itoa(argIdx)
		args = append(args, *req.NodeID)
		argIdx++
	}

	query += ` ORDER BY embedding <=> $1::vector LIMIT $` + strconv.Itoa(argIdx)
	args = append(args, req.Limit)

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	results := []MemorySearchResult{}
	for rows.Next() {
		var similarity float64
		memory, err := scanMemoryRowWithExtra(rows, &similarity)
		if err != nil {
			continue
		}
		results = append(results, MemorySearchResult{
			MemoryResponse: memory,
			Similarity:     similarity,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
	})
}

// GetRelevantMemories retrieves memories most relevant to a given context using
// a hybrid score: 60% vector similarity + 20% importance + 20% pinned bonus.
// POST /api/memories/relevant
func (h *MemoryHandler) GetRelevantMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req RelevantMemoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	if h.embeddingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Embedding service not available"})
		return
	}

	contextEmbedding, err := h.embeddingService.GenerateEmbedding(c.Request.Context(), req.Context)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "generate embedding", nil)
		return
	}

	embeddingJSON, _ := json.Marshal(contextEmbedding)

	// Hybrid scoring: semantic similarity + importance + pin bonus
	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at,
		       (1 - (embedding <=> $1::vector)) * 0.6 +
		       importance_score * 0.2 +
		       CASE WHEN is_pinned THEN 0.2 ELSE 0 END as relevance_score
		FROM memories
		WHERE user_id = $2 AND is_active = true AND embedding IS NOT NULL
	`
	args := []interface{}{string(embeddingJSON), user.ID}
	argIdx := 3

	if req.ProjectID != nil {
		query += ` AND (project_id = $` + strconv.Itoa(argIdx) + ` OR project_id IS NULL)`
		args = append(args, *req.ProjectID)
		argIdx++
	}
	if req.NodeID != nil {
		query += ` AND (node_id = $` + strconv.Itoa(argIdx) + ` OR node_id IS NULL)`
		args = append(args, *req.NodeID)
		argIdx++
	}
	if len(req.MemoryTypes) > 0 {
		query += ` AND memory_type = ANY($` + strconv.Itoa(argIdx) + `)`
		args = append(args, req.MemoryTypes)
		argIdx++
	}

	query += ` ORDER BY relevance_score DESC LIMIT $` + strconv.Itoa(argIdx)
	args = append(args, req.Limit)

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get relevant memories", nil)
		return
	}
	defer rows.Close()

	results := []MemoryRelevantResult{}
	for rows.Next() {
		var relevance float64
		memory, err := scanMemoryRowWithExtra(rows, &relevance)
		if err != nil {
			continue
		}
		results = append(results, MemoryRelevantResult{
			MemoryResponse: memory,
			RelevanceScore: relevance,
		})

		// Log access for learning
		if memID, err := uuid.Parse(memory.ID); err == nil {
			go h.recordMemoryAccess(memID, user.ID, "auto_inject", req.ConversationID, &relevance)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": results,
		"count":    len(results),
	})
}
