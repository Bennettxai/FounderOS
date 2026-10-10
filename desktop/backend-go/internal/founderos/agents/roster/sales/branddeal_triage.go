package sales

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The port of FounderOS v1 lib/agents/brand-deal-triage.ts: what the Brand Deal
// Agent is allowed to raise, and how loudly. The judgment is a tested rule,
// not a prompt, so the agent can never invent an action for a deal that does
// not warrant one or drop the unpaid invoice.

// BrandDeal is the slice of a founderos_brand_deals row the triage reads.
// Deadline and FollowUpDate are nil when unset.
type BrandDeal struct {
	ID               string
	Brand            string
	Status           string
	DealValueUSD     *float64
	BudgetUSD        *float64
	AmountAgreedUSD  *float64
	SuggestedRateUSD *float64
	PaidInFull       bool
	Deadline         *time.Time
	FollowUpDate     *time.Time
	NotionURL        string
	LastEdited       time.Time
	Seeded           bool
}

type DealActionKind string

const (
	KindOverdue      DealActionKind = "overdue"
	KindChasePayment DealActionKind = "chase-payment"
	KindDeadlineSoon DealActionKind = "deadline-soon"
	KindFollowUpDue  DealActionKind = "follow-up-due"
	KindNeedsPrice   DealActionKind = "needs-price"
	KindStaleInbound DealActionKind = "stale-inbound"
)

// Urgency: higher wins. Money already earned outranks money not yet agreed.
var Urgency = map[DealActionKind]int{
	KindOverdue:      100,
	KindChasePayment: 80,
	KindDeadlineSoon: 60,
	KindFollowUpDue:  40,
	KindNeedsPrice:   30,
	KindStaleInbound: 20,
}

// DealAction is one thing to do today (the TS DealAction, same JSON).
type DealAction struct {
	DealID    string         `json:"dealId"`
	Brand     string         `json:"brand"`
	Kind      DealActionKind `json:"kind"`
	Urgency   int            `json:"urgency"`
	Detail    string         `json:"detail"`
	NotionURL string         `json:"notionUrl"`
}

var (
	// delivered: the deliverable is already out of the operator's hands.
	delivered = map[string]bool{"Delivered": true, "Invoiced": true, "Approved": true, "Paid": true}
	// deadLane: nobody should be chased about these.
	deadLane = map[string]bool{"Declined": true, "Paused": true}
)

const (
	deadlineWarningDays = 7
	staleInboundDays    = 3 // inbound older than this without movement has gone cold
)

// usd is `$${Math.round(n).toLocaleString('en-US')}`.
func usd(n float64) string {
	r := int64(math.Floor(n + 0.5))
	neg := r < 0
	if neg {
		r = -r
	}
	s := strconv.FormatInt(r, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "$-" + b.String()
	}
	return "$" + b.String()
}

// daysAgo is whole days from t to today's UTC midnight; positive means t is
// in the past. ok is false when today does not parse.
func daysAgo(t time.Time, today string) (int, bool) {
	now, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 0, false
	}
	return int(math.Floor(now.Sub(t).Hours() / 24)), true
}

// dealDate renders a stored date the way FounderOS v1 stored it: a date-only
// Notion value (UTC midnight after the ETL) prints as YYYY-MM-DD, a timed one
// as its UTC ISO instant.
func dealDate(t time.Time) string {
	t = t.UTC()
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02T15:04:05.000Z")
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// TriageDeals is triageDeals: the pipeline as a ranked list of things to do
// today. today (YYYY-MM-DD, UTC) is passed in so a run can be replayed.
func TriageDeals(deals []BrandDeal, today string) []DealAction {
	actions := []DealAction{}
	for _, d := range deals {
		// Placeholder rows: chasing a fake brand is worse than showing nothing.
		if d.Seeded || deadLane[d.Status] {
			continue
		}
		raise := func(kind DealActionKind, detail string) {
			actions = append(actions, DealAction{DealID: d.ID, Brand: d.Brand, Kind: kind, Urgency: Urgency[kind], Detail: detail, NotionURL: d.NotionURL})
		}
		done := delivered[d.Status]

		if d.Deadline != nil && !done {
			if late, ok := daysAgo(*d.Deadline, today); ok {
				if late > 0 {
					raise(KindOverdue, fmt.Sprintf("Deadline was %s, %s ago, and it is still %s.", dealDate(*d.Deadline), plural(late, "day"), d.Status))
				} else if -late <= deadlineWarningDays {
					raise(KindDeadlineSoon, fmt.Sprintf("Due %s, in %s, currently %s.", dealDate(*d.Deadline), plural(-late, "day"), d.Status))
				}
			}
		}

		// Shipped and billed but never landed: the operator's money in someone
		// else's account.
		if !d.PaidInFull && (d.Status == "Invoiced" || d.Status == "Approved") {
			amount := d.AmountAgreedUSD
			if amount == nil {
				amount = d.DealValueUSD
			}
			note := ""
			if amount != nil && *amount != 0 && !math.IsNaN(*amount) {
				note = " (" + usd(*amount) + ")"
			}
			raise(KindChasePayment, fmt.Sprintf("%s and unpaid%s. Chase the invoice.", d.Status, note))
		}

		if d.FollowUpDate != nil {
			if due, ok := daysAgo(*d.FollowUpDate, today); ok && due >= 0 {
				raise(KindFollowUpDue, fmt.Sprintf("Follow-up was set for %s.", dealDate(*d.FollowUpDate)))
			}
		}

		if d.Status == "Negotiating" && d.AmountAgreedUSD == nil {
			opener := d.SuggestedRateUSD
			if opener == nil {
				opener = d.BudgetUSD
			}
			if opener != nil && *opener != 0 && !math.IsNaN(*opener) {
				raise(KindNeedsPrice, fmt.Sprintf("Negotiating with no agreed number. Open at %s.", usd(*opener)))
			} else {
				raise(KindNeedsPrice, "Negotiating with no agreed number and no suggested rate to open from.")
			}
		}

		if d.Status == "New" {
			if age, ok := daysAgo(d.LastEdited, today); ok && age >= staleInboundDays {
				raise(KindStaleInbound, fmt.Sprintf("Inbound has sat %d days with no movement.", age))
			}
		}
	}
	// Stable within a tie: pipeline order, so the same run reads the same way.
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].Urgency > actions[j].Urgency })
	return actions
}

// TriageSummary is triageSummary: the one-line run summary.
func TriageSummary(actions []DealAction, dealCount int) string {
	if len(actions) == 0 {
		return fmt.Sprintf("%d deals, nothing needs you today.", dealCount)
	}
	money, late := 0, 0
	for _, a := range actions {
		switch a.Kind {
		case KindChasePayment:
			money++
		case KindOverdue:
			late++
		}
	}
	parts := []string{fmt.Sprintf("%s across %d deals", plural(len(actions), "action"), dealCount)}
	if late > 0 {
		parts = append(parts, fmt.Sprintf("%d overdue", late))
	}
	if money > 0 {
		parts = append(parts, fmt.Sprintf("%d unpaid", money))
	}
	top := actions[0]
	parts = append(parts, fmt.Sprintf("first: %s (%s)", top.Brand, top.Kind))
	return strings.Join(parts, " · ")
}
