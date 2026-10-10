// Package zernio ports FounderOS v1's lib/connectors/zernio.ts: social posting
// and growth reads through Zernio (the Late API), six platforms under
// @founderos.ai.
//
// Credentials: ZERNIO_API_KEY (env.local, process env, ~/.social-media/.env,
// clue-agent/.env.agents). ~/.social-media/config.json supplies the API base
// URLs and the per-platform account map (handle, accountId, followers), which
// is only a fallback: the mini has no config.json at all, and the live
// /v1/accounts counts win wherever they answer.
//
// Reads (all GET, 60s in-memory cache, last good answer on an outage):
// followers per platform (/v1/accounts), recent published posts (/history),
// and posting days (/history?limit=200). Writes (the presigned upload slot
// and the post itself) go through guard.Outbound and are refused while
// FOUNDEROS_WRITES is off.
//
// FounderOS v1's gated demo mode (fake "connected" on shared deploys, seeded
// posts) is deliberately not ported: the bridge never reports a fake state.
package zernio

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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

var Meta = connectors.Meta{ID: "zernio", Name: "Zernio (Social)", Kind: connectors.KindSocial}

const (
	keyName        = "ZERNIO_API_KEY"
	defaultBaseURL = "https://getlate.dev/api"
	defaultV1URL   = "https://zernio.com/api/v1"
	liveTTL        = 60 * time.Second

	statusTimeout  = 4 * time.Second
	readTimeout    = 6 * time.Second
	uploadTimeout  = 10 * time.Second
	publishTimeout = 30 * time.Second
)

// ErrNotConfigured is returned by reads when no key resolves.
var ErrNotConfigured = errors.New("zernio: ZERNIO_API_KEY not configured")

// Account is one platform's handle and follower count. Followers is nil when
// unknown (a config entry without a count), never a fake zero.
type Account struct {
	Platform  string   `json:"platform"`
	Handle    string   `json:"handle,omitempty"`
	Followers *float64 `json:"followers,omitempty"`
}

// Accounts is an ordered platform → account map (payload or file order, as
// a JS object keeps it).
type Accounts []Account

func (a *Accounts) set(acct Account) {
	for i := range *a {
		if (*a)[i].Platform == acct.Platform {
			(*a)[i] = acct
			return
		}
	}
	*a = append(*a, acct)
}

// Get returns the account for a platform.
func (a Accounts) Get(platform string) (Account, bool) {
	for _, acct := range a {
		if acct.Platform == platform {
			return acct, true
		}
	}
	return Account{}, false
}

type Connector struct {
	res connectors.Resolver
	// ConfigPath overrides ~/.social-media/config.json (tests).
	ConfigPath string
	// CredFiles are the fallback env files, in order. nil means FounderOS v1's
	// order: ~/.social-media/.env, then clue-agent/.env.agents.
	CredFiles []string
	// Now is the cache clock (tests).
	Now    func() time.Time
	client *http.Client

	mu        sync.Mutex
	liveCache *cached[Accounts]
	daysCache *cached[[]PostDay]
	postCache *cached[[]Post]
}

type cached[T any] struct {
	at   time.Time
	data T
}

func New(res connectors.Resolver) *Connector {
	return &Connector{res: res, Now: time.Now, client: connectors.HTTPClient(publishTimeout)}
}

func (c *Connector) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Connector) key() string {
	files := c.CredFiles
	if files == nil {
		social, clue, _, _ := connectors.CredFiles()
		files = []string{social, clue}
	}
	return c.res.Resolve(keyName, files...)
}

type config struct {
	BaseURL  string
	V1URL    string
	Accounts Accounts
}

func (c *Connector) configPath() string {
	if c.ConfigPath != "" {
		return c.ConfigPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".social-media", "config.json")
}

func (c *Connector) readConfig() config {
	cfg := config{BaseURL: defaultBaseURL, V1URL: defaultV1URL}
	raw, err := os.ReadFile(c.configPath())
	if err != nil {
		return cfg
	}
	var doc struct {
		BaseURL  string          `json:"baseUrl"`
		V1URL    string          `json:"v1Url"`
		Accounts json.RawMessage `json:"accounts"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return cfg
	}
	if doc.BaseURL != "" {
		cfg.BaseURL = doc.BaseURL
	}
	if doc.V1URL != "" {
		cfg.V1URL = doc.V1URL
	}
	cfg.Accounts, _ = parseAccountsConfig(doc.Accounts)
	return cfg
}

func finiteNonNeg(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok && !math.IsInf(f, 0) && !math.IsNaN(f) && f >= 0
}

// parseAccountsConfig reads config.json's accounts object in file order.
func parseAccountsConfig(raw json.RawMessage) (Accounts, error) {
	out := Accounts{}
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return out, errors.New("accounts is not an object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return out, err
		}
		platform, _ := tok.(string)
		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			return out, err
		}
		acct := Account{Platform: platform}
		acct.Handle, _ = entry["handle"].(string)
		if f, ok := entry["followers"].(float64); ok {
			acct.Followers = &f
		}
		out.set(acct)
	}
	return out, nil
}

// ConfigAccounts is the static account map from ~/.social-media/config.json.
// It goes stale; prefer LiveAccounts.
func (c *Connector) ConfigAccounts() Accounts { return c.readConfig().Accounts }

// ---- live followers ----------------------------------------------------------

func pickFollowers(a map[string]any) (float64, bool) {
	md, _ := a["metadata"].(map[string]any)
	profile, _ := md["profileData"].(map[string]any)
	user, _ := md["userProfile"].(map[string]any)
	top, _ := a["profileData"].(map[string]any)
	var page map[string]any
	if pages, ok := md["availablePages"].([]any); ok && len(pages) > 0 {
		page, _ = pages[0].(map[string]any)
	}
	for _, v := range []any{profile["followersCount"], user["followersCount"], top["followersCount"], page["fan_count"]} {
		if f, ok := finiteNonNeg(v); ok {
			return f, true
		}
	}
	return 0, false
}

// ParseLiveAccounts maps a /v1/accounts payload to per-platform handles and
// live follower counts. Accounts without a resolvable count are dropped,
// never reported as zero.
func ParseLiveAccounts(body []byte) Accounts {
	out := Accounts{}
	var doc struct {
		Accounts []any `json:"accounts"`
	}
	if json.Unmarshal(body, &doc) != nil {
		return out
	}
	for _, raw := range doc.Accounts {
		a, _ := raw.(map[string]any)
		platform, _ := a["platform"].(string)
		if platform == "" {
			continue
		}
		f, ok := pickFollowers(a)
		if !ok {
			continue
		}
		acct := Account{Platform: platform, Followers: &f}
		if u, ok := a["username"].(string); ok && u != "" {
			acct.Handle = "@" + u
		}
		out.set(acct)
	}
	return out
}

// Summary is the connections card: platform count, follower total, handle.
type Summary struct {
	Platforms int     `json:"platforms"`
	Followers float64 `json:"followers"`
	Handle    *string `json:"handle"`
}

// Summarize prefers live counts and falls back to the static config map.
func Summarize(live, fallback Accounts) Summary {
	source := fallback
	if len(live) > 0 {
		source = live
	}
	s := Summary{Platforms: len(source)}
	for _, a := range source {
		if a.Followers != nil {
			s.Followers += *a.Followers
		}
	}
	if ig, ok := source.Get("instagram"); ok && ig.Handle != "" {
		h := ig.Handle
		s.Handle = &h
	} else {
		for _, a := range source {
			if a.Handle != "" {
				h := a.Handle
				s.Handle = &h
				break
			}
		}
	}
	return s
}

func (c *Connector) get(ctx context.Context, key, rawURL string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 32<<20))
}

// cachedRead serves a fresh cache entry, else fetches; on failure it serves
// the last good entry whatever its age, else the error.
func cachedRead[T any](c *Connector, slot **cached[T], fetch func(key string) (T, error)) (T, error) {
	var zero T
	c.mu.Lock()
	if *slot != nil && c.now().Sub((*slot).at) < liveTTL {
		data := (*slot).data
		c.mu.Unlock()
		return data, nil
	}
	c.mu.Unlock()
	key := c.key()
	if key == "" {
		return zero, ErrNotConfigured
	}
	data, err := fetch(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if *slot != nil {
			return (*slot).data, nil
		}
		return zero, err
	}
	*slot = &cached[T]{at: c.now(), data: data}
	return data, nil
}

// LiveAccounts returns live follower counts from Zernio (60s cache, 6s
// timeout, last good answer on an outage). With nothing cached, an outage is
// an error: unknown, not "no followers".
func (c *Connector) LiveAccounts(ctx context.Context) (Accounts, error) {
	return cachedRead(c, &c.liveCache, func(key string) (Accounts, error) {
		body, err := c.get(ctx, key, c.readConfig().V1URL+"/accounts", readTimeout)
		if err != nil {
			return nil, err
		}
		return ParseLiveAccounts(body), nil
	})
}

// ---- posts -------------------------------------------------------------------

// Post is one published post. Engagement is absent on purpose: it lives
// behind Late's paid analytics add-on, so it is never invented.
type Post struct {
	Platform    string  `json:"platform"`
	Caption     string  `json:"caption"`
	URL         string  `json:"url"`
	PublishedAt *string `json:"publishedAt"`
	Status      string  `json:"status"`
}

// PostDay is one post's date with its full cross-post platform list.
type PostDay struct {
	Date      string   `json:"date"`
	Platforms []string `json:"platforms"`
}

func historyEntries(body []byte) []map[string]any {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		obj, _ := v.(map[string]any)
		if arr, ok = obj["posts"].([]any); !ok {
			return nil
		}
	}
	out := make([]map[string]any, 0, len(arr))
	for _, raw := range arr {
		e, _ := raw.(map[string]any)
		if e == nil {
			e = map[string]any{}
		}
		out = append(out, e)
	}
	return out
}

func stampOf(e map[string]any) *string {
	if s, ok := e["created"].(string); ok {
		return &s
	}
	if s, ok := e["scheduleDate"].(string); ok {
		return &s
	}
	return nil
}

// ParseHistory maps a /history (or /v1/posts) payload to up to limit posts,
// taking the first platform's live post URL.
func ParseHistory(body []byte, limit int) []Post {
	entries := historyEntries(body)
	if len(entries) > limit {
		entries = entries[:max(0, limit)]
	}
	out := make([]Post, 0, len(entries))
	for _, e := range entries {
		ids, _ := e["postIds"].([]any)
		var primary map[string]any
		for _, raw := range ids {
			if p, ok := raw.(map[string]any); ok && p["postUrl"] != nil && p["postUrl"] != "" && p["postUrl"] != false {
				primary = p
				break
			}
		}
		if primary == nil && len(ids) > 0 {
			primary, _ = ids[0].(map[string]any)
		}
		platform := "unknown"
		if s, ok := primary["platform"].(string); ok && s != "" {
			platform = s
		} else if ps, ok := e["platforms"].([]any); ok && len(ps) > 0 {
			if s, ok := ps[0].(string); ok && s != "" {
				platform = s
			}
		}
		p := Post{Platform: platform, PublishedAt: stampOf(e), Status: "unknown"}
		if s, ok := e["post"].(string); ok {
			p.Caption = s
		} else if s, ok := e["content"].(string); ok {
			p.Caption = s
		}
		p.URL, _ = primary["postUrl"].(string)
		if s, ok := e["status"].(string); ok {
			p.Status = s
		}
		out = append(out, p)
	}
	return out
}

// ParsePostDays maps /history to one {date, platforms} per post. Posts with
// no date or no platforms are skipped.
func ParsePostDays(body []byte) []PostDay {
	out := []PostDay{}
	for _, e := range historyEntries(body) {
		stamp := stampOf(e)
		ps, _ := e["platforms"].([]any)
		platforms := []string{}
		for _, p := range ps {
			if s, ok := p.(string); ok {
				platforms = append(platforms, s)
			}
		}
		if stamp == nil || *stamp == "" || len(platforms) == 0 {
			continue
		}
		date := *stamp
		if len(date) > 10 {
			date = date[:10]
		}
		out = append(out, PostDay{Date: date, Platforms: platforms})
	}
	return out
}

// PostDays is the full posting history (one entry per post; the endpoint
// returns the whole set, no pagination), 60s-cached. An error means unknown
// (no key, or Zernio did not answer and nothing is cached), so a page never
// reads an outage as a quiet stretch. This is the TS zernioPostDaysKnown.
func (c *Connector) PostDays(ctx context.Context) ([]PostDay, error) {
	return cachedRead(c, &c.daysCache, func(key string) ([]PostDay, error) {
		body, err := c.get(ctx, key, c.readConfig().BaseURL+"/history?limit=200", readTimeout)
		if err != nil {
			return nil, err
		}
		return ParsePostDays(body), nil
	})
}

// RecentPosts returns up to limit recent published posts (24 cached for 60s).
func (c *Connector) RecentPosts(ctx context.Context, limit int) ([]Post, error) {
	posts, err := cachedRead(c, &c.postCache, func(key string) ([]Post, error) {
		body, err := c.get(ctx, key, c.readConfig().BaseURL+"/history", readTimeout)
		if err != nil {
			return nil, err
		}
		return ParseHistory(body, 24), nil
	})
	if err != nil {
		return nil, err
	}
	if len(posts) > limit {
		posts = posts[:max(0, limit)]
	}
	return posts, nil
}

// ---- status -----------------------------------------------------------------

func (c *Connector) Status(ctx context.Context) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	key := c.key()
	cfg := c.readConfig()
	if key == "" {
		s.State = connectors.StateNotConfigured
		s.Detail = connectors.NotFound("ZERNIO_API_KEY")
		return s
	}
	body, err := c.get(ctx, key, cfg.V1URL+"/accounts", statusTimeout)
	if err != nil {
		s.State = connectors.StateError
		s.Detail = "Key found but API check failed: " + err.Error()
		s.Meta = map[string]any{"platforms": len(cfg.Accounts)}
		return s
	}
	sum := Summarize(ParseLiveAccounts(body), cfg.Accounts)
	handle := ""
	if sum.Handle != nil {
		handle = " (" + *sum.Handle + ")"
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%d platforms%s · %s total followers", sum.Platforms, handle, formatEnUS(sum.Followers))
	s.Meta = map[string]any{"platforms": sum.Platforms, "followers": sum.Followers}
	return s
}

// formatEnUS mirrors Number.toLocaleString('en-US'): thousands separators
// and at most three fraction digits.
func formatEnUS(f float64) string {
	neg := f < 0
	str := strconv.FormatFloat(math.Abs(f), 'f', 3, 64)
	intPart, frac, _ := strings.Cut(str, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

// ---- writes (guarded) ---------------------------------------------------------

// UploadSlot is a presigned upload from Late: PUT the bytes to UploadURL,
// then post AccessURL.
type UploadSlot struct {
	UploadURL string `json:"uploadUrl"`
	AccessURL string `json:"accessUrl"`
}

// UploadTarget asks Late for a presigned upload slot. It is a GET, but it
// exists only to publish, so it is guarded like the post itself.
func (c *Connector) UploadTarget(ctx context.Context, fileName, contentType string) (UploadSlot, error) {
	var slot UploadSlot
	err := guard.Outbound("zernio.upload_url", func() error {
		key := c.key()
		if key == "" {
			return errors.New("ZERNIO_API_KEY not configured")
		}
		q := url.Values{"fileName": {fileName}, "contentType": {contentType}}
		body, err := c.get(ctx, key, c.readConfig().BaseURL+"/media/uploadUrl?"+q.Encode(), uploadTimeout)
		if err != nil {
			return fmt.Errorf("uploadUrl failed: %w", err)
		}
		if json.Unmarshal(body, &slot) != nil || slot.UploadURL == "" || slot.AccessURL == "" {
			return errors.New("uploadUrl response missing fields")
		}
		return nil
	})
	return slot, err
}

// PublishInput is one post. ScheduledFor nil publishes now.
type PublishInput struct {
	Caption      string
	MediaURLs    []string
	Platforms    []string
	ScheduledFor *string
}

var videoURL = regexp.MustCompile(`(?i)\.(mp4|mov|webm|m4v)(\?|$)`)

// Publish creates the post for real (POST /post): now, or at ScheduledFor.
// Refused with guard.ErrWritesDisabled while FOUNDEROS_WRITES is off.
func (c *Connector) Publish(ctx context.Context, in PublishInput) error {
	return guard.Outbound("zernio.post", func() error {
		key := c.key()
		if key == "" {
			return errors.New("ZERNIO_API_KEY not configured")
		}
		isVideo := false
		for _, u := range in.MediaURLs {
			if videoURL.MatchString(u) {
				isVideo = true
			}
		}
		media := in.MediaURLs
		if media == nil {
			media = []string{}
		}
		payload := map[string]any{"post": in.Caption, "mediaUrls": media, "platforms": in.Platforms, "isVideo": isVideo}
		if in.ScheduledFor != nil && *in.ScheduledFor != "" {
			payload["scheduledFor"] = *in.ScheduledFor
		} else {
			payload["publishNow"] = true
		}
		for _, p := range in.Platforms {
			if p == "youtube" {
				title := []rune(in.Caption)
				if len(title) > 95 {
					title = title[:95]
				}
				payload["youTubeOptions"] = map[string]any{"title": string(title), "visibility": "public", "shorts": isVideo}
			}
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(ctx, publishTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.readConfig().BaseURL+"/post", bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode > 299 {
			text, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
			msg := []rune(string(text))
			if len(msg) > 200 {
				msg = msg[:200]
			}
			return fmt.Errorf("HTTP %d: %s", res.StatusCode, string(msg))
		}
		return nil
	})
}
