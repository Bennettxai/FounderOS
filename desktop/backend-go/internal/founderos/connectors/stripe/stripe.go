// Package stripe ports FounderOS v1's Stripe reads (lib/connectors/payments.ts:
// stripeSnapshot, monthToDateIncome, stripeMtdForKey; lib/funnel-stripe.ts:
// the charge pull and mapStripeCharges) for both of the operator's Stripe accounts:
// Launchpad Cohort on STRIPE_SECRET_KEY and Vantage on STRIPE_VANTAGE_KEY.
//
// Plain HTTP rather than stripe-go: the connector needs two GET endpoints
// (/v1/balance, /v1/charges), and every request has to go through the guarded
// connectors.HTTPClient. The SDK would add a large dependency and its own
// transport for no gain. Nothing here creates, refunds or pays out.
package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Meta is the processor id FounderOS v1's configuredProcessors uses for the
// primary account. The Connections board carries Stripe inside "payments".
var Meta = connectors.Meta{ID: "stripe", Name: "Stripe", Kind: connectors.KindPayments}

// Account is one Stripe account: its processor id and name (as in
// configuredProcessors), the env var holding its secret key, and the funnel
// venture its charges belong to (lib/funnel-stripe.ts).
type Account struct {
	ID      string
	Name    string
	EnvKey  string
	Venture string
}

var (
	LaunchpadCohort = Account{ID: "stripe", Name: "Stripe", EnvKey: "STRIPE_SECRET_KEY", Venture: "launchpad-cohort"}
	Vantage         = Account{ID: "stripe-vantage", Name: "Stripe · Vantage", EnvKey: "STRIPE_VANTAGE_KEY", Venture: "vantage"}
)

const (
	defaultBaseURL = "https://api.stripe.com"
	// MaxCharges is FounderOS v1's auto-pagination safety cap (~20 pages of 100).
	MaxCharges = 2000
	// WinDays is how far back a payment still counts as a funnel win.
	WinDays = 90
)

var ErrNotConfigured = errors.New("stripe: not configured")

// APIError is a non-2xx Stripe response. Error() is Stripe's own message, the
// way the Node SDK surfaces it.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("Stripe HTTP %d", e.StatusCode)
}

type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type RecentCharge struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Created     int64  `json:"created"`
}

// Snapshot matches FounderOS v1's StripeSnapshot.
type Snapshot struct {
	Available     []Money        `json:"available"`
	Pending       []Money        `json:"pending"`
	RecentCharges []RecentCharge `json:"recentCharges"`
}

// IncomeMTD matches FounderOS v1's IncomeMtd.
type IncomeMTD struct {
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	Count       int    `json:"count"`
}

// Customer is a charge's customer: an id string when not expanded, the
// customer object when expanded with expand[]=data.customer.
type Customer struct {
	ID      string  `json:"id"`
	Email   *string `json:"email"`
	Name    *string `json:"name"`
	Deleted bool    `json:"deleted"`
	// Expanded is true when Stripe returned the object rather than an id.
	Expanded bool `json:"-"`
}

func (c *Customer) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &c.ID)
	}
	type plain Customer
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*c = Customer(p)
	c.Expanded = true
	return nil
}

type BillingDetails struct {
	Email *string `json:"email"`
	Name  *string `json:"name"`
}

// Charge is the slice of Stripe's charge object FounderOS v1 reads.
type Charge struct {
	ID             string          `json:"id"`
	Amount         int64           `json:"amount"`
	Currency       string          `json:"currency"`
	Paid           bool            `json:"paid"`
	Status         string          `json:"status"`
	Refunded       bool            `json:"refunded"`
	AmountRefunded int64           `json:"amount_refunded"`
	Created        int64           `json:"created"`
	Description    *string         `json:"description"`
	ReceiptEmail   *string         `json:"receipt_email"`
	BillingDetails *BillingDetails `json:"billing_details"`
	Customer       *Customer       `json:"customer"`
}

type chargeList struct {
	Data    []Charge `json:"data"`
	HasMore bool     `json:"has_more"`
}

type Connector struct {
	res     connectors.Resolver
	acct    Account
	client  *http.Client
	BaseURL string
}

// New is the Launchpad Cohort account (STRIPE_SECRET_KEY).
func New(res connectors.Resolver) *Connector { return NewAccount(res, LaunchpadCohort) }

// NewAccount builds a connector for one account. The key is resolved on every
// call, never cached.
func NewAccount(res connectors.Resolver, acct Account) *Connector {
	return &Connector{res: res, acct: acct, client: connectors.HTTPClient(8 * time.Second), BaseURL: defaultBaseURL}
}

func (c *Connector) Account() Account { return c.acct }

func (c *Connector) key() string { return c.res.Resolve(c.acct.EnvKey) }

func (c *Connector) Configured() bool { return c.key() != "" }

func (c *Connector) notConfigured() error {
	return fmt.Errorf("%w: %s is not set", ErrNotConfigured, c.acct.EnvKey)
}

// Snapshot reads the balance and the five most recent charges. Unkeyed is
// ErrNotConfigured (FounderOS v1 threw "STRIPE_SECRET_KEY is not set").
func (c *Connector) Snapshot(ctx context.Context) (Snapshot, error) {
	key := c.key()
	if key == "" {
		return Snapshot{}, c.notConfigured()
	}
	var (
		wg      sync.WaitGroup
		balance struct {
			Available []Money `json:"available"`
			Pending   []Money `json:"pending"`
		}
		charges        chargeList
		balErr, chaErr error
	)
	wg.Add(2)
	go func() { defer wg.Done(); balErr = c.get(ctx, key, "/v1/balance", nil, &balance) }()
	go func() {
		defer wg.Done()
		chaErr = c.get(ctx, key, "/v1/charges", url.Values{"limit": {"5"}}, &charges)
	}()
	wg.Wait()
	if balErr != nil {
		return Snapshot{}, balErr
	}
	if chaErr != nil {
		return Snapshot{}, chaErr
	}
	snap := Snapshot{Available: nonNil(balance.Available), Pending: nonNil(balance.Pending), RecentCharges: []RecentCharge{}}
	for _, ch := range charges.Data {
		desc := ch.ID
		if ch.Description != nil {
			desc = *ch.Description
		}
		snap.RecentCharges = append(snap.RecentCharges, RecentCharge{Amount: ch.Amount, Currency: ch.Currency, Description: desc, Created: ch.Created})
	}
	return snap, nil
}

func nonNil(m []Money) []Money {
	if m == nil {
		return []Money{}
	}
	return m
}

// ListParams narrows a charge listing. Zero values mean: no created filter,
// 100 per page, no expansion, and the MaxCharges cap.
type ListParams struct {
	CreatedGTE     int64
	Limit          int
	ExpandCustomer bool
	Max            int
}

// ListCharges auto-paginates /v1/charges (has_more + starting_after) up to
// Max charges. Any failure fails the whole read: a partial list would pass
// for a complete one.
func (c *Connector) ListCharges(ctx context.Context, p ListParams) ([]Charge, error) {
	key := c.key()
	if key == "" {
		return nil, c.notConfigured()
	}
	limit := p.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	max := p.Max
	if max <= 0 {
		max = MaxCharges
	}
	out := []Charge{}
	after := ""
	for {
		q := url.Values{"limit": {strconv.Itoa(limit)}}
		if p.CreatedGTE > 0 {
			q.Set("created[gte]", strconv.FormatInt(p.CreatedGTE, 10))
		}
		if p.ExpandCustomer {
			q.Set("expand[]", "data.customer")
		}
		if after != "" {
			q.Set("starting_after", after)
		}
		var page chargeList
		if err := c.get(ctx, key, "/v1/charges", q, &page); err != nil {
			return nil, err
		}
		for _, ch := range page.Data {
			out = append(out, ch)
			if len(out) >= max {
				return out, nil
			}
		}
		if !page.HasMore || len(page.Data) == 0 {
			return out, nil
		}
		after = page.Data[len(page.Data)-1].ID
	}
}

// MonthToDateIncome sums the charges that settled (paid + succeeded) since the
// first of now's UTC month. Unkeyed is (nil, nil): the card stays honest
// pending. An API failure is an error, never a zero.
func (c *Connector) MonthToDateIncome(ctx context.Context, now time.Time) (*IncomeMTD, error) {
	if !c.Configured() {
		return nil, nil
	}
	charges, err := c.ListCharges(ctx, ListParams{CreatedGTE: MonthStartUnix(now)})
	if err != nil {
		return nil, err
	}
	sum := SumChargeIncome(charges)
	return &sum, nil
}

// FunnelWins pulls the last WinDays of charges with customers expanded and
// maps them to settled wins for this account's venture.
func (c *Connector) FunnelWins(ctx context.Context, now time.Time) ([]Win, error) {
	charges, err := c.ListCharges(ctx, ListParams{CreatedGTE: now.Unix() - WinDays*86400, ExpandCustomer: true})
	if err != nil {
		return nil, err
	}
	return MapCharges(charges, c.acct.Venture), nil
}

// Status verifies the key with a snapshot read. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: c.acct.ID, Name: c.acct.Name, Kind: connectors.KindPayments}
	if !c.Configured() {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.SetKeys(c.acct.EnvKey)
		return st
	}
	snap, err := c.Snapshot(ctx)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "Stripe key set but verification failed: " + err.Error()
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = c.acct.Name + " · available balance " + AvailableLabel(snap)
	return st
}

// AvailableLabel renders the first available bucket the way FounderOS v1's
// payments status does: "1234.56 USD", and "0.00 USD" when Stripe returns none.
func AvailableLabel(s Snapshot) string {
	amount, currency := int64(0), "usd"
	if len(s.Available) > 0 {
		amount, currency = s.Available[0].Amount, s.Available[0].Currency
	}
	return fmt.Sprintf("%.2f %s", float64(amount)/100, strings.ToUpper(currency))
}

func (c *Connector) get(ctx context.Context, key, path string, q url.Values, into any) error {
	u := strings.TrimRight(c.BaseURL, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var doc struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &doc)
		return &APIError{StatusCode: res.StatusCode, Type: doc.Error.Type, Message: doc.Error.Message}
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("stripe %s: decode: %w", path, err)
	}
	return nil
}

// founderosZone is the operator's calendar (America/Chicago): a month is his month.
var founderosZone = func() *time.Location {
	if loc, err := time.LoadLocation("America/Chicago"); err == nil {
		return loc
	}
	return time.UTC
}()

// MonthStartUnix is the first instant of now's calendar month on the operator's
// clock. (FounderOS v1 used the UTC month, so on the evening of the last day
// its month-to-date already read the next month.)
func MonthStartUnix(now time.Time) int64 {
	c := now.In(founderosZone)
	return time.Date(c.Year(), c.Month(), 1, 0, 0, 0, 0, founderosZone).Unix()
}

// SumChargeIncome sums what settled charges actually kept (paid + succeeded,
// net of refunds): a fully refunded charge adds nothing and is not counted,
// a partial refund adds its remainder. Currency is the last counted charge's,
// "usd" when there are none.
func SumChargeIncome(charges []Charge) IncomeMTD {
	out := IncomeMTD{Currency: "usd"}
	for _, ch := range charges {
		net := ch.Amount - ch.AmountRefunded
		if ch.Paid && ch.Status == "succeeded" && net > 0 {
			out.AmountCents += net
			out.Count++
			out.Currency = ch.Currency
		}
	}
	return out
}

// Win is one settled payment reduced to what the funnel needs (StripeWin).
// Stripe charges by default; PayKit buyers (pages/funnel MapPaykitCustomers)
// take the same shape with Source "paykit" so both processors fold through
// one merge.
type Win struct {
	ID string `json:"id"` // charge id (Stripe) · fb-<customer id> (PayKit)
	// Source is "stripe" or "paykit"; empty reads stripe.
	Source    string  `json:"source,omitempty"`
	Venture   string  `json:"venture"`
	Email     *string `json:"email"`
	Name      *string `json:"name"`
	Phone     *string `json:"phone,omitempty"`
	AmountUSD float64 `json:"amountUsd"`
	// Payments is PayKit only: the lifetime payment count behind AmountUSD.
	Payments int     `json:"payments,omitempty"`
	Product  *string `json:"product"`
	At       string  `json:"at"` // YYYY-MM-DD of the charge (PayKit: latest payment)
}

func clean(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func first(vals ...*string) *string {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

// MapCharges ports mapStripeCharges: only money that settled counts (paid,
// succeeded, not refunded, positive); identity falls through billing details,
// then the receipt email, then the expanded (non-deleted) customer; a charge
// with no identity still surfaces.
func MapCharges(charges []Charge, venture string) []Win {
	out := []Win{}
	for _, ch := range charges {
		if !ch.Paid || ch.Status != "succeeded" || ch.Refunded || ch.Amount <= 0 {
			continue
		}
		var cust *Customer
		if ch.Customer != nil && ch.Customer.Expanded && !ch.Customer.Deleted {
			cust = ch.Customer
		}
		var billEmail, billName, custEmail, custName *string
		if ch.BillingDetails != nil {
			billEmail, billName = clean(ch.BillingDetails.Email), clean(ch.BillingDetails.Name)
		}
		if cust != nil {
			custEmail, custName = clean(cust.Email), clean(cust.Name)
		}
		out = append(out, Win{
			ID:        ch.ID,
			Venture:   venture,
			Email:     first(billEmail, clean(ch.ReceiptEmail), custEmail),
			Name:      first(billName, custName),
			AmountUSD: float64(ch.Amount) / 100,
			Product:   clean(ch.Description),
			At:        time.Unix(ch.Created, 0).UTC().Format("2006-01-02"),
		})
	}
	return out
}
