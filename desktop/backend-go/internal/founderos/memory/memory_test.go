package memory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

const topoYAML = `
version: 1
engines:
  macbook: {tier: device}
  hub: {tier: shared}
workspaces:
  - {slug: founderos, home: hub, class: business}
  - {slug: vantage, home: hub, class: business}
  - {slug: personal, home: macbook, class: personal}
brain_store: {default: founderos, routes: {}}
retired: {default: founderos}
`

type engine struct {
	mu      sync.Mutex
	ingests []map[string]any
	down    bool
	results map[string]string // workspace id → /api/search results JSON
	grep    map[string]string // workspace id → /api/grep results JSON
}

func (e *engine) srv(t *testing.T, key string) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e.down {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/ingest":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			e.mu.Lock()
			e.ingests = append(e.ingests, body)
			e.mu.Unlock()
			_, _ = w.Write([]byte(`{"ok":true,"signal_id":"sig-1"}`))
		case "/api/search":
			_, _ = w.Write([]byte(`{"results":` + orEmpty(e.results[r.URL.Query().Get("workspace")]) + `}`))
		case "/api/grep":
			_, _ = w.Write([]byte(`{"results":` + orEmpty(e.grep[r.URL.Query().Get("workspace")]) + `}`))
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func newRouter(t *testing.T, hub, mac *engine) *Router {
	topo, err := topology.Parse([]byte(topoYAML))
	if err != nil {
		t.Fatal(err)
	}
	h, m := hub.srv(t, "hk"), mac.srv(t, "mk")
	r := New(topo, map[string]Endpoint{"hub": {URL: h.URL, Key: "hk"}, "macbook": {URL: m.URL, Key: "mk"}})
	r.Mode = ModeSearch // the fixtures speak /api/search unless a test says otherwise
	return r
}

func TestCaptureGoesToTheWorkspaceHomeEngine(t *testing.T) {
	hub, mac := &engine{}, &engine{}
	r := newRouter(t, hub, mac)
	id, err := r.Capture(context.Background(), Capture{Workspace: "personal", Title: "Journal", Text: "felt good", Genre: "note"})
	if err != nil || id != "sig-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if len(mac.ingests) != 1 || len(hub.ingests) != 0 || mac.ingests[0]["workspace"] != "default:personal" || mac.ingests[0]["extract_claims"] != true {
		t.Fatalf("mac=%v hub=%v", mac.ingests, hub.ingests)
	}
	if _, err := r.Capture(context.Background(), Capture{Workspace: "default", Text: "legacy"}); err != nil || len(hub.ingests) != 1 || hub.ingests[0]["workspace"] != "default:founderos" {
		t.Fatalf("retired workspace must merge into founderos: %v %v", err, hub.ingests)
	}
	if _, err := r.Capture(context.Background(), Capture{Workspace: "ghost", Text: "x"}); err == nil {
		t.Fatal("unknown workspace must fail, never write somewhere else")
	}
	if _, err := r.Capture(context.Background(), Capture{Workspace: "personal", Text: "  "}); err == nil {
		t.Fatal("empty text must fail")
	}
}

func TestSearchFansOutAndInterleavesByRank(t *testing.T) {
	hub := &engine{results: map[string]string{
		"default:founderos": `[{"id":"f1","title":"SOP onboarding","uri":"u1"},{"id":"f2","title":"Tools","uri":"u2"}]`,
		"default:vantage":   `[{"id":"m1","title":"Morgan x Vantage","uri":"u3"}]`,
	}}
	mac := &engine{results: map[string]string{"default:personal": `[{"id":"p1","title":"Journal","uri":"u4"}]`}}
	r := newRouter(t, hub, mac)
	hits, err := r.Search(context.Background(), "onboarding", []string{"founderos", "vantage", "personal"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, h := range hits {
		got = append(got, h.Workspace+"/"+h.ID)
	}
	if strings.Join(got, ",") != "founderos/f1,vantage/m1,personal/p1" {
		t.Fatalf("hits = %v (rank-1 of each workspace first, capped at 3)", got)
	}
}

func TestSearchNeverReadsAnUnreachableEngineAsEmpty(t *testing.T) {
	hub, mac := &engine{down: true}, &engine{results: map[string]string{"default:personal": `[]`}}
	r := newRouter(t, hub, mac)
	_, err := r.Search(context.Background(), "q", []string{"founderos", "personal"}, 5)
	var un *UnreachableError
	if !errors.As(err, &un) || un.Engine != "hub" {
		t.Fatalf("err = %v, want UnreachableError for hub", err)
	}
	// Searching only healthy workspaces still works.
	if hits, err := r.Search(context.Background(), "q", []string{"personal"}, 5); err != nil || len(hits) != 0 {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
}

// Grep mode is how FounderOS v1 reads the engine (lib/connectors/optimal.ts):
// /api/grep returns the matching text and a relevance score, where /api/search
// returns only an L0 one-liner. The title comes from the snippet's heading or
// front-matter title, else the slug; the snippet is whitespace-collapsed and
// capped at 400 characters.
func TestGrepModeReadsLikeFounderosOS(t *testing.T) {
	long := strings.Repeat("word ", 120)
	hub := &engine{grep: map[string]string{
		"default:founderos": `[{"slug":"knowledge-base","score":0.8635,"snippet":"---\ntitle: Jordan Lee\nkind: person\n---\n\n# Jordan Lee"},` +
			`{"slug":"tech","score":0.8083,"snippet":"probe   paperclip\n:3100 ` + long + `"}]`,
		"default:vantage": `[{"slug":"morgan","score":0.7,"snippet":"## Morgan x Vantage\nnotes"}]`,
	}}
	r := newRouter(t, hub, &engine{})
	r.Mode = ModeGrep
	hits, err := r.Search(context.Background(), "jordan", []string{"founderos", "vantage"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("hits = %+v", hits)
	}
	g := hits[0]
	if g.Title != "Jordan Lee" || g.ID != "knowledge-base" || g.Workspace != "founderos" || g.Score == nil || *g.Score != 0.8635 || g.URI != "optimal://founderos/knowledge-base" {
		t.Fatalf("jordan hit = %+v", g)
	}
	if hits[1].Title != "Morgan x Vantage" || hits[1].Workspace != "vantage" {
		t.Fatalf("rank-1 of each workspace first: %+v", hits[1])
	}
	if hits[2].Title != "tech" || len([]rune(hits[2].Abstract)) != 400 || strings.Contains(hits[2].Abstract, "  ") || strings.Contains(hits[2].Abstract, "\n") {
		t.Fatalf("slug title, collapsed 400-char snippet: %q (%d)", hits[2].Title, len([]rune(hits[2].Abstract)))
	}
}

// Search stays the default: it keeps page identity, and on the parity misses
// grep did no better for natural-language questions ("who is Jordan Lee" finds
// the same meetings; only the bare keyword finds the person page). Grep is
// FounderOS v1's read and stays selectable.
func TestModeFromEnvDefaultsToSearch(t *testing.T) {
	t.Setenv("FOUNDEROS_MEMORY_SEARCH", "")
	if ModeFromEnv() != ModeSearch {
		t.Fatal("default must be search")
	}
	t.Setenv("FOUNDEROS_MEMORY_SEARCH", "grep")
	if ModeFromEnv() != ModeGrep {
		t.Fatal("grep must stay selectable")
	}
}

func orEmpty(s string) string {
	if s == "" {
		return "[]"
	}
	return s
}

// Hybrid mode pools both reads per workspace: /api/search keeps page
// identity, /api/grep brings the matching text, so the reranker finally
// judges content rather than one-line abstracts.
func TestHybridModePoolsSearchAndGrep(t *testing.T) {
	hub := &engine{
		results: map[string]string{"default:founderos": `[{"id":"m1","title":"sales team meeting","uri":"u/meeting.md"}]`},
		grep:    map[string]string{"default:founderos": `[{"slug":"knowledge-base","score":0.86,"snippet":"---\ntitle: Jordan Lee\nkind: person\n---"}]`},
	}
	r := newRouter(t, hub, &engine{})
	r.Mode = ModeHybrid
	hits, err := r.Search(context.Background(), "jordan", []string{"founderos"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, h := range hits {
		titles = append(titles, h.Title)
	}
	if strings.Join(titles, "|") != "sales team meeting|Jordan Lee" {
		t.Fatalf("hybrid pool = %v", titles)
	}
	t.Setenv("FOUNDEROS_MEMORY_SEARCH", "hybrid")
	if ModeFromEnv() != ModeHybrid {
		t.Fatal("hybrid must be selectable")
	}
}
