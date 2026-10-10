// Package memory is the operator's memory layer on the bridge, the replacement
// for GBrain (lib/memory-provider.ts + lib/brain-retrieval.ts). Every write
// goes to the home engine of its workspace (engine topology); searches fan
// out across workspaces. An unreachable engine is an error, never an empty
// brain; with one engine down and another answering, a search is partial
// (PartialError names the workspaces it could not read).
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// Tenant is the Optimal Engine tenant; engine workspace ids are "<tenant>:<slug>".
const Tenant = "default"

type Endpoint struct {
	URL string
	Key string
}

type Router struct {
	topo      *topology.Topology
	endpoints map[string]Endpoint
	client    *http.Client
	// Mode picks the engine read: ModeGrep (FounderOS v1's choice) or ModeSearch.
	Mode Mode
}

// Mode is how the router reads an engine.
type Mode string

const (
	// ModeGrep reads /api/grep: the matching text plus a relevance score.
	// FounderOS v1 reads the engine this way (lib/connectors/optimal.ts).
	ModeGrep Mode = "grep"
	// ModeSearch reads /api/search: page identity, but only an L0 one-liner.
	ModeSearch Mode = "search"
	// ModeHybrid pools both per workspace: search's page identity plus
	// grep's matching text, so the reranker judges content. Opt-in.
	ModeHybrid Mode = "hybrid"
)

// ModeFromEnv reads FOUNDEROS_MEMORY_SEARCH ("search" default, or "grep").
func ModeFromEnv() Mode {
	switch m := Mode(strings.TrimSpace(os.Getenv("FOUNDEROS_MEMORY_SEARCH"))); m {
	case ModeGrep, ModeHybrid:
		return m
	}
	return ModeSearch
}

func New(topo *topology.Topology, endpoints map[string]Endpoint) *Router {
	// Engine calls are local/tailnet ingestion, the bridge's own memory: the
	// guard's transport lets loopback through and would refuse only a remote
	// engine while writes are off, which is the right behaviour pre-cutover.
	return &Router{topo: topo, endpoints: endpoints, client: connectors.HTTPClient(60 * time.Second), Mode: ModeFromEnv()}
}

// UnreachableError marks an engine that could not answer.
type UnreachableError struct {
	Engine string
	Err    error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("engine %s unreachable: %v", e.Engine, e.Err)
}
func (e *UnreachableError) Unwrap() error { return e.Err }

// PartialError means some searched workspaces could not be read (their
// engine did not answer) while at least one did: the hits returned alongside
// it are real but incomplete. When no workspace answers, the search fails
// with a plain error instead, so an unreachable brain never reads as empty.
type PartialError struct {
	Failed []string // workspaces that were not searched, in request order
	Errs   []error  // why, one per Failed entry
}

func (e *PartialError) Error() string {
	parts := make([]string, len(e.Failed))
	for i, w := range e.Failed {
		parts[i] = w + " (" + e.Errs[i].Error() + ")"
	}
	return "partial results, not searched: " + strings.Join(parts, "; ")
}

func (e *PartialError) Unwrap() []error { return e.Errs }

type Capture struct {
	Workspace string
	Title     string
	Text      string
	Genre     string
	Node      string
}

func (r *Router) home(workspace string) (string, string, Endpoint, error) {
	if into, ok := r.topo.Retired[workspace]; ok {
		workspace = into
	}
	eng, err := r.topo.HomeOf(workspace)
	if err != nil {
		return "", "", Endpoint{}, err
	}
	ep, ok := r.endpoints[eng.Name]
	if !ok || ep.URL == "" {
		return "", "", Endpoint{}, fmt.Errorf("workspace %q: home engine %q has no endpoint", workspace, eng.Name)
	}
	return workspace, eng.Name, ep, nil
}

// Capture writes text into its workspace's home engine and returns the
// engine's signal id.
func (r *Router) Capture(ctx context.Context, c Capture) (string, error) {
	if strings.TrimSpace(c.Text) == "" {
		return "", errors.New("memory: text is required")
	}
	ws, engine, ep, err := r.home(c.Workspace)
	if err != nil {
		return "", err
	}
	body := map[string]any{"text": c.Text, "workspace": Tenant + ":" + ws, "extract_claims": true}
	for k, v := range map[string]string{"title": c.Title, "genre": c.Genre, "node": c.Node} {
		if v != "" {
			body[k] = v
		}
	}
	raw, _ := json.Marshal(body)
	var out struct {
		OK       *bool  `json:"ok"`
		SignalID string `json:"signal_id"`
		Error    string `json:"error"`
	}
	if err := r.call(ctx, engine, ep, http.MethodPost, "/api/ingest", raw, &out); err != nil {
		return "", err
	}
	if out.OK != nil && !*out.OK {
		return "", fmt.Errorf("engine %s refused: %s", engine, out.Error)
	}
	return out.SignalID, nil
}

type Hit struct {
	Workspace string `json:"workspace"`
	Engine    string `json:"engine"`
	Rank      int    `json:"rank"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	URI       string `json:"uri"`
	Abstract  string `json:"abstract,omitempty"`
	// Score is the engine's relevance (grep mode); nil when it gave none.
	Score *float64 `json:"score,omitempty"`
}

// Search queries each workspace in its home engine and interleaves the
// results by rank (rank 1 of every workspace, then rank 2, ...) up to limit.
// When some workspaces fail and others answer, it returns the answered hits
// with a *PartialError; when none answer, a plain error and no hits.
func (r *Router) Search(ctx context.Context, query string, workspaces []string, limit int) ([]Hit, error) {
	if limit <= 0 {
		limit = 10
	}
	return r.search(ctx, query, workspaces, limit, limit)
}

// searchEach takes up to perWorkspace hits from every workspace (the union,
// interleaved by rank) — the candidate pool for a rerank.
func (r *Router) searchEach(ctx context.Context, query string, workspaces []string, perWorkspace int) ([]Hit, error) {
	return r.search(ctx, query, workspaces, perWorkspace, perWorkspace*len(workspaces))
}

func (r *Router) search(ctx context.Context, query string, workspaces []string, perWorkspace, limit int) ([]Hit, error) {
	type result struct {
		hits []Hit
		err  error
	}
	results := make([]result, len(workspaces))
	var wg sync.WaitGroup
	for i, w := range workspaces {
		wg.Add(1)
		go func(i int, w string) {
			defer wg.Done()
			ws, engine, ep, err := r.home(w)
			if err != nil {
				results[i].err = err
				return
			}
			switch r.Mode {
			case ModeSearch:
				results[i].hits, results[i].err = r.searchOne(ctx, engine, ep, ws, query, perWorkspace)
			case ModeHybrid:
				results[i].hits, results[i].err = r.hybridOne(ctx, engine, ep, ws, query, perWorkspace)
			default:
				results[i].hits, results[i].err = r.grepOne(ctx, engine, ep, ws, query, perWorkspace)
			}
		}(i, w)
	}
	wg.Wait()
	var partial PartialError
	var firstErr error
	for i, res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			partial.Failed = append(partial.Failed, workspaces[i])
			partial.Errs = append(partial.Errs, res.err)
		}
	}
	if firstErr != nil && len(partial.Failed) == len(workspaces) {
		return nil, firstErr // nothing answered: unreachable, never empty
	}
	var out []Hit
	for rank := 0; len(out) < limit; rank++ {
		added := false
		for _, res := range results {
			if rank < len(res.hits) && len(out) < limit {
				out = append(out, res.hits[rank])
				added = true
			}
		}
		if !added {
			break
		}
	}
	if firstErr != nil {
		return out, &partial
	}
	return out, nil
}

func (r *Router) searchOne(ctx context.Context, engine string, ep Endpoint, ws, query string, n int) ([]Hit, error) {
	var body struct {
		Results []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			URI        string `json:"uri"`
			L0Abstract string `json:"l0_abstract"`
		} `json:"results"`
	}
	path := fmt.Sprintf("/api/search?q=%s&workspace=%s&limit=%d", url.QueryEscape(query), url.QueryEscape(Tenant+":"+ws), n)
	if err := r.call(ctx, engine, ep, http.MethodGet, path, nil, &body); err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(body.Results))
	for rank, h := range body.Results {
		hits = append(hits, Hit{Workspace: ws, Engine: engine, Rank: rank + 1, ID: h.ID, Title: h.Title, URI: h.URI, Abstract: h.L0Abstract})
	}
	return hits, nil
}

// hybridOne alternates search and grep hits (both reads must answer), with
// ranks renumbered so engine order still interleaves them.
func (r *Router) hybridOne(ctx context.Context, engine string, ep Endpoint, ws, query string, n int) ([]Hit, error) {
	s, err := r.searchOne(ctx, engine, ep, ws, query, n)
	if err != nil {
		return nil, err
	}
	g, err := r.grepOne(ctx, engine, ep, ws, query, n)
	if err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(s)+len(g))
	for i := 0; i < len(s) || i < len(g); i++ {
		if i < len(s) {
			out = append(out, s[i])
		}
		if i < len(g) {
			out = append(out, g[i])
		}
	}
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

// grepSnippetChars caps a grep snippet (SNIPPET_CHARS in lib/connectors/optimal.ts).
const grepSnippetChars = 400

var (
	headingLine = regexp.MustCompile(`(?m)^\s*#{1,6}\s+(.+?)\s*$`)
	titleLine   = regexp.MustCompile(`(?m)^\s*title:\s*(.+?)\s*$`)
	spaces      = regexp.MustCompile(`\s+`)
)

// titleFromSnippet is lib/connectors/optimal.ts titleFromSnippet.
func titleFromSnippet(snippet, slug string) string {
	if m := headingLine.FindStringSubmatch(snippet); m != nil {
		return m[1]
	}
	if m := titleLine.FindStringSubmatch(snippet); m != nil {
		return m[1]
	}
	return slug
}

func (r *Router) grepOne(ctx context.Context, engine string, ep Endpoint, ws, query string, n int) ([]Hit, error) {
	var body struct {
		Results []struct {
			Slug    string   `json:"slug"`
			Snippet string   `json:"snippet"`
			Score   *float64 `json:"score"`
		} `json:"results"`
	}
	path := fmt.Sprintf("/api/grep?q=%s&workspace=%s&limit=%d", url.QueryEscape(query), url.QueryEscape(Tenant+":"+ws), n)
	if err := r.call(ctx, engine, ep, http.MethodGet, path, nil, &body); err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(body.Results))
	for rank, h := range body.Results {
		snip := []rune(strings.TrimSpace(spaces.ReplaceAllString(h.Snippet, " ")))
		if len(snip) > grepSnippetChars {
			snip = snip[:grepSnippetChars]
		}
		hits = append(hits, Hit{
			Workspace: ws, Engine: engine, Rank: rank + 1, ID: h.Slug,
			Title: titleFromSnippet(h.Snippet, h.Slug), URI: "optimal://" + ws + "/" + h.Slug,
			Abstract: string(snip), Score: h.Score,
		})
	}
	return hits, nil
}

func (r *Router) call(ctx context.Context, engine string, ep Endpoint, method, path string, body []byte, out any) error {
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(ep.URL, "/")+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if ep.Key != "" {
			req.Header.Set("Authorization", "Bearer "+ep.Key)
		}
		resp, err := r.client.Do(req)
		if err != nil {
			return &UnreachableError{Engine: engine, Err: err}
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusTooManyRequests && attempt < 6:
			time.Sleep(time.Duration(attempt*attempt) * 100 * time.Millisecond)
			continue
		case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
			return &UnreachableError{Engine: engine, Err: fmt.Errorf("HTTP %d", resp.StatusCode)}
		case resp.StatusCode/100 != 2:
			return fmt.Errorf("engine %s: HTTP %d: %s", engine, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(raw, out)
	}
}
