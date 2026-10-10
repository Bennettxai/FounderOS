package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

const (
	projectFilesDir    = "./data/projects"
	maxProjectFileSize = 50 << 20 // 50 MB
)

// UploadProjectFile handles POST /projects/:id/files.
// Accepts a multipart file, saves it to disk, and returns the download URL.
func (h *ProjectHandler) UploadProjectFile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project id required"})
		return
	}

	// Limit request body to prevent abuse
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProjectFileSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Sanitize filename — strip directory traversal and non-printable chars
	filename := filepath.Base(header.Filename)
	filename = strings.ReplaceAll(filename, " ", "_")

	// Create project-specific upload directory
	dir := filepath.Join(projectFilesDir, projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("UploadProjectFile: failed to create directory", "dir", dir, "error", err)
		utils.RespondInternalError(c, slog.Default(), "create upload dir", nil)
		return
	}

	dst := filepath.Join(dir, filename)
	out, err := os.Create(dst)
	if err != nil {
		slog.Error("UploadProjectFile: failed to create file", "path", dst, "error", err)
		utils.RespondInternalError(c, slog.Default(), "create file", nil)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		slog.Error("UploadProjectFile: failed to write file", "path", dst, "error", err)
		utils.RespondInternalError(c, slog.Default(), "write file", nil)
		return
	}

	slog.Info("UploadProjectFile: saved", "project", projectID, "file", filename, "bytes", written)

	c.JSON(http.StatusOK, gin.H{
		"url":  fmt.Sprintf("/api/projects/%s/files/%s", projectID, filename),
		"name": header.Filename,
		"size": fmt.Sprintf("%.1f MB", float64(written)/1024/1024),
	})
}

// ServeProjectFile handles GET /projects/:id/files/:filename.
// Serves a previously uploaded project file from disk.
func (h *ProjectHandler) ServeProjectFile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	projectID := c.Param("id")
	filename := filepath.Base(c.Param("filename")) // prevent traversal

	path := filepath.Join(projectFilesDir, projectID, filename)

	// Confirm the resolved path is within the expected directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	absBase, _ := filepath.Abs(projectFilesDir)
	if !strings.HasPrefix(absPath, absBase) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.File(absPath)
}
