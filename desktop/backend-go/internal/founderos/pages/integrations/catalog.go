// Package integrations is the pure logic behind the /os/integrations
// Connections board, ported from FounderOS v1 lib/integrations-catalog.ts,
// lib/integrations-volume.ts, lib/keys.ts and lib/oauth (readiness only).
//
// A tile is connected only when its linked connector answered "connected" on
// this load; a stored key is shown as stored, never as connected. GBrain is
// retired on the bridge, so the Optimal Engine takes G-Brain's tile.
package integrations

import (
	"regexp"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Categories in canonical order (lib/schemas.ts INTEGRATION_CATEGORIES).
var Categories = []string{
	"Productivity", "Communication", "CRM & Sales", "Developer", "Scheduling",
	"Finance", "Marketing", "Storage", "Knowledge", "AI & Automation", "Creative",
}

var envName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// Integration is one catalog tile. EnvKeys nil = a generic <SLUG>_API_KEY;
// an empty (non-nil) slice = guidance only (connects through local setup).
type Integration struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Category    string   `json:"category"`
	ConnectorID string   `json:"connectorId,omitempty"`
	Popular     bool     `json:"popular,omitempty"`
	EnvKeys     []string `json:"envKeys,omitempty"`
}

var none = []string{}

// Catalog is FounderOS v1's INTEGRATIONS (2026-09-30), 70 tools, with the
// Optimal Engine in G-Brain's place.
var Catalog = []Integration{
	// Communication
	{Slug: "slack", Name: "Slack", Tagline: "Channels & DMs", Category: "Communication", ConnectorID: "slack", Popular: true, EnvKeys: []string{"SLACK_BOT_TOKEN"}},
	{Slug: "gmail", Name: "Gmail", Tagline: "Send & read email", Category: "Communication", ConnectorID: "email", Popular: true, EnvKeys: none},
	{Slug: "whatsapp", Name: "WhatsApp", Tagline: "Messages & broadcasts", Category: "Communication", ConnectorID: "whatsapp", EnvKeys: none},
	{Slug: "discord", Name: "Discord", Tagline: "Servers & channels", Category: "Communication"},
	{Slug: "telegram", Name: "Telegram", Tagline: "Chats & bots", Category: "Communication"},
	{Slug: "zoom", Name: "Zoom", Tagline: "Meetings & recordings", Category: "Communication", Popular: true},
	{Slug: "manychat", Name: "ManyChat", Tagline: "IG DM automation", Category: "Communication", ConnectorID: "manychat", EnvKeys: []string{"MANYCHAT_API_KEY"}},

	// Productivity
	{Slug: "airtable", Name: "Airtable", Tagline: "Bases & records", Category: "Productivity", Popular: true},
	{Slug: "googlesheets", Name: "Google Sheets", Tagline: "Read & write spreadsheets", Category: "Productivity"},
	{Slug: "googledocs", Name: "Google Docs", Tagline: "Create & edit documents", Category: "Productivity"},
	{Slug: "clickup", Name: "ClickUp", Tagline: "Docs, tasks & goals", Category: "Productivity"},
	{Slug: "trello", Name: "Trello", Tagline: "Boards & cards", Category: "Productivity"},
	{Slug: "coda", Name: "Coda", Tagline: "Docs that act like apps", Category: "Productivity"},
	{Slug: "wispr", Name: "Wispr Flow", Tagline: "Voice dictation", Category: "Productivity", ConnectorID: "wispr", EnvKeys: none},

	// CRM & Sales
	{Slug: "hubspot", Name: "HubSpot", Tagline: "Contacts & deals", Category: "CRM & Sales", Popular: true},
	{Slug: "salesforce", Name: "Salesforce", Tagline: "Accounts & pipeline", Category: "CRM & Sales"},
	{Slug: "zendesk", Name: "Zendesk", Tagline: "Tickets & support", Category: "CRM & Sales"},
	{Slug: "intercom", Name: "Intercom", Tagline: "Chat & lifecycle", Category: "CRM & Sales"},

	// Developer
	{Slug: "github", Name: "GitHub", Tagline: "Repos, issues & PRs", Category: "Developer", Popular: true},
	{Slug: "linear", Name: "Linear", Tagline: "Issues & projects", Category: "Developer"},
	{Slug: "jira", Name: "Jira", Tagline: "Boards & tickets", Category: "Developer"},
	{Slug: "vercel", Name: "Vercel", Tagline: "Deploys & logs", Category: "Developer"},
	{Slug: "sentry", Name: "Sentry", Tagline: "Errors & traces", Category: "Developer"},
	{Slug: "gitlab", Name: "GitLab", Tagline: "Repos & pipelines", Category: "Developer"},

	// Scheduling
	{Slug: "googlecalendar", Name: "Google Calendar", Tagline: "Events & availability", Category: "Scheduling", ConnectorID: "calendar", Popular: true, EnvKeys: none},
	{Slug: "calendly", Name: "Calendly", Tagline: "Booking links", Category: "Scheduling"},
	{Slug: "caldotcom", Name: "Cal.com", Tagline: "Open scheduling", Category: "Scheduling"},
	{Slug: "googlemeet", Name: "Google Meet", Tagline: "Video calls", Category: "Scheduling"},

	// Finance
	{Slug: "stripe", Name: "Stripe", Tagline: "Payments & invoices", Category: "Finance", ConnectorID: "payments", Popular: true, EnvKeys: []string{"STRIPE_SECRET_KEY"}},
	{Slug: "stripe-vantage", Name: "Stripe · Vantage", Tagline: "Vantage payments & invoices", Category: "Finance", ConnectorID: "payments", EnvKeys: []string{"STRIPE_VANTAGE_KEY"}},
	{Slug: "quickbooks", Name: "QuickBooks", Tagline: "Bookkeeping & P&L", Category: "Finance"},
	{Slug: "xero", Name: "Xero", Tagline: "Accounting & bills", Category: "Finance"},
	{Slug: "paypal", Name: "PayPal", Tagline: "Payments & payouts", Category: "Finance", EnvKeys: []string{"PAYPAL_CLIENT_ID", "PAYPAL_CLIENT_SECRET"}},
	{Slug: "wise", Name: "Wise", Tagline: "Multi-currency balances", Category: "Finance"},
	{Slug: "plaid", Name: "Plaid", Tagline: "Bank connections", Category: "Finance"},
	// The operator's own money stack: Robinhood is the /trading snapshot connector,
	// PayKit the Launchpad Cohort checkout in the payments registry, Phantom
	// the read-only SOL wallet on /trading (a public address, never a key).
	{Slug: "robinhood", Name: "Robinhood", Tagline: "Brokerage & agentic sleeve", Category: "Finance", ConnectorID: "robinhood", EnvKeys: none},
	{Slug: "paykit", Name: "PayKit", Tagline: "Launchpad Cohort checkouts", Category: "Finance", EnvKeys: []string{"PAYKIT_LC_KEY"}},
	{Slug: "phantom", Name: "Phantom", Tagline: "SOL wallet, read-only", Category: "Finance", EnvKeys: none},

	// Marketing
	{Slug: "mailchimp", Name: "Mailchimp", Tagline: "Email campaigns", Category: "Marketing"},
	{Slug: "googleanalytics", Name: "Google Analytics", Tagline: "Traffic & conversions", Category: "Marketing"},
	{Slug: "meta", Name: "Meta Ads", Tagline: "Campaigns & audiences", Category: "Marketing", ConnectorID: "meta-ads", EnvKeys: []string{"META_ADS_ACCESS_TOKEN"}},
	{Slug: "beehiiv", Name: "beehiiv", Tagline: "Newsletter & subscribers", Category: "Marketing", ConnectorID: "beehiiv", EnvKeys: []string{"BEEHIIV_API_KEY"}},
	{Slug: "buffer", Name: "Buffer", Tagline: "Schedule social posts", Category: "Marketing"},
	{Slug: "hootsuite", Name: "Hootsuite", Tagline: "Social management", Category: "Marketing"},
	{Slug: "zernio", Name: "Zernio", Tagline: "Cross-platform posting", Category: "Marketing", ConnectorID: "zernio", EnvKeys: []string{"ZERNIO_API_KEY"}},
	{Slug: "skool", Name: "Skool", Tagline: "Community & courses", Category: "Marketing"},
	{Slug: "trakyo", Name: "Trakyo", Tagline: "Organic attribution", Category: "Marketing", ConnectorID: "trakyo", EnvKeys: []string{"TRAKYO_API_KEY"}},
	{Slug: "fathom", Name: "Fathom", Tagline: "AI notetaker for calls", Category: "CRM & Sales", ConnectorID: "fathom", EnvKeys: []string{"FATHOM_API_KEY"}},
	{Slug: "plaud", Name: "Plaud", Tagline: "AI voice recorder for the room", Category: "CRM & Sales", ConnectorID: "plaud", EnvKeys: []string{"PLAUD_REFRESH_TOKEN"}},
	{Slug: "docusign", Name: "DocuSign", Tagline: "Contracts & e-signatures", Category: "CRM & Sales", ConnectorID: "docusign", EnvKeys: []string{"DOCUSIGN_INTEGRATION_KEY", "DOCUSIGN_USER_ID", "DOCUSIGN_ACCOUNT_ID", "DOCUSIGN_PRIVATE_KEY_B64"}},

	// Storage
	{Slug: "googledrive", Name: "Google Drive", Tagline: "Files & folders", Category: "Storage"},
	{Slug: "dropbox", Name: "Dropbox", Tagline: "Sync & share", Category: "Storage"},
	{Slug: "box", Name: "Box", Tagline: "Content cloud", Category: "Storage"},
	{Slug: "onedrive", Name: "OneDrive", Tagline: "Microsoft files", Category: "Storage"},
	{Slug: "obsidian", Name: "Obsidian", Tagline: "Markdown vault", Category: "Storage", ConnectorID: "obsidian", EnvKeys: none},

	// AI & Automation
	{Slug: "openai", Name: "OpenAI", Tagline: "GPT models & embeddings", Category: "AI & Automation"},
	{Slug: "anthropic", Name: "Anthropic", Tagline: "Claude models", Category: "AI & Automation", Popular: true},
	{Slug: "zapier", Name: "Zapier", Tagline: "Automate anything", Category: "AI & Automation"},
	{Slug: "make", Name: "Make", Tagline: "Visual workflows", Category: "AI & Automation"},
	{Slug: "n8n", Name: "n8n", Tagline: "Self-hosted automation", Category: "AI & Automation"},
	// The OS's own backbone: each has a live connector check. The Optimal
	// Engine sits where FounderOS v1 has G-Brain (GBrain is retired here).
	{Slug: "optimal-engine", Name: "Optimal Engine", Tagline: "Memory · engines & workspaces", Category: "AI & Automation", ConnectorID: "optimal-engine", EnvKeys: none},
	{Slug: "paperclip", Name: "Paperclip", Tagline: "Agent board & harness", Category: "AI & Automation", ConnectorID: "paperclip", EnvKeys: none},
	{Slug: "ollama", Name: "Local stack", Tagline: "Ollama, tmux & local ports", Category: "AI & Automation", ConnectorID: "local-stack", EnvKeys: none},

	// Creative
	{Slug: "figma", Name: "Figma", Tagline: "Design & prototypes", Category: "Creative", Popular: true},
	{Slug: "canva", Name: "Canva", Tagline: "Templates & graphics", Category: "Creative"},
	{Slug: "miro", Name: "Miro", Tagline: "Whiteboards & maps", Category: "Creative", ConnectorID: "miro", EnvKeys: []string{"MIRO_ACCESS_TOKEN"}},
	{Slug: "loom", Name: "Loom", Tagline: "Screen recordings", Category: "Creative", ConnectorID: "loom", EnvKeys: none},
	{Slug: "typeform", Name: "Typeform", Tagline: "Lead forms → funnel", Category: "Creative", ConnectorID: "typeform", EnvKeys: []string{"TYPEFORM_API_KEY"}},
	{Slug: "vidalytics", Name: "Vidalytics", Tagline: "VSL video hosting", Category: "Creative", ConnectorID: "vidalytics", EnvKeys: []string{"VIDALYTICS_API_KEY"}},
	{Slug: "arcads", Name: "Arcads", Tagline: "AI video ads", Category: "Creative", ConnectorID: "arcads", EnvKeys: []string{"ARCADS_BASIC_AUTH"}},
}

// Find returns the catalog entry for slug.
func Find(slug string) (Integration, bool) {
	for _, e := range Catalog {
		if e.Slug == slug {
			return e, true
		}
	}
	return Integration{}, false
}

var nonAlnum = regexp.MustCompile(`[^A-Z0-9]`)

// ConnectKeysFor lists the env names the connect flow may write for an entry.
func ConnectKeysFor(e Integration) []string {
	if e.EnvKeys != nil {
		return append([]string{}, e.EnvKeys...)
	}
	return []string{nonAlnum.ReplaceAllString(strings.ToUpper(e.Slug), "_") + "_API_KEY"}
}

// Entry is a tile with live state merged on. Keys is what Connect asks for.
type Entry struct {
	Integration
	Connected bool     `json:"connected"`
	KeySaved  bool     `json:"keySaved"`
	Keys      []string `json:"keys"`
}

// ConnectionCatalog merges live connector state onto the catalog. saved is a
// fresh read of the bridge's env.local.
func ConnectionCatalog(statuses []connectors.Status, saved map[string]string) []Entry {
	byID := map[string]connectors.State{}
	for _, s := range statuses {
		byID[s.ID] = s.State
	}
	out := make([]Entry, 0, len(Catalog))
	for _, i := range Catalog {
		keys := ConnectKeysFor(i)
		keySaved := len(keys) > 0
		for _, k := range keys {
			if saved[k] == "" {
				keySaved = false
			}
		}
		out = append(out, Entry{
			Integration: i,
			Connected:   i.ConnectorID != "" && byID[i.ConnectorID] == connectors.StateConnected,
			KeySaved:    keySaved,
			Keys:        keys,
		})
	}
	return out
}

// CategoryGroup is one "Browse by category" section.
type CategoryGroup struct {
	Name  string   `json:"name"`
	Slugs []string `json:"slugs"`
}

// ByCategory groups entries in canonical category order, skipping empty ones.
func ByCategory(entries []Integration) []CategoryGroup {
	out := []CategoryGroup{}
	for _, cat := range Categories {
		var slugs []string
		for _, e := range entries {
			if e.Category == cat {
				slugs = append(slugs, e.Slug)
			}
		}
		if len(slugs) > 0 {
			out = append(out, CategoryGroup{Name: cat, Slugs: slugs})
		}
	}
	return out
}
