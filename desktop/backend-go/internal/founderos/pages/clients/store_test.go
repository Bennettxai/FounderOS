package clients

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func throwawayDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_clientspage_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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
	return pool
}

func TestPgStoreWithoutFounderOSIsAnErrorNotEmpty(t *testing.T) {
	pool := throwawayDB(t)
	if _, err := NewPgStore(pool).All(context.Background()); err == nil {
		t.Fatal("a missing founderos workspace must fail the read")
	}
}

func TestPgStoreWithoutAPoolIsAnError(t *testing.T) {
	if _, err := NewPgStore(nil).All(context.Background()); err == nil {
		t.Fatal("nil pool")
	}
}

// Ported from FounderOS v1 tests/client-work.test.ts: briefs persist and a
// claimed launch is never silently retried.
func TestPgStorePersistsAndNeverReclaims(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	for _, slug := range []string{"founderos", "personal"} {
		if _, err := pool.Exec(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1')`, slug); err != nil {
			t.Fatal(err)
		}
	}
	var personal string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug='personal'`).Scan(&personal)
	// a row in another workspace never leaks onto the board
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_client_work (id, workspace_id, client_id, brief, status, created_at) VALUES ('other', $1, 'x', 'elsewhere', 'saved', now())`, personal); err != nil {
		t.Fatal(err)
	}
	s := NewPgStore(pool)
	if err := s.Add(ctx, Work{ID: "draft-0", ClientID: "silvio-big-mamas", Brief: "Older brief", Status: StatusSaved, CreatedAt: "2026-09-20T10:00:00.000Z"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(ctx, Work{ID: "draft-1", ClientID: "silvio-big-mamas", Brief: "Write a game day carousel", Status: StatusSaved, CreatedAt: "2026-09-23T10:00:00.000Z"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Claim(ctx, "draft-1"); !ok || err != nil {
		t.Fatalf("first claim %v %v", ok, err)
	}
	if ok, _ := s.Claim(ctx, "draft-1"); ok {
		t.Fatal("second claim must fail")
	}
	all, err := s.All(ctx)
	if err != nil || len(all) != 2 || all[0].ID != "draft-1" || all[0].Status != StatusLaunching || all[0].CreatedAt != "2026-09-23T10:00:00.000Z" {
		t.Fatalf("%+v %v", all, err)
	}
	ws := "workspace-1"
	if err := s.Finish(ctx, "draft-1", &ws); err != nil {
		t.Fatal(err)
	}
	w, err := s.Get(ctx, "draft-1")
	if err != nil || w.Status != StatusLaunched || w.WorkspaceID == nil || *w.WorkspaceID != ws || w.Detail == nil {
		t.Fatalf("%+v %v", w, err)
	}
	// finish only moves a launching row
	if err := s.Finish(ctx, "draft-0", nil); err != nil {
		t.Fatal(err)
	}
	if w, _ := s.Get(ctx, "draft-0"); w.Status != StatusSaved {
		t.Fatalf("finish moved a saved row: %+v", w)
	}
	s.Claim(ctx, "draft-0")
	_ = s.Finish(ctx, "draft-0", nil)
	if w, _ := s.Get(ctx, "draft-0"); w.Status != StatusNeedsAttention || w.WorkspaceID != nil || w.Detail == nil {
		t.Fatalf("%+v", w)
	}
	if w, err := s.Get(ctx, "other"); w != nil || err != nil {
		t.Fatalf("other workspace row visible: %+v %v", w, err)
	}
}
