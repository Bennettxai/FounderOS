package api

// /os/comms (spec 6.2): FounderOS v1's app/comms/page.tsx and its API routes
// /api/comms, /api/comms/digest, /api/comms/digest/read and /api/comms/reply,
// served under /api/founderos/pages/comms*. The page model is one GET; the
// pure shaping lives in internal/founderos/pages/comms.
//
// Honesty: every source is read in parallel and a failure is carried as a
// state or an error string next to its (empty) data, never as a quiet empty
// list. Replies go out only through the guarded Slack/SMTP connectors, so
// while FOUNDEROS_WRITES=0 they come back 502 with the refusal.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/agents/roster/clients"
	rostercomms "github.com/rhl/businessos-backend/internal/founderos/agents/roster/comms"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	commspage "github.com/rhl/businessos-backend/internal/founderos/pages/comms"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

func init() { RegisterPage(commsRoutes) }

const (
	commsPersonalSlug = "personal"  // comms_digests, digest_reads, contact_tags
	commsFounderSlug  = "founderos" // plaud_ingests
	// READ_RETENTION_DAYS (lib/comms-digest.ts): CARRY_MAX_DAYS (30) + 15.
	commsReadRetention = 45 * 24 * time.Hour
	commsRecordings    = 30
	commsFeedLimit     = 40
)

// commsSources is every read (and the two guarded writes) /comms needs.
// Production wires the live connectors; tests swap in fixtures.
type commsSources struct {
	Inboxes          func() []commspage.Inbox
	Emails           func(ctx context.Context) ([]email.CommsItem, error)
	EmailStatus      func(ctx context.Context) connectors.Status
	WhatsApp         func(ctx context.Context) ([]devicepush.Chat, connectors.Status)
	SlackStatus      func(ctx context.Context) connectors.Status
	SlackMessages    func(ctx context.Context, limit int) ([]slack.Message, error)
	SlackChannels    func(ctx context.Context) ([]slack.Channel, error)
	Roster           func(ctx context.Context) ([]commspage.RosterClient, connectors.State, string)
	CalendarStatus   func(ctx context.Context) connectors.Status
	CalendarAccounts func() []gcal.CalAccount
	Events           func(ctx context.Context) ([]gcal.CalEvent, error)
	PlaudRecordings  func(ctx context.Context, limit int) ([]plaud.Recording, error)
	PlaudStatus      func(ctx context.Context) connectors.Status
	FathomMeetings   func(ctx context.Context, limit int) ([]fathomcalls.Meeting, error)
	FathomStatus     func(ctx context.Context) connectors.Status
	SendSlack        func(ctx context.Context, channel, text string) (slack.SendResult, error)
	SendEmail        func(ctx context.Context, r email.Reply) error
	RunDigest        func(ctx context.Context) (rostercomms.RunResult, error)
}

var (
	commsLiveMu sync.Mutex
	commsLive   = map[*Deps]*commsSources{}
)

// commsSourcesFor returns the live sources for d, built once so the
// connectors' own caches (IMAP feed, Slack history) survive across requests.
var commsSourcesFor = func(d *Deps) *commsSources {
	commsLiveMu.Lock()
	defer commsLiveMu.Unlock()
	if s, ok := commsLive[d]; ok {
		return s
	}
	s := commsLiveSources(d)
	commsLive[d] = s
	return s
}

func commsLiveSources(d *Deps) *commsSources {
	res := d.Resolver
	mail, sl, cal := depsEmail(d), depsSlack(d), gcal.New(res)
	pl, fa := plaud.New(res), fathomcalls.New(res)
	roster := clients.NewStripeRoster(res)
	s := &commsSources{
		Inboxes: func() []commspage.Inbox {
			var out []commspage.Inbox
			for _, in := range mail.Inboxes() {
				out = append(out, commspage.Inbox{ID: in.ID, Name: in.Name})
			}
			return out
		},
		Emails:        func(ctx context.Context) ([]email.CommsItem, error) { return mail.LatestEmails(ctx, 40) },
		EmailStatus:   mail.Status,
		SlackStatus:   sl.Status,
		SlackMessages: sl.RecentMessages,
		SlackChannels: sl.ListChannels,
		Roster: func(ctx context.Context) ([]commspage.RosterClient, connectors.State, string) {
			r := roster.Roster(ctx)
			out := make([]commspage.RosterClient, 0, len(r.Clients))
			for _, c := range r.Clients {
				out = append(out, commspage.RosterClient{ID: c.ID, Name: c.Name, Status: c.Status})
			}
			return out, r.State, r.Detail
		},
		CalendarStatus:   cal.Status,
		CalendarAccounts: cal.Accounts,
		Events: func(ctx context.Context) ([]gcal.CalEvent, error) {
			return cal.UpcomingEvents(ctx, gcal.UpcomingOptions{Days: 7, Limit: 200})
		},
		PlaudRecordings: pl.RecentRecordings,
		PlaudStatus:     pl.Status,
		FathomMeetings:  fa.RecentMeetings,
		FathomStatus:    fa.Status,
		SendSlack:       sl.SendMessage,
		SendEmail: func(ctx context.Context, r email.Reply) error {
			err := mail.SendReply(ctx, r)
			if err == nil {
				mail.InvalidateCache() // a sent reply must not sit behind the cache window
			}
			return err
		},
		RunDigest: func(ctx context.Context) (rostercomms.RunResult, error) {
			for _, a := range rostercomms.Agents(RosterDeps(d)) {
				if a.Meta().ID != "comms-digest" {
					continue
				}
				res, err := a.Run(ctx)
				if err != nil {
					return rostercomms.RunResult{}, err
				}
				if r, ok := res.Data.(rostercomms.RunResult); ok {
					return r, nil
				}
				return rostercomms.RunResult{}, errors.New(res.Summary)
			}
			return rostercomms.RunResult{}, errors.New("comms-digest agent is not on the roster")
		},
	}
	s.WhatsApp = func(ctx context.Context) ([]devicepush.Chat, connectors.Status) {
		if d.Devices == nil {
			return nil, connectors.Status{ID: devicepush.SourceWhatsApp, Name: "WhatsApp", State: connectors.StateNotConfigured, Detail: "no device receiver"}
		}
		st := d.Devices.Connector(devicepush.SourceWhatsApp).Status(ctx)
		rd, err := d.Devices.WhatsAppChats(ctx)
		if err != nil || rd.Stale {
			// a stale push is not the current inbox: the lane shows the
			// status (error/stale) and no messages rather than old ones
			return nil, st
		}
		chats := rd.Data
		if len(chats) > 40 {
			chats = chats[:40] // recentChats(40)
		}
		return chats, st
	}
	return s
}

// ---- the page model -------------------------------------------------------------

type commsCalendarAccount struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type commsDigestBody struct {
	Digest      json.RawMessage `json:"digest"`
	Sources     json.RawMessage `json:"sources"`
	Gaps        []string        `json:"gaps,omitempty"`
	GeneratedAt *string         `json:"generatedAt"`
	Error       string          `json:"error,omitempty"`
}

type commsPage struct {
	Now         string                `json:"now"`
	Sources     []connectors.Status   `json:"sources"`
	Lanes       []commspage.Lane      `json:"lanes"`
	SlackCards  []commspage.SlackCard `json:"slackCards"`
	SlackRoster struct {
		State  connectors.State `json:"state"`
		Detail string           `json:"detail,omitempty"`
	} `json:"slackRoster"`
	Channels      []slack.Channel `json:"channels"`
	ChannelsError string          `json:"channelsError,omitempty"`
	Calendar      struct {
		Accounts []commsCalendarAccount `json:"accounts"`
		Events   []gcal.CalEvent        `json:"events"`
		Error    string                 `json:"error,omitempty"`
	} `json:"calendar"`
	Recordings struct {
		Recordings []commspage.Recording `json:"recordings"`
		Sources    []connectors.Status   `json:"sources"`
		Errors     []string              `json:"errors,omitempty"`
	} `json:"recordings"`
	Digest   commsDigestBody  `json:"digest"`
	ReadKeys []string         `json:"readKeys"`
	Volume   commspage.Volume `json:"volume"`
	Feed     []commspage.Item `json:"feed"`
	Gaps     []string         `json:"gaps,omitempty"`
}

func commsItemOf(e email.CommsItem) commspage.Item {
	return commspage.Item{Source: e.Source, Title: e.Title, Sender: e.Sender, ReplyTo: e.ReplyTo, Account: e.Account, Preview: e.Preview, TS: e.TS, Unread: e.Unread}
}

func commsChatItem(c devicepush.Chat) commspage.Item {
	src := c.Source
	if src == "" {
		src = "whatsapp"
	}
	return commspage.Item{Source: src, Title: c.Title, Sender: c.Sender, ReplyTo: c.ReplyTo, Preview: c.Preview, TS: c.TS, Unread: c.Unread}
}

// commsSlackItem is gatherCommsFeed's Slack mapping.
func commsSlackItem(m slack.Message) commspage.Item {
	ts := ""
	if f := commsSlackSeconds(m.TS); f > 0 {
		ts = time.UnixMilli(int64(f * 1000)).UTC().Format("2006-01-02T15:04:05.000Z")
	}
	preview := []rune(m.Text)
	if len(preview) > 140 {
		preview = preview[:140]
	}
	return commspage.Item{Source: "slack", Title: "#" + m.Channel + " — " + m.User, Sender: m.User, ReplyTo: m.Channel, Preview: string(preview), TS: ts}
}

func commsSlackSeconds(ts string) float64 {
	var f float64
	if err := json.Unmarshal([]byte(ts), &f); err != nil {
		return 0
	}
	return f
}

// commsGather reads every source concurrently.
func commsGather(ctx context.Context, d *Deps, src *commsSources, feedOnly bool) commsPage {
	var (
		wg                                sync.WaitGroup
		emails                            []email.CommsItem
		emailErr, msgErr, chanErr, evtErr error
		emailSt, slackSt, calSt, plSt     connectors.Status
		faSt                              connectors.Status
		chats                             []devicepush.Chat
		waSt                              connectors.Status
		msgs                              []slack.Message
		channels                          []slack.Channel
		roster                            []commspage.RosterClient
		rosterState                       connectors.State
		rosterDetail                      string
		events                            []gcal.CalEvent
		recs                              []plaud.Recording
		meets                             []fathomcalls.Meeting
		recErr, meetErr                   error
	)
	run := func(fn func()) {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	run(func() { emails, emailErr = src.Emails(ctx) })
	run(func() { chats, waSt = src.WhatsApp(ctx) })
	run(func() { msgs, msgErr = src.SlackMessages(ctx, 30) })
	if !feedOnly {
		run(func() { emailSt = src.EmailStatus(ctx) })
		run(func() { slackSt = src.SlackStatus(ctx) })
		run(func() { channels, chanErr = src.SlackChannels(ctx) })
		run(func() { roster, rosterState, rosterDetail = src.Roster(ctx) })
		run(func() { calSt = src.CalendarStatus(ctx) })
		run(func() { events, evtErr = src.Events(ctx) })
		run(func() { recs, recErr = src.PlaudRecordings(ctx, commsRecordings) })
		run(func() { meets, meetErr = src.FathomMeetings(ctx, commsRecordings) })
		run(func() { plSt = src.PlaudStatus(ctx) })
		run(func() { faSt = src.FathomStatus(ctx) })
	}
	wg.Wait()

	var p commsPage
	now := time.Now()
	p.Now = now.UTC().Format("2006-01-02T15:04:05.000Z")

	var emailItems, waItems, feedItems []commspage.Item
	if emailErr == nil {
		for _, e := range emails {
			emailItems = append(emailItems, commsItemOf(e))
		}
	}
	for _, c := range chats {
		waItems = append(waItems, commsChatItem(c))
	}
	// the unified feed (gatherCommsFeed): whatever each source answered
	feedItems = append(feedItems, waItems...)
	feedItems = append(feedItems, emailItems...)
	if msgErr == nil {
		for _, m := range msgs {
			feedItems = append(feedItems, commsSlackItem(m))
		}
	}
	p.Feed = commspage.MergeFeed(feedItems, commsFeedLimit)
	if feedOnly {
		return p
	}

	// An unconfigured source is known-empty (FounderOS v1 /comms renders its
	// not-configured card); only a configured source failing is a gap.
	if emailErr != nil && emailSt.State != connectors.StateNotConfigured {
		p.Gaps = append(p.Gaps, "email: "+emailErr.Error())
		if emailSt.State == connectors.StateConnected {
			// status answered but the feed did not: the lanes must not read as clear
			emailSt.State, emailSt.Detail = connectors.StateError, emailErr.Error()
		}
	}

	// Postgres: contact tags, read state, the stored digest, Plaud ingests
	var tags []commspage.ContactTag
	personal, perr := pagekit.WorkspaceID(ctx, d.Pool, commsPersonalSlug)
	if perr != nil {
		p.Gaps = append(p.Gaps, "contact tags and read state: "+perr.Error())
		p.Digest.Error = perr.Error()
	} else {
		var err error
		if tags, err = commsContactTags(ctx, d.Pool, personal); err != nil {
			p.Gaps = append(p.Gaps, "contact tags: "+err.Error())
		}
		if p.ReadKeys, err = commsReadKeys(ctx, d.Pool, personal); err != nil {
			p.Gaps = append(p.Gaps, "read state: "+err.Error())
		}
		p.Digest = commsStoredDigest(ctx, d.Pool, personal)
	}
	if p.ReadKeys == nil {
		p.ReadKeys = []string{}
	}
	if p.Digest.Digest == nil {
		p.Digest.Digest = json.RawMessage("null")
	}
	if p.Digest.Sources == nil {
		p.Digest.Sources = json.RawMessage("[]")
	}

	p.Lanes = commspage.BuildLanes(src.Inboxes(), emailItems, emailSt, waItems, waSt, tags)

	p.SlackRoster.State, p.SlackRoster.Detail = rosterState, rosterDetail
	if msgErr != nil {
		msgs = nil // no token / auth failure: cards go quiet, the status stays honest
	}
	p.SlackCards = commspage.SlackClientBoard(roster, msgs, now)
	p.Channels = channels
	if p.Channels == nil {
		p.Channels = []slack.Channel{}
	}
	if chanErr != nil {
		p.ChannelsError = chanErr.Error()
	}

	p.Calendar.Accounts = []commsCalendarAccount{}
	for _, a := range src.CalendarAccounts() {
		p.Calendar.Accounts = append(p.Calendar.Accounts, commsCalendarAccount{Name: a.Name, Color: a.Color})
	}
	p.Calendar.Events = events
	if evtErr != nil {
		p.Calendar.Events = nil
		if calSt.State != connectors.StateNotConfigured {
			p.Calendar.Error = evtErr.Error()
		}
	}
	if p.Calendar.Events == nil {
		p.Calendar.Events = []gcal.CalEvent{}
	}

	if recErr != nil {
		if plSt.State != connectors.StateNotConfigured {
			p.Recordings.Errors = append(p.Recordings.Errors, "plaud: "+recErr.Error())
		}
		recs = nil
	}
	if meetErr != nil {
		if faSt.State != connectors.StateNotConfigured {
			p.Recordings.Errors = append(p.Recordings.Errors, "fathom: "+meetErr.Error())
		}
		meets = nil
	}
	rows := commspage.MergeRecordings(recs, meets)
	if len(rows) > commsRecordings {
		rows = rows[:commsRecordings]
	}
	if founder, err := pagekit.WorkspaceID(ctx, d.Pool, commsFounderSlug); err == nil {
		via, err := commsPlaudIngests(ctx, d.Pool, founder)
		if err != nil {
			p.Gaps = append(p.Gaps, "plaud ingest marks: "+err.Error())
		}
		rows = commspage.MarkIngested(rows, via)
	} else if len(recs) > 0 {
		// a DB hiccup must not blank the tab; rows just show without the brain mark
		p.Gaps = append(p.Gaps, "plaud ingest marks: "+err.Error())
	}
	p.Recordings.Recordings = rows
	p.Recordings.Sources = []connectors.Status{plSt, faSt}

	// Plaud is the fifth source: the recorder in the room.
	p.Sources = []connectors.Status{emailSt, waSt, slackSt, calSt, plSt}

	vl := make([]commspage.VolumeLane, 0, len(p.Lanes))
	for _, l := range p.Lanes {
		items := make([]commspage.VolumeItem, 0, len(l.Items))
		for _, it := range l.Items {
			items = append(items, commspage.VolumeItem{Sender: it.Sender, TS: it.TS, Unread: it.Unread, Priority: it.Priority})
		}
		vl = append(vl, commspage.VolumeLane{Name: l.Name, Source: l.Source, Unread: l.Unread, Items: items})
	}
	cards := make([]commspage.VolumeCard, 0, len(p.SlackCards))
	for _, c := range p.SlackCards {
		cards = append(cards, commspage.VolumeCard{Name: c.Name, Unread: c.Unread, Waiting: c.Waiting})
	}
	states := make([]connectors.State, 0, len(p.Sources))
	for _, s := range p.Sources {
		states = append(states, s.State)
	}
	starts := make([]string, 0, len(p.Calendar.Events))
	for _, e := range p.Calendar.Events {
		starts = append(starts, e.Start)
	}
	ats := make([]string, 0, len(rows))
	for _, r := range rows {
		ats = append(ats, r.At)
	}
	p.Volume = commspage.CommsVolume(commspage.VolumeInput{Lanes: vl, SlackCards: cards, Sources: states, Events: starts, Recordings: ats, Now: now})
	return p
}

// ---- Postgres (bridge data) --------------------------------------------------------

func commsContactTags(ctx context.Context, pool *pgxpool.Pool, ws string) ([]commspage.ContactTag, error) {
	rows, err := pool.Query(ctx, `SELECT person, channel, tag, tier FROM founderos_contact_tags WHERE workspace_id = $1 ORDER BY person, channel`, ws)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (commspage.ContactTag, error) {
		var t commspage.ContactTag
		err := r.Scan(&t.Person, &t.Channel, &t.Tag, &t.Tier)
		return t, err
	})
}

func commsReadKeys(ctx context.Context, pool *pgxpool.Pool, ws string) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT key FROM founderos_digest_reads WHERE workspace_id = $1 ORDER BY read_at DESC`, ws)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func commsPlaudIngests(ctx context.Context, pool *pgxpool.Pool, ws string) (map[string]string, error) {
	out := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT file_id, via FROM founderos_plaud_ingests WHERE workspace_id = $1`, ws)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, via string
		if err := rows.Scan(&id, &via); err != nil {
			return out, err
		}
		out[id] = via
	}
	return out, rows.Err()
}

// commsStoredDigest is GET /api/comms/digest: the newest stored report with
// its generatedAt, or nulls when none was ever written.
func commsStoredDigest(ctx context.Context, pool *pgxpool.Pool, ws string) commsDigestBody {
	var body commsDigestBody
	var payload string
	var at time.Time
	err := pool.QueryRow(ctx, `
		SELECT payload::text, generated_at FROM founderos_comms_digests
		WHERE workspace_id = $1
		ORDER BY generated_at DESC, imported_at DESC, id DESC LIMIT 1`, ws).Scan(&payload, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return body
	}
	if err != nil {
		body.Error = err.Error()
		return body
	}
	stamp := at.UTC().Format("2006-01-02T15:04:05.000Z")
	body.GeneratedAt = &stamp
	var parsed struct {
		Digest  json.RawMessage `json:"digest"`
		Sources json.RawMessage `json:"sources"`
		Gaps    []string        `json:"gaps"`
	}
	if json.Unmarshal([]byte(payload), &parsed) != nil {
		return body // unreadable payload: no report, but its timestamp stays
	}
	body.Digest, body.Sources, body.Gaps = parsed.Digest, parsed.Sources, parsed.Gaps
	return body
}

// ---- routes ------------------------------------------------------------------------

func commsRoutes(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/comms", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		// ?view=feed is FounderOS v1's GET /api/comms: { feed } only.
		if c.Query("view") == "feed" {
			p := commsGather(ctx, d, commsSourcesFor(d), true)
			c.JSON(http.StatusOK, gin.H{"feed": p.Feed})
			return
		}
		c.JSON(http.StatusOK, commsGather(ctx, d, commsSourcesFor(d), false))
	})

	s.GET("/pages/comms/digest", func(c *gin.Context) {
		ws, err := pagekit.WorkspaceID(c.Request.Context(), d.Pool, commsPersonalSlug)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		b := commsStoredDigest(c.Request.Context(), d.Pool, ws)
		if b.Error != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": b.Error})
			return
		}
		if b.Digest == nil {
			b.Digest = json.RawMessage("null")
		}
		if b.Sources == nil {
			b.Sources = json.RawMessage("[]")
		}
		c.JSON(http.StatusOK, b)
	})

	// POST regenerates on demand ("run now"): the comms-digest agent scrapes
	// the connectors (reads only) and stores the row.
	s.POST("/pages/comms/digest", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
		defer cancel()
		r, err := commsSourcesFor(d).RunDigest(ctx)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"digest": r.Digest, "sources": r.Sources, "gaps": r.Gaps, "generatedAt": r.Digest.GeneratedAt})
	})

	readKey := func(c *gin.Context) (string, string, bool) {
		var body struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<16)).Decode(&body); err != nil || body.Key == "" || len(body.Key) > 400 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
			return "", "", false
		}
		ws, err := pagekit.WorkspaceID(c.Request.Context(), d.Pool, commsPersonalSlug)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return "", "", false
		}
		return body.Key, ws, true
	}
	s.GET("/pages/comms/digest/read", func(c *gin.Context) {
		ws, err := pagekit.WorkspaceID(c.Request.Context(), d.Pool, commsPersonalSlug)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		keys, err := commsReadKeys(c.Request.Context(), d.Pool, ws)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if keys == nil {
			keys = []string{}
		}
		c.JSON(http.StatusOK, gin.H{"keys": keys})
	})
	s.POST("/pages/comms/digest/read", func(c *gin.Context) {
		key, ws, ok := readKey(c)
		if !ok {
			return
		}
		ctx := c.Request.Context()
		if _, err := d.Pool.Exec(ctx, `
			INSERT INTO founderos_digest_reads (key, workspace_id, read_at) VALUES ($1, $2, now())
			ON CONFLICT (key) DO UPDATE SET read_at = EXCLUDED.read_at, workspace_id = EXCLUDED.workspace_id`, key, ws); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// keys older than the retention can never match again; it must outlive
		// CARRY_MAX_DAYS or a cleared message would come back
		if _, err := d.Pool.Exec(ctx, `DELETE FROM founderos_digest_reads WHERE workspace_id = $1 AND read_at < $2`, ws, time.Now().Add(-commsReadRetention)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	s.DELETE("/pages/comms/digest/read", func(c *gin.Context) {
		key, ws, ok := readKey(c)
		if !ok {
			return
		}
		if _, err := d.Pool.Exec(c.Request.Context(), `DELETE FROM founderos_digest_reads WHERE key = $1 AND workspace_id = $2`, key, ws); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	s.POST("/pages/comms/reply", func(c *gin.Context) {
		var body struct {
			Source  string  `json:"source"`
			Channel string  `json:"channel"`
			Account string  `json:"account"`
			To      string  `json:"to"`
			Subject *string `json:"subject"`
			Text    string  `json:"text"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<17)).Decode(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body must be JSON"})
			return
		}
		src := commsSourcesFor(d)
		text := []rune(body.Text)
		switch body.Source {
		case "slack":
			if body.Channel == "" || len(text) == 0 || len(text) > 4000 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "slack reply needs a channel and 1-4000 characters of text"})
				return
			}
			res, err := src.SendSlack(c.Request.Context(), body.Channel, body.Text)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"ok": false, "detail": err.Error(), "guarded": errors.Is(err, guard.ErrWritesDisabled)})
				return
			}
			code := http.StatusOK
			if !res.OK {
				code = http.StatusBadGateway
			}
			c.JSON(code, res)
		case "email":
			addr, err := mail.ParseAddress(body.To)
			if err != nil || addr.Address != strings.TrimSpace(body.To) || len(text) == 0 || len(text) > 20000 || (body.Subject != nil && len([]rune(*body.Subject)) > 300) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "email reply needs a valid to address, 1-20000 characters of text and a subject under 300"})
				return
			}
			subject := "(no subject)"
			if body.Subject != nil {
				subject = *body.Subject
			}
			if err := src.SendEmail(c.Request.Context(), email.Reply{AccountID: body.Account, To: body.To, Subject: subject, Text: body.Text}); err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error(), "guarded": errors.Is(err, guard.ErrWritesDisabled)})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "source must be slack or email"})
		}
	})
}
