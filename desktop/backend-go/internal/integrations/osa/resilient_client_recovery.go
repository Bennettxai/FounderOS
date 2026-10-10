package osa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// autoRecoveryLoop attempts to process queued requests when circuit recovers
func (r *ResilientClient) autoRecoveryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if r.circuitBreaker.State() != StateClosed {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := r.HealthCheck(ctx)
		cancel()

		if err != nil {
			continue
		}

		for r.requestQueue.Size() > 0 {
			req, ok := r.requestQueue.Dequeue()
			if !ok {
				break
			}

			slog.Info("processing queued request",
				"request_id", req.ID,
				"operation", req.Operation,
				"queued_at", req.QueuedAt)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := r.processQueuedRequest(ctx, req)
			cancel()

			if err != nil {
				slog.Error("failed to process queued request",
					"request_id", req.ID,
					"operation", req.Operation,
					"error", err,
					"retry_count", req.RetryCount)

				if req.RetryCount < 3 {
					req.RetryCount++
					payload := json.RawMessage(req.Payload)
					_, requeueErr := r.requestQueue.Enqueue(req.Operation, payload, req.UserID)
					if requeueErr != nil {
						slog.Error("failed to requeue request", "error", requeueErr)
					} else {
						slog.Info("request requeued for retry",
							"request_id", req.ID,
							"retry_count", req.RetryCount)
					}
				} else {
					slog.Error("request exceeded max retries, dropping",
						"request_id", req.ID,
						"operation", req.Operation)
				}
			} else {
				slog.Info("successfully processed queued request",
					"request_id", req.ID,
					"operation", req.Operation)
			}
		}
	}
}

// processQueuedRequest processes a single queued request based on its operation type
func (r *ResilientClient) processQueuedRequest(ctx context.Context, req *QueuedRequest) error {
	switch req.Operation {
	case "generate_app":
		var appReq AppGenerationRequest
		if err := json.Unmarshal(req.Payload, &appReq); err != nil {
			return fmt.Errorf("failed to unmarshal generate_app request: %w", err)
		}
		_, err := r.GenerateApp(ctx, &appReq)
		return err

	case "orchestrate":
		var orchReq OrchestrateRequest
		if err := json.Unmarshal(req.Payload, &orchReq); err != nil {
			return fmt.Errorf("failed to unmarshal orchestrate request: %w", err)
		}
		_, err := r.Orchestrate(ctx, &orchReq)
		return err

	default:
		return fmt.Errorf("unknown operation type: %s", req.Operation)
	}
}
