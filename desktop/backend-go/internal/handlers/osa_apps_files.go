package handlers

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// FileNode represents a file or directory in the file tree
type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	Size     int64      `json:"size,omitempty"`
	Children []FileNode `json:"children,omitempty"`
	Modified *string    `json:"modified,omitempty"`
}

// FileNodeFlat represents a file without nested children (flat list)
type FileNodeFlat struct {
	ID       uuid.UUID `json:"id"`
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Language *string   `json:"language,omitempty"`
	Size     int64     `json:"size"`
	Lines    *int32    `json:"lines,omitempty"`
	Modified *string   `json:"modified,omitempty"`
	Hash     string    `json:"hash"`
	Status   string    `json:"status"`
}

// FileListResponse represents the response for listing app files
type FileListResponse struct {
	AppID       uuid.UUID      `json:"app_id"`
	Repository  string         `json:"repository,omitempty"`
	Files       []FileNodeFlat `json:"files"`
	Total       int64          `json:"total"`
	Limit       int32          `json:"limit"`
	Offset      int32          `json:"offset"`
	CurrentPath string         `json:"current_path,omitempty"`
}

// SaveFileRequest is the request body for PUT /api/osa/apps/:id/files
type SaveFileRequest struct {
	FilePath string `json:"file_path" binding:"required"`
	Content  string `json:"content"   binding:"required"`
}

// GetAppFiles - GET /api/osa/apps/:id/files
func (h *OSAAppsHandler) GetAppFiles(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		h.logger.Error("invalid user ID", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	app, err := h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("app not found or access denied for files", "app_id", appID, "user_id", userID)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	// Parse query parameters
	pathFilter := c.Query("path")
	typeFilter := c.Query("type")
	languageFilter := c.Query("language")

	// Validate path parameter (prevent directory traversal attacks)
	if pathFilter != "" && !isValidPath(pathFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path parameter"})
		return
	}

	// Pagination
	limit := int32(50)
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseIntParam(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := int32(0)
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := parseIntParam(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Query files from database
	files, err := h.queries.ListFilesByApp(c.Request.Context(), pgAppID)
	if err != nil {
		h.logger.Error("failed to list files", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	// Filter files based on query parameters
	var filteredFiles []sqlc.OsaModuleFile
	for _, file := range files {
		// Filter by path
		if pathFilter != "" && !strings.HasPrefix(file.FilePath, pathFilter) {
			continue
		}

		// Filter by type
		if typeFilter != "" && file.FileType != typeFilter {
			continue
		}

		// Filter by language
		if languageFilter != "" {
			if file.Language == nil || *file.Language != languageFilter {
				continue
			}
		}

		filteredFiles = append(filteredFiles, file)
	}

	// Apply pagination
	total := int64(len(filteredFiles))
	start := int(offset)
	end := int(offset) + int(limit)

	if start > len(filteredFiles) {
		start = len(filteredFiles)
	}
	if end > len(filteredFiles) {
		end = len(filteredFiles)
	}

	paginatedFiles := filteredFiles[start:end]

	// Convert to response format
	responseFiles := make([]FileNodeFlat, len(paginatedFiles))
	for i, file := range paginatedFiles {
		responseFiles[i] = FileNodeFlat{
			ID:       file.ID.Bytes,
			Path:     file.FilePath,
			Name:     file.FileName,
			Type:     file.FileType,
			Language: file.Language,
			Size:     int64(file.FileSizeBytes),
			Lines:    file.LineCount,
			Modified: formatTimestamp(file.UpdatedAt),
			Hash:     file.ContentHash,
			Status:   getInstallationStatus(file.InstallationStatus),
		}
	}

	// Get repository path if available
	repository := ""
	if app.CodeRepository != nil {
		repository = *app.CodeRepository
	}

	response := FileListResponse{
		AppID:       appID,
		Repository:  repository,
		Files:       responseFiles,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
		CurrentPath: pathFilter,
	}

	h.logger.Info("files listed successfully",
		"app_id", appID,
		"total", total,
		"returned", len(responseFiles),
		"path_filter", pathFilter,
	)

	c.JSON(http.StatusOK, response)
}

// DownloadApp - GET /api/osa/apps/:id/download
// Downloads the generated app as a ZIP archive
func (h *OSAAppsHandler) DownloadApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership for download", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	// Check workspace directory exists
	workspaceDir := filepath.Join("/tmp/businessos-agent-workspaces", appID.String())
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Generated app files not found"})
		return
	}

	h.logger.Info("downloading generated app", "app_id", appID, "user_id", user.ID)

	// Set headers for ZIP download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"app-%s.zip\"", appIDStr[:8]))

	// Create ZIP writer directly to response
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// Walk workspace directory and add files to ZIP
	err = filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path for ZIP entry
		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}

		// Create ZIP entry
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		h.logger.Error("failed to create ZIP archive", "error", err, "app_id", appID)
		// Can't send JSON error since we already started writing ZIP headers
		return
	}

	h.logger.Info("app download completed", "app_id", appID)
}

// GetAppGeneratedFiles - GET /api/osa/apps/:id/generated-files
// Returns list of generated files with their content from the database
func (h *OSAAppsHandler) GetAppGeneratedFiles(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership for generated files", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	dbFiles, err := h.queries.ListFilesByApp(c.Request.Context(), pgAppID)
	if err != nil {
		h.logger.Error("failed to list generated files", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read generated files"})
		return
	}

	if len(dbFiles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Generated app files not found"})
		return
	}

	type FileEntry struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Size    int64  `json:"size"`
	}

	files := make([]FileEntry, 0, len(dbFiles))
	for _, f := range dbFiles {
		files = append(files, FileEntry{
			Path:    f.FilePath,
			Content: f.Content,
			Size:    int64(f.FileSizeBytes),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"app_id":      appID,
		"files":       files,
		"total_files": len(files),
	})
}

// SaveAppFile - PUT /api/osa/apps/:id/files
// Saves updated file content from the Monaco editor to the database.
func (h *OSAAppsHandler) SaveAppFile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	appIDStr := c.Param("id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid app ID"})
		return
	}

	// Verify ownership
	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	_, err = h.queries.GetOSAModuleInstanceByIDWithAuth(c.Request.Context(), sqlc.GetOSAModuleInstanceByIDWithAuthParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		h.logger.Error("failed to verify app ownership for file save", "error", err, "app_id", appID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ownership"})
		return
	}

	var req SaveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !isValidPath(req.FilePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	// Limit content size to 1MB
	const maxFileSize = 1 << 20
	if len(req.Content) > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File content exceeds maximum size of 1MB"})
		return
	}

	// Compute content hash (SHA256, same pattern as saveFileToDatabase)
	hash := sha256.Sum256([]byte(req.Content))
	contentHash := hex.EncodeToString(hash[:])
	lineCount := int32(strings.Count(req.Content, "\n") + 1)
	fileSize := int64(len(req.Content))

	var updatedID uuid.UUID
	var updatedPath, updatedName string
	var updatedSize int64
	var updatedLines int32
	var updatedHash string
	var updatedAt time.Time

	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE osa_module_files
		SET content = $1, content_hash = $2, file_size_bytes = $3, line_count = $4, updated_at = NOW()
		WHERE module_instance_id = $5 AND file_path = $6
		RETURNING id, file_path, file_name, file_size_bytes, line_count, content_hash, updated_at
	`, req.Content, contentHash, fileSize, lineCount, pgAppID, req.FilePath,
	).Scan(&updatedID, &updatedPath, &updatedName, &updatedSize, &updatedLines, &updatedHash, &updatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		h.logger.Error("failed to save file", "error", err, "app_id", appID, "file_path", req.FilePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	h.logger.Info("file saved successfully",
		"app_id", appID,
		"file_path", req.FilePath,
		"size_bytes", fileSize,
	)

	c.JSON(http.StatusOK, gin.H{
		"id":              updatedID,
		"file_path":       updatedPath,
		"file_name":       updatedName,
		"file_size_bytes": updatedSize,
		"line_count":      updatedLines,
		"content_hash":    updatedHash,
		"updated_at":      updatedAt.Format(time.RFC3339),
	})
}

// GetQueueItemStatus returns the current status of a queue item (for polling fallback)
func (h *OSAAppsHandler) GetQueueItemStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	queueItemIDStr := c.Param("queue_item_id")
	queueItemID, err := uuid.Parse(queueItemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid queue item ID"})
		return
	}

	var status string
	var completedAt *time.Time
	var errorMessage *string
	var createdAt time.Time
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT q.status, q.completed_at, q.error_message, q.created_at
		 FROM app_generation_queue q
		 JOIN workspaces w ON w.id = q.workspace_id
		 WHERE q.id = $1 AND w.owner_id = $2`,
		queueItemID, user.ID,
	).Scan(&status, &completedAt, &errorMessage, &createdAt)

	if err != nil {
		h.logger.Error("failed to get queue item status", "error", err, "queue_item_id", queueItemID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Queue item not found"})
		return
	}

	// Check if workspace has files
	workspaceDir := filepath.Join("/tmp/businessos-agent-workspaces", queueItemID.String())
	hasFiles := false
	if _, statErr := os.Stat(workspaceDir); statErr == nil {
		entries, _ := os.ReadDir(workspaceDir)
		hasFiles = len(entries) > 0
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_item_id": queueItemID,
		"status":        status,
		"completed_at":  completedAt,
		"created_at":    createdAt,
		"has_files":     hasFiles,
		"error_message": errorMessage,
	})
}
