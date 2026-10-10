package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

type communicationRoute struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"`
	Scope         string    `json:"scope"`
	ExternalID    string    `json:"external_id"`
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	Enabled       bool      `json:"enabled"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type communicationRouteIndex struct {
	accounts      map[string]communicationRoute
	conversations map[string]communicationRoute
}

func loadCommunicationRouteIndex(ctx context.Context, pool *pgxpool.Pool, userID string) communicationRouteIndex {
	index := communicationRouteIndex{accounts: map[string]communicationRoute{}, conversations: map[string]communicationRoute{}}
	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.provider, r.scope, r.external_id,
		       r.workspace_id::text, w.name, r.enabled, r.updated_at
		FROM communication_routes r
		JOIN workspaces w ON w.id=r.workspace_id
		WHERE r.user_id=$1 AND r.enabled
		  AND EXISTS (SELECT 1 FROM workspace_members wm
		      WHERE wm.workspace_id=r.workspace_id AND wm.user_id=$1 AND wm.status='active')
	`, userID)
	if err != nil {
		return index
	}
	defer rows.Close()
	for rows.Next() {
		var route communicationRoute
		if rows.Scan(&route.ID, &route.Provider, &route.Scope, &route.ExternalID,
			&route.WorkspaceID, &route.WorkspaceName, &route.Enabled, &route.UpdatedAt) != nil {
			continue
		}
		if route.Scope == "account" {
			index.accounts[route.Provider] = route
		} else {
			index.conversations[route.Provider+"\x00"+route.ExternalID] = route
		}
	}
	return index
}

func (i communicationRouteIndex) resolve(provider, externalID string) (communicationRoute, bool) {
	if route, ok := i.conversations[provider+"\x00"+externalID]; ok {
		return route, true
	}
	route, ok := i.accounts[provider]
	return route, ok
}

type upsertCommunicationRouteRequest struct {
	Provider      string `json:"provider" binding:"required"`
	Scope         string `json:"scope" binding:"required"`
	ExternalID    string `json:"external_id"`
	WorkspaceID   string `json:"workspace_id" binding:"required"`
	BackfillLimit int    `json:"backfill_limit"`
}

type syncCommunicationRoutesRequest struct {
	Provider string `json:"provider" binding:"required"`
}

func validCommunicationProvider(provider string) bool {
	switch provider {
	case "gmail", "outlook", "slack", "teams", "whatsapp":
		return true
	default:
		return false
	}
}

func normalizeRouteKey(scope, externalID string) (string, bool) {
	if scope == "account" {
		return "*", true
	}
	if scope == "conversation" && strings.TrimSpace(externalID) != "" {
		return strings.TrimSpace(externalID), true
	}
	return "", false
}

func resolveCommunicationRoute(ctx context.Context, pool *pgxpool.Pool, userID, provider, conversationID string) (*communicationRoute, error) {
	if pool == nil || userID == "" || !validCommunicationProvider(provider) {
		return nil, nil
	}
	var route communicationRoute
	err := pool.QueryRow(ctx, `
		SELECT r.id::text, r.provider, r.scope, r.external_id,
		       r.workspace_id::text, w.name, r.enabled, r.updated_at
		FROM communication_routes r
		JOIN workspaces w ON w.id = r.workspace_id
		WHERE r.user_id = $1 AND r.provider = $2 AND r.enabled
		  AND ((r.scope = 'conversation' AND r.external_id = $3)
		       OR (r.scope = 'account' AND r.external_id = '*'))
		  AND EXISTS (
		      SELECT 1 FROM workspace_members wm
		      WHERE wm.workspace_id = r.workspace_id
		        AND wm.user_id = $1 AND wm.status = 'active'
		  )
		ORDER BY CASE WHEN r.scope = 'conversation' THEN 0 ELSE 1 END
		LIMIT 1
	`, userID, provider, conversationID).Scan(
		&route.ID, &route.Provider, &route.Scope, &route.ExternalID,
		&route.WorkspaceID, &route.WorkspaceName, &route.Enabled, &route.UpdatedAt,
	)
	if err != nil {
		return nil, nil
	}
	return &route, nil
}

func (h *commsHandler) ListRoutes(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT r.id::text, r.provider, r.scope, r.external_id,
		       r.workspace_id::text, w.name, r.enabled, r.updated_at
		FROM communication_routes r
		JOIN workspaces w ON w.id = r.workspace_id
		WHERE r.user_id = $1
		ORDER BY r.provider, r.scope, r.external_id
	`, userID)
	if err != nil {
		RespondInternalErr(c, "list communication routes", err)
		return
	}
	defer rows.Close()
	routes := make([]communicationRoute, 0)
	for rows.Next() {
		var route communicationRoute
		if err := rows.Scan(&route.ID, &route.Provider, &route.Scope, &route.ExternalID,
			&route.WorkspaceID, &route.WorkspaceName, &route.Enabled, &route.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan communication route", err)
			return
		}
		routes = append(routes, route)
	}
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func (h *commsHandler) UpsertRoute(c *gin.Context) {
	userID := c.GetString("user_id")
	var req upsertCommunicationRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider, scope, and workspace_id are required"})
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Scope = strings.ToLower(strings.TrimSpace(req.Scope))
	externalID, ok := normalizeRouteKey(req.Scope, req.ExternalID)
	if !validCommunicationProvider(req.Provider) || !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid communication route"})
		return
	}
	var member bool
	if err := h.pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(SELECT 1 FROM workspace_members
		WHERE workspace_id=$1::uuid AND user_id=$2 AND status='active')
	`, req.WorkspaceID, userID).Scan(&member); err != nil || !member {
		c.JSON(http.StatusForbidden, gin.H{"error": "active workspace membership required"})
		return
	}
	var route communicationRoute
	err := h.pool.QueryRow(c.Request.Context(), `
		WITH saved AS (
			INSERT INTO communication_routes (user_id, provider, scope, external_id, workspace_id)
			VALUES ($1, $2, $3, $4, $5::uuid)
			ON CONFLICT (user_id, provider, scope, external_id) DO UPDATE SET
				workspace_id=EXCLUDED.workspace_id, enabled=TRUE, updated_at=NOW()
			RETURNING id, provider, scope, external_id, workspace_id, enabled, updated_at
		)
		SELECT saved.id::text, saved.provider, saved.scope, saved.external_id,
		       saved.workspace_id::text, w.name, saved.enabled, saved.updated_at
		FROM saved JOIN workspaces w ON w.id=saved.workspace_id
	`, userID, req.Provider, req.Scope, externalID, req.WorkspaceID).Scan(
		&route.ID, &route.Provider, &route.Scope, &route.ExternalID,
		&route.WorkspaceID, &route.WorkspaceName, &route.Enabled, &route.UpdatedAt,
	)
	if err != nil {
		RespondInternalErr(c, "save communication route", err)
		return
	}
	indexed := 0
	if req.Provider == whatsappProvider && req.Scope == "conversation" && req.BackfillLimit > 0 {
		limit := req.BackfillLimit
		if limit > 500 {
			limit = 500
		}
		indexed = h.backfillWhatsApp(c.Request.Context(), userID, externalID, route, limit)
	}
	c.JSON(http.StatusOK, gin.H{"route": route, "indexed": indexed})
}

func (h *commsHandler) DeleteRoute(c *gin.Context) {
	userID := c.GetString("user_id")
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	externalID, ok := normalizeRouteKey(scope, c.Query("external_id"))
	if !validCommunicationProvider(provider) || !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid communication route"})
		return
	}
	_, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM communication_routes
		WHERE user_id=$1 AND provider=$2 AND scope=$3 AND external_id=$4
	`, userID, provider, scope, externalID)
	if err != nil {
		RespondInternalErr(c, "delete communication route", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// SyncRoutes refreshes routed local communication sources without changing
// the source database. Provider APIs with native sync paths continue to use
// those paths; this endpoint currently handles routed WhatsApp conversations.
func (h *commsHandler) SyncRoutes(c *gin.Context) {
	userID := c.GetString("user_id")
	var req syncCommunicationRoutesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider != whatsappProvider {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not use routed sync"})
		return
	}

	routes, err := h.listConversationRoutes(c.Request.Context(), userID, req.Provider)
	if err != nil {
		RespondInternalErr(c, "list routed communication sources", err)
		return
	}
	indexed := 0
	for _, route := range routes {
		indexed += h.backfillWhatsApp(c.Request.Context(), userID, route.ExternalID, route, 200)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "indexed": indexed})
}

func (h *commsHandler) listConversationRoutes(ctx context.Context, userID, provider string) ([]communicationRoute, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT r.id::text, r.provider, r.scope, r.external_id,
		       r.workspace_id::text, w.name, r.enabled, r.updated_at
		FROM communication_routes r
		JOIN workspaces w ON w.id=r.workspace_id
		JOIN workspace_members wm ON wm.workspace_id=r.workspace_id
		WHERE r.user_id=$1 AND r.provider=$2 AND r.scope='conversation'
		  AND r.enabled AND wm.user_id=$1 AND wm.status='active'
		ORDER BY r.updated_at DESC
	`, userID, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	routes := make([]communicationRoute, 0)
	for rows.Next() {
		var route communicationRoute
		if err := rows.Scan(&route.ID, &route.Provider, &route.Scope, &route.ExternalID,
			&route.WorkspaceID, &route.WorkspaceName, &route.Enabled, &route.UpdatedAt); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (h *commsHandler) backfillWhatsApp(ctx context.Context, userID, channelID string, route communicationRoute, limit int) int {
	if h.whatsapp == nil || h.engineSync == nil {
		return 0
	}
	messages, _, err := h.whatsapp.messages(ctx, channelID, limit, nil)
	if err != nil {
		return 0
	}
	for _, msg := range messages {
		modified := time.Now()
		if msg.SentAt != nil {
			modified = *msg.SentAt
		}
		h.engineSync.Enqueue(ctx, services.Signal{
			Module: services.ModuleMessage, ID: msg.ID, AuthorID: userID,
			WorkspaceID: route.WorkspaceID, Title: truncateForTitle(msg.Content, 80),
			Body: msg.Content, Genre: "message", ModifiedAt: modified,
			Metadata: map[string]string{
				"provider": "whatsapp", "channel_id": channelID,
				"external_id": msg.ExternalID, "sender_id": msg.SenderID,
				"sender_name": msg.SenderName, "routing_scope": route.Scope,
			},
		})
	}
	return len(messages)
}
