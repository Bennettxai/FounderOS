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

// ShareContext makes a context publicly accessible
func (h *ContextHandler) ShareContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership
	_, err = queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	// Generate unique share ID
	shareID := generateShareID()

	ctx, err := queries.ShareContext(c.Request.Context(), sqlc.ShareContextParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		ShareID: &shareID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "share context", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// UnshareContext makes a context private
func (h *ContextHandler) UnshareContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership
	_, err = queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	ctx, err := queries.UnshareContext(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unshare context", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// GetPublicContext returns a publicly shared context by share ID
func (h *ContextHandler) GetPublicContext(c *gin.Context) {
	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Share ID required"})
		return
	}

	queries := sqlc.New(h.pool)
	ctx, err := queries.GetPublicContext(c.Request.Context(), &shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Context not found or not public"})
		return
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// DuplicateContext creates a copy of an existing context
func (h *ContextHandler) DuplicateContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Get original context
	original, err := queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	// Create duplicate
	newName := original.Name + " (Copy)"
	duplicate, err := queries.CreateContext(c.Request.Context(), sqlc.CreateContextParams{
		UserID:               user.ID,
		Name:                 newName,
		Type:                 original.Type,
		Content:              original.Content,
		StructuredData:       original.StructuredData,
		SystemPromptTemplate: original.SystemPromptTemplate,
		Blocks:               original.Blocks,
		CoverImage:           original.CoverImage,
		Icon:                 original.Icon,
		ParentID:             original.ParentID,
		IsTemplate:           original.IsTemplate,
		PropertySchema:       original.PropertySchema,
		Properties:           original.Properties,
		ClientID:             original.ClientID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "duplicate context", nil)
		return
	}

	c.JSON(http.StatusCreated, TransformContext(duplicate))
}

// ArchiveContext archives a context
func (h *ContextHandler) ArchiveContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership
	_, err = queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	ctx, err := queries.ArchiveContext(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "archive context", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// UnarchiveContext unarchives a context
func (h *ContextHandler) UnarchiveContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Verify ownership
	_, err = queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	ctx, err := queries.UnarchiveContext(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "unarchive context", nil)
		return
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// AggregateContext provides aggregated data for a context
func (h *ContextHandler) AggregateContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "context_id")
		return
	}

	queries := sqlc.New(h.pool)

	// Get the context
	ctx, err := queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	// Get children
	children, _ := queries.GetContextChildren(c.Request.Context(), sqlc.GetContextChildrenParams{
		ParentID: pgtype.UUID{Bytes: id, Valid: true},
		UserID:   user.ID,
	})

	// Get related artifacts
	artifacts, _ := queries.ListArtifacts(c.Request.Context(), sqlc.ListArtifactsParams{
		UserID:    user.ID,
		ContextID: pgtype.UUID{Bytes: id, Valid: true},
	})

	c.JSON(http.StatusOK, gin.H{
		"context":   TransformContext(ctx),
		"children":  TransformContexts(children),
		"artifacts": TransformArtifacts(artifacts),
	})
}
