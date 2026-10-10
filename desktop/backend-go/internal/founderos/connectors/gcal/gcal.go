// Package gcal ports FounderOS v1 lib/connectors/gcal.ts: Google Calendar read
// over the legacy CalDAV endpoint, reusing the Gmail app passwords the inbox
// slots already carry (INBOX_n_*). A Google app password also authorizes
// https://www.google.com/calendar/dav/{email}/events/ over Basic auth, so the
// Gmail inboxes get calendars with zero extra setup. Read-only: events are
// REPORTed for a window and recurrences expanded client-side. Calendar writes
// live in the sibling gcalwrite package (OAuth Calendar API).
package gcal

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "calendar", Name: "Calendar", Kind: connectors.KindCalendar}

// ErrNotConfigured means no Google inbox slot exists: no source, which must
// never read as "no events" or "no bookings".
var ErrNotConfigured = errors.New("gcal: no Google calendar configured")

const (
	defaultBaseURL = "https://www.google.com/calendar/dav"
	accountTimeout = 8 * time.Second
	day            = 24 * time.Hour
)

// One stable colour per inbox slot (the by-platform accent family).
var calColors = []string{"#3df08c", "#5ec9f8", "#e1306c", "#ffc53d"}

// CalAccount is one CalDAV-capable inbox slot. Pass never serializes.
type CalAccount struct {
	User  string `json:"user"`
	Pass  string `json:"-"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Connector struct {
	res     connectors.Resolver
	baseURL string
	now     func() time.Time
	reads   http.RoundTripper // where REPORT/PROPFIND go; nil = http.DefaultTransport
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, baseURL: defaultBaseURL, now: time.Now}
}

var googleHost = regexp.MustCompile(`(?i)gmail|google`)

// Accounts returns the inbox slots whose host is Google (the legacy CalDAV
// path is Google-only), with app-password spaces stripped.
func (c *Connector) Accounts() []CalAccount {
	var out []CalAccount
	for slot := 1; slot <= 4; slot++ {
		key := func(k string) string { return c.res.Resolve(fmt.Sprintf("INBOX_%d_%s", slot, k)) }
		host, user, pass := key("HOST"), key("USER"), key("PASS")
		if host == "" || user == "" || pass == "" || !googleHost.MatchString(host) {
			continue
		}
		name := key("NAME")
		if name == "" {
			name = user
		}
		out = append(out, CalAccount{
			User:  user,
			Pass:  strings.Join(strings.Fields(pass), ""),
			Name:  name,
			Color: calColors[(slot-1)%len(calColors)],
		})
	}
	return out
}

// ---- transport --------------------------------------------------------------

// davTransport exists because CalDAV reads are REPORT and PROPFIND, which
// are read-only by definition (RFC 4791 §7, RFC 4918 §9.1) but which the
// bridge guard's transport, knowing only GET/HEAD/OPTIONS and allow-listed
// POSTs, would refuse. Those two methods go straight to the wire; every other
// method is handed to the guarded transport, so PUT/DELETE/PROPPATCH/
// MKCALENDAR are still refused while FOUNDEROS_WRITES is off.
type davTransport struct {
	guarded http.RoundTripper
	reads   http.RoundTripper
}

func (t davTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == "REPORT" || req.Method == "PROPFIND" {
		return t.reads.RoundTrip(req)
	}
	return t.guarded.RoundTrip(req)
}

func (c *Connector) client() *http.Client {
	hc := connectors.HTTPClient(accountTimeout)
	reads := c.reads
	if reads == nil {
		reads = http.DefaultTransport
	}
	hc.Transport = davTransport{guarded: hc.Transport, reads: reads}
	return hc
}

// ---- CalDAV -----------------------------------------------------------------

func compact(t time.Time) string { return t.UTC().Format("20060102T150405") + "Z" }

func pathEscape(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }

// fetchAccountIcal REPORTs one account's events collection for the window.
func (c *Connector) fetchAccountIcal(ctx context.Context, hc *http.Client, a CalAccount, start, end time.Time) ([]string, error) {
	body := `<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav"><d:prop><c:calendar-data/></d:prop>` +
		`<c:filter><c:comp-filter name="VCALENDAR"><c:comp-filter name="VEVENT">` +
		`<c:time-range start="` + compact(start) + `" end="` + compact(end) + `"/>` +
		`</c:comp-filter></c:comp-filter></c:filter></c:calendar-query>`
	req, err := http.NewRequestWithContext(ctx, "REPORT", c.baseURL+"/"+pathEscape(a.User)+"/events/", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(a.User, a.Pass)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusMultiStatus && (res.StatusCode < 200 || res.StatusCode > 299) {
		io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
		return nil, fmt.Errorf("CalDAV %d", res.StatusCode)
	}
	return extractCalendarData(res.Body)
}

// extractCalendarData returns every calendar-data payload (any namespace
// prefix) that holds a VCALENDAR. A body that is not XML is an error, never
// zero events.
func extractCalendarData(r io.Reader) ([]string, error) {
	dec := xml.NewDecoder(r)
	var out []string
	var buf strings.Builder
	inData := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("CalDAV response unreadable: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "calendar-data" {
				inData = true
				buf.Reset()
			}
		case xml.CharData:
			if inData {
				buf.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "calendar-data" && inData {
				inData = false
				if s := strings.TrimSpace(buf.String()); strings.Contains(s, "BEGIN:VCALENDAR") {
					out = append(out, s)
				}
			}
		}
	}
}

// gather reads every account for the window concurrently. One account failing
// costs only its events; errs holds each account's failure (nil when it
// answered).
func (c *Connector) gather(ctx context.Context, accounts []CalAccount, start, end time.Time, keep func(CalEvent) bool) ([][]CalEvent, []error) {
	hc := c.client()
	per := make([][]CalEvent, len(accounts))
	errs := make([]error, len(accounts))
	var wg sync.WaitGroup
	for i, a := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			actx, cancel := context.WithTimeout(ctx, accountTimeout)
			defer cancel()
			blobs, err := c.fetchAccountIcal(actx, hc, a, start, end)
			if err != nil {
				errs[i] = fmt.Errorf("%s: %w", a.Name, err)
				return
			}
			for _, ev := range EventsFromIcal(blobs, a, start, end) {
				if keep == nil || keep(ev) {
					per[i] = append(per[i], ev)
				}
			}
		}()
	}
	wg.Wait()
	return per, errs
}

func allFailed(errs []error) error {
	for _, err := range errs {
		if err == nil {
			return nil
		}
	}
	return errs[0]
}

func failing(errs []error) int {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	return n
}

// UpcomingOptions: Days defaults to 14 and Limit to 25; the window starts at
// UTC midnight of Now (default time.Now).
type UpcomingOptions struct {
	Days  int
	Limit int
	Now   time.Time
}

func (c *Connector) upcoming(ctx context.Context, o UpcomingOptions) ([]CalEvent, []error, error) {
	if o.Days <= 0 {
		o.Days = 14
	}
	if o.Limit <= 0 {
		o.Limit = 25
	}
	if o.Now.IsZero() {
		o.Now = c.now()
	}
	accounts := c.Accounts()
	if len(accounts) == 0 {
		return nil, nil, ErrNotConfigured
	}
	n := o.Now.UTC()
	start := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Duration(o.Days) * day)
	per, errs := c.gather(ctx, accounts, start, end, nil)
	if err := allFailed(errs); err != nil {
		return nil, errs, err
	}
	return MergeUpcoming(per, o.Limit), errs, nil
}

// UpcomingEvents returns events across every Google calendar, sorted, within
// Days of today. Every account failing is an error, not an empty calendar.
func (c *Connector) UpcomingEvents(ctx context.Context, o UpcomingOptions) ([]CalEvent, error) {
	evs, _, err := c.upcoming(ctx, o)
	return evs, err
}

// BookingOptions: PastDays defaults to 60 and FutureDays to 30 around Now.
type BookingOptions struct {
	Now        time.Time
	PastDays   int
	FutureDays int
}

// CalendarBookings returns calls booked, for the funnel: events in
// [now-PastDays, now+FutureDays) that invite someone. Expanded recurring
// instances carry no attendees, so series drop out. ErrNotConfigured when no
// Google calendar exists, so "no source" never reads as "no bookings".
func (c *Connector) CalendarBookings(ctx context.Context, o BookingOptions) ([]CalEvent, error) {
	if o.Now.IsZero() {
		o.Now = c.now()
	}
	if o.PastDays <= 0 {
		o.PastDays = 60
	}
	if o.FutureDays <= 0 {
		o.FutureDays = 30
	}
	accounts := c.Accounts()
	if len(accounts) == 0 {
		return nil, ErrNotConfigured
	}
	start := o.Now.Add(-time.Duration(o.PastDays) * day)
	end := o.Now.Add(time.Duration(o.FutureDays) * day)
	per, errs := c.gather(ctx, accounts, start, end, func(ev CalEvent) bool { return len(ev.Attendees) > 0 })
	if err := allFailed(errs); err != nil {
		return nil, err
	}
	return MergeUpcoming(per, 500), nil
}

// Status reads the next 14 days (CalDAV REPORT only); it never writes. Unlike
// the TS, which swallowed per-account failures and could report "0 upcoming"
// for an unreachable calendar, every account failing reads as error and a
// partial failure is named in the detail.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	accounts := c.Accounts()
	if len(accounts) == 0 {
		s.State = connectors.StateNotConfigured
		s.Detail = "No Google inboxes configured — calendars reuse the INBOX_* Gmail app passwords."
		s.Meta = map[string]any{"calendars": 0}
		return s
	}
	evs, errs, err := c.upcoming(ctx, UpcomingOptions{Days: 14})
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "CalDAV read failed: " + err.Error()
		s.Meta = map[string]any{"calendars": len(accounts)}
		return s
	}
	plural := ""
	if len(accounts) > 1 {
		plural = "s"
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%d upcoming · %d calendar%s (next 14 days, CalDAV)", len(evs), len(accounts), plural)
	s.Meta = map[string]any{"calendars": len(accounts), "upcoming": len(evs)}
	if n := failing(errs); n > 0 {
		s.Detail += fmt.Sprintf(" · %d failing", n)
		s.Meta["failing"] = n
	}
	return s
}
