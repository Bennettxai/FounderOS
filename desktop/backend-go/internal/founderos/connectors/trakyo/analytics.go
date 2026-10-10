package trakyo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---- strict wire validation (lib/schemas.ts Trakyo*Schema) -----------------
// Missing metrics are never coerced to zero: a row that does not match the
// schema fails the whole page closed.

type fields map[string]json.RawMessage

func objectOf(raw json.RawMessage) (fields, error) {
	var f fields
	if err := json.Unmarshal(raw, &f); err != nil || f == nil {
		return nil, errors.New("not an object")
	}
	return f, nil
}

func isNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

func (f fields) str(k string) (string, error) {
	raw, ok := f[k]
	if !ok {
		return "", fmt.Errorf("%s missing", k)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%s is not a string", k)
	}
	return s, nil
}

// nullableStr requires the key to be present (zod .nullable() rejects undefined).
func (f fields) nullableStr(k string) (*string, error) {
	raw, ok := f[k]
	if !ok {
		return nil, fmt.Errorf("%s missing", k)
	}
	if isNull(raw) {
		return nil, nil
	}
	s, err := f.str(k)
	return &s, err
}

func (f fields) count(k string) (int64, error) {
	raw, ok := f[k]
	if !ok {
		return 0, fmt.Errorf("%s missing", k)
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err != nil || n < 0 || n != math.Trunc(n) || n > math.MaxInt64 {
		return 0, fmt.Errorf("%s is not a non-negative integer", k)
	}
	return int64(n), nil
}

func (f fields) nullableCount(k string) (*int64, error) {
	raw, ok := f[k]
	if !ok {
		return nil, fmt.Errorf("%s missing", k)
	}
	if isNull(raw) {
		return nil, nil
	}
	n, err := f.count(k)
	return &n, err
}

var moneyRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

func (f fields) money(k string) (string, error) {
	s, err := f.str(k)
	if err != nil {
		return "", err
	}
	if !moneyRe.MatchString(s) {
		return "", fmt.Errorf("%s is not a decimal amount", k)
	}
	if v, err := strconv.ParseFloat(s, 64); err != nil || math.IsInf(v, 0) {
		return "", fmt.Errorf("%s is not finite", k)
	}
	return s, nil
}

func (f fields) literal(k, want string) error {
	s, err := f.str(k)
	if err != nil || s != want {
		return fmt.Errorf("%s must be %q", k, want)
	}
	return nil
}

func (f fields) sub(k string) (fields, error) {
	raw, ok := f[k]
	if !ok {
		return nil, fmt.Errorf("%s missing", k)
	}
	return objectOf(raw)
}

func (f fields) boolean(k string) (bool, error) {
	var b bool
	raw, ok := f[k]
	if !ok || json.Unmarshal(raw, &b) != nil || isNull(raw) {
		return false, fmt.Errorf("%s is not a boolean", k)
	}
	return b, nil
}

// firstErr returns the first non-nil error.
func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// ---- content + funnel -------------------------------------------------------

// Revenue keeps first- and last-touch attribution separate, as decimal strings.
type Revenue struct {
	FirstTouch string `json:"first_touch"`
	LastTouch  string `json:"last_touch"`
}

// Content is one tracked content item (GET /content row).
type Content struct {
	ID                  *string `json:"id"`
	Type                string  `json:"type"`
	Name                string  `json:"name"`
	Source              string  `json:"source"`
	Views               *int64  `json:"views"`
	TrackedSince        *string `json:"tracked_since"`
	ViewsAtTrackedSince *int64  `json:"views_at_tracked_since"`
	PublishedAt         *string `json:"published_at"`
	Clicks              int64   `json:"clicks"`
	Visits              int64   `json:"visits"`
	FormSubmissions     int64   `json:"form_submissions"`
	Bookings            int64   `json:"bookings"`
	Revenue             Revenue `json:"revenue"`
}

var contentTypes = map[string]bool{"youtube": true, "custom": true, "meta_ad": true, "referrer": true, "other": true}

func parseContent(raw json.RawMessage) (Content, error) {
	f, err := objectOf(raw)
	if err != nil {
		return Content{}, err
	}
	var c Content
	var errs [13]error
	c.ID, errs[0] = f.nullableStr("id")
	c.Type, errs[1] = f.str("type")
	if errs[1] == nil && !contentTypes[c.Type] {
		errs[1] = fmt.Errorf("type %q unknown", c.Type)
	}
	c.Name, errs[2] = f.str("name")
	c.Source, errs[3] = f.str("source")
	c.Views, errs[4] = f.nullableCount("views")
	c.TrackedSince, errs[5] = f.nullableStr("tracked_since")
	c.ViewsAtTrackedSince, errs[6] = f.nullableCount("views_at_tracked_since")
	c.PublishedAt, errs[7] = f.nullableStr("published_at")
	c.Clicks, errs[8] = f.count("clicks")
	c.Visits, errs[9] = f.count("visits")
	c.FormSubmissions, errs[10] = f.count("form_submissions")
	c.Bookings, errs[11] = f.count("bookings")
	rev, err := f.sub("revenue")
	if err == nil {
		c.Revenue.FirstTouch, err = rev.money("first_touch")
		if err == nil {
			c.Revenue.LastTouch, err = rev.money("last_touch")
		}
	}
	errs[12] = err
	return c, firstErr(errs[:]...)
}

type contentPage struct {
	Data    []Content
	HasMore bool
}

func parseContentPage(body []byte) (contentPage, error) {
	f, err := objectOf(body)
	if err != nil {
		return contentPage{}, err
	}
	if err := firstErr(f.literal("object", "list"), f.literal("currency", "USD")); err != nil {
		return contentPage{}, err
	}
	var page contentPage
	if page.HasMore, err = f.boolean("has_more"); err != nil {
		return contentPage{}, err
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(f["data"], &rows); err != nil || rows == nil {
		return contentPage{}, errors.New("data is not an array")
	}
	page.Data = make([]Content, 0, len(rows))
	for _, r := range rows {
		c, err := parseContent(r)
		if err != nil {
			return contentPage{}, err
		}
		page.Data = append(page.Data, c)
	}
	return page, nil
}

// FunnelCounts are the funnel's stage counts.
type FunnelCounts struct {
	Clicks          int64 `json:"clicks"`
	Visits          int64 `json:"visits"`
	FormSubmissions int64 `json:"form_submissions"`
	Bookings        int64 `json:"bookings"`
	Closes          int64 `json:"closes"`
}

// Funnel is the GET /funnel summary (first-touch, USD).
type Funnel struct {
	Currency    string `json:"currency"`
	Attribution string `json:"attribution"`
	Range       struct {
		Start    string `json:"start"`
		End      string `json:"end"`
		Timezone string `json:"timezone"`
	} `json:"range"`
	Counts  FunnelCounts `json:"counts"`
	Revenue string       `json:"revenue"`
}

func parseFunnel(body []byte) (Funnel, error) {
	f, err := objectOf(body)
	if err != nil {
		return Funnel{}, err
	}
	if err := firstErr(f.literal("object", "funnel"), f.literal("currency", "USD"), f.literal("attribution", "first_touch")); err != nil {
		return Funnel{}, err
	}
	out := Funnel{Currency: "USD", Attribution: "first_touch"}
	rng, err := f.sub("range")
	if err != nil {
		return Funnel{}, err
	}
	var e [9]error
	out.Range.Start, e[0] = rng.str("start")
	out.Range.End, e[1] = rng.str("end")
	out.Range.Timezone, e[2] = rng.str("timezone")
	counts, err := f.sub("counts")
	if err != nil {
		return Funnel{}, err
	}
	out.Counts.Clicks, e[3] = counts.count("clicks")
	out.Counts.Visits, e[4] = counts.count("visits")
	out.Counts.FormSubmissions, e[5] = counts.count("form_submissions")
	out.Counts.Bookings, e[6] = counts.count("bookings")
	out.Counts.Closes, e[7] = counts.count("closes")
	out.Revenue, e[8] = f.money("revenue")
	return out, firstErr(e[:]...)
}

// ---- analytics (lib/funnel-analytics.ts getTrakyoAnalytics) ----------------

// Section states. not_configured only when no key resolves.
const (
	StateReady         = "ready"
	StatePartial       = "partial"
	StateError         = "error"
	StateNotConfigured = "not_configured"
)

var periods = map[string]bool{"7d": true, "30d": true, "90d": true}

type ContentSection struct {
	State   string    `json:"state"`
	Rows    []Content `json:"rows"`
	Message *string   `json:"message"`
}

type FunnelSection struct {
	State   string  `json:"state"`
	Data    *Funnel `json:"data"`
	Message *string `json:"message"`
}

// Analytics is account-wide (the API has no venture filter). Content is
// paginated independently of the funnel summary; a failure in either stays
// visible in its own section.
type Analytics struct {
	Period    string         `json:"period"`
	FetchedAt string         `json:"fetchedAt"`
	Content   ContentSection `json:"content"`
	Funnel    FunnelSection  `json:"funnel"`
}

const maxContentPages = 10

var (
	httpErrRe    = regexp.MustCompile(`^Trakyo HTTP \d+$`)
	payloadErrRe = regexp.MustCompile(`^Invalid (content|funnel) payload$|^Pagination stalled$`)
)

// analyticsMessage names the real failure (lib/funnel-analytics.ts,
// 2026-09-24): the HTTP status, a transport failure (no route / timeout), or
// a payload that fails validation. Never one catch-all that hides which.
func analyticsMessage(err error) *string {
	msg := "Trakyo data unavailable or invalid. Try refreshing."
	var ue *url.Error
	var ne net.Error
	switch {
	case err == nil:
	case httpErrRe.MatchString(err.Error()):
		msg = err.Error()
	case payloadErrRe.MatchString(err.Error()):
		msg = "Trakyo answered with a payload this OS does not recognise."
	case errors.As(err, &ue), errors.As(err, &ne), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		msg = "Trakyo API unreachable from this machine (network or timeout). Try refreshing."
	}
	return &msg
}

// Analytics reads content attribution (up to 10 pages of 100, by revenue)
// and the funnel summary for one period: 7d, 30d or 90d.
func (c *Connector) Analytics(ctx context.Context, period string) Analytics {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	out := Analytics{
		Period:    period,
		FetchedAt: now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Content:   ContentSection{State: StateNotConfigured, Rows: []Content{}},
		Funnel:    FunnelSection{State: StateNotConfigured},
	}
	key := c.key()
	if key == "" {
		return out
	}
	if !periods[period] {
		msg := fmt.Sprintf("Unknown period %q: use 7d, 30d or 90d.", period)
		out.Content.State, out.Content.Message = StateError, &msg
		out.Funnel.State, out.Funnel.Message = StateError, &msg
		return out
	}
	request := func(endpoint string, extra map[string]string) ([]byte, error) {
		q := url.Values{"period": {period}, "currency": {"USD"}, "attribution": {"first_touch"}, "timezone": {"America/Chicago"}}
		for k, v := range extra {
			q.Set(k, v)
		}
		body, err := c.get(ctx, key, endpoint+"?"+q.Encode(), analyticsTimeout)
		if err != nil && strings.HasPrefix(err.Error(), "HTTP ") {
			return nil, errors.New("Trakyo " + err.Error())
		}
		return body, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		sec := &out.Content
		for page := 0; page < maxContentPages; page++ {
			body, err := request("content", map[string]string{"limit": "100", "offset": strconv.Itoa(len(sec.Rows)), "sort": "revenue"})
			var parsed contentPage
			if err == nil {
				if parsed, err = parseContentPage(body); err != nil {
					err = errors.New("Invalid content payload")
				}
			}
			if err == nil && parsed.HasMore && len(parsed.Data) == 0 {
				err = errors.New("Pagination stalled")
			}
			if err != nil {
				sec.State = StateError
				if len(sec.Rows) > 0 {
					sec.State = StatePartial
				}
				sec.Message = analyticsMessage(err)
				return
			}
			sec.Rows = append(sec.Rows, parsed.Data...)
			if !parsed.HasMore {
				sec.State = StateReady
				return
			}
		}
		msg := fmt.Sprintf("Showing the first %d content rows. Totals below cover loaded rows only.", len(sec.Rows))
		sec.State, sec.Message = StatePartial, &msg
	}()
	go func() {
		defer wg.Done()
		body, err := request("funnel", nil)
		var f Funnel
		if err == nil {
			if f, err = parseFunnel(body); err != nil {
				err = errors.New("Invalid funnel payload")
			}
		}
		if err != nil {
			out.Funnel = FunnelSection{State: StateError, Message: analyticsMessage(err)}
			return
		}
		out.Funnel = FunnelSection{State: StateReady, Data: &f}
	}()
	wg.Wait()
	return out
}

// ---- channels (lib/funnel-analytics.ts) -------------------------------------

// Channel is one acquisition channel on the funnel page.
type Channel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var Channels = []Channel{
	{"youtube", "YouTube"},
	{"instagram_bio", "IG bio"},
	{"instagram", "Instagram"},
	{"linkedin", "LinkedIn"},
	{"x", "X"},
	{"word_of_mouth", "Word of mouth"},
	{"newsletter", "Newsletter"},
	{"forms", "Forms / waitlist"},
	{"paid", "Paid social"},
	{"facebook", "Facebook"},
	{"search", "Search"},
	{"unattributed", "Other / unattributed"},
}

var (
	reYouTube    = regexp.MustCompile(`(?i)youtube|youtu\.be|\byt\b`)
	reInstagram  = regexp.MustCompile(`(?i)instagram|\big\b|insta\b`)
	reBio        = regexp.MustCompile(`(?i)bio`)
	reLinkedIn   = regexp.MustCompile(`(?i)linkedin`)
	reX          = regexp.MustCompile(`(?i)twitter|(?:^|[/.])x\.com|t\.co(?:/|$)|^x$|\bx (post|thread|bio)\b`)
	reWordMouth  = regexp.MustCompile(`(?i)word[ -]of[ -]mouth|personal referral|recommendation`)
	reNewsletter = regexp.MustCompile(`(?i)newsletter|beehiiv|email list`)
	reFacebook   = regexp.MustCompile(`(?i)facebook|\bfb\b`)
	reSearch     = regexp.MustCompile(`(?i)google\.|bing\.|duckduckgo\.`)
	reForms      = regexp.MustCompile(`(?i)waitlist|typeform|\bform\b|opt.?in`)
	reIGPlaced   = regexp.MustCompile(`(?i)^\s*(instagram|ig)\s+(manychat|bio|dm)\b`)
)

func classify(s string) string {
	switch {
	case reYouTube.MatchString(s):
		return "youtube"
	case reInstagram.MatchString(s):
		if reBio.MatchString(s) {
			return "instagram_bio"
		}
		return "instagram"
	case reLinkedIn.MatchString(s):
		return "linkedin"
	case reX.MatchString(s):
		return "x"
	case reWordMouth.MatchString(s):
		return "word_of_mouth"
	case reNewsletter.MatchString(s):
		return "newsletter"
	case reFacebook.MatchString(s):
		return "facebook"
	case reSearch.MatchString(s):
		return "search"
	case reForms.MatchString(s):
		return "forms"
	}
	return ""
}

// ChannelForContent classifies a content item. Only explicit referral
// language counts as word of mouth; a web referrer or an unrecognised
// campaign is unattributed.
func ChannelForContent(typ, name, source string) string {
	if typ == "youtube" {
		return "youtube"
	}
	if typ == "meta_ad" {
		return "paid"
	}
	src := classify(source)
	// A campaign such as a waitlist is not an acquisition channel; an explicit
	// Instagram placement wins over that generic campaign label.
	if typ == "custom" && src == "forms" && reIGPlaced.MatchString(name) {
		if reBio.MatchString(name) {
			return "instagram_bio"
		}
		return "instagram"
	}
	if src == "instagram" && reBio.MatchString(name) {
		return "instagram_bio"
	}
	if src != "" {
		return src
	}
	if n := classify(name); n != "" {
		return n
	}
	return "unattributed"
}

// ChannelCard sums one channel's rows. Views is nil when no YouTube row in
// the channel reported views: unknown reads unknown, never 0.
type ChannelCard struct {
	Channel
	Items           []Content `json:"items"`
	Clicks          int64     `json:"clicks"`
	Visits          int64     `json:"visits"`
	Forms           int64     `json:"forms"`
	Bookings        int64     `json:"bookings"`
	Views           *int64    `json:"views"`
	VideosWithViews int       `json:"videosWithViews"`
	Revenue         struct {
		FirstTouch float64 `json:"first_touch"`
		LastTouch  float64 `json:"last_touch"`
	} `json:"revenue"`
}

// GroupChannels buckets content rows into the 12 channels, in display order.
func GroupChannels(items []Content) []ChannelCard {
	out := make([]ChannelCard, 0, len(Channels))
	for _, ch := range Channels {
		card := ChannelCard{Channel: ch, Items: []Content{}}
		var views int64
		for _, it := range items {
			if ChannelForContent(it.Type, it.Name, it.Source) != ch.ID {
				continue
			}
			card.Items = append(card.Items, it)
			card.Clicks += it.Clicks
			card.Visits += it.Visits
			card.Forms += it.FormSubmissions
			card.Bookings += it.Bookings
			ft, _ := strconv.ParseFloat(it.Revenue.FirstTouch, 64)
			lt, _ := strconv.ParseFloat(it.Revenue.LastTouch, 64)
			card.Revenue.FirstTouch += ft
			card.Revenue.LastTouch += lt
			if it.Type == "youtube" && it.Views != nil {
				views += *it.Views
				card.VideosWithViews++
			}
		}
		if card.VideosWithViews > 0 {
			card.Views = &views
		}
		out = append(out, card)
	}
	return out
}
