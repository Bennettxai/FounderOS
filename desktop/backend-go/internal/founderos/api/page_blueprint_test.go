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
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/blueprint"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

// blueprintPG is a throwaway database on the bridge Postgres, dropped after
// the test (never businessos_dev).
func blueprintPG(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_page_blueprint_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func blueprintGet(t *testing.T, d *Deps) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router(t, d).ServeHTTP(w, authed(http.MethodGet, "/api/founderos/pages/blueprint", nil))
	return w
}

func TestBlueprintNeedsASession(t *testing.T) {
	w := httptest.NewRecorder()
	router(t, &Deps{Board: connectors.NewRegistry()}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/blueprint", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
}

func TestBlueprintSaysWhenTheRosterIsUnreachable(t *testing.T) {
	w := blueprintGet(t, &Deps{Board: connectors.NewRegistry()})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no database: %d %s", w.Code, w.Body)
	}
}

func TestBlueprintCompilesFromTheLiveRegistries(t *testing.T) {
	prev := blueprintHost
	blueprintHost = func() string { return "" } // probe nothing in tests
	t.Cleanup(func() { blueprintHost = prev })

	pool := blueprintPG(t)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO founderos_departments (id, workspace_id, name, slug, tagline, color, ord) VALUES ('d-sales', $1, 'Sales', 'sales', 'pipeline', '#fff', 1)`,
		`INSERT INTO founderos_agents (id, workspace_id, department_id, name, role, status, tier, tools) VALUES
		   ('conductor', $1, 'd-sales', 'Conductor', 'Super agent', 'active', 'lead', '[]'),
		   ('inbox', $1, 'd-sales', 'Inbox', 'Mail', 'active', 'worker', '["slack"]')`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatal(err)
		}
	}
	reg := connectors.NewRegistry()
	reg.Register(connectors.Meta{ID: "slack", Name: "Slack", Kind: connectors.KindSlack}, fakeConn{connectors.Status{State: connectors.StateConnected, Detail: "ok"}})

	w := blueprintGet(t, &Deps{Pool: pool, Board: reg})
	var body struct {
		OK    bool            `json:"ok"`
		Graph blueprint.Graph `json:"graph"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || !body.OK {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if p := blueprint.Validate(body.Graph); len(p) > 0 {
		t.Fatalf("invalid graph: %v", p)
	}
	nodes := map[string]blueprint.Node{}
	for _, n := range body.Graph.Nodes {
		nodes[n.ID] = n
	}
	if nodes["connector-slack"].Status != "live" || nodes["dept-sales"].Name != "Sales" {
		t.Fatalf("nodes = %+v / %+v", nodes["connector-slack"], nodes["dept-sales"])
	}
	// no runtime on these Deps: the agent is never claimed live
	if nodes["agent-inbox"].Status == "live" {
		t.Fatal("agent live with no runtime registry")
	}
	uses := false
	for _, e := range body.Graph.Edges {
		if e.From == "agent-inbox" && e.To == "connector-slack" && e.Kind == "uses" {
			uses = true
		}
	}
	if !uses {
		t.Fatal("agent->connector edge missing")
	}
}
