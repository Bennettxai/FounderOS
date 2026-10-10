package sorx

import (
	"context"
)

// checkIntegrationAccess verifies a user has access to an integration.
func (e *Engine) checkIntegrationAccess(ctx context.Context, userID, provider string) (bool, error) {
	var exists bool
	err := e.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_integrations
			WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
		)
	`, userID, provider).Scan(&exists)
	return exists, err
}

// getCredentials retrieves encrypted credentials for an integration.
func (e *Engine) getCredentials(ctx context.Context, userID, provider string) (*Credentials, error) {
	var creds Credentials
	err := e.pool.QueryRow(ctx, `
		SELECT access_token_encrypted, refresh_token_encrypted, token_expires_at, scopes
		FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
	`, userID, provider).Scan(
		&creds.AccessTokenEncrypted,
		&creds.RefreshTokenEncrypted,
		&creds.ExpiresAt,
		&creds.Scopes,
	)
	if err != nil {
		return nil, err
	}
	creds.Provider = provider
	return &creds, nil
}
