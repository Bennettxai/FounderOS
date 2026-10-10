package funnel

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
)

// Ported from FounderOS v1 tests/funnel-stripe.test.ts and tests/funnel-trakyo.test.ts.

func crmJourney(id, name, status string) Journey {
	return Journey{ID: id, Name: name, Venture: "launchpad-cohort", Status: status, Relationship: "warm", Likelihood: 60, CreatedAt: "2026-06-01",
		Touches: []Touch{
			{ID: id + "-t1", ContactID: id, Seq: 1, Stage: "first_touch", Channel: "crm", Label: "Deal created in Attio", Source: "attio", At: "2026-06-01"},
			{ID: id + "-t2", ContactID: id, Seq: 2, Stage: status, Channel: "crm", Label: "Attio stage", Source: "attio", At: "2026-06-20"},
		}}
}

func win(id string, mod func(*stripe.Win)) stripe.Win {
	w := stripe.Win{ID: id, Venture: "launchpad-cohort", AmountUSD: 500, At: "2026-07-15"}
	if mod != nil {
		mod(&w)
	}
	return w
}

func TestStripeWinMatchedByEmailConverts(t *testing.T) {
	j := crmJourney("j1", "Drew Halpern", "opted_in")
	j.Email = sp("drew@x.com")
	got := MergeStripeWins([]Journey{j}, []stripe.Win{win("ch_1", func(w *stripe.Win) { w.Email, w.Product = sp("Drew@X.com"), sp("AI Accelerator") })})[0]
	last := got.Touches[len(got.Touches)-1]
	if got.Status != "converted" || got.Relationship != "hot" || got.Likelihood != 100 || *got.Product != "AI Accelerator" || *got.AmountUSD != 500 {
		t.Fatalf("journey = %+v", got)
	}
	if last.Stage != "converted" || last.Channel != "checkout" || last.Source != "stripe" || last.At != "2026-07-15" || last.Seq != 3 || !strings.Contains(last.Label, "$500") || len(got.Touches) != 3 {
		t.Fatalf("touch = %+v", last)
	}
}

func TestStripePaidAmountNeverSumsOntoDealValue(t *testing.T) {
	a := crmJourney("j1", "A", "opted_in")
	a.Email, a.AmountUSD = sp("a@x.com"), f64(10000)
	if got := MergeStripeWins([]Journey{a}, []stripe.Win{win("c1", func(w *stripe.Win) { w.Email, w.AmountUSD = sp("a@x.com"), 2000 })})[0]; *got.AmountUSD != 10000 {
		t.Fatal("deposit on a bigger deal keeps the deal value")
	}
	a.AmountUSD = f64(1000)
	if got := MergeStripeWins([]Journey{a}, []stripe.Win{win("c1", func(w *stripe.Win) { w.Email, w.AmountUSD = sp("a@x.com"), 3000 })})[0]; *got.AmountUSD != 3000 {
		t.Fatal("paid more wins")
	}
}

func TestStripeOrphansGroupByEmail(t *testing.T) {
	merged := MergeStripeWins([]Journey{crmJourney("j1", "Someone Else", "opted_in")},
		[]stripe.Win{win("ch_9", func(w *stripe.Win) {
			w.Email, w.Name, w.Venture, w.Product = sp("new@x.com"), sp("New Buyer"), "vantage", sp("Build sprint")
		})})
	solo := merged[1]
	if len(merged) != 2 || solo.ID != "stripe-ch_9" || solo.Name != "New Buyer" || solo.Venture != "vantage" || solo.Status != "converted" || *solo.AmountUSD != 500 || len(solo.Touches) != 1 {
		t.Fatalf("solo = %+v", solo)
	}
	merged = MergeStripeWins(nil, []stripe.Win{
		win("ch_a", func(w *stripe.Win) { w.Email, w.At = sp("repeat@x.com"), "2026-07-01" }),
		win("ch_b", func(w *stripe.Win) { w.Email, w.AmountUSD, w.At = sp("repeat@x.com"), 250, "2026-07-20" }),
		win("ch_c", nil),
	})
	if len(merged) != 2 || *merged[0].AmountUSD != 750 || merged[0].CreatedAt != "2026-07-01" || merged[1].Name != "Stripe customer" {
		t.Fatalf("merged = %+v", merged)
	}
	if got := MergeStripeWins([]Journey{crmJourney("j", "A", "engaged")}, nil); len(got) != 1 {
		t.Fatal("no wins pass through")
	}
}

func event(lead, label, at string, mod func(*trakyo.Event)) trakyo.Event {
	e := trakyo.Event{Lead: lead, Names: []string{lead}, Emails: []string{}, Label: label, Channel: "organic", At: at, SourceType: "custom"}
	if mod != nil {
		mod(&e)
	}
	return e
}

func TestTrakyoSwapsTheSyntheticFirstTouch(t *testing.T) {
	got := MergeTrakyoTouches([]Journey{crmJourney("j1", "Drew Halpern", "engaged")}, []trakyo.Event{event("Drew Halpern", `IG reel: "3 AI offers"`, "2026-05-28", nil)})[0]
	if got.Touches[0].Source != "trakyo" || got.Touches[0].At != "2026-05-28" || got.Touches[1].Source != "attio" || got.Touches[0].Acquisition != "instagram" {
		t.Fatalf("touches = %+v", got.Touches)
	}
	spaced := MergeTrakyoTouches([]Journey{crmJourney("j1", "Drew Halpern", "engaged")}, []trakyo.Event{event("  drew   HALPERN ", "YT long-form", "2026-05-20", nil)})[0]
	if spaced.Touches[0].Source != "trakyo" {
		t.Fatal("matching is name-normalized")
	}
	j := crmJourney("j1", "Drew Halpern — Vantage build", "engaged")
	j.Email = sp("drew@example.com")
	byMail := MergeTrakyoTouches([]Journey{j}, []trakyo.Event{event("D. Halpern", "YT", "2026-05-20", func(e *trakyo.Event) { e.Emails = []string{"DREW@example.com"} })})[0]
	if byMail.Touches[0].Source != "trakyo" {
		t.Fatal("matches by email")
	}
	orig := crmJourney("j2", "Someone Else", "engaged")
	if got := MergeTrakyoTouches([]Journey{orig}, []trakyo.Event{event("Drew Halpern", "x", "2026-05-28", nil)})[0]; !reflect.DeepEqual(got, orig) {
		t.Fatal("unmatched passes through")
	}
}

func TestTrakyoAcquisition(t *testing.T) {
	name := func(s string) func(*trakyo.Event) { return func(e *trakyo.Event) { e.SourceName = sp(s) } }
	typ := func(s string) func(*trakyo.Event) { return func(e *trakyo.Event) { e.SourceType = s } }
	cases := []struct {
		e    trakyo.Event
		want string
	}{
		{event("x", "anything at all", "2026-05-01", typ("youtube")), "youtube"},
		{event("x", "anything at all", "2026-05-01", typ("meta_ad")), "instagram"},
		{event("x", "June 16th Webinar", "2026-05-01", name("Instagram")), "instagram"},
		{event("x", "FounderOS launch", "2026-05-01", name("thefounderos-waitlist-launch")), "form"},
		{event("x", "instagram.com", "2026-05-01", typ("referrer")), "instagram"},
		{event("x", "June 16th - The operator", "2026-05-01", name("June 16th - The operator")), "word_of_mouth"},
	}
	for _, c := range cases {
		if got := TrakyoAcquisition(c.e); got != c.want {
			t.Errorf("%+v = %s, want %s", c.e, got, c.want)
		}
	}
}

func TestUSDLabel(t *testing.T) {
	for in, want := range map[float64]string{500: "500", 1234.5: "1,234.5", 1000000: "1,000,000", 99.999: "100", 12.345: "12.35"} {
		if got := enUS(in, 2); got != want {
			t.Errorf("enUS(%v) = %q, want %q", in, got, want)
		}
	}
}
