package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func TestStoreWithoutPostgresIsAnError(t *testing.T) {
	s := &PgStore{}
	if _, err := s.Runs(context.Background(), time.Now()); err == nil {
		t.Fatal("no pool must fail, never read as no runs")
	}
	if _, err := s.Social(context.Background()); err == nil {
		t.Fatal("no pool must fail")
	}
}

func TestStoreReadsTheAnalyticsRows(t *testing.T) {
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
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	fo, pb := ws["founderos"], ws["personal"]
	exec(`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('dept-x', $1, 'X', 'x', '#fff', 1)`, fo)
	exec(`INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier) VALUES ('scout', $1, 'dept-x', 'Scout', 'active', 'worker')`, fo)
	exec(`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok) VALUES
		('r1', $1, 'scout', '2026-09-29T10:00:00Z', '2026-09-29T10:01:00Z', true),
		('r2', $1, 'scout', '2026-09-20T10:00:00Z', '2026-09-20T10:01:00Z', false),
		('r3', $1, 'gone', '2026-06-01T10:00:00Z', '2026-06-01T10:01:00Z', true)`, fo)
	exec(`INSERT INTO founderos_social_accounts (platform, workspace_id, handle, ord) VALUES ('youtube', $1, '@y', 2), ('instagram', $1, '@i', 1)`, pb)
	exec(`INSERT INTO founderos_social_snapshots (platform, captured_on, workspace_id, followers, source) VALUES
		('instagram', '2026-09-20', $1, 100, 'zernio-config'), ('instagram', '2026-09-27', $1, 110, 'zernio-config')`, pb)
	exec(`INSERT INTO founderos_email_list_snapshots (captured_on, workspace_id, subscribers, source, quality) VALUES
		('2026-09-01', $1, 4812, 'seed', 'ok'), ('2026-09-10', $1, 900, 'beehiiv', 'ok'), ('2026-09-11', $1, 0, 'beehiiv', 'suspect')`, pb)
	exec(`INSERT INTO founderos_metric_snapshots (metric_id, captured_at, workspace_id, value) VALUES
		('leads', '2026-09-28T01:00:00Z', $1, 3), ('leads', '2026-09-28T09:00:00Z', $1, 5), ('leads', '2026-09-29T09:00:00Z', $1, 6)`, fo)

	s := &PgStore{Pool: pool, Workspaces: ws}
	runs, err := s.Runs(ctx, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(runs) != 2 || runs[0].StartedAt != "2026-09-29T10:00:00Z" || runs[1].OK {
		t.Fatalf("runs = %+v %v", runs, err)
	}
	all, last7, err := s.RunCount(ctx, time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC))
	if err != nil || all != 3 || last7 != 1 {
		t.Fatalf("counts = %d %d %v", all, last7, err)
	}
	names, err := s.AgentNames(ctx)
	if err != nil || names["scout"] != "Scout" {
		t.Fatalf("names = %v %v", names, err)
	}
	social, err := s.Social(ctx)
	if err != nil || len(social) != 2 || social[0].Platform != "instagram" || len(social[0].Points) != 2 || len(social[1].Points) != 0 {
		t.Fatalf("social = %+v %v", social, err)
	}
	email, err := s.EmailPoints(ctx)
	if err != nil || len(email) != 1 || email[0].Value != 900 {
		t.Fatalf("seeded and suspect email rows never count: %+v %v", email, err)
	}
	hist, err := s.MetricHistory(ctx, "leads", 7, "2026-09-30")
	if err != nil || len(hist) != 2 || hist[0] != 5 || hist[1] != 6 {
		t.Fatalf("history (last value per day) = %v %v", hist, err)
	}
}
