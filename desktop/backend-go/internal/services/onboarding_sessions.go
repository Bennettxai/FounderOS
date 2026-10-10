package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CheckOnboardingStatus checks if user needs onboarding
func (s *OnboardingService) CheckOnboardingStatus(ctx context.Context, userID string) (*OnboardingStatus, error) {
	status := &OnboardingStatus{}

	// Check workspace membership count
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_members
		WHERE user_id = $1 AND status = 'active'
	`, userID).Scan(&status.WorkspaceCount)
	if err != nil {
		return nil, fmt.Errorf("check workspace count: %w", err)
	}

	// If user has workspaces, no onboarding needed
	if status.WorkspaceCount > 0 {
		status.NeedsOnboarding = false
		return status, nil
	}

	// Check for existing in-progress session
	session, err := s.GetResumeableSession(ctx, userID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("check session: %w", err)
	}

	status.NeedsOnboarding = true
	if session != nil {
		status.HasSession = true
		status.Session = session
	}

	return status, nil
}

// CreateSession creates a new onboarding session
func (s *OnboardingService) CreateSession(ctx context.Context, userID string) (*OnboardingSession, error) {
	// First, expire/abandon any existing sessions for this user
	_, err := s.pool.Exec(ctx, `
		UPDATE onboarding_sessions
		SET status = 'abandoned', updated_at = NOW()
		WHERE user_id = $1 AND status = 'in_progress'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("abandon old sessions: %w", err)
	}

	// Create new session
	var session OnboardingSession
	err = s.pool.QueryRow(ctx, `
		INSERT INTO onboarding_sessions (user_id)
		VALUES ($1)
		RETURNING id, user_id, status, current_step, steps_completed, extracted_data,
		          low_confidence_count, fallback_triggered, workspace_id, started_at,
		          completed_at, expires_at, created_at, updated_at
	`, userID).Scan(
		&session.ID, &session.UserID, &session.Status, &session.CurrentStep,
		&session.StepsCompleted, &session.ExtractedData, &session.LowConfidenceCount,
		&session.FallbackTriggered, &session.WorkspaceID, &session.StartedAt,
		&session.CompletedAt, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Add initial agent message
	initialMessage := "Hi! I'm here to help set up your workspace. What's your company called?"
	_, err = s.AddMessage(ctx, session.ID, "agent", initialMessage, nil, onboardingStrPtr("company_name"))
	if err != nil {
		return nil, fmt.Errorf("add initial message: %w", err)
	}

	return &session, nil
}

// GetSession retrieves a session by ID
func (s *OnboardingService) GetSession(ctx context.Context, sessionID uuid.UUID) (*OnboardingSession, error) {
	var session OnboardingSession
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, status, current_step, steps_completed, extracted_data,
		       low_confidence_count, fallback_triggered, workspace_id, started_at,
		       completed_at, expires_at, created_at, updated_at
		FROM onboarding_sessions
		WHERE id = $1
	`, sessionID).Scan(
		&session.ID, &session.UserID, &session.Status, &session.CurrentStep,
		&session.StepsCompleted, &session.ExtractedData, &session.LowConfidenceCount,
		&session.FallbackTriggered, &session.WorkspaceID, &session.StartedAt,
		&session.CompletedAt, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &session, nil
}

// GetResumeableSession gets an existing in-progress session for a user
func (s *OnboardingService) GetResumeableSession(ctx context.Context, userID string) (*OnboardingSession, error) {
	var session OnboardingSession
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, status, current_step, steps_completed, extracted_data,
		       low_confidence_count, fallback_triggered, workspace_id, started_at,
		       completed_at, expires_at, created_at, updated_at
		FROM onboarding_sessions
		WHERE user_id = $1 AND status = 'in_progress' AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(
		&session.ID, &session.UserID, &session.Status, &session.CurrentStep,
		&session.StepsCompleted, &session.ExtractedData, &session.LowConfidenceCount,
		&session.FallbackTriggered, &session.WorkspaceID, &session.StartedAt,
		&session.CompletedAt, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get resumeable session: %w", err)
	}
	return &session, nil
}

// GetSessionWithHistory retrieves a session with its conversation history
func (s *OnboardingService) GetSessionWithHistory(ctx context.Context, sessionID uuid.UUID) (*OnboardingSession, []ConversationMessage, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	messages, err := s.GetConversationHistory(ctx, sessionID, 0) // 0 = all messages
	if err != nil {
		return nil, nil, err
	}

	return session, messages, nil
}

// AbandonSession marks a session as abandoned
func (s *OnboardingService) AbandonSession(ctx context.Context, sessionID uuid.UUID, userID string) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE onboarding_sessions
		SET status = 'abandoned', updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'in_progress'
	`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("abandon session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found or already completed")
	}
	return nil
}
