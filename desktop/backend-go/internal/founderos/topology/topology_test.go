package topology

import (
	"strings"
	"testing"
)

func TestRepoTopologyHoldsInvariants(t *testing.T) {
	topo, err := LoadRepo()
	if err != nil {
		t.Fatalf("load repo topology: %v", err)
	}
	if err := topo.Validate(); err != nil {
		t.Fatalf("repo topology violates invariants: %v", err)
	}
}

func TestRepoTopologyEncodesTheDefaultSplit(t *testing.T) {
	topo, err := LoadRepo()
	if err != nil {
		t.Fatalf("load repo topology: %v", err)
	}
	// A fresh install homes every workspace on the one bundled engine.
	for _, slug := range []string{"founderos", "vantage", "launchpad-cohort", "personal"} {
		got, err := topo.HomeOf(slug)
		if err != nil {
			t.Errorf("HomeOf(%q): %v", slug, err)
			continue
		}
		if got.Name != "local" {
			t.Errorf("HomeOf(%q) = %q, want local", slug, got.Name)
		}
		if got.Tier != "device" {
			t.Errorf("HomeOf(%q) tier = %q, want device (personal data must stay on a device engine)", slug, got.Tier)
		}
	}
	if got, want := len(topo.Workspaces), 4; got != want {
		t.Errorf("repo topology has %d workspaces, want %d", got, want)
	}
	if _, err := topo.HomeOf("personal-brand"); err == nil {
		t.Error("personal-brand is not a demo workspace and must not resolve to an engine")
	}
	if ws := topo.WorkspaceForBrainPath("conversations/2026-09-01-chat.md"); ws != "personal" {
		t.Errorf("conversations/ routes to %q, want personal", ws)
	}
	if ws := topo.WorkspaceForBrainPath("memories/2026-09-01.md"); ws != "personal" {
		t.Errorf("memories/ routes to %q, want personal", ws)
	}
	if ws := topo.WorkspaceForBrainPath("vantage/offer.md"); ws != "vantage" {
		t.Errorf("vantage/ routes to %q, want vantage", ws)
	}
	if ws := topo.WorkspaceForBrainPath("launchpad-cohort/curriculum.md"); ws != "launchpad-cohort" {
		t.Errorf("launchpad-cohort/ routes to %q, want launchpad-cohort", ws)
	}
	if ws := topo.WorkspaceForBrainPath("some-new-folder/x.md"); ws != topo.BrainStore.Default {
		t.Errorf("unknown folder routes to %q, want default %q", ws, topo.BrainStore.Default)
	}
	if ws := topo.WorkspaceForBrainPath("README.md"); ws != topo.BrainStore.Default {
		t.Errorf("root page routes to %q, want default %q", ws, topo.BrainStore.Default)
	}
}

func TestRetiredDefaultWorkspaceIsNotAHome(t *testing.T) {
	topo, err := LoadRepo()
	if err != nil {
		t.Fatalf("load repo topology: %v", err)
	}
	if _, err := topo.HomeOf("default"); err == nil {
		t.Fatal("workspace 'default' must not resolve to an engine in the repo topology")
	}

	// The retirement rule itself, on an inline topology: a retired slug
	// names the workspace it merged into and never resolves to an engine.
	retired := mustParse(t, strings.Replace(validYAML, "retired: {}", "retired: {default: acme}", 1))
	if err := retired.Validate(); err != nil {
		t.Fatalf("topology with a retired workspace should be valid: %v", err)
	}
	if _, err := retired.HomeOf("default"); err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("HomeOf(retired) = %v, want a retired error", err)
	}
	if got := retired.Retired["default"]; got != "acme" {
		t.Fatalf("default merges into %q, want acme", got)
	}
	bad := mustParse(t, strings.Replace(validYAML, "retired: {}", "retired: {default: nope}", 1))
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "unknown workspace") {
		t.Fatalf("Validate() = %v, want a retired-into-unknown error", err)
	}
}

const validYAML = `
version: 1
engines:
  macbook: {tier: device}
  hub: {tier: shared}
  cloud: {tier: shared, deferred: true}
workspaces:
  - {slug: acme, home: hub, class: business}
  - {slug: personal, home: macbook, class: personal}
brain_store:
  default: acme
  routes: {conversations: personal}
retired: {}
`

func mustParse(t *testing.T, doc string) *Topology {
	t.Helper()
	topo, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return topo
}

func TestValidateRejectsBrokenTopologies(t *testing.T) {
	if err := mustParse(t, validYAML).Validate(); err != nil {
		t.Fatalf("baseline fixture should be valid: %v", err)
	}
	cases := []struct {
		name, from, to, wantErr string
	}{
		{"duplicate home", "  - {slug: personal, home: macbook, class: personal}\n",
			"  - {slug: personal, home: macbook, class: personal}\n  - {slug: acme, home: macbook, class: business}\n", "more than one home"},
		{"missing home", "{slug: acme, home: hub, class: business}", "{slug: acme, home: \"\", class: business}", "no home"},
		{"unknown engine", "{slug: acme, home: hub, class: business}", "{slug: acme, home: moon, class: business}", "unknown engine"},
		{"personal on shared", "{slug: personal, home: macbook, class: personal}", "{slug: personal, home: hub, class: personal}", "must live on a device engine"},
		{"device on cloud", "{slug: personal, home: macbook, class: personal}", "{slug: personal, home: cloud, class: device}", "deferred"},
		{"conversations to business", "routes: {conversations: personal}", "routes: {conversations: acme}", "conversations"},
		{"route to unknown workspace", "routes: {conversations: personal}", "routes: {conversations: personal, x: nope}", "unknown workspace"},
		{"bad class", "{slug: acme, home: hub, class: business}", "{slug: acme, home: hub, class: secret}", "class"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := strings.Replace(validYAML, tc.from, tc.to, 1)
			if doc == validYAML {
				t.Fatalf("fixture edit %q did not apply", tc.from)
			}
			err := mustParse(t, doc).Validate()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestMeetingRulesRouteByHeaderInOrder(t *testing.T) {
	topo := mustParse(t, validYAML+`
meeting_rules:
  - {field: title, contains: launchpad, workspace: acme}
  - {field: participants, contains: "@personal.ai", workspace: personal}
`)
	topo.BrainStore.Routes["meetings"] = "acme"
	cases := []struct{ header, want string }{
		{"---\ntitle: \"Launchpad Cohort Morning Meeting\"\nparticipants: sam@personal.ai\n---", "acme"},
		{"---\ntitle: \"Jordan and Alex\"\nparticipants: alex@PERSONAL.ai, jordan@x.com\n---", "personal"},
		{"---\ntitle: \"Random call\"\nparticipants: a@b.com\n---", "acme"},
		{"no frontmatter at all", "acme"},
	}
	for _, c := range cases {
		if got := topo.WorkspaceForBrainPage("meetings/x.md", c.header+"\n\nbody mentions launchpad"); got != c.want {
			t.Errorf("header %q → %q, want %q", c.header, got, c.want)
		}
	}
	if got := topo.WorkspaceForBrainPage("sops/x.md", "---\ntitle: launchpad\n---"); got != topo.BrainStore.Default {
		t.Errorf("meeting rules must only apply to meetings/, got %q", got)
	}
	topo.MeetingRules[0].Workspace = "nope"
	if err := topo.Validate(); err == nil || !strings.Contains(err.Error(), "meeting rule") {
		t.Errorf("Validate() = %v, want a meeting rule error", err)
	}
}

func TestRepoMeetingRulesSendLaunchpadCohortCallsHome(t *testing.T) {
	topo, err := LoadRepo()
	if err != nil {
		t.Fatal(err)
	}
	page := "---\ntitle: \"Launchpad Cohort - Closers\"\nparticipants: sam@vantage.example\n---\n"
	if got := topo.WorkspaceForBrainPage("meetings/2026-05-01-aa.md", page); got != "launchpad-cohort" {
		t.Errorf("LC closers call → %q", got)
	}
	page = "---\ntitle: \"Riley X Vantage\"\nparticipants: casey@vantage.example\n---\n"
	if got := topo.WorkspaceForBrainPage("meetings/2026-03-18-riley.md", page); got != "vantage" {
		t.Errorf("Vantage call → %q", got)
	}
}

func TestMeetingHeaderIncludesMetadataBelowFrontmatter(t *testing.T) {
	topo, err := LoadRepo()
	if err != nil {
		t.Fatal(err)
	}
	page := "---\ntitle: \"Jordan and Alex\"\ntype: meeting\n---\n\n# Jordan and Alex\n\nsource: attio\nparticipants: alex@vantage.example, jordan@client.example\n\n## Transcript\n\nparticipants: launchpadcohort@x.com\n"
	if got := topo.WorkspaceForBrainPage("meetings/2026-05-02-jordan.md", page); got != "vantage" {
		t.Errorf("participants below frontmatter → %q, want vantage", got)
	}
}
