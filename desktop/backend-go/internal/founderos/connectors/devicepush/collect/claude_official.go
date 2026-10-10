package collect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// The official Claude plan gauge (5h session % and weekly %), as FounderOS v1
// lib/connectors/claude-usage.ts reads it: one GET to
// api.anthropic.com/api/oauth/usage, a free read. Its token is
// CLAUDE_OAUTH_TOKEN (or the first CLAUDE_OAUTH_TOKEN_n) when set, else
// Claude Code's own login in this Mac's Keychain, borrowed read-only: never
// refreshed, never written, ignored once expired (the CLI refreshes it).
//
// It lives in the collector because the Keychain login is per machine. Only
// the derived numbers travel: utilization %, reset times (the plan name comes
// from ~/.claude.json, no token involved). The token is never logged, stored,
// pushed or returned; errors are swallowed to a nil gauge, so the burn
// estimate never waits on it.

// ClaudeUsageURL is the only Anthropic endpoint the collector calls.
const ClaudeUsageURL = "https://api.anthropic.com/api/oauth/usage"

const (
	claudeGaugeTimeout    = 4 * time.Second  // AbortSignal.timeout(4000)
	claudeGaugeCacheFor   = 60 * time.Second // officialCache: answers, failures and "no token" alike
	keychainTimeout       = 2 * time.Second  // execFileSync(..., { timeout: 2000 })
	claudeKeychainService = "Claude Code-credentials"
	// claudeSessionScan mirrors claudeSessions(): CLAUDE_SESSION_LIMIT (8) + 4.
	claudeSessionScan = 12

	// Window lengths for the two gauges. The TS records windowMinutes 0, which
	// UsageSnapshotSchema (and this port's receiver) refuse as not positive,
	// so a pushed seat carrying it would be rejected; the real lengths are used.
	fiveHourMinutes = 300
	sevenDayMinutes = 7 * 24 * 60

	noteClaudeOfficial = "burn measured from transcripts; limit % is the official gauge from this login"
	noteClaudeEstimate = "local burn estimate; no Claude login on this box to read the official limit %"
)

// TokenSource yields a Claude OAuth token, or "" when there is none.
type TokenSource func(ctx context.Context) string

// ClaudeTokenSource is claudeSessions()[0]?.token ?? keychainClaudeToken():
// the first configured CLAUDE_OAUTH_TOKEN[_n] (trimmed), else the Keychain.
func ClaudeTokenSource(res connectors.Resolver, keychain TokenSource) TokenSource {
	return func(ctx context.Context) string {
		for i := 1; i <= claudeSessionScan; i++ {
			name := "CLAUDE_OAUTH_TOKEN"
			if i > 1 {
				name += "_" + strconv.Itoa(i)
			}
			if v := strings.TrimSpace(res.Resolve(name)); v != "" {
				return v
			}
		}
		if keychain == nil {
			return ""
		}
		return keychain(ctx)
	}
}

// AccessTokenFromKeychainJSON is the access token inside Claude Code's
// Keychain blob, or "" when it is missing, unparseable or already expired.
func AccessTokenFromKeychainJSON(raw string, nowMs int64) string {
	var doc struct {
		ClaudeAiOauth *struct {
			AccessToken any `json:"accessToken"`
			ExpiresAt   any `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal([]byte(raw), &doc) != nil || doc.ClaudeAiOauth == nil {
		return ""
	}
	tok, ok := doc.ClaudeAiOauth.AccessToken.(string)
	if !ok || tok == "" {
		return ""
	}
	if exp, ok := doc.ClaudeAiOauth.ExpiresAt.(float64); ok && exp <= float64(nowMs) {
		return ""
	}
	return tok
}

type cmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

func runOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stderr = nil, nil // stdio: ['ignore', 'pipe', 'ignore']
	return cmd.Output()
}

// KeychainClaudeToken is Claude Code's login on this Mac. macOS only, and
// never under go test (the TS returns null under VITEST the same way).
func KeychainClaudeToken(ctx context.Context) string {
	if testing.Testing() {
		return ""
	}
	return keychainClaudeTokenWith(ctx, runOutput)
}

func keychainClaudeTokenWith(ctx context.Context, run cmdRunner) string {
	if runtime.GOOS != "darwin" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, keychainTimeout)
	defer cancel()
	raw, err := run(ctx, "/usr/bin/security", "find-generic-password", "-s", claudeKeychainService, "-w")
	if err != nil {
		return ""
	}
	return AccessTokenFromKeychainJSON(strings.TrimSpace(string(raw)), time.Now().UnixMilli())
}

// ParseClaudeOfficial reads the usage body: five_hour is the session gauge,
// seven_day the weekly one, each from utilization (else used_percent) and
// resets_at. Nil when neither window carries a number.
func ParseClaudeOfficial(body []byte) *devicepush.Official {
	var doc map[string]any
	if json.Unmarshal(body, &doc) != nil {
		return nil
	}
	win := func(v any, minutes int) *devicepush.OfficialWindow {
		w, _ := v.(map[string]any)
		pct, ok := w["utilization"].(float64)
		if !ok {
			if pct, ok = w["used_percent"].(float64); !ok {
				return nil
			}
		}
		out := &devicepush.OfficialWindow{UsedPercent: pct, WindowMinutes: minutes}
		if r, ok := w["resets_at"].(string); ok {
			out.ResetsAt = &r
		}
		return out
	}
	session, weekly := win(doc["five_hour"], fiveHourMinutes), win(doc["seven_day"], sevenDayMinutes)
	if session == nil && weekly == nil {
		return nil
	}
	return &devicepush.Official{Session: session, Weekly: weekly}
}

// ClaudeGauge reads the official gauge, at most once a minute.
type ClaudeGauge struct {
	URL     string
	Token   TokenSource
	Client  *http.Client
	Timeout time.Duration
	Now     func() time.Time

	mu       sync.Mutex
	cachedAt time.Time
	cached   *devicepush.Official
	hasCache bool
}

func NewClaudeGauge(token TokenSource) *ClaudeGauge {
	return &ClaudeGauge{URL: ClaudeUsageURL, Token: token, Timeout: claudeGaugeTimeout, Now: time.Now}
}

// Read is officialClaudeUsage: the cached value within 60s, else a fresh
// read. A missing token, a refusal, a timeout or an unreadable body are all
// nil, and cached like an answer.
func (g *ClaudeGauge) Read(ctx context.Context) *devicepush.Official {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.Now()
	if g.hasCache && now.Sub(g.cachedAt) < claudeGaugeCacheFor {
		return g.cached
	}
	var v *devicepush.Official
	if !(testing.Testing() && g.URL == ClaudeUsageURL) { // never the real endpoint from a test
		if tok := g.tokenOf(ctx); tok != "" {
			v = g.fetch(ctx, tok)
		}
	}
	g.cached, g.cachedAt, g.hasCache = v, now, true
	return v
}

func (g *ClaudeGauge) tokenOf(ctx context.Context) string {
	if g.Token == nil {
		return ""
	}
	return g.Token(ctx)
}

func (g *ClaudeGauge) fetch(ctx context.Context, token string) *devicepush.Official {
	timeout := g.Timeout
	if timeout <= 0 {
		timeout = claudeGaugeTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.URL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil
	}
	return ParseClaudeOfficial(body)
}

// WithClaudeOfficial puts the gauge on the local seat and says where its
// limit % comes from. "No transcripts" still speaks first, as in the TS.
func WithClaudeOfficial(seat devicepush.SeatUsage, o *devicepush.Official) devicepush.SeatUsage {
	seat.Official = o
	if seat.Note == noteClaudeEstimate {
		if o != nil {
			seat.Note = noteClaudeOfficial
		}
	}
	return seat
}
