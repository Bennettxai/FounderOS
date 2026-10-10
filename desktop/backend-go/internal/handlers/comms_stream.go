package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/services"
)

// ============================================================================
// COMMS REALTIME — unified SSE stream + typed event publishers
// ============================================================================
//
// One Server-Sent Events endpoint multiplexes every realtime event across
// the Communications module:
//
//   GET /api/comms/stream
//
// Frontend opens a single EventSource from the comms hub layout and fans
// the events out into per-tab stores. Wave 3 events (the contract Fantem
// builds against):
//
//   event: connected           data: { user_id }                                first message after subscribe
//   event: heartbeat           data: { ts }                                     every 25s; defeats proxy idle timeouts
//   event: email.received      data: EmailReceivedPayload                       new email arrived (any provider)
//   event: email.updated       data: EmailUpdatedPayload                        existing email mutated (read/star/archive)
//   event: message.received    data: MessageReceivedPayload                     new channel/dm message
//   event: channel.updated     data: ChannelUpdatedPayload                      channel-level change (created, renamed, member added)
//
// Webhooks publish via the typed helpers below. Keep payloads narrow on
// purpose: the frontend uses these as invalidation cues — it then refetches
// the actual row through the existing GET endpoints. We don't push full
// row bodies because (a) keeps the SSE channel small, (b) preserves
// single-source-of-truth at the read endpoints, (c) lets the frontend
// reuse its cached fetch logic instead of duplicating row hydration.
//
// In-process only. Multi-instance fan-out is a Wave 4 concern (swap
// SSEBroadcaster's channel layer for Redis pub/sub).

// CommsEventType enumerates the event names emitted on /api/comms/stream.
// Defined as constants so webhook handlers don't typo a string literal —
// the frontend reads these names directly from the SSE wire and matches
// on string equality.
const (
	CommsEventConnected       = "connected"
	CommsEventHeartbeat       = "heartbeat"
	CommsEventEmailReceived   = "email.received"
	CommsEventEmailUpdated    = "email.updated"
	CommsEventMessageReceived = "message.received"
	CommsEventChannelUpdated  = "channel.updated"
)

// commsHeartbeatInterval is how often we emit a heartbeat to keep the
// connection from being reaped by intermediate proxies (most cap idle
// at 30-60s). 25s is the well-known SSE convention.
const commsHeartbeatInterval = 25 * time.Second

// EmailReceivedPayload is the data block for `email.received`.
// AccountID identifies which connected account the email belongs to so
// the frontend can route it to the right inbox view.
type EmailReceivedPayload struct {
	Provider  string `json:"provider"`   // "gmail" | "outlook"
	AccountID string `json:"account_id"` // email of the connected account
	EmailID   string `json:"email_id"`   // BO row uuid
}

// EmailUpdatedPayload is the data block for `email.updated`. Changed
// lists which fields shifted so the frontend can be selective about
// re-rendering ("read" → only flip the badge, "labels" → re-sort).
type EmailUpdatedPayload struct {
	Provider  string   `json:"provider"`
	AccountID string   `json:"account_id"`
	EmailID   string   `json:"email_id"`
	Changed   []string `json:"changed,omitempty"` // e.g. ["is_read", "labels"]
}

// MessageReceivedPayload is the data block for `message.received`.
type MessageReceivedPayload struct {
	Provider  string `json:"provider"`   // "slack" | "teams"
	ChannelID string `json:"channel_id"` // BO row uuid
	MessageID string `json:"message_id"` // BO row uuid
}

// ChannelUpdatedPayload is the data block for `channel.updated`. Change
// is a free-form discriminator ("created", "renamed", "member_added",
// "archived"). Frontend can ignore unknown values safely.
type ChannelUpdatedPayload struct {
	Provider  string `json:"provider"`
	ChannelID string `json:"channel_id"`
	Change    string `json:"change,omitempty"`
}

// ----------------------------------------------------------------------------
// Endpoint — GET /api/comms/stream
// ----------------------------------------------------------------------------

// Stream handles SSE connections for the unified Communications stream.
// One connection per browser tab; the frontend reconnects with backoff
// when the channel drops.
//
// Mirror of the established notifications.go::Stream pattern, with one
// addition: a 25s heartbeat ticker so proxies don't reap idle connections.
func (h *commsHandler) Stream(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// SSE headers — same set every framework expects.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// SSEBroadcaster.Subscribe returns nil when the per-user connection
	// limit (10) is reached. Surface that as 429 so the frontend knows to
	// back off rather than retry immediately.
	ch := h.broadcaster.Subscribe(userID)
	if ch == nil {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "Too many SSE connections for this user",
		})
		return
	}
	defer h.broadcaster.Unsubscribe(userID, ch)

	// Initial event — Fantem keys his "connected" UI state off this.
	c.SSEvent(CommsEventConnected, gin.H{"user_id": userID})
	c.Writer.Flush()

	// Heartbeat ticker — keeps the TCP/HTTP connection warm through
	// proxies that idle-timeout SSE in 30-60s. Frontend ignores the
	// payload but the bytes-on-the-wire keep the socket alive.
	heartbeat := time.NewTicker(commsHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				// Broadcaster closed our channel (cleanup or shutdown).
				return
			}
			c.SSEvent(event.Type, event.Data)
			c.Writer.Flush()
		case <-heartbeat.C:
			c.SSEvent(CommsEventHeartbeat, gin.H{"ts": time.Now().UTC().Format(time.RFC3339)})
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// ----------------------------------------------------------------------------
// Typed publishers — called from webhook handlers (Wave 3 chunks 2-4).
// ----------------------------------------------------------------------------
//
// Why typed wrappers instead of raw broadcaster.SendToUser calls? Two
// reasons: (a) every webhook handler ends up emitting one of these four
// events, so centralizing the SSEEvent shape stops the same JSON layout
// from being duplicated four times, and (b) compile-time checks against
// the payload structs catch shape drift faster than string-keyed maps.

// publishEmailReceived broadcasts a `email.received` event.
func publishEmailReceived(b *services.SSEBroadcaster, userID string, payload EmailReceivedPayload) {
	if b == nil {
		return
	}
	b.SendToUser(userID, services.SSEEvent{
		Type: CommsEventEmailReceived,
		Data: payload,
	})
}

// publishEmailUpdated broadcasts a `email.updated` event.
func publishEmailUpdated(b *services.SSEBroadcaster, userID string, payload EmailUpdatedPayload) {
	if b == nil {
		return
	}
	b.SendToUser(userID, services.SSEEvent{
		Type: CommsEventEmailUpdated,
		Data: payload,
	})
}

// publishMessageReceived broadcasts a `message.received` event.
func publishMessageReceived(b *services.SSEBroadcaster, userID string, payload MessageReceivedPayload) {
	if b == nil {
		return
	}
	b.SendToUser(userID, services.SSEEvent{
		Type: CommsEventMessageReceived,
		Data: payload,
	})
}

// publishChannelUpdated broadcasts a `channel.updated` event.
func publishChannelUpdated(b *services.SSEBroadcaster, userID string, payload ChannelUpdatedPayload) {
	if b == nil {
		return
	}
	b.SendToUser(userID, services.SSEEvent{
		Type: CommsEventChannelUpdated,
		Data: payload,
	})
}

// ----------------------------------------------------------------------------
// Internal accessors — called by the package's other comms files.
// ----------------------------------------------------------------------------

// emitEmailReceived is the in-handler shorthand. Methods on commsHandler
// can call h.emitEmailReceived(...) instead of plumbing the broadcaster
// reference manually.
func (h *commsHandler) emitEmailReceived(userID string, p EmailReceivedPayload) {
	publishEmailReceived(h.broadcaster, userID, p)
}

func (h *commsHandler) emitEmailUpdated(userID string, p EmailUpdatedPayload) {
	publishEmailUpdated(h.broadcaster, userID, p)
}

func (h *commsHandler) emitMessageReceived(userID string, p MessageReceivedPayload) {
	publishMessageReceived(h.broadcaster, userID, p)
}

func (h *commsHandler) emitChannelUpdated(userID string, p ChannelUpdatedPayload) {
	publishChannelUpdated(h.broadcaster, userID, p)
}

// commsConnectedCount surfaces the broadcaster's connected-user count so
// ops endpoints (or future /healthz subroutes) can report realtime
// presence. Not wired to a route in Wave 3; helper for ops use.
func (h *commsHandler) commsConnectedCount() int {
	if h.broadcaster == nil {
		return 0
	}
	return h.broadcaster.ConnectedCount()
}

// Note: keep this file thin. Webhook handlers in chunks 2-4 own their own
// signature/auth flow and call the emit* helpers above with a concrete
// payload. The broadcaster, the event-name constants, and the payload
// shapes are the entire wave-3 contract surface.