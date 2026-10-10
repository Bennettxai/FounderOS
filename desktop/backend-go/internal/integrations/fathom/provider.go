// Package fathom provides the Fathom Analytics integration.
// Fathom is a privacy-focused analytics platform that uses API key authentication.
package fathom

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/services"
)

const (
	ProviderID   = "fathom"
	ProviderName = "Fathom Analytics"
	Category     = "analytics"
	BaseAPIURL   = "https://api.usefathom.com/v1"
)

// Provider implements analytics integration for Fathom.
// Note: Fathom uses API key authentication, not OAuth.
type Provider struct {
	pool  *pgxpool.Pool
	vault *services.CredentialVaultService
}

// NewProvider creates a new Fathom provider.
func NewProvider(pool *pgxpool.Pool) *Provider {
	return &Provider{
		pool:  pool,
		vault: services.NewCredentialVaultService(pool),
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
	return "fathom"
}

// GetAuthURL returns empty since Fathom uses API key auth.
func (p *Provider) GetAuthURL(state string) string {
	// Fathom doesn't use OAuth - returns empty
	return ""
}

// ExchangeCode is not used for Fathom (API key auth).
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	return nil, fmt.Errorf("fathom uses API key authentication, not OAuth")
}

// RefreshToken is not applicable for API key auth.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	return nil, fmt.Errorf("fathom uses API key authentication - no refresh needed")
}

// SaveAPIKey stores the API key in the credential vault.
func (p *Provider) SaveAPIKey(ctx context.Context, userID, apiKey string) error {
	// Verify the API key is valid by making a test request
	sites, err := p.fetchSites(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	// Get account info from first site
	accountName := "Fathom Analytics"
	if len(sites) > 0 {
		accountName = fmt.Sprintf("Fathom (%d sites)", len(sites))
	}

	_, err = p.vault.StoreAPIKeyCredential(ctx, services.StoreAPIKeyInput{
		UserID:            userID,
		ProviderID:        ProviderID,
		APIKey:            apiKey,
		ExternalAccountID: userID, // Use userID as account ID
		Metadata: map[string]interface{}{
			"site_count": len(sites),
		},
	})
	if err != nil {
		return err
	}

	// Also update user_integrations
	_, err = p.pool.Exec(ctx, `
		INSERT INTO user_integrations (
			user_id, provider_id, status, connected_at,
			external_account_name, metadata
		) VALUES ($1, $2, 'connected', NOW(), $3, $4)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			status = 'connected',
			connected_at = NOW(),
			external_account_name = EXCLUDED.external_account_name,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, userID, ProviderID, accountName, map[string]interface{}{"site_count": len(sites)})

	return err
}

// SaveToken is a wrapper for SaveAPIKey for interface compatibility.
func (p *Provider) SaveToken(ctx context.Context, userID string, token *integrations.TokenResponse) error {
	return p.SaveAPIKey(ctx, userID, token.AccessToken)
}

// GetToken retrieves the API key for a user.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	cred, err := p.vault.GetCredential(ctx, userID, ProviderID)
	if err != nil {
		return nil, fmt.Errorf("Fathom not connected: %w", err)
	}

	if cred.APIKeyData == nil {
		return nil, fmt.Errorf("invalid Fathom credentials")
	}

	return &integrations.Token{
		AccessToken: cred.APIKeyData.APIKey,
		ExpiresAt:   time.Now().Add(100 * 365 * 24 * time.Hour), // API keys don't expire
	}, nil
}

// Disconnect removes the user's Fathom integration.
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
		Connected:  true,
		AccountID:  cred.ExternalAccountID,
		SyncStatus: "idle",
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
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &integrations.SyncResult{
		Success: true,
	}

	// Sync sites
	if options.Resources == nil || containsString(options.Resources, "sites") {
		sites, err := p.syncSites(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sites sync failed: %v", err))
		} else {
			result.ItemsCreated += sites.Created
			result.ItemsUpdated += sites.Updated
		}
	}

	// Sync aggregations for each site
	if options.Resources == nil || containsString(options.Resources, "aggregations") {
		agg, err := p.syncAggregations(ctx, userID, token.AccessToken)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("aggregations sync failed: %v", err))
		} else {
			result.ItemsCreated += agg.Created
		}
	}

	if len(result.Errors) > 0 && result.ItemsCreated == 0 && result.ItemsUpdated == 0 {
		result.Success = false
	}

	return result, nil
}

// makeRequest is a helper method for making HTTP requests to the Fathom API.
func (p *Provider) makeRequest(ctx context.Context, apiKey, method, endpoint string, params url.Values) ([]byte, error) {
	fullURL := BaseAPIURL + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}
