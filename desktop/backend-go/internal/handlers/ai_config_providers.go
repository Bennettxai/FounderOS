package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// GetLLMProviders returns available LLM providers and their configuration status
func (h *AIConfigHandler) GetLLMProviders(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	byokConfigured := map[string]bool{}
	if user != nil {
		vault := services.NewCredentialVaultService(h.pool)
		for _, provider := range services.CloudLLMProviderIDs() {
			ok, _ := vault.CredentialExists(c.Request.Context(), user.ID, services.AICredentialProviderID(provider))
			byokConfigured[provider] = ok
		}
	}

	providers := make([]LLMProvider, 0, len(services.LLMProviderDefinitions()))
	for _, definition := range services.LLMProviderDefinitions() {
		platformConfigured := h.platformProviderConfigured(definition.ID)
		byok := byokConfigured[definition.ID]
		provider := LLMProvider{
			ID:          definition.ID,
			Name:        definition.Name,
			Type:        definition.Type,
			Description: definition.Description,
			Configured:  platformConfigured || byok,
			KeySource:   services.ProviderKeySource(platformConfigured, byok),
			PlatformKey: platformConfigured,
			BYOKKey:     byok,
		}
		if definition.ID == services.LLMProviderOllamaLocal {
			provider.Configured = h.isOllamaAvailable()
			provider.BaseURL = h.cfg.OllamaLocalURL
			provider.KeySource = services.LLMKeySourceLocal
		}
		providers = append(providers, provider)
	}

	// Get user's default model from their settings (if authenticated)
	defaultModel := h.cfg.DefaultModel
	activeProvider := h.cfg.GetActiveProvider()
	if user != nil {
		queries := sqlc.New(h.pool)
		settings, err := queries.GetUserSettings(c.Request.Context(), user.ID)
		if err == nil && settings.DefaultModel != nil && *settings.DefaultModel != "" {
			defaultModel = *settings.DefaultModel
		}
		if err == nil && settings.CustomSettings != nil {
			if provider := aiAccessProvider(settings.CustomSettings); provider != "" {
				activeProvider = provider
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":       providers,
		"active_provider": activeProvider,
		"default_model":   defaultModel,
	})
}

// SaveAPIKey saves a user-scoped BYOK API key in the credential vault.
func (h *AIConfigHandler) SaveAPIKey(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Provider string `json:"provider" binding:"required"`
		APIKey   string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if !services.IsCloudLLMProvider(req.Provider) {
		utils.RespondBadRequest(c, slog.Default(), "Invalid provider")
		return
	}

	vault := services.NewCredentialVaultService(h.pool)
	if _, err := vault.StoreAPIKeyCredential(c.Request.Context(), services.StoreAPIKeyInput{
		UserID:     user.ID,
		ProviderID: services.AICredentialProviderID(req.Provider),
		APIKey:     req.APIKey,
		Metadata: map[string]interface{}{
			"scope":    "llm",
			"provider": req.Provider,
		},
	}); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save API key", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "API key saved",
		"provider":   req.Provider,
		"configured": true,
		"key_source": "byok",
	})
}

// UpdateAIProvider updates the active AI provider
func (h *AIConfigHandler) UpdateAIProvider(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Provider    string `json:"provider" binding:"required"`
		BillingMode string `json:"billing_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	if !services.IsKnownLLMProvider(req.Provider) {
		utils.RespondBadRequest(c, slog.Default(), "Invalid provider")
		return
	}

	billingMode := req.BillingMode
	if billingMode == "" {
		billingMode = "platform"
	}
	if req.Provider == services.LLMProviderOllamaLocal {
		billingMode = "local"
	}
	if billingMode != "platform" && billingMode != "byok" && billingMode != "local" {
		utils.RespondBadRequest(c, slog.Default(), "Invalid billing mode")
		return
	}

	if err := h.upsertAIAccess(c.Request.Context(), user.ID, req.Provider, billingMode); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save AI provider", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Provider updated",
		"provider":     req.Provider,
		"billing_mode": billingMode,
	})
}

func aiAccessProvider(customSettings []byte) string {
	var custom map[string]json.RawMessage
	if err := json.Unmarshal(customSettings, &custom); err != nil {
		return ""
	}
	var access struct {
		Provider string `json:"provider"`
	}
	_ = json.Unmarshal(custom["ai_access"], &access)
	return access.Provider
}

func (h *AIConfigHandler) upsertAIAccess(ctx context.Context, userID, provider, billingMode string) error {
	queries := sqlc.New(h.pool)

	var customSettings map[string]interface{}
	existing, err := queries.GetUserSettings(ctx, userID)
	if err == nil && existing.CustomSettings != nil {
		_ = json.Unmarshal(existing.CustomSettings, &customSettings)
	}
	if customSettings == nil {
		customSettings = map[string]interface{}{}
	}
	customSettings["ai_access"] = map[string]interface{}{
		"provider":     provider,
		"billing_mode": billingMode,
		"runtime":      "businessos",
	}
	customJSON, _ := json.Marshal(customSettings)

	defaultModel := services.DefaultModelForProvider(provider)
	emailNotifications := true
	dailySummary := false
	theme := "system"
	sidebarCollapsed := false
	shareAnalytics := false

	if err == nil {
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

	_, err = queries.UpsertUserSettings(ctx, sqlc.UpsertUserSettingsParams{
		UserID:             userID,
		DefaultModel:       &defaultModel,
		EmailNotifications: &emailNotifications,
		DailySummary:       &dailySummary,
		Theme:              &theme,
		SidebarCollapsed:   &sidebarCollapsed,
		ShareAnalytics:     &shareAnalytics,
		CustomSettings:     customJSON,
	})
	return err
}

func (h *AIConfigHandler) platformProviderConfigured(provider string) bool {
	switch provider {
	case services.LLMProviderOllamaCloud:
		return h.cfg.OllamaCloudAPIKey != ""
	case services.LLMProviderGroq:
		return h.cfg.GroqAPIKey != ""
	case services.LLMProviderAnthropic:
		return h.cfg.AnthropicAPIKey != ""
	default:
		return false
	}
}
