# Skills Tool Validation

## Overview

The Skills Loader now supports validating that all tools referenced in SKILL.md files actually exist in the tool registry. This prevents runtime errors and improves skill configuration reliability.

## Implementation Details

### CUS-94: Skills Tool Validation

**Status:** ✅ Completed

**Location:** `internal/services/skills_loader.go`

### Key Features

1. **Tool Registry Interface**: Generic interface that any tool registry can implement
2. **Validation Method**: `validateToolReferences()` checks tools_used against registry
3. **Integration**: Automatically runs during `ValidateSkill()` if registry is configured
4. **Optional**: System works with or without tool registry (backward compatible)

## Usage

### Basic Usage (without validation)

```go
// Create loader without tool registry (existing behavior)
loader := services.NewSkillsLoader("./skills/skills.yaml")
err := loader.LoadConfig()
```

### With Tool Validation

```go
// Create a tool registry (must implement ToolRegistry interface)
toolRegistry := tools.NewAgentToolRegistry(pool, userID)

// Option 1: Create loader with registry
loader := services.NewSkillsLoaderWithRegistry("./skills/skills.yaml", toolRegistry)
err := loader.LoadConfig()

// Option 2: Set registry after creation
loader := services.NewSkillsLoader("./skills/skills.yaml")
loader.SetToolRegistry(toolRegistry)
err := loader.LoadConfig()
```

### Validate a Skill

```go
// Validate a specific skill
issues := loader.ValidateSkill("dashboard-management")
if len(issues) > 0 {
    log.Printf("Skill validation issues: %v", issues)
}
```

**Example Output:**
```
[
  "tool not found in registry: configure_dashboard_v2",
  "tool not found in registry: invalid_tool_name"
]
```

## Tool Registry Interface

Any tool registry can be used as long as it implements:

```go
type ToolRegistry interface {
    GetTool(name string) (interface{}, bool)
}
```

### Compatible Registries

- `tools.AgentToolRegistry` - Main agent tool registry (default)
- Any custom tool registry implementing the interface

## SKILL.md Format

Tools are declared in the YAML frontmatter:

```yaml
---
name: dashboard-management
description: Manage custom dashboards
metadata:
  version: "1.0.0"
  tools_used:
    - configure_dashboard
    - create_task
    - update_task
  depends_on: []
---

# Skill Content...
```

## Validation Rules

The `ValidateSkill()` method checks:

1. ✅ SKILL.md file exists
2. ✅ Skill name matches folder name
3. ✅ Description is not empty
4. ✅ tools_used is specified (not empty)
5. ✅ **All tools in tools_used exist in registry** (NEW)

## Integration Example

### In main.go (Server Startup)

```go
// Initialize tool registry first
toolRegistry := tools.NewAgentToolRegistry(pool, "system")

// Create skills loader with registry
skillsLoader := services.NewSkillsLoaderWithRegistry(
    "./skills/skills.yaml",
    toolRegistry,
)

if err := skillsLoader.LoadConfig(); err != nil {
    log.Printf("Warning: Skills loader failed: %v", err)
}

// Validate all loaded skills
for _, skill := range skillsLoader.GetEnabledSkills() {
    issues := skillsLoader.ValidateSkill(skill.Name)
    if len(issues) > 0 {
        slog.Warn("Skill validation issues",
            "skill", skill.Name,
            "issues", issues)
    }
}
```

### In API Handler (Runtime Validation)

```go
func (h *SkillsHandler) ValidateSkill(c *gin.Context) {
    name := c.Param("name")

    issues := h.loader.ValidateSkill(name)

    c.JSON(http.StatusOK, gin.H{
        "skill":  name,
        "valid":  len(issues) == 0,
        "issues": issues,
    })
}
```

## Testing

### Unit Tests

```go
func TestValidateToolReferences(t *testing.T) {
    // Create mock registry
    registry := NewMockToolRegistry()
    registry.RegisterTool("configure_dashboard")

    // Create loader with registry
    loader := &SkillsLoader{
        toolRegistry: registry,
    }

    skill := &SkillMetadata{
        Name: "test-skill",
        ToolsUsed: []string{"configure_dashboard", "missing_tool"},
    }

    issues := loader.validateToolReferences(skill)

    assert.Contains(t, issues, "tool not found in registry: missing_tool")
}
```

### Integration Test

See: `internal/services/skills_loader_test.go`

## Benefits

1. **Early Detection**: Catch missing tools during skill loading, not runtime
2. **Better DX**: Clear error messages about which tools are missing
3. **Reliability**: Ensure skills only reference valid tools
4. **Backward Compatible**: Works with or without registry (optional)
5. **Flexible**: Any tool registry can be used via interface

## Migration Guide

### Existing Code (no changes needed)

```go
// This still works exactly as before
loader := services.NewSkillsLoader("./skills/skills.yaml")
loader.LoadConfig()
```

### Adding Validation (opt-in)

```go
// Add tool registry for validation
toolRegistry := tools.NewAgentToolRegistry(pool, userID)
loader := services.NewSkillsLoaderWithRegistry("./skills/skills.yaml", toolRegistry)
loader.LoadConfig()

// Or set later
loader.SetToolRegistry(toolRegistry)
```

## Files Modified

- `internal/services/skills_loader.go`
  - Added `ToolRegistry` interface
  - Added `toolRegistry` field to `SkillsLoader`
  - Added `NewSkillsLoaderWithRegistry()` constructor
  - Added `SetToolRegistry()` method
  - Added `validateToolReferences()` private method
  - Updated `ValidateSkill()` to call validation

- `internal/services/skills_loader_test.go` (new)
  - Unit tests for tool validation
  - Mock tool registry for testing
  - Integration tests with ValidateSkill

## TODO Resolved

**File:** `internal/services/skills_loader.go:540`

**Before:**
```go
// TODO: Validate that tools referenced in SKILL.md exist in tool registry
```

**After:**
```go
// Validate tool references against tool registry
if toolRegistry != nil {
    toolIssues := l.validateToolReferences(skill)
    issues = append(issues, toolIssues...)
}
```

## See Also

- [SKILL.md Format Documentation](./SKILLS_SYSTEM.md)
- [Tool Registry Documentation](./TOOLS_REGISTRY.md)
- [Agent Tools](../internal/tools/agent_tools.go)
