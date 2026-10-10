package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// WebSearchHandler handles web search and search history operations.
type WebSearchHandler struct {
	pool *pgxpool.Pool
}

// NewWebSearchHandler constructs a WebSearchHandler.
func NewWebSearchHandler(pool *pgxpool.Pool) *WebSearchHandler {
	return &WebSearchHandler{pool: pool}
}

// RegisterWebSearchRoutes wires the 6 web-search + history routes onto searchGroup.
// Hybrid/multimodal routes are NOT registered here — they stay on the god-struct.
func RegisterWebSearchRoutes(searchGroup *gin.RouterGroup, h *WebSearchHandler) {
	searchGroup.GET("/web", h.WebSearch)
	searchGroup.GET("/context", h.WebSearchWithContext)
	searchGroup.GET("/history", h.ListSearchHistory)
	searchGroup.GET("/history/:id", h.GetSearchHistoryEntry)
	searchGroup.DELETE("/history/:id", h.DeleteSearchHistoryEntry)
	searchGroup.DELETE("/history", h.ClearSearchHistory)
}

// WebSearch performs a web search.
func (h *WebSearchHandler) WebSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	maxResults := 10
	if max := c.Query("max"); max != "" {
		if parsed, err := strconv.Atoi(max); err == nil && parsed > 0 && parsed <= 20 {
			maxResults = parsed
		}
	}

	searchService := services.NewWebSearchService()
	results, err := searchService.Search(c.Request.Context(), query, maxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Search failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// WebSearchWithContext performs a web search and returns formatted context.
func (h *WebSearchHandler) WebSearchWithContext(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	maxResults := 5
	if max := c.Query("max"); max != "" {
		if parsed, err := strconv.Atoi(max); err == nil && parsed > 0 && parsed <= 10 {
			maxResults = parsed
		}
	}

	searchService := services.NewWebSearchService()
	results, err := searchService.Search(c.Request.Context(), query, maxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Search failed",
			"details": err.Error(),
		})
		return
	}

	// Format as context for AI
	contextText := searchService.FormatResultsAsContext(results)

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results.Results,
		"context": contextText,
		"meta": gin.H{
			"total_results": results.TotalResults,
			"search_time":   results.SearchTime,
			"provider":      results.Provider,
		},
	})
}

// SearchHistoryEntry represents a search history entry for the API.
type SearchHistoryEntry struct {
	ID             uuid.UUID `json:"id"`
	OriginalQuery  string    `json:"original_query"`
	OptimizedQuery *string   `json:"optimized_query,omitempty"`
	ResultCount    int       `json:"result_count"`
	Provider       string    `json:"provider"`
	SearchTimeMs   *float64  `json:"search_time_ms,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// SearchHistoryDetail represents full details of a search including results.
type SearchHistoryDetail struct {
	SearchHistoryEntry
	Results json.RawMessage `json:"results"`
}

// ListSearchHistory returns the user's search history.
func (h *WebSearchHandler) ListSearchHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Pagination
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Query search history
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, original_query, optimized_query, result_count, provider, search_time_ms, created_at
		FROM web_search_results
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query search history"})
		return
	}
	defer rows.Close()

	var entries []SearchHistoryEntry
	for rows.Next() {
		var entry SearchHistoryEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.OriginalQuery,
			&entry.OptimizedQuery,
			&entry.ResultCount,
			&entry.Provider,
			&entry.SearchTimeMs,
			&entry.CreatedAt,
		); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Get total count
	var total int
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM web_search_results WHERE user_id = $1
	`, userID).Scan(&total)
	if err != nil {
		total = len(entries)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": entries,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetSearchHistoryEntry returns details of a specific search including results.
func (h *WebSearchHandler) GetSearchHistoryEntry(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	searchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search ID"})
		return
	}

	var detail SearchHistoryDetail
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT id, original_query, optimized_query, result_count, provider, search_time_ms, created_at, results
		FROM web_search_results
		WHERE id = $1 AND user_id = $2
	`, searchID, userID).Scan(
		&detail.ID,
		&detail.OriginalQuery,
		&detail.OptimizedQuery,
		&detail.ResultCount,
		&detail.Provider,
		&detail.SearchTimeMs,
		&detail.CreatedAt,
		&detail.Results,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "search not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// DeleteSearchHistoryEntry deletes a specific search from history.
func (h *WebSearchHandler) DeleteSearchHistoryEntry(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	searchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search ID"})
		return
	}

	result, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM web_search_results WHERE id = $1 AND user_id = $2
	`, searchID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete search"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "search not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "search deleted"})
}

// ClearSearchHistory clears all search history for the user.
func (h *WebSearchHandler) ClearSearchHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	result, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM web_search_results WHERE user_id = $1
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear search history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "search history cleared",
		"deleted": result.RowsAffected(),
	})
}
