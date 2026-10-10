package services

import (
	"context"
	"log/slog"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// TypeChannelSettings represents per-type channel overrides from type_settings JSONB
type TypeChannelSettings struct {
	InApp *bool `json:"in_app,omitempty"`
	Push  *bool `json:"push,omitempty"`
	Email *bool `json:"email,omitempty"`
}

// NotificationDispatcher routes notifications to appropriate channels based on user preferences
type NotificationDispatcher struct {
	pool           *pgxpool.Pool
	sseBroadcaster *SSEBroadcaster
	webPushService *WebPushService
	emailService   *EmailService
}

// NewNotificationDispatcher creates a new dispatcher
func NewNotificationDispatcher(pool *pgxpool.Pool, sseBroadcaster *SSEBroadcaster) *NotificationDispatcher {
	return &NotificationDispatcher{
		pool:           pool,
		sseBroadcaster: sseBroadcaster,
	}
}

// SetWebPushService sets the Web Push service (optional, can be nil)
func (d *NotificationDispatcher) SetWebPushService(svc *WebPushService) {
	d.webPushService = svc
}

// SetEmailService sets the Email service (optional, can be nil)
func (d *NotificationDispatcher) SetEmailService(svc *EmailService) {
	d.emailService = svc
}

// Dispatch sends a notification through enabled channels based on user preferences
func (d *NotificationDispatcher) Dispatch(ctx context.Context, notif *Notification) error {
	queries := sqlc.New(d.pool)

	// Get user preferences
	prefs, err := queries.GetNotificationPreferencesByUser(ctx, notif.UserID)
	if err != nil {
		// If no preferences exist, use defaults (all channels enabled)
		prefs = d.getDefaultPreferences(notif.UserID)
	}

	// Parse per-type settings for this notification type
	typeSettings := d.parseTypeSettings(prefs.TypeSettings, notif.Type)

	// Track which channels we sent to
	var channelsSent []string

	// Dispatch to In-App (SSE) - always immediate, no quiet hours restriction
	if d.isChannelEnabled(prefs.InAppEnabled, typeSettings.InApp) {
		d.dispatchInApp(notif)
		channelsSent = append(channelsSent, ChannelInApp)
	}

	// Dispatch to Push - respect quiet hours
	if d.isChannelEnabled(prefs.PushEnabled, typeSettings.Push) {
		if !d.isInQuietHours(prefs) {
			d.dispatchPush(ctx, notif)
			channelsSent = append(channelsSent, ChannelPush)
		} else {
			slog.Info("[Dispatcher] Skipping push for user  - in quiet hours", "value", notif.UserID)
		}
	}

	// Dispatch to Email - respect quiet hours
	if d.isChannelEnabled(prefs.EmailEnabled, typeSettings.Email) {
		if !d.isInQuietHours(prefs) {
			d.dispatchEmail(ctx, notif)
			channelsSent = append(channelsSent, ChannelEmail)
		} else {
			slog.Info("[Dispatcher] Skipping email for user  - in quiet hours", "value", notif.UserID)
		}
	}

	// Update channels_sent in database if we have an ID
	if notif.ID != [16]byte{} && len(channelsSent) > 0 {
		_ = queries.UpdateNotificationChannelsSent(ctx, sqlc.UpdateNotificationChannelsSentParams{
			ID:           uuidToPgtype(notif.ID),
			ChannelsSent: channelsSent,
		})
	}

	return nil
}

// DispatchDirect dispatches a notification directly without batching (for summary notifications)
func (d *NotificationDispatcher) DispatchDirect(ctx context.Context, notif *Notification) error {
	return d.Dispatch(ctx, notif)
}

// dispatchInApp sends notification via SSE to connected clients
func (d *NotificationDispatcher) dispatchInApp(notif *Notification) {
	d.sseBroadcaster.SendToUser(notif.UserID, SSEEvent{
		Type: "notification",
		Data: notif,
	})
	slog.Info("[Dispatcher] In-app notification sent to user", "value0", notif.UserID, "value1", notif.Title)
}

// dispatchPush sends notification via Web Push
func (d *NotificationDispatcher) dispatchPush(ctx context.Context, notif *Notification) {
	if d.webPushService == nil || !d.webPushService.IsEnabled() {
		slog.Info("[Dispatcher] Push notification skipped for user  - Web Push not configured", "value", notif.UserID)
		return
	}

	if err := d.webPushService.SendNotification(ctx, notif); err != nil {
		slog.Info("[Dispatcher] Push notification failed for user", "id", notif.UserID, "error", err)
	} else {
		slog.Info("[Dispatcher] Push notification sent to user", "value0", notif.UserID, "value1", notif.Title)
	}
}

// dispatchEmail sends notification via Email
func (d *NotificationDispatcher) dispatchEmail(ctx context.Context, notif *Notification) {
	if d.emailService == nil || !d.emailService.IsEnabled() {
		slog.Info("[Dispatcher] Email notification skipped for user  - Email not configured", "value", notif.UserID)
		return
	}

	// Get user email from Better Auth user table
	var userEmail, userName string
	err := d.pool.QueryRow(ctx, `
		SELECT email, COALESCE(name, email) FROM "user" WHERE id = $1
	`, notif.UserID).Scan(&userEmail, &userName)
	if err != nil {
		slog.Info("[Dispatcher] Email notification failed for user  - could not get user email", "id", notif.UserID, "error", err)
		return
	}

	// Get action URL from metadata if available
	actionURL := ""
	actionText := "View Details"
	if notif.Metadata != nil {
		if url, ok := notif.Metadata["action_url"].(string); ok {
			// Sanitize URL to prevent XSS attacks
			actionURL = sanitizeURL(url)
		}
		if text, ok := notif.Metadata["action_text"].(string); ok {
			actionText = text
		}
	}

	// Build email data
	emailData := NotificationEmailData{
		RecipientEmail: userEmail,
		RecipientName:  userName,
		Subject:        notif.Title,
		Title:          notif.Title,
		Body:           notif.Body,
		ActionURL:      actionURL,
		ActionText:     actionText,
	}

	if err := d.emailService.SendNotificationEmail(ctx, emailData); err != nil {
		slog.Info("[Dispatcher] Email notification failed for user", "id", notif.UserID, "error", err)
	} else {
		slog.Info("[Dispatcher] Email notification sent to user", "value0", notif.UserID, "value1", notif.Title)
	}
}

// isChannelEnabled checks if a channel is enabled considering global setting and per-type override
func (d *NotificationDispatcher) isChannelEnabled(globalEnabled *bool, typeOverride *bool) bool {
	// Per-type override takes precedence
	if typeOverride != nil {
		return *typeOverride
	}
	// Fall back to global setting
	if globalEnabled != nil {
		return *globalEnabled
	}
	// Default to enabled
	return true
}

// isInQuietHours checks if the current time falls within the user's quiet hours
func (d *NotificationDispatcher) isInQuietHours(prefs sqlc.NotificationPreference) bool {
	if prefs.QuietHoursEnabled == nil || !*prefs.QuietHoursEnabled {
		return false
	}

	// Get quiet hours bounds (stored as microseconds since midnight)
	if !prefs.QuietHoursStart.Valid || !prefs.QuietHoursEnd.Valid {
		return false
	}

	startMicros := prefs.QuietHoursStart.Microseconds
	endMicros := prefs.QuietHoursEnd.Microseconds

	// Get current time in user's timezone (or UTC if not set)
	loc := time.UTC
	if prefs.QuietHoursTimezone != nil && *prefs.QuietHoursTimezone != "" {
		parsedLoc, err := time.LoadLocation(*prefs.QuietHoursTimezone)
		if err != nil {
			slog.Info("[Dispatcher] Invalid timezone  for user, falling back to UTC", "id", *prefs.QuietHoursTimezone, "id", err)
			// Fall back to UTC on error
		} else {
			loc = parsedLoc
		}
	}

	now := time.Now().In(loc)
	currentMicros := int64(now.Hour()*3600+now.Minute()*60+now.Second()) * 1_000_000

	// Handle overnight quiet hours (e.g., 22:00 - 08:00)
	if startMicros > endMicros {
		// Quiet hours span midnight
		return currentMicros >= startMicros || currentMicros <= endMicros
	}

	// Normal case (e.g., 09:00 - 17:00)
	return currentMicros >= startMicros && currentMicros <= endMicros
}

// parseTypeSettings extracts channel settings for a specific notification type from the JSONB
func (d *NotificationDispatcher) parseTypeSettings(typeSettingsJSON []byte, notificationType string) TypeChannelSettings {
	if len(typeSettingsJSON) == 0 {
		return TypeChannelSettings{}
	}

	var allSettings map[string]TypeChannelSettings
	if err := json.Unmarshal(typeSettingsJSON, &allSettings); err != nil {
		slog.Info("[Dispatcher] Failed to parse type_settings", "error", err)
		return TypeChannelSettings{}
	}

	if settings, ok := allSettings[notificationType]; ok {
		return settings
	}

	return TypeChannelSettings{}
}

// getDefaultPreferences returns default preferences when none exist
func (d *NotificationDispatcher) getDefaultPreferences(userID string) sqlc.NotificationPreference {
	t := true
	return sqlc.NotificationPreference{
		UserID:       userID,
		EmailEnabled: &t,
		PushEnabled:  &t,
		InAppEnabled: &t,
	}
}

// sanitizeURL validates and sanitizes URLs to prevent XSS attacks
func sanitizeURL(rawURL string) string {
	// Allow empty URLs
	if rawURL == "" {
		return ""
	}

	// Only allow http/https schemes
	if len(rawURL) > 0 {
		// Check for dangerous schemes
		lower := rawURL
		if len(lower) > 11 {
			lower = rawURL[:11]
		}
		// Block javascript:, data:, vbscript:, file:, etc.
		dangerousPrefixes := []string{"javascript:", "data:", "vbscript:", "file:", "about:"}
		for _, prefix := range dangerousPrefixes {
			if len(lower) >= len(prefix) && lower[:len(prefix)] == prefix {
				slog.Info("[Dispatcher] Blocked dangerous URL scheme", "value", rawURL[:20])
				return ""
			}
		}
	}

	// Must start with http:// or https:// or be a relative path starting with /
	if len(rawURL) >= 7 && (rawURL[:7] == "http://" || rawURL[:8] == "https://") {
		return rawURL
	}
	if len(rawURL) > 0 && rawURL[0] == '/' {
		return rawURL
	}

	// If it doesn't match safe patterns, reject it
	maxLen := 20
	if len(rawURL) < maxLen {
		maxLen = len(rawURL)
	}
	slog.Info("[Dispatcher] Rejected invalid URL format", "id", rawURL[:maxLen])
	return ""
}

// uuidToPgtype converts uuid.UUID to pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
