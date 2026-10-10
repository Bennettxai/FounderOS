// Package airtable provides the Airtable integration (Bases, Tables, Records).
package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations"
)

const (
	ProviderID   = "airtable"
	ProviderName = "Airtable"
	Category     = "productivity"
)

// OAuth endpoints
const (
	AuthURL  = "https://airtable.com/oauth2/v1/authorize"
	TokenURL = "https://airtable.com/oauth2/v1/token"
	APIURL   = "https://api.airtable.com/v0"
)

// Airtable OAuth Scopes - Comprehensive access
// Format: resource.action
var DefaultScopes = []string{
	// ============================================================================
	// DATA ACCESS - Records
	// ============================================================================
	"data.records:read",         // Read records from bases
	"data.records:write",        // Create, update, delete records
	"data.recordComments:read",  // Read record comments
	"data.recordComments:write", // Create, update, delete comments

	// ============================================================================
	// SCHEMA ACCESS - Bases and Tables structure
	// ============================================================================
	"schema.bases:read",  // Read base metadata (tables, fields, views)
	"schema.bases:write", // Create/modify bases, tables, fields

	// ============================================================================
	// WORKSPACE ACCESS
	// ============================================================================
	"workspacesAndBases:manage", // Manage workspaces and bases

	// ============================================================================
	// USER INFO
	// ============================================================================
	"user.email:read", // Read user email for identification

	// ============================================================================
	// WEBHOOKS
	// ============================================================================
	"webhook:manage", // Create and manage webhooks
}

// AllAirtableScopes contains all available scopes
var AllAirtableScopes = []string{
	"data.records:read",
	"data.records:write",
	"data.recordComments:read",
	"data.recordComments:write",
	"schema.bases:read",
	"schema.bases:write",
	"workspacesAndBases:manage",
	"user.email:read",
	"webhook:manage",
	// Enterprise scopes (may require specific plans)
	"enterprise:manage",
	"enterprise.user:read",
	"enterprise.user:write",
	"enterprise.account:read",
	"enterprise.auditLogs:read",
}

// Provider implements the integrations.Provider interface for Airtable.
type Provider struct {
	pool         *pgxpool.Pool
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

// NewProvider creates a new Airtable provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	cfg := config.AppConfig

	redirectURI := cfg.AirtableRedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s/api/integrations/airtable/callback", cfg.BaseURL)
	}

	return &Provider{
		pool:         pool,
		clientID:     cfg.AirtableClientID,
		clientSecret: cfg.AirtableClientSecret,
		redirectURI:  redirectURI,
		scopes:       DefaultScopes,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderID
}

// DisplayName returns the human-readable provider name.
func (p *Provider) DisplayName() string {
	return ProviderName
}

// Category returns the provider category.
func (p *Provider) Category() string {
	return Category
}

// Icon returns the provider icon URL.
func (p *Provider) Icon() string {
	return "/icons/airtable.svg"
}

// ============================================================================
// OAuth Methods
// ============================================================================

// GetAuthURL returns the OAuth authorization URL.
// Airtable uses OAuth 2.0 with PKCE, but we'll use the standard flow for simplicity.
func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(p.scopes, " "))
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("client_id", p.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Airtable uses HTTP Basic Auth for token exchange
	req.SetBasicAuth(p.clientID, p.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"` // Usually 2 hours (7200 seconds)
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Get user info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		userInfo = &airtableUser{}
	}

	// Calculate expiry
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Parse scopes
	scopes := strings.Split(tokenResp.Scope, " ")

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
		AccountID:    userInfo.ID,
		AccountEmail: userInfo.Email,
		Metadata: map[string]interface{}{
			"user_id": userInfo.ID,
			"email":   userInfo.Email,
		},
	}, nil
}

// RefreshToken refreshes an expired access token.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.clientID, p.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	scopes := strings.Split(tokenResp.Scope, " ")

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
	}, nil
}

// SaveToken saves OAuth tokens.
func (p *Provider) SaveToken(ctx context.Context, userID string, token *integrations.TokenResponse) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert or update user_integrations
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
	`, userID, ProviderID, token.AccountID, token.AccountEmail, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	// Save to credential_vault (store both access and refresh tokens)
	tokenData := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
	}
	tokenJSON, _ := json.Marshal(tokenData)

	_, err = tx.Exec(ctx, `
		INSERT INTO credential_vault (
			user_id, provider_id, credential_type, encrypted_data, expires_at
		) VALUES ($1, $2, 'oauth_token', $3, $4)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			encrypted_data = EXCLUDED.encrypted_data,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, userID, ProviderID, tokenJSON, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return tx.Commit(ctx)
}

// GetToken retrieves OAuth tokens for a user.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	var token integrations.Token
	var tokenData []byte
	var expiry time.Time

	err := p.pool.QueryRow(ctx, `
		SELECT encrypted_data, expires_at
		FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID).Scan(&tokenData, &expiry)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	// Parse token data
	var tokenMap map[string]string
	if err := json.Unmarshal(tokenData, &tokenMap); err != nil {
		// Fallback: assume it's just the access token as string
		token.AccessToken = string(tokenData)
	} else {
		token.AccessToken = tokenMap["access_token"]
		token.RefreshToken = tokenMap["refresh_token"]
	}
	token.ExpiresAt = expiry

	// Get scopes from user_integrations
	p.pool.QueryRow(ctx, `
		SELECT scopes FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID).Scan(&token.Scopes)

	return &token, nil
}

// ============================================================================
// Connection Methods
// ============================================================================

// GetConnectionStatus returns the connection status for a user.
func (p *Provider) GetConnectionStatus(ctx context.Context, userID string) (*integrations.ConnectionStatus, error) {
	var status integrations.ConnectionStatus

	err := p.pool.QueryRow(ctx, `
		SELECT
			COALESCE(status = 'connected', false) as connected,
			connected_at,
			external_account_id,
			external_account_name,
			scopes,
			updated_at
		FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID).Scan(
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

// Disconnect removes the user's connection to Airtable.
func (p *Provider) Disconnect(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID)
	if err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	_, err = p.pool.Exec(ctx, `
		DELETE FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID)
	if err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

// ============================================================================
// Sync Methods
// ============================================================================

// SupportsSync returns true since Airtable supports sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a sync operation for the specified resources.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	result := &integrations.SyncResult{
		Success: true,
	}
	start := time.Now()

	// Sync operations would be implemented here
	// - Bases
	// - Tables
	// - Records

	result.Duration = time.Since(start)
	return result, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// getUserInfo retrieves user information from Airtable.
func (p *Provider) getUserInfo(ctx context.Context, accessToken string) (*airtableUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.airtable.com/v0/meta/whoami", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &airtableUser{
		ID:    result.ID,
		Email: result.Email,
	}, nil
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}
