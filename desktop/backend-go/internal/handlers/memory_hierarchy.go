package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ================================================
// PROJECT/NODE SCOPED HANDLERS
// ================================================

// GetProjectMemories returns memories scoped to a specific project.
// GET /api/memories/project/:projectId
func (h *MemoryHandler) GetProjectMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND project_id = $2 AND is_active = true
		ORDER BY is_pinned DESC, importance_score DESC, created_at DESC
		LIMIT $3
	`

	rows, err := h.pool.Query(c.Request.Context(), query, user.ID, projectID, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get project memories", nil)
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
		"memories":   memories,
		"count":      len(memories),
		"project_id": projectID.String(),
	})
}

// GetNodeMemories returns memories scoped to a specific node.
// GET /api/memories/node/:nodeId
func (h *MemoryHandler) GetNodeMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("nodeId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := `
		SELECT id, user_id, title, summary, content, memory_type, category,
		       source_type, source_id, source_context, project_id, node_id,
		       importance_score, access_count, last_accessed_at,
		       is_active, is_pinned, expires_at, tags, metadata,
		       created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND node_id = $2 AND is_active = true
		ORDER BY is_pinned DESC, importance_score DESC, created_at DESC
		LIMIT $3
	`

	rows, err := h.pool.Query(c.Request.Context(), query, user.ID, nodeID, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get node memories", nil)
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
		"node_id":  nodeID.String(),
	})
}
