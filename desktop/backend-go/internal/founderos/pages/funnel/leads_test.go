package funnel

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// Ported from FounderOS v1 tests/funnel-leads.test.ts.

func tfLead(mod func(*typeform.Lead)) typeform.Lead {
	l := typeform.Lead{ResponseID: "resp1", FormID: "frmAA", FormTitle: "Launchpad Cohort application", Name: "Dana Reyes",
		Email: "dana@reyes.co", Phone: "+15125550100", SubmittedAt: "2026-09-21T14:03:00Z", Hidden: map[string]string{}}
	if mod != nil {
		mod(&l)
	}
	return l
}

func booking(mod func(*gcal.CalEvent)) gcal.CalEvent {
	e := gcal.CalEvent{ID: "b1", Account: "LC", Color: "#000", Title: "Strategy call (Dana Reyes)", Start: "2026-09-25T15:00:00.000Z",
		Attendees: []string{"founderos@launchpadcohort.ai", "dana@reyes.co"}, CreatedAt: "2026-09-21T14:20:00.000Z"}
	if mod != nil {
		mod(&e)
	}
	return e
}

func call(mod func(*fathomcalls.Call)) fathomcalls.Call {
	d := 42.0
	c := fathomcalls.Call{RecordingID: "901", Title: "Strategy call (Dana Reyes)", URL: sp("https://fathom.video/calls/901"), At: "2026-09-25T15:02:00Z",
		DurationMinutes: &d, Invitees: []fathomcalls.Invitee{
			{Name: sp("the operator"), Email: sp("founderos@launchpadcohort.ai")},
			{Name: sp("Dana Reyes"), Email: sp("dana@reyes.co"), External: true},
		}}
	if mod != nil {
		mod(&c)
	}
	return c
}

func TestTypeformSubmissionOpensAJourney(t *testing.T) {
	j := TypeformJourneys([]typeform.Lead{tfLead(nil)})[0]
	if j.ID != "typeform-resp1" || j.Status != "first_touch" || *j.Email != "dana@reyes.co" || *j.Phone != "+15125550100" || j.Venture != "launchpad-cohort" || j.Likelihood != 40 || j.Relationship != "warm" {
		t.Fatalf("journey = %+v", j)
	}
	if len(j.Touches) != 2 || j.Touches[0].Source != "typeform" || j.Touches[1].At != "2026-09-21" || !strings.Contains(j.Touches[1].Label, "Launchpad Cohort application") {
		t.Fatalf("touches = %+v", j.Touches)
	}
	if AcquisitionFor(j) != "form" {
		t.Fatal("an untagged form lands on Forms")
	}
	tagged := TypeformJourneys([]typeform.Lead{tfLead(func(l *typeform.Lead) { l.Hidden = map[string]string{"utm_source": "instagram", "utm_medium": "bio"} })})[0]
	if AcquisitionFor(tagged) != "instagram" || !strings.Contains(tagged.Touches[0].Label, "via: instagram, bio") {
		t.Fatalf("tagged = %+v", tagged.Touches[0])
	}
}

func TestOnePersonTwiceIsOneJourney(t *testing.T) {
	js := TypeformJourneys([]typeform.Lead{
		tfLead(func(l *typeform.Lead) {
			l.ResponseID, l.SubmittedAt, l.FormTitle = "r2", "2026-09-22T10:00:00Z", "Follow-up survey"
		}),
		tfLead(func(l *typeform.Lead) { l.ResponseID, l.SubmittedAt = "r1", "2026-09-10T10:00:00Z" }),
	})
	if len(js) != 1 || js[0].ID != "typeform-r1" {
		t.Fatalf("journeys = %+v", js)
	}
	var ats []string
	for _, tc := range js[0].Touches {
		if strings.HasPrefix(tc.Label, "Submitted") {
			ats = append(ats, tc.At)
		}
	}
	if !reflect.DeepEqual(ats, []string{"2026-09-10", "2026-09-22"}) {
		t.Fatalf("ats = %v", ats)
	}
	if j := TypeformJourneys([]typeform.Lead{tfLead(func(l *typeform.Lead) { l.Name = "" })})[0]; j.Name != "dana@reyes.co" || j.Person != nil {
		t.Fatalf("nameless = %+v", j)
	}
}

func TestVentureFor(t *testing.T) {
	if VentureFor("Vantage discovery") != "vantage" || VentureFor("Launchpad Cohort application") != "launchpad-cohort" {
		t.Fatal("venture heuristic")
	}
}

func TestMergeCallTouches(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	base := func() []Journey { return TypeformJourneys([]typeform.Lead{tfLead(nil)}) }

	j := MergeCallTouches(base(), []gcal.CalEvent{booking(nil)}, nil, now)[0]
	last := j.Touches[len(j.Touches)-1]
	if last.Stage != "engaged" || last.Channel != "call" || last.Source != "calendar" || last.At != "2026-09-21" || !strings.Contains(last.Label, "Strategy call") || j.Status != "engaged" {
		t.Fatalf("booked = %+v / %s", last, j.Status)
	}

	j = MergeCallTouches(base(), []gcal.CalEvent{booking(func(e *gcal.CalEvent) { e.CreatedAt, e.Start = "", "2026-10-05T15:00:00.000Z" })}, nil, now)[0]
	if j.Touches[len(j.Touches)-1].At != "2026-09-30" {
		t.Fatal("a booking without CREATED is dated no later than today")
	}

	js := MergeCallTouches(base(), []gcal.CalEvent{booking(func(e *gcal.CalEvent) { e.Attendees = []string{"someone@else.com"} })}, nil, now)
	if len(js) != 1 || len(js[0].Touches) != 2 {
		t.Fatal("the calendar is not a lead source")
	}

	j = MergeCallTouches(base(), []gcal.CalEvent{booking(nil)}, []fathomcalls.Call{call(nil)}, now)[0]
	held := j.Touches[len(j.Touches)-1]
	seqs := []int{}
	for _, tc := range j.Touches {
		seqs = append(seqs, tc.Seq)
	}
	if held.Stage != "nurtured" || held.Source != "fathom" || held.At != "2026-09-25" || !strings.Contains(held.Label, "42 min") || j.Status != "nurtured" || *j.URL != "https://fathom.video/calls/901" || !reflect.DeepEqual(seqs, []int{1, 2, 3, 4}) || j.Likelihood != 60 {
		t.Fatalf("held = %+v / %+v", held, j)
	}

	orphan := MergeCallTouches(nil, nil, []fathomcalls.Call{call(func(c *fathomcalls.Call) {
		c.Title = "Vantage x Acme discovery"
		c.Invitees = []fathomcalls.Invitee{{Name: sp("Pat Lee"), Email: sp("pat@acme.com"), External: true}}
	})}, now)
	if len(orphan) != 1 || orphan[0].ID != "fathom-901" || orphan[0].Name != "Pat Lee" || orphan[0].Venture != "vantage" || orphan[0].Status != "nurtured" || orphan[0].Touches[0].Source != "fathom" {
		t.Fatalf("orphan = %+v", orphan)
	}

	internal := call(func(c *fathomcalls.Call) {
		c.Invitees = []fathomcalls.Invitee{{Name: sp("the operator"), Email: sp("b@aa.ai")}}
	})
	group := call(func(c *fathomcalls.Call) {
		c.RecordingID = "902"
		c.Invitees = nil
		for i := 0; i < 6; i++ {
			c.Invitees = append(c.Invitees, fathomcalls.Invitee{Email: sp(string(rune('a'+i)) + "@x.com"), External: true})
		}
	})
	if got := MergeCallTouches(nil, nil, []fathomcalls.Call{internal, group}, now); len(got) != 0 {
		t.Fatalf("internal/group calls are not leads: %+v", got)
	}

	paid := base()
	paid[0].Status = "converted"
	if MergeCallTouches(paid, nil, []fathomcalls.Call{call(nil)}, now)[0].Status != "converted" {
		t.Fatal("converted stays converted")
	}
}
