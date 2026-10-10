package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/rhl/businessos-backend/internal/services"
)

// OSATemplateHandler handles OSA prompt template endpoints.
type OSATemplateHandler struct {
	osaClient        *osa.ResilientClient
	osaPromptBuilder *services.OSAPromptBuilder
}

// NewOSATemplateHandler creates a new OSATemplateHandler.
func NewOSATemplateHandler(osaClient *osa.ResilientClient, osaPromptBuilder *services.OSAPromptBuilder) *OSATemplateHandler {
	return &OSATemplateHandler{osaClient: osaClient, osaPromptBuilder: osaPromptBuilder}
}

// ListOSATemplates returns all available OSA templates
// GET /api/osa/templates
func (h *OSATemplateHandler) ListOSATemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get optional category filter
	category := c.Query("category")

	// Get templates from prompt builder
	if h.osaPromptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Template service not available",
		})
		return
	}

	templates, err := h.osaPromptBuilder.ListAvailableTemplates(ctx, category)
	if err != nil {
		slog.Error("failed to list templates", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list templates",
		})
		return
	}

	// Convert to response format
	type templateResponse struct {
		Name        string                        `json:"name"`
		DisplayName string                        `json:"display_name"`
		Description string                        `json:"description"`
		Category    string                        `json:"category"`
		Version     string                        `json:"version"`
		Tags        []string                      `json:"tags"`
		Variables   []services.VariableDefinition `json:"variables"`
	}

	response := make([]templateResponse, len(templates))
	for i, tpl := range templates {
		response[i] = templateResponse{
			Name:        tpl.Name,
			DisplayName: tpl.DisplayName,
			Description: tpl.Description,
			Category:    tpl.Category,
			Version:     tpl.Version,
			Tags:        tpl.Tags,
			Variables:   tpl.Variables,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": response,
		"count":     len(response),
	})
}

// GetOSATemplate returns details for a specific template
// GET /api/osa/templates/:name
func (h *OSATemplateHandler) GetOSATemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	templateName := c.Param("name")

	if h.osaPromptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Template service not available",
		})
		return
	}

	template, err := h.osaPromptBuilder.GetTemplateInfo(ctx, templateName)
	if err != nil {
		slog.Warn("template not found", "name", templateName, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Template not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":         template.Name,
		"display_name": template.DisplayName,
		"description":  template.Description,
		"category":     template.Category,
		"version":      template.Version,
		"tags":         template.Tags,
		"variables":    template.Variables,
	})
}

// GenerateFromOSATemplate generates an app from a template
// POST /api/osa/templates/:name/generate
func (h *OSATemplateHandler) GenerateFromOSATemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	templateName := c.Param("name")

	// Parse request
	type generateRequest struct {
		Variables   map[string]interface{} `json:"variables" binding:"required"`
		WorkspaceID *uuid.UUID             `json:"workspace_id"`
	}

	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from auth context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	uid := userID.(uuid.UUID)

	// Validate template service availability
	if h.osaPromptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Template service not available",
		})
		return
	}

	// Validate OSA client availability
	if h.osaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OSA client not available",
		})
		return
	}

	// Validate variables against template
	if err := h.osaPromptBuilder.ValidateTemplateVariables(ctx, templateName, req.Variables); err != nil {
		slog.Warn("template variable validation failed",
			"template", templateName,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid template variables",
			"details": err.Error(),
		})
		return
	}

	// Build prompt from template
	promptReq := services.AppGenerationRequest{
		TemplateName: templateName,
		Variables:    req.Variables,
		UserID:       &uid,
		WorkspaceID:  req.WorkspaceID,
	}

	promptResult, err := h.osaPromptBuilder.BuildAppGenerationPrompt(ctx, promptReq)
	if err != nil {
		slog.Error("failed to build prompt from template",
			"template", templateName,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to build prompt from template",
		})
		return
	}

	slog.Info("prompt built from template",
		"template", templateName,
		"render_time_ms", promptResult.RenderTimeMs,
		"prompt_length", len(promptResult.Prompt),
	)

	// Call OSA to generate app with template
	response, err := h.osaClient.GenerateAppFromTemplate(
		ctx,
		templateName,
		req.Variables,
		uid,
		req.WorkspaceID,
	)
	if err != nil {
		slog.Error("OSA generation from template failed",
			"template", templateName,
			"error", err,
		)

		// Log template usage with error
		h.osaPromptBuilder.LogTemplateUsage(
			ctx,
			promptResult,
			uid,
			req.WorkspaceID,
			"failed",
			err.Error(),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate app from template",
			"details": err.Error(),
		})
		return
	}

	// Log successful template usage
	h.osaPromptBuilder.LogTemplateUsage(
		ctx,
		promptResult,
		uid,
		req.WorkspaceID,
		"success",
		"",
	)

	slog.Info("app generation from template started",
		"template", templateName,
		"app_id", response.AppID,
		"status", response.Status,
	)

	c.JSON(http.StatusOK, gin.H{
		"app_id":       response.AppID,
		"workspace_id": response.WorkspaceID,
		"status":       response.Status,
		"message":      response.Message,
		"template":     templateName,
		"created_at":   response.CreatedAt,
	})
}

// PreviewTemplatePrompt previews the rendered prompt without generating
// POST /api/osa/templates/:name/preview
func (h *OSATemplateHandler) PreviewTemplatePrompt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	templateName := c.Param("name")

	// Parse request
	type previewRequest struct {
		Variables   map[string]interface{} `json:"variables" binding:"required"`
		WorkspaceID *uuid.UUID             `json:"workspace_id"`
	}

	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from auth context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	uid := userID.(uuid.UUID)

	// Validate template service availability
	if h.osaPromptBuilder == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Template service not available",
		})
		return
	}

	// Build prompt from template
	promptReq := services.AppGenerationRequest{
		TemplateName: templateName,
		Variables:    req.Variables,
		UserID:       &uid,
		WorkspaceID:  req.WorkspaceID,
	}

	promptResult, err := h.osaPromptBuilder.BuildAppGenerationPrompt(ctx, promptReq)
	if err != nil {
		slog.Error("failed to build prompt preview",
			"template", templateName,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to build prompt preview",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"template":       templateName,
		"prompt":         promptResult.Prompt,
		"variables":      promptResult.Variables,
		"render_time_ms": promptResult.RenderTimeMs,
		"prompt_length":  len(promptResult.Prompt),
	})
}
