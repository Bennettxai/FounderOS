package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetEnabledSkills returns metadata for all enabled skills, sorted by priority (highest first).
func (l *SkillsLoader) GetEnabledSkills() []*SkillMetadata {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.loaded {
		return nil
	}

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

	return skills
}

// GetSkillMetadata returns metadata for a specific skill.
// Returns nil if skill doesn't exist or isn't enabled.
func (l *SkillsLoader) GetSkillMetadata(name string) *SkillMetadata {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.skills[name]
}

// GetSkillContent returns the full SKILL.md content for a skill.
func (l *SkillsLoader) GetSkillContent(name string) (string, error) {
	l.mu.RLock()
	skill := l.skills[name]
	l.mu.RUnlock()

	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	path := filepath.Join(skill.Path, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read skill content: %w", err)
	}

	return string(data), nil
}

// GetSkillReference returns a reference file from a skill's references/ folder.
func (l *SkillsLoader) GetSkillReference(skillName, refName string) (string, error) {
	l.mu.RLock()
	skill := l.skills[skillName]
	l.mu.RUnlock()

	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	// Security: Prevent path traversal attacks
	if strings.Contains(refName, "/") || strings.Contains(refName, "\\") || strings.HasPrefix(refName, ".") {
		return "", fmt.Errorf("invalid reference name: %s", refName)
	}

	path := filepath.Join(skill.Path, "references", refName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read reference: %w", err)
	}

	return string(data), nil
}

// GetSkillSchema returns the JSON schema for a skill's tool input.
func (l *SkillsLoader) GetSkillSchema(skillName, schemaName string) (string, error) {
	l.mu.RLock()
	skill := l.skills[skillName]
	l.mu.RUnlock()

	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	// Security: Prevent path traversal
	if strings.Contains(schemaName, "/") || strings.Contains(schemaName, "\\") || strings.HasPrefix(schemaName, ".") {
		return "", fmt.Errorf("invalid schema name: %s", schemaName)
	}

	path := filepath.Join(skill.Path, "schemas", schemaName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read schema: %w", err)
	}

	return string(data), nil
}

// ListSkillReferences returns available reference files for a skill.
func (l *SkillsLoader) ListSkillReferences(skillName string) ([]string, error) {
	l.mu.RLock()
	skill := l.skills[skillName]
	l.mu.RUnlock()

	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", skillName)
	}

	refsPath := filepath.Join(skill.Path, "references")
	entries, err := os.ReadDir(refsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read references: %w", err)
	}

	refs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			refs = append(refs, entry.Name())
		}
	}

	return refs, nil
}

// GetSkillGroup returns skill names in a named group.
func (l *SkillsLoader) GetSkillGroup(groupName string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.config == nil || l.config.SkillGroups == nil {
		return nil
	}

	return l.config.SkillGroups[groupName]
}

// GetSkillsByTool returns skills that use a specific tool.
func (l *SkillsLoader) GetSkillsByTool(toolName string) []*SkillMetadata {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var matches []*SkillMetadata
	for _, skill := range l.skills {
		for _, tool := range skill.ToolsUsed {
			if tool == toolName {
				matches = append(matches, skill)
				break
			}
		}
	}

	return matches
}
