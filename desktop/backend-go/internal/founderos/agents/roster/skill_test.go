package roster

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestOpenSkillQuestionsReadsUncheckedBoldItemsIncludingBlockquotes(t *testing.T) {
	skill := strings.Join([]string{
		"# Skill",
		"- [ ] **Audience.** Platform by platform.",
		"> - [ ] **Ask and floor** for: single video.",
		">   - [ ] **Nested in a quote**",
		"- [x] **Already answered.**",
		"- [ ] not bold, not a question",
		"text - [ ] **mid-line does not count**",
	}, "\n")
	got := OpenSkillQuestions(skill)
	want := []string{"Audience.", "Ask and floor", "Nested in a quote"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := OpenSkillQuestions("no boxes"); got == nil || len(got) != 0 {
		t.Fatalf("no questions is an empty list, not nil: %#v", got)
	}
}

func TestLoadAgentSkillFromOverrideDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(AgentsDirEnv, dir)
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo", "skill.md"), []byte("\n  # Demo skill\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadAgentSkill("demo")
	if err != nil || got != "# Demo skill" {
		t.Fatalf("got %q err %v", got, err)
	}

	// Missing and empty are both loud: an agent must not run on no prompt.
	if _, err := LoadAgentSkill("absent"); err == nil || err.Error() != "missing or empty skill file: agents/absent/skill.md" {
		t.Fatalf("missing: %v", err)
	}
	_ = os.MkdirAll(filepath.Join(dir, "blank"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "blank", "skill.md"), []byte(" \n\t\n"), 0o644)
	if _, err := LoadAgentSkill("blank"); err == nil || err.Error() != "missing or empty skill file: agents/blank/skill.md" {
		t.Fatalf("blank: %v", err)
	}
}

// Without the override the skills are the repo's config/founderos/agents
// copies of FounderOS v1 agents/<folder>/skill.md, found above the working
// directory the way the engine topology is. Their open questions are the
// ones FounderOS v1 surfaces on every run.
func TestRepoSkillFilesMatchFounderosOSOpenQuestions(t *testing.T) {
	t.Setenv(AgentsDirEnv, "")
	want := map[string][]string{
		"brand-deals": {"Victor's sending address", "Audience.", "Email list numbers.", "Past partners", "Proof assets.",
			"Category exclusions.", "Ask and floor", "Opening anchor", "Payment terms.", "Usage window."},
		"newsletter": {"Who the list is.", "The format.", "Cadence.", "The job of the newsletter.", "Source material.", "Two issues he was proud of"},
	}
	for folder, q := range want {
		skill, err := LoadAgentSkill(folder)
		if err != nil {
			t.Fatalf("%s: %v", folder, err)
		}
		if got := OpenSkillQuestions(skill); !slices.Equal(got, q) {
			t.Errorf("%s questions = %q\nwant %q", folder, got, q)
		}
	}
}
