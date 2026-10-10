package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// NodeHandler handles business node operations
type NodeHandler struct {
	pool *pgxpool.Pool
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(pool *pgxpool.Pool) *NodeHandler {
	return &NodeHandler{pool: pool}
}

// RegisterNodeRoutes registers all node management routes on the given router group.
func RegisterNodeRoutes(api *gin.RouterGroup, h *NodeHandler, auth gin.HandlerFunc) {
	nodes := api.Group("/nodes")
	nodes.Use(auth, middleware.RequireAuth())
	{
		nodes.GET("", h.ListNodes)
		nodes.GET("/tree", h.GetNodeTree)
		nodes.GET("/active", h.GetActiveNode)
		nodes.POST("", h.CreateNode)
		nodes.GET("/:id", h.GetNode)
		nodes.PATCH("/:id", h.UpdateNode)
		nodes.POST("/:id/activate", h.ActivateNode)
		nodes.POST("/:id/deactivate", h.DeactivateNode)
		nodes.DELETE("/:id", h.DeleteNode)
		nodes.GET("/:id/children", h.GetNodeChildren)
		nodes.POST("/:id/reorder", h.ReorderNodes)
		nodes.POST("/:id/archive", h.ArchiveNode)
		nodes.POST("/:id/unarchive", h.UnarchiveNode)
		// Node linking
		nodes.GET("/:id/links", h.GetNodeLinks)
		nodes.GET("/:id/links/counts", h.GetNodeLinkCounts)
		nodes.POST("/:id/links/projects", h.LinkNodeProject)
		nodes.DELETE("/:id/links/projects/:projectId", h.UnlinkNodeProject)
		nodes.POST("/:id/links/contexts", h.LinkNodeContext)
		nodes.DELETE("/:id/links/contexts/:contextId", h.UnlinkNodeContext)
		nodes.POST("/:id/links/conversations", h.LinkNodeConversation)
		nodes.DELETE("/:id/links/conversations/:conversationId", h.UnlinkNodeConversation)
	}
}

// ListNodes returns all nodes for the current user
func (h *NodeHandler) ListNodes(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	queries := sqlc.New(h.pool)
	nodes, err := queries.ListNodes(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list nodes", nil)
		return
	}

	all := TransformNodes(nodes)
	total := int64(len(all))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(all[start:end], total, pg))
}

// CreateNode creates a new node
func (h *NodeHandler) CreateNode(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name            string   `json:"name" binding:"required"`
		Type            string   `json:"type" binding:"required"`
		ParentID        *string  `json:"parent_id"`
		ContextID       *string  `json:"context_id"`
		Health          *string  `json:"health"`
		Purpose         *string  `json:"purpose"`
		CurrentStatus   *string  `json:"current_status"`
		ThisWeekFocus   []string `json:"this_week_focus"`
		DecisionQueue   []string `json:"decision_queue"`
		DelegationReady []string `json:"delegation_ready"`
		SortOrder       *int32   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	var parentID, contextID pgtype.UUID
	if req.ParentID != nil {
		if parsed, err := uuid.Parse(*req.ParentID); err == nil {
			parentID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	var health sqlc.NullNodehealth
	if req.Health != nil {
		health = sqlc.NullNodehealth{
			Nodehealth: stringToNodeHealth(*req.Health),
			Valid:      true,
		}
	}

	var thisWeekFocus []byte
	if req.ThisWeekFocus != nil && len(req.ThisWeekFocus) > 0 {
		if focusJSON, err := json.Marshal(req.ThisWeekFocus); err == nil {
			thisWeekFocus = focusJSON
		}
	}

	var decisionQueue []byte
	if req.DecisionQueue != nil && len(req.DecisionQueue) > 0 {
		if queueJSON, err := json.Marshal(req.DecisionQueue); err == nil {
			decisionQueue = queueJSON
		}
	}

	var delegationReady []byte
	if req.DelegationReady != nil && len(req.DelegationReady) > 0 {
		if delegationJSON, err := json.Marshal(req.DelegationReady); err == nil {
			delegationReady = delegationJSON
		}
	}

	node, err := queries.CreateNode(c.Request.Context(), sqlc.CreateNodeParams{
		UserID:          user.ID,
		ParentID:        parentID,
		ContextID:       contextID,
		Name:            req.Name,
		Type:            stringToNodeType(req.Type),
		Health:          health,
		Purpose:         req.Purpose,
		CurrentStatus:   req.CurrentStatus,
		ThisWeekFocus:   thisWeekFocus,
		DecisionQueue:   decisionQueue,
		DelegationReady: delegationReady,
		SortOrder:       req.SortOrder,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create node", nil)
		return
	}

	activeNode, _ := queries.GetActiveNode(c.Request.Context(), user.ID)
	if activeNode.ID.Bytes == [16]byte{} {
		queries.ActivateNode(c.Request.Context(), user.ID)
		queries.SetNodeActive(c.Request.Context(), node.ID)
	}

	c.JSON(http.StatusCreated, TransformNode(node))
}

// GetNode returns a single node
func (h *NodeHandler) GetNode(c *gin.Context) {
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
	node, err := queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	if c.Query("include_children") == "true" {
		children, err := queries.GetNodeChildren(c.Request.Context(), sqlc.GetNodeChildrenParams{
			ParentID: pgtype.UUID{Bytes: id, Valid: true},
			UserID:   user.ID,
		})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"node":     TransformNode(node),
				"children": TransformNodes(children),
			})
			return
		}
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// UpdateNode updates a node
func (h *NodeHandler) UpdateNode(c *gin.Context) {
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

	var req struct {
		Name            *string  `json:"name"`
		Type            *string  `json:"type"`
		ContextID       *string  `json:"context_id"`
		Health          *string  `json:"health"`
		Purpose         *string  `json:"purpose"`
		CurrentStatus   *string  `json:"current_status"`
		ThisWeekFocus   []string `json:"this_week_focus"`
		DecisionQueue   []string `json:"decision_queue"`
		DelegationReady []string `json:"delegation_ready"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	existing, err := queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	nodeType := existing.Type
	if req.Type != nil {
		nodeType = stringToNodeType(*req.Type)
	}

	contextID := existing.ContextID
	if req.ContextID != nil {
		if parsed, err := uuid.Parse(*req.ContextID); err == nil {
			contextID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	health := existing.Health
	if req.Health != nil {
		health = sqlc.NullNodehealth{
			Nodehealth: stringToNodeHealth(*req.Health),
			Valid:      true,
		}
	}

	purpose := existing.Purpose
	if req.Purpose != nil {
		purpose = req.Purpose
	}

	currentStatus := existing.CurrentStatus
	if req.CurrentStatus != nil {
		currentStatus = req.CurrentStatus
	}

	thisWeekFocus := existing.ThisWeekFocus
	if req.ThisWeekFocus != nil {
		if focusJSON, err := json.Marshal(req.ThisWeekFocus); err == nil {
			thisWeekFocus = focusJSON
		}
	}

	decisionQueue := existing.DecisionQueue
	if req.DecisionQueue != nil {
		if queueJSON, err := json.Marshal(req.DecisionQueue); err == nil {
			decisionQueue = queueJSON
		}
	}

	delegationReady := existing.DelegationReady
	if req.DelegationReady != nil {
		if delegationJSON, err := json.Marshal(req.DelegationReady); err == nil {
			delegationReady = delegationJSON
		}
	}

	node, err := queries.UpdateNode(c.Request.Context(), sqlc.UpdateNodeParams{
		ID:              pgtype.UUID{Bytes: id, Valid: true},
		Name:            name,
		Type:            nodeType,
		ContextID:       contextID,
		Health:          health,
		Purpose:         purpose,
		CurrentStatus:   currentStatus,
		ThisWeekFocus:   thisWeekFocus,
		DecisionQueue:   decisionQueue,
		DelegationReady: delegationReady,
	})
	if err != nil {
		slog.Error("Failed to update node", "node_id", id.String(), "error", err)
		utils.RespondInternalError(c, slog.Default(), "update node", err)
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// ReorderNodes updates sort order for multiple nodes
func (h *NodeHandler) ReorderNodes(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Orders []struct {
			ID        string `json:"id" binding:"required"`
			SortOrder int32  `json:"sort_order" binding:"required"`
		} `json:"orders" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	for _, order := range req.Orders {
		id, err := uuid.Parse(order.ID)
		if err != nil {
			continue
		}

		_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
			ID:     pgtype.UUID{Bytes: id, Valid: true},
			UserID: user.ID,
		})
		if err != nil {
			continue
		}

		if err := queries.UpdateNodeSortOrder(c.Request.Context(), sqlc.UpdateNodeSortOrderParams{
			ID:        pgtype.UUID{Bytes: id, Valid: true},
			SortOrder: &order.SortOrder,
		}); err != nil {
			slog.Warn("Warning: failed to update node sort order", "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Nodes reordered"})
}

// DeleteNode deletes a node
func (h *NodeHandler) DeleteNode(c *gin.Context) {
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
	err = queries.DeleteNode(c.Request.Context(), sqlc.DeleteNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete node", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node deleted"})
}
