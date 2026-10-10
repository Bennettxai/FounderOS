// Package gcalwrite ports FounderOS v1 lib/connectors/gcal-write.ts (and the
// read half of scripts/gcal-add-guests.mjs): Google Calendar writes through
// the real Calendar API with OAuth (GCAL_CLIENT_ID / GCAL_CLIENT_SECRET /
// GCAL_REFRESH_TOKEN, scope calendar.events). The CalDAV read path (package
// gcal) cannot manage attendees: Google's legacy CalDAV accepts an ATTENDEE
// change but does not reliably send the invitation, so the two stay separate.
//
// Safety properties, each pinned by a test:
//  1. sendUpdates defaults to "none"; mailing guests is always deliberate.
//  2. Attendees are merged, never replaced: a PUT-style replace would drop
//     everyone invited, and re-listing an attendee resets their RSVP and
//     re-notifies them. The merge is case-insensitive and keeps rows verbatim.
//  3. The PATCH runs only inside guard.Outbound("gcal.addGuests").
//
// FounderOS v1 has no Connections-board row for this module (lib/connectors/
// index.ts lists only "calendar"), so Meta here is the bridge's own and the
// connector must not be registered on the /integrations board, or the count
// would stop matching FounderOS v1's /api/connections.
package gcalwrite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "gcal-write", Name: "Google Calendar (write)", Kind: connectors.KindCalendar}

const (
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	defaultAPIBase  = "https://www.googleapis.com/calendar/v3"
	// tokenReadPOST allow-lists the token mint on the guarded client: it is a
	// POST, but it changes nothing anyone can see.
	tokenReadPOST = "oauth2.googleapis.com/token"
	timeout       = 20 * time.Second
	isoMillis     = "2006-01-02T15:04:05.000Z"
)

var errNoCreds = errors.New("no Google Calendar credentials (GCAL_CLIENT_ID / GCAL_CLIENT_SECRET / GCAL_REFRESH_TOKEN)")

type Creds struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// Attendee is a row as Google returns it; unknown keys survive a merge.
type Attendee map[string]any

func (a Attendee) email() string { return strings.ToLower(strings.TrimSpace(fmt.Sprint(a["email"]))) }

type EventTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
}

// Event is the subset of a Calendar API event the bridge reads.
type Event struct {
	ID         string     `json:"id"`
	Summary    string     `json:"summary"`
	Start      EventTime  `json:"start"`
	Recurrence []string   `json:"recurrence,omitempty"`
	Attendees  []Attendee `json:"attendees,omitempty"`
}

// Recurring reports whether this is a series parent (what gets patched to
// change "the weekly call").
func (e Event) Recurring() bool { return len(e.Recurrence) > 0 }

func (e Event) StartString() string {
	if e.Start.DateTime != "" {
		return e.Start.DateTime
	}
	return e.Start.Date
}

type AddGuestsRequest struct {
	CalendarID string // default "primary"
	EventID    string
	Emails     []string
	// SendUpdates is "all", "externalOnly" or "none" (the default).
	SendUpdates string
}

type AddGuestsResult struct {
	OK bool `json:"ok"`
	// Added: addresses newly added by this call (by a plan: that would be).
	Added []string `json:"added"`
	// AlreadyPresent: valid addresses already on the invite, left alone.
	AlreadyPresent []string `json:"alreadyPresent"`
	// Rejected: malformed addresses that never reached Google.
	Rejected []string `json:"rejected"`
	Error    string   `json:"error,omitempty"`
}

type Connector struct {
	res      connectors.Resolver
	tokenURL string
	apiBase  string
	now      func() time.Time
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, tokenURL: defaultTokenURL, apiBase: defaultAPIBase, now: time.Now}
}

// Creds returns all three parts or nil: a client id without a refresh token
// cannot mint an access token, so a partial set is not "configured".
func (c *Connector) Creds() *Creds {
	cr := Creds{
		ClientID:     c.res.Resolve("GCAL_CLIENT_ID"),
		ClientSecret: c.res.Resolve("GCAL_CLIENT_SECRET"),
		RefreshToken: c.res.Resolve("GCAL_REFRESH_TOKEN"),
	}
	if cr.ClientID == "" || cr.ClientSecret == "" || cr.RefreshToken == "" {
		return nil
	}
	return &cr
}

// Deliberately strict: a domain must have a dot, so x@y is caught here rather
// than by Google after the invite has partly gone out.
var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@.]+(\.[^\s@.]+)+$`)

// ValidEmails separates sendable addresses (lowercased, deduped) from
// malformed ones, instead of silently dropping them.
func ValidEmails(in []string) (ok, bad []string) {
	seen := map[string]bool{}
	for _, raw := range in {
		e := strings.ToLower(strings.TrimSpace(raw))
		if !emailRE.MatchString(e) {
			bad = append(bad, strings.TrimSpace(raw))
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		ok = append(ok, e)
	}
	return ok, bad
}

// MergeAttendees keeps every existing row untouched and appends only
// genuinely new addresses.
func MergeAttendees(existing []Attendee, add []string) []Attendee {
	have := map[string]bool{}
	for _, a := range existing {
		have[a.email()] = true
	}
	out := append([]Attendee(nil), existing...)
	for _, e := range add {
		k := strings.ToLower(strings.TrimSpace(e))
		if have[k] {
			continue
		}
		have[k] = true
		out = append(out, Attendee{"email": k})
	}
	return out
}

// ---- HTTP -------------------------------------------------------------------

func (c *Connector) client() *http.Client { return connectors.HTTPClient(timeout, tokenReadPOST) }

func (c *Connector) mintToken(ctx context.Context, hc *http.Client, cr *Creds) (string, error) {
	form := url.Values{
		"client_id":     {cr.ClientID},
		"client_secret": {cr.ClientSecret},
		"refresh_token": {cr.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh failed: %w", err)
	}
	defer res.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body)
	if res.StatusCode/100 != 2 || body.AccessToken == "" {
		reason := body.Error
		if reason == "" {
			reason = fmt.Sprint(res.StatusCode)
		}
		return "", fmt.Errorf("token refresh failed: %s", reason)
	}
	return body.AccessToken, nil
}

func (c *Connector) eventsURL(calendarID string) string {
	if calendarID == "" {
		calendarID = "primary"
	}
	return c.apiBase + "/calendars/" + encodeURIComponent(calendarID) + "/events"
}

// googleError pulls {"error":{"message":...}} out of an API error body.
func googleError(r io.Reader) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&body)
	return body.Error.Message
}

func (c *Connector) get(ctx context.Context, hc *http.Client, token, u string, into any) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := hc.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return res.StatusCode, googleError(res.Body), nil
	}
	return res.StatusCode, "", json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(into)
}

func (c *Connector) session(ctx context.Context) (*http.Client, string, error) {
	cr := c.Creds()
	if cr == nil {
		return nil, "", errNoCreds
	}
	hc := c.client()
	token, err := c.mintToken(ctx, hc, cr)
	return hc, token, err
}

// ---- reads ------------------------------------------------------------------

// ListEvents lists up to 50 events updated in the last week onwards, keeping
// recurring parents (singleEvents=false, which is what gets patched), with an
// optional free-text q. As in scripts/gcal-add-guests.mjs --list.
func (c *Connector) ListEvents(ctx context.Context, calendarID, q string) ([]Event, error) {
	hc, token, err := c.session(ctx)
	if err != nil {
		return nil, err
	}
	return c.list(ctx, hc, token, calendarID, q, 50)
}

func (c *Connector) list(ctx context.Context, hc *http.Client, token, calendarID, q string, max int) ([]Event, error) {
	p := url.Values{
		"timeMin":      {c.now().Add(-7 * 24 * time.Hour).UTC().Format(isoMillis)},
		"maxResults":   {fmt.Sprint(max)},
		"singleEvents": {"false"},
		"orderBy":      {"updated"},
	}
	if q != "" {
		p.Set("q", q)
	}
	var body struct {
		Items []Event `json:"items"`
	}
	status, msg, err := c.get(ctx, hc, token, c.eventsURL(calendarID)+"?"+p.Encode(), &body)
	if err != nil {
		return nil, err
	}
	if status/100 != 2 {
		return nil, fmt.Errorf("Calendar API %d: %s", status, msg)
	}
	if body.Items == nil {
		body.Items = []Event{}
	}
	return body.Items, nil
}

// FindRecurringEvent finds the one recurring series whose title matches
// exactly (case-insensitive). Zero or several matches is an error: patch by id
// instead.
func (c *Connector) FindRecurringEvent(ctx context.Context, calendarID, title string) (Event, error) {
	evs, err := c.ListEvents(ctx, calendarID, title)
	if err != nil {
		return Event{}, err
	}
	var matches []Event
	for _, e := range evs {
		if strings.EqualFold(e.Summary, title) && e.Recurring() {
			matches = append(matches, e)
		}
	}
	if len(matches) != 1 {
		return Event{}, fmt.Errorf("find matched %d recurring events for %q; pass the event id instead", len(matches), title)
	}
	return matches[0], nil
}

// plan reads the event and splits the valid addresses into new and present.
func (c *Connector) plan(ctx context.Context, r AddGuestsRequest) (hc *http.Client, token string, existing []Attendee, res AddGuestsResult, err error) {
	valid, rejected := ValidEmails(r.Emails)
	res.Rejected = rejected
	if hc, token, err = c.session(ctx); err != nil {
		return
	}
	var ev Event
	status, _, err := c.get(ctx, hc, token, c.eventsURL(r.CalendarID)+"/"+encodeURIComponent(r.EventID), &ev)
	if err != nil {
		return
	}
	if status/100 != 2 {
		err = fmt.Errorf("event read failed: %d", status)
		return
	}
	existing = ev.Attendees
	have := map[string]bool{}
	for _, a := range existing {
		have[a.email()] = true
	}
	for _, e := range valid {
		if have[e] {
			res.AlreadyPresent = append(res.AlreadyPresent, e)
		} else {
			res.Added = append(res.Added, e)
		}
	}
	return
}

func failed(res AddGuestsResult, err error) (AddGuestsResult, error) {
	return AddGuestsResult{OK: false, Added: []string{}, AlreadyPresent: []string{}, Rejected: orEmpty(res.Rejected), Error: err.Error()}, err
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func finish(res AddGuestsResult) AddGuestsResult {
	res.OK = true
	res.Added, res.AlreadyPresent, res.Rejected = orEmpty(res.Added), orEmpty(res.AlreadyPresent), orEmpty(res.Rejected)
	return res
}

// PlanEventGuests is the dry run: it reads the event and reports who would be
// added, who is already invited, and which addresses are malformed. It never
// writes, whatever FOUNDEROS_WRITES says.
func (c *Connector) PlanEventGuests(ctx context.Context, r AddGuestsRequest) (AddGuestsResult, error) {
	_, _, _, res, err := c.plan(ctx, r)
	if err != nil {
		return failed(res, err)
	}
	return finish(res), nil
}

// ---- write (guarded) --------------------------------------------------------

// AddEventGuests adds guests to an event by merging them into its attendee
// list. Nothing to add is a success with no PATCH (and no re-notify). The
// PATCH runs only through guard.Outbound("gcal.addGuests"): while
// FOUNDEROS_WRITES is off it returns guard.ErrWritesDisabled.
func (c *Connector) AddEventGuests(ctx context.Context, r AddGuestsRequest) (AddGuestsResult, error) {
	sendUpdates := r.SendUpdates
	if sendUpdates == "" {
		sendUpdates = "none"
	}
	if sendUpdates != "all" && sendUpdates != "externalOnly" && sendUpdates != "none" {
		_, rejected := ValidEmails(r.Emails)
		return failed(AddGuestsResult{Rejected: rejected}, fmt.Errorf("invalid sendUpdates %q (all, externalOnly or none)", sendUpdates))
	}
	hc, token, existing, res, err := c.plan(ctx, r)
	if err != nil {
		return failed(res, err)
	}
	if len(res.Added) == 0 {
		return finish(res), nil
	}
	err = guard.Outbound("gcal.addGuests", func() error {
		body, err := json.Marshal(map[string]any{"attendees": MergeAttendees(existing, res.Added)})
		if err != nil {
			return err
		}
		u := c.eventsURL(r.CalendarID) + "/" + encodeURIComponent(r.EventID) + "?sendUpdates=" + url.QueryEscape(sendUpdates)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("patch failed (%d): %s", resp.StatusCode, googleError(resp.Body))
		}
		return nil
	})
	if err != nil {
		return failed(res, err)
	}
	return finish(res), nil
}

// ---- status -----------------------------------------------------------------

// Status mints an access token and lists one event: proof the OAuth grant and
// the API both work. It never writes (the token POST is allow-listed as a
// read) and says whether the bridge would currently let a write through.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if c.Creds() == nil {
		s.State = connectors.StateNotConfigured
		s.Detail = "No Google Calendar write credentials. Set GCAL_CLIENT_ID / GCAL_CLIENT_SECRET / GCAL_REFRESH_TOKEN (node scripts/gcal-auth.mjs mints the refresh token)."
		return s
	}
	hc, token, err := c.session(ctx)
	if err == nil {
		_, err = c.list(ctx, hc, token, "primary", "", 1)
	}
	if err != nil {
		s.State, s.Detail = connectors.StateError, err.Error()
		return s
	}
	s.State = connectors.StateConnected
	s.Detail = "Calendar API · OAuth ok (calendar.events)"
	if !guard.WritesEnabled() {
		s.Detail += " · writes off (FOUNDEROS_WRITES=0)"
	}
	s.Meta = map[string]any{"writesEnabled": guard.WritesEnabled()}
	return s
}

// encodeURIComponent escapes a path segment the way the TS does, so a
// calendar id such as team@group.calendar.google.com travels as %40.
func encodeURIComponent(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }
