package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
)

const skillsDemoMD = "---\nname: demo-skill\ndescription: A demo skill for tests.\n---\n\n# Demo skill\n\nBody text.\n"
const skillsAlphaMD = "---\nname: alpha\ndescription: First plugin skill.\n---\n\n# Alpha\n"

// skillsFixture points the catalog at a temp ~/.claude with one user skill
// and one plugin skill.
func skillsFixture(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	put := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	skillsDir, pluginsDir := filepath.Join(root, "skills"), filepath.Join(root, "plugins")
	put(filepath.Join(skillsDir, "demo-skill", "SKILL.md"), skillsDemoMD)
	live := filepath.Join(pluginsDir, "cache", "myplugin", "1.0.0")
	put(filepath.Join(live, "skills", "alpha", "SKILL.md"), skillsAlphaMD)
	manifest, _ := json.Marshal(map[string]any{"plugins": map[string]any{"myplugin@official": []map[string]string{{"installPath": live}}}})
	put(filepath.Join(pluginsDir, "installed_plugins.json"), string(manifest))
	t.Setenv("FOUNDEROS_OS_SKILLS_DIR", skillsDir)
	t.Setenv("FOUNDEROS_OS_PLUGINS_DIR", pluginsDir)
}

type skillsBody struct {
	Cards []struct {
		ID, Name, Group, Kind, Description, Meta, FilePath string
		Status                                             *string
		Markdown                                           *string
	} `json:"cards"`
	SourceNote    string  `json:"sourceNote"`
	OperatorError *string `json:"operatorError"`
	Volume        struct {
		Headline int `json:"headline"`
		Counts   struct {
			Claude, Operator int
		} `json:"counts"`
	} `json:"volume"`
}

func TestSkillsPageJoinsDiskSkillsAndOperatorSkills(t *testing.T) {
	skillsFixture(t)
	pool := catalogdb.TestDB(t)
	ws := catalogdb.Workspace(t, pool, "founderos")
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('ops', $1, 'Ops', 'ops', '#fff', 1)`,
		`INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier) VALUES ('conductor', $1, 'ops', 'Conductor', 'active', 'lead')`,
		`INSERT INTO founderos_skills (id, workspace_id, name, category, description, owner_agent_id, status, tools, markdown, ord)
		 VALUES ('skill-retrieval', $1, 'Knowledge retrieval', 'Ops', 'Finds things in the brain before anyone asks twice about the same thing again and again and again and again, every single morning without fail.', 'conductor', 'live', '["oe"]', '# Retrieval', 1)`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatal(err)
		}
	}
	r := router(t, &Deps{Pool: pool, Board: connectors.NewRegistry()})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/skills", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	w = catalogGet(t, r, "/api/founderos/pages/skills")
	var body skillsBody
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(body.Cards) != 3 {
		t.Fatalf("cards = %+v", body.Cards)
	}
	byID := map[string]int{}
	for i, c := range body.Cards {
		byID[c.ID] = i
	}
	demo := body.Cards[byID["demo-skill"]]
	if demo.Kind != "claude" || demo.Meta != "~/.claude/skills/demo-skill/SKILL.md" || demo.FilePath != demo.Meta || demo.Markdown != nil {
		t.Errorf("demo card %+v", demo)
	}
	if body.Cards[byID["myplugin:alpha"]].Group != "Plugin · myplugin" {
		t.Errorf("plugin card %+v", body.Cards[byID["myplugin:alpha"]])
	}
	op := body.Cards[byID["skill-retrieval"]]
	if op.Kind != "operator" || op.Group != "Operator · Ops" || op.Meta != "Conductor" || op.FilePath != "skills/skill-retrieval/SKILL.md" ||
		op.Status == nil || *op.Status != "live" || op.Markdown == nil || *op.Markdown != "# Retrieval" {
		t.Errorf("operator card %+v", op)
	}
	if !strings.HasSuffix(op.Description, "…") || len([]rune(op.Description)) > 111 {
		t.Errorf("description should be truncated to 110 at a word: %q", op.Description)
	}
	if body.Volume.Headline != 3 || body.Volume.Counts.Claude != 2 || body.Volume.Counts.Operator != 1 || body.OperatorError != nil {
		t.Errorf("volume %+v err %v", body.Volume, body.OperatorError)
	}
	if !strings.HasPrefix(body.SourceNote, "2 skills live from ~/.claude (user + plugins) + 1 operator skills") || strings.Contains(body.SourceNote, "—") {
		t.Errorf("source note %q", body.SourceNote)
	}
}

func TestSkillsPageKeepsDiskSkillsWhenOperatorSkillsAreUnreachable(t *testing.T) {
	skillsFixture(t)
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	w := catalogGet(t, r, "/api/founderos/pages/skills")
	var body skillsBody
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if len(body.Cards) != 2 || body.OperatorError == nil || *body.OperatorError == "" {
		t.Fatalf("want the 2 disk skills and an operator error, got %+v", body)
	}
	if !strings.Contains(body.SourceNote, "operator skills unavailable") {
		t.Errorf("source note %q", body.SourceNote)
	}
}

func TestSkillMarkdownRoute(t *testing.T) {
	skillsFixture(t)
	r := router(t, &Deps{Board: connectors.NewRegistry()})

	w := catalogGet(t, r, "/api/founderos/pages/skills/demo-skill")
	var body struct{ Markdown string }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Markdown != skillsDemoMD {
		t.Fatalf("json: %d %s", w.Code, w.Body)
	}
	w = catalogGet(t, r, "/api/founderos/pages/skills/myplugin:alpha")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "First plugin skill.") {
		t.Fatalf("plugin slug: %d %s", w.Code, w.Body)
	}
	w = catalogGet(t, r, "/api/founderos/pages/skills/myplugin%3Aalpha?download=1")
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), "text/markdown") ||
		w.Header().Get("Content-Disposition") != `attachment; filename="myplugin-alpha-SKILL.md"` || w.Body.String() != skillsAlphaMD {
		t.Fatalf("download: %d %v %q", w.Code, w.Header(), w.Body)
	}
	for _, p := range []string{"/api/founderos/pages/skills/nope", "/api/founderos/pages/skills/nope?download=1", "/api/founderos/pages/skills/..%2F..", "/api/founderos/pages/skills/myplugin:..%2Fevil"} {
		if w := catalogGet(t, r, p); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", p, w.Code)
		}
	}
}

// A host with no ~/.claude (the demo stack runs with an empty HOME) reads as
// zero disk skills, never an error, like FounderOS v1 on a bare machine.
func TestSkillsPageIsHonestWithNoClaudeDirOnTheHost(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FOUNDEROS_OS_SKILLS_DIR", "")
	t.Setenv("FOUNDEROS_OS_PLUGINS_DIR", "")
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	w := catalogGet(t, r, "/api/founderos/pages/skills")
	var body skillsBody
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if body.Volume.Counts.Claude != 0 || len(body.Cards) != 0 {
		t.Fatalf("want no disk skills from an empty HOME, got %+v", body)
	}
	if !strings.Contains(body.SourceNote, "no ~/.claude/skills on this machine") {
		t.Errorf("source note %q", body.SourceNote)
	}
}
