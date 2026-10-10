package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// USER APPS HANDLERS
// =====================================================================

// ListUserApps lists all visible user-generated apps for a workspace
// GET /api/workspaces/:id/apps
func (h *AppHandler) ListUserApps(c *gin.Context) {
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

	// Verify user is a member of workspace
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Get user apps service
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())

	// List apps
	apps, err := userAppsService.ListUserApps(c.Request.Context(), workspaceID)
	if err != nil {
		RespondInternalErr(c, "list user apps", err)
		return
	}

	// Return empty array instead of null if no apps
	if apps == nil {
		apps = []services.UserGeneratedApp{}
	}

	c.JSON(http.StatusOK, gin.H{"apps": apps})
}

// GetUserApp gets a single user-generated app
// GET /api/workspaces/:id/apps/:appId
func (h *AppHandler) GetUserApp(c *gin.Context) {
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

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user is a member of workspace
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Get user apps service
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())

	// Get app
	app, err := userAppsService.GetUserApp(c.Request.Context(), appID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "App not found"})
		return
	}

	c.JSON(http.StatusOK, app)
}

// UpdateUserApp updates user app settings (pin, favorite, visibility, etc.)
// PATCH /api/workspaces/:id/apps/:appId
func (h *AppHandler) UpdateUserApp(c *gin.Context) {
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

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user is a member of workspace
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Parse request body
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Get user apps service
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())

	// Update app
	err = userAppsService.UpdateUserApp(c.Request.Context(), appID, workspaceID, updates)
	if err != nil {
		RespondInternalErr(c, "update user app", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App updated successfully"})
}

// IncrementAppAccessCount increments the access count for an app
// POST /api/workspaces/:id/apps/:appId/access
func (h *AppHandler) IncrementAppAccessCount(c *gin.Context) {
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

	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "app_id")
		return
	}

	// Verify user is a member of workspace
	_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Get user apps service
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())

	// Increment access count
	err = userAppsService.IncrementAccessCount(c.Request.Context(), appID, workspaceID)
	if err != nil {
		RespondInternalErr(c, "increment app access count", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Access count incremented"})
}
