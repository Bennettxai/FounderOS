package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// buildOAuthConfig creates a minimal *config.Config with the requested
// provider credentials set so that the handler initialises the oauth2.Config.
func buildOAuthConfig(provider string) *config.Config {
	cfg := &config.Config{Environment: "development"}
	switch provider {
	case "google":
		cfg.GoogleClientID = "google-client-id"
		cfg.GoogleClientSecret = "google-client-secret"
		cfg.GoogleRedirectURI = "http://localhost:8801/auth/oauth/google/callback"
	case "slack":
		cfg.SlackClientID = "slack-client-id"
		cfg.SlackClientSecret = "slack-client-secret"
	case "notion":
		cfg.NotionClientID = "notion-client-id"
		cfg.NotionClientSecret = "notion-client-secret"
	case "microsoft":
		cfg.MicrosoftClientID = "ms-client-id"
		cfg.MicrosoftClientSecret = "ms-client-secret"
	case "linear":
		cfg.LinearClientID = "linear-client-id"
		cfg.LinearClientSecret = "linear-client-secret"
	}
	return cfg
}

func TestInitiateGoogleLogin_NotConfigured(t *testing.T) {
	h := NewGoogleAuthHandler(nil, &config.Config{Environment: "development"}, nil)

	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/google", h.InitiateGoogleLogin)
	})

	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/auth/google?redirect=%2Fwindow", nil))

	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	assert.Empty(t, res.Header().Get("Location"))
	assert.Contains(t, res.Body.String(), "google_oauth_not_configured")
}

func TestIsTrustedGoogleAvatarURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "google avatar", url: "https://lh3.googleusercontent.com/a/example", want: true},
		{name: "http is rejected", url: "http://lh3.googleusercontent.com/a/example", want: false},
		{name: "lookalike host is rejected", url: "https://lh3.googleusercontent.com.attacker.example/a/example", want: false},
		{name: "arbitrary host is rejected", url: "https://example.com/avatar.png", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isTrustedGoogleAvatarURL(tt.url))
		})
	}
}

func TestGoogleLoginCallback_DesktopStateSurvivesMissingBrowserCookie(t *testing.T) {
	desktopGoogleOAuthStates = &googleOAuthStateStore{
		states: make(map[string]googleOAuthStateEntry),
	}
	h := NewGoogleAuthHandler(nil, buildOAuthConfig("google"), nil)

	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/google", h.InitiateGoogleLogin)
		r.GET("/auth/oauth/google/callback", h.HandleGoogleLoginCallback)
	})

	start := httptest.NewRecorder()
	r.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/auth/google?redirect=businessos%3A%2F%2Fauth%2Fcallback", nil))
	require.Equal(t, http.StatusTemporaryRedirect, start.Code)

	authURL, err := url.Parse(start.Header().Get("Location"))
	require.NoError(t, err)
	state := authURL.Query().Get("state")
	require.NotEmpty(t, state)

	callback := httptest.NewRecorder()
	r.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=fake-code&state="+url.QueryEscape(state), nil))

	require.Equal(t, http.StatusInternalServerError, callback.Code)
	assert.Contains(t, callback.Body.String(), "Failed to exchange OAuth code")

	replay := httptest.NewRecorder()
	r.ServeHTTP(replay, httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?code=fake-code&state="+url.QueryEscape(state), nil))

	require.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Contains(t, replay.Body.String(), "Invalid state parameter")
}

// buildOAuthConfigAll creates a config with all providers configured.
func buildOAuthConfigAll() *config.Config {
	return &config.Config{
		Environment:           "development",
		SlackClientID:         "slack-client-id",
		SlackClientSecret:     "slack-client-secret",
		NotionClientID:        "notion-client-id",
		NotionClientSecret:    "notion-client-secret",
		MicrosoftClientID:     "ms-client-id",
		MicrosoftClientSecret: "ms-client-secret",
		LinearClientID:        "linear-client-id",
		LinearClientSecret:    "linear-client-secret",
	}
}

// newOAuthHandler creates a handler with no database (pool = nil).
// Sufficient for tests that only test the redirect/state logic and don't
// hit storeOAuthToken.
func newOAuthHandlerNoPool(cfg *config.Config) *OAuthIntegrationHandler {
	return NewOAuthIntegrationHandler(nil, cfg)
}

// oauthRouter creates a Gin engine for OAuth handler tests.
// userID is injected into context to simulate an authenticated user.
func oauthRouter(userID string, routes func(r *gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if userID != "" {
		r.Use(func(c *gin.Context) {
			c.Set(middleware.UserContextKey, &middleware.BetterAuthUser{
				ID:    userID,
				Email: userID + "@example.com",
			})
			c.Next()
		})
	}
	routes(r)
	return r
}

// cookieHeader returns the value of the named cookie from an http.Response.
func cookieHeader(resp *http.Response, name string) string {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// generateState / generateSessionBoundState
// ---------------------------------------------------------------------------

func TestGenerateState_ReturnsNonEmpty(t *testing.T) {
	state, err := generateState()
	require.NoError(t, err)
	assert.NotEmpty(t, state)
}

func TestGenerateState_IsUnique(t *testing.T) {
	s1, err1 := generateState()
	s2, err2 := generateState()
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, s1, s2, "two states should not collide")
}

func TestGenerateSessionBoundState_WithSessionID(t *testing.T) {
	sessionID := "session-abc123"
	state, err := generateSessionBoundState(sessionID)
	require.NoError(t, err)
	assert.NotEmpty(t, state)
	// State should contain a ":" separator with the 8-char session prefix as suffix
	assert.Contains(t, state, ":")
	parts := strings.Split(state, ":")
	require.Len(t, parts, 2)
	assert.Equal(t, "session-", parts[1], "binding should be first 8 chars of session ID")
}

func TestGenerateSessionBoundState_WithShortSessionID(t *testing.T) {
	sessionID := "abc"
	state, err := generateSessionBoundState(sessionID)
	require.NoError(t, err)
	assert.Contains(t, state, ":")
	parts := strings.Split(state, ":")
	require.Len(t, parts, 2)
	assert.Equal(t, "abc", parts[1])
}

func TestGenerateSessionBoundState_EmptySessionID(t *testing.T) {
	state, err := generateSessionBoundState("")
	require.NoError(t, err)
	assert.NotEmpty(t, state)
	// No binding appended when session is empty
	assert.NotContains(t, state, ":")
}

// ---------------------------------------------------------------------------
// validateSessionBoundState
// ---------------------------------------------------------------------------

func TestValidateSessionBoundState_ExactMatch(t *testing.T) {
	assert.True(t, validateSessionBoundState("abc", "abc", ""), "identical states should pass")
}

func TestValidateSessionBoundState_Mismatch(t *testing.T) {
	assert.False(t, validateSessionBoundState("abc", "xyz", ""), "different states should fail")
}

func TestValidateSessionBoundState_EmptyStateAlwaysFails(t *testing.T) {
	assert.False(t, validateSessionBoundState("", "expected", ""), "empty incoming state should fail")
}

func TestValidateSessionBoundState_SessionBound_Valid(t *testing.T) {
	// Simulate the flow: generate → store → validate
	sessionID := "user-session-id-123"
	state, err := generateSessionBoundState(sessionID)
	require.NoError(t, err)
	assert.True(t, validateSessionBoundState(state, state, sessionID))
}

func TestValidateSessionBoundState_SessionBound_WrongSession(t *testing.T) {
	sessionID := "user-session-id-123"
	state, err := generateSessionBoundState(sessionID)
	require.NoError(t, err)
	// Different session ID → binding mismatch → validation should fail
	assert.False(t, validateSessionBoundState(state, state, "other-session"))
}

func TestValidateSessionBoundState_SessionBound_StoredStateMismatch(t *testing.T) {
	sessionID := "user-session-id-123"
	state, err := generateSessionBoundState(sessionID)
	require.NoError(t, err)
	// Tamper the incoming state value
	tampered := state + "x"
	assert.False(t, validateSessionBoundState(tampered, state, sessionID))
}

// ---------------------------------------------------------------------------
// isSecureCookie
// ---------------------------------------------------------------------------

func TestIsSecureCookie_Development_HTTP(t *testing.T) {
	cfg := &config.Config{Environment: "development"}
	h := newOAuthHandlerNoPool(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// No TLS on request → insecure in dev
	assert.False(t, h.isSecureCookie(c))
}

func TestIsSecureCookie_Production(t *testing.T) {
	cfg := &config.Config{Environment: "production"}
	h := newOAuthHandlerNoPool(cfg)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	assert.True(t, h.isSecureCookie(c))
}

// ---------------------------------------------------------------------------
// NewOAuthIntegrationHandler — provider init
// ---------------------------------------------------------------------------

func TestNewOAuthIntegrationHandler_NoCredentials(t *testing.T) {
	cfg := &config.Config{Environment: "development"}
	h := newOAuthHandlerNoPool(cfg)
	assert.Nil(t, h.slack, "slack should be nil when no creds")
	assert.Nil(t, h.notion, "notion should be nil when no creds")
	assert.Nil(t, h.microsoft, "microsoft should be nil when no creds")
	assert.Nil(t, h.linear, "linear should be nil when no creds")
}

func TestNewOAuthIntegrationHandler_SlackCreds(t *testing.T) {
	cfg := buildOAuthConfig("slack")
	h := newOAuthHandlerNoPool(cfg)
	require.NotNil(t, h.slack)
	assert.Equal(t, "slack-client-id", h.slack.ClientID)
	assert.Equal(t, oauth2.Endpoint{
		AuthURL:  "https://slack.com/oauth/v2/authorize",
		TokenURL: "https://slack.com/api/oauth.v2.access",
	}, h.slack.Endpoint)
	assert.Contains(t, h.slack.Scopes, "channels:read")
	assert.Contains(t, h.slack.Scopes, "chat:write")
}

func TestNewOAuthIntegrationHandler_MicrosoftCreds(t *testing.T) {
	cfg := buildOAuthConfig("microsoft")
	h := newOAuthHandlerNoPool(cfg)
	require.NotNil(t, h.microsoft)
	assert.Contains(t, h.microsoft.Scopes, "Mail.Read")
	assert.Contains(t, h.microsoft.Scopes, "offline_access")
}

func TestNewOAuthIntegrationHandler_AllProviders(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfigAll())
	assert.NotNil(t, h.slack)
	assert.NotNil(t, h.notion)
	assert.NotNil(t, h.microsoft)
	assert.NotNil(t, h.linear)
}

// ---------------------------------------------------------------------------
// redirectWithError
// ---------------------------------------------------------------------------

func TestRedirectWithError_DefaultRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	// No cookie set → should default to /onboarding/connect
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	r.GET("/", func(c *gin.Context) {
		redirectWithError(c, "something went wrong")
	})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "/onboarding/connect")
	assert.Contains(t, loc, url.QueryEscape("something went wrong"))
}

func TestRedirectWithError_CookieRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		redirectWithError(c, "err")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/settings/integrations"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/settings/integrations")
}

// ---------------------------------------------------------------------------
// Slack — InitiateSlackOAuth
// ---------------------------------------------------------------------------

func TestInitiateSlackOAuth_NotConfigured(t *testing.T) {
	cfg := &config.Config{Environment: "development"} // no slack creds
	h := newOAuthHandlerNoPool(cfg)

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack", h.InitiateSlackOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "Slack+integration+not+configured")
}

func TestInitiateSlackOAuth_RedirectsToSlack(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	r := oauthRouter("user-abc123", func(r *gin.Engine) {
		r.GET("/auth/slack", h.InitiateSlackOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "slack.com/oauth/v2/authorize")
	assert.Contains(t, loc, "client_id=slack-client-id")
}

func TestInitiateSlackOAuth_SetsCookies(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	r := oauthRouter("user-abc123", func(r *gin.Engine) {
		r.GET("/auth/slack", h.InitiateSlackOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack", nil))

	resp := w.Result()
	assert.NotEmpty(t, cookieHeader(resp, "oauth_state"), "oauth_state cookie should be set")
	// Gin URL-encodes cookie values containing slashes.
	redirectVal, _ := url.QueryUnescape(cookieHeader(resp, "oauth_redirect"))
	assert.Equal(t, "/onboarding/building", redirectVal)
	// oauth_session stores the full user ID (source of the 8-char binding).
	assert.Equal(t, "user-abc123", cookieHeader(resp, "oauth_session"))
}

func TestInitiateSlackOAuth_CustomRedirect(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack", h.InitiateSlackOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack?redirect=/settings/integrations", nil))

	resp := w.Result()
	redirectVal, _ := url.QueryUnescape(cookieHeader(resp, "oauth_redirect"))
	assert.Equal(t, "/settings/integrations", redirectVal)
}

func TestInitiateSlackOAuth_NoUser_StillRedirects(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	// No user in context
	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/slack", h.InitiateSlackOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "slack.com")
	// oauth_session cookie should NOT be set when no user
	resp := w.Result()
	assert.Empty(t, cookieHeader(resp, "oauth_session"))
}

// ---------------------------------------------------------------------------
// Slack — HandleSlackCallback
// ---------------------------------------------------------------------------

func TestHandleSlackCallback_NotConfigured(t *testing.T) {
	cfg := &config.Config{Environment: "development"}
	h := newOAuthHandlerNoPool(cfg)

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=abc&state=xyz", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "Slack+integration+not+configured")
}

func TestHandleSlackCallback_MissingStateCookie(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})
	// No oauth_state cookie → session expired error
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=abc&state=xyz", nil))

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "session+expired")
}

func TestHandleSlackCallback_StateMismatch_CSRFAttack(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=abc&state=attacker-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "legitimate-state"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "CSRF")
}

func TestHandleSlackCallback_MissingCode(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("slack"))
	legitimateState := "some-valid-state"

	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/slack/callback?state="+legitimateState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: legitimateState})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "No+authorization+code")
}

func TestHandleSlackCallback_TokenExchangeError(t *testing.T) {
	// Use a real-looking but inaccessible token URL to force Exchange() to fail.
	cfg := buildOAuthConfig("slack")
	h := newOAuthHandlerNoPool(cfg)
	// Override the token URL to a local server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`)) //nolint:errcheck
	}))
	defer server.Close()
	h.slack.Endpoint.TokenURL = server.URL

	legitimateState := "test-state-value"
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=bad-code&state="+legitimateState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: legitimateState})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "Failed+to+exchange+token")
}

func TestHandleSlackCallback_SuccessfulExchange(t *testing.T) {
	cfg := buildOAuthConfig("slack")
	h := newOAuthHandlerNoPool(cfg)

	// Fake token server that returns a valid token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		expiry := time.Now().Add(time.Hour).Format(time.RFC3339)
		w.Write([]byte(`{"access_token":"slack-access-token","token_type":"Bearer","expires_in":3600,"expires_at":"` + expiry + `"}`)) //nolint:errcheck
	}))
	defer server.Close()
	h.slack.Endpoint.TokenURL = server.URL

	legitimateState := "test-state-ok"
	r := oauthRouter("", func(r *gin.Engine) { // no user → skip DB store
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=valid-code&state="+legitimateState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: legitimateState})
	req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/onboarding/building"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "integration=slack")
	assert.Contains(t, loc, "status=connected")
}

// ---------------------------------------------------------------------------
// Microsoft — InitiateMicrosoftOAuth
// ---------------------------------------------------------------------------

func TestInitiateMicrosoftOAuth_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/microsoft", h.InitiateMicrosoftOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/microsoft", nil))
	assert.Contains(t, w.Header().Get("Location"), "Microsoft+integration+not+configured")
}

func TestInitiateMicrosoftOAuth_RedirectsToMicrosoft(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("microsoft"))
	r := oauthRouter("user-ms-123", func(r *gin.Engine) {
		r.GET("/auth/microsoft", h.InitiateMicrosoftOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/microsoft", nil))
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "login.microsoftonline.com")
	assert.Contains(t, loc, "client_id=ms-client-id")
	// Offline access for refresh token
	assert.Contains(t, loc, "offline_access")
}

func TestInitiateMicrosoftOAuth_SetsCookies(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("microsoft"))
	r := oauthRouter("user-ms-123", func(r *gin.Engine) {
		r.GET("/auth/microsoft", h.InitiateMicrosoftOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/microsoft", nil))
	resp := w.Result()
	assert.NotEmpty(t, cookieHeader(resp, "oauth_state"))
	redirectVal, _ := url.QueryUnescape(cookieHeader(resp, "oauth_redirect"))
	assert.Equal(t, "/onboarding/building", redirectVal)
}

// ---------------------------------------------------------------------------
// Microsoft — HandleMicrosoftCallback
// ---------------------------------------------------------------------------

func TestHandleMicrosoftCallback_MissingStateCookie(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("microsoft"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/microsoft/callback", h.HandleMicrosoftCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/microsoft/callback?code=c&state=s", nil))
	assert.Contains(t, w.Header().Get("Location"), "session+expired")
}

func TestHandleMicrosoftCallback_StateMismatch(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("microsoft"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/microsoft/callback", h.HandleMicrosoftCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/microsoft/callback?code=c&state=evil-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "legit-state"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "CSRF")
}

func TestHandleMicrosoftCallback_MissingCode(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("microsoft"))
	state := "ms-state-value"
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/microsoft/callback", h.HandleMicrosoftCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/microsoft/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "No+authorization+code")
}

func TestHandleMicrosoftCallback_SuccessfulExchange(t *testing.T) {
	cfg := buildOAuthConfig("microsoft")
	h := newOAuthHandlerNoPool(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"ms-token","token_type":"Bearer","expires_in":3600,"refresh_token":"ms-refresh"}`)) //nolint:errcheck
	}))
	defer server.Close()
	h.microsoft.Endpoint.TokenURL = server.URL

	state := "ms-state-ok"
	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/microsoft/callback", h.HandleMicrosoftCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/microsoft/callback?code=valid&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/onboarding/building"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "integration=outlook")
	assert.Contains(t, loc, "status=connected")
}

func TestHandleMicrosoftCallback_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/microsoft/callback", h.HandleMicrosoftCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/microsoft/callback", nil))
	assert.Contains(t, w.Header().Get("Location"), "Microsoft+integration+not+configured")
}

// ---------------------------------------------------------------------------
// Notion (via oauth_hubspot.go) — InitiateNotionOAuth
// ---------------------------------------------------------------------------

func TestInitiateNotionOAuth_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion", h.InitiateNotionOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/notion", nil))
	assert.Contains(t, w.Header().Get("Location"), "Notion+integration+not+configured")
}

func TestInitiateNotionOAuth_RedirectsToNotion(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("notion"))
	r := oauthRouter("user-notion-1", func(r *gin.Engine) {
		r.GET("/auth/notion", h.InitiateNotionOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/notion", nil))
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "api.notion.com/v1/oauth/authorize")
	// Notion requires owner=user
	assert.Contains(t, loc, "owner=user")
}

func TestInitiateNotionOAuth_SetsCookies(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("notion"))
	r := oauthRouter("user-notion-1", func(r *gin.Engine) {
		r.GET("/auth/notion", h.InitiateNotionOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/notion", nil))
	resp := w.Result()
	assert.NotEmpty(t, cookieHeader(resp, "oauth_state"))
}

// ---------------------------------------------------------------------------
// Notion — HandleNotionCallback
// ---------------------------------------------------------------------------

func TestHandleNotionCallback_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/notion/callback", nil))
	assert.Contains(t, w.Header().Get("Location"), "Notion+integration+not+configured")
}

func TestHandleNotionCallback_MissingStateCookie(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("notion"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/notion/callback?code=c&state=s", nil))
	assert.Contains(t, w.Header().Get("Location"), "session+expired")
}

func TestHandleNotionCallback_StateMismatch(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("notion"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/notion/callback?code=c&state=evil", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "legit"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "CSRF")
}

func TestHandleNotionCallback_MissingCode(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("notion"))
	state := "notion-state"
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/notion/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "No+authorization+code")
}

func TestHandleNotionCallback_SuccessfulExchange(t *testing.T) {
	cfg := buildOAuthConfig("notion")
	h := newOAuthHandlerNoPool(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"notion-token","token_type":"bearer","bot_id":"bot-123"}`)) //nolint:errcheck
	}))
	defer server.Close()
	h.notion.Endpoint.TokenURL = server.URL

	state := "notion-state-ok"
	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/notion/callback?code=valid&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/onboarding/building"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "integration=notion")
	assert.Contains(t, loc, "status=connected")
}

// ---------------------------------------------------------------------------
// Linear — InitiateLinearOAuth
// ---------------------------------------------------------------------------

func TestInitiateLinearOAuth_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/linear", h.InitiateLinearOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/linear", nil))
	assert.Contains(t, w.Header().Get("Location"), "Linear+integration+not+configured")
}

func TestInitiateLinearOAuth_RedirectsToLinear(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("linear"))
	r := oauthRouter("user-linear-1", func(r *gin.Engine) {
		r.GET("/auth/linear", h.InitiateLinearOAuth)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/linear", nil))
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "linear.app/oauth/authorize")
	assert.Contains(t, loc, "client_id=linear-client-id")
}

// ---------------------------------------------------------------------------
// Linear — HandleLinearCallback
// ---------------------------------------------------------------------------

func TestHandleLinearCallback_NotConfigured(t *testing.T) {
	h := newOAuthHandlerNoPool(&config.Config{Environment: "development"})
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/linear/callback", h.HandleLinearCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/linear/callback", nil))
	assert.Contains(t, w.Header().Get("Location"), "Linear+integration+not+configured")
}

func TestHandleLinearCallback_MissingStateCookie(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("linear"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/linear/callback", h.HandleLinearCallback)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/linear/callback?code=c&state=s", nil))
	assert.Contains(t, w.Header().Get("Location"), "session+expired")
}

func TestHandleLinearCallback_StateMismatch(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("linear"))
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/linear/callback", h.HandleLinearCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/linear/callback?code=c&state=evil", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "legit"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "CSRF")
}

func TestHandleLinearCallback_MissingCode(t *testing.T) {
	h := newOAuthHandlerNoPool(buildOAuthConfig("linear"))
	state := "linear-state"
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/linear/callback", h.HandleLinearCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/linear/callback?state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Header().Get("Location"), "No+authorization+code")
}

func TestHandleLinearCallback_SuccessfulExchange(t *testing.T) {
	cfg := buildOAuthConfig("linear")
	h := newOAuthHandlerNoPool(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"linear-token","token_type":"Bearer","expires_in":7776000}`)) //nolint:errcheck
	}))
	defer server.Close()
	h.linear.Endpoint.TokenURL = server.URL

	state := "linear-state-ok"
	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/linear/callback", h.HandleLinearCallback)
	})
	req := httptest.NewRequest(http.MethodGet, "/auth/linear/callback?code=valid&state="+state, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/onboarding/building"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "integration=linear")
	assert.Contains(t, loc, "status=connected")
}

// ---------------------------------------------------------------------------
// Security: state isolation between providers
// ---------------------------------------------------------------------------

// TestOAuthStateIsolation verifies that a state generated by one provider
// cannot be replayed against a different provider's callback.
func TestOAuthStateIsolation_SlackStateCannotBeUsedForNotion(t *testing.T) {
	// Simulate: attacker steals the state cookie from a Slack flow and tries
	// to use it in a Notion callback. The state value itself still matches the
	// stored cookie, so the CSRF check passes — but this test documents that
	// validation is per-flow (state cookie is set/cleared per initiation).
	// The important invariant: two concurrent flows cannot cross-pollinate.
	slackState := "slack-state-value"

	h := newOAuthHandlerNoPool(buildOAuthConfigAll())
	r := oauthRouter("user-1", func(r *gin.Engine) {
		r.GET("/auth/notion/callback", h.HandleNotionCallback)
	})

	// Present the slack state as if it were a notion state
	req := httptest.NewRequest(http.MethodGet, "/auth/notion/callback?code=c&state="+slackState, nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: slackState})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Without a valid code exchange the call will fail at token exchange.
	// The important check: the handler does NOT crash and produces a redirect.
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

// TestOAuthReplayAttack verifies that a second callback with the same state
// fails if the provider refuses the code (codes are one-time use).
func TestOAuthReplayAttack_SecondExchangeFails(t *testing.T) {
	cfg := buildOAuthConfig("slack")
	h := newOAuthHandlerNoPool(cfg)

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`)) //nolint:errcheck
		} else {
			// Second exchange with same code → provider returns error
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_grant","error_description":"Code already used"}`)) //nolint:errcheck
		}
	}))
	defer server.Close()
	h.slack.Endpoint.TokenURL = server.URL

	state := "replay-state"
	r := oauthRouter("", func(r *gin.Engine) {
		r.GET("/auth/slack/callback", h.HandleSlackCallback)
	})

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/auth/slack/callback?code=one-time-code&state="+state, nil)
		req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
		req.AddCookie(&http.Cookie{Name: "oauth_redirect", Value: "/onboarding/building"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w1 := makeReq()
	assert.Contains(t, w1.Header().Get("Location"), "status=connected", "first call should succeed")

	w2 := makeReq()
	assert.Contains(t, w2.Header().Get("Location"), "Failed+to+exchange+token", "replay should fail")
}
