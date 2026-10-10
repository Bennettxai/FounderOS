package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// LeadMagnet is a founderos_lead_magnets row (FounderOS v1 LeadMagnetSchema).
type LeadMagnet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Offer       string `json:"offer"`
	URL         string `json:"url"`
	Status      string `json:"status"`
	Captures    string `json:"captures"`
	Destination string `json:"destination"`
	Source      string `json:"source"`
	LaunchedAt  string `json:"launchedAt"`
	Notes       string `json:"notes"`
	Origin      string `json:"origin"`
}

// Filters are the status pills over the table, in order.
var Filters = []string{"all", "live", "draft", "paused", "archived"}

var statuses = []string{"live", "draft", "paused", "archived"}
var captures = []string{"email", "booking", "none"}

func oneOf(v string, set []string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// FilterOf is leadMagnetFilter: a known status, anything else (or absent) is all.
func FilterOf(param string) string {
	if oneOf(param, Filters) {
		return param
	}
	return "all"
}

// FilterRows is filterLeadMagnets.
func FilterRows(rows []LeadMagnet, param string) []LeadMagnet {
	f := FilterOf(param)
	out := []LeadMagnet{}
	for _, r := range rows {
		if f == "all" || r.Status == f {
			out = append(out, r)
		}
	}
	return out
}

type StatusCounts struct {
	Live     int `json:"live"`
	Draft    int `json:"draft"`
	Paused   int `json:"paused"`
	Archived int `json:"archived"`
}

type LeadMagnetVolume struct {
	Headline        int             `json:"headline"`
	Total           int             `json:"total"`
	Counts          StatusCounts    `json:"counts"`
	Chips           []pagekit.Chip  `json:"chips"`
	Caption         string          `json:"caption"`
	Meters          []pagekit.Meter `json:"meters"`
	Foot            string          `json:"foot"`
	Series          []pagekit.Point `json:"series"`
	ShippedInWindow int             `json:"shippedInWindow"`
	Captures        []pagekit.Point `json:"captures"`
	Destinations    []pagekit.Point `json:"destinations"`
	Insight         Insight         `json:"insight"`
}

const destCols = 6

var destSplit = regexp.MustCompile(`\s*[·|(]\s*`)

// shortDestination: "Beehiiv · newsletter + Cohort 2" reads as "Beehiiv".
func shortDestination(d string) string {
	head := strings.TrimSpace(destSplit.Split(d, 2)[0])
	if head == "" {
		head = "Other"
	}
	if utf8.RuneCountInString(head) > 12 {
		if f := strings.Fields(head); len(f) > 0 {
			return f[0]
		}
	}
	return head
}

// dayKey pads a launchedAt ("2026", "2026-08") to a comparable UTC day.
func dayKey(s string) string {
	k := s
	if len(k) > 10 {
		k = k[:10]
	}
	switch len(k) {
	case 4:
		return k + "-01-01"
	case 7:
		return k + "-01"
	}
	return k
}

// MagnetVolume is leadMagnetVolume over every row; weeks defaults to 12.
func MagnetVolume(rows []LeadMagnet, today string, weeks int) LeadMagnetVolume {
	if weeks <= 0 {
		weeks = 12
	}
	total := len(rows)
	var counts StatusCounts
	email, booking, none := 0, 0, 0
	for _, r := range rows {
		switch r.Status {
		case "live":
			counts.Live++
		case "draft":
			counts.Draft++
		case "paused":
			counts.Paused++
		case "archived":
			counts.Archived++
		}
		switch r.Captures {
		case "email":
			email++
		case "booking":
			booking++
		case "none":
			none++
		}
	}

	chips := []pagekit.Chip{}
	if counts.Draft > 0 {
		chips = append(chips, pagekit.Chip{Tone: "warn", Text: fmt.Sprintf("%d draft", counts.Draft)})
	}
	if counts.Paused > 0 {
		chips = append(chips, pagekit.Chip{Tone: "err", Text: fmt.Sprintf("%d paused", counts.Paused)})
	}
	if counts.Archived > 0 {
		chips = append(chips, pagekit.Chip{Text: fmt.Sprintf("%d archived", counts.Archived)})
	}

	meters := []pagekit.Meter{}
	if total > 0 {
		m := func(label string, n int, hue string) pagekit.Meter {
			return pagekit.Known(fmt.Sprintf("%s (%d/%d)", label, n, total), float64(n)/float64(total), fmt.Sprintf("%d of %d", n, total), hue)
		}
		meters = append(meters, m("Live", counts.Live, pagekit.OK), m("Capturing email", email, pagekit.Ramp1),
			m("Capturing bookings", booking, pagekit.Ramp3), m("No capture yet", none, pagekit.Warn))
	}

	destMap := map[string]int{}
	campaigns := map[string]bool{}
	for _, r := range rows {
		destMap[shortDestination(r.Destination)]++
		if s := strings.TrimSpace(r.Source); s != "" {
			campaigns[s] = true
		}
	}
	type kv struct {
		k string
		n int
	}
	var dests []kv
	for k, n := range destMap {
		dests = append(dests, kv{k, n})
	}
	sort.Slice(dests, func(i, j int) bool {
		if dests[i].n != dests[j].n {
			return dests[i].n > dests[j].n
		}
		return localeLess(dests[i].k, dests[j].k)
	})
	destinations := []pagekit.Point{}
	for i, d := range dests {
		if i >= destCols {
			break
		}
		destinations = append(destinations, pagekit.Point{Label: d.k, Count: float64(d.n)})
	}

	end, _ := utcDay(today)
	keys := make([]string, len(rows))
	for i, r := range rows {
		keys[i] = dayKey(r.LaunchedAt)
	}
	series := []pagekit.Point{}
	for i := weeks - 1; i >= 0; i-- {
		d := end.Add(-time.Duration(i*7) * dayMs)
		k := d.Format("2006-01-02")
		n := 0
		for _, l := range keys {
			if l <= k {
				n++
			}
		}
		series = append(series, pagekit.Point{Label: pagekit.ShortDate(d), Count: float64(n)})
	}
	windowStart := end.Add(-time.Duration(weeks*7) * dayMs).Format("2006-01-02")
	shipped := 0
	for _, k := range keys {
		if k > windowStart && k <= today {
			shipped++
		}
	}

	newestName, newestKey := "", ""
	for i, r := range rows {
		k := keys[i]
		if _, ok := utcDay(k); !ok || k > today {
			continue
		}
		if newestKey == "" || k > newestKey {
			newestName, newestKey = r.Name, k
		}
	}
	var insight Insight
	if newestKey == "" {
		insight = Insight{Display: "none", Headline: "Nothing shipped yet.", Body: "The next landing page lands here."}
	} else {
		nk, _ := utcDay(newestKey)
		quiet := int(math.Max(0, math.Round(end.Sub(nk).Hours()/24)))
		headline := newestName + " went live today."
		if quiet != 0 {
			headline = fmt.Sprintf("%s since %s shipped.", pagekit.Plural(quiet, "day", ""), newestName)
		}
		insight = Insight{
			Value:    quiet,
			Headline: headline,
			Body:     fmt.Sprintf("%d of %d live · ticks light with the live share.", counts.Live, total),
			Frac:     pagekit.Frac(float64(counts.Live), float64(total)),
		}
	}

	caption := "no landing pages recorded yet"
	if total > 0 {
		caption = "of " + pagekit.Plural(total, "landing page", "") + " shipped"
	}
	return LeadMagnetVolume{
		Headline:        counts.Live,
		Total:           total,
		Counts:          counts,
		Chips:           chips,
		Caption:         caption,
		Meters:          meters,
		Foot:            pagekit.Plural(len(destMap), "destination", "") + " · " + pagekit.Plural(len(campaigns), "campaign", ""),
		Series:          series,
		ShippedInWindow: shipped,
		Captures:        []pagekit.Point{{Label: "Email", Count: float64(email)}, {Label: "Booking", Count: float64(booking)}, {Label: "None", Count: float64(none)}},
		Destinations:    destinations,
		Insight:         insight,
	}
}

// ---- create / patch validation (app/api/lead-magnets) -----------------------

var (
	nonSlug  = regexp.MustCompile(`[^a-z0-9]+`)
	edgeDash = regexp.MustCompile(`(^-|-$)`)
	isoDay   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Slugify: "The Claude Trading Setup" -> "the-claude-trading-setup" (max 60).
func Slugify(name string) string {
	s := edgeDash.ReplaceAllString(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

// UniqueID slugs name and suffixes -2, -3 … until it is free.
func UniqueID(name string, taken map[string]bool) string {
	base := Slugify(name)
	if base == "" {
		base = "lead-magnet"
	}
	id := base
	for n := 2; taken[id]; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// ValidationError is a 400: the body was not a lead magnet.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return &ValidationError{Msg: fmt.Sprintf(format, a...)} }

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && (u.Host != "" || u.Opaque != "")
}

type field struct {
	name string
	min  int
	max  int
	set  func(m *LeadMagnet, v string)
	ok   func(v string) bool
	why  string
}

var fields = []field{
	{"name", 1, 120, func(m *LeadMagnet, v string) { m.Name = v }, nil, ""},
	{"url", 0, 0, func(m *LeadMagnet, v string) { m.URL = v }, validURL, "url must be a URL"},
	{"offer", 0, 400, func(m *LeadMagnet, v string) { m.Offer = v }, nil, ""},
	{"status", 0, 0, func(m *LeadMagnet, v string) { m.Status = v }, func(v string) bool { return oneOf(v, statuses) }, "status must be live, draft, paused or archived"},
	{"captures", 0, 0, func(m *LeadMagnet, v string) { m.Captures = v }, func(v string) bool { return oneOf(v, captures) }, "captures must be email, booking or none"},
	{"destination", 0, 200, func(m *LeadMagnet, v string) { m.Destination = v }, nil, ""},
	{"source", 0, 300, func(m *LeadMagnet, v string) { m.Source = v }, nil, ""},
	{"launchedAt", 0, 0, func(m *LeadMagnet, v string) { m.LaunchedAt = v }, isoDay.MatchString, "launchedAt must be YYYY-MM-DD"},
	{"notes", 0, 2000, func(m *LeadMagnet, v string) { m.Notes = v }, nil, ""},
}

func decodeObject(body []byte) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(body), &obj); err != nil || obj == nil {
		return nil, invalid("body must be a JSON object")
	}
	return obj, nil
}

// apply sets every known field present in obj onto m, validating each.
func apply(m *LeadMagnet, obj map[string]json.RawMessage) error {
	for _, f := range fields {
		raw, ok := obj[f.name]
		if !ok {
			continue
		}
		var v string
		if err := json.Unmarshal(raw, &v); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return invalid("%s must be a string", f.name)
		}
		n := utf8.RuneCountInString(v)
		if n < f.min {
			return invalid("%s is required", f.name)
		}
		if f.max > 0 && n > f.max {
			return invalid("%s is longer than %d characters", f.name, f.max)
		}
		if f.ok != nil && !f.ok(v) {
			return invalid("%s", f.why)
		}
		f.set(m, v)
	}
	return nil
}

// ParseCreate is POST's CreateSchema: name and url required, the rest
// defaulted (status live, captures email, launchedAt today), origin 'os'.
// The id is assigned by the caller (UniqueID).
func ParseCreate(body []byte, today string) (LeadMagnet, error) {
	obj, err := decodeObject(body)
	if err != nil {
		return LeadMagnet{}, err
	}
	for _, req := range []string{"name", "url"} {
		if _, ok := obj[req]; !ok {
			return LeadMagnet{}, invalid("%s is required", req)
		}
	}
	m := LeadMagnet{Status: "live", Captures: "email", Origin: "os"}
	if err := apply(&m, obj); err != nil {
		return LeadMagnet{}, err
	}
	if m.LaunchedAt == "" {
		m.LaunchedAt = today
	}
	return m, nil
}

// ApplyPatch is PATCH's partial schema over the current row. id and origin
// are not editable: the id is referenced by whatever links to it, and origin
// protects an OS-made row from the seed.
func ApplyPatch(cur LeadMagnet, body []byte) (LeadMagnet, error) {
	obj, err := decodeObject(body)
	if err != nil {
		return cur, err
	}
	next := cur
	if err := apply(&next, obj); err != nil {
		return cur, err
	}
	next.ID, next.Origin = cur.ID, cur.Origin
	return next, nil
}

// IsValidation reports whether err is a 400.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}
