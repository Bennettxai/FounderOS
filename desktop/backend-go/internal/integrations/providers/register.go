// Package providers handles provider initialization and registration.
// Import this package in main.go to auto-register all providers.
package providers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/integrations/airtable"
	"github.com/rhl/businessos-backend/internal/integrations/clickup"
	"github.com/rhl/businessos-backend/internal/integrations/fathom"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/integrations/hubspot"
	"github.com/rhl/businessos-backend/internal/integrations/linear"
	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
	"github.com/rhl/businessos-backend/internal/integrations/notion"
	"github.com/rhl/businessos-backend/internal/integrations/slack"
)

// InitializeProviders creates and registers all available providers.
// Call this function during application startup.
func InitializeProviders(pool *pgxpool.Pool) {
	// ============================================================================
	// PRODUCTIVITY INTEGRATIONS
	// ============================================================================

	// Google Workspace - Only request basic scopes (calendar + gmail)
	// Other APIs must be enabled in Google Cloud Console first
	googleProvider := google.NewProvider(pool, []string{"calendar", "gmail"})
	integrations.Register(googleProvider)

	// Microsoft 365 - Only request basic scopes (calendar + mail)
	microsoftProvider := microsoft.NewProvider(pool, []string{"calendar", "mail"})
	integrations.Register(microsoftProvider)

	// Notion - Docs, Databases, Projects
	notionProvider := notion.NewProvider(pool)
	integrations.Register(notionProvider)

	// ============================================================================
	// COMMUNICATION INTEGRATIONS
	// ============================================================================

	// Slack - Messaging, Channels, Notifications (ALL scopes)
	slackProvider := slack.NewProvider(pool)
	integrations.Register(slackProvider)

	// ============================================================================
	// PROJECT MANAGEMENT INTEGRATIONS
	// ============================================================================

	// Linear - Issues, Projects, Teams
	linearProvider := linear.NewProvider(pool)
	integrations.Register(linearProvider)

	// ============================================================================
	// CRM INTEGRATIONS
	// ============================================================================

	// HubSpot - Contacts, Companies, Deals (ALL scopes)
	hubspotProvider := hubspot.NewProvider(pool)
	integrations.Register(hubspotProvider)

	// ============================================================================
	// ANALYTICS INTEGRATIONS
	// ============================================================================

	// Fathom Analytics - Website analytics (API key auth)
	fathomProvider := fathom.NewProvider(pool)
	integrations.Register(fathomProvider)

	// ============================================================================
	// ADDITIONAL PROJECT MANAGEMENT
	// ============================================================================

	// ClickUp - Tasks, Lists, Spaces, Folders
	clickupProvider := clickup.NewProvider(pool)
	integrations.Register(clickupProvider)

	// ============================================================================
	// DATABASE/SPREADSHEET INTEGRATIONS
	// ============================================================================

	// Airtable - Bases, Tables, Records
	airtableProvider := airtable.NewProvider(pool)
	integrations.Register(airtableProvider)
}

// InitializeProvider creates and registers a specific provider.
func InitializeProvider(pool *pgxpool.Pool, name string) error {
	switch name {
	case "google":
		integrations.Register(google.NewProvider(pool, []string{"calendar", "gmail"}))
	case "microsoft":
		integrations.Register(microsoft.NewProvider(pool, []string{"calendar", "mail"}))
	case "slack":
		integrations.Register(slack.NewProvider(pool))
	case "notion":
		integrations.Register(notion.NewProvider(pool))
	case "linear":
		integrations.Register(linear.NewProvider(pool))
	case "hubspot":
		integrations.Register(hubspot.NewProvider(pool))
	case "fathom":
		integrations.Register(fathom.NewProvider(pool))
	case "clickup":
		integrations.Register(clickup.NewProvider(pool))
	case "airtable":
		integrations.Register(airtable.NewProvider(pool))
	default:
		return nil // Unknown provider, skip
	}
	return nil
}

// GetGoogleProvider returns a Google provider instance with basic features.
// Use this when you need direct access to Google-specific methods.
func GetGoogleProvider(pool *pgxpool.Pool) *google.Provider {
	return google.NewProvider(pool, []string{"calendar", "gmail"})
}

// GetGoogleProviderWithFeatures returns a Google provider with specific features.
// Features: "calendar", "gmail", "drive", "contacts", "tasks", "sheets", "docs", etc.
func GetGoogleProviderWithFeatures(pool *pgxpool.Pool, features []string) *google.Provider {
	return google.NewProvider(pool, features)
}

// GetMicrosoftProvider returns a Microsoft 365 provider instance with all features.
// Use this when you need direct access to Microsoft-specific methods.
func GetMicrosoftProvider(pool *pgxpool.Pool) *microsoft.Provider {
	return microsoft.NewProviderWithAllFeatures(pool)
}

// GetMicrosoftProviderWithFeatures returns a Microsoft provider with specific features.
// Features: "mail", "calendar", "contacts", "files", "tasks", "onenote", "teams", etc.
func GetMicrosoftProviderWithFeatures(pool *pgxpool.Pool, features []string) *microsoft.Provider {
	return microsoft.NewProvider(pool, features)
}

// GetSlackProvider returns a Slack provider instance.
func GetSlackProvider(pool *pgxpool.Pool) *slack.Provider {
	return slack.NewProvider(pool)
}

// GetNotionProvider returns a Notion provider instance.
func GetNotionProvider(pool *pgxpool.Pool) *notion.Provider {
	return notion.NewProvider(pool)
}

// GetLinearProvider returns a Linear provider instance.
func GetLinearProvider(pool *pgxpool.Pool) *linear.Provider {
	return linear.NewProvider(pool)
}

// GetHubSpotProvider returns a HubSpot provider instance.
func GetHubSpotProvider(pool *pgxpool.Pool) *hubspot.Provider {
	return hubspot.NewProvider(pool)
}

// GetFathomProvider returns a Fathom Analytics provider instance.
func GetFathomProvider(pool *pgxpool.Pool) *fathom.Provider {
	return fathom.NewProvider(pool)
}

// GetClickUpProvider returns a ClickUp provider instance.
func GetClickUpProvider(pool *pgxpool.Pool) *clickup.Provider {
	return clickup.NewProvider(pool)
}

// GetAirtableProvider returns an Airtable provider instance.
func GetAirtableProvider(pool *pgxpool.Pool) *airtable.Provider {
	return airtable.NewProvider(pool)
}
