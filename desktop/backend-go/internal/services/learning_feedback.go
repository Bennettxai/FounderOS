package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Feedback Processing
// ============================================================================

// RecordFeedback records user feedback and triggers learning
func (s *LearningService) RecordFeedback(ctx context.Context, input FeedbackInput) (*FeedbackEntry, error) {
	feedback := &FeedbackEntry{
		ID:              uuid.New(),
		UserID:          input.UserID,
		TargetType:      input.TargetType,
		TargetID:        input.TargetID,
		FeedbackType:    input.FeedbackType,
		FeedbackValue:   input.FeedbackValue,
		Rating:          input.Rating,
		ConversationID:  input.ConversationID,
		AgentType:       input.AgentType,
		FocusMode:       input.FocusMode,
		OriginalContent: input.OriginalContent,
		ExpectedContent: input.ExpectedContent,
		WasProcessed:    false,
		CreatedAt:       time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO feedback_log (
			id, user_id, target_type, target_id, feedback_type, feedback_value, rating,
			conversation_id, agent_type, focus_mode, original_content, expected_content,
			was_processed, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, feedback.ID, feedback.UserID, feedback.TargetType, feedback.TargetID,
		feedback.FeedbackType, feedback.FeedbackValue, feedback.Rating,
		feedback.ConversationID, feedback.AgentType, feedback.FocusMode,
		feedback.OriginalContent, feedback.ExpectedContent, feedback.WasProcessed, feedback.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to record feedback: %w", err)
	}

	// Process feedback asynchronously
	go s.processFeedback(feedback)

	// Update personalization stats
	go s.updateFeedbackStats(input.UserID, input.FeedbackType)

	return feedback, nil
}

// processFeedback processes feedback to extract learnings
func (s *LearningService) processFeedback(feedback *FeedbackEntry) {
	ctx := context.Background()

	var learningID *uuid.UUID

	switch feedback.FeedbackType {
	case "thumbs_down", "correction":
		// This indicates the AI did something wrong - learn from it
		learning, err := s.createLearningFromCorrection(ctx, feedback)
		if err != nil {
			s.logger.Error("failed to create learning from correction", "error", err)
		} else if learning != nil {
			learningID = &learning.ID
		}

	case "thumbs_up":
		// Positive feedback - reinforce the pattern
		s.reinforcePattern(ctx, feedback)

	case "rating":
		// Rating feedback - update preferences based on rating
		if feedback.Rating != nil && *feedback.Rating >= 4 {
			s.reinforcePattern(ctx, feedback)
		} else if feedback.Rating != nil && *feedback.Rating <= 2 {
			learning, _ := s.createLearningFromCorrection(ctx, feedback)
			if learning != nil {
				learningID = &learning.ID
			}
		}
	}

	// Mark feedback as processed
	now := time.Now()
	s.pool.Exec(ctx, `
		UPDATE feedback_log SET was_processed = true, processed_at = $1, resulting_learning_id = $2 WHERE id = $3
	`, now, learningID, feedback.ID)
}

// createLearningFromCorrection creates a learning event from negative feedback
func (s *LearningService) createLearningFromCorrection(ctx context.Context, feedback *FeedbackEntry) (*LearningEvent, error) {
	// Determine learning type
	learningType := "correction"
	if feedback.FeedbackType == "thumbs_down" {
		learningType = "feedback"
	}

	// Generate summary
	summary := fmt.Sprintf("User indicated dissatisfaction with %s response", feedback.AgentType)
	if feedback.ExpectedContent != "" {
		summary = fmt.Sprintf("User expected: %s", truncate(feedback.ExpectedContent, 200))
	}

	learning := &LearningEvent{
		ID:              uuid.New(),
		UserID:          feedback.UserID,
		LearningType:    learningType,
		LearningContent: feedback.FeedbackValue,
		LearningSummary: summary,
		SourceType:      "explicit_feedback",
		SourceID:        &feedback.ID,
		SourceContext:   fmt.Sprintf("Agent: %s, Focus: %s", feedback.AgentType, feedback.FocusMode),
		ConfidenceScore: 0.8, // High confidence for explicit feedback
		Category:        feedback.AgentType,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO learning_events (
			id, user_id, learning_type, learning_content, learning_summary,
			source_type, source_id, source_context, confidence_score, category,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, learning.ID, learning.UserID, learning.LearningType, learning.LearningContent,
		learning.LearningSummary, learning.SourceType, learning.SourceID, learning.SourceContext,
		learning.ConfidenceScore, learning.Category, learning.IsActive, learning.CreatedAt, learning.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return learning, nil
}

// reinforcePattern reinforces positive patterns
func (s *LearningService) reinforcePattern(ctx context.Context, feedback *FeedbackEntry) {
	// Record as positive learning
	learning := &LearningEvent{
		ID:              uuid.New(),
		UserID:          feedback.UserID,
		LearningType:    "pattern",
		LearningContent: "Positive feedback received",
		LearningSummary: fmt.Sprintf("User approved %s response style", feedback.AgentType),
		SourceType:      "explicit_feedback",
		SourceID:        &feedback.ID,
		SourceContext:   fmt.Sprintf("Agent: %s, Focus: %s", feedback.AgentType, feedback.FocusMode),
		ConfidenceScore: 0.7,
		Category:        feedback.AgentType,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.pool.Exec(ctx, `
		INSERT INTO learning_events (
			id, user_id, learning_type, learning_content, learning_summary,
			source_type, source_id, source_context, confidence_score, category,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, learning.ID, learning.UserID, learning.LearningType, learning.LearningContent,
		learning.LearningSummary, learning.SourceType, learning.SourceID, learning.SourceContext,
		learning.ConfidenceScore, learning.Category, learning.IsActive, learning.CreatedAt, learning.UpdatedAt)
}

// updateFeedbackStats updates personalization profile stats after feedback
func (s *LearningService) updateFeedbackStats(userID, feedbackType string) {
	ctx := context.Background()

	// Update total feedback count
	s.pool.Exec(ctx, `
		UPDATE personalization_profiles
		SET total_feedback_given = total_feedback_given + 1,
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID)

	// Update positive ratio if thumbs_up
	if feedbackType == "thumbs_up" {
		s.pool.Exec(ctx, `
			UPDATE personalization_profiles
			SET positive_feedback_ratio = (
				SELECT COALESCE(
					COUNT(*) FILTER (WHERE feedback_type = 'thumbs_up')::float /
					NULLIF(COUNT(*), 0), 0
				)
				FROM feedback_log WHERE user_id = $1
			),
			updated_at = NOW()
			WHERE user_id = $1
		`, userID)
	}
}

// GetLearningsForContext retrieves relevant learnings for a context
func (s *LearningService) GetLearningsForContext(ctx context.Context, userID, agentType string, limit int) ([]LearningEvent, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, learning_type, learning_content, learning_summary,
		       source_type, source_id, source_context, confidence_score,
		       times_applied, last_applied_at, successful_applications,
		       category, tags, is_active, created_at, updated_at
		FROM learning_events
		WHERE user_id = $1 AND is_active = true
		AND (category = $2 OR category IS NULL)
		ORDER BY confidence_score DESC, created_at DESC
		LIMIT $3
	`, userID, agentType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var learnings []LearningEvent
	for rows.Next() {
		var l LearningEvent
		err := rows.Scan(&l.ID, &l.UserID, &l.LearningType, &l.LearningContent, &l.LearningSummary,
			&l.SourceType, &l.SourceID, &l.SourceContext, &l.ConfidenceScore,
			&l.TimesApplied, &l.LastAppliedAt, &l.SuccessfulApplications,
			&l.Category, &l.Tags, &l.IsActive, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			continue
		}
		learnings = append(learnings, l)
	}

	return learnings, nil
}

// ApplyLearning marks a learning as applied
func (s *LearningService) ApplyLearning(ctx context.Context, learningID uuid.UUID, successful bool) error {
	successIncrement := 0
	if successful {
		successIncrement = 1
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE learning_events
		SET times_applied = times_applied + 1,
		    successful_applications = successful_applications + $1,
		    last_applied_at = NOW(),
		    confidence_score = LEAST(1.0, confidence_score + CASE WHEN $2 THEN 0.05 ELSE -0.05 END),
		    updated_at = NOW()
		WHERE id = $3
	`, successIncrement, successful, learningID)

	return err
}
