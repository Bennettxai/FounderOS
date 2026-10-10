package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// SettingsHandler handles user and system settings operations.
type SettingsHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewSettingsHandler constructs a SettingsHandler.
func NewSettingsHandler(pool *pgxpool.Pool, cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{pool: pool, cfg: cfg}
}

// RegisterSettingsRoutes wires the settings routes onto api.
func RegisterSettingsRoutes(api *gin.RouterGroup, h *SettingsHandler, auth gin.HandlerFunc) {
	settings := api.Group("/settings")
	settings.Use(auth, middleware.RequireAuth())
	{
		settings.GET("", h.GetSettings)
		settings.PUT("", h.UpdateSettings)
		settings.GET("/system", h.GetSystemSettings)
		settings.GET("/full-state", h.GetFullState)
	}
}

// GetSettings returns settings for the current user.
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)
	settings, err := queries.GetUserSettings(c.Request.Context(), user.ID)
	if err != nil {
		// Return default settings if none exist
		c.JSON(http.StatusOK, gin.H{
			"default_model":       "llama3.2",
			"email_notifications": true,
			"daily_summary":       false,
			"theme":               "system",
			"sidebar_collapsed":   false,
			"share_analytics":     false,
			"custom_settings":     map[string]interface{}{},
			"model_settings": map[string]interface{}{
				"temperature":     0.7,
				"maxTokens":       2048,
				"topP":            0.9,
				"streamResponses": true,
				"showUsageInChat": true,
			},
			"agent_settings": map[string]interface{}{
				"default_agent":        "general",
				"enable_mentions":      true,
				"auto_select_agent":    true,
				"show_agent_reasoning": true,
			},
			"focus_settings": map[string]interface{}{
				"default_mode":         "quick",
				"remember_last_mode":   true,
				"show_mode_selector":   true,
				"auto_switch_by_query": false,
			},
			"ai_access": map[string]interface{}{
				"provider":     h.cfg.GetActiveProvider(),
				"billing_mode": "platform",
				"runtime":      "businessos",
			},
		})
		return
	}

	// Parse custom_settings to extract model_settings for response
	response := gin.H{
		"id":                  settings.ID,
		"user_id":             settings.UserID,
		"default_model":       settings.DefaultModel,
		"email_notifications": settings.EmailNotifications,
		"daily_summary":       settings.DailySummary,
		"theme":               settings.Theme,
		"sidebar_collapsed":   settings.SidebarCollapsed,
		"share_analytics":     settings.ShareAnalytics,
		"created_at":          settings.CreatedAt,
		"updated_at":          settings.UpdatedAt,
	}

	// Parse custom_settings and extract model_settings, agent_settings, focus_settings
	var customSettings map[string]interface{}
	if settings.CustomSettings != nil {
		if err := json.Unmarshal(settings.CustomSettings, &customSettings); err == nil {
			response["custom_settings"] = customSettings
			// Extract model_settings for top-level access
			if modelSettings, ok := customSettings["model_settings"]; ok {
				response["model_settings"] = modelSettings
			}
			// Extract agent_settings for top-level access
			if agentSettings, ok := customSettings["agent_settings"]; ok {
				response["agent_settings"] = agentSettings
			}
			// Extract focus_settings for top-level access
			if focusSettings, ok := customSettings["focus_settings"]; ok {
				response["focus_settings"] = focusSettings
			}
			if aiAccess, ok := customSettings["ai_access"]; ok {
				response["ai_access"] = aiAccess
			}
		}
	}

	// Set default model_settings if not present
	if response["model_settings"] == nil {
		response["model_settings"] = map[string]interface{}{
			"temperature":     0.7,
			"maxTokens":       2048,
			"topP":            0.9,
			"streamResponses": true,
			"showUsageInChat": true,
		}
	}

	// Set default agent_settings if not present
	if response["agent_settings"] == nil {
		response["agent_settings"] = map[string]interface{}{
			"default_agent":        "general",
			"enable_mentions":      true,
			"auto_select_agent":    true,
			"show_agent_reasoning": true,
		}
	}

	// Set default focus_settings if not present
	if response["focus_settings"] == nil {
		response["focus_settings"] = map[string]interface{}{
			"default_mode":         "quick",
			"remember_last_mode":   true,
			"show_mode_selector":   true,
			"auto_switch_by_query": false,
		}
	}
	if response["ai_access"] == nil {
		response["ai_access"] = map[string]interface{}{
			"provider":     h.cfg.GetActiveProvider(),
			"billing_mode": "platform",
			"runtime":      "businessos",
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSettings updates or creates user settings using atomic upsert.
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		DefaultModel       *string                `json:"default_model"`
		EmailNotifications *bool                  `json:"email_notifications"`
		DailySummary       *bool                  `json:"daily_summary"`
		Theme              *string                `json:"theme"`
		SidebarCollapsed   *bool                  `json:"sidebar_collapsed"`
		ShareAnalytics     *bool                  `json:"share_analytics"`
		CustomSettings     map[string]interface{} `json:"custom_settings"`
		ModelSettings      map[string]interface{} `json:"model_settings"`
		AgentSettings      map[string]interface{} `json:"agent_settings"`
		FocusSettings      map[string]interface{} `json:"focus_settings"`
		AIAccess           map[string]interface{} `json:"ai_access"`
		AIProvider         *string                `json:"ai_provider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)
	ctx := c.Request.Context()

	// Try to get existing settings for merging
	existing, existsErr := queries.GetUserSettings(ctx, user.ID)

	// Build custom_settings by merging with existing
	var customSettings map[string]interface{}
	if existsErr == nil && existing.CustomSettings != nil {
		// Parse existing custom_settings
		json.Unmarshal(existing.CustomSettings, &customSettings)
	}
	if customSettings == nil {
		customSettings = make(map[string]interface{})
	}

	// Merge new settings into custom_settings
	if req.ModelSettings != nil {
		customSettings["model_settings"] = req.ModelSettings
	}
	if req.AgentSettings != nil {
		customSettings["agent_settings"] = req.AgentSettings
	}
	if req.FocusSettings != nil {
		customSettings["focus_settings"] = req.FocusSettings
	}
	if req.AIAccess != nil {
		customSettings["ai_access"] = req.AIAccess
	} else if req.AIProvider != nil && *req.AIProvider != "" {
		customSettings["ai_access"] = map[string]interface{}{
			"provider":     *req.AIProvider,
			"billing_mode": "platform",
			"runtime":      "businessos",
		}
	}
	// Also merge any direct custom_settings provided
	if req.CustomSettings != nil {
		for k, v := range req.CustomSettings {
			customSettings[k] = v
		}
	}

	// Serialize custom_settings
	var customSettingsJSON []byte
	if len(customSettings) > 0 {
		if settingsJSON, err := json.Marshal(customSettings); err == nil {
			customSettingsJSON = settingsJSON
		}
	}

	// Build final values with defaults
	defaultModel := "llama3.2"
	emailNotifications := true
	dailySummary := false
	theme := "system"
	sidebarCollapsed := false
	shareAnalytics := false

	// Use existing values if available
	if existsErr == nil {
		if existing.DefaultModel != nil {
			defaultModel = *existing.DefaultModel
		}
		if existing.EmailNotifications != nil {
			emailNotifications = *existing.EmailNotifications
		}
		if existing.DailySummary != nil {
			dailySummary = *existing.DailySummary
		}
		if existing.Theme != nil {
			theme = *existing.Theme
		}
		if existing.SidebarCollapsed != nil {
			sidebarCollapsed = *existing.SidebarCollapsed
		}
		if existing.ShareAnalytics != nil {
			shareAnalytics = *existing.ShareAnalytics
		}
	}

	// Apply request values (override existing/defaults)
	if req.DefaultModel != nil {
		defaultModel = *req.DefaultModel
	}
	if req.EmailNotifications != nil {
		emailNotifications = *req.EmailNotifications
	}
	if req.DailySummary != nil {
		dailySummary = *req.DailySummary
	}
	if req.Theme != nil {
		theme = *req.Theme
	}
	if req.SidebarCollapsed != nil {
		sidebarCollapsed = *req.SidebarCollapsed
	}
	if req.ShareAnalytics != nil {
		shareAnalytics = *req.ShareAnalytics
	}

	// Use atomic upsert to avoid race conditions
	settings, err := queries.UpsertUserSettings(ctx, sqlc.UpsertUserSettingsParams{
		UserID:             user.ID,
		DefaultModel:       &defaultModel,
		EmailNotifications: &emailNotifications,
		DailySummary:       &dailySummary,
		Theme:              &theme,
		SidebarCollapsed:   &sidebarCollapsed,
		ShareAnalytics:     &shareAnalytics,
		CustomSettings:     customSettingsJSON,
	})
	if err != nil {
		slog.Warn("UpsertUserSettings failed", "user_id", user.ID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "save settings", err)
		return
	}

	c.JSON(http.StatusOK, settings)
}

// GetSystemSettings returns system-wide settings.
func (h *SettingsHandler) GetSystemSettings(c *gin.Context) {
	// Get the actual active provider based on configuration and API keys
	activeProvider := h.cfg.GetActiveProvider()
	// Map provider to mode string for frontend compatibility
	ollamaMode := "cloud"
	if activeProvider == "ollama_local" {
		ollamaMode = "local"
	}

	c.JSON(http.StatusOK, gin.H{
		"default_model":   h.cfg.DefaultModel,
		"ollama_mode":     ollamaMode,
		"active_provider": activeProvider,
	})
}

// GetFullState returns complete configuration state for UI synchronization.
// This endpoint provides all settings, preferences, and configurations in a single call.
func (h *SettingsHandler) GetFullState(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Initialize response with defaults
	state := gin.H{
		"version": "1.0",
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}

	// Get user settings
	userSettings, err := queries.GetUserSettings(ctx, user.ID)
	if err == nil {
		var customSettings map[string]interface{}
		if userSettings.CustomSettings != nil {
			json.Unmarshal(userSettings.CustomSettings, &customSettings)
		}

		state["settings"] = gin.H{
			"default_model":       userSettings.DefaultModel,
			"email_notifications": userSettings.EmailNotifications,
			"daily_summary":       userSettings.DailySummary,
			"theme":               userSettings.Theme,
			"sidebar_collapsed":   userSettings.SidebarCollapsed,
			"share_analytics":     userSettings.ShareAnalytics,
		}

		// Extract nested settings
		if modelSettings, ok := customSettings["model_settings"]; ok {
			state["model_settings"] = modelSettings
		} else {
			state["model_settings"] = gin.H{
				"temperature":     0.7,
				"maxTokens":       2048,
				"topP":            0.9,
				"streamResponses": true,
				"showUsageInChat": true,
			}
		}

		if agentSettings, ok := customSettings["agent_settings"]; ok {
			state["agent_settings"] = agentSettings
		} else {
			state["agent_settings"] = gin.H{
				"default_agent":        "general",
				"enable_mentions":      true,
				"auto_select_agent":    true,
				"show_agent_reasoning": true,
			}
		}

		if focusSettings, ok := customSettings["focus_settings"]; ok {
			state["focus_settings"] = focusSettings
		} else {
			state["focus_settings"] = gin.H{
				"default_mode":         "quick",
				"remember_last_mode":   true,
				"show_mode_selector":   true,
				"auto_switch_by_query": false,
			}
		}
	} else {
		// Set all defaults if no settings exist
		state["settings"] = gin.H{
			"default_model":       "llama3.2",
			"email_notifications": true,
			"daily_summary":       false,
			"theme":               "system",
			"sidebar_collapsed":   false,
			"share_analytics":     false,
		}
		state["model_settings"] = gin.H{
			"temperature":     0.7,
			"maxTokens":       2048,
			"topP":            0.9,
			"streamResponses": true,
			"showUsageInChat": true,
		}
		state["agent_settings"] = gin.H{
			"default_agent":        "general",
			"enable_mentions":      true,
			"auto_select_agent":    true,
			"show_agent_reasoning": true,
		}
		state["focus_settings"] = gin.H{
			"default_mode":         "quick",
			"remember_last_mode":   true,
			"show_mode_selector":   true,
			"auto_switch_by_query": false,
		}
	}

	// Get thinking settings
	thinkingSettings, err := queries.GetThinkingSettings(ctx, user.ID)
	if err == nil {
		state["thinking_settings"] = gin.H{
			"enabled":             thinkingSettings.ThinkingEnabled,
			"show_in_ui":          thinkingSettings.ThinkingShowInUi,
			"save_traces":         thinkingSettings.ThinkingSaveTraces,
			"default_template_id": thinkingSettings.ThinkingDefaultTemplateID,
			"max_tokens":          thinkingSettings.ThinkingMaxTokens,
		}
	} else {
		state["thinking_settings"] = gin.H{
			"enabled":     false,
			"show_in_ui":  true,
			"save_traces": true,
			"max_tokens":  4096,
		}
	}

	// Get custom agents (brief list)
	agents, err := queries.ListCustomAgents(ctx, user.ID)
	if err == nil {
		agentList := make([]gin.H, len(agents))
		for i, agent := range agents {
			agentList[i] = gin.H{
				"id":           agent.ID,
				"name":         agent.Name,
				"display_name": agent.DisplayName,
				"category":     agent.Category,
				"is_active":    agent.IsActive,
			}
		}
		state["agents"] = agentList
	} else {
		state["agents"] = []gin.H{}
	}

	// Get focus mode templates
	focusModes := []gin.H{
		{"id": "quick", "name": "Quick", "description": "Fast, concise responses"},
		{"id": "deep", "name": "Deep Research", "description": "Web search + comprehensive analysis"},
		{"id": "creative", "name": "Creative", "description": "Imaginative and innovative responses"},
		{"id": "analyze", "name": "Analysis", "description": "Data-driven insights and structured output"},
		{"id": "write", "name": "Writing", "description": "Polished documents and content"},
		{"id": "plan", "name": "Planning", "description": "Actionable plans and strategies"},
		{"id": "code", "name": "Coding", "description": "Clean, efficient code generation"},
		{"id": "research", "name": "Research", "description": "Thorough investigation with sources"},
		{"id": "build", "name": "Build", "description": "Implementation and construction"},
	}
	state["focus_modes"] = focusModes

	// Get system configuration
	activeProvider := h.cfg.GetActiveProvider()
	ollamaMode := "cloud"
	if activeProvider == "ollama_local" {
		ollamaMode = "local"
	}

	state["system"] = gin.H{
		"active_provider": activeProvider,
		"ollama_mode":     ollamaMode,
		"default_model":   h.cfg.DefaultModel,
		"features": gin.H{
			"web_search_enabled":  true,
			"artifacts_enabled":   true,
			"thinking_available":  true,
			"agents_enabled":      true,
			"focus_modes_enabled": true,
		},
	}

	c.JSON(http.StatusOK, state)
}
