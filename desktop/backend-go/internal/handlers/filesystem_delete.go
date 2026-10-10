package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// DeleteFileOrDir deletes a file or empty directory.
func (h *FilesystemHandler) DeleteFileOrDir(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	// Check if container isolation is enabled
	if h.containerMgr != nil {
		h.deleteFileOrDirContainer(c, filePath)
		return
	}

	// Fallback to local filesystem
	h.deleteFileOrDirLocal(c, filePath)
}

// deleteFileOrDirContainer deletes a file/directory from user's container workspace.
func (h *FilesystemHandler) deleteFileOrDirContainer(c *gin.Context, filePath string) {
	// Get user from context
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userIDStr := user.ID

	// Get or create filesystem container
	containerID, err := h.containerMgr.GetOrCreateFilesystemContainer(userIDStr)
	if err != nil {
		slog.Info("[Filesystem] Failed to get container for user", "id", userIDStr, "error", err)
		utils.RespondInternalError(c, slog.Default(), "access filesystem", nil)
		return
	}

	// Validate path to prevent traversal attacks
	filePath, err = validateContainerPath(filePath)
	if err != nil {
		slog.Info("[Filesystem] Path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	// Safety: prevent deleting workspace root
	if filePath == containerWorkspaceRoot || filePath == containerWorkspaceRoot+"/" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete workspace root"})
		return
	}

	// Delete in container
	if err := h.containerMgr.DeletePathInContainer(containerID, filePath); err != nil {
		slog.Info("[Filesystem] Failed to delete", "id", filePath, "error", err)
		utils.RespondInternalError(c, slog.Default(), "delete", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully", "path": filePath})
}

// deleteFileOrDirLocal deletes a file/directory from local filesystem (fallback).
func (h *FilesystemHandler) deleteFileOrDirLocal(c *gin.Context, filePath string) {
	// Expand ~ to home directory
	if strings.HasPrefix(filePath, "~/") {
		homeDir, _ := os.UserHomeDir()
		filePath = filepath.Join(homeDir, filePath[2:])
	}

	// Validate path to prevent path traversal attacks
	homeDir, _ := os.UserHomeDir()
	cleanPath, err := validatePathSafety(filePath, homeDir)
	if err != nil {
		slog.Warn("Path validation failed", "path", filePath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}
	filePath = cleanPath

	// Safety: prevent deleting system directories
	dangerousPaths := []string{"/", "/bin", "/usr", "/etc", "/System", homeDir}
	for _, dangerous := range dangerousPaths {
		if filePath == dangerous {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete this path"})
			return
		}
	}

	// Check if path exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.RespondNotFound(c, slog.Default(), "Path")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "access path", nil)
		return
	}

	// For directories, only allow deleting if empty (use recursive flag for non-empty)
	if info.IsDir() {
		entries, err := os.ReadDir(filePath)
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "read directory", nil)
			return
		}
		if len(entries) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Directory is not empty. Use recursive delete."})
			return
		}
	}

	// Delete the file/directory
	if err := os.Remove(filePath); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully", "path": filePath})
}
