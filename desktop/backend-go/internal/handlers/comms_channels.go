package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// COMMS CHANNELS — unified Slack + Teams + local WhatsApp listing
// ============================================================================
//
// Reads provider records and surfaces them as a single normalized shape so the
// channels tab can render them through the same components. WhatsApp remains
// read-only and local; workspace routing is a separate explicit operation.
//
//   GET /api/comms/channels?providers=slack,teams
//   GET /api/comms/messages?channel_id=<BO uuid>&provider=slack|teams
//
// Leon's spec at docs/design/comms-channels-spec.md §1.3 describes a richer
// contract (workspaces, threads, reactions, activity). Wave 2 ships only
// the shape Axis's brief asked for; the richer surface is flagged for a
// follow-up sprint and noted in the PR description.

// unifiedChannel is the merged Slack+Teams channel shape.
type unifiedChannel struct {
	ID                  string     `json:"id"`       // BO uuid or provider-safe local id
	Provider            string     `json:"provider"` // "slack" | "teams" | "whatsapp"
	WorkspaceID         string     `json:"workspace_id"`
	WorkspaceName       string     `json:"workspace_name"`
	ExternalID          string     `json:"external_id"` // Slack channel id or Graph channel id
	Name                string     `json:"name"`
	IsPrivate           bool       `json:"is_private"`
	IsDM                bool       `json:"is_dm"`
	MemberCount         int        `json:"member_count"`
	UnreadCount         int        `json:"unread_count"`
	MessageCount        int        `json:"message_count,omitempty"`
	IsArchived          bool       `json:"is_archived"`
	ReadOnly            bool       `json:"read_only"`
	Topic               string     `json:"topic,omitempty"`
	Description         string     `json:"description,omitempty"`
	LastActivity        *time.Time `json:"last_message_at,omitempty"`
	RoutedWorkspaceID   string     `json:"routed_workspace_id,omitempty"`
	RoutedWorkspaceName string     `json:"routed_workspace_name,omitempty"`
	RoutingScope        string     `json:"routing_scope,omitempty"`

	// Teams-specific (omitted from Slack rows)
	TeamID   string `json:"team_id,omitempty"`
	TeamName string `json:"team_name,omitempty"`
	WebURL   string `json:"web_url,omitempty"`
}

type unifiedChannelsResponse struct {
	Channels []unifiedChannel `json:"channels"`
	Total    int              `json:"total"`
}

// unifiedMessage is the merged Slack+Teams message shape.
type unifiedMessage struct {
	ID          string     `json:"id"`          // BO uuid
	Provider    string     `json:"provider"`    // "slack" | "teams"
	ChannelID   string     `json:"channel_id"`  // BO uuid of the parent channel
	ExternalID  string     `json:"external_id"` // slack_ts or Graph message id
	SenderID    string     `json:"sender_id,omitempty"`
	SenderName  string     `json:"sender_name,omitempty"`
	Content     string     `json:"content"`
	ContentHTML string     `json:"content_html,omitempty"`
	ReplyToID   string     `json:"reply_to_id,omitempty"` // Teams parent
	ThreadTS    string     `json:"thread_ts,omitempty"`   // Slack thread root
	ReplyCount  int        `json:"reply_count,omitempty"`
	IsEdited    bool       `json:"is_edited"`
	IsDeleted   bool       `json:"is_deleted"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
}

type unifiedMessagesResponse struct {
	Messages []unifiedMessage `json:"messages"`
	Total    int              `json:"total"`
	HasMore  bool             `json:"has_more"`
}

// registerChannelRoutes wires the unified channels + messages endpoints.
// Called from RegisterRoutes alongside the inbox routes.
//
// Path shape note: Axis's Wave 2 brief asked for flat
// `GET /comms/messages?channel_id=...`, but Leon's spec at
// docs/design/comms-channels-spec.md §1.3 specifies the nested
// `GET /comms/channels/:id/messages?limit=&before=` shape — and Fantem's
// frontend at frontend/src/lib/api/comms/channels.ts:304 already calls
// the nested form. Going with the nested shape so the frontend probe
// resolves; flagged for Axis in the PR description.
func (h *commsHandler) registerChannelRoutes(comms *gin.RouterGroup) {
	comms.GET("/workspaces", h.GetWorkspaces)
	comms.GET("/channels", h.GetChannels)
	comms.GET("/channels/:id/messages", h.GetChannelMessages)
}

// ----------------------------------------------------------------------------
// GET /api/comms/workspaces
// ----------------------------------------------------------------------------

// commsWorkspace mirrors frontend type CommsWorkspace
// (frontend/src/lib/api/comms/channels.ts:31).
type commsWorkspace struct {
	Provider            string     `json:"provider"` // "slack" | "teams"
	WorkspaceID         string     `json:"workspace_id"`
	WorkspaceName       string     `json:"workspace_name"`
	ConnectedAt         *time.Time `json:"connected_at,omitempty"`
	Status              string     `json:"status,omitempty"` // healthy | stale | reauth_required | error
	ConnectionStatus    string     `json:"connection_status,omitempty"`
	SyncStatus          string     `json:"sync_status,omitempty"`
	RoutingStatus       string     `json:"routing_status,omitempty"`
	ReadOnly            bool       `json:"read_only"`
	ChatCount           int        `json:"chat_count,omitempty"`
	MessageCount        int        `json:"message_count,omitempty"`
	LastMessageAt       *time.Time `json:"last_message_at,omitempty"`
	RoutedWorkspaceID   string     `json:"routed_workspace_id,omitempty"`
	RoutedWorkspaceName string     `json:"routed_workspace_name,omitempty"`
}

// GetWorkspaces returns the connected channel-style workspaces — one row
// per Slack workspace and one row per Microsoft tenant the user has
// connected. Drives the workspace switcher in the channels sidebar.
func (h *commsHandler) GetWorkspaces(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	ctx := c.Request.Context()
	out := make([]commsWorkspace, 0, 3)

	// Slack — one row per workspace token; today the schema is one workspace
	// per user, but we read the row through workspace_id/name to be ready
	// for multi-workspace later.
	rows, err := h.pool.Query(ctx, `
		SELECT workspace_id, workspace_name, created_at
		FROM slack_oauth_tokens
		WHERE user_id = $1
	`, userID)
	if err == nil {
		for rows.Next() {
			var (
				wid    string
				wname  *string
				connAt *time.Time
			)
			if err := rows.Scan(&wid, &wname, &connAt); err == nil && wid != "" {
				out = append(out, commsWorkspace{
					Provider:      "slack",
					WorkspaceID:   wid,
					WorkspaceName: deref(wname),
					ConnectedAt:   connAt,
					Status:        "healthy",
				})
			}
		}
		rows.Close()
	} else {
		slog.Info("comms workspaces: slack query failed", "error", err)
	}

	// Microsoft — one row per user token. workspace_id is the tenant-scoped
	// Microsoft account id; workspace_name is the email.
	var msID, msEmail *string
	var msCreated *time.Time
	if err := h.pool.QueryRow(ctx, `
		SELECT microsoft_id, microsoft_email, created_at
		FROM microsoft_oauth_tokens
		WHERE user_id = $1
	`, userID).Scan(&msID, &msEmail, &msCreated); err == nil {
		wid := deref(msID)
		if wid == "" {
			wid = deref(msEmail)
		}
		if wid != "" {
			out = append(out, commsWorkspace{
				Provider:      "teams",
				WorkspaceID:   wid,
				WorkspaceName: deref(msEmail),
				ConnectedAt:   msCreated,
				Status:        "healthy",
			})
		}
	}

	if h.whatsapp != nil {
		if status, err := h.whatsapp.status(ctx); err == nil {
			out = append(out, commsWorkspace{
				Provider:         whatsappProvider,
				WorkspaceID:      "whatsapp-local",
				WorkspaceName:    "WhatsApp on this Mac",
				Status:           "healthy",
				ConnectionStatus: "connected",
				SyncStatus:       "history_synced",
				RoutingStatus:    "unassigned",
				ReadOnly:         true,
				ChatCount:        status.ChatCount,
				MessageCount:     status.MessageCount,
				LastMessageAt:    status.LastMessage,
			})
		}
	}
	routeIndex := loadCommunicationRouteIndex(ctx, h.pool, userID)
	for i := range out {
		if route, ok := routeIndex.resolve(out[i].Provider, ""); ok {
			out[i].RoutingStatus = "configured"
			out[i].RoutedWorkspaceID = route.WorkspaceID
			out[i].RoutedWorkspaceName = route.WorkspaceName
		} else if out[i].RoutingStatus == "" {
			out[i].RoutingStatus = "unassigned"
		}
	}

	c.JSON(http.StatusOK, gin.H{"workspaces": out})
}

// ----------------------------------------------------------------------------
// GET /api/comms/channels
// ----------------------------------------------------------------------------

// GetChannels returns merged Slack + Teams channels for the user, ordered
// by recent activity. Returns 200 with empty list when no providers are
// connected (frontend probe-and-cache pattern relies on 200).
func (h *commsHandler) GetChannels(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	providers := parseChannelProviders(c.Query("providers"))
	ctx := c.Request.Context()

	var rows []unifiedChannel
	if providers["slack"] {
		s, err := h.fetchSlackChannels(ctx, userID)
		if err != nil {
			slog.Info("comms channels: slack fetch failed", "error", err)
		} else {
			rows = append(rows, s...)
		}
	}
	if providers["teams"] {
		t, err := h.fetchTeamsChannels(ctx, userID)
		if err != nil {
			slog.Info("comms channels: teams fetch failed", "error", err)
		} else {
			rows = append(rows, t...)
		}
	}
	if providers[whatsappProvider] && h.whatsapp != nil {
		w, err := h.whatsapp.channels(ctx)
		if err != nil {
			slog.Info("comms channels: whatsapp fetch unavailable", "error", err)
		} else {
			rows = append(rows, w...)
		}
	}
	routeIndex := loadCommunicationRouteIndex(ctx, h.pool, userID)
	for i := range rows {
		if route, ok := routeIndex.resolve(rows[i].Provider, rows[i].ID); ok {
			rows[i].RoutedWorkspaceID = route.WorkspaceID
			rows[i].RoutedWorkspaceName = route.WorkspaceName
			rows[i].RoutingScope = route.Scope
		}
	}

	// Sort by last_activity desc, NULLs last; tie-break on name.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].LastActivity == nil && rows[j].LastActivity == nil {
			return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
		}
		if rows[i].LastActivity == nil {
			return false
		}
		if rows[j].LastActivity == nil {
			return true
		}
		return rows[i].LastActivity.After(*rows[j].LastActivity)
	})

	c.JSON(http.StatusOK, unifiedChannelsResponse{
		Channels: rows,
		Total:    len(rows),
	})
}

func (h *commsHandler) fetchSlackChannels(ctx context.Context, userID string) ([]unifiedChannel, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, slack_id, name, is_private, is_dm, member_count, unread_count,
			topic, last_activity
		FROM slack_channels
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []unifiedChannel
	for rows.Next() {
		var (
			ch           unifiedChannel
			topic        *string
			lastActivity *time.Time
		)
		if err := rows.Scan(
			&ch.ID, &ch.ExternalID, &ch.Name, &ch.IsPrivate, &ch.IsDM,
			&ch.MemberCount, &ch.UnreadCount, &topic, &lastActivity,
		); err != nil {
			return nil, err
		}
		ch.Provider = "slack"
		if topic != nil {
			ch.Topic = *topic
		}
		ch.LastActivity = lastActivity
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (h *commsHandler) fetchTeamsChannels(ctx context.Context, userID string) ([]unifiedChannel, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, channel_id, team_id, team_name, display_name, description,
			email, web_url, membership_type, member_count, last_message_at
		FROM microsoft_teams_channels
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []unifiedChannel
	for rows.Next() {
		var (
			ch             unifiedChannel
			teamName       *string
			displayName    *string
			description    *string
			email          *string
			webURL         *string
			membershipType *string
			lastActivity   *time.Time
		)
		if err := rows.Scan(
			&ch.ID, &ch.ExternalID, &ch.TeamID, &teamName, &displayName, &description,
			&email, &webURL, &membershipType, &ch.MemberCount, &lastActivity,
		); err != nil {
			return nil, err
		}
		ch.Provider = "teams"
		ch.TeamName = deref(teamName)
		ch.Name = deref(displayName)
		ch.Description = deref(description)
		ch.WebURL = deref(webURL)
		// Map Teams membership_type to the unified is_private flag.
		// "private" and "shared" are non-default visibility.
		if membershipType != nil && *membershipType != "" && *membershipType != "standard" {
			ch.IsPrivate = true
		}
		// Optionally surface the channel email to the unified Topic slot
		// so the sidebar has something to show. Cheap; doesn't conflict
		// with Slack's actual topic semantic.
		if email != nil && *email != "" {
			ch.Topic = *email
		}
		ch.LastActivity = lastActivity
		out = append(out, ch)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// GET /api/comms/messages
// ----------------------------------------------------------------------------

// GetChannelMessages returns the most recent messages for a channel by
// BO uuid. URL: `GET /api/comms/channels/:id/messages`. The `:id` path
// param is the BO uuid; we resolve it to either slack_messages or
// microsoft_teams_messages.
//
// Query params:
//
//	provider (optional) — "slack" | "teams"; auto-detected when absent
//	limit    (optional) — default 50, max 200
//	before   (optional) — ISO timestamp; return messages strictly older
func (h *commsHandler) GetChannelMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	channelID := c.Param("id")
	if channelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel id is required"})
		return
	}

	limit := parseIntDefault(c.Query("limit"), 50, 1)
	if limit > 200 {
		limit = 200
	}
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	beforeStr := c.Query("before")
	var before *time.Time
	if beforeStr != "" {
		if t, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			before = &t
		}
	}

	ctx := c.Request.Context()

	// Resolve provider when not given.
	if provider == "" {
		resolved, err := h.resolveChannelProvider(ctx, userID, channelID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		provider = resolved
	}

	switch provider {
	case "slack":
		msgs, more, err := h.fetchSlackMessages(ctx, userID, channelID, limit, before)
		if err != nil {
			slog.Info("comms messages: slack fetch failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
			return
		}
		c.JSON(http.StatusOK, unifiedMessagesResponse{Messages: msgs, Total: len(msgs), HasMore: more})
	case "teams":
		msgs, more, err := h.fetchTeamsMessages(ctx, userID, channelID, limit, before)
		if err != nil {
			slog.Info("comms messages: teams fetch failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
			return
		}
		c.JSON(http.StatusOK, unifiedMessagesResponse{Messages: msgs, Total: len(msgs), HasMore: more})
	case whatsappProvider:
		if h.whatsapp == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WhatsApp history is not available"})
			return
		}
		msgs, more, err := h.whatsapp.messages(ctx, channelID, limit, before)
		if err != nil {
			slog.Info("comms messages: whatsapp fetch failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load WhatsApp messages"})
			return
		}
		c.JSON(http.StatusOK, unifiedMessagesResponse{Messages: msgs, Total: len(msgs), HasMore: more})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider must be 'slack', 'teams', or 'whatsapp'"})
	}
}

// resolveChannelProvider looks the BO uuid up in both channel tables and
// returns the matching provider. Errors when neither has it.
func (h *commsHandler) resolveChannelProvider(ctx context.Context, userID, channelID string) (string, error) {
	if strings.HasPrefix(channelID, "wa_") {
		if _, err := decodeWhatsAppChannelID(channelID); err == nil {
			return whatsappProvider, nil
		}
	}
	var exists bool
	if err := h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM slack_channels WHERE id = $1 AND user_id = $2)
	`, channelID, userID).Scan(&exists); err == nil && exists {
		return "slack", nil
	}
	if err := h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM microsoft_teams_channels WHERE id = $1 AND user_id = $2)
	`, channelID, userID).Scan(&exists); err == nil && exists {
		return "teams", nil
	}
	return "", errChannelNotFound
}

var errChannelNotFound = errChannelNotFoundType{}

type errChannelNotFoundType struct{}

func (errChannelNotFoundType) Error() string { return "channel not found" }

func (h *commsHandler) fetchSlackMessages(ctx context.Context, userID, channelID string, limit int, before *time.Time) ([]unifiedMessage, bool, error) {
	args := []interface{}{userID, channelID}
	beforeClause := ""
	if before != nil {
		args = append(args, *before)
		beforeClause = " AND sent_at < $3"
	}
	args = append(args, limit+1)
	limitPlaceholder := "$" + itoa(len(args))

	rows, err := h.pool.Query(ctx, `
		SELECT id, channel_id, slack_ts, sender_id, sender_name, content,
			thread_ts, reply_count, is_edited, sent_at
		FROM slack_messages
		WHERE user_id = $1 AND channel_id = $2`+beforeClause+`
		ORDER BY sent_at DESC NULLS LAST
		LIMIT `+limitPlaceholder, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var out []unifiedMessage
	for rows.Next() {
		var (
			m          unifiedMessage
			senderID   *string
			senderName *string
			content    *string
			threadTS   *string
			sentAt     *time.Time
			channelDB  string
		)
		if err := rows.Scan(
			&m.ID, &channelDB, &m.ExternalID, &senderID, &senderName, &content,
			&threadTS, &m.ReplyCount, &m.IsEdited, &sentAt,
		); err != nil {
			return nil, false, err
		}
		m.Provider = "slack"
		m.ChannelID = channelDB
		m.SenderID = deref(senderID)
		m.SenderName = deref(senderName)
		m.Content = deref(content)
		m.ThreadTS = deref(threadTS)
		m.SentAt = sentAt
		out = append(out, m)
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, rows.Err()
}

func (h *commsHandler) fetchTeamsMessages(ctx context.Context, userID, channelID string, limit int, before *time.Time) ([]unifiedMessage, bool, error) {
	// Resolve BO channel uuid → (team_id, channel_id) Graph identifiers.
	var teamID, graphChannelID string
	if err := h.pool.QueryRow(ctx, `
		SELECT team_id, channel_id
		FROM microsoft_teams_channels
		WHERE id = $1 AND user_id = $2
	`, channelID, userID).Scan(&teamID, &graphChannelID); err != nil {
		return nil, false, err
	}

	args := []interface{}{userID, teamID, graphChannelID}
	beforeClause := ""
	if before != nil {
		args = append(args, *before)
		beforeClause = " AND created_datetime < $4"
	}
	args = append(args, limit+1)
	limitPlaceholder := "$" + itoa(len(args))

	rows, err := h.pool.Query(ctx, `
		SELECT id, message_id, reply_to_id, sender_id, sender_name,
			body_content, body_content_type, is_edited, is_deleted, created_datetime
		FROM microsoft_teams_messages
		WHERE user_id = $1 AND team_id = $2 AND channel_id = $3`+beforeClause+`
		ORDER BY created_datetime DESC NULLS LAST
		LIMIT `+limitPlaceholder, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var out []unifiedMessage
	for rows.Next() {
		var (
			m               unifiedMessage
			replyToID       *string
			senderID        *string
			senderName      *string
			bodyContent     *string
			bodyContentType *string
			createdAt       *time.Time
		)
		if err := rows.Scan(
			&m.ID, &m.ExternalID, &replyToID, &senderID, &senderName,
			&bodyContent, &bodyContentType, &m.IsEdited, &m.IsDeleted, &createdAt,
		); err != nil {
			return nil, false, err
		}
		m.Provider = "teams"
		m.ChannelID = channelID
		m.ReplyToID = deref(replyToID)
		m.SenderID = deref(senderID)
		m.SenderName = deref(senderName)
		// body_content carries either text or HTML based on content_type.
		// Surface text in Content; HTML in ContentHTML so the frontend can
		// pick its rendering path without re-inspecting the type.
		ct := strings.ToLower(deref(bodyContentType))
		body := deref(bodyContent)
		if ct == "html" {
			m.ContentHTML = body
		} else {
			m.Content = body
		}
		m.SentAt = createdAt
		out = append(out, m)
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, rows.Err()
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func parseChannelProviders(raw string) map[string]bool {
	out := map[string]bool{}
	if raw == "" {
		out["slack"] = true
		out["teams"] = true
		out[whatsappProvider] = true
		return out
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "slack" || p == "teams" || p == whatsappProvider {
			out[p] = true
		}
	}
	if len(out) == 0 {
		out["slack"] = true
		out["teams"] = true
		out[whatsappProvider] = true
	}
	return out
}

// deref is defined in comms_contacts.go.

// itoa is strconv.Itoa wrapped to keep call sites readable in SQL string
// concatenation. Avoids importing strconv twice in this file (other helpers
// in this package already use it via comms_inbox.go).
func itoa(n int) string {
	// Local copy of strconv.Itoa to avoid an extra import for one call.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
