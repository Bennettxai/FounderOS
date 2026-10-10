// Package integrations provides OAuth helper utilities.
package integrations

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthConfig holds the configuration for an OAuth 2.0 provider.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURI  string
	Scopes       []string
}

// GenerateState creates a cryptographically secure random state string.
// Used for CSRF protection in OAuth flows.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// BuildAuthURL constructs the authorization URL for an OAuth flow.
func BuildAuthURL(config OAuthConfig, state string, extraParams map[string]string) string {
	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("state", state)

	if len(config.Scopes) > 0 {
		params.Set("scope", strings.Join(config.Scopes, " "))
	}

	for k, v := range extraParams {
		params.Set(k, v)
	}

	return config.AuthURL + "?" + params.Encode()
}

// TokenExchangeResponse represents a raw OAuth token response.
type TokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// ExchangeCode exchanges an authorization code for tokens.
func ExchangeCode(config OAuthConfig, code string) (*TokenExchangeResponse, error) {
	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshAccessToken refreshes an expired access token.
func RefreshAccessToken(config OAuthConfig, refreshToken string) (*TokenExchangeResponse, error) {
	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// ToTokenResponse converts a raw token exchange response to the standard TokenResponse.
func (t *TokenExchangeResponse) ToTokenResponse() *TokenResponse {
	expiresAt := time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)

	var scopes []string
	if t.Scope != "" {
		scopes = strings.Split(t.Scope, " ")
	}

	return &TokenResponse{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
	}
}

// IsTokenExpired checks if a token is expired or will expire within the buffer period.
func IsTokenExpired(expiresAt time.Time, buffer time.Duration) bool {
	return time.Now().Add(buffer).After(expiresAt)
}

// DefaultExpiryBuffer is the default time before expiry to consider a token expired.
const DefaultExpiryBuffer = 5 * time.Minute

// GenerateUserState creates an OAuth state token that includes the user ID.
// This allows the callback handler to identify which user initiated the flow.
func GenerateUserState(userID string) string {
	return GenerateUserStateWithRedirect(userID, "")
}

// GenerateUserStateWithRedirect creates an OAuth state token carrying the user
// ID and the final redirect target requested when the flow started. OAuth
// providers round-trip `state` verbatim, so this is the only reliable way to
// carry the desktop deep-link target through the consent hop (the provider does
// not preserve arbitrary query params on the callback URL).
func GenerateUserStateWithRedirect(userID, redirect string) string {
	data := map[string]string{
		"user_id":   userID,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if redirect != "" {
		data["redirect"] = redirect
	}
	b, _ := json.Marshal(data)
	return string(b)
}

// ExtractUserIDFromState extracts the user ID from an OAuth state token.
// Returns empty string if the state is invalid or doesn't contain a user ID.
func ExtractUserIDFromState(state string) string {
	var data map[string]string
	if err := json.Unmarshal([]byte(state), &data); err != nil {
		return ""
	}
	return data["user_id"]
}

// ExtractRedirectFromState extracts the requested final redirect target from an
// OAuth state token. Returns empty string when none was carried.
func ExtractRedirectFromState(state string) string {
	var data map[string]string
	if err := json.Unmarshal([]byte(state), &data); err != nil {
		return ""
	}
	return data["redirect"]
}
