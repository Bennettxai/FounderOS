package finances

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

// IncomeSource is the processor rollup (payments.Connector.FinancesIncome),
// injected so tests never reach Stripe, PayKit or Wise.
type IncomeSource func(ctx context.Context, history paykit.HistoryPort) payments.FinancesIncome

// Sources are the page's reads. A nil Store or Income reads as an error on
// the page, never as an empty ledger or zero income.
type Sources struct {
	Store   Store
	Income  IncomeSource
	History paykit.HistoryPort
	Now     func() time.Time
}

type LedgerView struct {
	// Rows is the whole ledger, oldest first, income rows included: the
	// month-to-month panels recompute from it client-side.
	Rows        []SpendRow      `json:"rows"`
	Months      []string        `json:"months"`
	LatestMonth *string         `json:"latestMonth"`
	ByCategory  []CategoryTotal `json:"byCategory"`
	Error       string          `json:"error,omitempty"`
}

type BankView struct {
	Series []BusinessMonths `json:"series"`
	Error  string           `json:"error,omitempty"`
}

type FallbackCategory struct {
	Category   string `json:"category"`
	TotalCents int64  `json:"totalCents"`
}

// Payload is GET /api/founderos/pages/finances: everything app/finances/page.tsx
// gathered server-side.
type Payload struct {
	GeneratedAt string                  `json:"generatedAt"`
	ThisMonth   string                  `json:"thisMonth"`
	Income      payments.FinancesIncome `json:"income"`
	IncomeError string                  `json:"incomeError,omitempty"`
	Ledger      LedgerView              `json:"ledger"`
	Bank        BankView                `json:"bank"`
	CardLanes   []CardLane              `json:"cardLanes"`
	// Fallback is the declared set fees in cents, used only with no statements.
	Fallback             []FallbackCategory `json:"fallback"`
	Expenses             float64            `json:"expenses"`
	ExpensesLive         bool               `json:"expensesLive"`
	MonthLabel           *string            `json:"monthLabel"`
	StatementIsThisMonth bool               `json:"statementIsThisMonth"`
	NetComparable        bool               `json:"netComparable"`
	NetMonthly           float64            `json:"netMonthly"`
	Volume               MoneyVolume        `json:"volume"`
	Spend                []Point            `json:"spend"`
	ChargeSizes          []Point            `json:"chargeSizes"`
	// LargestChargeUSD is nil unless Stripe is live with charges ("—").
	LargestChargeUSD *float64 `json:"largestChargeUsd"`
}

func strp(s string) *string { return &s }

// Build composes the page. Sources are read concurrently.
func Build(ctx context.Context, s Sources) Payload {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	t := now()
	p := Payload{GeneratedAt: t.UTC().Format(time.RFC3339), ThisMonth: t.Format("2006-01"), CardLanes: CardLanes}

	type ledgerRes struct {
		rows []SpendRow
		err  error
	}
	type bankRes struct {
		rows []BankSummary
		err  error
	}
	lc, bc, ic := make(chan ledgerRes, 1), make(chan bankRes, 1), make(chan payments.FinancesIncome, 1)
	noStore := errors.New("finances store: no database")
	go func() {
		if s.Store == nil {
			lc <- ledgerRes{err: noStore}
			return
		}
		r, err := s.Store.LedgerRows(ctx)
		lc <- ledgerRes{r, err}
	}()
	go func() {
		if s.Store == nil {
			bc <- bankRes{err: noStore}
			return
		}
		r, err := s.Store.BankSummaries(ctx)
		bc <- bankRes{r, err}
	}()
	go func() {
		if s.Income == nil {
			ic <- payments.FinancesIncome{Accounts: payments.IncomeAccounts(false, nil, nil, nil), Stripe: payments.StripeOverview{RecentCharges: []stripe.RecentCharge{}}}
			return
		}
		ic <- s.Income(ctx, s.History)
	}()
	led, bank, inc := <-lc, <-bc, <-ic
	if s.Income == nil {
		p.IncomeError = "payment processors are not wired on this backend"
	}
	if inc.Stripe.RecentCharges == nil {
		inc.Stripe.RecentCharges = []stripe.RecentCharge{}
	}
	p.Income = inc

	p.Ledger = LedgerView{Rows: []SpendRow{}, Months: []string{}, ByCategory: []CategoryTotal{}}
	if led.err != nil {
		p.Ledger.Error = led.err.Error()
	} else {
		p.Ledger.Rows = led.rows
		p.Ledger.Months = MonthsAscending(led.rows)
		if m := LatestMonth(led.rows); m != "" {
			p.Ledger.LatestMonth = strp(m)
			p.Ledger.ByCategory = ByCategory(led.rows, m)
		}
	}
	p.Bank = BankView{Series: []BusinessMonths{}}
	if bank.err != nil {
		p.Bank.Error = bank.err.Error()
	} else {
		p.Bank.Series = BusinessSeries(bank.rows)
	}

	for _, c := range ExpensesByCategory(DeclaredExpenses) {
		p.Fallback = append(p.Fallback, FallbackCategory{c.Category, int64(math.Round(c.Total * 100))})
	}
	p.ExpensesLive = len(p.Ledger.ByCategory) > 0
	if p.ExpensesLive {
		for _, c := range p.Ledger.ByCategory {
			p.Expenses += c.Total
		}
		p.MonthLabel = strp(MonthName(*p.Ledger.LatestMonth))
		p.StatementIsThisMonth = *p.Ledger.LatestMonth == p.ThisMonth
	} else {
		p.Expenses = TotalExpenses(DeclaredExpenses)
	}
	income := payments.TotalIncome(inc.Accounts)
	p.NetMonthly = income - p.Expenses
	p.NetComparable = !p.ExpensesLive || p.StatementIsThisMonth
	label := ""
	if p.MonthLabel != nil {
		label = *p.MonthLabel
	}
	p.Volume = MoneyVolumeOf(VolumeInput{Accounts: inc.Accounts, Expenses: p.Expenses, ExpensesLive: p.ExpensesLive, MonthLabel: label, StatementIsThisMonth: p.StatementIsThisMonth})
	p.Spend = SpendSeries(p.Ledger.Rows)
	amounts := make([]int64, 0, len(inc.Stripe.RecentCharges))
	var largest int64
	for _, c := range inc.Stripe.RecentCharges {
		amounts = append(amounts, c.Amount)
		if c.Amount > largest {
			largest = c.Amount
		}
	}
	p.ChargeSizes = ChargeSizes(amounts)
	if inc.Stripe.Live && len(amounts) > 0 {
		p.LargestChargeUSD = ptr(float64(largest) / 100)
	}
	return p
}

// ---- statement ingestion (app/api/finances/*/route.ts) ---------------------

var (
	ErrEmptyStatement    = errors.New("expected a statement (CSV, PDF, or extracted text)")
	ErrNoRows            = errors.New("no parseable rows — need a CSV with Date/Description/Amount columns, or statement text with dated charge lines")
	ErrNotABankStatement = errors.New("not a recognizable bank statement summary")
)

// StatementResult is the card upload's response.
type StatementResult struct {
	Inserted       int             `json:"inserted"`
	Parsed         int             `json:"parsed"`
	Card           string          `json:"card"`
	UploadedMonths []string        `json:"uploadedMonths"`
	Months         []string        `json:"months"`
	ByCategory     []CategoryTotal `json:"byCategory"`
}

// IngestCardStatement parses a card statement (CSV first, then statement
// text), categorizes it, files it under the card lane and reports the months
// this upload covered. Re-uploading inserts nothing new.
func IngestCardStatement(ctx context.Context, st Store, text, card string) (*StatementResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyStatement
	}
	parsed := ParseAnyStatement(text)
	if len(parsed) == 0 {
		return nil, ErrNoRows
	}
	card = NormalizeCardID(card)
	rows := make([]LedgerRow, len(parsed))
	seen := map[string]bool{}
	var uploaded []string
	for i, r := range parsed {
		rows[i] = LedgerRow{ParsedRow: r, Category: Categorize(r), Card: card}
		if m := monthOf(r.Date); !seen[m] {
			seen[m] = true
			uploaded = append(uploaded, m)
		}
	}
	sort.Strings(uploaded)
	inserted, err := st.InsertLedgerRows(ctx, rows)
	if err != nil {
		return nil, err
	}
	all, err := st.LedgerRows(ctx)
	if err != nil {
		return nil, err
	}
	res := &StatementResult{Inserted: inserted, Parsed: len(parsed), Card: card, UploadedMonths: uploaded, Months: MonthsAscending(all), ByCategory: []CategoryTotal{}}
	if m := LatestMonth(all); m != "" {
		res.ByCategory = ByCategory(all, m)
	}
	return res, nil
}

// IngestBankStatement parses a bank statement's summary and upserts it by
// account + month.
func IngestBankStatement(ctx context.Context, st Store, text string) (*BankSummary, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyStatement
	}
	s := ParseBankStatementSummary(text)
	if s == nil {
		return nil, ErrNotABankStatement
	}
	if err := st.UpsertBankSummary(ctx, *s); err != nil {
		return nil, err
	}
	return s, nil
}
