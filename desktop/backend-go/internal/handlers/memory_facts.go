package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ================================================
// USER FACTS HANDLERS
// ================================================

// ListUserFacts returns all user facts.
// GET /api/user-facts
func (h *MemoryHandler) ListUserFacts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	factType := c.Query("type")
	activeOnly := c.Query("active") != "false"

	query := `
		SELECT id, user_id, fact_key, fact_value, fact_type,
		       source_memory_id, confidence_score, is_active,
		       last_confirmed_at, created_at, updated_at
		FROM user_facts
		WHERE user_id = $1
	`
	args := []interface{}{user.ID}
	argIdx := 2

	if activeOnly {
		query += ` AND is_active = true`
	}
	if factType != "" {
		query += ` AND fact_type = $` + strconv.Itoa(argIdx)
		args = append(args, factType)
	}

	query += ` ORDER BY fact_type, fact_key`

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list user facts", nil)
		return
	}
	defer rows.Close()

	facts := []UserFactResponse{}
	for rows.Next() {
		fact, err := scanUserFactRow(rows)
		if err != nil {
			continue
		}
		facts = append(facts, fact)
	}

	c.JSON(http.StatusOK, gin.H{
		"facts": facts,
		"count": len(facts),
	})
}

// UpdateUserFact upserts a user fact by key.
// PUT /api/user-facts/:key
func (h *MemoryHandler) UpdateUserFact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	factKey := c.Param("key")
	if factKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fact key is required"})
		return
	}

	var req UpdateFactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	factType := "fact"
	if req.FactType != nil {
		factType = *req.FactType
	}

	confidence := 1.0
	if req.ConfidenceScore != nil {
		confidence = *req.ConfidenceScore
	}

	query := `
		INSERT INTO user_facts (user_id, fact_key, fact_value, fact_type, confidence_score, last_confirmed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, fact_key)
		DO UPDATE SET
			fact_value = EXCLUDED.fact_value,
			fact_type = EXCLUDED.fact_type,
			confidence_score = EXCLUDED.confidence_score,
			last_confirmed_at = NOW(),
			updated_at = NOW()
		RETURNING id, user_id, fact_key, fact_value, fact_type,
		          source_memory_id, confidence_score, is_active,
		          last_confirmed_at, created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), query, user.ID, factKey, req.FactValue, factType, confidence)
	fact, err := scanUserFactRowSingle(row)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update fact", nil)
		return
	}

	c.JSON(http.StatusOK, fact)
}

// ConfirmUserFact marks a user fact as confirmed/active.
// POST /api/user-facts/:key/confirm
func (h *MemoryHandler) ConfirmUserFact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	factKey := c.Param("key")
	if factKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fact key is required"})
		return
	}

	query := `
		UPDATE user_facts
		SET is_active = true,
		    last_confirmed_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $1 AND fact_key = $2
		RETURNING id, user_id, fact_key, fact_value, fact_type,
		          source_memory_id, confidence_score, is_active,
		          last_confirmed_at, created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), query, user.ID, factKey)
	fact, err := scanUserFactRowSingle(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Fact")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "confirm fact", nil)
		return
	}

	c.JSON(http.StatusOK, fact)
}

// RejectUserFact marks a user fact as inactive (will not be injected into context).
// POST /api/user-facts/:key/reject
func (h *MemoryHandler) RejectUserFact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	factKey := c.Param("key")
	if factKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fact key is required"})
		return
	}

	query := `
		UPDATE user_facts
		SET is_active = false,
		    updated_at = NOW()
		WHERE user_id = $1 AND fact_key = $2
		RETURNING id, user_id, fact_key, fact_value, fact_type,
		          source_memory_id, confidence_score, is_active,
		          last_confirmed_at, created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), query, user.ID, factKey)
	fact, err := scanUserFactRowSingle(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Fact")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "reject fact", nil)
		return
	}

	c.JSON(http.StatusOK, fact)
}

// DeleteUserFact deletes a user fact by key.
// DELETE /api/user-facts/:key
func (h *MemoryHandler) DeleteUserFact(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	factKey := c.Param("key")
	if factKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fact key is required"})
		return
	}

	query := `DELETE FROM user_facts WHERE user_id = $1 AND fact_key = $2`
	result, err := h.pool.Exec(c.Request.Context(), query, user.ID, factKey)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete fact", nil)
		return
	}

	if result.RowsAffected() == 0 {
		utils.RespondNotFound(c, slog.Default(), "Fact")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fact deleted"})
}

// ================================================
// STATISTICS HANDLER
// ================================================

// GetMemoryStats returns memory statistics for the user.
// GET /api/memories/stats
func (h *MemoryHandler) GetMemoryStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	stats := MemoryStatsResponse{
		ByType:     make(map[string]int),
		ByCategory: make(map[string]int),
	}

	// Total and active memories
	var total, active, pinned int
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE is_active = true),
			COUNT(*) FILTER (WHERE is_pinned = true)
		FROM memories WHERE user_id = $1
	`, user.ID).Scan(&total, &active, &pinned)
	if err == nil {
		stats.TotalMemories = total
		stats.ActiveMemories = active
		stats.PinnedMemories = pinned
	}

	// By type
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT memory_type, COUNT(*)
		FROM memories WHERE user_id = $1 AND is_active = true
		GROUP BY memory_type
	`, user.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var count int
			if rows.Scan(&t, &count) == nil {
				stats.ByType[t] = count
			}
		}
	}

	// By category
	rows, err = h.pool.Query(c.Request.Context(), `
		SELECT COALESCE(category, 'uncategorized'), COUNT(*)
		FROM memories WHERE user_id = $1 AND is_active = true
		GROUP BY category
	`, user.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if rows.Scan(&cat, &count) == nil {
				stats.ByCategory[cat] = count
			}
		}
	}

	// Total facts
	h.pool.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM user_facts WHERE user_id = $1 AND is_active = true
	`, user.ID).Scan(&stats.TotalFacts)

	// Recent access count (last 7 days)
	h.pool.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM memory_access_log
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '7 days'
	`, user.ID).Scan(&stats.RecentAccessCount)

	c.JSON(http.StatusOK, stats)
}
