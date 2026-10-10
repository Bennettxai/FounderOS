package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/manychat"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// ---- fakes --------------------------------------------------------------------

type socialFakeZernio struct {
	live      zernio.Accounts
	liveErr   error
	config    zernio.Accounts
	days      []zernio.PostDay
	daysErr   error
	posts     []zernio.Post
	postsErr  error
	published []zernio.PublishInput
}

func (f *socialFakeZernio) LiveAccounts(context.Context) (zernio.Accounts, error) {
	return f.live, f.liveErr
}
func (f *socialFakeZernio) ConfigAccounts() zernio.Accounts { return f.config }
func (f *socialFakeZernio) PostDays(context.Context) ([]zernio.PostDay, error) {
	return f.days, f.daysErr
}
func (f *socialFakeZernio) RecentPosts(_ context.Context, n int) ([]zernio.Post, error) {
	if f.postsErr != nil {
		return nil, f.postsErr
	}
	if len(f.posts) > n {
		return f.posts[:n], nil
	}
	return f.posts, nil
}
func (f *socialFakeZernio) UploadTarget(context.Context, string, string) (zernio.UploadSlot, error) {
	return zernio.UploadSlot{}, guard.Outbound("zernio.upload_url", func() error { return nil })
}
func (f *socialFakeZernio) Publish(_ context.Context, in zernio.PublishInput) error {
	return guard.Outbound("zernio.post", func() error { f.published = append(f.published, in); return nil })
}

type socialFakeBeehiiv struct {
	reading *beehiiv.Reading
	posts   []beehiiv.Newsletter
}

func (f *socialFakeBeehiiv) Reading(context.Context) *beehiiv.Reading   { return f.reading }
func (f *socialFakeBeehiiv) Posts(context.Context) []beehiiv.Newsletter { return f.posts }

type socialFakeManyChat struct{ sent int }

func (f *socialFakeManyChat) SendText(context.Context, string, string) (manychat.SendResult, error) {
	err := guard.Outbound("manychat.send_text", func() error { f.sent++; return nil })
	if err != nil {
		return manychat.SendResult{Detail: "ManyChat send failed: " + err.Error()}, err
	}
	return manychat.SendResult{OK: true, Detail: "sent"}, nil
}

// ---- harness --------------------------------------------------------------------

func socialThrowawayDB(t *testing.T) *pgxpool.Pool {
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
	name := fmt.Sprintf("founderos_social_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
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

type socialHarness struct {
	t    *testing.T
	pool *pgxpool.Pool
	ws   string
	z    *socialFakeZernio
	b    *socialFakeBeehiiv
	m    *socialFakeManyChat
	r    *gin.Engine
}

func newSocialHarness(t *testing.T) *socialHarness {
	pool := socialThrowawayDB(t)
	ctx := context.Background()
	var ws string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Personal', 'personal', 'u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	f := func(n float64) *float64 { return &n }
	h := &socialHarness{t: t, pool: pool, ws: ws,
		z: &socialFakeZernio{
			live:  zernio.Accounts{{Platform: "instagram", Handle: "@b", Followers: f(1100)}, {Platform: "facebook", Followers: f(5)}},
			days:  []zernio.PostDay{{Date: time.Now().UTC().Format("2006-01-02"), Platforms: []string{"instagram", "tiktok"}}},
			posts: []zernio.Post{{Platform: "instagram", Caption: "3 agents", URL: "https://ig/p/1", Status: "success"}},
		},
		b: &socialFakeBeehiiv{},
		m: &socialFakeManyChat{},
	}
	prev := socialNewSources
	socialNewSources = func(*Deps) socialSources {
		return socialSources{zernio: h.z, beehiiv: h.b, manychat: h.m, http: http.DefaultClient}
	}
	t.Cleanup(func() { socialNewSources = prev })
	h.exec(`INSERT INTO founderos_social_accounts (platform, workspace_id, handle, url, ord) VALUES
		('instagram', $1, '@founderos.ai', 'https://instagram.com/founderos.ai', 1), ('tiktok', $1, '@founderos.ai', NULL, 2)`, ws)
	h.exec(`INSERT INTO founderos_social_snapshots (platform, captured_on, workspace_id, followers, source) VALUES
		('instagram', current_date - 40, $1, 900, 'seed'), ('instagram', current_date - 7, $1, 1000, 'seed'), ('tiktok', current_date - 1, $1, 500, 'seed')`, ws)
	h.exec(`INSERT INTO founderos_social_dm_messages (id, workspace_id, platform, subscriber_id, name, handle, text, direction, ts, source) VALUES
		('m1', $1, 'instagram', 's1', 'Ava', 'ava', 'hey', 'out', now() - interval '2 hours', 'seed'),
		('m2', $1, 'instagram', 's1', 'Ava', 'ava', 'price?', 'in', now() - interval '1 hour', 'seed')`, ws)
	h.exec(`INSERT INTO founderos_email_list_snapshots (captured_on, workspace_id, subscribers, source) VALUES
		(current_date - 60, $1, 4812, 'seed-beehiiv'), (current_date - 1, $1, 24814, 'beehiiv')`, ws)
	h.r = router(t, &Deps{Pool: pool})
	return h
}

func (h *socialHarness) exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), sql, args...); err != nil {
		h.t.Fatal(err)
	}
}

func (h *socialHarness) do(method, path string, body []byte, ctype string) (int, map[string]any) {
	h.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Cookie", "session=ok")
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func (h *socialHarness) get(path string) (int, map[string]any) {
	return h.do(http.MethodGet, path, nil, "")
}

func (h *socialHarness) post(path string, v any) (int, map[string]any) {
	raw, _ := json.Marshal(v)
	return h.do(http.MethodPost, path, raw, "application/json")
}

// ---- tests ------------------------------------------------------------------------

func TestSocialPageNeedsASession(t *testing.T) {
	h := newSocialHarness(t)
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/social", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
}

func TestSocialPageSyncsLiveAndBuildsTheModel(t *testing.T) {
	h := newSocialHarness(t)
	code, body := h.get("/api/founderos/pages/social")
	if code != 200 {
		t.Fatalf("%d %v", code, body)
	}
	if body["sync"].(map[string]any)["source"] != "zernio-live" || body["sync"].(map[string]any)["recorded"].(float64) != 1 {
		t.Fatalf("sync %v", body["sync"])
	}
	platforms := body["platforms"].([]any)
	ig := platforms[0].(map[string]any)
	if ig["platform"] != "instagram" || ig["followers"].(float64) != 1100 {
		t.Fatalf("instagram %v", ig)
	}
	// 4812 is seed: the email list reads the real 24,814 only.
	if body["emailList"].(map[string]any)["subscribers"].(float64) != 24814 || body["audienceTotal"].(float64) != 1100+500+24814 {
		t.Fatalf("audience %v %v", body["emailList"], body["audienceTotal"])
	}
	v := body["volume"].(map[string]any)
	if v["postsInWindow"].(float64) != 1 || v["insight"].(map[string]any)["headline"] != "1 of 1 Instagram threads need a reply." {
		t.Fatalf("volume %v", v)
	}
	if body["postsKnown"] != true || len(body["recentPosts"].([]any)) != 1 || len(body["dmThreads"].([]any)) != 1 {
		t.Fatalf("posts %v %v", body["postsKnown"], body["recentPosts"])
	}
}

func TestSocialPageUnknownPostingHistoryIsNullNotEmpty(t *testing.T) {
	h := newSocialHarness(t)
	h.z.daysErr = errors.New("zernio down")
	h.z.postsErr = errors.New("zernio down")
	h.z.liveErr = errors.New("zernio down")
	_, body := h.get("/api/founderos/pages/social")
	if body["postsKnown"] != false || body["postDays"] != nil || body["recentPosts"] != nil || body["recentError"] == nil {
		t.Fatalf("%v %v %v", body["postsKnown"], body["postDays"], body["recentPosts"])
	}
	if body["sync"].(map[string]any)["source"] != "none" {
		t.Fatalf("no live and no config: %v", body["sync"])
	}
}

func TestSocialStaticRoutesBeatThePlatformParam(t *testing.T) {
	h := newSocialHarness(t)
	if code, body := h.get("/api/founderos/pages/social/history?limit=3"); code != 200 || len(body["posts"].([]any)) != 1 {
		t.Fatalf("history %d %v", code, body)
	}
	if code, body := h.get("/api/founderos/pages/social/series?metric=audience"); code != 200 || body["series"].([]any)[0].(map[string]any)["key"] != "all" {
		t.Fatalf("series %d %v", code, body)
	}
	if code, body := h.get("/api/founderos/pages/social/series?metric=dms"); code != 200 || body["series"].([]any)[0].(map[string]any)["key"] != "total" {
		t.Fatalf("dms series %d %v", code, body)
	}
	if code, _ := h.get("/api/founderos/pages/social/series?metric=nope"); code != 400 {
		t.Fatalf("bad metric %d", code)
	}
	if code, body := h.get("/api/founderos/pages/social/posts"); code != 200 || body["posts"] == nil {
		t.Fatalf("posts %d %v", code, body)
	}
	if code, body := h.get("/api/founderos/pages/social/beehiiv"); code != 200 || body["seeded"] != true {
		t.Fatalf("beehiiv %d %v", code, body)
	}
	if code, body := h.get("/api/founderos/pages/social/sync"); code != 200 || body["source"] != "zernio-live" {
		t.Fatalf("sync %d %v", code, body)
	}
	if code, body := h.get("/api/founderos/pages/social/instagram"); code != 200 || body["label"] != "Instagram" || body["volume"] == nil {
		t.Fatalf("platform %d %v", code, body)
	}
	if code, _ := h.get("/api/founderos/pages/social/myspace"); code != 404 {
		t.Fatalf("unknown platform %d", code)
	}
}

func TestSocialHistoryUnreachableIs502(t *testing.T) {
	h := newSocialHarness(t)
	h.z.postsErr = errors.New("zernio down")
	if code, body := h.get("/api/founderos/pages/social/history"); code != 502 || body["posts"] != nil {
		t.Fatalf("%d %v", code, body)
	}
}

func TestSocialPostIsRefusedWhileWritesAreOffAndRecordedFailed(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	h := newSocialHarness(t)
	code, body := h.post("/api/founderos/pages/social/posts", map[string]any{"caption": "hello", "platforms": []string{"instagram"}})
	if code != 502 || body["post"].(map[string]any)["status"] != "failed" || body["error"] == nil {
		t.Fatalf("%d %v", code, body)
	}
	if len(h.z.published) != 0 {
		t.Fatal("published while writes were off")
	}
	_, list := h.get("/api/founderos/pages/social/posts")
	if len(list["posts"].([]any)) != 1 {
		t.Fatalf("queue %v", list)
	}
	if code, _ := h.post("/api/founderos/pages/social/posts", map[string]any{"caption": "", "platforms": []string{}}); code != 400 {
		t.Fatalf("invalid body %d", code)
	}
	if code, _ := h.post("/api/founderos/pages/social/posts", map[string]any{"caption": "x", "platforms": []string{"myspace"}}); code != 400 {
		t.Fatalf("unknown platform %d", code)
	}
}

func TestSocialPostPublishesWhenWritesAreOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	h := newSocialHarness(t)
	code, body := h.post("/api/founderos/pages/social/posts", map[string]any{"caption": "hello", "platforms": []string{"instagram"}, "scheduledFor": "2026-10-01T15:00:00.000Z"})
	if code != 201 || body["post"].(map[string]any)["status"] != "queued" || len(h.z.published) != 1 {
		t.Fatalf("%d %v", code, body)
	}
}

func TestSocialDMReplyRefusedStoresNothing(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	h := newSocialHarness(t)
	code, body := h.post("/api/founderos/pages/social/dm/reply", map[string]any{"subscriberId": "s1", "text": "hi"})
	if code != 502 || body["ok"] != false {
		t.Fatalf("%d %v", code, body)
	}
	var n int
	_ = h.pool.QueryRow(context.Background(), `SELECT count(*) FROM founderos_social_dm_messages`).Scan(&n)
	if n != 2 || h.m.sent != 0 {
		t.Fatalf("stored %d sent %d", n, h.m.sent)
	}
	if code, _ := h.post("/api/founderos/pages/social/dm/reply", map[string]any{"subscriberId": "", "text": "hi"}); code != 400 {
		t.Fatalf("invalid %d", code)
	}
}

func TestSocialDMReplySentIsStoredWithTheThreadName(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	h := newSocialHarness(t)
	code, body := h.post("/api/founderos/pages/social/dm/reply", map[string]any{"subscriberId": "s1", "text": "on it"})
	msg, _ := body["message"].(map[string]any)
	if code != 200 || msg["name"] != "Ava" || msg["direction"] != "out" {
		t.Fatalf("%d %v", code, body)
	}
}

func TestSocialUploadRefusedWhileWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	h := newSocialHarness(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "clip.mp4")
	_, _ = fw.Write([]byte("bytes"))
	_ = mw.Close()
	if code, body := h.do(http.MethodPost, "/api/founderos/pages/social/upload", buf.Bytes(), mw.FormDataContentType()); code != 502 || body["error"] == nil {
		t.Fatalf("%d %v", code, body)
	}
	if code, _ := h.do(http.MethodPost, "/api/founderos/pages/social/upload", nil, "multipart/form-data; boundary=x"); code != 400 {
		t.Fatalf("no file %d", code)
	}
}

func TestSocialBeehiivLiveWhenPostsAndReadingExist(t *testing.T) {
	h := newSocialHarness(t)
	h.b.reading = &beehiiv.Reading{Subscribers: 24900, Fresh: true}
	h.b.posts = []beehiiv.Newsletter{{ID: "p1", Title: "One", PublishedAt: "2026-09-01T15:00:00.000Z", Recipients: 100, Delivered: 99, Opens: 30, OpenRate: 30}}
	_, body := h.get("/api/founderos/pages/social/beehiiv")
	if body["seeded"] != false || body["subscribers"].(float64) != 24900 || len(body["newsletters"].([]any)) != 1 {
		t.Fatalf("%v", body)
	}
}

func TestSocialWithoutWorkspaceIs503(t *testing.T) {
	pool := socialThrowawayDB(t)
	prev := socialNewSources
	socialNewSources = func(*Deps) socialSources {
		return socialSources{zernio: &socialFakeZernio{}, beehiiv: &socialFakeBeehiiv{}, manychat: &socialFakeManyChat{}, http: http.DefaultClient}
	}
	t.Cleanup(func() { socialNewSources = prev })
	r := router(t, &Deps{Pool: pool})
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/social", nil)
	req.Header.Set("Cookie", "session=ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}
