// Package workflows is the /workflows page's pure logic, ported from the operator
// OS lib/workflow-stats.ts, lib/workflows-volume.ts, app/api/workflows/shared.ts
// (builder input validation and step shaping) and
// app/api/workflows/draft/logic.ts (the claude-CLI drafting assistant).
package workflows

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
)

// WorkflowStats is a workflow's bottom-bar numbers.
type WorkflowStats struct {
	ManualHours         float64 `json:"manualHours"`
	AgentHours          float64 `json:"agentHours"`
	LeakUSD             float64 `json:"leakUsd"`
	LiveReturnsUSD      float64 `json:"liveReturnsUsd"`
	SuggestedReturnsUSD float64 `json:"suggestedReturnsUsd"`
	HumanSteps          int     `json:"humanSteps"`
	AgentSteps          int     `json:"agentSteps"`
	ToolCount           int     `json:"toolCount"`
	AutomationCount     int     `json:"automationCount"`
}

func Stats(w osdata.Workflow) WorkflowStats {
	var s WorkflowStats
	tools := map[string]bool{}
	for _, st := range w.Steps {
		if st.OwnerKind == "human" {
			s.HumanSteps++
			s.ManualHours += st.HoursPerWeek
		} else {
			s.AgentSteps++
			s.AgentHours += st.HoursPerWeek
		}
		if st.LeakUSD != nil {
			s.LeakUSD += *st.LeakUSD
		}
		for _, t := range st.Tools {
			tools[t] = true
		}
		if st.Automation != nil {
			s.AutomationCount++
			if st.Automation.State == "live" {
				s.LiveReturnsUSD += st.Automation.RecoveredUSD
			} else {
				s.SuggestedReturnsUSD += st.Automation.RecoveredUSD
			}
		}
	}
	s.ToolCount = len(tools)
	return s
}

type Counts struct {
	Healthy int `json:"healthy"`
	Overdue int `json:"overdue"`
	Failing int `json:"failing"`
	Paused  int `json:"paused"`
	Enabled int `json:"enabled"`
}

type Load struct {
	ManualHours float64              `json:"manualHours"`
	AgentHours  float64              `json:"agentHours"`
	PerWorkflow []shared.SeriesPoint `json:"perWorkflow"`
}

// WorkflowsVolume is every number the /workflows slab shows.
type WorkflowsVolume struct {
	Headline       int                  `json:"headline"`
	Counts         Counts               `json:"counts"`
	Chips          []shared.Chip        `json:"chips"`
	Caption        string               `json:"caption"`
	Meters         []shared.Meter       `json:"meters"`
	Foot           string               `json:"foot"`
	Series         []shared.SeriesPoint `json:"series"`
	RunsInWindow   int                  `json:"runsInWindow"`
	FailedInWindow int                  `json:"failedInWindow"`
	Rhythm         []shared.SeriesPoint `json:"rhythm"`
	Load           Load                 `json:"load"`
	Insight        shared.Insight       `json:"insight"`
}

// JobState mirrors ScheduledTasks' own state(): paused | failing | overdue | healthy.
func JobState(j shared.JobRow) string {
	switch {
	case !j.Enabled:
		return "paused"
	case j.UnknownAgent:
		return "failing"
	case j.Overdue:
		return "overdue"
	case j.LastOK != nil && !*j.LastOK:
		return "failing"
	}
	return "healthy"
}

func hours(n float64) float64 { return math.Round(n*10) / 10 }

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func usd(n float64) string {
	s := strconv.FormatInt(int64(math.Round(n)), 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 && c != '-' {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return "$" + b.String()
}

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Volume is one pure pass over the page's own rows.
func Volume(jobs []shared.JobRow, wfs []osdata.Workflow, runs []osdata.CronRun, now time.Time, days int) WorkflowsVolume {
	var c Counts
	states := make([]string, len(jobs))
	runsAll, okAll := 0, 0
	for i, j := range jobs {
		states[i] = JobState(j)
		switch states[i] {
		case "healthy":
			c.Healthy++
		case "overdue":
			c.Overdue++
		case "failing":
			c.Failing++
		case "paused":
			c.Paused++
		}
		runsAll += j.Runs
		okAll += j.OK
	}
	c.Enabled = len(jobs) - c.Paused

	chips := []shared.Chip{}
	if c.Healthy > 0 {
		chips = append(chips, shared.Chip{Tone: "ok", Text: strconv.Itoa(c.Healthy) + " healthy"})
	}
	if c.Overdue > 0 {
		chips = append(chips, shared.Chip{Tone: "warn", Text: strconv.Itoa(c.Overdue) + " overdue"})
	}
	if c.Failing > 0 {
		chips = append(chips, shared.Chip{Tone: "err", Text: strconv.Itoa(c.Failing) + " failing"})
	}
	if c.Paused > 0 {
		chips = append(chips, shared.Chip{Text: strconv.Itoa(c.Paused) + " paused"})
	}

	var manual, agent, leak, recovered float64
	steps := 0
	tools := map[string]bool{}
	stats := make([]WorkflowStats, len(wfs))
	for i, w := range wfs {
		stats[i] = Stats(w)
		manual += stats[i].ManualHours
		agent += stats[i].AgentHours
		leak += stats[i].LeakUSD
		recovered += stats[i].LiveReturnsUSD
		steps += len(w.Steps)
		for _, s := range w.Steps {
			for _, t := range s.Tools {
				tools[t] = true
			}
		}
	}
	manual, agent = hours(manual), hours(agent)
	total := hours(manual + agent)
	fracF := func(n, d float64) float64 {
		if d <= 0 {
			return 0
		}
		return math.Max(0, math.Min(1, n/d))
	}
	healthyF := shared.Frac(c.Healthy, c.Enabled)
	runsF := shared.Frac(okAll, runsAll)
	hoursF := fracF(agent, total)
	leakF := fracF(recovered, leak)
	pick := func(cond bool, a, b string) string {
		if cond {
			return a
		}
		return b
	}
	healthyHue := "var(--bn-warn)"
	if c.Enabled > 0 && c.Healthy == c.Enabled {
		healthyHue = "var(--bn-ok)"
	}
	meters := []shared.Meter{
		{Label: "Crons healthy (" + strconv.Itoa(c.Healthy) + "/" + strconv.Itoa(c.Enabled) + ")", Frac: shared.F(healthyF), Display: pick(c.Enabled > 0, shared.Pct(healthyF), "none enabled"), Hue: healthyHue},
		{Label: "Runs OK (" + strconv.Itoa(okAll) + "/" + strconv.Itoa(runsAll) + ")", Frac: shared.F(runsF), Display: pick(runsAll > 0, strconv.Itoa(okAll)+" ok · "+strconv.Itoa(runsAll-okAll)+" failed", "no runs yet"), Hue: "var(--bn-accent)"},
		{Label: "Hours carried by agents (" + num(agent) + "/" + num(total) + "h per wk)", Frac: shared.F(hoursF), Display: pick(total > 0, shared.Pct(hoursF), "no steps mapped"), Hue: "var(--bn-text-2)"},
		{Label: "Leak recovered (" + usd(recovered) + "/" + usd(leak) + " mo)", Frac: shared.F(leakF), Display: pick(leak > 0, shared.Pct(leakF), "no leaks mapped"), Hue: "var(--bn-text)"},
	}

	start := shared.WindowStart(now, days)
	var stamps []string
	rhythm := make([]shared.SeriesPoint, 7)
	for i, d := range weekdays {
		rhythm[i].Label = d
	}
	inWindow, failed := 0, 0
	for _, r := range runs {
		t, ok := shared.ParseISO(r.StartedAt)
		if !ok || t.Before(start) || t.After(now) {
			continue
		}
		stamps = append(stamps, r.StartedAt)
		rhythm[(int(t.In(time.Local).Weekday())+6)%7].Count++
		inWindow++
		if !r.OK {
			failed++
		}
	}
	series := shared.DailySeries(stamps, days, now)

	names := make([]string, len(wfs))
	for i, w := range wfs {
		names[i] = w.Name
	}
	labels := shared.ShortLabels(names, 0)
	per := make([]shared.SeriesPoint, len(wfs))
	for i := range wfs {
		// DotMatrix counts are whole; the hours are shown to the hour.
		per[i] = shared.SeriesPoint{Label: labels[i], Count: int(math.Round(hours(stats[i].ManualHours)))}
	}

	var needs []string
	for i, s := range states {
		if s == "overdue" || s == "failing" {
			needs = append(needs, jobs[i].Description)
		}
	}
	insight := shared.Insight{Value: len(needs), Frac: shared.Frac(len(needs), c.Enabled)}
	switch {
	case len(needs) > 0:
		insight.Headline = strconv.Itoa(c.Overdue) + " overdue · " + strconv.Itoa(c.Failing) + " failing."
		insight.Body = strings.Join(needs[:min(3, len(needs))], " · ")
	case len(jobs) > 0:
		insight.Headline = "Every enabled task ran on its slot."
		insight.Body = strconv.Itoa(c.Healthy) + " healthy, nothing waiting on you."
	default:
		insight.Headline = "Every enabled task ran on its slot."
		insight.Body = "No scheduled tasks yet."
	}

	return WorkflowsVolume{
		Headline: len(jobs), Counts: c, Chips: chips,
		Caption: strconv.Itoa(c.Enabled) + " enabled · " + strconv.Itoa(runsAll) + " runs recorded",
		Meters:  meters,
		Foot:    strconv.Itoa(len(wfs)) + " workflows · " + strconv.Itoa(steps) + " steps · " + strconv.Itoa(len(tools)) + " tools",
		Series:  series, RunsInWindow: inWindow, FailedInWindow: failed, Rhythm: rhythm,
		Load:    Load{ManualHours: manual, AgentHours: agent, PerWorkflow: per},
		Insight: insight,
	}
}
