package config

import (
	"bufio"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/integrations/osa"
	"github.com/spf13/viper"
)

// Load reads configuration from environment variables only.
//
// Secret injection contract:
//   - In production (ENVIRONMENT=production): ALL secrets must be injected as
//     real environment variables (e.g. via Cloud Run secrets, Kubernetes secrets,
//     or shell export). The .env file on disk is NEVER read in production.
//   - In development: a .env file in the working directory is read as a
//     convenience. Environment variables set in the shell always take priority
//     over the .env file for all fields except the subset explicitly re-applied
//     by applyDotenvOverrides (DATABASE_URL, REDIS_URL, and a handful of others
//     that are commonly shadowed by global shell exports in dev environments).
//
// To add a new secret: add a field to Config (config_types.go), set its default
// here, and add a production Validate() check in config_helpers.go.
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	// Set defaults
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("SERVER_PORT", "8001")
	viper.SetDefault("BASE_URL", "http://localhost:8001")
	viper.SetDefault("DATABASE_URL", "postgres://CHANGE_ME:CHANGE_ME@localhost:5432/business_os")
	viper.SetDefault("DATABASE_REQUIRED", true)
	viper.SetDefault("ENABLE_LOCAL_MODELS", true) // Disable in production
	viper.SetDefault("SECRET_KEY", "")
	viper.SetDefault("ALGORITHM", "HS256")
	viper.SetDefault("ACCESS_TOKEN_EXPIRE_MINUTES", 1440)

	// AI Provider - Default to Ollama Cloud for best out-of-box experience
	// Options: "ollama_cloud", "ollama_local", "anthropic", "groq"
	viper.SetDefault("AI_PROVIDER", "ollama_cloud")

	// Ollama Local
	viper.SetDefault("OLLAMA_LOCAL_URL", "http://localhost:11434")
	viper.SetDefault("DEFAULT_MODEL", "llama3.2:3b")

	// Ollama Cloud
	viper.SetDefault("OLLAMA_CLOUD_API_KEY", "")
	viper.SetDefault("OLLAMA_CLOUD_MODEL", "llama3.2")

	// Anthropic
	viper.SetDefault("ANTHROPIC_API_KEY", "")
	viper.SetDefault("ANTHROPIC_MODEL", "claude-sonnet-4-20250514")
	viper.SetDefault("ANTHROPIC_BASE_URL", "")

	// Groq
	viper.SetDefault("GROQ_API_KEY", "")
	viper.SetDefault("GROQ_MODEL", "llama-3.3-70b-versatile")

	// Legacy
	viper.SetDefault("OLLAMA_MODE", "cloud")

	// Redis
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_TLS_ENABLED", false)
	viper.SetDefault("REDIS_KEY_HMAC_SECRET", "") // CRITICAL: Set strong value in production (min 32 bytes)
	viper.SetDefault("TOKEN_ENCRYPTION_KEY", "")  // CRITICAL: Set in production for OAuth token encryption

	// Other services
	viper.SetDefault("SUPERMEMORY_API_KEY", "")
	viper.SetDefault("GOOGLE_CLIENT_ID", "")
	viper.SetDefault("GOOGLE_CLIENT_SECRET", "")
	viper.SetDefault("GOOGLE_REDIRECT_URI", "http://localhost:8001/api/auth/google/callback/login")
	viper.SetDefault("GOOGLE_INTEGRATION_REDIRECT_URI", "http://localhost:8001/api/integrations/google/callback")
	viper.SetDefault("GOOGLE_CALENDAR_REDIRECT_URI", "")
	viper.SetDefault("GOOGLE_GMAIL_REDIRECT_URI", "")
	viper.SetDefault("SLACK_CLIENT_ID", "")
	viper.SetDefault("SLACK_CLIENT_SECRET", "")
	viper.SetDefault("SLACK_REDIRECT_URI", "http://localhost:8001/api/integrations/slack/callback")
	viper.SetDefault("NOTION_CLIENT_ID", "")
	viper.SetDefault("NOTION_CLIENT_SECRET", "")
	viper.SetDefault("NOTION_REDIRECT_URI", "http://localhost:8001/api/integrations/notion/callback")
	viper.SetDefault("HUBSPOT_CLIENT_ID", "")
	viper.SetDefault("HUBSPOT_CLIENT_SECRET", "")
	viper.SetDefault("HUBSPOT_REDIRECT_URI", "http://localhost:8001/api/integrations/hubspot/callback")
	viper.SetDefault("LINEAR_CLIENT_ID", "")
	viper.SetDefault("LINEAR_CLIENT_SECRET", "")
	viper.SetDefault("LINEAR_REDIRECT_URI", "http://localhost:8001/api/integrations/linear/callback")
	viper.SetDefault("LINEAR_WEBHOOK_SECRET", "")
	viper.SetDefault("INSTAGRAM_CLIENT_ID", "")
	viper.SetDefault("INSTAGRAM_CLIENT_SECRET", "")
	viper.SetDefault("INSTAGRAM_REDIRECT_URI", "http://localhost:8001/api/integrations/instagram/callback")
	viper.SetDefault("META_APP_ID", "")
	viper.SetDefault("META_APP_SECRET", "")
	viper.SetDefault("META_REDIRECT_URI", "")

	// OSA Integration
	viper.SetDefault("OSA_ENABLED", false)
	viper.SetDefault("OSA_BASE_URL", "http://localhost:8089")
	viper.SetDefault("OSA_SHARED_SECRET", "")
	viper.SetDefault("OSA_TIMEOUT", 30) // seconds
	viper.SetDefault("OSA_MAX_RETRIES", 3)
	viper.SetDefault("OSA_RETRY_DELAY", 2) // seconds

	// Internal API Security
	viper.SetDefault("INTERNAL_API_SECRET", "")  // CRITICAL: Set in production
	viper.SetDefault("INTERNAL_ALLOWED_IPS", "") // Comma-separated IPs, optional

	// Web Search Providers
	viper.SetDefault("BRAVE_SEARCH_API_KEY", "")
	viper.SetDefault("SERPER_API_KEY", "")
	viper.SetDefault("TAVILY_API_KEY", "")
	viper.SetDefault("SEARCH_PROVIDER", "auto") // auto, brave, serper, tavily, duckduckgo

	// Web Push (VAPID) - Generate keys: npx web-push generate-vapid-keys
	viper.SetDefault("VAPID_PUBLIC_KEY", "")
	viper.SetDefault("VAPID_PRIVATE_KEY", "")
	viper.SetDefault("VAPID_CONTACT", "mailto:admin@businessos.app")

	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:5174,http://localhost:3000,app://localhost")

	// Background jobs
	viper.SetDefault("CONVERSATION_SUMMARY_JOB_ENABLED", false)
	viper.SetDefault("CONVERSATION_SUMMARY_JOB_INTERVAL_MINUTES", 30)
	viper.SetDefault("CONVERSATION_SUMMARY_JOB_BATCH_SIZE", 25)
	viper.SetDefault("CONVERSATION_SUMMARY_JOB_MAX_MESSAGES", 200)

	viper.SetDefault("APP_PROFILER_SYNC_JOB_ENABLED", false)
	viper.SetDefault("APP_PROFILER_SYNC_JOB_INTERVAL_MINUTES", 10)
	viper.SetDefault("APP_PROFILER_SYNC_JOB_BATCH_SIZE", 5)

	viper.SetDefault("BEHAVIOR_PATTERNS_JOB_ENABLED", false)
	viper.SetDefault("BEHAVIOR_PATTERNS_JOB_INTERVAL_MINUTES", 60)
	viper.SetDefault("BEHAVIOR_PATTERNS_JOB_USER_BATCH_SIZE", 50)

	// Webhooks
	viper.SetDefault("WEBHOOK_SIGNING_SECRET", "")
	viper.SetDefault("OSA_WEBHOOK_TIMEOUT_SECONDS", 30)

	// Sync & Multi-instance
	viper.SetDefault("NODE_ID", "businessos")

	// NATS Messaging
	viper.SetDefault("NATS_URL", "nats://localhost:4222")
	viper.SetDefault("NATS_ENABLED", false)
	viper.SetDefault("NATS_TTL_HOURS", 24)

	// Sandbox Container Configuration
	viper.SetDefault("CONTAINER_MANAGER_ENABLED", false)
	viper.SetDefault("SANDBOX_ENABLED", false)
	viper.SetDefault("SANDBOX_PORT_MIN", 9000)
	viper.SetDefault("SANDBOX_PORT_MAX", 9999)
	viper.SetDefault("SANDBOX_MAX_PER_USER", 5)
	viper.SetDefault("SANDBOX_DEFAULT_MEMORY", 512*1024*1024)       // 512MB
	viper.SetDefault("SANDBOX_DEFAULT_CPU", 50000)                  // 50% of 1 CPU
	viper.SetDefault("SANDBOX_MAX_TOTAL_MEMORY", 2*1024*1024*1024)  // 2GB
	viper.SetDefault("SANDBOX_MAX_TOTAL_STORAGE", 5*1024*1024*1024) // 5GB

	// MIOSA Cloud Sync (ADR-003)
	viper.SetDefault("OSA_MODE", "local") // "local" | "cloud"
	viper.SetDefault("MIOSA_API_KEY", "")
	viper.SetDefault("MIOSA_CLOUD_URL", "https://api.miosa.ai")

	// MIOSA Compute API (platform API for VM provisioning)
	viper.SetDefault("MIOSA_API_URL", "http://localhost:4000")

	// Cloud Deployment (MIOSA Firecracker VM)
	viper.SetDefault("DEPLOYMENT_MODE", "local") // "local" | "cloud"
	viper.SetDefault("MIOSA_TENANT_ID", "")
	viper.SetDefault("MIOSA_JWT_PUBLIC_KEY", "")
	viper.SetDefault("CLOUD_DOMAIN", "")

	// Environment variables are the authoritative source of truth.
	// They are read first and win over the config file for all keys.
	viper.AutomaticEnv()

	// Try to read config file (optional - for local development only).
	// Ignored silently in all cases; production deployments must not rely on it.
	_ = viper.ReadInConfig()

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	// .env file override logic — development only.
	//
	// Context: in local dev it is common to have a globally-exported DATABASE_URL
	// (e.g. pointing at a Supabase project) that would shadow this repo's local
	// .env value. To prevent that, we re-apply a curated subset of .env fields
	// after Unmarshal so that the local file wins for those keys in development.
	//
	// SECURITY: this block is completely skipped when ENVIRONMENT=production.
	// In production, every secret must arrive as a real environment variable;
	// no .env file should exist on the container filesystem.
	var dotenvVars map[string]string
	isProductionEnv := strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT"))) == "production"

	if !isProductionEnv {
		vars := readDotenvFile(".env")
		if len(vars) > 0 {
			dotenvEnv := strings.ToLower(strings.TrimSpace(vars["ENVIRONMENT"]))
			if dotenvEnv == "" || dotenvEnv == "development" {
				dotenvVars = vars
				applyDotenvOverrides(config, dotenvVars)
				slog.Debug("config: loaded .env file for local development",
					"path", ".env",
					"override_count", len(dotenvVars),
				)
			}
		}
	} else {
		// In production, confirm that we are running env-var-only mode.
		slog.Info("config: production mode — secrets loaded from environment variables only, .env file ignored")
	}

	// Default CORS origins for development
	defaultOrigins := []string{
		"http://localhost:5173",
		"http://localhost:5174",
		"http://localhost:3000",
		"app://localhost",
	}

	// Clear AllowedOrigins set by Unmarshal (may have garbage) and parse from string.
	// Prefer .env value in development to prevent ambient shell exports from
	// widening CORS unexpectedly.
	config.AllowedOrigins = nil
	originsStr := viper.GetString("ALLOWED_ORIGINS")
	if len(dotenvVars) > 0 {
		if v, ok := dotenvVars["ALLOWED_ORIGINS"]; ok && strings.TrimSpace(v) != "" {
			originsStr = v
		}
	}
	if originsStr != "" {
		origins := strings.Split(originsStr, ",")
		for _, o := range origins {
			trimmed := strings.TrimSpace(o)
			// Only include valid origins: http(s):// or the desktop app's
			// app:// custom scheme (the Electron renderer's origin).
			// SECURITY: Wildcard "*" is NEVER allowed (causes CSRF vulnerability with credentials)
			if strings.HasPrefix(trimmed, "http://") ||
				strings.HasPrefix(trimmed, "https://") ||
				strings.HasPrefix(trimmed, "app://") {
				config.AllowedOrigins = append(config.AllowedOrigins, trimmed)
			}
		}
	}

	// Fallback to defaults if no valid origins found
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = defaultOrigins
	}

	// Convert DATABASE_URL from asyncpg format to pgx format if needed
	// Python uses: postgresql+asyncpg://...
	// Go pgx uses: postgres://... or postgresql://...
	config.DatabaseURL = strings.Replace(config.DatabaseURL, "postgresql+asyncpg://", "postgres://", 1)

	// Build OSA config from flat fields
	if config.OSAEnabled {
		config.OSA = &osa.Config{
			BaseURL:      config.OSABaseURL,
			SharedSecret: config.OSASharedSecret,
			Timeout:      time.Duration(config.OSATimeout) * time.Second,
			MaxRetries:   config.OSAMaxRetries,
			RetryDelay:   time.Duration(config.OSARetryDelay) * time.Second,
		}
	} else {
		config.OSA = &osa.Config{} // Empty config when disabled
	}

	// SECURITY: Emit startup warnings for misconfigured secrets.
	// In development: warn about insecure defaults so devs know to set up .env.
	// In production: these conditions must never be reached (Validate() will refuse to start).
	if !config.IsProduction() {
		if config.SecretKey == "" {
			slog.Warn("config: SECRET_KEY is not set; set it in .env before deploying")
		} else if len(config.SecretKey) < 32 {
			slog.Warn("config: SECRET_KEY is shorter than 32 characters; use a stronger key before deploying")
		}
		if strings.Contains(config.DatabaseURL, "CHANGE_ME") {
			slog.Warn("config: DATABASE_URL is using the placeholder default; set DATABASE_URL in .env")
		}
		// Warn about empty security-critical keys so developers notice during local testing.
		if config.RedisKeyHMACSecret == "" {
			slog.Warn("config: REDIS_KEY_HMAC_SECRET is unset; session token hashing is degraded (acceptable in dev only)")
		}
		if config.TokenEncryptionKey == "" {
			slog.Warn("config: TOKEN_ENCRYPTION_KEY is unset; OAuth tokens will not be encrypted at rest (acceptable in dev only)")
		}
	}

	AppConfig = config
	return config, nil
}

// readDotenvFile parses a .env file and returns key/value pairs.
// Lines starting with # are ignored; export prefixes are stripped.
func readDotenvFile(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key != "" {
			vars[key] = val
		}
	}
	return vars
}

// applyDotenvOverrides re-applies select .env values onto the config struct,
// ensuring local development .env takes precedence over ambient environment variables.
func applyDotenvOverrides(cfg *Config, vars map[string]string) {
	if v := strings.TrimSpace(vars["ENVIRONMENT"]); v != "" {
		cfg.Environment = v
	}
	if v := strings.TrimSpace(vars["SERVER_PORT"]); v != "" {
		cfg.ServerPort = v
	}
	if v := strings.TrimSpace(vars["DATABASE_URL"]); v != "" {
		cfg.DatabaseURL = v
	}
	if v := strings.TrimSpace(vars["DATABASE_REQUIRED"]); v != "" {
		cfg.DatabaseRequired = strings.EqualFold(v, "true") || v == "1"
	}
	if v := strings.TrimSpace(vars["REDIS_URL"]); v != "" {
		cfg.RedisURL = v
	}
	if v := strings.TrimSpace(vars["REDIS_PASSWORD"]); v != "" {
		cfg.RedisPassword = v
	}
	if v := strings.TrimSpace(vars["REDIS_TLS_ENABLED"]); v != "" {
		cfg.RedisTLSEnabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := strings.TrimSpace(vars["REDIS_KEY_HMAC_SECRET"]); v != "" {
		cfg.RedisKeyHMACSecret = v
	}
	// Google OAuth
	if v := strings.TrimSpace(vars["GOOGLE_CLIENT_ID"]); v != "" {
		cfg.GoogleClientID = v
	}
	if v := strings.TrimSpace(vars["GOOGLE_CLIENT_SECRET"]); v != "" {
		cfg.GoogleClientSecret = v
	}
	if v := strings.TrimSpace(vars["GOOGLE_REDIRECT_URI"]); v != "" {
		cfg.GoogleRedirectURI = v
	}
	if v := strings.TrimSpace(vars["GOOGLE_INTEGRATION_REDIRECT_URI"]); v != "" {
		cfg.GoogleIntegrationRedirectURI = v
	}
	if v := strings.TrimSpace(vars["GOOGLE_CALENDAR_REDIRECT_URI"]); v != "" {
		cfg.GoogleCalendarRedirectURI = v
	}
	if v := strings.TrimSpace(vars["GOOGLE_GMAIL_REDIRECT_URI"]); v != "" {
		cfg.GoogleGmailRedirectURI = v
	}
}
