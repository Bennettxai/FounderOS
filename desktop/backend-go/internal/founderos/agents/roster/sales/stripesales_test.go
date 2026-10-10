package sales

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

type fakeStripe struct {
	configured bool
	snap       stripe.Snapshot
	err        error
	calls      int
}

func (f *fakeStripe) Configured() bool { return f.configured }

func (f *fakeStripe) Snapshot(context.Context) (stripe.Snapshot, error) {
	f.calls++
	return f.snap, f.err
}

// stripeSalesRun: unkeyed says how to key it and never calls Stripe; keyed
// reports the recent charges available for confirmation.
func TestStripeSalesUnconfiguredNeverCallsStripe(t *testing.T) {
	f := &fakeStripe{}
	res := run(t, &StripeSales{Stripe: f})
	if res.OK || res.Summary != "Stripe sales checks not configured — set STRIPE_SECRET_KEY in .env.local" {
		t.Fatalf("%+v", res)
	}
	if f.calls != 0 {
		t.Fatal("an unkeyed run must not reach Stripe")
	}
	if res := run(t, &StripeSales{}); res.OK {
		t.Fatalf("unwired = %+v", res)
	}
}

func TestStripeSalesReportsRecentCharges(t *testing.T) {
	snap := stripe.Snapshot{Available: []stripe.Money{{Amount: 1200, Currency: "usd"}}, Pending: []stripe.Money{},
		RecentCharges: []stripe.RecentCharge{{Amount: 50000, Currency: "usd", Description: "ch_1"}, {Amount: 9700, Currency: "usd", Description: "Coaching"}}}
	res := run(t, &StripeSales{Stripe: &fakeStripe{configured: true, snap: snap}})
	if !res.OK || res.Summary != "Stripe sales payments: 2 recent charges available for confirmation" {
		t.Fatalf("%+v", res)
	}
	if d, ok := res.Data.(stripe.Snapshot); !ok || len(d.RecentCharges) != 2 {
		t.Fatalf("data = %#v", res.Data)
	}
	res = run(t, &StripeSales{Stripe: &fakeStripe{configured: true, snap: stripe.Snapshot{RecentCharges: []stripe.RecentCharge{}}}})
	if !res.OK || res.Summary != "Stripe sales payments: 0 recent charges available for confirmation" {
		t.Fatalf("a keyed account with no charges is a real zero: %+v", res)
	}
}

// FounderOS v1 let stripeSnapshot throw, so the run failed with Stripe's own
// message; the bridge returns that error for the runtime to record the same way.
func TestStripeSalesFailsWithStripesMessage(t *testing.T) {
	a := &StripeSales{Stripe: &fakeStripe{configured: true, err: &stripe.APIError{StatusCode: 401, Message: "Invalid API Key provided: sk_live_****abcd"}}}
	res, err := a.Run(context.Background())
	if err == nil || err.Error() != "Invalid API Key provided: sk_live_****abcd" || res.OK {
		t.Fatalf("res %+v err %v", res, err)
	}
	// A key that vanishes between the check and the read is still "not configured".
	a = &StripeSales{Stripe: &fakeStripe{configured: true, err: errors.Join(stripe.ErrNotConfigured)}}
	if res := run(t, a); res.OK || res.Summary != "Stripe sales checks not configured — set STRIPE_SECRET_KEY in .env.local" {
		t.Fatalf("%+v", res)
	}
}
