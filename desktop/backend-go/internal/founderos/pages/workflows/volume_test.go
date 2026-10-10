package workflows

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
	"github.com/rhl/businessos-backend/internal/founderos/pages/shared"
)

// Ported from FounderOS v1 tests/workflows-volume.test.ts and workflow-stats.test.ts.

var vNow = time.Date(2026, 9, 24, 18, 0, 0, 0, time.Local)

func fp(f float64) *float64 { return &f }

func step(id, kind string, hours float64, tools []string) osdata.WorkflowStep {
	return osdata.WorkflowStep{ID: id, Title: "t", OwnerKind: kind, Owner: "the operator", HoursPerWeek: hours, Tools: tools}
}

func fixtureWorkflows() []osdata.Workflow {
	a := step("a", "human", 6, []string{"gmail"})
	a.LeakUSD = fp(1000)
	b := step("b", "agent", 4, []string{"gmail", "slack"})
	b.Automation = &osdata.Automation{Title: "x", State: "live", RecoveredUSD: 250}
	c := step("c", "human", 10, []string{"notion"})
	c.Automation = &osdata.Automation{Title: "y", State: "suggested", RecoveredUSD: 900}
	return []osdata.Workflow{
		{ID: "w1", Name: "Vantage sales machine", Steps: []osdata.WorkflowStep{a, b}},
		{ID: "w2", Name: "Vantage delivery", Order: 1, Steps: []osdata.WorkflowStep{c}},
	}
}

func job(desc string, mod func(*shared.JobRow)) shared.JobRow {
	t := true
	j := shared.JobRow{Description: desc, Enabled: true, LastOK: &t}
	if mod != nil {
		mod(&j)
	}
	return j
}

func runAt(iso string, ok bool) osdata.CronRun {
	t, _ := time.ParseInLocation("2006-01-02T15:04:05", iso, time.Local)
	return osdata.CronRun{StartedAt: t.UTC().Format(time.RFC3339), OK: ok}
}

func TestWorkflowStats(t *testing.T) {
	s := Stats(fixtureWorkflows()[0])
	if s.ManualHours != 6 || s.AgentHours != 4 || s.LeakUSD != 1000 || s.LiveReturnsUSD != 250 || s.HumanSteps != 1 || s.AgentSteps != 1 || s.ToolCount != 2 || s.AutomationCount != 1 {
		t.Fatalf("stats %+v", s)
	}
	if s := Stats(fixtureWorkflows()[1]); s.SuggestedReturnsUSD != 900 {
		t.Fatalf("suggested %+v", s)
	}
	if s := Stats(osdata.Workflow{}); s != (WorkflowStats{}) {
		t.Fatalf("empty %+v", s)
	}
}

func TestVolume(t *testing.T) {
	f := false
	jobs := []shared.JobRow{
		job("Healthy one", func(j *shared.JobRow) { j.Runs, j.OK = 5, 5 }),
		job("Healthy two", func(j *shared.JobRow) { j.Runs, j.OK = 3, 2 }),
		job("Late heartbeat", func(j *shared.JobRow) { j.Overdue, j.Runs, j.OK = true, 2, 2 }),
		job("Broken ingest", func(j *shared.JobRow) { j.LastOK, j.Runs, j.OK = &f, 4, 1 }),
		job("Ghost agent", func(j *shared.JobRow) { j.UnknownAgent = true }),
		job("Paused digest", func(j *shared.JobRow) { j.Enabled, j.Runs, j.OK = false, 1, 1 }),
	}
	runs := []osdata.CronRun{runAt("2026-09-24T09:00:00", true), runAt("2026-09-24T10:00:00", false), runAt("2026-09-23T09:00:00", true), runAt("2026-09-10T09:00:00", true)}
	v := Volume(jobs, fixtureWorkflows(), runs, vNow, 14)
	if v.Headline != 6 || v.Counts != (Counts{Healthy: 2, Overdue: 1, Failing: 2, Paused: 1, Enabled: 5}) {
		t.Fatalf("counts %+v", v.Counts)
	}
	if fmt.Sprint(v.Chips) != "[{ok 2 healthy} {warn 1 overdue} {err 2 failing} { 1 paused}]" || v.Caption != "5 enabled · 15 runs recorded" {
		t.Fatalf("chips %v %q", v.Chips, v.Caption)
	}
	labels := []string{"Crons healthy (2/5)", "Runs OK (11/15)", "Hours carried by agents (4/20h per wk)", "Leak recovered ($250/$1,000 mo)"}
	fracs := []float64{0.4, 11.0 / 15, 0.2, 0.25}
	for i, m := range v.Meters {
		if m.Label != labels[i] || math.Abs(*m.Frac-fracs[i]) > 1e-9 {
			t.Fatalf("meter %d %+v", i, m)
		}
	}
	if v.Meters[0].Display != "40%" || v.Meters[0].Hue != "var(--bn-warn)" || v.Meters[1].Display != "11 ok · 4 failed" || v.Meters[2].Display != "20%" || v.Meters[3].Display != "25%" {
		t.Fatalf("displays %+v", v.Meters)
	}
	if v.Foot != "2 workflows · 3 steps · 3 tools" {
		t.Fatalf("foot %q", v.Foot)
	}
	if len(v.Series) != 14 || v.Series[13] != (shared.SeriesPoint{Label: "Sep 24", Count: 2}) || v.Series[12].Count != 1 || v.Series[0].Label != "Sep 11" || v.RunsInWindow != 3 || v.FailedInWindow != 1 {
		t.Fatalf("series %+v", v.Series)
	}
	if v.Rhythm[0].Label != "Mon" || v.Rhythm[2].Count != 1 || v.Rhythm[3].Count != 2 {
		t.Fatalf("rhythm %+v", v.Rhythm)
	}
	if v.Load.ManualHours != 16 || v.Load.AgentHours != 4 || fmt.Sprint(v.Load.PerWorkflow) != "[{Vantage 6} {Vantage 2 10}]" {
		t.Fatalf("load %+v", v.Load)
	}
	if v.Insight.Value != 3 || v.Insight.Headline != "1 overdue · 2 failing." || v.Insight.Body != "Late heartbeat · Broken ingest · Ghost agent" || math.Abs(v.Insight.Frac-0.6) > 1e-9 {
		t.Fatalf("insight %+v", v.Insight)
	}
}

func TestVolumeEmptyAndAllGreen(t *testing.T) {
	e := Volume(nil, nil, nil, vNow, 14)
	want := []string{"none enabled", "no runs yet", "no steps mapped", "no leaks mapped"}
	for i, m := range e.Meters {
		if *m.Frac != 0 || m.Display != want[i] {
			t.Fatalf("empty meter %+v", m)
		}
	}
	if e.Headline != 0 || len(e.Chips) != 0 || e.Chips == nil || e.Insight.Value != 0 || e.Insight.Body != "No scheduled tasks yet." {
		t.Fatalf("empty %+v", e)
	}
	g := Volume([]shared.JobRow{job("x", func(j *shared.JobRow) { j.Runs, j.OK = 2, 2 })}, nil, nil, vNow, 14)
	if *g.Meters[0].Frac != 1 || g.Meters[0].Hue != "var(--bn-ok)" || g.Insight.Headline != "Every enabled task ran on its slot." {
		t.Fatalf("green %+v", g)
	}
}
