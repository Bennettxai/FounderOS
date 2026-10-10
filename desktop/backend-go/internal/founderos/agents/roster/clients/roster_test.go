package clients

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

func sp(s string) *string { return &s }

func win(id, venture string, email, name *string, usd float64) stripe.Win {
	return stripe.Win{ID: id, Venture: venture, Email: email, Name: name, AmountUSD: usd, At: "2026-09-01"}
}

// Mirrors tests/client-roster.test.ts: one row per paying customer, their
// payments summed, marked won, sourced from Stripe.
func TestRosterFromWinsOneRowPerPayingCustomer(t *testing.T) {
	rows := RosterFromWins([]stripe.Win{
		win("ch_1", "vantage", sp("pat@acme.com"), sp("Pat Lee"), 2000),
		win("ch_2", "vantage", sp(" PAT@acme.com "), sp("Pat Lee"), 500),
		win("ch_3", "vantage", sp("x@y.com"), sp("Xan"), 2000),
	})
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Name != "Pat Lee" || *rows[0].AmountUSD != 2500 || rows[0].ID != "stripe-ch_1" {
		t.Fatalf("pat = %+v", rows[0])
	}
	if rows[1].Name != "Xan" || *rows[1].AmountUSD != 2000 {
		t.Fatalf("xan = %+v", rows[1])
	}
	for _, r := range rows {
		if r.Status != "won" || r.Source != "stripe" {
			t.Fatalf("row not a paid client: %+v", r)
		}
	}
}

func TestRosterFromWinsGroupsByNameThenChargeAndNamesAnonymousHonestly(t *testing.T) {
	rows := RosterFromWins([]stripe.Win{
		win("ch_a", "launchpad-cohort", nil, sp("Zed"), 100),
		win("ch_b", "launchpad-cohort", nil, sp("zed"), 50),
		win("ch_c", "vantage", nil, sp("Zed"), 10), // another venture is another client
		win("ch_d", "vantage", nil, nil, 7),
		win("ch_e", "vantage", sp("only@mail.com"), nil, 3),
	})
	names := []string{}
	for _, r := range rows {
		names = append(names, r.Name)
	}
	if got := strings.Join(names, ","); got != "only@mail.com,Stripe customer,Zed,Zed" {
		t.Fatalf("names = %s", got)
	}
	var aa *RosterClient
	for i := range rows {
		if rows[i].Venture == "launchpad-cohort" {
			aa = &rows[i]
		}
	}
	if aa == nil || *aa.AmountUSD != 150 {
		t.Fatalf("LC Zed = %+v", aa)
	}
}

func TestRosterFromWinsEmptyIsEmptyNotNil(t *testing.T) {
	if rows := RosterFromWins(nil); rows == nil || len(rows) != 0 {
		t.Fatalf("rows = %#v", rows)
	}
}

type fakeAccount struct {
	acct       stripe.Account
	configured bool
	wins       []stripe.Win
	err        error
	gotNow     time.Time
}

func (f *fakeAccount) Account() stripe.Account { return f.acct }
func (f *fakeAccount) Configured() bool        { return f.configured }
func (f *fakeAccount) FunnelWins(_ context.Context, now time.Time) ([]stripe.Win, error) {
	f.gotNow = now
	return f.wins, f.err
}

var clock = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func source(pinned bool, accts ...*fakeAccount) *StripeRoster {
	s := &StripeRoster{Pinned: func() bool { return pinned }, Now: func() time.Time { return clock }}
	for _, a := range accts {
		s.Accounts = append(s.Accounts, a)
	}
	return s
}

func TestStripeRosterUnkeyedIsNotConfigured(t *testing.T) {
	r := source(false,
		&fakeAccount{acct: stripe.LaunchpadCohort},
		&fakeAccount{acct: stripe.Vantage},
	).Roster(context.Background())
	if r.State != connectors.StateNotConfigured || len(r.Clients) != 0 {
		t.Fatalf("roster = %+v", r)
	}
	if !strings.Contains(r.Detail, "STRIPE_SECRET_KEY") || !strings.Contains(r.Detail, "STRIPE_VANTAGE_KEY") {
		t.Fatalf("detail = %q", r.Detail)
	}
}

func TestStripeRosterPinnedSeedIsNotConfigured(t *testing.T) {
	a := &fakeAccount{acct: stripe.LaunchpadCohort, configured: true, wins: []stripe.Win{win("ch_1", "launchpad-cohort", sp("a@b.c"), nil, 1)}}
	r := source(true, a).Roster(context.Background())
	if r.State != connectors.StateNotConfigured || !strings.Contains(r.Detail, "FUNNEL_PROVIDER") {
		t.Fatalf("roster = %+v", r)
	}
	if !a.gotNow.IsZero() {
		t.Fatal("a pinned funnel must not read Stripe")
	}
}

// An unreachable Stripe is an error, never an empty roster (the TS
// swallowed it as not_configured; the bridge says what happened).
func TestStripeRosterEveryAccountFailingIsError(t *testing.T) {
	r := source(false,
		&fakeAccount{acct: stripe.LaunchpadCohort, configured: true, err: errors.New("Invalid API Key")},
		&fakeAccount{acct: stripe.Vantage},
	).Roster(context.Background())
	if r.State != connectors.StateError || len(r.Clients) != 0 {
		t.Fatalf("roster = %+v", r)
	}
	if !strings.Contains(r.Detail, "Invalid API Key") || !strings.Contains(r.Detail, "Stripe") {
		t.Fatalf("detail = %q", r.Detail)
	}
}

func TestStripeRosterMergesAccountsAndNamesAPartialFailure(t *testing.T) {
	aa := &fakeAccount{acct: stripe.LaunchpadCohort, configured: true, wins: []stripe.Win{win("ch_1", "launchpad-cohort", sp("a@x.com"), sp("Ann"), 10)}}
	me := &fakeAccount{acct: stripe.Vantage, configured: true, err: errors.New("timeout")}
	r := source(false, aa, me).Roster(context.Background())
	if r.State != connectors.StateConnected || len(r.Clients) != 1 || r.Clients[0].Name != "Ann" {
		t.Fatalf("roster = %+v", r)
	}
	if !strings.Contains(r.Detail, "Stripe · Vantage") || !strings.Contains(r.Detail, "timeout") {
		t.Fatalf("a partial failure must be named: %q", r.Detail)
	}
	if !aa.gotNow.Equal(clock) {
		t.Fatalf("clock not injected: %v", aa.gotNow)
	}
}

func TestStripeRosterConnectedWithNoPayersIsConnectedAndEmpty(t *testing.T) {
	r := source(false, &fakeAccount{acct: stripe.LaunchpadCohort, configured: true}).Roster(context.Background())
	if r.State != connectors.StateConnected || r.Clients == nil || len(r.Clients) != 0 {
		t.Fatalf("roster = %+v", r)
	}
}
