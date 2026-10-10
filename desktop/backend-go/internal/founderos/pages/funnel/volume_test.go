package funnel

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
)

// Ported from FounderOS v1 tests/funnel-volume.test.ts and tests/funnel-contact.test.ts.

func vj(id, status string, touches [][3]string, mod func(*Journey)) Journey {
	j := Journey{ID: id, Name: id, Venture: "vantage", Status: status, Relationship: "warm", Likelihood: 50, CreatedAt: touches[0][2]}
	for i, t := range touches {
		j.Touches = append(j.Touches, Touch{ID: id, ContactID: id, Seq: i + 1, Stage: t[0], Channel: t[1], Label: t[0] + " touch", Source: "typeform", At: t[2]})
	}
	if mod != nil {
		mod(&j)
	}
	return j
}

func volumeFixture() []Journey {
	return []Journey{
		vj("won-a", "converted", [][3]string{{"first_touch", "organic", "2026-09-01"}, {"engaged", "call", "2026-09-05"}, {"nurtured", "call", "2026-09-08"}, {"converted", "checkout", "2026-09-10"}}, func(j *Journey) { j.AmountUSD = f64(3000) }),
		vj("won-b", "converted", [][3]string{{"first_touch", "ads", "2026-09-02"}, {"engaged", "call", "2026-09-06"}, {"nurtured", "call", "2026-09-09"}, {"converted", "checkout", "2026-09-20"}}, func(j *Journey) { j.AmountUSD = f64(1500) }),
		vj("held", "nurtured", [][3]string{{"first_touch", "organic", "2026-09-03"}, {"engaged", "call", "2026-09-20"}, {"nurtured", "call", "2026-09-23"}}, func(j *Journey) {
			j.Likelihood, j.Relationship, j.Person = 80, "hot", sp("Hot Holly")
		}),
		vj("booked", "engaged", [][3]string{{"first_touch", "organic", "2026-09-04"}, {"engaged", "call", "2026-09-10"}}, func(j *Journey) { j.Person = sp("Stalled Sam") }),
		vj("lead", "first_touch", [][3]string{{"first_touch", "ads", "2026-09-22"}}, nil),
	}
}

func TestVolumeHeadlineChipsMetersSeriesInsight(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	v := VolumeOf(volumeFixture(), 2, now, 30)
	if v.RevenueUSD != 4500 {
		t.Fatalf("revenue = %v", v.RevenueUSD)
	}
	if !reflect.DeepEqual(v.Chips, []Chip{{Tone: "ok", Text: "2 closed"}, {Tone: "err", Text: "1 stalled"}, {Text: "2 archived"}}) {
		t.Fatalf("chips = %+v", v.Chips)
	}
	if v.Caption != "closed revenue across 5 active clients · 3 organic / 2 ads entry" {
		t.Fatalf("caption = %q", v.Caption)
	}
	labels, displays := []string{}, []string{}
	for _, m := range v.Meters {
		labels, displays = append(labels, m.Label), append(displays, m.Display)
	}
	// v1: four meters, one per stage hand-off
	if !reflect.DeepEqual(labels, []string{"First touch → Engaged (4/5)", "Engaged → Nurtured (3/4)", "Nurtured → Opted in (2/3)", "Opted in → Converted (2/2)"}) || !reflect.DeepEqual(displays, []string{"80%", "75%", "67%", "100%"}) {
		t.Fatalf("meters = %v %v", labels, displays)
	}
	hues := []string{}
	for _, m := range v.Meters {
		hues = append(hues, m.Hue)
	}
	if !reflect.DeepEqual(hues, []string{"var(--ramp-1)", "var(--ramp-2)", "var(--ramp-3)", "var(--bn-accent)"}) {
		t.Fatalf("hues = %v", hues)
	}
	if *v.Meters[0].Frac != 0.8 || v.Foot != "each bar is a stage over the one before · 40% end to end" {
		t.Fatalf("foot = %q", v.Foot)
	}
	if len(v.Series) != 30 || v.Series[29] != (Point{Label: "Sep 24", Count: 0}) || v.TouchesInWindow != 14 {
		t.Fatalf("series tail = %+v, total %d", v.Series[29], v.TouchesInWindow)
	}
	for _, p := range v.Series {
		if p.Label == "Sep 10" && p.Count != 2 {
			t.Fatalf("Sep 10 = %d", p.Count)
		}
	}
	if v.Insight.Value != 2 || v.Insight.Headline != "1 to push · 1 to save" || v.Insight.Body != "Hot Holly · Stalled Sam" || v.Insight.Frac != 0.4 {
		t.Fatalf("insight = %+v", v.Insight)
	}
}

func TestVolumeEmptyIsHonest(t *testing.T) {
	e := VolumeOf(nil, 0, time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), 30)
	if e.RevenueUSD != 0 || !reflect.DeepEqual(e.Chips, []Chip{{Tone: "ok", Text: "0 closed"}}) {
		t.Fatalf("empty = %+v", e)
	}
	for _, m := range e.Meters {
		if m.Frac != nil || m.Display != "no leads" {
			t.Fatalf("an empty hand-off is unknown, not 0: %+v", m)
		}
	}
	if e.Foot != "each bar is a stage over the one before · no leads yet" || e.TouchesInWindow != 0 || e.Insight.Value != 0 || e.Insight.Headline != "Nothing waiting on you" || e.Insight.Body != "Every lead is fresh or already closed." {
		t.Fatalf("empty = %+v", e)
	}
}

func TestLastMessageFor(t *testing.T) {
	items := []email.CommsItem{
		{Source: "email", Title: "Re: proposal", Sender: "Pat <x>", ReplyTo: "Pat@Acme.com", Preview: "old", TS: "2026-09-01T00:00:00Z"},
		{Source: "whatsapp", Title: "Samantha Reyes", Sender: "Samantha Reyes", Preview: "new", TS: "2026-09-20T00:00:00Z"},
		{Source: "email", Title: "hello from samantha reyes", Sender: "s@r.co", Preview: "older", TS: "2026-09-10T00:00:00Z"},
	}
	if got := LastMessageFor("Pat", sp("pat@acme.com"), items); got == nil || got.Preview != "old" {
		t.Fatalf("email match = %+v", got)
	}
	if got := LastMessageFor("Samantha Reyes", nil, items); got == nil || got.Preview != "new" {
		t.Fatalf("name match picks newest = %+v", got)
	}
	if got := LastMessageFor("Sam", nil, items); got != nil {
		t.Fatal("short names never fuzzy-match")
	}
	if got := LastMessageFor("Nobody Here", sp("n@h.com"), items); got != nil {
		t.Fatal("no match is nil")
	}
}

// Ported from FounderOS v1 tests/funnel-volume.test.ts "funnelVolume windows".
func TestVolumeWindowsCoverTheCohortThatEntered(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	old := vj("old-win", "converted", [][3]string{{"first_touch", "organic", "2026-07-20"}, {"engaged", "call", "2026-07-25"}, {"nurtured", "call", "2026-07-28"}, {"converted", "checkout", "2026-08-01"}}, func(j *Journey) { j.AmountUSD = f64(900) })
	w := VolumeWindows(append(volumeFixture(), old), 2, now)
	if !reflect.DeepEqual(VolumeWindowDays, []int{30, 60, 90}) || len(w) != 3 {
		t.Fatalf("windows = %v", VolumeWindowDays)
	}
	if w["30"].Leads != 5 || w["30"].RevenueUSD != 4500 || w["30"].Meters[0].Label != "First touch → Engaged (4/5)" || *w["30"].Window != 30 {
		t.Fatalf("30d = %+v", w["30"])
	}
	if w["60"].Leads != 5 || w["90"].Leads != 6 || w["90"].RevenueUSD != 5400 {
		t.Fatalf("60/90 = %d %d %v", w["60"].Leads, w["90"].Leads, w["90"].RevenueUSD)
	}
	if w["30"].Caption != "closed revenue from 5 leads that entered in the last 30d · 3 organic / 2 ads" {
		t.Fatalf("caption = %q", w["30"].Caption)
	}
	for _, c := range w["30"].Chips {
		if strings.Contains(c.Text, "archived") {
			t.Fatal("archived leads are in no entry window")
		}
	}
	if len(w["30"].Series) != 30 || len(w["60"].Series) != 60 || len(w["90"].Series) != 90 || w["90"].TouchesInWindow != 18 {
		t.Fatalf("series = %d %d %d, 90d touches %d", len(w["30"].Series), len(w["60"].Series), len(w["90"].Series), w["90"].TouchesInWindow)
	}
	// needs-you is never windowed: a stale lead still needs you
	if w["30"].Insight != w["90"].Insight || w["30"].Insight.Value != 2 {
		t.Fatalf("insight = %+v / %+v", w["30"].Insight, w["90"].Insight)
	}
}

func TestVolumeWindowWithNoEntriesSaysSo(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	old := vj("old-win", "converted", [][3]string{{"first_touch", "organic", "2026-07-20"}}, nil)
	e := VolumeWindows([]Journey{old}, 0, now)
	if e["30"].Leads != 0 || e["30"].Caption != "no leads entered in the last 30 days" || e["90"].Leads != 1 {
		t.Fatalf("empty window = %+v", e["30"])
	}
	if all := VolumeOf(volumeFixture(), 2, now, 30); all.Window != nil || all.Leads != 5 {
		t.Fatalf("the all-time volume has no window: %+v", all)
	}
}
