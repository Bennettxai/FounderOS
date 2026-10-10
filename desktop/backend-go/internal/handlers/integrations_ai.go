package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// Model Preferences Endpoints
// ============================================================================

// GetModelPreferences handles GET /api/integrations/ai/preferences
// Returns the user's AI model tier preferences.
func (h *IntegrationsHandler) GetModelPreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	var prefs struct {
		Tier2Model                 interface{}
		Tier3Model                 interface{}
		Tier4Model                 interface{}
		Tier2Fallbacks             interface{}
		Tier3Fallbacks             interface{}
		Tier4Fallbacks             interface{}
		SkillOverrides             interface{}
		AllowModelUpgradeOnFailure bool
		MaxLatencyMs               int
		PreferLocal                bool
	}

	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT tier_2_model, tier_3_model, tier_4_model,
		       tier_2_fallbacks, tier_3_fallbacks, tier_4_fallbacks,
		       skill_overrides, allow_model_upgrade_on_failure,
		       max_latency_ms, prefer_local
		FROM user_model_preferences
		WHERE user_id = $1
	`, userID).Scan(&prefs.Tier2Model, &prefs.Tier3Model, &prefs.Tier4Model,
		&prefs.Tier2Fallbacks, &prefs.Tier3Fallbacks, &prefs.Tier4Fallbacks,
		&prefs.SkillOverrides, &prefs.AllowModelUpgradeOnFailure,
		&prefs.MaxLatencyMs, &prefs.PreferLocal)

	if err != nil {
		// Return defaults if not found
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"preferences": map[string]interface{}{
				"tier_2_model":                   map[string]string{"model_id": "claude-3-5-haiku", "provider": "anthropic"},
				"tier_3_model":                   map[string]string{"model_id": "claude-sonnet-4", "provider": "anthropic"},
				"tier_4_model":                   map[string]string{"model_id": "claude-opus-4", "provider": "anthropic"},
				"tier_2_fallbacks":               []interface{}{},
				"tier_3_fallbacks":               []interface{}{},
				"tier_4_fallbacks":               []interface{}{},
				"skill_overrides":                map[string]interface{}{},
				"allow_model_upgrade_on_failure": true,
				"max_latency_ms":                 30000,
				"prefer_local":                   false,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"preferences": map[string]interface{}{
			"tier_2_model":                   prefs.Tier2Model,
			"tier_3_model":                   prefs.Tier3Model,
			"tier_4_model":                   prefs.Tier4Model,
			"tier_2_fallbacks":               prefs.Tier2Fallbacks,
			"tier_3_fallbacks":               prefs.Tier3Fallbacks,
			"tier_4_fallbacks":               prefs.Tier4Fallbacks,
			"skill_overrides":                prefs.SkillOverrides,
			"allow_model_upgrade_on_failure": prefs.AllowModelUpgradeOnFailure,
			"max_latency_ms":                 prefs.MaxLatencyMs,
			"prefer_local":                   prefs.PreferLocal,
		},
	})
}

// UpdateModelPreferencesRequest represents the request body for updating AI preferences.
type UpdateModelPreferencesRequest struct {
	Tier2Model                 map[string]string      `json:"tier_2_model"`
	Tier3Model                 map[string]string      `json:"tier_3_model"`
	Tier4Model                 map[string]string      `json:"tier_4_model"`
	Tier2Fallbacks             []map[string]string    `json:"tier_2_fallbacks"`
	Tier3Fallbacks             []map[string]string    `json:"tier_3_fallbacks"`
	Tier4Fallbacks             []map[string]string    `json:"tier_4_fallbacks"`
	SkillOverrides             map[string]interface{} `json:"skill_overrides"`
	AllowModelUpgradeOnFailure *bool                  `json:"allow_model_upgrade_on_failure"`
	MaxLatencyMs               *int                   `json:"max_latency_ms"`
	PreferLocal                *bool                  `json:"prefer_local"`
}

// UpdateModelPreferences handles PUT /api/integrations/ai/preferences
// Updates the user's AI model tier preferences.
func (h *IntegrationsHandler) UpdateModelPreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	if userID == "" {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req UpdateModelPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Set defaults
	allowUpgrade := true
	if req.AllowModelUpgradeOnFailure != nil {
		allowUpgrade = *req.AllowModelUpgradeOnFailure
	}
	maxLatency := 30000
	if req.MaxLatencyMs != nil {
		maxLatency = *req.MaxLatencyMs
	}
	preferLocal := false
	if req.PreferLocal != nil {
		preferLocal = *req.PreferLocal
	}

	_, err := h.pool.Exec(c.Request.Context(), `
		INSERT INTO user_model_preferences (
			user_id, tier_2_model, tier_3_model, tier_4_model,
			tier_2_fallbacks, tier_3_fallbacks, tier_4_fallbacks,
			skill_overrides, allow_model_upgrade_on_failure,
			max_latency_ms, prefer_local
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			tier_2_model = EXCLUDED.tier_2_model,
			tier_3_model = EXCLUDED.tier_3_model,
			tier_4_model = EXCLUDED.tier_4_model,
			tier_2_fallbacks = EXCLUDED.tier_2_fallbacks,
			tier_3_fallbacks = EXCLUDED.tier_3_fallbacks,
			tier_4_fallbacks = EXCLUDED.tier_4_fallbacks,
			skill_overrides = EXCLUDED.skill_overrides,
			allow_model_upgrade_on_failure = EXCLUDED.allow_model_upgrade_on_failure,
			max_latency_ms = EXCLUDED.max_latency_ms,
			prefer_local = EXCLUDED.prefer_local,
			updated_at = NOW()
	`, userID, req.Tier2Model, req.Tier3Model, req.Tier4Model,
		req.Tier2Fallbacks, req.Tier3Fallbacks, req.Tier4Fallbacks,
		req.SkillOverrides, allowUpgrade, maxLatency, preferLocal)

	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save preferences", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Preferences saved",
	})
}
