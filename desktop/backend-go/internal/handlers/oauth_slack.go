package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// InitiateSlackOAuth starts the Slack OAuth flow.
// GET /api/auth/slack
func (h *OAuthIntegrationHandler) InitiateSlackOAuth(c *gin.Context) {
	if h.slack == nil {
		redirectWithError(c, "Slack integration not configured")
		return
	}

	// Get session ID to bind state to session (prevents CSRF)
	user := middleware.GetCurrentUser(c)
	sessionID := ""
	if user != nil {
		sessionID = user.ID
	}
	state, err := generateSessionBoundState(sessionID)
	if err != nil {
		slog.Error("failed to generate OAuth state", "error", err)
		redirectWithError(c, "Failed to initiate OAuth flow")
		return
	}

	redirectAfter := c.Query("redirect")
	if redirectAfter == "" {
		redirectAfter = "/onboarding/building"
	}

	secure := h.isSecureCookie(c)
	c.SetCookie("oauth_state", state, 600, "/", "", secure, true)
	c.SetCookie("oauth_redirect", redirectAfter, 600, "/", "", secure, true)
	// Store session ID for callback validation
	if sessionID != "" {
		c.SetCookie("oauth_session", sessionID, 600, "/", "", secure, true)
	}

	authURL := h.slack.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleSlackCallback handles the Slack OAuth callback.
// GET /api/auth/slack/callback
func (h *OAuthIntegrationHandler) HandleSlackCallback(c *gin.Context) {
	if h.slack == nil {
		redirectWithError(c, "Slack integration not configured")
		return
	}

	// Verify state with session binding
	state := c.Query("state")
	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		slog.Warn("OAuth callback: missing state cookie")
		redirectWithError(c, "Invalid state parameter - session expired")
		return
	}

	// Get stored session ID for validation
	storedSessionID, _ := c.Cookie("oauth_session")
	if !validateSessionBoundState(state, storedState, storedSessionID) {
		slog.Warn("OAuth callback: state validation failed", "provider", "slack")
		redirectWithError(c, "Invalid state parameter - possible CSRF attack")
		return
	}

	// Get redirect URL
	redirectAfter, _ := c.Cookie("oauth_redirect")
	if redirectAfter == "" {
		redirectAfter = "/onboarding/building"
	}

	// Exchange code for token
	code := c.Query("code")
	if code == "" {
		redirectWithError(c, "No authorization code received")
		return
	}

	token, err := h.slack.Exchange(c.Request.Context(), code)
	if err != nil {
		redirectWithError(c, "Failed to exchange token: "+err.Error())
		return
	}

	// Store token in user_integrations table
	user := middleware.GetCurrentUser(c)
	userID := ""
	if user != nil {
		userID = user.ID
	}
	if userID != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := h.storeOAuthToken(ctx, userID, "slack", token, h.slack.Scopes); err != nil {
			slog.Warn("failed to store Slack token, continuing anyway", "error", err)
		}
	} else {
		slog.Warn("no user ID in context, skipping token storage for Slack")
	}

	// Redirect back to frontend
	c.Redirect(http.StatusTemporaryRedirect, redirectAfter+"?integration=slack&status=connected")
}
