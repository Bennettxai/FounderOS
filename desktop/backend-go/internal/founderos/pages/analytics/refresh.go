package analytics

// The analytics heartbeat's writers (FounderOS v1 app/api/analytics/refresh):
// lib/operating-metrics.ts recordOperatingSnapshots and the decision half of
// lib/email-list.ts syncBeehiivEmail, plus the Postgres rows they write.
// Every write here is inbound, into the bridge's own database.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
)

// MetricWriter is the snapshot writer's minimal surface (*PgStore has it).
type MetricWriter interface {
	RecordMetric(ctx context.Context, metricID string, value float64, capturedAt time.Time) error
}

// RecordOperatingSnapshots persists one sweep of metric values and returns
// how many were recorded. Honest-pending nulls are skipped: a missing key
// must never write a fake zero into history. A failed write is not counted.
func RecordOperatingSnapshots(ctx context.Context, w MetricWriter, inputs []MetricInput, at time.Time) int {
	n := 0
	for _, m := range inputs {
		if m.Value == nil {
			continue
		}
		if err := w.RecordMetric(ctx, m.ID, *m.Value, at); err != nil {
			slog.Warn("analytics refresh: metric snapshot not recorded", "metric", m.ID, "err", err)
			continue
		}
		n++
	}
	return n
}

// EmailListSnapshot is one founderos_email_list_snapshots row.
type EmailListSnapshot struct {
	CapturedOn    string // YYYY-MM-DD
	Subscribers   int
	Source        string
	PublicationID string
	Metric        string
	Quality       string
}

// EmailListOutcome is syncBeehiivEmail's decision: a fresh reading becomes
// today's snapshot; no reading, or the connector's last good number served
// through an outage (Fresh=false, FOS-659), becomes a miss with its reason.
func EmailListOutcome(r *beehiiv.Reading, today string) (*EmailListSnapshot, string) {
	if r == nil {
		return nil, "no reading from beehiiv"
	}
	if !r.Fresh {
		return nil, "stale reading (cached through an API failure) — not recorded as an observation"
	}
	return &EmailListSnapshot{
		CapturedOn: today, Subscribers: r.Subscribers, Source: "beehiiv",
		PublicationID: r.PublicationID, Metric: string(r.Metric), Quality: "ok",
	}, ""
}

// RecordMetric upserts one metric snapshot (founderos); the same metric at
// the same instant overwrites, as the SQLite INSERT OR REPLACE did.
func (s *PgStore) RecordMetric(ctx context.Context, metricID string, value float64, capturedAt time.Time) error {
	ws, err := s.ws("founderos")
	if err != nil {
		return err
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO founderos_metric_snapshots (metric_id, captured_at, workspace_id, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (metric_id, captured_at) DO UPDATE SET value = EXCLUDED.value, workspace_id = EXCLUDED.workspace_id`,
		metricID, capturedAt.UTC(), ws, value); err != nil {
		return fmt.Errorf("analytics metric snapshot: %w", err)
	}
	return nil
}

// RecordEmailListSnapshot upserts the day's email-list reading
// (personal); a same-day reading overwrites, so ticks are idempotent.
func (s *PgStore) RecordEmailListSnapshot(ctx context.Context, e EmailListSnapshot) error {
	ws, err := s.ws("personal")
	if err != nil {
		return err
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO founderos_email_list_snapshots (captured_on, workspace_id, subscribers, source, publication_id, metric, quality)
		VALUES ($1::date, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (captured_on) DO UPDATE SET subscribers = EXCLUDED.subscribers, source = EXCLUDED.source,
			publication_id = EXCLUDED.publication_id, metric = EXCLUDED.metric, quality = EXCLUDED.quality, workspace_id = EXCLUDED.workspace_id`,
		e.CapturedOn, ws, e.Subscribers, e.Source, e.PublicationID, e.Metric, e.Quality); err != nil {
		return fmt.Errorf("analytics email-list snapshot: %w", err)
	}
	return nil
}

// RecordEmailListMiss records a day with no fresh reading, so a hole in the
// series can be told apart from a day nobody looked. One row per day.
func (s *PgStore) RecordEmailListMiss(ctx context.Context, day, reason string) error {
	ws, err := s.ws("personal")
	if err != nil {
		return err
	}
	if _, err := s.Pool.Exec(ctx, `INSERT INTO founderos_email_list_sync_misses (attempted_on, workspace_id, reason)
		VALUES ($1::date, $2, $3)
		ON CONFLICT (attempted_on) DO UPDATE SET reason = EXCLUDED.reason, workspace_id = EXCLUDED.workspace_id`,
		day, ws, reason); err != nil {
		return fmt.Errorf("analytics email-list miss: %w", err)
	}
	return nil
}
