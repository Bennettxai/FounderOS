package tech

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/optimalengine"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// EngineSource resolves the engine topology and the engines staged on this
// machine. It is called on every run, so a key planted while the backend is
// up takes effect without a restart.
type EngineSource func() (*topology.Topology, []optimalengine.Engine, error)

// stagingPorts are the bridge's staging engines (spec: ground rules). They
// match api.TopologyEngines, which this package may not import.
var stagingPorts = map[string]int{"macbook": 4210, "hub": 4211, "mini": 4212}

// DefaultEngineSource loads the checked-in topology and resolves each engine
// the way api.TopologyEngines does: url_env/key_env first, else the local
// staging engine when ~/.founderos-bridge/keys/oe-<name>.key exists.
func DefaultEngineSource() (*topology.Topology, []optimalengine.Engine, error) {
	topo, err := topology.LoadRepo()
	if err != nil {
		return nil, nil, fmt.Errorf("engine topology unavailable: %w", err)
	}
	home, _ := os.UserHomeDir()
	return topo, stagedEngines(topo, home), nil
}

// stagedEngines returns the engines that are a workspace's home and have an
// endpoint on this machine, sorted by name. Deferred engines are skipped.
func stagedEngines(topo *topology.Topology, home string) []optimalengine.Engine {
	used := map[string]bool{}
	for _, w := range topo.Workspaces {
		used[w.Home] = true
	}
	var out []optimalengine.Engine
	for name, e := range topo.Engines {
		if e.Deferred || !used[name] {
			continue
		}
		var u, key string
		if e.URLEnv != "" {
			u = os.Getenv(e.URLEnv)
		}
		if e.KeyEnv != "" {
			key = os.Getenv(e.KeyEnv)
		}
		keyFile := filepath.Join(home, ".founderos-bridge", "keys", "oe-"+name+".key")
		if u == "" {
			port, known := stagingPorts[name]
			if _, err := os.Stat(keyFile); err != nil || !known {
				continue // not staged on this machine
			}
			u = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
		if key == "" {
			raw, _ := os.ReadFile(keyFile)
			key = strings.TrimSpace(string(raw))
		}
		out = append(out, optimalengine.Engine{Name: name, URL: u, Key: key})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// view is one run's picture of the topology: which workspaces live on a
// staged engine and which cannot be seen from here.
type view struct {
	topo     *topology.Topology
	engines  []optimalengine.Engine
	homed    map[string][]string // engine → topology workspaces homed there
	unstaged map[string][]string // unstaged engine → its workspaces
}

func resolveView(src EngineSource) (*view, error) {
	if src == nil {
		src = DefaultEngineSource
	}
	topo, engines, err := src()
	if err != nil {
		return nil, err
	}
	if topo == nil {
		return nil, errors.New("engine topology unavailable")
	}
	v := &view{topo: topo, engines: engines, homed: map[string][]string{}, unstaged: map[string][]string{}}
	staged := map[string]bool{}
	for _, e := range engines {
		staged[e.Name] = true
	}
	for _, w := range topo.Workspaces {
		if _, retired := topo.Retired[w.Slug]; retired {
			continue
		}
		if staged[w.Home] {
			v.homed[w.Home] = append(v.homed[w.Home], w.Slug)
		} else {
			v.unstaged[w.Home] = append(v.unstaged[w.Home], w.Slug)
		}
	}
	return v, nil
}

// stagedWorkspaces are every workspace whose home engine is reachable from
// here, in topology order.
func (v *view) stagedWorkspaces() []string {
	staged := map[string]bool{}
	for _, e := range v.engines {
		staged[e.Name] = true
	}
	var out []string
	for _, w := range v.topo.Workspaces {
		if _, retired := v.topo.Retired[w.Slug]; !retired && staged[w.Home] {
			out = append(out, w.Slug)
		}
	}
	return out
}

// unstagedNote reads "mini not staged (hermes, mini-device)", or "".
func (v *view) unstagedNote() string {
	names := make([]string, 0, len(v.unstaged))
	for n := range v.unstaged {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%s not staged (%s)", n, strings.Join(v.unstaged[n], ", ")))
	}
	return strings.Join(parts, "; ")
}

const noEnginesSummary = "no Optimal Engine is staged on this machine: set OE_<ENGINE>_URL and OE_<ENGINE>_KEY, or plant ~/.founderos-bridge/keys/oe-<engine>.key for the staging engine"

// ---- engine HTTP reads -------------------------------------------------------

// engineAPI reads one engine. Every call is a GET; nothing here writes.
type engineAPI struct {
	e      optimalengine.Engine
	client *http.Client
}

func newClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return connectors.HTTPClient(60 * time.Second)
}

// get decodes a JSON body. accept lists extra statuses whose body is still
// the answer (the engine's audit returns 503 with the full report).
func (a engineAPI) get(ctx context.Context, path string, out any, accept ...int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.e.URL, "/")+path, nil)
	if err != nil {
		return err
	}
	if a.e.Key != "" {
		req.Header.Set("Authorization", "Bearer "+a.e.Key)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("unreachable: %w", err)
	}
	defer resp.Body.Close()
	ok := resp.StatusCode/100 == 2
	for _, s := range accept {
		ok = ok || resp.StatusCode == s
	}
	if !ok {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s: HTTP %d %s", path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<20)).Decode(out); err != nil {
		return fmt.Errorf("%s: bad JSON: %w", path, err)
	}
	return nil
}

func wsID(slug string) string { return url.QueryEscape(memory.Tenant + ":" + slug) }

func (a engineAPI) health(ctx context.Context) (string, error) {
	var h struct {
		Status string `json:"status"`
	}
	if err := a.get(ctx, "/api/health", &h); err != nil {
		return "", err
	}
	return h.Status, nil
}

// workspaces lists the active workspace slugs on the engine.
func (a engineAPI) workspaces(ctx context.Context) (map[string]bool, error) {
	var list struct {
		Workspaces []struct {
			Slug string `json:"slug"`
		} `json:"workspaces"`
	}
	if err := a.get(ctx, "/api/workspaces?status=active&limit=200", &list); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, w := range list.Workspaces {
		out[w.Slug] = true
	}
	return out, nil
}

// counts merges every logical store's table counts. A table the engine could
// not count is nil (unknown), never 0.
func (a engineAPI) counts(ctx context.Context) (map[string]*int, error) {
	var body struct {
		Stores []struct {
			TableCounts map[string]*int `json:"table_counts"`
		} `json:"stores"`
	}
	if err := a.get(ctx, "/api/stores", &body); err != nil {
		return nil, err
	}
	out := map[string]*int{}
	for _, s := range body.Stores {
		for k, v := range s.TableCounts {
			out[k] = v
		}
	}
	return out, nil
}

type auditCheck struct {
	Name   string          `json:"name"`
	OK     bool            `json:"ok"`
	Detail json.RawMessage `json:"detail"`
}

// detail renders a check's detail compactly: a bare string unquoted, else JSON.
func (c auditCheck) detail() string {
	var s string
	if json.Unmarshal(c.Detail, &s) == nil {
		return s
	}
	return string(c.Detail)
}

type storesAudit struct {
	OK     bool         `json:"ok"`
	Checks []auditCheck `json:"checks"`
}

func (s storesAudit) failing() []auditCheck {
	var out []auditCheck
	for _, c := range s.Checks {
		if !c.OK {
			out = append(out, c)
		}
	}
	return out
}

func (s storesAudit) check(name string) (auditCheck, bool) {
	for _, c := range s.Checks {
		if c.Name == name {
			return c, true
		}
	}
	return auditCheck{}, false
}

func (a engineAPI) audit(ctx context.Context) (storesAudit, error) {
	var out storesAudit
	err := a.get(ctx, "/api/stores/audit", &out, http.StatusServiceUnavailable)
	if err == nil && len(out.Checks) == 0 {
		err = errors.New("/api/stores/audit: no checks in the report")
	}
	return out, err
}

type signal struct {
	Title      string `json:"title"`
	URI        string `json:"uri"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
}

// signals exports a workspace's sources (read-only batch export).
func (a engineAPI) signals(ctx context.Context, slug string) ([]signal, error) {
	var body struct {
		Signals *[]signal `json:"signals"`
	}
	if err := a.get(ctx, "/api/batch/export/signals?workspace="+wsID(slug), &body); err != nil {
		return nil, err
	}
	if body.Signals == nil {
		return nil, errors.New("signal export had no signals field")
	}
	return *body.Signals, nil
}

// pendingClaims counts a workspace's claims awaiting review.
func (a engineAPI) pendingClaims(ctx context.Context, slug string) (int, error) {
	var body struct {
		Count *int `json:"count"`
	}
	if err := a.get(ctx, "/api/memory-core/claims?workspace="+wsID(slug), &body); err != nil {
		return 0, err
	}
	if body.Count == nil {
		return 0, errors.New("claims response had no count")
	}
	return *body.Count, nil
}

// ---- engine overview (data agent + vector auditor) ---------------------------

// EngineOverview is one engine's storage picture.
type EngineOverview struct {
	Engine     string          `json:"engine"`
	Reachable  bool            `json:"reachable"`
	Error      string          `json:"error,omitempty"`
	Health     string          `json:"health,omitempty"`
	Counts     map[string]*int `json:"counts,omitempty"`
	Audit      *storesAudit    `json:"audit,omitempty"`
	AuditError string          `json:"auditError,omitempty"`
	Workspaces []string        `json:"workspaces"`           // homed here by the topology
	Missing    []string        `json:"missing,omitempty"`    // homed here but absent on the engine
	Pending    map[string]*int `json:"pending,omitempty"`    // claims awaiting review, per workspace
	PendingErr string          `json:"pendingErr,omitempty"` // first pending-claims read error
}

func (o EngineOverview) count(table string) *int { return o.Counts[table] }

// overview reads health, table counts, the storage audit and the workspace
// list of every staged engine concurrently. withPending also reads each
// homed workspace's pending claims.
func overview(ctx context.Context, v *view, client *http.Client, withPending bool) []EngineOverview {
	out := make([]EngineOverview, len(v.engines))
	var wg sync.WaitGroup
	for i, e := range v.engines {
		wg.Add(1)
		go func(i int, e optimalengine.Engine) {
			defer wg.Done()
			out[i] = overviewOne(ctx, engineAPI{e: e, client: client}, v.homed[e.Name], withPending)
		}(i, e)
	}
	wg.Wait()
	return out
}

func overviewOne(ctx context.Context, a engineAPI, homed []string, withPending bool) EngineOverview {
	o := EngineOverview{Engine: a.e.Name, Workspaces: homed}
	status, err := a.health(ctx)
	if err != nil {
		o.Error = err.Error()
		return o
	}
	o.Reachable, o.Health = true, status
	if o.Counts, err = a.counts(ctx); err != nil {
		o.Error = err.Error()
	}
	if audit, err := a.audit(ctx); err != nil {
		o.AuditError = err.Error()
	} else {
		o.Audit = &audit
	}
	present, err := a.workspaces(ctx)
	if err != nil {
		if o.Error == "" {
			o.Error = err.Error()
		}
	} else {
		for _, w := range homed {
			if !present[w] {
				o.Missing = append(o.Missing, w)
			}
		}
	}
	if withPending {
		o.Pending = map[string]*int{}
		for _, w := range homed {
			if present != nil && !present[w] {
				continue
			}
			n, err := a.pendingClaims(ctx, w)
			if err != nil {
				if o.PendingErr == "" {
					o.PendingErr = fmt.Sprintf("%s: %v", w, err)
				}
				o.Pending[w] = nil
				continue
			}
			o.Pending[w] = &n
		}
	}
	return o
}

func itoa(p *int) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprint(*p)
}
