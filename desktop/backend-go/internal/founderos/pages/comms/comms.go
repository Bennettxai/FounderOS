// Package comms is the pure logic behind the bridge's /os/comms page, ported
// from FounderOS v1: the unified feed and contact priorities (lib/comms.ts), the
// per-source lanes (lib/comms-lanes.ts), the Brand Deals-style numbers
// (lib/comms-volume.ts), the Slack client board (lib/slack-clients.ts) and the
// Recordings tab (lib/recordings.ts). Nothing here calls a connector: the API
// layer gathers and this package shapes, so every rule is unit-tested.
package comms

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// ---- lib/comms.ts -------------------------------------------------------------

// Item is CommsItem: one message from any source.
type Item struct {
	Source   string `json:"source"`
	Title    string `json:"title"`
	Preview  string `json:"preview"`
	TS       string `json:"ts"`
	Unread   int    `json:"unread"`
	Sender   string `json:"sender,omitempty"`
	ReplyTo  string `json:"replyTo,omitempty"`
	Account  string `json:"account,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

// ContactTag is a founderos_contact_tags row.
type ContactTag struct {
	Person  string `json:"person"`
	Channel string `json:"channel"`
	Tag     string `json:"tag"`
	Tier    int    `json:"tier"`
}

// parseTS is Date.parse for the ISO stamps the connectors emit.
func parseTS(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// MergeFeed drops unparseable stamps, sorts newest first and keeps limit.
func MergeFeed(items []Item, limit int) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if _, ok := parseTS(it.TS); ok {
			out = append(out, it)
		}
	}
	slices.SortStableFunc(out, func(a, b Item) int {
		ta, _ := parseTS(a.TS)
		tb, _ := parseTS(b.TS)
		return tb.Compare(ta)
	})
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ContactPriority is contactPriority: 1 red, 2 yellow, 3 green, 0 when no tag
// matches. A tag matches when the person equals the text or, for names of 5+
// characters, appears inside it; the lowest tier wins.
func ContactPriority(text string, tags []ContactTag) int {
	hay := strings.ToLower(strings.TrimSpace(text))
	best := 0
	for _, tag := range tags {
		person := strings.ToLower(strings.TrimSpace(tag.Person))
		hit := person == hay || (len([]rune(person)) >= 5 && strings.Contains(hay, person))
		if hit && (best == 0 || tag.Tier < best) {
			best = tag.Tier
		}
	}
	return best
}

// ---- lib/comms-lanes.ts ---------------------------------------------------------

// Inbox is one configured inbox slot.
type Inbox struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LaneItem is CommsLaneItem.
type LaneItem struct {
	ID       string `json:"id"`
	Sender   string `json:"sender"`
	Preview  string `json:"preview"`
	TS       string `json:"ts"`
	Unread   int    `json:"unread"`
	Priority int    `json:"priority,omitempty"`
	ReplyTo  string `json:"replyTo,omitempty"`
}

// Lane is CommsLane: one inbox, or WhatsApp, with its connector's own state.
type Lane struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Source string           `json:"source"`
	State  connectors.State `json:"state"`
	Detail string           `json:"detail"`
	Items  []LaneItem       `json:"items"`
	Unread int              `json:"unread"`
}

func senderOf(it Item) string {
	if it.Sender != "" {
		return it.Sender
	}
	return it.Title
}

func makeLane(id, name, source string, st connectors.Status, items []LaneItem, tags []ContactTag) Lane {
	for i := range items {
		if len(tags) > 0 {
			items[i].Priority = ContactPriority(items[i].Sender, tags)
		}
	}
	slices.SortStableFunc(items, func(a, b LaneItem) int {
		ta, oka := parseTS(a.TS)
		tb, okb := parseTS(b.TS)
		switch {
		case oka && okb:
			return tb.Compare(ta)
		case oka:
			return -1
		case okb:
			return 1
		}
		return 0
	})
	unread := 0
	for _, it := range items {
		unread += it.Unread
	}
	return Lane{ID: id, Name: name, Source: source, State: st.State, Detail: st.Detail, Items: items, Unread: unread}
}

// BuildLanes is buildCommsLanes: one lane per configured inbox in slot order
// (all sharing the email connector's status), then WhatsApp.
func BuildLanes(inboxes []Inbox, emails []Item, emailState connectors.Status, whatsapp []Item, whatsappState connectors.Status, tags []ContactTag) []Lane {
	lanes := make([]Lane, 0, len(inboxes)+1)
	for _, box := range inboxes {
		items := []LaneItem{}
		i := 0
		for _, e := range emails {
			if e.Account != box.ID {
				continue
			}
			items = append(items, LaneItem{ID: box.ID + ":" + e.TS + ":" + strconv.Itoa(i), Sender: senderOf(e), Preview: e.Preview, TS: e.TS, Unread: e.Unread, ReplyTo: e.ReplyTo})
			i++
		}
		lanes = append(lanes, makeLane(box.ID, box.Name, "email", emailState, items, tags))
	}
	wa := make([]LaneItem, 0, len(whatsapp))
	for i, w := range whatsapp {
		wa = append(wa, LaneItem{ID: "whatsapp:" + w.TS + ":" + strconv.Itoa(i), Sender: senderOf(w), Preview: w.Preview, TS: w.TS, Unread: w.Unread})
	}
	return append(lanes, makeLane("whatsapp", "WhatsApp", "whatsapp", whatsappState, wa, tags))
}

// ---- lib/slack-clients.ts -------------------------------------------------------

// RosterClient is the slice of a paying-customer roster row the board needs.
type RosterClient struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// SlackCard is SlackClientCard. Channel, LastTS and Heat are null when no
// Slack thread matched: nothing is invented.
type SlackCard struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Channel  *string `json:"channel"`
	LastText string  `json:"lastText"`
	LastTS   *string `json:"lastTs"`
	Unread   int     `json:"unread"`
	Heat     *string `json:"heat"`
	Waiting  string  `json:"waiting"`
	Live     bool    `json:"live"`
}

var currentStatus = regexp.MustCompile(`(?i)^(closed won|won|active|onboarding|delivering|retainer)$`)
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func IsCurrentClient(c RosterClient) bool {
	return currentStatus.MatchString(strings.TrimSpace(c.Status))
}

func normKey(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

func slackSeconds(ts string) float64 {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return 0
	}
	return f
}

// SlackClientBoard is slackClientBoard: one card per current client (plus any
// client with a real thread), live threads first and newest first, quiet
// clients after by name.
func SlackClientBoard(clients []RosterClient, messages []slack.Message, now time.Time) []SlackCard {
	sorted := slices.Clone(messages)
	slices.SortStableFunc(sorted, func(a, b slack.Message) int {
		fa, fb := slackSeconds(a.TS), slackSeconds(b.TS)
		switch {
		case fb > fa:
			return 1
		case fb < fa:
			return -1
		}
		return 0
	})
	cards := make([]SlackCard, 0, len(clients))
	for _, c := range clients {
		key := normKey(c.Name)
		var match *slack.Message
		if len(key) >= 3 {
			for i := range sorted {
				ch, usr := normKey(sorted[i].Channel), normKey(sorted[i].User)
				if strings.Contains(ch, key) || strings.Contains(key, ch) || strings.Contains(usr, key) {
					match = &sorted[i]
					break
				}
			}
		}
		if match == nil {
			if IsCurrentClient(c) {
				cards = append(cards, SlackCard{ID: c.ID, Name: c.Name, Waiting: "none"})
			}
			continue
		}
		last := now
		if sec := slackSeconds(match.TS); sec > 0 {
			last = time.UnixMilli(int64(sec * 1000)).UTC()
		}
		age := now.Sub(last)
		heat := "cold"
		if age < 24*time.Hour {
			heat = "hot"
		} else if age < 72*time.Hour {
			heat = "warm"
		}
		channel := "#" + strings.TrimPrefix(match.Channel, "#")
		stamp := last.UTC().Format("2006-01-02T15:04:05.000Z")
		cards = append(cards, SlackCard{ID: c.ID, Name: c.Name, Channel: &channel, LastText: match.Text, LastTS: &stamp, Heat: &heat, Waiting: "you", Live: true})
	}
	slices.SortStableFunc(cards, func(a, b SlackCard) int {
		if a.Live != b.Live {
			if a.Live {
				return -1
			}
			return 1
		}
		if a.Live {
			return strings.Compare(*b.LastTS, *a.LastTS)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return cards
}

// ---- lib/recordings.ts ----------------------------------------------------------

// Recording is one row of the Recordings tab.
type Recording struct {
	ID              string   `json:"id"`
	Source          string   `json:"source"` // plaud | fathom
	Title           string   `json:"title"`
	At              string   `json:"at"`
	DurationMinutes *float64 `json:"durationMinutes"`
	URL             *string  `json:"url"`
	// Brain is where the recording was filed once ingested (founderos_plaud_ingests.via).
	Brain *string `json:"brain,omitempty"`
}

// MergeRecordings interleaves Plaud and Fathom newest first; an undated row
// sorts to the bottom.
func MergeRecordings(p []plaud.Recording, fm []fathomcalls.Meeting) []Recording {
	rows := make([]Recording, 0, len(p)+len(fm))
	for _, r := range p {
		rows = append(rows, Recording{ID: "plaud-" + r.ID, Source: "plaud", Title: r.Title, At: r.At, DurationMinutes: r.DurationMinutes})
	}
	for _, m := range fm {
		id := "fathom-" + m.At + "-" + m.Title
		if m.URL != nil {
			id = "fathom-" + *m.URL
		}
		rows = append(rows, Recording{ID: id, Source: "fathom", Title: m.Title, At: m.At, DurationMinutes: m.DurationMinutes, URL: m.URL})
	}
	slices.SortStableFunc(rows, func(a, b Recording) int {
		ta, oka := parseTS(a.At)
		tb, okb := parseTS(b.At)
		switch {
		case oka && okb:
			return tb.Compare(ta)
		case oka:
			return -1
		case okb:
			return 1
		}
		return 0
	})
	return rows
}

// MarkIngested stamps where each ingested Plaud file landed (fileId → via).
func MarkIngested(rows []Recording, via map[string]string) []Recording {
	out := slices.Clone(rows)
	for i, r := range out {
		if v, ok := via[strings.TrimPrefix(r.ID, "plaud-")]; ok && r.Source == "plaud" {
			v := v
			out[i].Brain = &v
		}
	}
	return out
}

// ---- lib/comms-volume.ts ----------------------------------------------------------

type VolumeItem struct {
	Sender   string
	TS       string
	Unread   int
	Priority int
}

type VolumeLane struct {
	Name   string
	Source string
	Unread int
	Items  []VolumeItem
}

type VolumeCard struct {
	Name    string
	Unread  int
	Waiting string
}

type VolumeInput struct {
	Lanes      []VolumeLane
	SlackCards []VolumeCard
	Sources    []connectors.State
	Events     []string // meeting starts, ISO
	Recordings []string // recording stamps, ISO
	Now        time.Time
	Days       int // step-line window, default 14
}

type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

// Volume is CommsVolume, the page's Brand Deals-style numbers.
type Volume struct {
	Headline      int             `json:"headline"`
	Chips         []pagekit.Chip  `json:"chips"`
	Caption       string          `json:"caption"`
	Meta          string          `json:"meta"`
	Meters        []pagekit.Meter `json:"meters"`
	Foot          string          `json:"foot"`
	Series        []pagekit.Point `json:"series"`
	SeriesTotal   int             `json:"seriesTotal"`
	Meetings      []pagekit.Point `json:"meetings"`
	MeetingsTotal int             `json:"meetingsTotal"`
	Insight       Insight         `json:"insight"`
}

func localMidnight(t time.Time, offsetDays int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+offsetDays, 0, 0, 0, 0, t.Location())
}

// perDay counts stamps per local calendar day (the day the operator lived).
func perDay(stamps []string, start time.Time, days int, loc *time.Location, label func(time.Time) string) []pagekit.Point {
	counts := map[string]int{}
	key := func(t time.Time) string { return t.In(loc).Format("2006-01-02") }
	for _, s := range stamps {
		if t, ok := parseTS(s); ok {
			counts[key(t)]++
		}
	}
	out := make([]pagekit.Point, days)
	for i := range out {
		d := time.Date(start.Year(), start.Month(), start.Day()+i, 0, 0, 0, 0, loc)
		out[i] = pagekit.Point{Label: label(d), Count: float64(counts[key(d)])}
	}
	return out
}

// CommsVolume is commsVolume. Days are local calendar days in Now's zone.
func CommsVolume(x VolumeInput) Volume {
	now := x.Now
	if now.IsZero() {
		now = time.Now()
	}
	loc := now.Location()
	days := x.Days
	if days <= 0 {
		days = 14
	}
	emailLanes, emailUnread, waUnread := 0, 0, 0
	var items []VolumeItem
	for _, l := range x.Lanes {
		if l.Source == "email" {
			emailLanes++
			emailUnread += l.Unread
		} else if l.Source == "whatsapp" {
			waUnread += l.Unread
		}
		items = append(items, l.Items...)
	}
	headline := emailUnread + waUnread
	var urgent []VolumeItem
	priorityItems := 0
	for _, it := range items {
		if it.Priority == 1 {
			priorityItems++
			if it.Unread > 0 {
				urgent = append(urgent, it)
			}
		}
	}
	slackUnread := 0
	var waiting []VolumeCard
	for _, c := range x.SlackCards {
		slackUnread += c.Unread
		if c.Waiting == "you" {
			waiting = append(waiting, c)
		}
	}
	connected, errored := 0, 0
	for _, s := range x.Sources {
		switch s {
		case connectors.StateConnected:
			connected++
		case connectors.StateError:
			errored++
		}
	}

	chips := []pagekit.Chip{}
	if len(urgent) > 0 {
		chips = append(chips, pagekit.Chip{Tone: "warn", Text: strconv.Itoa(len(urgent)) + " priority unread"})
	}
	if slackUnread > 0 {
		chips = append(chips, pagekit.Chip{Tone: "accent", Text: strconv.Itoa(slackUnread) + " Slack unread"})
	}
	if errored > 0 {
		chips = append(chips, pagekit.Chip{Tone: "err", Text: pagekit.Plural(errored, "source error", "")})
	}

	caption := "no inboxes configured yet"
	if len(x.Lanes) > 0 {
		caption = "unread across " + pagekit.Plural(emailLanes, "inbox", "inboxes")
		if len(x.Lanes) > emailLanes {
			caption += " + WhatsApp"
		}
		caption += " · " + pagekit.Plural(len(items), "thread", "") + " in view"
	}

	var stamps []string
	for _, it := range items {
		if t, ok := parseTS(it.TS); ok && !t.After(now) {
			stamps = append(stamps, it.TS)
		}
	}
	series := perDay(stamps, localMidnight(now, -(days-1)), days, loc, pagekit.ShortDate)
	meetings := perDay(x.Events, localMidnight(now, 0), 7, loc, pagekit.Weekday)
	seriesTotal, meetingsTotal := 0, 0
	for _, p := range series {
		seriesTotal += int(p.Count)
	}
	for _, p := range meetings {
		meetingsTotal += int(p.Count)
	}
	recs := len(x.Recordings)

	nCards, nSources := len(x.SlackCards), len(x.Sources)
	waitingDisplay := "no clients linked"
	if nCards > 0 {
		waitingDisplay = strconv.Itoa(len(waiting)) + " of " + strconv.Itoa(nCards)
	}
	connectedFrac := pagekit.Frac(float64(connected), float64(nSources))
	meters := []pagekit.Meter{
		pagekit.Known("Email ("+strconv.Itoa(emailUnread)+")", pagekit.Frac(float64(emailUnread), float64(headline)), strconv.Itoa(emailUnread)+" unread", pagekit.Ramp1),
		pagekit.Known("WhatsApp ("+strconv.Itoa(waUnread)+")", pagekit.Frac(float64(waUnread), float64(headline)), strconv.Itoa(waUnread)+" unread", pagekit.Ramp3),
		pagekit.Known("Slack clients waiting on you ("+strconv.Itoa(len(waiting))+"/"+strconv.Itoa(nCards)+")", pagekit.Frac(float64(len(waiting)), float64(nCards)), waitingDisplay, pagekit.Warn),
		pagekit.Known("Sources connected ("+strconv.Itoa(connected)+"/"+strconv.Itoa(nSources)+")", connectedFrac, strconv.Itoa(int(connectedFrac*100+0.5))+"%", pagekit.Accent),
	}

	var names []string
	seen := map[string]bool{}
	for _, n := range append(cardNames(waiting), itemSenders(urgent)...) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	body := "Nobody is waiting on you. The threads that matter are answered."
	if len(names) > 0 {
		body = strings.Join(names[:min(3, len(names))], " · ")
	}
	value := len(waiting) + len(urgent)

	return Volume{
		Headline: headline,
		Chips:    chips,
		Caption:  caption,
		Meta: strconv.Itoa(headline) + " unread · " + strconv.Itoa(connected) + "/" + strconv.Itoa(nSources) + " sources connected · " +
			pagekit.Plural(meetingsTotal, "meeting", "") + " next 7 days · " + pagekit.Plural(recs, "recording", ""),
		Meters:        meters,
		Foot:          pagekit.Plural(meetingsTotal, "meeting", "") + " next 7 days · " + pagekit.Plural(recs, "recent recording", ""),
		Series:        series,
		SeriesTotal:   seriesTotal,
		Meetings:      meetings,
		MeetingsTotal: meetingsTotal,
		Insight: Insight{
			Value:    value,
			Headline: pagekit.Plural(len(waiting), "client thread", "") + " waiting on you · " + strconv.Itoa(len(urgent)) + " priority unread.",
			Body:     body,
			Frac:     pagekit.Frac(float64(value), float64(nCards+priorityItems)),
		},
	}
}

func cardNames(cs []VolumeCard) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func itemSenders(is []VolumeItem) []string {
	out := make([]string, 0, len(is))
	for _, i := range is {
		out = append(out, i.Sender)
	}
	return out
}
