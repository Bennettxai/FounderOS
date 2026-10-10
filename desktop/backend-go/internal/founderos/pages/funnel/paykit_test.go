package funnel

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

// Ported from FounderOS v1 tests/funnel-paykit.test.ts.

var fbNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

func fbRaw(customers ...map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{"data": map[string]any{"customers": customers}})
	return b
}

func fbCust(over map[string]any) map[string]any {
	c := map[string]any{
		"id": 452730, "name": "Repeat Buyer", "email": "Repeat@Example.com ", "phone": "+1 555 010 2000",
		"total_transactions": 1, "total_spent": "1,500.00", "last_transaction_date": "2026-09-10T10:00:00-05:00",
	}
	for k, v := range over {
		c[k] = v
	}
	return c
}

func TestMapPaykitCustomersTurnsARecentBuyerIntoAWin(t *testing.T) {
	got := MapPaykitCustomers(fbRaw(fbCust(nil)), "launchpad-cohort", fbNow)
	want := []stripe.Win{{ID: "fb-452730", Source: "paykit", Venture: "launchpad-cohort", Email: sp("Repeat@Example.com"), Name: sp("Repeat Buyer"),
		Phone: sp("+1 555 010 2000"), AmountUSD: 1500, Payments: 1, At: "2026-09-10"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wins = %+v", got)
	}
}

func TestMapPaykitCustomersDropsOldZeroAndUndatedBuyers(t *testing.T) {
	got := MapPaykitCustomers(fbRaw(
		fbCust(map[string]any{"id": 1, "last_transaction_date": "2026-05-01T10:00:00-05:00"}),
		fbCust(map[string]any{"id": 2, "total_spent": "0.00"}),
		fbCust(map[string]any{"id": 3, "last_transaction_date": nil}),
		fbCust(map[string]any{"id": nil}),
	), "launchpad-cohort", fbNow)
	if len(got) != 0 {
		t.Fatalf("wins = %+v", got)
	}
	if got := MapPaykitCustomers([]byte(`{"nope":true}`), "vantage", fbNow); len(got) != 0 {
		t.Fatalf("malformed = %+v", got)
	}
}

func TestPaykitWinConvertsTheMatchingJourney(t *testing.T) {
	j := Journey{ID: "tf-1", Name: "Repeat Buyer", Venture: "launchpad-cohort", Status: "nurtured", Relationship: "warm", Likelihood: 60,
		Email: sp("repeat@example.com"), Person: sp("Repeat Buyer"), CreatedAt: "2026-09-01", Touches: []Touch{}}
	wins := MapPaykitCustomers(fbRaw(fbCust(map[string]any{"total_transactions": 3})), "launchpad-cohort", fbNow)
	got := MergeStripeWins([]Journey{j}, wins)[0]
	last := got.Touches[len(got.Touches)-1]
	if got.Status != "converted" || *got.AmountUSD != 1500 || last.Source != "paykit" || last.Channel != "checkout" || last.ID != "tf-1-paykit-fb-452730" {
		t.Fatalf("merged = %+v", got)
	}
	if last.Label != "PayKit $1,500 lifetime · 3 payments" {
		t.Fatalf("label = %q", last.Label)
	}
}

func TestUnmatchedPaykitBuyerStandsAloneWithPhone(t *testing.T) {
	got := MergeStripeWins(nil, MapPaykitCustomers(fbRaw(fbCust(nil)), "launchpad-cohort", fbNow))
	if len(got) != 1 || got[0].ID != "paykit-fb-452730" || got[0].Status != "converted" || got[0].Phone == nil || *got[0].Phone != "+1 555 010 2000" || got[0].Touches[0].Source != "paykit" {
		t.Fatalf("standalone = %+v", got)
	}
	if got[0].Touches[0].Label != "PayKit payment $1,500" {
		t.Fatalf("label = %q", got[0].Touches[0].Label)
	}
}

func TestAClientWhoPaidOnBothProcessorsIsOneJourney(t *testing.T) {
	fb := MapPaykitCustomers(fbRaw(fbCust(nil)), "launchpad-cohort", fbNow)
	st := []stripe.Win{{ID: "ch_1", Venture: "launchpad-cohort", Email: sp("repeat@example.com"), AmountUSD: 500, At: "2026-09-01"}}
	out := MergeStripeWins(nil, append(st, fb...))
	if len(out) != 1 || *out[0].AmountUSD != 2000 || out[0].Touches[0].Source != "stripe" || out[0].Touches[1].Source != "paykit" || out[0].Name != "Repeat Buyer" {
		t.Fatalf("grouped = %+v", out)
	}
	// Stripe ids predate PayKit and stay unchanged.
	if out[0].ID != "stripe-ch_1" {
		t.Fatalf("id = %q", out[0].ID)
	}
}

func TestComposePaykitAloneMakesTheFunnelLive(t *testing.T) {
	c := Compose(context.Background(), fbNow, "", lanes(func(l *Lanes) {
		l.Paykit = func(context.Context, time.Time) ([]stripe.Win, error) {
			return MapPaykitCustomers(fbRaw(fbCust(nil)), "launchpad-cohort", fbNow), nil
		}
	}))
	if !c.IsLive || len(c.PaykitWins) != 1 || len(c.Journeys) != 1 || c.Journeys[0].Touches[0].Source != "paykit" || SourceLabel(c) != "paykit" {
		t.Fatalf("composition = %+v label %q", c, SourceLabel(c))
	}
	if !strings.Contains(c.Journeys[0].Touches[0].Label, "PayKit") {
		t.Fatalf("touch = %+v", c.Journeys[0].Touches[0])
	}
}
