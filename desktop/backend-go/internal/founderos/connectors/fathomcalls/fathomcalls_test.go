package fathomcalls

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Fixtures mirror GET api.fathom.ai/external/v1/meetings (developers.fathom.ai).
// Names and emails are invented.
const listBody = `{
  "items": [
    {"title": "Vantage x Acme discovery", "meeting_title": "Vantage x Acme discovery",
     "url": "https://fathom.video/calls/123", "recording_id": 123,
     "created_at": "2026-08-04T15:00:00Z", "recording_duration_in_minutes": 42,
     "recording_start_time": "2026-08-04T14:15:00Z", "recording_end_time": "2026-08-04T14:57:00Z",
     "scheduled_start_time": "2026-08-04T14:15:00Z",
     "calendar_invitees": [{"name": " Ada Example ", "email": " Ada@Example.com ", "is_external": true},
                           {"name": "Alex", "email": "alex@launchpadcohort.example", "is_external": false}]},
    {"title": "Cohort office hours", "url": "https://fathom.video/calls/124", "recording_id": "124",
     "created_at": "2026-08-02T17:00:00Z",
     "recording_start_time": "2026-08-02T17:00:00Z", "recording_end_time": "2026-08-02T18:01:00Z"},
    {"meeting_title": null, "title": null, "created_at": "2026-08-01T09:00:00Z"}
  ],
  "next_cursor": null
}`

func fullItem(id int, title, at string) string {
	return `{"title": "` + title + `", "meeting_title": "` + title + `",
	  "url": "https://fathom.video/calls/` + strconv.Itoa(id) + `", "recording_id": ` + strconv.Itoa(id) + `,
	  "created_at": "` + at + `", "recording_start_time": "` + at + `", "recording_end_time": "` + at + `",
	  "recorded_by": {"name": "Alex", "email": "alex@launchpadcohort.example"},
	  "calendar_invitees": [{"email": "jamie@example.com", "name": "Jamie O."}],
	  "transcript": [
	    {"speaker": {"display_name": "Jamie O."}, "text": "Doing well, thanks.", "timestamp": "00:00:00"},
	    {"speaker": {"display_name": "Jamie O."}, "text": "Busy week.", "timestamp": "00:00:02"},
	    {"speaker": {"display_name": "Alex"}, "text": "Tell me about the rollout.", "timestamp": "00:00:04"},
	    {"speaker": {"display_name": null}, "text": "   ", "timestamp": "00:00:06"}
	  ],
	  "default_summary": {"template_name": "Enhanced", "markdown_formatted": "## Meeting Purpose\n\nExplore AI for retention."},
	  "action_items": [{"description": "Draft proposal w/ CTO", "completed": false, "recording_timestamp": "00:31:12", "assignee": {"name": "Alex"}}]
	}`
}

type hit struct {
	method, path string
	query        map[string]string
	key          string
}

type fake struct {
	mu   sync.Mutex
	hits []hit
	srv  *httptest.Server
}

func newFake(t *testing.T, route func(r *http.Request) (int, string)) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		f.mu.Lock()
		f.hits = append(f.hits, hit{r.Method, r.URL.Path, q, r.Header.Get("X-Api-Key")})
		f.mu.Unlock()
		status, body := route(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func fixed(status int, body string) func(*http.Request) (int, string) {
	return func(*http.Request) (int, string) { return status, body }
}

func (f *fake) calls() []hit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hit(nil), f.hits...)
}

func keyed(t *testing.T, key, base string) *Connector {
	t.Helper()
	dir := t.TempDir()
	envLocal := filepath.Join(dir, "env.local")
	if key != "" {
		if err := os.WriteFile(envLocal, []byte("FATHOM_API_KEY="+key+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FATHOM_API_KEY", "")
	c := New(connectors.Resolver{EnvLocal: envLocal})
	c.BaseURL = base
	c.CredFiles = []string{filepath.Join(dir, "absent.env")}
	c.RetryDelay = time.Millisecond
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "fathom" || Meta.Name != "Fathom" || Meta.Kind != connectors.KindCRM {
		t.Fatalf("meta = %+v", Meta)
	}
}

func TestParseMeetings(t *testing.T) {
	rows := ParseMeetings([]byte(listBody))
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Title != "Vantage x Acme discovery" || rows[0].URL == nil || *rows[0].URL != "https://fathom.video/calls/123" ||
		rows[0].DurationMinutes == nil || *rows[0].DurationMinutes != 42 || rows[0].At != "2026-08-04T15:00:00Z" {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[1].DurationMinutes != nil {
		t.Error("the flat list reads duration only from recording_duration_in_minutes")
	}
	if rows[2].Title != "Untitled meeting" || rows[2].URL != nil {
		t.Errorf("row2 = %+v", rows[2])
	}
	for _, body := range []string{`{}`, `null`, `{"items":"nope"}`} {
		if got := ParseMeetings([]byte(body)); len(got) != 0 {
			t.Errorf("%s → %+v", body, got)
		}
	}
}

func TestParseCalls(t *testing.T) {
	calls, next := ParseCalls([]byte(listBody))
	if next != "" {
		t.Errorf("next = %q", next)
	}
	if len(calls) != 2 {
		t.Fatalf("a row without a recording id is skipped: %+v", calls)
	}
	c := calls[0]
	if c.RecordingID != "123" || c.At != "2026-08-04T14:15:00Z" || c.ScheduledAt == nil || c.DurationMinutes == nil || *c.DurationMinutes != 42 {
		t.Errorf("call0 = %+v", c)
	}
	if len(c.Invitees) != 2 || c.Invitees[0].Email == nil || *c.Invitees[0].Email != "ada@example.com" ||
		*c.Invitees[0].Name != "Ada Example" || !c.Invitees[0].External || c.Invitees[1].External {
		t.Errorf("invitees = %+v", c.Invitees)
	}
	if calls[1].RecordingID != "124" || calls[1].DurationMinutes == nil || *calls[1].DurationMinutes != 61 {
		t.Errorf("duration derives from start/end when not given: %+v", calls[1])
	}
}

func TestStatusStates(t *testing.T) {
	f := newFake(t, fixed(200, listBody))
	none := keyed(t, "", f.srv.URL).Status(context.Background())
	if none.State != connectors.StateNotConfigured || !strings.Contains(none.Detail, "FATHOM_API_KEY") || len(f.calls()) != 0 {
		t.Errorf("none = %+v calls=%d", none, len(f.calls()))
	}
	ok := keyed(t, "fk", f.srv.URL).Status(context.Background())
	if ok.State != connectors.StateConnected || ok.Detail != "Fathom reachable · 3 recorded meetings visible to this key" || ok.Meta["meetings"] != 3 {
		t.Errorf("ok = %+v", ok)
	}
	if c := f.calls(); c[0].path != "/meetings" || c[0].key != "fk" || c[0].method != "GET" {
		t.Errorf("calls = %+v", c)
	}
	bad := keyed(t, "fk", newFake(t, fixed(401, `{}`)).srv.URL).Status(context.Background())
	if bad.State != connectors.StateError || bad.Detail != "FATHOM_API_KEY is set but the call failed: HTTP 401" {
		t.Errorf("bad = %+v", bad)
	}
	down := keyed(t, "fk", "http://127.0.0.1:1").Status(context.Background())
	if down.State != connectors.StateError {
		t.Errorf("down = %+v", down)
	}
}

func TestRecentMeetings(t *testing.T) {
	c := keyed(t, "fk", newFake(t, fixed(200, listBody)).srv.URL)
	rows, err := c.RecentMeetings(context.Background(), 2)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if _, err := keyed(t, "fk", newFake(t, fixed(500, `{}`)).srv.URL).RecentMeetings(context.Background(), 5); err == nil {
		t.Error("a failure is an error, not an empty list")
	}
	if _, err := keyed(t, "", "http://127.0.0.1:1").RecentMeetings(context.Background(), 5); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
}

func TestCallsFollowsCursorWithinWindow(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	f := newFake(t, func(r *http.Request) (int, string) {
		switch r.URL.Query().Get("cursor") {
		case "":
			return 200, `{"items": [{"recording_id": 1, "created_at": "2026-09-01T00:00:00Z"}], "next_cursor": "p2"}`
		case "p2":
			return 200, `{"items": [{"recording_id": 2, "created_at": "2026-08-01T00:00:00Z"}], "next_cursor": "p3"}`
		}
		return 200, `{"items": [{"recording_id": 3, "created_at": "2026-07-01T00:00:00Z"}], "next_cursor": "p4"}`
	})
	calls, err := keyed(t, "fk", f.srv.URL).Calls(context.Background(), CallsOptions{Now: now})
	if err != nil || len(calls) != 3 {
		t.Fatalf("calls=%+v err=%v", calls, err)
	}
	hits := f.calls()
	if len(hits) != 3 {
		t.Fatalf("maxPages defaults to 3, got %d", len(hits))
	}
	if hits[0].query["created_after"] != "2026-07-01T12:00:00.000Z" || hits[1].query["cursor"] != "p2" {
		t.Errorf("hits = %+v", hits)
	}
}

func TestCallsUnknownIsNotEmpty(t *testing.T) {
	if _, err := keyed(t, "", "http://127.0.0.1:1").Calls(context.Background(), CallsOptions{}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("no key = %v", err)
	}
	calls, err := keyed(t, "fk", newFake(t, fixed(401, `{}`)).srv.URL).Calls(context.Background(), CallsOptions{})
	if err == nil || calls != nil {
		t.Errorf("refused must be an error with no calls: %+v %v", calls, err)
	}
	// A later page failing keeps the pages already read, and says so.
	f := newFake(t, func(r *http.Request) (int, string) {
		if r.URL.Query().Get("cursor") == "" {
			return 200, `{"items": [{"recording_id": 1, "created_at": "2026-09-01T00:00:00Z"}], "next_cursor": "p2"}`
		}
		return 500, `{}`
	})
	partial, err := keyed(t, "fk", f.srv.URL).Calls(context.Background(), CallsOptions{})
	if len(partial) != 1 || err == nil {
		t.Errorf("partial = %+v err=%v", partial, err)
	}
}

func TestMeetingsFullPagesWithTranscriptsAndRetries429(t *testing.T) {
	var mu sync.Mutex
	limited := false
	f := newFake(t, func(r *http.Request) (int, string) {
		q := r.URL.Query()
		if q.Get("include_transcript") != "true" || q.Get("include_summary") != "true" || q.Get("include_action_items") != "true" || q.Get("limit") != "100" {
			return 400, `{}`
		}
		mu.Lock()
		defer mu.Unlock()
		if q.Get("cursor") == "" {
			return 200, `{"items": [` + fullItem(1, "Jamie x Alex", "2026-08-25T16:03:36Z") + `], "next_cursor": "n2"}`
		}
		if !limited {
			limited = true
			return 429, `{}`
		}
		return 200, `{"items": [` + fullItem(3, "Fathom Demo", "2021-09-16T20:42:47Z") + `], "next_cursor": null}`
	})
	var got []MeetingFull
	err := keyed(t, "fk", f.srv.URL).MeetingsFull(context.Background(), func(page []MeetingFull) error {
		got = append(got, page...)
		return nil
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%d err=%v", len(got), err)
	}
	if len(f.calls()) != 3 {
		t.Errorf("one 429 retry expected, calls = %d", len(f.calls()))
	}
	m := got[0]
	if m.ID() != "1" || m.Title() != "Jamie x Alex" || m.At() != "2026-08-25T16:03:36Z" || m.DefaultSummary == nil ||
		!strings.Contains(m.DefaultSummary.MarkdownFormatted, "Meeting Purpose") || len(m.ActionItems) != 1 {
		t.Errorf("m = %+v", m)
	}
	lines := TranscriptLines(m)
	want := []string{"[00:00:00] Jamie O.: Doing well, thanks. Busy week.", "[00:00:04] Alex: Tell me about the rollout."}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("lines = %q", lines)
	}
	if IsSample(m) || !IsSample(got[1]) {
		t.Error("only the Fathom Demo call is a sample")
	}
	if !m.HasTranscript() {
		t.Error("has transcript")
	}
}

func TestMeetingsFullHelpers(t *testing.T) {
	var m MeetingFull
	m.URL = "https://fathom.video/calls/987"
	if m.ID() != "987" {
		t.Errorf("id from url = %q", m.ID())
	}
	m.CalendarInvitees = []Invitee{{Email: strPtr("susannah.durant@Fathom.video")}}
	if !IsSample(m) {
		t.Error("a fathom.video invitee marks the demo call")
	}
	m.RecordingStartTime, m.RecordingEndTime = "2021-09-16T20:42:47Z", "2031-09-16T20:42:47Z"
	if m.DurationMinutes() != nil {
		t.Error("a call is under a day; the demo's years-long end time reads unknown")
	}
	m.RecordingEndTime = "2021-09-16T21:12:47Z"
	if d := m.DurationMinutes(); d == nil || *d != 30 {
		t.Errorf("duration = %v", d)
	}
}

func TestMeetingsFullErrors(t *testing.T) {
	err := keyed(t, "fk", newFake(t, fixed(401, `{}`)).srv.URL).MeetingsFull(context.Background(), func([]MeetingFull) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("err = %v", err)
	}
	if err := keyed(t, "", "http://127.0.0.1:1").MeetingsFull(context.Background(), func([]MeetingFull) error { return nil }); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
}

func TestOnlyReadsLeave(t *testing.T) {
	f := newFake(t, fixed(200, listBody))
	c := keyed(t, "fk", f.srv.URL)
	c.Status(context.Background())
	_, _ = c.RecentMeetings(context.Background(), 5)
	_, _ = c.Calls(context.Background(), CallsOptions{})
	_ = c.MeetingsFull(context.Background(), func([]MeetingFull) error { return nil })
	for _, h := range f.calls() {
		if h.method != http.MethodGet {
			t.Fatalf("Fathom is read-only here: %+v", h)
		}
	}
}

func strPtr(s string) *string { return &s }
