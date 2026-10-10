package integrations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// KeySlot is one API-key row (FounderOS v1 lib/keys.ts KEY_SLOTS). ConnectorID
// names the connector whose live check answers "does this key work".
type KeySlot struct {
	EnvVar      string `json:"envVar"`
	Label       string `json:"label"`
	Group       string `json:"group"`
	Hint        string `json:"hint,omitempty"`
	ConnectorID string `json:"connectorId,omitempty"`
}

// KeySlots, with the G-Brain store override dropped: GBrain is retired and the
// Optimal Engine is configured through the engine topology, not env.local.
var KeySlots = []KeySlot{
	{EnvVar: "INBOX_1_HOST", Label: "Inbox 1 host", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_1_USER", Label: "Inbox 1 user", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_1_PASS", Label: "Inbox 1 app password", Group: "Email", Hint: "Gmail app password", ConnectorID: "email"},
	{EnvVar: "INBOX_2_HOST", Label: "Inbox 2 host", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_2_USER", Label: "Inbox 2 user", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_2_PASS", Label: "Inbox 2 app password", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_3_HOST", Label: "Inbox 3 host", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_3_USER", Label: "Inbox 3 user", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_3_PASS", Label: "Inbox 3 app password", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_4_HOST", Label: "Inbox 4 host", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_4_USER", Label: "Inbox 4 user", Group: "Email", ConnectorID: "email"},
	{EnvVar: "INBOX_4_PASS", Label: "Inbox 4 app password", Group: "Email", ConnectorID: "email"},
	{EnvVar: "SLACK_BOT_TOKEN", Label: "Slack bot token", Group: "Slack", Hint: "xoxb-… needs chat:write to reply from the OS", ConnectorID: "slack"},
	{EnvVar: "STRIPE_SECRET_KEY", Label: "Stripe secret key", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "CLAUDE_OAUTH_TOKEN", Label: "Claude plan token", Group: "Usage", Hint: "from `claude setup-token`; read-only use: the plan usage gauge"},
	{EnvVar: "NOTION_API_KEY", Label: "Notion integration secret", Group: "Knowledge", Hint: "internal integration; share Brand Deals Hub with it"},
	{EnvVar: "PHANTOM_WALLET_ADDRESS", Label: "Phantom wallet address (public)", Group: "Payments", Hint: "Solana address, read-only balance", ConnectorID: "payments"},
	{EnvVar: "STRIPE_VANTAGE_KEY", Label: "Stripe · Vantage secret key", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "PAYKIT_LC_KEY", Label: "PayKit · Launchpad Cohort API key", Group: "Payments", Hint: "x-api-key for /public-api; rotating it here beats a redeploy", ConnectorID: "payments"},
	{EnvVar: "PAYKIT_VANTAGE_KEY", Label: "PayKit · Vantage API key", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "PAYPAL_CLIENT_ID", Label: "PayPal client id", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "PAYPAL_CLIENT_SECRET", Label: "PayPal client secret", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "SQUARE_ACCESS_TOKEN", Label: "Square access token", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "WHOP_API_KEY", Label: "Whop API key", Group: "Payments", ConnectorID: "payments"},
	{EnvVar: "DOCUSIGN_INTEGRATION_KEY", Label: "DocuSign integration key", Group: "Contracts", Hint: "DocuSign admin → Apps & Keys", ConnectorID: "docusign"},
	{EnvVar: "DOCUSIGN_USER_ID", Label: "DocuSign user ID", Group: "Contracts", Hint: "the API user GUID under Apps & Keys", ConnectorID: "docusign"},
	{EnvVar: "DOCUSIGN_ACCOUNT_ID", Label: "DocuSign account ID", Group: "Contracts", ConnectorID: "docusign"},
	{EnvVar: "DOCUSIGN_PRIVATE_KEY_B64", Label: "DocuSign RSA key (base64)", Group: "Contracts", Hint: "base64 -i private.key | pbcopy", ConnectorID: "docusign"},
	// Production lists the Notion secret twice: under Knowledge and on its own card.
	{EnvVar: "NOTION_API_KEY", Label: "Notion integration secret", Group: "Notion"},
	{EnvVar: "MANYCHAT_API_KEY", Label: "ManyChat API key", Group: "Social", Hint: "ManyChat → Settings → API (Instagram DM automation)", ConnectorID: "manychat"},
	{EnvVar: "HERMES_GATEWAY_URL", Label: "Hermes gateway URL", Group: "Agents", Hint: "the worker-pool gateway the stack check pings; loopback :8642 on the mini", ConnectorID: "local-stack"},
	{EnvVar: "TYPEFORM_API_KEY", Label: "Typeform personal token", Group: "CRM", Hint: "Typeform → Account → Personal tokens; scopes forms:read + responses:read", ConnectorID: "typeform"},
	{EnvVar: "FATHOM_API_KEY", Label: "Fathom API key", Group: "CRM", Hint: "Fathom → Settings → API; sent as X-Api-Key", ConnectorID: "fathom"},
	{EnvVar: "VIDALYTICS_API_KEY", Label: "Vidalytics API key", Group: "Content", Hint: "Account Settings → Global Settings → API (Premium+); sent as X-API-Key", ConnectorID: "vidalytics"},
	{EnvVar: "PLAUD_REFRESH_TOKEN", Label: "Plaud refresh token", Group: "Knowledge", Hint: "refresh_token from ~/.plaud/tokens-mcp.json after `claude mcp` signs in; the OS mints its own access tokens", ConnectorID: "plaud"},
}

// SlotStatus is a slot plus whether a value resolves and production's last-4
// mask (lib/keys.ts maskSecret). Never the value.
type SlotStatus struct {
	KeySlot
	Present bool   `json:"present"`
	Masked  string `json:"masked"`
}

// MaskSecret is lib/keys.ts maskSecret: "" unset, "••••" for four runes or
// fewer, else "••••" plus the last four.
func MaskSecret(v string) string {
	if v == "" {
		return ""
	}
	r := []rune(v)
	if len(r) <= 4 {
		return "••••"
	}
	return "••••" + string(r[len(r)-4:])
}

func SlotStatuses(resolve func(string) string) []SlotStatus {
	out := make([]SlotStatus, 0, len(KeySlots))
	for _, s := range KeySlots {
		v := resolve(s.EnvVar)
		out = append(out, SlotStatus{KeySlot: s, Present: v != "", Masked: MaskSecret(v)})
	}
	return out
}

func SlotFor(envVar string) (KeySlot, bool) {
	for _, s := range KeySlots {
		if s.EnvVar == envVar {
			return s, true
		}
	}
	return KeySlot{}, false
}

// ---- OAuth readiness (lib/oauth/providers.ts + store.ts oauthReadiness) -----

type OAuthProvider struct {
	Slug, Name, RedirectKind, ConsoleURL string
	ClientIDEnv, ClientSecretEnv         string
	AccessTokenEnv                       string
}

// OAuthProviders are the authorization-code providers the board knows. The
// flow itself (start/callback) is not on the bridge yet; the board only shows
// readiness so it never offers a button that cannot finish.
var OAuthProviders = []OAuthProvider{
	{"github", "GitHub", "any", "https://github.com/settings/developers", "GITHUB_OAUTH_CLIENT_ID", "GITHUB_OAUTH_CLIENT_SECRET", "GITHUB_OAUTH_TOKEN"},
	{"google", "Google (Gmail, Calendar, Drive)", "loopback", "https://console.cloud.google.com/apis/credentials", "GOOGLE_OAUTH_CLIENT_ID", "GOOGLE_OAUTH_CLIENT_SECRET", "GOOGLE_OAUTH_TOKEN"},
	{"slack", "Slack", "https-public", "https://api.slack.com/apps", "SLACK_OAUTH_CLIENT_ID", "SLACK_OAUTH_CLIENT_SECRET", "SLACK_BOT_TOKEN"},
	{"notion", "Notion", "https-public", "https://www.notion.so/my-integrations", "NOTION_OAUTH_CLIENT_ID", "NOTION_OAUTH_CLIENT_SECRET", "NOTION_API_KEY"},
	{"hubspot", "HubSpot", "https-public", "https://developers.hubspot.com/", "HUBSPOT_OAUTH_CLIENT_ID", "HUBSPOT_OAUTH_CLIENT_SECRET", "HUBSPOT_ACCESS_TOKEN"},
	{"linear", "Linear", "https-public", "https://linear.app/settings/api/applications", "LINEAR_OAUTH_CLIENT_ID", "LINEAR_OAUTH_CLIENT_SECRET", "LINEAR_API_KEY"},
	{"stripe", "Stripe Connect", "https-public", "https://dashboard.stripe.com/settings/connect", "STRIPE_OAUTH_CLIENT_ID", "STRIPE_SECRET_KEY", "STRIPE_CONNECT_ACCESS_TOKEN"},
	{"zoom", "Zoom", "https-public", "https://marketplace.zoom.us/develop/create", "ZOOM_OAUTH_CLIENT_ID", "ZOOM_OAUTH_CLIENT_SECRET", "ZOOM_ACCESS_TOKEN"},
	{"jira", "Jira (Atlassian)", "https-public", "https://developer.atlassian.com/console/myapps/", "ATLASSIAN_OAUTH_CLIENT_ID", "ATLASSIAN_OAUTH_CLIENT_SECRET", "ATLASSIAN_ACCESS_TOKEN"},
	{"vercel", "Vercel", "https-public", "https://vercel.com/dashboard/integrations/console", "VERCEL_OAUTH_CLIENT_ID", "VERCEL_OAUTH_CLIENT_SECRET", "VERCEL_ACCESS_TOKEN"},
}

type OAuthReadiness struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	RedirectKind    string `json:"redirectKind"`
	ConsoleURL      string `json:"consoleUrl"`
	ClientIDEnv     string `json:"clientIdEnv"`
	ClientSecretEnv string `json:"clientSecretEnv"`
	AppConfigured   bool   `json:"appConfigured"`
	Connected       bool   `json:"connected"`
	Expired         bool   `json:"expired"`
}

// Readiness reports a provider's app-registration and token state; nil for a
// slug with no OAuth flow. nowMs is epoch milliseconds.
func Readiness(slug string, get func(string) string, nowMs int64) *OAuthReadiness {
	for _, p := range OAuthProviders {
		if p.Slug != slug {
			continue
		}
		expired := false
		if at, err := strconv.ParseInt(get(p.AccessTokenEnv+"_EXPIRES_AT"), 10, 64); err == nil && at != 0 {
			expired = nowMs >= at-60_000
		}
		return &OAuthReadiness{
			Slug: p.Slug, Name: p.Name, RedirectKind: p.RedirectKind, ConsoleURL: p.ConsoleURL,
			ClientIDEnv: p.ClientIDEnv, ClientSecretEnv: p.ClientSecretEnv,
			AppConfigured: get(p.ClientIDEnv) != "" && get(p.ClientSecretEnv) != "",
			Connected:     get(p.AccessTokenEnv) != "",
			Expired:       expired,
		}
	}
	return nil
}

// ---- env.local (the bridge's own; lib/creds.ts upsertEnvLocal/removeEnvLocal)

// ReadEnv parses an env file; a missing file is empty.
func ReadEnv(path string) map[string]string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	return connectors.ParseEnvFile(string(raw))
}

func lineKey(line string) string {
	l := strings.TrimPrefix(strings.TrimSpace(line), "export ")
	if i := strings.Index(l, "="); i > 0 {
		return strings.TrimSpace(l[:i])
	}
	return ""
}

func rewrite(path string, edit func([]string) []string) error {
	var lines []string
	if raw, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(raw), "\n")
	}
	lines = edit(lines)
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// UpsertEnv sets KEY=value lines, preserving every other line.
// refuseSymlink stops a write from landing in another checkout's file.
func refuseSymlink(path string) error {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink; refusing to write through it", path)
	}
	return nil
}

func UpsertEnv(path string, values map[string]string) error {
	if err := refuseSymlink(path); err != nil {
		return err
	}
	for k, v := range values {
		if !envName.MatchString(k) {
			return fmt.Errorf("invalid env var name: %s", k)
		}
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("%s: value must be a single line", k)
		}
	}
	return rewrite(path, func(lines []string) []string {
		done := map[string]bool{}
		for i, l := range lines {
			if k := lineKey(l); k != "" {
				if v, ok := values[k]; ok && !done[k] {
					lines[i], done[k] = k+"="+v, true
				}
			}
		}
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		for _, k := range sortedKeys(values) {
			if !done[k] {
				lines = append(lines, k+"="+values[k])
			}
		}
		return lines
	})
}

// RemoveEnv deletes the named keys' lines.
func RemoveEnv(path string, keys []string) error {
	if err := refuseSymlink(path); err != nil {
		return err
	}
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	return rewrite(path, func(lines []string) []string {
		out := lines[:0]
		for _, l := range lines {
			if !drop[lineKey(l)] {
				out = append(out, l)
			}
		}
		return out
	})
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
