package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// FounderOS v1 lib/brain-retrieval.ts: every brain read takes a pool of 15,
// reranks it with a local cross-encoder when one answers, and keeps the top 3.
// FounderOS v1 drew its pool from one store; the bridge's memory is split
// across workspaces, so each workspace contributes its own pool and the
// reranker scores the union.
const (
	RetrievePool = 8 // per workspace
	RetrieveTop  = 3
	rerankChars  = 1200
)

// Reranker scores documents for a query; ok=false means "no opinion" and the
// engine's own order stands.
type Reranker func(ctx context.Context, query string, docs []string) (scores []float64, ok bool)

type RetrieveOptions struct {
	Pool   int // candidates per workspace
	Top    int
	Rerank Reranker // nil: DefaultReranker()
}

type RetrievedHit struct {
	Hit
	RerankScore *float64 `json:"rerankScore,omitempty"`
}

type Retrieval struct {
	Hits   []RetrievedHit `json:"hits"`
	Ranked string         `json:"ranked"` // "rerank" | "provider"
	Error  string         `json:"error,omitempty"`
	// FailedWorkspaces were not searched (engine down) while others answered;
	// Degraded says why. Both are empty for a complete read.
	FailedWorkspaces []string `json:"failedWorkspaces,omitempty"`
	Degraded         string   `json:"degraded,omitempty"`
}

var (
	defaultRRMu  sync.Mutex
	defaultRR    Reranker
	defaultRRKey string
)

// DefaultReranker targets the bridge's llama-server reranker (RERANK_BASE_URL,
// default :8082, scripts/bridge/rerank-up.sh), behind the canary. It is built
// once per configuration so the canary verdict holds across reads.
// BRAIN_RERANK=0 turns it off.
func DefaultReranker() Reranker {
	if os.Getenv("BRAIN_RERANK") == "0" {
		return nil
	}
	base := os.Getenv("RERANK_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8082/v1"
	}
	model := os.Getenv("RERANK_MODEL")
	if model == "" {
		model = "qwen3-reranker-0.6b"
	}
	key := base + "\x00" + model + "\x00" + os.Getenv("RERANK_API_KEY")
	defaultRRMu.Lock()
	defer defaultRRMu.Unlock()
	if defaultRR == nil || defaultRRKey != key {
		defaultRR, defaultRRKey = Canaried(HTTPReranker(base, model, os.Getenv("RERANK_API_KEY")), time.Now), key
	}
	return defaultRR
}

// The canary a reranker must pass before its scores are used: the relevant
// document has to outscore both unrelated ones.
var (
	canaryQuery = "who is Jordan Lee"
	canaryDocs  = []string{
		"Jordan Lee is the operator's business partner and runs the local sales dashboard. kind: person",
		"The weather in Miami is hot in August.",
		"Recipe: banana bread with walnuts.",
	}
)

const canaryTTL = 10 * time.Minute

// Canaried trusts r only while it passes the canary (re-checked every
// canaryTTL). A reranker that ranks the weather above the page about the
// person asked for is noise, and noise must not reorder the engine's hits:
// failing the canary is "no opinion". A reranker that cannot answer the
// canary at all is also no opinion.
func Canaried(r Reranker, now func() time.Time) Reranker {
	if r == nil {
		return nil
	}
	var (
		mu      sync.Mutex
		checked time.Time
		passed  bool
	)
	return func(ctx context.Context, query string, docs []string) ([]float64, bool) {
		mu.Lock()
		if checked.IsZero() || now().Sub(checked) > canaryTTL {
			s, ok := r(ctx, canaryQuery, canaryDocs)
			passed = ok && len(s) == len(canaryDocs) && s[0] > s[1] && s[0] > s[2]
			checked = now()
		}
		pass := passed
		mu.Unlock()
		if !pass {
			return nil, false
		}
		return r(ctx, query, docs)
	}
}

// HTTPReranker speaks the llama.cpp / ZeroEntropy /rerank dialect and fails
// soft: any error is "no opinion", never a failed read.
func HTTPReranker(base, model, apiKey string) Reranker {
	client := &http.Client{Timeout: 6 * time.Second}
	url := strings.TrimRight(base, "/") + "/rerank"
	return func(ctx context.Context, query string, docs []string) ([]float64, bool) {
		if len(docs) < 2 || strings.TrimSpace(query) == "" {
			return nil, false
		}
		clipped := make([]string, len(docs))
		for i, d := range docs {
			clipped[i] = clip(d, rerankChars)
		}
		body, _ := json.Marshal(map[string]any{"model": model, "query": clip(query, rerankChars), "documents": clipped})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, false
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, false
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, false
		}
		var raw bytes.Buffer
		_, _ = raw.ReadFrom(resp.Body)
		return parseRerank(raw.Bytes(), len(docs))
	}
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func parseRerank(raw []byte, expected int) ([]float64, bool) {
	var body struct {
		Results []struct {
			Index *int     `json:"index"`
			Score *float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if json.Unmarshal(raw, &body) != nil || len(body.Results) != expected {
		return nil, false
	}
	scores := make([]float64, expected)
	seen := make([]bool, expected)
	for _, r := range body.Results {
		if r.Index == nil || r.Score == nil || *r.Index < 0 || *r.Index >= expected || seen[*r.Index] {
			return nil, false
		}
		seen[*r.Index] = true
		scores[*r.Index] = *r.Score
	}
	return scores, true
}

// Retrieve is the brain read path: fan-out search, pool, rerank, top.
func (r *Router) Retrieve(ctx context.Context, query string, workspaces []string, o RetrieveOptions) Retrieval {
	if strings.TrimSpace(query) == "" {
		return Retrieval{Ranked: "provider"}
	}
	if o.Pool <= 0 {
		o.Pool = RetrievePool
	}
	if o.Top <= 0 {
		o.Top = RetrieveTop
	}
	found, err := r.searchEach(ctx, query, workspaces, o.Pool)
	var partial *PartialError
	if err != nil && !errors.As(err, &partial) {
		return Retrieval{Ranked: "provider", Error: err.Error()}
	}
	rerank := o.Rerank
	if rerank == nil {
		rerank = DefaultReranker()
	}
	var scores []float64
	ok := false
	if rerank != nil && len(found) > 1 {
		docs := make([]string, len(found))
		for i, h := range found {
			docs[i] = h.Title + "\n" + h.Abstract
		}
		scores, ok = rerank(ctx, query, docs)
	}
	out := make([]RetrievedHit, len(found))
	for i, h := range found {
		out[i] = RetrievedHit{Hit: h}
		if ok {
			s := scores[i]
			out[i].RerankScore = &s
		}
	}
	ranked := "provider"
	if ok {
		ranked = "rerank"
		sort.SliceStable(out, func(i, j int) bool { return *out[i].RerankScore > *out[j].RerankScore })
	}
	out = dedupePages(out)
	if len(out) > o.Top {
		out = out[:o.Top]
	}
	res := Retrieval{Hits: out, Ranked: ranked}
	if partial != nil {
		res.FailedWorkspaces, res.Degraded = partial.Failed, partial.Error()
	}
	return res
}

// dedupePages keeps each page once, at its best position: an engine returns
// several chunks of one document as separate hits (same URI), and the top 3
// should be three pages (lib/brain-federated.ts dedupes by title). The key is
// URI plus title: grep-mode hits share a folder-level URI across different
// sections, which their titles (headings) tell apart.
func dedupePages(hits []RetrievedHit) []RetrievedHit {
	seen := map[string]bool{}
	out := hits[:0:0]
	for _, h := range hits {
		key := h.Workspace + "\x00" + h.URI + "\x00" + strings.ToLower(strings.TrimSpace(h.Title))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, h)
	}
	return out
}

// RerankerState probes a reranker for the doctor: "unreachable" when its
// server does not answer, "failing" when it answers but fails the canary
// (its scores are noise, so reads keep the engine's order), "passing" when
// it ranks the obvious document first.
func RerankerState(ctx context.Context, base, model, apiKey string) string {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	root := strings.TrimSuffix(strings.TrimRight(base, "/"), "/v1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/health", nil)
	if err != nil {
		return "unreachable"
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return "unreachable"
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "unreachable"
	}
	s, ok := HTTPReranker(base, model, apiKey)(ctx, canaryQuery, canaryDocs)
	switch {
	case !ok:
		return "unreachable"
	case len(s) == len(canaryDocs) && s[0] > s[1] && s[0] > s[2]:
		return "passing"
	}
	return "failing"
}
