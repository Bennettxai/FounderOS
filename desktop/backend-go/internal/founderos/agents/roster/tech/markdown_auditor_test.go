package tech

import (
	"context"
	"strings"
	"testing"
	"time"
)

var auditNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

const fresh = "2026-09-29T10:00:00Z"

func sig(slug, title, content string) fakeSignal {
	return fakeSignal{Title: title, URI: "optimal://inbox/x/" + slug + ".md", Content: content, Modified: fresh}
}

func cleanHub() *fakeEngine {
	f := newFakeEngine()
	f.workspaces = []string{"founderos", "vantage"}
	f.signals["founderos"] = []fakeSignal{
		sig("sops/pricing", "Pricing", "# Pricing\nsee [[sops/scripts]]"),
		sig("sops/scripts", "Scripts", "# Scripts\nback to [[pricing]]"),
	}
	f.signals["vantage"] = []fakeSignal{
		sig("offers/v2", "Offer v2", "# Offer v2\nsee [[Offer v1]]"),
		sig("offers/v1", "Offer v1", "# Offer v1\nsee [[offers/v2]]"),
	}
	return f
}

func cleanMacbook() *fakeEngine {
	f := newFakeEngine()
	f.workspaces = []string{"personal"}
	f.signals["personal"] = []fakeSignal{
		sig("notes/a", "A", "# A\n[[b]]"),
		sig("notes/b", "B", "# B\n[[a]]"),
	}
	return f
}

func runAuditor(t *testing.T, staged map[string]*fakeEngine) (bool, string, *KnowledgeAudit) {
	t.Helper()
	a := &MarkdownAuditor{Engines: source(t, staged), Now: func() time.Time { return auditNow }}
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	audit, _ := res.Data.(*KnowledgeAudit)
	return res.OK, res.Summary, audit
}

func findingOf(a *KnowledgeAudit, kind string) *AuditFinding {
	if a == nil {
		return nil
	}
	for i := range a.Findings {
		if a.Findings[i].Kind == kind {
			return &a.Findings[i]
		}
	}
	return nil
}

func TestMarkdownAuditorMeta(t *testing.T) {
	m := (&MarkdownAuditor{}).Meta()
	if m.ID != "markdown-auditor" || m.Name != "Markdown Auditor" || m.DepartmentID != "dept-tech" {
		t.Fatalf("meta = %+v", m)
	}
}

func TestMarkdownAuditorClean(t *testing.T) {
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": cleanHub(), "macbook": cleanMacbook()})
	if !ok {
		t.Fatalf("clean knowledge should pass: %s", summary)
	}
	for _, want := range []string{"6 sources in 3 workspaces", "founderos 2", "6/6 links resolve", "0 orphan(s)", "no findings", "unaudited: hermes (mini not staged)"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary missing %q: %s", want, summary)
		}
	}
	if audit == nil || audit.Sources != 6 || len(audit.Workspaces) != 3 {
		t.Fatalf("audit data = %+v", audit)
	}
}

func TestMarkdownAuditorFindsBrokenOrphanDuplicateUntitled(t *testing.T) {
	hub := cleanHub()
	hub.signals["founderos"] = append(hub.signals["founderos"],
		sig("sops/ghostly", "Ghostly", "# Ghostly\nsee [[ghost-page]]"),
		sig("old/pricing", "Pricing", "no heading"),
		sig("inbox/dump", "dump", "raw text"),
		sig("inbox/blank", "  ", "raw text"),
	)
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if !ok {
		t.Fatalf("warnings alone do not fail the audit: %s", summary)
	}
	if f := findingOf(audit, "broken-link"); f == nil || f.Count != 1 || !strings.Contains(f.Detail, "founderos: ghostly → ghost-page") {
		t.Errorf("broken link finding = %+v", f)
	}
	if f := findingOf(audit, "orphan"); f == nil || !strings.Contains(f.Detail, "founderos: dump") {
		t.Errorf("orphan finding = %+v", f)
	}
	if f := findingOf(audit, "duplicate-title"); f == nil || !strings.Contains(f.Detail, "founderos: Pricing (×2)") {
		t.Errorf("duplicate finding = %+v", f)
	}
	if f := findingOf(audit, "no-title"); f == nil || f.Count != 2 || !strings.Contains(f.Detail, "founderos: dump") || !strings.Contains(f.Detail, "founderos: blank") {
		t.Errorf("no-title finding = %+v", f)
	}
}

func TestMarkdownAuditorResolvesDatePrefixedSlugsAndTitles(t *testing.T) {
	hub := cleanHub()
	hub.signals["founderos"] = []fakeSignal{
		{Title: "Claude Code", URI: "optimal://inbox/founderos/inbox/signals/2026-09-30-claude-code.md", Content: "# Claude Code\n[[Tools]]", Modified: fresh},
		{Title: "Tools", URI: "optimal://inbox/founderos/inbox/signals/2026-09-30-tools.md", Content: "# Tools\n[[claude-code]] and `[[not a link]]`\n```\n[[also not]]\n```", Modified: fresh},
	}
	_, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if f := findingOf(audit, "broken-link"); f != nil {
		t.Fatalf("engine slugs carry a date prefix and code is not prose: %+v (%s)", f, summary)
	}
}

func TestMarkdownAuditorStaleWorkspace(t *testing.T) {
	hub := cleanHub()
	for i := range hub.signals["vantage"] {
		hub.signals["vantage"][i].Modified = "2026-07-01T00:00:00Z"
	}
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	f := findingOf(audit, "stale-workspace")
	if f == nil || !strings.Contains(f.Detail, "vantage") || strings.Contains(f.Detail, "founderos") {
		t.Fatalf("stale finding = %+v (%s)", f, summary)
	}
	if !ok {
		t.Errorf("a quiet workspace is a warning, not a failure: %s", summary)
	}
}

func TestMarkdownAuditorMissingWorkspaceIsAnError(t *testing.T) {
	hub := cleanHub()
	hub.workspaces = []string{"founderos"}
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if ok {
		t.Fatalf("a routed workspace missing on its engine must fail: %s", summary)
	}
	if f := findingOf(audit, "missing-workspace"); f == nil || f.Severity != "err" || !strings.Contains(f.Detail, "vantage") {
		t.Errorf("finding = %+v", f)
	}
	if strings.Contains(summary, "vantage 0") {
		t.Errorf("a missing workspace must not be counted as empty: %s", summary)
	}
}

func TestMarkdownAuditorIndexDrift(t *testing.T) {
	hub := cleanHub()
	hub.checks[1] = fakeCheck{"fts_parity", false, map[string]int{"contexts": 747, "indexed": 700}}
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if ok {
		t.Fatalf("index drift must fail the audit: %s", summary)
	}
	if f := findingOf(audit, "index-drift"); f == nil || !strings.Contains(f.Detail, "hub") || !strings.Contains(f.Detail, "700") {
		t.Errorf("finding = %+v", f)
	}
}

func TestMarkdownAuditorEngineDown(t *testing.T) {
	hub := cleanHub()
	hub.down = true
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if ok {
		t.Fatalf("unreachable engine must fail: %s", summary)
	}
	if f := findingOf(audit, "unreachable"); f == nil || !strings.Contains(f.Detail, "hub") {
		t.Errorf("finding = %+v", f)
	}
	if strings.Contains(summary, "founderos 0") {
		t.Errorf("unreachable must not read as empty: %s", summary)
	}
}

func TestMarkdownAuditorEmptyKnowledge(t *testing.T) {
	hub := newFakeEngine()
	hub.workspaces = []string{"founderos", "vantage"}
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub})
	if ok || findingOf(audit, "empty-store") == nil {
		t.Fatalf("no sources anywhere must fail: %s", summary)
	}
}

func TestMarkdownAuditorNoEngines(t *testing.T) {
	a := &MarkdownAuditor{Engines: noEngines(t)}
	res, _ := a.Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "no Optimal Engine is staged") {
		t.Fatalf("res = %+v", res)
	}
}

func TestMarkdownAuditorSameURIIsStillTwoSources(t *testing.T) {
	// The engine gives every "Impromptu Google Meet Meeting" the same URI;
	// they are separate sources with a shared title, not one page.
	hub := cleanHub()
	meet := fakeSignal{Title: "Impromptu Google Meet Meeting", URI: "optimal://inbox/founderos/inbox/signals/2026-09-30-impromptu-google-meet-meeting.md", Content: "# Impromptu\n[[pricing]]", Modified: fresh}
	hub.signals["founderos"] = append(hub.signals["founderos"], meet, meet, meet)
	_, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if audit.Sources != 9 {
		t.Fatalf("sources = %d (%s)", audit.Sources, summary)
	}
	if f := findingOf(audit, "duplicate-title"); f == nil || !strings.Contains(f.Detail, "Impromptu Google Meet Meeting (×3)") {
		t.Errorf("duplicate finding = %+v", f)
	}
	if f := findingOf(audit, "orphan"); f != nil && strings.Contains(f.Detail, "impromptu") {
		t.Errorf("each meeting links out, none is an orphan: %+v", f)
	}
}

func TestMarkdownAuditorOneExportFails(t *testing.T) {
	hub := cleanHub()
	hub.failExport = map[string]bool{"vantage": true}
	ok, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": cleanMacbook()})
	if ok {
		t.Fatalf("a failed export is not a pass: %s", summary)
	}
	if f := findingOf(audit, "unreachable"); f == nil || !strings.Contains(f.Detail, "vantage on hub could not be exported") {
		t.Errorf("finding = %+v", f)
	}
	if !strings.Contains(summary, "4 sources in 2 workspaces (founderos 2, personal 2)") {
		t.Errorf("the rest of the audit still reports: %s", summary)
	}
	if findingOf(audit, "empty-store") != nil {
		t.Errorf("a failed export must not read as an empty store")
	}
}

func TestMarkdownAuditorLinksResolveAcrossWorkspaces(t *testing.T) {
	// The brain-store was one corpus; placement split it across workspaces
	// and engines, so [[Claude Code]] in a personal chat still means the
	// founderos page.
	hub := cleanHub()
	hub.signals["founderos"] = append(hub.signals["founderos"], sig("projects/claude-code", "Claude Code", "# Claude Code"))
	mac := cleanMacbook()
	mac.signals["personal"] = append(mac.signals["personal"], sig("conversations/chat", "Chat", "# Chat\nusing [[Claude Code]] and [[nowhere]]"))
	_, summary, audit := runAuditor(t, map[string]*fakeEngine{"hub": hub, "macbook": mac})
	f := findingOf(audit, "broken-link")
	if f == nil || f.Count != 1 || !strings.Contains(f.Detail, "personal: chat → nowhere") {
		t.Fatalf("only [[nowhere]] is broken: %+v (%s)", f, summary)
	}
	if o := findingOf(audit, "orphan"); o != nil && (strings.Contains(o.Detail, "claude-code") || strings.Contains(o.Detail, "chat")) {
		t.Errorf("a cross-workspace link connects both ends: %+v", o)
	}
}
