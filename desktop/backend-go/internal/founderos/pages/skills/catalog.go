// Package skills serves /os/skills: the operator's real Claude Code skills read
// from the bridge host's ~/.claude (user skills plus every installed
// plugin's), the operator skills table, and the Skill Volume slab numbers.
// A port of FounderOS v1 lib/skills-catalog.ts and lib/skills-volume.ts.
package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// CatalogSkill is one SKILL.md on disk (metadata only; the body loads on demand).
type CatalogSkill struct {
	Slug        string `json:"slug"` // directory name, or plugin:skill
	Name        string `json:"name"`
	Description string `json:"description"`
	Group       string `json:"group"`
	Path        string `json:"path"` // ~-relative path to the SKILL.md
}

// Dirs says where to read. Tests point it at a fixture home.
type Dirs struct {
	Home    string
	Skills  string // ~/.claude/skills
	Plugins string // ~/.claude/plugins (installed_plugins.json + cache)
}

// DefaultDirs resolves per call, honouring FounderOS v1's FOUNDEROS_OS_SKILLS_DIR
// and FOUNDEROS_OS_PLUGINS_DIR overrides from the process env.
func DefaultDirs() Dirs { return DirsFrom(os.Getenv) }

// DirsFrom is DefaultDirs with the overrides read through lookup (the bridge
// passes its connectors Resolver, so env.local and planted keys count).
func DirsFrom(lookup func(string) string) Dirs {
	home, _ := os.UserHomeDir()
	d := Dirs{Home: home, Skills: lookup("FOUNDEROS_OS_SKILLS_DIR"), Plugins: lookup("FOUNDEROS_OS_PLUGINS_DIR")}
	if d.Skills == "" {
		d.Skills = filepath.Join(home, ".claude", "skills")
	}
	if d.Plugins == "" {
		d.Plugins = filepath.Join(home, ".claude", "plugins")
	}
	return d
}

var slugRE = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func validHalf(s string) bool { return slugRE.MatchString(s) && s != "." && s != ".." }

// Frontmatter holds the two fields the catalog reads; nil means absent.
type Frontmatter struct {
	Name        *string
	Description *string
}

var fmRE = regexp.MustCompile(`^---\r?\n([\s\S]*?)\r?\n---`)
var blockRE = regexp.MustCompile(`^[|>][+-]?$`)
var leadWS = regexp.MustCompile(`^\s*`)

// readField reads one frontmatter field: inline, quoted, or a block scalar (| / >).
func readField(fm, key string) *string {
	lines := strings.Split(fm, "\n")
	prefix := key + ":"
	for i, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		inline := strings.TrimSpace(line[len(prefix):])
		if blockRE.MatchString(inline) {
			var block []string
			for _, l := range lines[i+1:] {
				if strings.TrimSpace(l) == "" || (len(l) > 0 && (l[0] == ' ' || l[0] == '\t')) {
					block = append(block, l)
				} else {
					break
				}
			}
			strip := -1
			for _, l := range block {
				if strings.TrimSpace(l) == "" {
					continue
				}
				if n := len(leadWS.FindString(l)); strip < 0 || n < strip {
					strip = n
				}
			}
			if strip < 0 {
				strip = 0
			}
			dedented := make([]string, len(block))
			for j, l := range block {
				if len(l) >= strip {
					dedented[j] = l[strip:]
				} else {
					dedented[j] = ""
				}
			}
			sep := "\n"
			if strings.HasPrefix(inline, ">") {
				sep = " "
			}
			out := strings.TrimSpace(strings.Join(dedented, sep))
			return &out
		}
		// one leading and one trailing quote, as /^["']|["']$/g does
		out := inline
		if strings.HasPrefix(out, `"`) || strings.HasPrefix(out, `'`) {
			out = out[1:]
		}
		if strings.HasSuffix(out, `"`) || strings.HasSuffix(out, `'`) {
			out = out[:len(out)-1]
		}
		return &out
	}
	return nil
}

// ParseFrontmatter reads name and description from a SKILL.md.
func ParseFrontmatter(md string) Frontmatter {
	m := fmRE.FindStringSubmatch(md)
	if m == nil {
		return Frontmatter{}
	}
	fm := strings.ReplaceAll(m[1], "\r\n", "\n")
	return Frontmatter{Name: readField(fm, "name"), Description: readField(fm, "description")}
}

// Group buckets a user skill for display.
func Group(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "firecrawl"):
		return "Firecrawl"
	case n == "build" || n == "spec" || n == "review":
		return "Spec · build · review"
	case n == "codex" || n == "mcp-builder":
		return "Engineering"
	case n == "nano-banana":
		return "Creative"
	case n == "proposal-generator":
		return "Sales"
	}
	return "Skills"
}

func deref(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func sortSkills(out []CatalogSkill) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		return out[i].Name < out[j].Name
	})
}

// ReadUserSkills lists ~/.claude/skills/<name>/SKILL.md. Every entry is tried
// (no directory gate: many skills are symlinks); plain files and dirs with no
// SKILL.md drop out. Empty when the directory is absent.
func ReadUserSkills(d Dirs) []CatalogSkill {
	entries, err := os.ReadDir(d.Skills)
	if err != nil {
		return []CatalogSkill{}
	}
	out := []CatalogSkill{}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(d.Skills, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		fm := ParseFrontmatter(string(raw))
		name := deref(fm.Name, e.Name())
		out = append(out, CatalogSkill{
			Slug:        e.Name(),
			Name:        name,
			Description: deref(fm.Description, ""),
			Group:       Group(name),
			Path:        "~/.claude/skills/" + e.Name() + "/SKILL.md",
		})
	}
	sortSkills(out)
	return out
}

type pluginPath struct{ name, installPath string }

// installedPlugins maps plugin name → live install path through
// installed_plugins.json, in manifest order, so stale cached versions are
// never listed. Empty on any read failure.
func installedPlugins(dir string) []pluginPath {
	raw, err := os.ReadFile(filepath.Join(dir, "installed_plugins.json"))
	if err != nil {
		return nil
	}
	// Decode keys in file order: plugin order is the manifest's.
	var manifest struct {
		Plugins json.RawMessage `json:"plugins"`
	}
	if json.Unmarshal(raw, &manifest) != nil || len(manifest.Plugins) == 0 {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(string(manifest.Plugins)))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	seen := map[string]bool{}
	var out []pluginPath
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		key, _ := tok.(string)
		var entries []struct {
			InstallPath *string `json:"installPath"`
		}
		if err := dec.Decode(&entries); err != nil {
			continue
		}
		name := strings.SplitN(key, "@", 2)[0]
		if !validHalf(name) || seen[name] || len(entries) == 0 || entries[0].InstallPath == nil {
			continue
		}
		seen[name] = true
		out = append(out, pluginPath{name, *entries[0].InstallPath})
	}
	return out
}

func tildePath(home, p string) string {
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// ReadPluginSkills lists the skills of every installed plugin, slugged
// plugin:skill like the CLI names them. Empty when the manifest is absent.
func ReadPluginSkills(d Dirs) []CatalogSkill {
	out := []CatalogSkill{}
	for _, p := range installedPlugins(d.Plugins) {
		entries, err := os.ReadDir(filepath.Join(p.installPath, "skills"))
		if err != nil {
			continue
		}
		for _, e := range entries {
			file := filepath.Join(p.installPath, "skills", e.Name(), "SKILL.md")
			raw, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			fm := ParseFrontmatter(string(raw))
			out = append(out, CatalogSkill{
				Slug:        p.name + ":" + e.Name(),
				Name:        deref(fm.Name, e.Name()),
				Description: deref(fm.Description, ""),
				Group:       "Plugin · " + p.name,
				Path:        tildePath(d.Home, file),
			})
		}
	}
	sortSkills(out)
	return out
}

// ReadMarkdown returns one skill's full SKILL.md: a plain slug from
// ~/.claude/skills, or plugin:skill through the live plugin install. False on
// a bad slug (no path traversal) or a missing file.
func ReadMarkdown(d Dirs, slug string) (string, bool) {
	var file string
	if strings.Contains(slug, ":") {
		parts := strings.Split(slug, ":")
		if len(parts) != 2 || !validHalf(parts[0]) || !validHalf(parts[1]) {
			return "", false
		}
		for _, p := range installedPlugins(d.Plugins) {
			if p.name == parts[0] {
				file = filepath.Join(p.installPath, "skills", parts[1], "SKILL.md")
				break
			}
		}
		if file == "" {
			return "", false
		}
	} else {
		if !validHalf(slug) {
			return "", false
		}
		file = filepath.Join(d.Skills, slug, "SKILL.md")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", false
	}
	return string(raw), true
}
