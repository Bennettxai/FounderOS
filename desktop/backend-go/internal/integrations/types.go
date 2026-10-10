// Package integrations provides the interface and types for all integration providers.
// This follows the INTEGRATION_INFRASTRUCTURE.md architecture spec.
package integrations

import (
	"context"
	"time"
)

// Provider is the interface that all integration providers must implement.
// It defines the contract for OAuth, connection management, and sync operations.
type Provider interface {
	// Identity
	Name() string        // "google_calendar", "slack", "notion"
	DisplayName() string // "Google Calendar", "Slack"
	Category() string    // "calendar", "communication", "productivity"
	Icon() string        // URL or icon identifier

	// OAuth
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

	// Connection Management
	GetConnectionStatus(ctx context.Context, userID string) (*ConnectionStatus, error)
	Disconnect(ctx context.Context, userID string) error

	// Token Management
	SaveToken(ctx context.Context, userID string, token *TokenResponse) error
	GetToken(ctx context.Context, userID string) (*Token, error)

	// Sync (optional - providers can return false for SupportsSync)
	SupportsSync() bool
	Sync(ctx context.Context, userID string, options SyncOptions) (*SyncResult, error)
}

// TokenResponse represents the response from an OAuth token exchange.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scopes       []string
	AccountID    string
	AccountName  string
	AccountEmail string
	Metadata     map[string]interface{}
}

// ConnectionStatus represents the current state of a provider connection.
type ConnectionStatus struct {
	Connected    bool
	ConnectedAt  *time.Time
	AccountID    string
	AccountName  string
	AccountEmail string
	Scopes       []string
	LastSyncAt   *time.Time
	SyncStatus   string // "idle", "syncing", "error"
	Error        string
}

// Token represents stored OAuth tokens for a user.
type Token struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scopes       []string
}

// SyncOptions configures a sync operation.
type SyncOptions struct {
	Type      string     // "full", "incremental"
	Since     *time.Time // For incremental syncs
	Resources []string   // ["tasks", "projects", "events"]
}

// SyncResult represents the outcome of a sync operation.
type SyncResult struct {
	Success       bool
	ItemsCreated  int
	ItemsUpdated  int
	ItemsDeleted  int
	Errors        []string
	NextSyncToken string
	Duration      time.Duration
}

// ProviderInfo provides metadata about a registered provider.
type ProviderInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Scopes      []string `json:"scopes"`
	AuthURL     string   `json:"auth_url,omitempty"`
}

// UserIntegration represents a user's connection to a provider.
type UserIntegration struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	ProviderID          string     `json:"provider_id"`
	Status              string     `json:"status"` // "connected", "disconnected", "error"
	ConnectedAt         *time.Time `json:"connected_at"`
	ExternalAccountID   string     `json:"external_account_id,omitempty"`
	ExternalAccountName string     `json:"external_account_name,omitempty"`
	Scopes              []string   `json:"scopes"`
	Settings            any        `json:"settings,omitempty"`
	LastSyncAt          *time.Time `json:"last_sync_at,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
}
