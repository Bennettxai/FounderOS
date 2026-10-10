package clients

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// The view-model's shapes, matching the FounderOS kit's Meter / SeriesPoint /
// StatChip props so the page renders them unchanged.
type (
	ClientRef   struct{ ID, Name string }
	ClientCount struct {
		ID    string `json:"id"`
		Count int    `json:"count"`
	}
	Chip struct {
		Tone string `json:"tone,omitempty"`
		Text string `json:"text"`
	}
	Meter struct {
		Label   string   `json:"label"`
		Frac    *float64 `json:"frac"`
		Display string   `json:"display"`
		Hue     string   `json:"hue"`
	}
	Point struct {
		Label string `json:"label"`
		Count int    `json:"count"`
	}
	Insight struct {
		Value    int     `json:"value"`
		Headline string  `json:"headline"`
		Body     string  `json:"body"`
		Frac     float64 `json:"frac"`
	}
)

type Volume struct {
	Headline         int            `json:"headline"`
	Counts           map[Status]int `json:"counts"`
	Chips            []Chip         `json:"chips"`
	Caption          string         `json:"caption"`
	Meters           []Meter        `json:"meters"`
	Foot             string         `json:"foot"`
	Series           []Point        `json:"series"`
	RequestsInWindow int            `json:"requestsInWindow"`
	Rhythm           []Point        `json:"rhythm"`
	PerClient        []ClientCount  `json:"perClient"`
	Insight          Insight        `json:"insight"`
}

// StatusStep is one entry of the status order the meters, chips and filter
// pills all share. Hues are the bridge's --bn-* status tokens.
type StatusStep struct {
	Key   Status `json:"key"`
	Meter string `json:"meter"`
	Chip  string `json:"chip"`
	Tone  string `json:"tone"`
	Hue   string `json:"hue"`
}

var StatusOrder = []StatusStep{
	{StatusLaunched, "Launched", "launched", "ok", "var(--bn-ok)"},
	{StatusSaved, "Saved drafts", "saved", "accent", "var(--bn-accent)"},
	{StatusLaunching, "Launching", "launching", "warn", "var(--bn-warn)"},
	{StatusNeedsAttention, "Needs attention", "needs attention", "err", "var(--bn-err)"},
}

const briefMax = 28

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func frac(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return math.Max(0, math.Min(1, float64(n)/float64(d)))
}

func jsRound(x float64) int { return int(math.Floor(x + 0.5)) }

func pct(f float64) string { return fmt.Sprintf("%d%%", jsRound(f*100)) }

func plural(n int, one, many string) string {
	if many == "" {
		many = one + "s"
	}
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

var spaces = regexp.MustCompile(`\s+`)

func clip(s string) string {
	one := strings.TrimSpace(spaces.ReplaceAllString(s, " "))
	if utf8.RuneCountInString(one) <= briefMax {
		return one
	}
	r := []rune(one)
	return strings.TrimRight(string(r[:briefMax-1]), " \t\n") + "…"
}

// ParseInstant reads a stored createdAt (ISO-8601).
func ParseInstant(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ComputeVolume is clientsVolume: one pure pass over the confirmed clients
// and their requests. Days are local to now's location.
func ComputeVolume(clients []ClientRef, work []Work, now time.Time, days int) Volume {
	if days <= 0 {
		days = 14
	}
	loc := now.Location()
	total := len(work)

	counts := map[Status]int{StatusSaved: 0, StatusLaunching: 0, StatusLaunched: 0, StatusNeedsAttention: 0}
	for _, w := range work {
		if w.Status.Valid() {
			counts[w.Status]++
		}
	}

	chips := []Chip{}
	for _, s := range StatusOrder {
		if counts[s.Key] > 0 {
			chips = append(chips, Chip{Tone: s.Tone, Text: fmt.Sprintf("%d %s", counts[s.Key], s.Chip)})
		}
	}
	meters := []Meter{}
	if total > 0 {
		for _, s := range StatusOrder {
			f := frac(counts[s.Key], total)
			meters = append(meters, Meter{Label: fmt.Sprintf("%s (%d)", s.Meter, counts[s.Key]), Frac: &f, Display: pct(f), Hue: s.Hue})
		}
	}

	perClient := make([]ClientCount, 0, len(clients))
	withRequests := 0
	for _, c := range clients {
		n := 0
		for _, w := range work {
			if w.ClientID == c.ID {
				n++
			}
		}
		if n > 0 {
			withRequests++
		}
		perClient = append(perClient, ClientCount{ID: c.ID, Count: n})
	}
	tail := "no requests yet"
	if total > 0 {
		tail = fmt.Sprintf("%d with requests", withRequests)
	}
	caption := fmt.Sprintf("across %s · %s", plural(len(clients), "confirmed client", ""), tail)

	// Requests per local day over the window, oldest first, plus the weekday rhythm.
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day()-(days-1), 0, 0, 0, 0, loc)
	series := make([]Point, days)
	index := map[string]int{}
	for i := 0; i < days; i++ {
		d := time.Date(start.Year(), start.Month(), start.Day()+i, 0, 0, 0, 0, loc)
		series[i] = Point{Label: d.Format("Jan 2")}
		index[d.Format("2006-01-02")] = i
	}
	rhythm := make([]Point, len(weekdays))
	for i, l := range weekdays {
		rhythm[i] = Point{Label: l}
	}
	inWindow := 0
	for _, w := range work {
		t, ok := ParseInstant(w.CreatedAt)
		if !ok || t.Before(start) || t.After(now) {
			continue
		}
		lt := t.In(loc)
		i, ok := index[lt.Format("2006-01-02")]
		if !ok {
			continue
		}
		series[i].Count++
		rhythm[(int(lt.Weekday())+6)%7].Count++
		inWindow++
	}

	// The one gradient card: requests that need the operator's eyes before anything else.
	var needs []Work
	for _, w := range work {
		if w.Status == StatusNeedsAttention || w.Status == StatusLaunching {
			needs = append(needs, w)
		}
	}
	in := Insight{Value: len(needs), Frac: frac(len(needs), total)}
	switch {
	case len(needs) > 0:
		in.Headline = fmt.Sprintf("%d needs attention · %s unconfirmed.", counts[StatusNeedsAttention], plural(counts[StatusLaunching], "launch", "launches"))
		var parts []string
		for i, w := range needs {
			if i == 2 {
				break
			}
			parts = append(parts, clip(w.Brief))
		}
		in.Body = strings.Join(parts, " · ")
	case total > 0:
		in.Headline = "Nothing waiting on you."
		in.Body = fmt.Sprintf("%s launched, %d saved as drafts.", plural(counts[StatusLaunched], "request", ""), counts[StatusSaved])
	default:
		in.Headline = "Nothing waiting on you."
		in.Body = "No requests yet. Save a brief below to start one."
	}

	return Volume{
		Headline: total, Counts: counts, Chips: chips, Caption: caption, Meters: meters,
		Foot: "draft first · nothing publishes automatically", Series: series,
		RequestsInWindow: inWindow, Rhythm: rhythm, PerClient: perClient, Insight: in,
	}
}
