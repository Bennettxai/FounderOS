package console

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
	name := fmt.Sprintf("founderos_console_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func TestReadRosterScopesToFounderOS(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	var ws, other string
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws))
	must(pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Other','other','u1') RETURNING id::text`).Scan(&other))
	_, err := pool.Exec(ctx, `INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('d1',$1,'Sales','sales','#fff',1)`, ws)
	must(err)
	for i, st := range []string{"active", "active", "idle"} {
		_, err := pool.Exec(ctx, `INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier) VALUES ($1,$2,'d1',$1,$3,'worker')`, fmt.Sprintf("a%d", i), ws, st)
		must(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	runs := []struct {
		id, ws string
		age    time.Duration
		ok     bool
	}{
		{"r1", ws, time.Hour, true},
		{"r2", ws, 2 * time.Hour, false},
		{"r3", ws, 20 * 24 * time.Hour, true}, // outside the 15-day window
		{"x1", other, time.Hour, true},        // another workspace
	}
	for _, r := range runs {
		_, err := pool.Exec(ctx, `INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok, summary) VALUES ($1,$2,'a0',$3,$3,$4,'s')`, r.id, r.ws, now.Add(-r.age), r.ok)
		must(err)
	}

	roster, recent, window, err := ReadRoster(ctx, pool, now)
	must(err)
	if roster.Active != 2 || roster.Total != 3 {
		t.Fatalf("roster = %+v", roster)
	}
	if len(recent) != 3 || recent[0].ID != "r1" || recent[1].OK {
		t.Fatalf("recent = %+v", recent)
	}
	if len(window) != 2 {
		t.Fatalf("window = %+v", window)
	}
}

func TestReadRosterWithoutTheWorkspaceIsAnError(t *testing.T) {
	pool := throwawayDB(t)
	if _, _, _, err := ReadRoster(context.Background(), pool, time.Now()); err == nil {
		t.Fatal("a missing founderos workspace must read as unknown, not an empty roster")
	}
	if _, _, _, err := ReadRoster(context.Background(), nil, time.Now()); err == nil {
		t.Fatal("no pool must be an error")
	}
}
