package microsoft

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// MICROSOFT GRAPH SUBSCRIPTIONS
// ============================================================================
//
// Subscribes a user's Outlook mailbox and Teams channel inboxes to push
// notifications. Microsoft Graph delivers a webhook to our notification
// URL whenever a tracked resource changes; the receiver lives in
// internal/handlers/comms_webhooks_microsoft.go.
//
// Lifecycle:
//   - Subscribe* methods POST to /subscriptions, store the row in
//     microsoft_subscriptions, return the subscription id + clientState.
//   - The renewal scheduler (RenewalLoop) wakes every hour and PATCHes
//     any subscription expiring within 2 hours.
//   - Graph caps message subscriptions at ~71 hours; we ask for 70h
//     to leave ourselves margin against clock skew + scheduler latency.
//
// clientState is a random 32-byte hex secret stored alongside the
// subscription row. Every notification echoes it back; the webhook
// receiver compares to prove the notification really came from Graph
// for a subscription we own.

// SubscriptionService manages Graph subscriptions for one provider
// instance. Lives alongside OutlookService / TeamsService — separate
// so the renewal loop has a single owner regardless of which resource
// kind it's renewing.
type SubscriptionService struct {
	provider *Provider
	pool     *pgxpool.Pool
}

// NewSubscriptionService constructs a subscription service.
func NewSubscriptionService(provider *Provider) *SubscriptionService {
	return &SubscriptionService{
		provider: provider,
		pool:     provider.Pool(),
	}
}

// ----------------------------------------------------------------------------
// Resource kinds
// ----------------------------------------------------------------------------

// ResourceKind discriminates how the webhook receiver dispatches a
// notification — straight into OutlookService.FetchAndSaveMessage or
// TeamsService.FetchAndSaveChannelMessage. Stored alongside each row so
// the dispatcher doesn't re-parse Graph resource paths.
const (
	ResourceKindMail          = "mail"
	ResourceKindTeamsMessages = "teams_messages"
)

// SubscriptionTTL is how long we ask Graph to keep our subscriptions
// alive. Graph maxes out at ~71h for messages; 70h gives us a 1h safety
// margin over the renewal scheduler's hourly tick.
const SubscriptionTTL = 70 * time.Hour

// ----------------------------------------------------------------------------
// Graph DTOs (only the fields we send/receive)
// ----------------------------------------------------------------------------

type subscriptionCreateRequest struct {
	ChangeType         string `json:"changeType"`
	NotificationURL    string `json:"notificationUrl"`
	Resource           string `json:"resource"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState"`
}

type subscriptionResponse struct {
	ID                 string `json:"id"`
	Resource           string `json:"resource"`
	ChangeType         string `json:"changeType"`
	NotificationURL    string `json:"notificationUrl"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState"`
}

// ----------------------------------------------------------------------------
// Subscribe / Unsubscribe / Renew
// ----------------------------------------------------------------------------

// SubscribeMail creates a Graph subscription on /me/messages and
// records it. notificationURL is the public HTTPS URL Graph should POST
// to (typically https://<host>/api/integrations/microsoft/webhook).
//
// On success returns the Graph subscription id; on failure both the
// Graph call and the DB insert are reverted so partial state never
// leaks (Graph's own subscription is rolled back on insert failure via
// a follow-up DELETE, best-effort).
func (s *SubscriptionService) SubscribeMail(ctx context.Context, userID, notificationURL string) (string, error) {
	return s.subscribe(ctx, userID, "/me/messages", "created,updated", ResourceKindMail, notificationURL)
}

// SubscribeTeamsChannel subscribes to a single Teams channel's messages.
// teamID + channelID are the Graph identifiers (NOT BO uuids).
// Per Graph docs each channel subscription is scoped to one channel —
// users with N channels need N subscriptions. The renewal loop renews
// all of them.
func (s *SubscriptionService) SubscribeTeamsChannel(ctx context.Context, userID, teamID, channelID, notificationURL string) (string, error) {
	resource := fmt.Sprintf("/teams/%s/channels/%s/messages", teamID, channelID)
	return s.subscribe(ctx, userID, resource, "created", ResourceKindTeamsMessages, notificationURL)
}

func (s *SubscriptionService) subscribe(ctx context.Context, userID, resource, changeType, kind, notificationURL string) (string, error) {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get http client: %w", err)
	}

	clientState, err := randomHex(32)
	if err != nil {
		return "", fmt.Errorf("generate clientState: %w", err)
	}

	body := subscriptionCreateRequest{
		ChangeType:         changeType,
		NotificationURL:    notificationURL,
		Resource:           resource,
		ExpirationDateTime: time.Now().Add(SubscriptionTTL).UTC().Format(time.RFC3339),
		ClientState:        clientState,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := client.Post(GraphAPIBase+"/subscriptions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("graph subscribe: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("graph subscribe: %s", resp.Status)
	}

	var created subscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode subscribe response: %w", err)
	}

	expiresAt, _ := time.Parse(time.RFC3339, created.ExpirationDateTime)

	if _, dbErr := s.pool.Exec(ctx, `
		INSERT INTO microsoft_subscriptions (
			user_id, subscription_id, resource, resource_kind, change_type,
			notification_url, client_state, expires_at, last_renewed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id, subscription_id) DO UPDATE SET
			resource = EXCLUDED.resource,
			resource_kind = EXCLUDED.resource_kind,
			change_type = EXCLUDED.change_type,
			notification_url = EXCLUDED.notification_url,
			client_state = EXCLUDED.client_state,
			expires_at = EXCLUDED.expires_at,
			last_renewed_at = NOW(),
			updated_at = NOW()
	`, userID, created.ID, resource, kind, changeType, notificationURL, clientState, expiresAt); dbErr != nil {
		// Graph subscription created but DB insert failed — best-effort
		// rollback so we don't have an orphan Graph subscription pinging
		// a webhook we can't authenticate.
		_ = s.deleteFromGraph(ctx, userID, created.ID)
		return "", fmt.Errorf("save subscription: %w", dbErr)
	}

	slog.Info("microsoft subscription created",
		"user_id", userID,
		"subscription_id", created.ID,
		"resource_kind", kind,
		"expires_at", expiresAt)
	return created.ID, nil
}

// Unsubscribe deletes a Graph subscription and removes the local row.
func (s *SubscriptionService) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	if err := s.deleteFromGraph(ctx, userID, subscriptionID); err != nil {
		return err
	}
	_, _ = s.pool.Exec(ctx, `
		DELETE FROM microsoft_subscriptions
		WHERE user_id = $1 AND subscription_id = $2
	`, userID, subscriptionID)
	return nil
}

// Renew PATCHes a single subscription's expirationDateTime. The renewal
// loop calls this on rows expiring within 2h.
func (s *SubscriptionService) Renew(ctx context.Context, userID, subscriptionID string) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return fmt.Errorf("get http client: %w", err)
	}

	body := map[string]string{
		"expirationDateTime": time.Now().Add(SubscriptionTTL).UTC().Format(time.RFC3339),
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		GraphAPIBase+"/subscriptions/"+subscriptionID, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("graph renew: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graph renew: %s", resp.Status)
	}

	var renewed subscriptionResponse
	_ = json.NewDecoder(resp.Body).Decode(&renewed)
	expiresAt, _ := time.Parse(time.RFC3339, renewed.ExpirationDateTime)

	_, _ = s.pool.Exec(ctx, `
		UPDATE microsoft_subscriptions
		SET expires_at = $1, last_renewed_at = NOW(), updated_at = NOW()
		WHERE user_id = $2 AND subscription_id = $3
	`, expiresAt, userID, subscriptionID)
	return nil
}

func (s *SubscriptionService) deleteFromGraph(ctx context.Context, userID, subscriptionID string) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		GraphAPIBase+"/subscriptions/"+subscriptionID, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 204 NoContent on success; treat 404 as already-deleted.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("graph delete sub: %s", resp.Status)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Lookup — used by the webhook dispatcher
// ----------------------------------------------------------------------------

// SubscriptionRow is the persisted view a webhook handler reads to
// dispatch a notification: which user owns the subscription, what kind
// of resource it's for, and the expected clientState.
type SubscriptionRow struct {
	UserID         string
	SubscriptionID string
	Resource       string
	ResourceKind   string
	ClientState    string
	ExpiresAt      time.Time
}

// GetSubscription resolves a Graph subscription_id to its owning user
// + resource kind + expected clientState. Returns a not-found error
// when the subscription isn't ours (or has been deleted).
func (s *SubscriptionService) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionRow, error) {
	var row SubscriptionRow
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, subscription_id, resource, resource_kind, client_state, expires_at
		FROM microsoft_subscriptions
		WHERE subscription_id = $1
	`, subscriptionID).Scan(
		&row.UserID, &row.SubscriptionID, &row.Resource, &row.ResourceKind, &row.ClientState, &row.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListExpiringSoon returns subscriptions whose expires_at is within
// `within`. Used by the renewal loop.
func (s *SubscriptionService) ListExpiringSoon(ctx context.Context, within time.Duration) ([]SubscriptionRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, subscription_id, resource, resource_kind, client_state, expires_at
		FROM microsoft_subscriptions
		WHERE expires_at IS NOT NULL AND expires_at <= NOW() + $1::interval
		ORDER BY expires_at
	`, fmt.Sprintf("%d seconds", int(within.Seconds())))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubscriptionRow
	for rows.Next() {
		var r SubscriptionRow
		if err := rows.Scan(&r.UserID, &r.SubscriptionID, &r.Resource, &r.ResourceKind, &r.ClientState, &r.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// Renewal loop
// ----------------------------------------------------------------------------

// RenewalLoop runs forever, renewing subscriptions every hour. Designed
// to be started in a goroutine from the server bootstrap. Honors ctx
// for graceful shutdown.
func (s *SubscriptionService) RenewalLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Run once immediately on boot so a freshly-restarted server doesn't
	// wait an hour to discover overdue subs.
	s.renewExpiringSoon(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.renewExpiringSoon(ctx)
		}
	}
}

func (s *SubscriptionService) renewExpiringSoon(ctx context.Context) {
	// 2h window leaves margin for transient Graph errors + retries.
	expiring, err := s.ListExpiringSoon(ctx, 2*time.Hour)
	if err != nil {
		slog.Info("microsoft subscriptions: list expiring failed", "error", err)
		return
	}
	for _, sub := range expiring {
		if err := s.Renew(ctx, sub.UserID, sub.SubscriptionID); err != nil {
			slog.Info("microsoft subscriptions: renew failed",
				"subscription_id", sub.SubscriptionID, "error", err)
		}
	}
	if len(expiring) > 0 {
		slog.Info("microsoft subscriptions: renewal pass complete", "count", len(expiring))
	}
}

// randomHex returns n bytes of random data hex-encoded.
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
