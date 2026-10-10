package services

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Behavior Pattern Detection
// ============================================================================

// ObserveBehavior records a behavior observation
func (s *LearningService) ObserveBehavior(ctx context.Context, userID, patternType, patternKey, patternValue string) error {
	// Try to update existing pattern
	result, err := s.pool.Exec(ctx, `
		UPDATE user_behavior_patterns
		SET pattern_value = $4,
		    pattern_description = COALESCE(pattern_description, $5),
		    observation_count = observation_count + 1,
		    last_observed_at = NOW(),
		    confidence_score = LEAST(1.0, (observation_count + 1)::float / min_observations_for_confidence),
		    updated_at = NOW()
		WHERE user_id = $1 AND pattern_type = $2 AND pattern_key = $3
	`, userID, patternType, patternKey, patternValue, fmt.Sprintf("Observed %s: %s", patternType, patternKey))

	if err != nil {
		return err
	}

	// If no existing pattern, create new one
	if result.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO user_behavior_patterns (
				id, user_id, pattern_type, pattern_key, pattern_value, pattern_description,
				observation_count, first_observed_at, last_observed_at, confidence_score,
				is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, 1, NOW(), NOW(), 0.33, true, NOW(), NOW())
		`, uuid.New(), userID, patternType, patternKey, patternValue,
			fmt.Sprintf("Observed %s: %s", patternType, patternKey))
	}

	return nil
}

// normalizeUserFactKey normalizes a raw string into a valid user fact key
func normalizeUserFactKey(raw string) string {
	key := strings.TrimSpace(raw)
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "\t", "_")
	key = strings.ReplaceAll(key, "\n", "_")
	key = strings.ReplaceAll(key, ":", "-")
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, "\\", "-")
	if len(key) <= 255 {
		return key
	}
	h := sha1.Sum([]byte(key))
	shortHash := fmt.Sprintf("%x", h)[:10]
	maxPrefix := 255 - 1 - len(shortHash)
	if maxPrefix < 0 {
		maxPrefix = 0
	}
	return key[:maxPrefix] + "-" + shortHash
}

func (s *LearningService) upsertPatternAsUserFact(ctx context.Context, userID string, p BehaviorPattern) error {
	// Store as an inactive-by-default user fact so users can Confirm/Reject via the existing UI.
	factKey := normalizeUserFactKey(fmt.Sprintf("pattern:%s:%s", p.PatternType, p.PatternKey))

	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_facts (user_id, fact_key, fact_value, fact_type, confidence_score, is_active)
		VALUES ($1, $2, $3, 'pattern', $4, false)
		ON CONFLICT (user_id, fact_key)
		DO UPDATE SET
			fact_value = EXCLUDED.fact_value,
			fact_type = EXCLUDED.fact_type,
			confidence_score = EXCLUDED.confidence_score,
			updated_at = NOW()
	`, userID, factKey, p.PatternValue, p.ConfidenceScore)
	return err
}

// DetectPatternsToUserFacts runs pattern detection and persists high-confidence patterns into user_facts.
// Patterns are saved as inactive by default so they can be explicitly confirmed by the user.
func (s *LearningService) DetectPatternsToUserFacts(ctx context.Context, userID string) ([]BehaviorPattern, error) {
	patterns, err := s.DetectPatterns(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, p := range patterns {
		_ = s.upsertPatternAsUserFact(ctx, userID, p) // best-effort
	}

	return patterns, nil
}

// BackfillRecentUsersBehaviorPatterns finds users with recent activity and persists detected patterns into user_facts.
// Returns (usersProcessed, factsUpserted).
func (s *LearningService) BackfillRecentUsersBehaviorPatterns(ctx context.Context, userLimit int) (int, int, error) {
	if userLimit <= 0 {
		userLimit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT user_id
		FROM conversations
		WHERE created_at > NOW() - INTERVAL '30 days'
		ORDER BY user_id
		LIMIT $1
	`, userLimit)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	usersProcessed := 0
	factsUpserted := 0
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		usersProcessed++

		patterns, err := s.DetectPatterns(ctx, uid)
		if err != nil {
			continue
		}
		for _, p := range patterns {
			if err := s.upsertPatternAsUserFact(ctx, uid, p); err == nil {
				factsUpserted++
			}
		}
	}

	return usersProcessed, factsUpserted, nil
}

// DetectPatterns analyzes user behavior to detect patterns
func (s *LearningService) DetectPatterns(ctx context.Context, userID string) ([]BehaviorPattern, error) {
	// Detect time preferences
	s.detectTimePatterns(ctx, userID)

	// Detect topic interests
	s.detectTopicPatterns(ctx, userID)

	// Detect communication preferences
	s.detectCommunicationPatterns(ctx, userID)

	// Return high-confidence patterns
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, pattern_type, pattern_key, pattern_value, pattern_description,
		       observation_count, first_observed_at, last_observed_at, confidence_score,
		       is_applied, applied_in_prompt, is_active, created_at, updated_at
		FROM user_behavior_patterns
		WHERE user_id = $1 AND is_active = true AND confidence_score >= 0.6
		ORDER BY confidence_score DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []BehaviorPattern
	for rows.Next() {
		var p BehaviorPattern
		err := rows.Scan(&p.ID, &p.UserID, &p.PatternType, &p.PatternKey, &p.PatternValue,
			&p.PatternDescription, &p.ObservationCount, &p.FirstObservedAt, &p.LastObservedAt,
			&p.ConfidenceScore, &p.IsApplied, &p.AppliedInPrompt, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			continue
		}
		patterns = append(patterns, p)
	}

	return patterns, nil
}

// detectTimePatterns detects when user is most active
func (s *LearningService) detectTimePatterns(ctx context.Context, userID string) {
	// Analyze conversation start times
	rows, err := s.pool.Query(ctx, `
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM conversations
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '30 days'
		GROUP BY hour
		ORDER BY count DESC
		LIMIT 3
	`, userID)
	if err != nil {
		return
	}
	defer rows.Close()

	var activeHours []int
	for rows.Next() {
		var hour, count int
		if rows.Scan(&hour, &count) == nil && count >= 3 {
			activeHours = append(activeHours, hour)
		}
	}

	if len(activeHours) > 0 {
		s.ObserveBehavior(ctx, userID, "time_preference", "active_hours", fmt.Sprintf("%v", activeHours))
	}
}

// detectTopicPatterns detects common topics
func (s *LearningService) detectTopicPatterns(ctx context.Context, userID string) {
	// This would analyze conversation content for common themes
	// Simplified version using project names
	rows, err := s.pool.Query(ctx, `
		SELECT name, COUNT(*) as count
		FROM projects
		WHERE user_id = $1
		GROUP BY name
		ORDER BY count DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var count int
		if rows.Scan(&name, &count) == nil {
			s.ObserveBehavior(ctx, userID, "topic_interest", name, "project")
		}
	}
}

// detectCommunicationPatterns detects communication style preferences
func (s *LearningService) detectCommunicationPatterns(ctx context.Context, userID string) {
	// Analyze message lengths to determine verbosity preference
	var avgLength float64
	err := s.pool.QueryRow(ctx, `
		SELECT AVG(LENGTH(content)) FROM messages
		WHERE conversation_id IN (SELECT id FROM conversations WHERE user_id = $1)
		AND role = 'user' AND created_at > NOW() - INTERVAL '30 days'
	`, userID).Scan(&avgLength)

	if err == nil && avgLength > 0 {
		var preference string
		if avgLength < 100 {
			preference = "concise"
		} else if avgLength < 300 {
			preference = "balanced"
		} else {
			preference = "detailed"
		}
		s.ObserveBehavior(ctx, userID, "communication_style", "verbosity", preference)
	}
}
