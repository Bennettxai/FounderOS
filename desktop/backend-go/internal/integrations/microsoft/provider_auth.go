package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations"
	"golang.org/x/oauth2"
)

// MicrosoftUser represents basic Microsoft user info.
type MicrosoftUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Email             string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string) (*integrations.TokenResponse, error) {
	token, err := p.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Microsoft Graph
	userInfo, err := p.getUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
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
		AccountEmail: userInfo.Email,
		AccountName:  userInfo.DisplayName,
		Metadata: map[string]interface{}{
			"microsoft_id":    userInfo.ID,
			"microsoft_email": userInfo.Email,
			"microsoft_name":  userInfo.DisplayName,
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
		return &integrations.ConnectionStatus{
			Connected: false,
		}, nil
	}

	return &status, nil
}

// Disconnect removes the user's connection to Microsoft.
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
	`, userID, ProviderID, token.AccountEmail, token.AccountName, token.Scopes, token.Metadata)
	if err != nil {
		return fmt.Errorf("failed to save integration: %w", err)
	}

	// Save to microsoft_oauth_tokens table
	_, err = tx.Exec(ctx, `
		INSERT INTO microsoft_oauth_tokens (
			user_id, access_token, refresh_token, expiry, scopes, microsoft_email
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			microsoft_email = EXCLUDED.microsoft_email,
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
		FROM microsoft_oauth_tokens
		WHERE user_id = $1
	`, userID).Scan(&token.AccessToken, &token.RefreshToken, &expiry, &token.Scopes)

	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	token.ExpiresAt = expiry
	return &token, nil
}

// SupportsSync returns true since Microsoft supports sync.
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
		case "mail":
			if p.HasFeature("mail") {
				// Mail sync will be handled by OutlookService
			}
		case "calendar":
			if p.HasFeature("calendar") {
				// Calendar sync will be handled by OutlookService
			}
		case "files":
			if p.HasFeature("files") {
				// Files sync will be handled by OneDriveService
			}
		case "tasks":
			if p.HasFeature("tasks") {
				// Tasks sync will be handled by ToDoService
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// GetOAuth2Token returns an oauth2.Token for use with Microsoft APIs.
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

// GetHTTPClient returns an HTTP client with auto-refreshing tokens.
func (p *Provider) GetHTTPClient(ctx context.Context, userID string) (*http.Client, error) {
	tokenSource, err := p.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, tokenSource), nil
}

func (p *Provider) getUserInfo(ctx context.Context, token *oauth2.Token) (*MicrosoftUser, error) {
	client := p.oauthConfig.Client(ctx, token)
	resp, err := client.Get(GraphAPIBase + "/me")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo MicrosoftUser
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Use userPrincipalName as email fallback
	if userInfo.Email == "" {
		userInfo.Email = userInfo.UserPrincipalName
	}

	return &userInfo, nil
}

func splitScopes(scopeStr string) []string {
	if scopeStr == "" {
		return nil
	}
	var scopes []string
	for _, s := range splitString(scopeStr, " ") {
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
