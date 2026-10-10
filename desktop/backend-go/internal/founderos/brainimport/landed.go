package brainimport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type target struct{ engine, workspace string }

// WithoutLanded drops the items whose title the engine already holds in the
// item's workspace. Unlike the ledger, the engine itself is the record, so a
// re-run ingests nothing even from a fresh machine, and a reset engine gets
// everything again. An engine that cannot be read fails the check.
func WithoutLanded(items []Item, engines map[string]Endpoint) ([]Item, int, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	have := map[target]map[string]bool{}
	for _, it := range items {
		k := target{it.Engine, it.Workspace}
		if _, done := have[k]; done {
			continue
		}
		ep, ok := engines[it.Engine]
		if !ok {
			return nil, 0, fmt.Errorf("no endpoint configured for engine %s", it.Engine)
		}
		titles, err := landedTitles(client, ep, it.Workspace)
		if err != nil {
			return nil, 0, fmt.Errorf("read %s on %s: %w", it.Workspace, it.Engine, err)
		}
		have[k] = titles
	}
	var rest []Item
	landed := 0
	for _, it := range items {
		if have[target{it.Engine, it.Workspace}][it.Title] {
			landed++
			continue
		}
		rest = append(rest, it)
	}
	return rest, landed, nil
}

func landedTitles(client *http.Client, ep Endpoint, workspace string) (map[string]bool, error) {
	resp, err := do(client, ep, http.MethodGet, "/api/graph?workspace="+url.QueryEscape("default:"+workspace), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Contexts []struct {
			Title string `json:"title"`
		} `json:"contexts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	titles := map[string]bool{}
	for _, c := range out.Contexts {
		titles[c.Title] = true
	}
	return titles, nil
}

// EnsureEntities registers every page's slug (its file name) as an entity
// node in the page's workspace. The engine extracts entities from ingested
// text by matching known node names, so a page that links [[stripe]] gains
// the entity "stripe", and pages sharing an entity are joined in the graph:
// the wikilinks become the knowledge graph's edges. Run it before ingesting.
// Upserts, so re-runs are harmless.
func EnsureEntities(items []Item, engines map[string]Endpoint) error {
	client := &http.Client{Timeout: 15 * time.Second}
	type node struct{ engine, workspace, slug string }
	seen := map[node]bool{}
	var nodes []node
	for _, it := range items {
		base := filepath.Base(it.Rel)
		slug := strings.TrimSuffix(base, filepath.Ext(base))
		if generic(TitleKey(slug)) {
			continue
		}
		n := node{it.Engine, it.Workspace, slug}
		if !seen[n] {
			seen[n] = true
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].slug < nodes[j].slug })
	for _, n := range nodes {
		ep, ok := engines[n.engine]
		if !ok {
			return fmt.Errorf("no endpoint configured for engine %s", n.engine)
		}
		body, _ := json.Marshal(map[string]string{"name": n.slug, "slug": n.slug, "kind": "entity", "workspace": "default:" + n.workspace})
		resp, err := do(client, ep, http.MethodPost, "/api/nodes", body)
		if err != nil {
			return fmt.Errorf("entity %s on %s: %w", n.slug, n.engine, err)
		}
		resp.Body.Close()
	}
	return nil
}
