// Package doctor is the /doctor page's logic. In FounderOS v1 the Doctor
// reported GBrain's health (gbrain doctor, the brain-store walk, Supabase).
// GBrain is retired on the bridge, so the same page now reports Optimal
// Engine health: every engine in the topology, its workspaces, its store
// audit, and how many claims have been promoted to facts. Every read is a GET;
// an engine that does not answer is reported unreachable, never as empty.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Engine is one engine the bridge routes workspaces to.
type Engine struct {
	Name string
	URL  string
	Key  string
}

// Check is one doctor line: an engine health check, a degraded subsystem,
// or a named /api/stores/audit check. Status is ok | warn | error.
type Check struct {
	Engine  string `json:"engine"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Store is one logical store of an engine (/api/stores).
type Store struct {
	ID         string           `json:"id"`
	Status     string           `json:"status"`
	Technology string           `json:"technology"`
	RowCount   int64            `json:"rowCount"`
	Tables     map[string]int64 `json:"tables"`
}

// EngineReading is everything one engine said on this read. When Reachable
// is false, Error says why and every collection is nil (unknown), not empty.
type EngineReading struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Reachable  bool     `json:"reachable"`
	Error      string   `json:"error,omitempty"`
	Health     string   `json:"health,omitempty"`
	Workspaces []string `json:"workspaces"`
	Checks     []Check  `json:"checks"`
	Stores     []Store  `json:"stores"`
	// Searches is the engine's search.query counter since boot.
	Searches *float64 `json:"searches"`
	UptimeMs *int64   `json:"uptimeMs"`
}

type healthBody struct {
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
	Degraded []string          `json:"degraded"`
}

// Read asks every engine, concurrently, and keeps topology order.
func Read(ctx context.Context, client *http.Client, engines []Engine) []EngineReading {
	out := make([]EngineReading, len(engines))
	var wg sync.WaitGroup
	for i, e := range engines {
		wg.Add(1)
		go func(i int, e Engine) {
			defer wg.Done()
			out[i] = readOne(ctx, client, e)
		}(i, e)
	}
	wg.Wait()
	return out
}

func readOne(ctx context.Context, client *http.Client, e Engine) EngineReading {
	r := EngineReading{Name: e.Name, URL: e.URL}
	fail := func(what string, err error) EngineReading {
		return EngineReading{Name: e.Name, URL: e.URL, Error: fmt.Sprintf("%s: %v", what, err)}
	}

	var h healthBody
	if err := get(ctx, client, e, "/api/health", &h); err != nil {
		return fail("health", err)
	}
	var ws struct {
		Workspaces []struct {
			Slug string `json:"slug"`
		} `json:"workspaces"`
	}
	if err := get(ctx, client, e, "/api/workspaces?status=active&limit=200", &ws); err != nil {
		return fail("workspaces", err)
	}
	var stores struct {
		Stores []struct {
			ID         string           `json:"id"`
			Status     string           `json:"status"`
			Technology string           `json:"technology"`
			RowCount   int64            `json:"row_count"`
			Tables     map[string]int64 `json:"table_counts"`
		} `json:"stores"`
	}
	if err := get(ctx, client, e, "/api/stores", &stores); err != nil {
		return fail("stores", err)
	}
	var audit struct {
		Checks []struct {
			Name   string          `json:"name"`
			OK     bool            `json:"ok"`
			Detail json.RawMessage `json:"detail"`
		} `json:"checks"`
	}
	// A failing audit answers 503 with its checks in the body: that is a
	// reading, not an unreachable engine.
	if err := get(ctx, client, e, "/api/stores/audit", &audit, http.StatusServiceUnavailable); err != nil {
		return fail("audit", err)
	}

	r.Reachable = true
	r.Health = h.Status
	r.Workspaces = []string{}
	for _, w := range ws.Workspaces {
		if w.Slug != "default" { // the engine's legacy catch-all, not a workspace
			r.Workspaces = append(r.Workspaces, w.Slug)
		}
	}
	r.Stores = []Store{}
	for _, s := range stores.Stores {
		if s.Tables == nil {
			s.Tables = map[string]int64{}
		}
		r.Stores = append(r.Stores, Store{ID: s.ID, Status: s.Status, Technology: s.Technology, RowCount: s.RowCount, Tables: s.Tables})
	}
	r.Checks = healthChecks(e.Name, h)
	for _, c := range audit.Checks {
		msg := detailText(c.Detail)
		st := "ok"
		if !c.OK {
			st = "error"
			if strings.Contains(msg, `"optional":true`) {
				st = "warn"
			}
		}
		r.Checks = append(r.Checks, Check{Engine: e.Name, Name: c.Name, Status: st, Message: msg})
	}

	// Metrics are nice to have: an engine without them is still healthy.
	var m struct {
		Counters map[string]float64 `json:"counters"`
		UptimeMs *int64             `json:"uptime_ms"`
	}
	if err := get(ctx, client, e, "/api/metrics", &m); err == nil {
		if q, ok := m.Counters["optimal_engine.search.query"]; ok {
			r.Searches = &q
		}
		r.UptimeMs = m.UptimeMs
	}
	return r
}

// healthChecks turns /api/health into doctor lines: each named check
// (":ok" passes) and each degraded subsystem as a warning.
func healthChecks(engine string, h healthBody) []Check {
	names := make([]string, 0, len(h.Checks))
	for k := range h.Checks {
		names = append(names, k)
	}
	sort.Strings(names)
	out := []Check{}
	for _, k := range names {
		v := h.Checks[k]
		st := "ok"
		if v != ":ok" && v != "ok" {
			st = "error"
		}
		out = append(out, Check{Engine: engine, Name: "health." + k, Status: st, Message: strings.TrimPrefix(v, ":")})
	}
	for _, d := range h.Degraded {
		out = append(out, Check{Engine: engine, Name: "degraded." + d, Status: "warn", Message: "subsystem degraded"})
	}
	return out
}

// detailText renders an audit detail: strings bare, objects as compact JSON
// with sorted keys.
func detailText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	b, _ := json.Marshal(v) // map keys marshal sorted
	return string(b)
}

// get decodes a 200 (or any of also) into out.
func get(ctx context.Context, client *http.Client, e Engine, path string, out any, also ...int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.URL, "/")+path, nil)
	if err != nil {
		return err
	}
	if e.Key != "" {
		req.Header.Set("Authorization", "Bearer "+e.Key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	ok := resp.StatusCode == http.StatusOK
	for _, code := range also {
		ok = ok || resp.StatusCode == code
	}
	if !ok {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("HTTP %d: %w", resp.StatusCode, err)
	}
	return nil
}
