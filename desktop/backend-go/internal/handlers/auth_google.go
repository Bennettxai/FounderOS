package handlers

// Google OAuth Configuration & Gmail API Scope Setup
//
// IMPORTANT: This handler requests the gmail.readonly scope during OAuth.
// For this to work, you MUST complete the following steps in Google Cloud Console:
//
// 1. Enable Gmail API:
//    - Go to: https://console.cloud.google.com/apis/library
//    - Search for "Gmail API"
//    - Click "Enable"
//
// 2. Update OAuth Consent Screen:
//    - Go to: https://console.cloud.google.com/apis/credentials/consent
//    - Add "gmail.readonly" to the scopes list
//    - User-facing description: "Access your emails to analyze work patterns and create personalized recommendations"
//
// 3. User Experience:
//    - Users will see Gmail permission request during login
//    - Users CAN decline Gmail access (login continues without it)
//    - Frontend should detect missing scope and prompt re-authentication if email sync needed
//
// 4. Scope Verification:
//    - This handler checks granted_scopes and logs warning if Gmail denied
//    - internal/integrations/google/gmail.go checks IsConnected() before sync
//    - Returns error: "Gmail access not authorized" if scope missing
//
// 5. Token Refresh:
//    - Refresh tokens include all granted scopes
//    - Existing users upgrading scope: refresh token remains valid
//    - No special handling needed (OAuth2 library handles this)

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// maskToken returns a masked version of the token for safe logging.
// Shows first 8 and last 4 characters, masks the middle.
// Example: "abc123def456ghi789" -> "abc123de****i789"
func maskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:8] + "****" + token[len(token)-4:]
}

// GoogleAuthHandler handles Google OAuth for authentication
type GoogleAuthHandler struct {
	pool         *pgxpool.Pool
	cfg          *config.Config
	oauthConfig  *oauth2.Config
	sessionCache *middleware.SessionCache // Redis session cache for horizontal scaling
}

type googleOAuthStateEntry struct {
	redirectAfter string
	expiresAt     time.Time
}

type googleOAuthStateStore struct {
	mu     sync.Mutex
	states map[string]googleOAuthStateEntry
}

var desktopGoogleOAuthStates = &googleOAuthStateStore{
	states: make(map[string]googleOAuthStateEntry),
}

func (s *googleOAuthStateStore) store(state, redirectAfter string, ttl time.Duration) {
	if state == "" {
		return
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, entry := range s.states {
		if now.After(entry.expiresAt) {
			delete(s.states, key)
		}
	}

	s.states[state] = googleOAuthStateEntry{
		redirectAfter: redirectAfter,
		expiresAt:     now.Add(ttl),
	}
}

func (s *googleOAuthStateStore) consume(state string) (googleOAuthStateEntry, bool) {
	if state == "" {
		return googleOAuthStateEntry{}, false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.states[state]
	if !ok {
		return googleOAuthStateEntry{}, false
	}
	delete(s.states, state)
	if now.After(entry.expiresAt) {
		return googleOAuthStateEntry{}, false
	}
	return entry, true
}

// GoogleUserInfo represents the user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// NewGoogleAuthHandler creates a new Google Auth handler
func NewGoogleAuthHandler(pool *pgxpool.Pool, cfg *config.Config, sessionCache *middleware.SessionCache) *GoogleAuthHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			// NOTE: gmail.readonly removed from login flow — it's a sensitive scope
			// that requires Google verification. Gmail sync will request its own
			// scope via incremental auth when the user enables email integration.
			// See: internal/handlers/oauth_integrations.go for integration-specific scopes
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleAuthHandler{
		sessionCache: sessionCache,
		pool:         pool,
		cfg:          cfg,
		oauthConfig:  oauthConfig,
	}
}

// isValidRedirectURL validates that the redirect URL is safe
func isValidRedirectURL(redirectURL string) bool {
	if redirectURL == "" {
		return false
	}

	// Allow internal paths (e.g., "/dashboard")
	if strings.HasPrefix(redirectURL, "/") && !strings.HasPrefix(redirectURL, "//") {
		return true
	}

	// Allow the desktop app's registered deep-link scheme. The Electron app
	// can't receive cookies set in the system browser, so after OAuth we hand
	// the session back via this custom protocol (see callback token append).
	if strings.HasPrefix(redirectURL, "businessos://") {
		return true
	}

	// Allow absolute URLs to known frontend origins (for dev + production)
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" && strings.HasPrefix(redirectURL, origin) {
			return true
		}
	}

	return false
}

// InitiateGoogleLogin starts the Google OAuth flow for login
func (h *GoogleAuthHandler) InitiateGoogleLogin(c *gin.Context) {
	if strings.TrimSpace(h.oauthConfig.ClientID) == "" || strings.TrimSpace(h.oauthConfig.ClientSecret) == "" {
		slog.Warn("InitiateGoogleLogin: Google OAuth is not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":  "google_oauth_not_configured",
			"error": "Google sign-in is not configured for this BusinessOS instance. Use email and password, or set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET.",
		})
		return
	}

	// Generate random state for CSRF protection
	state := generateRandomState()

	// Get redirect URL from query (for desktop app flow)
	redirectAfter := c.Query("redirect")

	// SECURITY: Validate redirect URL to prevent open redirect attacks
	if !isValidRedirectURL(redirectAfter) {
		slog.Warn("InitiateGoogleLogin: invalid redirect URL blocked", "redirect", redirectAfter)
		redirectAfter = "/dashboard"
	}

	// Store state in cookie with strict security settings
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	cookieDomain := os.Getenv("COOKIE_DOMAIN") // e.g. .businessos.dev in prod
	// In dev the initiate + callback are both on localhost, so a .businessos.dev
	// domain (or the Secure flag over http) would drop the cookie and break state
	// validation ("Invalid state parameter"). Force a host-only, non-secure cookie.
	if !isProduction {
		cookieDomain = ""
	}
	// SameSite=Lax so the state cookie survives the cross-site redirect back from
	// Google (the callback is a top-level GET). The domain scopes it across
	// app./businessos.dev so the initiate host and the callback host can differ.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", state, 600, "/", cookieDomain, isProduction, true)
	c.SetCookie("oauth_redirect", redirectAfter, 600, "/", cookieDomain, isProduction, true)
	if strings.HasPrefix(redirectAfter, "businessos://") {
		desktopGoogleOAuthStates.store(state, redirectAfter, 10*time.Minute)
	}

	// Force Google to show account picker every time (don't auto-login)
	authURL := h.oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "select_account"))

	// Redirect to Google OAuth
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleGoogleLoginCallback handles the OAuth callback for login
func (h *GoogleAuthHandler) HandleGoogleLoginCallback(c *gin.Context) {
	// Verify state
	state := c.Query("state")
	storedState, err := c.Cookie("oauth_state")

	// Get redirect URL
	redirectAfter, _ := c.Cookie("oauth_redirect")
	stateValid := err == nil && state == storedState
	if !stateValid {
		if entry, ok := desktopGoogleOAuthStates.consume(state); ok {
			stateValid = true
			redirectAfter = entry.redirectAfter
		}
	}
	if !stateValid {
		utils.RespondBadRequest(c, slog.Default(), "Invalid state parameter")
		return
	}

	// SECURITY: Validate redirect URL to prevent open redirect attacks
	if !isValidRedirectURL(redirectAfter) {
		slog.Warn("HandleGoogleLoginCallback: invalid redirect URL blocked", "redirect", redirectAfter)
		redirectAfter = "/dashboard"
	}

	// Check for error from Google
	if errMsg := c.Query("error"); errMsg != "" {
		c.Redirect(http.StatusTemporaryRedirect, "/?error="+errMsg)
		return
	}

	// Exchange code for tokens
	code := c.Query("code")
	token, err := h.oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "exchange OAuth code", err)
		return
	}

	// Check granted scopes (OAuth2 token may contain this info)
	// If gmail.readonly scope was not granted, log warning but continue
	grantedScopes := []string{}
	if scopeInterface := token.Extra("scope"); scopeInterface != nil {
		if scopeStr, ok := scopeInterface.(string); ok {
			grantedScopes = strings.Split(scopeStr, " ")
		}
	}

	hasGmailScope := false
	for _, scope := range grantedScopes {
		if strings.Contains(scope, "gmail.readonly") {
			hasGmailScope = true
			break
		}
	}

	if !hasGmailScope {
		slog.Warn("Gmail scope not granted by user",
			"granted_scopes", grantedScopes)
		// Continue anyway - user can still use basic auth without Gmail
	}

	// Get user info from Google
	userInfo, err := h.getGoogleUserInfo(token.AccessToken)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "get user info from Google", err)
		return
	}

	// Create or update user in database
	userID, _, err := h.upsertUser(c.Request.Context(), userInfo)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create/update user", err)
		return
	}

	// Provision or repair synchronously so the renderer cannot reach workspace-
	// scoped routes before membership and canonical roles exist.
	if _, wsErr := services.NewWorkspaceService(h.pool).EnsureDefaultWorkspace(c.Request.Context(), userID); wsErr != nil {
		utils.RespondInternalError(c, slog.Default(), "ensure default workspace", wsErr)
		return
	}

	// Create session
	sessionToken, err := h.createSession(c.Request.Context(), userID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create session", err)
		return
	}

	// Clear OAuth cookies and set session cookie (match the domain used when set)
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	if !isProduction {
		cookieDomain = ""
	}
	c.SetCookie("oauth_state", "", -1, "/", cookieDomain, isProduction, true)
	c.SetCookie("oauth_redirect", "", -1, "/", cookieDomain, isProduction, true)
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" {
		domain = "" // Current domain
	}

	sameSite := sessionCookieSameSite(isProduction)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "better-auth.session_token",
		Value:    sessionToken,
		Path:     "/",
		Domain:   domain,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	// Desktop deep-link: the Electron app launched from the system browser can
	// not read the cookie we just set (different process / no shared cookie jar),
	// so carry the session token in the deep-link URL. The app extracts it and
	// installs it as its own session cookie. Web (http/path) redirects keep
	// using the cookie above and never receive the token in the URL.
	if strings.HasPrefix(redirectAfter, "businessos://") {
		sep := "?"
		if strings.Contains(redirectAfter, "?") {
			sep = "&"
		}
		redirectAfter = redirectAfter + sep + "token=" + url.QueryEscape(sessionToken)

		// A browser can't navigate to a custom scheme via a redirect (the tab just
		// hangs "loading"). Return a tiny page that fires the deep link to open the
		// app, shows a clear "you can close this" message, and tries to auto-close.
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; navigate-to businessos:; base-uri 'none'; form-action 'none'")
		c.String(http.StatusOK, desktopAuthCompletePage(redirectAfter))
		return
	}

	// Redirect to app
	c.Redirect(http.StatusTemporaryRedirect, redirectAfter)
}

// desktopAuthCompletePage returns an HTML page that hands control back to the
// desktop app via the businessos:// deep link, then tells the user they can
// close the window (and attempts to close it automatically).
func desktopAuthCompletePage(deepLink string) string {
	escapedDeepLink := jsonEscapeForHTMLScript(deepLink)
	linkHref := html.EscapeString(deepLink)
	return `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Signed in - BusinessOS</title>
<style>
  *{box-sizing:border-box}
  html,body{height:100%;margin:0}
  body{display:grid;place-items:center;overflow:hidden;background:#050506;color:#f4f4f5;
       font:14px/1.5 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}
  body:before{content:"";position:fixed;inset:0;background:
       linear-gradient(rgba(255,255,255,.025) 1px,transparent 1px),
       linear-gradient(90deg,rgba(255,255,255,.025) 1px,transparent 1px);
       background-size:36px 36px;mask-image:radial-gradient(circle at 50% 42%,#000 0,#000 42%,transparent 74%)}
  body:after{content:"";position:fixed;inset:auto 12% 0;height:1px;background:linear-gradient(90deg,transparent,#22c55e,transparent);opacity:.58}
  .shell{position:relative;width:min(430px,calc(100vw - 36px));padding:1px;border-radius:18px;background:linear-gradient(135deg,rgba(255,255,255,.18),rgba(255,255,255,.04) 38%,rgba(34,197,94,.22));box-shadow:0 24px 90px rgba(0,0,0,.62)}
  .card{position:relative;overflow:hidden;border-radius:17px;background:linear-gradient(180deg,#151518,#0c0c0e);border:1px solid rgba(255,255,255,.08);padding:0}
  .bar{height:36px;display:flex;align-items:center;gap:8px;padding:0 14px;border-bottom:1px solid rgba(255,255,255,.07);background:rgba(255,255,255,.035)}
  .dot{width:10px;height:10px;border-radius:50%}.r{background:#ff5f57}.y{background:#febc2e}.g{background:#28c840}
  .bar-title{margin-left:auto;color:#71717a;font-size:11px;letter-spacing:.08em;text-transform:uppercase}
  .content{padding:34px 34px 30px;text-align:left}
  .brand{display:flex;align-items:center;gap:13px;margin-bottom:28px}
  .logo{width:44px;height:44px;border-radius:12px;display:grid;place-items:center;background:#f4f4f5;color:#0b0b0c;box-shadow:0 0 0 1px rgba(255,255,255,.16),0 14px 34px rgba(255,255,255,.08)}
  .word{font-size:22px;font-weight:850;letter-spacing:.16em;line-height:1}.word span{color:#73737a;font-weight:350;letter-spacing:.02em}
  .status{display:inline-flex;align-items:center;gap:8px;color:#34d399;font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;margin-bottom:12px}
  .pulse{width:7px;height:7px;border-radius:50%;background:#34d399;box-shadow:0 0 0 6px rgba(52,211,153,.12)}
  h1{font:800 25px/1.12 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;letter-spacing:-.02em;margin:0 0 8px}
  p{margin:0;color:#a1a1aa;font-size:13px;max-width:34ch}
  .actions{display:flex;align-items:center;gap:12px;margin-top:24px}
  a{display:inline-flex;align-items:center;justify-content:center;height:40px;padding:0 15px;border-radius:10px;background:#f4f4f5;color:#0b0b0c;font-weight:760;text-decoration:none;box-shadow:0 10px 32px rgba(255,255,255,.12)}
  .hint{color:#71717a;font-size:12px}
</style></head>
<body><main class="shell"><section class="card" aria-labelledby="title">
  <div class="bar"><span class="dot r"></span><span class="dot y"></span><span class="dot g"></span><span class="bar-title">secure handoff</span></div>
  <div class="content">
    <div class="brand">
      <div class="logo" aria-hidden="true">
        <svg width="28" height="28" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M14 24C14 18.477 18.477 14 24 14C29.523 14 34 18.477 34 24C34 29.523 29.523 34 24 34C18.477 34 14 29.523 14 24Z" stroke="currentColor" stroke-width="3"/>
          <path d="M24 18V24L28 28" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </div>
      <div class="word">BUSINESS<span>OS</span></div>
    </div>
    <div class="status"><span class="pulse"></span>Signed in</div>
    <h1 id="title">Session handed off.</h1>
    <p>BusinessOS is opening with your authenticated session.</p>
    <div class="actions"><a href="` + linkHref + `">Open BusinessOS</a><span class="hint">This tab can close.</span></div>
  </div>
</section></main>
<script>
  var deepLink = ` + escapedDeepLink + `;
  try { window.location.replace(deepLink); } catch (e) {}
  setTimeout(function(){ try { window.location.href = deepLink; } catch (e) {} }, 250);
  setTimeout(function(){ try { window.close(); } catch (e) {} }, 900);
</script>
</body></html>`
}

func jsonEscapeForHTMLScript(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return strings.ReplaceAll(string(encoded), "</", "<\\/")
}

// getGoogleUserInfo fetches user info from Google API
func (h *GoogleAuthHandler) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// upsertUser creates or updates a user based on Google info.
// Returns (userID, isNewUser, error).
func (h *GoogleAuthHandler) upsertUser(ctx context.Context, info *GoogleUserInfo) (string, bool, error) {
	// Check if user exists by email
	var existingID string
	err := h.pool.QueryRow(ctx, `
		SELECT id FROM "user" WHERE email = $1
	`, info.Email).Scan(&existingID)

	if err == nil {
		// User exists, update their info
		_, err = h.pool.Exec(ctx, `
			UPDATE "user"
			SET name = $1, image = $2, "emailVerified" = $3, "updatedAt" = NOW()
			WHERE id = $4
		`, info.Name, info.Picture, info.VerifiedEmail, existingID)
		if err != nil {
			return "", false, fmt.Errorf("failed to update user: %w", err)
		}
		return existingID, false, nil
	}

	// Create new user
	userID := generateUserID()
	_, err = h.pool.Exec(ctx, `
		INSERT INTO "user" (id, name, email, "emailVerified", image, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, userID, info.Name, info.Email, info.VerifiedEmail, info.Picture)

	if err != nil {
		return "", false, fmt.Errorf("failed to create user: %w", err)
	}

	return userID, true, nil
}

// createSession creates a new session for the user
func (h *GoogleAuthHandler) createSession(ctx context.Context, userID string) (string, error) {
	sessionToken := generateSessionToken()
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	_, err := h.pool.Exec(ctx, `
		INSERT INTO session (id, "userId", token, "expiresAt", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, sessionID, userID, sessionToken, expiresAt)

	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionToken, nil
}

// GetCurrentSession returns the current user session
func (h *GoogleAuthHandler) GetCurrentSession(c *gin.Context) {
	sessionCookie, err := c.Cookie("better-auth.session_token")
	if err != nil || sessionCookie == "" {
		c.JSON(http.StatusOK, gin.H{
			"user":    nil,
			"session": nil,
		})
		return
	}

	// SECURITY: Never log session tokens, even masked versions in debug mode
	// Session tokens are sensitive credentials that can be used for account takeover

	// URL-decode the cookie (consistent with auth middleware)
	sessionCookie, err = url.QueryUnescape(sessionCookie)
	if err != nil {
		slog.Warn("get_session: URL decode failed", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"user":    nil,
			"session": nil,
		})
		return
	}

	// Strip signature part if present (consistent with auth middleware)
	sessionToken := sessionCookie
	if idx := strings.Index(sessionCookie, "."); idx != -1 {
		sessionToken = sessionCookie[:idx]
	}

	// Look up session
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var userID, userName, userEmail, sessionID, platformRole string
	var userImage *string
	var emailVerified bool
	var sessionExpiresAt time.Time
	var userCreatedAt time.Time

	err = h.pool.QueryRow(ctx, `
		SELECT u.id, u.name, u.email, u."emailVerified", u.image, u."createdAt", COALESCE(u.platform_role, 'user'), s.id, s."expiresAt"
		FROM session s
		JOIN "user" u ON s."userId" = u.id
		WHERE s.token = $1 AND s."expiresAt" > NOW()
	`, sessionToken).Scan(
		&userID, &userName, &userEmail, &emailVerified, &userImage, &userCreatedAt, &platformRole, &sessionID, &sessionExpiresAt,
	)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"user":    nil,
			"session": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":            userID,
			"name":          userName,
			"email":         userEmail,
			"emailVerified": emailVerified,
			"image":         userImage,
			"createdAt":     userCreatedAt,
			"platform_role": platformRole,
		},
		"session": gin.H{
			"id":        sessionID,
			"userId":    userID,
			"expiresAt": sessionExpiresAt,
		},
	})
}

// ServeCurrentUserAvatar returns the signed-in user's Google profile image from
// the local API origin. Electron can then render the image without depending on
// a direct third-party image request from the renderer process.
func (h *GoogleAuthHandler) ServeCurrentUserAvatar(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil || user.Image == nil || strings.TrimSpace(*user.Image) == "" {
		c.Status(http.StatusNotFound)
		return
	}

	imageURL := strings.TrimSpace(*user.Image)
	if !isTrustedGoogleAvatarURL(imageURL) {
		slog.Warn("profile avatar uses unsupported image host", "user_id", user.ID)
		c.Status(http.StatusNotFound)
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, imageURL, nil)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if isTrustedGoogleAvatarURL(req.URL.String()) {
				return nil
			}
			return fmt.Errorf("redirected to an unsupported avatar host")
		},
	}

	response, err := client.Do(req)
	if err != nil {
		slog.Warn("profile avatar fetch failed", "user_id", user.ID, "error", err)
		c.Status(http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		c.Status(http.StatusBadGateway)
		return
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.Status(http.StatusBadGateway)
		return
	}

	const maxAvatarBytes = 5 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAvatarBytes+1))
	if err != nil || len(body) > maxAvatarBytes {
		c.Status(http.StatusBadGateway)
		return
	}

	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Vary", "Cookie")
	c.Data(http.StatusOK, contentType, body)
}

func isTrustedGoogleAvatarURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return false
	}

	return strings.EqualFold(parsed.Hostname(), "lh3.googleusercontent.com")
}

// Logout clears the current session
func (h *GoogleAuthHandler) Logout(c *gin.Context) {
	sessionCookie, err := c.Cookie("better-auth.session_token")
	if err == nil && sessionCookie != "" {
		// URL decode the cookie
		sessionCookie, _ = url.QueryUnescape(sessionCookie)

		// Extract token (before HMAC signature dot)
		sessionToken := sessionCookie
		if idx := strings.Index(sessionCookie, "."); idx != -1 {
			sessionToken = sessionCookie[:idx]
		}

		// Invalidate Redis cache first (if available)
		cacheInvalidated := false
		if h.sessionCache != nil {
			if err := h.sessionCache.Invalidate(c.Request.Context(), sessionToken); err != nil {
				slog.Warn("Logout: cache invalidation error", "error", err)
			} else {
				cacheInvalidated = true
			}
		}

		// Delete session from database
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		_, err := h.pool.Exec(ctx, `DELETE FROM session WHERE token = $1`, sessionToken)
		if err != nil {
			slog.Error("Logout: database session deletion failed", "error", err)
			// SECURITY: If DB delete fails but cache was invalidated, session is partially logged out
			// This is a security concern - the session may still be valid in DB but not in cache
			if cacheInvalidated {
				slog.Warn("Logout: inconsistent state - cache invalidated but DB delete failed")
			}
			utils.RespondInternalError(c, slog.Default(), "logout session", err)
			return
		}
	}

	// Clear cookie with strict security configuration (must match how it was set)
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" {
		domain = "" // Current domain
	}

	sameSite := sessionCookieSameSite(isProduction)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "better-auth.session_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// LogoutAllSessions invalidates all sessions for the current user
// This is a critical security feature for:
// - Password changes
// - Suspected account compromise
// - Permission/role changes
// - User-initiated "logout from all devices"
func (h *GoogleAuthHandler) LogoutAllSessions(c *gin.Context) {
	// Get current user from context (set by auth middleware)
	userInterface, exists := c.Get(middleware.UserContextKey)
	if !exists {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	user, ok := userInterface.(*middleware.BetterAuthUser)
	if !ok {
		utils.RespondInternalError(c, slog.Default(), "get user context", fmt.Errorf("invalid user context type"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Invalidate all Redis cached sessions first
	if h.sessionCache != nil {
		if err := h.sessionCache.InvalidateUserSessions(ctx, user.ID); err != nil {
			slog.Warn("LogoutAllSessions: cache invalidation error", "user_id", user.ID, "error", err)
			// Continue to database cleanup even if cache fails
		}
	}

	// Delete all sessions from database
	result, err := h.pool.Exec(ctx, `DELETE FROM session WHERE "userId" = $1`, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "invalidate sessions", err)
		return
	}

	rowsAffected := result.RowsAffected()
	slog.Info("LogoutAllSessions: sessions invalidated", "sessions_deleted", rowsAffected, "user_id", user.ID)

	// Clear current session cookie with strict security configuration (must match how it was set)
	isProduction := os.Getenv("ENVIRONMENT") == "production"
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" {
		domain = "" // Current domain
	}

	sameSite := sessionCookieSameSite(isProduction)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "better-auth.session_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: sameSite,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":          "All sessions invalidated",
		"sessions_removed": rowsAffected,
	})
}

// Helper functions
func generateRandomState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// CRITICAL: rand.Read failure means cryptographic randomness is compromised
		panic(fmt.Sprintf("crypto/rand.Read failed: %v - system entropy exhausted", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generateUserID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// CRITICAL: rand.Read failure means cryptographic randomness is compromised
		panic(fmt.Sprintf("crypto/rand.Read failed: %v - system entropy exhausted", err))
	}
	return base64.URLEncoding.EncodeToString(b)[:22]
}

func generateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// CRITICAL: rand.Read failure means cryptographic randomness is compromised
		panic(fmt.Sprintf("crypto/rand.Read failed: %v - system entropy exhausted", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// CRITICAL: rand.Read failure means cryptographic randomness is compromised
		panic(fmt.Sprintf("crypto/rand.Read failed: %v - system entropy exhausted", err))
	}
	return base64.URLEncoding.EncodeToString(b)[:22]
}
