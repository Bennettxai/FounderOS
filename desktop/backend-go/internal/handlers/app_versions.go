package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// APP VERSIONS HANDLERS
// =====================================================================

// ListAppVersions lists all versions for a specific app
// GET /api/apps/:appId/versions
func (h *AppHandler) ListAppVersions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user has access to this app via workspace membership
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// List versions
	versions, err := versionService.ListVersions(c.Request.Context(), appID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list versions", err)
		return
	}

	// Return empty array instead of null if no versions
	if versions == nil {
		versions = []services.AppVersionSnapshot{}
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// GetAppVersion retrieves a specific version
// GET /api/apps/:appId/versions/:versionNumber
func (h *AppHandler) GetAppVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	versionNumber := c.Param("versionNumber")
	if versionNumber == "" {
		utils.RespondBadRequest(c, slog.Default(), "version_number is required")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Get version
	version, err := versionService.GetVersion(c.Request.Context(), appID, versionNumber)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Version")
		return
	}

	c.JSON(http.StatusOK, version)
}

// GetLatestAppVersion retrieves the latest version of an app
// GET /api/apps/:appId/versions/latest
func (h *AppHandler) GetLatestAppVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Get latest version
	version, err := versionService.GetLatestVersion(c.Request.Context(), appID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Version")
		return
	}

	c.JSON(http.StatusOK, version)
}

// CreateAppSnapshot creates a new version snapshot
// POST /api/apps/:appId/versions
func (h *AppHandler) CreateAppSnapshot(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse request body
	var req struct {
		ChangeSummary *string `json:"change_summary"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req.ChangeSummary = nil
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Create snapshot
	snapshot, err := versionService.CreateSnapshot(c.Request.Context(), appID, &user.ID, req.ChangeSummary)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create snapshot", err)
		return
	}

	c.JSON(http.StatusCreated, snapshot)
}

// RestoreAppVersion restores an app to a specific version
// POST /api/apps/:appId/restore/:versionNumber
func (h *AppHandler) RestoreAppVersion(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	versionNumber := c.Param("versionNumber")
	if versionNumber == "" {
		utils.RespondBadRequest(c, slog.Default(), "version_number is required")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Restore version
	if err := versionService.RestoreVersion(c.Request.Context(), appID, versionNumber); err != nil {
		utils.RespondInternalError(c, slog.Default(), "restore version", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "App restored successfully",
		"version_number": versionNumber,
	})
}

// GetAppVersionStats returns statistics about app versions
// GET /api/apps/:appId/versions/stats
func (h *AppHandler) GetAppVersionStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Get stats
	stats, err := versionService.GetVersionStats(c.Request.Context(), appID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get version stats", err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteOldAppVersions deletes old versions, keeping only N most recent
// DELETE /api/apps/:appId/versions/cleanup
func (h *AppHandler) DeleteOldAppVersions(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user has access to this app
	if err := h.verifyAppAccess(c, appID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get keep_count from query param (default to 10)
	keepCountStr := c.DefaultQuery("keep_count", "10")
	keepCount, err := strconv.Atoi(keepCountStr)
	if err != nil || keepCount < 1 {
		utils.RespondBadRequest(c, slog.Default(), "keep_count must be a positive integer")
		return
	}

	// Get app version service
	versionService := services.NewAppVersionService(h.pool, slog.Default())

	// Delete old versions
	if err := versionService.DeleteOldVersions(c.Request.Context(), appID, int32(keepCount)); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete old versions", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Old versions deleted successfully",
		"kept_count": keepCount,
	})
}

// =====================================================================
// HELPER FUNCTIONS
// =====================================================================

// verifyAppAccess checks if user has access to an app via workspace membership
func (h *AppHandler) verifyAppAccess(c *gin.Context, appID uuid.UUID, userID string) error {
	ctx := c.Request.Context()

	// Get the app to find its workspace
	var workspaceID uuid.UUID
	err := h.pool.QueryRow(ctx, `
		SELECT workspace_id FROM user_generated_apps WHERE id = $1
	`, appID).Scan(&workspaceID)
	if err != nil {
		return err
	}

	// Verify user is a member of the workspace
	_, err = h.workspaceService.GetUserRole(ctx, workspaceID, userID)
	return err
}
