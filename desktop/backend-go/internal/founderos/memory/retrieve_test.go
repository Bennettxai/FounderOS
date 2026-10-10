package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRerankResponse(t *testing.T) {
	ok := []byte(`{"results":[{"index":1,"relevance_score":0.9},{"index":0,"relevance_score":0.1}]}`)
	if s, good := parseRerank(ok, 2); !good || s[0] != 0.1 || s[1] != 0.9 {
		t.Fatalf("scores = %v", s)
	}
	for _, bad := range []string{
		`{"results":[{"index":0,"relevance_score":0.1}]}`,                                   // wrong count
		`{"results":[{"index":0,"relevance_score":0.1},{"index":0,"relevance_score":0.2}]}`, // duplicate index
		`{"results":[{"index":0,"relevance_score":0.1},{"index":5,"relevance_score":0.2}]}`, // out of range
		`{"results":[{"index":0,"relevance_score":"x"},{"index":1,"relevance_score":0.2}]}`, // not a number
		`nope`,
	} {
		if _, good := parseRerank([]byte(bad), 2); good {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestRetrieveReranksThePoolAndFallsBackSoftly(t *testing.T) {
	hub := &engine{results: map[string]string{
		"default:founderos": `[{"id":"a","title":"sales team meeting"},{"id":"b","title":"Taylor Brooks"},{"id":"c","title":"Jordan Lee"}]`,
	}}
	mac := &engine{}
	r := newRouter(t, hub, mac)

	rerankSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Documents []string `json:"documents"`
		}
		_ = json.NewDecoder(req.Body).Decode(&body)
		// Score the person page highest.
		out := `{"results":[`
		for i, d := range body.Documents {
			score := 0.1
			if d[:7] == "Jordan " {
				score = 0.99
			}
			if i > 0 {
				out += ","
			}
			out += `{"index":` + string(rune('0'+i)) + `,"relevance_score":` + map[bool]string{true: "0.99", false: "0.1"}[score > 0.5] + `}`
		}
		_, _ = w.Write([]byte(out + `]}`))
	}))
	defer rerankSrv.Close()

	got := r.Retrieve(context.Background(), "who is Jordan Lee", []string{"founderos"}, RetrieveOptions{Top: 2, Rerank: HTTPReranker(rerankSrv.URL, "m", "")})
	if got.Ranked != "rerank" || len(got.Hits) != 2 || got.Hits[0].Title != "Jordan Lee" || got.Hits[0].RerankScore == nil {
		t.Fatalf("reranked = %+v", got)
	}

	down := r.Retrieve(context.Background(), "who is Jordan Lee", []string{"founderos"}, RetrieveOptions{Top: 2, Rerank: HTTPReranker("http://127.0.0.1:1", "m", "")})
	if down.Ranked != "provider" || len(down.Hits) != 2 || down.Hits[0].Title != "sales team meeting" || down.Error != "" {
		t.Fatalf("a dead reranker must fall back to engine order: %+v", down)
	}

	hub.down = true
	failed := r.Retrieve(context.Background(), "q", []string{"founderos"}, RetrieveOptions{})
	if failed.Error == "" || len(failed.Hits) != 0 {
		t.Fatalf("an unreachable engine must surface as an error: %+v", failed)
	}
}

func TestRetrievePoolsEachWorkspaceBeforeReranking(t *testing.T) {
	// founderos's answer sits at rank 4; with a shared pool of 3 across three
	// workspaces it would never reach the reranker.
	hub := &engine{results: map[string]string{
		"default:founderos": `[{"id":"f1","title":"meeting 1"},{"id":"f2","title":"meeting 2"},{"id":"f3","title":"meeting 3"},{"id":"f4","title":"Jordan Lee"}]`,
		"default:vantage":   `[{"id":"m1","title":"vantage call"}]`,
	}}
	mac := &engine{results: map[string]string{"default:personal": `[{"id":"p1","title":"journal"}]`}}
	r := newRouter(t, hub, mac)
	var seen int
	rerank := func(_ context.Context, q string, docs []string) ([]float64, bool) {
		seen = len(docs)
		scores := make([]float64, len(docs))
		for i, d := range docs {
			if d[:7] == "Jordan " {
				scores[i] = 1
			}
		}
		return scores, true
	}
	got := r.Retrieve(context.Background(), "who is Jordan Lee", []string{"founderos", "vantage", "personal"}, RetrieveOptions{Pool: 4, Top: 1, Rerank: rerank})
	if seen != 6 || len(got.Hits) != 1 || got.Hits[0].Title != "Jordan Lee" {
		t.Fatalf("reranker saw %d docs, top = %+v", seen, got.Hits)
	}
}

// Two chunks of one page come back as two hits with the same URI; the top 3
// holds each page once (lib/brain-federated.ts dedupes by title), keeping its
// best-ranked copy.
func TestRetrieveKeepsEachPageOnce(t *testing.T) {
	hub := &engine{results: map[string]string{
		"default:founderos": `[{"id":"a1","title":"sales team meeting","uri":"u/meeting.md"},{"id":"a2","title":"sales team meeting","uri":"u/meeting.md"},{"id":"b","title":"Taylor Brooks","uri":"u/taylor.md"},{"id":"c","title":"Jordan Lee","uri":"u/jordan.md"}]`,
	}}
	r := newRouter(t, hub, &engine{})
	noRerank := func(context.Context, string, []string) ([]float64, bool) { return nil, false }
	got := r.Retrieve(context.Background(), "q", []string{"founderos"}, RetrieveOptions{Top: 3, Rerank: noRerank})
	var titles []string
	for _, h := range got.Hits {
		titles = append(titles, h.Title)
	}
	if strings.Join(titles, "|") != "sales team meeting|Taylor Brooks|Jordan Lee" {
		t.Fatalf("top 3 = %v", titles)
	}
}

// Grep hits share a folder-level URI; different sections (titles) of it are
// different answers and all stay.
func TestRetrieveKeepsDistinctSectionsOfOneFolder(t *testing.T) {
	hub := &engine{grep: map[string]string{
		"default:founderos": `[{"slug":"knowledge-base","score":0.9,"snippet":"# Jordan Lee\nperson"},{"slug":"knowledge-base","score":0.8,"snippet":"# Sales dashboard\nJordan's"}]`,
	}}
	r := newRouter(t, hub, &engine{})
	r.Mode = ModeGrep
	noRerank := func(context.Context, string, []string) ([]float64, bool) { return nil, false }
	if got := r.Retrieve(context.Background(), "jordan", []string{"founderos"}, RetrieveOptions{Top: 3, Rerank: noRerank}); len(got.Hits) != 2 {
		t.Fatalf("sections collapsed: %+v", got.Hits)
	}
}

// A reranker is trusted only after it passes a canary: an obviously relevant
// document must outscore two unrelated ones. The local Qwen3 GGUF scored
// "the weather in Miami" above a page about Jordan Lee for "who is Jordan Lee
// " (all scores ~1e-8): a model that fails the canary has no opinion,
// so the engine's order stands. The verdict is cached.
func TestRerankerMustPassTheCanary(t *testing.T) {
	calls := 0
	broken := Reranker(func(_ context.Context, q string, docs []string) ([]float64, bool) {
		calls++
		out := make([]float64, len(docs))
		for i := range out {
			out[i] = float64(i) * 1e-9 // prefers whatever comes last
		}
		return out, true
	})
	now := time.Date(2026, 9, 30, 4, 0, 0, 0, time.UTC)
	c := Canaried(broken, func() time.Time { return now })
	if _, ok := c(context.Background(), "q", []string{"a", "b"}); ok {
		t.Fatal("a reranker that fails the canary must have no opinion")
	}
	if _, ok := c(context.Background(), "q", []string{"a", "b"}); ok || calls != 1 {
		t.Fatalf("the failed verdict is cached (canary calls = %d)", calls)
	}

	sensible := Reranker(func(_ context.Context, q string, docs []string) ([]float64, bool) {
		out := make([]float64, len(docs))
		for i, d := range docs {
			if strings.Contains(strings.ToLower(d), "jordan") || d == "a" {
				out[i] = 0.9
			}
		}
		return out, true
	})
	g := Canaried(sensible, func() time.Time { return now })
	if s, ok := g(context.Background(), "q", []string{"a", "z"}); !ok || s[0] != 0.9 {
		t.Fatalf("a sensible reranker passes and scores: %v %v", s, ok)
	}
}

// DefaultReranker is built once per configuration, so the canary verdict is
// cached across reads instead of re-run before every query.
func TestDefaultRerankerIsSharedPerConfig(t *testing.T) {
	var canaries int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Query == canaryQuery {
			atomic.AddInt32(&canaries, 1)
		}
		out := `{"results":[`
		for i := range body.Documents {
			if i > 0 {
				out += ","
			}
			score := "0.0"
			if i == 0 {
				score = "0.99"
			}
			out += fmt.Sprintf(`{"index":%d,"relevance_score":%s}`, i, score)
		}
		_, _ = w.Write([]byte(out + "]}"))
	}))
	defer srv.Close()
	t.Setenv("BRAIN_RERANK", "")
	t.Setenv("RERANK_BASE_URL", srv.URL)
	t.Setenv("RERANK_MODEL", "m-"+t.Name())
	for i := 0; i < 3; i++ {
		if _, ok := DefaultReranker()(context.Background(), "q", []string{"a", "b"}); !ok {
			t.Fatal("a sensible reranker must pass")
		}
	}
	if n := atomic.LoadInt32(&canaries); n != 1 {
		t.Fatalf("canary ran %d times across 3 reads, want 1", n)
	}
	t.Setenv("BRAIN_RERANK", "0")
	if DefaultReranker() != nil {
		t.Fatal("BRAIN_RERANK=0 turns it off")
	}
}

func TestRerankerStateTellsPassingFromNoise(t *testing.T) {
	serve := func(good bool) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				_, _ = w.Write([]byte(`{"status":"ok"}`))
				return
			}
			first := "0.99"
			if !good {
				first = "0.00000001" // the broken GGUF: everything ~1e-8, the weather wins
			}
			_, _ = w.Write([]byte(`{"results":[{"index":0,"relevance_score":` + first + `},{"index":1,"relevance_score":0.0000002},{"index":2,"relevance_score":0.0}]}`))
		}))
	}
	good, bad := serve(true), serve(false)
	defer good.Close()
	defer bad.Close()
	if got := RerankerState(context.Background(), good.URL+"/v1", "m", ""); got != "passing" {
		t.Fatalf("good = %q", got)
	}
	if got := RerankerState(context.Background(), bad.URL+"/v1", "m", ""); got != "failing" {
		t.Fatalf("bad = %q", got)
	}
	if got := RerankerState(context.Background(), "http://127.0.0.1:1/v1", "m", ""); got != "unreachable" {
		t.Fatalf("down = %q", got)
	}
}
