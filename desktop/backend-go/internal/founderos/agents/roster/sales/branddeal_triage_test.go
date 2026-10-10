package sales

import (
	"slices"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/brand-deal-agent.test.ts (triageDeals), plus
// the detail strings and triageSummary the TS pins through the agent.
const triageToday = "2026-08-19"

func dayp(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func instant(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}

func money(n float64) *float64 { return &n }

func deal(over func(*BrandDeal)) BrandDeal {
	d := BrandDeal{ID: "d1", Brand: "Acme", Status: "New", NotionURL: "https://notion.so/x", LastEdited: instant("2026-08-19T00:00:00.000Z")}
	if over != nil {
		over(&d)
	}
	return d
}

func kinds(a []DealAction) []DealActionKind {
	out := []DealActionKind{}
	for _, x := range a {
		out = append(out, x.Kind)
	}
	return out
}

func TestTriageEmptyPipelineProducesNoBusywork(t *testing.T) {
	if got := TriageDeals(nil, triageToday); got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want an empty list", got)
	}
}

func TestTriageChasesUnpaidDeliveredWork(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.AmountAgreedUSD = "Invoiced", money(4000) })}, triageToday)
	if !slices.Contains(kinds(out), KindChasePayment) {
		t.Fatalf("%v", kinds(out))
	}
	if out[0].Detail != "Invoiced and unpaid ($4,000). Chase the invoice." || out[0].Urgency != 80 {
		t.Fatalf("%+v", out[0])
	}
	// amountAgreed ?? dealValue, and a zero amount is left out as in JS.
	out = TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.DealValueUSD = "Approved", money(1234567.5) })}, triageToday)
	if out[0].Detail != "Approved and unpaid ($1,234,568). Chase the invoice." {
		t.Fatalf("%q", out[0].Detail)
	}
	out = TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.AmountAgreedUSD = "Invoiced", money(0) })}, triageToday)
	if out[0].Detail != "Invoiced and unpaid. Chase the invoice." {
		t.Fatalf("%q", out[0].Detail)
	}
	paid := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.PaidInFull, d.AmountAgreedUSD = "Paid", true, money(4000) })}, triageToday)
	if len(paid) != 0 {
		t.Fatalf("paid in full is left alone: %+v", paid)
	}
}

func TestTriageMissedDeadlineOutranksEverything(t *testing.T) {
	out := TriageDeals([]BrandDeal{
		deal(func(d *BrandDeal) { d.ID, d.Status, d.Deadline = "late", "Filming", dayp("2026-08-10") }),
		deal(func(d *BrandDeal) { d.ID, d.Status, d.AmountAgreedUSD = "money", "Invoiced", money(9000) }),
	}, triageToday)
	if out[0].DealID != "late" || out[0].Kind != KindOverdue || out[0].Urgency != 100 {
		t.Fatalf("%+v", out[0])
	}
	if out[0].Detail != "Deadline was 2026-08-10, 9 days ago, and it is still Filming." {
		t.Fatalf("%q", out[0].Detail)
	}
	one := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.Deadline = "Filming", dayp("2026-08-18") })}, triageToday)
	if one[0].Detail != "Deadline was 2026-08-18, 1 day ago, and it is still Filming." {
		t.Fatalf("%q", one[0].Detail)
	}
}

func TestTriageDeadlineInsideTheWeekIsAWarning(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.Deadline = "Filming", dayp("2026-08-23") })}, triageToday)
	if !slices.Contains(kinds(out), KindDeadlineSoon) || out[0].Urgency >= 100 || out[0].Detail != "Due 2026-08-23, in 4 days, currently Filming." {
		t.Fatalf("%+v", out)
	}
	today := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.Deadline = "Filming", dayp("2026-08-19") })}, triageToday)
	if today[0].Detail != "Due 2026-08-19, in 0 days, currently Filming." {
		t.Fatalf("%q", today[0].Detail)
	}
	far := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.Deadline = "Filming", dayp("2026-08-27") })}, triageToday)
	if len(far) != 0 {
		t.Fatalf("eight days out is not yet a warning: %+v", far)
	}
	// A timed deadline keeps its instant in the detail; days floor as Math.floor does.
	timed := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) {
		t := instant("2026-08-20T15:30:00Z")
		d.Status, d.Deadline = "Filming", &t
	})}, triageToday)
	if timed[0].Detail != "Due 2026-08-20T15:30:00.000Z, in 2 days, currently Filming." {
		t.Fatalf("%q", timed[0].Detail)
	}
}

func TestTriageDeliveredDeadlineIsNotADeadlineProblem(t *testing.T) {
	for _, status := range []string{"Delivered", "Invoiced", "Approved", "Paid"} {
		out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.Deadline = status, dayp("2026-08-10") })}, triageToday)
		if slices.Contains(kinds(out), KindOverdue) {
			t.Fatalf("%s: %v", status, kinds(out))
		}
	}
}

func TestTriageFollowUpDue(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.FollowUpDate = dayp("2026-08-19") })}, triageToday)
	if !slices.Contains(kinds(out), KindFollowUpDue) || out[0].Detail != "Follow-up was set for 2026-08-19." {
		t.Fatalf("%+v", out)
	}
	if out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.FollowUpDate = dayp("2026-08-25") })}, triageToday); slices.Contains(kinds(out), KindFollowUpDue) {
		t.Fatalf("%v", kinds(out))
	}
}

func TestTriageNegotiationWithNoNumberNeedsOne(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.SuggestedRateUSD = "Negotiating", money(3500) })}, triageToday)
	if len(out) != 1 || out[0].Kind != KindNeedsPrice || out[0].Detail != "Negotiating with no agreed number. Open at $3,500." {
		t.Fatalf("%+v", out)
	}
	out = TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.BudgetUSD = "Negotiating", money(800) })}, triageToday)
	if out[0].Detail != "Negotiating with no agreed number. Open at $800." {
		t.Fatalf("budget is the fallback opener: %q", out[0].Detail)
	}
	out = TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status = "Negotiating" })}, triageToday)
	if out[0].Detail != "Negotiating with no agreed number and no suggested rate to open from." {
		t.Fatalf("%q", out[0].Detail)
	}
	if out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Status, d.AmountAgreedUSD = "Negotiating", money(0) })}, triageToday); len(out) != 0 {
		t.Fatalf("an agreed number, even 0, is a number: %+v", out)
	}
}

func TestTriageStaleInbound(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.LastEdited = instant("2026-08-14T00:00:00.000Z") })}, triageToday)
	if !slices.Contains(kinds(out), KindStaleInbound) || out[0].Detail != "Inbound has sat 5 days with no movement." {
		t.Fatalf("%+v", out)
	}
	fresh := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.LastEdited = instant("2026-08-19T09:00:00.000Z") })}, triageToday)
	if slices.Contains(kinds(fresh), KindStaleInbound) {
		t.Fatalf("%v", kinds(fresh))
	}
	// Two days and a bit floors to 2: not stale yet.
	if out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.LastEdited = instant("2026-08-16T12:00:00Z") })}, triageToday); len(out) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestTriageDeadLanesAndSeededRowsAreNeverChased(t *testing.T) {
	for _, status := range []string{"Declined", "Paused"} {
		out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) {
			d.Status, d.Deadline, d.FollowUpDate = status, dayp("2026-08-01"), dayp("2026-08-01")
		})}, triageToday)
		if len(out) != 0 {
			t.Fatalf("%s: %+v", status, out)
		}
	}
	if out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Seeded, d.Status = true, "Invoiced" })}, triageToday); len(out) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestTriageActionsAreActionableAndRankedStably(t *testing.T) {
	out := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.Brand, d.Status, d.AmountAgreedUSD = "Ridge", "Invoiced", money(2000) })}, triageToday)
	if out[0].Brand != "Ridge" || out[0].NotionURL != "https://notion.so/x" {
		t.Fatalf("%+v", out[0])
	}
	out = TriageDeals([]BrandDeal{deal(func(d *BrandDeal) {
		d.Status, d.SuggestedRateUSD, d.FollowUpDate, d.Deadline = "Negotiating", money(1000), dayp("2026-08-01"), dayp("2026-08-01")
	})}, triageToday)
	if !slices.Equal(kinds(out), []DealActionKind{KindOverdue, KindFollowUpDue, KindNeedsPrice}) {
		t.Fatalf("%v", kinds(out))
	}
	// Equal urgency keeps pipeline order.
	out = TriageDeals([]BrandDeal{
		deal(func(d *BrandDeal) { d.ID, d.FollowUpDate = "first", dayp("2026-08-01") }),
		deal(func(d *BrandDeal) { d.ID, d.FollowUpDate = "second", dayp("2026-08-02") }),
	}, triageToday)
	if out[0].DealID != "first" || out[1].DealID != "second" {
		t.Fatalf("%+v", out)
	}
}

func TestTriageSummary(t *testing.T) {
	if got := TriageSummary(nil, 4); got != "4 deals, nothing needs you today." {
		t.Fatalf("%q", got)
	}
	out := TriageDeals([]BrandDeal{
		deal(func(d *BrandDeal) {
			d.ID, d.Brand, d.Status, d.Deadline = "a", "Late Co", "Filming", dayp("2026-08-10")
		}),
		deal(func(d *BrandDeal) {
			d.ID, d.Brand, d.Status, d.AmountAgreedUSD = "b", "Owes Co", "Invoiced", money(9000)
		}),
		deal(func(d *BrandDeal) {
			d.ID, d.Brand, d.Status, d.AmountAgreedUSD = "c", "Owes Too", "Approved", money(100)
		}),
	}, triageToday)
	if got := TriageSummary(out, 3); got != "3 actions across 3 deals · 1 overdue · 2 unpaid · first: Late Co (overdue)" {
		t.Fatalf("%q", got)
	}
	one := TriageDeals([]BrandDeal{deal(func(d *BrandDeal) { d.FollowUpDate = dayp("2026-08-19") })}, triageToday)
	if got := TriageSummary(one, 1); got != "1 action across 1 deals · first: Acme (follow-up-due)" {
		t.Fatalf("%q", got)
	}
}
