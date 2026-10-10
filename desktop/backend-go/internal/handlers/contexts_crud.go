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
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// CreateContext creates a new context
func (h *ContextHandler) CreateContext(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name                 string          `json:"name" binding:"required"`
		Type                 *string         `json:"type"`
		Content              *string         `json:"content"`
		StructuredData       json.RawMessage `json:"structured_data"`
		SystemPromptTemplate *string         `json:"system_prompt_template"`
		Blocks               json.RawMessage `json:"blocks"`
		CoverImage           *string         `json:"cover_image"`
		Icon                 *string         `json:"icon"`
		ParentID             *string         `json:"parent_id"`
		IsTemplate           *bool           `json:"is_template"`
		PropertySchema       json.RawMessage `json:"property_schema"`
		Properties           json.RawMessage `json:"properties"`
		ClientID             *string         `json:"client_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional IDs
	var parentID, clientID pgtype.UUID
	if req.ParentID != nil {
		if parsed, err := uuid.Parse(*req.ParentID); err == nil {
			parentID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.ClientID != nil {
		if parsed, err := uuid.Parse(*req.ClientID); err == nil {
			clientID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Parse context type
	var ctxType sqlc.NullContexttype
	if req.Type != nil {
		ctxType = sqlc.NullContexttype{
			Contexttype: stringToContextType(*req.Type),
			Valid:       true,
		}
	}

	// Handle JSON fields (pass nil for empty jsonb — SimpleProtocol compatibility)
	var structuredData []byte
	if req.StructuredData != nil {
		structuredData = req.StructuredData
	}
	var blocks []byte
	if req.Blocks != nil {
		blocks = req.Blocks
	}
	var propertySchema []byte
	if req.PropertySchema != nil {
		propertySchema = req.PropertySchema
	}
	var properties []byte
	if req.Properties != nil {
		properties = req.Properties
	}

	// Default type to 'document' if not specified (lowercase for DB compatibility)
	if !ctxType.Valid {
		ctxType = sqlc.NullContexttype{
			Contexttype: sqlc.ContexttypeDocument, // lowercase 'document' for DB enum
			Valid:       true,
		}
	}

	// Default is_template to false if not specified
	isTemplate := false
	if req.IsTemplate != nil {
		isTemplate = *req.IsTemplate
	}

	ctx, err := queries.CreateContext(c.Request.Context(), sqlc.CreateContextParams{
		UserID:               user.ID,
		Name:                 req.Name,
		Type:                 ctxType,
		Content:              req.Content,
		StructuredData:       structuredData,
		SystemPromptTemplate: req.SystemPromptTemplate,
		Blocks:               blocks,
		CoverImage:           req.CoverImage,
		Icon:                 req.Icon,
		ParentID:             parentID,
		IsTemplate:           &isTemplate,
		PropertySchema:       propertySchema,
		Properties:           properties,
		ClientID:             clientID,
	})
	if err != nil {
		slog.Error("[Contexts] Failed to create context", "error", err)
		utils.RespondInternalError(c, slog.Default(), "create context", nil)
		return
	}

	// Invalidate cache for this user's contexts list
	if h.queryCache != nil {
		go h.invalidateContextsCachePattern(c.Request.Context(), user.ID)
	}

	// BusinessOSSync: when posted with an authorized X-Workspace-ID whose
	// knowledge module is in workspace sync mode, stamp workspace_id so every
	// member of that workspace sees this doc. No-op otherwise (stays private).
	wsID := h.shareContextToWorkspace(c, ctx.ID, user.ID)

	// Mirror the new Page into the OptimalEngine knowledge graph. No-op when
	// OPTIMAL_ENGINE_URL isn't configured.
	if h.engineSync != nil {
		sig := signalFromContext(ctx)
		sig.WorkspaceID = wsID
		h.engineSync.Enqueue(c.Request.Context(), sig)
	}

	c.JSON(http.StatusCreated, TransformContext(ctx))
}

// UpdateContext updates an existing context
func (h *ContextHandler) UpdateContext(c *gin.Context) {
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

	var req struct {
		Name                 *string         `json:"name"`
		Type                 *string         `json:"type"`
		Content              *string         `json:"content"`
		StructuredData       json.RawMessage `json:"structured_data"`
		SystemPromptTemplate *string         `json:"system_prompt_template"`
		Blocks               json.RawMessage `json:"blocks"`
		CoverImage           *string         `json:"cover_image"`
		Icon                 *string         `json:"icon"`
		ParentID             *string         `json:"parent_id"`
		IsTemplate           *bool           `json:"is_template"`
		PropertySchema       json.RawMessage `json:"property_schema"`
		Properties           json.RawMessage `json:"properties"`
		ClientID             *string         `json:"client_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Get existing context first
	existing, err := queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	// Build update params with existing values as defaults
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	var ctxType sqlc.NullContexttype
	if req.Type != nil {
		ctxType = sqlc.NullContexttype{
			Contexttype: stringToContextType(*req.Type),
			Valid:       true,
		}
	} else {
		ctxType = existing.Type
	}

	content := existing.Content
	if req.Content != nil {
		content = req.Content
	}

	structuredData := existing.StructuredData
	if req.StructuredData != nil {
		structuredData = req.StructuredData
	}

	systemPromptTemplate := existing.SystemPromptTemplate
	if req.SystemPromptTemplate != nil {
		systemPromptTemplate = req.SystemPromptTemplate
	}

	coverImage := existing.CoverImage
	if req.CoverImage != nil {
		coverImage = req.CoverImage
	}

	icon := existing.Icon
	if req.Icon != nil {
		icon = req.Icon
	}

	parentID := existing.ParentID
	if req.ParentID != nil {
		if parsed, err := uuid.Parse(*req.ParentID); err == nil {
			parentID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	isTemplate := existing.IsTemplate
	if req.IsTemplate != nil {
		isTemplate = req.IsTemplate
	}

	propertySchema := existing.PropertySchema
	if req.PropertySchema != nil {
		propertySchema = req.PropertySchema
	}

	properties := existing.Properties
	if req.Properties != nil {
		properties = req.Properties
	}

	clientID := existing.ClientID
	if req.ClientID != nil {
		if parsed, err := uuid.Parse(*req.ClientID); err == nil {
			clientID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	ctx, err := queries.UpdateContext(c.Request.Context(), sqlc.UpdateContextParams{
		ID:                   pgtype.UUID{Bytes: id, Valid: true},
		Name:                 name,
		Type:                 ctxType,
		Content:              content,
		StructuredData:       structuredData,
		SystemPromptTemplate: systemPromptTemplate,
		CoverImage:           coverImage,
		Icon:                 icon,
		ParentID:             parentID,
		IsTemplate:           isTemplate,
		PropertySchema:       propertySchema,
		Properties:           properties,
		ClientID:             clientID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update context", nil)
		return
	}

	// Also persist blocks if provided in the request
	if req.Blocks != nil {
		ctx, err = queries.UpdateContextBlocks(c.Request.Context(), sqlc.UpdateContextBlocksParams{
			ID:     pgtype.UUID{Bytes: id, Valid: true},
			Blocks: req.Blocks,
		})
		if err != nil {
			slog.Error("[Contexts] Failed to update blocks", "id", id, "error", err)
			utils.RespondInternalError(c, slog.Default(), "update context blocks", nil)
			return
		}
	}

	// Invalidate cache for this user's contexts list
	if h.queryCache != nil {
		go h.invalidateContextsCachePattern(c.Request.Context(), user.ID)
	}

	// BusinessOSSync: keep workspace sharing in sync on edit.
	wsID := h.shareContextToWorkspace(c, ctx.ID, user.ID)

	// Re-sync to the engine so edits propagate to the Pages graph view.
	if h.engineSync != nil {
		sig := signalFromContext(ctx)
		sig.WorkspaceID = wsID
		h.engineSync.Enqueue(c.Request.Context(), sig)
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// UpdateContextBlocks updates only the blocks field of a context
func (h *ContextHandler) UpdateContextBlocks(c *gin.Context) {
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

	var req struct {
		Blocks    json.RawMessage `json:"blocks" binding:"required"`
		WordCount *int32          `json:"word_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
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

	ctx, err := queries.UpdateContextBlocks(c.Request.Context(), sqlc.UpdateContextBlocksParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Blocks:    req.Blocks,
		WordCount: req.WordCount,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update blocks", nil)
		return
	}

	// Re-sync to the engine when block content changes.
	if h.engineSync != nil {
		h.engineSync.Enqueue(c.Request.Context(), signalFromContext(ctx))
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}

// DeleteContext deletes a context
func (h *ContextHandler) DeleteContext(c *gin.Context) {
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
	err = queries.DeleteContext(c.Request.Context(), sqlc.DeleteContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete context", nil)
		return
	}

	// Invalidate cache for this user's contexts list
	if h.queryCache != nil {
		go h.invalidateContextsCachePattern(c.Request.Context(), user.ID)
	}

	// Remove the corresponding markdown from the engine's nodes root + reindex.
	if h.engineSync != nil {
		h.engineSync.EnqueueDelete(c.Request.Context(), services.ModulePages, id.String())
	}

	c.JSON(http.StatusOK, gin.H{"message": "Context deleted"})
}
