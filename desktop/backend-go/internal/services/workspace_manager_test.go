package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestWorkspaceManager_CreateWorkspace(t *testing.T) {
	// Use temp directory for testing
	baseDir := filepath.Join(os.TempDir(), "test-workspaces")
	defer os.RemoveAll(baseDir)

	wm := NewWorkspaceManager(baseDir, nil)

	appID := uuid.New()
	workspacePath, err := wm.CreateWorkspace(appID)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Verify workspace directory exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Error("Workspace directory was not created")
	}

	// Verify subdirectories exist
	subdirs := []string{"frontend", "backend", "database", "tests"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(workspacePath, subdir)
		if _, err := os.Stat(subdirPath); os.IsNotExist(err) {
			t.Errorf("Subdirectory %s was not created", subdir)
		}
	}

	// Verify path format
	expectedPath := filepath.Join(baseDir, appID.String())
	if workspacePath != expectedPath {
		t.Errorf("Expected workspace path %s, got %s", expectedPath, workspacePath)
	}
}

func TestWorkspaceManager_SaveFile(t *testing.T) {
	baseDir := filepath.Join(os.TempDir(), "test-workspaces-save")
	defer os.RemoveAll(baseDir)

	wm := NewWorkspaceManager(baseDir, nil)

	appID := uuid.New()
	workspacePath, err := wm.CreateWorkspace(appID)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Save a file
	relativePath := "frontend/src/App.svelte"
	content := `<script>let count = 0;</script><button on:click={() => count++}>{count}</button>`

	err = wm.SaveFile(workspacePath, relativePath, content)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify file exists and has correct content
	fullPath := filepath.Join(workspacePath, relativePath)
	savedContent, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(savedContent) != content {
		t.Errorf("File content mismatch.\nExpected: %s\nGot: %s", content, string(savedContent))
	}
}

func TestWorkspaceManager_SaveFileCreatesParentDirs(t *testing.T) {
	baseDir := filepath.Join(os.TempDir(), "test-workspaces-dirs")
	defer os.RemoveAll(baseDir)

	wm := NewWorkspaceManager(baseDir, nil)

	appID := uuid.New()
	workspacePath, err := wm.CreateWorkspace(appID)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Save file with deep nested path
	relativePath := "backend/internal/handler/user.go"
	content := "package handler"

	err = wm.SaveFile(workspacePath, relativePath, content)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(workspacePath, relativePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("File with nested path was not created")
	}
}

func TestWorkspaceManager_CleanupWorkspace(t *testing.T) {
	baseDir := filepath.Join(os.TempDir(), "test-workspaces-cleanup")
	defer os.RemoveAll(baseDir)

	wm := NewWorkspaceManager(baseDir, nil)

	appID := uuid.New()
	workspacePath, err := wm.CreateWorkspace(appID)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Verify workspace exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		t.Fatal("Workspace was not created")
	}

	// Cleanup workspace
	err = wm.CleanupWorkspace(appID)
	if err != nil {
		t.Fatalf("CleanupWorkspace failed: %v", err)
	}

	// Verify workspace is gone
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Error("Workspace still exists after cleanup")
	}
}

func TestParseCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "single code block with colon syntax",
			input: `Here's a component:

` + "```svelte:src/App.svelte" + `
<script>
let count = 0;
</script>
<button on:click={() => count++}>{count}</button>
` + "```",
			expected: map[string]string{
				"src/App.svelte": `<script>
let count = 0;
</script>
<button on:click={() => count++}>{count}</button>`,
			},
		},
		{
			name: "multiple code blocks",
			input: `Here are the files:

` + "```go:internal/handler/user.go" + `
package handler

func GetUser() {}
` + "```" + `

` + "```go:internal/service/user.go" + `
package service

func FindUser() {}
` + "```",
			expected: map[string]string{
				"internal/handler/user.go": `package handler

func GetUser() {}`,
				"internal/service/user.go": `package service

func FindUser() {}`,
			},
		},
		{
			name: "code block with file comment",
			input: `Here's the file:

` + "```typescript" + `
// File: src/utils/helper.ts
export function helper() {
  return true;
}
` + "```",
			expected: map[string]string{
				"src/utils/helper.ts": `export function helper() {
  return true;
}`,
			},
		},
		{
			name: "code block without filename (skipped)",
			input: `Example code:

` + "```typescript" + `
const x = 1;
` + "```",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCodeBlocks(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d files, got %d", len(tt.expected), len(result))
			}

			for filename, expectedContent := range tt.expected {
				actualContent, exists := result[filename]
				if !exists {
					t.Errorf("Expected file %s not found in result", filename)
					continue
				}

				if actualContent != expectedContent {
					t.Errorf("Content mismatch for %s.\nExpected:\n%s\nGot:\n%s",
						filename, expectedContent, actualContent)
				}
			}
		})
	}
}

func TestInferFileCategory(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"App.svelte", "frontend"},
		{"Button.tsx", "frontend"},
		{"src/components/Header.jsx", "frontend"},
		{"user.go", "backend"},
		{"internal/handler/auth.go", "backend"},
		{"service/email.go", "backend"},
		{"001_create_users.sql", "database"},
		{"migrations/002_add_posts.sql", "database"},
		{"schema.sql", "database"},
		{"user_test.go", "tests"},
		{"App.test.ts", "tests"},
		{"Button.spec.tsx", "tests"},
		{"README.md", "frontend"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := InferFileCategory(tt.filename)
			if result != tt.expected {
				t.Errorf("InferFileCategory(%s) = %s, expected %s",
					tt.filename, result, tt.expected)
			}
		})
	}
}
