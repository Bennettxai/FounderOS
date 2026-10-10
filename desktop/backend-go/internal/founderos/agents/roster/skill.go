package roster

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Agent skill files: the port of FounderOS v1 lib/agents/skill-file.ts.
//
// In FounderOS v1 an agent's prompt is agents/<folder>/skill.md, read at run
// time so an edit changes the next run with no deploy. The bridge keeps the
// same contract over config/founderos/agents/<folder>/skill.md (verbatim copies
// of the FounderOS v1 files, found above the working directory the way the
// engine topology is), or over AgentsDirEnv when set. No bridge agent sends
// the skill to a model; they read its open "- [ ] **...**" questions.

// AgentsDirEnv overrides the skill directory (it holds <folder>/skill.md).
const AgentsDirEnv = "FOUNDEROS_AGENTS_DIR"

// AgentsRepoPath is where the skills live relative to the repo root.
const AgentsRepoPath = "config/founderos/agents"

// AgentsDir is AgentsDirEnv, else the nearest config/founderos/agents above the
// working directory; "" when neither exists.
func AgentsDir() string {
	if d := strings.TrimSpace(os.Getenv(AgentsDirEnv)); d != "" {
		return d
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, AgentsRepoPath)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadAgentSkill is readAgentSkill: the trimmed skill text, or an error when
// the file is missing or blank. Loud on purpose: an agent running on an
// empty prompt would still sound confident.
func LoadAgentSkill(folder string) (string, error) {
	fail := fmt.Errorf("missing or empty skill file: agents/%s/skill.md", folder)
	dir := AgentsDir()
	if dir == "" {
		return "", fail
	}
	raw, err := os.ReadFile(filepath.Join(dir, folder, "skill.md"))
	if err != nil {
		return "", fail
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", fail
	}
	return text, nil
}

// openQuestion matches FounderOS v1's /^\s*(?:>\s*)?- \[ \] \*\*(.+?)\*\*/gm:
// the optional "> " matters, the brand deal skill keeps its questions in
// blockquotes.
var openQuestion = regexp.MustCompile(`(?m)^\s*(?:>\s*)?- \[ \] \*\*(.+?)\*\*`)

// OpenSkillQuestions is openSkillQuestions: the checklist items the operator has
// not filled in yet, in file order. Never nil.
func OpenSkillQuestions(skill string) []string {
	out := []string{}
	for _, m := range openQuestion.FindAllStringSubmatch(skill, -1) {
		out = append(out, m[1])
	}
	return out
}
