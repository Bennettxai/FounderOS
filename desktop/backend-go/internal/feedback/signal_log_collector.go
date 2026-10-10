package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/signal"
)

// SignalLogActionCompletion implements signal.ActionCompletionRate by
// querying the signal_log table. Measures the ratio of signals with
// response_length > 0 (completed) vs total signals in the window.
//
// This is the only metric currently computable from signal_log data.
// Other metrics (re-encoding, bounce, genre recognition, feedback closure)
// require richer event data that doesn't exist yet. The homeostatic loop
// handles nil collectors gracefully, so only this metric runs.
type SignalLogActionCompletion struct {
	pool *pgxpool.Pool
}

// NewSignalLogActionCompletion creates a collector backed by signal_log.
func NewSignalLogActionCompletion(pool *pgxpool.Pool) *SignalLogActionCompletion {
	return &SignalLogActionCompletion{pool: pool}
}

// Name implements signal.MetricCollector.
func (c *SignalLogActionCompletion) Name() string {
	return "action_completion_rate"
}

// Collect implements signal.MetricCollector.
func (c *SignalLogActionCompletion) Collect(ctx context.Context) (signal.MetricValue, error) {
	rate, err := c.Rate(ctx, 5*time.Minute)
	if err != nil {
		return signal.MetricValue{}, err
	}
	return signal.MetricValue{
		Name:      c.Name(),
		Value:     rate,
		Timestamp: time.Now().UTC(),
	}, nil
}

// Register implements signal.MetricCollector. No-op for SQL-backed metrics.
func (c *SignalLogActionCompletion) Register(_ context.Context, _ signal.CollectionPoint) error {
	return nil
}

// Record implements signal.ActionCompletionRate. No-op because signal_log
// is already written by chat_streaming.go's logSignal() function.
func (c *SignalLogActionCompletion) Record(_ context.Context, _ *signal.Signal, _ bool) error {
	return nil
}

// Rate implements signal.ActionCompletionRate. Returns the ratio of
// signals with response_length > 0 to total signals in the window.
func (c *SignalLogActionCompletion) Rate(ctx context.Context, window time.Duration) (float64, error) {
	cutoff := time.Now().UTC().Add(-window)

	var completed, total int64
	err := c.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN response_length > 0 THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM signal_log
		WHERE created_at > $1
	`, cutoff).Scan(&completed, &total)
	if err != nil {
		return 0, fmt.Errorf("signal_log action_completion_rate: %w", err)
	}
	if total == 0 {
		return 1.0, nil // no signals = no failures
	}
	return float64(completed) / float64(total), nil
}

// Compile-time interface check.
var _ signal.ActionCompletionRate = (*SignalLogActionCompletion)(nil)
