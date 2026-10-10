package comms

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// ---- lib/comms.ts (tests/comms.test.ts) ------------------------------------

func TestMergeFeedSortsNewestFirstLimitsAndDropsBadStamps(t *testing.T) {
	items := []Item{
		{Source: "email", Title: "old", TS: "2026-07-01T00:00:00Z"},
		{Source: "slack", Title: "bad", TS: "not a date"},
		{Source: "whatsapp", Title: "new", TS: "2026-07-03T00:00:00Z"},
		{Source: "email", Title: "mid", TS: "2026-07-02T00:00:00Z"},
	}
	got := MergeFeed(items, 2)
	if len(got) != 2 || got[0].Title != "new" || got[1].Title != "mid" {
		t.Fatalf("MergeFeed = %+v", got)
	}
	if n := len(MergeFeed(items, 50)); n != 3 {
		t.Fatalf("invalid stamp kept: %d", n)
	}
}

func TestContactPriority(t *testing.T) {
	tags := []ContactTag{
		{Person: "Acme Brand", Tier: 2},
		{Person: "Max", Tier: 1},
		{Person: "acme brand team", Tier: 1},
	}
	if p := ContactPriority("ACME Brand Team", tags); p != 1 {
		t.Fatalf("lowest tier wins: %d", p)
	}
	if p := ContactPriority("Maximilian Corp", tags); p != 0 {
		t.Fatalf("short names only match exactly: %d", p)
	}
	if p := ContactPriority("max", tags); p != 1 {
		t.Fatalf("exact short match, case-insensitive: %d", p)
	}
	if p := ContactPriority("Nobody", nil); p != 0 {
		t.Fatalf("no tags: %d", p)
	}
}

// ---- lib/comms-lanes.ts (tests/comms-lanes.test.ts) -------------------------

func status(id string, state connectors.State) connectors.Status {
	return connectors.Status{ID: id, Name: id, State: state, Detail: id + " " + string(state)}
}

func lanesFixture() []Lane {
	inboxes := []Inbox{{ID: "inbox-1", Name: "Ops"}, {ID: "inbox-2", Name: "Personal"}, {ID: "inbox-3", Name: "Founders"}, {ID: "inbox-4", Name: "Billing"}}
	emails := []Item{
		{Source: "email", Account: "inbox-1", Title: "Ops — Acme Corp", Sender: "Acme Corp", ReplyTo: "billing@acme.com", Preview: "invoice", TS: "2026-07-27T10:00:00.000Z", Unread: 1},
		{Source: "email", Account: "inbox-1", Title: "Ops — Bob", Sender: "Bob", ReplyTo: "bob@example.com", Preview: "hi", TS: "2026-07-27T09:00:00.000Z", Unread: 1},
		{Source: "email", Account: "inbox-3", Title: "Founders — Carol", Sender: "Carol", Preview: "deck", TS: "2026-07-27T08:00:00.000Z"},
	}
	wa := []Item{{Source: "whatsapp", Title: "Dan", Sender: "Dan", Preview: "yo", TS: "2026-07-27T07:00:00.000Z", Unread: 2}}
	tags := []ContactTag{{Person: "Acme Corp", Channel: "email", Tag: "client", Tier: 1}}
	return BuildLanes(inboxes, emails, status("email", connectors.StateConnected), wa, status("whatsapp", connectors.StateConnected), tags)
}

func TestBuildLanesInboxesInOrderThenWhatsApp(t *testing.T) {
	lanes := lanesFixture()
	var ids, sources []string
	for _, l := range lanes {
		ids, sources = append(ids, l.ID), append(sources, l.Source)
	}
	if !slices.Equal(ids, []string{"inbox-1", "inbox-2", "inbox-3", "inbox-4", "whatsapp"}) ||
		!slices.Equal(sources, []string{"email", "email", "email", "email", "whatsapp"}) || lanes[0].Name != "Ops" {
		t.Fatalf("lanes = %v %v", ids, sources)
	}
}

func TestBuildLanesRoutesSumsStampsAndCarriesState(t *testing.T) {
	lanes := lanesFixture()
	in1 := lanes[0]
	if len(in1.Items) != 2 || in1.Items[0].Sender != "Acme Corp" || in1.Items[1].Sender != "Bob" || in1.Unread != 2 {
		t.Fatalf("inbox-1 = %+v", in1)
	}
	if len(lanes[1].Items) != 0 || lanes[1].Items == nil {
		t.Fatalf("an empty inbox still gets a lane with an empty (not null) list: %+v", lanes[1])
	}
	if in1.Items[0].ReplyTo != "billing@acme.com" || in1.Items[0].Preview != "invoice" || in1.Items[0].Priority != 1 || in1.Items[1].Priority != 0 {
		t.Fatalf("reply/priority = %+v", in1.Items)
	}
	if lanes[2].Items[0].ReplyTo != "" {
		t.Fatal("no address, no reply")
	}
	wa := lanes[4]
	if wa.Unread != 2 || wa.Items[0].Sender != "Dan" || wa.Detail != "whatsapp connected" || lanes[1].State != connectors.StateConnected {
		t.Fatalf("whatsapp = %+v", wa)
	}
	raw, _ := json.Marshal(in1.Items[1])
	if strings.Contains(string(raw), "priority") || strings.Contains(string(raw), "replyTo\":\"\"") {
		t.Fatalf("absent priority/replyTo are omitted: %s", raw)
	}
}

// ---- lib/comms-volume.ts (tests/comms-volume.test.ts) -----------------------

func day(d, h int) string {
	return time.Date(2026, 9, d, h, 0, 0, 0, time.Local).UTC().Format(time.RFC3339)
}

func volumeFixture() Volume {
	lanes := []VolumeLane{
		{Name: "Work", Source: "email", Unread: 3, Items: []VolumeItem{
			{Sender: "Acme", TS: day(24, 10), Unread: 2, Priority: 1},
			{Sender: "Beta", TS: day(23, 10), Unread: 1},
			{Sender: "Gamma", TS: day(23, 10), Unread: 0, Priority: 1},
		}},
		{Name: "Personal", Source: "email", Unread: 0, Items: []VolumeItem{{Sender: "Mom", TS: day(20, 10)}}},
		{Name: "WhatsApp", Source: "whatsapp", Unread: 1, Items: []VolumeItem{{Sender: "Sam", TS: day(24, 9), Unread: 1}}},
	}
	cards := []VolumeCard{{"Vantage", 2, "you"}, {"Zentra", 0, "them"}, {"Futurism", 1, "you"}, {"Quiet", 0, "none"}}
	sources := []connectors.State{"connected", "connected", "error", "not_configured", "connected"}
	events := []string{day(24, 16), day(25, 11), day(25, 14), day(30, 9)}
	recs := []string{day(22, 10), day(21, 10)}
	return CommsVolume(VolumeInput{Lanes: lanes, SlackCards: cards, Sources: sources, Events: events, Recordings: recs, Now: time.Date(2026, 9, 24, 15, 0, 0, 0, time.Local)})
}

func TestCommsVolumeHeadlineChipsCaptionMeta(t *testing.T) {
	v := volumeFixture()
	if v.Headline != 4 {
		t.Fatalf("headline %d", v.Headline)
	}
	want := []pagekit.Chip{{Tone: "warn", Text: "1 priority unread"}, {Tone: "accent", Text: "3 Slack unread"}, {Tone: "err", Text: "1 source error"}}
	if !slices.Equal(v.Chips, want) {
		t.Fatalf("chips %+v", v.Chips)
	}
	if v.Caption != "unread across 2 inboxes + WhatsApp · 5 threads in view" {
		t.Fatalf("caption %q", v.Caption)
	}
	if v.Meta != "4 unread · 3/5 sources connected · 4 meetings next 7 days · 2 recordings" {
		t.Fatalf("meta %q", v.Meta)
	}
}

func TestCommsVolumeMeters(t *testing.T) {
	v := volumeFixture()
	var labels []string
	hues := map[string]bool{}
	for _, m := range v.Meters {
		labels = append(labels, m.Label)
		hues[m.Hue] = true
		if !strings.HasPrefix(m.Hue, "var(--bn-") && !strings.HasPrefix(m.Hue, "color-mix(") {
			t.Fatalf("hue from the theme tokens only: %q", m.Hue)
		}
	}
	if !slices.Equal(labels, []string{"Email (3)", "WhatsApp (1)", "Slack clients waiting on you (2/4)", "Sources connected (3/5)"}) || len(hues) != 4 {
		t.Fatalf("meters %v hues %v", labels, hues)
	}
	for i, w := range []struct {
		frac    float64
		display string
	}{{0.75, "3 unread"}, {0.25, "1 unread"}, {0.5, "2 of 4"}, {0.6, "60%"}} {
		if m := v.Meters[i]; m.Frac == nil || math.Abs(*m.Frac-w.frac) > 1e-9 || m.Display != w.display {
			t.Fatalf("meter %d = %+v", i, m)
		}
	}
	if v.Foot != "4 meetings next 7 days · 2 recent recordings" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestCommsVolumeSeriesMeetingsInsight(t *testing.T) {
	v := volumeFixture()
	if len(v.Series) != 14 || v.Series[13] != (pagekit.Point{Label: "Sep 24", Count: 2}) || v.Series[12] != (pagekit.Point{Label: "Sep 23", Count: 2}) || v.SeriesTotal != 5 {
		t.Fatalf("series %+v total %d", v.Series, v.SeriesTotal)
	}
	if v.Series[9] != (pagekit.Point{Label: "Sep 20", Count: 1}) {
		t.Fatalf("Sep 20 %+v", v.Series[9])
	}
	var labels []string
	var counts []float64
	for _, m := range v.Meetings {
		labels, counts = append(labels, m.Label), append(counts, m.Count)
	}
	if !slices.Equal(labels, []string{"Thu", "Fri", "Sat", "Sun", "Mon", "Tue", "Wed"}) || !slices.Equal(counts, []float64{1, 2, 0, 0, 0, 0, 1}) || v.MeetingsTotal != 4 {
		t.Fatalf("meetings %v %v", labels, counts)
	}
	in := v.Insight
	if in.Value != 3 || in.Headline != "2 client threads waiting on you · 1 priority unread." || in.Body != "Vantage · Futurism · Acme" || math.Abs(in.Frac-0.5) > 1e-9 {
		t.Fatalf("insight %+v", in)
	}
}

func TestCommsVolumeEmptyIsHonest(t *testing.T) {
	e := CommsVolume(VolumeInput{Now: time.Date(2026, 9, 24, 15, 0, 0, 0, time.Local)})
	if e.Headline != 0 || len(e.Chips) != 0 || e.Caption != "no inboxes configured yet" || e.SeriesTotal != 0 || e.MeetingsTotal != 0 {
		t.Fatalf("empty %+v", e)
	}
	for _, m := range e.Meters {
		if m.Frac == nil || *m.Frac != 0 {
			t.Fatalf("empty meter %+v", m)
		}
	}
	if e.Meters[2].Display != "no clients linked" || e.Insight.Value != 0 || e.Insight.Frac != 0 || !strings.Contains(e.Insight.Body, "Nobody is waiting on you") {
		t.Fatalf("empty insight %+v", e.Insight)
	}
	raw, _ := json.Marshal(e)
	if strings.Contains(string(raw), `"chips":null`) {
		t.Fatalf("empty chips marshal as [] not null: %s", raw)
	}
}

// ---- lib/slack-clients.ts (tests/slack-clients*.test.ts) ---------------------

func rc(id, name, status string) RosterClient {
	return RosterClient{ID: id, Name: name, Status: status}
}

func TestSlackClientBoardLiveCardFromMatchingChannel(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	cards := SlackClientBoard([]RosterClient{rc("c1", "Acme Corp", "active")},
		[]slack.Message{{Channel: "acme-corp", User: "U123", Text: "Can you review the SOW?", TS: "1785499200.000100"}}, now)
	c := cards[0]
	if !c.Live || c.LastText != "Can you review the SOW?" || c.Channel == nil || *c.Channel != "#acme-corp" || c.Waiting != "you" || c.Heat == nil {
		t.Fatalf("card %+v", c)
	}
}

func TestSlackClientBoardQuietCardsAreHonestAndOnlyCurrentClients(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	msgs := []slack.Message{
		{Channel: "globex", User: "U1", Text: "new", TS: "1785499000"},
		{Channel: "initech-lead", User: "U2", Text: "hi", TS: "1785490000"},
	}
	clients := []RosterClient{rc("c1", "Zeta", "won"), rc("c2", "Globex", "won"), rc("c3", "Lead Co", "lead"), rc("c4", "Initech", "lead"), rc("c5", "Alpha", "active")}
	cards := SlackClientBoard(clients, msgs, now)
	var names []string
	for _, c := range cards {
		names = append(names, c.Name)
	}
	// live first (newest first), then quiet current clients by name; a lead
	// with a real thread counts, a lead without one does not
	if !slices.Equal(names, []string{"Globex", "Initech", "Alpha", "Zeta"}) {
		t.Fatalf("names %v", names)
	}
	quiet := cards[2]
	if quiet.Live || quiet.Channel != nil || quiet.LastText != "" || quiet.LastTS != nil || quiet.Heat != nil || quiet.Waiting != "none" {
		t.Fatalf("quiet %+v", quiet)
	}
}

// ---- lib/recordings.ts (tests/recordings.test.ts) ----------------------------

func f(v float64) *float64 { return &v }
func s(v string) *string   { return &v }

func TestMergeRecordingsNewestFirstWithSourceAndIDs(t *testing.T) {
	rows := MergeRecordings(
		[]plaud.Recording{{ID: "p1", Title: "Site walk", At: "2026-08-25T18:00:00Z", DurationMinutes: f(40)}, {ID: "p2", Title: "Voice memo", At: "2026-08-23T09:00:00Z", DurationMinutes: f(2)}},
		[]fathomcalls.Meeting{{Title: "Discovery call", URL: s("https://fathom.video/calls/1"), At: "2026-08-24T15:00:00Z", DurationMinutes: f(42)}},
	)
	var titles, sources []string
	for _, r := range rows {
		titles, sources = append(titles, r.Title), append(sources, r.Source)
	}
	if !slices.Equal(titles, []string{"Site walk", "Discovery call", "Voice memo"}) || !slices.Equal(sources, []string{"plaud", "fathom", "plaud"}) {
		t.Fatalf("rows %v %v", titles, sources)
	}
	if rows[0].ID != "plaud-p1" || rows[1].ID != "fathom-https://fathom.video/calls/1" || rows[1].URL == nil {
		t.Fatalf("ids %+v", rows)
	}
}

func TestMergeRecordingsBadDateSortsLast(t *testing.T) {
	rows := MergeRecordings([]plaud.Recording{{ID: "p1", Title: "bad", At: ""}}, []fathomcalls.Meeting{{Title: "good", At: "2026-08-24T15:00:00Z", DurationMinutes: f(1)}})
	if rows[0].Title != "good" {
		t.Fatalf("rows %+v", rows)
	}
	if rows[1].ID != "plaud-p1" {
		t.Fatal(rows[1].ID)
	}
	fb := MergeRecordings(nil, []fathomcalls.Meeting{{Title: "t", At: "2026-01-01T00:00:00Z"}})
	if fb[0].ID != "fathom-2026-01-01T00:00:00Z-t" {
		t.Fatalf("fathom id without url: %s", fb[0].ID)
	}
}

func TestMarkIngestedStampsOnlyFiledPlaudRows(t *testing.T) {
	rows := MergeRecordings([]plaud.Recording{{ID: "a", At: "2026-08-25T00:00:00Z"}, {ID: "b", At: "2026-08-24T00:00:00Z"}}, nil)
	marked := MarkIngested(rows, map[string]string{"a": "store"})
	if marked[0].Brain == nil || *marked[0].Brain != "store" || marked[1].Brain != nil {
		t.Fatalf("marked %+v", marked)
	}
}
