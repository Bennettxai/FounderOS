package middleware

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
)

// CORSMiddleware returns CORS middleware configured for Better Auth cookies
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Filter origins to only include valid http/https origins, plus the desktop
	// app's custom protocol origin (app://localhost) so the Electron renderer's
	// credentialed fetches are permitted.
	var validOrigins []string
	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(origin)
		if strings.HasPrefix(origin, "http://") ||
			strings.HasPrefix(origin, "https://") ||
			strings.HasPrefix(origin, "app://") {
			validOrigins = append(validOrigins, origin)
		}
	}

	// CRITICAL SECURITY CHECK: Reject wildcard origins in production
	if cfg.IsProduction() {
		for _, origin := range validOrigins {
			if origin == "*" {
				panic("SECURITY VIOLATION: Wildcard CORS origin (*) is FORBIDDEN in production. Set explicit ALLOWED_ORIGINS.")
			}
		}
	}

	// If no valid origins configured, use secure defaults
	if len(validOrigins) == 0 {
		if cfg.IsProduction() {
			// SECURITY: In production, NEVER allow wildcard origins with credentials
			// If ALLOWED_ORIGINS is not set, fail securely with empty whitelist
			// This forces explicit configuration and prevents CSRF attacks
			validOrigins = []string{} // Empty whitelist - no origins allowed
		} else {
			// Development: Allow localhost origins
			validOrigins = []string{
				"http://localhost:5173",
				"http://localhost:5174",
				"http://localhost:5175",
				"http://localhost:5176",
				"http://localhost:5177",
				"http://localhost:3000",
			}
		}
	}

	// Check if any origin is wildcard - can't use credentials with wildcard
	hasWildcard := false
	for _, o := range validOrigins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}

	corsConfig := cors.Config{
		AllowOrigins:     validOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Cookie", "X-CSRF-Token", "X-Workspace-ID"},
		ExposeHeaders:    []string{"X-Conversation-Id"},
		AllowCredentials: len(validOrigins) > 0 && !hasWildcard, // Can't use credentials with wildcard
		MaxAge:           43200,                                 // 12 hours (preflight cache)
		// Permit the desktop app's app:// origin; without this the library's
		// schema validation rejects app://localhost (only http/https by default).
		CustomSchemas: []string{"app://"},
	}

	return cors.New(corsConfig)
}
