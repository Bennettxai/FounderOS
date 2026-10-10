package services

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// WorkspaceManager handles isolated workspace directories for generated apps
type WorkspaceManager struct {
	baseDir string
	logger  *slog.Logger
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(baseDir string, logger *slog.Logger) *WorkspaceManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkspaceManager{
		baseDir: baseDir,
		logger:  logger,
	}
}

// CreateWorkspace creates an isolated directory for an app generation
// Returns the absolute path to the workspace
func (wm *WorkspaceManager) CreateWorkspace(appID uuid.UUID) (string, error) {
	workspacePath := filepath.Join(wm.baseDir, appID.String())

	// Create workspace directory structure
	dirs := []string{
		workspacePath,
		filepath.Join(workspacePath, "frontend"),
		filepath.Join(workspacePath, "backend"),
		filepath.Join(workspacePath, "database"),
		filepath.Join(workspacePath, "tests"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	wm.logger.Info("workspace created",
		"app_id", appID.String(),
		"path", workspacePath,
	)

	return workspacePath, nil
}

// SaveFile saves a file to the workspace with the given relative path
func (wm *WorkspaceManager) SaveFile(workspacePath, relativePath, content string) error {
	fullPath := filepath.Join(workspacePath, relativePath)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	wm.logger.Debug("file saved",
		"path", relativePath,
		"size", len(content),
	)

	return nil
}

// CleanupWorkspace removes a workspace directory
func (wm *WorkspaceManager) CleanupWorkspace(appID uuid.UUID) error {
	workspacePath := filepath.Join(wm.baseDir, appID.String())

	if err := os.RemoveAll(workspacePath); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}

	wm.logger.Info("workspace cleaned up", "app_id", appID.String())
	return nil
}

// GetWorkspacePath returns the path to a workspace
func (wm *WorkspaceManager) GetWorkspacePath(appID uuid.UUID) string {
	return filepath.Join(wm.baseDir, appID.String())
}

// ParseCodeBlocks extracts code blocks from markdown-formatted text
// Returns a map of file paths to file contents
func ParseCodeBlocks(text string) map[string]string {
	files := make(map[string]string)

	// Split by code fence markers
	lines := strings.Split(text, "\n")
	var currentFile string
	var currentContent []string
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for code block start
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				currentContent = []string{}
				currentFile = ""

				// Try to extract filename from code fence
				// Format: ```typescript:path/to/file.ts
				// or: ```typescript path/to/file.ts
				// or: ```go:internal/handler/user.go
				parts := strings.TrimPrefix(line, "```")
				parts = strings.TrimSpace(parts)

				// Look for colon or space separator
				if strings.Contains(parts, ":") {
					filenameParts := strings.SplitN(parts, ":", 2)
					if len(filenameParts) == 2 {
						currentFile = strings.TrimSpace(filenameParts[1])
					}
				} else if strings.Contains(parts, " ") {
					filenameParts := strings.SplitN(parts, " ", 2)
					if len(filenameParts) == 2 {
						potentialPath := strings.TrimSpace(filenameParts[1])
						// Only use if it looks like a path
						if strings.Contains(potentialPath, "/") || strings.Contains(potentialPath, ".") {
							currentFile = potentialPath
						}
					}
				}

				// If no filename in fence, check next few lines for File: comment
				if currentFile == "" && i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(nextLine, "// File:") {
						currentFile = strings.TrimSpace(strings.TrimPrefix(nextLine, "// File:"))
						i++ // Skip this comment line
					} else if strings.HasPrefix(nextLine, "# File:") {
						currentFile = strings.TrimSpace(strings.TrimPrefix(nextLine, "# File:"))
						i++ // Skip this comment line
					}
				}
			} else {
				// End of code block
				inCodeBlock = false
				if currentFile != "" && len(currentContent) > 0 {
					files[currentFile] = strings.Join(currentContent, "\n")
				}
				currentFile = ""
				currentContent = []string{}
			}
			continue
		}

		// Accumulate code block content
		if inCodeBlock {
			currentContent = append(currentContent, line)
		}
	}

	return files
}

// InferFileCategory infers which subdirectory a file belongs to
func InferFileCategory(filename string) string {
	filename = strings.ToLower(filename)

	// Test patterns (check first before other patterns)
	if strings.HasSuffix(filename, "_test.go") ||
		strings.HasSuffix(filename, ".test.ts") ||
		strings.HasSuffix(filename, ".test.tsx") ||
		strings.HasSuffix(filename, ".test.jsx") ||
		strings.HasSuffix(filename, ".spec.ts") ||
		strings.HasSuffix(filename, ".spec.tsx") ||
		strings.HasSuffix(filename, ".spec.jsx") ||
		strings.Contains(filename, "/test/") {
		return "tests"
	}

	// Frontend patterns
	if strings.Contains(filename, ".svelte") ||
		strings.Contains(filename, ".tsx") ||
		strings.Contains(filename, ".jsx") ||
		strings.Contains(filename, "component") ||
		strings.Contains(filename, "frontend") ||
		strings.HasPrefix(filename, "src/") {
		return "frontend"
	}

	// Backend patterns
	if strings.HasSuffix(filename, ".go") ||
		strings.Contains(filename, "handler") ||
		strings.Contains(filename, "service") ||
		strings.Contains(filename, "repository") ||
		strings.Contains(filename, "backend") ||
		strings.Contains(filename, "internal/") {
		return "backend"
	}

	// Database patterns
	if strings.HasSuffix(filename, ".sql") ||
		strings.Contains(filename, "migration") ||
		strings.Contains(filename, "schema") ||
		strings.Contains(filename, "database") {
		return "database"
	}

	// Default to frontend if unsure
	return "frontend"
}
