package analytics

import (
	"reflect"
	"testing"
)

// Ported from FounderOS v1 tests/analytics-volume.test.ts, tests/analytics.test.ts,
// tests/operating-metrics.test.ts and tests/growth*.test.ts.

func run(agent string, ok bool, day string) Run {
	return Run{AgentID: agent, OK: ok, StartedAt: day + "T10:00:00Z"}
}

func i64(v int64) *int64   { return &v }
func f(v float64) *float64 { return &v }

func base() VolumeInput {
	return VolumeInput{
		Channels: []Channel{{"instagram", "Instagram", i64(6000)}, {"youtube", "YouTube", i64(2000)}, {"tiktok", "TikTok", nil}},
		Subs:     i64(2000),
		Growth7d: f(2.5),
		Live:     6,
		Pending:  []Pending{{"Stripe MRR", "pending creds"}, {"Typeform leads", "pending creds"}},
		Runs: []Run{
			run("scout", true, "2026-09-21"), run("scout", true, "2026-09-21"), run("scout", false, "2026-09-22"),
			run("writer", true, "2026-09-23"), run("ledger", true, "2026-09-24"), run("ghost", true, "2026-09-24"),
			run("scout", true, "2026-07-01"),
		},
		AgentNames: map[string]string{"scout": "Scout", "writer": "Writer", "ledger": "Ledger", "ghost": "Ghost"},
		Today:      "2026-09-24",
	}
}

func TestVolumeReach(t *testing.T) {
	v := VolumeOf(base())
	if v.Reach != 10000 || v.Headline != "10,000" || v.Caption != "total reach across 3 channels" || v.Foot != "reach = followers + email list · 6 of 8 metrics live" {
		t.Fatalf("v = %+v", v)
	}
	if !reflect.DeepEqual(v.Chips, []Chip{{Tone: "ok", Text: "+2.5% 7d"}, {Text: "2,000 email subs"}}) {
		t.Fatalf("chips = %+v", v.Chips)
	}
	labels := []string{}
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Instagram", "YouTube", "Email list"}) || *v.Meters[0].Frac != 0.6 || v.Meters[0].Display != "6,000 · 60%" {
		t.Fatalf("meters = %+v", v.Meters)
	}
	many := base()
	many.Subs = nil
	many.Channels = nil
	for n := int64(1); n <= 5; n++ {
		many.Channels = append(many.Channels, Channel{"c", "C" + string(rune('0'+n)), i64(n * 100)})
	}
	mv := VolumeOf(many)
	if mv.Meters[3].Label != "Other channels (2)" || *mv.Meters[3].Frac != 300.0/1500 {
		t.Fatalf("other = %+v", mv.Meters)
	}
}

func TestVolumeRunsRhythmInsight(t *testing.T) {
	v := VolumeOf(base())
	if v.Runs.Total != 7 || v.Runs.OK != 6 || v.Runs.Failed != 1 || *v.Runs.OKPct != 86 || v.Runs.Agents != 4 {
		t.Fatalf("runs = %+v", v.Runs)
	}
	labels := []string{}
	for _, m := range v.Runs.Meters {
		labels = append(labels, m.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Scout (4)", "Writer (1)", "Ledger (1)", "Ghost (1)"}) {
		t.Fatalf("run meters = %v", labels)
	}
	counts := []int{}
	for _, d := range v.Rhythm {
		counts = append(counts, d.Count)
	}
	if v.Rhythm[0].Label != "Mon" || !reflect.DeepEqual(counts, []int{2, 1, 1, 2, 0, 0, 0}) || v.RhythmTotal != 6 {
		t.Fatalf("rhythm = %+v", v.Rhythm)
	}
	if v.Insight.Value != 2 || v.Insight.Headline != "2 metrics waiting on credentials" || v.Insight.Body != "Stripe MRR · Typeform leads" || v.Insight.Frac != 0.75 {
		t.Fatalf("insight = %+v", v.Insight)
	}
	shrink := base()
	shrink.Growth7d = f(-1.25)
	if c := VolumeOf(shrink).Chips[0]; c != (Chip{Tone: "err", Text: "-1.3% 7d"}) {
		t.Fatalf("shrink chip = %+v", c)
	}
}

func TestVolumeEmptyIsHonest(t *testing.T) {
	e := VolumeOf(VolumeInput{Today: "2026-09-24"})
	if e.Reach != 0 || len(e.Chips) != 0 || e.Chips == nil || len(e.Meters) != 0 || e.Runs.Total != 0 || e.Runs.OKPct != nil || e.RhythmTotal != 0 {
		t.Fatalf("empty = %+v", e)
	}
	if e.Insight.Value != 0 || e.Insight.Frac != 0 || e.Insight.Headline != "Every metric is live" || e.Caption != "no audience snapshots yet" {
		t.Fatalf("insight = %+v", e.Insight)
	}
}

func TestRunVolumeAndWithin(t *testing.T) {
	runs := []Run{run("a", true, "2026-09-24"), run("a", true, "2026-09-24"), run("a", true, "2026-09-20"), run("a", true, "2026-08-01")}
	v := RunVolume(runs, "2026-09-24", 7)
	if len(v) != 7 || v[0].Date != "2026-09-18" || v[6].Count != 2 || v[2].Count != 1 || v[1].Count != 0 {
		t.Fatalf("volume = %+v", v)
	}
	if RunsWithin(runs, "2026-09-24", 7) != 3 {
		t.Fatal("within")
	}
}

func TestSplitMetrics(t *testing.T) {
	live, pending := SplitMetrics([]MetricInput{
		{ID: "audience", Value: f(12000), Delta: 3.6, DeltaPct: true},
		{ID: "dictations"},
		{ID: "stripe", Value: f(0)},
		{ID: "brain", Value: f(10)},
	})
	if len(live) != 2 || live[0].ID != "audience" || live[0].Delta != 3.6 || !live[0].DeltaPct || live[1].ID != "brain" || live[1].Delta != 0 {
		t.Fatalf("live = %+v", live)
	}
	if len(pending) != 2 || pending[0].ID != "dictations" || pending[0].Value != 0 || pending[0].Live {
		t.Fatalf("pending = %+v", pending)
	}
}

func TestSparkSeries(t *testing.T) {
	if got := SparkSeries([]float64{1, 2, 3}, "x", 3); !reflect.DeepEqual(got, []float64{1, 2, 3}) {
		t.Fatal("real history wins once there are two points")
	}
	a, b := SparkSeries(nil, "leads", 50), SparkSeries([]float64{9}, "leads", 50)
	if len(a) != 7 || !reflect.DeepEqual(a, b) {
		t.Fatalf("placeholder is deterministic: %v %v", a, b)
	}
}

func TestGrowth(t *testing.T) {
	pts := []Point{{"2026-09-01", 100}, {"2026-09-17", 110}, {"2026-09-24", 121}}
	if g := GrowthOver(pts, 7); g == nil || *g < 9.99 || *g > 10.01 {
		t.Fatalf("7d = %v", g)
	}
	if GrowthOver(pts[1:], 30) != nil {
		t.Fatal("history too short is nil, never a fake zero")
	}
	if g := GrowthAllTime(pts); g == nil || *g < 20.99 || *g > 21.01 {
		t.Fatalf("all = %v", g)
	}
	aud := AudienceGrowthPct([][]Point{pts, {{"2026-09-24", 5}}}, 7)
	if aud == nil || *aud < 9.99 || *aud > 10.01 {
		t.Fatalf("channels without a baseline are left out of both sides: %v", aud)
	}
	if AudienceGrowthPct([][]Point{{{"2026-09-24", 5}}}, 7) != nil {
		t.Fatal("no qualifying channel is nil")
	}
}
