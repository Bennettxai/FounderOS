package finances

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/wise"
)

// memStore is an in-memory Store with the Postgres store's semantics.
type memStore struct {
	ledger  map[string]LedgerRow
	bank    map[string]BankSummary
	readErr error
}

func newMem() *memStore {
	return &memStore{ledger: map[string]LedgerRow{}, bank: map[string]BankSummary{}}
}

func (m *memStore) InsertLedgerRows(_ context.Context, rows []LedgerRow) (int, error) {
	n := 0
	for _, r := range rows {
		r.Card = NormalizeCardID(r.Card)
		if _, ok := m.ledger[LedgerHash(r)]; !ok {
			m.ledger[LedgerHash(r)] = r
			n++
		}
	}
	return n, nil
}

func (m *memStore) LedgerRows(context.Context) ([]SpendRow, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	out := []SpendRow{}
	for _, r := range m.ledger {
		out = append(out, SpendRow{r.Date, r.Description, r.AmountCents, r.Direction, r.Category, r.Card})
	}
	SortSpendRows(out)
	return out, nil
}

func (m *memStore) UpsertBankSummary(_ context.Context, b BankSummary) error {
	m.bank[b.Account+"|"+b.Month] = b
	return nil
}

func (m *memStore) BankSummaries(context.Context) ([]BankSummary, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	out := []BankSummary{}
	for _, b := range m.bank {
		out = append(out, b)
	}
	return out, nil
}

var sept = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

func incomeOf(fi payments.FinancesIncome) IncomeSource {
	return func(context.Context, paykit.HistoryPort) payments.FinancesIncome { return fi }
}

func liveIncome() payments.FinancesIncome {
	mtd := 6000.0
	accts := payments.IncomeAccounts(true, &mtd, map[string]bool{"stripe": true, "paykit-lc": true}, nil)
	ref := "rent"
	return payments.FinancesIncome{
		Stripe:       payments.StripeOverview{Keyed: true, Live: true, MtdUSD: &mtd, AvailableUSD: 12.5, RecentCharges: []stripe.RecentCharge{{Amount: 250000, Currency: "usd", Description: "Retainer", Created: 1}, {Amount: 5000, Currency: "usd", Description: "Tip", Created: 2}}},
		Accounts:     accts,
		TotalUSD:     payments.TotalIncome(accts),
		LiveCount:    1,
		WiseOutgoing: []wise.Transfer{{AmountCents: 10000, Currency: "USD", Status: "sent", Reference: &ref}},
	}
}

func TestBuildWithNoStatementsFallsBackToTheDeclaredFees(t *testing.T) {
	p := Build(context.Background(), Sources{Store: newMem(), Income: incomeOf(liveIncome()), Now: func() time.Time { return sept }})
	if p.ThisMonth != "2026-09" || p.ExpensesLive || p.Expenses != 1500 || p.MonthLabel != nil || !p.NetComparable || p.NetMonthly != 4500 {
		t.Fatalf("payload %+v", p)
	}
	if !reflect.DeepEqual(p.Fallback, []FallbackCategory{{"Contractors", 100000}, {"Software", 50000}}) {
		t.Fatalf("fallback %+v", p.Fallback)
	}
	if p.Ledger.Error != "" || len(p.Ledger.Rows) != 0 || p.Ledger.Months == nil || p.Bank.Series == nil {
		t.Fatalf("empty ledger / bank are empty lists: %+v %+v", p.Ledger, p.Bank)
	}
	if p.Volume.Headline != 6000 || len(p.Spend) != 0 || len(p.ChargeSizes) != 4 || p.ChargeSizes[3].Count != 1 {
		t.Fatalf("volume %+v spend %+v sizes %+v", p.Volume, p.Spend, p.ChargeSizes)
	}
	if p.LargestChargeUSD == nil || *p.LargestChargeUSD != 2500 {
		t.Fatalf("largest %v", p.LargestChargeUSD)
	}
	if len(p.CardLanes) != 3 || p.Income.WiseOutgoing[0].AmountCents != 10000 {
		t.Fatalf("lanes / wise %+v", p)
	}
}

func TestBuildUsesTheLatestStatementMonth(t *testing.T) {
	st := newMem()
	_, _ = st.InsertLedgerRows(context.Background(), []LedgerRow{
		lrow("2026-08-03", "AWS", 50000, "out", "Infrastructure", "platinum"),
		lrow("2026-09-03", "AWS", 100000, "out", "Infrastructure", "platinum"),
		lrow("2026-09-04", "Notion", 50000, "out", "Software", "blue"),
		lrow("2026-09-05", "Client", 900000, "in", "Income", "blue"),
	})
	p := Build(context.Background(), Sources{Store: st, Income: incomeOf(liveIncome()), Now: func() time.Time { return sept }})
	if !p.ExpensesLive || p.Expenses != 1500 || p.MonthLabel == nil || *p.MonthLabel != "Sep 2026" || !p.StatementIsThisMonth || p.NetMonthly != 4500 {
		t.Fatalf("payload %+v", p)
	}
	if p.Ledger.LatestMonth == nil || *p.Ledger.LatestMonth != "2026-09" || !reflect.DeepEqual(p.Ledger.Months, []string{"2026-08", "2026-09"}) || len(p.Ledger.Rows) != 4 {
		t.Fatalf("ledger %+v", p.Ledger)
	}
	if len(p.Spend) != 2 || p.Spend[1].Count != 1500 {
		t.Fatalf("spend %+v", p.Spend)
	}

	// a statement from an earlier month is never netted against this month
	p = Build(context.Background(), Sources{Store: st, Income: incomeOf(liveIncome()), Now: func() time.Time { return sept.AddDate(0, 1, 0) }})
	if p.StatementIsThisMonth || p.NetComparable || len(p.Volume.Chips) != 1 {
		t.Fatalf("stale statement %+v", p)
	}
}

func TestBuildSurfacesAnUnreachableStoreInsteadOfReadingEmpty(t *testing.T) {
	st := newMem()
	st.readErr = errors.New("connection refused")
	p := Build(context.Background(), Sources{Store: st, Income: incomeOf(liveIncome()), Now: func() time.Time { return sept }})
	if !strings.Contains(p.Ledger.Error, "connection refused") || !strings.Contains(p.Bank.Error, "connection refused") {
		t.Fatalf("errors %q %q", p.Ledger.Error, p.Bank.Error)
	}
	p = Build(context.Background(), Sources{Income: incomeOf(liveIncome()), Now: func() time.Time { return sept }})
	if p.Ledger.Error == "" || p.Bank.Error == "" {
		t.Fatal("no store is an error, not an empty ledger")
	}
}

func TestBuildStripeNotLiveHasNoLargestCharge(t *testing.T) {
	fi := payments.FinancesIncome{Accounts: payments.IncomeAccounts(false, nil, nil, nil)}
	p := Build(context.Background(), Sources{Store: newMem(), Income: incomeOf(fi), Now: func() time.Time { return sept }})
	if p.LargestChargeUSD != nil || p.Volume.Headline != 0 || p.Income.Stripe.RecentCharges == nil {
		t.Fatalf("payload %+v", p)
	}
}

func TestIngestCardStatement(t *testing.T) {
	st := newMem()
	text := "\nThe Platinum Card\nClosing Date 07/26/26\n07/01/26   ANTHROPIC*CLAUDE      SAN FRANCISCO CA     $200.00\n07/04/26   AWS                   SEATTLE WA            $57.00\n"
	res, err := IngestCardStatement(context.Background(), st, text, "gold")
	if err != nil || res.Inserted != 2 || res.Parsed != 2 || res.Card != "gold" || !reflect.DeepEqual(res.UploadedMonths, []string{"2026-07"}) {
		t.Fatalf("res %+v, %v", res, err)
	}
	if !reflect.DeepEqual(res.ByCategory, []CategoryTotal{{"Software", 200}, {"Infrastructure", 57}}) {
		t.Fatalf("by category %+v", res.ByCategory)
	}
	if res, _ = IngestCardStatement(context.Background(), st, text, "gold"); res.Inserted != 0 {
		t.Fatal("re-upload inserts nothing new")
	}
	backdated := "Date,Description,Amount\n03/02/2026,NOTION LABS,-10.00\n04/02/2026,NOTION LABS,-10.00"
	res, _ = IngestCardStatement(context.Background(), st, backdated, "")
	if !reflect.DeepEqual(res.UploadedMonths, []string{"2026-03", "2026-04"}) || !reflect.DeepEqual(res.Months, []string{"2026-03", "2026-04", "2026-07"}) || res.Card != "platinum" {
		t.Fatalf("back-dated %+v", res)
	}
	if _, err := IngestCardStatement(context.Background(), st, "just some notes", ""); !errors.Is(err, ErrNoRows) {
		t.Fatalf("err = %v", err)
	}
	if _, err := IngestCardStatement(context.Background(), st, "  ", ""); !errors.Is(err, ErrEmptyStatement) {
		t.Fatalf("err = %v", err)
	}
}

func TestIngestBankStatement(t *testing.T) {
	st := newMem()
	s, err := IngestBankStatement(context.Background(), st, bankSample)
	if err != nil || s.Account != "4219" || len(st.bank) != 1 {
		t.Fatalf("summary %+v, %v", s, err)
	}
	if _, err := IngestBankStatement(context.Background(), st, "just some notes"); !errors.Is(err, ErrNotABankStatement) {
		t.Fatalf("err = %v", err)
	}
	if _, err := IngestBankStatement(context.Background(), st, "   "); !errors.Is(err, ErrEmptyStatement) {
		t.Fatalf("err = %v", err)
	}
}

func TestExecPDFTextWithNoBinaryIsAnError(t *testing.T) {
	_, err := (ExecPDFText{Candidates: []string{"/nonexistent/pdftotext-bridge-test"}}).Text(context.Background(), []byte("%PDF"))
	if !errors.Is(err, ErrNoPdftotext) {
		t.Fatalf("err = %v", err)
	}
}
