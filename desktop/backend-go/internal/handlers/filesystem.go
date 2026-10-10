package handlers

import (
	"fmt"
	"time"

	"github.com/rhl/businessos-backend/internal/container"
)

const (
	// containerWorkspaceRoot is the root directory in containers.
	containerWorkspaceRoot = "/workspace"

	// maxUploadSize is the maximum allowed file upload size (50 MB).
	// Uploads exceeding this limit are rejected with HTTP 413.
	maxUploadSize = 50 * 1024 * 1024
)

// FilesystemHandler handles filesystem operations with container isolation.
type FilesystemHandler struct {
	containerMgr *container.ContainerManager
}

// NewFilesystemHandler creates a new FilesystemHandler.
func NewFilesystemHandler(containerMgr *container.ContainerManager) *FilesystemHandler {
	return &FilesystemHandler{containerMgr: containerMgr}
}

// FileItem represents a file or directory.
type FileItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"` // "file" or "folder"
	Path      string     `json:"path"`
	Size      int64      `json:"size,omitempty"`
	Modified  *time.Time `json:"modified,omitempty"`
	Extension string     `json:"extension,omitempty"`
	IsHidden  bool       `json:"isHidden"`
}

// ListDirectoryRequest represents the request for listing a directory.
type ListDirectoryRequest struct {
	Path       string `json:"path" form:"path"`
	ShowHidden bool   `json:"showHidden" form:"showHidden"`
}

// ListDirectoryResponse represents the response for listing a directory.
type ListDirectoryResponse struct {
	Path      string     `json:"path"`
	Items     []FileItem `json:"items"`
	ParentDir string     `json:"parentDir,omitempty"`
}

// FileContentResponse represents the response for reading a file.
type FileContentResponse struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
}

// generateFileID creates a stable, unique ID for a file path.
func generateFileID(path string) string {
	return fmt.Sprintf("file_%x", path)
}
