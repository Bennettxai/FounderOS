package funnel

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

// ---- PayKit wins (lib/funnel-paykit.ts) ---------------------------------
//
// PayKit has no transaction feed; /customers carries each buyer's identity,
// LIFETIME spend, payment count and the date of their LATEST payment. So a
// funnel win here is one customer, dated at their last payment, valued at what
// they have paid in total. Wins take the stripe.Win shape (Source "paykit")
// and fold through MergeStripeWins, so a buyer who paid on both processors is
// one client. Names, emails and phones live in memory for one render only.

var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func fbClean(v any) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if t := strings.TrimSpace(s); t != "" {
		return &t
	}
	return nil
}

// fbNumber is JavaScript's Number(String(v ?? ”).replace(/,/g, ”)).
func fbNumber(v any) float64 {
	var s string
	switch x := v.(type) {
	case nil:
		return 0
	case json.Number:
		s = x.String()
	case string:
		s = x
	case bool:
		if x {
			return 1
		}
		return 0
	default:
		return math.NaN()
	}
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

// MapPaykitCustomers turns one raw /customers page into the wins paid within
// the Stripe win window (stripe.WinDays).
func MapPaykitCustomers(raw []byte, venture string, now time.Time) []stripe.Win {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc struct {
		Data struct {
			Customers []map[string]any `json:"customers"`
		} `json:"data"`
	}
	if dec.Decode(&doc) != nil {
		return []stripe.Win{}
	}
	since := now.UTC().Add(-stripe.WinDays * 24 * time.Hour).Format("2006-01-02")
	out := []stripe.Win{}
	for _, c := range doc.Data.Customers {
		if c["id"] == nil {
			continue
		}
		amount := fbNumber(c["total_spent"])
		if !(amount > 0) {
			continue
		}
		// PayKit sends a local offset (…-05:00); the calendar date as written
		// is the day the buyer paid, so slice rather than shift it through UTC.
		at := fbClean(c["last_transaction_date"])
		if at == nil || len(*at) < 10 || !isoDate.MatchString((*at)[:10]) || (*at)[:10] < since {
			continue
		}
		id := ""
		switch x := c["id"].(type) {
		case json.Number:
			id = x.String()
		case string:
			id = x
		default:
			continue
		}
		w := stripe.Win{ID: "fb-" + id, Source: "paykit", Venture: venture, Email: fbClean(c["email"]), Name: fbClean(c["name"]), Phone: fbClean(c["phone"]),
			AmountUSD: math.Round(amount*100) / 100, At: (*at)[:10]}
		if n := fbNumber(c["total_transactions"]); !math.IsNaN(n) && n >= 1 {
			w.Payments = int(math.Round(n))
		}
		out = append(out, w)
	}
	return out
}
