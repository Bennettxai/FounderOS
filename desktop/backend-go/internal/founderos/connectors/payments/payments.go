// Package payments is the bridge's port of FounderOS v1's payment processor
// registry (lib/connectors/payments.ts) and the income rollup the /finances
// page builds on it (lib/finances.ts + app/finances/page.tsx). It aggregates
// one connector per provider:
//
//	stripe    Stripe · Launchpad Cohort (STRIPE_SECRET_KEY), Stripe · Vantage (STRIPE_VANTAGE_KEY)
//	paykit  PayKit · Launchpad Cohort (PAYKIT_LC_KEY), PayKit · Vantage (PAYKIT_VANTAGE_KEY)
//	paypal    PayPal (PAYPAL_CLIENT_ID + PAYPAL_CLIENT_SECRET)
//	wise      Wise (WISE_1_TOKEN)
//
// The Connections board shows one "payments" row, as FounderOS v1 does. Every
// read here is read-only; no provider exposes a money-moving path.
package payments

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paypal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/wise"
)

var Meta = connectors.Meta{ID: "payments", Name: "Payment Processors", Kind: connectors.KindPayments}

// ProcessorInfo matches FounderOS v1's ProcessorInfo. Ids match the finances
// page's income accounts so each card lights by config.
type ProcessorInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
}

// ConfiguredProcessors is the registry, in FounderOS v1's order, with
// configured derived from the credentials lookup returns.
func ConfiguredProcessors(lookup func(string) string) []ProcessorInfo {
	set := func(k string) bool { return lookup(k) != "" }
	return []ProcessorInfo{
		{stripe.LaunchpadCohort.ID, stripe.LaunchpadCohort.Name, set(stripe.LaunchpadCohort.EnvKey)},
		{stripe.Vantage.ID, stripe.Vantage.Name, set(stripe.Vantage.EnvKey)},
		{paypal.Meta.ID, paypal.Meta.Name, set(paypal.EnvClientID) && set(paypal.EnvClientSecret)},
		{paykit.Vantage.ID, paykit.Vantage.Name, set(paykit.Vantage.EnvKey)},
		{paykit.LaunchpadCohort.ID, paykit.LaunchpadCohort.Name, set(paykit.LaunchpadCohort.EnvKey)},
		{wise.Meta.ID, wise.Meta.Name, set(wise.EnvToken)},
	}
}

type Connector struct {
	res           connectors.Resolver
	Stripe        *stripe.Connector
	StripeVantage *stripe.Connector
	PaykitAA      *paykit.Connector
	PaykitVantage *paykit.Connector
	PayPal        *paypal.Connector
	Wise          *wise.Connector
	// Now dates month-to-date reads and PayKit snapshots.
	Now func() time.Time
}

func New(res connectors.Resolver) *Connector {
	c := &Connector{
		res:           res,
		Stripe:        stripe.New(res),
		StripeVantage: stripe.NewAccount(res, stripe.Vantage),
		PaykitAA:      paykit.New(res),
		PaykitVantage: paykit.NewAccount(res, paykit.Vantage),
		PayPal:        paypal.New(res),
		Wise:          wise.New(res),
		Now:           time.Now,
	}
	now := func() time.Time { return c.Now() }
	c.PaykitAA.Now, c.PaykitVantage.Now = now, now
	return c
}

// PointAt sends every provider to one base URL (tests and fixture replays).
func (c *Connector) PointAt(baseURL string) {
	c.Stripe.BaseURL, c.StripeVantage.BaseURL = baseURL, baseURL
	c.PaykitAA.BaseURL, c.PaykitVantage.BaseURL = baseURL, baseURL
	c.PayPal.BaseURL, c.Wise.BaseURL = baseURL, baseURL
}

func (c *Connector) Processors() []ProcessorInfo {
	return ConfiguredProcessors(func(k string) string { return c.res.Resolve(k) })
}

func (c *Connector) statusOf(id string) func(context.Context) connectors.Status {
	switch id {
	case stripe.LaunchpadCohort.ID:
		return c.Stripe.Status
	case stripe.Vantage.ID:
		return c.StripeVantage.Status
	case paypal.Meta.ID:
		return c.PayPal.Status
	case paykit.Vantage.ID:
		return c.PaykitVantage.Status
	case paykit.LaunchpadCohort.ID:
		return c.PaykitAA.Status
	case wise.Meta.ID:
		return c.Wise.Status
	}
	return nil
}

// Status ports paymentsStatus. With Stripe keyed it is FounderOS v1's check
// exactly: a live snapshot read, connected with the available balance, or
// error. Without Stripe, FounderOS v1 reported the other configured processors
// as connected unchecked; the bridge keeps the wording but verifies each one
// and says connected only when at least one answers. The demo-gate branch
// (GATED) is not ported: the bridge never reports a canned status.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	procs := c.Processors()
	var configured []ProcessorInfo
	for _, p := range procs {
		if p.Configured {
			configured = append(configured, p)
		}
	}
	if len(configured) == 0 {
		st.State = connectors.StateNotConfigured
		st.Detail = "None of 6 processors configured. Start with STRIPE_SECRET_KEY in ~/.founderos/.env or under API keys."
		st.Meta = map[string]any{"configured": 0, "known": len(procs)}
		return st
	}
	names := make([]string, len(configured))
	for i, p := range configured {
		names[i] = p.Name
	}
	joined := strings.Join(names, ", ")

	stripeKeyed := false
	for _, p := range configured {
		stripeKeyed = stripeKeyed || p.ID == stripe.LaunchpadCohort.ID
	}
	if stripeKeyed {
		st.Meta = map[string]any{"configured": len(configured)}
		snap, err := c.Stripe.Snapshot(ctx)
		if err != nil {
			st.State = connectors.StateError
			st.Detail = "Stripe key set but verification failed: " + err.Error()
			return st
		}
		st.State = connectors.StateConnected
		st.Detail = joined + " · Stripe available balance " + stripe.AvailableLabel(snap)
		return st
	}

	results := make([]connectors.Status, len(configured))
	var wg sync.WaitGroup
	for i, p := range configured {
		wg.Add(1)
		go func(i int, check func(context.Context) connectors.Status) {
			defer wg.Done()
			results[i] = check(ctx)
		}(i, c.statusOf(p.ID))
	}
	wg.Wait()
	verified, firstErr := 0, ""
	for _, r := range results {
		if r.State == connectors.StateConnected {
			verified++
		} else if firstErr == "" {
			firstErr = r.Detail
		}
	}
	st.Meta = map[string]any{"configured": len(configured), "verified": verified}
	if verified == 0 {
		st.State = connectors.StateError
		st.Detail = joined + " configured but none verified: " + firstErr
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = joined + " configured"
	return st
}

// ---- reads (the exported functions of payments.ts) ---------------------

// StripeSnapshot is stripeSnapshot: the Launchpad Cohort balance and five
// most recent charges (home page ledger, live metrics, the Stripe agents).
func (c *Connector) StripeSnapshot(ctx context.Context) (stripe.Snapshot, error) {
	return c.Stripe.Snapshot(ctx)
}

// MonthToDateIncome is monthToDateIncome (Launchpad Cohort Stripe).
func (c *Connector) MonthToDateIncome(ctx context.Context) (*stripe.IncomeMTD, error) {
	return c.Stripe.MonthToDateIncome(ctx, c.Now())
}

// WiseOutgoing is wiseOutgoing: (nil, nil) when unkeyed, so the section hides.
func (c *Connector) WiseOutgoing(ctx context.Context) ([]wise.Transfer, error) {
	return c.Wise.Outgoing(ctx)
}

// PaykitMonthToDateIncome is paykitMonthToDateIncome for the LC account
// (the only PayKit card the finances page shows).
func (c *Connector) PaykitMonthToDateIncome(ctx context.Context, month string, history paykit.HistoryPort) (*paykit.MonthIncomeUSD, error) {
	return c.PaykitAA.MonthToDateIncome(ctx, month, history)
}

// StripeFunnelWins is stripeFunnelWins: the last 90 days of settled charges
// from both accounts. Each account degrades on its own; ok is false only when
// no account answered, or when FUNNEL_PROVIDER pins the seeded funnel.
func (c *Connector) StripeFunnelWins(ctx context.Context, now time.Time) (wins []stripe.Win, ok bool) {
	if v := c.res.Resolve("FUNNEL_PROVIDER"); v != "" && v != "live" && v != "attio" {
		return nil, false
	}
	reachable := 0
	out := []stripe.Win{}
	for _, acct := range []*stripe.Connector{c.Stripe, c.StripeVantage} {
		if !acct.Configured() {
			continue
		}
		w, err := acct.FunnelWins(ctx, now)
		if err != nil {
			continue
		}
		out = append(out, w...)
		reachable++
	}
	if reachable == 0 {
		return nil, false
	}
	return out, true
}

// ---- the finances income rollup (lib/finances.ts) ----------------------

// IncomeAccount matches FounderOS v1's IncomeAccount. Income nil is pending,
// never a zero that reads as "earned nothing". Income is always the proven
// floor; IncomeUpper is set only when a source could merely bound the month.
type IncomeAccount struct {
	ID                    string   `json:"id"`
	Processor             string   `json:"processor"`
	Label                 string   `json:"label"`
	Configured            bool     `json:"configured"`
	Live                  bool     `json:"live"`
	Income                *float64 `json:"income"`
	IncomeUpper           *float64 `json:"incomeUpper"`
	UnsplittableCustomers int      `json:"unsplittableCustomers"`
}

// IncomeAccounts is incomeAccounts: the three accounts with real API
// connections. An exact live figure is passed as a band whose floor and
// ceiling are equal, which renders identically to FounderOS v1's plain number.
func IncomeAccounts(stripeConnected bool, stripeMtdUSD *float64, configured map[string]bool, live map[string]paykit.MonthIncomeUSD) []IncomeAccount {
	account := func(id, processor, label string) IncomeAccount {
		a := IncomeAccount{ID: id, Processor: processor, Label: label, Configured: configured[id]}
		if band, ok := live[id]; ok {
			exact := band.ExactUSD
			a.Live, a.Income, a.UnsplittableCustomers = true, &exact, band.UnsplittableCustomers
			if band.UnsplittableCustomers > 0 {
				upper := band.UpperUSD
				a.IncomeUpper = &upper
			}
		}
		return a
	}
	aa := IncomeAccount{ID: "stripe", Processor: "Stripe", Label: "Stripe · Launchpad Cohort", Live: stripeConnected}
	if v, ok := configured["stripe"]; ok {
		aa.Configured = v
	} else {
		aa.Configured = stripeConnected
	}
	if stripeConnected && stripeMtdUSD != nil {
		v := *stripeMtdUSD
		aa.Income = &v
	}
	return []IncomeAccount{
		aa,
		account("stripe-vantage", "Stripe", "Stripe · Vantage"),
		account("paykit-lc", "PayKit", "PayKit · Launchpad Cohort"),
	}
}

// TotalIncome is the proven floor across accounts; pending counts as zero.
func TotalIncome(accts []IncomeAccount) float64 {
	var sum float64
	for _, a := range accts {
		if a.Income != nil {
			sum += *a.Income
		}
	}
	return sum
}

// TotalIncomeUpper is the ceiling: every bounded account at its upper end.
func TotalIncomeUpper(accts []IncomeAccount) float64 {
	var sum float64
	for _, a := range accts {
		switch {
		case a.IncomeUpper != nil:
			sum += *a.IncomeUpper
		case a.Income != nil:
			sum += *a.Income
		}
	}
	return sum
}

func HasUnsplittableIncome(accts []IncomeAccount) bool {
	for _, a := range accts {
		if a.UnsplittableCustomers > 0 {
			return true
		}
	}
	return false
}

// StripeOverview is the finances page's Stripe block. Live only when the API
// actually answered; a present-but-invalid key stays honest pending.
type StripeOverview struct {
	Keyed         bool                  `json:"keyed"`
	Live          bool                  `json:"live"`
	MtdUSD        *float64              `json:"mtdUsd"`
	AvailableUSD  float64               `json:"availableUsd"`
	PendingUSD    float64               `json:"pendingUsd"`
	RecentCharges []stripe.RecentCharge `json:"recentCharges"`
}

// FinancesIncome is everything the /finances page reads from the processors.
type FinancesIncome struct {
	Stripe          StripeOverview  `json:"stripe"`
	Processors      []ProcessorInfo `json:"processors"`
	Accounts        []IncomeAccount `json:"accounts"`
	TotalUSD        float64         `json:"totalUsd"`
	TotalUpperUSD   float64         `json:"totalUpperUsd"`
	HasUnsplittable bool            `json:"hasUnsplittable"`
	LiveCount       int             `json:"liveCount"`
	// WiseOutgoing nil hides the section (no token). WiseError is set when a
	// token is set but the read failed; FounderOS v1 hid that case silently.
	WiseOutgoing []wise.Transfer `json:"wiseOutgoing"`
	WiseError    string          `json:"wiseError,omitempty"`
}

// FinancesIncome reproduces app/finances/page.tsx's processor reads, all in
// parallel. history keeps the PayKit pull as today's snapshot (nil to skip).
func (c *Connector) FinancesIncome(ctx context.Context, history paykit.HistoryPort) FinancesIncome {
	procs := c.Processors()
	configured := map[string]bool{}
	for _, p := range procs {
		configured[p.ID] = p.Configured
	}
	out := FinancesIncome{Processors: procs, Stripe: StripeOverview{Keyed: configured["stripe"], RecentCharges: []stripe.RecentCharge{}}}

	var (
		wg      sync.WaitGroup
		mtd     *stripe.IncomeMTD
		snap    stripe.Snapshot
		snapErr error = stripe.ErrNotConfigured
		fbAA    *paykit.MonthIncomeUSD
		mer     *stripe.IncomeMTD
		wiseOut []wise.Transfer
		wiseErr error
	)
	run := func(fn func()) { wg.Add(1); go func() { defer wg.Done(); fn() }() }
	if out.Stripe.Keyed {
		run(func() { mtd, _ = c.Stripe.MonthToDateIncome(ctx, c.Now()) })
		run(func() { snap, snapErr = c.Stripe.Snapshot(ctx) })
	}
	run(func() { fbAA, _ = c.PaykitAA.MonthToDateIncome(ctx, "", history) })
	run(func() { mer, _ = c.StripeVantage.MonthToDateIncome(ctx, c.Now()) })
	run(func() { wiseOut, wiseErr = c.Wise.Outgoing(ctx) })
	wg.Wait()

	if out.Stripe.Keyed && snapErr == nil {
		out.Stripe.Live = true
		if len(snap.Available) > 0 {
			out.Stripe.AvailableUSD = float64(snap.Available[0].Amount) / 100
		}
		if len(snap.Pending) > 0 {
			out.Stripe.PendingUSD = float64(snap.Pending[0].Amount) / 100
		}
		out.Stripe.RecentCharges = snap.RecentCharges
	}
	if mtd != nil {
		v := float64(mtd.AmountCents) / 100
		out.Stripe.MtdUSD = &v
	}

	live := map[string]paykit.MonthIncomeUSD{}
	if fbAA != nil {
		live["paykit-lc"] = *fbAA
	}
	if mer != nil {
		v := float64(mer.AmountCents) / 100
		live["stripe-vantage"] = paykit.MonthIncomeUSD{ExactUSD: v, UpperUSD: v}
	}
	out.Accounts = IncomeAccounts(out.Stripe.Live, out.Stripe.MtdUSD, configured, live)
	out.TotalUSD = TotalIncome(out.Accounts)
	out.TotalUpperUSD = TotalIncomeUpper(out.Accounts)
	out.HasUnsplittable = HasUnsplittableIncome(out.Accounts)
	for _, a := range out.Accounts {
		if a.Live {
			out.LiveCount++
		}
	}
	if wiseErr != nil {
		out.WiseError = wiseErr.Error()
	} else {
		out.WiseOutgoing = wiseOut
	}
	return out
}
