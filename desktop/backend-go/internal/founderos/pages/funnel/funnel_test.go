package funnel

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/funnel.test.ts (the pure stage math).

func journey(id, firstChannel, status string, amount *float64) Journey {
	source := "trakyo"
	if firstChannel == "ads" {
		source = "meta-ads"
	}
	return Journey{
		ID: id, Name: id, Venture: "vantage", Status: status, AmountUSD: amount,
		Relationship: "warm", Likelihood: 50, CreatedAt: "2026-06-01",
		Touches: []Touch{{ID: id + "-t1", ContactID: id, Seq: 1, Stage: "first_touch", Channel: firstChannel, Label: "first", Source: source, At: "2026-06-01"}},
	}
}

func f64(v float64) *float64 { return &v }

func TestSummaryCountsReachedStagesSplitAndConversion(t *testing.T) {
	s := Summarize([]Journey{
		journey("j1", "organic", "converted", f64(1000)),
		journey("j2", "ads", "converted", f64(500)),
		journey("j3", "organic", "opted_in", nil),
		journey("j4", "ads", "engaged", nil),
	})
	if s.Clients != 4 || s.Converted != 2 || s.RevenueUSD != 1500 {
		t.Fatalf("summary = %+v", s)
	}
	ids := []string{}
	for _, st := range s.Stages {
		ids = append(ids, st.Stage)
	}
	if !reflect.DeepEqual(ids, []string{"first_touch", "engaged", "nurtured", "opted_in", "converted"}) {
		t.Fatalf("stages = %v", ids)
	}
	by := map[string]StageRow{}
	for _, st := range s.Stages {
		by[st.Stage] = st
	}
	if r := by["first_touch"]; r.Total != 4 || r.Organic != 2 || r.Ads != 2 || r.ConversionFromPrev != nil {
		t.Fatalf("first_touch = %+v", r)
	}
	if r := by["engaged"]; r.Total != 4 || *r.ConversionFromPrev != 100 {
		t.Fatalf("engaged = %+v", r)
	}
	// v1's five stages: opted_in is its own column between nurtured and converted
	if r := by["nurtured"]; r.Total != 3 || r.Organic != 2 || r.Ads != 1 || *r.ConversionFromPrev != 75 {
		t.Fatalf("nurtured = %+v", r)
	}
	if r := by["opted_in"]; r.Total != 3 || *r.ConversionFromPrev != 100 {
		t.Fatalf("opted_in = %+v", r)
	}
	if r := by["converted"]; r.Total != 2 || *r.ConversionFromPrev != 66.7 {
		t.Fatalf("converted = %+v", r)
	}
}

func TestSummaryGuardsZeroDivision(t *testing.T) {
	s := Summarize(nil)
	if s.Clients != 0 || s.RevenueUSD != 0 {
		t.Fatalf("summary = %+v", s)
	}
	for _, st := range s.Stages {
		if st.Total != 0 || st.ConversionFromPrev != nil {
			t.Fatalf("stage = %+v", st)
		}
	}
}

func lastTouched(status, at string) Journey {
	return Journey{ID: "jm", Name: "jm", Venture: "vantage", Status: status, Relationship: "warm", Likelihood: 50, CreatedAt: at,
		Touches: []Touch{{ID: "jm-t1", ContactID: "jm", Seq: 1, Stage: "first_touch", Channel: "organic", Label: "x", Source: "trakyo", At: at}}}
}

func daysAgo(now time.Time, d int) string {
	return now.Add(-time.Duration(d) * 24 * time.Hour).UTC().Format("2006-01-02")
}

func TestJourneyMetaStates(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		status string
		days   int
		want   string
	}{
		{"converted", 30, "converted"},
		{"opted_in", 8, "stalled"},
		{"engaged", 7, "active"}, // stall needs MORE than a week
		{"first_touch", 60, "active"},
		{"engaged", 91, "decayed"},
		{"first_touch", 91, "decayed"},
	}
	for _, c := range cases {
		m := MetaOf(lastTouched(c.status, daysAgo(now, c.days)), now)
		if m.State != c.want || m.DaysSinceLastTouch != c.days {
			t.Errorf("%s at %dd = %+v, want %s", c.status, c.days, m, c.want)
		}
	}
}

func TestSplitSeparatesTheArchive(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	live, gone := lastTouched("engaged", daysAgo(now, 3)), lastTouched("engaged", daysAgo(now, 120))
	gone.ID = "gone"
	active, archived := Split([]Journey{live, gone}, now)
	if len(active) != 1 || len(archived) != 1 || archived[0].ID != "gone" {
		t.Fatalf("active=%v archived=%v", active, archived)
	}
}

func TestDecayFactorRampsAndClamps(t *testing.T) {
	if DecayFactor(DecayFadeStart, "engaged") != 0 || DecayFactor(0, "engaged") != 0 {
		t.Fatal("neutral through the fade start")
	}
	mid := DecayFactor((DecayFadeStart+DecayDays)/2, "engaged")
	if mid < 0.45 || mid > 0.55 {
		t.Fatalf("mid = %v", mid)
	}
	if DecayFactor(500, "engaged") != 1 || DecayFactor(500, "converted") != 0 {
		t.Fatal("clamps at 1; converted never decays")
	}
}

func TestSpaceModelHubsAndIdentity(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	j := journey("full", "organic", "converted", f64(900))
	j.Person, j.Email = sp("Ana"), sp("ana@x.com")
	j.Likelihood = 100
	j.Touches = []Touch{
		{Stage: "first_touch", At: daysAgo(now, 9)}, {Stage: "engaged", At: daysAgo(now, 8)}, {Stage: "engaged", At: daysAgo(now, 7)},
		{Stage: "opted_in", At: daysAgo(now, 5)}, {Stage: "converted", At: daysAgo(now, 2)},
	}
	n := SpaceModel([]Journey{j}, now)[0]
	// v1 five hubs: opted_in is hub 3, converted hub 4 (nurtured skipped)
	if !reflect.DeepEqual(n.Hubs, []int{0, 1, 3, 4}) || n.CurrentHub != 4 || n.State != "converted" {
		t.Fatalf("node = %+v", n)
	}
	if *n.Person != "Ana" || *n.Email != "ana@x.com" || n.Radius != 5.5 || n.Decay != 0 {
		t.Fatalf("node = %+v", n)
	}
	empty := journey("e", "organic", "first_touch", nil)
	empty.Touches = nil
	if got := SpaceModel([]Journey{empty}, now)[0]; !reflect.DeepEqual(got.Hubs, []int{0}) || got.Radius != 2.5+0.5*3 {
		t.Fatalf("empty = %+v", got)
	}
	if len(SpaceModel(nil, now)) != 0 {
		t.Fatal("empty model")
	}
}

func lead(id string, now time.Time, likelihood int, status string, quiet int) Journey {
	return Journey{ID: id, Name: id, Venture: "vantage", Status: status, Relationship: "warm", Likelihood: likelihood,
		CreatedAt: daysAgo(now, quiet+10),
		Touches:   []Touch{{ID: id + "-t1", ContactID: id, Seq: 1, Stage: "engaged", Channel: "crm", Label: "x", Source: "attio", At: daysAgo(now, quiet)}}}
}

func ids(js []Journey) []string {
	out := []string{}
	for _, j := range js {
		out = append(out, j.ID)
	}
	return out
}

func TestAttentionQueue(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	q := Attention([]Journey{
		lead("cool", now, 40, "engaged", 2),
		lead("hot-fresh", now, 90, "engaged", 1),
		lead("hot-later", now, 80, "engaged", 5),
		lead("hot-a", now, 85, "engaged", 2),
		lead("hot-b", now, 75, "engaged", 3),
		lead("hot-c", now, 71, "engaged", 4),
		lead("won", now, 95, "converted", 1),
		lead("dying", now, 95, "engaged", 30),
		lead("fading-inbound", now, 95, "first_touch", 30),
	}, now)
	if got := ids(q.PushNow); !reflect.DeepEqual(got, []string{"hot-fresh", "hot-a", "hot-b", "hot-c"}) {
		t.Fatalf("pushNow = %v", got)
	}
	q = Attention([]Journey{
		lead("fine", now, 90, "engaged", 3), lead("save-1", now, 88, "engaged", 25), lead("save-2", now, 70, "engaged", 40),
		lead("save-3", now, 55, "engaged", 30), lead("save-4", now, 50, "engaged", 22), lead("save-5", now, 20, "engaged", 35),
		lead("gone", now, 99, "engaged", 120),
	}, now)
	if got := ids(q.SaveNow); !reflect.DeepEqual(got, []string{"save-1", "save-2", "save-3", "save-4"}) {
		t.Fatalf("saveNow = %v", got)
	}
	for days := StallDays + 1; days <= DecayFadeStart; days++ {
		q := Attention([]Journey{lead(fmt.Sprint("q", days), now, 60, "engaged", days)}, now)
		if len(q.SaveNow) != 1 || len(q.PushNow) != 0 {
			t.Fatalf("%dd quiet: %+v", days, q)
		}
	}
	for _, status := range []string{"engaged", "first_touch"} {
		for days := 0; days <= 120; days++ {
			q := Attention([]Journey{lead("x", now, 90, status, days)}, now)
			if len(q.PushNow)+len(q.SaveNow) > 1 {
				t.Fatalf("%s at %dd in both rails", status, days)
			}
		}
	}
	if q := Attention(nil, now); len(q.PushNow) != 0 || len(q.SaveNow) != 0 || q.PushNow == nil || q.SaveNow == nil {
		t.Fatalf("empty = %+v (must be [] not null)", q)
	}
}

func TestStageLabels(t *testing.T) {
	got := []string{}
	for _, s := range Stages {
		got = append(got, s.Label)
	}
	// FounderOS v1 lib/funnel.ts FUNNEL_STAGES
	if !reflect.DeepEqual(got, []string{"First touch", "Engaged", "Nurtured", "Opted in", "Converted"}) {
		t.Fatalf("labels = %v", got)
	}
	for _, s := range Stages {
		if StageLabel[s.ID] != s.Label {
			t.Fatalf("StageLabel[%s] = %q", s.ID, StageLabel[s.ID])
		}
	}
}
