package console

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// EngineRef is one topology engine: its endpoint (empty when it is not
// staged on this machine) and the workspaces homed there.
type EngineRef struct {
	Name  string
	URL   string
	Key   string
	Homes []string
}

// EngineReading is the compact brain core's row for one engine.
type EngineReading struct {
	Name       string   `json:"name"`
	State      string   `json:"state"` // connected | error | not_configured
	Up         bool     `json:"up"`
	Detail     string   `json:"detail"`
	Workspaces *int     `json:"workspaces"` // live count on the engine; nil = unknown
	Homes      []string `json:"homes"`      // topology workspaces homed here
	Health     *int     `json:"health"`     // 0-100 from its own /api/health checks; nil = not read
	Warnings   bool     `json:"warnings"`   // a check failing or a subsystem degraded
}

// Brain is the Optimal Engine as the console sees it. Engines with no
// endpoint on this machine are listed but not counted.
type Brain struct {
	Connected    bool `json:"connected"`
	EnginesUp    int  `json:"enginesUp"`
	EnginesTotal int  `json:"enginesTotal"`
	Workspaces   *int `json:"workspaces"`
	// Health is where G-Brain's doctor score stood: the share of the engines'
	// own health checks passing, averaged over the staged engines (a down
	// engine scores 0). nil when no engine answers.
	Health *int `json:"health"`
	// Status is the doctor's word for it: ok | warnings | offline | not configured.
	Status  string          `json:"status"`
	Engines []EngineReading `json:"engines"`
}

// EngineHealth scores one engine from its /api/health checks: the share
// passing, 0-100. An engine that is up and reports no checks is healthy.
func EngineHealth(checks map[string]string, degraded []string) (score int, warnings bool) {
	if len(checks) == 0 {
		return 100, len(degraded) > 0
	}
	ok := 0
	for _, v := range checks {
		switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v)), ":") {
		case "ok", "true", "up":
			ok++
		}
	}
	score = int(float64(ok)/float64(len(checks))*100 + 0.5)
	return score, ok < len(checks) || len(degraded) > 0
}

var engineClient = connectors.HTTPClient(6 * time.Second)

func getJSON(ctx context.Context, e EngineRef, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.URL, "/")+path, nil)
	if err != nil {
		return err
	}
	if e.Key != "" {
		req.Header.Set("Authorization", "Bearer "+e.Key)
	}
	resp, err := engineClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func probe(ctx context.Context, e EngineRef) EngineReading {
	r := EngineReading{Name: e.Name, Homes: e.Homes}
	if r.Homes == nil {
		r.Homes = []string{}
	}
	if e.URL == "" {
		r.State, r.Detail = "not_configured", "not staged on this machine"
		return r
	}
	var health struct {
		Status   string            `json:"status"`
		Checks   map[string]string `json:"checks"`
		Degraded []string          `json:"degraded"`
	}
	if err := getJSON(ctx, e, "/api/health", &health); err != nil {
		r.State, r.Detail = "error", "unreachable: "+err.Error()
		return r
	}
	if health.Status != "up" {
		r.State, r.Detail = "error", fmt.Sprintf("health %q", health.Status)
		return r
	}
	var list struct {
		Workspaces []struct {
			Slug string `json:"slug"`
		} `json:"workspaces"`
	}
	if err := getJSON(ctx, e, "/api/workspaces?status=active&limit=200", &list); err != nil {
		r.State, r.Detail = "error", "workspaces unreadable: "+err.Error()
		return r
	}
	n := 0
	for _, w := range list.Workspaces {
		if w.Slug != "default" { // the engine's legacy catch-all
			n++
		}
	}
	score, warn := EngineHealth(health.Checks, health.Degraded)
	r.State, r.Up, r.Workspaces, r.Detail = "connected", true, &n, fmt.Sprintf("%d workspaces", n)
	r.Health, r.Warnings = &score, warn
	return r
}

// ProbeEngines reads each staged engine's health and workspaces (GETs only).
func ProbeEngines(ctx context.Context, engines []EngineRef) Brain {
	b := Brain{Engines: make([]EngineReading, len(engines))}
	var wg sync.WaitGroup
	for i, e := range engines {
		wg.Add(1)
		go func(i int, e EngineRef) {
			defer wg.Done()
			b.Engines[i] = probe(ctx, e)
		}(i, e)
	}
	wg.Wait()
	total, scoreSum, warn := 0, 0, false
	for _, r := range b.Engines {
		if r.State == "not_configured" {
			continue
		}
		b.EnginesTotal++
		if r.Up {
			b.EnginesUp++
			total += *r.Workspaces
			if r.Health != nil {
				scoreSum += *r.Health
			}
			warn = warn || r.Warnings
		}
	}
	if b.EnginesUp > 0 {
		b.Workspaces = &total
		h := int(float64(scoreSum)/float64(b.EnginesTotal) + 0.5)
		b.Health = &h
	}
	b.Connected = b.EnginesTotal > 0 && b.EnginesUp == b.EnginesTotal
	switch {
	case b.EnginesTotal == 0:
		b.Status = "not configured"
	case b.EnginesUp == 0:
		b.Status = "offline"
	case b.Connected && !warn && *b.Health == 100:
		b.Status = "ok"
	default:
		b.Status = "warnings"
	}
	return b
}
