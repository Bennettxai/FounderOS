package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// UsernameHandler handles username-related endpoints
type UsernameHandler struct {
	pool        *pgxpool.Pool
	userService *services.UserService
}

// NewUsernameHandler creates a new username handler
func NewUsernameHandler(pool *pgxpool.Pool) *UsernameHandler {
	return &UsernameHandler{
		pool:        pool,
		userService: services.NewUserService(pool),
	}
}

// CheckUsernameResponse represents the username availability check response
type CheckUsernameResponse struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// SetUsernameRequest represents the set username request
type SetUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}

// SetUsernameResponse represents the set username response
type SetUsernameResponse struct {
	Success  bool   `json:"success"`
	Username string `json:"username"`
}

// GetCurrentUserResponse represents the current user info response
type GetCurrentUserResponse struct {
	ID                string  `json:"id"`
	Username          *string `json:"username,omitempty"`
	Email             string  `json:"email"`
	FullName          *string `json:"full_name,omitempty"`
	HasUsername       bool    `json:"has_username"`
	UsernameClaimedAt *string `json:"username_claimed_at,omitempty"`
}

// CheckUsernameAvailability checks if a username is available
// GET /api/users/check-username/:username
func (h *UsernameHandler) CheckUsernameAvailability(c *gin.Context) {
	username := c.Param("username")

	ctx := c.Request.Context()
	available, err := h.userService.CheckUsernameAvailability(ctx, username)

	if err != nil {
		// Determine the reason based on the error
		var reason string
		switch {
		case errors.Is(err, services.ErrUsernameTooShort):
			reason = "Username must be at least 3 characters long"
		case errors.Is(err, services.ErrUsernameTooLong):
			reason = "Username must be 50 characters or less"
		case errors.Is(err, services.ErrUsernameInvalidChars):
			reason = "Username can only contain letters, numbers, underscores, and hyphens"
		case errors.Is(err, services.ErrUsernameInvalidFormat):
			reason = "Username cannot start or end with a hyphen"
		case errors.Is(err, services.ErrUsernameReserved):
			reason = "This username is reserved and cannot be used"
		case errors.Is(err, services.ErrUsernameTaken):
			reason = "This username is already taken"
		default:
			// Database or other error
			slog.Error("Failed to check username availability",
				slog.String("username", username),
				slog.Any("error", err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check username availability",
			})
			return
		}

		c.JSON(http.StatusOK, CheckUsernameResponse{
			Available: false,
			Reason:    reason,
		})
		return
	}

	// Username is available
	c.JSON(http.StatusOK, CheckUsernameResponse{
		Available: available,
	})
}

// SetUsername sets or updates the username for the current user
// PATCH /api/users/me/username
func (h *UsernameHandler) SetUsername(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	// Auth guaranteed by middleware - user cannot be nil here

	var req SetUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	finalUsername, err := h.userService.SetUsername(ctx, user.ID, req.Username)

	if err != nil {
		// Determine the status code and error message based on the error
		var statusCode int
		var errorMsg string
		var reason string

		switch {
		case errors.Is(err, services.ErrUsernameTooShort):
			statusCode = http.StatusUnprocessableEntity
			errorMsg = "Invalid username format"
			reason = "Username must be at least 3 characters long"
		case errors.Is(err, services.ErrUsernameTooLong):
			statusCode = http.StatusUnprocessableEntity
			errorMsg = "Invalid username format"
			reason = "Username must be 50 characters or less"
		case errors.Is(err, services.ErrUsernameInvalidChars):
			statusCode = http.StatusUnprocessableEntity
			errorMsg = "Invalid username format"
			reason = "Username can only contain letters, numbers, underscores, and hyphens"
		case errors.Is(err, services.ErrUsernameInvalidFormat):
			statusCode = http.StatusUnprocessableEntity
			errorMsg = "Invalid username format"
			reason = "Username cannot start or end with a hyphen"
		case errors.Is(err, services.ErrUsernameReserved):
			statusCode = http.StatusConflict
			errorMsg = "Username is reserved"
			reason = "This username is reserved and cannot be used"
		case errors.Is(err, services.ErrUsernameTaken):
			statusCode = http.StatusConflict
			errorMsg = "Username is already taken"
			reason = "This username is already in use by another user"
		default:
			// Database or other error
			statusCode = http.StatusInternalServerError
			errorMsg = "Failed to update username"
			slog.Error("Failed to update username",
				slog.String("user_id", user.ID),
				slog.String("username", req.Username),
				slog.Any("error", err),
			)
		}

		response := gin.H{"error": errorMsg}
		if reason != "" {
			response["reason"] = reason
		}
		c.JSON(statusCode, response)
		return
	}

	c.JSON(http.StatusOK, SetUsernameResponse{
		Success:  true,
		Username: finalUsername,
	})
}

// GetCurrentUser returns the current user's profile information
// GET /api/users/me
func (h *UsernameHandler) GetCurrentUser(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	// Auth guaranteed by middleware - user cannot be nil here

	ctx := c.Request.Context()
	profile, err := h.userService.GetUserByID(ctx, user.ID)
	if err != nil {
		slog.Error("Failed to get user profile",
			slog.String("user_id", user.ID),
			slog.Any("error", err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user profile",
		})
		return
	}

	var usernameClaimedAt *string
	if profile.UsernameClaimedAt != nil {
		claimedAtStr := profile.UsernameClaimedAt.Format("2006-01-02T15:04:05Z07:00")
		usernameClaimedAt = &claimedAtStr
	}

	c.JSON(http.StatusOK, GetCurrentUserResponse{
		ID:                profile.ID,
		Username:          profile.Username,
		Email:             profile.Email,
		FullName:          profile.FullName,
		HasUsername:       profile.Username != nil,
		UsernameClaimedAt: usernameClaimedAt,
	})
}

// CompleteOnboarding marks the user's onboarding as complete
func (h *UsernameHandler) CompleteOnboarding(c *gin.Context) {
	// Get authenticated user from middleware
	userInterface, exists := c.Get(middleware.UserContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, ok := userInterface.(*middleware.BetterAuthUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	// Parse user ID as UUID - handle parse error gracefully
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		slog.Error("Invalid user ID format", "error", err, "user_id", user.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Mark onboarding as complete
	err = h.userService.CompleteOnboarding(c.Request.Context(), userID)
	if err != nil {
		slog.Error("Failed to complete onboarding", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete onboarding"})
		return
	}

	// Check if user has a workspace, create one if not
	ctx := c.Request.Context()
	var workspaceCount int
	err = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces WHERE owner_id = $1`, user.ID).Scan(&workspaceCount)
	if err != nil {
		slog.Warn("Failed to check workspace count", "error", err, "user_id", user.ID)
	} else if workspaceCount == 0 {
		// Create a default workspace for the user
		slog.Info("Creating default workspace for user", "user_id", user.ID)

		var workspaceID uuid.UUID
		err = h.pool.QueryRow(ctx, `
			INSERT INTO workspaces (name, slug, owner_id, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			RETURNING id
		`, "My Workspace", "workspace-"+user.ID[:8], user.ID).Scan(&workspaceID)

		if err != nil {
			slog.Error("Failed to create workspace", "error", err, "user_id", user.ID)
		} else {
			// Create default roles
			_, err = h.pool.Exec(ctx, `
				INSERT INTO workspace_roles (workspace_id, name, display_name, is_system, hierarchy_level, permissions, is_default)
				VALUES
					($1, 'owner', 'Owner', true, 0, '{"workspace": {"manage": true}}'::jsonb, false),
					($1, 'member', 'Member', true, 30, '{}'::jsonb, true)
			`, workspaceID)

			if err != nil {
				slog.Warn("Failed to create workspace roles", "error", err)
			}

			// Add user as owner
			_, err = h.pool.Exec(ctx, `
				INSERT INTO workspace_members (workspace_id, user_id, role, status, joined_at)
				VALUES ($1, $2, 'owner', 'active', NOW())
			`, workspaceID, user.ID)

			if err != nil {
				slog.Error("Failed to add user as workspace member", "error", err)
			} else {
				slog.Info("Default workspace created successfully", "workspace_id", workspaceID, "user_id", user.ID)
			}
		}
	}

	slog.Info("User completed onboarding", "user_id", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Onboarding completed successfully",
	})
}
