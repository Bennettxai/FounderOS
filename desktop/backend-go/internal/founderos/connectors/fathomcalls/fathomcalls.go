// Package fathomcalls ports FounderOS v1's lib/connectors/fathom.ts: Fathom the
// MEETING RECORDER (the AI notetaker on the operator's sales calls). It is a
// different provider from BusinessOS's Fathom Analytics integration, hence
// the package name; that integration is untouched.
//
// Real API (developers.fathom.ai):
//
//	GET https://api.fathom.ai/external/v1/meetings   header X-Api-Key
//
// Keys are per user and see the meetings you recorded plus anything shared
// to your team. The rate limit is 60 calls/minute across a user's keys, so
// callers should cache rather than poll. Fathom is read-only here: nothing in
// this package sends anything but GET.
//
// Besides the connector's own reads it carries the Fathom half of
// lib/call-archive.ts: the full meeting pages (transcript, summary, action
// items), the demo-call check and the transcript line merge.
package fathomcalls

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "fathom", Name: "Fathom", Kind: connectors.KindCRM}

const (
	keyName = "FATHOM_API_KEY"
	// API is the Fathom external v1 base URL.
	API = "https://api.fathom.ai/external/v1"

	listTimeout    = 6 * time.Second
	archiveTimeout = 60 * time.Second
	// max429Retries bounds the call-archive's retry on rate limiting (the TS
	// retries without a bound).
	max429Retries = 5
)

// ErrNotConfigured is returned by reads when no key resolves.
var ErrNotConfigured = errors.New("fathom: FATHOM_API_KEY not configured")

type Connector struct {
	res connectors.Resolver
	// BaseURL overrides API (tests).
	BaseURL string
	// CredFiles are the fallback env files, in order. nil means FounderOS v1's
	// order for this key: clue-agent/.env.agents, then ~/.social-media/.env.
	CredFiles []string
	// RetryDelay is the wait after a 429 on the archive pull (default 2s).
	RetryDelay time.Duration
	client     *http.Client
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, BaseURL: API, RetryDelay: 2 * time.Second, client: connectors.HTTPClient(archiveTimeout)}
}

func (c *Connector) key() string {
	files := c.CredFiles
	if files == nil {
		social, clue, _, _ := connectors.CredFiles()
		files = []string{clue, social}
	}
	return c.res.Resolve(keyName, files...)
}

type httpStatusError struct {
	code int
	path string
}

func (e *httpStatusError) Error() string { return fmt.Sprintf("HTTP %d", e.code) }

func (c *Connector) get(ctx context.Context, key, pathAndQuery string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+pathAndQuery, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", key)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, &httpStatusError{code: res.StatusCode, path: req.URL.Path}
	}
	return io.ReadAll(io.LimitReader(res.Body, 32<<20))
}

func decodeItems(body []byte) ([]map[string]any, map[string]any) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var doc map[string]any
	if dec.Decode(&doc) != nil || doc == nil {
		return nil, nil
	}
	arr, ok := doc["items"].([]any)
	if !ok {
		return nil, doc
	}
	out := make([]map[string]any, 0, len(arr))
	for _, raw := range arr {
		m, _ := raw.(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		out = append(out, m)
	}
	return out, doc
}

func number(v any) (float64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil
}

func titleOf(m map[string]any) string {
	for _, k := range []string{"meeting_title", "title"} {
		switch v := m[k].(type) {
		case string:
			return v
		case json.Number:
			return v.String()
		}
	}
	return "Untitled meeting"
}

func nonEmpty(v any) *string {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return &s
}

// ---- the flat list (/comms Recordings) --------------------------------------

// Meeting is one recorded call on the Recordings tab.
type Meeting struct {
	Title           string   `json:"title"`
	URL             *string  `json:"url"`
	At              string   `json:"at"` // ISO
	DurationMinutes *float64 `json:"durationMinutes"`
}

// ParseMeetings maps the list payload to flat rows. A shape it does not
// recognise is an empty list, as in the TS.
func ParseMeetings(body []byte) []Meeting {
	items, _ := decodeItems(body)
	out := make([]Meeting, 0, len(items))
	for _, m := range items {
		row := Meeting{Title: titleOf(m)}
		if s, ok := m["url"].(string); ok {
			row.URL = &s
		}
		if s, ok := m["created_at"].(string); ok {
			row.At = s
		}
		if d, ok := number(m["recording_duration_in_minutes"]); ok {
			row.DurationMinutes = &d
		}
		out = append(out, row)
	}
	return out
}

func (c *Connector) getMeetings(ctx context.Context, key string, limit int) ([]Meeting, error) {
	body, err := c.get(ctx, key, "/meetings", listTimeout)
	if err != nil {
		return nil, err
	}
	rows := ParseMeetings(body)
	if limit >= 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

// RecentMeetings returns up to limit recorded calls from the first page.
// Unlike the TS (which reads [] on failure), a failure is an error here.
func (c *Connector) RecentMeetings(ctx context.Context, limit int) ([]Meeting, error) {
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	return c.getMeetings(ctx, key, limit)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	key := c.key()
	if key == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = "Set FATHOM_API_KEY in ~/.founderos/.env or under API keys to pull recorded calls, durations and transcripts. Create a key in Fathom under Settings → API."
		return s
	}
	meetings, err := c.getMeetings(ctx, key, 100)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "FATHOM_API_KEY is set but the call failed: " + err.Error()
		return s
	}
	plural := "s"
	if len(meetings) == 1 {
		plural = ""
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("Fathom reachable · %d recorded meeting%s visible to this key", len(meetings), plural)
	s.Meta = map[string]any{"meetings": len(meetings)}
	return s
}

// ---- held calls (the funnel's "calls had" lane) -----------------------------

// Invitee is one calendar invitee on a held call.
type Invitee struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	External bool    `json:"external"`
}

// Call is a held call with who was invited, which the funnel joins on.
type Call struct {
	RecordingID     string    `json:"recordingId"`
	Title           string    `json:"title"`
	URL             *string   `json:"url"`
	At              string    `json:"at"` // recording start, else creation
	ScheduledAt     *string   `json:"scheduledAt"`
	DurationMinutes *float64  `json:"durationMinutes"`
	Invitees        []Invitee `json:"invitees"`
}

func parseTime(v any) (time.Time, bool) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

func jsRound(f float64) float64 { return math.Floor(f + 0.5) }

// ParseCalls maps one list page to held calls and returns its next_cursor.
// A row with no recording id or no time is skipped, never guessed at.
func ParseCalls(body []byte) ([]Call, string) {
	items, doc := decodeItems(body)
	next, _ := doc["next_cursor"].(string)
	out := []Call{}
	for _, m := range items {
		var id string
		switch v := m["recording_id"].(type) {
		case json.Number:
			id = v.String()
		case string:
			id = v
		}
		if id == "" {
			continue
		}
		at := nonEmpty(m["recording_start_time"])
		if at == nil {
			at = nonEmpty(m["created_at"])
		}
		if at == nil {
			continue
		}
		call := Call{RecordingID: id, Title: titleOf(m), At: *at, ScheduledAt: nonEmpty(m["scheduled_start_time"]), Invitees: []Invitee{}}
		if s, ok := m["url"].(string); ok {
			call.URL = &s
		}
		if d, ok := number(m["recording_duration_in_minutes"]); ok {
			call.DurationMinutes = &d
		} else if start, ok1 := parseTime(m["recording_start_time"]); ok1 {
			if end, ok2 := parseTime(m["recording_end_time"]); ok2 && end.After(start) {
				d := jsRound(end.Sub(start).Minutes())
				call.DurationMinutes = &d
			}
		}
		invitees, _ := m["calendar_invitees"].([]any)
		for _, raw := range invitees {
			i, _ := raw.(map[string]any)
			var inv Invitee
			if s, ok := i["email"].(string); ok && strings.TrimSpace(s) != "" {
				e := strings.ToLower(strings.TrimSpace(s))
				inv.Email = &e
			}
			if s, ok := i["name"].(string); ok && strings.TrimSpace(s) != "" {
				n := strings.TrimSpace(s)
				inv.Name = &n
			}
			inv.External = i["is_external"] == true
			call.Invitees = append(call.Invitees, inv)
		}
		out = append(out, call)
	}
	return out, next
}

// CallsOptions bounds a Calls pull. Zero values mean now, 90 days, 3 pages.
type CallsOptions struct {
	Now      time.Time
	Days     int
	MaxPages int
}

// Calls returns held calls created in the last Days, following next_cursor
// for up to MaxPages pages (60 calls/minute is the per-user budget).
// No key is ErrNotConfigured and a refused first page is an error, so "no
// source" never reads as "no calls". If a later page fails, the calls already
// read come back together with the error.
func (c *Connector) Calls(ctx context.Context, opts CallsOptions) ([]Call, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.Days <= 0 {
		opts.Days = 90
	}
	if opts.MaxPages <= 0 {
		opts.MaxPages = 3
	}
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	after := opts.Now.Add(-time.Duration(opts.Days) * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
	var calls []Call
	cursor := ""
	for page := 0; page < opts.MaxPages; page++ {
		q := url.Values{"created_after": {after}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		body, err := c.get(ctx, key, "/meetings?"+q.Encode(), listTimeout)
		if err != nil {
			if len(calls) > 0 {
				return calls, err
			}
			return nil, err
		}
		pageCalls, next := ParseCalls(body)
		calls = append(calls, pageCalls...)
		if next == "" {
			break
		}
		cursor = next
	}
	if calls == nil {
		calls = []Call{}
	}
	return calls, nil
}

// ---- full meetings (lib/call-archive.ts) -------------------------------------

// FlexID is Fathom's recording_id, which arrives as a number or a string.
type FlexID string

func (f *FlexID) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		*f = FlexID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = FlexID(n.String())
	return nil
}

type Person struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type Segment struct {
	Speaker *struct {
		DisplayName *string `json:"display_name"`
	} `json:"speaker"`
	Text      string  `json:"text"`
	Timestamp *string `json:"timestamp"`
}

type Summary struct {
	TemplateName      string `json:"template_name"`
	MarkdownFormatted string `json:"markdown_formatted"`
}

type ActionItem struct {
	Description        string  `json:"description"`
	Completed          bool    `json:"completed"`
	RecordingTimestamp string  `json:"recording_timestamp"`
	Assignee           *Person `json:"assignee"`
}

// MeetingFull is one meeting with transcript, summary and action items
// inline (include_transcript/include_summary/include_action_items).
type MeetingFull struct {
	TitleRaw           *string      `json:"title"`
	MeetingTitle       *string      `json:"meeting_title"`
	URL                string       `json:"url"`
	RecordingID        FlexID       `json:"recording_id"`
	CreatedAt          string       `json:"created_at"`
	RecordingStartTime string       `json:"recording_start_time"`
	RecordingEndTime   string       `json:"recording_end_time"`
	RecordedBy         *Person      `json:"recorded_by"`
	CalendarInvitees   []Invitee    `json:"calendar_invitees"`
	Transcript         []Segment    `json:"transcript"`
	DefaultSummary     *Summary     `json:"default_summary"`
	ActionItems        []ActionItem `json:"action_items"`
}

var callsURLRe = regexp.MustCompile(`/calls/(\d+)`)

// ID is the recording id, else the number in a /calls/<id> URL, else "".
func (m MeetingFull) ID() string {
	if m.RecordingID != "" {
		return string(m.RecordingID)
	}
	if sub := callsURLRe.FindStringSubmatch(m.URL); sub != nil {
		return sub[1]
	}
	return ""
}

// Title is the trimmed meeting title, else "Untitled meeting".
func (m MeetingFull) Title() string {
	var t string
	switch {
	case m.MeetingTitle != nil:
		t = *m.MeetingTitle
	case m.TitleRaw != nil:
		t = *m.TitleRaw
	}
	if t = strings.TrimSpace(t); t == "" {
		return "Untitled meeting"
	}
	return t
}

// At is the recording start, else the creation time.
func (m MeetingFull) At() string {
	if m.RecordingStartTime != "" {
		return m.RecordingStartTime
	}
	return m.CreatedAt
}

// DurationMinutes is the rounded recording length, nil when unknown or a day
// or longer (Fathom's own demo call reports an end time years out).
func (m MeetingFull) DurationMinutes() *int {
	a, ok1 := parseTime(m.RecordingStartTime)
	b, ok2 := parseTime(m.RecordingEndTime)
	if !ok1 || !ok2 || !b.After(a) || b.Sub(a) >= 24*time.Hour {
		return nil
	}
	d := int(jsRound(b.Sub(a).Minutes()))
	return &d
}

// HasTranscript reports whether any transcript segment carries text.
func (m MeetingFull) HasTranscript() bool {
	for _, s := range m.Transcript {
		if strings.TrimSpace(s.Text) != "" {
			return true
		}
	}
	return false
}

// IsSample reports Fathom's bundled demo call ("Fathom Demo", or any
// fathom.video invitee), which the archive skips.
func IsSample(m MeetingFull) bool {
	title := ""
	if m.MeetingTitle != nil {
		title = *m.MeetingTitle
	} else if m.TitleRaw != nil {
		title = *m.TitleRaw
	}
	if strings.ToLower(strings.TrimSpace(title)) == "fathom demo" {
		return true
	}
	for _, i := range m.CalendarInvitees {
		if i.Email != nil && strings.HasSuffix(strings.ToLower(*i.Email), "@fathom.video") {
			return true
		}
	}
	return false
}

// TranscriptLines renders segments as "[HH:MM:SS] Speaker: text", merging
// consecutive runs of one speaker.
func TranscriptLines(m MeetingFull) []string {
	lines := []string{}
	var speaker, at string
	var parts []string
	flush := func() {
		if parts != nil {
			lines = append(lines, fmt.Sprintf("[%s] %s: %s", at, speaker, strings.Join(parts, " ")))
		}
	}
	for _, s := range m.Transcript {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		who := "Speaker"
		if s.Speaker != nil && s.Speaker.DisplayName != nil && strings.TrimSpace(*s.Speaker.DisplayName) != "" {
			who = strings.TrimSpace(*s.Speaker.DisplayName)
		}
		if parts != nil && who == speaker {
			parts = append(parts, text)
			continue
		}
		flush()
		speaker, parts, at = who, []string{text}, "00:00:00"
		if s.Timestamp != nil {
			at = *s.Timestamp
		}
	}
	flush()
	return lines
}

func (c *Connector) getWithRetry(ctx context.Context, key, pathAndQuery string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		body, err := c.get(ctx, key, pathAndQuery, archiveTimeout)
		var hs *httpStatusError
		if errors.As(err, &hs) && hs.code == http.StatusTooManyRequests && attempt < max429Retries {
			select {
			case <-time.After(c.RetryDelay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if hs != nil {
			return nil, fmt.Errorf("HTTP %d %s", hs.code, hs.path)
		}
		return body, err
	}
}

// MeetingsFull walks every meeting page (100 per page, transcript, summary
// and action items inline) and hands each page to visit. A 429 waits
// RetryDelay and retries; any other failure stops the walk with an error.
func (c *Connector) MeetingsFull(ctx context.Context, visit func(page []MeetingFull) error) error {
	key := c.key()
	if key == "" {
		return ErrNotConfigured
	}
	cursor := ""
	for {
		q := url.Values{"include_transcript": {"true"}, "include_summary": {"true"}, "include_action_items": {"true"}, "limit": {"100"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		body, err := c.getWithRetry(ctx, key, "/meetings?"+q.Encode())
		if err != nil {
			return err
		}
		var page struct {
			Items      []MeetingFull `json:"items"`
			NextCursor *string       `json:"next_cursor"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return fmt.Errorf("unreadable /meetings page: %w", err)
		}
		if err := visit(page.Items); err != nil {
			return err
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			return nil
		}
		cursor = *page.NextCursor
	}
}
