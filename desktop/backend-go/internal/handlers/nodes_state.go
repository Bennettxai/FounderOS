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

// GetActiveNode returns the currently active node
func (h *NodeHandler) GetActiveNode(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	node, err := queries.GetActiveNode(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active node found"})
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// ActivateNode sets a node as the active node
func (h *NodeHandler) ActivateNode(c *gin.Context) {
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

	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	if err := queries.ActivateNode(c.Request.Context(), user.ID); err != nil {
		slog.Warn("Warning: failed to deactivate other nodes", "error", err)
	}

	node, err := queries.SetNodeActive(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "activate node", nil)
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// DeactivateNode deactivates a node
func (h *NodeHandler) DeactivateNode(c *gin.Context) {
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

	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	node, err := queries.DeactivateNode(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "deactivate node", nil)
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// ArchiveNode archives a node
func (h *NodeHandler) ArchiveNode(c *gin.Context) {
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

	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	node, err := queries.ArchiveNode(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "archive node", nil)
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}

// UnarchiveNode restores an archived node
func (h *NodeHandler) UnarchiveNode(c *gin.Context) {
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

	_, err = queries.GetNode(c.Request.Context(), sqlc.GetNodeParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Node")
		return
	}

	node, err := queries.UnarchiveNode(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unarchive node", nil)
		return
	}

	c.JSON(http.StatusOK, TransformNode(node))
}
