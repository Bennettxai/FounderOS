package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenLength is the length of CSRF tokens in bytes (32 bytes = 256 bits)
	CSRFTokenLength = 32

	// CSRFCookieName is the name of the cookie storing the CSRF token
	CSRFCookieName = "csrf_token"

	// CSRFHeaderName is the name of the header containing the CSRF token
	CSRFHeaderName = "X-CSRF-Token"

	// CSRFTokenKey is the context key for storing the CSRF token
	CSRFTokenKey = "csrf_token"
)

// CSRFConfig holds CSRF middleware configuration
type CSRFConfig struct {
	// SkipMethods are HTTP methods that skip CSRF validation (default: GET, HEAD, OPTIONS)
	SkipMethods []string

	// TokenLength is the length of CSRF tokens in bytes (default: 32)
	TokenLength int

	// CookieName is the name of the CSRF cookie (default: "csrf_token")
	CookieName string

	// HeaderName is the name of the CSRF header (default: "X-CSRF-Token")
	HeaderName string

	// CookiePath is the path for the CSRF cookie (default: "/")
	CookiePath string

	// CookieDomain is the domain for the CSRF cookie (optional)
	CookieDomain string

	// CookieSecure enables Secure flag on cookie (default: true in production)
	CookieSecure bool

	// CookieSameSite sets SameSite attribute (default: Strict)
	CookieSameSite http.SameSite

	// Skipper defines a function to skip CSRF check for certain routes
	// Return true to skip CSRF validation for the request
	Skipper func(*gin.Context) bool

	// ErrorHandler is called when CSRF validation fails (optional)
	ErrorHandler func(*gin.Context, error)
}

// DefaultCSRFConfig returns the default CSRF configuration
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		SkipMethods:    []string{"GET", "HEAD", "OPTIONS"},
		TokenLength:    CSRFTokenLength,
		CookieName:     CSRFCookieName,
		HeaderName:     CSRFHeaderName,
		CookiePath:     "/",
		CookieSecure:   true,
		CookieSameSite: http.SameSiteStrictMode,
	}
}

// CSRF returns a middleware that provides CSRF protection using the Double Submit Cookie pattern
//
// How it works:
// 1. Generates a random token and stores it in a cookie
// 2. Client must send the same token in a custom header (X-CSRF-Token)
// 3. Middleware validates that cookie token == header token
// 4. Protects against CSRF attacks because attacker cannot read/set custom headers cross-origin
//
// Security properties:
// - Token is cryptographically random (32 bytes = 256 bits)
// - Cookie uses SameSite=Strict (additional protection)
// - Cookie is HttpOnly=false (client needs to read it for header)
// - Token rotates on each request (fresh token per request)
//
// Note: This works in combination with SameSite=Strict cookies for session tokens,
// providing defense-in-depth against CSRF attacks.
func CSRF(config ...CSRFConfig) gin.HandlerFunc {
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Ensure skip methods are uppercase
	for i, method := range cfg.SkipMethods {
		cfg.SkipMethods[i] = strings.ToUpper(method)
	}

	return func(c *gin.Context) {
		// Check if request should skip CSRF validation via Skipper function
		if cfg.Skipper != nil && cfg.Skipper(c) {
			c.Next()
			return
		}

		method := strings.ToUpper(c.Request.Method)

		// Skip CSRF validation for safe methods (GET, HEAD, OPTIONS)
		if containsMethod(cfg.SkipMethods, method) {
			// DO NOT generate new tokens automatically for GET requests!
			// This was causing token rotation on EVERY GET (including favicon.ico)
			// which would invalidate the token the frontend just retrieved.
			// Only the /api/auth/csrf endpoint should generate tokens.

			// Try to read existing token from cookie
			existingToken, _ := c.Cookie(cfg.CookieName)
			if existingToken != "" {
				c.Set(CSRFTokenKey, existingToken)
			}
			// If no token exists, that's OK for GET requests

			c.Next()
			return
		}

		// For state-changing methods (POST, PUT, PATCH, DELETE), validate CSRF token
		cookieToken, err := c.Cookie(cfg.CookieName)
		if err != nil {
			// No CSRF cookie found
			slog.Warn("csrf_validation_failed",
				"reason", "missing_cookie",
				"method", method,
				"path", c.Request.URL.Path,
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent())

			if cfg.ErrorHandler != nil {
				cfg.ErrorHandler(c, err)
				return
			}

			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing",
				"code":  "csrf_token_missing",
			})
			c.Abort()
			return
		}

		// Get token from header
		headerToken := c.GetHeader(cfg.HeaderName)
		if headerToken == "" {
			// No CSRF header found
			slog.Warn("csrf_validation_failed",
				"reason", "missing_header",
				"method", method,
				"path", c.Request.URL.Path,
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent())

			if cfg.ErrorHandler != nil {
				cfg.ErrorHandler(c, nil)
				return
			}

			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token missing in header",
				"code":  "csrf_header_missing",
			})
			c.Abort()
			return
		}

		// Validate tokens match (constant-time comparison to prevent timing attacks)
		if !secureCompare(cookieToken, headerToken) {
			// Token mismatch - potential CSRF attack
			slog.Warn("csrf_validation_failed",
				"reason", "token_mismatch",
				"method", method,
				"path", c.Request.URL.Path,
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent(),
				"has_cookie", cookieToken != "",
				"has_header", headerToken != "")

			if cfg.ErrorHandler != nil {
				cfg.ErrorHandler(c, nil)
				return
			}

			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token invalid",
				"code":  "csrf_token_invalid",
			})
			c.Abort()
			return
		}

		// CSRF validation passed
		slog.Debug("csrf_validation_success",
			"method", method,
			"path", c.Request.URL.Path)

		// DISABLED: Token rotation causes mismatch with frontend
		// The frontend reads the token from cookie before each request,
		// but if we rotate it here, the next request will have a stale token.
		// CSRF tokens don't need to be rotated on every request - only on login/logout.
		//
		// newToken := generateCSRFToken(cfg.TokenLength)
		// setCSRFCookie(c, newToken, cfg)
		// c.Set(CSRFTokenKey, newToken)

		// Keep the existing token
		c.Set(CSRFTokenKey, cookieToken)

		c.Next()
	}
}

// generateCSRFToken generates a cryptographically secure random token.
// Returns an error if the OS RNG fails so callers fail the request rather than
// handing out a constant, predictable token that would defeat CSRF entirely.
func generateCSRFToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// setCSRFCookie sets the CSRF token cookie
func setCSRFCookie(c *gin.Context, token string, cfg CSRFConfig) {
	// Use http.SetCookie directly to set SameSite attribute
	cookie := &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     cfg.CookiePath,
		Domain:   cfg.CookieDomain,
		MaxAge:   0, // Session cookie (expires when browser closes)
		Secure:   cfg.CookieSecure,
		HttpOnly: false, // Client needs to read token for header
		SameSite: cfg.CookieSameSite,
	}
	http.SetCookie(c.Writer, cookie)
}

// secureCompare performs constant-time comparison to prevent timing attacks
func secureCompare(a, b string) bool {
	// Convert strings to byte slices
	aBytes := []byte(a)
	bBytes := []byte(b)

	// If lengths differ, tokens don't match
	if len(aBytes) != len(bBytes) {
		return false
	}

	// Constant-time comparison
	var result byte
	for i := 0; i < len(aBytes); i++ {
		result |= aBytes[i] ^ bBytes[i]
	}

	return result == 0
}

// containsMethod checks if a method is in the skip list
func containsMethod(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// GetCSRFToken retrieves the CSRF token from the context
// Useful for handlers that need to include the token in responses
func GetCSRFToken(c *gin.Context) string {
	if token, exists := c.Get(CSRFTokenKey); exists {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

// CSRFTokenEndpoint is a handler that returns the current CSRF token
// Useful for SPA applications that need to retrieve the token
// Accepts optional config to match the CSRF middleware configuration
func CSRFTokenEndpoint(config ...CSRFConfig) gin.HandlerFunc {
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	} else {
		// Auto-detect development mode and disable Secure flag for HTTP
		// Check GIN_MODE or ENVIRONMENT - if not "release"/"production", it's dev
		ginMode := strings.ToLower(strings.TrimSpace(gin.Mode()))
		if ginMode != "release" {
			cfg.CookieSecure = false
		}
	}

	return func(c *gin.Context) {
		// First, try to get token from cookie (existing token)
		token, err := c.Cookie(cfg.CookieName)

		// Log token retrieval status (never log token values or cookies)
		slog.Debug("csrf_token_endpoint_called",
			"cookie_name", cfg.CookieName,
			"token_found", token != "")

		// If no token in cookie, generate a new one
		if err != nil || token == "" {
			newToken, genErr := generateCSRFToken(CSRFTokenLength)
			if genErr != nil {
				slog.Error("failed to generate CSRF token", "error", genErr)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			token = newToken
			setCSRFCookie(c, token, cfg)
			slog.Debug("csrf_token_endpoint_new_token_generated")
		} else {
			slog.Debug("csrf_token_endpoint_existing_token_returned")
		}

		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
		})
	}
}
