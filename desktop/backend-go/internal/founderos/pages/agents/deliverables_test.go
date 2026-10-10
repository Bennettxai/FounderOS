package agentspage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func writeFile(t *testing.T, path, body string, mod time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(path, mod, mod)
}

func TestListDeliverablesReadsTitlesAndCollapsesSiblings(t *testing.T) {
	base := t.TempDir()
	now := time.Now()
	writeFile(t, filepath.Join(base, "ws-aaaaaaaa11", "deliverables", "ben12-launch-plan.md"), "# Launch plan\n\nShip the reel on Friday. Then post.", now.Add(-time.Hour))
	writeFile(t, filepath.Join(base, "ws-aaaaaaaa11", "deliverables", "ben12-launch-plan.json"), `{"title":"Launch plan","body":"x"}`, now.Add(-time.Hour))
	writeFile(t, filepath.Join(base, "ws-bbbbbbbb22", "deliverables", "STAGED-reply-2026-09-30.json"), `{"body":"Hi Sam,\nThanks for the call. Next steps below."}`, now)
	writeFile(t, filepath.Join(base, "ws-bbbbbbbb22", "deliverables", "logo.png"), "\x89PNG", now.Add(-2*time.Hour))
	writeFile(t, filepath.Join(base, "ws-bbbbbbbb22", "notes.md"), "outside deliverables", now)

	list := ListDeliverables(base)
	if len(list) != 3 {
		t.Fatalf("list: %+v", list)
	}
	if list[0].Name != "STAGED-reply-2026-09-30.json" || list[0].Title != "Thanks for the call." || list[0].Summary != "Next steps below." {
		t.Fatalf("json prose brief: %+v", list[0])
	}
	if list[1].Name != "ben12-launch-plan.md" || list[1].Title != "Launch plan" || strings.Join(list[1].AlsoAs, ",") != "json" {
		t.Fatalf("collapsed md: %+v", list[1])
	}
	if list[2].Title != "logo" || list[2].Summary != "" {
		t.Fatalf("binary title from filename: %+v", list[2])
	}
	if ListDeliverables(filepath.Join(base, "missing")) == nil {
		t.Fatal("missing dir must be an empty list, not nil")
	}
}

func TestResolveDeliverableRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	for _, bad := range []string{"../x", "a/../b", "a/b/c", "a/", "/etc/passwd", "a/.."} {
		if ResolveDeliverable(bad, base) != "" {
			t.Errorf("accepted %q", bad)
		}
	}
	if got := ResolveDeliverable("ws/file.md", base); got != filepath.Join(base, "ws", "deliverables", "file.md") {
		t.Fatalf("resolve %q", got)
	}
}

func TestTitleFromFilename(t *testing.T) {
	cases := map[string]string{
		"STAGED-ben42-client-reply-DELIVER-BEFORE-1700Z-2026-09-30.md": "client reply",
		"fish-audio-reel-2026-08-27-FINAL.md":                          "fish audio reel",
		"x.md":                                                         "x",
	}
	for in, want := range cases {
		if got := TitleFromFilename(in); got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}

func TestGroupDeliverablesPinsProposalsFirst(t *testing.T) {
	amt := 4500.0
	props := []osdata.Proposal{{ID: "p1", Client: "Acme", Brand: "launchpad-cohort", URL: "https://x", Status: "sent", AmountUSD: &amt, CreatedAt: "2026-09-01T00:00:00.000Z"},
		{ID: "p2", Client: "Zed", Brand: "vantage", URL: "https://y", Status: "draft", CreatedAt: "2026-09-02T00:00:00.000Z"}}
	files := []Deliverable{{ID: "wsabcdefgh1/a.md", Name: "a.md", Workspace: "wsabcdefgh1", Title: "A", ModifiedAt: "2026-09-30T00:00:00.000Z"}}
	g := GroupDeliverables(files, props)
	if len(g) != 3 || g[0].Name != "Vantage proposals" || g[1].Name != "Launchpad Cohort proposals" || g[2].Name != "Agent files" {
		t.Fatalf("groups %+v", g)
	}
	if g[1].Items[0].Meta != "sent · $4,500" || g[1].Items[0].ID != "proposal:p1" || g[1].Items[0].Kind != "link" {
		t.Fatalf("proposal item %+v", g[1].Items[0])
	}
	if g[2].Items[0].Meta != "wsabcdef" || g[2].Items[0].Revision == "" {
		t.Fatalf("file item %+v", g[2].Items[0])
	}
	if len(GroupDeliverables(nil, nil)) != 0 {
		t.Fatal("empty groups omitted")
	}
}

func TestClassifyAndNeedsYou(t *testing.T) {
	now := time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)
	item := func(name, mod string) DeliverableItem {
		return DeliverableItem{ID: "w/" + name, Name: name, Kind: "file", ModifiedAt: mod, Title: name}
	}
	staged := Classify(item("STAGED-reply.md", "2026-09-30T10:00:00.000Z"), now)
	if staged.Ask != "staged" || !staged.NeedsYou || !staged.Person || staged.GlyphTone != "ok" {
		t.Fatalf("staged %+v", staged)
	}
	overdue := Classify(item("brief-DELIVER-BEFORE-1200Z.md", "2026-09-30T08:00:00.000Z"), now)
	if overdue.Ask != "request" || overdue.Label != "due" || !overdue.Overdue || overdue.Deadline == nil || *overdue.Deadline != "2026-09-30T12:00:00.000Z" || overdue.GlyphTone != "err" {
		t.Fatalf("overdue %+v", overdue)
	}
	if d := Classify(item("STAGED-reply-FINAL.md", "2026-09-30T10:00:00.000Z"), now); d.Ask != "done" || d.NeedsYou {
		t.Fatalf("done %+v", d)
	}
	if m := Classify(item("invoice-decision.md", "2026-09-30T10:00:00.000Z"), now); m.Glyph != "$" || m.Ask != "decision" {
		t.Fatalf("money %+v", m)
	}
	q := NeedsYou([]DeliverableItem{item("plain-output.md", "2026-09-30T10:00:00.000Z"), item("a-decision-x.md", "2026-09-30T11:00:00.000Z"),
		item("STAGED-reply.md", "2026-09-30T09:00:00.000Z"), item("x-DELIVER-BEFORE-1200Z.md", "2026-09-30T08:00:00.000Z")}, now)
	if len(q) != 3 || q[0].Overdue != true || q[1].Ask != "staged" || q[2].Ask != "decision" {
		t.Fatalf("needs you order: %+v", q)
	}
}

func TestPartitionByDecisionReopensOnRevision(t *testing.T) {
	a := DeliverableItem{ID: "w/a.md", Kind: "file", ModifiedAt: "2026-09-30T10:00:00.000Z", SizeBytes: iptr(10)}
	a.Revision = RevisionOf(a)
	b := DeliverableItem{ID: "w/b.md", Kind: "file", ModifiedAt: "2026-09-30T11:00:00.000Z", SizeBytes: iptr(5)}
	b.Revision = RevisionOf(b)
	decisions := []osdata.Decision{
		{ID: "w/a.md", Decision: "approved", DecidedAt: "2026-09-30T12:00:00.000Z", DecidedRevision: a.Revision},
		{ID: "w/b.md", Decision: "dismissed", DecidedAt: "2026-09-30T12:00:00.000Z", DecidedRevision: "file|old|5"},
		{ID: "w/gone.md", Decision: "approved", DecidedAt: "2026-09-30T12:00:00.000Z"},
	}
	open, decided := PartitionByDecision([]DeliverableItem{a, b}, decisions)
	if len(open) != 1 || open[0].ID != "w/b.md" || len(decided) != 1 || decided[0].Decision.Decision != "approved" {
		t.Fatalf("open %+v decided %+v", open, decided)
	}
}

func TestPreviewKindAndClamp(t *testing.T) {
	for name, want := range map[string]string{"a.md": "text", "b.SVG": "image", "c.pdf": "pdf", ".env": "binary", "d.zip": "binary", "e.yaml": "text"} {
		if got := PreviewKind(name); got != want {
			t.Errorf("%s = %s want %s", name, got, want)
		}
	}
	text, cut := ClampPreview(strings.Repeat("a", 10), 4)
	if text != "aaaa" || !cut {
		t.Fatal("clamp")
	}
}

func iptr(n int64) *int64 { return &n }

// FounderOS v1 stamps a file's revision with Node's Stats.mtime, which rounds
// mtimeMs to the millisecond; Go's .000 format truncates. A 1ms mismatch made
// a decision's saved revision stop matching and the handled item reopen.
func TestDeliverableModifiedAtRoundsToTheMillisecondLikeNode(t *testing.T) {
	base := t.TempDir()
	mod := time.Date(2026, 9, 30, 12, 0, 37, 516_700_000, time.UTC) // 37.5167s → Node: 37.517
	writeFile(t, filepath.Join(base, "ws-aaaaaaaa11", "deliverables", "a.md"), "# A", mod)
	st, err := os.Stat(filepath.Join(base, "ws-aaaaaaaa11", "deliverables", "a.md"))
	if err != nil || st.ModTime().Nanosecond()/1000 != 516700 {
		t.Skipf("filesystem keeps no sub-ms mtime here (%v)", st.ModTime())
	}
	list := ListDeliverables(base)
	if len(list) != 1 || list[0].ModifiedAt != "2026-09-30T12:00:37.517Z" {
		t.Fatalf("ModifiedAt = %+v, want 2026-09-30T12:00:37.517Z", list)
	}
}
