package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
)

var allKeys = []string{
	"STRIPE_SECRET_KEY", "STRIPE_VANTAGE_KEY", "PAYPAL_CLIENT_ID", "PAYPAL_CLIENT_SECRET",
	"PAYKIT_VANTAGE_KEY", "PAYKIT_LC_KEY", "WISE_1_TOKEN", "WISE_2_TOKEN", "FUNNEL_PROVIDER",
}

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	for _, k := range allKeys {
		t.Setenv(k, "")
	}
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "payments" || Meta.Name != "Payment Processors" || Meta.Kind != connectors.KindPayments {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestConfiguredProcessorsListsTheSixRealOnes(t *testing.T) {
	procs := ConfiguredProcessors(env(
		"STRIPE_SECRET_KEY", "sk_x", "STRIPE_VANTAGE_KEY", "sk_m", "PAYPAL_CLIENT_ID", "a",
		"PAYPAL_CLIENT_SECRET", "b", "PAYKIT_VANTAGE_KEY", "m", "WISE_1_TOKEN", "w1",
	))
	want := []ProcessorInfo{
		{"stripe", "Stripe", true},
		{"stripe-vantage", "Stripe · Vantage", true},
		{"paypal", "PayPal", true},
		{"paykit-vantage", "PayKit · Vantage", true},
		{"paykit-lc", "PayKit · Launchpad Cohort", false},
		{"wise-1", "Wise", true},
	}
	if !reflect.DeepEqual(procs, want) {
		t.Fatalf("got %+v", procs)
	}
	if ConfiguredProcessors(env("PAYPAL_CLIENT_ID", "a"))[2].Configured {
		t.Error("paypal needs both id and secret")
	}
	if ConfiguredProcessors(env("STRIPE_SECRET_KEY", "sk"))[1].Configured {
		t.Error("Vantage keys off STRIPE_VANTAGE_KEY alone")
	}
	for _, p := range ConfiguredProcessors(env("WISE_2_TOKEN", "w2")) {
		if p.ID == "wise-2" || p.ID == "square" || p.ID == "whop" {
			t.Errorf("dropped slot %s is back", p.ID)
		}
	}
	out, _ := json.Marshal(want[0])
	if string(out) != `{"id":"stripe","name":"Stripe","configured":true}` {
		t.Errorf("JSON shape: %s", out)
	}
}

// fake serves every provider's endpoints from one server, keyed by path and
// credential so the Stripe accounts can be told apart.
type fake struct {
	stripeBalance   int
	stripeFail      bool
	vantageFail     bool
	paykitFail      bool
	wiseFail        bool
	paykitCustomers string
}

func (f *fake) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/balance") || strings.HasPrefix(r.URL.Path, "/v1/charges"):
			if (auth == "Bearer sk_x" && f.stripeFail) || (auth == "Bearer sk_m" && f.vantageFail) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"Invalid API Key provided: sk_****"}}`))
				return
			}
			if r.URL.Path == "/v1/balance" {
				w.Write([]byte(`{"object":"balance","available":[{"amount":` + itoa(f.stripeBalance) + `,"currency":"usd"}],"pending":[{"amount":2500,"currency":"usd"}]}`))
				return
			}
			amount := "600000"
			if auth == "Bearer sk_m" {
				amount = "150000"
			}
			w.Write([]byte(`{"object":"list","has_more":false,"data":[{"id":"ch_` + auth[7:] + `","amount":` + amount + `,"currency":"usd","paid":true,"status":"succeeded","refunded":false,"created":1788998400,"description":"Mentorship","billing_details":{"email":"a@x.com","name":"Ann"},"customer":null}]}`))
		case r.URL.Path == "/customers":
			if f.paykitFail {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.URL.Query().Get("page") != "1" {
				w.Write([]byte(`{"data":{"customers":[]}}`))
				return
			}
			w.Write([]byte(f.paykitCustomers))
		case r.URL.Path == "/v1/transfers" || r.URL.Path == "/v1/profiles":
			if f.wiseFail {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.URL.Path == "/v1/profiles" {
				w.Write([]byte(`[{"id":1}]`))
				return
			}
			w.Write([]byte(`[{"targetValue":250.5,"targetCurrency":"USD","status":"outgoing_payment_sent","created":"2026-09-10 12:00:00","reference":"rent"}]`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func newWith(t *testing.T, f *fake, envLocal string) *Connector {
	t.Helper()
	s := httptest.NewServer(f.handler(t))
	t.Cleanup(s.Close)
	c := New(resolver(t, envLocal))
	c.PointAt(s.URL)
	c.Now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }
	return c
}

func TestStatusNotConfigured(t *testing.T) {
	st := New(resolver(t, "")).Status(context.Background())
	want := connectors.Status{
		ID: "payments", Name: "Payment Processors", Kind: connectors.KindPayments, State: connectors.StateNotConfigured,
		Detail: "None of 6 processors configured. Start with STRIPE_SECRET_KEY in ~/.founderos/.env or under API keys.",
		Meta:   map[string]any{"configured": 0, "known": 6},
	}
	if !reflect.DeepEqual(st, want) {
		t.Fatalf("got %+v", st)
	}
}

func TestStatusConnectedVerifiesStripe(t *testing.T) {
	c := newWith(t, &fake{stripeBalance: 123456}, "STRIPE_SECRET_KEY=sk_x\nPAYKIT_LC_KEY=fb\n")
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected ||
		st.Detail != "Stripe, PayKit · Launchpad Cohort · Stripe available balance 1234.56 USD" ||
		!reflect.DeepEqual(st.Meta, map[string]any{"configured": 2}) {
		t.Fatalf("got %+v", st)
	}
}

func TestStatusErrorWhenStripeFails(t *testing.T) {
	c := newWith(t, &fake{stripeFail: true}, "STRIPE_SECRET_KEY=sk_x\n")
	st := c.Status(context.Background())
	if st.State != connectors.StateError || st.Detail != "Stripe key set but verification failed: Invalid API Key provided: sk_****" {
		t.Fatalf("got %+v", st)
	}
	c.PointAt("http://127.0.0.1:1")
	if st := c.Status(context.Background()); st.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", st)
	}
}

// FounderOS v1 called the non-Stripe processors "configured" without checking
// any of them. The bridge keeps the wording but only says connected when at
// least one configured processor actually answers.
func TestStatusWithoutStripeVerifiesTheOthers(t *testing.T) {
	c := newWith(t, &fake{paykitCustomers: `{"data":{"customers":[{"id":1}]}}`, wiseFail: true}, "PAYKIT_LC_KEY=fb\nWISE_1_TOKEN=w\n")
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected || st.Detail != "PayKit · Launchpad Cohort, Wise configured" ||
		!reflect.DeepEqual(st.Meta, map[string]any{"configured": 2, "verified": 1}) {
		t.Fatalf("got %+v", st)
	}

	c = newWith(t, &fake{paykitFail: true, wiseFail: true}, "PAYKIT_LC_KEY=fb\nWISE_1_TOKEN=w\n")
	st = c.Status(context.Background())
	if st.State != connectors.StateError || !strings.HasPrefix(st.Detail, "PayKit · Launchpad Cohort, Wise configured but none verified: ") {
		t.Fatalf("got %+v", st)
	}
}

func TestFinancesIncomeRollup(t *testing.T) {
	f := &fake{
		stripeBalance: 100000,
		paykitCustomers: `{"data":{"customers":[
			{"id":1,"total_spent":"2,000.00","last_transaction_date":"2026-09-10 10:00:00","total_transactions":3},
			{"id":2,"total_spent":"500.00","last_transaction_date":"2026-09-12 10:00:00","total_transactions":1}]}}`,
	}
	c := newWith(t, f, "STRIPE_SECRET_KEY=sk_x\nSTRIPE_VANTAGE_KEY=sk_m\nPAYKIT_LC_KEY=fb\nWISE_1_TOKEN=w\n")
	h := paykit.NewMemoryHistory("paykit-vantage") // unseeded: the band stays a band
	got := c.FinancesIncome(context.Background(), h)

	if !got.Stripe.Live || got.Stripe.AvailableUSD != 1000 || got.Stripe.PendingUSD != 25 || len(got.Stripe.RecentCharges) != 1 {
		t.Errorf("stripe = %+v", got.Stripe)
	}
	f64 := func(v float64) *float64 { return &v }
	want := []IncomeAccount{
		{ID: "stripe", Processor: "Stripe", Label: "Stripe · Launchpad Cohort", Configured: true, Live: true, Income: f64(6000)},
		{ID: "stripe-vantage", Processor: "Stripe", Label: "Stripe · Vantage", Configured: true, Live: true, Income: f64(1500)},
		{ID: "paykit-lc", Processor: "PayKit", Label: "PayKit · Launchpad Cohort", Configured: true, Live: true, Income: f64(500), IncomeUpper: f64(2500), UnsplittableCustomers: 1},
	}
	if !reflect.DeepEqual(got.Accounts, want) {
		b, _ := json.MarshalIndent(got.Accounts, "", " ")
		t.Fatalf("accounts = %s", b)
	}
	if got.TotalUSD != 8000 || got.TotalUpperUSD != 10000 || !got.HasUnsplittable || got.LiveCount != 3 {
		t.Errorf("totals = %v / %v / %v / %d", got.TotalUSD, got.TotalUpperUSD, got.HasUnsplittable, got.LiveCount)
	}
	if len(got.WiseOutgoing) != 1 || got.WiseOutgoing[0].AmountCents != 25050 || got.WiseError != "" {
		t.Errorf("wise = %+v / %q", got.WiseOutgoing, got.WiseError)
	}
	if dates := h.CapturedDates(); len(dates) != 1 || dates[0] != "2026-09-29" {
		t.Errorf("the PayKit pull is kept as today's snapshot: %v", dates)
	}
}

// Unkeyed accounts stay pending (nil income), never a zero that reads as
// "earned nothing"; a failing source is pending too, and Wise says why.
func TestFinancesIncomeHonestPending(t *testing.T) {
	c := newWith(t, &fake{stripeFail: true, wiseFail: true}, "STRIPE_SECRET_KEY=sk_x\nWISE_1_TOKEN=w\n")
	got := c.FinancesIncome(context.Background(), nil)
	if got.Stripe.Live || got.Stripe.MtdUSD != nil {
		t.Errorf("a failing Stripe is not live: %+v", got.Stripe)
	}
	for _, a := range got.Accounts {
		if a.Live || a.Income != nil {
			t.Errorf("%s must be pending: %+v", a.ID, a)
		}
	}
	if !got.Accounts[0].Configured || got.Accounts[1].Configured {
		t.Errorf("configured flags: %+v", got.Accounts)
	}
	if got.TotalUSD != 0 || got.LiveCount != 0 {
		t.Errorf("totals: %+v", got)
	}
	if got.WiseOutgoing != nil || got.WiseError == "" {
		t.Errorf("wise failure must be reported, not shown as empty: %+v / %q", got.WiseOutgoing, got.WiseError)
	}
	out, _ := json.Marshal(got.Accounts[1])
	if !strings.Contains(string(out), `"income":null`) || !strings.Contains(string(out), `"incomeUpper":null`) {
		t.Errorf("JSON shape: %s", out)
	}
}

func TestIncomeAccountsBandCollapsesWhenNothingIsUnsplittable(t *testing.T) {
	accts := IncomeAccounts(false, nil, map[string]bool{"paykit-lc": true}, map[string]paykit.MonthIncomeUSD{
		"paykit-lc": {ExactUSD: 4500, UpperUSD: 4500},
	})
	fb := accts[2]
	if fb.Income == nil || *fb.Income != 4500 || fb.IncomeUpper != nil || fb.UnsplittableCustomers != 0 {
		t.Fatalf("got %+v", fb)
	}
	if accts[0].Configured || accts[0].Live || accts[0].Income != nil {
		t.Errorf("stripe: %+v", accts[0])
	}
	if TotalIncome(accts) != 4500 || TotalIncomeUpper(accts) != 4500 || HasUnsplittableIncome(accts) {
		t.Error("totals")
	}
}

func TestStripeFunnelWins(t *testing.T) {
	c := newWith(t, &fake{vantageFail: true}, "STRIPE_SECRET_KEY=sk_x\nSTRIPE_VANTAGE_KEY=sk_m\n")
	wins, ok := c.StripeFunnelWins(context.Background(), c.Now())
	if !ok || len(wins) != 1 || wins[0].Venture != "launchpad-cohort" || wins[0].AmountUSD != 6000 {
		t.Fatalf("one account answering is enough: %+v, %v", wins, ok)
	}

	c = newWith(t, &fake{stripeFail: true, vantageFail: true}, "STRIPE_SECRET_KEY=sk_x\nSTRIPE_VANTAGE_KEY=sk_m\n")
	if wins, ok := c.StripeFunnelWins(context.Background(), c.Now()); ok || wins != nil {
		t.Fatalf("none answering → not ok: %+v", wins)
	}

	c = newWith(t, &fake{}, "STRIPE_SECRET_KEY=sk_x\nFUNNEL_PROVIDER=seed\n")
	if _, ok := c.StripeFunnelWins(context.Background(), c.Now()); ok {
		t.Fatal("FUNNEL_PROVIDER=seed pins the seeded funnel")
	}
}
