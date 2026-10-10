package brainimport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// EnsureWorkspaces creates, on each configured engine, every workspace the
// topology homes there that the engine does not have yet. Engines without an
// endpoint are skipped.
func EnsureWorkspaces(topo *topology.Topology, engines map[string]Endpoint) error {
	client := &http.Client{Timeout: 15 * time.Second}
	for _, w := range topo.Workspaces {
		ep, ok := engines[w.Home]
		if !ok {
			continue
		}
		have, err := listSlugs(client, ep)
		if err != nil {
			return fmt.Errorf("list workspaces on %s: %w", w.Home, err)
		}
		if have[w.Slug] {
			continue
		}
		name := w.Name
		if name == "" {
			name = w.Slug
		}
		body, _ := json.Marshal(map[string]string{"slug": w.Slug, "name": name, "description": w.Holds})
		resp, err := do(client, ep, http.MethodPost, "/api/workspaces", body)
		if err != nil {
			return fmt.Errorf("create %s on %s: %w", w.Slug, w.Home, err)
		}
		resp.Body.Close()
	}
	return nil
}

func listSlugs(client *http.Client, ep Endpoint) (map[string]bool, error) {
	resp, err := do(client, ep, http.MethodGet, "/api/workspaces?limit=200", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Workspaces []struct {
			Slug string `json:"slug"`
		} `json:"workspaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	have := map[string]bool{}
	for _, w := range out.Workspaces {
		have[w.Slug] = true
	}
	return have, nil
}

// maxAttempts bounds retries when the engine answers 429 rate_limited.
const maxAttempts = 8

func do(client *http.Client, ep Endpoint, method, path string, body []byte) (*http.Response, error) {
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequest(method, strings.TrimRight(ep.URL, "/")+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if ep.Key != "" {
			req.Header.Set("Authorization", "Bearer "+ep.Key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode/100 == 2 {
			return resp, nil
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts {
			time.Sleep(retryAfter(raw, attempt))
			continue
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
}

// retryAfter honours the engine's retry_after_ms, with a growing floor.
func retryAfter(raw []byte, attempt int) time.Duration {
	var hint struct {
		RetryAfterMS int `json:"retry_after_ms"`
	}
	_ = json.Unmarshal(raw, &hint)
	wait := time.Duration(hint.RetryAfterMS) * time.Millisecond
	if floor := time.Duration(attempt*attempt) * 50 * time.Millisecond; wait < floor {
		wait = floor
	}
	return wait
}
