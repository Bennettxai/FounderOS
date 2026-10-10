package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/services/templates"
	"github.com/rhl/businessos-backend/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// Account lockout constants (OWASP A07: Authentication Failures)
const (
	loginMaxAttempts  = 5                // lock after this many consecutive failures
	loginLockDuration = 15 * time.Minute // duration of the lockout window
)

// loginAttemptRecord tracks consecutive failed login attempts per email address.
type loginAttemptRecord struct {
	mu          sync.Mutex
	count       int
	lockedUntil time.Time
}

// loginAttempts is a process-wide in-memory store keyed by normalised email.
// Follows the same sync.Map pattern used by middleware/rate_limit.go.
var loginAttempts sync.Map // map[string]*loginAttemptRecord

// normaliseEmail lowercases and trims the email for consistent lockout map keys.
func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// getLoginRecord returns the existing record for the email, creating one if absent.
func getLoginRecord(email string) *loginAttemptRecord {
	val, _ := loginAttempts.LoadOrStore(normaliseEmail(email), &loginAttemptRecord{})
	return val.(*loginAttemptRecord)
}

// isLockedOut returns true when the email is currently locked.
func isLockedOut(email string) bool {
	rec := getLoginRecord(email)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return time.Now().Before(rec.lockedUntil)
}

// recordFailedAttempt increments the failure counter and locks the account
// when the threshold is exceeded. Returns true if the account is now locked.
func recordFailedAttempt(email string) bool {
	rec := getLoginRecord(email)
	rec.mu.Lock()
	defer rec.mu.Unlock()

	// Reset counter if a previous lockout has already expired.
	var zero time.Time
	if rec.lockedUntil != zero && time.Now().After(rec.lockedUntil) {
		rec.count = 0
		rec.lockedUntil = zero
	}

	rec.count++
	if rec.count >= loginMaxAttempts {
		rec.lockedUntil = time.Now().Add(loginLockDuration)
		return true
	}
	return false
}

// resetLoginAttempts clears the failure counter for an email after successful login.
func resetLoginAttempts(email string) {
	loginAttempts.Delete(normaliseEmail(email))
}

// validatePassword checks password strength per OWASP A07.
// Requires: 8-128 chars, at least one uppercase, one lowercase, one digit, one special.
func validatePassword(password string) string {
	runeLen := utf8.RuneCountInString(password)
	if runeLen < 8 {
		return "Password must be at least 8 characters"
	}
	if runeLen > 128 {
		return "Password must not exceed 128 characters"
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasUpper {
		return "Password must contain at least one uppercase letter"
	}
	if !hasLower {
		return "Password must contain at least one lowercase letter"
	}
	if !hasDigit {
		return "Password must contain at least one digit"
	}
	if !hasSpecial {
		return "Password must contain at least one special character"
	}
	return ""
}

// EmailAuthHandler handles email/password authentication
type EmailAuthHandler struct {
	pool                 *pgxpool.Pool
	cfg                  *config.Config
	notificationTriggers *services.NotificationTriggers
	workspaceInit        *services.OSAWorkspaceInitService
	logger               *slog.Logger
}

// SignUpRequest represents the signup request body
type SignUpRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// SignInRequest represents the signin request body
type SignInRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// NewEmailAuthHandler creates a new Email Auth handler
func NewEmailAuthHandler(pool *pgxpool.Pool, cfg *config.Config, notifTriggers *services.NotificationTriggers, workspaceInit *services.OSAWorkspaceInitService, logger *slog.Logger) *EmailAuthHandler {
	return &EmailAuthHandler{
		pool:                 pool,
		cfg:                  cfg,
		notificationTriggers: notifTriggers,
		workspaceInit:        workspaceInit,
		logger:               logger,
	}
}

// SignUp handles user registration with email/password
func (h *EmailAuthHandler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Enforce password strength (OWASP A07: Authentication Failures).
	if msg := validatePassword(req.Password); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Check if user already exists
	var existingID string
	err := h.pool.QueryRow(ctx, `SELECT id FROM "user" WHERE email = $1`, req.Email).Scan(&existingID)
	if err == nil {
		utils.RespondConflict(c, slog.Default(), "User with this email already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "process password", err)
		return
	}

	// Begin transaction for atomic user creation
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "begin transaction", err)
		return
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Create user
	userID := generateUserID()
	_, err = tx.Exec(ctx, `
		INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, false, NOW(), NOW())
	`, userID, req.Name, req.Email)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create user", err)
		return
	}

	// Store password in account table (Better Auth compatible)
	accountID := generateUserID()
	_, err = tx.Exec(ctx, `
		INSERT INTO account (id, "userId", "accountId", "providerId", password, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, 'credential', $4, NOW(), NOW())
	`, accountID, userID, userID, string(hashedPassword))
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create account", err)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		utils.RespondInternalError(c, slog.Default(), "commit transaction", err)
		return
	}

	// Provision synchronously so the first authenticated request always has a
	// valid workspace, canonical roles, and a linked owner membership.
	wsSvc := services.NewWorkspaceService(h.pool)
	if _, wsErr := wsSvc.EnsureDefaultWorkspace(ctx, userID); wsErr != nil {
		utils.RespondInternalError(c, slog.Default(), "provision default workspace", wsErr)
		return
	}

	// Create session
	sessionToken, err := h.createSession(ctx, userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create session", err)
		return
	}

	// Set session cookie with strict security configuration
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	// IMPORTANT: In development, DO NOT set Domain attribute to avoid browser security
	// issues with cross-origin requests (localhost:5173 -> localhost:8001)
	// The browser will handle cookie scope automatically for same hostname
	var domain string
	if isProduction {
		domain = os.Getenv("COOKIE_DOMAIN")
	}
	// In development, leave domain empty so browser handles it

	sameSite := sessionCookieSameSite(isProduction)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "better-auth.session_token",
		Value:    sessionToken,
		Path:     "/",
		Domain:   domain,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	// Send welcome notification
	if h.notificationTriggers != nil {
		go h.notificationTriggers.OnWelcome(context.Background(), userID, req.Name)
	}

	// Create default OSA workspace for new user (non-blocking, errors logged)
	if h.workspaceInit != nil {
		userUUID, parseErr := uuid.Parse(userID)
		if parseErr == nil {
			go func() {
				wsCtx, wsCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer wsCancel()

				_, wsErr := h.workspaceInit.CreateDefaultWorkspace(wsCtx, userUUID)
				if wsErr != nil {
					h.logger.Error("Failed to create default workspace for new user",
						slog.String("user_id", userID),
						slog.Any("error", wsErr),
					)
				} else {
					h.logger.Info("Default workspace created for new user",
						slog.String("user_id", userID),
					)
				}
			}()
		}
	}

	// Welcome email (best-effort; no-op if Resend isn't configured).
	go func(to, name string) {
		emailSvc := templates.NewEmailTemplateService()
		if emailSvc.IsEnabled() {
			if mailErr := emailSvc.SendWelcome(context.Background(), to, templates.WelcomeData{
				UserName:      name,
				WorkspaceName: "My Workspace",
			}); mailErr != nil {
				slog.Warn("[Auth] failed to send welcome email", "error", mailErr, "to", to)
			}
		}
	}(req.Email, req.Name)

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":    userID,
			"email": req.Email,
			"name":  req.Name,
		},
		"message": "Account created successfully",
	})
}

// SignIn handles user login with email/password
func (h *EmailAuthHandler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// SECURITY (OWASP A07): Check account lockout before touching the database.
	// Using a generic error message to avoid revealing whether the email exists.
	if isLockedOut(req.Email) {
		slog.Warn("SignIn: account locked out due to too many failed attempts",
			slog.String("email", req.Email),
		)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Too many failed login attempts. Please try again later.",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get user and password
	var userID, userName, storedPassword string
	err := h.pool.QueryRow(ctx, `
		SELECT u.id, u.name, a.password
		FROM "user" u
		JOIN account a ON a."userId" = u.id
		WHERE u.email = $1 AND a."providerId" = 'credential'
	`, req.Email).Scan(&userID, &userName, &storedPassword)

	if err != nil {
		// Record failure regardless of whether the email exists.
		if locked := recordFailedAttempt(req.Email); locked {
			slog.Warn("SignIn: account locked after repeated failures",
				slog.String("email", req.Email),
				slog.Duration("lock_duration", loginLockDuration),
			)
		}
		// Generic message: do not reveal whether the email exists.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(req.Password)); err != nil {
		if locked := recordFailedAttempt(req.Email); locked {
			slog.Warn("SignIn: account locked after repeated failures",
				slog.String("email", req.Email),
				slog.Duration("lock_duration", loginLockDuration),
			)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Successful login: reset the failure counter.
	resetLoginAttempts(req.Email)

	// Repair legacy accounts before issuing a session. This is idempotent and
	// preserves existing non-owner memberships.
	if _, wsErr := services.NewWorkspaceService(h.pool).EnsureDefaultWorkspace(ctx, userID); wsErr != nil {
		utils.RespondInternalError(c, slog.Default(), "ensure default workspace", wsErr)
		return
	}

	// Create session
	sessionToken, err := h.createSession(ctx, userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create session", err)
		return
	}

	// Set session cookie with strict security configuration
	isProduction := os.Getenv("ENVIRONMENT") == "production"

	// IMPORTANT: In development, DO NOT set Domain attribute to avoid browser security
	// issues with cross-origin requests (localhost:5173 -> localhost:8001)
	// The browser will handle cookie scope automatically for same hostname
	var domain string
	if isProduction {
		domain = os.Getenv("COOKIE_DOMAIN")
	}
	// In development, leave domain empty so browser handles it

	slog.Debug("auth: setting session cookie",
		"domain", domain,
		"isProduction", isProduction)

	sameSite := sessionCookieSameSite(isProduction)

	cookie := &http.Cookie{
		Name:     "better-auth.session_token",
		Value:    sessionToken,
		Path:     "/",
		Domain:   domain,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	}

	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    userID,
			"email": req.Email,
			"name":  userName,
		},
		"message": "Signed in successfully",
	})
}

// createSession creates a new session for the user
func (h *EmailAuthHandler) createSession(ctx context.Context, userID string) (string, error) {
	sessionToken := generateSessionToken()
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	commandTag, err := h.pool.Exec(ctx, `
		INSERT INTO session (id, "userId", token, "expiresAt", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, sessionID, userID, sessionToken, expiresAt)

	if err != nil {
		h.logger.Error("Failed to create session",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		return "", err
	}

	// Verify the session was actually inserted
	if commandTag.RowsAffected() != 1 {
		h.logger.Error("Session insert did not affect expected rows",
			slog.String("user_id", userID),
			slog.Int64("rows_affected", commandTag.RowsAffected()),
		)
		return "", fmt.Errorf("session insert affected %d rows, expected 1", commandTag.RowsAffected())
	}

	return sessionToken, nil
}
