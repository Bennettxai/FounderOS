// Package slack provides the Slack integration (messages, channels, notifications).
package slack

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
	ProviderID   = "slack"
	ProviderName = "Slack"
	Category     = "communication"
)

// OAuth endpoints
const (
	AuthURL  = "https://slack.com/oauth/v2/authorize"
	TokenURL = "https://slack.com/api/oauth.v2.access"
)

// Scopes for Slack - COMPREHENSIVE access to workspace data
// Including ALL available user scopes for maximum API access
var DefaultScopes = []string{
	// ============================================================================
	// CHANNELS - Public channels
	// ============================================================================
	"channels:read",    // View basic channel info
	"channels:history", // Read channel messages
	"channels:join",    // Join public channels
	"channels:write",   // Create/archive/rename channels
	"channels:manage",  // Manage channel settings

	// ============================================================================
	// DIRECT MESSAGES (IMs)
	// ============================================================================
	"im:read",    // View direct messages info
	"im:history", // Read direct messages
	"im:write",   // Send direct messages

	// ============================================================================
	// PRIVATE CHANNELS (Groups)
	// ============================================================================
	"groups:read",    // View private channels
	"groups:history", // Read private channel messages
	"groups:write",   // Manage private channels

	// ============================================================================
	// MULTI-PARTY DIRECT MESSAGES (MPIMs)
	// ============================================================================
	"mpim:read",    // View multi-party DMs
	"mpim:history", // Read multi-party DM messages
	"mpim:write",   // Send multi-party DMs

	// ============================================================================
	// CHAT & MESSAGES
	// ============================================================================
	"chat:write",           // Send messages as user
	"chat:write.public",    // Send to channels without joining
	"chat:write.customize", // Send with custom username/avatar

	// ============================================================================
	// USERS & TEAM
	// ============================================================================
	"users:read",            // View users
	"users:read.email",      // View user email addresses
	"users.profile:read",    // View user profile details
	"users.profile:write",   // Edit user profile
	"users:write",           // Modify user data
	"team:read",             // View workspace info
	"team.billing:read",     // View billing info
	"team.preferences:read", // View team preferences

	// ============================================================================
	// FILES & ATTACHMENTS
	// ============================================================================
	"files:read",  // View files shared in channels
	"files:write", // Upload and edit files

	// ============================================================================
	// REACTIONS & INTERACTIONS
	// ============================================================================
	"reactions:read",  // View emoji reactions
	"reactions:write", // Add/remove reactions

	// ============================================================================
	// PINS, BOOKMARKS, STARS
	// ============================================================================
	"pins:read",       // View pinned messages
	"pins:write",      // Pin/unpin messages
	"bookmarks:read",  // View saved items
	"bookmarks:write", // Save items
	"stars:read",      // View starred items
	"stars:write",     // Star/unstar items

	// ============================================================================
	// SEARCH
	// ============================================================================
	"search:read", // Search messages and files

	// ============================================================================
	// REMINDERS
	// ============================================================================
	"reminders:read",  // View reminders
	"reminders:write", // Create/edit reminders

	// ============================================================================
	// USER GROUPS
	// ============================================================================
	"usergroups:read",  // View user groups (teams)
	"usergroups:write", // Manage user groups

	// ============================================================================
	// CALLS & VOICE
	// ============================================================================
	"calls:read",  // View call info
	"calls:write", // Start/manage calls

	// ============================================================================
	// DO NOT DISTURB
	// ============================================================================
	"dnd:read",  // View Do Not Disturb settings
	"dnd:write", // Set Do Not Disturb

	// ============================================================================
	// EMOJI
	// ============================================================================
	"emoji:read", // View custom emoji

	// ============================================================================
	// IDENTITY (OAuth)
	// ============================================================================
	"identity.basic",  // View user's basic info
	"identity.email",  // View user's email
	"identity.avatar", // View user's avatar
	"identity.team",   // View user's team

	// ============================================================================
	// CANVASES (New Feature)
	// ============================================================================
	"canvases:read",  // View canvases
	"canvases:write", // Edit canvases

	// ============================================================================
	// WORKFLOWS & AUTOMATION
	// ============================================================================
	"workflow.steps:execute", // Execute workflow steps
	"triggers:read",          // View triggers
	"triggers:write",         // Create triggers

	// ============================================================================
	// LINKS
	// ============================================================================
	"links:read",  // View URL info
	"links:write", // Unfurl URLs

	// ============================================================================
	// METADATA
	// ============================================================================
	"metadata.message:read", // Read message metadata

	// ============================================================================
	// ASSISTANT (Slack AI)
	// ============================================================================
	"assistant:write", // Interact as an assistant

	// ============================================================================
	// CONNECTIONS (External)
	// ============================================================================
	"connections:write", // Manage external connections

	// ============================================================================
	// APP HOME
	// ============================================================================
	"app_mentions:read", // View app mentions

	// ============================================================================
	// OPENID
	// ============================================================================
	"openid",  // OpenID Connect
	"profile", // View profile info
	"email",   // View email
}

// AdminScopes contains Slack admin scopes (requires admin privileges)
// These are separate because they require Enterprise Grid
var AdminScopes = []string{
	"admin",                     // Full admin access
	"admin.analytics:read",      // View analytics
	"admin.apps:read",           // View apps
	"admin.apps:write",          // Manage apps
	"admin.barriers:read",       // View info barriers
	"admin.barriers:write",      // Manage info barriers
	"admin.conversations:read",  // View conversations
	"admin.conversations:write", // Manage conversations
	"admin.invites:read",        // View invite requests
	"admin.invites:write",       // Manage invites
	"admin.roles:read",          // View roles
	"admin.roles:write",         // Manage roles
	"admin.teams:read",          // View teams
	"admin.teams:write",         // Manage teams
	"admin.usergroups:read",     // View user groups
	"admin.usergroups:write",    // Manage user groups
	"admin.users:read",          // View users
	"admin.users:write",         // Manage users
	"admin.workflows:read",      // View workflows
	"admin.workflows:write",     // Manage workflows
}

// AllSlackScopes contains ALL Slack scopes combined
var AllSlackScopes []string

func init() {
	AllSlackScopes = append(AllSlackScopes, DefaultScopes...)
	AllSlackScopes = append(AllSlackScopes, AdminScopes...)
}

// Provider implements the integrations.Provider interface for Slack.
type Provider struct {
	pool         *pgxpool.Pool
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

// NewProvider creates a new Slack provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	cfg := config.AppConfig

	redirectURI := cfg.SlackRedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s/api/integrations/slack/callback", cfg.BaseURL)
	}

	return &Provider{
		pool:         pool,
		clientID:     cfg.SlackClientID,
		clientSecret: cfg.SlackClientSecret,
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
	return "/icons/slack.svg"
}

// GetAuthURL returns the OAuth authorization URL.
func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("scope", strings.Join(p.scopes, ","))
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", AuthURL, params.Encode())
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", p.redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		BotUserID   string `json:"bot_user_id"`
		AppID       string `json:"app_id"`
		Team        struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		AuthedUser struct {
			ID          string `json:"id"`
			Scope       string `json:"scope"`
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		} `json:"authed_user"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if !tokenResp.OK {
		return nil, fmt.Errorf("slack oauth error: %s", tokenResp.Error)
	}

	// Use bot token if available, otherwise user token
	accessToken := tokenResp.AccessToken
	if accessToken == "" {
		accessToken = tokenResp.AuthedUser.AccessToken
	}

	// Parse scopes
	scopes := strings.Split(tokenResp.Scope, ",")

	return &integrations.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: "",                           // Slack doesn't use refresh tokens
		ExpiresAt:    time.Now().AddDate(10, 0, 0), // Slack tokens don't expire
		Scopes:       scopes,
		AccountID:    tokenResp.Team.ID,
		AccountName:  tokenResp.Team.Name,
		Metadata: map[string]interface{}{
			"team_id":     tokenResp.Team.ID,
			"team_name":   tokenResp.Team.Name,
			"bot_user_id": tokenResp.BotUserID,
			"app_id":      tokenResp.AppID,
			"user_id":     tokenResp.AuthedUser.ID,
		},
	}, nil
}

// RefreshToken refreshes an expired access token.
// Slack tokens don't expire, so this is a no-op.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	// Slack tokens don't expire
	return nil, fmt.Errorf("slack tokens do not expire")
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

// Disconnect removes the user's connection to Slack.
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
	`, userID, ProviderID, token.AccountID, token.AccountName, token.Scopes, token.Metadata)
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

	// Get scopes from user_integrations
	p.pool.QueryRow(ctx, `
		SELECT scopes FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID).Scan(&token.Scopes)

	return &token, nil
}

// SupportsSync returns true since Slack supports sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a sync operation for the specified resources.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	start := time.Now()

	return &integrations.SyncResult{
		Success:  false,
		Duration: time.Since(start),
		Errors:   []string{"slack provider sync is not implemented; use the Slack service sync path"},
	}, fmt.Errorf("slack provider sync is not implemented")
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}
