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

// InitiateNotionOAuth starts the Notion OAuth flow.
// GET /api/auth/notion
func (h *OAuthIntegrationHandler) InitiateNotionOAuth(c *gin.Context) {
	if h.notion == nil {
		redirectWithError(c, "Notion integration not configured")
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

	// Notion requires owner=user parameter
	authURL := h.notion.AuthCodeURL(state, oauth2.SetAuthURLParam("owner", "user"))
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleNotionCallback handles the Notion OAuth callback.
// GET /api/auth/notion/callback
func (h *OAuthIntegrationHandler) HandleNotionCallback(c *gin.Context) {
	if h.notion == nil {
		redirectWithError(c, "Notion integration not configured")
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
		slog.Warn("OAuth callback: state validation failed", "provider", "notion")
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

	token, err := h.notion.Exchange(c.Request.Context(), code)
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
		// Notion doesn't specify scopes in the same way
		if err := h.storeOAuthToken(ctx, userID, "notion", token, nil); err != nil {
			slog.Warn("failed to store Notion token, continuing anyway", "error", err)
		}
	} else {
		slog.Warn("no user ID in context, skipping token storage for Notion")
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectAfter+"?integration=notion&status=connected")
}

// InitiateLinearOAuth starts the Linear OAuth flow.
// GET /api/auth/linear
func (h *OAuthIntegrationHandler) InitiateLinearOAuth(c *gin.Context) {
	if h.linear == nil {
		redirectWithError(c, "Linear integration not configured")
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

	authURL := h.linear.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleLinearCallback handles the Linear OAuth callback.
// GET /api/auth/linear/callback
func (h *OAuthIntegrationHandler) HandleLinearCallback(c *gin.Context) {
	if h.linear == nil {
		redirectWithError(c, "Linear integration not configured")
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
		slog.Warn("OAuth callback: state validation failed", "provider", "linear")
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

	token, err := h.linear.Exchange(c.Request.Context(), code)
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
		if err := h.storeOAuthToken(ctx, userID, "linear", token, h.linear.Scopes); err != nil {
			slog.Warn("failed to store Linear token, continuing anyway", "error", err)
		}
	} else {
		slog.Warn("no user ID in context, skipping token storage for Linear")
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectAfter+"?integration=linear&status=connected")
}
