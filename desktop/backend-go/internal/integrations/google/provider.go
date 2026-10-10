// Package google provides the Google Workspace integration (Calendar, Gmail, Drive).
package google

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Provider implements the integrations.Provider interface for Google Workspace.
type Provider struct {
	pool        *pgxpool.Pool
	oauthConfig *oauth2.Config
	features    []string // enabled features: "calendar", "gmail", "drive"
}

// AllFeatures contains all available Google feature identifiers
var AllFeatures = []string{
	"calendar", "gmail", "drive", "contacts", "tasks",
	"sheets", "docs", "slides", "forms", "chat",
	"photos", "youtube", "blogger", "classroom", "cloud",
	"meet", "keep", "analytics", "admin", "fitness",
	"ads", "searchconsole", "bigquery", "pubsub", "storage",
}

// NewProvider creates a new Google provider with specified features.
// Available features: calendar, gmail, drive, contacts, tasks, sheets, docs, slides, forms, chat,
// photos, youtube, blogger, classroom, cloud, meet, keep, analytics, admin, fitness, ads, searchconsole, bigquery, pubsub, storage
// Use "all" to enable ALL features with maximum scope access.
func NewProvider(pool *pgxpool.Pool, features []string) *Provider {
	cfg := config.AppConfig

	// Check if "all" is requested
	for _, f := range features {
		if f == "all" {
			features = AllFeatures
			break
		}
	}

	// Build scopes based on enabled features
	scopes := append([]string{}, UserInfoScopes...)
	for _, feature := range features {
		switch feature {
		case "calendar":
			scopes = append(scopes, CalendarScopes...)
		case "gmail":
			scopes = append(scopes, GmailScopes...)
		case "drive":
			scopes = append(scopes, DriveScopes...)
		case "contacts":
			scopes = append(scopes, ContactsScopes...)
		case "tasks":
			scopes = append(scopes, TasksScopes...)
		case "sheets":
			scopes = append(scopes, SheetsScopes...)
		case "docs":
			scopes = append(scopes, DocsScopes...)
		case "slides":
			scopes = append(scopes, SlidesScopes...)
		case "forms":
			scopes = append(scopes, FormsScopes...)
		case "chat":
			scopes = append(scopes, ChatScopes...)
		case "photos":
			scopes = append(scopes, PhotosScopes...)
		case "youtube":
			scopes = append(scopes, YouTubeScopes...)
		case "blogger":
			scopes = append(scopes, BloggerScopes...)
		case "classroom":
			scopes = append(scopes, ClassroomScopes...)
		case "cloud":
			scopes = append(scopes, CloudScopes...)
		case "meet":
			scopes = append(scopes, MeetScopes...)
		case "keep":
			scopes = append(scopes, KeepScopes...)
		case "analytics":
			scopes = append(scopes, AnalyticsScopes...)
		case "admin":
			scopes = append(scopes, AdminScopes...)
		case "fitness":
			scopes = append(scopes, FitnessScopes...)
		case "ads":
			scopes = append(scopes, AdsScopes...)
		case "searchconsole":
			scopes = append(scopes, SearchConsoleScopes...)
		case "bigquery":
			scopes = append(scopes, BigQueryScopes...)
		case "pubsub":
			scopes = append(scopes, PubSubScopes...)
		case "storage":
			scopes = append(scopes, StorageScopes...)
		}
	}

	redirectURI := cfg.GoogleIntegrationRedirectURI
	if redirectURI == "" {
		redirectURI = cfg.GoogleRedirectURI
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	return &Provider{
		pool:        pool,
		oauthConfig: oauthConfig,
		features:    features,
	}
}

// NewProviderWithAllFeatures creates a provider with ALL Google features enabled.
// This requests the maximum possible OAuth scopes for full API access.
func NewProviderWithAllFeatures(pool *pgxpool.Pool) *Provider {
	return NewProvider(pool, []string{"all"})
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderID
}

// DisplayName returns the human-readable provider name.
func (p *Provider) DisplayName() string {
	return ProviderName
}

// Category returns the provider category.
func (p *Provider) Category() string {
	return Category
}

// Icon returns the provider icon URL.
func (p *Provider) Icon() string {
	return "/icons/google.svg"
}

// Features returns the enabled features for this provider.
func (p *Provider) Features() []string {
	return p.features
}

// HasFeature checks if a specific feature is enabled.
func (p *Provider) HasFeature(feature string) bool {
	for _, f := range p.features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetAuthURL returns the OAuth authorization URL.
func (p *Provider) GetAuthURL(state string) string {
	return p.oauthConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// GetAuthURLWithFeatures returns an OAuth URL with specific feature scopes.
// Available features: calendar, gmail, drive, contacts, tasks, sheets, docs, slides, forms, chat,
// photos, youtube, blogger, classroom, cloud, meet, keep, analytics, admin, fitness, ads, searchconsole, bigquery, pubsub, storage
// Use "all" to request ALL scopes.
func (p *Provider) GetAuthURLWithFeatures(state string, features []string) string {
	// Check if "all" is requested
	for _, f := range features {
		if f == "all" {
			features = AllFeatures
			break
		}
	}

	scopes := append([]string{}, UserInfoScopes...)
	for _, feature := range features {
		switch feature {
		case "calendar":
			scopes = append(scopes, CalendarScopes...)
		case "gmail":
			scopes = append(scopes, GmailScopes...)
		case "drive":
			scopes = append(scopes, DriveScopes...)
		case "contacts":
			scopes = append(scopes, ContactsScopes...)
		case "tasks":
			scopes = append(scopes, TasksScopes...)
		case "sheets":
			scopes = append(scopes, SheetsScopes...)
		case "docs":
			scopes = append(scopes, DocsScopes...)
		case "slides":
			scopes = append(scopes, SlidesScopes...)
		case "forms":
			scopes = append(scopes, FormsScopes...)
		case "chat":
			scopes = append(scopes, ChatScopes...)
		case "photos":
			scopes = append(scopes, PhotosScopes...)
		case "youtube":
			scopes = append(scopes, YouTubeScopes...)
		case "blogger":
			scopes = append(scopes, BloggerScopes...)
		case "classroom":
			scopes = append(scopes, ClassroomScopes...)
		case "cloud":
			scopes = append(scopes, CloudScopes...)
		case "meet":
			scopes = append(scopes, MeetScopes...)
		case "keep":
			scopes = append(scopes, KeepScopes...)
		case "analytics":
			scopes = append(scopes, AnalyticsScopes...)
		case "admin":
			scopes = append(scopes, AdminScopes...)
		case "fitness":
			scopes = append(scopes, FitnessScopes...)
		case "ads":
			scopes = append(scopes, AdsScopes...)
		case "searchconsole":
			scopes = append(scopes, SearchConsoleScopes...)
		case "bigquery":
			scopes = append(scopes, BigQueryScopes...)
		case "pubsub":
			scopes = append(scopes, PubSubScopes...)
		case "storage":
			scopes = append(scopes, StorageScopes...)
		}
	}

	// Create a temporary config with the requested scopes
	tempConfig := &oauth2.Config{
		ClientID:     p.oauthConfig.ClientID,
		ClientSecret: p.oauthConfig.ClientSecret,
		RedirectURL:  p.oauthConfig.RedirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}

	return tempConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// GetAuthURLWithAllScopes returns an OAuth URL with ALL available Google scopes.
func (p *Provider) GetAuthURLWithAllScopes(state string) string {
	return p.GetAuthURLWithFeatures(state, []string{"all"})
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}

// OAuthConfig returns the OAuth config.
func (p *Provider) OAuthConfig() *oauth2.Config {
	return p.oauthConfig
}

// Helper functions
