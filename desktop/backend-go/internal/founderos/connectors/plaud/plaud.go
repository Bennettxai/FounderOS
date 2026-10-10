// Package plaud ports FounderOS v1's lib/connectors/plaud.ts: the AI voice
// recorder for everything that is not a Zoom call (in-person visits, site
// walks, voice memos). Fathom covers the calls; Plaud covers the room.
//
// Real API, the one @plaud-ai/mcp calls (platform.plaud.ai):
//
//	GET  /developer/api/open/third-party/files/?page=1&page_size=20   Bearer <access>
//	GET  /developer/api/open/third-party/files/{id}   (transcript + AI note)
//	POST /developer/api/oauth/third-party/access-token/refresh   form: refresh_token
//
// Access tokens live hours; the refresh token is the durable credential. It
// resolves, in order, from PLAUD_REFRESH_TOKEN (planted.env, env.local,
// then the process env) and then the MCP's token file ~/.plaud/tokens-mcp.json
// (PLAUD_TOKEN_FILE overrides the path). When Plaud rotates the refresh
// token, the new one is written back to the file it came from (never through a
// symlink), so neither side is left holding a dead credential. That write-back
// is a local file write, not an outbound side effect.
package plaud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "plaud", Name: "Plaud", Kind: connectors.KindKnowledge}

const (
	// API is the Plaud developer API base.
	API             = "https://platform.plaud.ai/developer/api"
	refreshPath     = "/oauth/third-party/access-token/refresh"
	refreshTokenKey = "PLAUD_REFRESH_TOKEN"
	accessTokenKey  = "PLAUD_ACCESS_TOKEN"
	tokenFileEnv    = "PLAUD_TOKEN_FILE"
	skew            = 60 * time.Second

	refreshTimeout = 8 * time.Second
	listTimeout    = 8 * time.Second
	fileTimeout    = 15 * time.Second
)

// ReadPOSTs lists the POST endpoints that are reads for the bridge guard: the
// OAuth refresh mints a token and sends nothing anywhere.
var ReadPOSTs = []string{"platform.plaud.ai/developer/api" + refreshPath}

// ErrNotConfigured is returned by reads when no Plaud credential exists.
var ErrNotConfigured = errors.New("plaud: no PLAUD_REFRESH_TOKEN")

// tokenSet is the MCP token file's shape; field order matches the TS
// JSON.stringify so a write-back reads the same.
type tokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresAt    *int64 `json:"expires_at,omitempty"` // unix ms
}

type tokenSource struct {
	refreshToken string
	cached       *tokenSet
	writeBack    func(tokenSet)
}

type Connector struct {
	res connectors.Resolver
	// BaseURL overrides API (tests).
	BaseURL string
	// TokenFile overrides PLAUD_TOKEN_FILE / ~/.plaud/tokens-mcp.json (tests).
	TokenFile string
	// Now is the clock for token expiry (tests).
	Now    func() time.Time
	client *http.Client

	// mu serialises token resolution so two concurrent reads never refresh
	// (and so rotate) the same refresh token twice.
	mu    sync.Mutex
	cache *tokenSet
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, BaseURL: API, Now: time.Now, client: connectors.HTTPClient(fileTimeout, ReadPOSTs...)}
}

func (c *Connector) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Connector) tokenFilePath() string {
	if c.TokenFile != "" {
		return c.TokenFile
	}
	// Only an explicit PLAUD_TOKEN_FILE: FounderOS never reads another tool's
	// files from the home directory (connectors.CredFiles).
	return os.Getenv(tokenFileEnv)
}

func (c *Connector) readTokenFile() *tokenSet {
	raw, err := os.ReadFile(c.tokenFilePath())
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	refresh, _ := doc["refresh_token"].(string)
	if refresh == "" {
		return nil
	}
	set := &tokenSet{RefreshToken: refresh}
	set.AccessToken, _ = doc["access_token"].(string)
	set.TokenType, _ = doc["token_type"].(string)
	if f, ok := doc["expires_at"].(float64); ok {
		ms := int64(f)
		set.ExpiresAt = &ms
	}
	return set
}

func (c *Connector) fresh(set *tokenSet) bool {
	return set != nil && set.AccessToken != "" && set.ExpiresAt != nil &&
		*set.ExpiresAt-skew.Milliseconds() > c.now().UnixMilli()
}

func (c *Connector) resolveSource() *tokenSource {
	if envRefresh := c.res.Resolve(refreshTokenKey); envRefresh != "" {
		src := &tokenSource{
			refreshToken: envRefresh,
			writeBack: func(set tokenSet) {
				if path := c.refreshTokenHome(); set.RefreshToken != envRefresh && path != "" {
					// A symlinked or read-only file still leaves the in-memory session.
					_ = upsertEnvFile(path, map[string]string{refreshTokenKey: set.RefreshToken})
				}
			},
		}
		if access := c.res.Resolve(accessTokenKey); access != "" {
			// No expiry is known, so fresh() never trusts it: as in the TS.
			src.cached = &tokenSet{AccessToken: access, RefreshToken: envRefresh}
		}
		return src
	}
	if file := c.readTokenFile(); file != nil {
		path := c.tokenFilePath()
		return &tokenSource{
			refreshToken: file.RefreshToken,
			cached:       file,
			writeBack: func(set tokenSet) {
				raw, err := json.MarshalIndent(set, "", "  ")
				if err == nil {
					// A read-only disk still leaves the in-memory session.
					_ = os.WriteFile(path, raw, 0o600)
				}
			},
		}
	}
	return nil
}

// refreshTokenHome is the env file a rotated refresh token belongs in: the one
// the Resolver read it from (planted.env before env.local, as in Resolve), so
// the next resolve sees the new token instead of the revoked one. A token from
// the process env goes to env.local, as in the TS.
func (c *Connector) refreshTokenHome() string {
	for _, path := range []string{c.res.Planted, c.res.EnvLocal} {
		if path == "" {
			continue
		}
		if raw, err := os.ReadFile(path); err == nil && connectors.ParseEnvFile(string(raw))[refreshTokenKey] != "" {
			return path
		}
	}
	return c.res.EnvLocal
}

// Configured reports whether a Plaud credential exists (not whether it works).
func (c *Connector) Configured() bool { return c.resolveSource() != nil }

func (c *Connector) refresh(ctx context.Context, refreshToken string) (tokenSet, error) {
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	form := url.Values{"refresh_token": {refreshToken}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+refreshPath, strings.NewReader(form))
	if err != nil {
		return tokenSet{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return tokenSet{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return tokenSet{}, fmt.Errorf("token refresh HTTP %d", res.StatusCode)
	}
	var data map[string]any
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return tokenSet{}, fmt.Errorf("token refresh unreadable: %w", err)
	}
	access, _ := data["access_token"].(string)
	if access == "" {
		return tokenSet{}, errors.New("token refresh returned no access_token")
	}
	set := tokenSet{AccessToken: access, RefreshToken: refreshToken, TokenType: "Bearer"}
	if r, ok := data["refresh_token"].(string); ok && r != "" {
		set.RefreshToken = r
	}
	if t, ok := data["token_type"].(string); ok {
		set.TokenType = t
	}
	expires := c.now().Add(time.Hour).UnixMilli()
	if s, ok := data["expires_in"].(float64); ok {
		expires = c.now().UnixMilli() + int64(s*1000)
	}
	set.ExpiresAt = &expires
	return set, nil
}

// AccessToken returns a usable access token, minting one from the refresh
// token when the cached one is missing or stale. ErrNotConfigured without a
// credential; a rejected refresh is an error so the status can say so.
func (c *Connector) AccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	src := c.resolveSource()
	if src == nil {
		return "", ErrNotConfigured
	}
	if c.fresh(c.cache) && c.cache.RefreshToken == src.refreshToken {
		return c.cache.AccessToken, nil
	}
	if c.fresh(src.cached) {
		c.cache = src.cached
		return c.cache.AccessToken, nil
	}
	// A refresh rotates the refresh token, and prod on the mini shares its
	// lineage: until cutover the bridge must not rotate it out from under prod.
	if !guard.WritesEnabled() {
		return "", errors.New("Plaud access token expired; FounderOS will not rotate the shared refresh token while FOUNDEROS_WRITES=0")
	}
	minted, err := c.refresh(ctx, src.refreshToken)
	if err != nil {
		return "", err
	}
	c.cache = &minted
	src.writeBack(minted)
	return minted.AccessToken, nil
}

// ---- parsing ---------------------------------------------------------------

// Recording is one row of the Recordings tab.
type Recording struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	At              string   `json:"at"` // ISO (UTC)
	DurationMinutes *float64 `json:"durationMinutes"`
}

// Segment is one transcribed stretch of speech.
type Segment struct {
	StartMs float64 `json:"startMs"`
	EndMs   float64 `json:"endMs"`
	Speaker *string `json:"speaker"`
	Text    string  `json:"text"`
}

// File is one recording in full: what Plaud transcribed and what its AI
// wrote about it. Transcribed is false until Plaud has processed the upload.
type File struct {
	Recording
	Transcript  []Segment `json:"transcript"`
	Note        *string   `json:"note"` // Plaud's AI summary, markdown
	Transcribed bool      `json:"transcribed"`
}

var (
	zoned    = regexp.MustCompile(`[zZ]$|[+-]\d\d:\d\d$`)
	fraction = regexp.MustCompile(`\.\d+$`)
)

// toISO makes Plaud's naive UTC stamps ("2026-08-26T02:10:35") real ISO.
func toISO(v any) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return ""
	}
	if zoned.MatchString(s) {
		return s
	}
	return fraction.ReplaceAllString(s, "") + "Z"
}

func jsString(v any, fallback string) string {
	switch x := v.(type) {
	case nil:
		return fallback
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return fmt.Sprint(x)
	}
	return fallback
}

func number(v any) (float64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	return f, err == nil
}

func decode(body []byte) any {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return nil
	}
	return v
}

func recordingOf(f map[string]any) Recording {
	r := Recording{ID: jsString(f["id"], ""), Title: jsString(f["name"], "Untitled recording")}
	at := f["start_at"]
	if at == nil {
		at = f["created_at"]
	}
	r.At = toISO(at)
	if ms, ok := number(f["duration"]); ok {
		d := math.Floor(ms/60000 + 0.5)
		r.DurationMinutes = &d
	}
	return r
}

// ParseFiles maps the list payload to rows. An unrecognised shape is an
// empty list, as in the TS.
func ParseFiles(body []byte) []Recording {
	doc, _ := decode(body).(map[string]any)
	items, _ := doc["data"].([]any)
	out := make([]Recording, 0, len(items))
	for _, raw := range items {
		f, _ := raw.(map[string]any)
		if f == nil {
			f = map[string]any{}
		}
		out = append(out, recordingOf(f))
	}
	return out
}

func findByType(list any, dataType string) map[string]any {
	arr, _ := list.([]any)
	for _, raw := range arr {
		if m, ok := raw.(map[string]any); ok && m["data_type"] == dataType {
			return m
		}
	}
	return nil
}

// ParseFile maps a GET /files/{id} payload. source_list[transaction] holds a
// JSON-encoded array of segments; note_list[auto_sum_note] holds the summary
// markdown. Unknown shapes degrade to "not transcribed"; nil for a non-object.
func ParseFile(body []byte) *File {
	f, ok := decode(body).(map[string]any)
	if !ok {
		return nil
	}
	out := &File{Recording: recordingOf(f), Transcript: []Segment{}}
	if tx := findByType(f["source_list"], "transaction"); tx != nil {
		if content, ok := tx["data_content"].(string); ok && strings.TrimSpace(content) != "" {
			if segs, ok := decode([]byte(content)).([]any); ok {
				for _, raw := range segs {
					seg, _ := raw.(map[string]any)
					text := strings.TrimSpace(jsString(seg["content"], ""))
					if text == "" {
						continue
					}
					s := Segment{Text: text}
					s.StartMs, _ = number(seg["start_time"])
					s.EndMs, _ = number(seg["end_time"])
					if sp, ok := seg["speaker"].(string); ok && strings.TrimSpace(sp) != "" {
						v := strings.TrimSpace(sp)
						s.Speaker = &v
					}
					out.Transcript = append(out.Transcript, s)
				}
			}
		}
	}
	if sum := findByType(f["note_list"], "auto_sum_note"); sum != nil {
		if content, ok := sum["data_content"].(string); ok && strings.TrimSpace(content) != "" {
			note := strings.TrimSpace(content)
			out.Note = &note
		}
	}
	out.Transcribed = len(out.Transcript) > 0
	return out
}

// ---- samples and action items (lib/recordings-format.ts, lib/plaud-ingest.ts)

// The recordings Plaud ships in every new account; the ingest skips them by
// exact title (verified 2026-08-26) and the board labels them "sample".
var sampleTitles = map[string]bool{
	"welcome to plaud.ai": true,
	"how to use plaud":    true,
	"steve jobs & bill gates: a conversation that shaped technology": true,
}

// IsSample reports one of Plaud's three bundled sample recordings.
func IsSample(title string) bool {
	return sampleTitles[strings.ToLower(strings.TrimSpace(title))]
}

var (
	headingRe = regexp.MustCompile(`^#{1,6}\s+(.*)$`)
	todoRe    = regexp.MustCompile(`(?i)action items?|next steps?|to-?dos?|follow-?ups?`)
	bulletRe  = regexp.MustCompile(`^(?:[-*•]|\d+[.)])\s+(.*)$`)
)

// ActionItems returns the bullets (or numbered items) under headings that
// read like a to-do list in Plaud's AI note, verbatim minus bold markers.
func ActionItems(note *string) []string {
	if note == nil || *note == "" {
		return nil
	}
	out := []string{}
	inSection := false
	for _, raw := range strings.Split(*note, "\n") {
		line := strings.TrimSpace(raw)
		if h := headingRe.FindStringSubmatch(line); h != nil {
			inSection = todoRe.MatchString(h[1])
			continue
		}
		if !inSection {
			continue
		}
		if m := bulletRe.FindStringSubmatch(line); m != nil {
			if text := strings.TrimSpace(strings.ReplaceAll(m[1], "**", "")); text != "" {
				out = append(out, text)
			}
		}
	}
	return out
}

// ---- reads ------------------------------------------------------------------

func (c *Connector) get(ctx context.Context, token, path string, timeout time.Duration) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	return res.StatusCode, body, err
}

func (c *Connector) listFiles(ctx context.Context, limit int) ([]Recording, error) {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	// The API floors page_size at 10.
	size := min(100, max(10, limit))
	code, body, err := c.get(ctx, token, fmt.Sprintf("/open/third-party/files/?page=1&page_size=%d", size), listTimeout)
	if err != nil {
		return nil, err
	}
	if code < 200 || code > 299 {
		return nil, fmt.Errorf("HTTP %d", code)
	}
	rows := ParseFiles(body)
	if len(rows) > limit {
		rows = rows[:max(0, limit)]
	}
	return rows, nil
}

// RecentRecordings returns up to limit recordings, newest first. Samples are
// included (the board labels them); use IsSample to skip them. A failure is
// an error here, where the TS reads [].
func (c *Connector) RecentRecordings(ctx context.Context, limit int) ([]Recording, error) {
	return c.listFiles(ctx, limit)
}

// File returns one recording with its transcript and AI note. nil, nil when
// Plaud has no such file.
func (c *Connector) File(ctx context.Context, id string) (*File, error) {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	code, body, err := c.get(ctx, token, "/open/third-party/files/"+url.PathEscape(id), fileTimeout)
	if err != nil {
		return nil, err
	}
	if code == http.StatusNotFound {
		return nil, nil
	}
	if code < 200 || code > 299 {
		return nil, fmt.Errorf("HTTP %d", code)
	}
	return ParseFile(body), nil
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if !c.Configured() {
		s.State = connectors.StateNotConfigured
		s.Detail = "Set PLAUD_REFRESH_TOKEN in ~/.founderos/.env or under API keys to pull recordings, transcripts and AI notes."
		return s
	}
	recordings, err := c.listFiles(ctx, 100)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "Plaud credential is set but the call failed: " + err.Error()
		return s
	}
	plural := "s"
	if len(recordings) == 1 {
		plural = ""
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("Plaud reachable · %d recording%s visible to this account", len(recordings), plural)
	s.Meta = map[string]any{"recordings": len(recordings)}
	return s
}

// upsertEnvFile mirrors lib/creds.ts upsertEnvLocal: update or append
// KEY=value lines, keeping every unrelated line verbatim, mode 0600.
func upsertEnvFile(path string, values map[string]string) error {
	// Never write through a symlink: env.local can point into another
	// checkout's .env.local.
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink; refusing to write through it", path)
	}
	raw, _ := os.ReadFile(path)
	var lines []string
	if len(raw) > 0 {
		lines = strings.Split(string(raw), "\n")
	}
	pending := map[string]string{}
	order := []string{}
	for k, v := range values {
		pending[k] = v
		order = append(order, k)
	}
	for i, line := range lines {
		key := strings.TrimSpace(strings.SplitN(strings.TrimPrefix(strings.TrimSpace(line), "export "), "=", 2)[0])
		if v, ok := pending[key]; ok && key != "" {
			lines[i] = key + "=" + v
			delete(pending, key)
		}
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, k := range order {
		if v, ok := pending[k]; ok {
			lines = append(lines, k+"="+v)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
