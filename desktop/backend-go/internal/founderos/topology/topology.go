// Package topology decides which Optimal Engine is the home of each the operator
// workspace. Every workspace has exactly one home; personal and device data
// never leave a device engine. See config/founderos/engine-topology.yaml.
package topology

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// RepoPath is the topology file's location relative to the repository root.
const RepoPath = "config/founderos/engine-topology.yaml"

const (
	TierDevice = "device"
	TierShared = "shared"

	ClassBusiness = "business"
	ClassPersonal = "personal"
	ClassDevice   = "device"
)

type Engine struct {
	Name     string `yaml:"-"`
	Tier     string `yaml:"tier"`
	Deferred bool   `yaml:"deferred"`
	URLEnv   string `yaml:"url_env"`
	KeyEnv   string `yaml:"key_env"`
	ServedBy string `yaml:"served_by"`
	Note     string `yaml:"note"`
}

type Workspace struct {
	Slug  string `yaml:"slug"`
	Name  string `yaml:"name"`
	Home  string `yaml:"home"`
	Class string `yaml:"class"`
	Holds string `yaml:"holds"`
}

type BrainStore struct {
	Default string            `yaml:"default"`
	Routes  map[string]string `yaml:"routes"`
}

type Topology struct {
	Version    int               `yaml:"version"`
	Engines    map[string]Engine `yaml:"engines"`
	Workspaces []Workspace       `yaml:"workspaces"`
	BrainStore BrainStore        `yaml:"brain_store"`
	// Retired maps a retired workspace slug to the workspace it merges into.
	Retired map[string]string `yaml:"retired"`
	// MeetingRules route meetings/ pages by their frontmatter, first match
	// wins; unmatched meetings fall back to the folder route.
	MeetingRules []MeetingRule `yaml:"meeting_rules"`
}

type MeetingRule struct {
	Field     string `yaml:"field"` // a frontmatter key, e.g. title or participants
	Contains  string `yaml:"contains"`
	Workspace string `yaml:"workspace"`
}

func Parse(doc []byte) (*Topology, error) {
	var t Topology
	if err := yaml.Unmarshal(doc, &t); err != nil {
		return nil, fmt.Errorf("parse topology: %w", err)
	}
	for name, e := range t.Engines {
		e.Name = name
		t.Engines[name] = e
	}
	return &t, nil
}

func Load(path string) (*Topology, error) {
	doc, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(doc)
}

// LoadRepo finds the repository root by walking up from the working directory
// and loads the checked-in topology.
func LoadRepo() (*Topology, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(dir, RepoPath)
		if _, err := os.Stat(candidate); err == nil {
			return Load(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("%s not found above working directory", RepoPath)
		}
		dir = parent
	}
}

// Validate enforces the placement invariants and reports every violation.
func (t *Topology) Validate() error {
	var errs []error
	homes := map[string]int{}
	classOf := map[string]string{}
	for _, w := range t.Workspaces {
		homes[w.Slug]++
		classOf[w.Slug] = w.Class
		switch w.Class {
		case ClassBusiness, ClassPersonal, ClassDevice:
		default:
			errs = append(errs, fmt.Errorf("workspace %q: class %q is not business, personal or device", w.Slug, w.Class))
		}
		if w.Home == "" {
			errs = append(errs, fmt.Errorf("workspace %q has no home engine", w.Slug))
			continue
		}
		e, ok := t.Engines[w.Home]
		if !ok {
			errs = append(errs, fmt.Errorf("workspace %q: unknown engine %q", w.Slug, w.Home))
			continue
		}
		if e.Deferred {
			errs = append(errs, fmt.Errorf("workspace %q: engine %q is deferred and cannot be a home yet", w.Slug, w.Home))
		}
		if (w.Class == ClassPersonal || w.Class == ClassDevice) && e.Tier != TierDevice {
			errs = append(errs, fmt.Errorf("workspace %q is %s data and must live on a device engine, not %q", w.Slug, w.Class, w.Home))
		}
		if _, retired := t.Retired[w.Slug]; retired {
			errs = append(errs, fmt.Errorf("workspace %q is retired but still has a home", w.Slug))
		}
	}
	for slug, n := range homes {
		if n > 1 {
			errs = append(errs, fmt.Errorf("workspace %q has more than one home (%d entries)", slug, n))
		}
	}
	for slug, into := range t.Retired {
		if _, ok := homes[into]; !ok {
			errs = append(errs, fmt.Errorf("retired workspace %q merges into unknown workspace %q", slug, into))
		}
	}
	if _, ok := homes[t.BrainStore.Default]; !ok {
		errs = append(errs, fmt.Errorf("brain_store default %q is an unknown workspace", t.BrainStore.Default))
	}
	for folder, slug := range t.BrainStore.Routes {
		if _, ok := homes[slug]; !ok {
			errs = append(errs, fmt.Errorf("brain_store route %q points at unknown workspace %q", folder, slug))
		}
	}
	for i, r := range t.MeetingRules {
		if _, ok := homes[r.Workspace]; !ok {
			errs = append(errs, fmt.Errorf("meeting rule %d points at unknown workspace %q", i, r.Workspace))
		}
		if r.Field == "" || r.Contains == "" {
			errs = append(errs, fmt.Errorf("meeting rule %d needs field and contains", i))
		}
	}
	if slug, ok := t.BrainStore.Routes["conversations"]; ok && classOf[slug] != ClassPersonal {
		errs = append(errs, fmt.Errorf("brain_store conversations must route to a personal workspace, not %q", slug))
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errors.Join(errs...)
}

// HomeOf returns the engine that owns a workspace.
func (t *Topology) HomeOf(slug string) (Engine, error) {
	for _, w := range t.Workspaces {
		if w.Slug == slug {
			e, ok := t.Engines[w.Home]
			if !ok {
				return Engine{}, fmt.Errorf("workspace %q: unknown engine %q", slug, w.Home)
			}
			return e, nil
		}
	}
	if into, ok := t.Retired[slug]; ok {
		return Engine{}, fmt.Errorf("workspace %q is retired (merged into %q)", slug, into)
	}
	return Engine{}, fmt.Errorf("unknown workspace %q", slug)
}

// WorkspaceForBrainPath routes a brain-store page (path relative to the store
// root) to a workspace by its top-level folder.
func (t *Topology) WorkspaceForBrainPath(rel string) string {
	rel = filepath.ToSlash(rel)
	folder, _, nested := strings.Cut(rel, "/")
	if nested {
		if slug, ok := t.BrainStore.Routes[folder]; ok {
			return slug
		}
	}
	return t.BrainStore.Default
}

// WorkspaceForBrainPage routes a page using its path and, for meetings, its
// frontmatter header.
func (t *Topology) WorkspaceForBrainPage(rel, text string) string {
	folder, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
	if folder == "meetings" {
		header := frontmatter(text)
		for _, r := range t.MeetingRules {
			if strings.Contains(strings.ToLower(header[r.Field]), strings.ToLower(r.Contains)) {
				return r.Workspace
			}
		}
	}
	return t.WorkspaceForBrainPath(rel)
}

// frontmatter reads `key: value` lines from a page's header: the leading
// --- fenced block plus the metadata lines under the title, stopping at the
// first "## " section. Earlier keys win.
func frontmatter(text string) map[string]string {
	out := map[string]string{}
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") || i > 60 {
			break
		}
		if trimmed == "---" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if k, v, ok := strings.Cut(trimmed, ":"); ok && !strings.Contains(k, " ") {
			if _, seen := out[k]; !seen {
				out[k] = strings.Trim(strings.TrimSpace(v), `"`)
			}
		}
	}
	return out
}
