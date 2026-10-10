// Package google provides individual Google tool integrations.
// Each tool (Calendar, Gmail, Drive, etc.) is a separate integration with its own scopes.
package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/security"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ============================================================================
// TOOL DEFINITIONS
// Each Google service is a separate "tool" with its own provider ID and scopes
// ============================================================================

// ToolDefinition defines a single Google tool (Calendar, Gmail, etc.)
type ToolDefinition struct {
	ID          string   // e.g., "google_calendar"
	Name        string   // e.g., "Google Calendar"
	Description string   // Human-readable description
	Category    string   // e.g., "calendar", "email", "storage"
	Scopes      []string // OAuth scopes required for this tool only
	Modules     []string // BusinessOS modules this tool integrates with
}

// All available Google tools
// Each tool requests ALL useful scopes for that specific service
var GoogleTools = map[string]*ToolDefinition{
	"google_calendar": {
		ID:          "google_calendar",
		Name:        "Google Calendar",
		Description: "Full calendar access - view, create, edit, delete events and manage calendars",
		Category:    "calendar",
		Scopes: []string{
			// One scope covers the read/write operations used by CalendarService.
			"https://www.googleapis.com/auth/calendar",
		},
		Modules: []string{"calendar", "daily_log", "projects"},
	},
	"google_gmail": {
		ID:          "google_gmail",
		Name:        "Gmail",
		Description: "Full Gmail access - read, send, compose, organize emails and manage settings",
		Category:    "email",
		Scopes: []string{
			// Full Gmail access (read, compose, send, delete)
			"https://mail.google.com/",
			// Granular scopes for specific operations
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/gmail.settings.basic",
			"https://www.googleapis.com/auth/gmail.insert",
		},
		Modules: []string{"chat", "daily_log", "clients"},
	},
	"google_drive": {
		ID:          "google_drive",
		Name:        "Google Drive",
		Description: "Full Drive access - view, create, edit, organize files and folders",
		Category:    "storage",
		Scopes: []string{
			// Full Drive access
			"https://www.googleapis.com/auth/drive",
			// Granular scopes
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/drive.file",
			"https://www.googleapis.com/auth/drive.metadata",
			"https://www.googleapis.com/auth/drive.metadata.readonly",
			"https://www.googleapis.com/auth/drive.appdata",
		},
		Modules: []string{"contexts", "projects"},
	},
	"google_contacts": {
		ID:          "google_contacts",
		Name:        "Google Contacts",
		Description: "Full contacts access - view, create, edit, delete contacts",
		Category:    "contacts",
		Scopes: []string{
			// Full contacts access
			"https://www.googleapis.com/auth/contacts",
			"https://www.googleapis.com/auth/contacts.readonly",
			"https://www.googleapis.com/auth/contacts.other.readonly",
			"https://www.googleapis.com/auth/directory.readonly",
		},
		Modules: []string{"clients", "team"},
	},
	"google_tasks": {
		ID:          "google_tasks",
		Name:        "Google Tasks",
		Description: "Full tasks access - view, create, edit, delete tasks and task lists",
		Category:    "tasks",
		Scopes: []string{
			// Full tasks access
			"https://www.googleapis.com/auth/tasks",
			"https://www.googleapis.com/auth/tasks.readonly",
		},
		Modules: []string{"tasks", "projects"},
	},
}

// ============================================================================
// TOOL PROVIDER
// A provider instance for a specific Google tool
// ============================================================================

// ToolProvider provides OAuth and API access for a specific Google tool.
type ToolProvider struct {
	pool        *pgxpool.Pool
	tool        *ToolDefinition
	oauthConfig *oauth2.Config
}

// NewToolProvider creates a provider for a specific Google tool.
func NewToolProvider(pool *pgxpool.Pool, toolID string) (*ToolProvider, error) {
	tool, ok := GoogleTools[toolID]
	if !ok {
		return nil, fmt.Errorf("unknown Google tool: %s", toolID)
	}

	// Use the same config as the main Google provider
	cfg := config.AppConfig

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, fmt.Errorf("Google OAuth credentials not configured")
	}

	// Use tool-specific redirect URL
	redirectURL := resolveToolRedirectURL(cfg, toolID)

	// Always include basic profile scopes + tool-specific scopes
	scopes := append([]string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}, tool.Scopes...)

	return &ToolProvider{
		pool: pool,
		tool: tool,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
	}, nil
}

func resolveToolRedirectURL(cfg *config.Config, toolID string) string {
	redirectURL := cfg.GoogleIntegrationRedirectURI
	switch toolID {
	case "google_calendar":
		if cfg.GoogleCalendarRedirectURI != "" {
			redirectURL = cfg.GoogleCalendarRedirectURI
		}
	case "google_gmail":
		if cfg.GoogleGmailRedirectURI != "" {
			redirectURL = cfg.GoogleGmailRedirectURI
		}
	}
	if redirectURL == "" {
		return cfg.GoogleRedirectURI
	}
	return redirectURL
}

// ID returns the tool's provider ID.
func (tp *ToolProvider) ID() string {
	return tp.tool.ID
}

// Name returns the tool's display name.
func (tp *ToolProvider) Name() string {
	return tp.tool.Name
}

// Category returns the tool's category.
func (tp *ToolProvider) Category() string {
	return tp.tool.Category
}

// Pool returns the database pool.
func (tp *ToolProvider) Pool() *pgxpool.Pool {
	return tp.pool
}

// GetAuthURL returns the OAuth authorization URL for this specific tool.
func (tp *ToolProvider) GetAuthURL(state string) string {
	return tp.oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// ExchangeCode exchanges an authorization code for tokens.
func (tp *ToolProvider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	token, err := tp.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user email
	email, err := getUserEmail(ctx, tp.oauthConfig, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}

	// Extract scopes
	var scopes []string
	if scopeStr, ok := token.Extra("scope").(string); ok {
		scopes = strings.Split(scopeStr, " ")
	}
	if len(scopes) == 0 {
		scopes = append([]string(nil), tp.oauthConfig.Scopes...)
	}

	return &integrations.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		Scopes:       scopes,
		AccountEmail: email,
		AccountName:  email,
		Metadata: map[string]interface{}{
			"google_email": email,
			"tool_id":      tp.tool.ID,
		},
	}, nil
}

// SaveToken saves the OAuth token for this tool.
func (tp *ToolProvider) SaveToken(ctx context.Context, userID string, token *integrations.TokenResponse) error {
	tx, err := tp.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert or update user_integrations for this specific tool
	_, err = tx.Exec(ctx, `
		INSERT INTO user_integrations (
			user_id, provider_id, status, connected_at,
			external_account_id, external_account_name, scopes, metadata
		) VALUES ($1, $2, 'connected', NOW(), $3, $4, $5, $6)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			status = 'connected',
			connected_at = NOW(),
			external_account_id = EXCLUDED.external_account_id,
			external_account_name = EXCLUDED.external_account_name,
			scopes = EXCLUDED.scopes,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, userID, tp.tool.ID, token.AccountEmail, token.AccountName, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	payload, err := encodeToolToken(toolTokenPayload{
		AccessToken: token.AccessToken, RefreshToken: token.RefreshToken,
	})
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// Tool credentials must be isolated by provider ID. The legacy Google table
	// has one row per user, so using it here lets Gmail overwrite Calendar (and
	// vice versa) while both integrations still appear connected.
	_, err = tx.Exec(ctx, `
		INSERT INTO credential_vault (
			user_id, provider_id, credential_type, encrypted_data, expires_at,
			external_account_email, scopes, metadata
		) VALUES ($1, $2, 'oauth', $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			encrypted_data = EXCLUDED.encrypted_data,
			expires_at = EXCLUDED.expires_at,
			external_account_email = EXCLUDED.external_account_email,
			scopes = EXCLUDED.scopes,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, userID, tp.tool.ID, payload, token.ExpiresAt, token.AccountEmail, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return tx.Commit(ctx)
}

// GetToken retrieves the OAuth token for this tool.
func (tp *ToolProvider) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	var encrypted []byte
	var expiry time.Time
	var scopes []string

	err := tp.pool.QueryRow(ctx, `
		SELECT encrypted_data, expires_at, scopes
		FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, tp.tool.ID).Scan(&encrypted, &expiry, &scopes)
	if err == nil {
		payload, decodeErr := decodeToolToken(encrypted)
		if decodeErr != nil {
			return nil, fmt.Errorf("invalid credential for tool %s: %w", tp.tool.ID, decodeErr)
		}
		if !grantsToolAccess(tp.tool.ID, scopes) {
			return nil, fmt.Errorf("credential for tool %s lacks required scopes", tp.tool.ID)
		}
		return &oauth2.Token{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, Expiry: expiry}, nil
	}

	// Backward-compatible recovery for installations that have not reconnected
	// since provider-specific credentials were introduced. Never use a legacy
	// token unless its actual granted scopes satisfy this tool.
	var accessToken, refreshToken string
	err = tp.pool.QueryRow(ctx, `
		SELECT access_token, refresh_token, expiry, scopes
		FROM google_oauth_tokens
		WHERE user_id = $1
	`, userID).Scan(&accessToken, &refreshToken, &expiry, &scopes)
	if err != nil || !grantsToolAccess(tp.tool.ID, scopes) {
		return nil, fmt.Errorf("no authorized token found for tool %s", tp.tool.ID)
	}

	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}, nil
}

// GetTokenSource returns a TokenSource that auto-refreshes.
func (tp *ToolProvider) GetTokenSource(ctx context.Context, userID string) (oauth2.TokenSource, error) {
	token, err := tp.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create a token source that auto-refreshes
	tokenSource := tp.oauthConfig.TokenSource(ctx, token)

	// Get a fresh token to trigger refresh if needed
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// If token was refreshed, save the new one
	if newToken.AccessToken != token.AccessToken {
		slog.Info("Token refreshed for user , tool", "user_id", userID, "tool_id", tp.tool.ID)
		tp.saveRefreshedToken(ctx, userID, newToken)
	}

	return tokenSource, nil
}

// saveRefreshedToken saves a refreshed token.
func (tp *ToolProvider) saveRefreshedToken(ctx context.Context, userID string, token *oauth2.Token) {
	payload, err := encodeToolToken(toolTokenPayload{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken})
	if err != nil {
		slog.Info("Failed to encode refreshed token", "error", err)
		return
	}
	_, err = tp.pool.Exec(ctx, `
		UPDATE credential_vault
		SET encrypted_data = $1, expires_at = $2, updated_at = NOW()
		WHERE user_id = $3 AND provider_id = $4
	`, payload, token.Expiry, userID, tp.tool.ID)
	if err != nil {
		slog.Info("Failed to save refreshed token", "error", err)
	}
}

// IsConnected checks if the user has this tool connected.
func (tp *ToolProvider) IsConnected(ctx context.Context, userID string) bool {
	var count int
	err := tp.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
	`, userID, tp.tool.ID).Scan(&count)
	return err == nil && count > 0
}

// Disconnect removes the user's connection to this tool.
func (tp *ToolProvider) Disconnect(ctx context.Context, userID string) error {
	tx, err := tp.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete from user_integrations
	_, err = tx.Exec(ctx, `
		DELETE FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, tp.tool.ID)
	if err != nil {
		return err
	}

	// Delete from credential_vault
	_, err = tx.Exec(ctx, `
		DELETE FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, tp.tool.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetConnectionStatus returns the connection status for this tool.
func (tp *ToolProvider) GetConnectionStatus(ctx context.Context, userID string) (*integrations.ConnectionStatus, error) {
	var status integrations.ConnectionStatus

	err := tp.pool.QueryRow(ctx, `
		SELECT
			COALESCE(status = 'connected', false) as connected,
			connected_at,
			external_account_id,
			external_account_name,
			scopes,
			last_used_at
		FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, tp.tool.ID).Scan(
		&status.Connected,
		&status.ConnectedAt,
		&status.AccountID,
		&status.AccountName,
		&status.Scopes,
		&status.LastSyncAt,
	)

	if err != nil {
		return &integrations.ConnectionStatus{Connected: false}, nil
	}

	return &status, nil
}

// HasAuthorizedCredential checks the scopes on the credential that will
// actually be used, rather than trusting stale connection metadata.
func (tp *ToolProvider) HasAuthorizedCredential(ctx context.Context, userID string) bool {
	var scopes []string
	err := tp.pool.QueryRow(ctx, `
		SELECT scopes FROM credential_vault WHERE user_id = $1 AND provider_id = $2
	`, userID, tp.tool.ID).Scan(&scopes)
	if err == nil {
		return grantsToolAccess(tp.tool.ID, scopes)
	}
	err = tp.pool.QueryRow(ctx, `SELECT scopes FROM google_oauth_tokens WHERE user_id = $1`, userID).Scan(&scopes)
	return err == nil && grantsToolAccess(tp.tool.ID, scopes)
}

type toolTokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func encodeToolToken(token toolTokenPayload) ([]byte, error) {
	plain, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	if encryption := security.GetGlobalEncryption(); encryption != nil {
		return encryption.EncryptBytes(string(plain))
	}
	return plain, nil
}

func decodeToolToken(data []byte) (toolTokenPayload, error) {
	plain := string(data)
	if encryption := security.GetGlobalEncryption(); encryption != nil {
		decrypted, err := encryption.DecryptBytes(data)
		if err != nil {
			return toolTokenPayload{}, err
		}
		plain = decrypted
	}
	var token toolTokenPayload
	err := json.Unmarshal([]byte(plain), &token)
	return token, err
}

func grantsToolAccess(toolID string, scopes []string) bool {
	granted := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		granted[scope] = true
	}
	switch toolID {
	case "google_calendar":
		return granted["https://www.googleapis.com/auth/calendar"]
	case "google_gmail":
		return granted["https://mail.google.com/"]
	default:
		return len(scopes) > 0
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getUserEmail fetches the user's email from Google.
func getUserEmail(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (string, error) {
	client := config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
	}

	if err := decodeJSON(resp.Body, &userInfo); err != nil {
		return "", err
	}

	return userInfo.Email, nil
}
