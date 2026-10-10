package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/cache"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// ContextHandler handles context management operations
type ContextHandler struct {
	pool       *pgxpool.Pool
	queryCache *cache.QueryCache
	// engineSync mirrors every Page save/update/delete into OptimalEngine.
	// HTTP memory writes are primary when OPTIMAL_ENGINE_URL is configured;
	// local markdown mirroring is optional and nil-safe.
	engineSync *services.EngineSync
}

// NewContextHandler creates a new ContextHandler
func NewContextHandler(pool *pgxpool.Pool, queryCache *cache.QueryCache, engineSync *services.EngineSync) *ContextHandler {
	return &ContextHandler{
		pool:       pool,
		queryCache: queryCache,
		engineSync: engineSync,
	}
}

// RegisterContextRoutes registers all context routes on the given router group
func RegisterContextRoutes(api *gin.RouterGroup, h *ContextHandler, auth gin.HandlerFunc) {
	contexts := api.Group("/contexts")
	{
		// Public route (no auth)
		contexts.GET("/public/:shareId", h.GetPublicContext)

		// Protected routes
		protected := contexts.Group("")
		protected.Use(auth, middleware.RequireAuth())
		{
			protected.GET("", h.ListContexts)
			protected.POST("", h.CreateContext)
			protected.GET("/:id", h.GetContext)
			protected.PUT("/:id", h.UpdateContext)
			protected.PATCH("/:id/blocks", h.UpdateContextBlocks)
			protected.POST("/:id/share", h.ShareContext)
			protected.DELETE("/:id/share", h.UnshareContext)
			protected.POST("/:id/duplicate", h.DuplicateContext)
			protected.PATCH("/:id/archive", h.ArchiveContext)
			protected.PATCH("/:id/unarchive", h.UnarchiveContext)
			protected.DELETE("/:id", h.DeleteContext)
			protected.POST("/aggregate", h.AggregateContext)
		}
	}
}

// invalidateContextsCachePattern invalidates all cache entries for a user's contexts
func (h *ContextHandler) invalidateContextsCachePattern(ctx context.Context, userID string) {
	if h.queryCache == nil {
		return
	}

	pattern := fmt.Sprintf("contexts:user:%s:*", userID)
	if _, err := h.queryCache.DeleteByPattern(ctx, pattern); err != nil {
		slog.Warn("Failed to invalidate contexts cache",
			"user_id", userID,
			"pattern", pattern,
			"error", err)
	}
}

// stringToContextType converts a string to sqlc.Contexttype
func stringToContextType(t string) sqlc.Contexttype {
	typeMap := map[string]sqlc.Contexttype{
		"person":   sqlc.ContexttypePERSON,
		"business": sqlc.ContexttypeBUSINESS,
		"project":  sqlc.ContexttypePROJECT,
		"document": sqlc.ContexttypeDocument, // Use lowercase version for DB compatibility
		"custom":   sqlc.ContexttypeCUSTOM,
	}
	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.ContexttypeCUSTOM
}

// generateShareID generates a random share ID
func generateShareID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// signalFromContext maps a sqlc Context row onto the universal Signal
// struct EngineSync ingests. We keep the conversion here (where both the
// handler and the sqlc generated types are in scope) so engine_sync.go
// stays free of sqlc dependencies.
func signalFromContext(ctx sqlc.Context) services.Signal {
	id := ""
	if ctx.ID.Valid {
		id = uuidString(ctx.ID.Bytes)
	}
	body := ""
	if ctx.Content != nil {
		body = *ctx.Content
	}
	genre := "page"
	if ctx.Type.Valid {
		genre = strings.ToLower(string(ctx.Type.Contexttype))
	}
	updated := time.Time{}
	if ctx.UpdatedAt.Valid {
		updated = ctx.UpdatedAt.Time
	}
	wsID := ""
	if ctx.WorkspaceID.Valid {
		wsID = uuidString(ctx.WorkspaceID.Bytes)
	}
	return services.Signal{
		Module:      services.ModulePages,
		ID:          id,
		AuthorID:    ctx.UserID,
		Title:       ctx.Name,
		Body:        body,
		Genre:       genre,
		ModifiedAt:  updated,
		WorkspaceID: wsID,
	}
}

// uuidString formats a UUID byte array as the canonical hex form. Avoids
// pulling the uuid package at this layer just for a tiny formatter.
func uuidString(b [16]byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 36)
	idx := 0
	for i, x := range b {
		switch i {
		case 4, 6, 8, 10:
			out[idx] = '-'
			idx++
		}
		out[idx] = hexdigits[x>>4]
		out[idx+1] = hexdigits[x&0x0f]
		idx += 2
	}
	return string(out)
}
