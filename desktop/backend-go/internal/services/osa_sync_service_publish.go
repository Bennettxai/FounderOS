package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

func (s *OSASyncService) publishAppGenerated(ctx context.Context, event sqlc.SyncOutbox) error {
	// Parse payload
	var appData map[string]interface{}
	if err := json.Unmarshal(event.Payload, &appData); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	s.logger.InfoContext(ctx, "publishing app generated event",
		"event_id", event.ID,
		"app_id", appData["id"],
		"app_name", appData["name"])

	// Extract workspace_id and app_id from payload
	workspaceIDStr, ok := appData["workspace_id"].(string)
	if !ok {
		return fmt.Errorf("invalid workspace_id in payload")
	}
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		return fmt.Errorf("parse workspace_id: %w", err)
	}

	appIDStr, ok := appData["id"].(string)
	if !ok {
		return fmt.Errorf("invalid app id in payload")
	}
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		return fmt.Errorf("parse app_id: %w", err)
	}

	// Update workspace active_modules array
	// This happens in a transaction to ensure consistency
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Add app to workspace active_modules using raw SQL
	// (AddOSAWorkspaceModule query needs sqlc regeneration)
	_, err = tx.Exec(ctx, `
		UPDATE osa_workspaces
		SET active_modules = array_append(active_modules, $1),
		    updated_at = NOW()
		WHERE id = $2
	`, appID, workspaceID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update workspace modules",
			"workspace_id", workspaceID,
			"app_id", appID,
			"error", err)
		// Log error but don't fail - workspace update is secondary
	} else {
		s.logger.InfoContext(ctx, "Updated workspace active_modules",
			"workspace_id", workspaceID,
			"app_id", appID)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Publish to NATS if available
	if s.natsJS != nil {
		subject := "osa.app.generated"

		// Create rich event envelope with app metadata and files
		envelope := map[string]interface{}{
			"event_id":     event.ID,
			"event_type":   event.EventType,
			"aggregate_id": event.AggregateID,
			"timestamp":    time.Now().UTC(),
			"payload": map[string]interface{}{
				"app_id":       appData["id"],
				"workspace_id": appData["workspace_id"],
				"name":         appData["name"],
				"display_name": appData["display_name"],
				"description":  appData["description"],
				"status":       appData["status"],
				"files":        appData["files_created"],
				"metadata":     appData["metadata"],
			},
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

		s.logger.InfoContext(ctx, "Successfully published app generated to NATS",
			"subject", subject,
			"event_id", event.ID,
			"app_id", appID)
	} else {
		s.logger.DebugContext(ctx, "NATS not available, event stored in outbox only",
			"event_id", event.ID)
	}

	return nil
}

func (s *OSASyncService) calculateRetryTime(attempts int32) time.Time {
	// Exponential backoff: 2^attempts seconds, max 1 hour
	backoff := time.Duration(1<<uint(attempts)) * time.Second
	if backoff > time.Hour {
		backoff = time.Hour
	}
	return time.Now().Add(backoff)
}
