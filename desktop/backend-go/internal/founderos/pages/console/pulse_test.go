package console

import (
	"math"
	"reflect"
	"testing"
	"time"
)

// Ports tests/pulse-history.test.ts, tests/home-volume.test.ts and
// tests/run-digest.test.ts from FounderOS v1.

var pulseNow = time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC) // days 07-07 … 07-13

func TestRunsPerDayBucketsIntoTheLastSevenUTCDays(t *testing.T) {
	got := PerDay([]string{
		"2026-07-13T09:00:00Z", "2026-07-13T01:00:00Z", "2026-07-12T23:00:00Z",
		"2026-07-08T10:00:00Z", "2026-07-01T10:00:00Z", // outside the window
	}, 7, pulseNow)
	if want := []int{0, 1, 0, 0, 0, 1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PerDay = %v, want %v", got, want)
	}
	if got := PerDay(nil, 7, pulseNow); !reflect.DeepEqual(got, []int{0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("empty PerDay = %v", got)
	}
	if got := PerDay([]string{"nope", "2026-07-13T00:00:00Z"}, 7, pulseNow); !reflect.DeepEqual(got, []int{0, 0, 0, 0, 0, 0, 1}) {
		t.Fatalf("unparseable PerDay = %v", got)
	}
}

func baseFacts() PulseFacts {
	return PulseFacts{ActiveAgents: 5, TotalAgents: 8, Inbound: 0, BrainConnected: true, EnginesUp: 2, EnginesTotal: 2, Health: ptr(90)}
}

func has(segs []Segment, s Segment) bool {
	for _, x := range segs {
		if x == s {
			return true
		}
	}
	return false
}

func TestStateOfWorldLeadsWithAllNominal(t *testing.T) {
	segs := StateOfWorld(baseFacts())
	if segs[0] != (Segment{"All nominal", "ok"}) {
		t.Fatalf("first = %+v", segs[0])
	}
	for _, s := range []Segment{{"5 agents live", "ok"}, {"3 idle", "dim"}, {"brain 90/100", "ok"}} {
		if !has(segs, s) {
			t.Fatalf("missing %+v in %+v", s, segs)
		}
	}
}

func TestStateOfWorldIsWorstFirst(t *testing.T) {
	f := baseFacts()
	f.FailedRuns = 2
	segs := StateOfWorld(f)
	if segs[0] != (Segment{"2 runs failed", "err"}) {
		t.Fatalf("first = %+v", segs[0])
	}
	for _, s := range segs {
		if s.Text == "All nominal" {
			t.Fatal("All nominal with failed runs")
		}
	}
	f = baseFacts()
	f.FailedRuns, f.ConnectorsDown = 1, 1
	segs = StateOfWorld(f)
	if !has(segs, Segment{"1 run failed", "err"}) || !has(segs, Segment{"1 connector down", "warn"}) {
		t.Fatalf("singulars: %+v", segs)
	}
	f = baseFacts()
	f.Inbound = 7
	if !has(StateOfWorld(f), Segment{"7 inbound need reply", "accent"}) {
		t.Fatal("inbound segment missing")
	}
}

func TestStateOfWorldReportsTheOptimalEngineHonestly(t *testing.T) {
	f := baseFacts()
	f.BrainConnected, f.EnginesUp, f.Health = false, 0, nil
	if segs := StateOfWorld(f); !has(segs, Segment{"Optimal Engine offline", "err"}) {
		t.Fatalf("offline: %+v", segs)
	}
	// Prod: "G-Brain degraded 62/100" under 70; the engine's score reads the same.
	f = baseFacts()
	f.EnginesUp, f.Health = 1, ptr(50)
	segs := StateOfWorld(f)
	if !has(segs, Segment{"Optimal Engine degraded 50/100", "warn"}) {
		t.Fatalf("degraded: %+v", segs)
	}
	if has(segs, Segment{"brain 50/100", "ok"}) {
		t.Fatal("a degraded brain is not also stated as ok")
	}
	f = baseFacts()
	f.BrainConnected, f.EnginesUp, f.EnginesTotal, f.Health = false, 0, 0, nil
	if segs := StateOfWorld(f); !has(segs, Segment{"Optimal Engine not configured", "warn"}) {
		t.Fatalf("no engines: %+v", segs)
	}
}

// home-volume.test.ts: NOW is local 2026-09-24 18:00.
func homeRuns(loc *time.Location) ([]Run, time.Time) {
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, loc)
	at := func(h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	return []Run{
		{AgentID: "a", OK: true, FinishedAt: at(9), StartedAt: at(9)},
		{AgentID: "a", OK: true, FinishedAt: at(10), StartedAt: at(10)},
		{AgentID: "b", OK: false, FinishedAt: at(11), StartedAt: at(11)},
		{AgentID: "c", OK: true, FinishedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, loc), StartedAt: time.Date(2026, 9, 23, 12, 0, 0, 0, loc)},
	}, time.Date(2026, 9, 24, 18, 0, 0, 0, loc)
}

func ptr[T any](v T) *T { return &v }

func TestOperatingVolumeHeadlineAndMeters(t *testing.T) {
	runs, now := homeRuns(time.Local)
	v := OperatingVolume(VolumeInput{Connected: 19, TotalConnections: 24, ActiveAgents: ptr(19), TotalAgents: ptr(32), EnginesUp: 2, EnginesTotal: 2, Health: ptr(90), Runs: runs, Now: now})
	if v.RunsToday != 3 || v.FailedToday != 1 || v.AgentsToday != 2 {
		t.Fatalf("headline = %+v", v)
	}
	labels := []string{}
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
	}
	if want := []string{"Systems connected (19/24)", "Agents live (19/32)", "Runs OK today (2/3)", "Optimal Engine health"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("labels = %v", labels)
	}
	if math.Abs(*v.Meters[0].Frac-19.0/24) > 1e-9 || math.Abs(*v.Meters[2].Frac-2.0/3) > 1e-9 {
		t.Fatalf("fracs = %v %v", *v.Meters[0].Frac, *v.Meters[2].Frac)
	}
	if *v.Meters[3].Frac != 0.9 || v.Meters[3].Display != "90 / 100" {
		t.Fatalf("brain meter = %+v", v.Meters[3])
	}
	if v.Meters[2].Display != "2 ok · 1 failed" {
		t.Fatalf("runs display = %q", v.Meters[2].Display)
	}
}

func TestOperatingVolumeEmptyReadsEmptyAndUnknownReadsUnknown(t *testing.T) {
	_, now := homeRuns(time.Local)
	e := OperatingVolume(VolumeInput{Now: now})
	if *e.Meters[0].Frac != 0 || *e.Meters[2].Frac != 0 || e.Meters[2].Display != "no runs yet" {
		t.Fatalf("empty meters = %+v", e.Meters)
	}
	// The roster could not be read: unknown, not zero.
	if e.Meters[1].Frac != nil || e.Meters[1].Display != "unknown" {
		t.Fatalf("unknown roster meter = %+v", e.Meters[1])
	}
	if e.Meters[3].Frac != nil || e.Meters[3].Display != "not configured" {
		t.Fatalf("no engines meter = %+v", e.Meters[3])
	}
	off := OperatingVolume(VolumeInput{EnginesUp: 0, EnginesTotal: 2, Now: now})
	if *off.Meters[3].Frac != 0 || off.Meters[3].Display != "offline" {
		t.Fatalf("offline meter = %+v", off.Meters[3])
	}
}

func TestDailySeriesLabelsLocalDaysAndKeepsQuietOnes(t *testing.T) {
	loc := time.Local
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, loc)
	now := time.Date(2026, 9, 24, 18, 0, 0, 0, loc)
	s := DailySeries([]time.Time{day.Add(9 * time.Hour), day.Add(10 * time.Hour), time.Date(2026, 9, 22, 12, 0, 0, 0, loc), now.Add(time.Hour)}, 3, now)
	want := []Point{{"Sep 22", 1}, {"Sep 23", 0}, {"Sep 24", 2}}
	if !reflect.DeepEqual(s, want) {
		t.Fatalf("DailySeries = %+v", s)
	}
}

func TestSourceMixFixedOrderZerosKept(t *testing.T) {
	got := SourceMix([]Item{{Source: "email"}, {Source: "slack"}, {Source: "email"}})
	want := []Point{{"email", 2}, {"whatsapp", 0}, {"slack", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SourceMix = %+v", got)
	}
}

func TestHomeAttention(t *testing.T) {
	a := HomeAttention(3, 1, 2, 6)
	if a.Count != 6 || a.Headline != "3 inbound · 1 failed run · 2 connectors down" || math.Abs(a.Frac-0.5) > 1e-9 {
		t.Fatalf("attention = %+v", a)
	}
	calm := HomeAttention(0, 0, 0, 4)
	if calm.Count != 0 || calm.Frac != 0 || calm.Headline != "Nothing is waiting on you." {
		t.Fatalf("calm = %+v", calm)
	}
}

func TestInboundLast24hAndMergeFeed(t *testing.T) {
	items := []Item{
		{Source: "email", TS: "2026-07-13T11:00:00Z"},
		{Source: "slack", TS: "2026-07-12T11:59:00Z"}, // 24h01m ago
		{Source: "whatsapp", TS: "bad"},
		{Source: "email", TS: "2026-07-13T13:00:00Z"}, // future
	}
	if n := InboundLast24h(items, pulseNow); n != 1 {
		t.Fatalf("InboundLast24h = %d", n)
	}
	merged := MergeFeed(items, 2)
	if len(merged) != 2 || merged[0].TS != "2026-07-13T13:00:00Z" || merged[1].TS != "2026-07-13T11:00:00Z" {
		t.Fatalf("MergeFeed = %+v", merged)
	}
}

func TestCollapseRunsFoldsRepeatsAndKeepsTheNewest(t *testing.T) {
	mk := func(id, agent, summary string) Run { return Run{ID: id, AgentID: agent, Summary: summary, OK: true} }
	out := CollapseRuns([]Run{
		mk("9", "sales-calls-data", "Recorders: 20 recordings"),
		mk("8", "sales-calls-data", "Recorders: 20 recordings"),
		mk("7", "crm-pulse", "connected"),
		mk("6", "sales-calls-data", "Recorders: 20 recordings"),
	})
	if len(out) != 2 || out[0].ID != "9" || out[0].Repeat != 3 || out[1].ID != "7" || out[1].Repeat != 1 {
		t.Fatalf("CollapseRuns = %+v", out)
	}
	failed := Run{ID: "5", AgentID: "sales-calls-data", Summary: "Recorders: 20 recordings", OK: false}
	if got := CollapseRuns([]Run{mk("9", "sales-calls-data", "Recorders: 20 recordings"), failed}); len(got) != 2 {
		t.Fatalf("ok and failed runs must not fold together: %+v", got)
	}
}
