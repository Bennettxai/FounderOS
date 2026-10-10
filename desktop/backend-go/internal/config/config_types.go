package config

import "github.com/rhl/businessos-backend/internal/integrations/osa"

// DeploymentMode determines how BusinessOS was launched.
type DeploymentMode string

const (
	// DeploymentLocal is the default — Electron desktop app.
	DeploymentLocal DeploymentMode = "local"
	// DeploymentCloud runs inside a MIOSA Firecracker VM.
	DeploymentCloud DeploymentMode = "cloud"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Environment (development, production)
	Environment string `mapstructure:"ENVIRONMENT"`

	// Database
	DatabaseURL      string `mapstructure:"DATABASE_URL"`
	DatabaseRequired bool   `mapstructure:"DATABASE_REQUIRED"`

	// Server
	ServerPort string `mapstructure:"SERVER_PORT"`
	BaseURL    string `mapstructure:"BASE_URL"`

	// JWT Auth (kept for compatibility, Better Auth handles auth)
	SecretKey                string `mapstructure:"SECRET_KEY"`
	Algorithm                string `mapstructure:"ALGORITHM"`
	AccessTokenExpireMinutes int    `mapstructure:"ACCESS_TOKEN_EXPIRE_MINUTES"`

	// AI Provider Configuration
	// Options: "ollama_cloud", "ollama_local", "anthropic", "groq", "xai"
	AIProvider string `mapstructure:"AI_PROVIDER"`

	// Ollama Local Configuration
	OllamaLocalURL string `mapstructure:"OLLAMA_LOCAL_URL"`

	// Ollama Cloud Configuration (api.ollama.com)
	OllamaCloudAPIKey string `mapstructure:"OLLAMA_CLOUD_API_KEY"`
	OllamaCloudModel  string `mapstructure:"OLLAMA_CLOUD_MODEL"`

	// Anthropic Configuration
	AnthropicAPIKey  string `mapstructure:"ANTHROPIC_API_KEY"`
	AnthropicModel   string `mapstructure:"ANTHROPIC_MODEL"`
	AnthropicBaseURL string `mapstructure:"ANTHROPIC_BASE_URL"`

	// Groq Configuration
	GroqAPIKey string `mapstructure:"GROQ_API_KEY"`
	GroqModel  string `mapstructure:"GROQ_MODEL"`

	// OpenAI Configuration
	OpenAIAPIKey string `mapstructure:"OPENAI_API_KEY"`
	OpenAIModel  string `mapstructure:"OPENAI_MODEL"`

	// Grok (x.ai) Configuration - Used for onboarding AI
	XAIAPIKey  string `mapstructure:"XAI_API_KEY"`
	XAIModel   string `mapstructure:"XAI_MODEL"`
	XAIBaseURL string `mapstructure:"XAI_BASE_URL"`

	// Default Model (for local Ollama)
	DefaultModel string `mapstructure:"DEFAULT_MODEL"`

	// Legacy - kept for compatibility
	OllamaMode string `mapstructure:"OLLAMA_MODE"`

	// Redis
	RedisURL        string `mapstructure:"REDIS_URL"`
	RedisPassword   string `mapstructure:"REDIS_PASSWORD"`
	RedisTLSEnabled bool   `mapstructure:"REDIS_TLS_ENABLED"`

	// Security: HMAC secret for Redis key derivation (prevents token enumeration attacks)
	// CRITICAL: Must be set in production to a strong random value (min 32 bytes)
	// Used to hash session tokens before storing as Redis keys
	RedisKeyHMACSecret string `mapstructure:"REDIS_KEY_HMAC_SECRET"`

	// Security: Token encryption key for OAuth tokens stored in database
	// CRITICAL: Must be set in production - 32-byte base64-encoded key
	// Generate with: openssl rand -base64 32
	TokenEncryptionKey string `mapstructure:"TOKEN_ENCRYPTION_KEY"`

	// Supermemory
	SupermemoryAPIKey string `mapstructure:"SUPERMEMORY_API_KEY"`

	// Google OAuth
	GoogleClientID               string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret           string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURI            string `mapstructure:"GOOGLE_REDIRECT_URI"`             // For login flow
	GoogleIntegrationRedirectURI string `mapstructure:"GOOGLE_INTEGRATION_REDIRECT_URI"` // For calendar integration
	GoogleCalendarRedirectURI    string `mapstructure:"GOOGLE_CALENDAR_REDIRECT_URI"`    // Tool-specific calendar flow
	GoogleGmailRedirectURI       string `mapstructure:"GOOGLE_GMAIL_REDIRECT_URI"`       // Tool-specific Gmail flow

	// Gmail Pub/Sub watch — Wave 3 realtime push.
	// GmailPubSubTopic is the fully-qualified Cloud Pub/Sub topic Google
	// publishes Gmail change notifications to (e.g.
	// "projects/my-proj/topics/gmail-push"). Set per-deployment by
	// Roberto in GCP Console; absent means realtime is disabled and
	// the Wave 1 manual sync remains the only ingest path.
	GmailPubSubTopic string `mapstructure:"GOOGLE_GMAIL_PUBSUB_TOPIC"`
	// GmailPubSubAudience is the JWT `aud` claim our webhook expects
	// (typically the public webhook URL). When set, the webhook
	// verifies the inbound JWT against Google's public keys.
	GmailPubSubAudience string `mapstructure:"GOOGLE_GMAIL_PUBSUB_AUDIENCE"`

	// Slack OAuth
	SlackClientID     string `mapstructure:"SLACK_CLIENT_ID"`
	SlackClientSecret string `mapstructure:"SLACK_CLIENT_SECRET"`
	SlackRedirectURI  string `mapstructure:"SLACK_REDIRECT_URI"`
	// SlackSigningSecret verifies Events API webhook signatures.
	// HMAC-SHA256 over "v0:<timestamp>:<raw-body>" using this secret;
	// see https://api.slack.com/authentication/verifying-requests-from-slack
	SlackSigningSecret string `mapstructure:"SLACK_SIGNING_SECRET"`

	// Notion OAuth
	NotionClientID     string `mapstructure:"NOTION_CLIENT_ID"`
	NotionClientSecret string `mapstructure:"NOTION_CLIENT_SECRET"`
	NotionRedirectURI  string `mapstructure:"NOTION_REDIRECT_URI"`

	// HubSpot OAuth
	HubSpotClientID     string `mapstructure:"HUBSPOT_CLIENT_ID"`
	HubSpotClientSecret string `mapstructure:"HUBSPOT_CLIENT_SECRET"`
	HubSpotRedirectURI  string `mapstructure:"HUBSPOT_REDIRECT_URI"`

	// Linear OAuth
	LinearClientID      string `mapstructure:"LINEAR_CLIENT_ID"`
	LinearClientSecret  string `mapstructure:"LINEAR_CLIENT_SECRET"`
	LinearRedirectURI   string `mapstructure:"LINEAR_REDIRECT_URI"`
	LinearWebhookSecret string `mapstructure:"LINEAR_WEBHOOK_SECRET"` // For webhook signature verification

	// ClickUp OAuth
	ClickUpClientID     string `mapstructure:"CLICKUP_CLIENT_ID"`
	ClickUpClientSecret string `mapstructure:"CLICKUP_CLIENT_SECRET"`
	ClickUpRedirectURI  string `mapstructure:"CLICKUP_REDIRECT_URI"`

	// Instagram / Meta OAuth
	InstagramClientID     string `mapstructure:"INSTAGRAM_CLIENT_ID"`
	InstagramClientSecret string `mapstructure:"INSTAGRAM_CLIENT_SECRET"`
	InstagramRedirectURI  string `mapstructure:"INSTAGRAM_REDIRECT_URI"`
	MetaAppID             string `mapstructure:"META_APP_ID"`
	MetaAppSecret         string `mapstructure:"META_APP_SECRET"`
	MetaRedirectURI       string `mapstructure:"META_REDIRECT_URI"`

	// Airtable OAuth
	AirtableClientID     string `mapstructure:"AIRTABLE_CLIENT_ID"`
	AirtableClientSecret string `mapstructure:"AIRTABLE_CLIENT_SECRET"`
	AirtableRedirectURI  string `mapstructure:"AIRTABLE_REDIRECT_URI"`

	// Microsoft 365 OAuth
	MicrosoftClientID     string `mapstructure:"MICROSOFT_CLIENT_ID"`
	MicrosoftClientSecret string `mapstructure:"MICROSOFT_CLIENT_SECRET"`
	MicrosoftRedirectURI  string `mapstructure:"MICROSOFT_REDIRECT_URI"`

	// OSA (Open Source Agent) Integration
	OSAEnabled      bool        `mapstructure:"OSA_ENABLED"`
	OSABaseURL      string      `mapstructure:"OSA_BASE_URL"`
	OSASharedSecret string      `mapstructure:"OSA_SHARED_SECRET"`
	OSATimeout      int         `mapstructure:"OSA_TIMEOUT"` // seconds
	OSAMaxRetries   int         `mapstructure:"OSA_MAX_RETRIES"`
	OSARetryDelay   int         `mapstructure:"OSA_RETRY_DELAY"` // seconds
	OSA             *osa.Config // Built from above fields in Load()

	// Internal API Security
	// CRITICAL: INTERNAL_API_SECRET must be set in production for internal endpoint authentication
	// Used for HMAC-SHA256 signature verification on /api/internal/* endpoints
	InternalAPISecret  string   `mapstructure:"INTERNAL_API_SECRET"`
	InternalAllowedIPs []string `mapstructure:"INTERNAL_ALLOWED_IPS"` // IPs that can bypass signature verification

	// Web Search Providers
	// Priority: Brave > Serper > Tavily > DuckDuckGo (fallback)
	BraveSearchAPIKey string `mapstructure:"BRAVE_SEARCH_API_KEY"` // Free: 2000 queries/month
	SerperAPIKey      string `mapstructure:"SERPER_API_KEY"`       // Google results via API
	TavilyAPIKey      string `mapstructure:"TAVILY_API_KEY"`       // AI-focused search API
	SearchProvider    string `mapstructure:"SEARCH_PROVIDER"`      // Override: brave, serper, tavily, duckduckgo, auto

	// CORS
	AllowedOrigins []string `mapstructure:"ALLOWED_ORIGINS"`

	// Feature Flags
	EnableLocalModels bool `mapstructure:"ENABLE_LOCAL_MODELS"`

	// Web Push (VAPID)
	VAPIDPublicKey  string `mapstructure:"VAPID_PUBLIC_KEY"`
	VAPIDPrivateKey string `mapstructure:"VAPID_PRIVATE_KEY"`
	VAPIDContact    string `mapstructure:"VAPID_CONTACT"` // Email: mailto:admin@example.com

	// Background Jobs (disabled by default)
	ConversationSummaryJobEnabled         bool `mapstructure:"CONVERSATION_SUMMARY_JOB_ENABLED"`
	ConversationSummaryJobIntervalMinutes int  `mapstructure:"CONVERSATION_SUMMARY_JOB_INTERVAL_MINUTES"`
	ConversationSummaryJobBatchSize       int  `mapstructure:"CONVERSATION_SUMMARY_JOB_BATCH_SIZE"`
	ConversationSummaryJobMaxMessages     int  `mapstructure:"CONVERSATION_SUMMARY_JOB_MAX_MESSAGES"`

	BehaviorPatternsJobEnabled         bool `mapstructure:"BEHAVIOR_PATTERNS_JOB_ENABLED"`
	BehaviorPatternsJobIntervalMinutes int  `mapstructure:"BEHAVIOR_PATTERNS_JOB_INTERVAL_MINUTES"`
	BehaviorPatternsJobUserBatchSize   int  `mapstructure:"BEHAVIOR_PATTERNS_JOB_USER_BATCH_SIZE"`

	AppProfilerSyncJobEnabled         bool `mapstructure:"APP_PROFILER_SYNC_JOB_ENABLED"`
	AppProfilerSyncJobIntervalMinutes int  `mapstructure:"APP_PROFILER_SYNC_JOB_INTERVAL_MINUTES"`
	AppProfilerSyncJobBatchSize       int  `mapstructure:"APP_PROFILER_SYNC_JOB_BATCH_SIZE"`

	// Webhooks
	WebhookSigningSecret string `mapstructure:"WEBHOOK_SIGNING_SECRET"`
	OSAWebhookTimeout    int    `mapstructure:"OSA_WEBHOOK_TIMEOUT_SECONDS"` // Default: 30 seconds

	// Sync & Multi-instance
	NodeID string `mapstructure:"NODE_ID"` // Identifier for this instance (multi-instance deployments)

	// NATS Messaging (for OSA sync events)
	NATSURL     string `mapstructure:"NATS_URL"`       // NATS server URL
	NATSEnabled bool   `mapstructure:"NATS_ENABLED"`   // Enable NATS integration
	NATSTTL     int    `mapstructure:"NATS_TTL_HOURS"` // Message TTL in hours

	// Sandbox Container Configuration
	ContainerManagerEnabled bool  `mapstructure:"CONTAINER_MANAGER_ENABLED"` // Enable Docker-backed workspace containers
	SandboxEnabled          bool  `mapstructure:"SANDBOX_ENABLED"`           // Enable Docker-backed sandbox app deployment
	SandboxPortMin          int   `mapstructure:"SANDBOX_PORT_MIN"`          // Minimum port for sandbox containers (default: 9000)
	SandboxPortMax          int   `mapstructure:"SANDBOX_PORT_MAX"`          // Maximum port for sandbox containers (default: 9999)
	SandboxMaxPerUser       int   `mapstructure:"SANDBOX_MAX_PER_USER"`      // Max concurrent sandboxes per user (default: 5)
	SandboxDefaultMemory    int64 `mapstructure:"SANDBOX_DEFAULT_MEMORY"`    // Default memory per sandbox in bytes (default: 512MB)
	SandboxDefaultCPU       int   `mapstructure:"SANDBOX_DEFAULT_CPU"`       // Default CPU quota (100000 = 1 CPU, default: 50000 = 50%)
	SandboxMaxTotalMemory   int64 `mapstructure:"SANDBOX_MAX_TOTAL_MEMORY"`  // Max total memory across all sandboxes (default: 2GB)
	SandboxMaxTotalStorage  int64 `mapstructure:"SANDBOX_MAX_TOTAL_STORAGE"` // Max total storage for workspaces (default: 5GB)

	// MIOSA Cloud Sync (optional; see ADR-003)
	// OSAMode selects the OSA transport: "local" (default) or "cloud".
	// When "cloud", MIOSAAPIKey must be set and OSA routes via api.miosa.ai.
	OSAMode       string `mapstructure:"OSA_MODE"`        // "local" | "cloud" (default: "local")
	MIOSAAPIKey   string `mapstructure:"MIOSA_API_KEY"`   // MIOSA Cloud API key
	MIOSACloudURL string `mapstructure:"MIOSA_CLOUD_URL"` // Override cloud endpoint (default: https://api.miosa.ai)

	// MIOSA Compute API — platform API for provisioning and managing cloud VMs.
	// Distinct from MIOSACloudURL (workspace sync). Set MIOSA_API_URL for the
	// Phoenix/Elixir backend (prod: http://api.miosa.ai, dev: http://localhost:4000).
	MIOSAAPIUrl string `mapstructure:"MIOSA_API_URL"` // e.g. http://localhost:4000

	// Cloud Deployment (MIOSA Firecracker VM)
	// DeploymentMode selects how this instance was launched: "local" (Electron desktop,
	// default) or "cloud" (MIOSA Firecracker VM). In cloud mode the MIOSA JWT auth
	// middleware is active and the health endpoint exposes tenant/domain information.
	DeploymentMode    string `mapstructure:"DEPLOYMENT_MODE"`      // "local" | "cloud" (default: "local")
	MIOSATenantID     string `mapstructure:"MIOSA_TENANT_ID"`      // UUID of the MIOSA tenant (cloud only)
	MIOSAJWTPublicKey string `mapstructure:"MIOSA_JWT_PUBLIC_KEY"` // Base64-encoded RS256 public key (cloud only)
	CloudDomain       string `mapstructure:"CLOUD_DOMAIN"`         // Subdomain for this instance, e.g. acme-corp.bos.miosa.ai (cloud only)
}

// AppConfig is the global singleton set by Load().
var AppConfig *Config
