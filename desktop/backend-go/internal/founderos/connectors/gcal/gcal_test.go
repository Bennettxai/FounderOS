package gcal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var acct = CalAccount{User: "a@x.com", Pass: "pw", Name: "A", Color: "#3df08c"}

func wrap(lines ...string) string {
	return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Google Inc//Google Calendar 70.9054//EN\r\n" +
		strings.Join(lines, "\r\n") + "\r\nEND:VCALENDAR\r\n"
}

func ms(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func writeEnv(t *testing.T, kv map[string]string) connectors.Resolver {
	t.Helper()
	for slot := 1; slot <= 4; slot++ {
		for _, k := range []string{"HOST", "USER", "PASS", "NAME"} {
			t.Setenv(fmt.Sprintf("INBOX_%d_%s", slot, k), "")
		}
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

// ---- accounts ---------------------------------------------------------------

func TestAccountsDerivesGoogleInboxSlots(t *testing.T) {
	c := New(writeEnv(t, map[string]string{
		"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": "one@founderos.local", "INBOX_1_PASS": "aaaa bbbb cccc dddd", "INBOX_1_NAME": "HQ",
		"INBOX_2_HOST": "imap.gmail.com", "INBOX_2_USER": "two@gmail.com", // incomplete
		"INBOX_3_HOST": "outlook.office365.com", "INBOX_3_USER": "three@work.com", "INBOX_3_PASS": "zzzz", // non-Google
		"INBOX_4_HOST": "imap.gmail.com", "INBOX_4_USER": "four@vantage.example", "INBOX_4_PASS": "eeee", "INBOX_4_NAME": "Vantage",
	}))
	got := c.Accounts()
	if len(got) != 2 || got[0].User != "one@founderos.local" || got[1].User != "four@vantage.example" {
		t.Fatalf("accounts = %+v", got)
	}
	if got[0].Pass != "aaaabbbbccccdddd" {
		t.Errorf("app-password spaces must be stripped: %q", got[0].Pass)
	}
	if got[0].Name != "HQ" || got[0].Color == got[1].Color {
		t.Errorf("name/colour wrong: %+v", got)
	}
	if got[0].Color != "#3df08c" || got[1].Color != "#ffc53d" {
		t.Errorf("colour follows the slot: %+v", got)
	}
}

// ---- join url ---------------------------------------------------------------

func TestExtractJoinURL(t *testing.T) {
	cases := []struct {
		name string
		ev   JoinFields
		want string
	}{
		{"meet conference prop wins", JoinFields{Conference: "https://meet.google.com/abc-defg-hij", Description: "https://zoom.us/j/1"}, "https://meet.google.com/abc-defg-hij"},
		{"zoom in description", JoinFields{Description: "Join Zoom Meeting\nhttps://us02web.zoom.us/j/1234567890?pwd=abc\nDial-in..."}, "https://us02web.zoom.us/j/1234567890?pwd=abc"},
		{"teams in location", JoinFields{Location: "https://teams.microsoft.com/l/meetup-join/xyz"}, "https://teams.microsoft.com/l/meetup-join/xyz"},
		{"none", JoinFields{Location: "HQ conference room", Description: "bring coffee"}, ""},
	}
	for _, tc := range cases {
		if got := ExtractJoinURL(tc.ev); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// ---- iCal parsing -----------------------------------------------------------

func TestEventsFromIcalSingleEventWithMeet(t *testing.T) {
	ics := wrap("BEGIN:VEVENT", "UID:e1@g", "SUMMARY:Strategy call", "DTSTART:20260618T150000Z", "DTEND:20260618T153000Z",
		"LOCATION:Room 4\\, HQ", "X-GOOGLE-CONFERENCE:https://meet.google.com/abc-defg-hij", "END:VEVENT")
	evs := EventsFromIcal([]string{ics}, acct, ms("2026-06-16T00:00:00Z"), ms("2026-07-16T00:00:00Z"))
	if len(evs) != 1 {
		t.Fatalf("got %d", len(evs))
	}
	e := evs[0]
	if e.ID != "a@x.com:e1@g" || e.Title != "Strategy call" || e.Account != "A" || e.Color != "#3df08c" {
		t.Errorf("identity: %+v", e)
	}
	if e.Start != "2026-06-18T15:00:00.000Z" || e.End == nil || *e.End != "2026-06-18T15:30:00.000Z" || e.AllDay {
		t.Errorf("times: %+v", e)
	}
	if e.JoinURL == nil || *e.JoinURL != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("join: %v", e.JoinURL)
	}
	if e.Location == nil || *e.Location != "Room 4, HQ" {
		t.Errorf("location must be text-unescaped: %v", e.Location)
	}
}

func TestEventsFromIcalDropsEventsOutsideTheWindow(t *testing.T) {
	ics := wrap("BEGIN:VEVENT", "UID:old@g", "SUMMARY:Old", "DTSTART:20260101T090000Z", "DTEND:20260101T093000Z", "END:VEVENT")
	if evs := EventsFromIcal([]string{ics}, acct, ms("2026-06-16T00:00:00Z"), ms("2026-07-16T00:00:00Z")); len(evs) != 0 {
		t.Fatalf("got %+v", evs)
	}
}

func TestEventsFromIcalExpandsRecurrenceWithExdateAndOverride(t *testing.T) {
	ics := wrap(
		"BEGIN:VEVENT", "UID:rec@g", "SUMMARY:Weekly sync", "DTSTART:20260504T090000Z", "DTEND:20260504T093000Z",
		"RRULE:FREQ=WEEKLY;BYDAY=MO", "EXDATE:20260622T090000Z",
		"ATTENDEE:mailto:someone@x.com", "END:VEVENT",
		"BEGIN:VEVENT", "UID:rec@g", "RECURRENCE-ID:20260629T090000Z", "SUMMARY:Weekly sync (moved)",
		"DTSTART:20260630T100000Z", "DTEND:20260630T110000Z", "END:VEVENT",
	)
	evs := EventsFromIcal([]string{ics}, acct, ms("2026-06-16T00:00:00Z"), ms("2026-07-16T00:00:00Z"))
	// Mondays in window: 06-22 (excluded), 06-29 (overridden), 07-06, 07-13.
	var got []string
	for _, e := range evs {
		got = append(got, e.Start+" "+e.Title)
		if e.Attendees != nil {
			t.Errorf("recurring instances carry no attendees (they are not bookings): %+v", e)
		}
	}
	want := []string{
		"2026-06-30T10:00:00.000Z Weekly sync (moved)",
		"2026-07-06T09:00:00.000Z Weekly sync",
		"2026-07-13T09:00:00.000Z Weekly sync",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("instances:\n got %v\nwant %v", got, want)
	}
	if evs[1].ID != "a@x.com:rec@g-2026-07-06" || *evs[1].End != "2026-07-06T09:30:00.000Z" {
		t.Errorf("instance id/end: %+v", evs[1])
	}
}

func TestEventsFromIcalRecurrenceHonoursTimezoneAcrossDST(t *testing.T) {
	ics := wrap("BEGIN:VEVENT", "UID:tz@g", "SUMMARY:NY standup",
		"DTSTART;TZID=America/New_York:20261026T090000", "DTEND;TZID=America/New_York:20261026T091500",
		"RRULE:FREQ=WEEKLY;COUNT=3", "END:VEVENT")
	evs := EventsFromIcal([]string{ics}, acct, ms("2026-10-20T00:00:00Z"), ms("2026-11-20T00:00:00Z"))
	var got []string
	for _, e := range evs {
		got = append(got, e.Start)
	}
	// 9am New York is 13:00Z in EDT and 14:00Z after the 2026-11-01 switch.
	want := "2026-10-26T13:00:00.000Z,2026-11-02T14:00:00.000Z,2026-11-09T14:00:00.000Z"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v", got)
	}
}

func TestEventsFromIcalAttendeesAndBookingTime(t *testing.T) {
	ics := wrap("BEGIN:VEVENT", "UID:b1@g", "SUMMARY:Strategy call (Dana Reyes)",
		"DTSTART:20260925T150000Z", "DTEND:20260925T153000Z", "CREATED:20260921T142000Z",
		"ORGANIZER;CN=the operator:mailto:founderos@launchpadcohort.ai",
		"ATTENDEE;CN=the operator;PARTSTAT=ACCEPTED:mailto:founderos@launchpadcohort.ai",
		"ATTENDEE;CN=Dana Reyes;PARTSTAT=NEEDS-ACTION:mailto:Dana@Reyes.co",
		"ATTENDEE;CN=Dup:MAILTO:dana@reyes.co",
		"END:VEVENT")
	evs := EventsFromIcal([]string{ics}, acct, ms("2026-09-01T00:00:00Z"), ms("2026-10-01T00:00:00Z"))
	if len(evs) != 1 {
		t.Fatalf("got %d", len(evs))
	}
	if strings.Join(evs[0].Attendees, ",") != "founderos@launchpadcohort.ai,dana@reyes.co" {
		t.Errorf("attendees = %v", evs[0].Attendees)
	}
	if evs[0].CreatedAt != "2026-09-21T14:20:00.000Z" {
		t.Errorf("createdAt = %q", evs[0].CreatedAt)
	}
	solo := wrap("BEGIN:VEVENT", "UID:solo@g", "SUMMARY:Focus", "DTSTART:20260918T150000Z", "DTEND:20260918T160000Z", "END:VEVENT")
	e := EventsFromIcal([]string{solo}, acct, ms("2026-09-01T00:00:00Z"), ms("2026-10-01T00:00:00Z"))[0]
	if e.Attendees != nil || e.CreatedAt != "" {
		t.Errorf("an uninvited event carries no attendee list or createdAt: %+v", e)
	}
}

func TestEventsFromIcalAllDayAndMalformedBlobs(t *testing.T) {
	good := wrap("BEGIN:VEVENT", "UID:d@g", "SUMMARY:Offsite", "DTSTART;VALUE=DATE:20260920", "DTEND;VALUE=DATE:20260921", "END:VEVENT")
	evs := EventsFromIcal([]string{"not ical at all", "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n", good}, acct,
		ms("2026-09-01T00:00:00Z"), ms("2026-10-01T00:00:00Z"))
	if len(evs) != 1 || !evs[0].AllDay || evs[0].Title != "Offsite" {
		t.Fatalf("a malformed blob must be skipped, not fail the rest: %+v", evs)
	}
	untitled := wrap("BEGIN:VEVENT", "UID:u@g", "DTSTART:20260918T150000Z", "END:VEVENT")
	e := EventsFromIcal([]string{untitled}, acct, ms("2026-09-01T00:00:00Z"), ms("2026-10-01T00:00:00Z"))[0]
	if e.Title != "(no title)" || e.End != nil || e.Location != nil || e.JoinURL != nil {
		t.Errorf("fallbacks: %+v", e)
	}
}

// ---- merge ------------------------------------------------------------------

func mk(start, title, account string) CalEvent {
	return CalEvent{ID: account + ":" + title, Account: account, Title: title, Start: start}
}

func TestMergeUpcomingSortsLimitsAndDedupes(t *testing.T) {
	merged := MergeUpcoming([][]CalEvent{
		{mk("2026-06-20T10:00:00.000Z", "b", "A"), mk("2026-06-18T10:00:00.000Z", "a", "A")},
		{mk("2026-06-19T10:00:00.000Z", "c", "B")},
	}, 2)
	if len(merged) != 2 || merged[0].Title != "a" || merged[1].Title != "c" {
		t.Errorf("merged = %+v", merged)
	}
	dup := MergeUpcoming([][]CalEvent{
		{mk("2026-06-18T10:00:00.000Z", "LC EXEC", "HQ")},
		{mk("2026-06-18T10:00:00.000Z", "LC EXEC", "Launchpad Cohort")},
	}, 25)
	if len(dup) != 1 {
		t.Errorf("the same meeting on two calendars must dedupe: %+v", dup)
	}
}

// ---- CalDAV wire ------------------------------------------------------------

// multistatus renders a Google-shaped CalDAV 207 body.
func multistatus(icals ...string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<D:multistatus xmlns:D="DAV:" xmlns:caldav="urn:ietf:params:xml:ns:caldav">`)
	for i, ics := range icals {
		fmt.Fprintf(&b, `<D:response><D:href>/caldav/v2/a%%40x.com/events/%d.ics</D:href><D:propstat><D:status>HTTP/1.1 200 OK</D:status><D:prop><caldav:calendar-data>`, i)
		var esc bytes.Buffer
		xml.EscapeText(&esc, []byte(ics))
		b.Write(esc.Bytes())
		b.WriteString(`</caldav:calendar-data></D:prop></D:propstat></D:response>`)
	}
	b.WriteString(`</D:multistatus>`)
	return b.String()
}

func TestExtractCalendarData(t *testing.T) {
	ics := wrap("BEGIN:VEVENT", "UID:x", "SUMMARY:A & B <ok>", "DTSTART:20260918T150000Z", "END:VEVENT")
	got, err := extractCalendarData(strings.NewReader(multistatus(ics)))
	if err != nil || len(got) != 1 || got[0] != strings.TrimSpace(ics) {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := extractCalendarData(strings.NewReader("<html>not xml")); err == nil {
		t.Error("an unparseable body must be an error, not zero events")
	}
}

type davServer struct {
	mu       sync.Mutex
	byUser   map[string][]string // user -> ics blobs
	status   map[string]int      // user -> forced status
	requests []*http.Request
	bodies   []string
}

func (d *davServer) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		d.mu.Lock()
		d.requests = append(d.requests, r)
		d.bodies = append(d.bodies, string(body))
		d.mu.Unlock()
		// /calendar/dav/{email}/events/
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		user := parts[len(parts)-2]
		if code := d.status[user]; code != 0 {
			http.Error(w, "nope", code)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(207)
		io.WriteString(w, multistatus(d.byUser[user]...))
	}
}

func startDAV(t *testing.T, d *davServer) *httptest.Server {
	srv := httptest.NewServer(d.handler(t))
	t.Cleanup(srv.Close)
	return srv
}

func twoAccountEnv(t *testing.T) connectors.Resolver {
	return writeEnv(t, map[string]string{
		"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": "a@x.com", "INBOX_1_PASS": "pw a", "INBOX_1_NAME": "Agency",
		"INBOX_2_HOST": "imap.gmail.com", "INBOX_2_USER": "b@x.com", "INBOX_2_PASS": "pwb", "INBOX_2_NAME": "Vantage",
	})
}

var now0 = ms("2026-09-20T15:30:00Z")

func testConnector(res connectors.Resolver, base string) *Connector {
	c := New(res)
	c.baseURL = base + "/calendar/dav"
	c.now = func() time.Time { return now0 }
	return c
}

func TestUpcomingEventsSpeaksCalDAVAndMerges(t *testing.T) {
	shared := wrap("BEGIN:VEVENT", "UID:s@g", "SUMMARY:LC EXEC", "DTSTART:20260922T150000Z", "DTEND:20260922T160000Z", "END:VEVENT")
	d := &davServer{byUser: map[string][]string{
		"a@x.com": {shared, wrap("BEGIN:VEVENT", "UID:a1", "SUMMARY:Later", "DTSTART:20260925T150000Z", "END:VEVENT")},
		"b@x.com": {shared, wrap("BEGIN:VEVENT", "UID:b1", "SUMMARY:Sooner", "DTSTART:20260921T090000Z", "END:VEVENT")},
	}}
	srv := startDAV(t, d)
	c := testConnector(twoAccountEnv(t), srv.URL)
	evs, err := c.UpcomingEvents(context.Background(), UpcomingOptions{Days: 7, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, e := range evs {
		titles = append(titles, e.Title)
	}
	if strings.Join(titles, ",") != "Sooner,LC EXEC,Later" {
		t.Errorf("titles = %v", titles)
	}
	if len(d.requests) != 2 {
		t.Fatalf("want one REPORT per account, got %d", len(d.requests))
	}
	var sawA bool
	for i, r := range d.requests {
		if r.Method != "REPORT" || r.Header.Get("Depth") != "1" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/xml") {
			t.Errorf("request %d: %s depth=%q ct=%q", i, r.Method, r.Header.Get("Depth"), r.Header.Get("Content-Type"))
		}
		if r.URL.EscapedPath() == "/calendar/dav/a%40x.com/events/" {
			sawA = true
			want := "Basic " + base64.StdEncoding.EncodeToString([]byte("a@x.com:pwa"))
			if r.Header.Get("Authorization") != want {
				t.Errorf("auth header = %q (password spaces stripped)", r.Header.Get("Authorization"))
			}
		}
		// window starts at UTC midnight of now and spans Days.
		if !strings.Contains(d.bodies[i], `<c:time-range start="20260920T000000Z" end="20260927T000000Z"/>`) {
			t.Errorf("time-range wrong: %s", d.bodies[i])
		}
	}
	if !sawA {
		t.Errorf("no request hit the escaped account path")
	}
}

func TestUpcomingEventsOneAccountFailingCostsOnlyItsEvents(t *testing.T) {
	d := &davServer{
		byUser: map[string][]string{"b@x.com": {wrap("BEGIN:VEVENT", "UID:b1", "SUMMARY:Kept", "DTSTART:20260921T090000Z", "END:VEVENT")}},
		status: map[string]int{"a@x.com": 401},
	}
	srv := startDAV(t, d)
	c := testConnector(twoAccountEnv(t), srv.URL)
	evs, err := c.UpcomingEvents(context.Background(), UpcomingOptions{})
	if err != nil || len(evs) != 1 || evs[0].Title != "Kept" {
		t.Fatalf("evs=%+v err=%v", evs, err)
	}
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "1 upcoming · 2 calendars (next 14 days, CalDAV) · 1 failing" {
		t.Errorf("status = %s %q", s.State, s.Detail)
	}
}

func TestEveryAccountFailingIsAnErrorNotZeroEvents(t *testing.T) {
	d := &davServer{status: map[string]int{"a@x.com": 401, "b@x.com": 503}}
	srv := startDAV(t, d)
	c := testConnector(twoAccountEnv(t), srv.URL)
	if _, err := c.UpcomingEvents(context.Background(), UpcomingOptions{}); err == nil {
		t.Fatal("all accounts failing must be an error")
	}
	if _, err := c.CalendarBookings(context.Background(), BookingOptions{}); err == nil {
		t.Fatal("all accounts failing must be an error for bookings too")
	}
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "CalDAV read failed: ") || !strings.Contains(s.Detail, "CalDAV 401") {
		t.Errorf("status = %s %q", s.State, s.Detail)
	}
	if s.Meta["calendars"] != 2 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusConnected(t *testing.T) {
	d := &davServer{byUser: map[string][]string{
		"a@x.com": {wrap("BEGIN:VEVENT", "UID:a1", "SUMMARY:One", "DTSTART:20260921T090000Z", "END:VEVENT")},
	}}
	srv := startDAV(t, d)
	c := testConnector(writeEnv(t, map[string]string{"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": "a@x.com", "INBOX_1_PASS": "p"}), srv.URL)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "1 upcoming · 1 calendar (next 14 days, CalDAV)" {
		t.Fatalf("status = %s %q", s.State, s.Detail)
	}
	if s.ID != "calendar" || s.Name != "Calendar" || s.Kind != connectors.KindCalendar {
		t.Errorf("identity: %+v", s)
	}
	if s.Meta["calendars"] != 1 || s.Meta["upcoming"] != 1 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	c := New(writeEnv(t, map[string]string{"INBOX_1_HOST": "outlook.office365.com", "INBOX_1_USER": "x@y.com", "INBOX_1_PASS": "p"}))
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured {
		t.Fatalf("state = %s", s.State)
	}
	if s.Detail != "No Google inboxes configured — calendars reuse the INBOX_* Gmail app passwords." || s.Meta["calendars"] != 0 {
		t.Errorf("status = %+v", s)
	}
	if _, err := c.UpcomingEvents(context.Background(), UpcomingOptions{}); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
	if b, err := c.CalendarBookings(context.Background(), BookingOptions{}); !errors.Is(err, ErrNotConfigured) || b != nil {
		t.Errorf("no source must never read as no bookings: %v %v", b, err)
	}
}

func TestCalendarBookingsKeepsOnlyInvitedEventsInTheWindow(t *testing.T) {
	d := &davServer{byUser: map[string][]string{"a@x.com": {
		wrap("BEGIN:VEVENT", "UID:call", "SUMMARY:Strategy call", "DTSTART:20260801T150000Z", "DTEND:20260801T153000Z",
			"CREATED:20260728T100000Z", "ATTENDEE:mailto:lead@co.com", "END:VEVENT"),
		wrap("BEGIN:VEVENT", "UID:focus", "SUMMARY:Focus", "DTSTART:20260922T150000Z", "END:VEVENT"),
		wrap("BEGIN:VEVENT", "UID:rec", "SUMMARY:Weekly", "DTSTART:20260907T090000Z", "RRULE:FREQ=WEEKLY", "ATTENDEE:mailto:team@co.com", "END:VEVENT"),
	}}}
	srv := startDAV(t, d)
	c := testConnector(writeEnv(t, map[string]string{"INBOX_1_HOST": "imap.gmail.com", "INBOX_1_USER": "a@x.com", "INBOX_1_PASS": "p"}), srv.URL)
	got, err := c.CalendarBookings(context.Background(), BookingOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Strategy call" || got[0].CreatedAt != "2026-07-28T10:00:00.000Z" {
		t.Fatalf("bookings = %+v", got)
	}
	// default window: now-60d .. now+30d, exact (not day-aligned).
	if !strings.Contains(d.bodies[0], `start="20260722T153000Z" end="20261020T153000Z"`) {
		t.Errorf("window: %s", d.bodies[0])
	}
}

// ---- guard ------------------------------------------------------------------

type countingRT struct {
	mu    sync.Mutex
	calls []string
}

func (c *countingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.calls = append(c.calls, r.Method)
	c.mu.Unlock()
	return &http.Response{StatusCode: 207, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
}

// CalDAV reads use REPORT, which the bridge guard does not know is a read.
// The DAV transport lets REPORT/PROPFIND through and hands every other method
// to the guard, so a write to Google is still refused while writes are off.
func TestDAVTransportAllowsReadsAndRefusesWritesWhileOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	base := &countingRT{}
	rt := davTransport{guarded: guard.Transport(base), reads: base}
	for _, m := range []string{"REPORT", "PROPFIND"} {
		req, _ := http.NewRequest(m, "https://www.google.com/calendar/dav/a%40x.com/events/", nil)
		if _, err := rt.RoundTrip(req); err != nil {
			t.Errorf("%s must pass: %v", m, err)
		}
	}
	for _, m := range []string{"PUT", "DELETE", "PROPPATCH", "MKCALENDAR", "POST"} {
		req, _ := http.NewRequest(m, "https://www.google.com/calendar/dav/a%40x.com/events/x.ics", strings.NewReader("x"))
		if _, err := rt.RoundTrip(req); !errors.Is(err, guard.ErrWritesDisabled) {
			t.Errorf("%s must be refused, got %v", m, err)
		}
	}
	if strings.Join(base.calls, ",") != "REPORT,PROPFIND" {
		t.Errorf("only the reads may reach the wire: %v", base.calls)
	}
}

func TestConnectorClientUsesTheDAVTransport(t *testing.T) {
	c := New(writeEnv(t, nil))
	if _, ok := c.client().Transport.(davTransport); !ok {
		t.Fatalf("client transport = %T", c.client().Transport)
	}
}
