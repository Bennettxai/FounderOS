package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services/templates"
	"github.com/rhl/businessos-backend/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetHandler implements the forgot/reset password flow using the
// Better-Auth-compatible `verification` table for single-use, expiring tokens.
type PasswordResetHandler struct {
	pool *pgxpool.Pool
}

func NewPasswordResetHandler(pool *pgxpool.Pool) *PasswordResetHandler {
	return &PasswordResetHandler{pool: pool}
}

const resetTokenPrefix = "password-reset:"
const resetTokenTTL = time.Hour

// ForgetPassword starts a reset: emails a link if the account exists.
// POST /api/auth/forget-password  {email}
//
// Always returns 200 with the same message so the endpoint can't be used to
// enumerate which emails have accounts.
func (h *PasswordResetHandler) ForgetPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	okMsg := gin.H{"message": "If an account exists for that email, a reset link is on its way."}

	var userID, name string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT id, COALESCE(name, '') FROM "user" WHERE email = $1`, email).Scan(&userID, &name)
	if err != nil {
		// No such user — respond success anyway (no enumeration).
		c.JSON(http.StatusOK, okMsg)
		return
	}

	// Generate a single-use token and store it with a 1h expiry.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		utils.RespondInternalError(c, slog.Default(), "generate reset token", err)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Invalidate any prior tokens for this user, then insert the new one.
	_, _ = h.pool.Exec(c.Request.Context(),
		`DELETE FROM verification WHERE identifier = $1`, resetTokenPrefix+userID)
	_, err = h.pool.Exec(c.Request.Context(),
		`INSERT INTO verification (id, identifier, value, "expiresAt") VALUES ($1, $2, $3, $4)`,
		generateUserID(), resetTokenPrefix+userID, token, time.Now().Add(resetTokenTTL))
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "store reset token", err)
		return
	}

	// Send the email (best-effort; don't block or leak failures to the client).
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://app.businessos.dev"
	}
	resetLink := appURL + "/reset-password?token=" + token
	device := c.Request.UserAgent()
	ip := c.ClientIP()
	go func() {
		emailSvc := templates.NewEmailTemplateService()
		if !emailSvc.IsEnabled() {
			slog.Warn("[PasswordReset] email service disabled; reset link not sent", "email", email)
			return
		}
		if mailErr := emailSvc.SendPasswordReset(context.Background(), email, name, resetLink, "1 hour", ip, device); mailErr != nil {
			slog.Warn("[PasswordReset] failed to send reset email", "error", mailErr, "email", email)
		}
	}()

	c.JSON(http.StatusOK, okMsg)
}

// ResetPassword completes the reset.
// POST /api/auth/reset-password  {token, newPassword}
func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	if msg := validatePassword(req.NewPassword); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Look up a live token.
	var identifier string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT identifier FROM verification
		 WHERE value = $1 AND identifier LIKE $2 AND "expiresAt" > NOW()`,
		req.Token, resetTokenPrefix+"%").Scan(&identifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This reset link is invalid or has expired. Please request a new one."})
		return
	}
	userID := strings.TrimPrefix(identifier, resetTokenPrefix)

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "hash password", err)
		return
	}

	tag, err := h.pool.Exec(c.Request.Context(),
		`UPDATE account SET password = $1, "updatedAt" = NOW()
		 WHERE "userId" = $2 AND "providerId" = 'credential'`,
		string(hashed), userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update password", err)
		return
	}
	if tag.RowsAffected() == 0 {
		// User signed up via Google only (no credential account) — create one so
		// they can use a password too.
		_, err = h.pool.Exec(c.Request.Context(),
			`INSERT INTO account (id, "userId", "accountId", "providerId", password, "createdAt", "updatedAt")
			 VALUES ($1, $2, $2, 'credential', $3, NOW(), NOW())`,
			generateUserID(), userID, string(hashed))
		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "create credential account", err)
			return
		}
	}

	// Consume the token and revoke existing sessions (force re-login everywhere).
	_, _ = h.pool.Exec(c.Request.Context(), `DELETE FROM verification WHERE value = $1`, req.Token)
	_, _ = h.pool.Exec(c.Request.Context(), `DELETE FROM session WHERE "userId" = $1`, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Your password has been reset. You can now sign in."})
}
