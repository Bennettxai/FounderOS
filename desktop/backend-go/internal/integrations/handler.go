// Package integrations provides HTTP handlers for unified integration management.
package integrations

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// CredentialVault defines the interface for credential storage.
// This avoids import cycles - services.CredentialVaultService implements this.
type CredentialVault interface {
	CredentialExists(ctx context.Context, userID, providerID string) (bool, error)
	StoreOAuthCredential(ctx context.Context, input StoreOAuthInput) (string, error)
	GetCredential(ctx context.Context, userID, providerID string) (*Credential, error)
	GetUserCredentials(ctx context.Context, userID string) ([]*Credential, error)
	DeleteCredential(ctx context.Context, userID, providerID string) error
}

// StoreOAuthInput contains the data needed to store OAuth credentials.
type StoreOAuthInput struct {
	UserID               string
	ProviderID           string
	AccessToken          string
	RefreshToken         string
	ExpiresAt            *time.Time
	ExternalAccountID    string
	ExternalAccountEmail string
	Scopes               []string
	Metadata             map[string]interface{}
}

// Credential represents stored credentials.
type Credential struct {
	ID                    string
	UserID                string
	ProviderID            string
	CredentialType        string
	ExternalAccountID     string
	ExternalAccountEmail  string
	ExternalWorkspaceID   string
	ExternalWorkspaceName string
	Scopes                []string
	Metadata              map[string]interface{}
	ExpiresAt             *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastUsedAt            *time.Time
	OAuthData             *OAuthData
}

// OAuthData contains OAuth-specific token data.
type OAuthData struct {
	AccessToken  string
	RefreshToken string
}

// Handler provides HTTP endpoints for integration management.
type Handler struct {
	vault CredentialVault
}

// NewHandler creates a new integration handler.
func NewHandler(vault CredentialVault) *Handler {
	return &Handler{
		vault: vault,
	}
}

// RegisterRoutes registers integration routes on the router.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	integrations := r.Group("/integrations")
	{
		// List all available providers
		integrations.GET("/providers", h.ListProviders)
		integrations.GET("/providers/:name", h.GetProvider)

		// OAuth flow
		integrations.GET("/oauth/:provider/start", h.StartOAuth)
		integrations.GET("/oauth/:provider/callback", h.OAuthCallback)

		// User integrations (require auth)
		integrations.GET("", h.ListUserIntegrations)
		integrations.GET("/:provider", h.GetUserIntegration)
		integrations.DELETE("/:provider", h.DisconnectIntegration)
		integrations.POST("/:provider/sync", h.TriggerSync)
		integrations.GET("/:provider/status", h.GetIntegrationStatus)
	}
}

// ============================================================================
// Response Types
// ============================================================================

// ProviderResponse represents a provider in API responses.
type ProviderResponse struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Scopes      []string `json:"scopes,omitempty"`
	Connected   bool     `json:"connected,omitempty"`
}

// IntegrationStatusResponse represents the status of a user's integration.
type IntegrationStatusResponse struct {
	Provider     string     `json:"provider"`
	Connected    bool       `json:"connected"`
	ConnectedAt  *time.Time `json:"connected_at,omitempty"`
	AccountID    string     `json:"account_id,omitempty"`
	AccountName  string     `json:"account_name,omitempty"`
	AccountEmail string     `json:"account_email,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus   string     `json:"sync_status,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// OAuthStartResponse is returned when initiating OAuth.
type OAuthStartResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// SyncResponse is returned after triggering a sync.
type SyncResponse struct {
	Success      bool     `json:"success"`
	ItemsCreated int      `json:"items_created"`
	ItemsUpdated int      `json:"items_updated"`
	ItemsDeleted int      `json:"items_deleted"`
	Errors       []string `json:"errors,omitempty"`
	Duration     string   `json:"duration"`
}

// ============================================================================
// Helper Functions
// ============================================================================

// getUserID extracts the user ID from the Gin context.
// Returns empty string if not authenticated.
// SECURITY: Only trusts user_id from auth middleware context, never from headers.
func getUserID(c *gin.Context) string {
	// Only trust user_id set by auth middleware in the context
	// NEVER accept user ID from headers - that would allow impersonation
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}

	return ""
}

// requireUserID ensures the user is authenticated.
// Returns the userID if authenticated, or sends an error response and returns empty string.
func requireUserID(c *gin.Context) string {
	userID := getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return ""
	}
	return userID
}

// ============================================================================
// Provider Endpoints
// ============================================================================

// ListProviders returns all available integration providers.
func (h *Handler) ListProviders(c *gin.Context) {
	userID := getUserID(c) // Optional - used to check connection status
	providers := List()
	response := make([]ProviderResponse, 0, len(providers))

	for _, p := range providers {
		pr := ProviderResponse{
			Name:        p.Name(),
			DisplayName: p.DisplayName(),
			Category:    p.Category(),
			Icon:        p.Icon(),
		}

		// Check if user is connected to this provider
		if userID != "" && h.vault != nil {
			connected, _ := h.vault.CredentialExists(c.Request.Context(), userID, p.Name())
			pr.Connected = connected
		}

		response = append(response, pr)
	}

	c.JSON(http.StatusOK, response)
}

// GetProvider returns details about a specific provider.
func (h *Handler) GetProvider(c *gin.Context) {
	name := c.Param("name")
	userID := getUserID(c)

	provider, ok := Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	response := ProviderResponse{
		Name:        provider.Name(),
		DisplayName: provider.DisplayName(),
		Category:    provider.Category(),
		Icon:        provider.Icon(),
	}

	// Check if user is connected
	if userID != "" && h.vault != nil {
		connected, _ := h.vault.CredentialExists(c.Request.Context(), userID, provider.Name())
		response.Connected = connected
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// OAuth Endpoints
// ============================================================================

// StartOAuth initiates the OAuth flow for a provider.
func (h *Handler) StartOAuth(c *gin.Context) {
	providerName := c.Param("provider")

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Generate state for CSRF protection
	state, err := GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	// Store state in cookie for verification on callback
	// SECURITY: Secure=true in production, HttpOnly=true always, SameSite=Strict for CSRF protection
	isSecure := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release"

	// Set SameSite cookie attribute via SetSameSite (gin helper doesn't support it directly)
	// We'll use the lower-level http.SetCookie instead
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})

	// Get the authorization URL
	authURL := provider.GetAuthURL(state)

	response := OAuthStartResponse{
		AuthURL: authURL,
		State:   state,
	}

	c.JSON(http.StatusOK, response)
}

// OAuthCallback handles the OAuth callback from providers.
// This endpoint should verify state, exchange code, and save tokens.
func (h *Handler) OAuthCallback(c *gin.Context) {
	providerName := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
	errMsg := c.Query("error")

	if errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("OAuth error: %s", errMsg)})
		return
	}

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "authorization code required"})
		return
	}

	// Verify state matches cookie using constant-time comparison
	// This prevents timing attacks that could leak state information
	storedState, err := c.Cookie("oauth_state")
	if err != nil || subtle.ConstantTimeCompare([]byte(storedState), []byte(state)) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	// Clear the state cookie (use same secure setting and SameSite mode)
	clearSecure := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release"

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   clearSecure,
		SameSite: http.SameSiteStrictMode,
	})

	// Get user ID - required for callback
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Exchange code for tokens
	tokenResp, err := provider.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to exchange code: %v", err)})
		return
	}

	// Store credentials in vault
	if h.vault != nil {
		_, err = h.vault.StoreOAuthCredential(c.Request.Context(), StoreOAuthInput{
			UserID:               userID,
			ProviderID:           providerName,
			AccessToken:          tokenResp.AccessToken,
			RefreshToken:         tokenResp.RefreshToken,
			ExpiresAt:            &tokenResp.ExpiresAt,
			ExternalAccountID:    tokenResp.AccountID,
			ExternalAccountEmail: tokenResp.AccountEmail,
			Scopes:               tokenResp.Scopes,
			Metadata:             tokenResp.Metadata,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to store credentials: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"provider":      providerName,
		"account_email": tokenResp.AccountEmail,
		"account_name":  tokenResp.AccountName,
	})
}

// ============================================================================
// User Integration Endpoints
// ============================================================================

// ListUserIntegrations returns all integrations for the current user.
func (h *Handler) ListUserIntegrations(c *gin.Context) {
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	// Build response with all providers and their connection status
	providers := List()
	response := make([]IntegrationStatusResponse, 0, len(providers))

	// Get all credentials for the user if vault is available
	var connectedProviders map[string]*Credential
	if h.vault != nil {
		credentials, err := h.vault.GetUserCredentials(c.Request.Context(), userID)
		if err == nil {
			connectedProviders = make(map[string]*Credential)
			for _, cred := range credentials {
				connectedProviders[cred.ProviderID] = cred
			}
		}
	}

	for _, p := range providers {
		status := IntegrationStatusResponse{
			Provider:   p.Name(),
			Connected:  false,
			SyncStatus: "idle",
		}

		if cred, ok := connectedProviders[p.Name()]; ok {
			status.Connected = true
			status.AccountID = cred.ExternalAccountID
			status.AccountEmail = cred.ExternalAccountEmail
			status.Scopes = cred.Scopes
			if !cred.CreatedAt.IsZero() {
				status.ConnectedAt = &cred.CreatedAt
			}
			if cred.LastUsedAt != nil {
				status.LastSyncAt = cred.LastUsedAt
			}
		}

		response = append(response, status)
	}

	c.JSON(http.StatusOK, response)
}

// GetUserIntegration returns the status of a specific integration for the user.
func (h *Handler) GetUserIntegration(c *gin.Context) {
	providerName := c.Param("provider")
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	response := IntegrationStatusResponse{
		Provider:   providerName,
		Connected:  false,
		SyncStatus: "idle",
	}

	// Try to get the credential
	if h.vault != nil {
		cred, err := h.vault.GetCredential(c.Request.Context(), userID, providerName)
		if err == nil && cred != nil {
			response.Connected = true
			response.AccountID = cred.ExternalAccountID
			response.AccountEmail = cred.ExternalAccountEmail
			response.Scopes = cred.Scopes
			if !cred.CreatedAt.IsZero() {
				response.ConnectedAt = &cred.CreatedAt
			}
			if cred.LastUsedAt != nil {
				response.LastSyncAt = cred.LastUsedAt
			}

			// Check if token is expired
			if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
				response.Error = "token expired - reconnection required"
			}
		}
	}

	_ = provider // Used for type assertion if needed

	c.JSON(http.StatusOK, response)
}

// DisconnectIntegration disconnects a user's integration.
func (h *Handler) DisconnectIntegration(c *gin.Context) {
	providerName := c.Param("provider")
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Delete credential from vault
	if h.vault != nil {
		err := h.vault.DeleteCredential(c.Request.Context(), userID, providerName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect integration"})
			return
		}
	}

	// Call provider's disconnect method if it has additional cleanup
	if err := provider.Disconnect(c.Request.Context(), userID); err != nil {
		slog.Warn("Warning: provider disconnect cleanup failed", "id", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"provider": providerName,
	})
}

// TriggerSync triggers a sync for the specified integration.
func (h *Handler) TriggerSync(c *gin.Context) {
	providerName := c.Param("provider")
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	if !provider.SupportsSync() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support sync"})
		return
	}

	// Verify user has credentials for this provider
	if h.vault != nil {
		_, err := h.vault.GetCredential(c.Request.Context(), userID, providerName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "integration not connected"})
			return
		}
	}

	// Parse sync options from request body
	var options SyncOptions
	if c.Request.Body != nil {
		json.NewDecoder(c.Request.Body).Decode(&options)
	}

	// Execute sync
	startTime := time.Now()
	result, err := provider.Sync(c.Request.Context(), userID, options)
	duration := time.Since(startTime)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    fmt.Sprintf("sync failed: %v", err),
			"duration": duration.String(),
		})
		return
	}

	response := SyncResponse{
		Success:      result.Success,
		ItemsCreated: result.ItemsCreated,
		ItemsUpdated: result.ItemsUpdated,
		ItemsDeleted: result.ItemsDeleted,
		Errors:       result.Errors,
		Duration:     duration.String(),
	}

	c.JSON(http.StatusOK, response)
}

// GetIntegrationStatus returns detailed sync status for an integration.
func (h *Handler) GetIntegrationStatus(c *gin.Context) {
	providerName := c.Param("provider")
	userID := requireUserID(c)
	if userID == "" {
		return
	}

	provider, ok := Get(providerName)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	response := IntegrationStatusResponse{
		Provider:   providerName,
		Connected:  false,
		SyncStatus: "idle",
	}

	// Get connection status from provider
	status, err := provider.GetConnectionStatus(c.Request.Context(), userID)
	if err == nil && status != nil {
		response.Connected = status.Connected
		response.AccountID = status.AccountID
		response.AccountName = status.AccountName
		response.AccountEmail = status.AccountEmail
		response.Scopes = status.Scopes
		response.ConnectedAt = status.ConnectedAt
		response.LastSyncAt = status.LastSyncAt
		response.SyncStatus = status.SyncStatus
		response.Error = status.Error
	}

	c.JSON(http.StatusOK, response)
}
