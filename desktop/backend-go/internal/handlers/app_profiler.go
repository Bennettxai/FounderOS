package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// AppProfilerHandler handles application profiling operations
type AppProfilerHandler struct {
	profiler *services.AppProfilerService
}

// NewAppProfilerHandler creates a new app profiler handler
func NewAppProfilerHandler(profiler *services.AppProfilerService) *AppProfilerHandler {
	return &AppProfilerHandler{
		profiler: profiler,
	}
}

// requireAuthUser extracts the authenticated user or returns 401
func (h *AppProfilerHandler) requireAuthUser(c *gin.Context) (string, bool) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return "", false
	}
	return user.ID, true
}

// ProfileApplication profiles a codebase
func (h *AppProfilerHandler) ProfileApplication(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	var req struct {
		Name     string                   `json:"name"`
		RootPath string                   `json:"root_path"`
		Options  *services.ProfileOptions `json:"options,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if req.Name == "" {
		utils.RespondBadRequest(c, slog.Default(), "Name is required")
		return
	}

	if req.RootPath == "" {
		utils.RespondBadRequest(c, slog.Default(), "Root path is required")
		return
	}

	profile, err := h.profiler.ProfileApplication(ctx, userID, req.RootPath, req.Name, req.Options)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "profile application", err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// GetProfile retrieves an application profile
func (h *AppProfilerHandler) GetProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ListProfiles lists all profiles for a user
func (h *AppProfilerHandler) ListProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	profiles, err := h.profiler.ListProfiles(ctx, userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list profiles", err)
		return
	}

	c.JSON(http.StatusOK, profiles)
}

// RefreshProfile re-profiles an application
func (h *AppProfilerHandler) RefreshProfile(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	// Get existing profile to get the root path
	existing, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	// Re-profile
	profile, err := h.profiler.ProfileApplication(ctx, userID, existing.RootPath, name, nil)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "refresh profile", err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

// GetProfileComponents retrieves components for a profile
func (h *AppProfilerHandler) GetProfileComponents(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, profile.Components)
}

// GetProfileEndpoints retrieves API endpoints for a profile
func (h *AppProfilerHandler) GetProfileEndpoints(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, profile.APIEndpoints)
}

// GetProfileStructure retrieves directory structure for a profile
func (h *AppProfilerHandler) GetProfileStructure(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, profile.StructureTree)
}

// GetProfileModules retrieves modules for a profile
func (h *AppProfilerHandler) GetProfileModules(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, profile.Modules)
}

// GetProfileTechStack retrieves tech stack for a profile
func (h *AppProfilerHandler) GetProfileTechStack(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := h.requireAuthUser(c)
	if !ok {
		return
	}

	name := c.Param("name")

	profile, err := h.profiler.GetProfile(ctx, userID, name)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Profile")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tech_stack":  profile.TechStack,
		"languages":   profile.Languages,
		"frameworks":  profile.Frameworks,
		"conventions": profile.Conventions,
	})
}

// RegisterAppProfilerRoutes registers app profiler routes on a Gin router group
func RegisterAppProfilerRoutes(r *gin.RouterGroup, handler *AppProfilerHandler) {
	profiles := r.Group("/app-profiles")
	{
		profiles.POST("", handler.ProfileApplication)
		profiles.GET("", handler.ListProfiles)
		profiles.GET("/:name", handler.GetProfile)
		profiles.POST("/:name/refresh", handler.RefreshProfile)
		profiles.GET("/:name/components", handler.GetProfileComponents)
		profiles.GET("/:name/endpoints", handler.GetProfileEndpoints)
		profiles.GET("/:name/structure", handler.GetProfileStructure)
		profiles.GET("/:name/modules", handler.GetProfileModules)
		profiles.GET("/:name/tech-stack", handler.GetProfileTechStack)
	}
}
