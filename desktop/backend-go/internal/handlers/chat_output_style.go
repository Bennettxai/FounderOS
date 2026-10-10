package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// detectStyleFromContext determines the best output style based on focus mode and agent type
func detectStyleFromContext(focusMode string, agentType string) string {
	// Priority 1: Focus Mode
	switch focusMode {
	case "research":
		return "detailed"
	case "analyze":
		return "detailed"
	case "write":
		return "professional"
	case "build":
		return "technical"
	case "general":
		return "conversational"
	}

	// Priority 2: Agent Type
	switch agentType {
	case "analyst":
		return "detailed"
	case "document":
		return "professional"
	case "executive":
		return "executive"
	case "task":
		return "tutorial"
	case "project":
		return "professional"
	}

	return "professional" // Default
}

// getUserStylePreference fetches the user's preferred style for a given context
func (h *ChatHandler) getUserStylePreference(ctx context.Context, userID string, focusMode string, agentType string) string {
	var defaultStyleName sql.NullString
	var overrides []byte

	// Join with output_styles to get the name
	query := `
		SELECT s.name, p.style_overrides
		FROM user_output_preferences p
		LEFT JOIN output_styles s ON p.default_style_id = s.id
		WHERE p.user_id = $1
	`
	err := h.pool.QueryRow(ctx, query, userID).Scan(&defaultStyleName, &overrides)
	if err != nil {
		return "" // No preference found
	}

	// Check overrides first
	if len(overrides) > 0 {
		var mapping map[string]string
		if err := json.Unmarshal(overrides, &mapping); err == nil {
			// Check focus mode override
			if focusMode != "" {
				if styleName, ok := mapping["focus_mode:"+focusMode]; ok {
					return styleName
				}
			}
			// Check agent type override
			if agentType != "" {
				if styleName, ok := mapping["agent:"+agentType]; ok {
					return styleName
				}
			}
		}
	}

	if defaultStyleName.Valid {
		return defaultStyleName.String
	}

	return ""
}

// applyOutputStyle fetches style instructions and prepends them to the system prompt
func (h *ChatHandler) applyOutputStyle(ctx context.Context, styleName string, systemPrompt string) string {
	if styleName == "" {
		return systemPrompt
	}

	// Manual query
	var instructions string
	err := h.pool.QueryRow(ctx, "SELECT style_instructions FROM output_styles WHERE name = $1 AND is_active = TRUE", styleName).Scan(&instructions)
	if err != nil {
		slog.Debug("ChatV2: Output style not found or inactive", "style", styleName)
		return systemPrompt
	}

	// Prepend instructions to system prompt
	styledPrompt := fmt.Sprintf("## OUTPUT STYLE: %s\n\n%s\n\n---\n\n%s", strings.ToUpper(styleName), instructions, systemPrompt)
	slog.Debug("ChatV2: Applied output style", "style", styleName)
	return styledPrompt
}
