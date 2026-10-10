package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
	"github.com/rhl/businessos-backend/internal/services"
)

// ============================================================================
// MICROSOFT GRAPH WEBHOOK
// ============================================================================
//
// POST /api/integrations/microsoft/webhook
// GET  /api/integrations/microsoft/webhook    (validation handshake)
//
// Two roles:
//
//   1. Validation handshake. When Graph creates or renews a subscription
//      it pings the notification URL with ?validationToken=X and
//      expects a 200 text/plain response containing X within 10 seconds.
//      We accept either GET or POST per docs (Graph has used both).
//
//   2. Notifications. Body is {value: [{subscriptionId, changeType,
//      resource, resourceData{id}, clientState, ...}]}.
//      For each notification:
//        - look up the subscription row to get the owning user_id and
//          expected clientState
//        - constant-time compare clientState (proves Graph sent it for
//          a subscription we own)
//        - dispatch by resource_kind:
//            "mail"           → OutlookService.FetchAndSaveMessage
//            "teams_messages" → TeamsService.FetchAndSaveChannelMessage
//        - broadcast SSE
//
// Per-user routing — clientState in the row encodes nothing user-
// identifying; we look up user_id via subscription_id. The clientState
// check is purely an authenticity proof.
//
// Acknowledge fast: Graph retries non-2xx responses and treats slow
// responses as failures. Heavy work (Graph fetch, DB upsert) runs
// inline today; if latency becomes a problem we can offload to a
// background worker and 202-ack immediately.

// microsoftWebhookHandler owns the Graph notification endpoint.
type microsoftWebhookHandler struct {
	pool        *pgxpool.Pool
	subs        *microsoft.SubscriptionService
	outlook     *microsoft.OutlookService
	teams       *microsoft.TeamsService
	broadcaster *services.SSEBroadcaster
}

func newMicrosoftWebhookHandler(
	pool *pgxpool.Pool,
	subs *microsoft.SubscriptionService,
	outlook *microsoft.OutlookService,
	teams *microsoft.TeamsService,
	broadcaster *services.SSEBroadcaster,
) *microsoftWebhookHandler {
	return &microsoftWebhookHandler{
		pool:        pool,
		subs:        subs,
		outlook:     outlook,
		teams:       teams,
		broadcaster: broadcaster,
	}
}

// HandleNotification is the single entry point — switches between
// validation handshake and event dispatch by inspecting query params.
//
// Graph spec: validation request always carries ?validationToken=...;
// notification requests don't. POST is used for both kinds (some docs
// pages mention GET for validation, but production Graph uses POST).
func (h *microsoftWebhookHandler) HandleNotification(c *gin.Context) {
	if token := c.Query("validationToken"); token != "" {
		// Echo back as text/plain. Graph rejects JSON or extra
		// quoting — must be the bare token, content-type text/plain.
		c.Header("Content-Type", "text/plain")
		_, _ = c.Writer.WriteString(token)
		c.Status(http.StatusOK)
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var envelope struct {
		Value []graphNotification `json:"value"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	for _, n := range envelope.Value {
		h.dispatchNotification(c.Request.Context(), n)
	}

	// Graph just wants a 2xx — body is ignored.
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// graphNotification is the single-element shape Microsoft Graph delivers
// inside the {value: [...]} envelope. We only care about the fields
// below; Graph also sends tenantId, subscriptionExpirationDateTime, etc.
type graphNotification struct {
	SubscriptionID string `json:"subscriptionId"`
	ChangeType     string `json:"changeType"`
	Resource       string `json:"resource"`
	ResourceData   struct {
		ID       string `json:"id"`
		ODataID  string `json:"@odata.id"`
		ODataEtag string `json:"@odata.etag"`
	} `json:"resourceData"`
	ClientState string `json:"clientState"`
}

// dispatchNotification looks up the owning subscription row, verifies
// clientState, then routes to the right service. Errors are logged but
// not surfaced — Graph would just retry, and our upserts are idempotent
// so retries are safe but wasteful.
func (h *microsoftWebhookHandler) dispatchNotification(ctx context.Context, n graphNotification) {
	if n.SubscriptionID == "" {
		return
	}

	sub, err := h.subs.GetSubscription(ctx, n.SubscriptionID)
	if err != nil {
		// Notification for a subscription we don't recognize. Could be
		// a stale Graph subscription pointing at our URL after a DB
		// reset, or a clientState mismatch attacker probe. Log + drop.
		slog.Info("microsoft webhook: unknown subscription", "subscription_id", n.SubscriptionID)
		return
	}

	// Constant-time comparison so an attacker can't time the secret out.
	if subtle.ConstantTimeCompare([]byte(sub.ClientState), []byte(n.ClientState)) != 1 {
		slog.Warn("microsoft webhook: clientState mismatch",
			"subscription_id", n.SubscriptionID, "user_id", sub.UserID)
		return
	}

	switch sub.ResourceKind {
	case microsoft.ResourceKindMail:
		h.handleMailNotification(ctx, sub, n)
	case microsoft.ResourceKindTeamsMessages:
		h.handleTeamsNotification(ctx, sub, n)
	default:
		slog.Info("microsoft webhook: unsupported resource_kind",
			"resource_kind", sub.ResourceKind, "subscription_id", n.SubscriptionID)
	}
}

// handleMailNotification fetches the email by id, upserts (which fires
// OnEmailSaved → engine), and broadcasts SSE.
func (h *microsoftWebhookHandler) handleMailNotification(ctx context.Context, sub *microsoft.SubscriptionRow, n graphNotification) {
	if n.ResourceData.ID == "" {
		return
	}
	if err := h.outlook.FetchAndSaveMessage(ctx, sub.UserID, n.ResourceData.ID); err != nil {
		slog.Info("microsoft webhook: outlook fetch+save failed",
			"message_id", n.ResourceData.ID, "error", err)
		return
	}

	// Resolve canonical BO uuid for the SSE payload so the frontend
	// can refetch through the existing GET endpoints.
	var emailUUID string
	_ = h.pool.QueryRow(ctx, `
		SELECT id FROM microsoft_mail_messages
		WHERE user_id = $1 AND message_id = $2
	`, sub.UserID, n.ResourceData.ID).Scan(&emailUUID)

	// changeType is comma-separated when multiple kinds bundled. For
	// "created" we emit email.received; for "updated" we emit
	// email.updated. The notification's discriminator is informational —
	// frontend reacts to the row state regardless.
	if strings.Contains(n.ChangeType, "updated") && !strings.Contains(n.ChangeType, "created") {
		publishEmailUpdated(h.broadcaster, sub.UserID, EmailUpdatedPayload{
			Provider: "outlook",
			EmailID:  emailUUID,
		})
	} else {
		publishEmailReceived(h.broadcaster, sub.UserID, EmailReceivedPayload{
			Provider: "outlook",
			EmailID:  emailUUID,
		})
	}
}

// handleTeamsNotification fetches the Teams chatMessage and upserts.
// Resource path looks like:
//
//	teams/{team-id}/channels/{channel-id}/messages/{message-id}
//
// We parse out team_id + channel_id from the resource path on the
// subscription row (more reliable than the per-notification resource
// string, which can vary slightly in beta endpoints).
func (h *microsoftWebhookHandler) handleTeamsNotification(ctx context.Context, sub *microsoft.SubscriptionRow, n graphNotification) {
	if n.ResourceData.ID == "" {
		return
	}
	teamID, channelID, ok := parseTeamsResource(sub.Resource)
	if !ok {
		slog.Info("microsoft webhook: cannot parse teams resource",
			"resource", sub.Resource)
		return
	}
	if err := h.teams.FetchAndSaveChannelMessage(ctx, sub.UserID, teamID, channelID, n.ResourceData.ID); err != nil {
		slog.Info("microsoft webhook: teams fetch+save failed",
			"message_id", n.ResourceData.ID, "error", err)
		return
	}

	// Resolve BO uuids for the SSE payload — channel uuid for ChannelID,
	// message uuid for MessageID, both keyed off the Graph identifiers.
	var channelUUID string
	_ = h.pool.QueryRow(ctx, `
		SELECT id FROM microsoft_teams_channels
		WHERE user_id = $1 AND team_id = $2 AND channel_id = $3
	`, sub.UserID, teamID, channelID).Scan(&channelUUID)

	var messageUUID string
	_ = h.pool.QueryRow(ctx, `
		SELECT id FROM microsoft_teams_messages
		WHERE user_id = $1 AND team_id = $2 AND channel_id = $3 AND message_id = $4
	`, sub.UserID, teamID, channelID, n.ResourceData.ID).Scan(&messageUUID)

	publishMessageReceived(h.broadcaster, sub.UserID, MessageReceivedPayload{
		Provider:  "teams",
		ChannelID: channelUUID,
		MessageID: messageUUID,
	})
}

// parseTeamsResource extracts (teamID, channelID) from a resource path
// like "teams/<tid>/channels/<cid>/messages". Returns ok=false on
// any other shape so callers can log + skip.
func parseTeamsResource(resource string) (string, string, bool) {
	// Expected: ["teams", "<tid>", "channels", "<cid>", "messages"]
	parts := strings.Split(strings.Trim(resource, "/"), "/")
	if len(parts) < 5 || parts[0] != "teams" || parts[2] != "channels" || parts[4] != "messages" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

// validationFailedf is shorthand for the validation-handshake error
// path we'd want if the token format ever drifts. Unused today but
// kept as a reminder that Graph's validation contract is brittle —
// add explicit error text here if Microsoft updates the spec.
var _ = func() error { return fmt.Errorf("validation token unexpected format") }
