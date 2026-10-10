package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

type BatchWorker struct {
	pool       *pgxpool.Pool
	dispatcher *services.NotificationDispatcher
	interval   time.Duration
	stopCh     chan struct{}
}

func NewBatchWorker(pool *pgxpool.Pool, dispatcher *services.NotificationDispatcher) *BatchWorker {
	return &BatchWorker{
		pool:       pool,
		dispatcher: dispatcher,
		interval:   10 * time.Second,
		stopCh:     make(chan struct{}),
	}
}

func (w *BatchWorker) Start(ctx context.Context) {
	slog.Info("[BatchWorker] Starting notification batch worker...")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("[BatchWorker] Shutting down (context cancelled)")
			return
		case <-w.stopCh:
			slog.Info("[BatchWorker] Shutting down (stop signal)")
			return
		case <-ticker.C:
			w.processDueBatches(ctx)
		}
	}
}

func (w *BatchWorker) Stop() {
	close(w.stopCh)
}

func (w *BatchWorker) processDueBatches(ctx context.Context) {
	queries := sqlc.New(w.pool)

	batches, err := queries.GetBatchesReadyToDispatch(ctx)
	if err != nil {
		slog.Info("[BatchWorker] Error fetching due batches", "error", err)
		return
	}

	if len(batches) == 0 {
		return
	}

	slog.Info("[BatchWorker] Processing  due batches", "value", len(batches))

	for _, batch := range batches {
		w.dispatchBatch(ctx, queries, batch)
	}
}

// dispatchBatch creates a summary notification and dispatches it
func (w *BatchWorker) dispatchBatch(ctx context.Context, queries *sqlc.Queries, batch sqlc.NotificationBatch) {
	batchID := uuid.UUID(batch.ID.Bytes)

	count := 0
	if batch.PendingCount != nil {
		count = int(*batch.PendingCount)
	}

	if count == 0 {
		// Empty batch, just mark as dispatched
		_ = queries.MarkBatchDispatched(ctx, batch.ID)
		return
	}

	slog.Info("[BatchWorker] Dispatching batch :,", "type", batchID, "type", batch.Type, "count", count)

	title := services.GetBatchTitle(batch.Type, count)

	// Build metadata for the summary
	metadata := map[string]interface{}{
		"batch_id":      batchID.String(),
		"batch_count":   count,
		"original_type": batch.Type,
	}

	if batch.EntityType != nil {
		metadata["entity_type"] = *batch.EntityType
	}
	if batch.EntityID.Valid {
		metadata["entity_id"] = uuid.UUID(batch.EntityID.Bytes).String()
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		slog.Info("[BatchWorker] Error marshaling notification metadata for batch", "value", batchID, "error", err)
		return
	}

	// Get priority from notification type config
	priority := services.PriorityNormal
	if cfg, ok := services.GetTypeConfig(batch.Type); ok {
		priority = cfg.Priority
	}

	// Create summary notification in database
	var entityID pgtype.UUID
	if batch.EntityID.Valid {
		entityID = batch.EntityID
	}

	batchCount := int32(count)
	summaryNotif, err := queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
		UserID:     batch.UserID,
		Type:       batch.Type,
		Title:      title,
		Body:       utils.StringPtr(generateBatchBody(batch.Type, count)),
		EntityType: batch.EntityType,
		EntityID:   entityID,
		BatchID:    pgtype.UUID{Bytes: batch.ID.Bytes, Valid: true},
		BatchCount: &batchCount,
		Priority:   &priority,
		Metadata:   metadataJSON,
	})
	if err != nil {
		slog.Info("[BatchWorker] Error creating summary notification", "error", err)
		// Still mark batch as dispatched to prevent infinite retries
		_ = queries.MarkBatchDispatched(ctx, batch.ID)
		return
	}

	// Map to service notification type
	notif := mapSqlcToNotification(summaryNotif)

	// Dispatch through the dispatcher (respects preferences, quiet hours, etc.)
	if err := w.dispatcher.Dispatch(ctx, notif); err != nil {
		slog.Info("[BatchWorker] Error dispatching batch", "value", batchID, "error", err)
	}

	if err := queries.MarkBatchDispatched(ctx, batch.ID); err != nil {
		slog.Info("[BatchWorker] Error marking batch  as dispatched", "value", batchID, "error", err)
	}

	slog.Info("[BatchWorker] Successfully dispatched batch", "value", batchID)
}

// creates a body message for batch summaries
func generateBatchBody(notifType string, count int) string {
	switch notifType {
	case services.NotifTaskAssigned:
		return "Click to view your newly assigned tasks"
	case services.NotifTaskComment:
		return "Click to view the new comments"
	case services.NotifMentionTask, services.NotifMentionProject, services.NotifMentionComment:
		return "Click to view where you were mentioned"
	case services.NotifProjectAdded:
		return "Click to view your new projects"
	default:
		return "Click to view details"
	}
}

// mapSqlcToNotification converts SQLC type to service Notification
func mapSqlcToNotification(n sqlc.Notification) *services.Notification {
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

	return &services.Notification{
		ID:           uuid.UUID(n.ID.Bytes),
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
