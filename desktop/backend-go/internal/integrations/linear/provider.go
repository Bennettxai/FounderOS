// Package linear provides the Linear project management integration.
package linear

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
	"github.com/rhl/businessos-backend/internal/services"
)

const (
	ProviderID   = "linear"
	ProviderName = "Linear"
	Category     = "project_management"
	AuthURL      = "https://linear.app/oauth/authorize"
	TokenURL     = "https://api.linear.app/oauth/token"
	GraphQLURL   = "https://api.linear.app/graphql"
)

// OAuth scopes for Linear
var DefaultScopes = []string{
	"read",  // Read access to issues, projects, teams
	"write", // Write access to create/update issues
}

// Provider implements the integrations.Provider interface for Linear.
type Provider struct {
	pool         *pgxpool.Pool
	vault        *services.CredentialVaultService
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

// NewProvider creates a new Linear provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	cfg := config.AppConfig

	return &Provider{
		pool:         pool,
		vault:        services.NewCredentialVaultService(pool),
		clientID:     cfg.LinearClientID,
		clientSecret: cfg.LinearClientSecret,
		redirectURI:  cfg.LinearRedirectURI,
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

// Icon returns the icon identifier.
func (p *Provider) Icon() string {
	return "linear"
}

// ============================================================================
// OAuth Methods
// ============================================================================

// GetAuthURL generates the OAuth authorization URL.
func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("scope", strings.Join(p.scopes, ","))
	params.Set("state", state)
	params.Set("response_type", "code")
	params.Set("prompt", "consent")

	return AuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("redirect_uri", p.redirectURI)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user/organization info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		// Log but don't fail
		userInfo = &linearUserInfo{}
	}

	// Parse scopes
	scopes := strings.Split(tokenResp.Scope, ",")

	// Linear tokens don't expire (but we'll set a long expiry)
	expiresAt := time.Now().Add(365 * 24 * time.Hour)
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: "", // Linear doesn't use refresh tokens
		ExpiresAt:    expiresAt,
		AccountID:    userInfo.OrganizationID,
		AccountName:  userInfo.OrganizationName,
		AccountEmail: userInfo.Email,
		Scopes:       scopes,
		Metadata: map[string]interface{}{
			"user_id":      userInfo.UserID,
			"user_name":    userInfo.UserName,
			"organization": userInfo.OrganizationName,
		},
	}, nil
}

// RefreshToken refreshes an expired access token.
// Note: Linear uses long-lived tokens that don't typically need refresh.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	// Linear doesn't support refresh tokens - tokens are long-lived
	return nil, fmt.Errorf("linear tokens do not support refresh - user must re-authenticate")
}

// SaveToken stores the tokens in the credential vault.
func (p *Provider) SaveToken(ctx context.Context, userID string, token *integrations.TokenResponse) error {
	_, err := p.vault.StoreOAuthCredential(ctx, services.StoreOAuthInput{
		UserID:               userID,
		ProviderID:           ProviderID,
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		ExpiresAt:            &token.ExpiresAt,
		ExternalAccountID:    token.AccountID,
		ExternalAccountEmail: token.AccountEmail,
		Scopes:               token.Scopes,
	})
	return err
}

// GetToken retrieves tokens for a user.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	cred, err := p.vault.GetCredential(ctx, userID, ProviderID)
	if err != nil {
		return nil, fmt.Errorf("Linear not connected: %w", err)
	}

	if cred.OAuthData == nil {
		return nil, fmt.Errorf("invalid Linear credentials")
	}

	expiresAt := time.Now().Add(time.Hour)
	if cred.ExpiresAt != nil {
		expiresAt = *cred.ExpiresAt
	}

	return &integrations.Token{
		AccessToken:  cred.OAuthData.AccessToken,
		RefreshToken: cred.OAuthData.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       cred.Scopes,
	}, nil
}

// ============================================================================
// Connection Methods
// ============================================================================

// Disconnect removes the user's Linear integration.
func (p *Provider) Disconnect(ctx context.Context, userID string) error {
	return p.vault.DeleteCredential(ctx, userID, ProviderID)
}

// GetConnectionStatus returns the current connection status.
func (p *Provider) GetConnectionStatus(ctx context.Context, userID string) (*integrations.ConnectionStatus, error) {
	cred, err := p.vault.GetCredential(ctx, userID, ProviderID)
	if err != nil {
		return &integrations.ConnectionStatus{Connected: false}, nil
	}

	status := &integrations.ConnectionStatus{
		Connected:    true,
		AccountID:    cred.ExternalAccountID,
		AccountEmail: cred.ExternalAccountEmail,
		Scopes:       cred.Scopes,
		SyncStatus:   "idle",
	}

	if !cred.CreatedAt.IsZero() {
		status.ConnectedAt = &cred.CreatedAt
	}
	if cred.LastUsedAt != nil {
		status.LastSyncAt = cred.LastUsedAt
	}

	return status, nil
}

// ============================================================================
// Sync Methods
// ============================================================================

// SupportsSync returns whether this provider supports data sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a full data sync for the user.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &integrations.SyncResult{
		Success: true,
	}

	// Sync issues
	if options.Resources == nil || containsString(options.Resources, "issues") {
		issues, err := p.syncIssues(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("issues sync failed: %v", err))
		} else {
			result.ItemsCreated += issues.Created
			result.ItemsUpdated += issues.Updated
		}
	}

	// Sync projects
	if options.Resources == nil || containsString(options.Resources, "projects") {
		projects, err := p.syncProjects(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("projects sync failed: %v", err))
		} else {
			result.ItemsCreated += projects.Created
			result.ItemsUpdated += projects.Updated
		}
	}

	// Sync teams
	if options.Resources == nil || containsString(options.Resources, "teams") {
		teams, err := p.syncTeams(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("teams sync failed: %v", err))
		} else {
			result.ItemsCreated += teams.Created
			result.ItemsUpdated += teams.Updated
		}
	}

	if len(result.Errors) > 0 && result.ItemsCreated == 0 && result.ItemsUpdated == 0 {
		result.Success = false
	}

	return result, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}

// UpdateIssue updates an existing Linear issue.
func (p *Provider) UpdateIssue(ctx context.Context, userID, issueID string, input map[string]interface{}) (*Issue, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	mutation := `
		mutation($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					state {
						name
					}
					assignee {
						name
					}
					project {
						name
					}
					team {
						name
					}
					dueDate
					createdAt
					updatedAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id":    issueID,
		"input": input,
	}

	resp, err := p.executeGraphQL(ctx, token.AccessToken, mutation, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to execute graphql mutation: %w", err)
	}

	var data struct {
		IssueUpdate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Priority    int    `json:"priority"`
				State       struct {
					Name string `json:"name"`
				} `json:"state"`
				Assignee *struct {
					Name string `json:"name"`
				} `json:"assignee"`
				Project *struct {
					Name string `json:"name"`
				} `json:"project"`
				Team struct {
					Name string `json:"name"`
				} `json:"team"`
				DueDate   *time.Time `json:"dueDate"`
				CreatedAt time.Time  `json:"createdAt"`
				UpdatedAt time.Time  `json:"updatedAt"`
			} `json:"issue"`
		} `json:"issueUpdate"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !data.IssueUpdate.Success {
		return nil, fmt.Errorf("issue update failed")
	}

	issue := &Issue{
		ID:          data.IssueUpdate.Issue.ID,
		Identifier:  data.IssueUpdate.Issue.Identifier,
		Title:       data.IssueUpdate.Issue.Title,
		Description: data.IssueUpdate.Issue.Description,
		State:       data.IssueUpdate.Issue.State.Name,
		Priority:    data.IssueUpdate.Issue.Priority,
		Team:        data.IssueUpdate.Issue.Team.Name,
		DueDate:     data.IssueUpdate.Issue.DueDate,
		CreatedAt:   data.IssueUpdate.Issue.CreatedAt,
		UpdatedAt:   data.IssueUpdate.Issue.UpdatedAt,
	}

	if data.IssueUpdate.Issue.Assignee != nil {
		issue.Assignee = data.IssueUpdate.Issue.Assignee.Name
	}

	if data.IssueUpdate.Issue.Project != nil {
		issue.Project = data.IssueUpdate.Issue.Project.Name
	}

	return issue, nil
}
