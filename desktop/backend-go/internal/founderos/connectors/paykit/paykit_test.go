package paykit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv(LaunchpadCohort.EnvKey, "")
	t.Setenv(Vantage.EnvKey, "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

func sp(s string) *string { return &s }

func cust(id string, cents int64, txns int, at string) Customer {
	c := Customer{ID: id, TotalSpentCents: cents, Transactions: txns}
	if at != "" {
		c.LastTransactionDate = sp(at)
		c.Month = sp(at[:7])
	}
	return c
}

func snap(on string, cs ...Customer) Snapshot {
	return Snapshot{CapturedOn: on, Source: SourceLive, Customers: cs}
}

// A /public-api/customers page shaped like PayKit's real response. Names,
// emails and phones ride along in the payload and are dropped by the parser.
func page(rows ...string) string {
	return `{"status":"success","data":{"customers":[` + strings.Join(rows, ",") + `]}}`
}

func row(id any, spent any, at string, txns any) string {
	m := map[string]any{"name": "Pat Doe", "email": "pat@example.com", "phone": "+15550100"}
	if id != nil {
		m["id"] = id
	}
	if spent != nil {
		m["total_spent"] = spent
	}
	if at != "" {
		m["last_transaction_date"] = at
	}
	if txns != nil {
		m["total_transactions"] = txns
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func TestMetaAndAccountsMatchFounderosOS(t *testing.T) {
	if Meta.ID != "paykit-lc" || Meta.Name != "PayKit · Launchpad Cohort" || Meta.Kind != connectors.KindPayments {
		t.Fatalf("Meta = %+v", Meta)
	}
	if LaunchpadCohort != (Account{ID: "paykit-lc", Name: "PayKit · Launchpad Cohort", EnvKey: "PAYKIT_LC_KEY"}) {
		t.Errorf("LC = %+v", LaunchpadCohort)
	}
	if Vantage != (Account{ID: "paykit-vantage", Name: "PayKit · Vantage", EnvKey: "PAYKIT_VANTAGE_KEY"}) {
		t.Errorf("Vantage = %+v", Vantage)
	}
}

// ---- parse --------------------------------------------------------------

func TestParseCustomersNormalizes(t *testing.T) {
	got := ParseCustomers([]byte(page(
		row(1, "3400.00", "2026-06-10T12:12:13-05:00", 1),
		row(2, "1,250.00", "2026-06-02T09:00:00-05:00", 1),
		row(3, "500.00", "2026-05-20T09:00:00-05:00", 1),
	)))
	want := []Customer{
		cust("1", 340000, 1, "2026-06-10T12:12:13-05:00"),
		cust("2", 125000, 1, "2026-06-02T09:00:00-05:00"),
		cust("3", 50000, 1, "2026-05-20T09:00:00-05:00"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if SumMonthCents(got, "2026-06") != 465000 || SumMonthCents(got, "2026-05") != 50000 {
		t.Error("SumMonthCents")
	}
	out, _ := json.Marshal(got[0])
	if string(out) != `{"id":"1","totalSpentCents":340000,"month":"2026-06","transactions":1,"lastTransactionDate":"2026-06-10T12:12:13-05:00"}` {
		t.Errorf("JSON shape: %s", out)
	}
	if strings.Contains(string(out), "pat@") {
		t.Error("PII must be dropped")
	}
}

func TestParseCustomersKeepsLargeIDsExactAndAbsentIDEmpty(t *testing.T) {
	got := ParseCustomers([]byte(page(row(1922443, 1500, "", nil), row(nil, "1.00", "", nil), row("abc", 2.5, "", "3"))))
	if got[0].ID != "1922443" || got[0].TotalSpentCents != 150000 {
		t.Errorf("numeric id / numeric spend: %+v", got[0])
	}
	if got[1].ID != "" {
		t.Errorf("absent id must be empty: %+v", got[1])
	}
	if got[2].ID != "abc" || got[2].TotalSpentCents != 250 || got[2].Transactions != 3 {
		t.Errorf("string id / string count: %+v", got[2])
	}
	if got[0].Month != nil || got[0].LastTransactionDate != nil {
		t.Errorf("absent date must be nil: %+v", got[0])
	}
}

func TestParseCustomersRejectsJunk(t *testing.T) {
	for _, junk := range []string{`null`, `{}`, `{"data":{}}`, `{"data":{"customers":{}}}`, `nope`} {
		if got := ParseCustomers([]byte(junk)); len(got) != 0 {
			t.Errorf("%s → %+v", junk, got)
		}
	}
}

// A missing or garbled total_transactions must NOT be read as a one-time buyer:
// that assumption overstated six months (FOS-655).
func TestAbsentOrGarbledCountIsUnsplittable(t *testing.T) {
	cs := ParseCustomers([]byte(page(
		row(nil, "100.00", "2026-06-01 09:00:00", nil),
		row(nil, "200.00", "2026-06-02 09:00:00", "many"),
		row(nil, "garbage", "2026-06-03 09:00:00", 0),
	)))
	for _, c := range cs {
		if c.Transactions != 0 {
			t.Errorf("transactions = %d, want 0", c.Transactions)
		}
	}
	if cs[2].TotalSpentCents != 0 {
		t.Errorf("unparseable spend reads 0 cents, got %d", cs[2].TotalSpentCents)
	}
	if got := MonthIncomeCents(cs, "2026-06"); got != (MonthIncome{ExactCents: 0, UpperCents: 30000, UnsplittableCustomers: 3}) {
		t.Errorf("got %+v", got)
	}
}

// ---- the lifetime band (FOS-655) ---------------------------------------

func TestMonthIncomeRefusesToGuess(t *testing.T) {
	sept := ParseCustomers([]byte(page(row(nil, "2,000.00", "2026-09-10 10:00:00", 3))))
	if got := MonthIncomeCents(sept, "2026-09"); got != (MonthIncome{0, 200000, 1}) {
		t.Errorf("September shape: %+v", got)
	}
	aug := ParseCustomers([]byte(page(
		row(nil, "1,500.00", "2026-08-09 10:00:00", 1),
		row(nil, "1,500.00", "2026-08-09 11:00:00", 1),
	)))
	if got := MonthIncomeCents(aug, "2026-08"); got != (MonthIncome{300000, 300000, 0}) {
		t.Errorf("one-time buyers collapse the band: %+v", got)
	}
	oct := ParseCustomers([]byte(page(
		row(nil, "9,125.00", "2025-10-07 10:00:00", 1),
		row(nil, "1,250.00", "2025-10-07 11:00:00", 2),
		row(nil, "6,000.00", "2025-09-01 11:00:00", 1),
	)))
	if got := MonthIncomeCents(oct, "2025-10"); got != (MonthIncome{912500, 1037500, 1}) {
		t.Errorf("mixed month: %+v", got)
	}
}

// ---- snapshot differencing (FOS-658) -----------------------------------

func TestMonthsSpanned(t *testing.T) {
	cases := map[[2]string][]string{
		{"2026-09-01", "2026-09-30"}: {"2026-09"},
		{"2026-08-20", "2026-09-17"}: {"2026-08", "2026-09"},
		{"2025-11-30", "2026-02-01"}: {"2025-11", "2025-12", "2026-01", "2026-02"},
	}
	for in, want := range cases {
		if got := MonthsSpanned(in[0], in[1]); !reflect.DeepEqual(got, want) {
			t.Errorf("%v → %v, want %v", in, got, want)
		}
	}
}

func TestDiffSnapshots(t *testing.T) {
	d := DiffSnapshots(snap("2026-09-10", cust("a", 100000, 2, "2026-09-04T09:00:00-05:00")), snap("2026-09-11", cust("a", 250000, 4, "2026-09-11T09:00:00-05:00")))
	if !reflect.DeepEqual(d, WindowDelta{ExactCents: map[string]int64{"2026-09": 150000}, UpperCents: map[string]int64{"2026-09": 150000}}) {
		t.Errorf("one-month window is exact for a repeat buyer: %+v", d)
	}

	d = DiffSnapshots(snap("2026-08-31", cust("a", 100000, 2, "2026-07-02T09:00:00-05:00")), snap("2026-09-01", cust("a", 200000, 3, "2026-09-01T09:00:00-05:00")))
	if !reflect.DeepEqual(d.ExactCents, map[string]int64{"2026-09": 100000}) || d.AmbiguousCustomers != 0 {
		t.Errorf("a single new transaction dates itself: %+v", d)
	}

	d = DiffSnapshots(snap("2026-08-31", cust("a", 100000, 1, "2026-08-02T09:00:00-05:00")), snap("2026-09-01", cust("a", 400000, 3, "2026-09-01T09:00:00-05:00")))
	if len(d.ExactCents) != 0 || !reflect.DeepEqual(d.UpperCents, map[string]int64{"2026-08": 300000, "2026-09": 300000}) || d.AmbiguousCustomers != 1 {
		t.Errorf("several txns across a boundary must not split: %+v", d)
	}

	d = DiffSnapshots(snap("2026-09-10"), snap("2026-09-11", cust("new", 500000, 1, "2026-09-11T09:00:00-05:00")))
	if !reflect.DeepEqual(d.ExactCents, map[string]int64{"2026-09": 500000}) {
		t.Errorf("a new customer is all-new money: %+v", d)
	}

	d = DiffSnapshots(snap("2026-09-10", cust("a", 500000, 2, "2026-08-01T09:00:00-05:00")), snap("2026-09-11", cust("a", 350000, 2, "2026-08-01T09:00:00-05:00")))
	if !reflect.DeepEqual(d, WindowDelta{ExactCents: map[string]int64{}, UpperCents: map[string]int64{}}) {
		t.Errorf("a decrease is reported nowhere: %+v", d)
	}
}

func one(on string, cents int64, txns int, at string) Snapshot {
	c := cust("a", cents, txns, at)
	return snap(on, c)
}

func TestMonthFromSnapshotsCoversOnlyWhatItCanProve(t *testing.T) {
	if MonthFromSnapshots([]Snapshot{one("2026-09-17", 100000, 1, "")}, "2026-09") != nil {
		t.Error("fewer than two snapshots → nil")
	}
	late := []Snapshot{one("2026-09-05", 100000, 1, ""), one("2026-09-17", 200000, 2, "2026-09-10T09:00:00-05:00")}
	if MonthFromSnapshots(late, "2026-09") != nil || MonthFromSnapshots(late, "2026-08") != nil {
		t.Error("a month that began before the first snapshot → nil")
	}
	onFirst := []Snapshot{one("2026-09-01", 100000, 1, ""), one("2026-09-17", 200000, 2, "2026-09-10T09:00:00-05:00")}
	if MonthFromSnapshots(onFirst, "2026-09") != nil {
		t.Error("a snapshot ON the 1st is too late")
	}
	// Out of order on purpose: the function sorts.
	sept := []Snapshot{one("2026-09-17", 200000, 3, "2026-09-10T14:49:30-05:00"), one("2026-08-20", 100000, 2, "")}
	if got := MonthFromSnapshots(sept, "2026-09"); got == nil || *got != (MonthIncome{100000, 100000, 0}) {
		t.Errorf("September 2026 recovered exactly: %+v", got)
	}
	weekly := []Snapshot{
		one("2026-08-31", 0, 0, ""),
		one("2026-09-05", 30000, 1, "2026-09-05T09:00:00-05:00"),
		one("2026-09-12", 80000, 2, "2026-09-12T09:00:00-05:00"),
		one("2026-09-19", 95000, 3, "2026-09-19T09:00:00-05:00"),
	}
	if got := MonthFromSnapshots(weekly, "2026-09"); got == nil || got.ExactCents != 95000 {
		t.Errorf("sums across windows: %+v", got)
	}
}

// ---- the reconstructed seed and snapshot rows (paykit.db) ------------

func TestSeedReproducesBEN234Aggregate(t *testing.T) {
	s := Seed20260820
	if s.CapturedOn != "2026-08-20" || s.Source != SourceReconstructed || len(s.Customers) != 31 {
		t.Fatalf("seed header: %s %s %d", s.CapturedOn, s.Source, len(s.Customers))
	}
	var txns int
	var cents int64
	lo, hi := "9999", ""
	for _, c := range s.Customers {
		txns += c.Transactions
		cents += c.TotalSpentCents
		if c.LastTransactionDate != nil {
			if d := *c.LastTransactionDate; d < lo {
				lo = d
			}
			if d := *c.LastTransactionDate; d > hi {
				hi = d
			}
		}
	}
	if txns != 42 || cents != 9648900 || lo[:10] != "2025-07-10" || hi[:10] != "2026-08-09" {
		t.Fatalf("aggregate: %d txns, %d cents, %s → %s", txns, cents, lo, hi)
	}
}

func TestSnapshotRowsSkipIDlessCustomersAndRoundTrip(t *testing.T) {
	s := snap("2026-09-17", cust("452730", 200000, 3, "2026-09-10T14:49:30-05:00"), cust("", 999, 1, ""), cust("1", 5, 1, ""))
	rows := SnapshotRows("paykit-lc", s)
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	want := SnapshotRow{Account: "paykit-lc", CapturedOn: "2026-09-17", CustomerID: "452730", TotalSpentCents: 200000, TotalTransactions: 3, LastTransactionDate: sp("2026-09-10T14:49:30-05:00"), Source: "live"}
	if !reflect.DeepEqual(rows[0], want) {
		t.Errorf("row = %+v", rows[0])
	}
	back := SnapshotsFromRows(append(SnapshotRows("paykit-lc", Seed20260820), rows...))
	if len(back) != 2 || back[0].CapturedOn != "2026-08-20" || back[0].Source != SourceReconstructed || back[1].Source != SourceLive {
		t.Fatalf("round trip: %+v", back)
	}
	if !reflect.DeepEqual(back[1].Customers[0], s.Customers[0]) {
		t.Errorf("customer round trip: %+v", back[1].Customers[0])
	}
}

func TestSeedRowsOnlyForAAAndOnlyWhenAbsent(t *testing.T) {
	if got := SeedRows("paykit-lc", nil); len(got) != 31 {
		t.Errorf("LC with no history gets the seed: %d", len(got))
	}
	if got := SeedRows("paykit-lc", []string{"2026-08-20"}); got != nil {
		t.Error("a captured 2026-08-20 must never be overwritten by the inference")
	}
	if got := SeedRows("paykit-vantage", nil); got != nil {
		t.Error("the seed is LC's only")
	}
}

func TestMemoryHistoryReplacesADayWholesale(t *testing.T) {
	h := NewMemoryHistory("paykit-lc")
	if dates := h.CapturedDates(); !reflect.DeepEqual(dates, []string{"2026-08-20"}) {
		t.Fatalf("LC history starts seeded: %v", dates)
	}
	h.Record(snap("2026-09-17", cust("a", 1, 1, ""), cust("b", 2, 1, "")))
	h.Record(snap("2026-09-17", cust("a", 3, 1, "")))
	ss, _ := h.Snapshots()
	if len(ss) != 2 || len(ss[1].Customers) != 1 || ss[1].Customers[0].TotalSpentCents != 3 {
		t.Fatalf("a later same-day pull replaces the day: %+v", ss)
	}
	if got := NewMemoryHistory("paykit-vantage").CapturedDates(); len(got) != 0 {
		t.Errorf("Vantage history starts empty: %v", got)
	}
}

// ---- the live pull ------------------------------------------------------

type pages struct {
	calls atomic.Int32
	serve func(n int32, w http.ResponseWriter, r *http.Request)
}

func (p *pages) server(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := p.calls.Add(1)
		if r.Header.Get("x-api-key") != "key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.URL.Path != "/customers" && r.URL.Query().Get("per_page") != "1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		p.serve(n, w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func pull(t *testing.T, s *httptest.Server) *Connector {
	t.Helper()
	c := New(resolver(t, "PAYKIT_LC_KEY=key\n"))
	c.BaseURL = s.URL
	c.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
	return c
}

func TestCustomersPaginatesUntilAnEmptyPage(t *testing.T) {
	p := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != strconv.Itoa(int(n)) || q.Get("per_page") != "100" {
			t.Errorf("page %d query = %s", n, r.URL.RawQuery)
		}
		if n <= 2 {
			rows := make([]string, 100)
			for i := range rows {
				rows[i] = row(fmt.Sprintf("%d-%d", n, i), "10.00", "2026-08-09 12:00:00", 1)
			}
			w.Write([]byte(page(rows...)))
			return
		}
		w.Write([]byte(page()))
	}}
	got, err := pull(t, p.server(t)).Customers(context.Background())
	if err != nil || len(got) != 200 || p.calls.Load() != 3 {
		t.Fatalf("got %d customers over %d calls, err %v", len(got), p.calls.Load(), err)
	}
}

// The funnel (lib/funnel-paykit.ts) needs each buyer's identity, which
// Customers drops: CustomerPages hands back the raw pages, all or nothing.
func TestCustomerPagesAreRawAndAllOrNothing(t *testing.T) {
	p := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			w.Write([]byte(page(row("7", "1,500.00", "2026-09-10T10:00:00-05:00", 1))))
			return
		}
		w.Write([]byte(page()))
	}}
	got, err := pull(t, p.server(t)).CustomerPages(context.Background())
	if err != nil || len(got) != 1 || !strings.Contains(string(got[0]), `"email":"pat@example.com"`) || p.calls.Load() != 2 {
		t.Fatalf("pages = %q over %d calls, err %v", got, p.calls.Load(), err)
	}
	bad := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		if n == 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(page(row(n, "1.00", "", 1))))
	}}
	if got, err := pull(t, bad.server(t)).CustomerPages(context.Background()); err == nil || got != nil {
		t.Fatalf("a failed page must fail the pull: %q %v", got, err)
	}
	if _, err := New(resolver(t, "")).CustomerPages(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unkeyed = %v", err)
	}
}

func TestCustomersStopsAtFiftyPages(t *testing.T) {
	p := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(page(row(n, "1.00", "", 1))))
	}}
	got, err := pull(t, p.server(t)).Customers(context.Background())
	if err != nil || len(got) != 50 || p.calls.Load() != 50 {
		t.Fatalf("got %d over %d calls, err %v", len(got), p.calls.Load(), err)
	}
}

// A dead key once rendered "live · $0 this month" over $4,500 of real August
// income (2026-08-16). Any failure is an error, never a zero.
func TestMonthToDateIncomeHonesty(t *testing.T) {
	if got, err := New(resolver(t, "")).MonthToDateIncome(context.Background(), "2026-08", nil); got != nil || err != nil {
		t.Fatalf("unkeyed → (nil, nil), got %v, %v", got, err)
	}

	p := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthenticated."}`))
	}}
	if got, err := pull(t, p.server(t)).MonthToDateIncome(context.Background(), "2026-08", nil); got != nil || err == nil {
		t.Fatalf("auth failure must be an error, got %v, %v", got, err)
	}

	mid := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			w.Write([]byte(page(row(nil, "10.00", "2026-08-09 12:00:00", nil))))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}}
	if got, err := pull(t, mid.server(t)).MonthToDateIncome(context.Background(), "2026-08", nil); got != nil || err == nil {
		t.Fatalf("a mid-pagination failure must not return a partial sum, got %v, %v", got, err)
	}
}

func servePages(first string) *pages {
	return &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			w.Write([]byte(first))
			return
		}
		w.Write([]byte(page()))
	}}
}

func TestMonthToDateIncomeBandWithoutHistory(t *testing.T) {
	aug := servePages(page(
		row(nil, "1,500.00", "2026-08-09 10:00:00", 1),
		row(nil, "1,500.00", "2026-08-09 11:00:00", 1),
		row(nil, "1,500.00", "2026-08-09 12:00:00", 1),
		row(nil, "999.00", "2026-07-01 09:00:00", 1),
	))
	got, err := pull(t, aug.server(t)).MonthToDateIncome(context.Background(), "2026-08", nil)
	if err != nil || *got != (MonthIncomeUSD{ExactUSD: 4500, UpperUSD: 4500}) {
		t.Fatalf("August cohort: %+v, %v", got, err)
	}
	sept := servePages(page(row(nil, "2,000.00", "2026-09-10 10:00:00", 3)))
	got, err = pull(t, sept.server(t)).MonthToDateIncome(context.Background(), "", nil)
	if err != nil || *got != (MonthIncomeUSD{ExactUSD: 0, UpperUSD: 2000, UnsplittableCustomers: 1}) {
		t.Fatalf("repeat buyer band, default month = now's: %+v, %v", got, err)
	}
	out, _ := json.Marshal(got)
	if string(out) != `{"exactUsd":0,"upperUsd":2000,"unsplittableCustomers":1}` {
		t.Errorf("JSON shape: %s", out)
	}
}

func TestMonthToDateIncomeWithHistoryTurnsTheBandExact(t *testing.T) {
	h := NewMemoryHistory("paykit-vantage") // unseeded; supply the August point by hand
	h.Record(Snapshot{CapturedOn: "2026-08-20", Source: SourceReconstructed, Customers: []Customer{{ID: "452730", TotalSpentCents: 100000, Transactions: 2}}})
	p := servePages(page(row(452730, "2,000.00", "2026-09-10T14:49:30-05:00", 3)))
	got, err := pull(t, p.server(t)).MonthToDateIncome(context.Background(), "2026-09", h)
	if err != nil || *got != (MonthIncomeUSD{ExactUSD: 1000, UpperUSD: 1000}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	if dates := h.CapturedDates(); !reflect.DeepEqual(dates, []string{"2026-08-20", "2026-09-29"}) {
		t.Errorf("the pull is recorded as today's snapshot: %v", dates)
	}
}

type brokenHistory struct{}

func (brokenHistory) Record(Snapshot) error          { return errors.New("disk full") }
func (brokenHistory) Snapshots() ([]Snapshot, error) { return nil, errors.New("disk full") }

func TestMonthToDateIncomeDegradesToTheBandWhenTheStoreFails(t *testing.T) {
	p := servePages(page(row(1, "2,000.00", "2026-09-10 10:00:00", 3)))
	got, err := pull(t, p.server(t)).MonthToDateIncome(context.Background(), "2026-09", brokenHistory{})
	if err != nil || *got != (MonthIncomeUSD{ExactUSD: 0, UpperUSD: 2000, UnsplittableCustomers: 1}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// ---- status -------------------------------------------------------------

func TestStatusAllThreeStates(t *testing.T) {
	if st := NewAccount(resolver(t, ""), Vantage).Status(context.Background()); st.State != connectors.StateNotConfigured || !strings.Contains(st.Detail, "PAYKIT_VANTAGE_KEY") || st.ID != "paykit-vantage" {
		t.Fatalf("not configured: %+v", st)
	}
	ok := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "1" {
			t.Errorf("status reads one row, got %s", r.URL.RawQuery)
		}
		w.Write([]byte(page(row(1, "1.00", "", 1))))
	}}
	c := pull(t, ok.server(t))
	if st := c.Status(context.Background()); st.State != connectors.StateConnected || st.Detail != "PayKit · Launchpad Cohort key verified" {
		t.Fatalf("connected: %+v", st)
	}
	bad := &pages{serve: func(n int32, w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }}
	c.BaseURL = bad.server(t).URL
	if st := c.Status(context.Background()); st.State != connectors.StateError || !strings.Contains(st.Detail, "401") {
		t.Fatalf("auth failure: %+v", st)
	}
	c.BaseURL = "http://127.0.0.1:1"
	if st := c.Status(context.Background()); st.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", st)
	}
}

func TestClientRefusesWritesWhenDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := New(resolver(t, "PAYKIT_LC_KEY=key\n"))
	req, _ := http.NewRequest(http.MethodPost, "https://www.paykit.com/public-api/refunds", strings.NewReader("{}"))
	if _, err := c.client.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("want ErrWritesDisabled, got %v", err)
	}
}
