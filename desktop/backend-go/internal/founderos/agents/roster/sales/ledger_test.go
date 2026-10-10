package sales

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

// throwawayDB is a fresh, migrated database on the bridge Postgres; the
// test skips when that Postgres is not running.
func throwawayDB(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_sales_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	return pool, ws
}

func TestPgPlaudLedger(t *testing.T) {
	pool, ws := throwawayDB(t)
	ctx := context.Background()
	l := NewPgPlaudLedger(pool, ws)

	// Migration 161 widened the via CHECK to 'oe': the ledger is ready, and
	// the readiness probe leaves nothing behind.
	if err := l.Ready(ctx); err != nil {
		t.Fatalf("Ready after migration 161: %v", err)
	}
	if got, _ := l.Ingested(ctx); len(got) != 0 {
		t.Fatalf("probe leaked: %v", got)
	}
	if err := NewPgPlaudLedger(pool, "").Ready(ctx); err == nil {
		t.Fatal("an unresolved workspace must not be ready")
	}
	row := PlaudIngest{FileID: "f1", Title: "Site walk", RecordedAt: "2026-08-26T15:00:00Z", IngestedAt: time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC), Via: ViaOE, Slug: "2026-08-26-site-walk", Claims: 2}
	if err := l.Insert(ctx, row); err != nil {
		t.Fatal(err)
	}
	if err := l.Insert(ctx, PlaudIngest{FileID: "f2", Title: "Undated", IngestedAt: time.Now(), Via: ViaOE, Slug: "undated-x"}); err != nil {
		t.Fatalf("an empty recorded_at is NULL: %v", err)
	}
	if err := l.Insert(ctx, row); err != nil {
		t.Fatalf("a repeat insert is a no-op: %v", err)
	}
	got, err := l.Ingested(ctx)
	if err != nil || len(got) != 2 || !got["f1"] || !got["f2"] {
		t.Fatalf("ingested %v err %v", got, err)
	}
	var via string
	var claims int
	var recorded *time.Time
	_ = pool.QueryRow(ctx, `SELECT via, claims, recorded_at FROM founderos_plaud_ingests WHERE file_id='f1' AND workspace_id=$1`, ws).Scan(&via, &claims, &recorded)
	if via != "oe" || claims != 2 || recorded == nil || !recorded.Equal(time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)) {
		t.Fatalf("row via=%s claims=%d recorded=%v", via, claims, recorded)
	}
}

func TestPgArchiveLedger(t *testing.T) {
	pool, ws := throwawayDB(t)
	ctx := context.Background()
	l := NewPgArchiveLedger(pool, ws)
	if got, err := l.Archived(ctx); err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
	// Another source's archive rows are not Fathom's.
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_call_archive (source, external_id, workspace_id, slug) VALUES ('plaud', 'p1', $1, 'x')`, ws); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	e := ArchiveEntry{RecordingID: "12", Version: ArchiveVersion - 1, Workspace: "vantage", SignalID: "sig", Page: "meetings/x--fathom-12.md", Title: "T", ArchivedAt: at}
	if err := l.Record(ctx, e); err != nil {
		t.Fatal(err)
	}
	e.Version = ArchiveVersion
	if err := l.Record(ctx, e); err != nil {
		t.Fatalf("re-recording upserts: %v", err)
	}
	got, err := l.Archived(ctx)
	if err != nil || len(got) != 1 || got["12"] != e {
		t.Fatalf("got %+v err %v", got, err)
	}
	var slug, signal string
	if err := pool.QueryRow(ctx, `SELECT slug, coalesce(signal_id,'') FROM founderos_call_archive WHERE source='fathom' AND external_id='12'`).Scan(&slug, &signal); err != nil || slug != e.Page || signal != "sig" {
		t.Fatalf("row in founderos_call_archive: slug=%q signal=%q err=%v", slug, signal, err)
	}
	if err := NewPgArchiveLedger(pool, "").Record(ctx, e); err == nil {
		t.Fatal("an unresolved workspace must refuse")
	}
}
