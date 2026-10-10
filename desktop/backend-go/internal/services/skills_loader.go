package services

import (
	"path/filepath"
	"sync"
)

type SkillsConfig struct {
	Version         string              `yaml:"version"`
	SkillsDirectory string              `yaml:"skills_directory"`
	Settings        SkillsSettings      `yaml:"settings"`
	Skills          []SkillDefinition   `yaml:"skills"`
	SkillGroups     map[string][]string `yaml:"skill_groups"`
}

// SkillsSettings contains global settings for skills loading
type SkillsSettings struct {
	MaxSkillsInContext     int  `yaml:"max_skills_in_context"`
	DefaultTokenBudget     int  `yaml:"default_token_budget"`
	EnableTelemetry        bool `yaml:"enable_telemetry"`
	EnableSchemaValidation bool `yaml:"enable_schema_validation"`
}

// SkillDefinition is an entry in skills.yaml
type SkillDefinition struct {
	Name            string `yaml:"name"`
	Path            string `yaml:"path"`
	Enabled         bool   `yaml:"enabled"`
	Priority        int    `yaml:"priority"`
	AlwaysAvailable bool   `yaml:"always_available"`
}

// SkillMetadata is parsed from the YAML frontmatter of SKILL.md
// This is the "discovery" data - minimal info for the agent to decide
// whether to load the full skill.
type SkillMetadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	ToolsUsed   []string `yaml:"tools_used"`
	DependsOn   []string `yaml:"depends_on"`

	// Runtime fields (not from frontmatter)
	Path     string `yaml:"-"` // Filesystem path to skill folder
	Enabled  bool   `yaml:"-"` // From skills.yaml
	Priority int    `yaml:"-"` // From skills.yaml
}

// SkillMetadataWrapper wraps the nested metadata structure in frontmatter
type SkillMetadataWrapper struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Metadata    struct {
		Version   string   `yaml:"version"`
		Author    string   `yaml:"author"`
		ToolsUsed []string `yaml:"tools_used"`
		DependsOn []string `yaml:"depends_on"`
	} `yaml:"metadata"`
}

// ============================================================================
// SKILLS LOADER
// ============================================================================

// SkillsLoader manages loading and caching of skill definitions.
//
// Thread Safety:
// - Uses RWMutex for concurrent access
// - Safe to call from multiple goroutines (e.g., multiple agent conversations)
//
// Caching:
// - Skill metadata is cached after first load
// - Call Reload() to refresh from disk
type SkillsLoader struct {
	// Configuration
	configPath string // Path to skills.yaml
	basePath   string // Base directory for skill folders

	// Cached data
	config *SkillsConfig
	skills map[string]*SkillMetadata // name -> metadata

	// Tool validation
	toolRegistry ToolRegistry // Optional: validates tool references

	// Thread safety
	mu     sync.RWMutex
	loaded bool
}

// ToolRegistry interface for validating tool existence
type ToolRegistry interface {
	// GetTool returns a tool by name
	GetTool(name string) (interface{}, bool)
}

// NewSkillsLoader creates a new skills loader.
//
// configPath: Path to skills.yaml (e.g., "./skills/skills.yaml")
//
// The loader doesn't read files until LoadConfig() is called.
// This allows graceful handling if skills directory doesn't exist.
func NewSkillsLoader(configPath string) *SkillsLoader {
	return &SkillsLoader{
		configPath: configPath,
		basePath:   filepath.Dir(configPath), // skills.yaml is in skills/ folder
		skills:     make(map[string]*SkillMetadata),
	}
}

// NewSkillsLoaderWithRegistry creates a skills loader with tool registry validation.
//
// The tool registry is used to validate that all tools referenced in SKILL.md
// actually exist in the system.
func NewSkillsLoaderWithRegistry(configPath string, toolRegistry ToolRegistry) *SkillsLoader {
	return &SkillsLoader{
		configPath:   configPath,
		basePath:     filepath.Dir(configPath),
		skills:       make(map[string]*SkillMetadata),
		toolRegistry: toolRegistry,
	}
}

// SetToolRegistry sets the tool registry for validation (can be set after creation)
func (l *SkillsLoader) SetToolRegistry(toolRegistry ToolRegistry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.toolRegistry = toolRegistry
}
