package zernio

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// Mirrors the Zernio/Late /v1/accounts payload: live followers sit at
// metadata.profileData.followersCount; Facebook pages carry fan_count.
const accountsBody = `{"accounts": [
  {"platform": "instagram", "username": "founderos.ai", "metadata": {"profileData": {"followersCount": 41200}}},
  {"platform": "tiktok", "username": "founderos.ai", "metadata": {"profileData": {"followersCount": 8300}}},
  {"platform": "youtube", "username": "founderosai", "metadata": {"profileData": {"followersCount": 950}}},
  {"platform": "twitter", "username": "Founderosai", "metadata": {"userProfile": {"followersCount": 3100}}},
  {"platform": "linkedin", "username": "Alex", "profileData": {"followersCount": 1500}},
  {"platform": "facebook", "username": "Alex", "metadata": {"availablePages": [{"fan_count": 42}]}},
  {"platform": "pinterest", "username": "x", "metadata": {"profileData": {}}},
  {"platform": "threads", "metadata": {"profileData": {"followersCount": -1}}},
  {"username": "no-platform", "metadata": {"profileData": {"followersCount": 9}}}
]}`

// Mirrors the /history shape: published posts with live post URLs.
const historyBody = `[
  {"id": "6a32c34d", "post": "Comment \"loop\" and I'll send you the guide.", "platforms": ["instagram", "tiktok", "youtube"],
   "status": "success", "created": "2026-06-17T15:54:53.452Z",
   "postIds": [{"status": "success", "platform": "instagram", "postUrl": "https://www.instagram.com/reel/DZsVsK_DaTn/"}]},
  {"id": "b2", "content": "second post", "platforms": ["twitter"], "status": "success", "created": "2026-06-15T10:00:00.000Z",
   "postIds": [{"status": "success", "platform": "twitter", "postUrl": "https://x.com/Founderosai/status/1"}]},
  {"platforms": ["tiktok"], "scheduleDate": "2026-06-10T09:00:00.000Z"},
  {"platforms": ["instagram"]},
  {"created": "2026-06-09T00:00:00.000Z", "platforms": []}
]`

const configAccounts = `{
  "instagram": {"handle": "@founderos.ai", "accountId": "acc_ig", "followers": 50000, "requiresMedia": true},
  "tiktok": {"handle": "@founderos.ai", "accountId": "acc_tt", "followers": 9000},
  "linkedin": {"accountId": "acc_li"}
}`

type hit struct {
	method, path, rawQuery, auth, body string
}

type fake struct {
	mu     sync.Mutex
	hits   []hit
	status int
	srv    *httptest.Server
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{status: 200}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.hits = append(f.hits, hit{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), string(b)})
		status := f.status
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if status != 200 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
			return
		}
		switch r.URL.Path {
		case "/v1/accounts":
			_, _ = w.Write([]byte(accountsBody))
		case "/api/history":
			_, _ = w.Write([]byte(historyBody))
		case "/api/media/uploadUrl":
			_, _ = w.Write([]byte(`{"uploadUrl": "https://up.example/slot", "accessUrl": "https://cdn.example/v.mp4"}`))
		case "/api/post":
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"ok": true}`))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) setStatus(code int) {
	f.mu.Lock()
	f.status = code
	f.mu.Unlock()
}

func (f *fake) calls() []hit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hit(nil), f.hits...)
}

func newConn(t *testing.T, key, base, accounts string) *Connector {
	t.Helper()
	dir := t.TempDir()
	envLocal := filepath.Join(dir, "env.local")
	if key != "" {
		if err := os.WriteFile(envLocal, []byte("ZERNIO_API_KEY="+key+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := map[string]any{"provider": "late", "baseUrl": base + "/api", "v1Url": base + "/v1"}
	if accounts != "" {
		cfg["accounts"] = json.RawMessage(accounts)
	}
	raw, _ := json.Marshal(cfg)
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZERNIO_API_KEY", "")
	c := New(connectors.Resolver{EnvLocal: envLocal})
	c.ConfigPath = cfgPath
	c.CredFiles = []string{filepath.Join(dir, "absent.env")}
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "zernio" || Meta.Name != "Zernio (Social)" || Meta.Kind != connectors.KindSocial {
		t.Fatalf("meta = %+v", Meta)
	}
}

func followers(a Account) float64 {
	if a.Followers == nil {
		return -1
	}
	return *a.Followers
}

func TestParseLiveAccounts(t *testing.T) {
	got := ParseLiveAccounts([]byte(accountsBody))
	if len(got) != 6 {
		t.Fatalf("accounts without a follower count are dropped, never zero: %+v", got)
	}
	want := map[string]float64{"instagram": 41200, "tiktok": 8300, "youtube": 950, "twitter": 3100, "linkedin": 1500, "facebook": 42}
	for i, a := range got {
		if want[a.Platform] != followers(a) {
			t.Errorf("%d %s = %v", i, a.Platform, followers(a))
		}
	}
	if got[0].Platform != "instagram" || got[0].Handle != "@founderos.ai" {
		t.Errorf("payload order and @handle kept: %+v", got[0])
	}
	if len(ParseLiveAccounts([]byte(`{}`))) != 0 || len(ParseLiveAccounts([]byte(`nope`))) != 0 {
		t.Error("unknown shapes are empty")
	}
}

func TestSummary(t *testing.T) {
	live := ParseLiveAccounts([]byte(accountsBody))
	s := Summarize(live, nil)
	if s.Platforms != 6 || s.Followers != 55092 || s.Handle == nil || *s.Handle != "@founderos.ai" {
		t.Errorf("live summary = %+v", s)
	}
	cfg, err := parseAccountsConfig([]byte(configAccounts))
	if err != nil {
		t.Fatal(err)
	}
	fb := Summarize(nil, cfg)
	if fb.Platforms != 3 || fb.Followers != 59000 || *fb.Handle != "@founderos.ai" {
		t.Errorf("fallback summary = %+v", fb)
	}
	if e := Summarize(nil, nil); e.Platforms != 0 || e.Followers != 0 || e.Handle != nil {
		t.Errorf("empty = %+v", e)
	}
	noIG := Summarize(Accounts{{Platform: "tiktok"}, {Platform: "youtube", Handle: "@yt"}}, nil)
	if noIG.Handle == nil || *noIG.Handle != "@yt" {
		t.Errorf("first account with a handle when no instagram: %+v", noIG)
	}
}

func TestParseHistoryAndPostDays(t *testing.T) {
	posts := ParseHistory([]byte(historyBody), 6)
	if len(posts) != 5 {
		t.Fatalf("posts = %+v", posts)
	}
	p := posts[0]
	if p.Platform != "instagram" || p.URL != "https://www.instagram.com/reel/DZsVsK_DaTn/" || p.PublishedAt == nil ||
		*p.PublishedAt != "2026-06-17T15:54:53.452Z" || p.Status != "success" || !strings.Contains(p.Caption, `Comment "loop"`) {
		t.Errorf("post0 = %+v", p)
	}
	if posts[1].Caption != "second post" || posts[2].Platform != "tiktok" || *posts[2].PublishedAt != "2026-06-10T09:00:00.000Z" || posts[2].Status != "unknown" {
		t.Errorf("fallbacks = %+v %+v", posts[1], posts[2])
	}
	if posts[4].Platform != "unknown" || posts[3].PublishedAt != nil {
		t.Errorf("no platform / no date = %+v %+v", posts[4], posts[3])
	}
	if len(ParseHistory([]byte(historyBody), 1)) != 1 {
		t.Error("limit")
	}
	wrapped := ParseHistory([]byte(`{"posts": `+historyBody+`}`), 6)
	if len(wrapped) != 5 {
		t.Error("the /v1/posts wrapper is read too")
	}
	if len(ParseHistory([]byte(`{}`), 6)) != 0 {
		t.Error("malformed is empty")
	}
	days := ParsePostDays([]byte(historyBody))
	if len(days) != 3 || days[0].Date != "2026-06-17" || strings.Join(days[0].Platforms, ",") != "instagram,tiktok,youtube" || days[2].Date != "2026-06-10" {
		t.Errorf("days = %+v", days)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	f := newFake(t)
	s := newConn(t, "", f.srv.URL, configAccounts).Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.Detail != "ZERNIO_API_KEY not found in env or ~/.founderos/.env." || len(f.calls()) != 0 {
		t.Fatalf("status = %+v calls = %d", s, len(f.calls()))
	}
}

func TestStatusConnectedUsesLiveCounts(t *testing.T) {
	f := newFake(t)
	s := newConn(t, "sk_z", f.srv.URL, configAccounts).Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "6 platforms (@founderos.ai) · 55,092 total followers" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["platforms"] != 6 || s.Meta["followers"] != float64(55092) {
		t.Errorf("meta = %+v", s.Meta)
	}
	if h := f.calls()[0]; h.path != "/v1/accounts" || h.auth != "Bearer sk_z" || h.method != "GET" {
		t.Errorf("call = %+v", h)
	}
}

func TestStatusErrorKeepsConfigPlatformCount(t *testing.T) {
	f := newFake(t)
	f.setStatus(401)
	s := newConn(t, "sk_bad", f.srv.URL, configAccounts).Status(context.Background())
	if s.State != connectors.StateError || s.Detail != "Key found but API check failed: HTTP 401" || s.Meta["platforms"] != 3 {
		t.Fatalf("status = %+v", s)
	}
	down := newConn(t, "sk", "http://127.0.0.1:1", "").Status(context.Background())
	if down.State != connectors.StateError {
		t.Errorf("down = %+v", down)
	}
}

func TestLiveAccountsCachesAndFallsBackToLastGood(t *testing.T) {
	f := newFake(t)
	c := newConn(t, "sk", f.srv.URL, "")
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	first, err := c.LiveAccounts(context.Background())
	if err != nil || len(first) != 6 {
		t.Fatalf("first=%v err=%v", first, err)
	}
	if _, err := c.LiveAccounts(context.Background()); err != nil || len(f.calls()) != 1 {
		t.Errorf("within 60s the cache answers, calls = %d", len(f.calls()))
	}
	now = now.Add(61 * time.Second)
	f.setStatus(503)
	stale, err := c.LiveAccounts(context.Background())
	if err != nil || len(stale) != 6 || len(f.calls()) != 2 {
		t.Errorf("an outage serves the last good answer: %v %v calls=%d", stale, err, len(f.calls()))
	}
	cold := newConn(t, "sk", f.srv.URL, "")
	if got, err := cold.LiveAccounts(context.Background()); err == nil || got != nil {
		t.Errorf("no answer and nothing cached is unknown, not empty: %v %v", got, err)
	}
	if _, err := newConn(t, "", f.srv.URL, "").LiveAccounts(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
}

func TestConfigAccountsFallback(t *testing.T) {
	c := newConn(t, "", "http://127.0.0.1:1", configAccounts)
	accts := c.ConfigAccounts()
	if len(accts) != 3 || accts[0].Platform != "instagram" || accts[2].Platform != "linkedin" || accts[2].Followers != nil {
		t.Errorf("config accounts keep file order: %+v", accts)
	}
}

func TestRecentPostsAndPostDays(t *testing.T) {
	f := newFake(t)
	c := newConn(t, "sk", f.srv.URL, "")
	posts, err := c.RecentPosts(context.Background(), 2)
	if err != nil || len(posts) != 2 {
		t.Fatalf("posts=%v err=%v", posts, err)
	}
	if more, _ := c.RecentPosts(context.Background(), 5); len(more) != 5 || len(f.calls()) != 1 {
		t.Errorf("the cache holds 24 and slices per call: %d calls=%d", len(more), len(f.calls()))
	}
	days, err := c.PostDays(context.Background())
	if err != nil || len(days) != 3 {
		t.Fatalf("days=%v err=%v", days, err)
	}
	calls := f.calls()
	if calls[1].path != "/api/history" || calls[1].rawQuery != "limit=200" {
		t.Errorf("post days read /history?limit=200: %+v", calls[1])
	}
	f.setStatus(500)
	cold := newConn(t, "sk", f.srv.URL, "")
	if d, err := cold.PostDays(context.Background()); err == nil || d != nil {
		t.Errorf("an outage is unknown, never 'nothing posted': %v %v", d, err)
	}
	if p, err := cold.RecentPosts(context.Background(), 3); err == nil || p != nil {
		t.Errorf("recent posts outage = %v %v", p, err)
	}
}

func TestWritesRefusedWhileDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := newFake(t)
	c := newConn(t, "sk", f.srv.URL, "")
	if _, err := c.UploadTarget(context.Background(), "v.mp4", "video/mp4"); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Errorf("upload err = %v", err)
	}
	if err := c.Publish(context.Background(), PublishInput{Caption: "hi", Platforms: []string{"instagram"}}); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Errorf("publish err = %v", err)
	}
	if len(f.calls()) != 0 {
		t.Fatalf("nothing may leave: %+v", f.calls())
	}
	r := guard.Refused()
	if len(r) != 2 || r[0].Action != "zernio.upload_url" || r[1].Action != "zernio.post" {
		t.Errorf("refusals = %+v", r)
	}
}

func TestPublishWhenWritesEnabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := newFake(t)
	c := newConn(t, "sk", f.srv.URL, "")
	target, err := c.UploadTarget(context.Background(), "v.mp4", "video/mp4")
	if err != nil || target.UploadURL != "https://up.example/slot" || target.AccessURL != "https://cdn.example/v.mp4" {
		t.Fatalf("target=%+v err=%v", target, err)
	}
	if q := f.calls()[0].rawQuery; !strings.Contains(q, "fileName=v.mp4") || !strings.Contains(q, "contentType=video%2Fmp4") {
		t.Errorf("upload query = %q", q)
	}
	err = c.Publish(context.Background(), PublishInput{Caption: "Watch this", MediaURLs: []string{target.AccessURL}, Platforms: []string{"instagram", "youtube"}})
	if err != nil {
		t.Fatal(err)
	}
	post := f.calls()[1]
	if post.method != "POST" || post.path != "/api/post" || post.auth != "Bearer sk" {
		t.Fatalf("post = %+v", post)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(post.body), &sent)
	yt, _ := sent["youTubeOptions"].(map[string]any)
	if sent["post"] != "Watch this" || sent["isVideo"] != true || sent["publishNow"] != true || sent["scheduledFor"] != nil ||
		yt["title"] != "Watch this" || yt["visibility"] != "public" || yt["shorts"] != true {
		t.Errorf("payload = %v", sent)
	}
	when := "2026-10-01T15:00:00Z"
	if err := c.Publish(context.Background(), PublishInput{Caption: "later", MediaURLs: []string{"https://cdn.example/a.png"}, Platforms: []string{"tiktok"}, ScheduledFor: &when}); err != nil {
		t.Fatal(err)
	}
	sent = nil // Unmarshal into a live map merges keys
	_ = json.Unmarshal([]byte(f.calls()[2].body), &sent)
	if sent["scheduledFor"] != when || sent["publishNow"] != nil || sent["isVideo"] != false || sent["youTubeOptions"] != nil {
		t.Errorf("scheduled payload = %v", sent)
	}
	f.setStatus(400)
	if err := c.Publish(context.Background(), PublishInput{Caption: "x", Platforms: []string{"tiktok"}}); err == nil || !strings.HasPrefix(err.Error(), "HTTP 400: ") {
		t.Errorf("err = %v", err)
	}
}
