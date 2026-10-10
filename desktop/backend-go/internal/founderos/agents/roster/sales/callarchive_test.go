package sales

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

func fathomItem(id int, title, at string) map[string]any {
	return map[string]any{
		"title": title, "meeting_title": title,
		"url":          "https://fathom.video/calls/" + itoa(id),
		"recording_id": id,
		"created_at":   at, "recording_start_time": at, "recording_end_time": at,
		"recorded_by":       map[string]any{"name": "Alex", "email": "alex@launchpadcohort.example"},
		"calendar_invitees": []map[string]any{{"email": "jamie@example.com", "name": "Jamie O."}},
		"transcript": []map[string]any{
			{"speaker": map[string]any{"display_name": "Jamie O."}, "text": "Doing well, thanks.", "timestamp": "00:00:00"},
			{"speaker": map[string]any{"display_name": "Alex"}, "text": "Tell me about the rollout.", "timestamp": "00:00:04"},
		},
		"default_summary": map[string]any{"template_name": "Enhanced", "markdown_formatted": "## Meeting Purpose\n\nExplore AI for retention."},
		"action_items": []map[string]any{
			{"description": "Draft proposal w/ CTO", "completed": false, "recording_timestamp": "00:31:12", "assignee": map[string]any{"name": "Alex"}},
		},
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func meeting(t *testing.T, item map[string]any) fathomcalls.MeetingFull {
	t.Helper()
	raw, _ := json.Marshal(item)
	var m fathomcalls.MeetingFull
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// fakeFathom serves pages of meetings; err ends the walk after the pages.
type fakeFathom struct {
	pages [][]fathomcalls.MeetingFull
	err   error
	// gate, when set, blocks the walk until closed (single-flight test).
	gate chan struct{}
}

func (f *fakeFathom) MeetingsFull(_ context.Context, visit func([]fathomcalls.MeetingFull) error) error {
	if f.gate != nil {
		<-f.gate
	}
	for _, p := range f.pages {
		if err := visit(p); err != nil {
			return err
		}
	}
	return f.err
}

func twoPages(t *testing.T) *fakeFathom {
	demo := fathomItem(3, "Fathom Demo", "2021-09-16T20:42:47Z")
	demo["calendar_invitees"] = []map[string]any{{"email": "susannah.durant@fathom.video", "name": "Susannah"}}
	empty := fathomItem(2, "Empty call", "2026-07-01T10:00:00Z")
	empty["transcript"] = []any{}
	return &fakeFathom{pages: [][]fathomcalls.MeetingFull{
		{meeting(t, fathomItem(1, "Jamie Ortíz x Alex", "2026-08-25T16:03:36Z"))},
		{meeting(t, empty), meeting(t, demo)},
	}}
}

type archiveFixture struct {
	fathom *fakeFathom
	mem    *fakeMemory
	ledger *fakeArchiveLedger
	arch   *CallArchiver
}

func newArchiveFixture(t *testing.T) *archiveFixture {
	t.Helper()
	// these passes write to the engine and the ledger; the guard allows them only with writes on
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &archiveFixture{fathom: twoPages(t), mem: &fakeMemory{}, ledger: newArchiveLedger()}
	f.arch = &CallArchiver{
		Fathom: f.fathom, Memory: f.mem, Ledger: f.ledger,
		Route: repoTopology(t).WorkspaceForBrainPage,
		Now:   func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) },
	}
	return f
}

func TestRenderFathomCallPage(t *testing.T) {
	md := RenderFathomCallPage(meeting(t, fathomItem(1, "Jamie Ortíz x Alex", "2026-08-25T16:03:36Z")))
	if !strings.HasPrefix(md, "---\ntitle: \"Jamie Ortíz x Alex\"\ntype: meeting\ndate: 2026-08-25\nsource: fathom\n---\n\n# Jamie Ortíz x Alex") {
		t.Fatalf("head:\n%s", md)
	}
	for _, want := range []string{
		"archive_version: 3", "source: fathom", "fathom_recording_id: 1", "url: https://fathom.video/calls/1",
		"recorded: 2026-08-25T16:03:36Z",
		"recorded_by: Alex <alex@launchpadcohort.example>", "invitees: Jamie O. <jamie@example.com>",
		"## Summary", "Explore AI for retention.", "## Action items", "- [ ] Draft proposal w/ CTO (Alex, 00:31:12)",
		"## Transcript", "[00:00:04] Alex: Tell me about the rollout.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
	a, b, c := strings.Index(md, "## Summary"), strings.Index(md, "## Action items"), strings.Index(md, "## Transcript")
	if !(a < b && b < c) {
		t.Error("summary, action items, transcript must be in that order")
	}
	quoted := RenderFathomCallPage(meeting(t, fathomItem(9, `Bob "Sales" Smith`, "2026-01-01T00:00:00Z")))
	if !strings.Contains(quoted, `title: "Bob \"Sales\" Smith"`) {
		t.Fatal("a double quote must survive the YAML frontmatter")
	}

	m := fathomItem(1, "T", "2026-08-25T16:03:36Z")
	m["transcript"] = []map[string]any{
		{"speaker": map[string]any{"display_name": "Richard White"}, "text": "All right.", "timestamp": "00:00:00"},
		{"speaker": map[string]any{"display_name": "Richard White"}, "text": "Hello.", "timestamp": "00:00:01"},
		{"speaker": map[string]any{"display_name": "Alex"}, "text": "Hi.", "timestamp": "00:00:03"},
	}
	m["recording_end_time"] = "2031-01-01T00:00:00Z"
	md = RenderFathomCallPage(meeting(t, m))
	if !strings.Contains(md, "[00:00:00] Richard White: All right. Hello.") || strings.Contains(md, "duration:") {
		t.Fatalf("merge / nonsense duration:\n%s", md)
	}
	m["recording_end_time"] = "2026-08-25T16:45:36Z"
	if md = RenderFathomCallPage(meeting(t, m)); !strings.Contains(md, "duration: 42m") {
		t.Fatalf("duration:\n%s", md)
	}
}

func TestArchiveCapturesEachTranscribedCallOnce(t *testing.T) {
	f := newArchiveFixture(t)
	ctx := context.Background()
	r, err := f.arch.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r.Source != "fathom" || r.MeetingsScanned != 3 || r.Found != 3 {
		t.Fatalf("%+v", r)
	}
	if !reflect.DeepEqual(r.Exported, []string{"meetings/2026-08-25-jamie-ortiz-x-alex--fathom-1.md"}) ||
		!reflect.DeepEqual(r.NoTranscript, []string{"2"}) || !reflect.DeepEqual(r.SkippedSample, []string{"3"}) {
		t.Fatalf("%+v", r)
	}
	if len(f.mem.captured) != 1 {
		t.Fatalf("captures %+v", f.mem.captured)
	}
	c := f.mem.captured[0]
	if c.Workspace != "founderos" || c.Genre != "transcript" || c.Title != "Jamie Ortíz x Alex" || !strings.Contains(c.Text, "fathom_recording_id: 1") {
		t.Fatalf("capture %+v", c)
	}
	e := f.ledger.rows["1"]
	if e != (ArchiveEntry{RecordingID: "1", Version: ArchiveVersion, Workspace: "founderos", SignalID: "sig_1",
		Page: "meetings/2026-08-25-jamie-ortiz-x-alex--fathom-1.md", Title: "Jamie Ortíz x Alex",
		ArchivedAt: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}) {
		t.Fatalf("ledger %+v", e)
	}

	again, err := f.arch.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again.SkippedExisting, []string{"1"}) || len(again.Exported) != 0 || len(f.mem.captured) != 1 {
		t.Fatalf("second pass must capture nothing: %+v", again)
	}
}

func TestArchiveRoutesMeetingsByTopologyRules(t *testing.T) {
	f := newArchiveFixture(t)
	f.fathom.pages = [][]fathomcalls.MeetingFull{{
		meeting(t, fathomItem(11, "Launchpad Cohort weekly", "2026-09-01T10:00:00Z")),
		meeting(t, fathomItem(12, "Vantage x Acme discovery", "2026-09-02T10:00:00Z")),
	}}
	if _, err := f.arch.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range f.mem.captured {
		got[c.Title] = c.Workspace
	}
	want := map[string]string{"Launchpad Cohort weekly": "launchpad-cohort", "Vantage x Acme discovery": "vantage"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routing %v", got)
	}
	if f.ledger.rows["12"].Workspace != "vantage" {
		t.Fatalf("ledger %+v", f.ledger.rows["12"])
	}
}

func TestArchiveRecapturesAnOlderVersionEntry(t *testing.T) {
	f := newArchiveFixture(t)
	f.ledger.rows["1"] = ArchiveEntry{RecordingID: "1", Version: ArchiveVersion - 1}
	r, _ := f.arch.Run(context.Background())
	if len(r.Exported) != 1 || f.ledger.rows["1"].Version != ArchiveVersion {
		t.Fatalf("%+v", r)
	}
}

func TestArchiveFailures(t *testing.T) {
	t.Run("capture fails: reported, not recorded, retried", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.mem.fail = func(memory.Capture) error { return errors.New("engine hub unreachable") }
		r, err := f.arch.Run(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Failed) != 1 || r.Failed[0].ID != "1" || !strings.Contains(r.Failed[0].Error, "unreachable") || len(f.ledger.rows) != 0 {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("ledger record fails after capture: says where the page went", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.ledger.recErr = errBoom
		r, _ := f.arch.Run(context.Background())
		if len(r.Failed) != 1 || !strings.Contains(r.Failed[0].Error, "sig_1") || len(r.Exported) != 0 {
			t.Fatalf("%+v", r)
		}
	})
	t.Run("no Fathom key is an error, not an empty archive", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.fathom.pages, f.fathom.err = nil, fathomcalls.ErrNotConfigured
		r, err := f.arch.Run(context.Background())
		if !errors.Is(err, fathomcalls.ErrNotConfigured) || len(r.Exported) != 0 {
			t.Fatalf("r=%+v err=%v", r, err)
		}
	})
	t.Run("a walk that dies mid-way keeps what landed and returns the error", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.fathom.err = errors.New("HTTP 500 /external/v1/meetings")
		r, err := f.arch.Run(context.Background())
		if err == nil || len(r.Exported) != 1 {
			t.Fatalf("r=%+v err=%v", r, err)
		}
	})
	t.Run("unreadable ledger stops the pass before any capture", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.ledger.readErr = errBoom
		if _, err := f.arch.Run(context.Background()); err == nil || len(f.mem.captured) != 0 {
			t.Fatalf("err=%v captures=%d", err, len(f.mem.captured))
		}
	})
	t.Run("no memory router", func(t *testing.T) {
		f := newArchiveFixture(t)
		f.arch.Memory = nil
		if _, err := f.arch.Run(context.Background()); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestArchiveIsSingleFlight(t *testing.T) {
	f := newArchiveFixture(t)
	f.fathom.gate = make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = f.arch.Run(context.Background())
	}()
	// Wait until the first pass holds the lock.
	for !f.arch.Running() {
		time.Sleep(time.Millisecond)
	}
	if _, err := f.arch.Run(context.Background()); !errors.Is(err, ErrArchiveRunning) {
		t.Fatalf("second concurrent pass: %v", err)
	}
	close(f.fathom.gate)
	wg.Wait()
	if len(f.ledger.rows) != 1 {
		t.Fatalf("first pass should have finished: %v", f.ledger.ids())
	}
}

func TestSeedArchiveLedgerFromStore(t *testing.T) {
	dir := t.TempDir()
	meetings := filepath.Join(dir, "meetings")
	if err := os.MkdirAll(meetings, 0o755); err != nil {
		t.Fatal(err)
	}
	current := RenderFathomCallPage(meeting(t, fathomItem(12, "Vantage x Acme discovery", "2026-09-02T10:00:00Z")))
	files := map[string]string{
		"2026-09-02-vantage-x-acme-discovery--fathom-12.md": current,
		"2026-08-01-old-format--fathom-13.md":               "# old format page\n\nsource: fathom\n",
		"2026-03-18-anthony-x-vantage--attio-r1.md":         "# Anthony X Vantage\n",
		"notes.md": "# not a call page\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(meetings, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ledger := newArchiveLedger()
	ledger.rows["99"] = ArchiveEntry{RecordingID: "99", Version: ArchiveVersion}
	n, err := SeedArchiveLedgerFromStore(context.Background(), ledger, dir, repoTopology(t).WorkspaceForBrainPage, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	e := ledger.rows["12"]
	if e.Version != ArchiveVersion || e.Workspace != "vantage" || e.SignalID != "" || e.Page != "meetings/2026-09-02-vantage-x-acme-discovery--fathom-12.md" {
		t.Fatalf("%+v", e)
	}
	if _, ok := ledger.rows["13"]; ok {
		t.Fatal("an old-format page is not current, so the archive must re-capture it")
	}
	// A second seed adds nothing.
	if n, _ := SeedArchiveLedgerFromStore(context.Background(), ledger, dir, repoTopology(t).WorkspaceForBrainPage, time.Now()); n != 0 {
		t.Fatalf("reseed added %d", n)
	}
	if _, err := SeedArchiveLedgerFromStore(context.Background(), ledger, filepath.Join(dir, "nope"), repoTopology(t).WorkspaceForBrainPage, time.Now()); err == nil {
		t.Fatal("a missing store must be an error, not zero pages")
	}
}
