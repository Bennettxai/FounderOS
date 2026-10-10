package manychat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Fixture shaped like ManyChat's GET /fb/page/getInfo answer.
const pageInfoFixture = `{"status":"success","data":{"id":1234567,"name":"Alex","category":"Entrepreneur","avatar_link":"https://example.invalid/a.jpg","username":"founderos.ai","about":"","description":"","is_pro":true,"timezone":"America/Chicago"}}`

// harness isolates a connector from the real env: an empty env.local, no
// process key, and a ~/.claude.json stand-in the test controls.
type harness struct {
	dir        string
	envLocal   string
	claudeJSON string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	h := &harness{dir: dir, envLocal: filepath.Join(dir, "env.local"), claudeJSON: filepath.Join(dir, "claude.json")}
	t.Setenv("MANYCHAT_API_KEY", "")
	t.Setenv("MANYCHAT_WEBHOOK_SECRET", "")
	return h
}

func (h *harness) setEnvLocal(t *testing.T, body string) {
	t.Helper()
	if err := os.WriteFile(h.envLocal, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (h *harness) connector(baseURL string) *Connector {
	c := New(connectors.Resolver{EnvLocal: h.envLocal})
	c.ClaudeJSON = h.claudeJSON
	if baseURL != "" {
		c.BaseURL = baseURL
	}
	return c
}

var keySeq atomic.Int64

// uniqueKey keeps the package-level status cache from leaking between tests,
// including across -count=N runs in one process.
func uniqueKey(t *testing.T) string {
	return fmt.Sprintf("mc-test-key-%s-%d", t.Name(), keySeq.Add(1))
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "manychat" || Meta.Name != "ManyChat (IG DMs)" || Meta.Kind != connectors.KindSocial {
		t.Fatalf("Meta = %+v", Meta)
	}
	if StatusTTL.Hours() != 3 {
		t.Errorf("StatusTTL = %v, want 3h", StatusTTL)
	}
	if Timeout.Seconds() != 8 {
		t.Errorf("Timeout = %v, want 8s", Timeout)
	}
}

func TestParsePageInfo(t *testing.T) {
	p := ParsePageInfo([]byte(pageInfoFixture))
	if p == nil || p.Name != "Alex" || p.Username != "founderos.ai" || !p.IsPro {
		t.Fatalf("got %+v", p)
	}
	bare := ParsePageInfo([]byte(`{"name":"Alex"}`))
	if bare == nil || bare.Username != "" || bare.IsPro {
		t.Fatalf("bare payload: %+v", bare)
	}
	for _, raw := range []string{`null`, `{}`, `{"data":{}}`, `{"data":{"name":""}}`, `not json`} {
		if got := ParsePageInfo([]byte(raw)); got != nil {
			t.Errorf("%s: want nil, got %+v", raw, got)
		}
	}
}

func TestStatusNotConfiguredNeverCallsOut(t *testing.T) {
	h := newHarness(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.Kind != connectors.KindSocial || s.ID != "manychat" {
		t.Fatalf("status = %+v", s)
	}
	if !strings.Contains(s.Detail, "MANYCHAT_API_KEY") {
		t.Errorf("detail = %q", s.Detail)
	}
	if hits.Load() != 0 {
		t.Errorf("a missing key must never call out, got %d hits", hits.Load())
	}
}

func TestStatusConnectedShowsHandle(t *testing.T) {
	h := newHarness(t)
	key := uniqueKey(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+key+"\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fb/page/getInfo" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		io.WriteString(w, pageInfoFixture)
	}))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("status = %+v", s)
	}
	if s.Detail != "@founderos.ai · Instagram · Pro" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["handle"] != "@founderos.ai" {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusKeyFromClaudeJSONWhenEnvIsEmpty(t *testing.T) {
	h := newHarness(t)
	key := uniqueKey(t)
	cj := `{"mcpServers":{"manychat":{"command":"node","env":{"MANYCHAT_API_KEY":"` + key + `"}}}}`
	if err := os.WriteFile(h.claudeJSON, []byte(cj), 0o600); err != nil {
		t.Fatal(err)
	}
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		io.WriteString(w, `{"data":{"name":"Alex"}}`)
	}))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "Alex · Instagram" {
		t.Fatalf("status = %+v", s)
	}
	if gotAuth != "Bearer "+key {
		t.Errorf("auth = %q", gotAuth)
	}
}

func TestStatusErrorOnRejectedKeyAndNoRetry(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "429") {
		t.Fatalf("status = %+v", s)
	}
	if !strings.HasPrefix(s.Detail, "Key set but API check failed: ") {
		t.Errorf("detail = %q", s.Detail)
	}
	if hits.Load() != 1 {
		t.Errorf("an HTTP answer must never be retried, got %d hits", hits.Load())
	}
}

func TestStatusErrorWhenResponseHasNoAccount(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":"success","data":{}}`)
	}))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "no account info in response") {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusIsCachedInsideTheWindowIncludingFailures(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	var hits atomic.Int32
	fail := atomic.Bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.WriteString(w, pageInfoFixture)
	}))
	defer srv.Close()

	c := h.connector(srv.URL)
	first := c.Status(context.Background())
	fail.Store(true)
	second := h.connector(srv.URL).Status(context.Background()) // a fresh instance shares the cap
	if first.State != connectors.StateConnected || second.State != connectors.StateConnected {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if hits.Load() != 1 {
		t.Errorf("a second check inside the window must not touch the API, got %d hits", hits.Load())
	}

	// A failure is cached too, so an outage cannot hammer the cap.
	h2 := newHarness(t)
	h2.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"-outage\n")
	hits.Store(0)
	a := h2.connector(srv.URL).Status(context.Background())
	b := h2.connector(srv.URL).Status(context.Background())
	if a.State != connectors.StateError || b.State != connectors.StateError {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	if hits.Load() != 1 {
		t.Errorf("a cached failure must not call again, got %d hits", hits.Load())
	}
}

func TestStatusCacheExpiresAfterTTL(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		io.WriteString(w, pageInfoFixture)
	}))
	defer srv.Close()

	c := h.connector(srv.URL)
	now := c.Now()
	c.Now = func() time.Time { return now }
	c.Status(context.Background())
	c.Now = func() time.Time { return now.Add(StatusTTL + time.Second) }
	c.Status(context.Background())
	if hits.Load() != 2 {
		t.Errorf("an expired window must re-check, got %d hits", hits.Load())
	}
}

// A dropped connection is transient: retried once, and two in a row give up.
func TestStatusRetriesTransientFailureOnce(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		io.WriteString(w, pageInfoFixture)
	}))
	defer srv.Close()

	s := h.connector(srv.URL).Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("one transient failure must be retried: %+v", s)
	}
	if hits.Load() != 2 {
		t.Errorf("hits = %d, want 2", hits.Load())
	}
}

func TestStatusTwoTransientFailuresGiveUp(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var hits atomic.Int32
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			hits.Add(1)
			conn.Close()
		}
	}()
	defer ln.Close()

	s := h.connector("http://" + ln.Addr().String()).Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "Key set but API check failed: ") {
		t.Fatalf("status = %+v", s)
	}
	if hits.Load() != 2 {
		t.Errorf("hits = %d, want exactly 2 (one retry, no loop)", hits.Load())
	}
}

func TestSendTextRefusesWithoutKey(t *testing.T) {
	h := newHarness(t)
	r, err := h.connector("http://127.0.0.1:1").SendText(context.Background(), "42", "on it")
	if err == nil || r.OK || !strings.Contains(r.Detail, "MANYCHAT_API_KEY") {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestSendTextRefusedWhileBridgeWritesOff(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()

	r, err := h.connector(srv.URL).SendText(context.Background(), "42", "on it")
	if !errors.Is(err, guard.ErrWritesDisabled) || r.OK {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	if hits.Load() != 0 {
		t.Errorf("a refused send must never reach ManyChat, got %d hits", hits.Load())
	}
	found := false
	for _, ref := range guard.Refused() {
		if ref.Action == "manychat.send_text" {
			found = true
		}
	}
	if !found {
		t.Errorf("refusal not recorded: %v", guard.Refused())
	}
}

func TestSendTextPostsV2PayloadWhenWritesOn(t *testing.T) {
	h := newHarness(t)
	key := uniqueKey(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+key+"\n")
	t.Setenv("FOUNDEROS_WRITES", "1")
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/fb/sending/sendContent" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"status":"success"}`)
	}))
	defer srv.Close()

	r, err := h.connector(srv.URL).SendText(context.Background(), "42", "on it")
	if err != nil || !r.OK || r.Detail != "sent" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	if body["subscriber_id"] != "42" || body["message_tag"] != "ACCOUNT_UPDATE" {
		t.Errorf("body = %v", body)
	}
	data := body["data"].(map[string]any)
	content := data["content"].(map[string]any)
	msgs := content["messages"].([]any)
	if data["version"] != "v2" || content["type"] != "instagram" || len(msgs) != 1 {
		t.Fatalf("data = %v", data)
	}
	m := msgs[0].(map[string]any)
	if m["type"] != "text" || m["text"] != "on it" {
		t.Errorf("message = %v", m)
	}
}

func TestSendTextReportsNon2xxHonestly(t *testing.T) {
	h := newHarness(t)
	h.setEnvLocal(t, "MANYCHAT_API_KEY="+uniqueKey(t)+"\n")
	t.Setenv("FOUNDEROS_WRITES", "1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	r, err := h.connector(srv.URL).SendText(context.Background(), "42", "on it")
	if err == nil || r.OK || r.Detail != "ManyChat send failed: HTTP 400" {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}
