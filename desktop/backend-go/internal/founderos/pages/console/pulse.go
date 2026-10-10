// Package console is the operator console's (FounderOS v1 app/page.tsx) pure
// logic: lib/pulse-history.ts and lib/agents/run-digest.ts, ported. The
// endpoint composes live readings into a View; the Svelte page only renders.
//
// Bridge change: G-Brain is retired. Wherever FounderOS v1 stated the brain's
// health score, the console states the Optimal Engine's: the share of the
// engines' own health checks passing (see Brain.Health), in the same
// "N/100" shape.
package console

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Run is one founderos_agent_runs row.
type Run struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agentId"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	OK         bool      `json:"ok"`
	Summary    string    `json:"summary"`
}

// Item is a CommsItem (lib/comms.ts) as far as the console needs it.
type Item struct {
	Source string `json:"source"`
	Title  string `json:"title"`
	TS     string `json:"ts"`
}

// Point is a kit SeriesPoint: one labelled count.
type Point struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

const day = 24 * time.Hour

func parseTS(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// PerDay counts ISO timestamps into the last `days` UTC-day buckets, oldest
// first (runsPerDay / inboundPerDay). Unparseable stamps are skipped.
func PerDay(stamps []string, days int, now time.Time) []int {
	counts := make([]int, days)
	index := map[string]int{}
	for i := 0; i < days; i++ {
		index[now.UTC().Add(-time.Duration(days-1-i)*day).Format("2006-01-02")] = i
	}
	for _, s := range stamps {
		t, ok := parseTS(s)
		if !ok {
			continue
		}
		if i, ok := index[t.UTC().Format("2006-01-02")]; ok {
			counts[i]++
		}
	}
	return counts
}

// Segment is one piece of the state-of-the-world line.
type Segment struct {
	Text string `json:"text"`
	Tone string `json:"tone"` // ok | warn | err | accent | dim
}

// PulseFacts feeds StateOfWorld.
type PulseFacts struct {
	ActiveAgents   int
	TotalAgents    int
	ConnectorsDown int // connectors in an ERROR state; not_configured is NOT "down"
	Inbound        int
	BrainConnected bool
	EnginesUp      int
	EnginesTotal   int
	Health         *int // Brain.Health: nil when no engine answers
	FailedRuns     int
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// StateOfWorld is one honest sentence about what needs the operator,
// worst-first, anchored by the live roster, leading with "All nominal" only
// when nothing is wrong.
func StateOfWorld(f PulseFacts) []Segment {
	var segs []Segment
	if f.FailedRuns > 0 {
		segs = append(segs, Segment{plural(f.FailedRuns, "run") + " failed", "err"})
	}
	switch {
	case f.EnginesTotal == 0:
		segs = append(segs, Segment{"Optimal Engine not configured", "warn"})
	case f.EnginesUp == 0 || f.Health == nil:
		segs = append(segs, Segment{"Optimal Engine offline", "err"})
	case *f.Health < 70:
		segs = append(segs, Segment{fmt.Sprintf("Optimal Engine degraded %d/100", *f.Health), "warn"})
	}
	if f.ConnectorsDown > 0 {
		segs = append(segs, Segment{plural(f.ConnectorsDown, "connector") + " down", "warn"})
	}
	if f.Inbound > 0 {
		segs = append(segs, Segment{fmt.Sprintf("%d inbound need reply", f.Inbound), "accent"})
	}
	hadAttention := len(segs) > 0

	idle := f.TotalAgents - f.ActiveAgents
	if idle < 0 {
		idle = 0
	}
	tone := "dim"
	if f.ActiveAgents > 0 {
		tone = "ok"
	}
	segs = append(segs, Segment{fmt.Sprintf("%d agents live", f.ActiveAgents), tone}, Segment{fmt.Sprintf("%d idle", idle), "dim"})

	// Stated at every health, not only on failure (mock 3a).
	if f.EnginesUp > 0 && f.Health != nil && *f.Health >= 70 {
		segs = append(segs, Segment{fmt.Sprintf("brain %d/100", *f.Health), "ok"})
	}
	if !hadAttention {
		segs = append([]Segment{{"All nominal", "ok"}}, segs...)
	}
	return segs
}

// Meter is a kit Meter; a nil Frac is unknown, drawn as an unknown track.
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

type Volume struct {
	RunsToday   int     `json:"runsToday"`
	FailedToday int     `json:"failedToday"`
	AgentsToday int     `json:"agentsToday"`
	Meters      []Meter `json:"meters"`
}

type VolumeInput struct {
	Connected, TotalConnections int
	ActiveAgents, TotalAgents   *int // nil: the roster could not be read
	EnginesUp, EnginesTotal     int
	Health                      *int // Brain.Health
	Runs                        []Run
	Now                         time.Time
}

func frac(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func f64(v float64) *float64 { return &v }

// LocalMidnight is the start of now's local day: the day the operator lived.
func LocalMidnight(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

// OperatingVolume is Home's Brand Deals volume card: today's runs plus four
// meters, each a real fraction of its own whole.
func OperatingVolume(x VolumeInput) Volume {
	start := LocalMidnight(x.Now)
	var today []Run
	for _, r := range x.Runs {
		if !r.FinishedAt.Before(start) && !r.FinishedAt.After(x.Now) {
			today = append(today, r)
		}
	}
	ok := 0
	agents := map[string]bool{}
	for _, r := range today {
		if r.OK {
			ok++
		}
		agents[r.AgentID] = true
	}
	failed := len(today) - ok

	systems := Meter{
		Label:   fmt.Sprintf("Systems connected (%d/%d)", x.Connected, x.TotalConnections),
		Frac:    f64(frac(x.Connected, x.TotalConnections)),
		Display: fmt.Sprintf("%d%%", int(frac(x.Connected, x.TotalConnections)*100+0.5)),
		Hue:     "var(--bn-accent)",
	}
	roster := Meter{Label: "Agents live", Display: "unknown", Hue: "var(--bn-text-2)"}
	if x.ActiveAgents != nil && x.TotalAgents != nil {
		fr := frac(*x.ActiveAgents, *x.TotalAgents)
		roster = Meter{Label: fmt.Sprintf("Agents live (%d/%d)", *x.ActiveAgents, *x.TotalAgents), Frac: f64(fr), Display: fmt.Sprintf("%d%%", int(fr*100+0.5)), Hue: "var(--bn-text-2)"}
	}
	runs := Meter{Label: fmt.Sprintf("Runs OK today (%d/%d)", ok, len(today)), Frac: f64(frac(ok, len(today))), Display: "no runs yet", Hue: "var(--bn-ok)"}
	if len(today) > 0 {
		runs.Display = fmt.Sprintf("%d ok · %d failed", ok, failed)
	}
	if failed > 0 {
		runs.Hue = "var(--bn-warn)"
	}
	// Prod's "G-Brain health · 90 / 100", worn by the engine's score.
	brain := Meter{Label: "Optimal Engine health", Display: "not configured", Hue: "var(--bn-text)"}
	if x.EnginesTotal > 0 {
		brain.Frac, brain.Display = f64(0), "offline"
		if x.EnginesUp > 0 && x.Health != nil {
			h := max(0, min(100, *x.Health))
			brain.Frac = f64(float64(h) / 100)
			brain.Display = fmt.Sprintf("%d / 100", h)
		}
	}
	return Volume{RunsToday: len(today), FailedToday: failed, AgentsToday: len(agents), Meters: []Meter{systems, roster, runs, brain}}
}

var months = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// DailySeries is the last `days` LOCAL days, oldest first, labelled "Sep 3",
// quiet days kept as zeros so the line is honest. Future stamps are dropped.
func DailySeries(stamps []time.Time, days int, now time.Time) []Point {
	start := LocalMidnight(now)
	out := make([]Point, days)
	keys := map[string]int{}
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, -(days - 1 - i))
		out[i] = Point{Label: fmt.Sprintf("%s %d", months[d.Month()-1], d.Day())}
		keys[d.Format("2006-01-02")] = i
	}
	for _, t := range stamps {
		if t.IsZero() || t.After(now) {
			continue
		}
		if i, ok := keys[t.In(now.Location()).Format("2006-01-02")]; ok {
			out[i].Count++
		}
	}
	return out
}

var mixSources = []string{"email", "whatsapp", "slack"}

// SourceMix is inbound per source in a fixed order, zeros kept.
func SourceMix(items []Item) []Point {
	out := make([]Point, len(mixSources))
	for i, s := range mixSources {
		out[i].Label = s
		for _, it := range items {
			if it.Source == s {
				out[i].Count++
			}
		}
	}
	return out
}

type Attention struct {
	Count    int     `json:"count"`
	Headline string  `json:"headline"`
	Frac     float64 `json:"frac"`
}

// HomeAttention is everything waiting on the operator right now, as a share of
// today's work: waiting / (waiting + done today).
func HomeAttention(inbound, failedToday, connectorsDown, doneToday int) Attention {
	count := inbound + failedToday + connectorsDown
	var parts []string
	if inbound > 0 {
		parts = append(parts, fmt.Sprintf("%d inbound", inbound))
	}
	if failedToday > 0 {
		parts = append(parts, plural(failedToday, "failed run"))
	}
	if connectorsDown > 0 {
		parts = append(parts, plural(connectorsDown, "connector")+" down")
	}
	if count == 0 {
		return Attention{Headline: "Nothing is waiting on you."}
	}
	if doneToday < 0 {
		doneToday = 0
	}
	return Attention{Count: count, Headline: strings.Join(parts, " · "), Frac: float64(count) / float64(count+doneToday)}
}

// InboundLast24h counts feed items stamped in the 24h before now.
func InboundLast24h(items []Item, now time.Time) int {
	n := 0
	for _, it := range items {
		t, ok := parseTS(it.TS)
		if ok && !t.Before(now.Add(-day)) && !t.After(now) {
			n++
		}
	}
	return n
}

// MergeFeed drops unparseable stamps, sorts newest first and caps at limit.
func MergeFeed(items []Item, limit int) []Item {
	var out []Item
	for _, it := range items {
		if _, ok := parseTS(it.TS); ok {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, _ := parseTS(out[i].TS)
		b, _ := parseTS(out[j].TS)
		return a.After(b)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// CollapsedRun is a run standing for Repeat identical (agent, ok, summary) runs.
type CollapsedRun struct {
	Run
	Repeat int `json:"repeat"`
}

// CollapseRuns folds identical (agent, ok, summary) runs into their newest
// row with a count, so a polling cron costs one line (run-digest.ts).
func CollapseRuns(runs []Run) []CollapsedRun {
	seen := map[string]int{}
	var out []CollapsedRun
	for _, r := range runs {
		key := fmt.Sprintf("%s\x00%t\x00%s", r.AgentID, r.OK, r.Summary)
		if i, ok := seen[key]; ok {
			out[i].Repeat++
			continue
		}
		seen[key] = len(out)
		out = append(out, CollapsedRun{Run: r, Repeat: 1})
	}
	return out
}
