// Package hubspot provides the HubSpot CRM integration.
package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
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
	ProviderID   = "hubspot"
	ProviderName = "HubSpot"
	Category     = "crm"
	AuthURL      = "https://app.hubspot.com/oauth/authorize"
	TokenURL     = "https://api.hubapi.com/oauth/v1/token"
	BaseAPIURL   = "https://api.hubapi.com"
)

// OAuth scopes for HubSpot - COMPREHENSIVE access to ALL APIs
// Including CRM, Marketing, Sales, Service, CMS, and more
var DefaultScopes = []string{
	// ============================================================================
	// CRM - CONTACTS
	// ============================================================================
	"crm.objects.contacts.read",
	"crm.objects.contacts.write",
	"crm.schemas.contacts.read",
	"crm.schemas.contacts.write",

	// ============================================================================
	// CRM - COMPANIES
	// ============================================================================
	"crm.objects.companies.read",
	"crm.objects.companies.write",
	"crm.schemas.companies.read",
	"crm.schemas.companies.write",

	// ============================================================================
	// CRM - DEALS
	// ============================================================================
	"crm.objects.deals.read",
	"crm.objects.deals.write",
	"crm.schemas.deals.read",
	"crm.schemas.deals.write",

	// ============================================================================
	// CRM - TICKETS (Service Hub)
	// ============================================================================
	"crm.objects.tickets.read",
	"crm.objects.tickets.write",
	"crm.schemas.tickets.read",
	"crm.schemas.tickets.write",

	// ============================================================================
	// CRM - QUOTES (Sales Hub)
	// ============================================================================
	"crm.objects.quotes.read",
	"crm.objects.quotes.write",
	"crm.schemas.quotes.read",
	"crm.schemas.quotes.write",

	// ============================================================================
	// CRM - LINE ITEMS
	// ============================================================================
	"crm.objects.line_items.read",
	"crm.objects.line_items.write",
	"crm.schemas.line_items.read",
	"crm.schemas.line_items.write",

	// ============================================================================
	// CRM - PRODUCTS
	// ============================================================================
	"crm.objects.products.read",
	"crm.objects.products.write",
	"crm.schemas.products.read",
	"crm.schemas.products.write",

	// ============================================================================
	// CRM - OWNERS & USERS
	// ============================================================================
	"crm.objects.owners.read",
	"crm.objects.users.read",
	"crm.objects.users.write",

	// ============================================================================
	// CRM - ENGAGEMENT (Tasks, Notes, Meetings, Emails, Calls)
	// ============================================================================
	"crm.objects.tasks.read",
	"crm.objects.tasks.write",
	"crm.objects.notes.read",
	"crm.objects.notes.write",
	"crm.objects.meetings.read",
	"crm.objects.meetings.write",
	"crm.objects.emails.read",
	"crm.objects.emails.write",
	"crm.objects.calls.read",
	"crm.objects.calls.write",
	"crm.objects.communications.read",
	"crm.objects.communications.write",

	// ============================================================================
	// CRM - LISTS
	// ============================================================================
	"crm.lists.read",
	"crm.lists.write",

	// ============================================================================
	// CRM - CUSTOM OBJECTS
	// ============================================================================
	"crm.objects.custom.read",
	"crm.objects.custom.write",
	"crm.schemas.custom.read",
	"crm.schemas.custom.write",

	// ============================================================================
	// CRM - FEEDBACK SUBMISSIONS
	// ============================================================================
	"crm.objects.feedback_submissions.read",

	// ============================================================================
	// CRM - GOALS
	// ============================================================================
	"crm.objects.goals.read",

	// ============================================================================
	// CRM - MARKETING EVENTS
	// ============================================================================
	"crm.objects.marketing_events.read",
	"crm.objects.marketing_events.write",

	// ============================================================================
	// CRM - IMPORTS & EXPORTS
	// ============================================================================
	"crm.import",
	"crm.export",

	// ============================================================================
	// TIMELINE - Activity
	// ============================================================================
	"timeline",

	// ============================================================================
	// MARKETING - EMAIL
	// ============================================================================
	"marketing-email",
	"marketing-email.read",
	"marketing-email.write",

	// ============================================================================
	// MARKETING - FORMS
	// ============================================================================
	"forms",
	"forms.read",
	"forms.write",
	"forms-uploaded-files",

	// ============================================================================
	// MARKETING - CAMPAIGNS
	// ============================================================================
	"campaigns",
	"campaigns.read",
	"campaigns.write",

	// ============================================================================
	// MARKETING - AUTOMATION (Workflows)
	// ============================================================================
	"automation",
	"automation.read",
	"automation.write",

	// ============================================================================
	// MARKETING - ANALYTICS
	// ============================================================================
	"analytics.behavioral_events.send",

	// ============================================================================
	// MARKETING - TRANSACTIONAL EMAIL
	// ============================================================================
	"transactional-email",

	// ============================================================================
	// SALES - REPORTS
	// ============================================================================
	"sales-email-read",

	// ============================================================================
	// CONVERSATIONS - Inbox & Chat
	// ============================================================================
	"conversations.read",
	"conversations.write",
	"conversations.visitor_identification.tokens.create",

	// ============================================================================
	// CMS - CONTENT
	// ============================================================================
	"content",

	// ============================================================================
	// CMS - BLOG
	// ============================================================================
	"blog.read",
	"blog.write",

	// ============================================================================
	// CMS - SITE PAGES
	// ============================================================================
	"pages.read",
	"pages.write",

	// ============================================================================
	// CMS - LANDING PAGES
	// ============================================================================
	"landing-pages.read",
	"landing-pages.write",

	// ============================================================================
	// CMS - TEMPLATES & THEMES
	// ============================================================================
	"templates.read",
	"templates.write",

	// ============================================================================
	// CMS - HubDB (Database Tables)
	// ============================================================================
	"hubdb",

	// ============================================================================
	// CMS - FILES (File Manager)
	// ============================================================================
	"files",
	"files.read",
	"files.write",
	"files.ui_hidden.read",

	// ============================================================================
	// CMS - URL REDIRECTS
	// ============================================================================
	"url-redirects.read",
	"url-redirects.write",

	// ============================================================================
	// CMS - DOMAINS
	// ============================================================================
	"domains.read",
	"domains.write",

	// ============================================================================
	// SETTINGS - ACCOUNT
	// ============================================================================
	"settings.billing.read",
	"settings.currencies.read",
	"settings.currencies.write",
	"settings.users.read",
	"settings.users.write",
	"settings.users.teams.read",
	"settings.users.teams.write",

	// ============================================================================
	// INTEGRATIONS
	// ============================================================================
	"integration-sync",
	"integration-sync.read",
	"integration-sync.write",

	// ============================================================================
	// E-COMMERCE
	// ============================================================================
	"e-commerce",

	// ============================================================================
	// TICKETS PIPELINE
	// ============================================================================
	"tickets",

	// ============================================================================
	// OAUTH
	// ============================================================================
	"oauth",

	// ============================================================================
	// ACCOUNT INFO
	// ============================================================================
	"account-info.security.read",

	// ============================================================================
	// COLLECTOR (Data Collection)
	// ============================================================================
	"collector.graphql_query.execute",
	"collector.graphql_schema.read",

	// ============================================================================
	// BUSINESS INTELLIGENCE
	// ============================================================================
	"business-intelligence",

	// ============================================================================
	// COMMUNICATIONS (Subscriptions)
	// ============================================================================
	"communication_preferences.read",
	"communication_preferences.read_write",
	"communication_preferences.write",

	// ============================================================================
	// MEDIA BRIDGE (External Media)
	// ============================================================================
	"media_bridge.read",
	"media_bridge.write",

	// ============================================================================
	// CRM DATA QUALITY
	// ============================================================================
	"crm.data_quality.read",
}

// AllHubSpotScopes contains all HubSpot scopes
var AllHubSpotScopes []string

func init() {
	AllHubSpotScopes = append(AllHubSpotScopes, DefaultScopes...)
}

// Provider implements the integrations.Provider interface for HubSpot.
type Provider struct {
	pool         *pgxpool.Pool
	vault        *services.CredentialVaultService
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

// NewProvider creates a new HubSpot provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	cfg := config.AppConfig

	return &Provider{
		pool:         pool,
		vault:        services.NewCredentialVaultService(pool),
		clientID:     cfg.HubSpotClientID,
		clientSecret: cfg.HubSpotClientSecret,
		redirectURI:  cfg.HubSpotRedirectURI,
		scopes:       DefaultScopes,
	}
}

// Pool returns the database connection pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
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
	return "hubspot"
}

// GetAuthURL generates the OAuth authorization URL.
func (p *Provider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("scope", strings.Join(p.scopes, " "))
	params.Set("state", state)

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

	resp, err := http.PostForm(TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Get user info to identify the account
	accountInfo, err := p.getAccountInfo(tokenResp.AccessToken)
	if err != nil {
		// Log but don't fail
		accountInfo = &hubSpotAccountInfo{}
	}

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		AccountID:    fmt.Sprintf("%d", accountInfo.PortalID),
		AccountName:  accountInfo.HubDomain,
		AccountEmail: accountInfo.Email,
		Scopes:       p.scopes,
	}, nil
}

// RefreshToken refreshes an expired access token.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm(TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &integrations.TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       p.scopes,
	}, nil
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

// GetToken retrieves and refreshes tokens if needed.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	cred, err := p.vault.GetCredential(ctx, userID, ProviderID)
	if err != nil {
		return nil, fmt.Errorf("HubSpot not connected: %w", err)
	}

	if cred.OAuthData == nil {
		return nil, fmt.Errorf("invalid HubSpot credentials")
	}

	// Check if token needs refresh
	if cred.ExpiresAt != nil && time.Now().Add(5*time.Minute).After(*cred.ExpiresAt) {
		// Refresh the token
		newToken, err := p.RefreshToken(ctx, cred.OAuthData.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Save the new token
		newToken.AccountID = cred.ExternalAccountID
		newToken.AccountEmail = cred.ExternalAccountEmail
		if err := p.SaveToken(ctx, userID, newToken); err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}

		return &integrations.Token{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
			ExpiresAt:    newToken.ExpiresAt,
			Scopes:       newToken.Scopes,
		}, nil
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

// Disconnect removes the user's HubSpot integration.
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

// SupportsSync returns whether this provider supports data sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a full data sync for the user.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	// Get access token
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &integrations.SyncResult{
		Success: true,
	}

	// Sync contacts
	if options.Resources == nil || containsString(options.Resources, "contacts") {
		contacts, err := p.syncContacts(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("contacts sync failed: %v", err))
		} else {
			result.ItemsCreated += contacts.Created
			result.ItemsUpdated += contacts.Updated
		}
	}

	// Sync companies
	if options.Resources == nil || containsString(options.Resources, "companies") {
		companies, err := p.syncCompanies(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("companies sync failed: %v", err))
		} else {
			result.ItemsCreated += companies.Created
			result.ItemsUpdated += companies.Updated
		}
	}

	// Sync deals
	if options.Resources == nil || containsString(options.Resources, "deals") {
		deals, err := p.syncDeals(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("deals sync failed: %v", err))
		} else {
			result.ItemsCreated += deals.Created
			result.ItemsUpdated += deals.Updated
		}
	}

	if len(result.Errors) > 0 && result.ItemsCreated == 0 && result.ItemsUpdated == 0 {
		result.Success = false
	}

	return result, nil
}

// ============================================================================
// Account Info
// ============================================================================

// hubSpotAccountInfo represents account information from HubSpot.
type hubSpotAccountInfo struct {
	PortalID  int64  `json:"portalId"`
	HubDomain string `json:"hub_domain"`
	Email     string `json:"user"`
}

// getAccountInfo retrieves account information from HubSpot.
func (p *Provider) getAccountInfo(accessToken string) (*hubSpotAccountInfo, error) {
	req, err := http.NewRequest("GET", BaseAPIURL+"/oauth/v1/access-tokens/"+accessToken, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info hubSpotAccountInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// ============================================================================
// Register Provider
// ============================================================================

// Note: Provider registration should happen in main.go or handlers initialization
// where config.AppConfig and database pool are available.
