package seed

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/brainimport"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

func TestMaterializeBrainWritesTheDemoKnowledgePages(t *testing.T) {
	dir := t.TempDir()
	n, err := MaterializeBrain(dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != BrainDocs {
		t.Fatalf("wrote %d pages, want %d", n, BrainDocs)
	}
	// v1's generator: one page per agent, SOP, tool, human and pillar.
	for _, rel := range []string{"agents/conductor.md", "sops/sop-conductor.md", "tools/stripe.md", "people/person-dana.md", "org/pillar-sales.md", "tools/optimal-engine.md"} {
		raw, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if !strings.Contains(string(raw), "generated: founder-os") {
			t.Errorf("%s lost v1's generated marker", rel)
		}
	}
	counts := map[string]int{}
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			counts[filepath.Dir(rel)]++
		}
		return nil
	})
	want := map[string]int{"agents": 32, "sops": 37, "tools": 34, "people": 5, "org": 6}
	for folder, w := range want {
		if counts[folder] != w {
			t.Errorf("%s/ has %d pages, want %d", folder, counts[folder], w)
		}
	}
}

// The fixtures are demo data: G-Brain is gone (the Optimal Engine is the
// brain now) and nothing points at a real machine, person or account.
func TestBrainFixturesCarryNoGBrainAndNoPrivateData(t *testing.T) {
	dir := t.TempDir()
	if _, err := MaterializeBrain(dir); err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`(?i)g-?brain|/Users/|~/|\.env\b|@gmail\.|clue-agent|bennett`)
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, _ := os.ReadFile(p)
		if m := banned.FindString(string(raw)); m != "" {
			t.Errorf("%s contains %q", p, m)
		}
		return nil
	})
}

// fakeEngine holds what was ingested per workspace and serves it back.
type fakeEngine struct {
	mu       sync.Mutex
	contexts map[string][]string
	nodes    map[string]bool
	ingests  int
}

func (f *fakeEngine) serve(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		switch r.URL.Path {
		case "/api/workspaces":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"workspaces":[]}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		case "/api/nodes":
			f.nodes[body["slug"].(string)] = true
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/ingest":
			if len(f.nodes) == 0 {
				t.Error("ingest before the entity gazetteer was registered")
			}
			ws := body["workspace"].(string)
			f.contexts[ws] = append(f.contexts[ws], body["title"].(string))
			f.ingests++
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/graph":
			var ctxs []map[string]string
			for _, title := range f.contexts[r.URL.Query().Get("workspace")] {
				ctxs = append(ctxs, map[string]string{"title": title})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"contexts": ctxs})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

const seedTopology = `
version: 1
engines:
  local: {tier: device, url_env: OPTIMAL_ENGINE_URL}
workspaces:
  - {slug: founderos, name: HQ, home: local, class: business}
  - {slug: personal, name: Personal, home: local, class: personal}
brain_store: {default: founderos}
`

func TestSeedBrainImportsOnceIntoHQ(t *testing.T) {
	topo, err := topology.Parse([]byte(seedTopology))
	if err != nil {
		t.Fatal(err)
	}
	eng := &fakeEngine{contexts: map[string][]string{}, nodes: map[string]bool{}}
	engines := map[string]brainimport.Endpoint{"local": {URL: eng.serve(t)}}

	first, err := SeedBrain(topo, engines)
	if err != nil {
		t.Fatal(err)
	}
	if first.Ingested == 0 || first.Ingested+first.Skipped != BrainDocs || len(first.Errors) != 0 {
		t.Fatalf("first run = %+v", first)
	}
	if got := len(eng.contexts["default:founderos"]); got != first.Ingested {
		t.Fatalf("HQ holds %d pages, want all %d in default:founderos (%v)", got, first.Ingested, eng.contexts)
	}
	if !eng.nodes["stripe"] || !eng.nodes["pillar-sales"] {
		t.Errorf("page slugs not registered as entities: %d nodes", len(eng.nodes))
	}

	again, err := SeedBrain(topo, engines)
	if err != nil {
		t.Fatal(err)
	}
	if again.Ingested != 0 || again.Landed != first.Ingested {
		t.Fatalf("re-run = %+v, want 0 ingested and %d already landed", again, first.Ingested)
	}
	if eng.ingests != first.Ingested {
		t.Fatalf("re-run sent %d more pages", eng.ingests-first.Ingested)
	}
}
