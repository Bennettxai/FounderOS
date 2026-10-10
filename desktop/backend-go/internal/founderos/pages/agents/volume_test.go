package agentspage

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
)

// Ported from FounderOS v1 tests/agents-volume.test.ts and board-live.test.ts.

var volNow = time.Date(2026, 9, 24, 18, 0, 0, 0, time.Local)

func hoursAgo(h float64) *string {
	s := volNow.Add(-time.Duration(h * float64(time.Hour))).UTC().Format(time.RFC3339)
	return &s
}

func ptr(s string) *string { return &s }

func seat(name, status string, model *string, beat *string) paperclip.Agent {
	return paperclip.Agent{ID: strings.ReplaceAll(strings.ToLower(name), " ", "-"), Name: name, Status: status, AdapterType: ptr("claude_local"), Model: model, LastHeartbeatAt: beat}
}

func issue(status string, n int) paperclip.Issue {
	return paperclip.Issue{ID: fmt.Sprintf("i-%s-%d", status, n), Identifier: fmt.Sprintf("BEN-%d", n), Title: "task", Status: status}
}

var runSeq int

func run(agentID, status string, started *string) paperclip.Run {
	runSeq++
	return paperclip.Run{ID: fmt.Sprintf("r%d", runSeq), AgentID: agentID, Status: status, StartedAt: started}
}

func liveBoard() Board {
	return Board{
		Connected: true,
		Agents: []paperclip.Agent{
			seat("Conductor", "running", ptr("claude-opus"), hoursAgo(1)),
			seat("Tech Lead", "idle", ptr("gpt-5"), hoursAgo(2)),
			seat("Marketing Head", "paused", ptr("claude-opus"), hoursAgo(30)),
			seat("Finance Head", "error", nil, nil),
		},
		Issues: []paperclip.Issue{
			issue("in_progress", 1), issue("in_review", 2), issue("blocked", 3), issue("todo", 4), issue("todo", 5),
			issue("done", 6), issue("done", 7), issue("done", 8), issue("cancelled", 9),
		},
		Runs: []paperclip.Run{
			run("conductor", "running", hoursAgo(0.1)),
			run("conductor", "succeeded", hoursAgo(3)),
			run("conductor", "succeeded", hoursAgo(5)),
			run("tech-lead", "failed", hoursAgo(6)),
			run("tech-lead", "completed", hoursAgo(50)),
			run("finance-head", "timed_out", hoursAgo(100)),
			run("conductor", "succeeded", hoursAgo(24*20)),
			run("conductor", "succeeded", nil),
		},
	}
}

func TestVolumeCard(t *testing.T) {
	v := AgentsVolume(liveBoard(), volNow, 14)
	if v.Headline != 4 || v.Counts != (Counts{Running: 1, Idle: 1, Paused: 1, Error: 1}) {
		t.Fatalf("headline/counts: %+v", v)
	}
	want := []string{"ok:1 running", ":1 idle", "warn:1 paused", "err:1 in error"}
	for i, c := range v.Chips {
		if c.Tone+":"+c.Text != want[i] {
			t.Fatalf("chips: %+v", v.Chips)
		}
	}
	if !strings.Contains(v.Caption, "2 models") {
		t.Fatalf("caption %q", v.Caption)
	}
	labels := []string{"Seats running (1/4)", "Heartbeat in 24h (2/4)", "Runs OK · 14d (3/5)", "Tasks done (3/8)"}
	fracs := []float64{0.25, 0.5, 0.6, 3.0 / 8}
	for i, m := range v.Meters {
		if m.Label != labels[i] || math.Abs(*m.Frac-fracs[i]) > 1e-9 || !strings.HasPrefix(m.Hue, "var(--") {
			t.Fatalf("meter %d: %+v", i, m)
		}
	}
	if v.Meters[2].Display != "3 ok · 2 failed" {
		t.Fatalf("runs display %q", v.Meters[2].Display)
	}
	if v.OpenTasks != 5 || v.Foot != "5 open tasks · 4 runs in 24h" {
		t.Fatalf("foot %q open %d", v.Foot, v.OpenTasks)
	}
}

func TestVolumeActivityAndSecondRow(t *testing.T) {
	v := AgentsVolume(liveBoard(), volNow, 14)
	total := 0
	for _, s := range v.Series {
		total += s.Count
	}
	if len(v.Series) != 14 || v.Series[13].Label != "Sep 24" || total != 6 || v.RunsInWindow != 6 || v.FailedInWindow != 2 {
		t.Fatalf("series: %+v in=%d failed=%d", v.Series, v.RunsInWindow, v.FailedInWindow)
	}
	seats := fmt.Sprint(v.BySeat)
	if seats != "[{Conductor 3} {Tech 2} {Finance 1}]" {
		t.Fatalf("by seat %s", seats)
	}
	if fmt.Sprint(v.Lanes) != "[{doing 1} {review 1} {todo 2} {blocked 1} {backlog 0}]" {
		t.Fatalf("lanes %v", v.Lanes)
	}
	if v.Insight.Value != 2 || !strings.Contains(v.Insight.Headline, "2 tasks") ||
		!strings.Contains(v.Insight.Body, "1 seat in error") || !strings.Contains(v.Insight.Body, "1 failed run in 24h") ||
		math.Abs(v.Insight.Frac-0.4) > 1e-9 {
		t.Fatalf("insight %+v", v.Insight)
	}
}

func TestVolumeUnreachableBoardSaysSo(t *testing.T) {
	v := AgentsVolume(Board{}, volNow, 14)
	if v.Headline != 0 || len(v.Meters) != 0 || len(v.BySeat) != 0 || v.Insight.Value != 0 || v.Insight.Frac != 0 {
		t.Fatalf("down board: %+v", v)
	}
	if len(v.Chips) != 1 || v.Chips[0].Text != "board unreachable" || !strings.Contains(strings.ToLower(v.Insight.Headline), "unreachable") {
		t.Fatalf("down chips: %+v", v)
	}
	for _, s := range v.Series {
		if s.Count != 0 {
			t.Fatal("series drawn for a dead board")
		}
	}
}

func TestVolumeQuietBoard(t *testing.T) {
	v := AgentsVolume(Board{Connected: true, Agents: []paperclip.Agent{seat("Conductor", "idle", ptr("m"), nil)}}, volNow, 14)
	for _, m := range v.Meters {
		if *m.Frac != 0 {
			t.Fatalf("meter lit on a quiet board: %+v", m)
		}
	}
	if v.Meters[2].Display != "no finished runs" || v.Meters[3].Display != "no tasks" || !strings.Contains(strings.ToLower(v.Insight.Headline), "nothing") {
		t.Fatalf("quiet: %+v", v)
	}
}

func TestVolumeWindowSaysWhatTheRunsCover(t *testing.T) {
	var runs []paperclip.Run
	for i := 0; i < 120; i++ {
		runs = append(runs, run("sales-agent", "succeeded", hoursAgo(float64(i)*0.5)))
	}
	b := Board{Connected: true, Agents: []paperclip.Agent{seat("Sales Agent", "idle", nil, nil)}, Runs: runs}
	v := AgentsVolume(b, volNow, 14)
	if v.Window != "since Sep 22" || !strings.HasPrefix(v.Meters[2].Label, "Runs OK · since Sep 22") {
		t.Fatalf("window %q %q", v.Window, v.Meters[2].Label)
	}
	b.Runs = runs[:1]
	if v := AgentsVolume(b, volNow, 14); v.Window != "14d" {
		t.Fatalf("short page window %q", v.Window)
	}
}

func TestBoardStats(t *testing.T) {
	s := Stats(liveBoard(), volNow)
	if s != (BoardStats{Seats: 4, Running: 1, OpenTasks: 5, Runs24h: 4, Heartbeats24h: 2}) {
		t.Fatalf("stats %+v", s)
	}
}
