package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/utils"
)

type BatchManager struct {
	pool *pgxpool.Pool
}

func NewBatchManager(pool *pgxpool.Pool) *BatchManager {
	return &BatchManager{pool: pool}
}

// Queue adds a notification to a batch or creates a new batch
// Returns the batch ID if batched, or empty UUID if immediate dispatch is needed
func (m *BatchManager) Queue(ctx context.Context, input CreateInput) (*uuid.UUID, error) {
	queries := sqlc.New(m.pool)

	// Get batch config for this notification type
	cfg, ok := GetTypeConfig(input.Type)
	if !ok || cfg.Batch == nil {
		return nil, nil
	}

	batchConfig := cfg.Batch

	// Generate batch key based on grouping strategy
	batchKey := m.generateBatchKey(input, batchConfig.GroupBy)

	// Look for existing pending batch
	existingBatch, err := queries.GetPendingBatch(ctx, sqlc.GetPendingBatchParams{
		UserID:   input.UserID,
		BatchKey: batchKey,
	})

	if err == nil {
		pendingCount := int32(0)
		if existingBatch.PendingCount != nil {
			pendingCount = *existingBatch.PendingCount
		}

		if pendingCount < int32(batchConfig.Max) {
			// Create a placeholder UUID for the pending notification
			pendingID := uuid.New()
			_, err = queries.AddToBatch(ctx, sqlc.AddToBatchParams{
				ID:          pgtype.UUID{Bytes: existingBatch.ID.Bytes, Valid: true},
				ArrayAppend: pgtype.UUID{Bytes: pendingID, Valid: true},
			})
			if err != nil {
				slog.Info("[BatchManager] Failed to add to batch", "error", err)
				return nil, nil
			}

			batchID := uuid.UUID(existingBatch.ID.Bytes)
			slog.Info("[BatchManager] Added to existing batch", "batch_id", batchID, "count", pendingCount+1)
			return &batchID, nil
		}

		// Batch is full, dispatch it now and create new one
		slog.Info("[BatchManager] Batch  is full, will be dispatched", "value", existingBatch.ID.Bytes)
	}

	// Create new batch
	dispatchAt := time.Now().Add(batchConfig.Window)

	var entityID pgtype.UUID
	if input.EntityID != nil {
		entityID = pgtype.UUID{Bytes: *input.EntityID, Valid: true}
	}

	pendingID := uuid.New()
	newBatch, err := queries.CreateBatch(ctx, sqlc.CreateBatchParams{
		UserID:     input.UserID,
		BatchKey:   batchKey,
		Type:       input.Type,
		EntityType: utils.StringPtr(input.EntityType),
		EntityID:   entityID,
		PendingIds: []pgtype.UUID{{Bytes: pendingID, Valid: true}},
		DispatchAt: pgtype.Timestamptz{Time: dispatchAt, Valid: true},
	})
	if err != nil {
		slog.Info("[BatchManager] Failed to create batch", "error", err)
		return nil, nil
	}

	batchID := uuid.UUID(newBatch.ID.Bytes)
	slog.Info("[BatchManager] Created new batch  for type , dispatching at", "value0", batchID, "value1", input.Type, "value2", dispatchAt.Format(time.RFC3339))
	return &batchID, nil
}

func (m *BatchManager) generateBatchKey(input CreateInput, groupBy string) string {
	base := fmt.Sprintf("%s:%s", input.UserID, input.Type)

	switch groupBy {
	case "entity_id":
		if input.EntityID != nil {
			return fmt.Sprintf("%s:%s", base, input.EntityID.String())
		}
		return base
	case "sender_id":
		if input.SenderID != "" {
			return fmt.Sprintf("%s:%s", base, input.SenderID)
		}
		return base
	case "type":
		return base
	default:
		return base
	}
}

// GetPendingBatchCount returns the count of pending notifications in a specific batch
func (m *BatchManager) GetPendingBatchCount(ctx context.Context, batchID uuid.UUID) (int, error) {
	// Query the batch directly by ID instead of using empty params
	var pendingCount int32
	err := m.pool.QueryRow(ctx, `
		SELECT COALESCE(array_length(pending_ids, 1), 0) as pending_count
		FROM notification_batches
		WHERE id = $1 AND status = 'pending'
	`, batchID).Scan(&pendingCount)

	if err != nil {
		return 0, fmt.Errorf("get pending batch count: %w", err)
	}

	return int(pendingCount), nil
}

// Batch title templates
var batchTitleTemplates = map[string]string{
	NotifTaskAssigned:      "You were assigned %d tasks",
	NotifTaskUnassigned:    "You were unassigned from %d tasks",
	NotifTaskCompleted:     "%d tasks were completed",
	NotifTaskComment:       "%d new comments on your tasks",
	NotifTaskStatusChanged: "%d task status updates",
	NotifProjectAdded:      "You were added to %d projects",
	NotifTeamMemberJoined:  "%d new team members joined",
	NotifClientDealUpdate:  "%d deal updates",
	NotifMentionTask:       "You were mentioned in %d tasks",
	NotifMentionProject:    "You were mentioned in %d projects",
	NotifMentionComment:    "You were mentioned in %d comments",
	NotifMentionDailyLog:   "You were mentioned in %d daily logs",
	NotifDailyLogMention:   "You were mentioned in %d daily logs",
}

func GetBatchTitle(notifType string, count int) string {
	if template, ok := batchTitleTemplates[notifType]; ok {
		return fmt.Sprintf(template, count)
	}
	return fmt.Sprintf("%d new notifications", count)
}
