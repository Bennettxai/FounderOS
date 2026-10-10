// Package trakyo ports FounderOS v1's Trakyo connector: organic attribution
// (content → clicks → leads → booked calls → revenue). The operator uses it daily.
//
// Sources ported, all against Trakyo API v1 (https://api.trakyo.io/v1,
// Authorization: Bearer tky_live_…):
//
//	lib/connectors/trakyo.ts   GET /metrics, the Connections status
//	lib/funnel-trakyo.ts       GET /leads → first-touch events
//	lib/funnel-analytics.ts    GET /content (paged) + GET /funnel, channels
//	lib/trakyo-links.ts        GET /keys/self, /domains, /links; POST /content
//	                           and POST /links (guarded writes)
//
// Keys carry scopes; a valid key without an endpoint's scope answers 403,
// which surfaces as an honest error, never a fake "connected".
package trakyo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "trakyo", Name: "Trakyo", Kind: connectors.KindCRM}

const (
	keyName = "TRAKYO_API_KEY"
	// API is the Trakyo v1 base URL.
	API = "https://api.trakyo.io/v1"

	statusTimeout    = 6 * time.Second
	analyticsTimeout = 8 * time.Second
	linksTimeout     = 10 * time.Second
	// leadLimit is how many leads one funnel render pulls (the recent cohort).
	leadLimit = 100
)

// ErrNotConfigured is returned by reads when no key resolves.
var ErrNotConfigured = errors.New("trakyo: TRAKYO_API_KEY not configured")

type Connector struct {
	res connectors.Resolver
	// BaseURL overrides API (tests).
	BaseURL string
	// CredFiles are the fallback env files, in order. nil means FounderOS v1's
	// order for this key: clue-agent/.env.agents, then ~/.social-media/.env.
	CredFiles []string
	// Now stamps Analytics.FetchedAt (tests).
	Now    func() time.Time
	client *http.Client
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, BaseURL: API, Now: time.Now, client: connectors.HTTPClient(linksTimeout)}
}

func (c *Connector) key() string {
	files := c.CredFiles
	if files == nil {
		social, clue, _, _ := connectors.CredFiles()
		files = []string{clue, social}
	}
	return c.res.Resolve(keyName, files...)
}

func (c *Connector) url(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// get performs an authenticated GET and returns the body of a 2xx answer.
// A non-2xx answer is an error reading "HTTP <status>".
func (c *Connector) get(ctx context.Context, key, path string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 32<<20))
}

// ---- metrics + status (lib/connectors/trakyo.ts) ---------------------------

// Metrics are the account-wide totals for Trakyo's default window.
type Metrics struct {
	Clicks          float64 `json:"clicks"`
	Visits          float64 `json:"visits"`
	FormSubmissions float64 `json:"formSubmissions"`
	Bookings        float64 `json:"bookings"`
	Transactions    float64 `json:"transactions"`
	RevenueUSD      float64 `json:"revenueUsd"`
	RangeStart      string  `json:"rangeStart"`
	RangeEnd        string  `json:"rangeEnd"`
}

// num coerces Trakyo's wire numbers: money arrives as a decimal string
// ("4500.50") and counters that are zero for the range are omitted.
func num(v any) float64 {
	switch x := v.(type) {
	case float64:
		if !math.IsInf(x, 0) && !math.IsNaN(x) {
			return x
		}
	case string:
		if strings.TrimSpace(x) == "" {
			return 0 // Number("") is 0 in the TS
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil && !math.IsInf(f, 0) {
			return f
		}
	}
	return 0
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// ParseMetrics maps a GET /metrics envelope. ok is false when there is no
// usable totals object: callers treat that as an error, never as zeros.
func ParseMetrics(body []byte) (Metrics, bool) {
	var doc map[string]any
	if json.Unmarshal(body, &doc) != nil || doc == nil {
		return Metrics{}, false
	}
	t, ok := doc["totals"].(map[string]any)
	if !ok {
		return Metrics{}, false
	}
	rng, _ := doc["range"].(map[string]any)
	return Metrics{
		Clicks:          num(t["clicks"]),
		Visits:          num(t["visits"]),
		FormSubmissions: num(t["form_submissions"]),
		Bookings:        num(t["bookings"]),
		Transactions:    num(t["transactions"]),
		RevenueUSD:      num(t["revenue"]),
		RangeStart:      str(rng["start"]),
		RangeEnd:        str(rng["end"]),
	}, true
}

// Metrics reads account-wide totals. The error carries the real reason.
func (c *Connector) Metrics(ctx context.Context) (Metrics, error) {
	key := c.key()
	if key == "" {
		return Metrics{}, ErrNotConfigured
	}
	body, err := c.get(ctx, key, "metrics", statusTimeout)
	if err != nil {
		return Metrics{}, err
	}
	m, ok := ParseMetrics(body)
	if !ok {
		return Metrics{}, errors.New("metrics response carried no totals")
	}
	return m, nil
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if c.key() == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = "Organic attribution (content → leads → booked calls → revenue). Set TRAKYO_API_KEY."
		return s
	}
	m, err := c.Metrics(ctx)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "TRAKYO_API_KEY set but the Trakyo API call failed: " + err.Error()
		return s
	}
	money := ""
	if m.RevenueUSD > 0 {
		money = " · $" + formatEnUS(m.RevenueUSD) + " revenue"
	}
	window := ""
	if m.RangeStart != "" && m.RangeEnd != "" {
		window = fmt.Sprintf(" (%s → %s)", m.RangeStart, m.RangeEnd)
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%s clicks · %s visits · %s form submissions%s%s",
		jsNum(m.Clicks), jsNum(m.Visits), jsNum(m.FormSubmissions), money, window)
	s.Meta = map[string]any{
		"clicks":          m.Clicks,
		"visits":          m.Visits,
		"formSubmissions": m.FormSubmissions,
		"bookings":        m.Bookings,
		"revenueUsd":      m.RevenueUSD,
	}
	return s
}

// jsNum prints a number the way a JS template literal does.
func jsNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// formatEnUS mirrors Number.toLocaleString('en-US'): thousands separators
// and at most three fraction digits.
func formatEnUS(f float64) string {
	neg := f < 0
	s := strconv.FormatFloat(math.Abs(f), 'f', 3, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

// ---- leads (lib/funnel-trakyo.ts) -------------------------------------------

// Event is one attributed first touch as Trakyo reports it.
type Event struct {
	Lead       string   `json:"lead"`   // first name, else first email
	Names      []string `json:"names"`  // every name Trakyo captured
	Emails     []string `json:"emails"` // every email Trakyo captured
	Label      string   `json:"label"`  // the content piece
	Channel    string   `json:"channel"`
	At         string   `json:"at"`         // YYYY-MM-DD
	SourceType string   `json:"sourceType"` // youtube | custom | meta_ad | referrer | other
	SourceName *string  `json:"sourceName"`
}

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func nonBlankStrings(v any) []string {
	arr, _ := v.([]any)
	out := []string{}
	for _, x := range arr {
		if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstNonBlank(vals ...any) (string, bool) {
	for _, v := range vals {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s, true
		}
	}
	return "", false
}

// MapLeads maps GET /leads into first-touch events. A lead is usable only
// when Trakyo captured an identity and an attributed first touch; anything
// else is skipped rather than guessed at. meta_ad first touches are ads,
// everything else organic.
func MapLeads(body []byte) []Event {
	var doc struct {
		Data []any `json:"data"`
	}
	out := []Event{}
	if json.Unmarshal(body, &doc) != nil {
		return out
	}
	for _, rec := range doc.Data {
		r, ok := rec.(map[string]any)
		if !ok {
			continue
		}
		t, ok := r["first_touch"].(map[string]any)
		if !ok {
			continue
		}
		ids, _ := r["identifiers"].(map[string]any)
		names := nonBlankStrings(ids["names"])
		emails := nonBlankStrings(ids["emails"])
		var lead string
		switch {
		case len(names) > 0:
			lead = names[0]
		case len(emails) > 0:
			lead = emails[0]
		default:
			continue
		}
		content, _ := t["content_item"].(map[string]any)
		label, ok := firstNonBlank(content["name"], t["name"], t["source_name"])
		if !ok {
			continue
		}
		at := str(t["occurred_at"])
		if len(at) > 10 {
			at = at[:10]
		}
		if !isoDay.MatchString(at) {
			continue
		}
		sourceType := "other"
		if s, ok := t["type"].(string); ok {
			sourceType = s
		}
		var sourceName *string
		if s, ok := t["source_name"].(string); ok && strings.TrimSpace(s) != "" {
			sourceName = &s
		}
		channel := "organic"
		if sourceType == "meta_ad" {
			channel = "ads"
		}
		out = append(out, Event{Lead: lead, Names: names, Emails: emails, Label: label, Channel: channel, At: at, SourceType: sourceType, SourceName: sourceName})
	}
	return out
}

// Leads pulls the recent lead cohort (GET /leads?limit=100) as first-touch
// events. The TS degrades to [] on failure; here a failure is an error so an
// outage never reads as "no attributed leads".
func (c *Connector) Leads(ctx context.Context) ([]Event, error) {
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	body, err := c.get(ctx, key, fmt.Sprintf("leads?limit=%d", leadLimit), statusTimeout)
	if err != nil {
		return nil, err
	}
	return MapLeads(body), nil
}
