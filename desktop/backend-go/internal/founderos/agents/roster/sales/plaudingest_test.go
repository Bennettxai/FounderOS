package sales

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
)

func plaudFileBody(t *testing.T, id, name string, transcribed bool) []byte {
	t.Helper()
	segs, _ := json.Marshal([]map[string]any{
		{"start_time": 0, "end_time": 31200, "content": "We walked the north lot first.", "speaker": "Alex"},
		{"start_time": 32110, "end_time": 62776, "content": "Gate two needs a new reader before the 16th.", "speaker": "Sam"},
	})
	body := map[string]any{
		"id": id, "name": name,
		"created_at": "2026-08-26T15:00:00", "start_at": "2026-08-26T15:00:00.100000",
		"duration": 1_500_000,
	}
	if transcribed {
		body["source_list"] = []map[string]any{{"data_type": "transaction", "data_content": string(segs)}}
		body["note_list"] = []map[string]any{{
			"data_type":    "auto_sum_note",
			"data_content": "## Overview\nSite walk with Sam.\n\n## Action Items\n* Send the gate-two reader quote by Friday\n- Confirm the Sept 16 on-site window\n\n## Notes\nNothing else.",
		}}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type ingestFixture struct {
	plaud  *fakePlaud
	mem    *fakeMemory
	ledger *fakePlaudLedger
	ing    *PlaudIngester
}

func newIngestFixture(t *testing.T) *ingestFixture {
	t.Helper()
	// these passes write to the engine and the ledger; the guard allows them only with writes on
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &ingestFixture{
		plaud: &fakePlaud{
			list: []plaud.Recording{
				{ID: "f1", Title: "Northgate Storage site walk", At: "2026-08-26T15:00:00Z"},
				{ID: "f2", Title: "Not synced yet", At: "2026-08-26T16:00:00Z"},
			},
			files: map[string]*plaud.File{
				"f1": plaud.ParseFile(plaudFileBody(t, "f1", "Northgate Storage site walk", true)),
				"f2": plaud.ParseFile(plaudFileBody(t, "f2", "Not synced yet", false)),
			},
		},
		mem:    &fakeMemory{},
		ledger: newPlaudLedger(),
	}
	topo := repoTopology(t)
	f.ing = &PlaudIngester{
		Plaud:  f.plaud,
		Memory: f.mem,
		Ledger: f.ledger,
		Route:  topo.WorkspaceForBrainPage,
		Now:    func() time.Time { return time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC) },
	}
	return f
}

func TestRenderRecordingMarkdown(t *testing.T) {
	file := plaud.ParseFile(plaudFileBody(t, "f1", "Northgate Storage site walk", true))
	md := RenderRecordingMarkdown(file)
	if !strings.HasPrefix(md, "# Northgate Storage site walk\n\nsource: plaud\nplaud_file_id: f1\nrecorded: 2026-08-26\nduration: 25m\n") {
		t.Fatalf("header:\n%s", md)
	}
	for _, want := range []string{"## Summary", "Site walk with Sam.", "## Transcript", "[00:32 - 01:02] Sam: Gate two needs a new reader before the 16th.", "[00:00 - 00:31] Alex: We walked the north lot first."} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in\n%s", want, md)
		}
	}
	if strings.Index(md, "## Summary") > strings.Index(md, "## Transcript") {
		t.Error("summary must come before the transcript")
	}
}

func TestRecordingSlugAndSlugify(t *testing.T) {
	if got := RecordingSlug("Northgate Storage site walk", "2026-08-26T15:00:00Z", "f1"); got != "2026-08-26-northgate-storage-site-walk" {
		t.Fatal(got)
	}
	if got := RecordingSlug("!!!", "", "abcdef123456"); got != "undated-untitled" {
		t.Fatal(got)
	}
	for in, want := range map[string]string{
		" Jamie Ortíz x Alex ":    "jamie-ortiz-x-alex",
		"  Q3 -- plan__review  ":  "q3-plan-review",
		"Steve Jobs & Bill Gates": "steve-jobs-bill-gates",
		"":                        "untitled",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIngestCapturesEachTranscribedRecordingOnce(t *testing.T) {
	f := newIngestFixture(t)
	ctx := context.Background()
	r, err := f.ing.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r.Scanned != 2 || len(r.Ingested) != 1 || r.Ingested[0].FileID != "f1" {
		t.Fatalf("result %+v", r)
	}
	if !reflect.DeepEqual(r.Skipped.NotTranscribed, []string{"f2"}) || len(r.Failed) != 0 {
		t.Fatalf("result %+v", r)
	}

	pages := f.mem.genre("transcript")
	if len(pages) != 1 {
		t.Fatalf("pages = %+v", f.mem.captured)
	}
	p := pages[0]
	// inbox/recordings/ is where the TS filed a page; the topology routes it
	// to its default workspace, which is also the table map's home for it.
	if p.Workspace != "founderos" || p.Title != "Northgate Storage site walk" || !strings.Contains(p.Text, "## Transcript") {
		t.Fatalf("page capture = %+v", p)
	}
	claims := f.mem.genre("claims")
	if len(claims) != 1 || claims[0].Workspace != "founderos" ||
		!strings.Contains(claims[0].Text, "Send the gate-two reader quote by Friday") ||
		!strings.HasPrefix(claims[0].Text, `Action items from "Northgate Storage site walk" (Plaud recording, 2026-08-26):`) ||
		claims[0].Title != "Action items: Northgate Storage site walk" {
		t.Fatalf("claims capture = %+v", claims)
	}
	if r.Claims != 2 {
		t.Fatalf("claims = %d", r.Claims)
	}

	row := f.ledger.rows["f1"]
	want := PlaudIngest{
		FileID: "f1", Title: "Northgate Storage site walk", RecordedAt: "2026-08-26T15:00:00Z",
		IngestedAt: time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC), Via: "oe",
		Slug: "2026-08-26-northgate-storage-site-walk", Claims: 2, Workspace: "founderos", SignalID: "sig_1",
	}
	if row != want {
		t.Fatalf("ledger row\n got %+v\nwant %+v", row, want)
	}

	again, err := f.ing.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Ingested) != 0 || !reflect.DeepEqual(again.Skipped.AlreadyIngested, []string{"f1"}) || len(f.mem.captured) != 2 {
		t.Fatalf("second pass must do nothing: %+v, captures %d", again, len(f.mem.captured))
	}
}

func TestIngestRoutesByTopologyMeetingTitle(t *testing.T) {
	f := newIngestFixture(t)
	f.plaud.list = []plaud.Recording{{ID: "m1", Title: "Vantage onboarding walk", At: "2026-08-27T10:00:00Z"}}
	f.plaud.files = map[string]*plaud.File{"m1": plaud.ParseFile(plaudFileBody(t, "m1", "Vantage onboarding walk", true))}
	var seen []string
	f.ing.Route = func(rel, text string) string {
		seen = append(seen, rel)
		return "vantage"
	}
	if _, err := f.ing.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, []string{"inbox/recordings/2026-08-26-vantage-onboarding-walk.md"}) {
		t.Fatalf("routed paths %v", seen)
	}
	for _, c := range f.mem.captured {
		if c.Workspace != "vantage" {
			t.Fatalf("capture went to %q", c.Workspace)
		}
	}
	if f.ledger.rows["m1"].Workspace != "vantage" {
		t.Fatalf("ledger row %+v", f.ledger.rows["m1"])
	}
}

func TestIngestSkipsPlaudSampleRecordings(t *testing.T) {
	f := newIngestFixture(t)
	f.plaud.list = []plaud.Recording{
		{ID: "s1", Title: "Welcome to Plaud.ai"},
		{ID: "s2", Title: "How to use Plaud"},
		{ID: "s3", Title: "Steve Jobs & Bill Gates: A Conversation That Shaped Technology"},
		{ID: "f1", Title: "Northgate Storage site walk", At: "2026-08-26T15:00:00Z"},
	}
	r, err := f.ing.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Skipped.Sample, []string{"s1", "s2", "s3"}) || len(r.Ingested) != 1 {
		t.Fatalf("%+v", r)
	}
	if len(f.mem.genre("transcript")) != 1 {
		t.Fatal("samples must never be captured")
	}
}

func TestIngestFailedCaptureIsReportedAndRetried(t *testing.T) {
	f := newIngestFixture(t)
	f.mem.fail = func(c memory.Capture) error { return errors.New("engine hub unreachable") }
	r, err := f.ing.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ingested) != 0 || len(r.Failed) != 1 || r.Failed[0].FileID != "f1" || !strings.Contains(r.Failed[0].Error, "engine hub unreachable") {
		t.Fatalf("%+v", r)
	}
	if len(f.ledger.rows) != 0 {
		t.Fatal("a failed capture must not be recorded, so the next pass retries it")
	}
	f.mem.fail = nil
	r, _ = f.ing.Run(context.Background())
	if len(r.Ingested) != 1 {
		t.Fatalf("retry pass: %+v", r)
	}
}

func TestIngestClaimFailureKeepsThePageButSaysSo(t *testing.T) {
	f := newIngestFixture(t)
	f.mem.fail = func(c memory.Capture) error {
		if c.Genre == "claims" {
			return errors.New("HTTP 422")
		}
		return nil
	}
	r, err := f.ing.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ingested) != 1 || r.Claims != 0 || len(r.ClaimFailures) != 1 || f.ledger.rows["f1"].Claims != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestIngestNoClaimsWithoutActionItems(t *testing.T) {
	f := newIngestFixture(t)
	note := "## Overview\n- just notes"
	f.plaud.files["f1"].Note = &note
	r, _ := f.ing.Run(context.Background())
	if r.Claims != 0 || len(f.mem.genre("claims")) != 0 || len(r.Ingested) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestIngestFetchErrorFailsOnlyThatRecording(t *testing.T) {
	f := newIngestFixture(t)
	f.plaud.fileErr = map[string]error{"f1": errors.New("HTTP 500")}
	r, err := f.ing.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Failed) != 1 || r.Failed[0] != (IngestFailure{FileID: "f1", Error: "HTTP 500"}) || !reflect.DeepEqual(r.Skipped.NotTranscribed, []string{"f2"}) {
		t.Fatalf("%+v", r)
	}
}

func TestIngestLedgerInsertFailureIsAFailure(t *testing.T) {
	f := newIngestFixture(t)
	f.ledger.insErr = errors.New("constraint violation")
	r, _ := f.ing.Run(context.Background())
	if len(r.Ingested) != 0 || len(r.Failed) != 1 || !strings.Contains(r.Failed[0].Error, "constraint violation") || !strings.Contains(r.Failed[0].Error, "sig_1") {
		t.Fatalf("%+v", r)
	}
}

func TestIngestPassLevelFailuresAreErrorsNotEmptyResults(t *testing.T) {
	for name, mutate := range map[string]func(*ingestFixture){
		"ledger not ready":  func(f *ingestFixture) { f.ledger.notReady = errors.New("via CHECK refuses 'oe'") },
		"ledger unreadable": func(f *ingestFixture) { f.ledger.readErr = errBoom },
		"list fails":        func(f *ingestFixture) { f.plaud.listErr = errors.New("Plaud access token expired") },
		"no memory":         func(f *ingestFixture) { f.ing.Memory = nil },
		"no ledger":         func(f *ingestFixture) { f.ing.Ledger = nil },
	} {
		t.Run(name, func(t *testing.T) {
			f := newIngestFixture(t)
			mutate(f)
			if _, err := f.ing.Run(context.Background()); err == nil {
				t.Fatal("want an error")
			}
			if len(f.mem.captured) != 0 {
				t.Fatal("nothing may be captured when the pass cannot run")
			}
		})
	}
}

// One Plaud pass at a time across the whole process: the cron, a manual Run
// and POST /pages/plaud/ingest each build their own ingester, and the engine
// gives every capture a fresh id, so two overlapping passes would file the
// same recording twice.
func TestIngestOnePassAtATimeAcrossInstances(t *testing.T) {
	f := newIngestFixture(t)
	fp := f.ing.Plaud.(*fakePlaud)
	fp.gate, fp.entered = make(chan struct{}), make(chan struct{})
	entered := fp.entered
	done := make(chan error, 1)
	go func() { _, err := f.ing.Run(context.Background()); done <- err }()
	<-entered
	other := &PlaudIngester{Plaud: fp, Memory: f.ing.Memory, Ledger: f.ing.Ledger, Route: f.ing.Route, Now: f.ing.Now}
	if _, err := other.Run(context.Background()); !errors.Is(err, ErrIngestRunning) {
		t.Fatalf("a second ingester must be refused while a pass runs: %v", err)
	}
	close(fp.gate)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if n := len(f.mem.genre("transcript")); n != 1 {
		t.Fatalf("captured %d pages, want 1", n)
	}
	if _, err := other.Run(context.Background()); err != nil {
		t.Fatalf("after the pass ends a new one may start: %v", err)
	}
}

// Until the flip, prod owns what gets filed from Plaud: a pass with
// FOUNDEROS_WRITES=0 is refused before it lists anything, so no page lands in
// the staging engine and no ledger row exists for the cutover ETL to fight.
func TestIngestIsRefusedWhileWritesAreOff(t *testing.T) {
	f := newIngestFixture(t)
	t.Setenv("FOUNDEROS_WRITES", "0")
	if _, err := f.ing.Run(context.Background()); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("err = %v", err)
	}
	if len(f.mem.captured) != 0 {
		t.Fatalf("captured while writes are off: %+v", f.mem.captured)
	}
}
