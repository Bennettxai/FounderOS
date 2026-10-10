// Package services provides business logic for the application
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/security"
)

// OAuthTokenData represents the decrypted OAuth token data stored in the vault
type OAuthTokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	// For Slack - bot tokens
	BotToken  string `json:"bot_token,omitempty"`
	UserToken string `json:"user_token,omitempty"`
}

// APIKeyData represents the decrypted API key data stored in the vault
type APIKeyData struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret,omitempty"`
}

// CredentialType constants
const (
	CredentialTypeOAuth  = "oauth"
	CredentialTypeAPIKey = "api_key"
	CredentialTypeCustom = "custom"
)

// Provider ID constants - matches integration_providers table
const (
	ProviderGoogle     = "google"
	ProviderSlack      = "slack"
	ProviderNotion     = "notion"
	ProviderHubSpot    = "hubspot"
	ProviderLinear     = "linear"
	ProviderGitHub     = "github"
	ProviderDropbox    = "dropbox"
	ProviderZoom       = "zoom"
	ProviderStripe     = "stripe"
	ProviderQuickBooks = "quickbooks"
)

// Credential represents a decrypted credential from the vault
type Credential struct {
	ID                    string
	UserID                string
	ProviderID            string
	CredentialType        string
	ExpiresAt             *time.Time
	ExternalAccountID     string
	ExternalAccountEmail  string
	ExternalWorkspaceID   string
	ExternalWorkspaceName string
	Scopes                []string
	Metadata              map[string]interface{}
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastUsedAt            *time.Time
	LastRotatedAt         *time.Time

	// Decrypted data (not stored, populated on read)
	OAuthData  *OAuthTokenData
	APIKeyData *APIKeyData
	RawData    string // For custom types
}

// StoreOAuthInput contains the data needed to store OAuth credentials
type StoreOAuthInput struct {
	UserID                string
	ProviderID            string
	AccessToken           string
	RefreshToken          string
	TokenType             string
	ExpiresAt             *time.Time
	ExternalAccountID     string
	ExternalAccountEmail  string
	ExternalWorkspaceID   string
	ExternalWorkspaceName string
	Scopes                []string
	Metadata              map[string]interface{}
	// For Slack
	BotToken  string
	UserToken string
}

// StoreAPIKeyInput contains the data needed to store API key credentials
type StoreAPIKeyInput struct {
	UserID               string
	ProviderID           string
	APIKey               string
	APISecret            string
	ExternalAccountID    string
	ExternalAccountEmail string
	Metadata             map[string]interface{}
}

// CredentialVaultService handles secure credential storage and retrieval
type CredentialVaultService struct {
	pool       *pgxpool.Pool
	encryption *security.TokenEncryption
}

// NewCredentialVaultService creates a new credential vault service
func NewCredentialVaultService(pool *pgxpool.Pool) *CredentialVaultService {
	return &CredentialVaultService{
		pool:       pool,
		encryption: security.GetGlobalEncryption(),
	}
}

// StoreOAuthCredential stores OAuth tokens securely in the vault
func (s *CredentialVaultService) StoreOAuthCredential(ctx context.Context, input StoreOAuthInput) (*Credential, error) {
	// Build token data
	tokenData := OAuthTokenData{
		AccessToken:  input.AccessToken,
		RefreshToken: input.RefreshToken,
		TokenType:    input.TokenType,
		BotToken:     input.BotToken,
		UserToken:    input.UserToken,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize token data: %w", err)
	}

	// Encrypt the JSON
	encryptedData, err := s.encryptData(string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token data: %w", err)
	}

	// Prepare metadata
	var metadataJSON []byte
	if input.Metadata != nil {
		metadataJSON, err = json.Marshal(input.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize metadata: %w", err)
		}
	} else {
		metadataJSON = nil
	}

	// Prepare expiry
	var expiresAt pgtype.Timestamptz
	if input.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *input.ExpiresAt, Valid: true}
	}

	// Get encryption version
	encVersion := int32(1)

	queries := sqlc.New(s.pool)
	result, err := queries.StoreCredential(ctx, sqlc.StoreCredentialParams{
		UserID:                input.UserID,
		ProviderID:            input.ProviderID,
		CredentialType:        CredentialTypeOAuth,
		EncryptedData:         encryptedData,
		EncryptionVersion:     &encVersion,
		ExpiresAt:             expiresAt,
		ExternalAccountID:     nilIfEmpty(input.ExternalAccountID),
		ExternalAccountEmail:  nilIfEmpty(input.ExternalAccountEmail),
		ExternalWorkspaceID:   nilIfEmpty(input.ExternalWorkspaceID),
		ExternalWorkspaceName: nilIfEmpty(input.ExternalWorkspaceName),
		Scopes:                input.Scopes,
		Metadata:              metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return s.dbCredentialToCredential(result, &tokenData, nil)
}

// StoreAPIKeyCredential stores API key credentials securely in the vault
func (s *CredentialVaultService) StoreAPIKeyCredential(ctx context.Context, input StoreAPIKeyInput) (*Credential, error) {
	// Build key data
	keyData := APIKeyData{
		APIKey:    input.APIKey,
		APISecret: input.APISecret,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize key data: %w", err)
	}

	// Encrypt the JSON
	encryptedData, err := s.encryptData(string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key data: %w", err)
	}

	// Prepare metadata
	var metadataJSON []byte
	if input.Metadata != nil {
		metadataJSON, err = json.Marshal(input.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize metadata: %w", err)
		}
	} else {
		metadataJSON = nil
	}

	encVersion := int32(1)

	queries := sqlc.New(s.pool)
	result, err := queries.StoreCredential(ctx, sqlc.StoreCredentialParams{
		UserID:               input.UserID,
		ProviderID:           input.ProviderID,
		CredentialType:       CredentialTypeAPIKey,
		EncryptedData:        encryptedData,
		EncryptionVersion:    &encVersion,
		ExternalAccountID:    nilIfEmpty(input.ExternalAccountID),
		ExternalAccountEmail: nilIfEmpty(input.ExternalAccountEmail),
		Scopes:               []string{},
		Metadata:             metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return s.dbCredentialToCredential(result, nil, &keyData)
}

// GetCredential retrieves and decrypts a credential from the vault
func (s *CredentialVaultService) GetCredential(ctx context.Context, userID, providerID string) (*Credential, error) {
	queries := sqlc.New(s.pool)

	result, err := queries.GetCredential(ctx, sqlc.GetCredentialParams{
		UserID:     userID,
		ProviderID: providerID,
	})
	if err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}

	// Update last used timestamp (async, don't fail on error)
	go func() {
		_ = queries.UpdateCredentialLastUsed(context.Background(), sqlc.UpdateCredentialLastUsedParams{
			UserID:     userID,
			ProviderID: providerID,
		})
	}()

	return s.decryptCredential(result)
}

// GetOAuthToken is a convenience method that returns just the OAuth token data
// This is useful for integrating with existing code that expects oauth2.Token
func (s *CredentialVaultService) GetOAuthToken(ctx context.Context, userID, providerID string) (*OAuthTokenData, error) {
	cred, err := s.GetCredential(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	if cred.OAuthData == nil {
		return nil, errors.New("credential is not OAuth type")
	}

	return cred.OAuthData, nil
}

// GetAPIKey is a convenience method that returns just the API key data
func (s *CredentialVaultService) GetAPIKey(ctx context.Context, userID, providerID string) (*APIKeyData, error) {
	cred, err := s.GetCredential(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	if cred.APIKeyData == nil {
		return nil, errors.New("credential is not API key type")
	}

	return cred.APIKeyData, nil
}

// GetUserCredentials retrieves all credentials for a user
func (s *CredentialVaultService) GetUserCredentials(ctx context.Context, userID string) ([]*Credential, error) {
	queries := sqlc.New(s.pool)

	results, err := queries.GetCredentialsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	credentials := make([]*Credential, 0, len(results))
	for _, r := range results {
		cred, err := s.decryptCredential(r)
		if err != nil {
			// Log error but continue - don't fail all if one is corrupted
			slog.Warn("failed to decrypt credential", "provider_id", r.ProviderID, "error", err)
			continue
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// CredentialExists checks if a credential exists for the user/provider combination
func (s *CredentialVaultService) CredentialExists(ctx context.Context, userID, providerID string) (bool, error) {
	queries := sqlc.New(s.pool)
	return queries.CredentialExists(ctx, sqlc.CredentialExistsParams{
		UserID:     userID,
		ProviderID: providerID,
	})
}

// DeleteCredential removes a credential from the vault
func (s *CredentialVaultService) DeleteCredential(ctx context.Context, userID, providerID string) error {
	queries := sqlc.New(s.pool)
	return queries.DeleteCredential(ctx, sqlc.DeleteCredentialParams{
		UserID:     userID,
		ProviderID: providerID,
	})
}

// DeleteAllUserCredentials removes all credentials for a user (for account deletion)
func (s *CredentialVaultService) DeleteAllUserCredentials(ctx context.Context, userID string) error {
	queries := sqlc.New(s.pool)
	return queries.DeleteAllUserCredentials(ctx, userID)
}

// RefreshOAuthToken updates the OAuth token after a refresh
func (s *CredentialVaultService) RefreshOAuthToken(ctx context.Context, userID, providerID string, newAccessToken string, newExpiresAt *time.Time) error {
	// First get existing credential to preserve other data
	cred, err := s.GetCredential(ctx, userID, providerID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}

	if cred.OAuthData == nil {
		return errors.New("credential is not OAuth type")
	}

	// Update with new access token
	cred.OAuthData.AccessToken = newAccessToken

	// Serialize to JSON
	jsonData, err := json.Marshal(cred.OAuthData)
	if err != nil {
		return fmt.Errorf("failed to serialize token data: %w", err)
	}

	// Encrypt the JSON
	encryptedData, err := s.encryptData(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to encrypt token data: %w", err)
	}

	// Prepare expiry
	var expiresAt pgtype.Timestamptz
	if newExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *newExpiresAt, Valid: true}
	}

	queries := sqlc.New(s.pool)
	return queries.UpdateCredentialExpiry(ctx, sqlc.UpdateCredentialExpiryParams{
		UserID:        userID,
		ProviderID:    providerID,
		EncryptedData: encryptedData,
		ExpiresAt:     expiresAt,
	})
}

// GetExpiringCredentials returns credentials expiring within the given duration
// Useful for proactive token refresh
func (s *CredentialVaultService) GetExpiringCredentials(ctx context.Context, within time.Duration) ([]*Credential, error) {
	queries := sqlc.New(s.pool)

	// Convert duration to PostgreSQL interval
	interval := pgtype.Interval{
		Microseconds: int64(within / time.Microsecond),
		Valid:        true,
	}

	results, err := queries.GetExpiringCredentials(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring credentials: %w", err)
	}

	credentials := make([]*Credential, 0, len(results))
	for _, r := range results {
		cred, err := s.decryptCredential(r)
		if err != nil {
			continue
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// GetExpiredCredentials returns all credentials that have already expired
func (s *CredentialVaultService) GetExpiredCredentials(ctx context.Context) ([]*Credential, error) {
	queries := sqlc.New(s.pool)

	results, err := queries.GetExpiredCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired credentials: %w", err)
	}

	credentials := make([]*Credential, 0, len(results))
	for _, r := range results {
		cred, err := s.decryptCredential(r)
		if err != nil {
			continue
		}
		credentials = append(credentials, cred)
	}

	return credentials, nil
}

// --- Internal helper methods ---

func (s *CredentialVaultService) encryptData(plaintext string) ([]byte, error) {
	if s.encryption == nil {
		// Development mode - no encryption configured
		// Store as-is (not recommended for production)
		return []byte(plaintext), nil
	}
	return s.encryption.EncryptBytes(plaintext)
}

func (s *CredentialVaultService) decryptData(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	if s.encryption == nil {
		// Development mode - data stored as plaintext
		return string(ciphertext), nil
	}
	return s.encryption.DecryptBytes(ciphertext)
}

func (s *CredentialVaultService) decryptCredential(dbCred sqlc.CredentialVault) (*Credential, error) {
	// Decrypt the data
	plaintext, err := s.decryptData(dbCred.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	var oauthData *OAuthTokenData
	var apiKeyData *APIKeyData

	switch dbCred.CredentialType {
	case CredentialTypeOAuth:
		oauthData = &OAuthTokenData{}
		if err := json.Unmarshal([]byte(plaintext), oauthData); err != nil {
			return nil, fmt.Errorf("failed to parse OAuth data: %w", err)
		}
	case CredentialTypeAPIKey:
		apiKeyData = &APIKeyData{}
		if err := json.Unmarshal([]byte(plaintext), apiKeyData); err != nil {
			return nil, fmt.Errorf("failed to parse API key data: %w", err)
		}
	case CredentialTypeCustom:
		// Custom types store raw data - kept as plaintext for flexibility
		// Could be parsed by the caller based on provider-specific format
	}

	cred, err := s.dbCredentialToCredential(dbCred, oauthData, apiKeyData)
	if err != nil {
		return nil, err
	}

	// For custom types, store the raw decrypted data
	if dbCred.CredentialType == CredentialTypeCustom {
		cred.RawData = plaintext
	}

	return cred, nil
}

func (s *CredentialVaultService) dbCredentialToCredential(dbCred sqlc.CredentialVault, oauthData *OAuthTokenData, apiKeyData *APIKeyData) (*Credential, error) {
	cred := &Credential{
		ID:             dbCred.ID.String(),
		UserID:         dbCred.UserID,
		ProviderID:     dbCred.ProviderID,
		CredentialType: dbCred.CredentialType,
		Scopes:         dbCred.Scopes,
		CreatedAt:      dbCred.CreatedAt.Time,
		UpdatedAt:      dbCred.UpdatedAt.Time,
		OAuthData:      oauthData,
		APIKeyData:     apiKeyData,
	}

	// Handle optional fields
	if dbCred.ExpiresAt.Valid {
		t := dbCred.ExpiresAt.Time
		cred.ExpiresAt = &t
	}
	if dbCred.ExternalAccountID != nil {
		cred.ExternalAccountID = *dbCred.ExternalAccountID
	}
	if dbCred.ExternalAccountEmail != nil {
		cred.ExternalAccountEmail = *dbCred.ExternalAccountEmail
	}
	if dbCred.ExternalWorkspaceID != nil {
		cred.ExternalWorkspaceID = *dbCred.ExternalWorkspaceID
	}
	if dbCred.ExternalWorkspaceName != nil {
		cred.ExternalWorkspaceName = *dbCred.ExternalWorkspaceName
	}
	if dbCred.LastUsedAt.Valid {
		t := dbCred.LastUsedAt.Time
		cred.LastUsedAt = &t
	}
	if dbCred.LastRotatedAt.Valid {
		t := dbCred.LastRotatedAt.Time
		cred.LastRotatedAt = &t
	}

	// Parse metadata
	if len(dbCred.Metadata) > 0 {
		cred.Metadata = make(map[string]interface{})
		_ = json.Unmarshal(dbCred.Metadata, &cred.Metadata)
	}

	return cred, nil
}

// Helper function to convert empty string to nil pointer
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- Sorx Integration Methods ---
// These methods provide the interface that Sorx skills will use to get credentials

// GetCredentialForSkill retrieves a credential for use by a Sorx skill
// This is the main entry point for skill execution
func (s *CredentialVaultService) GetCredentialForSkill(ctx context.Context, userID, providerID string) (*Credential, error) {
	cred, err := s.GetCredential(ctx, userID, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential for skill: %w", err)
	}

	// Check if expired
	if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("credential has expired - user needs to reconnect")
	}

	return cred, nil
}

// GetAccessToken is a simplified method for skills that just need the access token
func (s *CredentialVaultService) GetAccessToken(ctx context.Context, userID, providerID string) (string, error) {
	cred, err := s.GetCredentialForSkill(ctx, userID, providerID)
	if err != nil {
		return "", err
	}

	if cred.OAuthData != nil {
		return cred.OAuthData.AccessToken, nil
	}
	if cred.APIKeyData != nil {
		return cred.APIKeyData.APIKey, nil
	}

	return "", errors.New("credential type not supported for access token retrieval")
}
