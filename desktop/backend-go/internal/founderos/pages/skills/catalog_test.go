package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	for _, tc := range []struct {
		name, md, wantName, wantDesc string
	}{
		{"inline", "---\nname: codex\ndescription: Delegate a task to Codex.\n---\n\n# Codex\nbody", "codex", "Delegate a task to Codex."},
		{"quotes", "---\nname: \"nano-banana\"\ndescription: 'Make images'\n---\n", "nano-banana", "Make images"},
		{"literal block", "---\nname: fc\ndescription: |\n  Line one.\n  Line two.\n---\n", "fc", "Line one.\nLine two."},
		{"folded block", "---\nname: dr\ndescription: >-\n  Folded line one\n  folded line two\n---\n", "dr", "Folded line one folded line two"},
		{"crlf", "---\r\nname: w\r\ndescription: win\r\n---\r\n", "w", "win"},
	} {
		fm := ParseFrontmatter(tc.md)
		if fm.Name == nil || *fm.Name != tc.wantName || fm.Description == nil || *fm.Description != tc.wantDesc {
			t.Errorf("%s: got %+v", tc.name, fm)
		}
	}
	if fm := ParseFrontmatter("# just a heading\ntext"); fm.Name != nil || fm.Description != nil {
		t.Errorf("no frontmatter: got %+v", fm)
	}
}

func TestGroup(t *testing.T) {
	for in, want := range map[string]string{
		"firecrawl-scrape": "Firecrawl", "firecrawl": "Firecrawl", "build": "Spec · build · review", "spec": "Spec · build · review",
		"codex": "Engineering", "mcp-builder": "Engineering", "nano-banana": "Creative", "proposal-generator": "Sales", "something-else": "Skills",
	} {
		if got := Group(in); got != want {
			t.Errorf("Group(%q) = %q, want %q", in, got, want)
		}
	}
}

const demoMD = "---\nname: demo-skill\ndescription: A demo skill for tests.\n---\n\n# Demo skill\n\nBody text.\n"
const alphaMD = "---\nname: alpha\ndescription: First plugin skill.\n---\n\n# Alpha\n\nBody.\n"

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixture builds a fake home with ~/.claude/skills and ~/.claude/plugins.
func fixture(t *testing.T) Dirs {
	home := t.TempDir()
	d := Dirs{Home: home, Skills: filepath.Join(home, ".claude", "skills"), Plugins: filepath.Join(home, ".claude", "plugins")}
	write(t, filepath.Join(d.Skills, "demo-skill", "SKILL.md"), demoMD)
	write(t, filepath.Join(d.Skills, "codex", "SKILL.md"), "---\nname: codex\n---\n")
	write(t, filepath.Join(d.Skills, "auto-mode.md"), "not a skill dir")
	if err := os.MkdirAll(filepath.Join(d.Skills, "empty-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	// a symlinked skill (firecrawl-* are symlinks on the operator's machine)
	write(t, filepath.Join(home, "elsewhere", "firecrawl-scrape", "SKILL.md"), "---\nname: firecrawl-scrape\ndescription: Scrape.\n---\n")
	if err := os.Symlink(filepath.Join(home, "elsewhere", "firecrawl-scrape"), filepath.Join(d.Skills, "firecrawl-scrape")); err != nil {
		t.Fatal(err)
	}

	live := filepath.Join(d.Plugins, "cache", "official", "myplugin", "1.0.0")
	stale := filepath.Join(d.Plugins, "cache", "official", "myplugin", "0.9.0")
	bare := filepath.Join(d.Plugins, "cache", "official", "no-skills-plugin", "1.0.0")
	write(t, filepath.Join(live, "skills", "alpha", "SKILL.md"), alphaMD)
	write(t, filepath.Join(stale, "skills", "old-skill", "SKILL.md"), "---\nname: old-skill\n---\n")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(map[string]any{"version": 2, "plugins": map[string]any{
		"myplugin@official": []map[string]string{
			{"scope": "user", "installPath": live, "version": "1.0.0"},
			{"scope": "project", "installPath": live, "version": "1.0.0"},
		},
		"no-skills-plugin@official": []map[string]string{{"scope": "user", "installPath": bare}},
		"ghost@official":            []map[string]string{{"scope": "user", "installPath": filepath.Join(d.Plugins, "nope")}},
	}})
	write(t, filepath.Join(d.Plugins, "installed_plugins.json"), string(manifest))
	return d
}

func TestReadUserSkills(t *testing.T) {
	d := fixture(t)
	got := ReadUserSkills(d)
	slugs := map[string]CatalogSkill{}
	for _, s := range got {
		slugs[s.Slug] = s
	}
	if len(got) != 3 {
		t.Fatalf("want demo-skill, codex and the symlinked firecrawl-scrape, got %+v", got)
	}
	demo := slugs["demo-skill"]
	if demo.Name != "demo-skill" || demo.Description != "A demo skill for tests." || demo.Group != "Skills" || demo.Path != "~/.claude/skills/demo-skill/SKILL.md" {
		t.Errorf("demo = %+v", demo)
	}
	if slugs["firecrawl-scrape"].Group != "Firecrawl" {
		t.Errorf("symlinked skill missing or misgrouped: %+v", slugs["firecrawl-scrape"])
	}
	if slugs["codex"].Description != "" || slugs["codex"].Group != "Engineering" {
		t.Errorf("codex = %+v", slugs["codex"])
	}
	// sorted by group then name
	if got[0].Group != "Engineering" || got[1].Group != "Firecrawl" || got[2].Group != "Skills" {
		t.Errorf("order = %+v", got)
	}
	if n := len(ReadUserSkills(Dirs{Home: d.Home, Skills: filepath.Join(d.Home, "absent")})); n != 0 {
		t.Errorf("absent dir: %d skills, want 0", n)
	}
}

func TestReadPluginSkills(t *testing.T) {
	d := fixture(t)
	got := ReadPluginSkills(d)
	if len(got) != 1 {
		t.Fatalf("want only myplugin:alpha (once, live version only), got %+v", got)
	}
	a := got[0]
	if a.Slug != "myplugin:alpha" || a.Name != "alpha" || a.Description != "First plugin skill." || a.Group != "Plugin · myplugin" {
		t.Errorf("alpha = %+v", a)
	}
	if !strings.HasPrefix(a.Path, "~/") || !strings.HasSuffix(a.Path, "skills/alpha/SKILL.md") {
		t.Errorf("path %q should be ~-relative", a.Path)
	}
	if n := len(ReadPluginSkills(Dirs{Home: d.Home, Plugins: filepath.Join(d.Home, "not-a-dir")})); n != 0 {
		t.Errorf("absent plugins dir: %d, want 0", n)
	}
}

func TestReadMarkdown(t *testing.T) {
	d := fixture(t)
	if md, ok := ReadMarkdown(d, "demo-skill"); !ok || md != demoMD {
		t.Errorf("demo-skill: %v %q", ok, md)
	}
	if md, ok := ReadMarkdown(d, "myplugin:alpha"); !ok || md != alphaMD {
		t.Errorf("plugin slug: %v %q", ok, md)
	}
	for _, bad := range []string{"nope", "nope:alpha", "myplugin:nope", "..:alpha", "myplugin:../evil", "myplugin:alpha:extra", "../..", "..", ".", "a/b", ""} {
		if _, ok := ReadMarkdown(d, bad); ok {
			t.Errorf("%q must not resolve", bad)
		}
	}
}

func TestDefaultDirsHonourTheOverrides(t *testing.T) {
	t.Setenv("FOUNDEROS_OS_SKILLS_DIR", "/x/skills")
	t.Setenv("FOUNDEROS_OS_PLUGINS_DIR", "/x/plugins")
	d := DefaultDirs()
	if d.Skills != "/x/skills" || d.Plugins != "/x/plugins" || d.Home == "" {
		t.Fatalf("dirs = %+v", d)
	}
	t.Setenv("FOUNDEROS_OS_SKILLS_DIR", "")
	t.Setenv("FOUNDEROS_OS_PLUGINS_DIR", "")
	d = DefaultDirs()
	if d.Skills != filepath.Join(d.Home, ".claude", "skills") || d.Plugins != filepath.Join(d.Home, ".claude", "plugins") {
		t.Fatalf("default dirs = %+v", d)
	}
}

// The bridge's env.local (read through the connectors Resolver, not the
// process env) carries FOUNDEROS_OS_SKILLS_DIR / FOUNDEROS_OS_PLUGINS_DIR on the
// mini; reading only os.Getenv showed 0 Claude Code skills there.
func TestDirsFromHonoursALookup(t *testing.T) {
	lookup := func(k string) string {
		return map[string]string{"FOUNDEROS_OS_SKILLS_DIR": "/x/skills", "FOUNDEROS_OS_PLUGINS_DIR": "/x/plugins"}[k]
	}
	d := DirsFrom(lookup)
	if d.Skills != "/x/skills" || d.Plugins != "/x/plugins" {
		t.Fatalf("dirs = %+v", d)
	}
	none := DirsFrom(func(string) string { return "" })
	if filepath.Base(none.Skills) != "skills" || filepath.Base(filepath.Dir(none.Skills)) != ".claude" {
		t.Fatalf("default skills dir = %q", none.Skills)
	}
}
