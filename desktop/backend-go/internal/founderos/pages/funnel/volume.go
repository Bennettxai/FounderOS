package funnel

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
)

// ---- Funnel Volume (lib/funnel-volume.ts) ------------------------------------

// Meter is one hand-off bar. Frac nil is unknown (no leads at the stage
// before): the kit draws no bar instead of a fake 0.
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

type Point struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type Volume struct {
	RevenueUSD      float64 `json:"revenueUsd"`
	Chips           []Chip  `json:"chips"`
	Caption         string  `json:"caption"`
	Meters          []Meter `json:"meters"`
	Foot            string  `json:"foot"`
	Series          []Point `json:"series"`
	TouchesInWindow int     `json:"touchesInWindow"`
	Insight         Insight `json:"insight"`
	// Window is the entry window this volume covers (days), nil for every
	// active lead; Leads is how many leads the headline and meters cover.
	Window *int `json:"window"`
	Leads  int  `json:"leads"`
}

// VolumeWindowDays are the windows the /funnel volume selector offers
// (the operator, 2026-09-24).
var VolumeWindowDays = []int{30, 60, 90}

// VolumeWindows is funnelVolumeWindows: one volume per selector window,
// precomputed so the 30 / 60 / 90 toggle never refetches. Keys are "30",
// "60", "90".
func VolumeWindows(active []Journey, archived int, now time.Time) map[string]Volume {
	out := make(map[string]Volume, len(VolumeWindowDays))
	for _, w := range VolumeWindowDays {
		out[fmt.Sprint(w)] = volumeOf(active, archived, now, w, w)
	}
	return out
}

var (
	// handoff: one short label per stage hand-off ("First touch → Engaged"),
	// as v1 lib/funnel-volume.ts HANDOFF.
	handoff = func() []string {
		out := make([]string, 0, len(Stages)-1)
		for i := 1; i < len(Stages); i++ {
			out = append(out, Stages[i-1].Label+" → "+Stages[i].Label)
		}
		return out
	}()
	// Hues resolve against the page's ramp tokens (frontend funnel tokens.css).
	hues   = []string{"var(--ramp-1)", "var(--ramp-2)", "var(--ramp-3)", "var(--bn-accent)"}
	months = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
)

func plural(n int, one string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %ss", n, one)
}

func jsRound(f float64) float64 { return math.Floor(f + 0.5) }

// VolumeOf shapes the active journeys like Deal Volume: closed revenue, one
// meter per stage hand-off, touches per UTC day, and the "needs you" count.
func VolumeOf(journeys []Journey, archived int, now time.Time, days int) Volume {
	return volumeOf(journeys, archived, now, days, 0)
}

// volumeOf with window > 0 narrows the headline, chips and meters to the
// cohort that ENTERED (first touch) in the last window days; the touch series
// follows the same window over every active journey, and needs-you is never
// windowed (a stale lead still needs you).
func volumeOf(active []Journey, archived int, now time.Time, days, window int) Volume {
	if window > 0 {
		days = window
	}
	end := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	keys := make([]string, days)
	for i := range keys {
		keys[i] = end.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
	}
	journeys := active
	if window > 0 {
		journeys = []Journey{}
		for _, j := range active {
			entered := j.CreatedAt
			if len(j.Touches) > 0 {
				entered = j.Touches[0].At
			}
			if len(entered) > 10 {
				entered = entered[:10]
			}
			if entered >= keys[0] {
				journeys = append(journeys, j)
			}
		}
	}
	s := Summarize(journeys)
	v := Volume{RevenueUSD: s.RevenueUSD, Meters: []Meter{}, Leads: len(journeys)}
	if window > 0 {
		w := window
		v.Window = &w
	}
	for k := range handoff {
		prev, next := s.Stages[k].Total, s.Stages[k+1].Total
		m := Meter{Label: fmt.Sprintf("%s (%d/%d)", handoff[k], next, prev), Display: "no leads", Hue: hues[k]}
		if prev > 0 {
			f := float64(next) / float64(prev)
			m.Frac, m.Display = &f, fmt.Sprintf("%d%%", int(jsRound(f*100)))
		}
		v.Meters = append(v.Meters, m)
	}
	stalled := 0
	for _, j := range journeys {
		if MetaOf(j, now).State == "stalled" {
			stalled++
		}
	}
	v.Chips = []Chip{{Tone: "ok", Text: fmt.Sprintf("%d closed", s.Converted)}}
	if stalled > 0 {
		v.Chips = append(v.Chips, Chip{Tone: "err", Text: fmt.Sprintf("%d stalled", stalled)})
	}
	// archived leads are in no entry window, so only the all-time view counts them
	if archived > 0 && window == 0 {
		v.Chips = append(v.Chips, Chip{Text: fmt.Sprintf("%d archived", archived)})
	}
	first := s.Stages[0]
	switch {
	case window == 0:
		v.Caption = fmt.Sprintf("closed revenue across %s · %d organic / %d ads entry", plural(len(journeys), "active client"), first.Organic, first.Ads)
	case len(journeys) == 0:
		v.Caption = fmt.Sprintf("no leads entered in the last %d days", window)
	default:
		v.Caption = fmt.Sprintf("closed revenue from %s that entered in the last %dd · %d organic / %d ads", plural(len(journeys), "lead"), window, first.Organic, first.Ads)
	}

	// Touches per UTC day over the trailing window ending today (every active lead).
	counts := map[string]int{}
	for _, k := range keys {
		counts[k] = 0
	}
	for _, j := range active {
		for _, t := range j.Touches {
			if _, ok := counts[t.At]; ok {
				counts[t.At]++
			}
		}
	}
	for _, k := range keys {
		d, _ := time.Parse("2006-01-02", k)
		v.Series = append(v.Series, Point{Label: fmt.Sprintf("%s %d", months[d.Month()-1], d.Day()), Count: counts[k]})
		v.TouchesInWindow += counts[k]
	}

	endToEnd := "no leads yet"
	if first.Total > 0 {
		endToEnd = fmt.Sprintf("%d%% end to end", int(jsRound(float64(s.Converted)/float64(first.Total)*100)))
	}
	v.Foot = "each bar is a stage over the one before · " + endToEnd

	q := Attention(active, now)
	value := len(q.PushNow) + len(q.SaveNow)
	v.Insight = Insight{Value: value, Headline: "Nothing waiting on you", Body: "Every lead is fresh or already closed."}
	if value > 0 {
		v.Insight.Headline = fmt.Sprintf("%d to push · %d to save", len(q.PushNow), len(q.SaveNow))
		var names []string
		for _, j := range append(append([]Journey{}, q.PushNow...), q.SaveNow...) {
			if len(names) == 3 {
				break
			}
			if j.Person != nil {
				names = append(names, *j.Person)
			} else {
				names = append(names, j.Name)
			}
		}
		v.Insight.Body = strings.Join(names, " · ")
	}
	if len(active) > 0 {
		v.Insight.Frac = float64(value) / float64(len(active))
	}
	return v
}

// ---- contact layer (lib/funnel-contact.ts) ----------------------------------

const minNameMatch = 5 // "Sam" must never hijack "Samantha Corp Billing"

// LastMessageFor is the newest comms item that plausibly belongs to a lead:
// exact email on replyTo/sender, else the full name in sender or title.
func LastMessageFor(name string, mail *string, items []email.CommsItem) *email.CommsItem {
	em := ""
	if mail != nil {
		em = norm(*mail)
	}
	n := norm(name)
	usable := len(n) >= minNameMatch
	var best *email.CommsItem
	var bestTS time.Time
	for i := range items {
		it := items[i]
		replyTo, sender, title := norm(it.ReplyTo), norm(it.Sender), norm(it.Title)
		ok := (em != "" && (replyTo == em || sender == em)) || (usable && (strings.Contains(sender, n) || strings.Contains(title, n)))
		if !ok {
			continue
		}
		ts, _ := time.Parse(time.RFC3339Nano, it.TS)
		if best == nil || ts.After(bestTS) {
			best, bestTS = &items[i], ts
		}
	}
	return best
}
