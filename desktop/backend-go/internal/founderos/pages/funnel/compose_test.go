package funnel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// Ported from FounderOS v1 tests/funnel-compose.test.ts.

var errDown = errors.New("not configured")

func lanes(mod func(*Lanes)) Lanes {
	l := Lanes{
		Typeform: func(context.Context, time.Time) ([]typeform.Lead, error) { return nil, errDown },
		Calendar: func(context.Context, time.Time) ([]gcal.CalEvent, error) { return nil, errDown },
		Fathom:   func(context.Context, time.Time) ([]fathomcalls.Call, error) { return nil, errDown },
		Stripe:   func(context.Context, time.Time) ([]stripe.Win, error) { return []stripe.Win{}, nil },
		Paykit:   func(context.Context, time.Time) ([]stripe.Win, error) { return nil, errDown },
		Trakyo:   func(context.Context) ([]trakyo.Event, error) { return nil, nil },
		Seed:     func(context.Context, string) ([]Journey, error) { return []Journey{}, nil },
	}
	if mod != nil {
		mod(&l)
	}
	return l
}

func buyer(mod func(*stripe.Win)) stripe.Win {
	w := stripe.Win{ID: "ch_1", Venture: "launchpad-cohort", Email: sp("buyer@example.com"), Name: sp("Cohort Buyer"), AmountUSD: 1497, Product: sp("FounderOS Cohort"), At: "2026-08-10"}
	if mod != nil {
		mod(&w)
	}
	return w
}

var now0 = time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

func TestComposeSurfacesAStripeBuyerWithNoLead(t *testing.T) {
	c := Compose(context.Background(), now0, "", lanes(func(l *Lanes) {
		l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) { return []stripe.Win{buyer(nil)}, nil }
	}))
	if !c.IsLive || c.TypeformLeads != nil || len(c.Journeys) != 1 || c.Journeys[0].Status != "converted" || *c.Journeys[0].AmountUSD != 1497 {
		t.Fatalf("composition = %+v", c)
	}
}

func TestComposeFiltersToTheVenture(t *testing.T) {
	c := Compose(context.Background(), now0, "vantage", lanes(func(l *Lanes) {
		l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) { return []stripe.Win{buyer(nil)}, nil }
	}))
	if len(c.Journeys) != 0 {
		t.Fatalf("journeys = %+v", c.Journeys)
	}
}

func TestComposeFallsBackToSeedWhenNothingIsLive(t *testing.T) {
	var asked string
	c := Compose(context.Background(), now0, "vantage", lanes(func(l *Lanes) {
		l.Seed = func(_ context.Context, v string) ([]Journey, error) { asked = v; return []Journey{{ID: "seed-1"}}, nil }
	}))
	if c.IsLive || len(c.Journeys) != 1 || c.Journeys[0].ID != "seed-1" || asked != "vantage" || SourceLabel(c) != "seed" {
		t.Fatalf("composition = %+v", c)
	}
	broken := Compose(context.Background(), now0, "", lanes(func(l *Lanes) {
		l.Seed = func(context.Context, string) ([]Journey, error) { return nil, errors.New("db down") }
	}))
	if broken.SeedError == "" || len(broken.Journeys) != 0 {
		t.Fatalf("an unreadable seed is surfaced, not an empty funnel: %+v", broken)
	}
}

func TestComposeTheTypeformToStripeLane(t *testing.T) {
	lead := typeform.Lead{ResponseID: "resp1", FormID: "frmAA", FormTitle: "Launchpad Cohort application", Name: "Dana Reyes", Email: "dana@reyes.co", SubmittedAt: "2026-08-01T10:00:00Z", Hidden: map[string]string{}}
	tf := func(context.Context, time.Time) ([]typeform.Lead, error) { return []typeform.Lead{lead}, nil }

	c := Compose(context.Background(), now0, "", lanes(func(l *Lanes) { l.Typeform = tf }))
	if !c.IsLive || len(c.Journeys) != 1 || c.Journeys[0].ID != "typeform-resp1" {
		t.Fatalf("composition = %+v", c)
	}

	c = Compose(context.Background(), now0, "", lanes(func(l *Lanes) {
		l.Typeform = tf
		l.Trakyo = func(context.Context) ([]trakyo.Event, error) {
			return []trakyo.Event{{Lead: "Dana", Names: []string{"Dana Reyes"}, Emails: []string{"dana@reyes.co"}, Label: `IG reel: "AI receptionist"`, Channel: "organic", At: "2026-07-30", SourceType: "custom", SourceName: sp("Instagram")}}, nil
		}
		l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) {
			return []stripe.Win{buyer(func(w *stripe.Win) { w.Email, w.Name = sp("dana@reyes.co"), sp("Dana Reyes") })}, nil
		}
	}))
	if len(c.Journeys) != 1 || c.Journeys[0].Touches[0].Source != "trakyo" || c.Journeys[0].Status != "converted" {
		t.Fatalf("composition = %+v", c.Journeys)
	}

	c = Compose(context.Background(), now0, "", lanes(func(l *Lanes) {
		l.Typeform = tf
		l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) { return []stripe.Win{buyer(nil)}, nil }
	}))
	if SourceLabel(c) != "typeform+stripe" {
		t.Fatalf("label = %q", SourceLabel(c))
	}
	if c.Errors["calendar"] == "" || c.Errors["fathom"] == "" {
		t.Fatalf("lanes that did not answer say why: %+v", c.Errors)
	}
}

// Ported from FounderOS v1 tests/funnel-archive.test.ts: the restored Attio /
// GoHighLevel history rides beside the live journeys, never inside them.
func TestComposeReturnsTheRetiredCRMArchiveSeparately(t *testing.T) {
	old := crmJourney("attio-old", "Historic lead", "engaged")
	old.Venture = "vantage"
	var asked string
	c := Compose(context.Background(), now0, "vantage", lanes(func(l *Lanes) {
		l.Stripe = func(context.Context, time.Time) ([]stripe.Win, error) {
			return []stripe.Win{buyer(func(w *stripe.Win) { w.Venture, w.Email = "vantage", nil })}, nil
		}
		l.Archive = func(_ context.Context, v string) ([]Journey, error) { asked = v; return []Journey{old}, nil }
	}))
	if !c.IsLive || asked != "vantage" || len(c.ArchivedJourneys) != 1 || c.ArchivedJourneys[0].ID != "attio-old" {
		t.Fatalf("composition = %+v", c)
	}
	for _, j := range c.Journeys {
		if j.ID == "attio-old" {
			t.Fatal("an archived CRM journey must never land in the live space")
		}
	}
	broken := Compose(context.Background(), now0, "", lanes(func(l *Lanes) {
		l.Archive = func(context.Context, string) ([]Journey, error) { return nil, errors.New("no table") }
	}))
	if broken.Errors["archive"] != "no table" || broken.ArchivedJourneys == nil || len(broken.ArchivedJourneys) != 0 {
		t.Fatalf("an unreadable archive says why and reads empty: %+v", broken)
	}
	if none := Compose(context.Background(), now0, "", lanes(nil)); none.ArchivedJourneys == nil {
		t.Fatal("no archive lane is an empty archive, not null")
	}
}

func TestComposePaykitLaneThatDidNotAnswerSaysWhy(t *testing.T) {
	c := Compose(context.Background(), now0, "", lanes(nil))
	if c.Errors["paykit"] == "" || c.PaykitWins != nil {
		t.Fatalf("composition = %+v", c)
	}
}
