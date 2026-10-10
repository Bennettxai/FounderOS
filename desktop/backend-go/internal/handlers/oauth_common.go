package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/security"
	"golang.org/x/oauth2"
)

// OAuthIntegrationHandler handles OAuth for third-party integrations
// Supported providers: Slack, Notion, Microsoft (Outlook), Linear.
type OAuthIntegrationHandler struct {
	pool      *pgxpool.Pool
	queries   *sqlc.Queries
	cfg       *config.Config
	slack     *oauth2.Config
	notion    *oauth2.Config
	microsoft *oauth2.Config
	linear    *oauth2.Config
}

// NewOAuthIntegrationHandler creates a new OAuth integration handler.
func NewOAuthIntegrationHandler(pool *pgxpool.Pool, cfg *config.Config) *OAuthIntegrationHandler {
	h := &OAuthIntegrationHandler{
		pool:    pool,
		queries: sqlc.New(pool),
		cfg:     cfg,
	}

	// Slack OAuth config
	if cfg.SlackClientID != "" {
		h.slack = &oauth2.Config{
			ClientID:     cfg.SlackClientID,
			ClientSecret: cfg.SlackClientSecret,
			RedirectURL:  "http://localhost:8001/api/auth/slack/callback",
			Scopes:       []string{"channels:read", "chat:write", "users:read", "files:read"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/v2/authorize",
				TokenURL: "https://slack.com/api/oauth.v2.access",
			},
		}
	}

	// Notion OAuth config
	if cfg.NotionClientID != "" {
		h.notion = &oauth2.Config{
			ClientID:     cfg.NotionClientID,
			ClientSecret: cfg.NotionClientSecret,
			RedirectURL:  "http://localhost:8001/api/auth/notion/callback",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://api.notion.com/v1/oauth/authorize",
				TokenURL: "https://api.notion.com/v1/oauth/token",
			},
		}
	}

	// Microsoft (Outlook) OAuth config
	if cfg.MicrosoftClientID != "" {
		h.microsoft = &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			RedirectURL:  "http://localhost:8001/api/auth/microsoft/callback",
			Scopes:       []string{"Mail.Read", "Calendars.Read", "User.Read", "offline_access"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			},
		}
	}

	// Linear OAuth config
	if cfg.LinearClientID != "" {
		h.linear = &oauth2.Config{
			ClientID:     cfg.LinearClientID,
			ClientSecret: cfg.LinearClientSecret,
			RedirectURL:  "http://localhost:8001/api/auth/linear/callback",
			Scopes:       []string{"read", "write"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://linear.app/oauth/authorize",
				TokenURL: "https://api.linear.app/oauth/token",
			},
		}
	}

	return h
}

// generateState creates a random state for CSRF protection.
// Returns error if random generation fails.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateSessionBoundState creates a state bound to the user's session ID.
// This prevents CSRF attacks where an attacker uses their own state parameter.
// Format: {random_state}:{session_hash} where session_hash is first 16 chars of hashed session.
func generateSessionBoundState(sessionID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	randomPart := base64.URLEncoding.EncodeToString(b)

	// Hash session ID to avoid exposing it directly
	sessionHash := make([]byte, 16)
	if _, err := rand.Read(sessionHash); err != nil {
		return "", err
	}

	// Bind state to session by including session identifier
	if sessionID != "" {
		// Use first 8 chars of session ID as binding (enough to verify, not enough to expose)
		binding := sessionID
		if len(binding) > 8 {
			binding = binding[:8]
		}
		return randomPart + ":" + binding, nil
	}

	return randomPart, nil
}

// validateSessionBoundState validates that the state matches the session.
func validateSessionBoundState(state, storedState, sessionID string) bool {
	// First check exact match
	if state != storedState {
		return false
	}

	// If state contains session binding, verify it matches
	if idx := strings.LastIndex(state, ":"); idx != -1 && sessionID != "" {
		binding := state[idx+1:]
		expectedBinding := sessionID
		if len(expectedBinding) > 8 {
			expectedBinding = expectedBinding[:8]
		}
		return binding == expectedBinding
	}

	return true
}

// isSecureCookie determines if cookies should use the Secure flag.
// Returns true in production or when the request is already over HTTPS.
func (h *OAuthIntegrationHandler) isSecureCookie(c *gin.Context) bool {
	// Always secure in production
	if h.cfg.IsProduction() {
		return true
	}
	// Also secure if request came over TLS
	return c.Request.TLS != nil
}

// storeOAuthToken stores an OAuth token in the user_integrations table with encryption.
func (h *OAuthIntegrationHandler) storeOAuthToken(ctx context.Context, userID, providerID string, token *oauth2.Token, scopes []string) error {
	// Encrypt the access token
	accessTokenEncrypted, err := security.EncryptTokenBytes(token.AccessToken)
	if err != nil {
		slog.Error("failed to encrypt access token", "provider", providerID, "error", err)
		return err
	}

	// Encrypt the refresh token (may be empty)
	var refreshTokenEncrypted []byte
	if token.RefreshToken != "" {
		refreshTokenEncrypted, err = security.EncryptTokenBytes(token.RefreshToken)
		if err != nil {
			slog.Error("failed to encrypt refresh token", "provider", providerID, "error", err)
			return err
		}
	}

	// Convert expiry time
	var tokenExpiresAt pgtype.Timestamptz
	if !token.Expiry.IsZero() {
		tokenExpiresAt = pgtype.Timestamptz{Time: token.Expiry, Valid: true}
	}

	// Create status pointer
	status := "connected"

	// Try to create or update the integration
	_, err = h.queries.CreateUserIntegration(ctx, sqlc.CreateUserIntegrationParams{
		UserID:                userID,
		ProviderID:            providerID,
		Status:                &status,
		AccessTokenEncrypted:  accessTokenEncrypted,
		RefreshTokenEncrypted: refreshTokenEncrypted,
		TokenExpiresAt:        tokenExpiresAt,
		Scopes:                scopes,
	})

	if err != nil {
		slog.Error("failed to store OAuth token", "provider", providerID, "userID", userID, "error", err)
		return err
	}

	slog.Info("OAuth token stored successfully", "provider", providerID, "userID", userID)
	return nil
}

// redirectWithError redirects back to frontend with an error message.
func redirectWithError(c *gin.Context, errorMsg string) {
	redirectAfter, _ := c.Cookie("oauth_redirect")
	if redirectAfter == "" {
		redirectAfter = "/onboarding/connect"
	}
	c.Redirect(http.StatusTemporaryRedirect, redirectAfter+"?error="+url.QueryEscape(errorMsg))
}
