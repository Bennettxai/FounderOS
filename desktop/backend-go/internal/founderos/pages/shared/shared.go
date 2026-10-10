// Package shared holds the view-model pieces several agent-side /os pages
// compute the same way: the slab kit's wire types (series points, meters,
// chips, the insight card), FounderOS v1 lib/pulse-history dailySeries,
// lib/short-labels and lib/scheduled-jobs.
package shared

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// SeriesPoint is one StepLine / DotMatrix column.
type SeriesPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Meter is one MeterStack row; a nil Frac reads unknown.
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

// Chip is a BigStat chip; Tone is ok|warn|err|accent or empty.
type Chip struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

// Insight is the page's one gradient "Needs you" card.
type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

// Frac clamps n/d into [0,1]; a zero denominator is 0.
func Frac(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return max(0, min(1, float64(n)/float64(d)))
}

func F(v float64) *float64 { return &v }

// Pct is Math.round(f*100)%.
func Pct(f float64) string { return fmt.Sprintf("%d%%", int(f*100+0.5)) }

func Plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// ParseISO accepts the RFC 3339 stamps the repos and the board emit.
func ParseISO(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.In(time.Local).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// DailySeries counts timestamps per local day over the last `days` days
// ending today ("Sep 30"); unparseable and future stamps are skipped.
func DailySeries(timestamps []string, days int, now time.Time) []SeriesPoint {
	start := dayStart(now)
	out := make([]SeriesPoint, days)
	keys := make([]time.Time, days)
	for i := range out {
		d := start.AddDate(0, 0, -(days - 1 - i))
		keys[i] = d
		out[i] = SeriesPoint{Label: d.Format("Jan 2")}
	}
	for _, ts := range timestamps {
		t, ok := ParseISO(ts)
		if !ok || t.After(now) {
			continue
		}
		k := dayStart(t)
		for i := range keys {
			if keys[i].Equal(k) {
				out[i].Count++
				break
			}
		}
	}
	return out
}

// WindowStart is local midnight days-1 days before now.
func WindowStart(now time.Time, days int) time.Time {
	return dayStart(now).AddDate(0, 0, -(days - 1))
}

// ShortLabels is the first word of each name (cut to max runes when max > 0),
// numbered only when two collide.
func ShortLabels(names []string, max int) []string {
	seen := map[string]int{}
	out := make([]string, len(names))
	for i, name := range names {
		trim := strings.TrimSpace(name)
		base := ""
		if f := strings.Fields(trim); len(f) > 0 {
			base = f[0]
		}
		if base == "" {
			base = trim
		}
		if max > 0 && utf8.RuneCountInString(base) > max {
			base = string([]rune(base)[:max])
		}
		if base == "" {
			base = "Agent"
		}
		seen[base]++
		if n := seen[base]; n == 1 {
			out[i] = base
		} else {
			out[i] = fmt.Sprintf("%s %d", base, n)
		}
	}
	return out
}

// ---- lib/scheduled-jobs.ts --------------------------------------------------

// JobRow is ScheduledJobRow: a cron with what actually happened.
type JobRow struct {
	ID            string  `json:"id"`
	Description   string  `json:"description"`
	AgentID       string  `json:"agentId"`
	AgentName     string  `json:"agentName"`
	UnknownAgent  bool    `json:"unknownAgent"`
	Schedule      string  `json:"schedule"`
	ScheduleLabel string  `json:"scheduleLabel"`
	Enabled       bool    `json:"enabled"`
	Runs          int     `json:"runs"`
	OK            int     `json:"ok"`
	LastRunAt     *string `json:"lastRunAt"`
	LastOK        *bool   `json:"lastOk"`
	NextRunAt     *string `json:"nextRunAt"`
	Overdue       bool    `json:"overdue"`
	History       []bool  `json:"history"`
	LastSummary   *string `json:"lastSummary"`
}

// ScheduledJobRows shapes crons for the Scheduled panels. Overdue is the
// runner's own view (agents.DueCrons), so the panel and runner never drift.
// recent is per cron, newest first (CronRunsByCron). Order: overdue, then
// healthy, then disabled; alphabetical inside each band.
func ScheduledJobRows(crons []osdata.Cron, stats map[string]osdata.CronStat, agentNames map[string]string, recent map[string][]osdata.CronRun, now time.Time) []JobRow {
	lite := make([]agents.Cron, 0, len(crons))
	for _, c := range crons {
		ac := agents.Cron{ID: c.ID, AgentID: c.AgentID, Schedule: c.Schedule, Description: c.Description, Enabled: c.Enabled}
		if st, ok := stats[c.ID]; ok && st.LastRunAt != nil {
			if t, ok := ParseISO(*st.LastRunAt); ok {
				ac.LastRunAt = &t
			}
		}
		lite = append(lite, ac)
	}
	overdue := map[string]bool{}
	for _, c := range agents.DueCrons(lite, now) {
		overdue[c.ID] = true
	}

	rows := make([]JobRow, 0, len(crons))
	for _, c := range crons {
		st := stats[c.ID]
		name, known := agentNames[c.AgentID]
		if !known {
			name = c.AgentID
		}
		label := agents.DescribeCron(c.Schedule)
		if label == "" {
			label = c.Schedule
		}
		r := JobRow{
			ID: c.ID, Description: c.Description, AgentID: c.AgentID, AgentName: name, UnknownAgent: !known,
			Schedule: c.Schedule, ScheduleLabel: label, Enabled: c.Enabled,
			Runs: st.Runs, OK: st.OK, LastRunAt: st.LastRunAt, LastOK: st.LastOK,
			Overdue: overdue[c.ID], History: []bool{},
		}
		if c.Enabled {
			if next, ok := agents.NextOccurrence(c.Schedule, now); ok {
				s := next.UTC().Format(time.RFC3339)
				r.NextRunAt = &s
			}
		}
		runs := recent[c.ID]
		for i := len(runs) - 1; i >= 0; i-- {
			r.History = append(r.History, runs[i].OK)
		}
		if len(runs) > 0 {
			s := runs[0].Summary
			r.LastSummary = &s
		}
		rows = append(rows, r)
	}
	band := func(r JobRow) int {
		switch {
		case !r.Enabled:
			return 2
		case r.Overdue:
			return 0
		}
		return 1
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if bi, bj := band(rows[i]), band(rows[j]); bi != bj {
			return bi < bj
		}
		return rows[i].Description < rows[j].Description
	})
	return rows
}
