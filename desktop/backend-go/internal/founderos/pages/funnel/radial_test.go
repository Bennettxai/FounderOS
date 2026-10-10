package funnel

import (
	"reflect"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/funnel-radial.test.ts.

func tch(label, channel string) Touch {
	return Touch{ID: "ft-test", ContactID: "fc-test", Seq: 1, Stage: "first_touch", Channel: channel, Label: label, Source: "trakyo", At: "2026-06-01"}
}

func jr(id string, touches ...Touch) Journey {
	return Journey{ID: id, Name: "Test Client", Venture: "vantage", Status: "engaged", Relationship: "warm", Likelihood: 50, CreatedAt: "2026-06-01", Touches: touches}
}

func TestAcquisitionsAreSixInCanonicalOrder(t *testing.T) {
	got := []string{}
	for _, a := range Acquisitions {
		got = append(got, a.ID)
	}
	if !reflect.DeepEqual(got, []string{"instagram", "youtube", "newsletter", "x_linkedin", "form", "word_of_mouth"}) {
		t.Fatalf("acquisitions = %v", got)
	}
}

func TestAcquisitionFor(t *testing.T) {
	stamped := tch("How the operator console works", "organic")
	stamped.Acquisition = "youtube"
	if AcquisitionFor(jr("a", stamped)) != "youtube" {
		t.Fatal("the stamp wins")
	}
	cases := map[string][2]string{
		`IG reel: "3 AI offers that close themselves"`:      {"organic", "instagram"},
		`TikTok: "day in the life running an AI agency"`:    {"organic", "instagram"},
		`Meta ad: "stop selling hours" (cold traffic)`:      {"ads", "instagram"},
		`ManyChat keyword "SCALE" → DM flow`:                {"dm", "instagram"},
		`YT long-form: "how I'd start an agency in 2026"`:   {"organic", "youtube"},
		`Newsletter welcome: pricing-psychology issue`:      {"email", "newsletter"},
		`X thread: client-onboarding agent breakdown`:       {"organic", "x_linkedin"},
		`Twitter reply guy turned lead`:                     {"organic", "x_linkedin"},
		`LinkedIn post: legal-intake automation teardown`:   {"organic", "x_linkedin"},
		`Case study: 3x pipeline in 60 days`:                {"organic", "word_of_mouth"},
		`Website form: enterprise intake`:                   {"organic", "form"},
		`Application submitted from landing page`:           {"organic", "form"},
		`Cold-traffic campaign #4`:                          {"ads", "instagram"},
		`Opportunity created in GHL`:                        {"crm", "word_of_mouth"},
		`Opportunity created in GHL · source: Instagram DM`: {"crm", "instagram"},
	}
	for label, c := range cases {
		if got := AcquisitionFor(jr("a", tch(label, c[0]))); got != c[1] {
			t.Errorf("%q = %s, want %s", label, got, c[1])
		}
	}
	if AcquisitionFor(jr("a")) != "word_of_mouth" {
		t.Fatal("no touches is word of mouth")
	}
}

func TestRadialModel(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	yt := tch("YT long-form: agency", "organic")
	yt.At = "2026-06-20"
	paid := Touch{ID: "t2", Seq: 2, Stage: "converted", Channel: "checkout", Label: "Paid in full", Source: "stripe", At: "2026-07-01"}
	b := jr("b", yt, paid)
	b.Status = "converted"
	m := RadialModel([]Journey{jr("a", tch("IG reel: offer", "organic")), b, jr("c", tch("Opportunity created in GHL", "crm"))}, now)
	if len(m.Segments) != 6 {
		t.Fatalf("segments = %v", m.Segments)
	}
	want := map[string][2]int{"youtube": {1, 1}, "instagram": {1, 0}, "word_of_mouth": {1, 0}}
	for _, s := range m.Segments {
		if w, ok := want[s.ID]; ok && (s.Count != w[0] || s.Converted != w[1]) {
			t.Errorf("%s = %+v", s.ID, s)
		}
	}

	x := tch("X thread: breakdown", "organic")
	dm := Touch{ID: "t2", Seq: 2, Stage: "engaged", Channel: "dm", Label: "DM convo", Source: "manual", At: "2026-06-10"}
	pd := Touch{ID: "t3", Seq: 3, Stage: "converted", Channel: "checkout", Label: "Paid", Source: "stripe", At: "2026-06-20"}
	won := jr("won", x, dm, pd)
	won.Status = "converted"
	n := RadialModel([]Journey{won}, now).Nodes[0]
	if n.Segment != 3 || !reflect.DeepEqual(n.Rings, []int{0, 1, 4}) || n.CurrentRing != 4 || n.State != "converted" {
		t.Fatalf("node = %+v", n)
	}
	if n.Origin.Segment != "X / LinkedIn" || n.Origin.Entry == nil || *n.Origin.Entry != "X thread: breakdown" {
		t.Fatalf("origin = %+v", n.Origin)
	}

	empty := RadialModel(nil, now)
	if len(empty.Nodes) != 0 || empty.Nodes == nil || len(empty.Segments) != 6 {
		t.Fatalf("empty = %+v", empty)
	}
}

func TestOriginOf(t *testing.T) {
	o := OriginOf(jr("a", tch("IG reel: 3 offers", "organic")))
	if o.Segment != "Instagram" || *o.Entry != "IG reel: 3 offers" || *o.Source != "trakyo" || *o.At != "2026-06-01" || *o.Channel != "organic" {
		t.Fatalf("origin = %+v", o)
	}
	if o := OriginOf(jr("a")); o.Segment != "Word of mouth" || o.Entry != nil {
		t.Fatalf("origin = %+v", o)
	}
}
