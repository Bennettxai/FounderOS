package services

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// WebPushConfig holds VAPID configuration
type WebPushConfig struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDContact    string // Email or URL for VAPID contact
}

// WebPushService handles Web Push notifications
type WebPushService struct {
	pool   *pgxpool.Pool
	config *WebPushConfig
}

// PushPayload represents the notification payload sent to browsers
type PushPayload struct {
	Title    string                 `json:"title"`
	Body     string                 `json:"body,omitempty"`
	Icon     string                 `json:"icon,omitempty"`
	Badge    string                 `json:"badge,omitempty"`
	Tag      string                 `json:"tag,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Actions  []PushAction           `json:"actions,omitempty"`
	Vibrate  []int                  `json:"vibrate,omitempty"`
	Priority string                 `json:"priority,omitempty"` // "high", "normal", "low"
}

// PushAction represents an action button on the notification
type PushAction struct {
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// SubscriptionInput represents input for subscribing to push
type SubscriptionInput struct {
	Endpoint  string `json:"endpoint"`
	P256dh    string `json:"p256dh"`
	Auth      string `json:"auth"`
	UserAgent string `json:"user_agent,omitempty"`
}

// NewWebPushService creates a new Web Push service
func NewWebPushService(pool *pgxpool.Pool, config *WebPushConfig) *WebPushService {
	if config == nil {
		// Return a service that will log but not send
		slog.Info("[WebPush] No VAPID config provided - push notifications disabled")
		return &WebPushService{pool: pool}
	}

	return &WebPushService{
		pool:   pool,
		config: config,
	}
}

// GetPublicKey returns the VAPID public key for frontend use
func (s *WebPushService) GetPublicKey() string {
	if s.config == nil {
		return ""
	}
	return s.config.VAPIDPublicKey
}

// IsEnabled returns true if Web Push is configured
func (s *WebPushService) IsEnabled() bool {
	return s.config != nil && s.config.VAPIDPublicKey != "" && s.config.VAPIDPrivateKey != ""
}

// Subscribe saves a push subscription for a user
func (s *WebPushService) Subscribe(ctx context.Context, userID string, input SubscriptionInput) error {
	queries := sqlc.New(s.pool)

	_, err := queries.CreatePushSubscription(ctx, sqlc.CreatePushSubscriptionParams{
		UserID:    userID,
		Endpoint:  input.Endpoint,
		P256dh:    input.P256dh,
		Auth:      input.Auth,
		UserAgent: &input.UserAgent,
	})
	if err != nil {
		return err
	}

	slog.Info("[WebPush] Subscription saved for user", "user_id", userID)
	return nil
}

// Unsubscribe removes a push subscription
func (s *WebPushService) Unsubscribe(ctx context.Context, userID string, endpoint string) error {
	queries := sqlc.New(s.pool)
	return queries.DeletePushSubscription(ctx, sqlc.DeletePushSubscriptionParams{
		Endpoint: endpoint,
		UserID:   userID,
	})
}

// SendToUser sends a push notification to all of a user's subscribed devices
func (s *WebPushService) SendToUser(ctx context.Context, userID string, payload PushPayload) error {
	if !s.IsEnabled() {
		slog.Info("[WebPush] Skipping push to user  - not configured", "user_id", userID)
		return nil
	}

	queries := sqlc.New(s.pool)
	subscriptions, err := queries.GetPushSubscriptionsByUser(ctx, userID)
	if err != nil {
		return err
	}

	if len(subscriptions) == 0 {
		slog.Info("[WebPush] No subscriptions found for user", "user_id", userID)
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	successCount := 0

	for _, sub := range subscriptions {
		subscription := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payloadJSON, subscription, &webpush.Options{
			VAPIDPublicKey:  s.config.VAPIDPublicKey,
			VAPIDPrivateKey: s.config.VAPIDPrivateKey,
			Subscriber:      s.config.VAPIDContact,
			TTL:             86400, // 24 hours
		})

		if err != nil {
			slog.Info("[WebPush] Failed to send to endpoint", "id", sub.Endpoint[:50], "error", err)
			lastErr = err

			// If subscription is expired or invalid, remove it
			if resp != nil && (resp.StatusCode == 404 || resp.StatusCode == 410) {
				_ = queries.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint)
				slog.Info("[WebPush] Removed expired subscription for user", "user_id", userID)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successCount++
		} else {
			slog.Info("[WebPush] Unexpected status  for endpoint", "value", resp.StatusCode)
		}
	}

	slog.Info("[WebPush] Sent push notifications", "success", successCount, "total", len(subscriptions), "user_id", userID)
	return lastErr
}

// SendNotification converts a Notification to PushPayload and sends it
func (s *WebPushService) SendNotification(ctx context.Context, notif *Notification) error {
	payload := PushPayload{
		Title:    notif.Title,
		Body:     notif.Body,
		Icon:     "/icon-192.png",
		Badge:    "/badge-72.png",
		Tag:      notif.Type, // Group notifications by type
		Priority: notif.Priority,
		Data: map[string]interface{}{
			"notification_id": notif.ID.String(),
			"type":            notif.Type,
			"entity_type":     notif.EntityType,
			"entity_id":       notif.EntityID,
			"url":             s.getNotificationURL(notif),
		},
	}

	return s.SendToUser(ctx, notif.UserID, payload)
}

// getNotificationURL returns the URL to navigate to when notification is clicked
func (s *WebPushService) getNotificationURL(notif *Notification) string {
	if notif.EntityType == "" || notif.EntityID == nil {
		return "/"
	}

	switch notif.EntityType {
	case "task":
		return "/tasks/" + notif.EntityID.String()
	case "project":
		return "/projects/" + notif.EntityID.String()
	case "client":
		return "/clients/" + notif.EntityID.String()
	default:
		return "/"
	}
}

// GenerateVAPIDKeys generates a new VAPID key pair (for initial setup)
func GenerateVAPIDKeys() (publicKey, privateKey string, err error) {
	return webpush.GenerateVAPIDKeys()
}
