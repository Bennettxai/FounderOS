package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/slack"
	"github.com/rhl/businessos-backend/internal/services"
)

// ============================================================================
// SLACK EVENTS API WEBHOOK
// ============================================================================
//
// POST /api/integrations/slack/events
//
// Slack pushes one of three payload kinds to this endpoint:
//
//   1. url_verification   — first-setup handshake. Slack sends {type, challenge};
//                           we echo back {challenge}. No signature check needed
//                           because Slack does it before the secret is even
//                           configurable, but we verify anyway for consistency.
//   2. event_callback     — actual event. {type:"event_callback", event:{type:"message", ...}}
//                           Routed by event.type below.
//   3. app_rate_limited   — 429-style notice; no-op (we just 200 it).
//
// Signature verification — Slack signs every request with HMAC-SHA256 over
// the string "v0:<timestamp>:<raw-body>" using SLACK_SIGNING_SECRET. The
// signature lives in X-Slack-Signature; the timestamp in
// X-Slack-Request-Timestamp. Reject requests older than 5 minutes for
// replay protection. Reference:
// https://api.slack.com/authentication/verifying-requests-from-slack
//
// Per-user routing — payloads carry team_id (the Slack workspace). We
// look it up against slack_oauth_tokens.workspace_id to find the BO
// user_id whose token authorized this workspace. Today the schema is
// one user per workspace; if a future change allows multi-user-per-
// workspace, this lookup needs to fan out and fire SSE to all matching
// users.

// slackWebhookHandler owns POST /api/integrations/slack/events.
// Constructed once in registerIntegrationRoutes; the SaveMessage path
// reuses the existing MessageService singleton so OnMessageSaved (engine
// sync) fires identically to manual sync.
type slackWebhookHandler struct {
	pool          *pgxpool.Pool
	signingSecret string
	messages      *slack.MessageService
	broadcaster   *services.SSEBroadcaster
}

func newSlackWebhookHandler(pool *pgxpool.Pool, signingSecret string, messages *slack.MessageService, broadcaster *services.SSEBroadcaster) *slackWebhookHandler {
	return &slackWebhookHandler{
		pool:          pool,
		signingSecret: signingSecret,
		messages:      messages,
		broadcaster:   broadcaster,
	}
}

// HandleEvents is the single entry point for Slack's Events API.
// Mounted UNAUTHENTICATED — Slack identifies itself via the signing
// secret instead of a logged-in session. Always returns 200 quickly
// (Slack retries up to 3x on non-2xx within 60s, so failures generate
// duplicate inserts thanks to the ON CONFLICT in SaveMessage).
func (h *slackWebhookHandler) HandleEvents(c *gin.Context) {
	// Read body once so we can both verify the signature and re-bind it
	// for JSON parsing. Gin's c.ShouldBindJSON consumes the reader, so
	// signature verification needs the raw bytes captured first.
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	// Signature verification (skipped only when no secret configured —
	// dev environments without SLACK_SIGNING_SECRET should still receive
	// payloads for testing, but log loudly).
	if h.signingSecret == "" {
		slog.Warn("slack webhook: SLACK_SIGNING_SECRET not configured — accepting unsigned payload")
	} else if err := h.verifySignature(c, rawBody); err != nil {
		slog.Warn("slack webhook: signature verification failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// Discriminator-only parse to dispatch by type.
	var envelope struct {
		Type      string          `json:"type"`
		Challenge string          `json:"challenge"`
		TeamID    string          `json:"team_id"`
		Event     json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	switch envelope.Type {
	case "url_verification":
		// First-time setup handshake. Slack expects either {challenge}
		// or just the challenge string in the response body.
		c.JSON(http.StatusOK, gin.H{"challenge": envelope.Challenge})
	case "event_callback":
		h.handleEventCallback(c, envelope.TeamID, envelope.Event)
	case "app_rate_limited":
		// Nothing to do — Slack tells us we're being throttled. 200 ack.
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		// Unknown envelope type — 200 anyway so Slack doesn't retry.
		// Logged so we notice if Slack adds new envelope kinds.
		slog.Info("slack webhook: unknown envelope type", "type", envelope.Type)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ----------------------------------------------------------------------------
// Signature verification
// ----------------------------------------------------------------------------

// verifySignature implements the standard Slack request verification:
// HMAC-SHA256 over "v0:<timestamp>:<raw-body>" using the signing secret,
// hex-encoded with a "v0=" prefix, compared against X-Slack-Signature.
// Timestamp must be within 5 minutes (replay protection per Slack docs).
func (h *slackWebhookHandler) verifySignature(c *gin.Context, body []byte) error {
	timestampStr := c.GetHeader("X-Slack-Request-Timestamp")
	if timestampStr == "" {
		return fmt.Errorf("missing X-Slack-Request-Timestamp")
	}
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	// 5-minute replay window per Slack's recommendation.
	if abs64(time.Now().Unix()-timestamp) > 300 {
		return fmt.Errorf("timestamp too old or future")
	}

	provided := c.GetHeader("X-Slack-Signature")
	if provided == "" {
		return fmt.Errorf("missing X-Slack-Signature")
	}

	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	fmt.Fprintf(mac, "v0:%d:%s", timestamp, body)
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// constant-time comparison
	if !hmac.Equal([]byte(expected), []byte(provided)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ----------------------------------------------------------------------------
// event_callback dispatch
// ----------------------------------------------------------------------------

// handleEventCallback routes the inner event by type. We focus on the
// minimum set needed to drive the realtime UI:
//
//   message            — new channel/dm message → SaveMessage + SSE message.received
//   message.channels   — Slack alias for message in a channel
//   message.im         — Slack alias for message in a DM
//   channel_created    — new channel → SSE channel.updated{change:"created"}
//   channel_joined     — user joined a channel → SSE channel.updated{change:"member_added"}
//
// Unknown event types are 200-acked so Slack stops retrying. Adding
// support for `reaction_added`, `member_joined_channel`, etc. is a Wave
// 4 follow-up.
func (h *slackWebhookHandler) handleEventCallback(c *gin.Context, teamID string, eventRaw json.RawMessage) {
	if teamID == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "missing team_id"})
		return
	}
	userID, err := h.lookupUserByTeam(c.Request.Context(), teamID)
	if err != nil || userID == "" {
		// We have no user mapping for this workspace yet — happens if the
		// webhook fires before OAuth completes, or after disconnect.
		// 200 ack; nothing else to do.
		slog.Info("slack webhook: no user for team", "team_id", teamID, "error", err)
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	var inner struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	_ = json.Unmarshal(eventRaw, &inner)

	switch inner.Type {
	case "message", "message.channels", "message.im":
		// Slack collapses sub-flavors here; subtype="message_deleted" or
		// "message_changed" arrive as type="message" with a subtype. We
		// only persist new messages in Wave 3 — edits/deletes are Wave 4.
		if inner.Subtype != "" && inner.Subtype != "bot_message" && inner.Subtype != "thread_broadcast" {
			c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "subtype " + inner.Subtype})
			return
		}
		h.handleMessageEvent(c, userID, eventRaw)
	case "channel_created":
		h.handleChannelEvent(c, userID, eventRaw, "created")
	case "channel_joined":
		h.handleChannelEvent(c, userID, eventRaw, "member_added")
	default:
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// handleMessageEvent persists the message and broadcasts SSE.
// Resolves the Slack channel id → BO uuid before calling SaveMessage,
// since slack_messages.channel_id is FK to slack_channels.id (uuid).
func (h *slackWebhookHandler) handleMessageEvent(c *gin.Context, userID string, eventRaw json.RawMessage) {
	var ev struct {
		Channel    string `json:"channel"` // Slack channel id, NOT BO uuid
		User       string `json:"user"`
		Text       string `json:"text"`
		TS         string `json:"ts"`
		ThreadTS   string `json:"thread_ts"`
		ReplyCount int    `json:"reply_count"`
	}
	if err := json.Unmarshal(eventRaw, &ev); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "bad message event"})
		return
	}
	if ev.Channel == "" || ev.TS == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "missing channel or ts"})
		return
	}

	// Resolve Slack channel id → BO uuid.
	var channelUUID string
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT id FROM slack_channels WHERE user_id = $1 AND slack_id = $2
	`, userID, ev.Channel).Scan(&channelUUID)
	if err != nil {
		// Channel hasn't been synced yet. Skip the message; the next
		// manual sync will pick it up via conversations.history. Wave 4
		// can pre-seed the channel row from the event.
		slog.Info("slack webhook: channel not yet synced", "slack_channel", ev.Channel)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "channel not synced"})
		return
	}

	// Reuse the same upsert path the manual sync uses so OnMessageSaved
	// (engine sync) fires identically.
	msg := slack.SlackMessage{
		TS:         ev.TS,
		User:       ev.User,
		Text:       ev.Text,
		ThreadTS:   ev.ThreadTS,
		ReplyCount: ev.ReplyCount,
	}
	if err := h.messages.SaveMessage(c.Request.Context(), userID, channelUUID, msg); err != nil {
		slog.Info("slack webhook: SaveMessage failed", "error", err)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "save failed"})
		return
	}

	// Resolve the canonical BO uuid for the message we just upserted so
	// the SSE payload matches what the GET endpoints return.
	var messageUUID string
	if rerr := h.pool.QueryRow(c.Request.Context(), `
		SELECT id FROM slack_messages WHERE user_id = $1 AND channel_id = $2 AND slack_ts = $3
	`, userID, channelUUID, ev.TS).Scan(&messageUUID); rerr != nil {
		slog.Info("slack webhook: refetch message failed", "error", rerr)
	}

	publishMessageReceived(h.broadcaster, userID, MessageReceivedPayload{
		Provider:  "slack",
		ChannelID: channelUUID,
		MessageID: messageUUID,
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleChannelEvent broadcasts a channel.updated SSE without persisting
// in Wave 3 — channel_created arrives before the row exists in
// slack_channels (the next manual sync will populate it). Frontend can
// trigger a re-fetch on the SSE event.
func (h *slackWebhookHandler) handleChannelEvent(c *gin.Context, userID string, eventRaw json.RawMessage, change string) {
	var ev struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	_ = json.Unmarshal(eventRaw, &ev)

	publishChannelUpdated(h.broadcaster, userID, ChannelUpdatedPayload{
		Provider:  "slack",
		ChannelID: ev.Channel.ID, // Slack id; frontend should re-fetch by listing
		Change:    change,
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// lookupUserByTeam resolves Slack team_id → BO user_id. Returns empty
// string when no mapping exists (e.g. webhook fires before OAuth saves
// the token, or after disconnect).
func (h *slackWebhookHandler) lookupUserByTeam(ctx context.Context, teamID string) (string, error) {
	var userID string
	err := h.pool.QueryRow(ctx, `
		SELECT user_id FROM slack_oauth_tokens WHERE workspace_id = $1
	`, teamID).Scan(&userID)
	return userID, err
}
