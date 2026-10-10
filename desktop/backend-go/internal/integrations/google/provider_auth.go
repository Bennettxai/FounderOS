package google

import (
	"context"
	"fmt"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations"
	"golang.org/x/oauth2"
)

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	token, err := p.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	email, err := p.getUserEmail(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}

	// Extract scopes
	var scopes []string
	if scopeStr, ok := token.Extra("scope").(string); ok {
		scopes = splitScopes(scopeStr)
	}

	return &integrations.TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
		Scopes:       scopes,
		AccountEmail: email,
		AccountName:  email, // Use email as name for now
		Metadata: map[string]interface{}{
			"google_email": email,
		},
	}, nil
}

// RefreshToken refreshes an expired access token.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*integrations.TokenResponse, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := p.oauthConfig.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &integrations.TokenResponse{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

// GetConnectionStatus returns the connection status for a user.
func (p *Provider) GetConnectionStatus(ctx context.Context, userID string) (*integrations.ConnectionStatus, error) {
	// Query the database for connection status
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
		// Not connected
		return &integrations.ConnectionStatus{
			Connected: false,
		}, nil
	}

	return &status, nil
}

// Disconnect removes the user's connection to Google.
func (p *Provider) Disconnect(ctx context.Context, userID string) error {
	// Delete from user_integrations
	_, err := p.pool.Exec(ctx, `
		DELETE FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID)
	if err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	// Delete from credential_vault
	_, err = p.pool.Exec(ctx, `
		DELETE FROM credential_vault
		WHERE user_id = $1 AND provider_id = $2
	`, userID, ProviderID)
	if err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

// SaveToken saves OAuth tokens to the credential vault.
func (p *Provider) SaveToken(ctx context.Context, userID string, token *integrations.TokenResponse) error {
	// Start transaction
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
	`, userID, ProviderID, token.AccountEmail, token.AccountName, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	// Also save to legacy google_oauth_tokens for backward compatibility
	_, err = tx.Exec(ctx, `
		INSERT INTO google_oauth_tokens (
			user_id, access_token, refresh_token, expiry, scopes, google_email
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			google_email = EXCLUDED.google_email,
			updated_at = NOW()
	`, userID, token.AccessToken, token.RefreshToken, token.ExpiresAt, token.Scopes, token.AccountEmail)
	if err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return tx.Commit(ctx)
}

// GetToken retrieves OAuth tokens for a user.
func (p *Provider) GetToken(ctx context.Context, userID string) (*integrations.Token, error) {
	var token integrations.Token
	var expiry time.Time

	err := p.pool.QueryRow(ctx, `
		SELECT access_token, refresh_token, expiry, scopes
		FROM google_oauth_tokens
		WHERE user_id = $1
	`, userID).Scan(&token.AccessToken, &token.RefreshToken, &expiry, &token.Scopes)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	token.ExpiresAt = expiry
	return &token, nil
}

// SupportsSync returns true since Google supports sync.
func (p *Provider) SupportsSync() bool {
	return true
}

// Sync performs a sync operation for the specified resources.
func (p *Provider) Sync(ctx context.Context, userID string, options integrations.SyncOptions) (*integrations.SyncResult, error) {
	result := &integrations.SyncResult{
		Success: true,
	}
	start := time.Now()

	for _, resource := range options.Resources {
		switch resource {
		case "calendar":
			if p.HasFeature("calendar") {
				// Calendar sync will be handled by CalendarService
			}
		case "gmail":
			if p.HasFeature("gmail") {
				// Gmail sync will be handled by GmailService
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// GetOAuth2Token returns an oauth2.Token for use with Google APIs.
func (p *Provider) GetOAuth2Token(ctx context.Context, userID string) (*oauth2.Token, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.ExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// GetTokenSource returns a token source that auto-refreshes.
func (p *Provider) GetTokenSource(ctx context.Context, userID string) (oauth2.TokenSource, error) {
	token, err := p.GetOAuth2Token(ctx, userID)
	if err != nil {
		return nil, err
	}
	return p.oauthConfig.TokenSource(ctx, token), nil
}

func (p *Provider) getUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.oauthConfig.Client(ctx, token)
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

func splitScopes(scopeStr string) []string {
	if scopeStr == "" {
		return nil
	}
	// Scopes are space-separated
	var scopes []string
	for _, s := range split(scopeStr, " ") {
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

func split(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
