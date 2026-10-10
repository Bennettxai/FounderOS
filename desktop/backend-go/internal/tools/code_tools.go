package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CodeToolRegistry manages code-related tools for the Coding Agent
type CodeToolRegistry struct {
	workspaceRoot string
	backupMgr     *BackupManager
	allowedCmds   map[string]bool
}

// NewCodeToolRegistry creates a new code tool registry
func NewCodeToolRegistry(workspaceRoot string) *CodeToolRegistry {
	return &CodeToolRegistry{
		workspaceRoot: workspaceRoot,
		backupMgr:     NewBackupManager(workspaceRoot),
		allowedCmds: map[string]bool{
			"go":     true,
			"npm":    true,
			"node":   true,
			"git":    true,
			"cat":    true,
			"ls":     true,
			"find":   true,
			"grep":   true,
			"head":   true,
			"tail":   true,
			"wc":     true,
			"diff":   true,
			"make":   true,
			"cargo":  true,
			"python": true,
			"pip":    true,
			"yarn":   true,
			"pnpm":   true,
			"tsc":    true,
			"eslint": true,
		},
	}
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (r *CodeToolRegistry) resolveAndValidatePath(path string) (string, error) {
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(r.workspaceRoot, path))
	}

	// Validate path is under workspace
	if !strings.HasPrefix(absPath, r.workspaceRoot) {
		return "", fmt.Errorf("path must be under workspace: %s", path)
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}

	return absPath, nil
}

func generateSimpleDiff(old, new string) string {
	var sb strings.Builder
	sb.WriteString("--- old\n+++ new\n")

	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	for _, line := range oldLines {
		sb.WriteString("- " + line + "\n")
	}
	for _, line := range newLines {
		sb.WriteString("+ " + line + "\n")
	}

	return sb.String()
}

func isBinaryFile(name string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".db": true, ".sqlite": true,
	}
	ext := strings.ToLower(filepath.Ext(name))
	return binaryExts[ext]
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
