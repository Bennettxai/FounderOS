package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePathSafety checks for path traversal attacks and other malicious patterns.
// Returns the absolute, cleaned path and an error if the path is unsafe.
func validatePathSafety(inputPath string, baseDir string) (string, error) {
	// Reject paths containing dangerous patterns
	if strings.Contains(inputPath, "..") {
		return "", fmt.Errorf("path contains directory traversal sequence")
	}
	if strings.Contains(inputPath, "://") {
		return "", fmt.Errorf("path contains URL scheme")
	}

	// Get absolute path
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Clean the path to remove any remaining dangerous elements
	cleanPath := filepath.Clean(absPath)

	// Validate that the resolved path stays within the allowed base directory
	if baseDir != "" {
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve base directory: %w", err)
		}

		// Ensure the clean path is within the base directory (strict prefix with separator)
		if cleanPath != absBase && !strings.HasPrefix(cleanPath, absBase+string(filepath.Separator)) {
			return "", fmt.Errorf("path escapes base directory")
		}
	}

	return cleanPath, nil
}

// validateContainerPath validates paths for container operations.
// Ensures paths stay within containerWorkspaceRoot.
func validateContainerPath(inputPath string) (string, error) {
	// Normalize path - default to workspace root
	dirPath := inputPath
	if dirPath == "" || dirPath == "~" || dirPath == "/" {
		dirPath = containerWorkspaceRoot
	}

	// Expand ~ to workspace root
	if strings.HasPrefix(dirPath, "~/") {
		dirPath = filepath.Join(containerWorkspaceRoot, dirPath[2:])
	}

	// Ensure path is under workspace
	if !strings.HasPrefix(dirPath, containerWorkspaceRoot) {
		dirPath = filepath.Join(containerWorkspaceRoot, dirPath)
	}

	// Reject paths containing dangerous patterns
	if strings.Contains(dirPath, "..") {
		return "", fmt.Errorf("path contains directory traversal sequence")
	}
	if strings.Contains(dirPath, "://") {
		return "", fmt.Errorf("path contains URL scheme")
	}

	// Clean and validate
	cleanPath := filepath.Clean(dirPath)

	// Final check: ensure we're still within workspace
	if !strings.HasPrefix(cleanPath, containerWorkspaceRoot) {
		return "", fmt.Errorf("path escapes workspace boundary")
	}

	return cleanPath, nil
}
