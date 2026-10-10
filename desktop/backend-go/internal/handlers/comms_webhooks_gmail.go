package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/services"
	"google.golang.org/api/idtoken"
)

// ============================================================================
// GMAIL PUB/SUB WEBHOOK
// ============================================================================
//
// POST /api/integrations/google_gmail/webhook
//
// Google Cloud Pub/Sub pushes notifications here whenever the watched
// Gmail account changes. The body is a Pub/Sub envelope:
//
//   {
//     "message": {
//       "data": "<base64-encoded JSON>",            // {emailAddress, historyId}
//       "messageId": "...",
//       "publishTime": "..."
//     },
//     "subscription": "projects/<proj>/subscriptions/<sub>"
//   }
//
// Authentication: Pub/Sub signs every push with a Google-issued JWT in
// Authorization: Bearer <token>. We verify the JWT against Google's
// public keys when GOOGLE_GMAIL_PUBSUB_AUDIENCE is configured (the
// audience must be the public webhook URL). Without the audience set,
// we accept unverified pushes — dev mode.
//
// Per-user routing: notification body has emailAddress; we look it up
// in google_oauth_tokens.google_email to find the BO user_id. From
// there GmailService.HandleHistoryEvent does the diff fetch + saves
// (which fires Wave 1's OnEmailSaved hook).

type gmailWebhookHandler struct {
	pool        *pgxpool.Pool
	gmail       *google.GmailService
	broadcaster *services.SSEBroadcaster
	audience    string // when non-empty, JWT verification is enforced
}

func newGmailWebhookHandler(
	pool *pgxpool.Pool,
	gmailSvc *google.GmailService,
	broadcaster *services.SSEBroadcaster,
	audience string,
) *gmailWebhookHandler {
	return &gmailWebhookHandler{
		pool:        pool,
		gmail:       gmailSvc,
		broadcaster: broadcaster,
		audience:    audience,
	}
}

// HandleNotification is the Pub/Sub push endpoint. Always 2xx unless
// the body is unrecoverable — Pub/Sub retries non-2xx responses with
// exponential backoff, which would create duplicate work for transient
// errors that don't deserve a retry.
func (h *gmailWebhookHandler) HandleNotification(c *gin.Context) {
	if h.audience == "" {
		slog.Warn("gmail webhook: GOOGLE_GMAIL_PUBSUB_AUDIENCE not configured — accepting unverified push")
	} else if err := h.verifyJWT(c); err != nil {
		slog.Warn("gmail webhook: JWT verification failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var envelope struct {
		Message struct {
			Data      string `json:"data"`      // base64 of inner JSON
			MessageID string `json:"messageId"`
		} `json:"message"`
		Subscription string `json:"subscription"`
	}
	if err := json.Unmarshal(rawBody, &envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid envelope"})
		return
	}

	innerBytes, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		// 200 anyway — bad data is not retriable.
		slog.Info("gmail webhook: base64 decode failed", "error", err)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "bad data"})
		return
	}

	var inner struct {
		EmailAddress string `json:"emailAddress"`
		HistoryID    uint64 `json:"historyId"`
	}
	if err := json.Unmarshal(innerBytes, &inner); err != nil {
		slog.Info("gmail webhook: inner json failed", "error", err)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "bad inner json"})
		return
	}

	if inner.EmailAddress == "" {
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "missing email"})
		return
	}

	userID, err := h.gmail.LookupUserByEmail(c.Request.Context(), inner.EmailAddress)
	if err != nil || userID == "" {
		// Pre-OAuth or post-disconnect notification. Ack so Pub/Sub
		// doesn't retry forever.
		slog.Info("gmail webhook: no user for email", "email", inner.EmailAddress)
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "no user"})
		return
	}

	// HandleHistoryEvent lists every messageAdded since the stored
	// history_id, fetches each, and saves (firing OnEmailSaved).
	saved, err := h.gmail.HandleHistoryEvent(c.Request.Context(), userID)
	if err != nil {
		slog.Info("gmail webhook: history event failed", "user_id", userID, "error", err)
		// Still 200 — retry won't help if the watch row is stale.
		c.JSON(http.StatusOK, gin.H{"ok": true, "skipped": "history failed"})
		return
	}

	// Broadcast one SSE per saved message would over-fire; the realistic
	// frontend cue is "something arrived for this account, refetch the
	// inbox view". Send one email.received marker per push with EmailID
	// left empty — the frontend uses this as an invalidation cue.
	if saved > 0 {
		publishEmailReceived(h.broadcaster, userID, EmailReceivedPayload{
			Provider:  "gmail",
			AccountID: inner.EmailAddress,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
}

// verifyJWT validates the Pub/Sub Authorization Bearer token against
// Google's public keys, asserting the audience matches our configured
// webhook URL. Pub/Sub uses Google-issued OIDC tokens for push auth.
//
// Reference: https://cloud.google.com/pubsub/docs/authenticate-push-subscriptions
func (h *gmailWebhookHandler) verifyJWT(c *gin.Context) error {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return errAuthMissing
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return errAuthFormat
	}
	token := strings.TrimPrefix(auth, "Bearer ")

	_, err := idtoken.Validate(c.Request.Context(), token, h.audience)
	return err
}

// Sentinel errors so callers can branch without string matching.
var (
	errAuthMissing = simpleErr("authorization header missing")
	errAuthFormat  = simpleErr("authorization header not Bearer token")
)

type simpleErr string

func (e simpleErr) Error() string { return string(e) }