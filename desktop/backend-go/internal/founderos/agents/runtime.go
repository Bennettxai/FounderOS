// Package agents is the FounderOS agent runtime on the bridge: the Go port of
// FounderOS v1 lib/agents/runtime.ts, lib/agent-costs.ts and the cron
// scheduler. Every agent row maps 1:1 to an implementation; every run is
// stored with its cost; scheduled runs go through the bridge cron guard.
package agents

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrUnknownAgent = errors.New("unknown agent")

// Result is what one run produces (AgentRunResult).
type Result struct {
	OK        bool   `json:"ok"`
	Summary   string `json:"summary"`
	Data      any    `json:"data,omitempty"`
	Model     string `json:"model,omitempty"`
	TokensIn  *int   `json:"tokensIn,omitempty"`
	TokensOut *int   `json:"tokensOut,omitempty"`
}

type Meta struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	DepartmentID string `json:"departmentId"`
}

// Agent is a runnable the operator agent (RuntimeAgent).
type Agent interface {
	Meta() Meta
	Run(ctx context.Context) (Result, error)
}

// Responder is implemented by agents that answer a broadcast in their own
// voice; the rest reply with a run.
type Responder interface {
	Respond(ctx context.Context, message string) (Result, error)
}

// Run is one stored agent run (founderos_agent_runs).
type Run struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agentId"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	OK         bool      `json:"ok"`
	Summary    string    `json:"summary"`
	Model      *string   `json:"model"`
	TokensIn   *int      `json:"tokensIn"`
	TokensOut  *int      `json:"tokensOut"`
	CostUSD    *float64  `json:"costUsd"`
}

type Reply struct {
	AgentID    string    `json:"agentId"`
	OK         bool      `json:"ok"`
	Reply      string    `json:"reply"`
	FinishedAt time.Time `json:"finishedAt"`
}

type Broadcast struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
	Replies   []Reply   `json:"replies"`
}

// Store persists runs and broadcasts.
type Store interface {
	InsertRun(ctx context.Context, r Run) error
	Recent(ctx context.Context, agentID string, limit int) ([]Run, error)
	InsertBroadcast(ctx context.Context, b Broadcast) error
}

type Runtime struct {
	store  Store
	order  []string
	agents map[string]Agent
}

func New(store Store, list ...Agent) *Runtime {
	rt := &Runtime{store: store, agents: map[string]Agent{}}
	for _, a := range list {
		id := a.Meta().ID
		if _, dup := rt.agents[id]; dup {
			panic(fmt.Sprintf("agents: duplicate id %q", id))
		}
		rt.agents[id] = a
		rt.order = append(rt.order, id)
	}
	return rt
}

// List returns agent metadata in registration order.
func (rt *Runtime) List() []Meta {
	out := make([]Meta, 0, len(rt.order))
	for _, id := range rt.order {
		out = append(out, rt.agents[id].Meta())
	}
	return out
}

// Has reports whether an agent id is registered.
func (rt *Runtime) Has(id string) bool { _, ok := rt.agents[id]; return ok }

func (rt *Runtime) Recent(ctx context.Context, agentID string, limit int) ([]Run, error) {
	return rt.store.Recent(ctx, agentID, limit)
}

// Run executes one agent and stores the run whatever happens, including a
// panic inside the agent.
func (rt *Runtime) Run(ctx context.Context, id string) (Run, error) {
	a, ok := rt.agents[id]
	if !ok {
		return Run{}, fmt.Errorf("%w: %s", ErrUnknownAgent, id)
	}
	started := time.Now().UTC()
	res := safely(func() (Result, error) { return a.Run(ctx) })
	run := Run{
		ID: uuid.NewString(), AgentID: id, StartedAt: started, FinishedAt: time.Now().UTC(),
		OK: res.OK, Summary: res.Summary, TokensIn: res.TokensIn, TokensOut: res.TokensOut,
	}
	if res.Model != "" {
		m := res.Model
		run.Model = &m
	}
	if res.TokensIn != nil || res.TokensOut != nil {
		c := RunCostUSD(deref(res.TokensIn), deref(res.TokensOut), res.Model)
		run.CostUSD = &c
	}
	pctx, cancel := persistCtx(ctx)
	defer cancel()
	if err := rt.store.InsertRun(pctx, run); err != nil {
		return run, fmt.Errorf("store run: %w", err)
	}
	return run, nil
}

// persistTimeout bounds a save that has outlived its request.
const persistTimeout = 10 * time.Second

// persistCtx detaches a save from the caller's cancellation: FounderOS v1 always
// stores the run, so a client hanging up mid-run must not lose it. The save
// keeps the caller's values and gets its own short deadline.
func persistCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), persistTimeout)
}

// Broadcast speaks to every agent at once; each replies in parallel.
func (rt *Runtime) Broadcast(ctx context.Context, message string) (Broadcast, error) {
	b := Broadcast{ID: uuid.NewString(), Message: message, CreatedAt: time.Now().UTC(), Replies: make([]Reply, len(rt.order))}
	var wg sync.WaitGroup
	for i, id := range rt.order {
		wg.Add(1)
		go func(i int, a Agent) {
			defer wg.Done()
			res := safely(func() (Result, error) {
				if r, ok := a.(Responder); ok {
					return r.Respond(ctx, message)
				}
				return a.Run(ctx)
			})
			b.Replies[i] = Reply{AgentID: a.Meta().ID, OK: res.OK, Reply: res.Summary, FinishedAt: time.Now().UTC()}
		}(i, rt.agents[id])
	}
	wg.Wait()
	pctx, cancel := persistCtx(ctx)
	defer cancel()
	return b, rt.store.InsertBroadcast(pctx, b)
}

func safely(fn func() (Result, error)) (res Result) {
	defer func() {
		if p := recover(); p != nil {
			res = Result{OK: false, Summary: fmt.Sprintf("agent panicked: %v", p)}
		}
	}()
	r, err := fn()
	if err != nil {
		return Result{OK: false, Summary: err.Error(), Model: r.Model, TokensIn: r.TokensIn, TokensOut: r.TokensOut}
	}
	return r
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ---- costs (lib/agent-costs.ts) ----------------------------------------------

type pricing struct{ inPerM, outPerM float64 }

var modelPricing = map[string]pricing{
	"claude-sonnet-5":  {3, 15},
	"sonnet-5":         {3, 15},
	"sonnet-4.5":       {3, 15},
	"claude-haiku-4.5": {0.8, 4},
	"haiku-4.5":        {0.8, 4},
	"claude-opus-4.8":  {15, 75},
	"opus-4.8":         {15, 75},
}

var defaultPricing = pricing{3, 15}

// RunCostUSD prices a run's tokens; unknown models fall back to Sonnet.
func RunCostUSD(tokensIn, tokensOut int, model string) float64 {
	p := defaultPricing
	if model != "" {
		bare := strings.TrimPrefix(model, "anthropic/")
		if v, ok := modelPricing[bare]; ok {
			p = v
		} else if v, ok := modelPricing[model]; ok {
			p = v
		}
	}
	in, out := float64(max(tokensIn, 0)), float64(max(tokensOut, 0))
	return in/1e6*p.inPerM + out/1e6*p.outPerM
}

// ---- memory store (tests and dev) -------------------------------------------

type MemStore struct {
	mu         sync.Mutex
	runs       []Run
	broadcasts []Broadcast
}

func NewMemStore() *MemStore { return &MemStore{} }

func (m *MemStore) InsertRun(_ context.Context, r Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = append(m.runs, r)
	return nil
}

func (m *MemStore) Recent(_ context.Context, agentID string, limit int) ([]Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Run
	for _, r := range m.runs {
		if agentID == "" || r.AgentID == agentID {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemStore) InsertBroadcast(_ context.Context, b Broadcast) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, b)
	return nil
}
