package comms

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

var runNow = time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)

type fakeEmail struct {
	items []email.CommsItem
	err   error
	limit int
}

func (f *fakeEmail) LatestEmails(_ context.Context, n int) ([]email.CommsItem, error) {
	f.limit = n
	return f.items, f.err
}

type fakeSlack struct {
	msgs []slack.Message
	err  error
}

func (f fakeSlack) RecentMessages(context.Context, int) ([]slack.Message, error) {
	return f.msgs, f.err
}

type fakeCal struct {
	evs []gcal.CalEvent
	err error
}

func (f fakeCal) UpcomingEvents(context.Context, gcal.UpcomingOptions) ([]gcal.CalEvent, error) {
	return f.evs, f.err
}

type fakeWins struct {
	wins []stripe.Win
	ok   bool
}

func (f fakeWins) StripeFunnelWins(context.Context, time.Time) ([]stripe.Win, bool) {
	return f.wins, f.ok
}

type fakeDigestStore struct {
	latest     []byte
	latestErr  error
	cleared    []string
	clearedErr error
	tags       []ContactTag
	tagsErr    error
	insertErr  error
	inserted   []struct {
		id      string
		at      time.Time
		payload []byte
	}
}

func (f *fakeDigestStore) Latest(context.Context) ([]byte, bool, error) {
	return f.latest, f.latest != nil, f.latestErr
}
func (f *fakeDigestStore) ClearedKeys(context.Context) ([]string, error) {
	return f.cleared, f.clearedErr
}
func (f *fakeDigestStore) ContactTags(context.Context) ([]ContactTag, error) {
	return f.tags, f.tagsErr
}
func (f *fakeDigestStore) Insert(_ context.Context, id string, at time.Time, payload []byte) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = append(f.inserted, struct {
		id      string
		at      time.Time
		payload []byte
	}{id, at, payload})
	return nil
}

func ago(d time.Duration) string { return runNow.Add(-d).UTC().Format(isoMillis) }

func sp(s string) *string { return &s }

func whatsappOK(chats ...devicepush.Chat) WhatsAppFeed {
	return func(context.Context) (devicepush.Reading[[]devicepush.Chat], error) {
		return devicepush.Reading[[]devicepush.Chat]{Device: "alexs-macbook-pro", Label: "MacBook", PushedAt: runNow.Add(-5 * time.Minute), Data: chats}, nil
	}
}

func liveDigestAgent(st *fakeDigestStore) (*DigestAgent, *fakeEmail) {
	em := &fakeEmail{items: []email.CommsItem{
		{Source: "email", Title: "Launchpad Cohort — Morgan Hale", Sender: "Morgan Hale", ReplyTo: "morgan@hale.example", Account: "inbox-1", Preview: "Re: the two week audit", TS: ago(2 * time.Hour)},
		{Source: "email", Title: "Personal — TikTok Shop", Sender: "TikTok Shop", Preview: "Your receipt", TS: ago(3 * time.Hour)},
		{Source: "email", Title: "Personal — Harbor Residences Ops", Sender: "Harbor Residences Ops", Preview: "question", TS: ago(4 * time.Hour)},
	}}
	a := &DigestAgent{
		Email: em,
		Slack: fakeSlack{msgs: []slack.Message{{Channel: "#vantage-team", User: "U123", Text: "lol", TS: "1790756400.000100"}}},
		WhatsApp: whatsappOK(devicepush.Chat{Source: "whatsapp", Title: "Riley Novak", Sender: "Riley Novak", ReplyTo: "15551234567@s.whatsapp.net",
			Preview: "question on the lesson", TS: ago(time.Hour)}),
		Calendar: fakeCal{evs: []gcal.CalEvent{{Title: "Alex <> Morgan Hale"}}},
		Wins:     fakeWins{ok: true, wins: []stripe.Win{{ID: "ch_1", Venture: "vantage", Name: sp("Harbor Residences"), Email: sp("ops@harbor.example"), AmountUSD: 500}}},
		Store:    st,
		Now:      func() time.Time { return runNow },
		NewID:    func() string { return "digest-1" },
	}
	return a, em
}

func TestDigestMetaMatchesFounderosOS(t *testing.T) {
	m := (&DigestAgent{}).Meta()
	if m.ID != "comms-digest" || m.Name != "Comms Digest" || m.DepartmentID != "dept-comms" || !strings.HasPrefix(m.Description, "Scrapes the last 24h") {
		t.Fatalf("meta = %+v", m)
	}
}

func TestDigestRunRanksAcrossChannelsAndPersists(t *testing.T) {
	st := &fakeDigestStore{tags: []ContactTag{{Person: "Riley Novak", Channel: "whatsapp", Tag: "student", Tier: 1}}}
	a, em := liveDigestAgent(st)
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("not ok: %s", res.Summary)
	}
	if em.limit != 120 {
		t.Errorf("latestEmails limit %d, want 120 as in the TS", em.limit)
	}
	r := res.Data.(RunResult)
	if len(r.Sources) != 3 || r.Sources[0].Source != "email" || r.Sources[1].Source != "whatsapp" || r.Sources[2].Source != "slack" {
		t.Fatalf("sources = %+v", r.Sources)
	}
	var tiers []string
	for _, e := range r.Digest.Entries {
		tiers = append(tiers, string(e.Tier)+":"+e.Sender)
	}
	want := "call:Morgan Hale,client:Harbor Residences Ops,people:U123,people:Riley Novak,noise:TikTok Shop"
	if strings.Join(tiers, ",") != want {
		t.Fatalf("entries = %v\nwant %s", tiers, want)
	}
	// Slack is mapped as in slackItems(): #channel title, ms ISO ts, channel replyTo
	for _, e := range r.Digest.Entries {
		if e.Source == "slack" && (e.Title != "#vantage-team" || e.TS != "2026-09-30T08:20:00.000Z" || e.ReplyTo != "#vantage-team") {
			t.Fatalf("slack entry = %+v", e)
		}
	}
	wantSummary := "4 need a reply (1 call · 1 client · 2 people · 0 brand) of 5 in 24h; 1 to unsubscribe"
	if res.Summary != wantSummary {
		t.Fatalf("summary = %q", res.Summary)
	}
	if len(st.inserted) != 1 || st.inserted[0].id != "digest-1" || !st.inserted[0].at.Equal(runNow) {
		t.Fatalf("inserted = %+v", st.inserted)
	}
	var back map[string]any
	if err := json.Unmarshal(st.inserted[0].payload, &back); err != nil || back["digest"] == nil || back["sources"] == nil {
		t.Fatalf("stored payload shape: %v %v", back, err)
	}
	if res.Model != "" || res.TokensIn != nil {
		t.Fatal("the digest is rules + connectors; it must not report a model")
	}
}

func TestDigestCarriesYesterdayAndHonoursClears(t *testing.T) {
	prev := RunResult{Digest: Digest{Entries: []Entry{
		{Tier: TierPeople, Rank: 2, Source: "email", Sender: "waiting@example.com", Title: "t", Preview: "p", TS: ago(48 * time.Hour)},
		{Tier: TierPeople, Rank: 2, Source: "email", Sender: "answered@example.com", Title: "t", Preview: "p", TS: ago(50 * time.Hour)},
	}}}
	raw, _ := json.Marshal(prev)
	st := &fakeDigestStore{latest: raw, cleared: []string{EntryKey("email", "answered@example.com", ago(50*time.Hour))}}
	a, _ := liveDigestAgent(st)
	res, _ := a.Run(context.Background())
	r := res.Data.(RunResult)
	var held, answered bool
	for _, e := range r.Digest.Entries {
		held = held || (e.Sender == "waiting@example.com" && e.Carried)
		answered = answered || e.Sender == "answered@example.com"
	}
	if !held || answered {
		t.Fatalf("held %v answered %v", held, answered)
	}
	if !strings.Contains(res.Summary, "of 6 open (1 held over)") {
		t.Fatalf("summary = %q", res.Summary)
	}
}

func TestDigestOneDeadSourceDegradesAndSaysSo(t *testing.T) {
	st := &fakeDigestStore{}
	a, em := liveDigestAgent(st)
	em.err = errors.New("all 4 inbox connections failed")
	a.Slack = fakeSlack{err: slack.ErrNotConfigured}
	res, _ := a.Run(context.Background())
	if !res.OK {
		t.Fatal("WhatsApp answered, so the run is ok")
	}
	if !strings.HasSuffix(res.Summary, " — email, slack unavailable") {
		t.Fatalf("summary = %q", res.Summary)
	}
	r := res.Data.(RunResult)
	if r.Sources[0].OK || r.Sources[0].Error == "" || r.Sources[0].Count != 0 {
		t.Fatalf("email source = %+v", r.Sources[0])
	}
}

func TestDigestAllSourcesDeadIsAFailureNotAnEmptyReport(t *testing.T) {
	st := &fakeDigestStore{}
	a, em := liveDigestAgent(st)
	em.err = email.ErrNotConfigured
	a.Slack = fakeSlack{err: slack.ErrNotConfigured}
	a.WhatsApp = func(context.Context) (devicepush.Reading[[]devicepush.Chat], error) {
		return devicepush.Reading[[]devicepush.Chat]{}, devicepush.ErrNoPush
	}
	res, _ := a.Run(context.Background())
	if res.OK {
		t.Fatalf("an all-dead run must fail: %s", res.Summary)
	}
	if !strings.Contains(res.Summary, "email, whatsapp, slack unavailable") {
		t.Fatalf("summary = %q", res.Summary)
	}
}

func TestDigestStaleWhatsAppPushIsUnavailableNotQuiet(t *testing.T) {
	st := &fakeDigestStore{}
	a, _ := liveDigestAgent(st)
	a.WhatsApp = func(context.Context) (devicepush.Reading[[]devicepush.Chat], error) {
		return devicepush.Reading[[]devicepush.Chat]{Label: "Mac mini", PushedAt: runNow.Add(-30 * time.Hour), Stale: true,
			Data: []devicepush.Chat{{Source: "whatsapp", Sender: "Mom", Preview: "hi", TS: ago(31 * time.Hour)}}}, nil
	}
	res, _ := a.Run(context.Background())
	r := res.Data.(RunResult)
	if r.Sources[1].OK || !strings.Contains(r.Sources[1].Error, "stale") {
		t.Fatalf("whatsapp source = %+v", r.Sources[1])
	}
}

func TestDigestNoDeviceReceiverIsUnavailable(t *testing.T) {
	st := &fakeDigestStore{}
	a, _ := liveDigestAgent(st)
	a.WhatsApp = nil
	res, _ := a.Run(context.Background())
	r := res.Data.(RunResult)
	if r.Sources[1].OK || r.Sources[1].Error == "" {
		t.Fatalf("whatsapp source = %+v", r.Sources[1])
	}
}

func TestDigestStorageFailureKeepsTheReportButFails(t *testing.T) {
	st := &fakeDigestStore{insertErr: errors.New("pg down")}
	a, _ := liveDigestAgent(st)
	res, _ := a.Run(context.Background())
	if res.OK {
		t.Fatal("a report that was not stored cannot be shown on /comms; the run must say so")
	}
	if !strings.Contains(res.Summary, "not stored: pg down") || res.Data.(RunResult).Digest.Total == 0 {
		t.Fatalf("summary = %q", res.Summary)
	}
}

func TestDigestContextAndBacklogGapsAreReported(t *testing.T) {
	st := &fakeDigestStore{latestErr: errors.New("boom"), clearedErr: errors.New("boom"), tagsErr: errors.New("boom")}
	a, _ := liveDigestAgent(st)
	a.Calendar = fakeCal{err: gcal.ErrNotConfigured}
	a.Wins = fakeWins{ok: false}
	res, _ := a.Run(context.Background())
	if !res.OK {
		t.Fatalf("context gaps degrade the ranking, they do not fail the run: %s", res.Summary)
	}
	r := res.Data.(RunResult)
	joined := strings.Join(r.Gaps, " | ")
	for _, want := range []string{"calendar", "clients", "contact tags", "backlog"} {
		if !strings.Contains(joined, want) {
			t.Errorf("gap %q missing from %q", want, joined)
		}
	}
	if !strings.Contains(res.Summary, "ranking without: calendar, clients, contact tags, backlog") {
		t.Fatalf("summary = %q", res.Summary)
	}
	// without the calendar Morgan cannot be a call
	for _, e := range r.Digest.Entries {
		if e.Tier == TierCall {
			t.Fatal("no calendar, no call tier")
		}
	}
}

func TestDigestCorruptStoredPayloadIsABacklogGap(t *testing.T) {
	st := &fakeDigestStore{latest: []byte("{")}
	a, _ := liveDigestAgent(st)
	res, _ := a.Run(context.Background())
	if !strings.Contains(strings.Join(res.Data.(RunResult).Gaps, ","), "backlog") {
		t.Fatalf("gaps = %v", res.Data.(RunResult).Gaps)
	}
}

func TestRosterNamesGroupPerCustomer(t *testing.T) {
	wins := []stripe.Win{
		{ID: "a", Venture: "vantage", Email: sp("X@a.com"), Name: sp("Alpha Co")},
		{ID: "b", Venture: "vantage", Email: sp("x@a.com"), Name: sp("Alpha Dup")},
		{ID: "c", Venture: "launchpad-cohort", Email: sp("c@c.com")},
		{ID: "d", Venture: "vantage"},
	}
	got := strings.Join(RosterNames(wins), ",")
	if got != "Alpha Co,c@c.com,Stripe customer" {
		t.Fatalf("names = %s", got)
	}
}

// JS new Date(Number(ts) * 1000) truncates the fraction (TimeClip →
// ToIntegerOrInfinity); rounding shifts about half of Slack timestamps by a
// millisecond and breaks their entry keys against imported TS digests.
func TestSlackISOTruncatesLikeJS(t *testing.T) {
	for ts, want := range map[string]string{
		"1727000000.123789": "2024-09-22T10:13:20.123Z",
		"1727000000.999999": "2024-09-22T10:13:20.999Z",
		"1727000000.000400": "2024-09-22T10:13:20.000Z",
	} {
		if got := slackISO(ts); got != want {
			t.Errorf("slackISO(%s) = %s, want %s", ts, got, want)
		}
	}
}
