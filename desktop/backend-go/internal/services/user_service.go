package services

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrUsernameInvalid is returned when the username format is invalid
	ErrUsernameInvalid = errors.New("username format is invalid")
	// ErrUsernameTaken is returned when the username is already taken
	ErrUsernameTaken = errors.New("username is already taken")
	// ErrUsernameReserved is returned when the username is reserved
	ErrUsernameReserved = errors.New("username is reserved")
	// ErrUsernameTooShort is returned when the username is too short
	ErrUsernameTooShort = errors.New("username must be at least 3 characters")
	// ErrUsernameTooLong is returned when the username is too long
	ErrUsernameTooLong = errors.New("username must be 50 characters or less")
	// ErrUsernameInvalidChars is returned when the username contains invalid characters
	ErrUsernameInvalidChars = errors.New("username can only contain letters, numbers, underscores, and hyphens")
	// ErrUsernameInvalidFormat is returned when the username starts or ends with a hyphen
	ErrUsernameInvalidFormat = errors.New("username cannot start or end with a hyphen")
)

// UserService handles username validation and management
type UserService struct {
	pool *pgxpool.Pool
}

// NewUserService creates a new UserService
func NewUserService(pool *pgxpool.Pool) *UserService {
	return &UserService{
		pool: pool,
	}
}

// Username validation regex: alphanumeric, underscore, hyphen
// Must be 3-50 characters
// Cannot start or end with hyphen
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]{1,48}[a-zA-Z0-9])?$`)

// ValidateUsername validates the username format according to the rules:
// - 3-50 characters
// - Only alphanumeric, underscore, hyphen
// - Cannot start or end with hyphen
func (s *UserService) ValidateUsername(username string) error {
	// Check length
	if len(username) < 3 {
		return ErrUsernameTooShort
	}
	if len(username) > 50 {
		return ErrUsernameTooLong
	}

	// Check if starts or ends with hyphen
	if strings.HasPrefix(username, "-") || strings.HasSuffix(username, "-") {
		return ErrUsernameInvalidFormat
	}

	// Check format (alphanumeric, underscore, hyphen)
	if !usernameRegex.MatchString(username) {
		return ErrUsernameInvalidChars
	}

	return nil
}

// IsReservedUsername checks if a username is reserved (case-insensitive)
func (s *UserService) IsReservedUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reserved_usernames
			WHERE LOWER(username) = LOWER($1)
		)
	`, username).Scan(&exists)

	if err != nil {
		slog.Error("Failed to check reserved username",
			slog.String("username", username),
			slog.Any("error", err),
		)
		return false, err
	}

	return exists, nil
}

// CheckUsernameAvailability checks if a username is available
// Returns (available, error)
func (s *UserService) CheckUsernameAvailability(ctx context.Context, username string) (bool, error) {
	// Validate format first
	if err := s.ValidateUsername(username); err != nil {
		return false, err
	}

	// Check if reserved
	reserved, err := s.IsReservedUsername(ctx, username)
	if err != nil {
		return false, err
	}
	if reserved {
		return false, ErrUsernameReserved
	}

	// Check if already taken (case-insensitive)
	var existingID string
	err = s.pool.QueryRow(ctx, `
		SELECT id FROM "user" WHERE LOWER(username) = LOWER($1)
	`, username).Scan(&existingID)

	if err == nil {
		// Username is taken
		return false, ErrUsernameTaken
	}

	if err != pgx.ErrNoRows {
		// Database error
		slog.Error("Failed to check username availability",
			slog.String("username", username),
			slog.Any("error", err),
		)
		return false, err
	}

	// Username is available
	return true, nil
}

// SetUsername sets or updates the username for a user
// Returns the final username and any error
func (s *UserService) SetUsername(ctx context.Context, userID string, username string) (string, error) {
	// Validate format
	if err := s.ValidateUsername(username); err != nil {
		return "", err
	}

	// Check if reserved
	reserved, err := s.IsReservedUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if reserved {
		return "", ErrUsernameReserved
	}

	// Start a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("Failed to start transaction",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		return "", err
	}
	defer tx.Rollback(ctx)

	// Check if user already has a username
	var currentUsername *string
	var currentClaimedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT username, username_claimed_at FROM "user" WHERE id = $1
	`, userID).Scan(&currentUsername, &currentClaimedAt)

	if err != nil {
		slog.Error("Failed to get current username",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		return "", err
	}

	// Check if trying to set the same username (case-insensitive)
	if currentUsername != nil && strings.EqualFold(*currentUsername, username) {
		// Already has this username, just return it
		return *currentUsername, nil
	}

	// Check if username is available (case-insensitive, excluding current user)
	var existingID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM "user" WHERE LOWER(username) = LOWER($1) AND id != $2
	`, username, userID).Scan(&existingID)

	if err == nil {
		// Username is taken by another user
		return "", ErrUsernameTaken
	}

	if err != pgx.ErrNoRows {
		// Database error
		slog.Error("Failed to check username availability",
			slog.String("user_id", userID),
			slog.String("username", username),
			slog.Any("error", err),
		)
		return "", err
	}

	// Set the claimed_at timestamp only if this is the first time claiming
	claimedAt := currentClaimedAt
	if claimedAt == nil {
		now := time.Now()
		claimedAt = &now
	}

	// Update username
	_, err = tx.Exec(ctx, `
		UPDATE "user"
		SET username = $1,
		    username_claimed_at = $2,
		    "updatedAt" = NOW()
		WHERE id = $3
	`, username, claimedAt, userID)

	if err != nil {
		slog.Error("Failed to update username",
			slog.String("user_id", userID),
			slog.String("username", username),
			slog.Any("error", err),
		)
		return "", err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit username update",
			slog.String("user_id", userID),
			slog.String("username", username),
			slog.Any("error", err),
		)
		return "", err
	}

	slog.Info("Username updated successfully",
		slog.String("user_id", userID),
		slog.String("username", username),
		slog.Bool("first_time", currentUsername == nil),
	)

	return username, nil
}

// GetUserByUsername retrieves a user by their username (case-insensitive)
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*UserProfile, error) {
	var user UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, full_name, "createdAt", "updatedAt", username_claimed_at
		FROM "user"
		WHERE LOWER(username) = LOWER($1)
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.UsernameClaimedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}

	if err != nil {
		slog.Error("Failed to get user by username",
			slog.String("username", username),
			slog.Any("error", err),
		)
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*UserProfile, error) {
	var user UserProfile
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, email, full_name, "createdAt", "updatedAt", username_claimed_at
		FROM "user"
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.UsernameClaimedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}

	if err != nil {
		slog.Error("Failed to get user by ID",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return &user, nil
}

// CompleteOnboarding marks the user's onboarding as complete
func (s *UserService) CompleteOnboarding(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE "user"
		SET onboarding_completed = TRUE,
		    "updatedAt" = NOW()
		WHERE id = $1
	`, userID.String())

	if err != nil {
		slog.Error("Failed to complete onboarding",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return err
	}

	slog.Info("Onboarding completed", slog.String("user_id", userID.String()))
	return nil
}

// UserProfile represents a user's public profile information
type UserProfile struct {
	ID                string     `json:"id"`
	Username          *string    `json:"username,omitempty"`
	Email             string     `json:"email"`
	FullName          *string    `json:"full_name,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UsernameClaimedAt *time.Time `json:"username_claimed_at,omitempty"`
}
