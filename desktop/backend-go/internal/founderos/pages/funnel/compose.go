package funnel

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// Lanes are the funnel's sources (lib/funnel-compose.ts FunnelComposeDeps).
// An error means that lane did not answer (unkeyed or down), never "zero".
type Lanes struct {
	Typeform func(ctx context.Context, now time.Time) ([]typeform.Lead, error)
	Calendar func(ctx context.Context, now time.Time) ([]gcal.CalEvent, error)
	Fathom   func(ctx context.Context, now time.Time) ([]fathomcalls.Call, error)
	Stripe   func(ctx context.Context, now time.Time) ([]stripe.Win, error)
	// Paykit is the second processor (lib/funnel-paykit.ts): buyers in
	// the stripe.Win shape with Source "paykit".
	Paykit func(ctx context.Context, now time.Time) ([]stripe.Win, error)
	Trakyo func(ctx context.Context) ([]trakyo.Event, error)
	// Seed reads the seeded funnel (founderos_funnel_contacts + touches);
	// venture "" means both.
	Seed func(ctx context.Context, venture string) ([]Journey, error)
	// Archive reads the restored Attio / GoHighLevel history
	// (db.funnel.archivedJourneys); nil means no archive. venture "" = both.
	Archive func(ctx context.Context, venture string) ([]Journey, error)
}

// Composition is what lands in the funnel space. A nil slice means that
// source did not answer; Errors says why, per lane.
type Composition struct {
	Journeys      []Journey          `json:"journeys"`
	IsLive        bool               `json:"isLive"`
	TypeformLeads []typeform.Lead    `json:"-"`
	Bookings      []gcal.CalEvent    `json:"-"`
	Calls         []fathomcalls.Call `json:"-"`
	StripeWins    []stripe.Win       `json:"-"`
	PaykitWins    []stripe.Win       `json:"-"`
	// ArchivedJourneys is the retired CRM history, kept apart from the live
	// space; the page and route append it to the archive regardless of age.
	ArchivedJourneys []Journey         `json:"-"`
	Errors           map[string]string `json:"errors"`
	SeedError        string            `json:"seedError,omitempty"`
}

// Compose: Typeform leads open journeys, bookings and held calls fold on,
// Trakyo swaps in the attributed first touch, Stripe charges and PayKit
// buyers convert; filtered to one venture. The seeded funnel only when nothing
// live answered. The retired CRM archive is read beside it, never merged in.
func Compose(ctx context.Context, now time.Time, venture string, l Lanes) Composition {
	c := Composition{Errors: map[string]string{}}
	var mu sync.Mutex
	fail := func(lane string, err error) {
		mu.Lock()
		c.Errors[lane] = err.Error()
		mu.Unlock()
	}
	var wg sync.WaitGroup
	wg.Add(6)
	go func() {
		defer wg.Done()
		if v, err := l.Typeform(ctx, now); err != nil {
			fail("typeform", err)
		} else {
			c.TypeformLeads = nonNil(v)
		}
	}()
	go func() {
		defer wg.Done()
		if v, err := l.Calendar(ctx, now); err != nil {
			fail("calendar", err)
		} else {
			c.Bookings = nonNil(v)
		}
	}()
	go func() {
		defer wg.Done()
		if v, err := l.Fathom(ctx, now); err != nil {
			fail("fathom", err)
		} else {
			c.Calls = nonNil(v)
		}
	}()
	go func() {
		defer wg.Done()
		if v, err := l.Stripe(ctx, now); err != nil {
			fail("stripe", err)
		} else {
			c.StripeWins = nonNil(v)
		}
	}()
	go func() {
		defer wg.Done()
		if l.Paykit == nil {
			fail("paykit", errors.New("paykit lane not wired"))
			return
		}
		if v, err := l.Paykit(ctx, now); err != nil {
			fail("paykit", err)
		} else {
			c.PaykitWins = nonNil(v)
		}
	}()
	go func() {
		defer wg.Done()
		c.ArchivedJourneys = []Journey{}
		if l.Archive == nil {
			return
		}
		if v, err := l.Archive(ctx, venture); err != nil {
			fail("archive", err)
		} else if v != nil {
			c.ArchivedJourneys = v
		}
	}()
	wg.Wait()

	// Both processors fold through one merge, so a client who paid on each is one journey.
	wins := append(append([]stripe.Win{}, c.StripeWins...), c.PaykitWins...)
	live := MergeCallTouches(TypeformJourneys(c.TypeformLeads), c.Bookings, c.Calls, now)
	// Money alone (Stripe or PayKit) makes the funnel live: buyers are
	// never gated behind whether a lead source happened to answer.
	c.IsLive = len(live) > 0 || len(wins) > 0
	if !c.IsLive {
		seed, err := l.Seed(ctx, venture)
		if err != nil {
			c.SeedError = err.Error()
			seed = []Journey{}
		}
		c.Journeys = seed
		return c
	}
	events, err := l.Trakyo(ctx)
	if err != nil {
		c.Errors["trakyo"] = err.Error() // the funnel keeps its own first touches
	}
	merged := MergeStripeWins(MergeTrakyoTouches(live, events), wins)
	c.Journeys = []Journey{}
	for _, j := range merged {
		if venture == "" || j.Venture == venture {
			c.Journeys = append(c.Journeys, j)
		}
	}
	return c
}

func nonNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

// SourceLabel names the live sources ("typeform+fathom+stripe+paykit") or "seed".
func SourceLabel(c Composition) string {
	if !c.IsLive {
		return "seed"
	}
	has := func(source string) bool {
		for _, j := range c.Journeys {
			for _, t := range j.Touches {
				if t.Source == source {
					return true
				}
			}
		}
		return false
	}
	var parts []string
	if len(c.TypeformLeads) > 0 {
		parts = append(parts, "typeform")
	}
	if has("calendar") {
		parts = append(parts, "calendar")
	}
	if has("fathom") {
		parts = append(parts, "fathom")
	}
	if len(c.StripeWins) > 0 {
		parts = append(parts, "stripe")
	}
	if len(c.PaykitWins) > 0 {
		parts = append(parts, "paykit")
	}
	if len(parts) == 0 {
		return "stripe"
	}
	return strings.Join(parts, "+")
}
