package gcal

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/teambition/rrule-go"
)

const isoMillis = "2006-01-02T15:04:05.000Z"

func iso(t time.Time) string { return t.UTC().Format(isoMillis) }

func dayKey(t time.Time) string { return t.UTC().Format("2006-01-02") }

// CalEvent mirrors FounderOS v1's CalEvent. Nullable fields are pointers so the
// JSON reads null, exactly as the TS does.
type CalEvent struct {
	ID       string  `json:"id"`
	Account  string  `json:"account"`
	Color    string  `json:"color"`
	Title    string  `json:"title"`
	Start    string  `json:"start"`
	End      *string `json:"end"`
	AllDay   bool    `json:"allDay"`
	Location *string `json:"location"`
	JoinURL  *string `json:"joinUrl"`
	// Invitee emails (lowercased, mailto: stripped). Present only when someone
	// was invited: this is what joins a booked call to a lead.
	Attendees []string `json:"attendees,omitempty"`
	// iCal CREATED: when the call was booked, not when it happens.
	CreatedAt string `json:"createdAt,omitempty"`
}

// JoinFields are the event fields a video-call link can hide in.
type JoinFields struct {
	Conference  string // X-GOOGLE-CONFERENCE
	Location    string
	Description string
	URL         string
}

var joinRE = regexp.MustCompile(`(?i)(https?://(?:[a-z0-9-]+\.)?(?:meet\.google\.com|zoom\.us|teams\.microsoft\.com|teams\.live\.com|whereby\.com|meet\.jit\.si)/[^\s"'<>]+)`)

var httpPrefix = regexp.MustCompile(`^https?://`)

// ExtractJoinURL pulls a Meet/Zoom/Teams/Whereby/Jitsi link, or "".
func ExtractJoinURL(f JoinFields) string {
	if httpPrefix.MatchString(f.Conference) {
		return f.Conference
	}
	for _, field := range []string{f.Location, f.Description, f.URL} {
		if m := joinRE.FindStringSubmatch(field); m != nil {
			return m[1]
		}
	}
	return ""
}

var textUnescape = strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)

func text(c *ical.Component, name string) string {
	p := c.Props.Get(name)
	if p == nil {
		return ""
	}
	return textUnescape.Replace(p.Value)
}

func raw(c *ical.Component, name string) string {
	if p := c.Props.Get(name); p != nil {
		return p.Value
	}
	return ""
}

func joinOf(c *ical.Component) string {
	return ExtractJoinURL(JoinFields{
		Conference:  raw(c, "X-GOOGLE-CONFERENCE"),
		Location:    text(c, ical.PropLocation),
		Description: text(c, ical.PropDescription),
		URL:         raw(c, ical.PropURL),
	})
}

// propTime parses a DATE or DATE-TIME property. Floating times and dates read
// in the server's zone, as node-ical does. A TZID Go cannot load (a Windows
// zone name from an Outlook invite) reads its wall time as UTC rather than
// dropping the event.
func propTime(p *ical.Prop) (t time.Time, allDay, ok bool) {
	if p == nil || p.Value == "" {
		return time.Time{}, false, false
	}
	allDay = p.ValueType() == ical.ValueDate || len(p.Value) == len("20060102")
	t, err := p.DateTime(time.Local)
	if err != nil {
		t, err = time.ParseInLocation("20060102T150405", p.Value, time.UTC)
		if err != nil {
			return time.Time{}, false, false
		}
	}
	return t, allDay, true
}

// AttendeeEmails returns the invitees' addresses, lowercased and deduped.
func attendeeEmails(c *ical.Component) []string {
	var out []string
	for _, p := range c.Props.Values(ical.PropAttendee) {
		v := strings.TrimSpace(p.Value)
		if len(v) >= 7 && strings.EqualFold(v[:7], "mailto:") {
			v = v[7:]
		}
		v = strings.ToLower(strings.TrimSpace(v))
		if strings.Contains(v, "@") && !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func exdateKeys(c *ical.Component) map[string]bool {
	keys := map[string]bool{}
	for _, p := range c.Props.Values(ical.PropExceptionDates) {
		for _, v := range strings.Split(p.Value, ",") {
			one := p
			one.Value = strings.TrimSpace(v)
			if t, _, ok := propTime(&one); ok {
				keys[dayKey(t)] = true
			}
		}
	}
	return keys
}

// decodeBlob parses one VCALENDAR. go-ical can panic on some malformed
// parameters; a bad blob from the network is skipped, never fatal.
func decodeBlob(s string) (cal *ical.Calendar, err error) {
	defer func() {
		if r := recover(); r != nil {
			cal, err = nil, fmt.Errorf("ical: %v", r)
		}
	}()
	return ical.NewDecoder(strings.NewReader(s)).Decode()
}

func build(a CalAccount, id, title string, start time.Time, end *time.Time, allDay bool, location, join string) CalEvent {
	ev := CalEvent{
		// namespaced by account: the same meeting can live on several calendars.
		ID:      a.User + ":" + id,
		Account: a.Name,
		Color:   a.Color,
		Title:   title,
		Start:   iso(start),
		AllDay:  allDay,
	}
	if ev.Title == "" {
		ev.Title = "(no title)"
	}
	if end != nil {
		s := iso(*end)
		ev.End = &s
	}
	if location != "" {
		ev.Location = &location
	}
	if join != "" {
		ev.JoinURL = &join
	}
	return ev
}

// EventsFromIcal parses iCal blobs into events within [start, end), expanding
// recurring series client-side (Google's legacy CalDAV ignores <expand>) and
// applying EXDATE and RECURRENCE-ID overrides by UTC day, as the TS does.
// Expanded instances carry no attendees: a recurring series is not a booking.
func EventsFromIcal(blobs []string, a CalAccount, winStart, winEnd time.Time) []CalEvent {
	var out []CalEvent
	for _, s := range blobs {
		cal, err := decodeBlob(s)
		if err != nil {
			continue
		}
		events := cal.Children
		recurringUIDs := map[string]bool{}
		overrides := map[string]map[string]*ical.Component{}
		for _, c := range events {
			if c.Name == ical.CompEvent && c.Props.Get(ical.PropRecurrenceID) == nil && c.Props.Get(ical.PropRecurrenceRule) != nil {
				recurringUIDs[raw(c, ical.PropUID)] = true
			}
		}
		for _, c := range events {
			rid := c.Props.Get(ical.PropRecurrenceID)
			if c.Name != ical.CompEvent || rid == nil || !recurringUIDs[raw(c, ical.PropUID)] {
				continue
			}
			if t, _, ok := propTime(rid); ok {
				uid := raw(c, ical.PropUID)
				if overrides[uid] == nil {
					overrides[uid] = map[string]*ical.Component{}
				}
				overrides[uid][dayKey(t)] = c
			}
		}

		for _, c := range events {
			if c.Name != ical.CompEvent {
				continue
			}
			uid := raw(c, ical.PropUID)
			if c.Props.Get(ical.PropRecurrenceID) != nil && recurringUIDs[uid] {
				continue // applied through its parent
			}
			start, allDay, ok := propTime(c.Props.Get(ical.PropDateTimeStart))
			if !ok {
				continue
			}
			var end *time.Time
			if e, _, ok := propTime(c.Props.Get(ical.PropDateTimeEnd)); ok {
				end = &e
			}
			var duration time.Duration
			if end != nil {
				duration = end.Sub(start)
			}
			title := text(c, ical.PropSummary)
			location := text(c, ical.PropLocation)
			join := joinOf(c)

			if rr := c.Props.Get(ical.PropRecurrenceRule); rr != nil {
				opt, err := rrule.StrToROptionInLocation(rr.Value, start.Location())
				if err != nil {
					continue
				}
				opt.Dtstart = start
				rule, err := rrule.NewRRule(*opt)
				if err != nil {
					continue
				}
				ex := exdateKeys(c)
				for _, d := range rule.Between(winStart, winEnd, true) {
					key := dayKey(d)
					if ex[key] {
						continue
					}
					id := uid + "-" + key
					if ov := overrides[uid][key]; ov != nil {
						if ovStart, _, ok := propTime(ov.Props.Get(ical.PropDateTimeStart)); ok {
							oe := ovStart.Add(duration)
							if e, _, ok := propTime(ov.Props.Get(ical.PropDateTimeEnd)); ok {
								oe = e
							}
							out = append(out, build(a, id, firstNonEmpty(text(ov, ical.PropSummary), title), ovStart, &oe, allDay,
								firstNonEmpty(text(ov, ical.PropLocation), location), firstNonEmpty(joinOf(ov), join)))
							continue
						}
					}
					de := d.Add(duration)
					out = append(out, build(a, id, title, d, &de, allDay, location, join))
				}
				continue
			}

			evEnd := start
			if end != nil {
				evEnd = *end
			}
			if !start.Before(winEnd) || evEnd.Before(winStart) {
				continue
			}
			if uid == "" {
				uid = fmt.Sprintf("event-%d", len(out))
			}
			ev := build(a, uid, title, start, end, allDay, location, join)
			if att := attendeeEmails(c); len(att) > 0 {
				ev.Attendees = att
			}
			if created, _, ok := propTime(c.Props.Get(ical.PropCreated)); ok {
				ev.CreatedAt = iso(created)
			}
			out = append(out, ev)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// MergeUpcoming flattens per-account events, sorts by start, drops the same
// meeting shared across calendars (same title + start) and caps at limit.
func MergeUpcoming(perAccount [][]CalEvent, limit int) []CalEvent {
	var all []CalEvent
	for _, evs := range perAccount {
		all = append(all, evs...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Start < all[j].Start })
	seen := map[string]bool{}
	out := []CalEvent{}
	for _, ev := range all {
		key := ev.Title + "|" + ev.Start
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ev)
	}
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
