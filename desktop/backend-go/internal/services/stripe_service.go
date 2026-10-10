package services

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/stripe/stripe-go/v82"
	bpsession "github.com/stripe/stripe-go/v82/billingportal/session"
	chsession "github.com/stripe/stripe-go/v82/checkout/session"
	stripesubscription "github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"
)

// ErrStripeNotConfigured is returned when STRIPE_SECRET_KEY is not set.
var ErrStripeNotConfigured = errors.New("stripe: not configured")

// StripeSubscription is a simplified view of a Stripe subscription.
type StripeSubscription struct {
	CustomerID        string `json:"customer_id"`
	SubscriptionID    string `json:"subscription_id"`
	Plan              string `json:"plan"`   // "pro" | "growth" | "business"
	Status            string `json:"status"` // "active" | "past_due" | "canceled" | "trialing"
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}

// StripeService wraps Stripe API calls.
// When secretKey is empty every method returns ErrStripeNotConfigured —
// callers should return an explicit billing-unavailable response.
type StripeService struct {
	secretKey       string
	webhookSecret   string
	priceProID      string
	priceGrowthID   string
	priceBusinessID string
}

// NewStripeService creates a StripeService.
// stripe.Key is set once here so per-method assignments are not needed.
func NewStripeService(secretKey, webhookSecret, priceProID, priceGrowthID, priceBusinessID string) *StripeService {
	if secretKey != "" {
		stripe.Key = secretKey
	}
	return &StripeService{
		secretKey:       secretKey,
		webhookSecret:   webhookSecret,
		priceProID:      priceProID,
		priceGrowthID:   priceGrowthID,
		priceBusinessID: priceBusinessID,
	}
}

// Enabled reports whether Stripe credentials are configured.
func (s *StripeService) Enabled() bool {
	return s.secretKey != ""
}

// priceIDForPlan maps a plan slug to a Stripe Price ID.
func (s *StripeService) priceIDForPlan(plan string) (string, error) {
	switch plan {
	case "pro":
		if s.priceProID == "" {
			return "", fmt.Errorf("stripe: STRIPE_PRICE_PRO is not set")
		}
		return s.priceProID, nil
	case "growth":
		if s.priceGrowthID == "" {
			return "", fmt.Errorf("stripe: STRIPE_PRICE_GROWTH is not set")
		}
		return s.priceGrowthID, nil
	case "business":
		if s.priceBusinessID == "" {
			return "", fmt.Errorf("stripe: STRIPE_PRICE_BUSINESS is not set")
		}
		return s.priceBusinessID, nil
	default:
		return "", fmt.Errorf("stripe: unknown plan %q", plan)
	}
}

// CreateCheckoutSession creates a Stripe Checkout Session and returns the
// hosted URL the frontend should redirect to.
//
// workspaceID is stored as metadata so the webhook handler can associate
// the completed session with the right workspace.
func (s *StripeService) CreateCheckoutSession(workspaceID, plan, successURL, cancelURL string) (string, error) {
	if !s.Enabled() {
		return "", ErrStripeNotConfigured
	}

	priceID, err := s.priceIDForPlan(plan)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"workspace_id": workspaceID,
			"plan":         plan,
		},
	}

	sess, err := chsession.New(params)
	if err != nil {
		slog.Error("stripe: failed to create checkout session", "workspace_id", workspaceID, "plan", plan, "error", err)
		return "", fmt.Errorf("stripe: create checkout session: %w", err)
	}

	slog.Info("stripe: checkout session created", "workspace_id", workspaceID, "plan", plan, "session_id", sess.ID)
	return sess.URL, nil
}

// CreatePortalSession creates a Stripe Customer Portal session so the user
// can manage their subscription (cancel, change plan, update payment info).
func (s *StripeService) CreatePortalSession(customerID, returnURL string) (string, error) {
	if !s.Enabled() {
		return "", ErrStripeNotConfigured
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	sess, err := bpsession.New(params)
	if err != nil {
		slog.Error("stripe: failed to create portal session", "customer_id", customerID, "error", err)
		return "", fmt.Errorf("stripe: create portal session: %w", err)
	}

	return sess.URL, nil
}

// GetSubscription fetches the active Stripe subscription for a Stripe customer ID.
// Returns nil, nil when the customer has no active subscription.
func (s *StripeService) GetSubscription(customerID string) (*StripeSubscription, error) {
	if !s.Enabled() {
		return nil, ErrStripeNotConfigured
	}

	status := string(stripe.SubscriptionStatusActive)
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String(status),
	}
	params.AddExpand("data.items.data.price")
	params.Filters.AddFilter("limit", "", "1")

	iter := stripesubscription.List(params)
	if iter.Err() != nil {
		return nil, fmt.Errorf("stripe: list subscriptions: %w", iter.Err())
	}

	if !iter.Next() {
		return nil, nil // no active subscription
	}

	sub := iter.Subscription()
	plan := s.planFromPriceID(sub.Items.Data[0].Price.ID)

	return &StripeSubscription{
		CustomerID:        customerID,
		SubscriptionID:    sub.ID,
		Plan:              plan,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	}, nil
}

// planFromPriceID reverses a Stripe Price ID to our internal plan slug.
func (s *StripeService) planFromPriceID(priceID string) string {
	switch priceID {
	case s.priceProID:
		return "pro"
	case s.priceGrowthID:
		return "growth"
	case s.priceBusinessID:
		return "business"
	default:
		return "unknown"
	}
}

// HandleWebhook verifies the Stripe webhook signature and returns the parsed
// event. The raw body must be passed as-is (not decoded) for signature
// verification to succeed.
func (s *StripeService) HandleWebhook(payload []byte, signature string) (stripe.Event, error) {
	if s.webhookSecret == "" {
		return stripe.Event{}, fmt.Errorf("stripe: STRIPE_WEBHOOK_SECRET is not set")
	}

	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("stripe: webhook signature verification failed: %w", err)
	}

	return event, nil
}
