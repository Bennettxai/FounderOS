package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PromptPersonalizer enriches system prompts with user-specific context
type PromptPersonalizer struct {
	pool         *pgxpool.Pool
	learningSvc  *LearningService
	memorySvc    *MemoryService
	embeddingSvc *EmbeddingService
	logger       *slog.Logger
}

// NewPromptPersonalizer creates a new prompt personalizer
func NewPromptPersonalizer(
	pool *pgxpool.Pool,
	learningSvc *LearningService,
	memorySvc *MemoryService,
	embeddingSvc *EmbeddingService,
) *PromptPersonalizer {
	return &PromptPersonalizer{
		pool:         pool,
		learningSvc:  learningSvc,
		memorySvc:    memorySvc,
		embeddingSvc: embeddingSvc,
		logger:       slog.Default().With("service", "prompt_personalizer"),
	}
}

// PersonalizationContext holds user-specific personalization data
type PersonalizationContext struct {
	ProfileData     *PersonalizationProfile
	RelevantMemories []Memory
	UserFacts       []UserFact
	BehaviorPatterns []BehaviorPattern
}

// BuildPersonalizedPrompt enriches a base prompt with user personalization
func (p *PromptPersonalizer) BuildPersonalizedPrompt(ctx context.Context, userID string, basePrompt string, userMessage string) (string, error) {
	// Fetch personalization context
	persCtx, err := p.fetchPersonalizationContext(ctx, userID, userMessage)
	if err != nil {
		p.logger.Warn("Failed to fetch personalization context", "err", err)
		return basePrompt, nil // Return base prompt on error (graceful degradation)
	}

	// Build personalization section
	personalSection := p.buildPersonalizationSection(persCtx)

	if personalSection == "" {
		return basePrompt, nil // No personalization data available
	}

	// Inject personalization BEFORE the base prompt
	enrichedPrompt := personalSection + "\n\n---\n\n" + basePrompt

	p.logger.Debug("Personalized prompt built",
		"user_id", userID,
		"personalization_chars", len(personalSection),
		"total_chars", len(enrichedPrompt),
	)

	return enrichedPrompt, nil
}

// fetchPersonalizationContext retrieves all personalization data for a user
func (p *PromptPersonalizer) fetchPersonalizationContext(ctx context.Context, userID string, userMessage string) (*PersonalizationContext, error) {
	persCtx := &PersonalizationContext{}

	// 1. Fetch personalization profile
	if p.learningSvc != nil {
		profile, err := p.learningSvc.GetPersonalizationProfile(ctx, userID)
		if err == nil && profile != nil {
			persCtx.ProfileData = profile
		}
	}

	// 2. Fetch relevant memories (semantic search on user message)
	if p.memorySvc != nil && p.embeddingSvc != nil && userMessage != "" {
		// Generate embedding for semantic search
		embedding, err := p.embeddingSvc.GenerateEmbedding(ctx, userMessage)
		if err != nil {
			// Fallback to recent memories if embedding generation fails
			slog.Warn("Failed to generate embedding for semantic search, falling back to recent memories",
				"error", err,
				"user_id", userID,
			)
			memories, err := p.memorySvc.ListMemories(ctx, userID, nil, 5)
			if err == nil {
				persCtx.RelevantMemories = memories
			}
		} else {
			// Use pgvector semantic similarity search
			memories, err := p.memorySvc.SearchByEmbedding(ctx, userID, embedding, 5)
			if err != nil {
				// Fallback to recent memories if semantic search fails
				slog.Warn("Semantic memory search failed, falling back to recent memories",
					"error", err,
					"user_id", userID,
				)
				memories, err = p.memorySvc.ListMemories(ctx, userID, nil, 5)
				if err == nil {
					persCtx.RelevantMemories = memories
				}
			} else {
				persCtx.RelevantMemories = memories
			}
		}
	}

	// 3. Fetch confirmed user facts
	facts, err := p.fetchUserFacts(ctx, userID)
	if err == nil {
		persCtx.UserFacts = facts
	}

	// 4. Fetch behavior patterns
	patterns, err := p.fetchBehaviorPatterns(ctx, userID)
	if err == nil {
		persCtx.BehaviorPatterns = patterns
	}

	return persCtx, nil
}

// buildPersonalizationSection constructs the personalization prompt section
func (p *PromptPersonalizer) buildPersonalizationSection(persCtx *PersonalizationContext) string {
	if persCtx == nil {
		return ""
	}

	var sections []string

	// Section 1: User Profile
	if persCtx.ProfileData != nil {
		profile := persCtx.ProfileData
		profileSection := "## USER PROFILE\n\n"

		if profile.PreferredTone != "" {
			profileSection += fmt.Sprintf("**Preferred Tone**: %s\n", profile.PreferredTone)
		}
		if profile.PreferredVerbosity != "" {
			profileSection += fmt.Sprintf("**Verbosity**: %s\n", profile.PreferredVerbosity)
		}
		if profile.PreferredFormat != "" {
			profileSection += fmt.Sprintf("**Preferred Format**: %s\n", profile.PreferredFormat)
		}
		if profile.PrefersExamples {
			profileSection += "**Prefers**: Examples and demonstrations\n"
		}
		if profile.PrefersAnalogies {
			profileSection += "**Prefers**: Analogies and metaphors\n"
		}
		if profile.PrefersCodeSamples {
			profileSection += "**Prefers**: Code samples and technical examples\n"
		}
		if len(profile.ExpertiseAreas) > 0 {
			profileSection += fmt.Sprintf("**Expertise Areas**: %s\n", strings.Join(profile.ExpertiseAreas, ", "))
		}
		if len(profile.LearningAreas) > 0 {
			profileSection += fmt.Sprintf("**Learning Areas**: %s\n", strings.Join(profile.LearningAreas, ", "))
		}
		if len(profile.CommonTopics) > 0 {
			profileSection += fmt.Sprintf("**Common Topics**: %s\n", strings.Join(profile.CommonTopics, ", "))
		}

		// Only add section if we have at least one field
		if len(profileSection) > len("## USER PROFILE\n\n") {
			sections = append(sections, profileSection)
		}
	}

	// Section 2: User Facts & Preferences
	if len(persCtx.UserFacts) > 0 {
		factsSection := "## USER FACTS & PREFERENCES\n\n"
		for _, fact := range persCtx.UserFacts {
			if fact.IsActive && fact.ConfidenceScore >= 0.7 { // Only show high confidence facts
				factsSection += fmt.Sprintf("- **%s**: %s\n", fact.FactType, fact.FactValue)
			}
		}
		// Only add section if we have facts
		if len(factsSection) > len("## USER FACTS & PREFERENCES\n\n") {
			sections = append(sections, factsSection)
		}
	}

	// Section 3: Behavior Patterns
	if len(persCtx.BehaviorPatterns) > 0 {
		patternsSection := "## BEHAVIOR PATTERNS\n\n"

		// Group patterns by type
		patternsByType := make(map[string][]BehaviorPattern)
		for _, pattern := range persCtx.BehaviorPatterns {
			patternsByType[pattern.PatternType] = append(patternsByType[pattern.PatternType], pattern)
		}

		for patternType, patterns := range patternsByType {
			patternsSection += fmt.Sprintf("**%s**:\n", strings.ReplaceAll(strings.Title(strings.ReplaceAll(patternType, "_", " ")), " ", " "))
			for _, p := range patterns {
				if p.ObservationCount >= 3 { // Only show patterns that occur at least 3 times
					patternsSection += fmt.Sprintf("- %s (observed: %d times)\n", p.PatternKey, p.ObservationCount)
				}
			}
		}

		sections = append(sections, patternsSection)
	}

	// Section 4: Recent Important Memories
	if len(persCtx.RelevantMemories) > 0 {
		memoriesSection := "## RECENT IMPORTANT CONTEXT\n\n"
		for i, memory := range persCtx.RelevantMemories {
			if i >= 3 { // Limit to top 3 memories to avoid prompt bloat
				break
			}
			memoriesSection += fmt.Sprintf("- **%s** (%s): %s\n", memory.Title, memory.MemoryType, memory.Summary)
		}
		sections = append(sections, memoriesSection)
	}

	if len(sections) == 0 {
		return ""
	}

	// Combine all sections with a header
	result := "# PERSONALIZATION CONTEXT\n\n"
	result += "The following context has been automatically learned from past interactions. Use this to personalize your responses:\n\n"
	result += strings.Join(sections, "\n")

	return result
}

// fetchUserFacts retrieves confirmed user facts
func (p *PromptPersonalizer) fetchUserFacts(ctx context.Context, userID string) ([]UserFact, error) {
	query := `
		SELECT id, fact_key, fact_value, fact_type, confidence_score, is_active, created_at
		FROM user_facts
		WHERE user_id = $1 AND is_active = TRUE AND confidence_score >= 0.7
		ORDER BY confidence_score DESC
		LIMIT 10
	`

	rows, err := p.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user facts: %w", err)
	}
	defer rows.Close()

	var facts []UserFact
	for rows.Next() {
		var fact UserFact
		fact.UserID = userID

		err := rows.Scan(
			&fact.ID,
			&fact.FactKey,
			&fact.FactValue,
			&fact.FactType,
			&fact.ConfidenceScore,
			&fact.IsActive,
			&fact.CreatedAt,
		)
		if err != nil {
			continue
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// fetchBehaviorPatterns retrieves significant behavior patterns
func (p *PromptPersonalizer) fetchBehaviorPatterns(ctx context.Context, userID string) ([]BehaviorPattern, error) {
	query := `
		SELECT id, pattern_type, pattern_key, pattern_value, pattern_description,
		       observation_count, first_observed_at, last_observed_at, confidence_score,
		       is_applied, applied_in_prompt, is_active, created_at
		FROM user_behavior_patterns
		WHERE user_id = $1 AND observation_count >= 3 AND is_active = TRUE
		ORDER BY observation_count DESC, last_observed_at DESC
		LIMIT 10
	`

	rows, err := p.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query behavior patterns: %w", err)
	}
	defer rows.Close()

	var patterns []BehaviorPattern
	for rows.Next() {
		var pattern BehaviorPattern
		pattern.UserID = userID

		err := rows.Scan(
			&pattern.ID,
			&pattern.PatternType,
			&pattern.PatternKey,
			&pattern.PatternValue,
			&pattern.PatternDescription,
			&pattern.ObservationCount,
			&pattern.FirstObservedAt,
			&pattern.LastObservedAt,
			&pattern.ConfidenceScore,
			&pattern.IsApplied,
			&pattern.AppliedInPrompt,
			&pattern.IsActive,
			&pattern.CreatedAt,
		)
		if err != nil {
			continue
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}
