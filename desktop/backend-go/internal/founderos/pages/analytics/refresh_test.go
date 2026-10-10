package analytics

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

type fakeMetricWriter struct {
	got  map[string]float64
	fail map[string]bool
}

func (f *fakeMetricWriter) RecordMetric(_ context.Context, id string, v float64, _ time.Time) error {
	if f.fail[id] {
		return errors.New("write failed")
	}
	f.got[id] = v
	return nil
}

func fp(v float64) *float64 { return &v }

// recordOperatingSnapshots: honest-pending nulls are skipped, never written
// as a fake zero; a real zero is a reading and is recorded.
func TestRecordOperatingSnapshotsSkipsPendingNulls(t *testing.T) {
	w := &fakeMetricWriter{got: map[string]float64{}, fail: map[string]bool{"stripe": true}}
	inputs := []MetricInput{
		{ID: "audience", Value: fp(1200)},
		{ID: "subscribers", Value: nil},
		{ID: "brain", Value: fp(0)},
		{ID: "stripe", Value: fp(40)},
	}
	n := RecordOperatingSnapshots(context.Background(), w, inputs, time.Now())
	if n != 2 || w.got["audience"] != 1200 || len(w.got) != 2 {
		t.Fatalf("recorded %d: %v", n, w.got)
	}
	if _, ok := w.got["brain"]; !ok {
		t.Fatal("a real zero is a reading (the TS only skips null)")
	}
	if _, ok := w.got["subscribers"]; ok {
		t.Fatal("a pending null was written")
	}
}

// emailListOutcome is syncBeehiivEmail's decision: only a fresh reading is an
// observation; no reading or a stale one is recorded as a miss.
func TestEmailListOutcome(t *testing.T) {
	snap, miss := EmailListOutcome(nil, "2026-09-30")
	if snap != nil || miss != "no reading from beehiiv" {
		t.Fatalf("nil: %v %q", snap, miss)
	}
	snap, miss = EmailListOutcome(&beehiiv.Reading{Subscribers: 18266, Fresh: false}, "2026-09-30")
	if snap != nil || !strings.HasPrefix(miss, "stale reading") {
		t.Fatalf("stale: %v %q", snap, miss)
	}
	snap, miss = EmailListOutcome(&beehiiv.Reading{Subscribers: 20315, PublicationID: "pub_1", Metric: "active_subscriptions", Fresh: true}, "2026-09-30")
	if miss != "" || snap == nil {
		t.Fatalf("fresh: %v %q", snap, miss)
	}
	want := EmailListSnapshot{CapturedOn: "2026-09-30", Subscribers: 20315, Source: "beehiiv", PublicationID: "pub_1", Metric: "active_subscriptions", Quality: "ok"}
	if *snap != want {
		t.Fatalf("snapshot = %+v", *snap)
	}
}

func TestStoreWritersWithoutPostgresAreErrors(t *testing.T) {
	s := &PgStore{}
	ctx := context.Background()
	if s.RecordMetric(ctx, "audience", 1, time.Now()) == nil ||
		s.RecordEmailListSnapshot(ctx, EmailListSnapshot{CapturedOn: "2026-09-30"}) == nil ||
		s.RecordEmailListMiss(ctx, "2026-09-30", "x") == nil {
		t.Fatal("a write without a pool must fail, never read as recorded")
	}
}

func TestStoreWritesTheSnapshotRows(t *testing.T) {
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"founderos", "personal"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	s := &PgStore{Pool: pool, Workspaces: ws}
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	// Metric: the same (metric, instant) overwrites; another instant accrues.
	for _, v := range []float64{10, 11} {
		if err := s.RecordMetric(ctx, "audience", v, at); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordMetric(ctx, "audience", 12, at.Add(15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var n int
	var sum float64
	var wsID string
	if err := pool.QueryRow(ctx, `SELECT count(*), sum(value), min(workspace_id::text) FROM founderos_metric_snapshots WHERE metric_id = 'audience'`).Scan(&n, &sum, &wsID); err != nil {
		t.Fatal(err)
	}
	if n != 2 || sum != 23 || wsID != ws["founderos"] {
		t.Fatalf("metric rows: n=%d sum=%v ws=%s", n, sum, wsID)
	}

	// Email list: a same-day reading overwrites (idempotent ticks).
	for _, subs := range []int{20303, 20315} {
		if err := s.RecordEmailListSnapshot(ctx, EmailListSnapshot{CapturedOn: "2026-09-30", Subscribers: subs, Source: "beehiiv", PublicationID: "pub_1", Metric: "m", Quality: "ok"}); err != nil {
			t.Fatal(err)
		}
	}
	var subs int
	if err := pool.QueryRow(ctx, `SELECT count(*), max(subscribers) FROM founderos_email_list_snapshots WHERE workspace_id = $1`, ws["personal"]).Scan(&n, &subs); err != nil {
		t.Fatal(err)
	}
	if n != 1 || subs != 20315 {
		t.Fatalf("email rows: n=%d subs=%d", n, subs)
	}
	pts, err := s.EmailPoints(ctx)
	if err != nil || len(pts) != 1 || pts[0].Value != 20315 {
		t.Fatalf("points = %v %v", pts, err)
	}

	// Miss: one per day, the latest reason wins.
	for _, r := range []string{"no reading from beehiiv", "stale reading"} {
		if err := s.RecordEmailListMiss(ctx, "2026-09-29", r); err != nil {
			t.Fatal(err)
		}
	}
	var reason string
	if err := pool.QueryRow(ctx, `SELECT count(*), max(reason) FROM founderos_email_list_sync_misses WHERE workspace_id = $1`, ws["personal"]).Scan(&n, &reason); err != nil {
		t.Fatal(err)
	}
	if n != 1 || reason != "stale reading" {
		t.Fatalf("miss rows: n=%d reason=%q", n, reason)
	}
}
