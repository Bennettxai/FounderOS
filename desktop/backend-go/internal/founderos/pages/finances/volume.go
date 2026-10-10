package finances

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
)

// ---- ledger rows and their derived reads (lib/ledger.ts, lib/spend-report.ts)

// SpendRow is one ledger row as the page sees it (FounderOS v1 SpendRow).
type SpendRow struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	AmountCents int64  `json:"amountCents"`
	Direction   string `json:"direction"`
	Category    string `json:"category"`
	Card        string `json:"card"`
}

type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

func monthOf(date string) string {
	if len(date) < 7 {
		return date
	}
	return date[:7]
}

// MonthsAscending lists every month with spend, oldest first (the order the
// /finances month steppers walk).
func MonthsAscending(rows []SpendRow) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rows {
		if r.Direction != "out" || seen[monthOf(r.Date)] {
			continue
		}
		seen[monthOf(r.Date)] = true
		out = append(out, monthOf(r.Date))
	}
	sort.Strings(out)
	return out
}

// LatestMonth is the newest month with spend, "" when the ledger has none.
func LatestMonth(rows []SpendRow) string {
	ms := MonthsAscending(rows)
	if len(ms) == 0 {
		return ""
	}
	return ms[len(ms)-1]
}

// ByCategory is out-row spend per category in USD for one month ("" = all),
// largest first.
func ByCategory(rows []SpendRow, month string) []CategoryTotal {
	totals := map[string]int64{}
	for _, r := range rows {
		if r.Direction != "out" || (month != "" && monthOf(r.Date) != month) {
			continue
		}
		totals[r.Category] += r.AmountCents
	}
	out := []CategoryTotal{}
	for c, t := range totals {
		out = append(out, CategoryTotal{c, float64(t) / 100})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Category < out[j].Category
	})
	return out
}

// ---- declared set fees (lib/finances.ts) -----------------------------------

type ExpenseItem struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Category string  `json:"category"`
	Monthly  float64 `json:"monthly"`
}

// DeclaredExpenses are FounderOS v1's declared set fees: a CSM contractor and
// the core software stack. Real months arrive through the card statement upload.
var DeclaredExpenses = []ExpenseItem{
	{"contractor-csm", "Contractor · CSM", "Contractors", 1000},
	{"software-stack", "Core software stack", "Software", 500},
}

func TotalExpenses(items []ExpenseItem) float64 {
	var s float64
	for _, e := range items {
		s += e.Monthly
	}
	return s
}

func ExpensesByCategory(items []ExpenseItem) []CategoryTotal {
	totals := map[string]float64{}
	var order []string
	for _, e := range items {
		if _, ok := totals[e.Category]; !ok {
			order = append(order, e.Category)
		}
		totals[e.Category] += e.Monthly
	}
	out := make([]CategoryTotal, 0, len(order))
	for _, c := range order {
		out = append(out, CategoryTotal{c, totals[c]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// ---- Money Volume (lib/finances-volume.ts) ---------------------------------

// Meter matches the kit's Meter: Frac nil is unknown (no bar, "unknown").
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

type Chip struct {
	Tone string `json:"tone"`
	Text string `json:"text"`
}

// Point is the kit's SeriesPoint.
type Point struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type Insight struct {
	Display  string   `json:"display"`
	Headline string   `json:"headline"`
	Body     string   `json:"body"`
	Frac     *float64 `json:"frac"`
}

type MoneyVolume struct {
	// Headline is month-to-date income, the PROVEN floor.
	Headline float64 `json:"headline"`
	// Upper is the ceiling when a source could only bound its month.
	Upper   *float64 `json:"upper"`
	Chips   []Chip   `json:"chips"`
	Caption string   `json:"caption"`
	Meters  []Meter  `json:"meters"`
	Foot    string   `json:"foot"`
	Insight Insight  `json:"insight"`
}

type VolumeInput struct {
	Accounts []payments.IncomeAccount
	// Expenses is USD for the month the figure covers.
	Expenses float64
	// ExpensesLive: from an uploaded statement (false: declared set fees).
	ExpensesLive bool
	// MonthLabel is "Aug 2026" for the uploaded month, "" for set fees.
	MonthLabel string
	// StatementIsThisMonth: income is month to date, so only this month's
	// statement may be divided into it.
	StatementIsThisMonth bool
}

// Hues on the Monolith: one per income source, status colors for spend.
const (
	HueWarn = "var(--bn-warn)"
	HueErr  = "var(--bn-err)"
)

var sourceHues = []string{"var(--bn-text)", "var(--bn-text-2)", "var(--bn-text-3)", "var(--bn-accent)"}

// USD matches toLocaleString('en-US', {style:'currency', currency:'USD',
// maximumFractionDigits:0}).
func USD(n float64) string {
	r := math.Round(n)
	neg := r < 0
	s := strconv.FormatInt(int64(math.Abs(r)), 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-$" + b.String()
	}
	return "$" + b.String()
}

func signed(n float64) string {
	if n < 0 {
		return "−" + USD(math.Abs(n))
	}
	return "+" + USD(math.Abs(n))
}

func clamp01(n float64) float64 {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0
	}
	return math.Max(0, math.Min(1, n))
}

func ptr(v float64) *float64 { return &v }

// MonthName is "2026-08" → "Aug 2026".
func MonthName(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("Jan 2006")
}

// MoneyVolumeOf is moneyVolume. Nothing here invents money: a processor with
// no pull, or a ratio across two different months, is unknown (Frac nil).
func MoneyVolumeOf(x VolumeInput) MoneyVolume {
	income := payments.TotalIncome(x.Accounts)
	ceiling := payments.TotalIncomeUpper(x.Accounts)
	net := income - x.Expenses
	comparable := !x.ExpensesLive || x.StatementIsThisMonth
	live := 0
	for _, a := range x.Accounts {
		if a.Live {
			live++
		}
	}

	meters := make([]Meter, 0, len(x.Accounts)+1)
	for i, a := range x.Accounts {
		m := Meter{Label: a.Label, Hue: sourceHues[i%len(sourceHues)]}
		switch {
		case a.Income == nil && a.Configured:
			m.Display = "pull pending"
		case a.Income == nil:
			m.Display = "awaiting key"
		case a.IncomeUpper != nil:
			m.Display = USD(*a.Income) + " – " + USD(*a.IncomeUpper)
		default:
			m.Display = USD(*a.Income)
		}
		if a.Income != nil && income > 0 {
			m.Frac = ptr(clamp01(*a.Income / income))
		}
		meters = append(meters, m)
	}

	spend := Meter{Label: "Set fees of income", Hue: HueWarn}
	if x.ExpensesLive {
		spend.Label = "Spent of income"
		if x.MonthLabel != "" {
			spend.Label += " · " + x.MonthLabel
		}
	}
	switch {
	case !comparable:
		spend.Display = "no statement this month"
	case income > 0:
		ratio := x.Expenses / income
		spend.Frac = ptr(clamp01(ratio))
		spend.Display = fmt.Sprintf("%d%%", int64(math.Round(ratio*100)))
		if x.Expenses > income {
			spend.Hue = HueErr
		}
	default:
		spend.Display = "no income yet"
	}
	meters = append(meters, spend)

	label := x.MonthLabel
	v := MoneyVolume{
		Headline: income,
		Caption:  fmt.Sprintf("income this month · %d/%d processors live", live, len(x.Accounts)),
		Meters:   meters,
	}
	if ceiling > income {
		v.Upper = ptr(ceiling)
	}
	if comparable {
		tone := "ok"
		if net < 0 {
			tone = "err"
		}
		v.Chips = []Chip{{"err", USD(x.Expenses) + " out"}, {tone, signed(net) + " net"}}
	} else {
		l := label
		if l == "" {
			l = "latest statement"
		}
		v.Chips = []Chip{{"err", USD(x.Expenses) + " out · " + l}}
	}
	latest := label
	if latest == "" {
		latest = "latest"
	}
	if x.ExpensesLive {
		v.Foot = "expenses from the uploaded " + latest + " statement"
	} else {
		v.Foot = "expenses are declared set fees · upload a card statement for real months"
	}
	if !comparable {
		other := label
		if other == "" {
			other = "from another month"
		}
		v.Insight = Insight{
			Display:  "—",
			Headline: "Upload this month's statement to see what was kept.",
			Body:     USD(income) + " in so far this month; the latest statement is " + other + ".",
		}
		return v
	}
	headline := "more out than in this month."
	if net >= 0 {
		headline = "kept of " + USD(income) + " in this month."
	}
	body := USD(income) + " in − " + USD(x.Expenses) + " out"
	if x.ExpensesLive {
		body += ", spend from the " + latest + " statement."
	} else {
		body += ", spend is the declared set fees."
	}
	frac := 0.0
	if income > 0 {
		frac = clamp01(net / income)
	}
	v.Insight = Insight{Display: signed(net), Headline: headline, Body: body, Frac: ptr(frac)}
	return v
}

// SpendSeries is spend per ledger month, oldest first, in whole dollars.
func SpendSeries(rows []SpendRow) []Point {
	totals := map[string]int64{}
	for _, r := range rows {
		if r.Direction == "out" {
			totals[monthOf(r.Date)] += r.AmountCents
		}
	}
	months := make([]string, 0, len(totals))
	for m := range totals {
		months = append(months, m)
	}
	sort.Strings(months)
	out := make([]Point, 0, len(months))
	for _, m := range months {
		out = append(out, Point{MonthName(m), int64(math.Floor(float64(totals[m])/100 + 0.5))})
	}
	return out
}

var sizeBuckets = []struct {
	label string
	max   int64
}{{"<$100", 10_000}, {"$100-500", 50_000}, {"$500-2k", 200_000}, {"$2k+", math.MaxInt64}}

// ChargeSizes buckets recent Stripe charges (cents) by size, every bucket
// present so the dot matrix keeps its columns on a quiet month.
func ChargeSizes(amounts []int64) []Point {
	out := make([]Point, len(sizeBuckets))
	for i, b := range sizeBuckets {
		out[i].Label = b.label
		min := int64(math.MinInt64)
		if i > 0 {
			min = sizeBuckets[i-1].max
		}
		for _, a := range amounts {
			if a >= min && a < b.max {
				out[i].Count++
			}
		}
	}
	return out
}
