package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/utils"
)

type NotificationService struct {
	pool         *pgxpool.Pool
	sse          *SSEBroadcaster
	dispatcher   *NotificationDispatcher
	batchManager *BatchManager
}

func NewNotificationService(pool *pgxpool.Pool, sse *SSEBroadcaster) *NotificationService {
	dispatcher := NewNotificationDispatcher(pool, sse)
	batchManager := NewBatchManager(pool)

	return &NotificationService{
		pool:         pool,
		sse:          sse,
		dispatcher:   dispatcher,
		batchManager: batchManager,
	}
}

func (s *NotificationService) SSE() *SSEBroadcaster {
	return s.sse
}

func (s *NotificationService) Dispatcher() *NotificationDispatcher {
	return s.dispatcher
}

type CreateInput struct {
	UserID      string
	WorkspaceID *uuid.UUID
	Type        string
	Title       string
	Body        string
	EntityType  string
	EntityID    *uuid.UUID
	SenderID    string
	SenderName  string
	Priority    string
	Metadata    map[string]interface{}
}

type Notification struct {
	ID           uuid.UUID              `json:"id"`
	UserID       string                 `json:"user_id"`
	WorkspaceID  *uuid.UUID             `json:"workspace_id,omitempty"`
	Type         string                 `json:"type"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body,omitempty"`
	EntityType   string                 `json:"entity_type,omitempty"`
	EntityID     *uuid.UUID             `json:"entity_id,omitempty"`
	SenderID     string                 `json:"sender_id,omitempty"`
	SenderName   string                 `json:"sender_name,omitempty"`
	IsRead       bool                   `json:"is_read"`
	ReadAt       *time.Time             `json:"read_at,omitempty"`
	BatchCount   int                    `json:"batch_count,omitempty"`
	Priority     string                 `json:"priority"`
	ChannelsSent []string               `json:"channels_sent"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

func (s *NotificationService) Create(ctx context.Context, input CreateInput) (*Notification, error) {
	cfg, ok := GetTypeConfig(input.Type)
	if !ok {
		return nil, fmt.Errorf("unknown notification type: %s", input.Type)
	}

	if input.Priority == "" {
		input.Priority = cfg.Priority
	}

	// Check if this notification type is batchable
	if cfg.Batch != nil {
		batchID, err := s.batchManager.Queue(ctx, input)
		if err != nil {
			slog.Info("[NotificationService] Batching failed, falling back to immediate", "error", err)
		} else if batchID != nil {
			// Successfully queued for batching - don't create individual notification
			slog.Info("[NotificationService] Notification queued in batch", "value", batchID)
			// Return a placeholder notification indicating it was batched
			return &Notification{
				UserID:   input.UserID,
				Type:     input.Type,
				Title:    input.Title,
				Priority: input.Priority,
				Metadata: map[string]interface{}{"batched": true, "batch_id": batchID.String()},
			}, nil
		}
	}

	// Create immediate notification (not batched)
	return s.createImmediate(ctx, input)
}

// createImmediate creates and dispatches a notification immediately
func (s *NotificationService) createImmediate(ctx context.Context, input CreateInput) (*Notification, error) {
	queries := sqlc.New(s.pool)

	var workspaceID pgtype.UUID
	if input.WorkspaceID != nil {
		workspaceID = pgtype.UUID{Bytes: *input.WorkspaceID, Valid: true}
	}

	var entityID pgtype.UUID
	if input.EntityID != nil {
		entityID = pgtype.UUID{Bytes: *input.EntityID, Valid: true}
	}

	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		slog.Info("[NotificationService] Failed to marshal metadata", "error", err)
		metadata = nil
	}
	batchCount := int32(1)

	result, err := queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
		UserID:      input.UserID,
		WorkspaceID: workspaceID,
		Type:        input.Type,
		Title:       input.Title,
		Body:        utils.StringPtr(input.Body),
		EntityType:  utils.StringPtr(input.EntityType),
		EntityID:    entityID,
		SenderID:    utils.StringPtr(input.SenderID),
		SenderName:  utils.StringPtr(input.SenderName),
		Priority:    &input.Priority,
		Metadata:    metadata,
		BatchCount:  &batchCount,
	})
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	notif := mapNotification(result)

	// Dispatch through the dispatcher (handles preferences, channels, quiet hours)
	if err := s.dispatcher.Dispatch(ctx, notif); err != nil {
		slog.Info("[NotificationService] Dispatch failed for notification", "id", notif.ID, "error", err)
		// Mark notification as having dispatch issues in metadata
		if notif.Metadata == nil {
			notif.Metadata = make(map[string]interface{})
		}
		notif.Metadata["dispatch_error"] = err.Error()
		notif.Metadata["dispatch_failed_at"] = time.Now().Format(time.RFC3339)

		// Still return the notification - it's persisted in DB even if dispatch fails
		// This allows manual retry or background job to retry later
		slog.Info("[NotificationService] Notification  saved to DB but dispatch failed, can be retried", "value", notif.ID)
	}

	return notif, nil
}

func (s *NotificationService) GetForUser(ctx context.Context, userID string, limit, offset int32) ([]Notification, error) {
	queries := sqlc.New(s.pool)
	results, err := queries.GetNotificationsForUser(ctx, sqlc.GetNotificationsForUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	notifications := make([]Notification, len(results))
	for i, r := range results {
		notifications[i] = *mapNotification(r)
	}
	return notifications, nil
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	queries := sqlc.New(s.pool)
	return queries.GetUnreadCount(ctx, userID)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, userID string, id uuid.UUID) error {
	queries := sqlc.New(s.pool)
	_, err := queries.MarkNotificationAsRead(ctx, sqlc.MarkNotificationAsReadParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: userID,
	})
	if err != nil {
		return err
	}

	s.sse.SendToUser(userID, SSEEvent{
		Type: "read_sync",
		Data: map[string]interface{}{
			"read_ids": []uuid.UUID{id},
			"read_at":  time.Now(),
		},
	})
	return nil
}

func (s *NotificationService) MarkMultipleAsRead(ctx context.Context, userID string, ids []uuid.UUID) error {
	queries := sqlc.New(s.pool)

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	err := queries.MarkMultipleAsRead(ctx, sqlc.MarkMultipleAsReadParams{
		Column1: pgIDs,
		UserID:  userID,
	})
	if err != nil {
		return err
	}

	s.sse.SendToUser(userID, SSEEvent{
		Type: "read_sync",
		Data: map[string]interface{}{
			"read_ids": ids,
			"read_at":  time.Now(),
		},
	})
	return nil
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	queries := sqlc.New(s.pool)
	err := queries.MarkAllAsRead(ctx, userID)
	if err != nil {
		return err
	}

	s.sse.SendToUser(userID, SSEEvent{
		Type: "read_sync",
		Data: map[string]interface{}{
			"read_all": true,
			"read_at":  time.Now(),
		},
	})
	return nil
}

func (s *NotificationService) GetPreferences(ctx context.Context, userID string) (*sqlc.NotificationPreference, error) {
	queries := sqlc.New(s.pool)
	prefs, err := queries.GetNotificationPreferencesByUser(ctx, userID)
	if err != nil {
		t := true
		return &sqlc.NotificationPreference{
			UserID:       userID,
			EmailEnabled: &t,
			PushEnabled:  &t,
			InAppEnabled: &t,
		}, nil
	}
	return &prefs, nil
}

// UpdatePreferencesInput defines the input for updating notification preferences
type UpdatePreferencesInput struct {
	EmailEnabled        *bool                  `json:"email_enabled"`
	PushEnabled         *bool                  `json:"push_enabled"`
	InAppEnabled        *bool                  `json:"in_app_enabled"`
	TypeSettings        map[string]interface{} `json:"type_settings"`
	QuietHoursEnabled   *bool                  `json:"quiet_hours_enabled"`
	QuietHoursStart     *string                `json:"quiet_hours_start"` // "HH:MM" format
	QuietHoursEnd       *string                `json:"quiet_hours_end"`   // "HH:MM" format
	QuietHoursTimezone  *string                `json:"quiet_hours_timezone"`
	EmailDigestEnabled  *bool                  `json:"email_digest_enabled"`
	EmailDigestTime     *string                `json:"email_digest_time"` // "HH:MM" format
	EmailDigestTimezone *string                `json:"email_digest_timezone"`
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, userID string, input UpdatePreferencesInput) (*sqlc.NotificationPreference, error) {
	queries := sqlc.New(s.pool)

	// Convert type_settings to JSON bytes
	var typeSettings []byte
	if input.TypeSettings != nil {
		typeSettings, _ = json.Marshal(input.TypeSettings)
	}

	// Parse time strings to pgtype.Time
	var quietHoursStart, quietHoursEnd, emailDigestTime pgtype.Time
	if input.QuietHoursStart != nil {
		if t, err := parseTimeString(*input.QuietHoursStart); err == nil {
			quietHoursStart = t
		}
	}
	if input.QuietHoursEnd != nil {
		if t, err := parseTimeString(*input.QuietHoursEnd); err == nil {
			quietHoursEnd = t
		}
	}
	if input.EmailDigestTime != nil {
		if t, err := parseTimeString(*input.EmailDigestTime); err == nil {
			emailDigestTime = t
		}
	}

	// Set defaults for required fields
	emailEnabled := true
	if input.EmailEnabled != nil {
		emailEnabled = *input.EmailEnabled
	}
	pushEnabled := true
	if input.PushEnabled != nil {
		pushEnabled = *input.PushEnabled
	}
	inAppEnabled := true
	if input.InAppEnabled != nil {
		inAppEnabled = *input.InAppEnabled
	}

	prefs, err := queries.UpsertNotificationPreferences(ctx, sqlc.UpsertNotificationPreferencesParams{
		UserID:              userID,
		EmailEnabled:        &emailEnabled,
		PushEnabled:         &pushEnabled,
		InAppEnabled:        &inAppEnabled,
		TypeSettings:        typeSettings,
		QuietHoursEnabled:   input.QuietHoursEnabled,
		QuietHoursStart:     quietHoursStart,
		QuietHoursEnd:       quietHoursEnd,
		QuietHoursTimezone:  input.QuietHoursTimezone,
		EmailDigestEnabled:  input.EmailDigestEnabled,
		EmailDigestTime:     emailDigestTime,
		EmailDigestTimezone: input.EmailDigestTimezone,
	})
	if err != nil {
		return nil, fmt.Errorf("update preferences: %w", err)
	}

	return &prefs, nil
}

// DeleteNotification deletes a notification for a user
func (s *NotificationService) DeleteNotification(ctx context.Context, userID string, id uuid.UUID) error {
	queries := sqlc.New(s.pool)
	return queries.DeleteNotification(ctx, sqlc.DeleteNotificationParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: userID,
	})
}

// parseTimeString parses "HH:MM" format to pgtype.Time
func parseTimeString(s string) (pgtype.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}, err
	}
	// Convert to microseconds since midnight
	microseconds := int64(t.Hour()*3600+t.Minute()*60) * 1_000_000
	return pgtype.Time{Microseconds: microseconds, Valid: true}, nil
}

func mapNotification(n sqlc.Notification) *Notification {
	var metadata map[string]interface{}
	if len(n.Metadata) > 0 {
		json.Unmarshal(n.Metadata, &metadata)
	}

	var workspaceID *uuid.UUID
	if n.WorkspaceID.Valid {
		id := uuid.UUID(n.WorkspaceID.Bytes)
		workspaceID = &id
	}

	var entityID *uuid.UUID
	if n.EntityID.Valid {
		id := uuid.UUID(n.EntityID.Bytes)
		entityID = &id
	}

	var readAt *time.Time
	if n.ReadAt.Valid {
		readAt = &n.ReadAt.Time
	}

	isRead := false
	if n.IsRead != nil {
		isRead = *n.IsRead
	}

	batchCount := 1
	if n.BatchCount != nil {
		batchCount = int(*n.BatchCount)
	}

	priority := "normal"
	if n.Priority != nil {
		priority = *n.Priority
	}

	body := ""
	if n.Body != nil {
		body = *n.Body
	}

	entityType := ""
	if n.EntityType != nil {
		entityType = *n.EntityType
	}

	senderID := ""
	if n.SenderID != nil {
		senderID = *n.SenderID
	}

	senderName := ""
	if n.SenderName != nil {
		senderName = *n.SenderName
	}

	return &Notification{
		ID:           n.ID.Bytes,
		UserID:       n.UserID,
		WorkspaceID:  workspaceID,
		Type:         n.Type,
		Title:        n.Title,
		Body:         body,
		EntityType:   entityType,
		EntityID:     entityID,
		SenderID:     senderID,
		SenderName:   senderName,
		IsRead:       isRead,
		ReadAt:       readAt,
		BatchCount:   batchCount,
		Priority:     priority,
		ChannelsSent: n.ChannelsSent,
		Metadata:     metadata,
		CreatedAt:    n.CreatedAt.Time,
	}
}
