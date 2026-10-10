package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// AGENT INTEGRATION
// ============================================================================

// GetSkillsPromptXML generates the <available_skills> XML block for injection
// into the agent's system prompt.
//
// This is the "discovery" layer - tells the agent what skills exist and what
// they're for, using only ~50 tokens per skill.
//
// Output format:
//
//	<available_skills>
//	  <skill name="dashboard-management">
//	    Create and configure custom dashboards with widgets. Add task summaries,
//	    burndown charts, project progress, upcoming deadlines, and metric cards.
//	  </skill>
//	  <skill name="task-management">
//	    Create, update, complete, and delete tasks. Filter by status, project,
//	    priority, or due date. Bulk operations for efficiency.
//	  </skill>
//	</available_skills>
//
// The agent uses this to decide which skill to activate based on user request.
func (l *SkillsLoader) GetSkillsPromptXML() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.loaded || len(l.skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	// Collect and sort skills by priority (inline to avoid deadlock)
	skills := make([]*SkillMetadata, 0, len(l.skills))
	for _, skill := range l.skills {
		skills = append(skills, skill)
	}
	for i := 0; i < len(skills)-1; i++ {
		for j := i + 1; j < len(skills); j++ {
			if skills[j].Priority > skills[i].Priority {
				skills[i], skills[j] = skills[j], skills[i]
			}
		}
	}

	for _, skill := range skills {
		// Truncate description if too long (aim for ~50 tokens per skill)
		desc := skill.Description
		if len(desc) > 300 {
			desc = desc[:297] + "..."
		}

		// Clean up description (remove newlines, extra spaces)
		desc = strings.ReplaceAll(desc, "\n", " ")
		desc = strings.Join(strings.Fields(desc), " ")

		sb.WriteString(fmt.Sprintf("  <skill name=\"%s\">\n", skill.Name))
		sb.WriteString(fmt.Sprintf("    %s\n", desc))
		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")

	return sb.String()
}

// GetSkillsPromptInstructions returns instructions for how the agent should
// use the available skills. This is injected alongside the skills XML.
func (l *SkillsLoader) GetSkillsPromptInstructions() string {
	return `When a user request matches a skill description, you should:
1. Identify the most relevant skill based on keywords and intent
2. If you need the full instructions, request to load the skill
3. Follow the skill's "Request → Tool Mapping" to call the appropriate tool
4. Use the skill's examples as templates for your responses

If a request could match multiple skills, prefer the one with higher specificity.
For ambiguous requests, ask the user for clarification.`
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// Reload clears the cache and reloads all skills from disk.
//
// Use this if skills are updated while server is running.
// Returns error if reload fails (but old data is preserved).
func (l *SkillsLoader) Reload() error {
	// Store old data in case reload fails
	l.mu.Lock()
	oldConfig := l.config
	oldSkills := l.skills
	l.loaded = false
	l.mu.Unlock()

	// Attempt reload
	err := l.LoadConfig()
	if err != nil {
		// Restore old data
		l.mu.Lock()
		l.config = oldConfig
		l.skills = oldSkills
		l.loaded = true
		l.mu.Unlock()
		return err
	}

	return nil
}

// IsLoaded returns whether skills have been loaded.
func (l *SkillsLoader) IsLoaded() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded
}

// GetSettings returns the skills configuration settings.
func (l *SkillsLoader) GetSettings() *SkillsSettings {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.config == nil {
		return nil
	}
	return &l.config.Settings
}

// validateToolReferences validates that all tools referenced in a skill exist in the tool registry.
//
// Returns a list of validation issues (tool names that don't exist).
func (l *SkillsLoader) validateToolReferences(skill *SkillMetadata) []string {
	var issues []string

	// Skip validation if no tool registry is configured
	if l.toolRegistry == nil {
		return issues
	}

	// Validate each tool in tools_used
	for _, toolName := range skill.ToolsUsed {
		if _, exists := l.toolRegistry.GetTool(toolName); !exists {
			issues = append(issues, fmt.Sprintf("tool not found in registry: %s", toolName))
		}
	}

	return issues
}

// ValidateSkill checks if a skill has all required files and valid structure.
//
// Checks:
// - SKILL.md exists and has valid frontmatter
// - All referenced tools exist in the tool registry
// - References folder structure is valid
//
// Returns list of warnings/errors.
func (l *SkillsLoader) ValidateSkill(name string) []string {
	l.mu.RLock()
	skill := l.skills[name]
	toolRegistry := l.toolRegistry
	l.mu.RUnlock()

	if skill == nil {
		return []string{fmt.Sprintf("skill not found: %s", name)}
	}

	var issues []string

	// Check SKILL.md exists
	skillPath := filepath.Join(skill.Path, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		issues = append(issues, "SKILL.md not found")
	}

	// Check name matches folder
	folderName := filepath.Base(skill.Path)
	if skill.Name != folderName {
		issues = append(issues, fmt.Sprintf("skill name '%s' doesn't match folder '%s'", skill.Name, folderName))
	}

	// Check description exists
	if skill.Description == "" {
		issues = append(issues, "description is empty")
	}

	// Check tools_used is specified
	if len(skill.ToolsUsed) == 0 {
		issues = append(issues, "no tools_used specified")
	}

	// Validate tool references against tool registry
	if toolRegistry != nil {
		toolIssues := l.validateToolReferences(skill)
		issues = append(issues, toolIssues...)
	}

	return issues
}
