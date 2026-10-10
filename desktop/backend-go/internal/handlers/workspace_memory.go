package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// WORKSPACE MEMORY HANDLERS (CUS-25)
// =====================================================================

// WorkspaceMemoryHandlers handles workspace memory API endpoints
type WorkspaceMemoryHandlers struct {
	pool          *pgxpool.Pool
	memoryService *services.MemoryHierarchyService
}

// NewWorkspaceMemoryHandlers creates a new workspace memory handlers instance
func NewWorkspaceMemoryHandlers(pool *pgxpool.Pool) *WorkspaceMemoryHandlers {
	return &WorkspaceMemoryHandlers{
		pool:          pool,
		memoryService: services.NewMemoryHierarchyService(pool),
	}
}

// WorkspaceMemory represents a workspace memory entity
type WorkspaceMemory struct {
	ID             uuid.UUID       `json:"id"`
	WorkspaceID    uuid.UUID       `json:"workspace_id"`
	Title          string          `json:"title"`
	Summary        string          `json:"summary"`
	Content        string          `json:"content"`
	MemoryType     string          `json:"memory_type"`
	Category       *string         `json:"category,omitempty"`
	Visibility     string          `json:"visibility"`
	OwnerUserID    *string         `json:"owner_user_id,omitempty"`
	SharedWith     []string        `json:"shared_with,omitempty"`
	Importance     float64         `json:"importance"`
	Tags           []string        `json:"tags,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	AccessCount    int             `json:"access_count"`
	LastAccessedAt *string         `json:"last_accessed_at,omitempty"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	CreatedBy      string          `json:"created_by"`
}

// CreateWorkspaceMemoryRequest represents the request body for creating a workspace memory
type CreateWorkspaceMemoryRequest struct {
	Title      string          `json:"title" binding:"required"`
	Summary    string          `json:"summary" binding:"required"`
	Content    string          `json:"content" binding:"required"`
	MemoryType string          `json:"memory_type" binding:"required"`
	Category   *string         `json:"category"`
	Visibility string          `json:"visibility"` // workspace, private, shared
	Tags       []string        `json:"tags"`
	Metadata   json.RawMessage `json:"metadata"`
	Importance float64         `json:"importance"`
}

// ShareMemoryRequest represents the request body for sharing a memory
type ShareMemoryRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

// =====================================================================
// ENDPOINT HANDLERS
// =====================================================================

// CreateMemory creates a new workspace or private memory
// POST /api/workspaces/:id/memories
func (h *WorkspaceMemoryHandlers) CreateMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	var req CreateWorkspaceMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Default visibility to workspace if not specified
	if req.Visibility == "" {
		req.Visibility = "workspace"
	}

	// Validate visibility
	if req.Visibility != "workspace" && req.Visibility != "private" && req.Visibility != "shared" {
		utils.RespondBadRequest(c, slog.Default(), "Invalid visibility. Must be workspace, private, or shared")
		return
	}

	// Validate memory_type
	validTypes := []string{"general", "decision", "pattern", "context", "learning", "preference"}
	isValidType := false
	for _, t := range validTypes {
		if req.MemoryType == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		utils.RespondBadRequest(c, slog.Default(), "Invalid memory_type. Must be one of: general, decision, pattern, context, learning, preference")
		return
	}

	// Default importance to 0.5 if not specified
	if req.Importance == 0 {
		req.Importance = 0.5
	}

	// Default tags if nil
	if req.Tags == nil {
		req.Tags = []string{}
	}

	// Parse metadata JSON
	var metadata map[string]interface{}
	if req.Metadata != nil {
		if err := json.Unmarshal(req.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	// Use MemoryHierarchyService based on visibility
	var createdMemory *services.WorkspaceMemoryItem
	var createErr error

	if req.Visibility == "workspace" {
		// Create workspace-level memory
		createdMemory, createErr = h.memoryService.CreateWorkspaceMemory(
			c.Request.Context(),
			workspaceID,
			req.Title,
			req.Content,
			req.MemoryType,
			user.ID,
			req.Tags,
			metadata,
		)
	} else {
		// Create private memory (private or shared visibility)
		createdMemory, createErr = h.memoryService.CreatePrivateMemory(
			c.Request.Context(),
			workspaceID,
			user.ID,
			req.Title,
			req.Content,
			req.MemoryType,
			req.Tags,
			metadata,
		)
	}

	if createErr != nil {
		utils.RespondInternalError(c, slog.Default(), "create memory", createErr)
		return
	}

	c.JSON(http.StatusCreated, createdMemory)
}

// ListWorkspaceMemories lists workspace-level memories (shared with all team)
// GET /api/workspaces/:id/memories
func (h *WorkspaceMemoryHandlers) ListWorkspaceMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Query params
	memoryType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var memoryTypeParam *string
	if memoryType != "" {
		memoryTypeParam = &memoryType
	}

	// Use MemoryHierarchyService
	memories, err := h.memoryService.GetWorkspaceMemories(c.Request.Context(), workspaceID, user.ID, memoryTypeParam, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch memories", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
		"count":    len(memories),
	})
}

// ListPrivateMemories lists user's private memories
// GET /api/workspaces/:id/memories/private
func (h *WorkspaceMemoryHandlers) ListPrivateMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Query params
	memoryType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var memoryTypeParam *string
	if memoryType != "" {
		memoryTypeParam = &memoryType
	}

	// Use MemoryHierarchyService
	memories, err := h.memoryService.GetUserMemories(c.Request.Context(), workspaceID, user.ID, memoryTypeParam, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch memories", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
		"count":    len(memories),
	})
}

// ListAccessibleMemories lists all accessible memories (workspace + private + shared)
// GET /api/workspaces/:id/memories/accessible
func (h *WorkspaceMemoryHandlers) ListAccessibleMemories(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	// Query params
	memoryType := c.Query("type")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var memoryTypeParam *string
	if memoryType != "" {
		memoryTypeParam = &memoryType
	}

	// Use MemoryHierarchyService
	memories, err := h.memoryService.GetAccessibleMemories(c.Request.Context(), workspaceID, user.ID, memoryTypeParam, limit)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch memories", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memories": memories,
		"count":    len(memories),
	})
}

// ShareMemory shares a private memory with specific users
// POST /api/workspaces/:id/memories/:memoryId/share
func (h *WorkspaceMemoryHandlers) ShareMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	memoryID, err := uuid.Parse(c.Param("memoryId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory ID")
		return
	}

	var req ShareMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.RespondBadRequest(c, slog.Default(), "At least one user ID is required")
		return
	}

	// Verify the memory belongs to the workspace
	var memoryWorkspaceID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(),
		"SELECT workspace_id FROM workspace_memories WHERE id = $1",
		memoryID,
	).Scan(&memoryWorkspaceID)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "verify memory", err)
		}
		return
	}

	if memoryWorkspaceID != workspaceID {
		utils.RespondForbidden(c, slog.Default(), "Memory does not belong to this workspace")
		return
	}

	// Use MemoryHierarchyService
	err = h.memoryService.ShareMemory(c.Request.Context(), memoryID, user.ID, req.UserIDs)
	if err != nil {
		utils.RespondForbidden(c, slog.Default(), err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Memory shared successfully",
		"memory_id":   memoryID,
		"shared_with": req.UserIDs,
	})
}

// UnshareMemory unshares a memory (makes it private again)
// DELETE /api/workspaces/:id/memories/:memoryId/share
func (h *WorkspaceMemoryHandlers) UnshareMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	memoryID, err := uuid.Parse(c.Param("memoryId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory ID")
		return
	}

	// Verify the memory belongs to the workspace
	var memoryWorkspaceID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(),
		"SELECT workspace_id FROM workspace_memories WHERE id = $1",
		memoryID,
	).Scan(&memoryWorkspaceID)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "verify memory", err)
		}
		return
	}

	if memoryWorkspaceID != workspaceID {
		utils.RespondForbidden(c, slog.Default(), "Memory does not belong to this workspace")
		return
	}

	// Use MemoryHierarchyService
	err = h.memoryService.UnshareMemory(c.Request.Context(), memoryID, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unshare memory", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Memory unshared successfully",
		"memory_id": memoryID,
	})
}

// DeleteMemory deletes a memory (soft delete)
// DELETE /api/workspaces/:id/memories/:memoryId
func (h *WorkspaceMemoryHandlers) DeleteMemory(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace ID")
		return
	}

	memoryID, err := uuid.Parse(c.Param("memoryId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "memory ID")
		return
	}

	// Get memory details to verify ownership and workspace
	var ownerUserID *string
	var visibility string
	var memoryWorkspaceID uuid.UUID

	query := `
		SELECT workspace_id, visibility, owner_user_id
		FROM workspace_memories
		WHERE id = $1
	`
	err = h.pool.QueryRow(c.Request.Context(), query, memoryID).Scan(&memoryWorkspaceID, &visibility, &ownerUserID)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondNotFound(c, slog.Default(), "Memory")
		} else {
			utils.RespondInternalError(c, slog.Default(), "verify memory", err)
		}
		return
	}

	if memoryWorkspaceID != workspaceID {
		utils.RespondForbidden(c, slog.Default(), "Memory does not belong to this workspace")
		return
	}

	// Check permissions
	// Workspace memories can be deleted by admins/owners
	// Private/shared memories can only be deleted by the owner
	if visibility != "workspace" {
		if ownerUserID == nil || *ownerUserID != user.ID {
			utils.RespondForbidden(c, slog.Default(), "Only the owner can delete this memory")
			return
		}
	} else {
		// For workspace memories, check if user is admin or owner
		var role string
		err = h.pool.QueryRow(c.Request.Context(), `
			SELECT wm.role
			FROM workspace_members wm
			WHERE wm.workspace_id = $1 AND wm.user_id = $2 AND wm.status = 'active'
		`, workspaceID, user.ID).Scan(&role)

		if err != nil {
			utils.RespondForbidden(c, slog.Default(), "Not authorized to delete workspace memories")
			return
		}

		if role != "owner" && role != "admin" {
			utils.RespondForbidden(c, slog.Default(), "Only workspace owners and admins can delete workspace memories")
			return
		}
	}

	// Delete the memory (hard delete for now, could be soft delete with is_active flag)
	deleteQuery := `DELETE FROM workspace_memories WHERE id = $1`
	_, err = h.pool.Exec(c.Request.Context(), deleteQuery, memoryID)

	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete memory", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Memory deleted successfully",
		"memory_id": memoryID,
	})
}

// =====================================================================
// ROUTE REGISTRATION HELPER
// =====================================================================

// RegisterWorkspaceMemoryRoutes registers workspace memory routes
func RegisterWorkspaceMemoryRoutes(workspaceGroup *gin.RouterGroup, h *WorkspaceMemoryHandlers) {
	memories := workspaceGroup.Group("/memories")
	{
		memories.POST("", h.CreateMemory)
		memories.GET("", h.ListWorkspaceMemories)
		memories.GET("/private", h.ListPrivateMemories)
		memories.GET("/accessible", h.ListAccessibleMemories)
		memories.POST("/:memoryId/share", h.ShareMemory)
		memories.DELETE("/:memoryId/share", h.UnshareMemory)
		memories.DELETE("/:memoryId", h.DeleteMemory)
	}
}
