package clients

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

// RosterClient matches FounderOS v1's RosterClient (lib/schemas.ts).
// AmountUSD nil is unknown, never zero.
type RosterClient struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Venture   string   `json:"venture"`
	Status    string   `json:"status"`
	AmountUSD *float64 `json:"amountUsd"`
	Source    string   `json:"source"` // stripe | funnel
}

// Roster is the live client roster (ClientRosterResult). Unlike the TS,
// which reported every Stripe failure as not_configured, an unreachable
// Stripe reads StateError with the reason in Detail.
type Roster struct {
	State   connectors.State `json:"state"`
	Detail  string           `json:"detail,omitempty"`
	Clients []RosterClient   `json:"clients"`
}

// RosterSource answers who paid. It never returns an error: failure is a
// state.
type RosterSource interface {
	Roster(ctx context.Context) Roster
}

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// RosterFromWins ports rosterFromStripeWins: settled charges → one row per
// paying customer (keyed by venture + email, else venture + name, else the
// charge), payments summed, sorted by name.
func RosterFromWins(wins []stripe.Win) []RosterClient {
	var order []string
	groups := map[string][]stripe.Win{}
	for _, w := range wins {
		key := w.ID
		switch {
		case w.Email != nil:
			key = w.Venture + ":" + norm(*w.Email)
		case w.Name != nil:
			key = w.Venture + ":name:" + norm(*w.Name)
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], w)
	}
	rows := make([]RosterClient, 0, len(order))
	for _, key := range order {
		g := groups[key]
		first := g[0]
		name := "Stripe customer"
		if first.Name != nil {
			name = *first.Name
		} else if first.Email != nil {
			name = *first.Email
		}
		sum := 0.0
		for _, w := range g {
			sum += w.AmountUSD
		}
		rows = append(rows, RosterClient{
			ID: "stripe-" + first.ID, Name: name, Venture: first.Venture,
			Status: "won", AmountUSD: &sum, Source: "stripe",
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name) })
	return rows
}

// StripeAccount is the slice of *stripe.Connector the roster reads.
type StripeAccount interface {
	Account() stripe.Account
	Configured() bool
	FunnelWins(ctx context.Context, now time.Time) ([]stripe.Win, error)
}

// StripeRoster is liveClientRoster over both Stripe accounts: the last 90
// days of settled charges (stripeFunnelWins), each account degrading on its
// own, a partial failure named in Detail.
type StripeRoster struct {
	Accounts []StripeAccount
	// Pinned reports FUNNEL_PROVIDER pinning the seeded funnel.
	Pinned func() bool
	Now    func() time.Time
}

// NewStripeRoster reads both accounts through the ported Stripe connector.
func NewStripeRoster(res connectors.Resolver) *StripeRoster {
	return &StripeRoster{
		Accounts: []StripeAccount{stripe.New(res), stripe.NewAccount(res, stripe.Vantage)},
		Pinned: func() bool {
			v := res.Resolve("FUNNEL_PROVIDER")
			return v != "" && v != "live" && v != "attio"
		},
		Now: time.Now,
	}
}

func (s *StripeRoster) Roster(ctx context.Context) Roster {
	if s.Pinned != nil && s.Pinned() {
		return Roster{State: connectors.StateNotConfigured, Detail: "FUNNEL_PROVIDER pins the seeded funnel", Clients: []RosterClient{}}
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	var keys, failures []string
	wins := []stripe.Win{}
	reachable := 0
	for _, a := range s.Accounts {
		if !a.Configured() {
			keys = append(keys, a.Account().EnvKey)
			continue
		}
		w, err := a.FunnelWins(ctx, now())
		if err != nil {
			failures = append(failures, a.Account().Name+": "+err.Error())
			continue
		}
		wins = append(wins, w...)
		reachable++
	}
	if reachable == 0 && len(failures) == 0 {
		return Roster{State: connectors.StateNotConfigured, Detail: "no Stripe account keyed (set " + strings.Join(keys, " or ") + ")", Clients: []RosterClient{}}
	}
	if reachable == 0 {
		return Roster{State: connectors.StateError, Detail: strings.Join(failures, "; "), Clients: []RosterClient{}}
	}
	r := Roster{State: connectors.StateConnected, Clients: RosterFromWins(wins)}
	if len(failures) > 0 {
		r.Detail = "partial: " + strings.Join(failures, "; ")
	}
	return r
}
