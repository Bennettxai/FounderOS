// Package microsoft provides Microsoft 365 integration (Outlook, OneDrive, Teams, etc.).
package microsoft

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"golang.org/x/oauth2"
)

// Provider implements the integrations.Provider interface for Microsoft 365.
type Provider struct {
	pool        *pgxpool.Pool
	oauthConfig *oauth2.Config
	features    []string
}

// AllFeatures contains all available Microsoft feature identifiers
var AllFeatures = []string{
	"mail", "calendar", "contacts", "files", "tasks",
	"onenote", "teams", "sites", "groups", "planner",
	"directory", "people", "bookings", "reports", "security",
}

// NewProvider creates a new Microsoft provider with specified features.
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
	scopes := append([]string{}, OpenIDScopes...)
	scopes = append(scopes, OfflineScopes...)
	scopes = append(scopes, UserScopes[:2]...) // Basic user scopes

	for _, feature := range features {
		switch feature {
		case "mail":
			scopes = append(scopes, MailScopes...)
		case "calendar":
			scopes = append(scopes, CalendarScopes...)
		case "contacts":
			scopes = append(scopes, ContactsScopes...)
		case "files":
			scopes = append(scopes, FilesScopes...)
		case "tasks":
			scopes = append(scopes, TasksScopes...)
		case "onenote":
			scopes = append(scopes, OneNoteScopes...)
		case "teams":
			scopes = append(scopes, TeamsScopes...)
		case "sites":
			scopes = append(scopes, SitesScopes...)
		case "groups":
			scopes = append(scopes, GroupsScopes...)
		case "planner":
			scopes = append(scopes, PlannerScopes...)
		case "directory":
			scopes = append(scopes, DirectoryScopes...)
		case "people":
			scopes = append(scopes, PeopleScopes...)
		case "bookings":
			scopes = append(scopes, BookingsScopes...)
		case "reports":
			scopes = append(scopes, ReportsScopes...)
		case "security":
			scopes = append(scopes, SecurityScopes...)
		}
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.MicrosoftClientID,
		ClientSecret: cfg.MicrosoftClientSecret,
		RedirectURL:  cfg.MicrosoftRedirectURI,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}

	return &Provider{
		pool:        pool,
		oauthConfig: oauthConfig,
		features:    features,
	}
}

// NewProviderWithAllFeatures creates a provider with ALL Microsoft features enabled.
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
	return "/icons/microsoft.svg"
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
func (p *Provider) GetAuthURLWithFeatures(state string, features []string) string {
	// Check if "all" is requested
	for _, f := range features {
		if f == "all" {
			features = AllFeatures
			break
		}
	}

	scopes := append([]string{}, OpenIDScopes...)
	scopes = append(scopes, OfflineScopes...)
	scopes = append(scopes, UserScopes[:2]...)

	for _, feature := range features {
		switch feature {
		case "mail":
			scopes = append(scopes, MailScopes...)
		case "calendar":
			scopes = append(scopes, CalendarScopes...)
		case "contacts":
			scopes = append(scopes, ContactsScopes...)
		case "files":
			scopes = append(scopes, FilesScopes...)
		case "tasks":
			scopes = append(scopes, TasksScopes...)
		case "onenote":
			scopes = append(scopes, OneNoteScopes...)
		case "teams":
			scopes = append(scopes, TeamsScopes...)
		case "sites":
			scopes = append(scopes, SitesScopes...)
		case "groups":
			scopes = append(scopes, GroupsScopes...)
		case "planner":
			scopes = append(scopes, PlannerScopes...)
		case "directory":
			scopes = append(scopes, DirectoryScopes...)
		case "people":
			scopes = append(scopes, PeopleScopes...)
		case "bookings":
			scopes = append(scopes, BookingsScopes...)
		case "reports":
			scopes = append(scopes, ReportsScopes...)
		case "security":
			scopes = append(scopes, SecurityScopes...)
		}
	}

	tempConfig := &oauth2.Config{
		ClientID:     p.oauthConfig.ClientID,
		ClientSecret: p.oauthConfig.ClientSecret,
		RedirectURL:  p.oauthConfig.RedirectURL,
		Scopes:       scopes,
		Endpoint:     p.oauthConfig.Endpoint,
	}

	return tempConfig.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// Pool returns the database pool.
func (p *Provider) Pool() *pgxpool.Pool {
	return p.pool
}

// OAuthConfig returns the OAuth config.
func (p *Provider) OAuthConfig() *oauth2.Config {
	return p.oauthConfig
}

// Helper types and functions
