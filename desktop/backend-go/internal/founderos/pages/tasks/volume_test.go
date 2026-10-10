package tasks

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/tasks-volume.test.ts.

var now = time.Date(2026, 9, 24, 18, 0, 0, 0, time.Local)

func at(s string) string {
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t.UTC().Format(time.RFC3339)
}

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }

func job(mod func(*Job)) Job {
	j := Job{Description: "Stack monitor", Enabled: true, LastOK: bp(true)}
	if mod != nil {
		mod(&j)
	}
	return j
}

func task(agent, status, updated string) Task {
	return Task{AgentID: agent, Status: status, UpdatedAt: at(updated)}
}

func input() Input {
	return Input{
		Tasks: []Task{
			task("conductor", "open", "2026-09-24T09:00:00"),
			task("conductor", "doing", "2026-09-23T10:00:00"),
			task("closer", "review", "2026-09-24T09:00:00"),
			task("closer", "review", "2026-09-01T10:00:00"),
			task("scout", "done", "2026-09-24T09:00:00"),
			task("scout", "done", "2026-09-24T09:00:00"),
		},
		Issues:         []Issue{{"in_progress", sp(at("2026-09-23T09:00:00"))}, {"blocked", sp(at("2026-09-23T09:00:00"))}, {"todo", nil}, {"done", sp(at("2026-09-23T09:00:00"))}},
		BoardConnected: true,
		Jobs: []Job{
			job(func(j *Job) { j.Description, j.Runs, j.OK = "Healthy", 4, 4 }),
			job(func(j *Job) { j.Description, j.Overdue, j.Runs, j.OK = "Late heartbeat", true, 2, 2 }),
			job(func(j *Job) { j.Description, j.LastOK, j.Runs, j.OK = "Broken ingest", bp(false), 4, 1 }),
			job(func(j *Job) { j.Description, j.Enabled, j.Runs, j.OK = "Paused digest", false, 1, 1 }),
		},
		Runs: []Run{
			{at("2026-09-24T08:00:00"), true}, {at("2026-09-24T09:00:00"), false},
			{at("2026-09-23T08:00:00"), true}, {at("2026-09-01T08:00:00"), true},
		},
		AgentNames: map[string]string{"conductor": "Conductor Prime", "closer": "Closer", "scout": "Scout"},
		Now:        now, Days: 14,
	}
}

func near(a *float64, b float64) bool { return a != nil && math.Abs(*a-b) < 1e-9 }

func TestHeadlineChipsAndMeters(t *testing.T) {
	v := Volume(input())
	if v.Headline != 10 || v.Counts != (Counts{1, 1, 2, 2}) || v.Board != (BoardCounts{4, 1, 1, 1}) || v.Caption != "6 on the local kanban · 4 on the board" {
		t.Fatalf("headline %+v", v)
	}
	if fmt.Sprint(v.Chips) != "[{ 1 to do} {warn 1 in progress} {accent 2 in review} {ok 2 done}]" {
		t.Fatalf("chips %v", v.Chips)
	}
	labels := []string{"Kanban shipped (2/6)", "Waiting on review (2/4 in flight)", "Board issues in progress (1/4)", "Crons healthy (1/3)"}
	fracs := []float64{2.0 / 6, 0.5, 0.25, 1.0 / 3}
	displays := []string{"33%", "50%", "25%", "33%"}
	hues := []string{"var(--bn-ok)", "var(--bn-accent)", "var(--bn-warn)", "var(--bn-warn)"}
	for i, m := range v.Meters {
		if m.Label != labels[i] || !near(m.Frac, fracs[i]) || m.Display != displays[i] || m.Hue != hues[i] {
			t.Fatalf("meter %d %+v", i, m)
		}
	}
	if v.Foot != "4 crons · 11 runs recorded · 3 agents on the kanban" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestSeriesRhythmOwnersInsight(t *testing.T) {
	v := Volume(input())
	if len(v.Series) != 14 || v.Series[0].Label != "Sep 11" || v.Series[13].Count != 4 || v.Series[13].Label != "Sep 24" || v.Series[12].Count != 4 || v.TouchesInWindow != 8 {
		t.Fatalf("series %+v touches %d", v.Series, v.TouchesInWindow)
	}
	if v.Cron.RunsInWindow != 3 || v.Cron.FailedInWindow != 1 || fmt.Sprint(v.Cron.Rhythm) != "[{Mon 0} {Tue 0} {Wed 1} {Thu 2} {Fri 0} {Sat 0} {Sun 0}]" {
		t.Fatalf("cron %+v", v.Cron)
	}
	if fmt.Sprint(v.Owners) != "[{Conductor 2} {Closer 2}]" || v.BusiestOwner == nil || *v.BusiestOwner != (Owner{"Conductor Prime", 2}) {
		t.Fatalf("owners %v %+v", v.Owners, v.BusiestOwner)
	}
	if v.Insight.Value != 5 || v.Insight.Headline != "2 in review · 1 blocked · 2 crons late or failing." || v.Insight.Body != "Late heartbeat · Broken ingest" || math.Abs(v.Insight.Frac-0.5) > 1e-9 {
		t.Fatalf("insight %+v", v.Insight)
	}
}

func TestEmptyIsHonest(t *testing.T) {
	e := Volume(Input{BoardConnected: true, Now: now})
	if e.Headline != 0 || len(e.Chips) != 0 || len(e.Owners) != 0 || e.BusiestOwner != nil {
		t.Fatalf("empty %+v", e)
	}
	want := []string{"no tasks yet", "nothing in flight", "board empty or offline", "none enabled"}
	for i, m := range e.Meters {
		if *m.Frac != 0 || m.Display != want[i] {
			t.Fatalf("meter %d %+v", i, m)
		}
	}
	if e.Insight.Value != 0 || e.Insight.Frac != 0 || e.Insight.Headline != "Nothing is waiting on you." || e.Insight.Body != "No tasks, issues or crons yet." {
		t.Fatalf("insight %+v", e.Insight)
	}
	g := Volume(Input{Tasks: []Task{task("ghost-agent", "open", "2026-09-24T09:00:00")}, BoardConnected: true, Jobs: []Job{job(func(j *Job) { j.Runs, j.OK = 1, 1 })}, Now: now})
	if fmt.Sprint(g.Owners) != "[{ghost-agent 1}]" || !near(g.Meters[3].Frac, 1) || g.Meters[3].Hue != "var(--bn-ok)" || g.Insight.Body != "1 task open, nothing waiting on you." {
		t.Fatalf("ghost %+v", g)
	}
}

func TestOfflineBoardReadsLikeV1(t *testing.T) {
	// v1 lib/tasks-volume.ts: an unreachable board is an empty board, and the
	// meter says so honestly ("board empty or offline"), never a fake count.
	in := input()
	in.Issues, in.BoardConnected = nil, false
	v := Volume(in)
	m := v.Meters[2]
	if m.Frac == nil || *m.Frac != 0 || m.Display != "board empty or offline" || m.Label != "Board issues in progress (0/0)" || v.BoardOnline {
		t.Fatalf("offline meter %+v", m)
	}
	if v.Caption != "6 on the local kanban · 0 on the board" {
		t.Fatalf("caption %q", v.Caption)
	}
}
