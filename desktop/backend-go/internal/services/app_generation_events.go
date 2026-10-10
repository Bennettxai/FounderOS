package services

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/appgen"
)

// handleProgressEvent converts agent progress to BuildEvent
func (o *AppGenerationOrchestrator) handleProgressEvent(queueItemID string, event appgen.ProgressEvent) {
	appID, _ := uuid.Parse(queueItemID)

	buildEvent := BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "agent_progress",
		Phase:           string(event.AgentType),
		ProgressPercent: event.Progress,
		StatusMessage:   event.Message,
		Data: map[string]interface{}{
			"task_id":    event.TaskID,
			"agent_type": string(event.AgentType),
			"status":     event.Status,
		},
		Timestamp: event.Timestamp,
	}

	o.eventBus.Publish(buildEvent)
}

// emitEvent publishes a build event
func (o *AppGenerationOrchestrator) emitEvent(queueItemID string, eventType string, data map[string]interface{}) {
	if o.eventBus == nil {
		return
	}

	appID, _ := uuid.Parse(queueItemID)

	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       eventType,
		ProgressPercent: getProgressFromData(data),
		StatusMessage:   getMessageFromData(data),
		Data:            data,
		Timestamp:       time.Now(),
	}

	o.eventBus.Publish(event)
}

func getProgressFromData(data map[string]interface{}) int {
	if p, ok := data["progress"].(float64); ok {
		return int(p * 100)
	}
	return 0
}

func getMessageFromData(data map[string]interface{}) string {
	if msg, ok := data["message"].(string); ok {
		return msg
	}
	return ""
}

// withRetry implements exponential backoff with jitter
func (o *AppGenerationOrchestrator) withRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		// Acquire API semaphore
		if err := o.apiSem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("acquire semaphore: %w", err)
		}

		err := fn()
		o.apiSem.Release(1)

		if err == nil {
			return nil
		}

		lastErr = err

		// Check if retriable
		if !o.isRetriable(err) {
			return fmt.Errorf("non-retriable error: %w", err)
		}

		if attempt < o.maxRetries {
			// Calculate backoff
			backoff := o.calculateBackoff(attempt)

			o.logger.WarnContext(ctx, "retrying after error",
				slog.Any("error", err),
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
			)

			// Wait with context cancellation support
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetriable checks if error should be retried
func (o *AppGenerationOrchestrator) isRetriable(err error) bool {
	if err == nil {
		return false
	}

	// Add error type checking here
	// For now, retry most errors except context cancellation
	return err != context.Canceled
}

// calculateBackoff returns exponential backoff with jitter
func (o *AppGenerationOrchestrator) calculateBackoff(attempt int) time.Duration {
	// Exponential: 1s, 2s, 4s, 8s, 16s
	base := time.Duration(1<<uint(attempt)) * time.Second

	// Cap at 60 seconds
	if base > 60*time.Second {
		base = 60 * time.Second
	}

	// Add jitter (±10%)
	jitter := time.Duration(float64(base) * 0.1 * (2.0*rand.Float64() - 1.0))

	return base + jitter
}

// Metrics tracking
func (o *AppGenerationOrchestrator) recordSuccess() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.totalRuns++
	o.successRuns++
}

func (o *AppGenerationOrchestrator) recordFailure() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.totalRuns++
	o.failedRuns++
}

// GetMetrics returns current metrics
func (o *AppGenerationOrchestrator) GetMetrics() map[string]int64 {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return map[string]int64{
		"total_runs":   o.totalRuns,
		"success_runs": o.successRuns,
		"failed_runs":  o.failedRuns,
	}
}

// Shutdown gracefully shuts down the orchestrator
func (o *AppGenerationOrchestrator) Shutdown() error {
	o.logger.Info("app generation orchestrator shutting down...")
	return o.orchestrator.Shutdown()
}

// GetCircuitBreakerMetrics returns circuit breaker state
func (o *AppGenerationOrchestrator) GetCircuitBreakerMetrics() map[string]interface{} {
	return o.orchestrator.GetCircuitBreakerMetrics()
}
