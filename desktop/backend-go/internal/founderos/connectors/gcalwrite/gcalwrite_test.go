package gcalwrite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

func writeEnv(t *testing.T, kv map[string]string) connectors.Resolver {
	t.Helper()
	for _, k := range []string{"GCAL_CLIENT_ID", "GCAL_CLIENT_SECRET", "GCAL_REFRESH_TOKEN"} {
		t.Setenv(k, "")
	}
	var b strings.Builder
	for k, v := range kv {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

var fullCreds = map[string]string{"GCAL_CLIENT_ID": "cid", "GCAL_CLIENT_SECRET": "secret", "GCAL_REFRESH_TOKEN": "rt"}

// fakeGoogle serves the OAuth token endpoint and the Calendar v3 events API
// with response bodies shaped like Google's.
type fakeGoogle struct {
	mu          sync.Mutex
	tokenStatus int
	tokenBody   string
	event       map[string]any // GET events/{id}
	eventStatus int
	patchStatus int
	patchBody   string
	list        []map[string]any
	listStatus  int
	calls       []call
}

type call struct {
	method, path string
	query        url.Values
	auth         string
	body         string
}

func (f *fakeGoogle) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.calls = append(f.calls, call{r.Method, r.URL.EscapedPath(), r.URL.Query(), r.Header.Get("Authorization"), string(body)})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/token":
			form, _ := url.ParseQuery(string(body))
			if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "rt" || form.Get("client_id") != "cid" || form.Get("client_secret") != "secret" {
				w.WriteHeader(400)
				io.WriteString(w, `{"error":"invalid_request"}`)
				return
			}
			if f.tokenStatus != 0 {
				w.WriteHeader(f.tokenStatus)
				io.WriteString(w, f.tokenBody)
				return
			}
			io.WriteString(w, `{"access_token":"at","expires_in":3599,"scope":"https://www.googleapis.com/auth/calendar.events","token_type":"Bearer"}`)
		case r.Header.Get("Authorization") != "Bearer at":
			w.WriteHeader(401)
			io.WriteString(w, `{"error":{"code":401,"message":"Invalid Credentials"}}`)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/events"):
			if f.listStatus != 0 {
				w.WriteHeader(f.listStatus)
				io.WriteString(w, `{"error":{"code":403,"message":"Insufficient Permission"}}`)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"kind": "calendar#events", "items": f.list})
		case r.Method == http.MethodGet:
			if f.eventStatus != 0 {
				w.WriteHeader(f.eventStatus)
				io.WriteString(w, `{"error":{"code":404,"message":"Not Found"}}`)
				return
			}
			json.NewEncoder(w).Encode(f.event)
		case r.Method == http.MethodPatch:
			if f.patchStatus != 0 {
				w.WriteHeader(f.patchStatus)
				io.WriteString(w, f.patchBody)
				return
			}
			io.WriteString(w, `{"id":"evt123"}`)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fakeGoogle) patches() []call {
	var out []call
	for _, c := range f.calls {
		if c.method == http.MethodPatch {
			out = append(out, c)
		}
	}
	return out
}

func testConnector(t *testing.T, f *fakeGoogle, kv map[string]string) *Connector {
	srv := f.server(t)
	c := New(writeEnv(t, kv))
	c.tokenURL = srv.URL + "/token"
	c.apiBase = srv.URL + "/calendar/v3"
	return c
}

func baseReq(emails ...string) AddGuestsRequest {
	if len(emails) == 0 {
		emails = []string{"new@y.com"}
	}
	return AddGuestsRequest{CalendarID: "primary", EventID: "evt123", Emails: emails}
}

// ---- pure -------------------------------------------------------------------

func TestCredsNeedAllThreeParts(t *testing.T) {
	if c := New(writeEnv(t, map[string]string{"GCAL_CLIENT_ID": "cid", "GCAL_CLIENT_SECRET": "secret"})).Creds(); c != nil {
		t.Errorf("two parts cannot mint a token: %+v", c)
	}
	c := New(writeEnv(t, fullCreds)).Creds()
	if c == nil || c.ClientID != "cid" || c.ClientSecret != "secret" || c.RefreshToken != "rt" {
		t.Errorf("creds = %+v", c)
	}
}

func TestValidEmails(t *testing.T) {
	ok, bad := ValidEmails([]string{"a@b.com", "not-an-email", " C@D.COM ", "x@y"})
	if !reflect.DeepEqual(ok, []string{"a@b.com", "c@d.com"}) || !reflect.DeepEqual(bad, []string{"not-an-email", "x@y"}) {
		t.Errorf("ok=%v bad=%v", ok, bad)
	}
	if ok, _ := ValidEmails([]string{"A@b.com", "a@B.com"}); !reflect.DeepEqual(ok, []string{"a@b.com"}) {
		t.Errorf("dedupe: %v", ok)
	}
}

func TestMergeAttendees(t *testing.T) {
	got := MergeAttendees([]Attendee{{"email": "keep@x.com", "responseStatus": "accepted"}}, []string{"new@y.com"})
	want := []Attendee{{"email": "keep@x.com", "responseStatus": "accepted"}, {"email": "new@y.com"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keeps RSVP: %v", got)
	}
	got = MergeAttendees([]Attendee{{"email": "Dupe@X.com", "responseStatus": "declined"}}, []string{"dupe@x.com", "fresh@z.com"})
	want = []Attendee{{"email": "Dupe@X.com", "responseStatus": "declined"}, {"email": "fresh@z.com"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("never re-adds: %v", got)
	}
	got = MergeAttendees([]Attendee{{"email": "boss@x.com", "organizer": true, "responseStatus": "accepted"}}, []string{"a@b.com"})
	if !reflect.DeepEqual(got[0], Attendee{"email": "boss@x.com", "organizer": true, "responseStatus": "accepted"}) {
		t.Errorf("organizer row: %v", got[0])
	}
}

// ---- add guests -------------------------------------------------------------

func TestAddEventGuestsRefusesWithoutCredentials(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{}
	c := testConnector(t, f, map[string]string{"GCAL_CLIENT_ID": "cid"})
	res, err := c.AddEventGuests(context.Background(), baseReq("ok@x.com", "bogus"))
	if err == nil || res.OK || !strings.Contains(strings.ToLower(res.Error), "credential") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !reflect.DeepEqual(res.Rejected, []string{"bogus"}) || len(f.calls) != 0 {
		t.Errorf("rejected=%v calls=%d", res.Rejected, len(f.calls))
	}
}

func TestAddEventGuestsReadsMergesAndPatchesTheUnion(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{event: map[string]any{"id": "evt123", "attendees": []any{map[string]any{"email": "old@x.com", "responseStatus": "accepted"}}}}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), AddGuestsRequest{CalendarID: "team@group.calendar.google.com", EventID: "evt123", Emails: []string{"new@y.com"}})
	if err != nil || !res.OK || !reflect.DeepEqual(res.Added, []string{"new@y.com"}) || len(res.AlreadyPresent) != 0 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	p := f.patches()
	if len(p) != 1 {
		t.Fatalf("want one PATCH, got %d", len(p))
	}
	if p[0].path != "/calendar/v3/calendars/team%40group.calendar.google.com/events/evt123" || p[0].auth != "Bearer at" {
		t.Errorf("patch path/auth: %+v", p[0])
	}
	var body struct {
		Attendees []Attendee `json:"attendees"`
	}
	json.Unmarshal([]byte(p[0].body), &body)
	want := []Attendee{{"email": "old@x.com", "responseStatus": "accepted"}, {"email": "new@y.com"}}
	if !reflect.DeepEqual(body.Attendees, want) {
		t.Errorf("patched attendees = %v", body.Attendees)
	}
	if p[0].query.Get("sendUpdates") != "none" {
		t.Errorf("default must be sendUpdates=none, got %q", p[0].query.Get("sendUpdates"))
	}
}

func TestAddEventGuestsMailsOnlyWhenExplicitlyTold(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{event: map[string]any{"id": "evt123", "attendees": []any{}}}
	c := testConnector(t, f, fullCreds)
	req := baseReq()
	req.SendUpdates = "all"
	if _, err := c.AddEventGuests(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := f.patches()[0].query.Get("sendUpdates"); got != "all" {
		t.Errorf("sendUpdates = %q", got)
	}
	req.SendUpdates = "everyone!"
	if _, err := c.AddEventGuests(context.Background(), req); err == nil {
		t.Errorf("an unknown sendUpdates value must be refused")
	}
}

func TestAddEventGuestsNoPatchWhenEveryoneIsAlreadyInvited(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0") // nothing to write, so the guard is not even asked
	f := &fakeGoogle{event: map[string]any{"id": "evt123", "attendees": []any{map[string]any{"email": "New@Y.com"}}}}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), baseReq())
	if err != nil || !res.OK || len(res.Added) != 0 || !reflect.DeepEqual(res.AlreadyPresent, []string{"new@y.com"}) {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(f.patches()) != 0 {
		t.Error("must not PATCH (and re-notify) when nothing changes")
	}
}

func TestAddEventGuestsReportsMalformedAddresses(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{event: map[string]any{"id": "evt123"}}
	c := testConnector(t, f, fullCreds)
	res, _ := c.AddEventGuests(context.Background(), baseReq("good@x.com", "bogus"))
	if !reflect.DeepEqual(res.Rejected, []string{"bogus"}) || !reflect.DeepEqual(res.Added, []string{"good@x.com"}) {
		t.Errorf("res = %+v", res)
	}
	if strings.Contains(f.patches()[0].body, "bogus") {
		t.Error("a malformed address reached Google")
	}
}

func TestAddEventGuestsSurfacesARefusedToken(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{tokenStatus: 400, tokenBody: `{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), baseReq())
	if err == nil || res.OK || res.Error != "token refresh failed: invalid_grant" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAddEventGuestsSurfacesAFailedRead(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{eventStatus: 404}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), baseReq())
	if err == nil || res.Error != "event read failed: 404" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAddEventGuestsSurfacesAFailedPatch(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{event: map[string]any{"id": "evt123"}, patchStatus: 403, patchBody: `{"error":{"code":403,"message":"forbidden"}}`}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), baseReq())
	if err == nil || res.OK || res.Error != "patch failed (403): forbidden" || len(res.Added) != 0 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestAddEventGuestsRefusedWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := &fakeGoogle{event: map[string]any{"id": "evt123", "attendees": []any{}}}
	c := testConnector(t, f, fullCreds)
	res, err := c.AddEventGuests(context.Background(), baseReq())
	if !errors.Is(err, guard.ErrWritesDisabled) || res.OK || len(res.Added) != 0 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(f.patches()) != 0 {
		t.Fatal("a PATCH left the bridge while writes were off")
	}
	r := guard.Refused()
	if len(r) == 0 || r[len(r)-1].Action != "gcal.addGuests" {
		t.Errorf("refusal not recorded: %+v", r)
	}
}

func TestPlanEventGuestsNeverPatches(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeGoogle{event: map[string]any{"id": "evt123", "attendees": []any{map[string]any{"email": "old@x.com"}}}}
	c := testConnector(t, f, fullCreds)
	res, err := c.PlanEventGuests(context.Background(), baseReq("old@x.com", "new@y.com", "bad"))
	if err != nil || !res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !reflect.DeepEqual(res.Added, []string{"new@y.com"}) || !reflect.DeepEqual(res.AlreadyPresent, []string{"old@x.com"}) || !reflect.DeepEqual(res.Rejected, []string{"bad"}) {
		t.Errorf("plan = %+v", res)
	}
	if len(f.patches()) != 0 {
		t.Error("a dry run must not PATCH")
	}
}

// ---- reads ------------------------------------------------------------------

var sampleList = []map[string]any{
	{"id": "weekly1", "summary": "LC/VIP Weekly Call", "recurrence": []any{"RRULE:FREQ=WEEKLY;BYDAY=TH"}, "start": map[string]any{"dateTime": "2026-09-03T17:00:00-04:00"}, "attendees": []any{map[string]any{"email": "a@x.com"}}},
	{"id": "oneoff", "summary": "LC/VIP Weekly Call", "start": map[string]any{"dateTime": "2026-09-10T17:00:00-04:00"}},
	{"id": "other", "summary": "Offsite", "start": map[string]any{"date": "2026-09-20"}},
}

func TestListEvents(t *testing.T) {
	f := &fakeGoogle{list: sampleList}
	c := testConnector(t, f, fullCreds)
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	evs, err := c.ListEvents(context.Background(), "", "Weekly")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 3 || evs[0].ID != "weekly1" || !evs[0].Recurring() || evs[1].Recurring() || len(evs[0].Attendees) != 1 {
		t.Fatalf("events = %+v", evs)
	}
	if evs[0].StartString() != "2026-09-03T17:00:00-04:00" || evs[2].StartString() != "2026-09-20" {
		t.Errorf("starts = %q %q", evs[0].StartString(), evs[2].StartString())
	}
	var list call
	for _, cl := range f.calls {
		if strings.HasSuffix(cl.path, "/events") {
			list = cl
		}
	}
	q := list.query
	if list.path != "/calendar/v3/calendars/primary/events" || q.Get("q") != "Weekly" || q.Get("maxResults") != "50" ||
		q.Get("singleEvents") != "false" || q.Get("orderBy") != "updated" || q.Get("timeMin") != "2026-09-13T12:00:00.000Z" {
		t.Errorf("list request = %s %v", list.path, q)
	}
}

func TestListEventsErrorIsNotAnEmptyCalendar(t *testing.T) {
	f := &fakeGoogle{listStatus: 403}
	c := testConnector(t, f, fullCreds)
	if evs, err := c.ListEvents(context.Background(), "primary", ""); err == nil || evs != nil {
		t.Fatalf("evs=%v err=%v", evs, err)
	}
}

func TestFindRecurringEventNeedsExactlyOneMatch(t *testing.T) {
	f := &fakeGoogle{list: sampleList}
	c := testConnector(t, f, fullCreds)
	ev, err := c.FindRecurringEvent(context.Background(), "primary", "lc/vip weekly call")
	if err != nil || ev.ID != "weekly1" {
		t.Fatalf("ev=%+v err=%v", ev, err)
	}
	if _, err := c.FindRecurringEvent(context.Background(), "primary", "Offsite"); err == nil {
		t.Error("a non-recurring title must not match")
	}
	f.list = append(f.list, map[string]any{"id": "weekly2", "summary": "LC/VIP Weekly Call", "recurrence": []any{"RRULE:FREQ=WEEKLY"}})
	if _, err := c.FindRecurringEvent(context.Background(), "primary", "LC/VIP Weekly Call"); err == nil || !strings.Contains(err.Error(), "matched 2") {
		t.Errorf("two matches must be refused, got %v", err)
	}
}

// ---- status -----------------------------------------------------------------

func TestStatusNotConfigured(t *testing.T) {
	s := New(writeEnv(t, map[string]string{"GCAL_CLIENT_ID": "cid", "GCAL_CLIENT_SECRET": "s"})).Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.ID != "gcal-write" || s.Kind != connectors.KindCalendar {
		t.Fatalf("status = %+v", s)
	}
	if !strings.Contains(s.Detail, "GCAL_CLIENT_ID / GCAL_CLIENT_SECRET / GCAL_REFRESH_TOKEN") {
		t.Errorf("detail = %q", s.Detail)
	}
}

func TestStatusConnectedSaysWhetherWritesAreOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	f := &fakeGoogle{list: sampleList[:1]}
	c := testConnector(t, f, fullCreds)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Calendar API · OAuth ok (calendar.events) · writes off (FOUNDEROS_WRITES=0)" {
		t.Fatalf("status = %s %q", s.State, s.Detail)
	}
	for _, cl := range f.calls {
		if cl.method != http.MethodGet && cl.path != "/token" {
			t.Errorf("status made a write: %+v", cl)
		}
	}
	t.Setenv("FOUNDEROS_WRITES", "1")
	if s := c.Status(context.Background()); s.Detail != "Calendar API · OAuth ok (calendar.events)" {
		t.Errorf("detail = %q", s.Detail)
	}
}

func TestStatusErrorOnRefusedTokenOrAPI(t *testing.T) {
	f := &fakeGoogle{tokenStatus: 400, tokenBody: `{"error":"invalid_grant"}`}
	s := testConnector(t, f, fullCreds).Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "token refresh failed: invalid_grant" {
		t.Errorf("status = %s %q", s.State, s.Detail)
	}
	f2 := &fakeGoogle{listStatus: 403}
	s = testConnector(t, f2, fullCreds).Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Calendar API 403: Insufficient Permission" {
		t.Errorf("status = %s %q", s.State, s.Detail)
	}
}

func TestTokenPOSTIsAllowedAsARead(t *testing.T) {
	// The token mint is a POST with no side effect; it is allow-listed so the
	// guarded client lets Status run while writes are off.
	if !strings.HasPrefix(defaultTokenURL, "https://"+tokenReadPOST) {
		t.Fatalf("allow-list %q does not cover %q", tokenReadPOST, defaultTokenURL)
	}
}
