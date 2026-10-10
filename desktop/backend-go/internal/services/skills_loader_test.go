package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockToolRegistry is a mock implementation of ToolRegistry for testing
type MockToolRegistry struct {
	tools map[string]interface{}
}

func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools: make(map[string]interface{}),
	}
}

func (m *MockToolRegistry) RegisterTool(name string) {
	m.tools[name] = struct{}{}
}

func (m *MockToolRegistry) GetTool(name string) (interface{}, bool) {
	tool, exists := m.tools[name]
	return tool, exists
}

func TestValidateToolReferences(t *testing.T) {
	tests := []struct {
		name           string
		skillToolsUsed []string
		registryTools  []string
		expectedIssues int
	}{
		{
			name:           "all tools exist",
			skillToolsUsed: []string{"configure_dashboard", "create_task"},
			registryTools:  []string{"configure_dashboard", "create_task", "update_task"},
			expectedIssues: 0,
		},
		{
			name:           "one tool missing",
			skillToolsUsed: []string{"configure_dashboard", "missing_tool"},
			registryTools:  []string{"configure_dashboard", "create_task"},
			expectedIssues: 1,
		},
		{
			name:           "all tools missing",
			skillToolsUsed: []string{"missing_tool1", "missing_tool2"},
			registryTools:  []string{"configure_dashboard", "create_task"},
			expectedIssues: 2,
		},
		{
			name:           "empty tools_used",
			skillToolsUsed: []string{},
			registryTools:  []string{"configure_dashboard", "create_task"},
			expectedIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry and register tools
			registry := NewMockToolRegistry()
			for _, toolName := range tt.registryTools {
				registry.RegisterTool(toolName)
			}

			// Create skills loader with registry
			loader := &SkillsLoader{
				toolRegistry: registry,
			}

			// Create test skill metadata
			skill := &SkillMetadata{
				Name:      "test-skill",
				ToolsUsed: tt.skillToolsUsed,
			}

			// Validate tool references
			issues := loader.validateToolReferences(skill)

			// Assert expected number of issues
			assert.Equal(t, tt.expectedIssues, len(issues), "Expected %d issues, got %d: %v", tt.expectedIssues, len(issues), issues)

			// If there are expected issues, verify they contain the expected text
			if tt.expectedIssues > 0 {
				for _, issue := range issues {
					assert.Contains(t, issue, "tool not found in registry:", "Issue should mention tool not found")
				}
			}
		})
	}
}

func TestValidateToolReferences_NoRegistry(t *testing.T) {
	// Create loader without registry
	loader := &SkillsLoader{
		toolRegistry: nil,
	}

	skill := &SkillMetadata{
		Name:      "test-skill",
		ToolsUsed: []string{"any_tool", "another_tool"},
	}

	// Should not return any issues when no registry is set
	issues := loader.validateToolReferences(skill)
	assert.Empty(t, issues, "Should not validate when no tool registry is configured")
}

func TestValidateSkill_WithToolValidation(t *testing.T) {
	// Create mock registry
	registry := NewMockToolRegistry()
	registry.RegisterTool("configure_dashboard")
	registry.RegisterTool("create_task")

	// Create loader with registry
	loader := &SkillsLoader{
		toolRegistry: registry,
		skills: map[string]*SkillMetadata{
			"test-skill": {
				Name:        "test-skill",
				Description: "Test skill description",
				Path:        "/fake/path/test-skill",
				ToolsUsed:   []string{"configure_dashboard", "missing_tool"},
			},
		},
		loaded: true,
	}

	// Validate skill
	issues := loader.ValidateSkill("test-skill")

	// Should have at least one issue for the missing tool
	foundToolIssue := false
	for _, issue := range issues {
		if issue == "tool not found in registry: missing_tool" {
			foundToolIssue = true
			break
		}
	}

	assert.True(t, foundToolIssue, "Should report missing tool. Issues: %v", issues)
}
