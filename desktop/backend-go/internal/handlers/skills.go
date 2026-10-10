package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// SkillsHandler handles skill-related API endpoints
type SkillsHandler struct {
	loader       *services.SkillsLoader
	pool         *pgxpool.Pool
	sessionCache *middleware.SessionCache
}

// NewSkillsHandler creates a new skills handler with database pool for authentication
func NewSkillsHandler(loader *services.SkillsLoader, pool *pgxpool.Pool, sessionCache *middleware.SessionCache) *SkillsHandler {
	return &SkillsHandler{
		loader:       loader,
		pool:         pool,
		sessionCache: sessionCache,
	}
}

// ListSkills returns all enabled skills with metadata
// GET /api/skills
func (h *SkillsHandler) ListSkills(c *gin.Context) {
	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	skills := h.loader.GetEnabledSkills()

	response := make([]gin.H, 0, len(skills))
	for _, skill := range skills {
		response = append(response, gin.H{
			"name":        skill.Name,
			"description": skill.Description,
			"version":     skill.Version,
			"priority":    skill.Priority,
			"tools_used":  skill.ToolsUsed,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"skills": response,
		"count":  len(response),
	})
}

// GetSkill returns a specific skill's full content
// GET /api/skills/:name
func (h *SkillsHandler) GetSkill(c *gin.Context) {
	name := c.Param("name")

	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	metadata := h.loader.GetSkillMetadata(name)
	if metadata == nil {
		utils.RespondNotFound(c, slog.Default(), "Skill")
		return
	}

	content, err := h.loader.GetSkillContent(name)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "load skill content", err)
		return
	}

	refs, _ := h.loader.ListSkillReferences(name)

	c.JSON(http.StatusOK, gin.H{
		"name":        metadata.Name,
		"description": metadata.Description,
		"version":     metadata.Version,
		"priority":    metadata.Priority,
		"tools_used":  metadata.ToolsUsed,
		"content":     content,
		"references":  refs,
	})
}

// GetSkillReference returns a specific reference file
// GET /api/skills/:name/references/:ref
func (h *SkillsHandler) GetSkillReference(c *gin.Context) {
	name := c.Param("name")
	ref := c.Param("ref")

	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	content, err := h.loader.GetSkillReference(name, ref)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Reference")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"skill":     name,
		"reference": ref,
		"content":   content,
	})
}

// GetSkillSchema returns the JSON schema for a skill
// GET /api/skills/:name/schemas/:schema
func (h *SkillsHandler) GetSkillSchema(c *gin.Context) {
	name := c.Param("name")
	schema := c.Param("schema")

	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	content, err := h.loader.GetSkillSchema(name, schema)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Schema")
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(content))
}

// ValidateSkill checks a skill for issues
// GET /api/skills/:name/validate
func (h *SkillsHandler) ValidateSkill(c *gin.Context) {
	name := c.Param("name")

	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	issues := h.loader.ValidateSkill(name)

	valid := len(issues) == 0
	c.JSON(http.StatusOK, gin.H{
		"skill":  name,
		"valid":  valid,
		"issues": issues,
	})
}

// ReloadSkills reloads all skills from disk
// POST /api/skills/reload
// SECURITY: This endpoint requires authentication. Consider adding admin-only restriction
// when a global admin role system is implemented.
func (h *SkillsHandler) ReloadSkills(c *gin.Context) {
	// Get authenticated user for audit logging
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Require workspace owner or admin (hierarchy_level <= 2: owner=1, admin=2).
	// Reloading skills affects every agent system-wide and must be restricted
	// to privileged users only.
	if h.pool != nil {
		var hierarchyLevel int
		err := h.pool.QueryRow(c.Request.Context(), `
			SELECT COALESCE(MIN(wr.hierarchy_level), 999)
			FROM workspace_members wm
			JOIN workspace_roles wr ON wr.name = wm.role_name AND wr.workspace_id = wm.workspace_id
			WHERE wm.user_id = $1 AND wm.status = 'active'
		`, user.ID).Scan(&hierarchyLevel)
		if err != nil || hierarchyLevel > 2 {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: admin or owner role required"})
			return
		}
	}

	if h.loader == nil {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	// Log the reload action for audit purposes
	slog.Info("Skills reload initiated",
		"user_id", user.ID,
		"user_email", user.Email,
	)

	if err := h.loader.Reload(); err != nil {
		utils.RespondInternalError(c, slog.Default(), "reload skills", err)
		return
	}

	skills := h.loader.GetEnabledSkills()
	slog.Info("Skills reload completed",
		"user_id", user.ID,
		"skills_count", len(skills),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Skills reloaded",
		"count":   len(skills),
	})
}

// GetSkillsPrompt returns the XML prompt for agent integration
// GET /api/skills/prompt
func (h *SkillsHandler) GetSkillsPrompt(c *gin.Context) {
	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	xml := h.loader.GetSkillsPromptXML()
	instructions := h.loader.GetSkillsPromptInstructions()

	c.JSON(http.StatusOK, gin.H{
		"skills_xml":   xml,
		"instructions": instructions,
	})
}

// GetSkillGroups returns available skill groups
// GET /api/skills/groups
func (h *SkillsHandler) GetSkillGroups(c *gin.Context) {
	if h.loader == nil || !h.loader.IsLoaded() {
		utils.BadRequest(slog.Default(), "Skills system not initialized").Respond(c)
		return
	}

	settings := h.loader.GetSettings()

	groups := make(map[string][]string)
	for _, groupName := range []string{"productivity", "intelligence", "system"} {
		if skills := h.loader.GetSkillGroup(groupName); len(skills) > 0 {
			groups[groupName] = skills
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"groups":   groups,
		"settings": settings,
	})
}

// RegisterRoutes registers skill routes on the given router group with authentication.
// All skills endpoints require authentication to prevent unauthorized access to:
// - Skill definitions and prompts
// - Skill content and references
// - Skill reload operations (potentially sensitive)
//
// SECURITY FIX: Previously these routes had no authentication, exposing:
// - Internal skill definitions and prompts
// - Configuration details
// - Ability to reload skills without authorization
func (h *SkillsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Create auth middleware - use Redis cache if available for better performance
	var auth gin.HandlerFunc
	if h.sessionCache != nil {
		auth = middleware.CachedAuthMiddleware(h.pool, h.sessionCache)
	} else {
		auth = middleware.AuthMiddleware(h.pool)
	}

	skills := rg.Group("/skills")
	skills.Use(auth, middleware.RequireAuth())
	{
		skills.GET("", h.ListSkills)
		skills.GET("/prompt", h.GetSkillsPrompt)
		skills.GET("/groups", h.GetSkillGroups)
		skills.POST("/reload", h.ReloadSkills) // TODO: Consider admin-only restriction
		skills.GET("/:name", h.GetSkill)
		skills.GET("/:name/validate", h.ValidateSkill)
		skills.GET("/:name/references/:ref", h.GetSkillReference)
		skills.GET("/:name/schemas/:schema", h.GetSkillSchema)
	}
}
