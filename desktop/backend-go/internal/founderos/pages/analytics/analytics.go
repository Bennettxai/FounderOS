// Package analytics is the /analytics page's pure logic, ported from the operator
// OS lib/analytics.ts, lib/analytics-volume.ts, lib/operating-metrics.ts and
// lib/growth.ts (+ the audience-growth slice of lib/social.ts). Nothing here
// invents a number: an empty input stays empty and unknown stays nil.
package analytics

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Run is the slice of a founderos_agent_runs row the page reads.
type Run struct {
	AgentID   string `json:"agentId"`
	OK        bool   `json:"ok"`
	StartedAt string `json:"startedAt"` // RFC 3339
}

// LabelCount is the kit's SeriesPoint ({label, count}).
type LabelCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func dateRange(end string, days int) []string {
	e, err := time.Parse("2006-01-02", end)
	if err != nil {
		return nil
	}
	out := make([]string, 0, days)
	for i := days - 1; i >= 0; i-- {
		out = append(out, e.AddDate(0, 0, -i).Format("2006-01-02"))
	}
	return out
}

func day(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// RunVolume buckets runs per UTC day over the trailing window ending on end;
// quiet days are present with 0, so the chart never lies about them.
func RunVolume(runs []Run, end string, days int) []DayCount {
	counts := map[string]int{}
	for _, r := range runs {
		counts[day(r.StartedAt)]++
	}
	out := []DayCount{}
	for _, d := range dateRange(end, days) {
		out = append(out, DayCount{Date: d, Count: counts[d]})
	}
	return out
}

// RunsWithin counts runs started within the trailing window ending on end.
func RunsWithin(runs []Run, end string, days int) int {
	r := dateRange(end, days)
	if len(r) == 0 {
		return 0
	}
	n := 0
	for _, x := range runs {
		if day(x.StartedAt) >= r[0] {
			n++
		}
	}
	return n
}

// ---- Volume (lib/analytics-volume.ts) ---------------------------------------

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

// Channel is one audience channel; Followers nil is unmeasured, not zero.
type Channel struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Followers *int64 `json:"followers"`
}

type Pending struct {
	Label  string `json:"label"`
	Source string `json:"source"`
}

type VolumeInput struct {
	Channels   []Channel
	Subs       *int64
	Growth7d   *float64
	Live       int
	Pending    []Pending
	Runs       []Run
	AgentNames map[string]string
	Today      string // YYYY-MM-DD, the rhythm window's last day
	Days       int    // 0 means 30
}

type RunsVolume struct {
	Total  int     `json:"total"`
	OK     int     `json:"ok"`
	Failed int     `json:"failed"`
	OKPct  *int    `json:"okPct"`
	Agents int     `json:"agents"`
	Meters []Meter `json:"meters"`
}

type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type Volume struct {
	Reach       int64        `json:"reach"`
	Headline    string       `json:"headline"`
	Chips       []Chip       `json:"chips"`
	Caption     string       `json:"caption"`
	Meters      []Meter      `json:"meters"`
	Foot        string       `json:"foot"`
	Runs        RunsVolume   `json:"runs"`
	Rhythm      []LabelCount `json:"rhythm"` // Mon..Sun
	RhythmTotal int          `json:"rhythmTotal"`
	Insight     Insight      `json:"insight"`
}

var (
	hues     = []string{"var(--ramp-1)", "var(--ramp-2)", "var(--ramp-3)", "var(--ramp-4)"}
	weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
)

// EnUS is Number.toLocaleString('en-US') for integers.
func EnUS(v int64) string {
	s := strconv.FormatInt(v, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

// toFixed1 is JS Number.toFixed(1): ties round away from zero.
func toFixed1(x float64) string {
	a := math.Floor(math.Abs(x)*10+0.5) / 10
	if x < 0 && a != 0 {
		return "-" + strconv.FormatFloat(a, 'f', 1, 64)
	}
	return strconv.FormatFloat(a, 'f', 1, 64)
}

type row struct {
	label string
	value int64
}

// topWithOther: biggest first; past max rows the tail folds into one "Other".
func topWithOther(rows []row, max int, other func(int) string) []row {
	sorted := append([]row(nil), rows...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].value > sorted[b].value })
	if len(sorted) <= max {
		return sorted
	}
	head, tail := sorted[:max-1], sorted[max-1:]
	var sum int64
	for _, r := range tail {
		sum += r.value
	}
	return append(append([]row(nil), head...), row{other(len(tail)), sum})
}

// VolumeOf shapes the page's own numbers like Deal Volume.
func VolumeOf(x VolumeInput) Volume {
	var channels []row
	for _, c := range x.Channels {
		if c.Followers != nil {
			channels = append(channels, row{c.Label, *c.Followers})
		}
	}
	if x.Subs != nil && *x.Subs != 0 {
		channels = append(channels, row{"Email list", *x.Subs})
	}
	var reach int64
	for _, c := range channels {
		reach += c.value
	}
	v := Volume{Reach: reach, Headline: EnUS(reach), Chips: []Chip{}, Meters: []Meter{}}
	for i, c := range topWithOther(channels, 4, func(k int) string { return fmt.Sprintf("Other channels (%d)", k) }) {
		m := Meter{Label: c.label, Hue: hues[i%len(hues)], Display: EnUS(c.value) + " · 0%"}
		if reach > 0 {
			fr := float64(c.value) / float64(reach)
			m.Frac, m.Display = &fr, fmt.Sprintf("%s · %d%%", EnUS(c.value), int(jsRound(fr*100)))
		}
		v.Meters = append(v.Meters, m)
	}
	if x.Growth7d != nil && *x.Growth7d != 0 {
		tone, sign := "err", ""
		if *x.Growth7d > 0 {
			tone, sign = "ok", "+"
		}
		v.Chips = append(v.Chips, Chip{Tone: tone, Text: sign + toFixed1(*x.Growth7d) + "% 7d"})
	}
	if x.Subs != nil && *x.Subs != 0 {
		v.Chips = append(v.Chips, Chip{Text: EnUS(*x.Subs) + " email subs"})
	}
	total := x.Live + len(x.Pending)
	v.Caption = "no audience snapshots yet"
	if len(channels) > 0 {
		s := "s"
		if len(channels) == 1 {
			s = ""
		}
		v.Caption = fmt.Sprintf("total reach across %d channel%s", len(channels), s)
	}
	v.Foot = fmt.Sprintf("reach = followers + email list · %d of %d metrics live", x.Live, total)

	// Runs by agent, in first-seen order before the sort (Map insertion order).
	var order []string
	byAgent := map[string]int64{}
	ok := 0
	for _, r := range x.Runs {
		if _, seen := byAgent[r.AgentID]; !seen {
			order = append(order, r.AgentID)
		}
		byAgent[r.AgentID]++
		if r.OK {
			ok++
		}
	}
	n := len(x.Runs)
	v.Runs = RunsVolume{Total: n, OK: ok, Failed: n - ok, Agents: len(byAgent), Meters: []Meter{}}
	if n > 0 {
		p := int(jsRound(float64(ok) / float64(n) * 100))
		v.Runs.OKPct = &p
	}
	var agentRows []row
	for _, id := range order {
		label := id
		if name, has := x.AgentNames[id]; has {
			label = name
		}
		agentRows = append(agentRows, row{label, byAgent[id]})
	}
	for i, a := range topWithOther(agentRows, 4, func(k int) string { return fmt.Sprintf("Other agents (%d)", k) }) {
		fr := float64(a.value) / float64(n)
		v.Runs.Meters = append(v.Runs.Meters, Meter{Label: fmt.Sprintf("%s (%d)", a.label, a.value), Frac: &fr,
			Display: fmt.Sprintf("%d%%", int(jsRound(fr*100))), Hue: hues[i%len(hues)]})
	}

	days := x.Days
	if days == 0 {
		days = 30
	}
	byDay := make([]int, 7)
	if r := dateRange(x.Today, days); len(r) > 0 {
		for _, run := range x.Runs {
			d := day(run.StartedAt)
			if d < r[0] || d > x.Today {
				continue
			}
			t, err := time.Parse("2006-01-02", d)
			if err != nil {
				continue
			}
			byDay[(int(t.Weekday())+6)%7]++
		}
	}
	for i, label := range weekdays {
		v.Rhythm = append(v.Rhythm, LabelCount{Label: label, Count: byDay[i]})
		v.RhythmTotal += byDay[i]
	}

	pending := len(x.Pending)
	v.Insight = Insight{Value: pending, Headline: "Every metric is live", Body: "Nothing to wire. Every tile reads a real connector."}
	if pending > 0 {
		s := "s"
		if pending == 1 {
			s = ""
		}
		v.Insight.Headline = fmt.Sprintf("%d metric%s waiting on credentials", pending, s)
		var labels []string
		for i := 0; i < pending && i < 3; i++ {
			labels = append(labels, x.Pending[i].Label)
		}
		v.Insight.Body = strings.Join(labels, " · ")
	}
	if total > 0 {
		v.Insight.Frac = float64(x.Live) / float64(total)
	}
	return v
}

// ---- operating metrics (lib/operating-metrics.ts) -----------------------------

// MetricInput is one tile's read: Value nil (or <= 0) is pending, never faked.
type MetricInput struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Unit     string   `json:"unit"`
	Source   string   `json:"source"`
	Value    *float64 `json:"value"`
	Delta    float64  `json:"delta"`
	DeltaPct bool     `json:"deltaPct"`
}

type MetricTile struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Unit     string  `json:"unit"`
	Source   string  `json:"source"`
	Value    float64 `json:"value"`
	Delta    float64 `json:"delta"`
	DeltaPct bool    `json:"deltaPct"`
	Live     bool    `json:"live"`
}

// SplitMetrics: a tile is live only when a connector handed back a positive value.
func SplitMetrics(inputs []MetricInput) (live, pending []MetricTile) {
	live, pending = []MetricTile{}, []MetricTile{}
	for _, m := range inputs {
		isLive := m.Value != nil && *m.Value > 0
		t := MetricTile{ID: m.ID, Label: m.Label, Unit: m.Unit, Source: m.Source, Delta: m.Delta, DeltaPct: m.DeltaPct, Live: isLive}
		if m.Value != nil {
			t.Value = *m.Value
		}
		if isLive {
			live = append(live, t)
		} else {
			pending = append(pending, t)
		}
	}
	return live, pending
}

// SparkSeries: real per-day history once two points exist; until then a
// deterministic placeholder shape (stable per id and value, never random).
func SparkSeries(history []float64, id string, value float64) []float64 {
	if len(history) >= 2 {
		return history
	}
	seed := 0
	for _, r := range id {
		seed += int(r)
	}
	out := make([]float64, 7)
	for i := range out {
		wobble := float64((seed*(i+3))%17)/17 - 0.5
		out[i] = math.Max(0, value*(0.82+0.18*(float64(i)/6)+wobble*0.08))
	}
	return out
}

// ---- growth (lib/growth.ts + lib/social.ts audienceGrowthPct) -----------------

// Point is one dated value, oldest → newest.
type Point struct {
	CapturedAt string  `json:"capturedAt"` // YYYY-MM-DD
	Value      float64 `json:"value"`
}

func dateMs(d string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", day(d))
	return t, err == nil
}

// WindowDelta is the latest value and the baseline at/before the window start;
// ok=false when history does not reach back that far (or the baseline is 0).
func WindowDelta(points []Point, days int) (current, baseline float64, ok bool) {
	if len(points) < 2 {
		return 0, 0, false
	}
	latest := points[len(points)-1]
	lt, okT := dateMs(latest.CapturedAt)
	if !okT {
		return 0, 0, false
	}
	start := lt.AddDate(0, 0, -days)
	for i := len(points) - 1; i >= 0; i-- {
		t, okP := dateMs(points[i].CapturedAt)
		if okP && !t.After(start) {
			if points[i].Value == 0 {
				return 0, 0, false
			}
			return latest.Value, points[i].Value, true
		}
	}
	return 0, 0, false
}

// GrowthOver is percentage growth over the trailing window, nil without history.
func GrowthOver(points []Point, days int) *float64 {
	c, b, ok := WindowDelta(points, days)
	if !ok {
		return nil
	}
	g := (c - b) / b * 100
	return &g
}

// GrowthAllTime is first-to-last growth, nil under two points or a zero start.
func GrowthAllTime(points []Point) *float64 {
	if len(points) < 2 || points[0].Value == 0 {
		return nil
	}
	g := (points[len(points)-1].Value - points[0].Value) / points[0].Value * 100
	return &g
}

// AudienceGrowthPct aggregates growth across channels; a channel counts only
// when its history reaches back the whole window. Nil when none qualifies.
func AudienceGrowthPct(channels [][]Point, days int) *float64 {
	var cur, base float64
	q := 0
	for _, s := range channels {
		c, b, ok := WindowDelta(s, days)
		if !ok {
			continue
		}
		cur, base, q = cur+c, base+b, q+1
	}
	if q == 0 || base == 0 {
		return nil
	}
	g := (cur - base) / base * 100
	return &g
}
