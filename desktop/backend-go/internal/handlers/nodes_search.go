package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// NodeTreeItem represents a node with its children for tree display
type NodeTreeItem struct {
	ID            string          `json:"id"`
	ParentID      *string         `json:"parent_id"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Health        *string         `json:"health"`
	Purpose       *string         `json:"purpose"`
	ThisWeekFocus json.RawMessage `json:"this_week_focus"`
	IsActive      bool            `json:"is_active"`
	IsArchived    bool            `json:"is_archived"`
	SortOrder     *int32          `json:"sort_order"`
	UpdatedAt     string          `json:"updated_at"`
	Children      []NodeTreeItem  `json:"children"`
	ChildrenCount int             `json:"children_count"`
}

// buildNodeTree converts a flat list of nodes into a hierarchical tree
func buildNodeTree(nodes []sqlc.Node, parentID *string) []NodeTreeItem {
	var result []NodeTreeItem

	for _, node := range nodes {
		nodeParentID := getNodeParentIDString(node.ParentID)

		// Check if this node's parent matches the requested parentID
		if (parentID == nil && nodeParentID == nil) || (parentID != nil && nodeParentID != nil && *parentID == *nodeParentID) {
			nodeIDStr := nodeUUIDToString(node.ID)
			children := buildNodeTree(nodes, &nodeIDStr)

			var health *string
			if node.Health.Valid {
				h := string(node.Health.Nodehealth)
				health = &h
			}

			isActive := false
			if node.IsActive != nil {
				isActive = *node.IsActive
			}
			isArchived := false
			if node.IsArchived != nil {
				isArchived = *node.IsArchived
			}

			item := NodeTreeItem{
				ID:            nodeIDStr,
				ParentID:      nodeParentID,
				Name:          node.Name,
				Type:          string(node.Type),
				Health:        health,
				Purpose:       node.Purpose,
				ThisWeekFocus: node.ThisWeekFocus,
				IsActive:      isActive,
				IsArchived:    isArchived,
				SortOrder:     node.SortOrder,
				UpdatedAt:     node.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
				Children:      children,
				ChildrenCount: len(children),
			}
			result = append(result, item)
		}
	}

	// Sort by sort_order
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			orderI := int32(0)
			orderJ := int32(0)
			if result[i].SortOrder != nil {
				orderI = *result[i].SortOrder
			}
			if result[j].SortOrder != nil {
				orderJ = *result[j].SortOrder
			}
			if orderI > orderJ {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// getNodeParentIDString converts a pgtype.UUID to a nullable string pointer
func getNodeParentIDString(parentID pgtype.UUID) *string {
	if !parentID.Valid {
		return nil
	}
	s := uuid.UUID(parentID.Bytes).String()
	return &s
}

// nodeUUIDToString converts a pgtype.UUID to a string
func nodeUUIDToString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

// GetNodeTree returns all nodes organised as a hierarchical tree
func (h *NodeHandler) GetNodeTree(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	nodes, err := queries.GetNodeTree(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get node tree", nil)
		return
	}

	// Build tree structure starting from root nodes (no parent)
	tree := buildNodeTree(nodes, nil)

	c.JSON(http.StatusOK, tree)
}

// GetNodeChildren returns the direct children of a node
func (h *NodeHandler) GetNodeChildren(c *gin.Context) {
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
	children, err := queries.GetNodeChildren(c.Request.Context(), sqlc.GetNodeChildrenParams{
		ParentID: pgtype.UUID{Bytes: id, Valid: true},
		UserID:   user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get children", nil)
		return
	}

	c.JSON(http.StatusOK, TransformNodes(children))
}
