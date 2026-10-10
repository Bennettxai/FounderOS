package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

const topoYAML = `
version: 1
engines:
  macbook: {tier: device}
  hub: {tier: shared}
  mini: {tier: device}
workspaces:
  - {slug: vantage, name: Vantage, home: hub, class: business}
  - {slug: personal, name: Personal, home: macbook, class: personal}
  - {slug: hermes, name: Hermes, home: mini, class: device}
brain_store: {default: vantage, routes: {}}
retired: {}
`

func topo(t *testing.T) *topology.Topology {
	t.Helper()
	tp, err := topology.Parse([]byte(topoYAML))
	if err != nil {
		t.Fatal(err)
	}
	return tp
}

var engines = map[string]Engine{
	"hub":     {URL: "http://127.0.0.1:4211", Key: "hub-key"},
	"macbook": {URL: "http://127.0.0.1:4210", Key: "mac-key"},
}

func TestPlanCoversBusinessAndPersonalWorkspacesOnly(t *testing.T) {
	specs, err := Plan(topo(t), engines)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("specs = %+v; device workspaces (hermes) are engine-only, not BusinessOS workspaces", specs)
	}
	m := specs[0]
	if m.Slug != "vantage" || m.Name != "Vantage" || m.Engine.BaseURL != "http://127.0.0.1:4211" || m.Engine.APIKey != "hub-key" || m.Engine.Workspace != "default:vantage" || !m.Engine.Enabled {
		t.Fatalf("vantage spec = %+v", m)
	}
	if specs[1].Engine.BaseURL != "http://127.0.0.1:4210" {
		t.Fatalf("personal must route to the macbook engine: %+v", specs[1])
	}
}

func TestPlanFailsWhenAHomeEngineHasNoEndpoint(t *testing.T) {
	if _, err := Plan(topo(t), map[string]Engine{"hub": engines["hub"]}); err == nil {
		t.Fatal("personal's home engine (macbook) has no endpoint; Plan must refuse rather than create an unrouted workspace")
	}
}

// throwawayDB creates a fresh database with the full BusinessOS schema on the
// bridge Postgres and drops it afterwards. Never touches businessos_dev.
func throwawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = adminPool.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v); start it with make bridge-up", err)
	}
	name := fmt.Sprintf("founderos_bootstrap_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		adminPool.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return pool
}

func TestApplyCreatesRoutedWorkspacesIdempotently(t *testing.T) {
	pool := throwawayDB(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO "user" (id, name, email) VALUES ('u1', 'Owner', 'owner@example.com')`); err != nil {
		t.Fatal(err)
	}
	specs, err := Plan(topo(t), engines)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(ctx, pool, "missing@example.com", specs); err == nil {
		t.Fatal("unknown owner email must fail with a sign-up hint")
	}
	for run := 1; run <= 2; run++ {
		res, err := Apply(ctx, pool, "owner@example.com", specs)
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if run == 1 && (res.Created != 2 || res.Updated != 0) {
			t.Fatalf("first run = %+v", res)
		}
		if run == 2 && (res.Created != 0 || res.Updated != 2) {
			t.Fatalf("second run = %+v", res)
		}
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM workspaces`).Scan(&n)
	if n != 2 {
		t.Fatalf("workspaces = %d, want 2", n)
	}
	var raw []byte
	var owner, role string
	err = pool.QueryRow(ctx, `SELECT w.settings->'optimal_engine', w.owner_id, m.role
		FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id AND m.user_id = w.owner_id
		WHERE w.slug = 'vantage'`).Scan(&raw, &owner, &role)
	if err != nil {
		t.Fatal(err)
	}
	var eng map[string]any
	_ = json.Unmarshal(raw, &eng)
	if eng["base_url"] != "http://127.0.0.1:4211" || eng["api_key"] != "hub-key" || eng["workspace"] != "default:vantage" || eng["enabled"] != true {
		t.Fatalf("engine settings = %s", raw)
	}
	if owner != "u1" || role != "owner" {
		t.Fatalf("owner=%s role=%s", owner, role)
	}
}
