package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/container"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// ReadFile reads the content of a file.
func (h *FilesystemHandler) ReadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

	// Check if container isolation is enabled
	if h.containerMgr != nil {
		h.readFileContainer(c, filePath)
		return
	}

	// Fallback to local filesystem
	h.readFileLocal(c, filePath)
}

// readFileContainer reads file content from user's container workspace.
func (h *FilesystemHandler) readFileContainer(c *gin.Context, filePath string) {
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

	// Read file from container
	content, err := h.containerMgr.ReadFileFromContainer(containerID, filePath)
	if err != nil {
		slog.Info("[Filesystem] Failed to read file", "id", filePath, "error", err)
		// Check for file size error using type assertion
		var fileSizeErr *container.FileSizeError
		if errors.As(err, &fileSizeErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large for preview"})
			return
		}
		utils.RespondNotFound(c, slog.Default(), "File")
		return
	}

	// Detect MIME type
	mimeType := http.DetectContentType(content)

	c.JSON(http.StatusOK, FileContentResponse{
		Path:     filePath,
		Name:     filepath.Base(filePath),
		Content:  string(content),
		Size:     int64(len(content)),
		MimeType: mimeType,
	})
}

// readFileLocal reads file content from local filesystem (fallback).
func (h *FilesystemHandler) readFileLocal(c *gin.Context, filePath string) {
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

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.RespondNotFound(c, slog.Default(), "File")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "access file", nil)
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory, not a file"})
		return
	}

	// Limit file size for reading (10MB max for text preview)
	const maxReadSize = 10 * 1024 * 1024
	if info.Size() > maxReadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large for preview"})
		return
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "read file", nil)
		return
	}

	// Detect MIME type
	mimeType := http.DetectContentType(content)

	c.JSON(http.StatusOK, FileContentResponse{
		Path:     filePath,
		Name:     filepath.Base(filePath),
		Content:  string(content),
		Size:     info.Size(),
		MimeType: mimeType,
	})
}

// DownloadFile downloads a file via the local filesystem.
func (h *FilesystemHandler) DownloadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

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

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.RespondNotFound(c, slog.Default(), "File")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "access file", nil)
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot download a directory"})
		return
	}

	// Serve the file
	c.File(filePath)
}

// GetFileInfo returns information about a file or directory.
func (h *FilesystemHandler) GetFileInfo(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is required"})
		return
	}

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

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.RespondNotFound(c, slog.Default(), "Path")
			return
		}
		utils.RespondInternalError(c, slog.Default(), "access path", nil)
		return
	}

	name := filepath.Base(filePath)
	modTime := info.ModTime()

	item := FileItem{
		ID:       generateFileID(filePath),
		Name:     name,
		Path:     filePath,
		IsHidden: strings.HasPrefix(name, "."),
		Modified: &modTime,
	}

	if info.IsDir() {
		item.Type = "folder"
		// Count items in directory
		entries, err := os.ReadDir(filePath)
		if err == nil {
			item.Size = int64(len(entries))
		}
	} else {
		item.Type = "file"
		item.Size = info.Size()
		item.Extension = strings.TrimPrefix(filepath.Ext(name), ".")
	}

	c.JSON(http.StatusOK, item)
}

// UploadFile handles file uploads.
func (h *FilesystemHandler) UploadFile(c *gin.Context) {
	// Enforce upload size limit before parsing the multipart body.
	// This prevents memory exhaustion and DoS attacks (OWASP A03).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	destPath := c.PostForm("path")
	if destPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Destination path is required"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		// http.MaxBytesReader surfaces as a *http.MaxBytesError when the body is too large.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("File too large: maximum allowed size is %d MB", maxUploadSize/(1024*1024)),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Read file content (MaxBytesReader already limits the stream).
	content, err := io.ReadAll(file)
	if err != nil {
		// If the limit is hit mid-stream, MaxBytesReader also returns an error here.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("File too large: maximum allowed size is %d MB", maxUploadSize/(1024*1024)),
			})
			return
		}
		utils.RespondInternalError(c, slog.Default(), "read uploaded file", nil)
		return
	}

	// Check if container isolation is enabled
	if h.containerMgr != nil {
		h.uploadFileContainer(c, destPath, header.Filename, content)
		return
	}

	// Fallback to local filesystem
	h.uploadFileLocal(c, destPath, header.Filename, content)
}

// uploadFileContainer uploads a file to user's container workspace.
func (h *FilesystemHandler) uploadFileContainer(c *gin.Context, destPath, filename string, content []byte) {
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

	// Validate destination path to prevent traversal attacks
	destPath, err = validateContainerPath(destPath)
	if err != nil {
		slog.Info("[Filesystem] Path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path: " + err.Error()})
		return
	}

	fullPath := filepath.Join(destPath, filename)

	// Validate the final file path
	fullPath, err = validateContainerPath(fullPath)
	if err != nil {
		slog.Info("[Filesystem] File path validation failed for user", "id", userIDStr, "id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: " + err.Error()})
		return
	}

	// Write file to container
	if err := h.containerMgr.WriteFileToContainer(containerID, fullPath, content, 0644); err != nil {
		slog.Info("[Filesystem] Failed to upload file", "id", fullPath, "error", err)
		utils.RespondInternalError(c, slog.Default(), "save file", nil)
		return
	}

	now := time.Now()
	c.JSON(http.StatusCreated, FileItem{
		ID:        generateFileID(fullPath),
		Name:      filename,
		Type:      "file",
		Path:      fullPath,
		Size:      int64(len(content)),
		Modified:  &now,
		Extension: strings.TrimPrefix(filepath.Ext(filename), "."),
		IsHidden:  strings.HasPrefix(filename, "."),
	})
}

// uploadFileLocal uploads a file to local filesystem (fallback).
func (h *FilesystemHandler) uploadFileLocal(c *gin.Context, destPath, filename string, content []byte) {
	// Expand ~ to home directory
	if strings.HasPrefix(destPath, "~/") {
		homeDir, _ := os.UserHomeDir()
		destPath = filepath.Join(homeDir, destPath[2:])
	}

	// Validate destination path to prevent path traversal attacks
	homeDir, _ := os.UserHomeDir()
	cleanDestPath, err := validatePathSafety(destPath, homeDir)
	if err != nil {
		slog.Warn("Destination path validation failed", "path", destPath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path: " + err.Error()})
		return
	}

	// Create full file path
	fullPath := filepath.Join(cleanDestPath, filename)

	// Validate the final file path
	cleanFullPath, err := validatePathSafety(fullPath, homeDir)
	if err != nil {
		slog.Warn("File path validation failed", "path", fullPath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: " + err.Error()})
		return
	}
	fullPath = cleanFullPath

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create file", nil)
		return
	}
	defer dst.Close()

	// Write content
	if _, err := dst.Write(content); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", nil)
		return
	}

	// Get file info
	info, _ := os.Stat(fullPath)
	modTime := info.ModTime()

	c.JSON(http.StatusCreated, FileItem{
		ID:        generateFileID(fullPath),
		Name:      filename,
		Type:      "file",
		Path:      fullPath,
		Size:      info.Size(),
		Modified:  &modTime,
		Extension: strings.TrimPrefix(filepath.Ext(filename), "."),
		IsHidden:  strings.HasPrefix(filename, "."),
	})
}
