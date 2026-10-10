// Package notion provides the Notion integration (databases, pages, tasks).
package notion

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
	ProviderID   = "notion"
	ProviderName = "Notion"
	Category     = "productivity"
)

// OAuth endpoints and API configuration
const (
	AuthURL    = "https://api.notion.com/v1/oauth/authorize"
	TokenURL   = "https://api.notion.com/v1/oauth/token"
	APIURL     = "https://api.notion.com/v1"
	APIVersion = "2022-06-28" // Notion API version - update as new features are needed
	// Note: Notion releases new API versions periodically with new features
	// See: https://developers.notion.com/reference/changes-by-version
)

// Provider implements the integrations.Provider interface for Notion.
type Provider struct {
	pool         *pgxpool.Pool
	clientID     string
	clientSecret string
	redirectURI  string
}

// NewProvider creates a new Notion provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	cfg := config.AppConfig

	redirectURI := cfg.NotionRedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s/api/integrations/notion/callback", cfg.BaseURL)
	}

	return &Provider{
		pool:         pool,
		clientID:     cfg.NotionClientID,
		clientSecret: cfg.NotionClientSecret,
		redirectURI:  redirectURI,
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
	return "/icons/notion.svg"
}

// GetAuthURL returns the OAuth authorization URL.
func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("response_type", "code")
	params.Set("owner", "user")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	// Notion uses Basic Auth for token exchange
	data := map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": p.redirectURI,
	}
	jsonData, _ := json.Marshal(data)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken          string `json:"access_token"`
		TokenType            string `json:"token_type"`
		BotID                string `json:"bot_id"`
		DuplicatedTemplateID string `json:"duplicated_template_id"`
		WorkspaceID          string `json:"workspace_id"`
		WorkspaceName        string `json:"workspace_name"`
		WorkspaceIcon        string `json:"workspace_icon"`
		Owner                struct {
			Type string `json:"type"`
			User struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"owner"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("notion oauth error: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: "", // Notion doesn't use refresh tokens
		ExpiresAt:    time.Now().AddDate(10, 0, 0), // Notion tokens don't expire
		AccountID:    tokenResp.WorkspaceID,
		AccountName:  tokenResp.WorkspaceName,
		AccountEmail: tokenResp.Owner.User.Email,
		Metadata: map[string]interface{}{
			"workspace_id":   tokenResp.WorkspaceID,
			"workspace_name": tokenResp.WorkspaceName,
			"workspace_icon": tokenResp.WorkspaceIcon,
			"bot_id":         tokenResp.BotID,
			"owner_id":       tokenResp.Owner.User.ID,
			"owner_name":     tokenResp.Owner.User.Name,
		},
	}, nil
}

// RefreshToken refreshes an expired access token.
// Notion tokens don't expire, so this is a no-op.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	return nil, fmt.Errorf("notion tokens do not expire")
}

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

// Disconnect removes the user's connection to Notion.
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
			external_account_id, external_account_name, metadata
		) VALUES ($1, $2, 'connected', NOW(), $3, $4, $5)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			status = 'connected',
			connected_at = NOW(),
			external_account_id = EXCLUDED.external_account_id,
			external_account_name = EXCLUDED.external_account_name,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, userID, ProviderID, token.AccountID, token.AccountName, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	// Save to credential_vault
	_, err = tx.Exec(ctx, `
		INSERT INTO credential_vault (
			user_id, provider_id, credential_type, encrypted_data, expires_at
		) VALUES ($1, $2, 'oauth_token', $3, $4)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			encrypted_data = EXCLUDED.encrypted_data,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, userID, ProviderID, token.AccessToken, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	return tx.Commit(ctx)
}

// GetToken retrieves OAuth tokens for a user.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	var token integrations.Token
	var accessToken string
	var expiry time.Time

	err := p.pool.QueryRow(ctx, `
		SELECT encrypted_data, expires_at
		FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID).Scan(&accessToken, &expiry)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	token.AccessToken = accessToken
	token.ExpiresAt = expiry

	return &token, nil
}

// SupportsSync returns true since Notion supports sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a sync operation for the specified resources.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	result := &integrations.SyncResult{
		Success: true,
	}
	start := time.Now()

	// Sync operations are handled by service classes

	result.Duration = time.Since(start)
	return result, nil
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}

// APIRequest makes an authenticated request to the Notion API.
func (p *Provider) APIRequest(ctx context.Context, userID, method, endpoint string, body io.Reader) ([]byte, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, APIURL+endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Notion-Version", APIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
