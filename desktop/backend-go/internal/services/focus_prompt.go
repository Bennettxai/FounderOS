package services

import (
	"context"
	"fmt"
	"strings"
)

// buildSystemPrompt builds the combined system prompt
func (s *FocusService) buildSystemPrompt(settings *FocusSettings) string {
	var parts []string

	// Add prefix from template
	if settings.SystemPromptPrefix != "" {
		parts = append(parts, settings.SystemPromptPrefix)
	}

	// Add custom user prompt if set
	if settings.CustomSystemPrompt != "" {
		parts = append(parts, settings.CustomSystemPrompt)
	}

	// Add output style instructions
	styleInstructions := s.getOutputStyleInstructions(settings.OutputStyle)
	if styleInstructions != "" {
		parts = append(parts, styleInstructions)
	}

	// Add source requirements
	if settings.RequireSources {
		parts = append(parts, "IMPORTANT: Include sources and citations for your claims. Format sources as [Source Title](URL) where available.")
	}

	// Add suffix from template
	if settings.SystemPromptSuffix != "" {
		parts = append(parts, settings.SystemPromptSuffix)
	}

	return strings.Join(parts, "\n\n")
}

// getOutputStyleInstructions returns instructions for output style
func (s *FocusService) getOutputStyleInstructions(style string) string {
	switch style {
	case "concise":
		return `## Output Style: Concise
- Keep responses brief and to the point
- Use bullet points where appropriate
- Avoid unnecessary elaboration
- Focus on actionable information
- Target 2-4 paragraphs maximum for most responses`

	case "detailed":
		return `## Output Style: Detailed
- Provide comprehensive, thorough responses
- Include relevant context and background
- Explain reasoning and methodology
- Cover edge cases and considerations
- Use examples to illustrate points`

	case "structured":
		return `## Output Style: Structured
- Organize response with clear sections and headers
- Use numbered lists for sequential steps
- Use bullet points for related items
- Include summary at the beginning or end
- Format data in tables where appropriate`

	case "balanced":
		fallthrough
	default:
		return "" // No additional instructions for balanced
	}
}

// trackFocusModeUsage increments usage counter
func (s *FocusService) trackFocusModeUsage(ctx context.Context, userID string, focusMode string) {
	if s.pool == nil {
		return
	}

	_, _ = s.pool.Exec(ctx, `
		UPDATE focus_configurations SET
			use_count = use_count + 1,
			last_used_at = NOW()
		WHERE user_id = $1 AND template_id = (
			SELECT id FROM focus_mode_templates WHERE name = $2
		)
	`, userID, focusMode)
}

// FormatContextForPrompt formats the focus context for injection into the prompt
func (s *FocusService) FormatContextForPrompt(focusCtx *FocusContext) string {
	var parts []string

	// Add KB context
	if len(focusCtx.KBContext) > 0 {
		parts = append(parts, "## Relevant Knowledge Base Context:")
		for _, item := range focusCtx.KBContext {
			parts = append(parts, fmt.Sprintf("### %s\n%s", item.Title, truncateContent(item.Content, 2000)))
		}
	}

	// Add search context with explicit instructions
	if len(focusCtx.SearchContext) > 0 {
		parts = append(parts, `## WEB SEARCH RESULTS
The following are real-time search results. You MUST use these to answer the question:`)
		for i, item := range focusCtx.SearchContext {
			parts = append(parts, fmt.Sprintf("\n### Source %d: %s\n- **URL:** %s\n- **Summary:** %s", i+1, item.Title, item.URL, item.Snippet))
		}
		parts = append(parts, "\n---\nIMPORTANT: Base your answer on the sources above. Do not hallucinate information.")
	}

	// Add project context
	if len(focusCtx.ProjectContext) > 0 {
		parts = append(parts, "## Project Context:")
		for _, item := range focusCtx.ProjectContext {
			parts = append(parts, fmt.Sprintf("**Project: %s** (%s)\n%s", item.Name, item.Status, item.Description))
		}
	}

	return strings.Join(parts, "\n\n")
}

// truncateContent truncates content to a maximum length
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}
