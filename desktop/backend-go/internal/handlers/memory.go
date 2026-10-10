package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// MemoryHandler handles memory-related HTTP endpoints
type MemoryHandler struct {
	pool             *pgxpool.Pool
	embeddingService *services.EmbeddingService
}

// NewMemoryHandler creates a new MemoryHandler
func NewMemoryHandler(pool *pgxpool.Pool, embeddingService *services.EmbeddingService) *MemoryHandler {
	return &MemoryHandler{
		pool:             pool,
		embeddingService: embeddingService,
	}
}

// ================================================
// MEMORY CRUD HANDLERS
// ================================================

// ListMemories returns all memories for the current user.
// GET /api/memories
func (h *MemoryHandler) ListMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse query params
	memoryType := c.Query("type")
	category := c.Query("category")
	pinnedOnly := c.Query("pinned") == "true"
	activeOnly := c.Query("active") != "false" // Default to active only
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Build query
	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at
		FROM memories
		WHERE user_id = $1
	`
	args := []interface{}{user.ID}
	argIdx := 2

	if activeOnly {
		query += ` AND is_active = true`
	}
	if pinnedOnly {
		query += ` AND is_pinned = true`
	}
	if memoryType != "" {
		query += ` AND memory_type = $` + strconv.Itoa(argIdx)
		args = append(args, memoryType)
		argIdx++
	}
	if category != "" {
		query += ` AND category = $` + strconv.Itoa(argIdx)
		args = append(args, category)
		argIdx++
	}

	query += ` ORDER BY is_pinned DESC, importance_score DESC, created_at DESC`
	query += ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list memories", nil)
		return
	}
	defer rows.Close()

	memories := []MemoryResponse{}
	for rows.Next() {
		memory, err := scanMemoryRow(rows)
		if err != nil {
			continue
		}
		memories = append(memories, memory)
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
		"count":    len(memories),
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateMemory creates a new memory.
// POST /api/memories
func (h *MemoryHandler) CreateMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate memory type
	validTypes := map[string]bool{
		"fact": true, "preference": true, "decision": true,
		"pattern": true, "insight": true, "interaction": true, "learning": true,
	}
	if !validTypes[req.MemoryType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid memory_type"})
		return
	}

	// Validate source type
	validSources := map[string]bool{
		"conversation": true, "voice_note": true, "document": true,
		"task": true, "project": true, "manual": true, "inferred": true,
	}
	if !validSources[req.SourceType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_type"})
		return
	}

	// Generate embedding if service is available
	var embeddingJSON []byte
	var embeddingModel *string
	if h.embeddingService != nil {
		textToEmbed := req.Title + " " + req.Summary + " " + req.Content
		embedding, err := h.embeddingService.GenerateEmbedding(c.Request.Context(), textToEmbed)
		if err == nil && len(embedding) > 0 {
			embeddingJSON, _ = json.Marshal(embedding)
			model := "text-embedding-ada-002"
			embeddingModel = &model
		}
	}

	// Serialize tags and metadata
	var tagsJSON []byte
	if req.Tags != nil {
		tagsJSON, _ = json.Marshal(req.Tags)
	}
	metadataJSON, _ := json.Marshal(req.Metadata)
	if req.Metadata == nil {
		metadataJSON = nil
	}

	// Parse optional UUIDs
	var sourceID, projectID, nodeID *uuid.UUID
	if req.SourceID != nil {
		if parsed, err := uuid.Parse(*req.SourceID); err == nil {
			sourceID = &parsed
		}
	}
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = &parsed
		}
	}
	if req.NodeID != nil {
		if parsed, err := uuid.Parse(*req.NodeID); err == nil {
			nodeID = &parsed
		}
	}

	query := `
		INSERT INTO memories (
			user_id, title, summary, content, memory_type, category,
			source_type, source_id, source_context, project_id, node_id,
			tags, metadata, embedding, embedding_model
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, user_id, title, summary, content, memory_type, category,
		          source_type, source_id, source_context, project_id, node_id,
		          importance_score, access_count, last_accessed_at,
		          is_active, is_pinned, expires_at, tags, metadata,
		          created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), query,
		user.ID, req.Title, req.Summary, req.Content, req.MemoryType, req.Category,
		req.SourceType, sourceID, req.SourceContext, projectID, nodeID,
		tagsJSON, metadataJSON, embeddingJSON, embeddingModel,
	)

	memory, err := scanMemoryRowSingle(row)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create memory", err)
		return
	}

	c.JSON(http.StatusCreated, memory)
}

// GetMemory returns a specific memory.
// GET /api/memories/:id
func (h *MemoryHandler) GetMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory_id")
		return
	}

	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at
		FROM memories
		WHERE id = $1 AND user_id = $2
	`

	row := h.pool.QueryRow(c.Request.Context(), query, id, user.ID)
	memory, err := scanMemoryRowSingle(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "get memory", nil)
		}
		return
	}

	// Update access count and last accessed
	go h.recordMemoryAccess(id, user.ID, "user_view", nil, nil)

	c.JSON(http.StatusOK, memory)
}

// UpdateMemory updates a memory.
// PUT /api/memories/:id
func (h *MemoryHandler) UpdateMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory_id")
		return
	}

	var req UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Verify ownership and get existing
	var existingTitle, existingSummary, existingContent, existingType string
	var existingCategory *string
	var existingScore float64
	var existingActive bool
	var existingTags, existingMetadata []byte

	checkQuery := `
		SELECT title, summary, content, memory_type, category, importance_score, is_active, tags, metadata
		FROM memories WHERE id = $1 AND user_id = $2
	`
	err = h.pool.QueryRow(c.Request.Context(), checkQuery, id, user.ID).Scan(
		&existingTitle, &existingSummary, &existingContent, &existingType,
		&existingCategory, &existingScore, &existingActive, &existingTags, &existingMetadata,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "verify memory", nil)
		}
		return
	}

	// Apply updates
	title := existingTitle
	if req.Title != nil {
		title = *req.Title
	}
	summary := existingSummary
	if req.Summary != nil {
		summary = *req.Summary
	}
	content := existingContent
	if req.Content != nil {
		content = *req.Content
	}
	memoryType := existingType
	if req.MemoryType != nil {
		memoryType = *req.MemoryType
	}
	category := existingCategory
	if req.Category != nil {
		category = req.Category
	}
	importanceScore := existingScore
	if req.ImportanceScore != nil {
		importanceScore = *req.ImportanceScore
	}
	isActive := existingActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tagsJSON := existingTags
	if req.Tags != nil {
		tagsJSON, _ = json.Marshal(req.Tags)
	}
	metadataJSON := existingMetadata
	if req.Metadata != nil {
		metadataJSON, _ = json.Marshal(req.Metadata)
	}

	// Regenerate embedding if content changed
	var embeddingJSON []byte
	if req.Title != nil || req.Summary != nil || req.Content != nil {
		if h.embeddingService != nil {
			textToEmbed := title + " " + summary + " " + content
			embedding, err := h.embeddingService.GenerateEmbedding(c.Request.Context(), textToEmbed)
			if err == nil && len(embedding) > 0 {
				embeddingJSON, _ = json.Marshal(embedding)
			}
		}
	}

	updateQuery := `
		UPDATE memories
		SET title = $1, summary = $2, content = $3, memory_type = $4, category = $5,
		    importance_score = $6, is_active = $7, tags = $8, metadata = $9,
		    embedding = COALESCE($10, embedding), updated_at = NOW()
		WHERE id = $11 AND user_id = $12
		RETURNING id, user_id, title, summary, content, memory_type, category,
		          source_type, source_id, source_context, project_id, node_id,
		          importance_score, access_count, last_accessed_at,
		          is_active, is_pinned, expires_at, tags, metadata,
		          created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), updateQuery,
		title, summary, content, memoryType, category,
		importanceScore, isActive, tagsJSON, metadataJSON,
		embeddingJSON, id, user.ID,
	)

	memory, err := scanMemoryRowSingle(row)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update memory", nil)
		return
	}

	c.JSON(http.StatusOK, memory)
}

// DeleteMemory deletes a memory.
// DELETE /api/memories/:id
func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory_id")
		return
	}

	query := `DELETE FROM memories WHERE id = $1 AND user_id = $2`
	result, err := h.pool.Exec(c.Request.Context(), query, id, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete memory", nil)
		return
	}

	if result.RowsAffected() == 0 {
		utils.RespondNotFound(c, slog.Default(), "Memory")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Memory deleted"})
}

// PinMemory pins or unpins a memory.
// POST /api/memories/:id/pin
func (h *MemoryHandler) PinMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory_id")
		return
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to toggle
		req.Pinned = true
	}

	query := `
		UPDATE memories
		SET is_pinned = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, title, summary, content, memory_type, category,
		          source_type, source_id, source_context, project_id, node_id,
		          importance_score, access_count, last_accessed_at,
		          is_active, is_pinned, expires_at, tags, metadata,
		          created_at, updated_at
	`

	row := h.pool.QueryRow(c.Request.Context(), query, req.Pinned, id, user.ID)
	memory, err := scanMemoryRowSingle(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "update memory", nil)
		}
		return
	}

	c.JSON(http.StatusOK, memory)
}
