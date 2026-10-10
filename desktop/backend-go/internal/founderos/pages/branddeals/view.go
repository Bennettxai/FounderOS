// Package branddeals is the view model for the /brand-deals slab, ported
// from FounderOS v1 lib/brand-deals-view.ts (the GladOS "Deal Journeys" layout
// fed by the Notion Brand Deals Hub). Pure: every figure the slab shows is
// derived here from the deals the Notion connector already parsed, so the
// Svelte page stays a renderer and the numbers stay testable.
//
// Money rule: a deal is worth what was agreed, else its stated value, else
// the brand's budget, else nothing. The suggested rate is our wish, not a
// number the brand said, so it never counts.
//
// The interactive pieces (the prompt bar's parseQuery/filterDeals, timeAgo on
// the viewer's clock) live in the page's TS module, which carries their
// ported tests.
package branddeals

import (
	"fmt"
	"sort"
	"strings"
	"time"

	bd "github.com/rhl/businessos-backend/internal/founderos/connectors/branddeals"
)

type Bucket string

const (
	Inbound   Bucket = "inbound"
	Talking   Bucket = "talking"
	Producing Bucket = "producing"
	Billing   Bucket = "billing"
	Paid      Bucket = "paid"
	Declined  Bucket = "declined"
	Paused    Bucket = "paused"
	Other     Bucket = "other"
)

var buckets = map[string]Bucket{
	"New":         Inbound,
	"Negotiating": Talking,
	"Aligned":     Talking,
	"Researching": Producing,
	"Filming":     Producing,
	"Editing":     Producing,
	"Delivered":   Producing,
	"Invoiced":    Billing,
	"Approved":    Billing,
	"Paid":        Paid,
	"Declined":    Declined,
	"Paused":      Paused,
}

// BucketOf maps a Notion lane to its journey bucket. Unknown lanes are kept
// as other rather than dropped: hiding a deal is worse than an odd label.
func BucketOf(status string) Bucket {
	if b, ok := buckets[status]; ok {
		return b
	}
	return Other
}

// DealUSD is agreed, else value, else budget, else 0 (never the suggested rate).
func DealUSD(d bd.Deal) float64 {
	switch {
	case d.AmountAgreedUSD != nil:
		return *d.AmountAgreedUSD
	case d.DealValueUSD != nil:
		return *d.DealValueUSD
	case d.BudgetUSD != nil:
		return *d.BudgetUSD
	}
	return 0
}

func IsOpen(d bd.Deal) bool {
	b := BucketOf(d.Status)
	return b != Paid && b != Declined && b != Paused
}

type Filter string

const (
	All        Filter = "all"
	Talks      Filter = "talks"
	Production Filter = "production"
	PaidF      Filter = "paid"
	DeclinedF  Filter = "declined"
	TierS      Filter = "tier-s"
)

func MatchesFilter(d bd.Deal, f Filter) bool {
	b := BucketOf(d.Status)
	switch f {
	case All:
		return true
	case Talks:
		return b == Inbound || b == Talking
	case Production:
		return b == Producing || b == Billing
	case PaidF:
		return b == Paid
	case DeclinedF:
		return b == Declined || b == Paused
	case TierS:
		return d.Tier != nil && strings.ToUpper(strings.TrimSpace(*d.Tier)) == "S"
	}
	return false
}

type Counts struct {
	All        int `json:"all"`
	Talks      int `json:"talks"`
	Production int `json:"production"`
	Paid       int `json:"paid"`
	Declined   int `json:"declined"`
	TierS      int `json:"tierS"`
}

type Volume struct {
	OpenUSD       float64 `json:"openUsd"`
	TalksUSD      float64 `json:"talksUsd"`
	ProductionUSD float64 `json:"productionUsd"`
	PaidUSD       float64 `json:"paidUsd"`
	DeclinedUSD   float64 `json:"declinedUsd"`
	QuotedDeals   int     `json:"quotedDeals"`
	Counts        Counts  `json:"counts"`
}

func sumUSD(deals []bd.Deal, f Filter) float64 {
	var n float64
	for _, d := range deals {
		if MatchesFilter(d, f) {
			n += DealUSD(d)
		}
	}
	return n
}

func count(deals []bd.Deal, f Filter) int {
	n := 0
	for _, d := range deals {
		if MatchesFilter(d, f) {
			n++
		}
	}
	return n
}

func ComputeVolume(deals []bd.Deal) Volume {
	talks, prod := sumUSD(deals, Talks), sumUSD(deals, Production)
	quoted := 0
	for _, d := range deals {
		if DealUSD(d) > 0 {
			quoted++
		}
	}
	return Volume{
		OpenUSD:       talks + prod,
		TalksUSD:      talks,
		ProductionUSD: prod,
		PaidUSD:       sumUSD(deals, PaidF),
		DeclinedUSD:   sumUSD(deals, DeclinedF),
		QuotedDeals:   quoted,
		Counts: Counts{
			All:        len(deals),
			Talks:      count(deals, Talks),
			Production: count(deals, Production),
			Paid:       count(deals, PaidF),
			Declined:   count(deals, DeclinedF),
			TierS:      count(deals, TierS),
		},
	}
}

type FunnelStage struct {
	Label  string  `json:"label"`
	Value  int     `json:"value"`
	USD    float64 `json:"usd"`
	Note   string  `json:"note"`
	Filter Filter  `json:"filter"`
}

// FunnelStages is the four hero columns, each a working filter carrying its dollars.
func FunnelStages(deals []bd.Deal) []FunnelStage {
	v := ComputeVolume(deals)
	declined, paused := 0, 0
	var total float64
	for _, d := range deals {
		switch BucketOf(d.Status) {
		case Declined:
			declined++
		case Paused:
			paused++
		}
		total += DealUSD(d)
	}
	return []FunnelStage{
		{Label: "All deals", Value: v.Counts.All, USD: total, Note: "every deal in the hub", Filter: All},
		{Label: "In talks", Value: v.Counts.Talks, USD: v.TalksUSD, Note: "inbound and negotiating", Filter: Talks},
		{Label: "In production", Value: v.Counts.Production, USD: v.ProductionUSD, Note: "researching through invoiced", Filter: Production},
		{Label: "Paid", Value: v.Counts.Paid, USD: v.PaidUSD, Note: fmt.Sprintf("%d declined · %d paused", declined, paused), Filter: PaidF},
	}
}

type Point struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// parseInstant is JS Date.parse for the shapes Notion sends: a full ISO
// instant or a bare date (UTC midnight).
func parseInstant(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func utcDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// ActivitySeries is Notion edits per UTC day over the window, quiet days
// kept so the step line has real steps. Newest day last.
func ActivitySeries(deals []bd.Deal, now time.Time, days int) []Point {
	end := utcDay(now)
	per := map[time.Time]int{}
	for _, d := range deals {
		if t, ok := parseInstant(d.LastEdited); ok {
			per[utcDay(t)]++
		}
	}
	out := make([]Point, 0, days)
	for i := days - 1; i >= 0; i-- {
		t := end.AddDate(0, 0, -i)
		out = append(out, Point{Label: t.Format("01/02"), Count: per[t]})
	}
	return out
}

var sizeBuckets = []struct {
	label  string
	lo, hi float64 // [lo, hi); hi 0 means open
}{
	{"<1k", 0, 1000}, {"1-3k", 1000, 3000}, {"3-5k", 3000, 5000}, {"5-10k", 5000, 10_000}, {"10k+", 10_000, 0},
}

// SizeMatrix is the deal-size distribution for the dot matrix; unpriced
// deals are not sized.
func SizeMatrix(deals []bd.Deal) []Point {
	out := make([]Point, len(sizeBuckets))
	for i, b := range sizeBuckets {
		out[i].Label = b.label
	}
	for _, d := range deals {
		n := DealUSD(d)
		if n <= 0 {
			continue
		}
		for i, b := range sizeBuckets {
			if n >= b.lo && (b.hi == 0 || n < b.hi) {
				out[i].Count++
				break
			}
		}
	}
	return out
}

type Due struct {
	FollowUps []bd.Deal
	Deadlines []bd.Deal
}

// DueSoon is what needs the operator this week: follow-ups due (overdue included)
// and deadlines inside the window, on open deals only, soonest first.
func DueSoon(deals []bd.Deal, now time.Time, days int) Due {
	limit := now.Add(time.Duration(days) * 24 * time.Hour)
	pick := func(get func(bd.Deal) *string) []bd.Deal {
		type hit struct {
			d  bd.Deal
			at time.Time
		}
		var hits []hit
		for _, d := range deals {
			if !IsOpen(d) || get(d) == nil {
				continue
			}
			if t, ok := parseInstant(*get(d)); ok && !t.After(limit) {
				hits = append(hits, hit{d, t})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].at.Before(hits[j].at) })
		out := make([]bd.Deal, len(hits))
		for i, h := range hits {
			out[i] = h.d
		}
		return out
	}
	return Due{
		FollowUps: pick(func(d bd.Deal) *string { return d.FollowUpDate }),
		Deadlines: pick(func(d bd.Deal) *string { return d.Deadline }),
	}
}

// DueIDs is DueSoon as deal ids, the shape the page receives.
type DueIDs struct {
	FollowUps []string `json:"followUps"`
	Deadlines []string `json:"deadlines"`
}

// View is everything the slab renders beyond the deals themselves.
type View struct {
	Volume        Volume        `json:"volume"`
	Stages        []FunnelStage `json:"stages"`
	Activity      []Point       `json:"activity"`
	EditsInWindow int           `json:"editsInWindow"`
	Sizes         []Point       `json:"sizes"`
	LargestUSD    float64       `json:"largestUsd"`
	Due           DueIDs        `json:"due"`
	NeedsYou      int           `json:"needsYou"`
	// NeedsYouBrands names the first three deals that need the operator.
	NeedsYouBrands []string `json:"needsYouBrands"`
	OpenCount      int      `json:"openCount"`
	// NeedsYouFrac lights the insight card's ticks: needsYou over open deals.
	NeedsYouFrac float64 `json:"needsYouFrac"`
	// HubURL is the first live (non-seeded) deal's Notion link, nil when
	// only examples are showing.
	HubURL *string `json:"hubUrl"`
}

const (
	ActivityDays = 30
	DueDays      = 7
)

func ids(ds []bd.Deal) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.ID
	}
	return out
}

// BuildView derives the slab's figures at now.
func BuildView(deals []bd.Deal, now time.Time) View {
	vol := ComputeVolume(deals)
	series := ActivitySeries(deals, now, ActivityDays)
	edits := 0
	for _, p := range series {
		edits += p.Count
	}
	var largest float64
	var hub *string
	for _, d := range deals {
		if n := DealUSD(d); n > largest {
			largest = n
		}
		if hub == nil && !d.Seeded && d.NotionURL != "" {
			u := d.NotionURL
			hub = &u
		}
	}
	due := DueSoon(deals, now, DueDays)
	needs := len(due.FollowUps) + len(due.Deadlines)
	brands := []string{}
	for _, d := range append(append([]bd.Deal{}, due.FollowUps...), due.Deadlines...) {
		if len(brands) == 3 {
			break
		}
		brands = append(brands, d.Brand)
	}
	open := vol.Counts.Talks + vol.Counts.Production
	denom := open
	if denom < 1 {
		denom = 1
	}
	return View{
		Volume:         vol,
		Stages:         FunnelStages(deals),
		Activity:       series,
		EditsInWindow:  edits,
		Sizes:          SizeMatrix(deals),
		LargestUSD:     largest,
		Due:            DueIDs{FollowUps: ids(due.FollowUps), Deadlines: ids(due.Deadlines)},
		NeedsYou:       needs,
		NeedsYouBrands: brands,
		OpenCount:      open,
		NeedsYouFrac:   float64(needs) / float64(denom),
		HubURL:         hub,
	}
}
