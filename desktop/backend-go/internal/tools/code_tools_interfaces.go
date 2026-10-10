package tools

import (
	"context"
	"encoding/json"
)

// ========================================
// TOOL INTERFACE IMPLEMENTATIONS
// ========================================

// ReadFileTool implements AgentTool interface
type ReadFileTool struct {
	registry *CodeToolRegistry
}

func NewReadFileTool(workspaceRoot string) *ReadFileTool {
	return &ReadFileTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read file content with line numbers" }
func (t *ReadFileTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":       map[string]interface{}{"type": "string", "description": "File path relative to workspace"},
			"start_line": map[string]interface{}{"type": "integer", "description": "Start line (1-indexed)"},
			"end_line":   map[string]interface{}{"type": "integer", "description": "End line (1-indexed)"},
		},
		"required": []string{"path"},
	}
}
func (t *ReadFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.ReadFile(ctx, input)
}

// WriteFileTool implements AgentTool interface
type WriteFileTool struct {
	registry *CodeToolRegistry
}

func NewWriteFileTool(workspaceRoot string) *WriteFileTool {
	return &WriteFileTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Write content to file (creates backup automatically)"
}
func (t *WriteFileTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string", "description": "File path"},
			"content": map[string]interface{}{"type": "string", "description": "Full file content"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *WriteFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.WriteFile(ctx, input)
}

// EditFileTool implements AgentTool interface
type EditFileTool struct {
	registry *CodeToolRegistry
}

func NewEditFileTool(workspaceRoot string) *EditFileTool {
	return &EditFileTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *EditFileTool) Name() string        { return "edit_file" }
func (t *EditFileTool) Description() string { return "Make surgical edit to file (find and replace)" }
func (t *EditFileTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":       map[string]interface{}{"type": "string", "description": "File path"},
			"old_string": map[string]interface{}{"type": "string", "description": "Exact text to find"},
			"new_string": map[string]interface{}{"type": "string", "description": "Replacement text"},
			"occurrence": map[string]interface{}{"type": "integer", "description": "Which occurrence (0=all, 1=first)"},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}
func (t *EditFileTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.EditFile(ctx, input)
}

// SearchCodeTool implements AgentTool interface
type SearchCodeTool struct {
	registry *CodeToolRegistry
}

func NewSearchCodeTool(workspaceRoot string) *SearchCodeTool {
	return &SearchCodeTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *SearchCodeTool) Name() string        { return "search_code" }
func (t *SearchCodeTool) Description() string { return "Search for pattern in codebase" }
func (t *SearchCodeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query":        map[string]interface{}{"type": "string", "description": "Search pattern"},
			"path":         map[string]interface{}{"type": "string", "description": "Directory to search"},
			"file_pattern": map[string]interface{}{"type": "string", "description": "Glob pattern (e.g., *.go)"},
			"max_results":  map[string]interface{}{"type": "integer", "description": "Max results"},
			"regex":        map[string]interface{}{"type": "boolean", "description": "Use regex"},
		},
		"required": []string{"query"},
	}
}
func (t *SearchCodeTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.SearchCode(ctx, input)
}

// ListFilesTool implements AgentTool interface
type ListFilesTool struct {
	registry *CodeToolRegistry
}

func NewListFilesTool(workspaceRoot string) *ListFilesTool {
	return &ListFilesTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *ListFilesTool) Name() string        { return "list_files" }
func (t *ListFilesTool) Description() string { return "List files in directory" }
func (t *ListFilesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":      map[string]interface{}{"type": "string", "description": "Directory path"},
			"pattern":   map[string]interface{}{"type": "string", "description": "Glob pattern filter"},
			"recursive": map[string]interface{}{"type": "boolean", "description": "Include subdirectories"},
			"max_depth": map[string]interface{}{"type": "integer", "description": "Max recursion depth"},
		},
	}
}
func (t *ListFilesTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.ListFiles(ctx, input)
}

// RunCommandTool implements AgentTool interface
type RunCommandTool struct {
	registry *CodeToolRegistry
}

func NewRunCommandTool(workspaceRoot string) *RunCommandTool {
	return &RunCommandTool{registry: NewCodeToolRegistry(workspaceRoot)}
}

func (t *RunCommandTool) Name() string { return "run_command" }
func (t *RunCommandTool) Description() string {
	return "Execute shell command in workspace (sandboxed)"
}
func (t *RunCommandTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command":     map[string]interface{}{"type": "string", "description": "Command to execute"},
			"working_dir": map[string]interface{}{"type": "string", "description": "Working directory"},
			"timeout":     map[string]interface{}{"type": "integer", "description": "Timeout in seconds"},
		},
		"required": []string{"command"},
	}
}
func (t *RunCommandTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.registry.RunCommand(ctx, input)
}
