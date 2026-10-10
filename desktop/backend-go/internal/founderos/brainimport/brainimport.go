// Package brainimport moves GBrain brain-store pages into Optimal Engine.
// Each page goes to the home engine of its workspace (per the engine
// topology) through the engine's own /api/ingest, and a content-hash ledger
// makes re-runs ingest nothing new.
package brainimport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// MinChars skips README scaffolds and empty stubs.
const MinChars = 150

var genreByFolder = map[string]string{
	"sops":     "sop",
	"tools":    "reference",
	"org":      "reference",
	"meetings": "transcript",
}

var nodeByFolder = map[string]string{
	"tools": "tech", "agents": "tech", "sops": "tech", "org": "tech", "hiring": "tech",
	"inbox": "inbox", "archive": "inbox",
}

// Item is one unit of text headed for one engine workspace.
type Item struct {
	Rel       string `json:"rel"`
	Workspace string `json:"workspace"`
	Engine    string `json:"engine"`
	Title     string `json:"title"`
	Genre     string `json:"genre"`
	Node      string `json:"node"`
	Hash      string `json:"hash"`
	Text      string `json:"-"`
}

// Endpoint is how to reach one engine.
type Endpoint struct {
	URL string
	Key string
}

type Failure struct {
	Rel    string `json:"rel"`
	Reason string `json:"reason"`
}

type Report struct {
	Planned    int                       `json:"planned"`
	Ingested   int                       `json:"ingested"`
	Duplicates int                       `json:"duplicates"`
	Errors     []Failure                 `json:"errors"`
	ByTarget   map[string]map[string]int `json:"by_target"` // engine → workspace → ingested
}

// Plan walks a brain-store and routes every markdown page by topology. It
// returns the pages to send and how many were skipped as too short.
func Plan(store string, topo *topology.Topology) ([]Item, int, error) {
	var items []Item
	skipped := 0
	err := filepath.WalkDir(store, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != store && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		if len(strings.TrimSpace(text)) < MinChars {
			skipped++
			return nil
		}
		rel, _ := filepath.Rel(store, path)
		rel = filepath.ToSlash(rel)
		folder, _, nested := strings.Cut(rel, "/")
		if !nested {
			folder = ""
		}
		it, err := NewItem(topo, rel, topo.WorkspaceForBrainPage(rel, text), text)
		if err != nil {
			return err
		}
		if g, ok := genreByFolder[folder]; ok {
			it.Genre = g
		}
		if n, ok := nodeByFolder[folder]; ok {
			it.Node = n
		}
		items = append(items, it)
		return nil
	})
	sort.Slice(items, func(i, j int) bool { return items[i].Rel < items[j].Rel })
	return items, skipped, err
}

// NewItem routes text to a workspace's home engine, following a retired
// workspace to the one it merges into.
func NewItem(topo *topology.Topology, rel, workspace, text string) (Item, error) {
	if into, ok := topo.Retired[workspace]; ok {
		workspace = into
	}
	engine, err := topo.HomeOf(workspace)
	if err != nil {
		return Item{}, fmt.Errorf("%s: %w", rel, err)
	}
	sum := sha256.Sum256([]byte(text))
	return Item{
		Rel:       rel,
		Workspace: workspace,
		Engine:    engine.Name,
		Title:     titleOf(rel, text),
		Genre:     "note",
		Node:      "knowledge-base",
		Hash:      hex.EncodeToString(sum[:]),
		Text:      text,
	}, nil
}

func titleOf(rel, text string) string {
	if t := h1(text); t != "" {
		return t
	}
	if strings.HasPrefix(rel, "source_packages/") {
		// No heading and no file name: the first line is the best title.
		for _, line := range strings.Split(text, "\n") {
			if line = strings.TrimSpace(line); line != "" && line != "---" {
				if len(line) > 80 {
					line = line[:80]
				}
				return line
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
}

func h1(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(line), "# "); ok && strings.TrimSpace(t) != "" {
			return strings.TrimSpace(t)
		}
	}
	return ""
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

// TitleKey folds a title or file name for matching; "" means no usable title.
func TitleKey(s string) string {
	return strings.Trim(nonWord.ReplaceAllString(strings.ToLower(s), " "), " ")
}

func generic(k string) bool { return k == "" || k == "untitled" || k == "readme" }

// StoreTitleKeys collects the file-name and H1 keys of every store page.
func StoreTitleKeys(store string) (map[string]bool, error) {
	keys := map[string]bool{}
	err := filepath.WalkDir(store, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != store && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		for _, k := range []string{TitleKey(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())))} {
			if !generic(k) {
				keys[k] = true
			}
		}
		if raw, err := os.ReadFile(p); err == nil {
			if k := TitleKey(h1(string(raw))); !generic(k) {
				keys[k] = true
			}
		}
		return nil
	})
	return keys, err
}

// WithoutSuperseded drops snapshot items that are older copies of store
// pages (same title or file name); the store import carries the fresh copy.
func WithoutSuperseded(items []Item, storeKeys map[string]bool) ([]Item, int) {
	var kept []Item
	dropped := 0
	for _, it := range items {
		superseded := false
		for _, k := range []string{TitleKey(it.Title), TitleKey(h1(it.Text))} {
			if !generic(k) && storeKeys[k] {
				superseded = true
				break
			}
		}
		if superseded {
			dropped++
			continue
		}
		kept = append(kept, it)
	}
	return kept, dropped
}

// ledger records what already landed: hash|workspace → time.
type ledger map[string]string

func ledgerKey(it Item) string { return it.Hash + "|" + it.Workspace }

func loadLedger(path string) (ledger, error) {
	l := ledger{}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("ledger %s: %w", path, err)
	}
	return l, nil
}

func (l ledger) save(path string) error {
	raw, err := json.MarshalIndent(l, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Run sends every item not already in the ledger to its engine. dryRun counts
// what would be sent and touches neither the engines nor the ledger.
func Run(items []Item, engines map[string]Endpoint, ledgerPath string, dryRun bool) (*Report, error) {
	var missing []string
	seen := map[string]bool{}
	for _, it := range items {
		if _, ok := engines[it.Engine]; !ok && !seen[it.Engine] {
			seen[it.Engine] = true
			missing = append(missing, it.Engine)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("no endpoint configured for engine(s): %s", strings.Join(missing, ", "))
	}
	led, err := loadLedger(ledgerPath)
	if err != nil {
		return nil, err
	}
	rep := &Report{ByTarget: map[string]map[string]int{}}
	client := &http.Client{Timeout: 120 * time.Second}
	for _, it := range items {
		if _, done := led[ledgerKey(it)]; done {
			rep.Duplicates++
			continue
		}
		rep.Planned++
		if dryRun {
			continue
		}
		if err := ingest(client, engines[it.Engine], it); err != nil {
			rep.Errors = append(rep.Errors, Failure{Rel: it.Rel, Reason: err.Error()})
			continue
		}
		led[ledgerKey(it)] = time.Now().UTC().Format(time.RFC3339)
		rep.Ingested++
		if rep.ByTarget[it.Engine] == nil {
			rep.ByTarget[it.Engine] = map[string]int{}
		}
		rep.ByTarget[it.Engine][it.Workspace]++
		// Save as we go so a crash mid-run never re-sends what already landed.
		if err := led.save(ledgerPath); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

func ingest(client *http.Client, ep Endpoint, it Item) error {
	body, _ := json.Marshal(map[string]any{
		"text":           it.Text,
		"title":          it.Title,
		"genre":          it.Genre,
		"node":           it.Node,
		"workspace":      "default:" + it.Workspace,
		"extract_claims": true,
	})
	resp, err := do(client, ep, http.MethodPost, "/api/ingest", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var out struct {
		OK *bool `json:"ok"`
	}
	if json.Unmarshal(raw, &out) == nil && out.OK != nil && !*out.OK {
		return fmt.Errorf("engine refused: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}
