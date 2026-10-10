package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// CommentHandler handles all comment-related HTTP requests.
type CommentHandler struct {
	commentService *services.CommentService
}

// NewCommentHandler constructs a CommentHandler.
func NewCommentHandler(commentService *services.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// RegisterCommentRoutes wires comment routes onto the given api group.
func RegisterCommentRoutes(api *gin.RouterGroup, h *CommentHandler, auth gin.HandlerFunc) {
	comments := api.Group("/comments")
	comments.Use(auth, middleware.RequireAuth())
	{
		comments.GET("", h.GetComments)
		comments.POST("", h.CreateComment)
		comments.GET("/:id", h.GetComment)
		comments.PUT("/:id", h.UpdateComment)
		comments.DELETE("/:id", h.DeleteComment)
		comments.POST("/:id/reactions", h.AddCommentReaction)
		comments.DELETE("/:id/reactions/:emoji", h.RemoveCommentReaction)
	}

	// Task comments - /api/tasks/:id/comments
	api.GET("/tasks/:id/comments", auth, middleware.RequireAuth(), h.GetTaskComments)
	api.POST("/tasks/:id/comments", auth, middleware.RequireAuth(), h.CreateTaskComment)

	// Project comments - /api/projects/:id/comments
	api.GET("/projects/:id/comments", auth, middleware.RequireAuth(), h.GetProjectComments)
	api.POST("/projects/:id/comments", auth, middleware.RequireAuth(), h.CreateProjectComment)
}

// ========== COMMENT HANDLERS ==========

// CreateCommentRequest is the request body for creating a comment
type CreateCommentRequest struct {
	Content    string  `json:"content" binding:"required"`
	EntityType string  `json:"entity_type" binding:"required"` // task, project, note
	EntityID   string  `json:"entity_id" binding:"required,uuid"`
	ParentID   *string `json:"parent_id,omitempty"` // For replies
}

// CreateComment creates a new comment
// POST /api/comments
func (h *CommentHandler) CreateComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity_id"})
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id"})
			return
		}
		parentID = &pid
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), services.CreateCommentInput{
		UserID:     user.ID,
		EntityType: req.EntityType,
		EntityID:   entityID,
		Content:    req.Content,
		ParentID:   parentID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create comment", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

// GetComments retrieves comments for an entity
// GET /api/comments?entity_type=task&entity_id=uuid
func (h *CommentHandler) GetComments(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	entityType := c.Query("entity_type")
	entityIDStr := c.Query("entity_id")

	if entityType == "" || entityIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type and entity_id are required"})
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entity_id"})
		return
	}

	comments, err := h.commentService.GetCommentsByEntity(c.Request.Context(), entityType, entityID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get comments", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"count":    len(comments),
	})
}

// GetComment retrieves a single comment by ID
// GET /api/comments/:id
func (h *CommentHandler) GetComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "comment_id")
		return
	}

	comment, err := h.commentService.GetCommentByID(c.Request.Context(), commentID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Comment")
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment": comment})
}

// UpdateCommentRequest is the request body for updating a comment
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// UpdateComment updates a comment's content
// PUT /api/comments/:id
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "comment_id")
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	comment, err := h.commentService.UpdateComment(c.Request.Context(), commentID, user.ID, req.Content)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update comment", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment": comment})
}

// DeleteComment soft-deletes a comment
// DELETE /api/comments/:id
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "comment_id")
		return
	}

	if err := h.commentService.DeleteComment(c.Request.Context(), commentID, user.ID); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete comment", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}

// ========== REACTION HANDLERS ==========

// AddReactionRequest is the request body for adding a reaction
type AddReactionRequest struct {
	Emoji string `json:"emoji" binding:"required"`
}

// AddCommentReaction adds a reaction to a comment
// POST /api/comments/:id/reactions
func (h *CommentHandler) AddCommentReaction(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "comment_id")
		return
	}

	var req AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if err := h.commentService.AddReaction(c.Request.Context(), commentID, user.ID, req.Emoji); err != nil {
		utils.RespondInternalError(c, slog.Default(), "add reaction", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction added"})
}

// RemoveCommentReaction removes a reaction from a comment
// DELETE /api/comments/:id/reactions/:emoji
func (h *CommentHandler) RemoveCommentReaction(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "comment_id")
		return
	}

	emoji := c.Param("emoji")
	if emoji == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Emoji is required"})
		return
	}

	if err := h.commentService.RemoveReaction(c.Request.Context(), commentID, user.ID, emoji); err != nil {
		utils.RespondInternalError(c, slog.Default(), "remove reaction", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}

// ========== ENTITY-SPECIFIC COMMENT ENDPOINTS ==========

// GetTaskComments retrieves comments for a specific task
// GET /api/tasks/:id/comments
func (h *CommentHandler) GetTaskComments(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "task_id")
		return
	}

	comments, err := h.commentService.GetCommentsByEntity(c.Request.Context(), "task", taskID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get comments", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"count":    len(comments),
	})
}

// CreateTaskComment creates a comment on a task
// POST /api/tasks/:id/comments
func (h *CommentHandler) CreateTaskComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "task_id")
		return
	}

	var req struct {
		Content  string  `json:"content" binding:"required"`
		ParentID *string `json:"parent_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id"})
			return
		}
		parentID = &pid
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), services.CreateCommentInput{
		UserID:     user.ID,
		EntityType: "task",
		EntityID:   taskID,
		Content:    req.Content,
		ParentID:   parentID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create comment", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

// GetProjectComments retrieves comments for a specific project
// GET /api/projects/:id/comments
func (h *CommentHandler) GetProjectComments(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	comments, err := h.commentService.GetCommentsByEntity(c.Request.Context(), "project", projectID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get comments", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"count":    len(comments),
	})
}

// CreateProjectComment creates a comment on a project
// POST /api/projects/:id/comments
func (h *CommentHandler) CreateProjectComment(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	var req struct {
		Content  string  `json:"content" binding:"required"`
		ParentID *string `json:"parent_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent_id"})
			return
		}
		parentID = &pid
	}

	comment, err := h.commentService.CreateComment(c.Request.Context(), services.CreateCommentInput{
		UserID:     user.ID,
		EntityType: "project",
		EntityID:   projectID,
		Content:    req.Content,
		ParentID:   parentID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create comment", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}
