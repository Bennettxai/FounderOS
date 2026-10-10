package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

// Engine is one Optimal Engine endpoint from the topology.
type Engine struct {
	Name string
	URL  string
	Key  string
}

// EngineClient reads the engines' graph and health endpoints (read-only).
type EngineClient struct{ client *http.Client }

func NewEngineClient() *EngineClient {
	return &EngineClient{client: connectors.HTTPClient(20 * time.Second)}
}

// get reads one engine endpoint. The engine rate-limits bursts (429 with a
// retry hint): a page that reads every workspace's graph at once backs off
// and retries (as memory.Router does) rather than calling the engine
// unreachable; a 429 that outlasts the retries says rate-limited.
func (c *EngineClient) get(ctx context.Context, e Engine, path string, out any) error {
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.URL, "/")+path, nil)
		if err != nil {
			return err
		}
		if e.Key != "" {
			req.Header.Set("Authorization", "Bearer "+e.Key)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt >= 5 {
				return fmt.Errorf("HTTP 429: rate-limited")
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt*attempt) * 100 * time.Millisecond):
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return json.Unmarshal(raw, out)
	}
}

// Graph reads one workspace's GET /api/graph. A failure is carried in Err.
func (c *EngineClient) Graph(ctx context.Context, e Engine, workspace string) WorkspaceGraph {
	g := WorkspaceGraph{Workspace: workspace, Engine: e.Name}
	var body struct {
		Contexts []struct {
			ID         string `json:"id"`
			Node       string `json:"node"`
			Title      string `json:"title"`
			URI        string `json:"uri"`
			L0Abstract string `json:"l0_abstract"`
			Genre      string `json:"genre"`
			ModifiedAt string `json:"modified_at"`
		} `json:"contexts"`
		Edges []EngineEdge `json:"edges"`
	}
	path := "/api/graph?workspace=" + url.QueryEscape(memory.Tenant+":"+workspace)
	if err := c.get(ctx, e, path, &body); err != nil {
		g.Err = fmt.Sprintf("engine %s: %v", e.Name, err)
		return g
	}
	for _, x := range body.Contexts {
		t, _ := time.Parse(time.RFC3339Nano, x.ModifiedAt)
		title := x.Title
		if title == "" {
			title = x.ID
		}
		g.Contexts = append(g.Contexts, EngineContext{ID: x.ID, Node: x.Node, Title: title, URI: x.URI, Abstract: x.L0Abstract, Genre: x.Genre, Modified: t})
	}
	g.Contexts = withoutCrossRefCopies(g.Contexts)
	g.Edges = body.Edges
	return g
}

// copyNodes are where the engine's router copies pages on its own
// (optimal-engine pipeline/router.ex cross-cutting rules: any page that
// mentions an entity goes to team, financial genres to operation-revenue).
var copyNodes = map[string]bool{"team": true, "operation-revenue": true}

// withoutCrossRefCopies drops those copies: a context in a copy node whose
// title and file name match a page elsewhere in the workspace is that page
// again. Edges to a dropped copy fall away when the graph is built.
func withoutCrossRefCopies(cs []EngineContext) []EngineContext {
	key := func(c EngineContext) string { return c.Title + "\x00" + path.Base(c.URI) }
	primary := map[string]bool{}
	for _, c := range cs {
		if !copyNodes[c.Node] {
			primary[key(c)] = true
		}
	}
	out := cs[:0:0]
	for _, c := range cs {
		if copyNodes[c.Node] && primary[key(c)] {
			continue
		}
		out = append(out, c)
	}
	return out
}

// EngineHealth is one engine's GET /api/health, or why it could not answer.
type EngineHealth struct {
	Engine   string            `json:"engine"`
	Up       bool              `json:"up"`
	Status   string            `json:"status,omitempty"`
	Checks   map[string]string `json:"checks,omitempty"`
	Degraded []string          `json:"degraded,omitempty"`
	Err      string            `json:"error,omitempty"`
}

func (c *EngineClient) Health(ctx context.Context, e Engine) EngineHealth {
	h := EngineHealth{Engine: e.Name}
	var body struct {
		Status   string            `json:"status"`
		OK       *bool             `json:"ok?"`
		Checks   map[string]string `json:"checks"`
		Degraded []string          `json:"degraded"`
	}
	if err := c.get(ctx, e, "/api/health", &body); err != nil {
		h.Err = err.Error()
		return h
	}
	h.Status, h.Checks, h.Degraded = body.Status, body.Checks, body.Degraded
	h.Up = body.Status == "up" && (body.OK == nil || *body.OK)
	return h
}

// Check / Doctor: FounderOS v1 BrainOverview.doctor, built from engine health
// instead of `gbrain doctor`.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | warn | error
	Message string `json:"message"`
}

type Doctor struct {
	Connected   bool     `json:"connected"`
	Status      string   `json:"status"`
	HealthScore *float64 `json:"healthScore"` // the engine computes none; never invented
	Checks      []Check  `json:"checks"`
	Detail      string   `json:"detail"`
}

// BuildDoctor turns engine health, workspace reads, unstaged workspaces and
// the reranker probe (nil = not probed) into doctor-style checks.
// RerankState is the reranker probe's verdict (memory.RerankerState).
type RerankState string

const (
	RerankUnchecked   RerankState = ""
	RerankOff         RerankState = "off"
	RerankPassing     RerankState = "passing"
	RerankFailing     RerankState = "failing"
	RerankUnreachable RerankState = "unreachable"
)

func BuildDoctor(healths []EngineHealth, gs []WorkspaceGraph, unstaged []string, rerank RerankState) Doctor {
	d := Doctor{Checks: []Check{}}
	up := 0
	for _, h := range healths {
		c := Check{Name: h.Engine + " engine"}
		switch {
		case h.Err != "":
			c.Status, c.Message = "error", "unreachable: "+h.Err
		case !h.Up:
			c.Status, c.Message = "error", fmt.Sprintf("health %q, degraded: %s", h.Status, strings.Join(h.Degraded, ", "))
		default:
			up++
			c.Status = "ok"
			names := make([]string, 0, len(h.Checks))
			for k, v := range h.Checks {
				names = append(names, k+" "+strings.TrimPrefix(v, ":"))
			}
			sort.Strings(names)
			c.Message = "up"
			if len(names) > 0 {
				c.Message += " · " + strings.Join(names, ", ")
			}
			if len(h.Degraded) > 0 {
				c.Status, c.Message = "warn", c.Message+" · degraded: "+strings.Join(h.Degraded, ", ")
			}
		}
		d.Checks = append(d.Checks, c)
	}
	pages := 0
	for _, g := range gs {
		if g.Err != "" {
			d.Checks = append(d.Checks, Check{Name: g.Workspace + " workspace", Status: "error", Message: g.Err})
			continue
		}
		pages += len(g.Contexts)
	}
	for _, w := range unstaged {
		d.Checks = append(d.Checks, Check{Name: w + " workspace", Status: "warn", Message: "home engine is not staged on this machine; its memory is not read"})
	}
	rr := Check{Name: "reranker"}
	switch rerank {
	case RerankPassing:
		rr.Status, rr.Message = "ok", "cross-encoder passes the canary: each workspace's pool reranked to the top 3"
	case RerankFailing:
		rr.Status, rr.Message = "warn", "answers but fails the canary (its scores are noise): results stay in engine order"
	case RerankUnreachable:
		rr.Status, rr.Message = "warn", "offline: results stay in engine order"
	case RerankOff:
		rr.Status, rr.Message = "warn", "off (BRAIN_RERANK=0): results stay in engine order"
	default:
		rr.Status, rr.Message = "warn", "not checked: results may stay in engine order"
	}
	d.Checks = append(d.Checks, rr)

	d.Status = "ok"
	for _, c := range d.Checks {
		if c.Status == "error" {
			d.Status = "error"
			break
		}
		if c.Status == "warn" {
			d.Status = "warn"
		}
	}
	d.Connected = len(healths) > 0 && up == len(healths)
	d.Detail = fmt.Sprintf("%d/%d engines up · %d workspaces · %d pages", up, len(healths), len(gs), pages)
	return d
}
