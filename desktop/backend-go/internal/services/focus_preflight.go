package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildPreflightContext builds the complete context for a chat request
func (s *FocusService) BuildPreflightContext(
	ctx context.Context,
	userID string,
	focusMode string,
	userMessage string,
	contextIDs []uuid.UUID,
	projectID *uuid.UUID,
) (*FocusContext, error) {
	// Get effective settings
	settings, err := s.GetEffectiveSettings(ctx, userID, focusMode)
	if err != nil {
		return nil, fmt.Errorf("failed to get focus settings: %w", err)
	}

	focusCtx := &FocusContext{
		OutputConstraints: s.buildOutputConstraints(settings),
		LLMOptions:        s.buildLLMOptions(settings),
	}

	// Build system prompt with output constraints
	basePrompt := s.buildSystemPrompt(settings)
	constraintInstructions := s.GetOutputConstraintsInstructions(focusCtx.OutputConstraints)
	if constraintInstructions != "" {
		focusCtx.SystemPrompt = basePrompt + "\n\n" + constraintInstructions
	} else {
		focusCtx.SystemPrompt = basePrompt
	}

	// Auto-load KB items - use intelligent loading if no specific categories configured
	if len(settings.AutoLoadKBCategories) > 0 {
		kbItems, err := s.loadKBItemsByCategories(ctx, userID, settings.AutoLoadKBCategories, settings.KBContextLimit)
		if err == nil && len(kbItems) > 0 {
			focusCtx.KBContext = kbItems
		}
	} else if userMessage != "" && settings.KBContextLimit > 0 {
		// Use intelligent auto-load based on query content and focus mode
		kbItems, err := s.AutoLoadKBContext(ctx, userID, focusMode, userMessage, settings.KBContextLimit)
		if err == nil && len(kbItems) > 0 {
			focusCtx.KBContext = kbItems
		}
	}

	// Perform web search if AutoSearch is enabled and user message is provided
	if settings.AutoSearch && userMessage != "" {
		searchResults, err := s.performWebSearch(ctx, userMessage, settings.SearchDepth)
		if err == nil && len(searchResults) > 0 {
			focusCtx.SearchContext = searchResults
		}
	}

	// Track usage (fire and forget with timeout)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.trackFocusModeUsage(ctx, userID, focusMode)
	}()

	return focusCtx, nil
}

// performWebSearch executes web search based on search depth
func (s *FocusService) performWebSearch(ctx context.Context, query string, searchDepth string) ([]SearchContextItem, error) {
	searchService := NewWebSearchService()

	// Determine max results based on search depth
	maxResults := 5
	switch searchDepth {
	case "quick":
		maxResults = 3
	case "standard":
		maxResults = 5
	case "deep":
		maxResults = 10
	}

	results, err := searchService.Search(ctx, query, maxResults)
	if err != nil {
		return nil, err
	}

	// Convert to SearchContextItem
	var contextItems []SearchContextItem
	for _, r := range results.Results {
		contextItems = append(contextItems, SearchContextItem{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
			Source:  r.Source,
		})
	}

	return contextItems, nil
}

// buildOutputConstraints creates output constraints from settings
func (s *FocusService) buildOutputConstraints(settings *FocusSettings) OutputConstraints {
	constraints := OutputConstraints{
		Style:          settings.OutputStyle,
		Format:         settings.ResponseFormat,
		RequireSources: settings.RequireSources,
	}

	if settings.MaxResponseLength != nil {
		constraints.MaxLength = settings.MaxResponseLength
	} else {
		// Set default max lengths per mode/style
		constraints.MaxLength = s.getDefaultMaxLength(settings.Name, settings.OutputStyle)
	}

	// Auto-artifact for write mode or long content
	if settings.ResponseFormat == "artifact" {
		constraints.RequireArtifact = true
	}

	return constraints
}

// getDefaultMaxLength returns default max length based on mode and style
func (s *FocusService) getDefaultMaxLength(modeName string, style string) *int {
	// Mode-specific defaults (in characters)
	modeDefaults := map[string]int{
		"quick":    2000,  // ~500 words - concise
		"creative": 8000,  // ~2000 words - flexible
		"analyze":  12000, // ~3000 words - detailed analysis
		"write":    20000, // ~5000 words - full documents
		"plan":     10000, // ~2500 words - structured plans
		"code":     16000, // ~4000 words - code + explanations
		"deep":     16000, // ~4000 words - research
		"research": 16000, // ~4000 words - research
		"build":    12000, // ~3000 words - implementation
	}

	// Style overrides
	styleDefaults := map[string]int{
		"concise":    2000,  // Short responses
		"balanced":   6000,  // Medium responses
		"detailed":   12000, // Long responses
		"structured": 10000, // Organized responses
	}

	// Prefer mode-specific default
	if maxLen, ok := modeDefaults[modeName]; ok {
		return &maxLen
	}

	// Fall back to style default
	if maxLen, ok := styleDefaults[style]; ok {
		return &maxLen
	}

	// General default
	defaultLen := 6000
	return &defaultLen
}

// GetOutputConstraintsInstructions returns system prompt instructions for output constraints
func (s *FocusService) GetOutputConstraintsInstructions(constraints OutputConstraints) string {
	var instructions []string

	// Max length guidance
	if constraints.MaxLength != nil {
		maxWords := *constraints.MaxLength / 4 // ~4 chars per word
		switch {
		case maxWords <= 500:
			instructions = append(instructions, "Keep your response brief and focused. Target 2-4 paragraphs maximum.")
		case maxWords <= 1500:
			instructions = append(instructions, "Provide a moderate-length response. Be thorough but avoid unnecessary repetition.")
		case maxWords <= 3000:
			instructions = append(instructions, "You may provide a detailed response. Include relevant context and explanations.")
		default:
			instructions = append(instructions, "You may provide a comprehensive, in-depth response as needed.")
		}
	}

	// Format requirements
	if constraints.RequireArtifact {
		instructions = append(instructions, "For substantial content (documents, code, plans), generate an artifact that can be saved separately.")
	}

	// Source requirements
	if constraints.RequireSources {
		instructions = append(instructions, "CRITICAL: Include sources and citations. End with a '## Sources' section listing all references.")
	}

	if len(instructions) == 0 {
		return ""
	}

	return "## Output Requirements\n" + strings.Join(instructions, "\n")
}

// buildLLMOptions creates LLM options from focus settings
func (s *FocusService) buildLLMOptions(settings *FocusSettings) LLMOptions {
	opts := DefaultLLMOptions()
	opts.Temperature = settings.Temperature
	opts.MaxTokens = settings.MaxTokens
	opts.ThinkingEnabled = settings.ThinkingEnabled
	opts.Model = settings.EffectiveModel // Model override from focus mode

	return opts
}
