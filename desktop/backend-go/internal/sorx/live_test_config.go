package sorx

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// LiveTestConfig holds credentials and configuration for live API testing
type LiveTestConfig struct {
	// Google (Gmail + Calendar)
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRefreshToken string
	GoogleTestEmail    string

	// Slack
	SlackBotToken      string
	SlackTestChannelID string
	SlackTestChannelName string

	// Notion
	NotionAPIKey        string
	NotionTestDatabaseID string
	NotionTestPageID    string

	// Linear
	LinearAPIKey       string
	LinearTestTeamID   string
	LinearTestProjectID string

	// HubSpot
	HubSpotAPIKey         string
	HubSpotTestContactEmail string

	// AI Providers
	AnthropicAPIKey string
	OpenAIAPIKey    string
	GroqAPIKey      string

	// BusinessOS Backend
	DatabaseURL         string
	RedisURL            string
	SecretKey           string
	TokenEncryptionKey  string

	// Test Configuration
	TestCategories   []string // Which action categories to test
	TestCleanupMode  string   // always, never, on_success
	TestVerbose      bool
}

// LoadLiveTestConfig loads configuration from .env.test.local or .env.test
func LoadLiveTestConfig() (*LiveTestConfig, error) {
	// Try .env.test.local first (gitignored, contains real credentials)
	envFile := ".env.test.local"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// Fall back to .env.test (template)
		envFile = ".env.test"
	}

	// Load environment variables
	if err := godotenv.Load(envFile); err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", envFile, err)
	}

	config := &LiveTestConfig{
		// Google
		GoogleClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		GoogleRefreshToken: os.Getenv("GOOGLE_OAUTH_REFRESH_TOKEN"),
		GoogleTestEmail:    os.Getenv("GOOGLE_TEST_EMAIL"),

		// Slack
		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
		SlackTestChannelID: os.Getenv("SLACK_TEST_CHANNEL_ID"),
		SlackTestChannelName: os.Getenv("SLACK_TEST_CHANNEL_NAME"),

		// Notion
		NotionAPIKey:        os.Getenv("NOTION_API_KEY"),
		NotionTestDatabaseID: os.Getenv("NOTION_TEST_DATABASE_ID"),
		NotionTestPageID:    os.Getenv("NOTION_TEST_PAGE_ID"),

		// Linear
		LinearAPIKey:       os.Getenv("LINEAR_API_KEY"),
		LinearTestTeamID:   os.Getenv("LINEAR_TEST_TEAM_ID"),
		LinearTestProjectID: os.Getenv("LINEAR_TEST_PROJECT_ID"),

		// HubSpot
		HubSpotAPIKey:         os.Getenv("HUBSPOT_API_KEY"),
		HubSpotTestContactEmail: os.Getenv("HUBSPOT_TEST_CONTACT_EMAIL"),

		// AI Providers
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),

		// BusinessOS Backend
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		SecretKey:           os.Getenv("SECRET_KEY"),
		TokenEncryptionKey:  os.Getenv("TOKEN_ENCRYPTION_KEY"),

		// Test Configuration
		TestCleanupMode: os.Getenv("TEST_CLEANUP_MODE"),
		TestVerbose:     os.Getenv("TEST_VERBOSE") == "true",
	}

	// Parse test categories
	categoriesStr := os.Getenv("TEST_CATEGORIES")
	if categoriesStr == "" {
		categoriesStr = "all"
	}
	config.TestCategories = strings.Split(categoriesStr, ",")
	for i := range config.TestCategories {
		config.TestCategories[i] = strings.TrimSpace(config.TestCategories[i])
	}

	// Default cleanup mode
	if config.TestCleanupMode == "" {
		config.TestCleanupMode = "on_success"
	}

	return config, nil
}

// ShouldTestCategory checks if a category should be tested
func (c *LiveTestConfig) ShouldTestCategory(category string) bool {
	for _, cat := range c.TestCategories {
		if cat == "all" || cat == category {
			return true
		}
	}
	return false
}

// ValidateForCategory checks if required credentials are present for a category
func (c *LiveTestConfig) ValidateForCategory(category string) error {
	switch category {
	case "gmail":
		if c.GoogleClientID == "" || c.GoogleClientSecret == "" || c.GoogleRefreshToken == "" {
			return fmt.Errorf("missing Google OAuth credentials")
		}
		if c.GoogleTestEmail == "" {
			return fmt.Errorf("missing GOOGLE_TEST_EMAIL")
		}
	case "calendar":
		if c.GoogleClientID == "" || c.GoogleClientSecret == "" || c.GoogleRefreshToken == "" {
			return fmt.Errorf("missing Google OAuth credentials")
		}
	case "slack":
		if c.SlackBotToken == "" {
			return fmt.Errorf("missing SLACK_BOT_TOKEN")
		}
		if c.SlackTestChannelID == "" {
			return fmt.Errorf("missing SLACK_TEST_CHANNEL_ID")
		}
	case "notion":
		if c.NotionAPIKey == "" {
			return fmt.Errorf("missing NOTION_API_KEY")
		}
		if c.NotionTestDatabaseID == "" {
			return fmt.Errorf("missing NOTION_TEST_DATABASE_ID")
		}
	case "linear":
		if c.LinearAPIKey == "" {
			return fmt.Errorf("missing LINEAR_API_KEY")
		}
		if c.LinearTestTeamID == "" {
			return fmt.Errorf("missing LINEAR_TEST_TEAM_ID")
		}
	case "hubspot":
		if c.HubSpotAPIKey == "" {
			return fmt.Errorf("missing HUBSPOT_API_KEY")
		}
	case "ai":
		// At least one AI provider required
		if c.AnthropicAPIKey == "" && c.OpenAIAPIKey == "" && c.GroqAPIKey == "" {
			return fmt.Errorf("missing AI provider credentials (need at least one: Anthropic, OpenAI, or Groq)")
		}
	case "businessos":
		if c.DatabaseURL == "" {
			return fmt.Errorf("missing DATABASE_URL")
		}
	}
	return nil
}

// GetAvailableProviders returns list of AI providers with credentials configured
func (c *LiveTestConfig) GetAvailableProviders() []string {
	var providers []string
	if c.AnthropicAPIKey != "" {
		providers = append(providers, "anthropic")
	}
	if c.OpenAIAPIKey != "" {
		providers = append(providers, "openai")
	}
	if c.GroqAPIKey != "" {
		providers = append(providers, "groq")
	}
	return providers
}
