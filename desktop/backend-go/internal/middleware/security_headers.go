package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
)

// SecurityHeadersConfig holds configuration for security headers middleware
type SecurityHeadersConfig struct {
	// IsProduction determines if HSTS should be enabled
	IsProduction bool
}

// SecurityHeaders returns a middleware that sets security-related HTTP headers.
// These headers protect against common web vulnerabilities:
// - Clickjacking (X-Frame-Options)
// - MIME sniffing (X-Content-Type-Options)
// - XSS attacks (X-XSS-Protection, CSP)
// - Man-in-the-middle attacks (HSTS)
// - Information leakage (Referrer-Policy, Permissions-Policy)
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		// DENY prevents any frame embedding
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		// Forces browsers to respect declared Content-Type
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS filter in browsers
		// Note: Modern browsers have built-in XSS protection, but this adds defense-in-depth
		c.Header("X-XSS-Protection", "1; mode=block")

		// HTTP Strict Transport Security (HSTS)
		// Only enable in production with HTTPS
		// 1 year duration, include subdomains
		if cfg.IsProduction() {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy
		// Restricts sources of executable scripts, styles, and other resources
		// SECURITY: No 'unsafe-inline' or 'unsafe-eval' for maximum protection
		csp := buildCSP()
		c.Header("Content-Security-Policy", csp)

		// Referrer Policy
		// Only send origin for cross-origin requests
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature-Policy)
		// Restricts access to browser features and APIs
		c.Header("Permissions-Policy", buildPermissionsPolicy())

		// Prevent cross-domain policy files from being used
		c.Header("X-Permitted-Cross-Domain-Policies", "none")

		// Cross-Origin isolation headers
		// These provide protection against Spectre-style attacks
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")

		// Cache control for API responses
		// Prevents sensitive data from being cached
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")

		c.Next()
	}
}

// buildCSP constructs the Content Security Policy header value
func buildCSP() string {
	// CSP directives for a secure API backend
	// Note: This is a backend API, so we use restrictive defaults
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

// buildPermissionsPolicy constructs the Permissions Policy header value
func buildPermissionsPolicy() string {
	// Restrict all dangerous browser features by default
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
