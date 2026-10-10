package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// resolver points at a temp env.local and blanks the process env, so a real
// token in the developer's shell can never leak into a test.
func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv("SLACK_BOT_TOKEN", "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

type fakeSlack struct {
	mu       sync.Mutex
	calls    map[string]int
	auth     string // raw auth.test body
	pages    [][]map[string]any
	history  map[string][]map[string]any
	failHist map[string]bool
	posts    []map[string]any
}

func (f *fakeSlack) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.calls == nil {
			f.calls = map[string]int{}
		}
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		f.calls[method]++
		if r.Header.Get("Authorization") != "Bearer xoxb-test" {
			fmt.Fprint(w, `{"ok":false,"error":"not_authed"}`)
			return
		}
		switch method {
		case "auth.test":
			fmt.Fprint(w, f.auth)
		case "conversations.list":
			if r.URL.Query().Get("types") != "public_channel,private_channel" {
				t.Errorf("types = %q", r.URL.Query().Get("types"))
			}
			idx := 0
			if c := r.URL.Query().Get("cursor"); c != "" {
				fmt.Sscanf(c, "page-%d", &idx)
			}
			next := ""
			if idx+1 < len(f.pages) {
				next = fmt.Sprintf("page-%d", idx+1)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true, "channels": f.pages[idx],
				"response_metadata": map[string]any{"next_cursor": next},
			})
		case "conversations.history":
			ch := r.URL.Query().Get("channel")
			if r.URL.Query().Get("limit") != "10" {
				t.Errorf("history limit = %q", r.URL.Query().Get("limit"))
			}
			if f.failHist[ch] {
				fmt.Fprint(w, `{"ok":false,"error":"not_in_channel"}`)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "messages": f.history[ch]})
		case "chat.postMessage":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			f.posts = append(f.posts, body)
			fmt.Fprint(w, `{"ok":true,"ts":"1.1"}`)
		default:
			fmt.Fprint(w, `{"ok":false,"error":"unknown_method"}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTest(t *testing.T, f *fakeSlack, env string) *Connector {
	c := New(resolver(t, env))
	c.baseURL = f.serve(t).URL + "/api/"
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta != (connectors.Meta{ID: "slack", Name: "Slack", Kind: connectors.KindSlack}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfiguredWithoutToken(t *testing.T) {
	f := &fakeSlack{}
	c := newTest(t, f, "")
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured {
		t.Fatalf("state = %s", s.State)
	}
	if s.Detail != "Set SLACK_BOT_TOKEN (xoxb-…) in ~/.founderos/.env or under API keys. Needs channels:read, channels:history, users:read scopes." {
		t.Fatalf("detail = %q", s.Detail)
	}
	if len(f.calls) != 0 {
		t.Fatal("no token must mean no network call")
	}
}

func TestStatusConnected(t *testing.T) {
	f := &fakeSlack{auth: `{"ok":true,"url":"https://acme.slack.com/","team":"Acme","user":"founderos_os","team_id":"T1","user_id":"U1"}`}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Connected to Acme as founderos_os" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["team"] != "Acme" || s.Meta["user"] != "founderos_os" {
		t.Fatalf("meta = %+v", s.Meta)
	}
}

func TestStatusErrorOnAuthFailureAndUnreachable(t *testing.T) {
	f := &fakeSlack{auth: `{"ok":false,"error":"invalid_auth"}`}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	s := c.Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Token set but auth failed: An API error occurred: invalid_auth" {
		t.Fatalf("status = %+v", s)
	}

	c.baseURL = "http://127.0.0.1:1/api/" // nothing listens on port 1
	s = c.Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "Token set but auth failed: ") {
		t.Fatalf("unreachable must read error, got %+v", s)
	}
}

func TestListChannelsWalksEveryPageAndShapes(t *testing.T) {
	f := &fakeSlack{pages: [][]map[string]any{
		{
			{"id": "C2", "name": "sales", "is_member": true, "num_members": 4, "topic": map[string]any{"value": "deals"}},
			{"id": "C9", "name": "old", "is_archived": true},
		},
		{
			{"id": "C1", "name": "general", "is_private": true},
			{"id": "", "name": "noid"},
		},
	}}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	got, err := c.ListChannels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Channel{
		{ID: "C1", Name: "general", IsPrivate: true},
		{ID: "C2", Name: "sales", IsMember: true, Members: 4, Topic: "deals"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	if f.calls["conversations.list"] != 2 {
		t.Fatalf("list calls = %d, want 2 pages", f.calls["conversations.list"])
	}
}

func TestListChannelsWithoutTokenIsNotConfiguredNotEmpty(t *testing.T) {
	c := newTest(t, &fakeSlack{}, "")
	if _, err := c.ListChannels(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestRecentMessagesRanksChannelsMergesAndCaches(t *testing.T) {
	// 14 joined channels: only the 12 most recently updated are read.
	var chans []map[string]any
	hist := map[string][]map[string]any{}
	for i := 0; i < 14; i++ {
		id := fmt.Sprintf("C%02d", i)
		chans = append(chans, map[string]any{"id": id, "name": "ch" + id, "is_member": true, "updated": 1000 + i})
		hist[id] = []map[string]any{{"user": "U1", "text": "msg " + id, "ts": fmt.Sprintf("%d.000100", 1700000000+i)}}
	}
	chans = append(chans, map[string]any{"id": "CX", "name": "notjoined", "is_member": false, "updated": 9999})
	hist["C13"] = append(hist["C13"], map[string]any{"text": "anon", "ts": "1600000000.5"})
	f := &fakeSlack{pages: [][]map[string]any{chans}, history: hist, failHist: map[string]bool{"C12": true}}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	now := time.Unix(1_800_000_000, 0)
	c.now = func() time.Time { return now }

	got, err := c.RecentMessages(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d", len(got))
	}
	// C12 failed (dropped silently), so newest is C13 then C11, C10 ...
	if got[0].Channel != "chC13" || got[1].Channel != "chC11" || got[0].User != "U1" {
		t.Fatalf("order = %+v", got)
	}
	if f.calls["conversations.history"] != 12 {
		t.Fatalf("history calls = %d, want 12", f.calls["conversations.history"])
	}
	for _, m := range got {
		if m.Channel == "chC00" || m.Channel == "chC01" || m.Channel == "notjoined" {
			t.Fatalf("read a channel outside the top 12 joined: %+v", m)
		}
	}

	// A cached scan of 5 answers a request for 3 without the network.
	before := f.calls["conversations.list"]
	if got, _ := c.RecentMessages(context.Background(), 3); len(got) != 3 || f.calls["conversations.list"] != before {
		t.Fatalf("superset cache missed: len=%d calls=%d", len(got), f.calls["conversations.list"])
	}
	// A bigger request than the cache rescans.
	c.RecentMessages(context.Background(), 20)
	if f.calls["conversations.list"] != before+1 {
		t.Fatal("larger limit must rescan")
	}
	// The TTL expires after 20 minutes.
	now = now.Add(21 * time.Minute)
	c.RecentMessages(context.Background(), 3)
	if f.calls["conversations.list"] != before+2 {
		t.Fatal("expired cache must rescan")
	}
	// An anonymous message reads "unknown".
	all, _ := c.RecentMessages(context.Background(), 20)
	if last := all[len(all)-1]; last.User != "unknown" || last.Text != "anon" {
		t.Fatalf("last = %+v", last)
	}
}

func TestRecentMessagesWithoutTokenErrors(t *testing.T) {
	c := newTest(t, &fakeSlack{}, "")
	if _, err := c.RecentMessages(context.Background(), 10); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}

func TestSendMessageRefusedWhenWritesOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := &fakeSlack{pages: [][]map[string]any{{{"id": "C1", "name": "general", "is_member": true}}}}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	res, err := c.SendMessage(context.Background(), "#general", "hello")
	if !errors.Is(err, guard.ErrWritesDisabled) || res.OK {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(f.posts) != 0 || len(f.calls) != 0 {
		t.Fatalf("a refused send touched Slack: %v", f.calls)
	}
	if r := guard.Refused(); len(r) != 1 || r[0].Action != "slack.chat.postMessage" {
		t.Fatalf("refusals = %+v", r)
	}
}

func TestSendMessagePostsWhenWritesOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fakeSlack{pages: [][]map[string]any{{{"id": "C1", "name": "general", "is_member": true}}}}
	c := newTest(t, f, "SLACK_BOT_TOKEN=xoxb-test\n")
	res, err := c.SendMessage(context.Background(), "#general", "hello")
	if err != nil || !res.OK || res.Detail != "sent to ##general" /* TS interpolates the name as given */ {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(f.posts) != 1 || f.posts[0]["channel"] != "C1" || f.posts[0]["text"] != "hello" {
		t.Fatalf("posts = %+v", f.posts)
	}
	res, _ = c.SendMessage(context.Background(), "nope", "x")
	if res.OK || res.Detail != "channel #nope not found or bot not invited" {
		t.Fatalf("missing channel: %+v", res)
	}
}
