package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/doctor"
)

// doctorTestDB is a throwaway database on the bridge Postgres, dropped after
// the test. Never businessos_dev.
func doctorTestDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_doctor_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func doctorFakeEngine(t *testing.T) *httptest.Server {
	t.Helper()
	bodies := map[string]string{
		"/api/health":       `{"status":"up","checks":{"store":":ok"},"degraded":[]}`,
		"/api/workspaces":   `{"workspaces":[{"slug":"founderos"},{"slug":"default"}]}`,
		"/api/stores":       `{"stores":[{"id":"relational","status":"available","row_count":1500,"table_counts":{"claims":721,"contexts":747,"facts":0}}]}`,
		"/api/stores/audit": `{"ok":false,"checks":[{"name":"sqlite_integrity","ok":true,"detail":"ok"},{"name":"verified_backup","ok":false,"detail":":no_verified_backup"}]}`,
		"/api/metrics":      `{"counters":{"optimal_engine.search.query":3}}`,
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("doctor wrote to an engine: %s %s", r.Method, r.URL.Path)
		}
		if b, ok := bodies[r.URL.Path]; ok {
			if r.URL.Path == "/api/stores/audit" {
				w.WriteHeader(http.StatusServiceUnavailable) // a failing audit, as the engines answer it
			}
			_, _ = w.Write([]byte(b))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

type doctorBody struct {
	Engines    []doctor.EngineReading `json:"engines"`
	Checks     []doctor.Check         `json:"checks"`
	Score      *int                   `json:"score"`
	Volume     doctor.Volume          `json:"volume"`
	Layers     []doctor.Layer         `json:"layers"`
	Axes       []doctor.PillarAxis    `json:"axes"`
	Relational struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	} `json:"relational"`
	Brain struct {
		Agents []string `json:"agents"`
		Last   *struct {
			AgentID    string    `json:"agentId"`
			FinishedAt time.Time `json:"finishedAt"`
			OK         bool      `json:"ok"`
		} `json:"last"`
	} `json:"brain"`
}

func doctorGet(t *testing.T, d *Deps) doctorBody {
	t.Helper()
	r := router(t, d)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/doctor", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("doctor without a session: %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/doctor", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("doctor: %d %s", w.Code, w.Body.String())
	}
	var b doctorBody
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDoctorPageReportsEnginesAndRelationalHealth(t *testing.T) {
	pool := doctorTestDB(t)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('dept-comms',$1,'Communications','comms','#fff',2),('dept-sales',$1,'Sales','sales','#fff',1)`, []any{ws}},
		{`INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier) VALUES ('sales-agent',$1,'dept-sales','Sales','active','lead'),('data-agent',$1,'dept-comms','Data','idle','worker')`, []any{ws}},
		{`INSERT INTO founderos_sop_tasks (id, workspace_id, department_id, title, assignee_kind, assignee_id) VALUES ('t1',$1,'dept-sales','Call','agent','sales-agent')`, []any{ws}},
		{`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok) VALUES
			('r1',$1,'sales-agent',$2,$2,true),
			('r2',$1,'data-agent',$3,$3,true),
			('r3',$1,'data-agent',$2,$2,false),
			('r4',$1,'sales-calls-data',$4,$4,true)`, []any{ws, now.Add(-30 * time.Minute), now.Add(-3 * time.Hour), now.Add(-10 * time.Minute)}},
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s.q, s.args...); err != nil {
			t.Fatal(err)
		}
	}

	srv := doctorFakeEngine(t)
	defer srv.Close()
	prev := doctorEngines
	doctorEngines = func() []doctor.Engine {
		return []doctor.Engine{{Name: "hub", URL: srv.URL}, {Name: "macbook", URL: "http://127.0.0.1:1"}}
	}
	t.Cleanup(func() { doctorEngines = prev })

	b := doctorGet(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})

	if len(b.Engines) != 2 || !b.Engines[0].Reachable || b.Engines[1].Reachable || b.Engines[1].Error == "" {
		t.Fatalf("engines = %+v", b.Engines)
	}
	// hub: 3 checks, 2 ok → 67%, times one of two engines up → 33.
	if b.Score == nil || *b.Score != 33 || b.Volume.Headline == nil || *b.Volume.Headline != 33 {
		t.Fatalf("score = %v / %v", b.Score, b.Volume.Headline)
	}
	if len(b.Checks) != 3 {
		t.Fatalf("checks = %+v", b.Checks)
	}
	if b.Volume.Claims == nil || *b.Volume.Claims != 721 || b.Volume.Facts == nil || *b.Volume.Facts != 0 {
		t.Fatalf("claims/facts = %v/%v", b.Volume.Claims, b.Volume.Facts)
	}
	if !b.Relational.OK {
		t.Fatalf("relational = %+v", b.Relational)
	}
	if len(b.Axes) != 2 || b.Axes[0].ID != "dept-sales" || b.Axes[0].Freshness != 100 {
		t.Fatalf("axes = %+v", b.Axes)
	}
	// Like v1, Brain Runs are the data agent's alone: its two runs count, the
	// newer sales-calls-data run does not.
	if len(b.Brain.Agents) != 1 || b.Brain.Agents[0] != "data-agent" {
		t.Fatalf("brain agents = %v, want [data-agent]", b.Brain.Agents)
	}
	if b.Volume.RunsInWindow != 2 || b.Volume.FailedInWindow != 1 {
		t.Fatalf("brain runs = %d (%d failed)", b.Volume.RunsInWindow, b.Volume.FailedInWindow)
	}
	if b.Brain.Last == nil || b.Brain.Last.AgentID != "data-agent" || b.Brain.Last.OK {
		t.Fatalf("last brain run = %+v", b.Brain.Last)
	}
	if len(b.Layers) != 6 {
		t.Fatalf("layers = %+v", b.Layers)
	}
}

func TestDoctorPageWithoutPostgresSaysSo(t *testing.T) {
	prev := doctorEngines
	doctorEngines = func() []doctor.Engine { return nil }
	t.Cleanup(func() { doctorEngines = prev })

	b := doctorGet(t, &Deps{Board: connectors.NewRegistry()})
	if b.Relational.OK || b.Relational.Error == "" {
		t.Fatalf("relational = %+v", b.Relational)
	}
	if b.Axes != nil {
		t.Fatalf("axes invented without Postgres: %+v", b.Axes)
	}
	if b.Score != nil || len(b.Engines) != 0 {
		t.Fatalf("score = %v engines = %v", b.Score, b.Engines)
	}
	if len(b.Brain.Agents) == 0 {
		t.Fatal("the memory agents the step line counts are not named")
	}
}
