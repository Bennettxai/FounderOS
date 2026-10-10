package content

import (
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// Ported from FounderOS v1 tests/content-volume.test.ts (2026-09-24).
const today = "2026-09-24"

func baseInput() VolumeInput {
	return VolumeInput{
		Crew: []Agent{
			{ID: "social-agent", Name: "Social Agent", Status: "active"},
			{ID: "postly-publisher", Name: "Zernio Publisher", Status: "active"},
			{ID: "adsmith-creative", Name: "Arcads Creative", Status: "planned"},
		},
		Runs: []Run{
			{AgentID: "social-agent", StartedAt: "2026-09-23T10:00:00Z", OK: true},
			{AgentID: "social-agent", StartedAt: "2026-09-20T10:00:00Z", OK: false},
			{AgentID: "postly-publisher", StartedAt: "2026-09-22T10:00:00Z", OK: true},
			{AgentID: "postly-publisher", StartedAt: "2026-07-01T10:00:00Z", OK: true}, // outside
		},
		LeadMagnets: []LeadMagnet{{Name: "Hook vault", Status: "live"}, {Name: "VSL teardown", Status: "live"}, {Name: "Offer doc", Status: "draft"}},
		RecentCount: 3,
		PostDays: []zernio.PostDay{
			{Date: "2026-09-22", Platforms: []string{"instagram", "tiktok"}},
			{Date: "2026-09-22", Platforms: []string{"youtube"}},
			{Date: "2026-09-15T12:00:00Z", Platforms: []string{"instagram"}},
			{Date: "2026-06-01", Platforms: []string{"instagram"}},
		},
		PostsKnown: true,
		Today:      today,
	}
}

func meterBy(ms []pagekit.Meter, label string) *pagekit.Meter {
	for i := range ms {
		if ms[i].Label == label {
			return &ms[i]
		}
	}
	return nil
}

func near(a, b float64) bool { d := a - b; return d < 1e-9 && d > -1e-9 }

func TestVolumeHeadlineChipsCaption(t *testing.T) {
	v := Volume(baseInput())
	if v.Headline != 3 || v.PostsInWindow != 3 || v.ActiveDays != 2 {
		t.Fatalf("headline %d posts %d active %d", v.Headline, v.PostsInWindow, v.ActiveDays)
	}
	if len(v.Chips) != 2 || v.Chips[0].Text != "2 active days" || v.Chips[1].Text != "2 magnets live" {
		t.Fatalf("chips %+v", v.Chips)
	}
	if v.Caption != "posts out through Zernio, last 30 days" {
		t.Fatalf("caption %q", v.Caption)
	}
}

func TestVolumeMetersAreHonestFractions(t *testing.T) {
	v := Volume(baseInput())
	for label, want := range map[string]float64{
		"Active posting days (2/30)": 2.0 / 30, "Lead magnets live (2/3)": 2.0 / 3,
		"Crew active (2/3)": 2.0 / 3, "Crew runs OK (2/3)": 2.0 / 3,
	} {
		m := meterBy(v.Meters, label)
		if m == nil || m.Frac == nil || !near(*m.Frac, want) {
			t.Fatalf("meter %q = %+v", label, m)
		}
	}
	if m := meterBy(v.Meters, "Crew runs OK (2/3)"); m.Display != "2 ok · 1 failed" {
		t.Fatalf("display %q", m.Display)
	}
	if v.Foot != "3 agents · 3 lead magnets · 3 recent posts pulled" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestVolumeStepLineEndsToday(t *testing.T) {
	v := Volume(baseInput())
	if len(v.Series) != 30 || v.Series[29].Label != "Sep 24" {
		t.Fatalf("series %d %+v", len(v.Series), v.Series[29])
	}
	total := 0.0
	for _, s := range v.Series {
		total += s.Count
		if s.Label == "Sep 22" && s.Count != 2 {
			t.Fatalf("sep 22 = %v", s.Count)
		}
	}
	if total != 3 {
		t.Fatalf("total %v", total)
	}
}

func TestVolumeCrewRuns(t *testing.T) {
	v := Volume(baseInput())
	want := []pagekit.Point{{Label: "Social", Count: 2}, {Label: "Zernio", Count: 1}, {Label: "Arcads", Count: 0}}
	if len(v.CrewRuns) != 3 {
		t.Fatalf("%+v", v.CrewRuns)
	}
	for i := range want {
		if v.CrewRuns[i] != want[i] {
			t.Fatalf("%+v", v.CrewRuns)
		}
	}
	if v.RunsInWindow != 3 {
		t.Fatal(v.RunsInWindow)
	}
}

func TestVolumeMagnetCounts(t *testing.T) {
	in := baseInput()
	in.LeadMagnets = []LeadMagnet{{Status: "live"}, {Status: "draft"}, {Status: "paused"}, {Status: "draft"}}
	if m := Volume(in).Magnets; m != (MagnetCounts{Total: 4, Live: 1, Draft: 2, Paused: 1}) {
		t.Fatalf("%+v", m)
	}
}

func TestVolumeInsightDaysSinceLastPost(t *testing.T) {
	v := Volume(baseInput())
	if v.Insight.Value != 2 || v.Insight.Display != "" || v.Insight.Headline != "2 days since the last post went out." || !near(v.Insight.Frac, 2.0/30) {
		t.Fatalf("%+v", v.Insight)
	}
	in := baseInput()
	in.PostDays = []zernio.PostDay{{Date: today, Platforms: []string{"instagram"}}}
	if v := Volume(in); v.Insight.Value != 0 || v.Insight.Headline != "Posted today." {
		t.Fatalf("%+v", v.Insight)
	}
}

func TestVolumeEmptyStaysHonest(t *testing.T) {
	in := baseInput()
	in.Runs, in.LeadMagnets, in.RecentCount, in.PostDays, in.Crew = nil, nil, 0, nil, nil
	v := Volume(in)
	if v.Headline != 0 || len(v.Chips) != 0 || len(v.Meters) != 0 || len(v.CrewRuns) != 0 {
		t.Fatalf("%+v", v)
	}
	if v.Chips == nil || v.Meters == nil || v.CrewRuns == nil {
		t.Fatal("empty lists must marshal as [] not null")
	}
	if v.Caption != "no posts on record in the last 30 days" || v.Insight.Display != "none" || v.Insight.Headline != "Nothing posted on record yet." || v.Insight.Frac != 0 {
		t.Fatalf("%+v", v.Insight)
	}
}

func TestVolumeCrewColumnsCapAtSix(t *testing.T) {
	in := baseInput()
	in.Crew = nil
	for i := 0; i < 9; i++ {
		in.Crew = append(in.Crew, Agent{ID: string(rune('a' + i)), Name: "Agent " + string(rune('0'+i)), Status: "active"})
	}
	v := Volume(in)
	if len(v.CrewRuns) != 6 || v.CrewRuns[0].Label != "Agent" || v.CrewRuns[1].Label != "Agent 2" {
		t.Fatalf("%+v", v.CrewRuns)
	}
}

func TestVolumeUnreachableZernioIsUnknown(t *testing.T) {
	in := baseInput()
	in.PostDays, in.PostsKnown = nil, false
	v := Volume(in)
	if v.Caption != "posting history unavailable · Zernio not answering" || v.Insight.Display != "—" || v.Insight.Headline != "Posting history unavailable." {
		t.Fatalf("%q %+v", v.Caption, v.Insight)
	}
}

func TestShortLabels(t *testing.T) {
	got := ShortLabels([]string{"Social Agent", "Social Media", "  ", "Zernio"})
	want := []string{"Social", "Social 2", "Agent", "Zernio"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v", got)
		}
	}
}

func TestContentAgentsLeadFirstThenByName(t *testing.T) {
	social := "social-agent"
	all := []Agent{
		{ID: "postly-publisher", Name: "Zernio Publisher", DepartmentID: DeptID, Tier: "worker", ParentID: &social},
		{ID: "sales-agent", Name: "Sales Agent", DepartmentID: "dept-sales", Tier: "lead"},
		{ID: "adsmith-creative", Name: "Arcads Creative", DepartmentID: DeptID, Tier: "worker", ParentID: &social},
		{ID: "social-agent", Name: "Social Agent", DepartmentID: DeptID, Tier: "lead"},
	}
	crew := ContentAgents(all)
	if len(crew) != 3 || crew[0].ID != "social-agent" || crew[1].ID != "adsmith-creative" || crew[2].ID != "postly-publisher" {
		t.Fatalf("%+v", crew)
	}
}
