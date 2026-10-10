package finances

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
)

// Ports of tests/finances-volume.test.ts, the expense half of
// tests/finances.test.ts, and tests/ledger.test.ts's derived reads.
//
// One deliberate change from FounderOS v1: a processor with no pull, and a
// ratio that cannot be drawn, has frac null (the kit meter reads "unknown"
// over an empty track) instead of 0 (a sliver that reads as a real zero).

func f(v float64) *float64 { return &v }

func acct(id, label string, income *float64, live, configured bool) payments.IncomeAccount {
	return payments.IncomeAccount{ID: id, Processor: "Stripe", Label: label, Configured: configured, Live: live, Income: income}
}

func fracOf(m Meter) float64 {
	if m.Frac == nil {
		return -1
	}
	return *m.Frac
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func TestMoneyVolumeHeadlineChipsCaption(t *testing.T) {
	accounts := []payments.IncomeAccount{
		acct("stripe", "Stripe · Launchpad Cohort", f(6000), true, true),
		acct("stripe-vantage", "Stripe · Vantage", f(2000), true, true),
		acct("paykit-lc", "PayKit · Launchpad Cohort", nil, false, true),
	}
	v := MoneyVolumeOf(VolumeInput{Accounts: accounts, Expenses: 2000, ExpensesLive: true, MonthLabel: "Aug 2026", StatementIsThisMonth: true})
	if v.Headline != 8000 || v.Upper != nil {
		t.Fatalf("headline %v upper %v", v.Headline, v.Upper)
	}
	if !reflect.DeepEqual(v.Chips, []Chip{{Tone: "err", Text: "$2,000 out"}, {Tone: "ok", Text: "+$6,000 net"}}) {
		t.Fatalf("chips %+v", v.Chips)
	}
	if v.Caption != "income this month · 2/3 processors live" {
		t.Fatalf("caption %q", v.Caption)
	}
	var labels []string
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Stripe · Launchpad Cohort", "Stripe · Vantage", "PayKit · Launchpad Cohort", "Spent of income · Aug 2026"}) {
		t.Fatalf("labels %v", labels)
	}
	if !near(fracOf(v.Meters[0]), 0.75) || !near(fracOf(v.Meters[1]), 0.25) || v.Meters[0].Display != "$6,000" {
		t.Fatalf("source meters %+v", v.Meters[:2])
	}
	// a keyed account that is not pulling reads pending: unknown, never $0
	if v.Meters[2].Frac != nil || v.Meters[2].Display != "pull pending" {
		t.Fatalf("pending meter %+v", v.Meters[2])
	}
	if v.Meters[3].Display != "25%" || v.Meters[3].Hue != HueWarn || !near(fracOf(v.Meters[3]), 0.25) {
		t.Fatalf("spend meter %+v", v.Meters[3])
	}
	if v.Foot != "expenses from the uploaded Aug 2026 statement" {
		t.Fatalf("foot %q", v.Foot)
	}
	if v.Insight.Display != "+$6,000" || !near(fracOf(Meter{Frac: v.Insight.Frac}), 0.75) || v.Insight.Headline != "kept of $8,000 in this month." {
		t.Fatalf("insight %+v", v.Insight)
	}
}

func TestMoneyVolumeBoundedMonth(t *testing.T) {
	a := acct("paykit-lc", "PayKit · Launchpad Cohort", f(1000), true, true)
	a.IncomeUpper, a.UnsplittableCustomers = f(1500), 2
	b := MoneyVolumeOf(VolumeInput{Accounts: []payments.IncomeAccount{a}})
	if b.Headline != 1000 || b.Upper == nil || *b.Upper != 1500 || b.Meters[0].Display != "$1,000 – $1,500" {
		t.Fatalf("bounded %+v", b)
	}
}

func TestMoneyVolumeOverspend(t *testing.T) {
	r := MoneyVolumeOf(VolumeInput{Accounts: []payments.IncomeAccount{acct("stripe", "S", f(1000), true, true)}, Expenses: 1500})
	last := r.Meters[len(r.Meters)-1]
	if fracOf(last) != 1 || last.Display != "150%" || last.Hue != HueErr || last.Label != "Set fees of income" {
		t.Fatalf("spend %+v", last)
	}
	if !reflect.DeepEqual(r.Chips[1], Chip{Tone: "err", Text: "−$500 net"}) {
		t.Fatalf("net chip %+v", r.Chips[1])
	}
	if fracOf(Meter{Frac: r.Insight.Frac}) != 0 || r.Insight.Headline != "more out than in this month." {
		t.Fatalf("insight %+v", r.Insight)
	}
	if r.Foot != "expenses are declared set fees · upload a card statement for real months" {
		t.Fatalf("foot %q", r.Foot)
	}
}

func TestMoneyVolumeNoIncome(t *testing.T) {
	e := MoneyVolumeOf(VolumeInput{Accounts: []payments.IncomeAccount{acct("stripe", "S", nil, false, false)}, Expenses: 1200})
	if e.Headline != 0 {
		t.Fatal("headline")
	}
	for _, m := range e.Meters {
		if m.Frac != nil {
			t.Fatalf("no income draws no bars, got %+v", m)
		}
	}
	if e.Meters[0].Display != "awaiting key" || e.Meters[len(e.Meters)-1].Display != "no income yet" {
		t.Fatalf("displays %+v", e.Meters)
	}
	if e.Insight.Frac == nil || *e.Insight.Frac != 0 {
		t.Fatalf("insight %+v", e.Insight)
	}
}

func TestMoneyVolumeNeverDividesOneMonthByAnother(t *testing.T) {
	stale := MoneyVolumeOf(VolumeInput{Accounts: []payments.IncomeAccount{acct("stripe", "S", f(500), true, true)}, Expenses: 8000, ExpensesLive: true, MonthLabel: "Aug 2026"})
	m := stale.Meters[len(stale.Meters)-1]
	if m.Frac != nil || m.Display != "no statement this month" || m.Label != "Spent of income · Aug 2026" {
		t.Fatalf("spend %+v", m)
	}
	if !reflect.DeepEqual(stale.Chips, []Chip{{Tone: "err", Text: "$8,000 out · Aug 2026"}}) {
		t.Fatalf("chips %+v", stale.Chips)
	}
	if stale.Insight.Display != "—" || !strings.Contains(stale.Insight.Headline, "this month's statement") {
		t.Fatalf("insight %+v", stale.Insight)
	}
	fees := MoneyVolumeOf(VolumeInput{Accounts: []payments.IncomeAccount{acct("stripe", "S", f(500), true, true)}, Expenses: 1200})
	if fees.Chips[0].Text != "$1,200 out" || fees.Chips[1].Text != "−$700 net" {
		t.Fatalf("set fees compare with this month: %+v", fees.Chips)
	}
}

func srow(date string, dollars float64, dir string) SpendRow {
	return SpendRow{Date: date, Description: "x", AmountCents: int64(dollars*100 + 0.5), Direction: dir, Category: "Software", Card: "platinum"}
}

func TestSpendSeries(t *testing.T) {
	s := SpendSeries([]SpendRow{srow("2026-07-03", 100.4, "out"), srow("2026-07-20", 50, "out"), srow("2026-08-02", 900, "out"), srow("2026-08-05", 5000, "in")})
	if !reflect.DeepEqual(s, []Point{{Label: "Jul 2026", Count: 150}, {Label: "Aug 2026", Count: 900}}) {
		t.Fatalf("series %+v", s)
	}
	if s := SpendSeries(nil); s == nil || len(s) != 0 {
		t.Fatal("empty ledger is an empty series")
	}
}

func TestChargeSizes(t *testing.T) {
	cols := ChargeSizes([]int64{5000, 20000, 30000, 150000, 500000})
	if !reflect.DeepEqual(cols, []Point{{"<$100", 1}, {"$100-500", 2}, {"$500-2k", 1}, {"$2k+", 1}}) {
		t.Fatalf("cols %+v", cols)
	}
	if e := ChargeSizes(nil); len(e) != 4 || e[0].Count+e[1].Count+e[2].Count+e[3].Count != 0 {
		t.Fatalf("empty %+v", e)
	}
}

func TestDeclaredExpenses(t *testing.T) {
	// v1 lib/finances.ts DECLARED_EXPENSES: a CSM contractor and the core software stack, no names.
	if !reflect.DeepEqual(DeclaredExpenses, []ExpenseItem{
		{"contractor-csm", "Contractor · CSM", "Contractors", 1000},
		{"software-stack", "Core software stack", "Software", 500},
	}) {
		t.Fatalf("declared %+v", DeclaredExpenses)
	}
	fix := []ExpenseItem{{"a", "A", "Software", 20}, {"b", "B", "Software", 30}, {"c", "C", "Advertising", 100}}
	if TotalExpenses(fix) != 150 {
		t.Fatal("total")
	}
	if !reflect.DeepEqual(ExpensesByCategory(fix), []CategoryTotal{{"Advertising", 100}, {"Software", 50}}) {
		t.Fatalf("by category %+v", ExpensesByCategory(fix))
	}
}

var ledgerRows = []SpendRow{
	{Date: "2026-05-10", Description: "May AWS", AmountCents: 1000, Direction: "out", Category: "Infrastructure", Card: "platinum"},
	{Date: "2026-06-10", Description: "Jun Ads", AmountCents: 5000, Direction: "out", Category: "Advertising", Card: "platinum"},
	{Date: "2026-06-12", Description: "Jun AWS", AmountCents: 2000, Direction: "out", Category: "Infrastructure", Card: "blue"},
	{Date: "2026-06-14", Description: "Client", AmountCents: 900000, Direction: "in", Category: "Income", Card: "blue"},
}

func TestLedgerDerivedReads(t *testing.T) {
	if LatestMonth(ledgerRows) != "2026-06" {
		t.Fatalf("latest %q", LatestMonth(ledgerRows))
	}
	if LatestMonth(nil) != "" {
		t.Fatal("empty ledger has no month")
	}
	if !reflect.DeepEqual(MonthsAscending(ledgerRows), []string{"2026-05", "2026-06"}) {
		t.Fatalf("months %v", MonthsAscending(ledgerRows))
	}
	// monthly() is the latest month only, in USD, largest first, income excluded
	if !reflect.DeepEqual(ByCategory(ledgerRows, "2026-06"), []CategoryTotal{{"Advertising", 50}, {"Infrastructure", 20}}) {
		t.Fatalf("by category %+v", ByCategory(ledgerRows, "2026-06"))
	}
	if !reflect.DeepEqual(ByCategory(ledgerRows, ""), []CategoryTotal{{"Advertising", 50}, {"Infrastructure", 30}}) {
		t.Fatalf("all months %+v", ByCategory(ledgerRows, ""))
	}
}

func TestFormatUSD(t *testing.T) {
	for in, want := range map[float64]string{0: "$0", 6000: "$6,000", 1234567.5: "$1,234,568", 999.49: "$999", -500: "-$500"} {
		if got := USD(in); got != want {
			t.Errorf("USD(%v) = %q, want %q", in, got, want)
		}
	}
	if MonthName("2026-08") != "Aug 2026" {
		t.Fatal(MonthName("2026-08"))
	}
}
