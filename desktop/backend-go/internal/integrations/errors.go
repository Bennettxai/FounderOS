// Package integrations provides integration error types.
package integrations

import "errors"

// Common integration errors.
var (
	// ErrProviderNotFound indicates the requested provider doesn't exist.
	ErrProviderNotFound = errors.New("integration provider not found")

	// ErrNotConnected indicates the user hasn't connected this integration.
	ErrNotConnected = errors.New("integration not connected")

	// ErrTokenExpired indicates the OAuth token has expired and refresh failed.
	ErrTokenExpired = errors.New("token expired and refresh failed")

	// ErrTokenRefreshFailed indicates the token refresh attempt failed.
	ErrTokenRefreshFailed = errors.New("token refresh failed")

	// ErrInvalidState indicates the OAuth state parameter doesn't match.
	ErrInvalidState = errors.New("invalid OAuth state parameter")

	// ErrCodeExchangeFailed indicates the authorization code exchange failed.
	ErrCodeExchangeFailed = errors.New("authorization code exchange failed")

	// ErrSyncInProgress indicates a sync is already running.
	ErrSyncInProgress = errors.New("sync already in progress")

	// ErrSyncNotSupported indicates the provider doesn't support sync.
	ErrSyncNotSupported = errors.New("provider does not support sync")

	// ErrRateLimited indicates the external API rate limited the request.
	ErrRateLimited = errors.New("external API rate limited")

	// ErrPermissionDenied indicates insufficient permissions for the operation.
	ErrPermissionDenied = errors.New("insufficient permissions")

	// ErrExternalAPIError indicates an error from the external service.
	ErrExternalAPIError = errors.New("external API error")
)

// ProviderError wraps an error with provider context.
type ProviderError struct {
	Provider string
	Op       string // operation that failed
	Err      error
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Op + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider, op string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Err:      err,
	}
}

// SyncError represents an error that occurred during sync.
type SyncError struct {
	Provider  string
	Resource  string // e.g., "tasks", "events", "contacts"
	ItemID    string
	Err       error
	Retryable bool
}

func (e *SyncError) Error() string {
	msg := e.Provider + ": sync " + e.Resource
	if e.ItemID != "" {
		msg += " [" + e.ItemID + "]"
	}
	msg += ": " + e.Err.Error()
	return msg
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

// NewSyncError creates a new SyncError.
func NewSyncError(provider, resource, itemID string, err error, retryable bool) *SyncError {
	return &SyncError{
		Provider:  provider,
		Resource:  resource,
		ItemID:    itemID,
		Err:       err,
		Retryable: retryable,
	}
}
