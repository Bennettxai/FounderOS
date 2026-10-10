package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"golang.org/x/oauth2"
)

// InitiateMicrosoftOAuth starts the Microsoft OAuth flow.
// GET /api/auth/microsoft
func (h *OAuthIntegrationHandler) InitiateMicrosoftOAuth(c *gin.Context) {
	if h.microsoft == nil {
		redirectWithError(c, "Microsoft integration not configured")
		return
	}

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
	if sessionID != "" {
		c.SetCookie("oauth_session", sessionID, 600, "/", "", secure, true)
	}

	authURL := h.microsoft.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleMicrosoftCallback handles the Microsoft OAuth callback.
// GET /api/auth/microsoft/callback
func (h *OAuthIntegrationHandler) HandleMicrosoftCallback(c *gin.Context) {
	if h.microsoft == nil {
		redirectWithError(c, "Microsoft integration not configured")
		return
	}

	state := c.Query("state")
	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		slog.Warn("OAuth callback: missing state cookie")
		redirectWithError(c, "Invalid state parameter - session expired")
		return
	}

	storedSessionID, _ := c.Cookie("oauth_session")
	if !validateSessionBoundState(state, storedState, storedSessionID) {
		slog.Warn("OAuth callback: state validation failed", "provider", "microsoft")
		redirectWithError(c, "Invalid state parameter - possible CSRF attack")
		return
	}

	redirectAfter, _ := c.Cookie("oauth_redirect")
	if redirectAfter == "" {
		redirectAfter = "/onboarding/building"
	}

	code := c.Query("code")
	if code == "" {
		redirectWithError(c, "No authorization code received")
		return
	}

	token, err := h.microsoft.Exchange(c.Request.Context(), code)
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
		if err := h.storeOAuthToken(ctx, userID, "outlook", token, h.microsoft.Scopes); err != nil {
			slog.Warn("failed to store Microsoft token, continuing anyway", "error", err)
		}
	} else {
		slog.Warn("no user ID in context, skipping token storage for Microsoft")
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectAfter+"?integration=outlook&status=connected")
}
