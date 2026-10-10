package config

import (
	"errors"
	"log/slog"
	"strings"
)

// GetSearchProvider returns the active search provider name.
// Priority when "auto": Brave > Serper > Tavily > DuckDuckGo.
func (c *Config) GetSearchProvider() string {
	if c.SearchProvider != "" && c.SearchProvider != "auto" {
		return c.SearchProvider
	}

	// Auto-select based on available API keys
	if c.BraveSearchAPIKey != "" {
		return "brave"
	}
	if c.SerperAPIKey != "" {
		return "serper"
	}
	if c.TavilyAPIKey != "" {
		return "tavily"
	}
	return "duckduckgo"
}

// HasBraveSearch returns true if Brave Search API is configured.
func (c *Config) HasBraveSearch() bool {
	return c.BraveSearchAPIKey != ""
}

// HasSerper returns true if Serper API is configured.
func (c *Config) HasSerper() bool {
	return c.SerperAPIKey != ""
}

// HasTavily returns true if Tavily API is configured.
func (c *Config) HasTavily() bool {
	return c.TavilyAPIKey != ""
}

// Validate checks that the configuration is secure for the current environment.
//
// SECURITY: Call this immediately after Load() on startup. It returns an error
// (never just a warning) for any condition that would allow a production deploy
// to run with insecure or default secrets. The caller should treat a non-nil
// error from Validate as fatal.
//
// Secret injection checklist for production:
//   - SECRET_KEY            — JWT signing key (>= 32 chars, not the default)
//   - TOKEN_ENCRYPTION_KEY  — OAuth token encryption (>= 32 chars)
//   - REDIS_KEY_HMAC_SECRET — Session HMAC key (>= 32 chars)
//   - INTERNAL_API_SECRET   — Internal endpoint HMAC (>= 32 chars)
//   - WEBHOOK_SIGNING_SECRET — Webhook payload signing (>= 16 chars)
//   - DATABASE_URL          — Must not point at localhost
//   - ALLOWED_ORIGINS       — Must be explicit, no wildcard
func (c *Config) Validate() error {
	var errs []string

	// ── JWT / Auth — enforced in ALL environments ────────────────────────────
	// An empty or insecure SECRET_KEY allows JWT forgery regardless of env.
	if c.SecretKey == "" {
		errs = append(errs, "SECRET_KEY must be set (generate: openssl rand -base64 48)")
	} else if c.SecretKey == "INSECURE-DEFAULT-CHANGE-IN-PRODUCTION" {
		errs = append(errs, "SECRET_KEY must be changed from the insecure default value")
	} else if len(c.SecretKey) < 32 {
		errs = append(errs, "SECRET_KEY must be at least 32 characters")
	}

	if c.IsProduction() {
		// In production, warn when the key is technically valid but shorter than
		// the recommended 64-character threshold for HS256 signing keys.
		if len(c.SecretKey) > 0 && len(c.SecretKey) < 64 {
			slog.Warn("config: SECRET_KEY is valid but shorter than the recommended 64 characters for production; consider using a longer key")
		}

		// ── OAuth token encryption ───────────────────────────────────────────
		// CRITICAL: without this key, OAuth tokens are stored unencrypted in the DB.
		if c.TokenEncryptionKey == "" {
			errs = append(errs, "TOKEN_ENCRYPTION_KEY must be set in production (generate: openssl rand -base64 32)")
		}
		if len(c.TokenEncryptionKey) > 0 && len(c.TokenEncryptionKey) < 32 {
			errs = append(errs, "TOKEN_ENCRYPTION_KEY must be at least 32 characters in production")
		}

		// ── Redis session HMAC ───────────────────────────────────────────────
		// CRITICAL: without this, session tokens are stored as-is in Redis,
		// enabling token enumeration attacks.
		if c.RedisKeyHMACSecret == "" {
			errs = append(errs, "REDIS_KEY_HMAC_SECRET must be set in production (generate: openssl rand -base64 32)")
		}
		if len(c.RedisKeyHMACSecret) > 0 && len(c.RedisKeyHMACSecret) < 32 {
			errs = append(errs, "REDIS_KEY_HMAC_SECRET must be at least 32 characters in production")
		}

		// ── Internal API HMAC ────────────────────────────────────────────────
		// CRITICAL: without this, /api/internal/* endpoints are unauthenticated.
		if c.InternalAPISecret == "" {
			errs = append(errs, "INTERNAL_API_SECRET must be set in production (generate: openssl rand -base64 32)")
		}
		if len(c.InternalAPISecret) > 0 && len(c.InternalAPISecret) < 32 {
			errs = append(errs, "INTERNAL_API_SECRET must be at least 32 characters in production")
		}

		// ── Webhook signing ──────────────────────────────────────────────────
		if c.WebhookSigningSecret == "" {
			errs = append(errs, "WEBHOOK_SIGNING_SECRET must be set in production (generate: openssl rand -base64 32)")
		}
		if len(c.WebhookSigningSecret) > 0 && len(c.WebhookSigningSecret) < 16 {
			errs = append(errs, "WEBHOOK_SIGNING_SECRET must be at least 16 characters in production")
		}

		// ── Database ─────────────────────────────────────────────────────────
		if strings.Contains(c.DatabaseURL, "localhost") {
			errs = append(errs, "DATABASE_URL appears to be a development URL (contains 'localhost'); use a cloud database in production")
		}
		if strings.Contains(c.DatabaseURL, "CHANGE_ME") || strings.Contains(c.DatabaseURL, "CHANGEME") {
			errs = append(errs, "DATABASE_URL contains a placeholder value; set the real connection string")
		}

		// ── CORS ─────────────────────────────────────────────────────────────
		if len(c.AllowedOrigins) == 0 {
			errs = append(errs, "ALLOWED_ORIGINS must be explicitly set in production (e.g. https://app.businessos.com)")
		}
		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				errs = append(errs, "ALLOWED_ORIGINS contains wildcard '*' which is forbidden in production (enables CSRF with credentials)")
				break
			}
		}

		// ── Feature flags ────────────────────────────────────────────────────
		if c.EnableLocalModels {
			// Warning only — some edge deployments intentionally run local models.
			slog.Warn("config: ENABLE_LOCAL_MODELS is true in production; verify this is intentional")
		}
	}

	if len(errs) > 0 {
		return errors.New("configuration validation failed:\n  - " + strings.Join(errs, "\n  - "))
	}

	return nil
}
