package api

// /os/brain (spec 6.6): the knowledge core, on the Optimal Engine. FounderOS v1
// app/brain/page.tsx + /api/brain/{dump,graph,overview,satellites}. The
// ask bar reads /pages/brain/query (GET /api/brain itself is the compat
// listener's). Every engine read is surfaced when it fails: an unreachable
// engine is never an empty brain.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/pages/brain"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

func init() { RegisterPage(registerBrain) }

type brainWorkspace struct {
	Slug   string
	Engine brain.Engine
}

// brainRun is the latest founderos_agent_runs row per agent.
type brainRun struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agentId"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	OK         bool      `json:"ok"`
	Summary    string    `json:"summary"`
	Model      *string   `json:"model"`
}

type brainOrg struct {
	Departments []brain.Department
	Agents      []brain.Agent
	People      []brain.Person
	Tasks       []brain.SopTask
	Runs        map[string]brainRun
}

// brainEnv is everything the brain routes read. Tests swap newBrainEnv.
type brainEnv struct {
	Topo       *topology.Topology
	Engines    []brain.Engine
	Workspaces []brainWorkspace // workspaces whose home engine is staged here
	Unstaged   []string         // workspaces whose home engine has no endpoint
	Memory     *memory.Router
	Client     *brain.EngineClient
	Rerank     memory.Reranker // nil: memory.DefaultReranker()
	RerankUp   func(context.Context) brain.RerankState
	Now        func() time.Time
	Org        func(context.Context) (*brainOrg, error)
	Board      func(context.Context) ([]paperclip.Agent, error)

	mu      sync.Mutex
	graphs  []brain.WorkspaceGraph
	graphAt time.Time
}

const (
	brainGraphTTL     = 60 * time.Second
	brainBoardTimeout = 4 * time.Second
)

var errNoOrg = errors.New("the founderos workspace is not bootstrapped: no org to draw")

var newBrainEnv = func(d *Deps) *brainEnv {
	env := &brainEnv{Client: brain.NewEngineClient(), Now: time.Now, RerankUp: probeReranker}
	topo, err := topology.LoadRepo()
	if err == nil {
		env.Topo = topo
		eps := map[string]memory.Endpoint{}
		byName := map[string]brain.Engine{}
		for _, e := range TopologyEngines() {
			eps[e.Name] = memory.Endpoint{URL: e.URL, Key: e.Key}
			be := brain.Engine{Name: e.Name, URL: e.URL, Key: e.Key}
			byName[e.Name] = be
			env.Engines = append(env.Engines, be)
		}
		env.Memory = memory.New(topo, eps)
		for _, w := range topo.Workspaces {
			if e, ok := byName[w.Home]; ok {
				env.Workspaces = append(env.Workspaces, brainWorkspace{Slug: w.Slug, Engine: e})
			} else {
				env.Unstaged = append(env.Unstaged, w.Slug)
			}
		}
	}
	env.Org = func(ctx context.Context) (*brainOrg, error) {
		if d.Pool == nil {
			return nil, errNoOrg
		}
		var ws string
		if err := d.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err != nil {
			return nil, errNoOrg
		}
		return readBrainOrg(ctx, d.Pool, ws)
	}
	env.Board = func(ctx context.Context) ([]paperclip.Agent, error) {
		return paperclip.New(d.Resolver).Agents(ctx)
	}
	return env
}

// probeReranker checks the local llama-server reranker's /health (nil when
// reranking is switched off with BRAIN_RERANK=0).
// probeReranker asks the configured reranker the canary: up alone is not
// enough (the :8081 GGUF answered and ranked noise).
func probeReranker(ctx context.Context) brain.RerankState {
	if os.Getenv("BRAIN_RERANK") == "0" {
		return brain.RerankOff
	}
	base := os.Getenv("RERANK_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8082/v1"
	}
	model := os.Getenv("RERANK_MODEL")
	if model == "" {
		model = "qwen3-reranker-0.6b"
	}
	return brain.RerankState(memory.RerankerState(ctx, base, model, os.Getenv("RERANK_API_KEY")))
}

// readGraphs reads every staged workspace's graph, cached for a minute.
func (e *brainEnv) readGraphs(ctx context.Context) []brain.WorkspaceGraph {
	e.mu.Lock()
	if e.graphs != nil && e.Now().Sub(e.graphAt) < brainGraphTTL {
		gs := e.graphs
		e.mu.Unlock()
		return gs
	}
	e.mu.Unlock()
	out := make([]brain.WorkspaceGraph, len(e.Workspaces))
	var wg sync.WaitGroup
	for i, w := range e.Workspaces {
		wg.Add(1)
		go func(i int, w brainWorkspace) {
			defer wg.Done()
			out[i] = e.Client.Graph(ctx, w.Engine, w.Slug)
		}(i, w)
	}
	wg.Wait()
	ok := true
	for _, g := range out {
		ok = ok && g.Err == ""
	}
	if ok { // never cache a failed read: the next click retries
		e.mu.Lock()
		e.graphs, e.graphAt = out, e.Now()
		e.mu.Unlock()
	}
	return out
}

func (e *brainEnv) healths(ctx context.Context) []brain.EngineHealth {
	out := make([]brain.EngineHealth, len(e.Engines))
	var wg sync.WaitGroup
	for i, en := range e.Engines {
		wg.Add(1)
		go func(i int, en brain.Engine) {
			defer wg.Done()
			out[i] = e.Client.Health(ctx, en)
		}(i, en)
	}
	wg.Wait()
	return out
}

func (e *brainEnv) status(healths []brain.EngineHealth) brain.ProviderStatus {
	st := brain.ProviderStatus{Provider: "optimal-engine"}
	if len(healths) == 0 {
		st.Detail = "no engines in the topology have an endpoint"
		return st
	}
	up := 0
	var down []string
	for _, h := range healths {
		if h.Up {
			up++
		} else {
			down = append(down, h.Engine)
		}
	}
	st.Connected = up == len(healths)
	st.Detail = strconv.Itoa(up) + "/" + strconv.Itoa(len(healths)) + " engines up"
	if len(down) > 0 {
		st.Detail += " · down: " + strings.Join(down, ", ")
	}
	return st
}

func registerBrain(s *gin.RouterGroup, d *Deps) {
	env := newBrainEnv(d)

	s.GET("/pages/brain", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()
		board := make(chan struct {
			agents []paperclip.Agent
			err    error
		}, 1)
		go func() {
			// The board is off this box (the mini, over the tailnet): it gets
			// a short leash so a dead tailnet costs the page seconds, not the
			// connector's full timeout. A timeout reads as an error.
			bctx, bcancel := context.WithTimeout(ctx, brainBoardTimeout)
			defer bcancel()
			a, err := env.Board(bctx)
			board <- struct {
				agents []paperclip.Agent
				err    error
			}{a, err}
		}()
		org, err := env.Org(ctx)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, errNoOrg) {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		b := <-board
		c.JSON(http.StatusOK, composeBrainPage(org, b.agents, b.err))
	})

	s.GET("/pages/brain/graph", func(c *gin.Context) {
		gs := env.readGraphs(c.Request.Context())
		limit := brain.DefaultMaxPages
		if c.Query("full") == "1" {
			limit = 1 << 30
		}
		type wsRow struct {
			Workspace string `json:"workspace"`
			Engine    string `json:"engine"`
			Pages     *int   `json:"pages"` // null when unreadable
			Edges     *int   `json:"edges"`
			Error     string `json:"error,omitempty"`
		}
		rows := []wsRow{}
		for _, g := range gs {
			r := wsRow{Workspace: g.Workspace, Engine: g.Engine, Error: g.Err}
			if g.Err == "" {
				p, e := len(g.Contexts), len(g.Edges)
				r.Pages, r.Edges = &p, &e
			}
			rows = append(rows, r)
		}
		c.JSON(http.StatusOK, gin.H{"constellation": brain.BuildMemoryGraph(gs, limit), "workspaces": rows, "unstaged": nonNilStrings(env.Unstaged)})
	})

	s.GET("/pages/brain/overview", func(c *gin.Context) {
		ctx := c.Request.Context()
		var (
			gs []brain.WorkspaceGraph
			hs []brain.EngineHealth
			rr brain.RerankState
		)
		var wg sync.WaitGroup
		wg.Add(3)
		go func() { defer wg.Done(); gs = env.readGraphs(ctx) }()
		go func() { defer wg.Done(); hs = env.healths(ctx) }()
		go func() { defer wg.Done(); rr = env.RerankUp(ctx) }()
		wg.Wait()
		c.JSON(http.StatusOK, gin.H{"store": brain.StoreFrom(gs), "doctor": brain.BuildDoctor(hs, gs, env.Unstaged, rr)})
	})

	s.GET("/pages/brain/satellites", func(c *gin.Context) {
		ctx := c.Request.Context()
		var (
			gs []brain.WorkspaceGraph
			hs []brain.EngineHealth
		)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); gs = env.readGraphs(ctx) }()
		go func() { defer wg.Done(); hs = env.healths(ctx) }()
		wg.Wait()
		c.JSON(http.StatusOK, brain.SummarizeSatellites(brain.StoreFrom(gs), brain.ModifiedTimes(gs), env.status(hs), env.Now()))
	})

	s.GET("/pages/brain/query", func(c *gin.Context) {
		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
			return
		}
		if env.Memory == nil || len(env.Workspaces) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no staged engine to search"})
			return
		}
		slugs := make([]string, len(env.Workspaces))
		for i, w := range env.Workspaces {
			slugs[i] = w.Slug
		}
		r := env.Memory.Retrieve(c.Request.Context(), q, slugs, memory.RetrieveOptions{Rerank: env.Rerank})
		if r.Error != "" {
			c.JSON(http.StatusBadGateway, gin.H{"query": q, "provider": "optimal-engine", "error": r.Error, "results": []any{}})
			return
		}
		type hit struct {
			Title     string   `json:"title"`
			Snippet   string   `json:"snippet"`
			Source    string   `json:"source"`
			Workspace string   `json:"workspace"`
			Engine    string   `json:"engine"`
			URI       string   `json:"uri"`
			Score     *float64 `json:"score"`
		}
		out := []hit{}
		for _, h := range r.Hits {
			out = append(out, hit{Title: h.Title, Snippet: h.Abstract, Source: h.URI, Workspace: h.Workspace, Engine: h.Engine, URI: h.URI, Score: h.RerankScore})
		}
		body := gin.H{"query": q, "provider": "optimal-engine", "ranked": r.Ranked, "results": out}
		if len(r.FailedWorkspaces) > 0 {
			// partial: say which workspaces went unsearched instead of passing
			// the answer off as the whole brain
			body["failedWorkspaces"], body["degraded"] = r.FailedWorkspaces, r.Degraded
		}
		c.JSON(http.StatusOK, body)
	})

	s.POST("/pages/brain/dump", func(c *gin.Context) {
		var in brain.DumpInput
		if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)).Decode(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body is not JSON: " + err.Error()})
			return
		}
		d, err := in.Validate()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if env.Memory == nil || env.Topo == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no engine topology", "embedded": false})
			return
		}
		now := env.Now()
		ws := d.Workspace(env.Topo)
		id, err := env.Memory.Capture(c.Request.Context(), memory.Capture{Workspace: ws, Title: d.Title, Text: d.Document(now), Genre: "note", Node: d.Node()})
		if err != nil {
			// Nothing was saved: the engine is the store on the bridge.
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "embedded": false, "workspace": ws})
			return
		}
		env.mu.Lock()
		env.graphs = nil // the next graph read shows the new page
		env.mu.Unlock()
		c.JSON(http.StatusOK, gin.H{"ok": true, "title": d.Title, "relPath": d.RelPath(now), "workspace": ws, "embedded": true, "slug": id})
	})
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// boardLeadNames: dept id → the board lead's Paperclip seat name.
var boardLeadNames = map[string]string{
	"dept-sales":            "Sales",
	"dept-marketing-growth": "Marketing/Growth",
	"dept-tech":             "TECH",
	"dept-finance":          "Finances",
	"dept-comms":            "Communications",
}

type brainSeat struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Status string  `json:"status"`
	Model  *string `json:"model"`
}

func composeBrainPage(org *brainOrg, live []paperclip.Agent, boardErr error) gin.H {
	leads := map[string]brainSeat{}
	leadNames := map[string]bool{}
	for dept, name := range boardLeadNames {
		leadNames[strings.ToLower(name)] = true
		for _, a := range live {
			if strings.EqualFold(a.Name, name) {
				leads[dept] = brainSeat{a.ID, a.Name, a.Status, a.Model}
				break
			}
		}
	}
	seats := []brainSeat{}
	var ring []brain.BoardAgent
	for _, a := range live {
		if leadNames[strings.ToLower(a.Name)] {
			continue
		}
		seats = append(seats, brainSeat{a.ID, a.Name, a.Status, a.Model})
		ring = append(ring, brain.BoardAgent{ID: a.ID, Name: a.Name})
	}
	board := gin.H{"state": "connected", "detail": strconv.Itoa(len(live)) + " live seats"}
	if boardErr != nil {
		board = gin.H{"state": "error", "detail": boardErr.Error()}
	}
	graph := brain.BuildKnowledgeGraph(org.Agents, org.Departments, org.People, org.Tasks, ring)
	runs := org.Runs
	if runs == nil {
		runs = map[string]brainRun{}
	}
	return gin.H{
		"graph":       graph,
		"directory":   brain.GraphDirectory(org.Agents, org.Departments, org.People, org.Tasks, graph),
		"departments": brain.OrderGraphDepartments(org.Departments),
		"agents":      org.Agents,
		"people":      org.People,
		"tasks":       org.Tasks,
		"execTitles":  brain.DeptExecTitles,
		"runsByAgent": runs,
		"boardLeads":  leads,
		"boardAgents": seats,
		"board":       board,
	}
}

func readBrainOrg(ctx context.Context, pool *pgxpool.Pool, ws string) (*brainOrg, error) {
	org := &brainOrg{Departments: []brain.Department{}, Agents: []brain.Agent{}, People: []brain.Person{}, Tasks: []brain.SopTask{}, Runs: map[string]brainRun{}}
	strs := func(raw []byte) []string {
		var out []string
		_ = json.Unmarshal(raw, &out)
		if out == nil {
			out = []string{}
		}
		return out
	}

	rows, err := pool.Query(ctx, `SELECT id, name, slug, tagline, color, ord FROM founderos_departments WHERE workspace_id = $1 ORDER BY ord, id`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x brain.Department
		if err := rows.Scan(&x.ID, &x.Name, &x.Slug, &x.Tagline, &x.Color, &x.Order); err != nil {
			rows.Close()
			return nil, err
		}
		org.Departments = append(org.Departments, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = pool.Query(ctx, `SELECT id, department_id, name, role, status, tier, description, model, tools, parent_id, instance FROM founderos_agents WHERE workspace_id = $1 ORDER BY tier COLLATE "C", name COLLATE "C", id`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x brain.Agent
		var tools []byte
		if err := rows.Scan(&x.ID, &x.DepartmentID, &x.Name, &x.Role, &x.Status, &x.Tier, &x.Description, &x.Model, &tools, &x.ParentID, &x.Instance); err != nil {
			rows.Close()
			return nil, err
		}
		x.Tools = strs(tools)
		org.Agents = append(org.Agents, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = pool.Query(ctx, `SELECT id, department_id, name, role, tools FROM founderos_people WHERE workspace_id = $1 ORDER BY department_id COLLATE "C", name COLLATE "C", id`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x brain.Person
		var tools []byte
		if err := rows.Scan(&x.ID, &x.DepartmentID, &x.Name, &x.Role, &tools); err != nil {
			rows.Close()
			return nil, err
		}
		x.Tools = strs(tools)
		org.People = append(org.People, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = pool.Query(ctx, `SELECT id, department_id, title, summary, steps, assignee_kind, assignee_id FROM founderos_sop_tasks WHERE workspace_id = $1 ORDER BY department_id COLLATE "C", title COLLATE "C", id`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var x brain.SopTask
		var steps []byte
		if err := rows.Scan(&x.ID, &x.DepartmentID, &x.Title, &x.Summary, &steps, &x.AssigneeKind, &x.AssigneeID); err != nil {
			rows.Close()
			return nil, err
		}
		x.Steps = strs(steps)
		org.Tasks = append(org.Tasks, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = pool.Query(ctx, `SELECT DISTINCT ON (agent_id) id, agent_id, started_at, finished_at, ok, summary, model
		FROM founderos_agent_runs WHERE workspace_id = $1 ORDER BY agent_id, started_at DESC`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var r brainRun
		if err := rows.Scan(&r.ID, &r.AgentID, &r.StartedAt, &r.FinishedAt, &r.OK, &r.Summary, &r.Model); err != nil {
			rows.Close()
			return nil, err
		}
		org.Runs[r.AgentID] = r
	}
	rows.Close()
	return org, rows.Err()
}
