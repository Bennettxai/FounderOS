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

// FocusHandler handles focus mode templates and settings.
type FocusHandler struct {
	pool *pgxpool.Pool
}

// NewFocusHandler creates a new FocusHandler.
func NewFocusHandler(pool *pgxpool.Pool) *FocusHandler {
	return &FocusHandler{pool: pool}
}

// RegisterFocusRoutes registers focus mode routes under /api/focus.
func RegisterFocusRoutes(api *gin.RouterGroup, h *FocusHandler, auth gin.HandlerFunc) {
	focus := api.Group("/focus")
	focus.Use(auth, middleware.RequireAuth())
	{
		focus.GET("/templates", h.GetFocusModeTemplates)
		focus.GET("/settings", h.GetEffectiveFocusSettings)
		focus.POST("/preflight", h.BuildPreflightContext)
	}
}

// FocusTemplateResponse represents a focus mode template
type FocusTemplateResponse struct {
	Name            string  `json:"name"`
	DisplayName     string  `json:"display_name"`
	Description     string  `json:"description"`
	Icon            string  `json:"icon"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	OutputStyle     string  `json:"output_style"`
	AutoSearch      bool    `json:"auto_search"`
	ThinkingEnabled bool    `json:"thinking_enabled"`
}

// GetFocusModeTemplates returns all available focus mode templates
func (h *FocusHandler) GetFocusModeTemplates(c *gin.Context) {
	// Return hardcoded templates (from FocusService defaults)
	templates := []FocusTemplateResponse{
		{Name: "quick", DisplayName: "Quick", Description: "Fast, concise responses", Icon: "zap", Temperature: 0.5, MaxTokens: 2048, OutputStyle: "concise", AutoSearch: false, ThinkingEnabled: false},
		{Name: "deep", DisplayName: "Deep Research", Description: "Thorough research with sources", Icon: "search", Temperature: 0.7, MaxTokens: 8192, OutputStyle: "detailed", AutoSearch: true, ThinkingEnabled: true},
		{Name: "creative", DisplayName: "Creative", Description: "Imaginative responses", Icon: "sparkles", Temperature: 0.9, MaxTokens: 4096, OutputStyle: "balanced", AutoSearch: false, ThinkingEnabled: true},
		{Name: "analyze", DisplayName: "Analysis", Description: "Data-driven analysis", Icon: "chart-bar", Temperature: 0.6, MaxTokens: 6144, OutputStyle: "structured", AutoSearch: false, ThinkingEnabled: true},
		{Name: "write", DisplayName: "Writing", Description: "Document creation", Icon: "file-text", Temperature: 0.7, MaxTokens: 8192, OutputStyle: "detailed", AutoSearch: false, ThinkingEnabled: false},
		{Name: "plan", DisplayName: "Planning", Description: "Strategic planning", Icon: "clipboard-list", Temperature: 0.6, MaxTokens: 6144, OutputStyle: "structured", AutoSearch: false, ThinkingEnabled: true},
		{Name: "code", DisplayName: "Coding", Description: "Software development", Icon: "code", Temperature: 0.4, MaxTokens: 8192, OutputStyle: "structured", AutoSearch: false, ThinkingEnabled: true},
	}

	c.JSON(http.StatusOK, templates)
}

// GetEffectiveFocusSettings returns merged settings for a focus mode
func (h *FocusHandler) GetEffectiveFocusSettings(c *gin.Context) {
	user := c.MustGet("user").(*middleware.BetterAuthUser)
	focusMode := c.Query("mode")

	if focusMode == "" {
		utils.RespondInvalidRequest(c, slog.Default(), nil)
		return
	}

	focusService := services.NewFocusService(h.pool)
	settings, err := focusService.GetEffectiveSettings(c.Request.Context(), user.ID, focusMode)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get settings", err)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// BuildPreflightContext builds the preflight context for a focus mode
func (h *FocusHandler) BuildPreflightContext(c *gin.Context) {
	user := c.MustGet("user").(*middleware.BetterAuthUser)

	var req struct {
		FocusMode   string `json:"focus_mode" binding:"required"`
		UserMessage string `json:"user_message"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	focusService := services.NewFocusService(h.pool)
	focusCtx, err := focusService.BuildPreflightContext(
		c.Request.Context(),
		user.ID,
		req.FocusMode,
		req.UserMessage,
		nil,
		nil,
	)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "build context", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"system_prompt": focusCtx.SystemPrompt,
		"llm_options":   focusCtx.LLMOptions,
		"constraints":   focusCtx.OutputConstraints,
	})
}
