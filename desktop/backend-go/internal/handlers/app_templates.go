package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// AppHandler handles app templates, user apps, and app versioning
type AppHandler struct {
	pool             *pgxpool.Pool
	workspaceService *services.WorkspaceService
}

// NewAppHandler creates a new AppHandler
func NewAppHandler(pool *pgxpool.Pool, workspaceService *services.WorkspaceService) *AppHandler {
	return &AppHandler{pool: pool, workspaceService: workspaceService}
}

// =====================================================================
// APP TEMPLATE HANDLERS
// =====================================================================

// ListAppTemplates lists all available app templates with optional filtering
// GET /api/app-templates
func (h *AppHandler) ListAppTemplates(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse optional filters
	filters := services.AppTemplateFilters{}

	if category := c.Query("category"); category != "" {
		filters.Category = &category
	}

	if businessType := c.Query("business_type"); businessType != "" {
		filters.BusinessType = &businessType
	}

	if challenge := c.Query("challenge"); challenge != "" {
		filters.Challenge = &challenge
	}

	if teamSize := c.Query("team_size"); teamSize != "" {
		filters.TeamSize = &teamSize
	}

	// Get app template service
	templateService := services.NewAppTemplateService(h.pool, slog.Default())

	// List templates
	templates, err := templateService.ListTemplates(c.Request.Context(), filters)
	if err != nil {
		slog.Error("failed to list app templates", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list templates"})
		return
	}

	// Return empty array instead of null if no templates
	if templates == nil {
		templates = []services.AppTemplate{}
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// GetAppTemplate gets a single app template by ID
// GET /api/app-templates/:id
func (h *AppHandler) GetAppTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template_id")
		return
	}

	// Get app template service
	templateService := services.NewAppTemplateService(h.pool, slog.Default())

	// Get template
	template, err := templateService.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		slog.Error("failed to get app template", "template_id", templateID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// GetTemplateRecommendations gets personalized template recommendations for a workspace
// GET /api/workspaces/:id/template-recommendations
func (h *AppHandler) GetTemplateRecommendations(c *gin.Context) {
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

	// Parse user ID to UUID
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		slog.Error("invalid user id format", "user_id", user.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse optional limit parameter
	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Get app template service
	templateService := services.NewAppTemplateService(h.pool, slog.Default())

	// Get recommendations
	recommendations, err := templateService.GetRecommendedTemplates(c.Request.Context(), userID, workspaceID, limit)
	if err != nil {
		slog.Error("failed to get template recommendations", "workspace_id", workspaceID, "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	// Return empty array instead of null if no recommendations
	if recommendations == nil {
		recommendations = []services.MatchedTemplate{}
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

// CreateUserAppFromTemplate creates a new user app from a template
// POST /api/workspaces/:id/apps
// Supports both template-based and pure AI generation modes:
// - With template_id: Uses existing template as base
// - Without template_id: Pure AI generation from description (generative mode)
func (h *AppHandler) CreateUserAppFromTemplate(c *gin.Context) {
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

	// Parse request body - template_id is now OPTIONAL for pure generative mode
	var req struct {
		TemplateID  *string                `json:"template_id"` // Optional - omit for pure AI generation
		AppName     string                 `json:"app_name" binding:"required"`
		Description string                 `json:"description"` // Used as AI prompt when no template
		Config      map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// GENERATIVE MODE: No template_id provided - queue for AI generation
	if req.TemplateID == nil || *req.TemplateID == "" {
		slog.Info("queueing pure AI app generation (no template)",
			"workspace_id", workspaceID,
			"app_name", req.AppName,
			"user_id", user.ID,
		)

		// Build generation prompt from name and description
		prompt := req.AppName
		if req.Description != "" {
			prompt = fmt.Sprintf("%s: %s", req.AppName, req.Description)
		}

		// Build generation context
		generationContext := map[string]interface{}{
			"app_name":    req.AppName,
			"description": req.Description,
			"prompt":      prompt,
			"mode":        "generative", // Flag for pure AI generation
		}

		// Merge user-provided config
		if req.Config != nil {
			for k, v := range req.Config {
				generationContext[k] = v
			}
		}

		// Marshal generation context to JSON
		contextJSON, err := json.Marshal(generationContext)
		if err != nil {
			slog.Error("failed to marshal generation context", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare generation context"})
			return
		}

		// Insert into app_generation_queue (without template_id for generative mode)
		var queueItemID uuid.UUID
		err = h.pool.QueryRow(c.Request.Context(), `
			INSERT INTO app_generation_queue (
				workspace_id,
				status,
				priority,
				generation_context,
				max_retries
			) VALUES ($1, 'pending', 5, $2, 3)
			RETURNING id
		`, workspaceID, contextJSON).Scan(&queueItemID)

		if err != nil {
			slog.Error("failed to insert into app_generation_queue", "error", err, "workspace_id", workspaceID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue app generation"})
			return
		}

		slog.Info("app generation queued (generative mode)",
			"queue_item_id", queueItemID,
			"workspace_id", workspaceID,
			"app_name", req.AppName,
		)

		c.JSON(http.StatusCreated, gin.H{
			"message":       "App generation queued. AI will build your app from your description.",
			"queue_item_id": queueItemID,
			"status":        "pending",
			"mode":          "generative",
		})
		return
	}

	// TEMPLATE MODE: template_id provided - use existing template-based flow
	templateID, err := uuid.Parse(*req.TemplateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template_id format"})
		return
	}

	// Validate template exists
	templateService := services.NewAppTemplateService(h.pool, slog.Default())
	template, err := templateService.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		slog.Error("template not found", "template_id", templateID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template not found"})
		return
	}

	slog.Info("creating user app from template", "workspace_id", workspaceID, "template_id", templateID, "app_name", req.AppName)

	// Create user app
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())
	app, err := userAppsService.CreateUserApp(c.Request.Context(), workspaceID, templateID, req.AppName, req.Config)
	if err != nil {
		slog.Error("failed to create user app", "workspace_id", workspaceID, "template_id", templateID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create app"})
		return
	}

	slog.Info("user app created successfully", "app_id", app.ID, "workspace_id", workspaceID, "template", template.DisplayName)

	c.JSON(http.StatusCreated, gin.H{
		"message": "App created successfully",
		"app":     app,
		"mode":    "template",
	})
}

// DeleteUserApp deletes a user-generated app
// DELETE /api/workspaces/:id/apps/:appId
func (h *AppHandler) DeleteUserApp(c *gin.Context) {
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

	// Delete user app
	userAppsService := services.NewUserAppsService(h.pool, slog.Default())
	err = userAppsService.DeleteUserApp(c.Request.Context(), appID, workspaceID)
	if err != nil {
		slog.Error("failed to delete user app", "app_id", appID, "workspace_id", workspaceID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete app"})
		return
	}

	slog.Info("user app deleted", "app_id", appID, "workspace_id", workspaceID)

	c.JSON(http.StatusOK, gin.H{"message": "App deleted successfully"})
}

// GenerateFromTemplate generates an app from a template with user configuration
// POST /api/app-templates/:id/generate
func (h *AppHandler) GenerateFromTemplate(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "template_id")
		return
	}

	// Parse request body
	var req struct {
		WorkspaceID string                 `json:"workspace_id" binding:"required"`
		AppName     string                 `json:"app_name" binding:"required"`
		Config      map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace_id format"})
		return
	}

	// Verify workspace membership
	if h.workspaceService != nil {
		_, err = h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
			return
		}
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		slog.Error("invalid user id format", "user_id", user.ID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Generate app from template
	generationService := services.NewTemplateGenerationService(h.pool, slog.Default())
	result, err := generationService.GenerateFromTemplate(c.Request.Context(), userID, services.GenerateFromTemplateRequest{
		TemplateID:  templateID,
		WorkspaceID: workspaceID,
		AppName:     req.AppName,
		Config:      req.Config,
	})
	if err != nil {
		slog.Error("failed to generate app from template",
			"template_id", templateID,
			"workspace_id", workspaceID,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate app: %v", err)})
		return
	}

	slog.Info("app generated from template",
		"app_id", result.AppID,
		"template_id", templateID,
		"files_count", result.TotalFiles,
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "App generated successfully",
		"result":  result,
	})
}

// GetBuiltInTemplates returns the list of built-in template definitions with their config schemas
// GET /api/app-templates/builtin
func (h *AppHandler) GetBuiltInTemplates(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	templates := services.GetAllBuiltInTemplates()

	type BuiltInTemplateInfo struct {
		ID           string                          `json:"id"`
		Name         string                          `json:"name"`
		Description  string                          `json:"description"`
		Category     string                          `json:"category"`
		StackType    string                          `json:"stack_type"`
		ConfigSchema map[string]services.ConfigField `json:"config_schema"`
		FileCount    int                             `json:"file_count"`
	}

	result := make([]BuiltInTemplateInfo, 0, len(templates))
	for _, t := range templates {
		result = append(result, BuiltInTemplateInfo{
			ID:           t.ID,
			Name:         t.Name,
			Description:  t.Description,
			Category:     t.Category,
			StackType:    t.StackType,
			ConfigSchema: t.ConfigSchema,
			FileCount:    len(t.FilesTemplate),
		})
	}

	c.JSON(http.StatusOK, gin.H{"templates": result})
}
