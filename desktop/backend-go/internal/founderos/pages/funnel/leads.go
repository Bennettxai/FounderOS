package funnel

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// The live lead lane (lib/funnel-leads.ts): Typeform submission opens a
// journey, a calendar booking moves it to Call booked, a Fathom recording to
// Call transcribed. Joins are by email; venture is a name heuristic.

var vantageRe = regexp.MustCompile(`(?i)vantage`)

// VentureFor: Vantage when the form or call names it, else Launchpad Cohort.
func VentureFor(text string) string {
	if vantageRe.MatchString(text) {
		return "vantage"
	}
	return "launchpad-cohort"
}

func day(iso string) string {
	if len(iso) > 10 {
		return iso[:10]
	}
	return iso
}

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// maxExternalInvitees: a group call (office hours, a webinar) is not one lead.
const maxExternalInvitees = 3

var stageOrder = map[string]int{"first_touch": 0, "engaged": 1, "nurtured": 2, "opted_in": 3, "converted": 4}

func furthest(a, b string) string {
	if stageOrder[a] >= stageOrder[b] {
		return a
	}
	return b
}

func viaOf(l typeform.Lead) []string {
	var out []string
	for _, k := range []string{"utm_source", "utm_medium", "utm_campaign", "source"} {
		if v := l.Hidden[k]; v != "" {
			out = append(out, v)
		}
	}
	return out
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// TypeformJourneys: one journey per person (by email, else per response).
func TypeformJourneys(leads []typeform.Lead) []Journey {
	var order []string
	groups := map[string][]typeform.Lead{}
	for _, l := range leads {
		key := "resp:" + l.ResponseID
		if l.Email != "" {
			key = "email:" + norm(l.Email)
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], l)
	}
	out := []Journey{}
	for _, key := range order {
		group := groups[key]
		sort.SliceStable(group, func(a, b int) bool { return group[a].SubmittedAt < group[b].SubmittedAt })
		first := group[0]
		id := "typeform-" + first.ResponseID
		via := viaOf(first)
		label := "Lead via Typeform"
		acq := "form"
		if len(via) > 0 {
			label += " · via: " + strings.Join(via, ", ")
			if m := MatchAcquisition(strings.Join(via, " ")); m != "" {
				acq = m
			}
		}
		touches := []Touch{{ID: id + "-t1", ContactID: id, Seq: 1, Stage: "first_touch", Channel: "crm", Label: label, Source: "typeform", At: day(first.SubmittedAt), Acquisition: acq}}
		var titles []string
		var person, phone string
		for i, l := range group {
			touches = append(touches, Touch{ID: fmt.Sprintf("%s-t%d", id, i+2), ContactID: id, Seq: i + 2, Stage: "first_touch", Channel: "crm",
				Label: "Submitted \"" + l.FormTitle + "\"", Source: "typeform", At: day(l.SubmittedAt)})
			titles = append(titles, l.FormTitle)
			if person == "" {
				person = l.Name
			}
			if phone == "" {
				phone = l.Phone
			}
		}
		name := person
		for _, alt := range []string{first.Email, first.Phone, "Typeform lead"} {
			if name == "" {
				name = alt
			}
		}
		if len(day(first.SubmittedAt)) != 10 {
			continue // a malformed lead is skipped, never fatal
		}
		out = append(out, Journey{
			ID: id, Name: name, Venture: VentureFor(strings.Join(titles, " ")), Status: "first_touch",
			Relationship: "warm", Likelihood: 40, Email: nonEmpty(first.Email), Phone: nonEmpty(phone),
			Person: nonEmpty(person), CreatedAt: day(first.SubmittedAt), Touches: touches,
		})
	}
	return out
}

type callTouch struct {
	stage, channel, label, source, at string
	url                               *string
}

func bookingTouch(ev gcal.CalEvent, now time.Time) callTouch {
	booked := ev.CreatedAt
	if booked == "" {
		booked = ev.Start
		if st, err := time.Parse(time.RFC3339Nano, ev.Start); err == nil && st.After(now) {
			booked = now.UTC().Format("2006-01-02T15:04:05.000Z")
		}
	}
	return callTouch{stage: "engaged", channel: "call", label: fmt.Sprintf("Call booked: %s (%s)", ev.Title, day(ev.Start)), source: "calendar", at: day(booked)}
}

func heldTouch(c fathomcalls.Call) callTouch {
	label := "Call held: " + c.Title
	if c.DurationMinutes != nil && *c.DurationMinutes != 0 {
		label += " · " + strconv.FormatFloat(*c.DurationMinutes, 'f', -1, 64) + " min"
	}
	return callTouch{stage: "nurtured", channel: "call", label: label, source: "fathom", at: day(c.At), url: c.URL}
}

func withTouches(j Journey, extra []callTouch) Journey {
	if len(extra) == 0 {
		return j
	}
	sorted := append([]callTouch(nil), extra...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].at < sorted[b].at })
	n := len(j.Touches)
	touches := append([]Touch(nil), j.Touches...)
	status := j.Status
	var recording *string
	for i, t := range sorted {
		touches = append(touches, Touch{ID: fmt.Sprintf("%s-%s-%d", j.ID, t.source, n+i+1), ContactID: j.ID, Seq: n + i + 1,
			Stage: t.stage, Channel: t.channel, Label: t.label, Source: t.source, At: t.at})
		status = furthest(status, t.stage)
		if recording == nil && t.url != nil {
			recording = t.url
		}
	}
	j.Touches = touches
	j.Status = status
	if j.Relationship == "cold" {
		j.Relationship = "warm"
	}
	if j.Likelihood < 60 {
		j.Likelihood = 60 // a booked or held call is the strongest pre-payment signal
	}
	if j.URL == nil {
		j.URL = recording
	}
	return j
}

// MergeCallTouches folds bookings and held calls onto the journeys whose
// email is on the invite. Bookings that invite no lead add nothing; held
// calls that match no lead open their own journey when the invite is small.
func MergeCallTouches(journeys []Journey, bookings []gcal.CalEvent, calls []fathomcalls.Call, now time.Time) []Journey {
	byEmail := map[string]string{}
	for _, j := range journeys {
		if j.Email != nil {
			if _, ok := byEmail[norm(*j.Email)]; !ok {
				byEmail[norm(*j.Email)] = j.ID
			}
		}
	}
	extra := map[string][]callTouch{}
	for _, ev := range bookings {
		for _, e := range ev.Attendees {
			if id, ok := byEmail[norm(e)]; ok {
				extra[id] = append(extra[id], bookingTouch(ev, now))
				break
			}
		}
	}
	var orphans []Journey
	for _, c := range calls {
		matched := ""
		for _, inv := range c.Invitees {
			if inv.Email != nil {
				if id, ok := byEmail[*inv.Email]; ok {
					matched = id
					break
				}
			}
		}
		if matched != "" {
			extra[matched] = append(extra[matched], heldTouch(c))
			continue
		}
		var external []fathomcalls.Invitee
		for _, inv := range c.Invitees {
			if inv.External && inv.Email != nil {
				external = append(external, inv)
			}
		}
		if len(external) == 0 || len(external) > maxExternalInvitees {
			continue
		}
		who := external[0]
		jid := "fathom-" + c.RecordingID
		t := heldTouch(c)
		name := *who.Email
		if who.Name != nil {
			name = *who.Name
		}
		orphans = append(orphans, Journey{
			ID: jid, Name: name, Venture: VentureFor(c.Title), Status: "nurtured", Relationship: "warm", Likelihood: 60,
			URL: c.URL, Email: who.Email, Person: who.Name, CreatedAt: t.at,
			Touches: []Touch{{ID: jid + "-t1", ContactID: jid, Seq: 1, Stage: t.stage, Channel: t.channel, Label: t.label, Source: t.source, At: t.at}},
		})
		byEmail[*who.Email] = jid // a second call with them lands on this journey
	}
	all := append(append([]Journey{}, journeys...), orphans...)
	out := make([]Journey, 0, len(all))
	for _, j := range all {
		out = append(out, withTouches(j, extra[j.ID]))
	}
	return out
}
