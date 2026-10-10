package agents

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
	name := fmt.Sprintf("founderos_agents_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func TestPgStoreScopesRunsToTheWorkspace(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	st := NewPgStore(pool, ws)
	in := 10
	cost := 0.5
	model := "claude-sonnet-5"
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i, id := range []string{"a", "b", "a"} {
		r := Run{ID: fmt.Sprintf("r%d", i), AgentID: id, StartedAt: now.Add(time.Duration(i) * time.Second), FinishedAt: now.Add(time.Duration(i)*time.Second + time.Millisecond), OK: i != 1, Summary: "s", Model: &model, TokensIn: &in, CostUSD: &cost}
		if err := st.InsertRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	runs, err := st.Recent(ctx, "a", 10)
	if err != nil || len(runs) != 2 || runs[0].ID != "r2" || *runs[0].CostUSD != 0.5 || *runs[0].Model != model {
		t.Fatalf("runs = %+v err = %v (newest first)", runs, err)
	}
	all, _ := st.Recent(ctx, "", 1)
	if len(all) != 1 {
		t.Fatalf("limit ignored: %d", len(all))
	}
	b := Broadcast{ID: "b1", Message: "hi", CreatedAt: now, Replies: []Reply{{AgentID: "a", OK: true, Reply: "yo", FinishedAt: now}}}
	if err := st.InsertBroadcast(ctx, b); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM founderos_broadcast_replies WHERE broadcast_id='b1' AND workspace_id=$1`, ws).Scan(&n)
	if n != 1 {
		t.Fatalf("replies stored = %d", n)
	}
}

func TestPgCronSourceReadsDefinitionsWithLastRun(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_agent_crons (id, workspace_id, agent_id, schedule, description, enabled, created_at)
		VALUES ('cron-comms-digest', $1, 'comms-digest', '0 9 * * *', 'Morning digest', true, now())`, ws); err != nil {
		t.Fatal(err)
	}
	src := NewPgCronSource(pool, ws)
	crons, err := src.Crons(ctx)
	if err != nil || len(crons) != 1 || crons[0].LastRunAt != nil || crons[0].AgentID != "comms-digest" {
		t.Fatalf("crons = %+v err=%v", crons, err)
	}
	started := time.Now().UTC().Truncate(time.Millisecond)
	if err := src.RecordCronRun(ctx, crons[0], Run{ID: "run-1", AgentID: "comms-digest", StartedAt: started, FinishedAt: started.Add(time.Second), OK: true, Summary: "ok"}); err != nil {
		t.Fatal(err)
	}
	crons, _ = src.Crons(ctx)
	if crons[0].LastRunAt == nil || !crons[0].LastRunAt.Equal(started) {
		t.Fatalf("last run = %v, want %v", crons[0].LastRunAt, started)
	}
}
