// Package optimalengine is the bridge's memory row on the Connections board.
// It replaces FounderOS v1's G-Brain row: GBrain is retired, and every
// workspace's memory lives in its home Optimal Engine (engine topology).
package optimalengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "optimal-engine", Name: "Optimal Engine", Kind: connectors.KindBrain}

// Engine is one engine the bridge routes workspaces to.
type Engine struct {
	Name string
	URL  string
	Key  string
}

type Connector struct {
	engines []Engine
	client  *http.Client
}

func New(engines []Engine) *Connector {
	return &Connector{engines: engines, client: connectors.HTTPClient(8 * time.Second)}
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	if len(c.engines) == 0 {
		return connectors.Status{State: connectors.StateNotConfigured, Detail: "no engines in the topology have an endpoint"}
	}
	up, workspaces := 0, 0
	var problems []string
	for _, e := range c.engines {
		n, err := c.check(ctx, e)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", e.Name, err))
			continue
		}
		up++
		workspaces += n
	}
	detail := fmt.Sprintf("%d/%d engines up · %d workspaces", up, len(c.engines), workspaces)
	if len(problems) > 0 {
		return connectors.Status{State: connectors.StateError, Detail: detail + " · " + strings.Join(problems, "; ")}
	}
	return connectors.Status{State: connectors.StateConnected, Detail: detail, Meta: map[string]any{"engines": up, "workspaces": workspaces}}
}

func (c *Connector) check(ctx context.Context, e Engine) (int, error) {
	var health struct {
		Status string `json:"status"`
	}
	if err := c.get(ctx, e, "/api/health", &health); err != nil {
		return 0, err
	}
	if health.Status != "up" {
		return 0, fmt.Errorf("health %q", health.Status)
	}
	var list struct {
		Workspaces []struct {
			Slug string `json:"slug"`
		} `json:"workspaces"`
	}
	if err := c.get(ctx, e, "/api/workspaces?status=active&limit=200", &list); err != nil {
		return 0, err
	}
	n := 0
	for _, w := range list.Workspaces {
		if w.Slug != "default" { // the engine's legacy catch-all, not a real workspace
			n++
		}
	}
	return n, nil
}

func (c *Connector) get(ctx context.Context, e Engine, path string, out any) error {
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
