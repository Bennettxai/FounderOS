package tech

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/optimalengine"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// testTopo has two staged engines (hub, macbook) and one that is not staged
// on the test machine (mini), so every agent has to say what it could not see.
const testTopo = `
version: 1
engines:
  macbook: {tier: device}
  mini: {tier: device}
  hub: {tier: shared}
workspaces:
  - {slug: founderos, home: hub, class: business}
  - {slug: vantage, home: hub, class: business}
  - {slug: personal, home: macbook, class: personal}
  - {slug: hermes, home: mini, class: device}
brain_store: {default: founderos, routes: {}}
retired: {default: founderos}
`

type fakeSignal struct {
	Title, URI, Content, Modified string
}

type fakeCheck struct {
	Name   string
	OK     bool
	Detail any
}

// fakeEngine serves the slice of the Optimal Engine HTTP API the tech agents
// read. Everything is local (httptest), never the network.
type fakeEngine struct {
	mu         sync.Mutex
	key        string
	down       bool
	health     string
	workspaces []string // slugs active on the engine
	counts     map[string]int
	checks     []fakeCheck
	signals    map[string][]fakeSignal
	pending    map[string]int
	failExport map[string]bool
	paths      []string
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{
		key:    "k",
		health: "up",
		counts: map[string]int{},
		checks: []fakeCheck{
			{"sqlite_integrity", true, "ok"},
			{"fts_parity", true, map[string]int{"contexts": 3, "indexed": 3}},
			{"vector_integrity", true, "dimensions and references valid"},
			{"workspace_scope", true, "no legacy-default rows"},
		},
		signals: map[string][]fakeSignal{},
		pending: map[string]int{},
	}
}

func (f *fakeEngine) serve(t *testing.T) string {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.paths = append(f.paths, r.URL.Path)
		if f.down {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+f.key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("tech agents must only read the engine, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ws := strings.TrimPrefix(r.URL.Query().Get("workspace"), "default:")
		switch r.URL.Path {
		case "/api/health":
			writeJSON(w, http.StatusOK, map[string]any{"status": f.health})
		case "/api/workspaces":
			var list []map[string]string
			for _, s := range f.workspaces {
				list = append(list, map[string]string{"id": "default:" + s, "slug": s, "status": "active"})
			}
			writeJSON(w, http.StatusOK, map[string]any{"workspaces": list})
		case "/api/stores":
			rel, vec, graph := map[string]int{}, map[string]int{}, map[string]int{}
			for k, v := range f.counts {
				switch k {
				case "vectors", "chunk_embeddings":
					vec[k] = v
				case "edges":
					graph[k] = v
				default:
					rel[k] = v
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{"stores": []map[string]any{
				{"id": "relational", "status": "available", "table_counts": rel},
				{"id": "vector", "status": "available", "table_counts": vec},
				{"id": "graph", "status": "available", "table_counts": graph},
			}})
		case "/api/stores/audit":
			ok, failures := true, 0
			var checks []map[string]any
			for _, c := range f.checks {
				if !c.OK {
					ok = false
					failures++
				}
				checks = append(checks, map[string]any{"name": c.Name, "ok": c.OK, "detail": c.Detail})
			}
			status := http.StatusOK
			if !ok {
				status = http.StatusServiceUnavailable // the engine's own convention
			}
			writeJSON(w, status, map[string]any{"ok": ok, "checks": checks, "failures": failures})
		case "/api/batch/export/signals":
			if f.failExport[ws] {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			out := []map[string]string{}
			for i, s := range f.signals[ws] {
				out = append(out, map[string]string{"id": fmt.Sprint(i), "title": s.Title, "uri": s.URI, "content": s.Content,
					"modified_at": s.Modified, "created_at": s.Modified, "workspace_id": "default:" + ws})
			}
			writeJSON(w, http.StatusOK, map[string]any{"workspace_id": "default:" + ws, "signals": out})
		case "/api/memory-core/claims":
			writeJSON(w, http.StatusOK, map[string]any{"count": f.pending[ws], "claims": []any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.Close)
	return s.URL
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// source builds an EngineSource over the test topology with the given
// staged engines (name → fake).
func source(t *testing.T, staged map[string]*fakeEngine) EngineSource {
	t.Helper()
	topo, err := topology.Parse([]byte(testTopo))
	if err != nil {
		t.Fatal(err)
	}
	var engines []optimalengine.Engine
	for _, name := range []string{"hub", "macbook"} {
		if f, ok := staged[name]; ok {
			engines = append(engines, optimalengine.Engine{Name: name, URL: f.serve(t), Key: f.key})
		}
	}
	return func() (*topology.Topology, []optimalengine.Engine, error) { return topo, engines, nil }
}

func failingSource(err error) EngineSource {
	return func() (*topology.Topology, []optimalengine.Engine, error) { return nil, nil, err }
}

func noEngines(t *testing.T) EngineSource { return source(t, nil) }
