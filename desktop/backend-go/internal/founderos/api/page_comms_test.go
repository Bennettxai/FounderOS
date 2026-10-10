package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

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
)

// commsThrowawayDB is a fresh migrated database on the bridge Postgres,
// dropped afterwards (never businessos_dev); skipped when Postgres is down.
func commsThrowawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_pagecomms_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func commsWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func commsStatus(id string, state connectors.State, detail string) connectors.Status {
	return connectors.Status{ID: id, Name: id, State: state, Detail: detail}
}

// commsFakes is every source /comms reads, answering from fixtures.
func commsFakes() *commsSources {
	now := time.Now().UTC()
	iso := func(d time.Duration) string { return now.Add(-d).Format(time.RFC3339) }
	url := "https://fathom.video/calls/9"
	mins := 30.0
	return &commsSources{
		Inboxes: func() []commspage.Inbox {
			return []commspage.Inbox{{ID: "inbox-1", Name: "Ops"}, {ID: "inbox-2", Name: "Personal"}}
		},
		Emails: func(context.Context) ([]email.CommsItem, error) {
			return []email.CommsItem{
				{Source: "email", Account: "inbox-1", Title: "Ops — Acme Corp", Sender: "Acme Corp", ReplyTo: "ceo@acme.com", Preview: "renewal", TS: iso(time.Hour), Unread: 1},
				{Source: "email", Account: "inbox-2", Title: "Personal — Mom", Sender: "Mom", Preview: "dinner", TS: iso(2 * time.Hour)},
			}, nil
		},
		EmailStatus: func(context.Context) connectors.Status {
			return commsStatus("email", connectors.StateConnected, "2 inboxes")
		},
		WhatsApp: func(context.Context) ([]devicepush.Chat, connectors.Status) {
			return []devicepush.Chat{{Source: "whatsapp", Title: "Sam", Sender: "Sam", Preview: "yo", TS: iso(3 * time.Hour), Unread: 2}},
				commsStatus("whatsapp", connectors.StateConnected, "pushed 1m ago")
		},
		SlackStatus: func(context.Context) connectors.Status {
			return commsStatus("slack", connectors.StateConnected, "bot ok")
		},
		SlackMessages: func(context.Context, int) ([]slack.Message, error) {
			return []slack.Message{{Channel: "vantage-acme", User: "U1", Text: "ship it", TS: fmt.Sprintf("%d.0001", now.Add(-30*time.Minute).Unix())}}, nil
		},
		SlackChannels: func(context.Context) ([]slack.Channel, error) {
			return nil, errors.New("slack: invalid_auth")
		},
		Roster: func(context.Context) ([]commspage.RosterClient, connectors.State, string) {
			return []commspage.RosterClient{{ID: "s1", Name: "Acme", Status: "won"}, {ID: "s2", Name: "Quiet Co", Status: "won"}}, connectors.StateConnected, ""
		},
		CalendarStatus: func(context.Context) connectors.Status {
			return commsStatus("calendar", connectors.StateConnected, "2 calendars")
		},
		CalendarAccounts: func() []gcal.CalAccount { return []gcal.CalAccount{{Name: "Ops", Color: "#aaa", Pass: "secret"}} },
		Events: func(context.Context) ([]gcal.CalEvent, error) {
			return nil, errors.New("caldav: 401")
		},
		PlaudRecordings: func(context.Context, int) ([]plaud.Recording, error) {
			return []plaud.Recording{{ID: "f1", Title: "Site walk", At: iso(5 * time.Hour), DurationMinutes: &mins}}, nil
		},
		PlaudStatus: func(context.Context) connectors.Status {
			return commsStatus("plaud", connectors.StateConnected, "token ok")
		},
		FathomMeetings: func(context.Context, int) ([]fathomcalls.Meeting, error) {
			return []fathomcalls.Meeting{{Title: "Discovery", URL: &url, At: iso(4 * time.Hour)}}, nil
		},
		FathomStatus: func(context.Context) connectors.Status {
			return commsStatus("fathom", connectors.StateConnected, "key ok")
		},
		SendSlack: slack.New(connectors.Resolver{}).SendMessage, // the real, guarded connector
		SendEmail: func(context.Context, email.Reply) error { return guard.ErrWritesDisabled },
		RunDigest: func(context.Context) (rostercomms.RunResult, error) {
			return rostercomms.RunResult{
				Digest:  rostercomms.Digest{GeneratedAt: "2026-09-30T09:00:00.000Z", WindowHours: 24, Entries: []rostercomms.Entry{}, Counts: map[rostercomms.Tier]int{}},
				Sources: []rostercomms.SourceState{{Source: "email", OK: true, Count: 3}},
			}, nil
		},
	}
}

func commsRouter(t *testing.T, pool *pgxpool.Pool, src *commsSources) http.Handler {
	t.Helper()
	prev := commsSourcesFor
	commsSourcesFor = func(*Deps) *commsSources { return src }
	t.Cleanup(func() { commsSourcesFor = prev })
	return router(t, &Deps{Pool: pool})
}

func commsDo(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	var rd *bytes.Reader
	if body != "" {
		rd = bytes.NewReader([]byte(body))
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestCommsPageNeedsASession(t *testing.T) {
	h := commsRouter(t, nil, commsFakes())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/comms", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
}

func TestCommsPageAssemblesEverySourceHonestly(t *testing.T) {
	pool := commsThrowawayDB(t)
	ctx := context.Background()
	personal := commsWorkspace(t, pool, "personal")
	founder := commsWorkspace(t, pool, "founderos")
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_contact_tags (person, channel, workspace_id, tag, tier) VALUES ('Acme Corp', 'email', $1, 'client', 1)`, personal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_plaud_ingests (file_id, workspace_id, title, ingested_at, via, slug) VALUES ('f1', $1, 'Site walk', now(), 'store', 'site-walk')`, founder); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_comms_digests (id, workspace_id, generated_at, payload) VALUES ('d1', $1, '2026-09-30T09:00:00Z', '{"digest":{"total":1,"entries":[]},"sources":[{"source":"email","ok":true,"count":1}]}')`, personal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_digest_reads (key, workspace_id, read_at) VALUES ('email|Mom|x', $1, now())`, personal); err != nil {
		t.Fatal(err)
	}

	code, body := commsDo(t, commsRouter(t, pool, commsFakes()), http.MethodGet, "/api/founderos/pages/comms", "")
	if code != http.StatusOK {
		t.Fatalf("GET: %d %v", code, body)
	}
	var page commsPage
	raw, _ := json.Marshal(body)
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, s := range page.Sources {
		ids = append(ids, s.ID)
	}
	if !slices.Equal(ids, []string{"email", "whatsapp", "slack", "calendar", "plaud"}) {
		t.Fatalf("sources = %v", ids)
	}
	if len(page.Lanes) != 3 || page.Lanes[0].Items[0].Priority != 1 || page.Lanes[2].ID != "whatsapp" || page.Lanes[2].Unread != 2 {
		t.Fatalf("lanes = %+v", page.Lanes)
	}
	if page.Volume.Headline != 3 {
		t.Fatalf("headline = %d", page.Volume.Headline)
	}
	// Slack: a live card for the client whose channel matched, a quiet one for the other
	if len(page.SlackCards) != 2 || !page.SlackCards[0].Live || page.SlackCards[1].Live || page.SlackRoster.State != connectors.StateConnected {
		t.Fatalf("slack cards = %+v", page.SlackCards)
	}
	// unreachable reads are errors, not empty lists passed off as quiet
	if page.ChannelsError == "" || page.Calendar.Error == "" || len(page.Calendar.Events) != 0 {
		t.Fatalf("channels/calendar errors not surfaced: %q %q", page.ChannelsError, page.Calendar.Error)
	}
	if strings.Contains(string(raw), "secret") || len(page.Calendar.Accounts) != 1 || page.Calendar.Accounts[0].Color != "#aaa" {
		t.Fatalf("calendar legend = %+v (and no password)", page.Calendar.Accounts)
	}
	if len(page.Recordings.Recordings) != 2 || page.Recordings.Recordings[0].Source != "fathom" || page.Recordings.Recordings[1].Brain == nil || *page.Recordings.Recordings[1].Brain != "store" {
		t.Fatalf("recordings = %+v", page.Recordings.Recordings)
	}
	if page.Digest.GeneratedAt == nil || !strings.HasPrefix(*page.Digest.GeneratedAt, "2026-09-30T09:00:00") || page.Digest.Digest == nil {
		t.Fatalf("digest = %+v", page.Digest)
	}
	if !slices.Equal(page.ReadKeys, []string{"email|Mom|x"}) {
		t.Fatalf("read keys = %v", page.ReadKeys)
	}
	if len(page.Feed) != 4 || page.Feed[0].Source != "slack" {
		t.Fatalf("feed = %+v", page.Feed)
	}
}

// An unconfigured source is known-empty (FounderOS v1 /comms): its read
// failing for want of credentials is neither a gap nor an "unreachable" error,
// so the page renders v1's not-configured states, never a raw warning.
func TestCommsPageNotConfiguredIsNotAnError(t *testing.T) {
	pool := commsThrowawayDB(t)
	commsWorkspace(t, pool, "personal")
	commsWorkspace(t, pool, "founderos")
	src := commsFakes()
	notCfg := func(id string) func(context.Context) connectors.Status {
		return func(context.Context) connectors.Status {
			return commsStatus(id, connectors.StateNotConfigured, "set the key")
		}
	}
	src.Inboxes = func() []commspage.Inbox { return nil }
	src.Emails = func(context.Context) ([]email.CommsItem, error) {
		return nil, errors.New("email: no inboxes configured")
	}
	src.EmailStatus = notCfg("email")
	src.CalendarStatus = notCfg("calendar")
	src.Events = func(context.Context) ([]gcal.CalEvent, error) {
		return nil, errors.New("gcal: no Google calendar configured")
	}
	src.PlaudStatus = notCfg("plaud")
	src.PlaudRecordings = func(context.Context, int) ([]plaud.Recording, error) {
		return nil, errors.New("plaud: no PLAUD_REFRESH_TOKEN")
	}
	src.FathomStatus = notCfg("fathom")
	src.FathomMeetings = func(context.Context, int) ([]fathomcalls.Meeting, error) {
		return nil, errors.New("fathom: FATHOM_API_KEY not configured")
	}

	code, body := commsDo(t, commsRouter(t, pool, src), http.MethodGet, "/api/founderos/pages/comms", "")
	if code != http.StatusOK {
		t.Fatalf("GET: %d %v", code, body)
	}
	var page commsPage
	raw, _ := json.Marshal(body)
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Gaps) != 0 {
		t.Fatalf("not configured must not be a gap: %v", page.Gaps)
	}
	if page.Calendar.Error != "" || len(page.Calendar.Events) != 0 {
		t.Fatalf("unconfigured calendar reads known-empty, not unreachable: %q", page.Calendar.Error)
	}
	if len(page.Recordings.Errors) != 0 || len(page.Recordings.Recordings) != 0 {
		t.Fatalf("unconfigured recorders read known-empty: %v", page.Recordings.Errors)
	}
	if page.Sources[0].State != connectors.StateNotConfigured || page.Sources[3].State != connectors.StateNotConfigured {
		t.Fatalf("sources stay honest: %+v", page.Sources)
	}
}

func TestCommsPageWithoutWorkspacesSaysSo(t *testing.T) {
	pool := commsThrowawayDB(t)
	code, body := commsDo(t, commsRouter(t, pool, commsFakes()), http.MethodGet, "/api/founderos/pages/comms", "")
	if code != http.StatusOK {
		t.Fatalf("GET: %d", code)
	}
	digest := body["digest"].(map[string]any)
	if digest["error"] == nil || !strings.Contains(digest["error"].(string), "bootstrap") {
		t.Fatalf("digest error = %v", digest)
	}
	if gaps, _ := body["gaps"].([]any); len(gaps) == 0 {
		t.Fatalf("contact tags / read state unreadable must be listed as gaps: %v", body)
	}
}

func TestCommsFeedRouteIsTheFounderosOSFeed(t *testing.T) {
	code, body := commsDo(t, commsRouter(t, nil, commsFakes()), http.MethodGet, "/api/founderos/pages/comms?view=feed", "")
	if code != http.StatusOK || len(body["feed"].([]any)) != 4 || body["lanes"] != nil {
		t.Fatalf("feed: %d %v", code, body)
	}
}

func TestCommsDigestReadRoundTrip(t *testing.T) {
	pool := commsThrowawayDB(t)
	personal := commsWorkspace(t, pool, "personal")
	if _, err := pool.Exec(context.Background(), `INSERT INTO founderos_digest_reads (key, workspace_id, read_at) VALUES ('ancient', $1, now() - interval '60 days')`, personal); err != nil {
		t.Fatal(err)
	}
	h := commsRouter(t, pool, commsFakes())
	if code, _ := commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/digest/read", `{"key":""}`); code != http.StatusBadRequest {
		t.Fatalf("empty key: %d", code)
	}
	for i := 0; i < 2; i++ { // idempotent
		if code, body := commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/digest/read", `{"key":"email|Acme|t1"}`); code != http.StatusOK || body["ok"] != true {
			t.Fatalf("mark: %d %v", code, body)
		}
	}
	_, body := commsDo(t, h, http.MethodGet, "/api/founderos/pages/comms/digest/read", "")
	if keys := body["keys"].([]any); len(keys) != 1 || keys[0] != "email|Acme|t1" {
		t.Fatalf("keys after mark (the 60-day-old key is pruned): %v", keys)
	}
	if code, _ := commsDo(t, h, http.MethodDelete, "/api/founderos/pages/comms/digest/read", `{"key":"email|Acme|t1"}`); code != http.StatusOK {
		t.Fatalf("unmark: %d", code)
	}
	_, body = commsDo(t, h, http.MethodGet, "/api/founderos/pages/comms/digest/read", "")
	if keys := body["keys"].([]any); len(keys) != 0 {
		t.Fatalf("keys after unmark: %v", keys)
	}
}

func TestCommsDigestGetAndRunNow(t *testing.T) {
	pool := commsThrowawayDB(t)
	commsWorkspace(t, pool, "personal")
	h := commsRouter(t, pool, commsFakes())
	code, body := commsDo(t, h, http.MethodGet, "/api/founderos/pages/comms/digest", "")
	if code != http.StatusOK || body["digest"] != nil || body["generatedAt"] != nil {
		t.Fatalf("no report yet: %d %v", code, body)
	}
	code, body = commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/digest", "")
	if code != http.StatusOK || body["generatedAt"] != "2026-09-30T09:00:00.000Z" || len(body["sources"].([]any)) != 1 {
		t.Fatalf("run now: %d %v", code, body)
	}
}

func TestCommsReplyIsGuarded(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	h := commsRouter(t, nil, commsFakes())
	for _, bad := range []string{`{}`, `{"source":"slack","channel":"","text":"x"}`, `{"source":"email","to":"not-an-address","text":"x"}`, `{"source":"fax","text":"x"}`} {
		if code, _ := commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/reply", bad); code != http.StatusBadRequest {
			t.Fatalf("%s: %d", bad, code)
		}
	}
	guard.Reset()
	code, body := commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/reply", `{"source":"slack","channel":"general","text":"hi"}`)
	if code != http.StatusBadGateway || body["ok"] != false || body["guarded"] != true || !strings.Contains(fmt.Sprint(body["detail"]), "FOUNDEROS_WRITES") {
		t.Fatalf("slack reply with writes off: %d %v", code, body)
	}
	if r := guard.Refused(); len(r) == 0 || r[len(r)-1].Action != "slack.chat.postMessage" {
		t.Fatalf("refusal not logged: %v", r)
	}
	code, body = commsDo(t, h, http.MethodPost, "/api/founderos/pages/comms/reply", `{"source":"email","account":"inbox-1","to":"ceo@acme.com","subject":"Re: renewal","text":"yes"}`)
	if code != http.StatusBadGateway || body["ok"] != false || body["guarded"] != true || !strings.Contains(fmt.Sprint(body["error"]), "FOUNDEROS_WRITES") {
		t.Fatalf("email reply with writes off: %d %v", code, body)
	}
}
