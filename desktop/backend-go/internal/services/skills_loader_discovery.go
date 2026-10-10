package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads skills.yaml and parses all skill metadata.
//
// Returns error if skills.yaml doesn't exist or is invalid.
// Individual skill errors are logged but don't fail the whole load.
func (l *SkillsLoader) LoadConfig() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return fmt.Errorf("failed to read skills config: %w", err)
	}

	var config SkillsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse skills config: %w", err)
	}

	l.config = &config
	l.skills = make(map[string]*SkillMetadata)

	for _, def := range config.Skills {
		if !def.Enabled {
			continue
		}

		skillPath := filepath.Join(l.basePath, def.Name, "SKILL.md")

		metadata, err := l.parseSkillFile(skillPath)
		if err != nil {
			slog.Default().Warn("[SkillsLoader] Failed to load skill",
				"skill", def.Name,
				"error", err)
			continue
		}

		metadata.Path = filepath.Join(l.basePath, def.Name)
		metadata.Enabled = def.Enabled
		metadata.Priority = def.Priority

		l.skills[def.Name] = metadata
	}

	l.loaded = true
	slog.Default().Info("[SkillsLoader] Loaded skills", "count", len(l.skills))

	return nil
}

// parseSkillFile reads a SKILL.md file and extracts the YAML frontmatter.
func (l *SkillsLoader) parseSkillFile(path string) (*SkillMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	frontmatter, _, err := extractFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	var wrapper SkillMetadataWrapper
	if err := yaml.Unmarshal([]byte(frontmatter), &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter YAML: %w", err)
	}

	metadata := &SkillMetadata{
		Name:        wrapper.Name,
		Description: wrapper.Description,
		Version:     wrapper.Metadata.Version,
		Author:      wrapper.Metadata.Author,
		ToolsUsed:   wrapper.Metadata.ToolsUsed,
		DependsOn:   wrapper.Metadata.DependsOn,
	}

	return metadata, nil
}

// extractFrontmatter splits a markdown file into frontmatter and content.
func extractFrontmatter(content string) (frontmatter, body string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)

	matches := re.FindStringSubmatch(content)
	if len(matches) != 3 {
		return "", content, fmt.Errorf("no valid frontmatter found")
	}

	return matches[1], matches[2], nil
}
