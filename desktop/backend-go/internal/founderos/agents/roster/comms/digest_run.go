package comms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
)

// This file ports lib/comms-digest-run.ts: gather the last 24 hours across
// the four inboxes, WhatsApp and Slack through the same connectors /comms
// uses, rank it, stack it on the previous report and store it in
// founderos_comms_digests (personal workspace).

// ---- seams ------------------------------------------------------------------

type EmailFeed interface {
	LatestEmails(ctx context.Context, limitPerInbox int) ([]email.CommsItem, error)
}

type SlackFeed interface {
	RecentMessages(ctx context.Context, limit int) ([]slack.Message, error)
}

type CalendarFeed interface {
	UpcomingEvents(ctx context.Context, o gcal.UpcomingOptions) ([]gcal.CalEvent, error)
}

// ClientWins is the Stripe side of lib/client-roster.ts (payments.Connector).
type ClientWins interface {
	StripeFunnelWins(ctx context.Context, now time.Time) ([]stripe.Win, bool)
}

// WhatsAppFeed is the pushed WhatsApp lane (devicepush.Receiver.WhatsAppChats).
type WhatsAppFeed func(ctx context.Context) (devicepush.Reading[[]devicepush.Chat], error)

// ContactTag is a founderos_contact_tags row.
type ContactTag struct {
	Person  string `json:"person"`
	Channel string `json:"channel"`
	Tag     string `json:"tag"`
	Tier    int    `json:"tier"`
}

// DigestStore is the digest's slice of Postgres. Latest reports found=false
// when no digest was ever stored; an error is never an empty backlog.
type DigestStore interface {
	Latest(ctx context.Context) (payload []byte, found bool, err error)
	ClearedKeys(ctx context.Context) ([]string, error)
	ContactTags(ctx context.Context) ([]ContactTag, error)
	Insert(ctx context.Context, id string, generatedAt time.Time, payload []byte) error
}

// ---- result -----------------------------------------------------------------

// SourceState is DigestSourceState.
type SourceState struct {
	Source string `json:"source"`
	OK     bool   `json:"ok"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// RunResult is DigestRunResult, the stored payload. Gaps is a bridge
// addition (the TS swallowed these): ranking context or backlog that could
// not be read, so a thinner report is not mistaken for a quieter morning.
type RunResult struct {
	Digest  Digest        `json:"digest"`
	Sources []SourceState `json:"sources"`
	Gaps    []string      `json:"gaps,omitempty"`
}

// ---- the agent --------------------------------------------------------------

// DigestAgent is comms-digest, the 9am report.
type DigestAgent struct {
	Email    EmailFeed
	Slack    SlackFeed
	WhatsApp WhatsAppFeed
	Calendar CalendarFeed
	Wins     ClientWins
	Store    DigestStore
	Now      func() time.Time
	NewID    func() string
}

func (a *DigestAgent) Meta() agents.Meta {
	return agents.Meta{
		ID:           "comms-digest",
		Name:         "Comms Digest",
		DepartmentID: "dept-comms",
		Description:  "Scrapes the last 24h across all four inboxes, WhatsApp and Slack and ranks who needs a reply: calls first, then clients, students and family, brand deals, group chats, companies last. Also lists what to unsubscribe from.",
	}
}

func (a *DigestAgent) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *DigestAgent) Run(ctx context.Context) (agents.Result, error) {
	now := a.now().UTC()
	r := a.runDigest(ctx, now)

	ok := false
	for _, s := range r.Sources {
		ok = ok || s.OK
	}
	summary := DigestSummary(r)
	if err := a.store(ctx, r, now); err != nil {
		ok = false
		summary += " · not stored: " + err.Error()
	}
	return agents.Result{OK: ok, Summary: summary, Data: r}, nil
}

func (a *DigestAgent) store(ctx context.Context, r RunResult, now time.Time) error {
	if a.Store == nil {
		return errors.New("no digest store")
	}
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	id := uuid.NewString()
	if a.NewID != nil {
		id = a.NewID()
	}
	return a.Store.Insert(ctx, id, now, payload)
}

type sourceOut struct {
	items []Item
	state SourceState
}

func guarded(source string, fn func() ([]Item, error)) sourceOut {
	items, err := fn()
	if err != nil {
		return sourceOut{items: nil, state: SourceState{Source: source, OK: false, Count: 0, Error: err.Error()}}
	}
	return sourceOut{items: items, state: SourceState{Source: source, OK: true, Count: len(items)}}
}

func (a *DigestAgent) runDigest(ctx context.Context, now time.Time) RunResult {
	var (
		wg                   sync.WaitGroup
		mail, whatsapp, slck sourceOut
		dctx                 DigestContext
		gaps                 []string
	)
	wg.Add(4)
	go func() { defer wg.Done(); mail = guarded("email", func() ([]Item, error) { return a.emailItems(ctx) }) }()
	go func() {
		defer wg.Done()
		whatsapp = guarded("whatsapp", func() ([]Item, error) { return a.whatsappItems(ctx) })
	}()
	go func() { defer wg.Done(); slck = guarded("slack", func() ([]Item, error) { return a.slackItems(ctx) }) }()
	go func() { defer wg.Done(); dctx, gaps = a.gatherContext(ctx, now) }()
	wg.Wait()

	items := append(append(append([]Item{}, mail.items...), whatsapp.items...), slck.items...)
	previous, cleared, err := a.backlog(ctx)
	if err != nil {
		gaps = append(gaps, "backlog: "+err.Error())
	}
	digest := StackDigest(BuildDigest(items, dctx), previous, cleared, now)
	return RunResult{Digest: digest, Sources: []SourceState{mail.state, whatsapp.state, slck.state}, Gaps: gaps}
}

func (a *DigestAgent) emailItems(ctx context.Context) ([]Item, error) {
	if a.Email == nil {
		return nil, errors.New("email connector not wired")
	}
	raw, err := a.Email.LatestEmails(ctx, 120)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(raw))
	for _, m := range raw {
		out = append(out, Item{Source: "email", Title: m.Title, Preview: m.Preview, TS: m.TS, Sender: m.Sender, ReplyTo: m.ReplyTo, Account: m.Account})
	}
	return out, nil
}

func (a *DigestAgent) whatsappItems(ctx context.Context) ([]Item, error) {
	if a.WhatsApp == nil {
		return nil, errors.New("no device receiver: WhatsApp is pushed by founderos-collector")
	}
	rd, err := a.WhatsApp(ctx)
	if err != nil {
		return nil, err
	}
	if rd.Stale {
		return nil, fmt.Errorf("stale: last push from %s %s", rd.Label, devicepush.Age(a.now().Sub(rd.PushedAt)))
	}
	chats := rd.Data
	if len(chats) > 80 {
		chats = chats[:80] // recentChats(80)
	}
	out := make([]Item, 0, len(chats))
	for _, c := range chats {
		out = append(out, Item{Source: "whatsapp", Title: c.Title, Preview: c.Preview, TS: c.TS, Sender: c.Sender, ReplyTo: c.ReplyTo})
	}
	return out, nil
}

// slackItems maps Slack history onto CommsItems, as the TS slackItems() does.
func (a *DigestAgent) slackItems(ctx context.Context) ([]Item, error) {
	if a.Slack == nil {
		return nil, errors.New("slack connector not wired")
	}
	msgs, err := a.Slack.RecentMessages(ctx, 60)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(msgs))
	for _, m := range msgs {
		title := "#" + strings.TrimPrefix(m.Channel, "#")
		sender := m.User
		if sender == "" {
			sender = title
		}
		out = append(out, Item{Source: "slack", Title: title, Preview: m.Text, TS: slackISO(m.TS), Sender: sender, ReplyTo: m.Channel})
	}
	return out, nil
}

// slackISO is new Date(Number(ts) * 1000).toISOString(); an unparsable ts
// becomes "" and falls outside every window rather than failing the lane.
func slackISO(ts string) string {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	return time.UnixMilli(int64(f * 1000)).UTC().Format(isoMillis) // truncates, as JS Date does
}

var (
	studentTag = regexp.MustCompile(`(?i)student|cohort`)
	familyTag  = regexp.MustCompile(`(?i)family|personal`)
)

// gatherContext is gatherDigestContext. Each part degrades on its own and is
// named in gaps when it could not be read.
func (a *DigestAgent) gatherContext(ctx context.Context, now time.Time) (DigestContext, []string) {
	dctx := DigestContext{Now: now}
	var gaps []string

	if a.Calendar == nil {
		gaps = append(gaps, "calendar: not wired")
	} else if evs, err := a.Calendar.UpcomingEvents(ctx, gcal.UpcomingOptions{Days: 7, Limit: 40, Now: now}); err != nil {
		gaps = append(gaps, "calendar: "+err.Error())
	} else {
		for _, e := range evs {
			if e.Title != "" {
				dctx.MeetingTitles = append(dctx.MeetingTitles, e.Title)
			}
		}
	}

	if a.Wins == nil {
		gaps = append(gaps, "clients: not wired")
	} else if wins, ok := a.Wins.StripeFunnelWins(ctx, now); !ok {
		gaps = append(gaps, "clients: no Stripe account answered")
	} else {
		dctx.ClientNames = RosterNames(wins)
	}

	if a.Store == nil {
		gaps = append(gaps, "contact tags: no store")
	} else if tags, err := a.Store.ContactTags(ctx); err != nil {
		gaps = append(gaps, "contact tags: "+err.Error())
	} else {
		for _, t := range tags {
			if studentTag.MatchString(t.Tag) {
				dctx.Students = append(dctx.Students, t.Person)
			}
			if familyTag.MatchString(t.Tag) {
				dctx.Family = append(dctx.Family, t.Person)
			}
		}
	}
	return dctx, gaps
}

// backlog is what the previous report still owes: its entries plus every
// cleared key. Any failure means no carry-over (and a gap), never a lost report.
func (a *DigestAgent) backlog(ctx context.Context) ([]Entry, []string, error) {
	if a.Store == nil {
		return nil, nil, errors.New("no digest store")
	}
	payload, found, err := a.Store.Latest(ctx)
	if err != nil {
		return nil, nil, err
	}
	var previous []Entry
	if found {
		if previous, err = PreviousEntries(payload); err != nil {
			return nil, nil, err
		}
	}
	cleared, err := a.Store.ClearedKeys(ctx)
	if err != nil {
		return nil, nil, err
	}
	return previous, cleared, nil
}

// RosterNames is rosterFromStripeWins reduced to names: one row per paying
// customer (venture + email, else venture + name, else the charge).
func RosterNames(wins []stripe.Win) []string {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	firstOf := map[string]stripe.Win{}
	var order []string
	for _, w := range wins {
		key := w.ID
		if w.Email != nil && *w.Email != "" {
			key = w.Venture + ":" + norm(*w.Email)
		} else if w.Name != nil && *w.Name != "" {
			key = w.Venture + ":name:" + norm(*w.Name)
		}
		if _, seen := firstOf[key]; !seen {
			firstOf[key] = w
			order = append(order, key)
		}
	}
	names := make([]string, 0, len(order))
	for _, k := range order {
		w := firstOf[k]
		switch {
		case w.Name != nil && *w.Name != "":
			names = append(names, *w.Name)
		case w.Email != nil && *w.Email != "":
			names = append(names, *w.Email)
		default:
			names = append(names, "Stripe customer")
		}
	}
	sort.SliceStable(names, func(i, j int) bool { return localeLess(names[i], names[j]) })
	return names
}

// DigestSummary is digestSummary: the one-line headline for the run log.
func DigestSummary(r RunResult) string {
	d := r.Digest
	held := 0
	for _, e := range d.Entries {
		if e.Carried {
			held++
		}
	}
	scope := fmt.Sprintf("%d in 24h", d.Total)
	if held > 0 {
		scope = fmt.Sprintf("%d open (%d held over)", d.Total, held)
	}
	head := fmt.Sprintf("%d need a reply (%d call · %d client · %d people · %d brand) of %s; %d to unsubscribe",
		d.NeedsReply, d.Counts[TierCall], d.Counts[TierClient], d.Counts[TierPeople], d.Counts[TierBrandDeal], scope, len(d.Unsubscribes))
	var dead []string
	for _, s := range r.Sources {
		if !s.OK {
			dead = append(dead, s.Source)
		}
	}
	if len(dead) > 0 {
		head += " — " + strings.Join(dead, ", ") + " unavailable"
	}
	if len(r.Gaps) > 0 {
		var names []string
		for _, g := range r.Gaps {
			name, _, _ := strings.Cut(g, ":")
			names = append(names, name)
		}
		head += " · ranking without: " + strings.Join(names, ", ")
	}
	return head
}
