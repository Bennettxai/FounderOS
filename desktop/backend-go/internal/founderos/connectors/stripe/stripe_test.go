package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// Fixtures shaped like Stripe's real /v1/balance and /v1/charges objects.
const balanceFixture = `{
  "object": "balance",
  "available": [{"amount": 123456, "currency": "usd", "source_types": {"card": 123456}}],
  "pending": [{"amount": 5000, "currency": "usd", "source_types": {"card": 5000}}],
  "livemode": true
}`

func charge(id string, amount int64, created int64, extra string) string {
	if extra != "" {
		extra = "," + extra
	}
	return fmt.Sprintf(`{"id":%q,"object":"charge","amount":%d,"currency":"usd","paid":true,"status":"succeeded","refunded":false,"created":%d,"description":null,"receipt_email":null,"billing_details":{"email":null,"name":null},"customer":null%s}`, id, amount, created, extra)
}

func list(hasMore bool, rows ...string) string {
	return fmt.Sprintf(`{"object":"list","url":"/v1/charges","has_more":%v,"data":[%s]}`, hasMore, strings.Join(rows, ","))
}

func TestMetaAndAccountsMatchFounderosOS(t *testing.T) {
	if Meta.ID != "stripe" || Meta.Name != "Stripe" || Meta.Kind != connectors.KindPayments {
		t.Fatalf("Meta = %+v", Meta)
	}
	if LaunchpadCohort != (Account{ID: "stripe", Name: "Stripe", EnvKey: "STRIPE_SECRET_KEY", Venture: "launchpad-cohort"}) {
		t.Errorf("LC = %+v", LaunchpadCohort)
	}
	if Vantage != (Account{ID: "stripe-vantage", Name: "Stripe · Vantage", EnvKey: "STRIPE_VANTAGE_KEY", Venture: "vantage"}) {
		t.Errorf("Vantage = %+v", Vantage)
	}
}

func TestVantageKeysOffItsOwnKey(t *testing.T) {
	res := resolver(t, "STRIPE_SECRET_KEY=sk_x\n")
	if !New(res).Configured() {
		t.Error("LC must read STRIPE_SECRET_KEY")
	}
	if NewAccount(res, Vantage).Configured() {
		t.Error("Vantage must key off STRIPE_VANTAGE_KEY alone")
	}
}

func TestSnapshotReadsBalanceAndFiveRecentCharges(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_x" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/balance":
			w.Write([]byte(balanceFixture))
		case "/v1/charges":
			if r.URL.Query().Get("limit") != "5" {
				t.Errorf("limit = %q", r.URL.Query().Get("limit"))
			}
			w.Write([]byte(list(true,
				charge("ch_1", 600000, 1758000000, `"description":"Vantage LLC"`),
				charge("ch_2", 2500, 1757000000, ""),
			)))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer s.Close()
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	c.BaseURL = s.URL
	snap, err := c.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Available) != 1 || snap.Available[0] != (Money{Amount: 123456, Currency: "usd"}) {
		t.Errorf("available = %+v", snap.Available)
	}
	if len(snap.Pending) != 1 || snap.Pending[0].Amount != 5000 {
		t.Errorf("pending = %+v", snap.Pending)
	}
	want := []RecentCharge{
		{Amount: 600000, Currency: "usd", Description: "Vantage LLC", Created: 1758000000},
		{Amount: 2500, Currency: "usd", Description: "ch_2", Created: 1757000000}, // description ?? id
	}
	if len(snap.RecentCharges) != 2 || snap.RecentCharges[0] != want[0] || snap.RecentCharges[1] != want[1] {
		t.Errorf("recent = %+v", snap.RecentCharges)
	}
	out, _ := json.Marshal(snap)
	if !strings.Contains(string(out), `"recentCharges"`) {
		t.Errorf("JSON shape: %s", out)
	}
}

func TestSnapshotUnkeyedIsNotConfigured(t *testing.T) {
	_, err := New(resolver(t, "")).Snapshot(context.Background())
	if !errors.Is(err, ErrNotConfigured) || !strings.Contains(err.Error(), "STRIPE_SECRET_KEY is not set") {
		t.Fatalf("err = %v", err)
	}
}

// Stripe's list API paginates with has_more + starting_after=<last id>.
func TestListChargesPaginatesAndHonoursTheCap(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("created[gte]") != "1788220800" || q.Get("limit") != "100" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		if got := q["expand[]"]; len(got) != 1 || got[0] != "data.customer" {
			t.Errorf("expand = %v", got)
		}
		n := calls.Add(1)
		want := map[int32]string{1: "", 2: "ch_p1_99", 3: "ch_p2_99"}[n]
		if q.Get("starting_after") != want {
			t.Errorf("page %d starting_after = %q, want %q", n, q.Get("starting_after"), want)
		}
		rows := make([]string, 100)
		for i := range rows {
			rows[i] = charge(fmt.Sprintf("ch_p%d_%d", n, i), 100, 1757000000, "")
		}
		w.Write([]byte(list(true, rows...)))
	}))
	defer s.Close()
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	c.BaseURL = s.URL
	got, err := c.ListCharges(context.Background(), ListParams{CreatedGTE: 1788220800, ExpandCustomer: true, Max: 250})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 250 || calls.Load() != 3 {
		t.Fatalf("got %d charges over %d calls, want 250 over 3", len(got), calls.Load())
	}
}

func TestListChargesStopsWhenHasMoreIsFalse(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(list(false, charge("ch_1", 1, 1, ""))))
	}))
	defer s.Close()
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	c.BaseURL = s.URL
	got, err := c.ListCharges(context.Background(), ListParams{})
	if err != nil || len(got) != 1 || calls.Load() != 1 {
		t.Fatalf("got %d, calls %d, err %v", len(got), calls.Load(), err)
	}
}

func TestListChargesErrorMidPaginationFailsTheRead(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Write([]byte(list(true, charge("ch_1", 1, 1, ""))))
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"Too many requests","type":"rate_limit_error"}}`))
	}))
	defer s.Close()
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	c.BaseURL = s.URL
	got, err := c.ListCharges(context.Background(), ListParams{})
	if err == nil || got != nil || !strings.Contains(err.Error(), "Too many requests") {
		t.Fatalf("a partial list must not pass as complete: %v, %v", got, err)
	}
}

func TestMonthStartUnixAndSumChargeIncome(t *testing.T) {
	now := time.Date(2026, 9, 29, 15, 0, 0, 0, time.FixedZone("CDT", -5*3600))
	if got := MonthStartUnix(now); got != 1788238800 { // 2026-09-01T00:00:00 CDT (05:00Z)
		t.Errorf("MonthStartUnix = %d", got)
	}
	got := SumChargeIncome([]Charge{
		{Amount: 1000, Currency: "usd", Paid: true, Status: "succeeded"},
		{Amount: 9999, Currency: "usd", Paid: false, Status: "failed"},
		{Amount: 7777, Currency: "usd", Paid: true, Status: "pending"},
		{Amount: 500, Currency: "eur", Paid: true, Status: "succeeded"},
	})
	if got != (IncomeMTD{AmountCents: 1500, Currency: "eur", Count: 2}) {
		t.Errorf("SumChargeIncome = %+v", got)
	}
	if got := SumChargeIncome(nil); got != (IncomeMTD{Currency: "usd"}) {
		t.Errorf("empty = %+v", got)
	}
}

func TestMonthToDateIncome(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("created[gte]") != "1788238800" { // Sept 1 00:00 Chicago
			t.Errorf("gte = %q", r.URL.Query().Get("created[gte]"))
		}
		w.Write([]byte(list(false,
			charge("ch_1", 600000, 1758000000, ""),
			charge("ch_2", 100, 1758000001, `"paid":false,"status":"failed"`),
		)))
	}))
	defer s.Close()
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)

	if got, err := NewAccount(resolver(t, ""), Vantage).MonthToDateIncome(context.Background(), now); got != nil || err != nil {
		t.Fatalf("unkeyed must be (nil, nil), got %v, %v", got, err)
	}
	c := NewAccount(resolver(t, "STRIPE_VANTAGE_KEY=sk_m\n"), Vantage)
	c.BaseURL = s.URL
	got, err := c.MonthToDateIncome(context.Background(), now)
	if err != nil || got == nil || *got != (IncomeMTD{AmountCents: 600000, Currency: "usd", Count: 1}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestMapChargesKeepsOnlySettledMoneyAndResolvesIdentity(t *testing.T) {
	raw := list(false,
		charge("ch_bill", 600000, 1788998400, `"description":" Mentorship ","billing_details":{"email":" A@x.com ","name":"Ann"},"receipt_email":"r@x.com"`),
		charge("ch_receipt", 5000, 1788998400, `"receipt_email":"r@x.com","customer":{"id":"cus_1","object":"customer","email":"c@x.com","name":"Cus"}`),
		charge("ch_cust", 5000, 1788998400, `"customer":{"id":"cus_2","object":"customer","email":"c@x.com","name":"Cus"}`),
		charge("ch_deleted", 5000, 1788998400, `"customer":{"id":"cus_3","object":"customer","deleted":true,"email":"gone@x.com"}`),
		charge("ch_idonly", 5000, 1788998400, `"customer":"cus_4"`),
		charge("ch_refunded", 5000, 1788998400, `"refunded":true`),
		charge("ch_failed", 5000, 1788998400, `"paid":false,"status":"failed"`),
		charge("ch_zero", 0, 1788998400, ""),
	)
	var page chargeList
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}
	wins := MapCharges(page.Data, "vantage")
	ids := []string{}
	for _, w := range wins {
		ids = append(ids, w.ID)
	}
	if strings.Join(ids, ",") != "ch_bill,ch_receipt,ch_cust,ch_deleted,ch_idonly" {
		t.Fatalf("ids = %v", ids)
	}
	str := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	w := wins[0]
	if str(w.Email) != "A@x.com" || str(w.Name) != "Ann" || str(w.Product) != "Mentorship" || w.AmountUSD != 6000 || w.At != "2026-09-10" || w.Venture != "vantage" {
		t.Errorf("bill = %+v", w)
	}
	if str(wins[1].Email) != "r@x.com" || str(wins[1].Name) != "Cus" {
		t.Errorf("receipt email beats customer email: %+v", wins[1])
	}
	if str(wins[2].Email) != "c@x.com" {
		t.Errorf("expanded customer email: %+v", wins[2])
	}
	if wins[3].Email != nil || wins[3].Name != nil {
		t.Errorf("a deleted customer carries no identity: %+v", wins[3])
	}
	if wins[4].Email != nil || wins[4].Product != nil {
		t.Errorf("anonymous charge still surfaces, unnamed: %+v", wins[4])
	}
	out, _ := json.Marshal(wins[4])
	if !strings.Contains(string(out), `"email":null`) || !strings.Contains(string(out), `"amountUsd":50`) {
		t.Errorf("JSON shape: %s", out)
	}
}

func TestFunnelWinsReadsNinetyDaysWithCustomersExpanded(t *testing.T) {
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantGte := strconv.FormatInt(now.Unix()-90*86400, 10)
		if r.URL.Query().Get("created[gte]") != wantGte || r.URL.Query().Get("expand[]") != "data.customer" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		w.Write([]byte(list(false, charge("ch_1", 100000, 1788998400, ""))))
	}))
	defer s.Close()
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	c.BaseURL = s.URL
	wins, err := c.FunnelWins(context.Background(), now)
	if err != nil || len(wins) != 1 || wins[0].Venture != "launchpad-cohort" {
		t.Fatalf("wins = %+v, %v", wins, err)
	}
	if _, err := New(resolver(t, "")).FunnelWins(context.Background(), now); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("unkeyed err = %v", err)
	}
}

func TestStatusAllThreeStates(t *testing.T) {
	if st := New(resolver(t, "")).Status(context.Background()); st.State != connectors.StateNotConfigured || !strings.Contains(st.Detail, "STRIPE_SECRET_KEY") {
		t.Fatalf("not configured: %+v", st)
	}

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/balance" {
			w.Write([]byte(balanceFixture))
			return
		}
		w.Write([]byte(list(false)))
	}))
	defer ok.Close()
	c := NewAccount(resolver(t, "STRIPE_VANTAGE_KEY=sk_m\n"), Vantage)
	c.BaseURL = ok.URL
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected || st.Detail != "Stripe · Vantage · available balance 1234.56 USD" || st.ID != "stripe-vantage" {
		t.Fatalf("connected: %+v", st)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API Key provided: sk_live_****abcd","type":"invalid_request_error"}}`))
	}))
	c.BaseURL = bad.URL
	st = c.Status(context.Background())
	if st.State != connectors.StateError || st.Detail != "Stripe key set but verification failed: Invalid API Key provided: sk_live_****abcd" {
		t.Fatalf("auth failure: %+v", st)
	}
	bad.Close()
	if st := c.Status(context.Background()); st.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", st)
	}
}

// Stripe is read-only here: no refund, payout or charge path exists, and the
// client is the guarded one.
func TestClientRefusesWritesWhenDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	c := New(resolver(t, "STRIPE_SECRET_KEY=sk_x\n"))
	req, _ := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/refunds", strings.NewReader("charge=ch_1"))
	if _, err := c.client.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("want ErrWritesDisabled, got %v", err)
	}
}

// Income is what was kept: a refunded charge adds nothing, a partial refund
// only its remainder. (FounderOS v1 summed the gross amount; a finance number
// that counts refunded money as income is wrong, so the bridge does not.)
func TestSumChargeIncomeNetsOutRefunds(t *testing.T) {
	got := SumChargeIncome([]Charge{
		{Amount: 10000, Currency: "usd", Paid: true, Status: "succeeded"},
		{Amount: 5000, AmountRefunded: 5000, Refunded: true, Currency: "usd", Paid: true, Status: "succeeded"},
		{Amount: 4000, AmountRefunded: 1500, Currency: "usd", Paid: true, Status: "succeeded"},
		{Amount: 900, Currency: "usd", Paid: false, Status: "failed"},
	})
	if got.AmountCents != 12500 || got.Count != 2 {
		t.Fatalf("income = %+v, want 12500 cents over 2 charges", got)
	}
}

// Month to date is the operator's month (America/Chicago): at 20:00 CDT on
// Sept 30 it is still September, though UTC has already reached October.
func TestMonthStartIsChicagoMonth(t *testing.T) {
	now := time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC)        // 2026-09-30 20:00 CDT
	want := time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC).Unix() // Sept 1 00:00 CDT
	if got := MonthStartUnix(now); got != want {
		t.Fatalf("month start = %s, want %s", time.Unix(got, 0).UTC(), time.Unix(want, 0).UTC())
	}
}
