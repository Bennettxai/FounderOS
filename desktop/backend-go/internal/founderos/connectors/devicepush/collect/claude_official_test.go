package collect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// The official Claude plan gauge (FounderOS v1 lib/connectors/claude-usage.ts
// officialClaudeUsage / officialUsageForToken / keychainClaudeToken). Every
// test here talks to an httptest server or a fake Keychain: the real
// endpoint and the real Keychain are never touched.

// A token shaped like Claude Code's, so a leak would be unmistakable.
const fakeClaudeToken = "sk-ant-oat01-NEVER-LEAVE-THIS-MAC-0123456789abcdef"

// ---- Keychain blob (accessTokenFromKeychainJson) ----------------------------

func TestAccessTokenFromKeychainJSON(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC).UnixMilli()
	blob := func(o map[string]any) string {
		raw, _ := json.Marshal(map[string]any{"claudeAiOauth": o})
		return string(raw)
	}
	if got := AccessTokenFromKeychainJSON(blob(map[string]any{"accessToken": "tok-1", "expiresAt": now + 60_000}), now); got != "tok-1" {
		t.Errorf("a live token = %q, want tok-1", got)
	}
	if got := AccessTokenFromKeychainJSON(blob(map[string]any{"accessToken": "tok-1", "expiresAt": now - 1}), now); got != "" {
		t.Errorf("an expired token is ignored (the CLI refreshes it; we never do): %q", got)
	}
	if got := AccessTokenFromKeychainJSON(blob(map[string]any{"accessToken": "tok-1", "expiresAt": now}), now); got != "" {
		t.Errorf("expiresAt <= now is expired: %q", got)
	}
	if got := AccessTokenFromKeychainJSON(blob(map[string]any{"accessToken": "tok-2"}), now); got != "tok-2" {
		t.Errorf("no expiresAt is taken as live, as the TS does: %q", got)
	}
	for _, raw := range []string{"", "not json", `{}`, blob(map[string]any{"expiresAt": now + 1}), blob(map[string]any{"accessToken": ""}), blob(map[string]any{"accessToken": 7})} {
		if got := AccessTokenFromKeychainJSON(raw, now); got != "" {
			t.Errorf("%q -> %q, want none", raw, got)
		}
	}
}

// ---- token source order (claudeSessions()[0]?.token ?? keychainClaudeToken()) -

func TestClaudeTokenEnvComesBeforeTheKeychain(t *testing.T) {
	keychainCalls := 0
	keychain := func(context.Context) string { keychainCalls++; return "from-keychain" }

	src := ClaudeTokenSource(envResolver(t, "CLAUDE_OAUTH_TOKEN=  from-env  "), keychain)
	if got := src(context.Background()); got != "from-env" || keychainCalls != 0 {
		t.Fatalf("CLAUDE_OAUTH_TOKEN wins, trimmed, without touching the Keychain: %q (keychain calls %d)", got, keychainCalls)
	}
	// claudeSessions()[0] is the FIRST configured session, whichever suffix it has
	src = ClaudeTokenSource(envResolver(t, "CLAUDE_OAUTH_TOKEN_3=third", "CLAUDE_OAUTH_TOKEN_2=second"), keychain)
	if got := src(context.Background()); got != "second" {
		t.Fatalf("first configured session = %q, want second", got)
	}
	src = ClaudeTokenSource(envResolver(t), keychain)
	if got := src(context.Background()); got != "from-keychain" || keychainCalls != 1 {
		t.Fatalf("no env token falls back to the Keychain: %q (%d)", got, keychainCalls)
	}
	src = ClaudeTokenSource(envResolver(t), nil)
	if got := src(context.Background()); got != "" {
		t.Fatalf("no env and no Keychain reader = %q", got)
	}
}

func TestKeychainReadsClaudeCodesEntryReadOnly(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the Keychain read is macOS only")
	}
	var gotName string
	var gotArgs []string
	var gotTimeout time.Duration
	live := time.Now().Add(time.Hour).UnixMilli()
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		gotName, gotArgs = name, args
		if dl, ok := ctx.Deadline(); ok {
			gotTimeout = time.Until(dl)
		}
		raw, _ := json.Marshal(map[string]any{"claudeAiOauth": map[string]any{"accessToken": fakeClaudeToken, "expiresAt": live, "refreshToken": "rt-never-used"}})
		return append(raw, '\n'), nil
	}
	if got := keychainClaudeTokenWith(context.Background(), run); got != fakeClaudeToken {
		t.Fatalf("token = %q", got)
	}
	if gotName != "/usr/bin/security" || strings.Join(gotArgs, " ") != "find-generic-password -s Claude Code-credentials -w" {
		t.Fatalf("command = %s %q", gotName, gotArgs)
	}
	if gotTimeout <= 0 || gotTimeout > 2*time.Second {
		t.Fatalf("the Keychain read has its own 2s budget, got %s", gotTimeout)
	}
	failing := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("item not found") }
	if got := keychainClaudeTokenWith(context.Background(), failing); got != "" {
		t.Fatalf("no Keychain entry = %q", got)
	}
}

func TestTheRealKeychainAndEndpointAreNeverTouchedUnderTest(t *testing.T) {
	// the TS returns null under VITEST; the Go collector does the same under go test
	if got := KeychainClaudeToken(context.Background()); got != "" {
		t.Fatal("KeychainClaudeToken read the real Keychain inside a test")
	}
	g := NewClaudeGauge(func(context.Context) string { return fakeClaudeToken })
	if g.URL != ClaudeUsageURL {
		t.Fatalf("default URL = %q", g.URL)
	}
	if o := g.Read(context.Background()); o != nil {
		t.Fatalf("the real endpoint answered inside a test: %+v", o)
	}
}

// ---- response parsing (officialUsageForToken's win()) ---------------------------

func TestParseClaudeOfficial(t *testing.T) {
	o := ParseClaudeOfficial([]byte(`{
		"five_hour": {"utilization": 37.5, "resets_at": "2026-09-25T17:00:00.000Z"},
		"seven_day": {"utilization": 62, "resets_at": "2026-09-30T09:00:00.000Z"},
		"seven_day_opus": {"utilization": 99}
	}`))
	if o == nil || o.Session == nil || o.Weekly == nil {
		t.Fatalf("official = %+v", o)
	}
	if o.Session.UsedPercent != 37.5 || *o.Session.ResetsAt != "2026-09-25T17:00:00.000Z" || o.Session.WindowMinutes != 300 {
		t.Errorf("session = %+v", *o.Session)
	}
	if o.Weekly.UsedPercent != 62 || *o.Weekly.ResetsAt != "2026-09-30T09:00:00.000Z" || o.Weekly.WindowMinutes != 10080 {
		t.Errorf("weekly = %+v", *o.Weekly)
	}

	// used_percent is the fallback; a non-string resets_at is null
	o = ParseClaudeOfficial([]byte(`{"five_hour": {"used_percent": 12, "resets_at": 1790000000}, "seven_day": null}`))
	if o == nil || o.Session == nil || o.Session.UsedPercent != 12 || o.Session.ResetsAt != nil || o.Weekly != nil {
		t.Fatalf("fallback = %+v", o)
	}
	// utilization wins over used_percent when both are numbers
	if o = ParseClaudeOfficial([]byte(`{"seven_day": {"utilization": 5, "used_percent": 50}}`)); o == nil || o.Weekly.UsedPercent != 5 || o.Session != nil {
		t.Fatalf("utilization first = %+v", o)
	}
	// no number in either window is no gauge at all
	for _, body := range []string{`{}`, `{"five_hour": {"utilization": "37"}}`, `{"five_hour": {}, "seven_day": {"resets_at": "x"}}`, `not json`, `[]`} {
		if o := ParseClaudeOfficial([]byte(body)); o != nil {
			t.Errorf("%s -> %+v, want nil", body, o)
		}
	}
}

// ---- the call itself: GET, headers, 4s timeout, 60s cache ------------------------

func gaugeServer(t *testing.T, status int, body string, delay time.Duration) (*httptest.Server, *atomic.Int32, *http.Request) {
	t.Helper()
	var hits atomic.Int32
	seen := &http.Request{Header: http.Header{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		seen.Method, seen.URL = r.Method, r.URL
		seen.Header = r.Header.Clone()
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits, seen
}

const gaugeBody = `{"five_hour":{"utilization":37,"resets_at":"2026-09-29T15:00:00.000Z"},"seven_day":{"utilization":64,"resets_at":"2026-10-02T09:00:00.000Z"}}`

func testGauge(url string, token string, clock *time.Time) *ClaudeGauge {
	g := NewClaudeGauge(func(context.Context) string { return token })
	g.URL = url
	g.Now = func() time.Time { return *clock }
	return g
}

func TestClaudeGaugeMirrorsTheCallAndCachesForAMinute(t *testing.T) {
	srv, hits, seen := gaugeServer(t, 200, gaugeBody, 0)
	clock := now
	g := testGauge(srv.URL+"/api/oauth/usage", fakeClaudeToken, &clock)

	o := g.Read(context.Background())
	if o == nil || o.Session.UsedPercent != 37 || o.Weekly.UsedPercent != 64 {
		t.Fatalf("official = %+v", o)
	}
	if seen.Method != http.MethodGet || seen.URL.Path != "/api/oauth/usage" {
		t.Fatalf("request = %s %s", seen.Method, seen.URL)
	}
	if seen.Header.Get("Authorization") != "Bearer "+fakeClaudeToken || seen.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
		t.Fatalf("headers = %v", seen.Header)
	}
	if g.Timeout != 4*time.Second {
		t.Fatalf("the HTTP call has its own 4s budget, got %s", g.Timeout)
	}

	clock = now.Add(59 * time.Second)
	if o := g.Read(context.Background()); o == nil || hits.Load() != 1 {
		t.Fatalf("a read within 60s comes from the cache (hits %d)", hits.Load())
	}
	clock = now.Add(61 * time.Second)
	if g.Read(context.Background()); hits.Load() != 2 {
		t.Fatalf("after 60s the gauge is read again (hits %d)", hits.Load())
	}
}

func TestClaudeGaugeFailsToNilAndBacksOffForAMinute(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, `{"error":"token expired"}`},
		{"rate limited", 429, ``},
		{"garbage", 200, `<html>`},
		{"no windows", 200, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, hits, _ := gaugeServer(t, tc.status, tc.body, 0)
			clock := now
			g := testGauge(srv.URL, fakeClaudeToken, &clock)
			if o := g.Read(context.Background()); o != nil {
				t.Fatalf("official = %+v, want nil", o)
			}
			clock = now.Add(30 * time.Second)
			if g.Read(context.Background()); hits.Load() != 1 {
				t.Fatalf("a failure is cached for 60s too (hits %d)", hits.Load())
			}
		})
	}
}

func TestClaudeGaugeWithoutATokenAsksNobody(t *testing.T) {
	srv, hits, _ := gaugeServer(t, 200, gaugeBody, 0)
	clock := now
	tokenReads := 0
	g := NewClaudeGauge(func(context.Context) string { tokenReads++; return "" })
	g.URL, g.Now = srv.URL, func() time.Time { return clock }
	if o := g.Read(context.Background()); o != nil || hits.Load() != 0 {
		t.Fatalf("no token: official = %+v, hits %d", o, hits.Load())
	}
	clock = now.Add(10 * time.Second)
	if g.Read(context.Background()); tokenReads != 1 {
		t.Fatalf("no token is cached like any other answer (token reads %d)", tokenReads)
	}
}

func TestClaudeGaugeGivesUpAtItsOwnTimeout(t *testing.T) {
	srv, _, _ := gaugeServer(t, 200, gaugeBody, 5*time.Second)
	clock := now
	g := testGauge(srv.URL, fakeClaudeToken, &clock)
	g.Timeout = 150 * time.Millisecond
	start := time.Now()
	if o := g.Read(context.Background()); o != nil {
		t.Fatalf("a slow gauge is nil, got %+v", o)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("Read waited %s past its timeout", took)
	}
	// and the caller's context (the collector's source budget) bounds it too
	g2 := testGauge(srv.URL, fakeClaudeToken, &clock)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start = time.Now()
	if o := g2.Read(ctx); o != nil || time.Since(start) > 2*time.Second {
		t.Fatalf("the caller's deadline was ignored: %+v after %s", o, time.Since(start))
	}
}

// ---- the seat ---------------------------------------------------------------------

func TestWithClaudeOfficialSetsTheGaugeAndProdsNote(t *testing.T) {
	dir := t.TempDir()
	write(t, dir+"/p/s.jsonl", aline("2026-09-05T12:00:00.000Z", 100))
	seat := ScanClaudeProjects(dir, usageNow, NewScanCache(), SeatInfo{ID: "x", Label: "x"})
	if seat.Note != "local burn estimate; no Claude login on this box to read the official limit %" {
		t.Fatalf("estimate note = %q", seat.Note)
	}
	r := "2026-09-05T17:00:00.000Z"
	o := &devicepush.Official{Session: &devicepush.OfficialWindow{UsedPercent: 37, WindowMinutes: 300, ResetsAt: &r}}
	withGauge := WithClaudeOfficial(seat, o)
	if withGauge.Official != o || withGauge.Note != "burn measured from transcripts; limit % is the official gauge from this login" {
		t.Fatalf("seat = %+v", withGauge)
	}
	if same := WithClaudeOfficial(seat, nil); same.Official != nil || same.Note != seat.Note {
		t.Fatalf("no gauge leaves the estimate as it was: %+v", same)
	}
	empty := ScanClaudeProjects(dir+"/nope", usageNow, NewScanCache(), SeatInfo{ID: "x", Label: "x"})
	if got := WithClaudeOfficial(empty, o); !strings.Contains(got.Note, "no transcripts") || got.Official != o {
		t.Fatalf("no transcripts speaks first, as in the TS: %+v", got)
	}
}

func TestCollectCarriesTheOfficialClaudeGaugeButNeverTheToken(t *testing.T) {
	srv, hits, seen := gaugeServer(t, 200, gaugeBody, 0)
	cfg := fakeMac(t)
	cfg.ClaudeToken = func(context.Context) string { return fakeClaudeToken }
	cfg.ClaudeUsageURL = srv.URL
	c := NewCollector(cfg)

	p := c.Collect(context.Background(), now)
	if err := devicepush.Validate(p); err != nil {
		t.Fatalf("a payload with the official gauge must pass the receiver: %v", err)
	}
	if hits.Load() != 1 || seen.Header.Get("Authorization") != "Bearer "+fakeClaudeToken {
		t.Fatalf("gauge hits = %d", hits.Load())
	}
	claude := p.Usage[0]
	if claude.Kind != "claude" || claude.Official == nil || claude.Official.Session.UsedPercent != 37 || claude.Official.Weekly.UsedPercent != 64 {
		t.Fatalf("claude seat = %+v", claude)
	}
	if claude.Note != "burn measured from transcripts; limit % is the official gauge from this login" {
		t.Errorf("note = %q", claude.Note)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), fakeClaudeToken) || strings.Contains(string(raw), "sk-ant-") {
		t.Fatal("the OAuth token reached the push payload")
	}

	// the next tick inside a minute reuses the reading instead of asking again
	if p2 := c.Collect(context.Background(), now.Add(30*time.Second)); p2.Usage[0].Official == nil || hits.Load() != 1 {
		t.Fatalf("second tick: official = %+v, hits %d", p2.Usage[0].Official, hits.Load())
	}
}

func TestCollectWithoutAClaudeLoginStaysAnEstimate(t *testing.T) {
	srv, hits, _ := gaugeServer(t, 200, gaugeBody, 0)
	cfg := fakeMac(t)
	cfg.ClaudeToken = func(context.Context) string { return "" }
	cfg.ClaudeUsageURL = srv.URL
	p := NewCollector(cfg).Collect(context.Background(), now)
	if hits.Load() != 0 || p.Usage[0].Official != nil {
		t.Fatalf("no login: hits %d official %+v", hits.Load(), p.Usage[0].Official)
	}
	if p.Usage[0].Note != "local burn estimate; no Claude login on this box to read the official limit %" {
		t.Errorf("note = %q", p.Usage[0].Note)
	}
}

func TestDefaultConfigWiresTheClaudeTokenSource(t *testing.T) {
	c := DefaultConfig(envResolver(t, "CLAUDE_OAUTH_TOKEN=from-env"), "/Users/x", "h")
	if c.ClaudeToken == nil || c.ClaudeToken(context.Background()) != "from-env" {
		t.Fatal("DefaultConfig reads CLAUDE_OAUTH_TOKEN through the resolver")
	}
	if c.ClaudeUsageURL != ClaudeUsageURL {
		t.Fatalf("url = %q", c.ClaudeUsageURL)
	}
}
