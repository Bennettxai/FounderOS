package brainimport

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// memEngine is an in-memory engine: workspaces, topology nodes, ingested
// contexts (title per workspace) and the graph read back.
type memEngine struct {
	mu         sync.Mutex
	workspaces map[string]bool
	nodes      map[string]map[string]any // slug → body
	contexts   map[string][]string       // workspace id → titles
	order      []string                  // "node:<slug>" / "ingest:<title>" in arrival order
}

func newMemEngine() *memEngine {
	return &memEngine{workspaces: map[string]bool{}, nodes: map[string]map[string]any{}, contexts: map[string][]string{}}
}

func (m *memEngine) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/workspaces":
			var ws []map[string]string
			for s := range m.workspaces {
				ws = append(ws, map[string]string{"slug": s})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"workspaces": ws})
		case r.Method == http.MethodPost && r.URL.Path == "/api/workspaces":
			m.workspaces[body["slug"].(string)] = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/nodes":
			slug, _ := body["slug"].(string)
			m.nodes[slug] = body
			m.order = append(m.order, "node:"+slug)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/ingest":
			ws, _ := body["workspace"].(string)
			title, _ := body["title"].(string)
			m.contexts[ws] = append(m.contexts[ws], title)
			m.order = append(m.order, "ingest:"+title)
			_, _ = w.Write([]byte(`{"ok":true}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/graph":
			var ctxs []map[string]string
			for i, title := range m.contexts[r.URL.Query().Get("workspace")] {
				ctxs = append(ctxs, map[string]string{"id": strings.Repeat("c", i+1), "title": title})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"contexts": ctxs, "edges": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWithoutLandedDropsPagesTheEngineAlreadyHolds(t *testing.T) {
	items, _, err := Plan(writeStore(t), topo(t))
	if err != nil {
		t.Fatal(err)
	}
	hub, mac := newMemEngine(), newMemEngine()
	hub.contexts["default:founderos"] = []string{"SOP", "Unrelated page"}
	hub.contexts["default:vantage"] = []string{"Call"} // same title, other workspace: not landed
	engines := map[string]Endpoint{"hub": {URL: hub.server(t).URL}, "macbook": {URL: mac.server(t).URL}}

	rest, landed, err := WithoutLanded(items, engines)
	if err != nil {
		t.Fatal(err)
	}
	if landed != 1 {
		t.Fatalf("landed = %d, want 1 (the founderos SOP page)", landed)
	}
	for _, it := range rest {
		if it.Title == "SOP" {
			t.Fatal("a page already in its workspace was kept")
		}
	}
	if len(rest) != len(items)-1 {
		t.Fatalf("kept %d of %d", len(rest), len(items))
	}
}

func TestWithoutLandedFailsWhenTheEngineCannotBeRead(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	_, _, err := WithoutLanded(items, map[string]Endpoint{"hub": {URL: "http://127.0.0.1:1"}, "macbook": {URL: "http://127.0.0.1:1"}})
	if err == nil {
		t.Fatal("an unreadable engine must fail the check, not re-send everything")
	}
}

func TestEnsureEntitiesRegistersEachPageSlugInItsWorkspace(t *testing.T) {
	items, _, _ := Plan(writeStore(t), topo(t))
	hub, mac := newMemEngine(), newMemEngine()
	engines := map[string]Endpoint{"hub": {URL: hub.server(t).URL}, "macbook": {URL: mac.server(t).URL}}

	if err := EnsureEntities(items, engines); err != nil {
		t.Fatal(err)
	}
	// "README" is a scaffold name, never an entity; the rest are page slugs.
	for slug, ws := range map[string]string{"onboarding": "default:founderos", "offer": "default:vantage", "2026-09-02-call": "default:founderos"} {
		n, ok := hub.nodes[slug]
		if !ok {
			t.Errorf("no entity node %q on the hub: %v", slug, hub.nodes)
			continue
		}
		if n["kind"] != "entity" || n["name"] != slug || n["workspace"] != ws {
			t.Errorf("node %q = %v", slug, n)
		}
	}
	if _, ok := mac.nodes["2026-09-01-chat"]; !ok {
		t.Errorf("personal page slug not registered on its own engine: %v", mac.nodes)
	}
	if _, ok := hub.nodes["readme"]; ok {
		t.Error("README registered as an entity")
	}
}
