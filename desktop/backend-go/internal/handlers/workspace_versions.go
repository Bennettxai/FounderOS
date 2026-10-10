package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// WORKSPACE VERSION HANDLERS
// =====================================================================

// CreateWorkspaceVersion creates a snapshot of the current workspace state.
// POST /api/workspaces/:id/versions
func (h *WorkspaceHandler) CreateWorkspaceVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Verify user has admin permissions
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only workspace owners and admins can create versions"})
		return
	}

	versionNumber, err := h.workspaceVersionService.CreateSnapshot(
		c.Request.Context(),
		workspaceID,
		user.ID,
	)
	if err != nil {
		slog.Error("failed to create workspace version", "error", err, "workspace_id", workspaceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create version"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"version_number": versionNumber,
		"message":        "Workspace snapshot created successfully",
	})
}

// ListWorkspaceVersions lists all versions for a workspace.
// GET /api/workspaces/:id/versions
func (h *WorkspaceHandler) ListWorkspaceVersions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	versions, err := h.workspaceVersionService.ListVersions(c.Request.Context(), workspaceID)
	if err != nil {
		slog.Error("failed to list workspace versions", "error", err, "workspace_id", workspaceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// GetWorkspaceVersion gets details of a specific version.
// GET /api/workspaces/:id/versions/:version
func (h *WorkspaceHandler) GetWorkspaceVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	versionNumber := c.Param("version")
	if versionNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version number is required"})
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	type VersionInfo struct {
		ID               string      `json:"id"`
		VersionNumber    string      `json:"version_number"`
		CreatedBy        *string     `json:"created_by"`
		CreatedAt        interface{} `json:"created_at"`
		SnapshotMetadata interface{} `json:"snapshot_metadata"`
	}

	var version VersionInfo
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT id, version_number, created_by, created_at, snapshot_metadata
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, versionNumber).Scan(
		&version.ID,
		&version.VersionNumber,
		&version.CreatedBy,
		&version.CreatedAt,
		&version.SnapshotMetadata,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	c.JSON(http.StatusOK, version)
}

// RestoreWorkspaceVersion restores a workspace to a specific version.
// POST /api/workspaces/:id/restore/:version
func (h *WorkspaceHandler) RestoreWorkspaceVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	versionNumber := c.Param("version")
	if versionNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version number is required"})
		return
	}

	// Verify user has admin permissions
	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	if role != "owner" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only workspace owners and admins can restore versions"})
		return
	}

	// Parse optional dry_run flag
	var req struct {
		DryRun bool `json:"dry_run"`
	}
	c.ShouldBindJSON(&req)

	if req.DryRun {
		// Dry run mode - preview changes without applying them
		preview, err := h.workspaceVersionService.PreviewRestore(
			c.Request.Context(),
			workspaceID,
			versionNumber,
		)
		if err != nil {
			slog.Error("failed to preview restore",
				"error", err,
				"workspace_id", workspaceID,
				"version", versionNumber)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to preview restore"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Dry run: Preview of changes that would be applied",
			"dry_run":      true,
			"preview":      preview,
			"version":      versionNumber,
			"workspace_id": workspaceID,
		})
		return
	}

	// Perform restore
	err = h.workspaceVersionService.RestoreSnapshot(
		c.Request.Context(),
		workspaceID,
		versionNumber,
		user.ID,
	)
	if err != nil {
		slog.Error("failed to restore workspace version",
			"error", err,
			"workspace_id", workspaceID,
			"version", versionNumber)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Workspace restored successfully",
		"version": versionNumber,
	})
}

// CompareWorkspaceVersions compares two workspace versions and returns file-level diffs.
// GET /api/workspaces/:id/versions/compare/:v1/:v2
func (h *WorkspaceHandler) CompareWorkspaceVersions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	fromVersion := c.Param("v1")
	toVersion := c.Param("v2")
	if fromVersion == "" || toVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both version numbers are required"})
		return
	}

	// Verify user is a member
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Optional file filter with path traversal protection
	filterFile := c.Query("file")
	if filterFile != "" {
		// Decode URL-encoded characters to catch bypass attempts (e.g., ..%2F)
		decoded, err := url.QueryUnescape(filterFile)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: malformed encoding"})
			return
		}

		// Reject backslashes (Windows path separator used for traversal)
		if strings.Contains(filterFile, "\\") || strings.Contains(decoded, "\\") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: backslashes not allowed"})
			return
		}

		// Check both raw and decoded forms for path traversal
		if strings.Contains(filterFile, "..") || strings.Contains(decoded, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: path traversal not allowed"})
			return
		}

		// Reject absolute paths (Unix and Windows) in both forms
		if filepath.IsAbs(filterFile) || filepath.IsAbs(decoded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: absolute paths not allowed"})
			return
		}

		// Reject null bytes in both forms
		if strings.Contains(filterFile, "\x00") || strings.Contains(decoded, "\x00") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path: null bytes not allowed"})
			return
		}
	}

	result, err := h.workspaceVersionService.CompareVersions(
		c.Request.Context(),
		workspaceID,
		fromVersion,
		toVersion,
		filterFile,
	)
	if err != nil {
		slog.Error("failed to compare workspace versions",
			"error", err,
			"workspace_id", workspaceID,
			"from", fromVersion,
			"to", toVersion)

		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, gin.H{"error": "One or both versions not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compare versions"})
		return
	}

	c.JSON(http.StatusOK, result)
}
