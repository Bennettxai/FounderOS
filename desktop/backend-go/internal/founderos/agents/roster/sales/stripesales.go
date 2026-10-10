package sales

import (
	"context"
	"errors"
	"fmt"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

var metaStripeSales = agents.Meta{ID: "stripe-sales", Name: "Stripe", DepartmentID: "dept-finance",
	Description: "Stripe payment confirmation lane for sales workflows and account-level revenue checks."}

const stripeSalesNotConfigured = "Stripe sales checks not configured — set STRIPE_SECRET_KEY in .env.local"

// StripeReader is the Launchpad Cohort Stripe account (STRIPE_SECRET_KEY):
// whether it is keyed, and its balance plus five most recent charges.
type StripeReader interface {
	Configured() bool
	Snapshot(ctx context.Context) (stripe.Snapshot, error)
}

// StripeSales is stripe-sales (stripeSalesRun): the recent charges a sales
// workflow could confirm a payment against. Reads only (GET balance and
// charges through the guarded client); it never charges or refunds.
type StripeSales struct{ Stripe StripeReader }

func (s *StripeSales) Meta() agents.Meta { return metaStripeSales }

// Run fails with Stripe's own message when the read fails, as FounderOS v1's
// thrown stripeSnapshot did; the runtime records it as a failed run.
func (s *StripeSales) Run(ctx context.Context) (agents.Result, error) {
	if s.Stripe == nil || !s.Stripe.Configured() {
		return agents.Result{OK: false, Summary: stripeSalesNotConfigured}, nil
	}
	snap, err := s.Stripe.Snapshot(ctx)
	if errors.Is(err, stripe.ErrNotConfigured) {
		return agents.Result{OK: false, Summary: stripeSalesNotConfigured}, nil
	}
	if err != nil {
		return agents.Result{}, err
	}
	return agents.Result{
		OK:      true,
		Summary: fmt.Sprintf("Stripe sales payments: %d recent charges available for confirmation", len(snap.RecentCharges)),
		Data:    snap,
	}, nil
}
