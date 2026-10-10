package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ===== NODE LINKING HANDLERS =====

// GetNodeLinks returns all linked items (projects, contexts, conversations) for a node
func (h *NodeHandler) GetNodeLinks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	queries := sqlc.New(h.pool)
	nodeID := pgtype.UUID{Bytes: id, Valid: true}

	// Verify ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     nodeID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	// Get all linked items
	projects, _ := queries.GetNodeLinkedProjects(c.Request.Context(), nodeID)
	contexts, _ := queries.GetNodeLinkedContexts(c.Request.Context(), nodeID)
	conversations, _ := queries.GetNodeLinkedConversations(c.Request.Context(), nodeID)

	c.JSON(http.StatusOK, gin.H{
		"projects":      transformNodeLinkedProjects(projects),
		"contexts":      transformNodeLinkedContexts(contexts),
		"conversations": transformNodeLinkedConversations(conversations),
	})
}

// GetNodeLinkCounts returns counts of linked items for a node
func (h *NodeHandler) GetNodeLinkCounts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	queries := sqlc.New(h.pool)
	nodeID := pgtype.UUID{Bytes: id, Valid: true}

	// Verify ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     nodeID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	counts, err := queries.GetNodeLinkCounts(c.Request.Context(), nodeID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get link counts", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"linked_projects_count":      counts.LinkedProjectsCount,
		"linked_contexts_count":      counts.LinkedContextsCount,
		"linked_conversations_count": counts.LinkedConversationsCount,
	})
}

// LinkNodeProject links a project to a node
func (h *NodeHandler) LinkNodeProject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.LinkNodeProject(c.Request.Context(), sqlc.LinkNodeProjectParams{
		NodeID:    pgtype.UUID{Bytes: nodeID, Valid: true},
		ProjectID: pgtype.UUID{Bytes: projectID, Valid: true},
		LinkedBy:  &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "link project", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project linked"})
}

// UnlinkNodeProject unlinks a project from a node
func (h *NodeHandler) UnlinkNodeProject(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "project_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.UnlinkNodeProject(c.Request.Context(), sqlc.UnlinkNodeProjectParams{
		NodeID:    pgtype.UUID{Bytes: nodeID, Valid: true},
		ProjectID: pgtype.UUID{Bytes: projectID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unlink project", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project unlinked"})
}

// LinkNodeContext links a context to a node
func (h *NodeHandler) LinkNodeContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	var req struct {
		ContextID string `json:"context_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	contextID, err := uuid.Parse(req.ContextID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.LinkNodeContext(c.Request.Context(), sqlc.LinkNodeContextParams{
		NodeID:    pgtype.UUID{Bytes: nodeID, Valid: true},
		ContextID: pgtype.UUID{Bytes: contextID, Valid: true},
		LinkedBy:  &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "link context", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Context linked"})
}

// UnlinkNodeContext unlinks a context from a node
func (h *NodeHandler) UnlinkNodeContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	contextID, err := uuid.Parse(c.Param("contextId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.UnlinkNodeContext(c.Request.Context(), sqlc.UnlinkNodeContextParams{
		NodeID:    pgtype.UUID{Bytes: nodeID, Valid: true},
		ContextID: pgtype.UUID{Bytes: contextID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unlink context", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Context unlinked"})
}

// LinkNodeConversation links a conversation to a node
func (h *NodeHandler) LinkNodeConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	var req struct {
		ConversationID string `json:"conversation_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.LinkNodeConversation(c.Request.Context(), sqlc.LinkNodeConversationParams{
		NodeID:         pgtype.UUID{Bytes: nodeID, Valid: true},
		ConversationID: pgtype.UUID{Bytes: conversationID, Valid: true},
		LinkedBy:       &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "link conversation", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation linked"})
}

// UnlinkNodeConversation unlinks a conversation from a node
func (h *NodeHandler) UnlinkNodeConversation(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "node_id")
		return
	}

	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "conversation_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify node ownership
	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: nodeID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	err = queries.UnlinkNodeConversation(c.Request.Context(), sqlc.UnlinkNodeConversationParams{
		NodeID:         pgtype.UUID{Bytes: nodeID, Valid: true},
		ConversationID: pgtype.UUID{Bytes: conversationID, Valid: true},
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unlink conversation", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation unlinked"})
}

// ===== TRANSFORM HELPERS FOR LINKED ITEMS =====

func transformNodeLinkedProjects(projects []sqlc.GetNodeLinkedProjectsRow) []gin.H {
	result := make([]gin.H, 0, len(projects))
	for _, p := range projects {
		item := gin.H{
			"id":        uuid.UUID(p.ID.Bytes).String(),
			"name":      p.Name,
			"linked_at": p.LinkedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
		if p.Description != nil {
			item["description"] = *p.Description
		}
		if p.Status.Valid {
			item["status"] = string(p.Status.Projectstatus)
		}
		if p.Priority.Valid {
			item["priority"] = string(p.Priority.Projectpriority)
		}
		result = append(result, item)
	}
	return result
}

func transformNodeLinkedContexts(contexts []sqlc.GetNodeLinkedContextsRow) []gin.H {
	result := make([]gin.H, 0, len(contexts))
	for _, ctx := range contexts {
		item := gin.H{
			"id":        uuid.UUID(ctx.ID.Bytes).String(),
			"name":      ctx.Name,
			"linked_at": ctx.LinkedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
		if ctx.Type.Valid {
			item["type"] = string(ctx.Type.Contexttype)
		}
		if ctx.Icon != nil {
			item["icon"] = *ctx.Icon
		}
		if ctx.WordCount != nil {
			item["word_count"] = *ctx.WordCount
		}
		result = append(result, item)
	}
	return result
}

func transformNodeLinkedConversations(conversations []sqlc.GetNodeLinkedConversationsRow) []gin.H {
	result := make([]gin.H, 0, len(conversations))
	for _, conv := range conversations {
		item := gin.H{
			"id":         uuid.UUID(conv.ID.Bytes).String(),
			"linked_at":  conv.LinkedAt.Time.Format("2006-01-02T15:04:05Z"),
			"created_at": conv.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
		if conv.Title != nil {
			item["title"] = *conv.Title
		} else {
			item["title"] = "New Conversation"
		}
		result = append(result, item)
	}
	return result
}
