package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// ListDirectory lists the contents of a directory.
func (h *FilesystemHandler) ListDirectory(c *gin.Context) {
	var req ListDirectoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// Check if container isolation is enabled
	if h.containerMgr != nil {
		h.listDirectoryContainer(c, req)
		return
	}

	// Fallback to local filesystem (development mode)
	h.listDirectoryLocal(c, req)
}

// listDirectoryContainer lists directory contents from user's container workspace.
func (h *FilesystemHandler) listDirectoryContainer(c *gin.Context, req ListDirectoryRequest) {
	// Get user from context
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userIDStr := user.ID

	// Get or create filesystem container for this user
	containerID, err := h.containerMgr.GetOrCreateFilesystemContainer(userIDStr)
	if err != nil {
		slog.Info("[Filesystem] Failed to get container for user", "id", userIDStr, "error", err)
		utils.RespondInternalError(c, slog.Default(), "access filesystem", nil)
		return
	}

	// Validate path to prevent traversal attacks
	dirPath, err := validateContainerPath(req.Path)
	if err != nil {
		slog.Info("[Filesystem] Path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	// List directory in container
	files, err := h.containerMgr.ListDirectoryInContainer(containerID, dirPath)
	if err != nil {
		slog.Info("[Filesystem] Failed to list directory", "id", dirPath, "error", err)
		utils.RespondNotFound(c, slog.Default(), "Directory")
		return
	}

	// Convert container.FileInfo to FileItem
	items := make([]FileItem, 0, len(files))
	for _, f := range files {
		name := f.Name
		isHidden := strings.HasPrefix(name, ".")

		// Skip hidden files unless requested
		if isHidden && !req.ShowHidden {
			continue
		}

		// Skip the base directory entry itself
		if name == "" || name == "." || name == filepath.Base(dirPath) {
			continue
		}

		item := FileItem{
			ID:       generateFileID(f.Path),
			Name:     name,
			Path:     f.Path,
			IsHidden: isHidden,
			Modified: &f.ModTime,
		}

		if f.IsDir {
			item.Type = "folder"
		} else {
			item.Type = "file"
			item.Size = f.Size
			item.Extension = strings.TrimPrefix(filepath.Ext(name), ".")
		}

		items = append(items, item)
	}

	sortFileItems(items)

	// Get parent directory
	parentDir := filepath.Dir(dirPath)
	if parentDir == dirPath || parentDir == containerWorkspaceRoot {
		parentDir = containerWorkspaceRoot // Can't go above workspace
	}

	c.JSON(http.StatusOK, ListDirectoryResponse{
		Path:      dirPath,
		Items:     items,
		ParentDir: parentDir,
	})
}

// listDirectoryLocal lists directory contents from local filesystem (fallback).
func (h *FilesystemHandler) listDirectoryLocal(c *gin.Context, req ListDirectoryRequest) {
	// Default to home directory if no path specified
	dirPath := req.Path
	if dirPath == "" || dirPath == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "get home directory", nil)
			return
		}
		dirPath = homeDir
	}

	// Expand ~ to home directory
	if strings.HasPrefix(dirPath, "~/") {
		homeDir, _ := os.UserHomeDir()
		dirPath = filepath.Join(homeDir, dirPath[2:])
	}

	// Validate path to prevent path traversal attacks
	homeDir, _ := os.UserHomeDir()
	cleanPath, err := validatePathSafety(dirPath, homeDir)
	if err != nil {
		slog.Warn("Path validation failed", "path", dirPath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}
	dirPath = cleanPath

	// Check if path exists and is a directory
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.RespondNotFound(c, slog.Default(), "Directory")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "access directory", nil)
		return
	}

	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is not a directory"})
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "read directory", nil)
		return
	}

	items := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		isHidden := strings.HasPrefix(name, ".")

		// Skip hidden files unless requested
		if isHidden && !req.ShowHidden {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't access
		}

		item := FileItem{
			ID:       generateFileID(filepath.Join(dirPath, name)),
			Name:     name,
			Path:     filepath.Join(dirPath, name),
			IsHidden: isHidden,
		}

		if entry.IsDir() {
			item.Type = "folder"
		} else {
			item.Type = "file"
			item.Size = info.Size()
			item.Extension = strings.TrimPrefix(filepath.Ext(name), ".")
		}

		modTime := info.ModTime()
		item.Modified = &modTime

		items = append(items, item)
	}

	sortFileItems(items)

	// Get parent directory
	parentDir := filepath.Dir(dirPath)
	if parentDir == dirPath {
		parentDir = "" // At root
	}

	c.JSON(http.StatusOK, ListDirectoryResponse{
		Path:      dirPath,
		Items:     items,
		ParentDir: parentDir,
	})
}

// CreateDirectory creates a new directory.
func (h *FilesystemHandler) CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Sanitize directory name
	if strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid directory name"})
		return
	}

	// Check if container isolation is enabled
	if h.containerMgr != nil {
		h.createDirectoryContainer(c, req.Path, req.Name)
		return
	}

	// Fallback to local filesystem
	h.createDirectoryLocal(c, req.Path, req.Name)
}

// createDirectoryContainer creates a directory in user's container workspace.
func (h *FilesystemHandler) createDirectoryContainer(c *gin.Context, basePath, name string) {
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

	// Validate base path to prevent traversal attacks
	basePath, err = validateContainerPath(basePath)
	if err != nil {
		slog.Info("[Filesystem] Path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	newDirPath := filepath.Join(basePath, name)

	// Validate the final directory path
	newDirPath, err = validateContainerPath(newDirPath)
	if err != nil {
		slog.Info("[Filesystem] New directory path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid directory path: " + err.Error()})
		return
	}

	// Create directory in container
	if err := h.containerMgr.CreateDirectoryInContainer(containerID, newDirPath); err != nil {
		slog.Info("[Filesystem] Failed to create directory", "id", newDirPath, "error", err)
		utils.RespondInternalError(c, slog.Default(), "create directory", nil)
		return
	}

	now := time.Now()
	c.JSON(http.StatusCreated, FileItem{
		ID:       generateFileID(newDirPath),
		Name:     name,
		Type:     "folder",
		Path:     newDirPath,
		IsHidden: strings.HasPrefix(name, "."),
		Modified: &now,
	})
}

// createDirectoryLocal creates a directory on local filesystem (fallback).
func (h *FilesystemHandler) createDirectoryLocal(c *gin.Context, basePath, name string) {
	// Expand ~ to home directory
	if strings.HasPrefix(basePath, "~/") {
		homeDir, _ := os.UserHomeDir()
		basePath = filepath.Join(homeDir, basePath[2:])
	}

	// Validate base path to prevent path traversal attacks
	homeDir, _ := os.UserHomeDir()
	cleanBasePath, err := validatePathSafety(basePath, homeDir)
	if err != nil {
		slog.Warn("Base path validation failed", "path", basePath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base path: " + err.Error()})
		return
	}

	newDirPath := filepath.Join(cleanBasePath, name)

	// Validate the final directory path
	cleanNewPath, err := validatePathSafety(newDirPath, homeDir)
	if err != nil {
		slog.Warn("New directory path validation failed", "path", newDirPath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid directory path: " + err.Error()})
		return
	}
	newDirPath = cleanNewPath

	// Check if already exists
	if _, err := os.Stat(newDirPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Directory already exists"})
		return
	}

	// Create directory
	if err := os.MkdirAll(newDirPath, 0755); err != nil {
		utils.RespondInternalError(c, slog.Default(), "create directory", nil)
		return
	}

	// Return info about new directory
	info, _ := os.Stat(newDirPath)
	modTime := info.ModTime()

	c.JSON(http.StatusCreated, FileItem{
		ID:       generateFileID(newDirPath),
		Name:     name,
		Type:     "folder",
		Path:     newDirPath,
		IsHidden: strings.HasPrefix(name, "."),
		Modified: &modTime,
	})
}

// GetQuickAccessPaths returns commonly used paths for navigation shortcuts.
func (h *FilesystemHandler) GetQuickAccessPaths(c *gin.Context) {
	// Check if container isolation is enabled
	if h.containerMgr != nil {
		// Return container workspace paths
		paths := []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Icon string `json:"icon"`
		}{
			{Name: "Workspace", Path: containerWorkspaceRoot, Icon: "home"},
			{Name: "Documents", Path: filepath.Join(containerWorkspaceRoot, "documents"), Icon: "document"},
			{Name: "Projects", Path: filepath.Join(containerWorkspaceRoot, "projects"), Icon: "folder"},
			{Name: "Downloads", Path: filepath.Join(containerWorkspaceRoot, "downloads"), Icon: "download"},
		}
		c.JSON(http.StatusOK, gin.H{"paths": paths})
		return
	}

	// Fallback to local filesystem paths
	homeDir, _ := os.UserHomeDir()

	paths := []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Icon string `json:"icon"`
	}{
		{Name: "Home", Path: homeDir, Icon: "home"},
		{Name: "Desktop", Path: filepath.Join(homeDir, "Desktop"), Icon: "desktop"},
		{Name: "Documents", Path: filepath.Join(homeDir, "Documents"), Icon: "document"},
		{Name: "Downloads", Path: filepath.Join(homeDir, "Downloads"), Icon: "download"},
		{Name: "Pictures", Path: filepath.Join(homeDir, "Pictures"), Icon: "image"},
		{Name: "Music", Path: filepath.Join(homeDir, "Music"), Icon: "music"},
		{Name: "Videos", Path: filepath.Join(homeDir, "Movies"), Icon: "video"},
	}

	// Filter to only paths that exist
	validPaths := make([]struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Icon string `json:"icon"`
	}, 0)

	for _, p := range paths {
		if _, err := os.Stat(p.Path); err == nil {
			validPaths = append(validPaths, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{"paths": validPaths})
}

// sortFileItems sorts items: folders first, then alphabetically (case-insensitive).
func sortFileItems(items []FileItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type == "folder"
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}
