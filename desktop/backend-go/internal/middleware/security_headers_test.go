package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityHeaders(t *testing.T) {
	// Create test config (development mode)
	cfg := &config.Config{
		Environment: "development",
	}

	tests := []struct {
		name           string
		expectedHeader string
		expectedValue  string
	}{
		{
			name:           "X-Frame-Options",
			expectedHeader: "X-Frame-Options",
			expectedValue:  "DENY",
		},
		{
			name:           "X-Content-Type-Options",
			expectedHeader: "X-Content-Type-Options",
			expectedValue:  "nosniff",
		},
		{
			name:           "X-XSS-Protection",
			expectedHeader: "X-XSS-Protection",
			expectedValue:  "1; mode=block",
		},
		{
			name:           "Referrer-Policy",
			expectedHeader: "Referrer-Policy",
			expectedValue:  "strict-origin-when-cross-origin",
		},
		{
			name:           "X-Permitted-Cross-Domain-Policies",
			expectedHeader: "X-Permitted-Cross-Domain-Policies",
			expectedValue:  "none",
		},
		{
			name:           "Cross-Origin-Opener-Policy",
			expectedHeader: "Cross-Origin-Opener-Policy",
			expectedValue:  "same-origin",
		},
		{
			name:           "Cross-Origin-Resource-Policy",
			expectedHeader: "Cross-Origin-Resource-Policy",
			expectedValue:  "same-origin",
		},
		{
			name:           "Cross-Origin-Embedder-Policy",
			expectedHeader: "Cross-Origin-Embedder-Policy",
			expectedValue:  "require-corp",
		},
		{
			name:           "Cache-Control",
			expectedHeader: "Cache-Control",
			expectedValue:  "no-store, no-cache, must-revalidate, private",
		},
		{
			name:           "Pragma",
			expectedHeader: "Pragma",
			expectedValue:  "no-cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeaders(cfg))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedValue, w.Header().Get(tt.expectedHeader))
		})
	}
}

func TestSecurityHeaders_HSTS_InProduction(t *testing.T) {
	cfg := &config.Config{
		Environment: "production",
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

func TestSecurityHeaders_HSTS_NotInDevelopment(t *testing.T) {
	cfg := &config.Config{
		Environment: "development",
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// HSTS should NOT be set in development
	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Empty(t, hsts, "HSTS should not be set in development mode")
}

func TestSecurityHeaders_CSP_NoUnsafeInline(t *testing.T) {
	cfg := &config.Config{Environment: "production"}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	assert.NotContains(t, csp, "unsafe-inline", "CSP should not contain unsafe-inline")
	assert.NotContains(t, csp, "unsafe-eval", "CSP should not contain unsafe-eval")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecurityHeaders_PermissionsPolicy(t *testing.T) {
	cfg := &config.Config{Environment: "production"}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	policy := w.Header().Get("Permissions-Policy")
	
	// All dangerous features should be restricted
	dangerousFeatures := []string{
		"geolocation=()",
		"microphone=()",
		"camera=()",
		"payment=()",
		"usb=()",
	}
	
	for _, feature := range dangerousFeatures {
		assert.Contains(t, policy, feature, "Permissions-Policy should restrict "+feature)
	}
}

func TestBuildCSP(t *testing.T) {
	csp := buildCSP()

	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self'")
	assert.Contains(t, csp, "style-src 'self'")
	assert.Contains(t, csp, "img-src 'self' data:")
	assert.Contains(t, csp, "font-src 'self'")
	assert.Contains(t, csp, "connect-src 'self'")
	assert.Contains(t, csp, "media-src 'self'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "base-uri 'self'")
	assert.Contains(t, csp, "form-action 'self'")
}

func TestBuildPermissionsPolicy(t *testing.T) {
	policy := buildPermissionsPolicy()

	expectedRestrictions := []string{
		"geolocation=()",
		"microphone=()",
		"camera=()",
		"payment=()",
		"usb=()",
		"magnetometer=()",
		"gyroscope=()",
		"accelerometer=()",
	}

	for _, expected := range expectedRestrictions {
		assert.Contains(t, policy, expected)
	}
}
