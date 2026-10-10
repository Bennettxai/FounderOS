package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupCSRFTestContext creates a Gin test context for CSRF middleware testing
func setupCSRFTestContext(method string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(method, "/test", nil)
	return c, w
}

// TestCSRF_SafeMethods verifies CSRF middleware allows safe methods without token
func TestCSRF_SafeMethods(t *testing.T) {
	safeMethods := []string{"GET", "HEAD", "OPTIONS"}

	for _, method := range safeMethods {
		t.Run(method+"_NoExistingCookie", func(t *testing.T) {
			c, _ := setupCSRFTestContext(method)

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was NOT aborted (safe methods always pass)
			assert.False(t, c.IsAborted(), "CSRF should not abort %s requests", method)

			// Assert: no token in context (no cookie existed, no new token generated)
			_, exists := c.Get(CSRFTokenKey)
			assert.False(t, exists, "CSRF token should NOT be auto-generated for %s without cookie", method)
		})

		t.Run(method+"_WithExistingCookie", func(t *testing.T) {
			c, _ := setupCSRFTestContext(method)

			// Set existing CSRF cookie
			c.Request.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: "existing-csrf-token-123",
			})

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was NOT aborted
			assert.False(t, c.IsAborted(), "CSRF should not abort %s requests", method)

			// Assert: existing token preserved in context
			token, exists := c.Get(CSRFTokenKey)
			assert.True(t, exists, "Existing CSRF token should be preserved in context for %s", method)
			assert.Equal(t, "existing-csrf-token-123", token, "Token should match cookie value")
		})
	}
}

// TestCSRF_StateMethods_MissingToken verifies CSRF middleware blocks state-changing methods without token
func TestCSRF_StateMethods_MissingToken(t *testing.T) {
	stateMethods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range stateMethods {
		t.Run(method+"_MissingCookie", func(t *testing.T) {
			c, w := setupCSRFTestContext(method)

			// No CSRF cookie set

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was aborted with 403
			assert.True(t, c.IsAborted(), "CSRF should abort %s request without cookie", method)
			assert.Equal(t, http.StatusForbidden, w.Code, "Response code should be 403 for %s without cookie", method)
		})

		t.Run(method+"_MissingHeader", func(t *testing.T) {
			c, w := setupCSRFTestContext(method)

			// Set CSRF cookie but no header
			c.Request.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: "test-token-123",
			})

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was aborted with 403
			assert.True(t, c.IsAborted(), "CSRF should abort %s request without header", method)
			assert.Equal(t, http.StatusForbidden, w.Code, "Response code should be 403 for %s without header", method)
		})
	}
}

// TestCSRF_StateMethods_TokenMismatch verifies CSRF middleware blocks requests with mismatched tokens
func TestCSRF_StateMethods_TokenMismatch(t *testing.T) {
	stateMethods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range stateMethods {
		t.Run(method, func(t *testing.T) {
			c, w := setupCSRFTestContext(method)

			// Set CSRF cookie with one token
			c.Request.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: "cookie-token-123",
			})

			// Set CSRF header with different token
			c.Request.Header.Set(CSRFHeaderName, "header-token-456")

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was aborted with 403
			assert.True(t, c.IsAborted(), "CSRF should abort %s request with mismatched tokens", method)
			assert.Equal(t, http.StatusForbidden, w.Code, "Response code should be 403 for %s with mismatched tokens", method)
		})
	}
}

// TestCSRF_StateMethods_ValidToken verifies CSRF middleware allows requests with valid matching tokens
func TestCSRF_StateMethods_ValidToken(t *testing.T) {
	stateMethods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range stateMethods {
		t.Run(method, func(t *testing.T) {
			c, _ := setupCSRFTestContext(method)

			// Set matching CSRF cookie and header
			token := "valid-token-abc123"
			c.Request.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: token,
			})
			c.Request.Header.Set(CSRFHeaderName, token)

			// Execute middleware
			middleware := CSRF()
			middleware(c)

			// Assert: request was NOT aborted
			assert.False(t, c.IsAborted(), "CSRF should not abort %s request with valid token", method)

			// Assert: existing token preserved in context (no rotation)
			contextToken, exists := c.Get(CSRFTokenKey)
			assert.True(t, exists, "CSRF token should be set in context for %s", method)
			assert.NotEmpty(t, contextToken, "CSRF token should not be empty for %s", method)
			assert.Equal(t, token, contextToken, "CSRF token should be preserved (no rotation) for %s", method)
		})
	}
}

// TestCSRF_CustomConfig verifies CSRF middleware respects custom configuration
func TestCSRF_CustomConfig(t *testing.T) {
	config := CSRFConfig{
		SkipMethods:    []string{"GET", "POST"}, // Custom: skip POST as well
		TokenLength:    16,                      // Custom: shorter token
		CookieName:     "custom_csrf",
		HeaderName:     "X-Custom-CSRF",
		CookiePath:     "/api",
		CookieSecure:   false, // For testing
		CookieSameSite: http.SameSiteLaxMode,
	}

	t.Run("CustomSkipMethods", func(t *testing.T) {
		c, _ := setupCSRFTestContext("POST")

		// Execute middleware with custom config
		middleware := CSRF(config)
		middleware(c)

		// Assert: POST was not aborted (custom skip)
		assert.False(t, c.IsAborted(), "CSRF should not abort POST with custom skip config")
	})

	t.Run("CustomCookieName", func(t *testing.T) {
		c, _ := setupCSRFTestContext("GET")

		// Set existing cookie with custom name
		c.Request.AddCookie(&http.Cookie{
			Name:  "custom_csrf",
			Value: "custom-token-value",
		})

		// Execute middleware with custom config
		middleware := CSRF(config)
		middleware(c)

		// Assert: request was not aborted
		assert.False(t, c.IsAborted(), "GET should not be aborted")

		// Assert: custom cookie token preserved in context
		token, exists := c.Get(CSRFTokenKey)
		assert.True(t, exists, "Custom cookie token should be preserved in context")
		assert.Equal(t, "custom-token-value", token, "Token should match custom cookie value")
	})

	t.Run("CustomHeaderName", func(t *testing.T) {
		c, w := setupCSRFTestContext("PUT")

		// Set CSRF cookie with custom name
		token := "custom-token-123"
		c.Request.AddCookie(&http.Cookie{
			Name:  "custom_csrf",
			Value: token,
		})

		// Missing custom header
		// Execute middleware
		middleware := CSRF(config)
		middleware(c)

		// Assert: aborted because custom header is missing
		assert.True(t, c.IsAborted(), "CSRF should abort PUT without custom header")
		assert.Equal(t, http.StatusForbidden, w.Code, "Response code should be 403")
	})
}

// TestGenerateCSRFToken verifies token generation produces valid tokens
func TestGenerateCSRFToken(t *testing.T) {
	token1, err1 := generateCSRFToken(CSRFTokenLength)
	token2, err2 := generateCSRFToken(CSRFTokenLength)
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Assert: tokens are not empty
	assert.NotEmpty(t, token1, "Generated token should not be empty")
	assert.NotEmpty(t, token2, "Generated token should not be empty")

	// Assert: tokens are unique (cryptographically random)
	assert.NotEqual(t, token1, token2, "Generated tokens should be unique")

	// Assert: tokens are base64 encoded (length should be ~43 chars for 32 bytes)
	assert.Greater(t, len(token1), 40, "Token should be properly base64 encoded")
	assert.Greater(t, len(token2), 40, "Token should be properly base64 encoded")
}

// TestSecureCompare verifies constant-time comparison
func TestSecureCompare(t *testing.T) {
	t.Run("MatchingStrings", func(t *testing.T) {
		result := secureCompare("abc123", "abc123")
		assert.True(t, result, "secureCompare should return true for matching strings")
	})

	t.Run("DifferentStrings", func(t *testing.T) {
		result := secureCompare("abc123", "xyz789")
		assert.False(t, result, "secureCompare should return false for different strings")
	})

	t.Run("DifferentLengths", func(t *testing.T) {
		result := secureCompare("abc", "abcdef")
		assert.False(t, result, "secureCompare should return false for different lengths")
	})

	t.Run("EmptyStrings", func(t *testing.T) {
		result := secureCompare("", "")
		assert.True(t, result, "secureCompare should return true for empty strings")
	})
}

// TestGetCSRFToken verifies GetCSRFToken utility function
func TestGetCSRFToken(t *testing.T) {
	t.Run("TokenExists", func(t *testing.T) {
		c, _ := setupCSRFTestContext("GET")
		expectedToken := "test-token-123"
		c.Set(CSRFTokenKey, expectedToken)

		token := GetCSRFToken(c)
		assert.Equal(t, expectedToken, token, "GetCSRFToken should return token from context")
	})

	t.Run("TokenMissing", func(t *testing.T) {
		c, _ := setupCSRFTestContext("GET")

		token := GetCSRFToken(c)
		assert.Empty(t, token, "GetCSRFToken should return empty string when token is missing")
	})

	t.Run("InvalidTokenType", func(t *testing.T) {
		c, _ := setupCSRFTestContext("GET")
		c.Set(CSRFTokenKey, 12345) // Wrong type

		token := GetCSRFToken(c)
		assert.Empty(t, token, "GetCSRFToken should return empty string for invalid type")
	})
}

// TestCSRFTokenEndpoint verifies CSRFTokenEndpoint handler
func TestCSRFTokenEndpoint(t *testing.T) {
	t.Run("TokenExists", func(t *testing.T) {
		c, w := setupCSRFTestContext("GET")
		expectedToken := "existing-token-123"

		// Set token as request cookie (endpoint reads from cookie)
		c.Request.AddCookie(&http.Cookie{
			Name:  CSRFCookieName,
			Value: expectedToken,
		})

		handler := CSRFTokenEndpoint()
		handler(c)

		// Assert: response is 200 OK
		assert.Equal(t, http.StatusOK, w.Code, "Response code should be 200")

		// Assert: response body contains token
		assert.Contains(t, w.Body.String(), "csrf_token", "Response should contain csrf_token field")
		assert.Contains(t, w.Body.String(), expectedToken, "Response should contain the existing token")
	})

	t.Run("TokenMissing_Generated", func(t *testing.T) {
		c, w := setupCSRFTestContext("GET")

		handler := CSRFTokenEndpoint()
		handler(c)

		// Assert: response is 200 OK
		assert.Equal(t, http.StatusOK, w.Code, "Response code should be 200")

		// Assert: response body contains token
		assert.Contains(t, w.Body.String(), "csrf_token", "Response should contain csrf_token field")

		// Assert: new cookie was set
		cookies := w.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == CSRFCookieName {
				csrfCookie = cookie
				break
			}
		}
		assert.NotNil(t, csrfCookie, "CSRF cookie should be set when generating new token")
	})
}

// TestCSRF_ErrorHandler verifies custom error handler is called
func TestCSRF_ErrorHandler(t *testing.T) {
	errorHandlerCalled := false
	config := CSRFConfig{
		SkipMethods: []string{"GET", "HEAD", "OPTIONS"},
		ErrorHandler: func(c *gin.Context, err error) {
			errorHandlerCalled = true
			c.JSON(http.StatusTeapot, gin.H{"custom_error": "CSRF failed"})
			c.Abort()
		},
	}

	c, w := setupCSRFTestContext("POST")
	// No CSRF cookie set

	// Execute middleware with custom error handler
	middleware := CSRF(config)
	middleware(c)

	// Assert: custom error handler was called
	assert.True(t, errorHandlerCalled, "Custom error handler should be called")

	// Assert: custom response was sent
	assert.Equal(t, http.StatusTeapot, w.Code, "Response code should be 418 (custom)")
	assert.Contains(t, w.Body.String(), "custom_error", "Response should contain custom error message")
}

// TestCSRF_Skipper verifies Skipper function allows bypassing CSRF validation
func TestCSRF_Skipper(t *testing.T) {
	t.Run("SkipWebhook", func(t *testing.T) {
		config := CSRFConfig{
			SkipMethods: []string{"GET", "HEAD", "OPTIONS"},
			Skipper: func(c *gin.Context) bool {
				// Skip CSRF for webhook endpoints
				return strings.HasPrefix(c.Request.URL.Path, "/webhooks/")
			},
		}

		c, _ := setupCSRFTestContext("POST")
		c.Request.URL.Path = "/webhooks/stripe"

		// Execute middleware with skipper
		middleware := CSRF(config)
		middleware(c)

		// Assert: request was NOT aborted (skipper returned true)
		assert.False(t, c.IsAborted(), "CSRF should not abort webhook POST with skipper")
	})

	t.Run("SkipHealthCheck", func(t *testing.T) {
		config := CSRFConfig{
			SkipMethods: []string{"GET", "HEAD", "OPTIONS"},
			Skipper: func(c *gin.Context) bool {
				// Skip CSRF for health check endpoints
				return c.Request.URL.Path == "/health" || c.Request.URL.Path == "/ready"
			},
		}

		c, _ := setupCSRFTestContext("POST")
		c.Request.URL.Path = "/health"

		// Execute middleware with skipper
		middleware := CSRF(config)
		middleware(c)

		// Assert: request was NOT aborted (skipper returned true)
		assert.False(t, c.IsAborted(), "CSRF should not abort health POST with skipper")
	})

	t.Run("NoSkip", func(t *testing.T) {
		config := CSRFConfig{
			SkipMethods: []string{"GET", "HEAD", "OPTIONS"},
			Skipper: func(c *gin.Context) bool {
				// Only skip webhooks
				return strings.HasPrefix(c.Request.URL.Path, "/webhooks/")
			},
		}

		c, w := setupCSRFTestContext("POST")
		c.Request.URL.Path = "/api/users"

		// Execute middleware with skipper
		middleware := CSRF(config)
		middleware(c)

		// Assert: request WAS aborted (skipper returned false, no token)
		assert.True(t, c.IsAborted(), "CSRF should abort non-webhook POST without token")
		assert.Equal(t, http.StatusForbidden, w.Code, "Response code should be 403")
	})
}
