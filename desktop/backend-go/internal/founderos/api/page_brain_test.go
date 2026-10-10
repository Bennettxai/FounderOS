package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/pages/brain"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// brainFakeOE serves /api/health, /api/graph, /api/search and /api/ingest for the
// founderos and vantage workspaces, and records ingests.
type brainFakeOE struct {
	srv     *httptest.Server
	mu      sync.Mutex
	ingests []map[string]any
	down    bool
}

func newBrainFakeOE(t *testing.T) *brainFakeOE {
	f := &brainFakeOE{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(401)
			return
		}
		f.mu.Lock()
		down := f.down
		f.mu.Unlock()
		if down {
			w.WriteHeader(503)
			return
		}
		ws := r.URL.Query().Get("workspace")
		switch r.URL.Path {
		case "/api/health":
			_, _ = w.Write([]byte(`{"status":"up","ok?":true,"checks":{"store":":ok"},"degraded":[]}`))
		case "/api/graph":
			switch ws {
			case "default:founderos":
				_, _ = w.Write([]byte(`{"contexts":[{"id":"a","node":"inbox","title":"Hermes worker pool","l0_abstract":"how the pool runs","genre":"sop","modified_at":"2026-09-30T06:40:38Z"},{"id":"b","node":"tech","title":"Conductor","modified_at":"2026-09-30T06:40:38Z"}],"edges":[{"source":"a","target":"b","relation":"cross_ref"}]}`))
			case "default:vantage":
				_, _ = w.Write([]byte(`{"contexts":[{"id":"m","node":"team","title":"Vantage offer","modified_at":"2026-09-30T06:40:38Z"}],"edges":[]}`))
			default:
				_, _ = w.Write([]byte(`{"contexts":[],"edges":[]}`))
			}
		case "/api/search":
			if ws == "default:founderos" {
				_, _ = w.Write([]byte(`{"results":[{"id":"a","title":"Hermes worker pool","uri":"optimal://a","l0_abstract":"how the pool runs"},{"id":"b","title":"Conductor","uri":"optimal://b","l0_abstract":"the super agent"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"results":[]}`))
		case "/api/ingest":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.mu.Lock()
			f.ingests = append(f.ingests, body)
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"ok":true,"signal_id":"sig-1"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// brainTestTopology splits the workspaces over two engines so the page's
// multi-engine rules stay under test: founderos and vantage live on the
// local engine, personal on a second device engine that the tests either
// leave without an endpoint (unstaged) or point at a dead server.
const brainTestTopology = `
version: 1
engines:
  local: {tier: device}
  device: {tier: device}
workspaces:
  - {slug: founderos, home: local, class: business}
  - {slug: vantage, home: local, class: business}
  - {slug: personal, home: device, class: personal}
brain_store:
  default: founderos
  routes: {}
retired: {}
`

// brainTestEnv routes founderos + vantage to one fake local engine; personal's
// engine has no endpoint, so it is unstaged.
func brainTestEnv(t *testing.T, oe *brainFakeOE) *brainEnv {
	t.Helper()
	t.Setenv("FOUNDEROS_MEMORY_SEARCH", "search") // the fake engine speaks /api/search; grep is covered in memory
	topo, err := topology.Parse([]byte(brainTestTopology))
	if err != nil {
		t.Fatal(err)
	}
	if err := topo.Validate(); err != nil {
		t.Fatal(err)
	}
	local := brain.Engine{Name: "local", URL: oe.srv.URL, Key: "k"}
	up := false
	return &brainEnv{
		Topo:       topo,
		Engines:    []brain.Engine{local},
		Workspaces: []brainWorkspace{{Slug: "founderos", Engine: local}, {Slug: "vantage", Engine: local}},
		Unstaged:   []string{"personal"},
		Memory:     memory.New(topo, map[string]memory.Endpoint{"local": {URL: oe.srv.URL, Key: "k"}}),
		Client:     brain.NewEngineClient(),
		Rerank:     func(context.Context, string, []string) ([]float64, bool) { return nil, false },
		RerankUp:   func(context.Context) brain.RerankState { _ = up; return brain.RerankUnchecked },
		Now:        func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) },
		Org: func(context.Context) (*brainOrg, error) {
			return nil, errNoOrg
		},
		Board: func(context.Context) ([]paperclip.Agent, error) { return nil, errors.New("board unreachable") },
	}
}

func brainRouter(t *testing.T, env *brainEnv) http.Handler {
	t.Helper()
	prev := newBrainEnv
	newBrainEnv = func(*Deps) *brainEnv { return env }
	t.Cleanup(func() { newBrainEnv = prev })
	return router(t, &Deps{Board: connectors.NewRegistry()})
}

func brainCall(t *testing.T, h http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestBrainSatellitesOverviewAndGraphReadTheEngines(t *testing.T) {
	oe := newBrainFakeOE(t)
	h := brainRouter(t, brainTestEnv(t, oe))

	code, sat := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/satellites", nil)
	if code != 200 || sat["mounted"] != true || sat["pages"].(float64) != 3 || sat["folders"].(float64) != 2 {
		t.Fatalf("satellites %d %v", code, sat)
	}
	if !strings.HasPrefix(sat["statusLine"].(string), "optimal-engine · connected") {
		t.Fatalf("status line %v", sat["statusLine"])
	}

	code, ov := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/overview", nil)
	store := ov["store"].(map[string]any)
	doctor := ov["doctor"].(map[string]any)
	if code != 200 || store["totalFiles"].(float64) != 3 || doctor["connected"] != true || doctor["healthScore"] != nil {
		t.Fatalf("overview %d %v", code, ov)
	}
	if doctor["status"] != "warn" || !strings.Contains(fmt.Sprint(doctor["checks"]), "not staged") || !strings.Contains(fmt.Sprint(doctor["checks"]), "engine order") {
		t.Fatalf("doctor warnings %v", doctor)
	}

	code, g := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/graph", nil)
	cons := g["constellation"].(map[string]any)
	if code != 200 || len(cons["nodes"].([]any)) != 5 || len(g["workspaces"].([]any)) != 2 {
		t.Fatalf("graph %d %v", code, g)
	}

	oe.mu.Lock()
	oe.down = true
	oe.mu.Unlock()
	fresh := brainTestEnv(t, oe)
	h2 := brainRouter(t, fresh)
	code, sat = brainCall(t, h2, http.MethodGet, "/api/founderos/pages/brain/satellites", nil)
	if code != 200 || sat["mounted"] != false || sat["freshPct"] != nil || !strings.Contains(sat["statusLine"].(string), "no notes mounted") {
		t.Fatalf("down engine must read unmounted, never an empty-but-fine brain: %v", sat)
	}
	code, ov = brainCall(t, h2, http.MethodGet, "/api/founderos/pages/brain/overview", nil)
	if code != 200 || ov["doctor"].(map[string]any)["status"] != "error" || ov["doctor"].(map[string]any)["connected"] != false {
		t.Fatalf("down overview %v", ov)
	}
}

func TestBrainQueryRetrievesThroughTheRouter(t *testing.T) {
	oe := newBrainFakeOE(t)
	h := brainRouter(t, brainTestEnv(t, oe))
	code, body := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/query?q=hermes", nil)
	if code != 200 || body["provider"] != "optimal-engine" || body["ranked"] != "provider" || body["query"] != "hermes" {
		t.Fatalf("query %d %v", code, body)
	}
	results := body["results"].([]any)
	first := results[0].(map[string]any)
	if len(results) != 2 || first["title"] != "Hermes worker pool" || first["snippet"] != "how the pool runs" || first["workspace"] != "founderos" {
		t.Fatalf("results %v", results)
	}
	if code, _ := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/query?q=%20", nil); code != 400 {
		t.Fatalf("blank query: %d", code)
	}
	oe.mu.Lock()
	oe.down = true
	oe.mu.Unlock()
	code, body = brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/query?q=hermes", nil)
	if code != http.StatusBadGateway || body["error"] == nil {
		t.Fatalf("an unreachable engine is an error, never zero hits: %d %v", code, body)
	}
}

func TestBrainDumpCapturesIntoTheHomeEngine(t *testing.T) {
	oe := newBrainFakeOE(t)
	h := brainRouter(t, brainTestEnv(t, oe))
	code, body := brainCall(t, h, http.MethodPost, "/api/founderos/pages/brain/dump", map[string]any{"text": "call Dana about the SOP", "folder": "inbox", "tags": []string{}})
	if code != 200 || body["ok"] != true || body["embedded"] != true || body["workspace"] != "founderos" || body["slug"] != "sig-1" || body["title"] != "call Dana about the SOP" {
		t.Fatalf("dump %d %v", code, body)
	}
	if !strings.HasPrefix(body["relPath"].(string), "inbox/2026-09-30-call-dana-about-the-sop") {
		t.Fatalf("relPath %v", body["relPath"])
	}
	oe.mu.Lock()
	got := oe.ingests
	oe.mu.Unlock()
	if len(got) != 1 || got[0]["workspace"] != "default:founderos" || got[0]["node"] != "inbox" || !strings.Contains(got[0]["text"].(string), "source: founderos-os-brain-dump") {
		t.Fatalf("ingests %v", got)
	}
	for _, bad := range []map[string]any{{"text": "", "folder": "inbox"}, {"text": "x", "folder": "../etc"}} {
		if code, _ := brainCall(t, h, http.MethodPost, "/api/founderos/pages/brain/dump", bad); code != 400 {
			t.Fatalf("bad dump %v: %d", bad, code)
		}
	}
	oe.mu.Lock()
	oe.down = true
	oe.mu.Unlock()
	code, body = brainCall(t, h, http.MethodPost, "/api/founderos/pages/brain/dump", map[string]any{"text": "x", "folder": "inbox"})
	if code != http.StatusBadGateway || body["embedded"] != false || body["error"] == nil {
		t.Fatalf("failed capture must say nothing was saved: %d %v", code, body)
	}
}

func TestBrainPageWithoutOrgIsUnavailable(t *testing.T) {
	oe := newBrainFakeOE(t)
	h := brainRouter(t, brainTestEnv(t, oe))
	code, body := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain", nil)
	if code != http.StatusServiceUnavailable || body["error"] == nil {
		t.Fatalf("no org: %d %v", code, body)
	}
}

func brainDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_brain_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

func TestBrainPageComposesTheOrgGraphFromPostgres(t *testing.T) {
	pool := brainDB(t)
	ctx := context.Background()
	var ws, other string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Other','other','u1') RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('dept-tech',$1,'TECH','tech','#fff',5),('dept-sales',$1,'Sales','sales','#fff',1)`,
		`INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier, tools, parent_id) VALUES ('conductor',$1,'dept-tech','Conductor','active','lead','["openclaw"]',NULL),('data-agent',$1,'dept-tech','Data Agent','active','worker','["openclaw","comms-feed"]','conductor'),('sales-agent',$1,'dept-sales','Sales Agent','idle','worker','["stripe"]',NULL),('aaa',$1,'dept-tech','Zed','idle','worker','[]',NULL)`,
		`INSERT INTO founderos_people (id, workspace_id, department_id, name, role, tools) VALUES ('person-len',$1,'dept-sales','Len','Closer','["fathom"]'),('a-person',$1,'dept-tech','Ann','Ops','[]')`,
		`INSERT INTO founderos_sop_tasks (id, workspace_id, department_id, title, steps, assignee_kind, assignee_id) VALUES ('sop-c',$1,'dept-tech','Run the board','["a","b","c"]','agent','conductor'),('sop-l',$1,'dept-sales','Close deals','["a","b","c"]','person','person-len')`,
		`INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok, summary) VALUES ('r1',$1,'conductor',now()-interval '2 hours',now()-interval '2 hours','false','old'),('r2',$1,'conductor',now()-interval '1 hour',now()-interval '1 hour','true','newest')`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	// another workspace's run never leaks in
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok, summary) VALUES ('r3',$1,'conductor',now(),now(),'true','other ws')`, other); err != nil {
		t.Fatal(err)
	}

	org, err := readBrainOrg(ctx, pool, ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(org.Departments) != 2 || org.Departments[0].ID != "dept-sales" || len(org.Agents) != 4 || len(org.People) != 2 || len(org.Tasks) != 2 {
		t.Fatalf("org = %+v", org)
	}
	// production's row order (lib/db.ts): agents by tier then name, people by
	// department then name, SOPs by department then title. The wheel seats
	// each pillar's tasks and workers in this order, so it has to match.
	ids := func(n int, at func(int) string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = at(i)
		}
		return out
	}
	if got := ids(len(org.Agents), func(i int) string { return org.Agents[i].ID }); fmt.Sprint(got) != "[conductor data-agent sales-agent aaa]" {
		t.Fatalf("agent order %v, want tier then name", got)
	}
	if got := ids(len(org.People), func(i int) string { return org.People[i].ID }); fmt.Sprint(got) != "[person-len a-person]" {
		t.Fatalf("people order %v, want department then name", got)
	}
	if got := ids(len(org.Tasks), func(i int) string { return org.Tasks[i].ID }); fmt.Sprint(got) != "[sop-l sop-c]" {
		t.Fatalf("SOP order %v, want department then title", got)
	}
	if org.Agents[1].ParentID == nil || *org.Agents[1].ParentID != "conductor" || len(org.Agents[1].Tools) != 2 || len(org.Tasks[0].Steps) != 3 {
		t.Fatalf("agent/task decode = %+v %+v", org.Agents[1], org.Tasks[0])
	}
	if r := org.Runs["conductor"]; r.Summary != "newest" || !r.OK {
		t.Fatalf("latest run per agent = %+v", org.Runs)
	}

	oe := newBrainFakeOE(t)
	env := brainTestEnv(t, oe)
	env.Org = func(ctx context.Context) (*brainOrg, error) { return readBrainOrg(ctx, pool, ws) }
	model := "claude-opus"
	env.Board = func(context.Context) ([]paperclip.Agent, error) {
		return []paperclip.Agent{{ID: "p1", Name: "Sales", Status: "idle", Model: &model}, {ID: "p2", Name: "Forge", Status: "running"}}, nil
	}
	h := brainRouter(t, env)
	code, body := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain", nil)
	if code != 200 {
		t.Fatalf("page %d %v", code, body)
	}
	g := body["graph"].(map[string]any)
	if len(g["nodes"].([]any)) == 0 || len(body["directory"].([]any)) != 4 {
		t.Fatalf("graph/directory %v", body)
	}
	leads := body["boardLeads"].(map[string]any)
	if leads["dept-sales"].(map[string]any)["id"] != "p1" || leads["dept-sales"].(map[string]any)["model"] != "claude-opus" {
		t.Fatalf("board leads %v", leads)
	}
	ba := body["boardAgents"].([]any)
	if len(ba) != 1 || ba[0].(map[string]any)["name"] != "Forge" {
		t.Fatalf("board agents (leads excluded) %v", ba)
	}
	if !strings.Contains(fmt.Sprint(g["nodes"]), "board:p2") || body["board"].(map[string]any)["state"] != "connected" {
		t.Fatalf("board ring %v", body["board"])
	}
	if body["runsByAgent"].(map[string]any)["conductor"].(map[string]any)["summary"] != "newest" {
		t.Fatalf("runs %v", body["runsByAgent"])
	}

	env.Board = func(context.Context) ([]paperclip.Agent, error) { return nil, errors.New("dial tcp: refused") }
	h = brainRouter(t, env)
	_, body = brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain", nil)
	if b := body["board"].(map[string]any); b["state"] != "error" || !strings.Contains(b["detail"].(string), "refused") || len(body["boardAgents"].([]any)) != 0 {
		t.Fatalf("an unreachable board is surfaced, not silently empty: %v", b)
	}
}

// One engine down is a partial answer, not a 502: the hits that answered plus
// the workspaces that could not be searched, named.
func TestBrainQueryWithOneEngineDownIsPartialAndSaysWhich(t *testing.T) {
	oe := newBrainFakeOE(t)
	env := brainTestEnv(t, oe)
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	t.Cleanup(dead.Close)
	device := brain.Engine{Name: "device", URL: dead.URL, Key: "k"}
	env.Engines = append(env.Engines, device)
	env.Workspaces = append(env.Workspaces, brainWorkspace{Slug: "personal", Engine: device})
	env.Unstaged = nil
	env.Memory = memory.New(env.Topo, map[string]memory.Endpoint{"local": {URL: oe.srv.URL, Key: "k"}, "device": {URL: dead.URL, Key: "k"}})
	env.Memory.Mode = memory.ModeSearch
	h := brainRouter(t, env)
	code, body := brainCall(t, h, http.MethodGet, "/api/founderos/pages/brain/query?q=hermes", nil)
	if code != 200 || len(body["results"].([]any)) == 0 {
		t.Fatalf("partial read must answer with what it found: %d %v", code, body)
	}
	if fmt.Sprint(body["failedWorkspaces"]) != "[personal]" || body["degraded"] == nil || body["degraded"] == "" {
		t.Fatalf("the unsearched workspace must be named: %v", body)
	}
}
