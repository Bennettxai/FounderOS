// Package paykit ports FounderOS v1's PayKit reads (lib/connectors/payments.ts,
// the PayKit section) for both accounts: Launchpad Cohort on PAYKIT_LC_KEY
// and Vantage on PAYKIT_VANTAGE_KEY.
//
// PayKit exposes no payments list. /customers and /products are the only
// public endpoints, and /customers carries each customer's LIFETIME total_spent,
// total_transactions and latest transaction date. So a month is either:
//
//   - a band (FOS-655): the exact floor from one-time buyers, an upper bound, and
//     a count of repeat buyers whose split the API cannot give; or
//   - exact (FOS-658), by differencing two stored snapshots of the running
//     totals that bracket the month.
//
// Snapshots are kept by a HistoryPort. This package produces the rows of
// paykit.db's paykit_customer_snapshots table (SnapshotRows) and reads them
// back (SnapshotsFromRows); persistence itself belongs to the ETL (§3).
// Nothing here writes to PayKit.
package paykit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Meta is the primary account's processor id in FounderOS v1's
// configuredProcessors. The Connections board carries it inside "payments".
var Meta = connectors.Meta{ID: "paykit-lc", Name: "PayKit · Launchpad Cohort", Kind: connectors.KindPayments}

type Account struct {
	ID     string
	Name   string
	EnvKey string
}

var (
	LaunchpadCohort = Account{ID: "paykit-lc", Name: "PayKit · Launchpad Cohort", EnvKey: "PAYKIT_LC_KEY"}
	Vantage         = Account{ID: "paykit-vantage", Name: "PayKit · Vantage", EnvKey: "PAYKIT_VANTAGE_KEY"}
)

const (
	defaultBaseURL = "https://www.paykit.com/public-api"
	perPage        = 100
	maxPages       = 50
)

var ErrNotConfigured = errors.New("paykit: not configured")

// Customer is one /customers row reduced to ids and numbers. Names, emails and
// phones come back in the same payload and are dropped.
type Customer struct {
	// ID is the stable PayKit customer id, the key differencing joins on.
	// Empty when the row carries none, which leaves it undiffable.
	ID string `json:"id"`
	// TotalSpentCents is LIFETIME spend, not this month's.
	TotalSpentCents int64 `json:"totalSpentCents"`
	// Month (YYYY-MM) of the LATEST transaction; nil when absent.
	Month *string `json:"month"`
	// Transactions is the lifetime count. 1 means the spend is splittable; an
	// absent or unparseable count reads 0, i.e. unsplittable.
	Transactions int `json:"transactions"`
	// LastTransactionDate verbatim (PayKit sends a local offset); nil when absent.
	LastTransactionDate *string `json:"lastTransactionDate"`
}

// MonthIncome is a month split into what is proven and what is possible.
type MonthIncome struct {
	ExactCents            int64 `json:"exactCents"`
	UpperCents            int64 `json:"upperCents"`
	UnsplittableCustomers int   `json:"unsplittableCustomers"`
}

// MonthIncomeUSD is the finances page's IncomeBand.
type MonthIncomeUSD struct {
	ExactUSD              float64 `json:"exactUsd"`
	UpperUSD              float64 `json:"upperUsd"`
	UnsplittableCustomers int     `json:"unsplittableCustomers"`
}

func (m MonthIncome) USD() MonthIncomeUSD {
	return MonthIncomeUSD{ExactUSD: float64(m.ExactCents) / 100, UpperUSD: float64(m.UpperCents) / 100, UnsplittableCustomers: m.UnsplittableCustomers}
}

type Connector struct {
	res     connectors.Resolver
	acct    Account
	client  *http.Client
	BaseURL string
	// Now dates the snapshot a pull records and the default month.
	Now func() time.Time
}

// New is the Launchpad Cohort account (PAYKIT_LC_KEY).
func New(res connectors.Resolver) *Connector { return NewAccount(res, LaunchpadCohort) }

func NewAccount(res connectors.Resolver, acct Account) *Connector {
	return &Connector{res: res, acct: acct, client: connectors.HTTPClient(8 * time.Second), BaseURL: defaultBaseURL, Now: time.Now}
}

func (c *Connector) Account() Account { return c.acct }

func (c *Connector) key() string { return c.res.Resolve(c.acct.EnvKey) }

func (c *Connector) Configured() bool { return c.key() != "" }

// Customers paginates /customers (x-api-key) until an empty page, up to 50
// pages of 100. Any non-OK page fails the whole read: a sum over a partial
// list would be a lie.
func (c *Connector) Customers(ctx context.Context) ([]Customer, error) {
	pages, err := c.CustomerPages(ctx)
	if err != nil {
		return nil, err
	}
	all := []Customer{}
	for _, body := range pages {
		all = append(all, ParseCustomers(body)...)
	}
	return all, nil
}

// CustomerPages is the same walk returning each non-empty /customers page as
// PayKit sent it. The funnel (lib/funnel-paykit.ts) matches buyers to
// leads by name, email and phone, which Customers drops; it holds them in
// memory for one render and never stores them. All or nothing, like Customers.
func (c *Connector) CustomerPages(ctx context.Context) ([][]byte, error) {
	key := c.key()
	if key == "" {
		return nil, fmt.Errorf("%w: %s is not set", ErrNotConfigured, c.acct.EnvKey)
	}
	out := [][]byte{}
	for p := 1; p <= maxPages; p++ {
		body, err := c.get(ctx, key, p, perPage)
		if err != nil {
			return nil, err
		}
		if len(ParseCustomers(body)) == 0 {
			break
		}
		out = append(out, body)
	}
	return out, nil
}

// MonthToDateIncome is paykitMonthToDateIncome. month is YYYY-MM; empty
// means now's UTC month.
//
//   - unkeyed: (nil, nil), the card stays honest pending;
//   - any HTTP failure: an error, never a fake $0;
//   - with a history: the pull is recorded as today's snapshot FIRST, then the
//     month is differenced, exact when a snapshot predates the month;
//   - otherwise, or if the store fails: the FOS-655 band. A broken store costs
//     precision, never the figure.
func (c *Connector) MonthToDateIncome(ctx context.Context, month string, history HistoryPort) (*MonthIncomeUSD, error) {
	if !c.Configured() {
		return nil, nil
	}
	now := c.Now().UTC()
	if month == "" {
		month = now.Format("2006-01")
	}
	all, err := c.Customers(ctx)
	if err != nil {
		return nil, err
	}
	if history != nil {
		if exact := differenced(history, Snapshot{CapturedOn: now.Format("2006-01-02"), Source: SourceLive, Customers: all}, month); exact != nil {
			usd := exact.USD()
			return &usd, nil
		}
	}
	band := MonthIncomeCents(all, month).USD()
	return &band, nil
}

func differenced(h HistoryPort, today Snapshot, month string) *MonthIncome {
	if h.Record(today) != nil {
		return nil
	}
	snaps, err := h.Snapshots()
	if err != nil {
		return nil
	}
	return MonthFromSnapshots(snaps, month)
}

// Status verifies the key with a one-row read of /customers. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	st := connectors.Status{ID: c.acct.ID, Name: c.acct.Name, Kind: connectors.KindPayments}
	key := c.key()
	if key == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = connectors.SetKeys(c.acct.EnvKey)
		return st
	}
	body, err := c.get(ctx, key, 1, 1)
	if err != nil {
		st.State = connectors.StateError
		st.Detail = "PayKit key set but verification failed: " + err.Error()
		return st
	}
	var doc struct {
		Data struct {
			Customers []json.RawMessage `json:"customers"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &doc) != nil || doc.Data.Customers == nil {
		st.State = connectors.StateError
		st.Detail = "PayKit key set but verification failed: unexpected /customers payload"
		return st
	}
	st.State = connectors.StateConnected
	st.Detail = c.acct.Name + " key verified"
	return st
}

func (c *Connector) get(ctx context.Context, key string, page, per int) ([]byte, error) {
	q := url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(per)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/customers?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("paykit /customers page %d: HTTP %d", page, res.StatusCode)
	}
	return body, nil
}

// ParseCustomers maps a /customers payload ({data: {customers: [...]}}) with
// JavaScript's Number() semantics, as FounderOS v1 does: total_spent has its
// commas stripped, and an absent, zero or unparseable total_transactions
// reads 0 (unsplittable), never 1.
func ParseCustomers(raw []byte) []Customer {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc struct {
		Data struct {
			Customers []map[string]any `json:"customers"`
		} `json:"data"`
	}
	if dec.Decode(&doc) != nil {
		return []Customer{}
	}
	out := make([]Customer, 0, len(doc.Data.Customers))
	for _, c := range doc.Data.Customers {
		cu := Customer{ID: idString(c["id"])}
		if amt, ok := jsNumber(c["total_spent"], true); ok {
			cu.TotalSpentCents = jsRound(amt * 100)
		}
		if at, ok := c["last_transaction_date"].(string); ok {
			cu.LastTransactionDate = &at
			m := at
			if len(m) > 7 {
				m = m[:7]
			}
			cu.Month = &m
		}
		if n, ok := jsNumber(c["total_transactions"], false); ok && n >= 1 {
			cu.Transactions = int(jsRound(n))
		}
		out = append(out, cu)
	}
	return out
}

func idString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

// jsNumber approximates JavaScript's Number(v) for the JSON values PayKit
// sends. nullAsZero mirrors `total_spent ?? ”`, where null becomes "" → 0;
// otherwise null → Number(null) = 0 and absent → NaN, both non-counting.
func jsNumber(v any, nullAsZero bool) (float64, bool) {
	var s string
	switch x := v.(type) {
	case nil:
		if nullAsZero {
			return 0, true
		}
		return 0, false
	case json.Number:
		s = x.String()
	case string:
		s = x
	default:
		return 0, false
	}
	if nullAsZero {
		s = strings.ReplaceAll(s, ",", "")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, false
	}
	return f, true
}

// jsRound matches JavaScript's Math.round (half rounds toward +Inf).
func jsRound(x float64) int64 { return int64(math.Floor(x + 0.5)) }

// SumMonthCents is the month's CEILING: the lifetime spend of every customer
// whose latest transaction is in it. It is never a figure on its own.
func SumMonthCents(cs []Customer, month string) int64 {
	var sum int64
	for _, c := range cs {
		if c.Month != nil && *c.Month == month {
			sum += c.TotalSpentCents
		}
	}
	return sum
}

// MonthIncomeCents splits a month into the proven floor (one-time buyers), the
// ceiling, and the count of repeat buyers between them.
func MonthIncomeCents(cs []Customer, month string) MonthIncome {
	var m MonthIncome
	for _, c := range cs {
		if c.Month == nil || *c.Month != month {
			continue
		}
		m.UpperCents += c.TotalSpentCents
		if c.Transactions == 1 {
			m.ExactCents += c.TotalSpentCents
		} else {
			m.UnsplittableCustomers++
		}
	}
	return m
}
