package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ProcessOutbox processes pending outbox events and publishes them to OSA
// This method implements the transactional outbox pattern for reliable event delivery
// It processes events in batches with automatic retry and dead letter queue support
func (s *OSASyncService) ProcessOutbox(ctx context.Context) error {
	s.logger.InfoContext(ctx, "processing outbox events")

	// Reset stuck processing events first
	if err := s.queries.ResetStuckProcessingEvents(ctx); err != nil {
		s.logger.WarnContext(ctx, "failed to reset stuck events", "error", err)
	}

	// Process pending events in batches
	batchSize := int32(50)
	processedCount := 0
	failedCount := 0

	for {
		// Fetch batch of pending events (with row-level locking)
		events, err := s.queries.GetPendingOutboxEvents(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("fetch pending events: %w", err)
		}

		if len(events) == 0 {
			break
		}

		// Process each event
		for _, event := range events {
			if err := s.processOutboxEvent(ctx, event); err != nil {
				s.logger.ErrorContext(ctx, "failed to process event",
					"event_id", event.ID, "error", err)
				failedCount++
			} else {
				processedCount++
			}
		}

		// Break if we processed less than a full batch
		if len(events) < int(batchSize) {
			break
		}
	}

	// Cleanup old completed events (older than 7 days)
	if err := s.queries.CleanupOldCompletedEvents(ctx); err != nil {
		s.logger.WarnContext(ctx, "failed to cleanup old events", "error", err)
	}

	s.logger.InfoContext(ctx, "outbox processing complete",
		"processed", processedCount, "failed", failedCount)

	return nil
}

// processOutboxEvent processes a single outbox event
func (s *OSASyncService) processOutboxEvent(ctx context.Context, event sqlc.SyncOutbox) error {
	eventID := event.ID

	// Mark as processing
	if err := s.queries.MarkOutboxEventProcessing(ctx, eventID); err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	// Process based on event type
	var processErr error
	switch event.EventType {
	case "user.sync":
		processErr = s.publishUserSync(ctx, event)
	case "workspace.sync":
		processErr = s.publishWorkspaceSync(ctx, event)
	case "app.generated":
		processErr = s.publishAppGenerated(ctx, event)
	default:
		s.logger.WarnContext(ctx, "unknown event type", "type", event.EventType)
	}

	// Handle result
	if processErr != nil {
		// Calculate next retry time with exponential backoff
		nextRetry := s.calculateRetryTime(event.Attempts)
		errorMsg := processErr.Error()

		err := s.queries.MarkOutboxEventFailed(ctx, sqlc.MarkOutboxEventFailedParams{
			ID:           eventID,
			LastError:    &errorMsg,
			ScheduledFor: pgtype.Timestamptz{Time: nextRetry, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("mark failed: %w", err)
		}

		// Move to DLQ if max attempts exceeded
		if event.Attempts+1 >= event.MaxAttempts {
			reason := fmt.Sprintf("Max attempts exceeded: %s", processErr.Error())
			_, err := s.queries.MoveEventToDLQ(ctx, sqlc.MoveEventToDLQParams{
				ID:            eventID,
				FailureReason: &reason,
			})
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to move event to DLQ",
					"event_id", eventID, "error", err)
			}
		}

		return processErr
	}

	// Mark as completed
	if err := s.queries.MarkOutboxEventCompleted(ctx, eventID); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	return nil
}

func (s *OSASyncService) getOrCreateSyncStatus(ctx context.Context, qtx *sqlc.Queries, entityType string, entityID uuid.UUID) (sqlc.OsaSyncStatus, error) {
	status, err := qtx.GetOSASyncStatus(ctx, sqlc.GetOSASyncStatusParams{
		EntityType: entityType,
		EntityID:   pgTypeUUID(entityID),
	})

	if err == pgx.ErrNoRows {
		// Create new sync status
		metadata := map[string]interface{}{}
		metadataJSON, _ := json.Marshal(metadata)

		pendingStatus := "pending"
		pushDirection := "push"

		status, err = qtx.CreateOSASyncStatus(ctx, sqlc.CreateOSASyncStatusParams{
			EntityType:    entityType,
			EntityID:      pgTypeUUID(entityID),
			OsaEntityID:   nil,
			OsaEntityType: nil,
			SyncStatus:    &pendingStatus,
			SyncDirection: &pushDirection,
			Metadata:      metadataJSON,
		})
	}

	return status, err
}

func (s *OSASyncService) handleSyncError(ctx context.Context, qtx *sqlc.Queries, entityType string, entityID uuid.UUID, syncErr error) error {
	s.logger.ErrorContext(ctx, "sync error",
		"entity_type", entityType, "entity_id", entityID, "error", syncErr)

	failedStatus := "failed"
	errorMsg := syncErr.Error()

	_, err := qtx.UpdateOSASyncStatus(ctx, sqlc.UpdateOSASyncStatusParams{
		EntityType: entityType,
		SyncStatus: &failedStatus,
		LastError:  &errorMsg,
		EntityID:   pgTypeUUID(entityID),
	})

	if err != nil {
		return fmt.Errorf("update sync status failed: %w (original error: %v)", err, syncErr)
	}

	return syncErr
}

func (s *OSASyncService) createOutboxEvent(ctx context.Context, qtx *sqlc.Queries, aggregateType string, aggregateID uuid.UUID, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Create vector clock for conflict resolution
	vectorClock := map[string]int{
		aggregateID.String(): int(time.Now().Unix()),
	}
	vectorClockJSON, _ := json.Marshal(vectorClock)

	_, err = qtx.CreateOutboxEvent(ctx, sqlc.CreateOutboxEventParams{
		AggregateType: aggregateType,
		AggregateID:   pgTypeUUID(aggregateID),
		EventType:     eventType,
		Payload:       payloadJSON,
		VectorClock:   vectorClockJSON,
		MaxAttempts:   5,
	})

	return err
}

func (s *OSASyncService) publishUserSync(ctx context.Context, event sqlc.SyncOutbox) error {
	// Parse payload
	var userData map[string]interface{}
	if err := json.Unmarshal(event.Payload, &userData); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	s.logger.InfoContext(ctx, "publishing user sync event",
		"event_id", event.ID,
		"user_id", userData["id"])

	// Publish to NATS if available
	if s.natsJS != nil {
		subject := "osa.user.sync"

		// Create event envelope
		envelope := map[string]interface{}{
			"event_id":     event.ID,
			"event_type":   event.EventType,
			"aggregate_id": event.AggregateID,
			"timestamp":    time.Now().UTC(),
			"payload":      userData,
		}

		envBytes, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("marshal envelope: %w", err)
		}

		// Publish with acknowledgment
		_, err = s.natsJS.Publish(subject, envBytes)
		if err != nil {
			s.logger.ErrorContext(ctx, "NATS publish failed, will retry via outbox",
				"subject", subject,
				"event_id", event.ID,
				"error", err)
			return fmt.Errorf("publish to NATS: %w", err)
		}

		s.logger.InfoContext(ctx, "Successfully published user sync to NATS",
			"subject", subject,
			"event_id", event.ID)
	} else {
		s.logger.DebugContext(ctx, "NATS not available, event stored in outbox only",
			"event_id", event.ID)
	}

	return nil
}

func (s *OSASyncService) publishWorkspaceSync(ctx context.Context, event sqlc.SyncOutbox) error {
	// Parse payload
	var workspaceData map[string]interface{}
	if err := json.Unmarshal(event.Payload, &workspaceData); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	s.logger.InfoContext(ctx, "publishing workspace sync event",
		"event_id", event.ID,
		"workspace_id", workspaceData["id"])

	// Publish to NATS if available
	if s.natsJS != nil {
		subject := "osa.workspace.sync"

		// Create event envelope with workspace metadata
		envelope := map[string]interface{}{
			"event_id":     event.ID,
			"event_type":   event.EventType,
			"aggregate_id": event.AggregateID,
			"timestamp":    time.Now().UTC(),
			"payload":      workspaceData,
		}

		envBytes, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("marshal envelope: %w", err)
		}

		// Publish with acknowledgment
		_, err = s.natsJS.Publish(subject, envBytes)
		if err != nil {
			s.logger.ErrorContext(ctx, "NATS publish failed, will retry via outbox",
				"subject", subject,
				"event_id", event.ID,
				"error", err)
			return fmt.Errorf("publish to NATS: %w", err)
		}

		s.logger.InfoContext(ctx, "Successfully published workspace sync to NATS",
			"subject", subject,
			"event_id", event.ID)
	} else {
		s.logger.DebugContext(ctx, "NATS not available, event stored in outbox only",
			"event_id", event.ID)
	}

	return nil
}
