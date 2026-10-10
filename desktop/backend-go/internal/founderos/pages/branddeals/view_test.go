package branddeals

import (
	"reflect"
	"testing"
	"time"

	bd "github.com/rhl/businessos-backend/internal/founderos/connectors/branddeals"
)

// Ported from FounderOS v1 tests/brand-deals-view.test.ts (the Go half: the
// figures the endpoint precomputes). The prompt-bar and timeAgo cases are
// ported next to the page's TS module.

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func sp(s string) *string   { return &s }
func fp(f float64) *float64 { return &f }

func deal(id, brand, status string, mut func(*bd.Deal)) bd.Deal {
	d := bd.Deal{ID: id, Brand: brand, Status: status, NotionURL: "https://www.notion.so/x", LastEdited: "2026-09-16T09:00:00.000Z"}
	if mut != nil {
		mut(&d)
	}
	return d
}

var deals = []bd.Deal{
	deal("a", "Notion", "Negotiating", func(d *bd.Deal) {
		d.Tier, d.DealValueUSD, d.FollowUpDate, d.ContactName, d.MainChannel = sp("S"), fp(6000), sp("2026-09-18"), sp("Sam Rivera"), sp("Instagram")
		d.LastEdited = "2026-09-17T08:00:00.000Z"
	}),
	deal("b", "Framer", "New", func(d *bd.Deal) {
		d.Tier, d.SuggestedRateUSD, d.FollowUpDate, d.LastEdited = sp("A"), fp(4000), sp("2026-09-10"), "2026-09-15T08:00:00.000Z"
	}),
	deal("c", "Shopify", "Filming", func(d *bd.Deal) {
		d.Tier, d.DealValueUSD, d.AmountAgreedUSD, d.Deadline, d.LastEdited = sp("S"), fp(9000), fp(8500), sp("2026-09-20"), "2026-09-15T20:00:00.000Z"
	}),
	deal("d", "Riverside", "Paid", func(d *bd.Deal) {
		d.Tier, d.AmountAgreedUSD, d.PaidInFull, d.LastEdited = sp("B"), fp(2500), true, "2026-08-01T08:00:00.000Z"
	}),
	deal("e", "Loom", "Declined", func(d *bd.Deal) { d.BudgetUSD, d.LastEdited = fp(1200), "2026-09-01T08:00:00.000Z" }),
	deal("f", "Descript", "Invoiced", func(d *bd.Deal) {
		d.AmountAgreedUSD, d.Deadline, d.LastEdited = fp(12000), sp("2026-10-30"), "2026-09-16T08:00:00.000Z"
	}),
	deal("g", "Mystery", "Vibing", func(d *bd.Deal) { d.LastEdited = "2026-09-16T10:00:00.000Z" }),
}

func TestEveryNotionLaneLandsInABucketAndUnknownLanesAreKept(t *testing.T) {
	want := map[string]Bucket{
		"New": Inbound, "Negotiating": Talking, "Aligned": Talking, "Researching": Producing, "Filming": Producing,
		"Editing": Producing, "Delivered": Producing, "Invoiced": Billing, "Approved": Billing, "Paid": Paid,
		"Declined": Declined, "Paused": Paused, "Vibing": Other,
	}
	for status, b := range want {
		if got := BucketOf(status); got != b {
			t.Errorf("BucketOf(%q) = %q, want %q", status, got, b)
		}
	}
	// every lane the connector knows has a bucket
	for _, s := range bd.PipelineOrder {
		if BucketOf(s) == Other {
			t.Errorf("pipeline lane %q fell into other", s)
		}
	}
}

func TestDealIsWorthAgreedElseValueElseBudgetNeverSuggested(t *testing.T) {
	for i, want := range map[int]float64{2: 8500, 0: 6000, 4: 1200, 1: 0} {
		if got := DealUSD(deals[i]); got != want {
			t.Errorf("DealUSD(%s) = %v, want %v", deals[i].ID, got, want)
		}
	}
}

func TestFiltersGroupBucketsTheWayTheChipsRead(t *testing.T) {
	cases := []struct {
		i    int
		f    Filter
		want bool
	}{{1, Talks, true}, {0, Talks, true}, {2, Production, true}, {5, Production, true}, {3, PaidF, true}, {4, DeclinedF, true}, {0, TierS, true}, {1, TierS, false}, {6, All, true}}
	for _, c := range cases {
		if got := MatchesFilter(deals[c.i], c.f); got != c.want {
			t.Errorf("MatchesFilter(%s, %s) = %v", deals[c.i].ID, c.f, got)
		}
	}
}

func TestOpenDollarsExcludeClosedDealsAndTheSplitReconciles(t *testing.T) {
	v := ComputeVolume(deals)
	if v.TalksUSD != 6000 || v.ProductionUSD != 8500+12000 || v.OpenUSD != v.TalksUSD+v.ProductionUSD {
		t.Fatalf("split: %+v", v)
	}
	if v.PaidUSD != 2500 || v.DeclinedUSD != 1200 || v.QuotedDeals != 5 {
		t.Fatalf("closed/quoted: %+v", v)
	}
	if want := (Counts{All: 7, Talks: 2, Production: 2, Paid: 1, Declined: 1, TierS: 2}); v.Counts != want {
		t.Fatalf("counts = %+v, want %+v", v.Counts, want)
	}
}

func TestFourHeroStagesEachAFilterCarryingItsDollars(t *testing.T) {
	s := FunnelStages(deals)
	var filters []Filter
	var values []int
	for _, x := range s {
		filters = append(filters, x.Filter)
		values = append(values, x.Value)
	}
	if !reflect.DeepEqual(filters, []Filter{All, Talks, Production, PaidF}) || !reflect.DeepEqual(values, []int{7, 2, 2, 1}) {
		t.Fatalf("stages %v %v", filters, values)
	}
	if s[0].USD != 6000+8500+2500+1200+12000 || s[3].USD != 2500 || s[3].Note != "1 declined · 0 paused" {
		t.Fatalf("stage dollars/notes: %+v", s)
	}
}

func TestActivityBinsEditsPerDayKeepingQuietDays(t *testing.T) {
	series := ActivitySeries(deals, now, 7)
	if len(series) != 7 || series[6] != (Point{"09/17", 1}) {
		t.Fatalf("series = %v", series)
	}
	by := map[string]int{}
	total := 0
	for _, p := range series {
		by[p.Label] = p.Count
		total += p.Count
	}
	if by["09/15"] != 2 || by["09/14"] != 0 || total != 5 {
		t.Fatalf("series = %v", series)
	}
}

func TestDealSizesBucketByDollarsAndSkipUnquoted(t *testing.T) {
	want := []Point{{"<1k", 0}, {"1-3k", 2}, {"3-5k", 0}, {"5-10k", 2}, {"10k+", 1}}
	if got := SizeMatrix(deals); !reflect.DeepEqual(got, want) {
		t.Fatalf("sizes = %v", got)
	}
}

func TestDueSoonOverdueFollowUpsAndDeadlinesInsideTheWeekNeverClosed(t *testing.T) {
	d := DueSoon(deals, now, 7)
	if got := ids(d.FollowUps); !reflect.DeepEqual(got, []string{"b", "a"}) {
		t.Fatalf("follow-ups = %v", got)
	}
	if got := ids(d.Deadlines); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("deadlines = %v", got)
	}
	closed := deal("z", "Closed", "Paid", func(x *bd.Deal) { x.FollowUpDate = sp("2026-09-01") })
	paused := deal("y", "Paused", "Paused", func(x *bd.Deal) { x.Deadline = sp("2026-09-18") })
	if got := DueSoon([]bd.Deal{closed, paused}, now, 7); len(got.FollowUps)+len(got.Deadlines) != 0 {
		t.Fatalf("paid and paused deals are never due: %v", got)
	}
}

func TestBuildViewBundlesTheSlab(t *testing.T) {
	v := BuildView(deals, now)
	if v.EditsInWindow != 6 { // 30-day window: every edit but Riverside's (Aug 1)
		t.Fatalf("edits = %d", v.EditsInWindow)
	}
	if v.LargestUSD != 12000 || v.NeedsYou != 3 || v.OpenCount != 4 {
		t.Fatalf("view = %+v", v)
	}
	if !reflect.DeepEqual(v.NeedsYouBrands, []string{"Framer", "Notion", "Shopify"}) {
		t.Fatalf("brands = %v", v.NeedsYouBrands)
	}
	if v.NeedsYouFrac != 0.75 || v.HubURL == nil || *v.HubURL != "https://www.notion.so/x" {
		t.Fatalf("frac/hub = %v %v", v.NeedsYouFrac, v.HubURL)
	}
	if len(v.Activity) != ActivityDays || len(v.Stages) != 4 || len(v.Sizes) != 5 {
		t.Fatalf("shapes: %d %d %d", len(v.Activity), len(v.Stages), len(v.Sizes))
	}
}

func TestBuildViewOnExamplesOnlyHasNoHubAndEmptyIsHonestZeroes(t *testing.T) {
	v := BuildView(bd.SeededDeals(), now)
	if v.HubURL != nil {
		t.Fatalf("seeded rows must not name a hub: %v", *v.HubURL)
	}
	e := BuildView(nil, now)
	if e.NeedsYouFrac != 0 || e.NeedsYouBrands == nil || e.Due.FollowUps == nil || e.Volume.Counts.All != 0 {
		t.Fatalf("empty view = %+v", e)
	}
}
