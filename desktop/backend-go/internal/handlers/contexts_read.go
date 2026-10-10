package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ListContexts returns all contexts for the current user
// Results are cached in Redis for 5 minutes with user-specific keys
func (h *ContextHandler) ListContexts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse optional filters
	contextType := c.Query("type")
	isArchived := c.Query("is_archived") == "true"
	isTemplate := c.Query("is_template") == "true"
	search := c.Query("search")
	page := c.DefaultQuery("page", "1")

	// Try cache first if available
	if h.queryCache != nil {
		pageNum, _ := strconv.Atoi(page)
		cacheKey := fmt.Sprintf("contexts:user:%s:ws:%s:type:%s:archived:%t:template:%t:search:%s:page:%d",
			user.ID, c.GetHeader("X-Workspace-ID"), contextType, isArchived, isTemplate, search, pageNum)

		var cachedContexts []map[string]interface{}
		if err := h.queryCache.GetOrCompute(
			c.Request.Context(),
			cacheKey,
			5*time.Minute,
			&cachedContexts,
			func() (interface{}, error) {
				return h.fetchContextsFromDB(c, user.ID, contextType, isArchived, isTemplate, search)
			},
		); err == nil {
			c.JSON(http.StatusOK, cachedContexts)
			return
		}
		// If cache error, fall through to direct DB query
		slog.Debug("Cache error for ListContexts, falling back to direct DB query")
	}

	// Fallback: Direct database query without cache
	contexts, err := h.fetchContextsFromDB(c, user.ID, contextType, isArchived, isTemplate, search)
	if err != nil {
		slog.Error("[Contexts] Failed to list contexts", "user_id", user.ID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "list contexts", nil)
		return
	}

	c.JSON(http.StatusOK, contexts)
}

// fetchContextsFromDB queries the database for contexts
func (h *ContextHandler) fetchContextsFromDB(c *gin.Context, userID, contextType string, isArchived, isTemplate bool, search string) (interface{}, error) {
	queries := sqlc.New(h.pool)

	var ctxType sqlc.Contexttype
	if contextType != "" {
		ctxType = stringToContextType(contextType)
	}
	ctxTypeArg := sqlc.NullContexttype{Contexttype: ctxType, Valid: contextType != ""}

	// BusinessOSSync: when the request carries an authorized X-Workspace-ID,
	// include docs shared into that workspace so members see each other's
	// published docs (not just their own).
	if ws, ok := h.workspaceFromHeader(c, userID); ok {
		contexts, err := queries.ListContextsForWorkspace(c.Request.Context(), sqlc.ListContextsForWorkspaceParams{
			UserID:      userID,
			WorkspaceID: pgtype.UUID{Bytes: ws, Valid: true},
			IsArchived:  &isArchived,
			ContextType: ctxTypeArg,
			IsTemplate:  &isTemplate,
			Search:      utils.StringPtr(search),
		})
		if err != nil {
			return nil, err
		}
		return TransformContexts(contexts), nil
	}

	contexts, err := queries.ListContexts(c.Request.Context(), sqlc.ListContextsParams{
		UserID:      userID,
		IsArchived:  &isArchived,
		ContextType: ctxTypeArg,
		IsTemplate:  &isTemplate,
		Search:      utils.StringPtr(search),
	})
	if err != nil {
		return nil, err
	}

	return TransformContexts(contexts), nil
}

// GetContext returns a single context
func (h *ContextHandler) GetContext(c *gin.Context) {
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
	ctx, err := queries.GetContext(c.Request.Context(), sqlc.GetContextParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Context")
		return
	}

	// Check if children are requested
	if c.Query("include_children") == "true" {
		children, err := queries.GetContextChildren(c.Request.Context(), sqlc.GetContextChildrenParams{
			ParentID: pgtype.UUID{Bytes: id, Valid: true},
			UserID:   user.ID,
		})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"context":  TransformContext(ctx),
				"children": TransformContexts(children),
			})
			return
		}
	}

	c.JSON(http.StatusOK, TransformContext(ctx))
}
