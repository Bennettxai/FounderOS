package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Personalization Profile
// ============================================================================

// GetPersonalizationProfile retrieves or creates a user's personalization profile
func (s *LearningService) GetPersonalizationProfile(ctx context.Context, userID string) (*PersonalizationProfile, error) {
	var profile PersonalizationProfile
	var workingHoursJSON []byte

	// Use nullable wrappers for array types to handle NULL values
	var expertiseAreas, learningAreas, commonTopics []string
	var mostActiveHours []int

	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, preferred_tone, preferred_verbosity, preferred_format,
		       prefers_examples, prefers_analogies, prefers_code_samples, prefers_visual_aids,
		       COALESCE(expertise_areas, '{}'), COALESCE(learning_areas, '{}'), COALESCE(common_topics, '{}'),
		       COALESCE(timezone, ''), preferred_working_hours,
		       COALESCE(most_active_hours, '{}'), total_conversations, total_feedback_given, positive_feedback_ratio,
		       profile_completeness, last_profile_update, created_at, updated_at
		FROM personalization_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.PreferredTone, &profile.PreferredVerbosity,
		&profile.PreferredFormat, &profile.PrefersExamples, &profile.PrefersAnalogies,
		&profile.PrefersCodeSamples, &profile.PrefersVisualAids, &expertiseAreas,
		&learningAreas, &commonTopics, &profile.Timezone, &workingHoursJSON,
		&mostActiveHours, &profile.TotalConversations, &profile.TotalFeedbackGiven,
		&profile.PositiveFeedbackRatio, &profile.ProfileCompleteness, &profile.LastProfileUpdate,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	// Assign arrays after scan
	profile.ExpertiseAreas = expertiseAreas
	profile.LearningAreas = learningAreas
	profile.CommonTopics = commonTopics
	profile.MostActiveHours = mostActiveHours

	if err == pgx.ErrNoRows {
		// Create default profile
		profile = PersonalizationProfile{
			ID:                  uuid.New(),
			UserID:              userID,
			PreferredTone:       "professional",
			PreferredVerbosity:  "balanced",
			PreferredFormat:     "structured",
			PrefersExamples:     true,
			PrefersAnalogies:    false,
			PrefersCodeSamples:  false,
			PrefersVisualAids:   false,
			ProfileCompleteness: 0.1,
			LastProfileUpdate:   time.Now(),
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		_, err = s.pool.Exec(ctx, `
			INSERT INTO personalization_profiles (
				id, user_id, preferred_tone, preferred_verbosity, preferred_format,
				prefers_examples, prefers_analogies, prefers_code_samples, prefers_visual_aids,
				profile_completeness, last_profile_update, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, profile.ID, profile.UserID, profile.PreferredTone, profile.PreferredVerbosity,
			profile.PreferredFormat, profile.PrefersExamples, profile.PrefersAnalogies,
			profile.PrefersCodeSamples, profile.PrefersVisualAids, profile.ProfileCompleteness,
			profile.LastProfileUpdate, profile.CreatedAt, profile.UpdatedAt)

		if err != nil {
			return nil, err
		}

		return &profile, nil
	}

	if err != nil {
		return nil, err
	}

	if workingHoursJSON != nil {
		json.Unmarshal(workingHoursJSON, &profile.PreferredWorkingHours)
	}

	return &profile, nil
}

// UpdatePersonalizationProfile updates or creates a user's profile (UPSERT)
func (s *LearningService) UpdatePersonalizationProfile(ctx context.Context, profile *PersonalizationProfile) error {
	workingHoursJSON, _ := json.Marshal(profile.PreferredWorkingHours)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO personalization_profiles (
			user_id, preferred_tone, preferred_verbosity, preferred_format,
			prefers_examples, prefers_analogies, prefers_code_samples, prefers_visual_aids,
			expertise_areas, learning_areas, common_topics, timezone,
			preferred_working_hours, most_active_hours, profile_completeness,
			last_profile_update, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW(), NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			preferred_tone = EXCLUDED.preferred_tone,
			preferred_verbosity = EXCLUDED.preferred_verbosity,
			preferred_format = EXCLUDED.preferred_format,
			prefers_examples = EXCLUDED.prefers_examples,
			prefers_analogies = EXCLUDED.prefers_analogies,
			prefers_code_samples = EXCLUDED.prefers_code_samples,
			prefers_visual_aids = EXCLUDED.prefers_visual_aids,
			expertise_areas = EXCLUDED.expertise_areas,
			learning_areas = EXCLUDED.learning_areas,
			common_topics = EXCLUDED.common_topics,
			timezone = EXCLUDED.timezone,
			preferred_working_hours = EXCLUDED.preferred_working_hours,
			most_active_hours = EXCLUDED.most_active_hours,
			profile_completeness = EXCLUDED.profile_completeness,
			last_profile_update = NOW(),
			updated_at = NOW()
	`, profile.UserID, profile.PreferredTone, profile.PreferredVerbosity, profile.PreferredFormat,
		profile.PrefersExamples, profile.PrefersAnalogies, profile.PrefersCodeSamples,
		profile.PrefersVisualAids, profile.ExpertiseAreas, profile.LearningAreas,
		profile.CommonTopics, profile.Timezone, workingHoursJSON, profile.MostActiveHours,
		profile.ProfileCompleteness)

	return err
}

// RefreshProfileFromPatterns updates profile based on detected patterns
func (s *LearningService) RefreshProfileFromPatterns(ctx context.Context, userID string) error {
	profile, err := s.GetPersonalizationProfile(ctx, userID)
	if err != nil {
		return err
	}

	// Get high-confidence patterns
	patterns, err := s.DetectPatterns(ctx, userID)
	if err != nil {
		return err
	}

	// Update profile based on patterns
	for _, p := range patterns {
		switch p.PatternType {
		case "communication_style":
			if p.PatternKey == "verbosity" {
				profile.PreferredVerbosity = p.PatternValue
			}
		case "time_preference":
			if p.PatternKey == "active_hours" {
				// Parse hours
				var hours []int
				json.Unmarshal([]byte(p.PatternValue), &hours)
				profile.MostActiveHours = hours
			}
		case "topic_interest":
			if !sliceContains(profile.CommonTopics, p.PatternKey) {
				profile.CommonTopics = append(profile.CommonTopics, p.PatternKey)
			}
		}
	}

	// Calculate profile completeness
	profile.ProfileCompleteness = s.calculateCompleteness(profile)

	return s.UpdatePersonalizationProfile(ctx, profile)
}

// calculateCompleteness calculates how complete a profile is
func (s *LearningService) calculateCompleteness(profile *PersonalizationProfile) float64 {
	var score float64
	total := 10.0

	if profile.PreferredTone != "" {
		score++
	}
	if profile.PreferredVerbosity != "" {
		score++
	}
	if profile.PreferredFormat != "" {
		score++
	}
	if len(profile.ExpertiseAreas) > 0 {
		score++
	}
	if len(profile.CommonTopics) > 0 {
		score++
	}
	if profile.Timezone != "" {
		score++
	}
	if len(profile.MostActiveHours) > 0 {
		score++
	}
	if profile.TotalConversations > 10 {
		score++
	}
	if profile.TotalFeedbackGiven > 5 {
		score++
	}
	if profile.PositiveFeedbackRatio > 0 {
		score++
	}

	return score / total
}

// ============================================================================
// Shared Helper Functions
// ============================================================================

// truncate truncates a string to maxLen characters, appending "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// sliceContains checks if a string slice contains a specific item (case-insensitive)
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
