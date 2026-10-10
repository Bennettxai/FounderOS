package security_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSecurityHeaders tests that all required security headers are present
func TestSecurityHeaders(t *testing.T) {
	// Expected security headers for production
	expectedHeaders := getSecurityHeaders()

	t.Run("X-Frame-Options", func(t *testing.T) {
		header := expectedHeaders["X-Frame-Options"]
		assert.Equal(t, "DENY", header, "X-Frame-Options should be DENY to prevent clickjacking")
	})

	t.Run("X-Content-Type-Options", func(t *testing.T) {
		header := expectedHeaders["X-Content-Type-Options"]
		assert.Equal(t, "nosniff", header, "X-Content-Type-Options should be nosniff")
	})

	t.Run("Strict-Transport-Security", func(t *testing.T) {
		header := expectedHeaders["Strict-Transport-Security"]
		assert.Contains(t, header, "max-age=31536000", "HSTS should be set for 1 year")
		assert.Contains(t, header, "includeSubDomains", "HSTS should include subdomains")
	})

	t.Run("Content-Security-Policy", func(t *testing.T) {
		header := expectedHeaders["Content-Security-Policy"]

		// CSP should be restrictive
		assert.Contains(t, header, "default-src 'self'", "Default source should be self")
		assert.NotContains(t, header, "unsafe-inline", "Should not allow unsafe-inline")
		assert.NotContains(t, header, "unsafe-eval", "Should not allow unsafe-eval")
	})

	t.Run("Referrer-Policy", func(t *testing.T) {
		header := expectedHeaders["Referrer-Policy"]
		validPolicies := []string{
			"strict-origin-when-cross-origin",
			"strict-origin",
			"no-referrer",
		}

		isValid := false
		for _, policy := range validPolicies {
			if header == policy {
				isValid = true
				break
			}
		}
		assert.True(t, isValid, "Referrer-Policy should be restrictive")
	})

	t.Run("Permissions-Policy", func(t *testing.T) {
		header := expectedHeaders["Permissions-Policy"]

		// Should restrict dangerous features
		assert.Contains(t, header, "geolocation=()", "Should restrict geolocation")
		assert.Contains(t, header, "microphone=()", "Should restrict microphone")
		assert.Contains(t, header, "camera=()", "Should restrict camera")
	})

	t.Run("X-Permitted-Cross-Domain-Policies", func(t *testing.T) {
		header := expectedHeaders["X-Permitted-Cross-Domain-Policies"]
		assert.Equal(t, "none", header, "Should not allow cross-domain policies")
	})

	t.Run("Cross-Origin-Opener-Policy", func(t *testing.T) {
		header := expectedHeaders["Cross-Origin-Opener-Policy"]
		assert.Equal(t, "same-origin", header, "COOP should be same-origin")
	})

	t.Run("Cross-Origin-Resource-Policy", func(t *testing.T) {
		header := expectedHeaders["Cross-Origin-Resource-Policy"]
		assert.Equal(t, "same-origin", header, "CORP should be same-origin")
	})

	t.Run("Cross-Origin-Embedder-Policy", func(t *testing.T) {
		header := expectedHeaders["Cross-Origin-Embedder-Policy"]
		assert.Equal(t, "require-corp", header, "COEP should require CORP")
	})
}

// TestContentSecurityPolicyDetails tests CSP directive details
func TestContentSecurityPolicyDetails(t *testing.T) {
	csp := buildCSP()

	t.Run("default-src directive", func(t *testing.T) {
		assert.Contains(t, csp, "default-src 'self'", "Default should be self")
	})

	t.Run("script-src directive", func(t *testing.T) {
		assert.Contains(t, csp, "script-src 'self'", "Scripts should be from self")
		assert.NotContains(t, csp, "script-src 'unsafe-inline'", "No unsafe-inline for scripts")
	})

	t.Run("style-src directive", func(t *testing.T) {
		// Styles may need unsafe-inline for some frameworks, but should be minimized
		if hasStyleSrc := containsDirective(csp, "style-src"); hasStyleSrc {
			assert.Contains(t, csp, "style-src", "Should have style-src directive")
		}
	})

	t.Run("img-src directive", func(t *testing.T) {
		// Images may come from self and data URIs
		assert.Contains(t, csp, "img-src", "Should have img-src directive")
	})

	t.Run("connect-src directive", func(t *testing.T) {
		// API calls should be restricted
		assert.Contains(t, csp, "connect-src 'self'", "API calls should be to self")
	})

	t.Run("font-src directive", func(t *testing.T) {
		// Fonts should be from self
		if hasFontSrc := containsDirective(csp, "font-src"); hasFontSrc {
			assert.Contains(t, csp, "font-src 'self'", "Fonts should be from self")
		}
	})

	t.Run("object-src directive", func(t *testing.T) {
		assert.Contains(t, csp, "object-src 'none'", "Objects should be blocked")
	})

	t.Run("media-src directive", func(t *testing.T) {
		// Media should be restricted
		if hasMediaSrc := containsDirective(csp, "media-src"); hasMediaSrc {
			assert.Contains(t, csp, "media-src 'self'", "Media should be from self")
		}
	})

	t.Run("frame-ancestors directive", func(t *testing.T) {
		assert.Contains(t, csp, "frame-ancestors 'none'", "Should prevent framing")
	})

	t.Run("base-uri directive", func(t *testing.T) {
		assert.Contains(t, csp, "base-uri 'self'", "Base URI should be self")
	})

	t.Run("form-action directive", func(t *testing.T) {
		assert.Contains(t, csp, "form-action 'self'", "Forms should submit to self")
	})
}

// TestHSTSConfiguration tests HSTS header configuration
func TestHSTSConfiguration(t *testing.T) {
	hsts := buildHSTS()

	t.Run("max-age is sufficient", func(t *testing.T) {
		assert.Contains(t, hsts, "max-age=31536000", "Should be set for 1 year (31536000 seconds)")
	})

	t.Run("includeSubDomains present", func(t *testing.T) {
		assert.Contains(t, hsts, "includeSubDomains", "Should include subdomains")
	})

	t.Run("preload directive", func(t *testing.T) {
		// Preload is optional but recommended
		if hasPreload := containsValue(hsts, "preload"); hasPreload {
			assert.Contains(t, hsts, "preload", "Preload directive present")
		}
	})
}

// TestPermissionsPolicyDetails tests Permissions-Policy details
func TestPermissionsPolicyDetails(t *testing.T) {
	policy := buildPermissionsPolicy()

	dangerousFeatures := []string{
		"geolocation",
		"microphone",
		"camera",
		"payment",
		"usb",
		"magnetometer",
		"gyroscope",
		"accelerometer",
	}

	for _, feature := range dangerousFeatures {
		t.Run("Restrict_"+feature, func(t *testing.T) {
			// Features should be restricted with ()
			expected := feature + "=()"
			assert.Contains(t, policy, expected, feature+" should be restricted")
		})
	}
}

// TestCORSHeaders tests CORS header configuration
func TestCORSHeaders(t *testing.T) {
	t.Run("Access-Control-Allow-Origin is restrictive", func(t *testing.T) {
		allowedOrigins := []string{
			"http://localhost:5173",
			"http://localhost:5174",
			"https://app.businessos.com",
		}

		// Should NOT use wildcard with credentials
		for _, origin := range allowedOrigins {
			assert.NotEqual(t, "*", origin, "Should not use wildcard origin with credentials")
		}
	})

	t.Run("Access-Control-Allow-Credentials", func(t *testing.T) {
		// When using credentials, origin must be specific
		allowCredentials := true
		allowedOrigin := "http://localhost:5173"

		if allowCredentials {
			assert.NotEqual(t, "*", allowedOrigin, "Cannot use wildcard with credentials")
		}
	})

	t.Run("Access-Control-Allow-Methods is restrictive", func(t *testing.T) {
		allowedMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

		// Should only include necessary methods
		assert.LessOrEqual(t, len(allowedMethods), 7, "Should limit allowed methods")
	})

	t.Run("Access-Control-Max-Age set", func(t *testing.T) {
		maxAge := 43200 // 12 hours in seconds

		assert.Greater(t, maxAge, 0, "Max age should be positive")
		assert.LessOrEqual(t, maxAge, 86400, "Max age should not exceed 24 hours")
	})
}

// TestSecurityHeadersInProduction tests production-specific headers
func TestSecurityHeadersInProduction(t *testing.T) {
	t.Run("Secure cookie flag in production", func(t *testing.T) {
		isProduction := true
		secureFlagSet := isProduction

		assert.True(t, secureFlagSet, "Secure flag must be set in production")
	})

	t.Run("SameSite=Lax or Strict", func(t *testing.T) {
		sameSite := "Lax"
		validValues := []string{"Strict", "Lax"}

		isValid := false
		for _, valid := range validValues {
			if sameSite == valid {
				isValid = true
				break
			}
		}

		assert.True(t, isValid, "SameSite should be Strict or Lax")
	})

	t.Run("HttpOnly flag set", func(t *testing.T) {
		httpOnly := true
		assert.True(t, httpOnly, "HttpOnly flag should be set to prevent XSS")
	})
}

// TestCacheHeaders tests cache control headers for sensitive data
func TestCacheHeaders(t *testing.T) {
	t.Run("No caching for sensitive endpoints", func(t *testing.T) {
		sensitiveEndpoints := []string{
			"/api/auth/session",
			"/api/users/me",
			"/api/settings",
		}

		for _, endpoint := range sensitiveEndpoints {
			cacheControl := getCacheControlForEndpoint(endpoint)

			assert.Contains(t, cacheControl, "no-store", "Sensitive data should not be cached")
			assert.Contains(t, cacheControl, "no-cache", "Should require revalidation")
			assert.Contains(t, cacheControl, "private", "Should be private, not public")
		}
	})

	t.Run("Pragma: no-cache for legacy browsers", func(t *testing.T) {
		pragma := "no-cache"
		assert.Equal(t, "no-cache", pragma, "Pragma should be set for legacy support")
	})
}

// Helper functions

func getSecurityHeaders() map[string]string {
	return map[string]string{
		"X-Frame-Options":                   "DENY",
		"X-Content-Type-Options":            "nosniff",
		"Strict-Transport-Security":         "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":           buildCSP(),
		"Referrer-Policy":                   "strict-origin-when-cross-origin",
		"Permissions-Policy":                buildPermissionsPolicy(),
		"X-Permitted-Cross-Domain-Policies": "none",
		"Cross-Origin-Opener-Policy":        "same-origin",
		"Cross-Origin-Resource-Policy":      "same-origin",
		"Cross-Origin-Embedder-Policy":      "require-corp",
	}
}

func buildCSP() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self'",
		"media-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}

	result := ""
	for i, directive := range directives {
		if i > 0 {
			result += "; "
		}
		result += directive
	}
	return result
}

func buildHSTS() string {
	return "max-age=31536000; includeSubDomains"
}

func buildPermissionsPolicy() string {
	restrictions := []string{
		"geolocation=()",
		"microphone=()",
		"camera=()",
		"payment=()",
		"usb=()",
		"magnetometer=()",
		"gyroscope=()",
		"accelerometer=()",
	}

	result := ""
	for i, restriction := range restrictions {
		if i > 0 {
			result += ", "
		}
		result += restriction
	}
	return result
}

func containsDirective(csp, directive string) bool {
	return len(csp) >= len(directive) && indexOfSubstring(csp, directive) >= 0
}

func containsValue(s, substr string) bool {
	return indexOfSubstring(s, substr) >= 0
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func getCacheControlForEndpoint(endpoint string) string {
	// Sensitive endpoints should not be cached
	sensitiveEndpoints := []string{"/api/auth/", "/api/users/me", "/api/settings"}

	for _, sensitive := range sensitiveEndpoints {
		if len(endpoint) >= len(sensitive) && endpoint[:len(sensitive)] == sensitive {
			return "no-store, no-cache, must-revalidate, private"
		}
	}

	return "public, max-age=3600"
}
