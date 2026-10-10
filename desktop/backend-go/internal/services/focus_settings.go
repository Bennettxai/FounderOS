package services

import (
	"context"
	"fmt"
)

// GetEffectiveSettings retrieves merged focus settings for a user and mode
func (s *FocusService) GetEffectiveSettings(ctx context.Context, userID string, focusMode string) (*FocusSettings, error) {
	// First try to get from database
	settings, err := s.getSettingsFromDB(ctx, userID, focusMode)
	if err == nil && settings != nil {
		return settings, nil
	}

	// Fall back to hardcoded defaults
	return s.getDefaultSettings(focusMode), nil
}

// getSettingsFromDB attempts to load settings from database
func (s *FocusService) getSettingsFromDB(ctx context.Context, userID string, focusMode string) (*FocusSettings, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database pool")
	}

	// Query for focus mode template and user override
	row := s.pool.QueryRow(ctx, `
		SELECT
			fmt.name,
			fmt.display_name,
			COALESCE(fc.preferred_model, fmt.default_model) as effective_model,
			COALESCE(fc.temperature, fmt.temperature, 0.7) as temperature,
			COALESCE(fc.max_tokens, fmt.max_tokens, 4096) as max_tokens,
			COALESCE(fc.output_style, fmt.output_style, 'balanced') as output_style,
			COALESCE(fc.response_format, fmt.response_format, 'markdown') as response_format,
			COALESCE(fc.require_sources, fmt.require_sources, false) as require_sources,
			COALESCE(fc.auto_search, fmt.auto_search, false) as auto_search,
			COALESCE(fc.search_depth, fmt.search_depth, 'quick') as search_depth,
			COALESCE(fc.kb_context_limit, fmt.kb_context_limit, 5) as kb_context_limit,
			COALESCE(fc.thinking_enabled, fmt.thinking_enabled, false) as thinking_enabled,
			COALESCE(fc.thinking_style, fmt.thinking_style) as thinking_style,
			COALESCE(fc.custom_system_prompt, '') as custom_system_prompt,
			fmt.system_prompt_prefix,
			fmt.system_prompt_suffix,
			fc.auto_load_kb_categories
		FROM focus_mode_templates fmt
		LEFT JOIN focus_configurations fc ON fc.template_id = fmt.id AND fc.user_id = $1
		WHERE fmt.name = $2 AND fmt.is_active = true
	`, userID, focusMode)

	var settings FocusSettings
	var effectiveModel *string
	var thinkingStyle *string
	var systemPromptPrefix, systemPromptSuffix *string
	var autoLoadKBCategories []string

	err := row.Scan(
		&settings.Name,
		&settings.DisplayName,
		&effectiveModel,
		&settings.Temperature,
		&settings.MaxTokens,
		&settings.OutputStyle,
		&settings.ResponseFormat,
		&settings.RequireSources,
		&settings.AutoSearch,
		&settings.SearchDepth,
		&settings.KBContextLimit,
		&settings.ThinkingEnabled,
		&thinkingStyle,
		&settings.CustomSystemPrompt,
		&systemPromptPrefix,
		&systemPromptSuffix,
		&autoLoadKBCategories,
	)
	if err != nil {
		return nil, err
	}

	settings.EffectiveModel = effectiveModel
	settings.ThinkingStyle = thinkingStyle
	settings.AutoLoadKBCategories = autoLoadKBCategories
	if systemPromptPrefix != nil {
		settings.SystemPromptPrefix = *systemPromptPrefix
	}
	if systemPromptSuffix != nil {
		settings.SystemPromptSuffix = *systemPromptSuffix
	}

	// Merge with hardcoded defaults to ensure correct values
	// (DB may have NULL or outdated values)
	if defaults, ok := focusModeDefaults[focusMode]; ok {
		// Use hardcoded MaxTokens if DB returned default fallback (4096)
		if settings.MaxTokens == 4096 && defaults.MaxTokens > 4096 {
			settings.MaxTokens = defaults.MaxTokens
		}
		// Use hardcoded SystemPromptPrefix if DB returned empty
		if settings.SystemPromptPrefix == "" && defaults.SystemPromptPrefix != "" {
			settings.SystemPromptPrefix = defaults.SystemPromptPrefix
		}
		// Ensure AutoSearch and RequireSources from defaults
		if defaults.AutoSearch && !settings.AutoSearch {
			settings.AutoSearch = defaults.AutoSearch
		}
		if defaults.RequireSources && !settings.RequireSources {
			settings.RequireSources = defaults.RequireSources
		}
		if defaults.ThinkingEnabled && !settings.ThinkingEnabled {
			settings.ThinkingEnabled = defaults.ThinkingEnabled
		}
	}

	return &settings, nil
}

// getDefaultSettings returns hardcoded defaults for a focus mode
func (s *FocusService) getDefaultSettings(focusMode string) *FocusSettings {
	if settings, ok := focusModeDefaults[focusMode]; ok {
		// Return a copy to prevent mutation
		copy := *settings
		return &copy
	}

	// Return general defaults for unknown modes
	return &FocusSettings{
		Name:           focusMode,
		DisplayName:    focusMode,
		Temperature:    0.7,
		MaxTokens:      4096,
		OutputStyle:    "balanced",
		ResponseFormat: "markdown",
	}
}
