package integrations

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// ---- catalog (FounderOS v1 tests/integrations-catalog.test.ts) ----------------

func TestCatalogIsRichValidAndUnique(t *testing.T) {
	if len(Catalog) < 50 {
		t.Fatalf("catalog has %d entries, want a rich catalog (50+)", len(Catalog))
	}
	seen := map[string]bool{}
	allowed := map[string]bool{}
	for _, c := range Categories {
		allowed[c] = true
	}
	perCat := map[string]int{}
	popular := 0
	for _, e := range Catalog {
		if e.Slug == "" || e.Name == "" || e.Tagline == "" {
			t.Fatalf("entry %+v is missing a field", e)
		}
		if seen[e.Slug] {
			t.Fatalf("duplicate slug %q", e.Slug)
		}
		seen[e.Slug] = true
		if !allowed[e.Category] {
			t.Fatalf("%s: category %q not allowed", e.Slug, e.Category)
		}
		for _, k := range e.EnvKeys {
			if !envName.MatchString(k) {
				t.Fatalf("%s: bad env key %q", e.Slug, k)
			}
		}
		perCat[e.Category]++
		if e.Popular {
			popular++
		}
	}
	for cat, n := range perCat {
		if n < 1 {
			t.Fatalf("category %s empty", cat)
		}
	}
	if popular < 6 {
		t.Fatalf("%d popular, want >= 6", popular)
	}
}

// Every connectorId must be a real row on the bridge's Connections board.
func TestCatalogConnectorIDsAreOnTheBoard(t *testing.T) {
	board := map[string]bool{}
	for _, id := range []string{
		"optimal-engine", "paperclip", "whatsapp", "zernio", "beehiiv", "manychat", "trakyo",
		"typeform", "fathom", "plaud", "docusign", "loom", "vidalytics", "meta-ads", "arcads",
		"wispr", "local-stack", "obsidian", "miro", "email", "calendar", "slack", "payments", "robinhood",
	} {
		board[id] = true
	}
	for _, e := range Catalog {
		if e.ConnectorID != "" && !board[e.ConnectorID] {
			t.Fatalf("%s points at %q, not on the board", e.Slug, e.ConnectorID)
		}
	}
	for _, s := range KeySlots {
		if s.ConnectorID != "" && !board[s.ConnectorID] {
			t.Fatalf("slot %s points at %q, not on the board", s.EnvVar, s.ConnectorID)
		}
	}
}

// GBrain is retired: the Optimal Engine takes G-Brain's tile (same place in
// AI & Automation), and no key slot feeds a gbrain connector.
func TestOptimalEngineReplacesGBrain(t *testing.T) {
	var oe *Integration
	for i := range Catalog {
		if Catalog[i].ConnectorID == "optimal-engine" {
			oe = &Catalog[i]
		}
		if strings.Contains(strings.ToLower(Catalog[i].Slug), "gbrain") {
			t.Fatalf("catalog still lists %s", Catalog[i].Slug)
		}
	}
	if oe == nil || oe.Category != "AI & Automation" {
		t.Fatalf("want an Optimal Engine tile in AI & Automation (G-Brain's place), got %+v", oe)
	}
	if len(ConnectKeysFor(*oe)) != 0 {
		t.Fatal("the engine connects through the topology, not a pasted key")
	}
	for _, s := range KeySlots {
		if strings.Contains(strings.ToUpper(s.EnvVar), "GBRAIN") || s.ConnectorID == "gbrain" {
			t.Fatalf("slot %s still feeds GBrain", s.EnvVar)
		}
	}
}

// FounderOS v1 lib/integrations-catalog.ts (2026-09-30): 70 tools in this
// order, 10 categories, with the Optimal Engine where G-Brain sits.
var prodSlugs = []string{
	"slack", "gmail", "whatsapp", "discord", "telegram", "zoom", "manychat",
	"airtable", "googlesheets", "googledocs", "clickup", "trello", "coda", "wispr",
	"hubspot", "salesforce", "zendesk", "intercom",
	"github", "linear", "jira", "vercel", "sentry", "gitlab",
	"googlecalendar", "calendly", "caldotcom", "googlemeet",
	"stripe", "stripe-vantage", "quickbooks", "xero", "paypal", "wise", "plaid", "robinhood", "paykit", "phantom",
	"mailchimp", "googleanalytics", "meta", "beehiiv", "buffer", "hootsuite", "zernio", "skool", "trakyo", "fathom", "plaud", "docusign",
	"googledrive", "dropbox", "box", "onedrive", "obsidian",
	"openai", "anthropic", "zapier", "make", "n8n", "optimal-engine", "paperclip", "ollama",
	"figma", "canva", "miro", "loom", "typeform", "vidalytics", "arcads",
}

func TestCatalogMatchesProduction(t *testing.T) {
	var got []string
	cats := map[string]bool{}
	for _, e := range Catalog {
		got = append(got, e.Slug)
		cats[e.Category] = true
	}
	if !reflect.DeepEqual(got, prodSlugs) {
		t.Fatalf("catalog slugs differ from production:\n got %v\nwant %v", got, prodSlugs)
	}
	if len(cats) != 10 {
		t.Fatalf("%d categories, production shows 10", len(cats))
	}
	want := map[string]Integration{
		"robinhood": {Slug: "robinhood", Name: "Robinhood", Tagline: "Brokerage & agentic sleeve", Category: "Finance", ConnectorID: "robinhood", EnvKeys: none},
		"paykit":    {Slug: "paykit", Name: "PayKit", Tagline: "Launchpad Cohort checkouts", Category: "Finance", EnvKeys: []string{"PAYKIT_LC_KEY"}},
		"phantom":   {Slug: "phantom", Name: "Phantom", Tagline: "SOL wallet, read-only", Category: "Finance", EnvKeys: none},
		"skool":     {Slug: "skool", Name: "Skool", Tagline: "Community & courses", Category: "Marketing"},
		"paperclip": {Slug: "paperclip", Name: "Paperclip", Tagline: "Agent board & harness", Category: "AI & Automation", ConnectorID: "paperclip", EnvKeys: none},
		"ollama":    {Slug: "ollama", Name: "Local stack", Tagline: "Ollama, tmux & local ports", Category: "AI & Automation", ConnectorID: "local-stack", EnvKeys: none},
	}
	for slug, w := range want {
		e, ok := Find(slug)
		if !ok || !reflect.DeepEqual(e, w) {
			t.Fatalf("%s = %+v, want %+v", slug, e, w)
		}
	}
}

func TestConnectKeysFor(t *testing.T) {
	cases := []struct {
		in   Integration
		want []string
	}{
		{Integration{Slug: "fathom", EnvKeys: []string{"FATHOM_API_KEY"}}, []string{"FATHOM_API_KEY"}},
		{Integration{Slug: "google-docs"}, []string{"GOOGLE_DOCS_API_KEY"}},
		{Integration{Slug: "whatsapp", EnvKeys: []string{}}, []string{}},
	}
	for _, c := range cases {
		if got := ConnectKeysFor(c.in); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s: got %v want %v", c.in.Slug, got, c.want)
		}
	}
}

func st(id string, s connectors.State) connectors.Status {
	return connectors.Status{ID: id, Name: strings.ToUpper(id[:1]) + id[1:], State: s}
}

func TestConnectionCatalogMergesLiveStateOnly(t *testing.T) {
	statuses := []connectors.Status{st("slack", connectors.StateConnected), st("zernio", connectors.StateError), st("typeform", connectors.StateNotConfigured)}
	saved := map[string]string{"TYPEFORM_API_KEY": "x", "DISCORD_API_KEY": "y"}
	cat := ConnectionCatalog(statuses, saved)
	if len(cat) != len(Catalog) {
		t.Fatalf("merge dropped rows: %d vs %d", len(cat), len(Catalog))
	}
	by := map[string]Entry{}
	for _, e := range cat {
		by[e.Slug] = e
	}
	if !by["slack"].Connected {
		t.Fatal("connected connector must mark its tile connected")
	}
	if by["zernio"].Connected || by["typeform"].Connected {
		t.Fatal("error / not_configured must not read connected")
	}
	if !by["typeform"].KeySaved || by["typeform"].Connected {
		t.Fatal("a saved key is stored, never connected")
	}
	if !by["discord"].KeySaved || by["discord"].Connected {
		t.Fatal("a connector-less tile with a key is saved, never connected")
	}
	if by["whatsapp"].KeySaved {
		t.Fatal("guidance-only tiles never read key saved")
	}
	if !reflect.DeepEqual(by["fathom"].Keys, []string{"FATHOM_API_KEY"}) {
		t.Fatalf("entry carries its connect keys, got %v", by["fathom"].Keys)
	}
}

func TestByCategoryKeepsCanonicalOrderAndSkipsEmpty(t *testing.T) {
	cats := ByCategory(Catalog)
	var order []string
	for _, c := range cats {
		order = append(order, c.Name)
		if len(c.Slugs) == 0 {
			t.Fatalf("%s is empty", c.Name)
		}
	}
	idx := map[string]int{}
	for i, c := range Categories {
		idx[c] = i
	}
	for i := 1; i < len(order); i++ {
		if idx[order[i]] < idx[order[i-1]] {
			t.Fatalf("categories out of order: %v", order)
		}
	}
}

// ---- volume (FounderOS v1 tests/integrations-volume.test.ts) ------------------

func tile(slug, cat string, connected, saved, popular bool, keys []string) Entry {
	return Entry{Integration: Integration{Slug: slug, Name: slug, Category: cat, Popular: popular, EnvKeys: keys}, Connected: connected, KeySaved: saved}
}

var volStatuses = []connectors.Status{
	st("slack", connectors.StateConnected),
	st("email", connectors.StateConnected),
	st("stripe", connectors.StateConnected),
	st("notion", connectors.StateNotConfigured),
	st("typeform", connectors.StateNotConfigured),
	st("zernio", connectors.StateError),
}

var volCatalog = []Entry{
	tile("slack", "Communication", true, true, true, nil),
	tile("gmail", "Communication", true, false, true, []string{}),
	tile("zoom", "Communication", false, false, true, nil),
	tile("stripe", "Finance", true, true, false, nil),
	tile("notion", "Knowledge", false, true, false, nil),
	tile("airtable", "Productivity", false, false, true, nil),
}

func TestVolumeHeadlineChipsAndMeters(t *testing.T) {
	v := Volume(volStatuses, volCatalog)
	if v.Headline != 3 {
		t.Fatalf("headline %d", v.Headline)
	}
	if v.Counts != (Counts{Connected: 3, NotConfigured: 2, Error: 1, Total: 6}) {
		t.Fatalf("counts %+v", v.Counts)
	}
	wantChips := []Chip{{Tone: "ok", Text: "3 live"}, {Text: "2 not configured"}, {Tone: "err", Text: "1 erroring"}}
	if !reflect.DeepEqual(v.Chips, wantChips) {
		t.Fatalf("chips %+v", v.Chips)
	}
	if !strings.Contains(v.Caption, "of 6 connector checks") {
		t.Fatalf("caption %q", v.Caption)
	}
	var labels []string
	var fracs []float64
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
		fracs = append(fracs, *m.Frac)
		if !strings.HasPrefix(m.Hue, "var(--bn-") {
			t.Fatalf("meter hue %q is not a theme token", m.Hue)
		}
	}
	if !reflect.DeepEqual(labels, []string{"Connectors live (3/6)", "Catalog tools connected (3/6)", "Keys saved (3/5)", "Popular connected (2/4)"}) {
		t.Fatalf("labels %v", labels)
	}
	if !reflect.DeepEqual(fracs, []float64{0.5, 0.5, 0.6, 0.5}) {
		t.Fatalf("fracs %v", fracs)
	}
	if v.Foot != "6 tools · 4 categories · 1 saved key not live" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestVolumeCategoriesHealthAndInsight(t *testing.T) {
	v := Volume(volStatuses, volCatalog)
	want := []Point{{"Comm", 2}, {"Fin", 1}, {"Prod", 0}, {"Know", 0}}
	if !reflect.DeepEqual(v.ByCategory, want) {
		t.Fatalf("byCategory %+v", v.ByCategory)
	}
	if v.TopCategory == nil || *v.TopCategory != (TopCategory{Name: "Communication", Count: 2}) {
		t.Fatalf("top %+v", v.TopCategory)
	}
	if !reflect.DeepEqual(v.Health, []Point{{"live", 3}, {"unset", 2}, {"error", 1}}) {
		t.Fatalf("health %+v", v.Health)
	}
	if v.Insight.Value != 1 || !strings.Contains(v.Insight.Headline, "1 connector erroring") || !strings.Contains(v.Insight.Body, "Zernio") {
		t.Fatalf("insight %+v", v.Insight)
	}
	if d := v.Insight.Frac - 1.0/6; d > 1e-9 || d < -1e-9 {
		t.Fatalf("insight frac %v", v.Insight.Frac)
	}
}

func TestVolumeHonestWhenEmpty(t *testing.T) {
	v := Volume(nil, nil)
	if v.Headline != 0 || len(v.Meters) != 0 || len(v.Chips) != 0 || len(v.ByCategory) != 0 || v.TopCategory != nil || v.Insight.Frac != 0 {
		t.Fatalf("empty volume invented numbers: %+v", v)
	}
	if v.Meters == nil || v.Chips == nil || v.ByCategory == nil {
		t.Fatal("empty lists must marshal as [], not null")
	}
}

func TestVolumeNothingErroringFallsBackToSavedKeys(t *testing.T) {
	v := Volume(volStatuses[:5], volCatalog)
	if v.Insight.Value != 0 || !strings.Contains(strings.ToLower(v.Insight.Headline), "no connector is erroring") || !strings.Contains(v.Insight.Body, "1 saved key not live") {
		t.Fatalf("insight %+v", v.Insight)
	}
}

// ---- key slots (FounderOS v1 lib/keys.ts) --------------------------------------

func TestSlotStatusesReportPresenceNeverValues(t *testing.T) {
	got := SlotStatuses(func(name string) string {
		if name == "SLACK_BOT_TOKEN" {
			return "xoxb-secret-value"
		}
		return ""
	})
	if len(got) != len(KeySlots) {
		t.Fatalf("%d slots, want %d", len(got), len(KeySlots))
	}
	for _, s := range got {
		if s.EnvVar == "SLACK_BOT_TOKEN" && !s.Present {
			t.Fatal("slack token should read present")
		}
		if s.EnvVar == "STRIPE_SECRET_KEY" && s.Present {
			t.Fatal("stripe key should read unset")
		}
	}
	if raw := strings.Join(func() []string {
		var out []string
		for _, s := range got {
			out = append(out, s.EnvVar, s.Label, s.Hint)
		}
		return out
	}(), " "); strings.Contains(raw, "secret-value") {
		t.Fatal("a slot status leaked the value")
	}
}

func TestSlotByEnvVar(t *testing.T) {
	if s, ok := SlotFor("FATHOM_API_KEY"); !ok || s.ConnectorID != "fathom" {
		t.Fatalf("got %+v %v", s, ok)
	}
	if _, ok := SlotFor("NOPE"); ok {
		t.Fatal("unknown slot resolved")
	}
}

// ---- OAuth readiness (FounderOS v1 tests/oauth-connect-ui.test.ts) -------------

func TestOAuthReadiness(t *testing.T) {
	env := map[string]string{}
	get := func(k string) string { return env[k] }
	if Readiness("fathom", get, 0) != nil {
		t.Fatal("a non-OAuth slug has no readiness")
	}
	r := Readiness("github", get, 0)
	if r == nil || r.AppConfigured || r.Connected || r.RedirectKind != "any" {
		t.Fatalf("bare github: %+v", r)
	}
	env["GITHUB_OAUTH_CLIENT_ID"], env["GITHUB_OAUTH_CLIENT_SECRET"], env["GITHUB_OAUTH_TOKEN"] = "id", "sec", "tok"
	r = Readiness("github", get, 0)
	if !r.AppConfigured || !r.Connected || r.Expired {
		t.Fatalf("registered github: %+v", r)
	}
	env["GOOGLE_OAUTH_TOKEN"], env["GOOGLE_OAUTH_TOKEN_EXPIRES_AT"] = "t", "1000"
	if g := Readiness("google", get, 1_000_000); !g.Expired || g.RedirectKind != "loopback" {
		t.Fatalf("google: %+v", g)
	}
	n := 0
	for _, p := range OAuthProviders {
		if p.RedirectKind == "any" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d providers usable over the tailnet, want exactly 1", n)
	}
}

// ---- env.local writes (FounderOS v1 lib/creds.ts upsert/remove) ----------------

func TestEnvFileUpsertAndRemovePreserveOtherLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte("# bridge\nKEEP=1\nFATHOM_API_KEY=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertEnv(p, map[string]string{"FATHOM_API_KEY": "new", "ZERNIO_API_KEY": "z"}); err != nil {
		t.Fatal(err)
	}
	got := ReadEnv(p)
	if got["KEEP"] != "1" || got["FATHOM_API_KEY"] != "new" || got["ZERNIO_API_KEY"] != "z" {
		t.Fatalf("after upsert %v", got)
	}
	raw, _ := os.ReadFile(p)
	if !strings.HasPrefix(string(raw), "# bridge\n") {
		t.Fatalf("comment lost: %q", raw)
	}
	if fi, _ := os.Stat(p); fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want 0600", fi.Mode().Perm())
	}
	if err := RemoveEnv(p, []string{"FATHOM_API_KEY"}); err != nil {
		t.Fatal(err)
	}
	got = ReadEnv(p)
	if _, ok := got["FATHOM_API_KEY"]; ok || got["ZERNIO_API_KEY"] != "z" || got["KEEP"] != "1" {
		t.Fatalf("after remove %v", got)
	}
}

func TestEnvFileRefusesUnsafeInput(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "env.local")
	if err := UpsertEnv(p, map[string]string{"lower": "x"}); err == nil {
		t.Fatal("bad name accepted")
	}
	if err := UpsertEnv(p, map[string]string{"A_KEY": "a\nb"}); err == nil {
		t.Fatal("multi-line value accepted")
	}
	if err := UpsertEnv(p, map[string]string{"A_KEY": "v"}); err != nil {
		t.Fatalf("fresh file in a new dir: %v", err)
	}
	if ReadEnv(p)["A_KEY"] != "v" {
		t.Fatal("not written")
	}
}

// Production's slot statuses carry a last-4 mask (lib/keys.ts maskSecret) so
// "reveal" can tell WHICH key is set; never more than the tail.
func TestSlotStatusesCarryProductionMask(t *testing.T) {
	got := SlotStatuses(func(name string) string {
		switch name {
		case "SLACK_BOT_TOKEN":
			return "xoxb-secret-value"
		case "WHOP_API_KEY":
			return "abc"
		}
		return ""
	})
	for _, s := range got {
		switch s.EnvVar {
		case "SLACK_BOT_TOKEN":
			if s.Masked != "••••alue" {
				t.Fatalf("slack mask %q", s.Masked)
			}
		case "WHOP_API_KEY":
			if s.Masked != "••••" {
				t.Fatalf("short value mask %q", s.Masked)
			}
		case "STRIPE_SECRET_KEY":
			if s.Masked != "" {
				t.Fatalf("unset slot mask %q", s.Masked)
			}
		}
	}
}

// The API-key groups render in production's order (lib/keys.ts KEY_SLOTS
// first appearance), Notion's own card included; only G-Brain's is gone.
func TestKeySlotGroupsFollowProduction(t *testing.T) {
	var order []string
	seen := map[string]bool{}
	notion := false
	for _, s := range KeySlots {
		if !seen[s.Group] {
			seen[s.Group] = true
			order = append(order, s.Group)
		}
		if s.Group == "Notion" && s.EnvVar == "NOTION_API_KEY" {
			notion = true
		}
	}
	want := []string{"Email", "Slack", "Payments", "Usage", "Knowledge", "Contracts", "Notion", "Social", "Agents", "CRM", "Content"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("groups %v, want %v", order, want)
	}
	if !notion {
		t.Fatal("the Notion card carries NOTION_API_KEY")
	}
}
