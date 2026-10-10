package vidalytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Fixture shaped like the real GET /public/v1/video payload (Postman
// collection behind api-docs.vidalytics.com, verified 2026-09-23).
const videosBody = `{
  "status": true,
  "content": {
    "data": [
      {"id": "v1", "title": "LC VSL v3", "date_created": "2026-08-01T00:00:00Z", "last_published": "2026-09-01T00:00:00Z", "status": "ready", "folder_id": "f1", "embedGuid": "g1"},
      {"id": "v2", "title": "Vantage explainer", "date_created": "2026-07-01T00:00:00Z", "last_published": null, "status": "processing", "folder_id": null, "embedGuid": "g2"},
      {"id": "", "title": "no id, dropped"},
      {"id": "v3", "title": "   "}
    ]
  }
}`

type hit struct {
	method, path, key, auth string
}

type fake struct {
	mu   sync.Mutex
	hits []hit
	srv  *httptest.Server
}

func newFake(t *testing.T, status int, body string) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits = append(f.hits, hit{r.Method, r.URL.Path, r.Header.Get("X-API-Key"), r.Header.Get("Authorization")})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func keyed(t *testing.T, key, base string) *Connector {
	t.Helper()
	dir := t.TempDir()
	envLocal := filepath.Join(dir, "env.local")
	if key != "" {
		if err := os.WriteFile(envLocal, []byte("VIDALYTICS_API_KEY="+key+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("VIDALYTICS_API_KEY", "")
	c := New(connectors.Resolver{EnvLocal: envLocal})
	c.BaseURL = base
	c.CredFiles = []string{filepath.Join(dir, "absent.env")}
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "vidalytics" || Meta.Name != "Vidalytics" || Meta.Kind != connectors.KindCreative {
		t.Fatalf("meta = %+v", Meta)
	}
}

func TestParseVideosMapsContentData(t *testing.T) {
	got, ok := ParseVideos([]byte(videosBody))
	if !ok {
		t.Fatal("payload should parse")
	}
	if len(got) != 3 {
		t.Fatalf("want 3 rows (id-less row dropped), got %d: %+v", len(got), got)
	}
	v1 := got[0]
	if v1.ID != "v1" || v1.Title != "LC VSL v3" || v1.Status == nil || *v1.Status != "ready" ||
		v1.CreatedAt == nil || *v1.CreatedAt != "2026-08-01T00:00:00Z" || v1.LastPublished == nil || v1.FolderID == nil || *v1.FolderID != "f1" {
		t.Errorf("v1 = %+v", v1)
	}
	if got[1].LastPublished != nil || got[1].FolderID != nil {
		t.Errorf("nulls must stay nil: %+v", got[1])
	}
	if got[2].Title != "Untitled video" {
		t.Errorf("blank title falls back, got %q", got[2].Title)
	}
}

func TestParseVideosUnknownShapeIsNotEmpty(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"content":{"data":"x"}}`, `not json`} {
		if _, ok := ParseVideos([]byte(body)); ok {
			t.Errorf("%q must be unreadable, not an empty library", body)
		}
	}
}

func TestStatusNotConfiguredWithoutKey(t *testing.T) {
	f := newFake(t, 200, videosBody)
	s := keyed(t, "", f.srv.URL).Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.ID != "vidalytics" || !strings.Contains(s.Detail, "VIDALYTICS_API_KEY") {
		t.Fatalf("status = %+v", s)
	}
	if len(f.hits) != 0 {
		t.Fatalf("no call without a key, got %+v", f.hits)
	}
}

func TestStatusConnectedListsLibraryOnly(t *testing.T) {
	f := newFake(t, 200, videosBody)
	s := keyed(t, "vid_test", f.srv.URL).Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("status = %+v", s)
	}
	if s.Detail != "Vidalytics reachable · 3 videos in the library (stats are quota-metered and not polled)" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["videos"] != 3 {
		t.Errorf("meta = %+v", s.Meta)
	}
	if len(f.hits) != 1 || f.hits[0].path != "/video" || f.hits[0].method != "GET" {
		t.Fatalf("exactly one GET /video, got %+v", f.hits)
	}
	if f.hits[0].key != "vid_test" || f.hits[0].auth != "" {
		t.Errorf("key goes verbatim in X-API-Key, no Bearer: %+v", f.hits[0])
	}
}

func TestStatusErrorOnRejectedKeyOrOddPayload(t *testing.T) {
	rejected := keyed(t, "vid_bad", newFake(t, 401, `{"message":"Unauthorized"}`).srv.URL).Status(context.Background())
	if rejected.State != connectors.StateError || !strings.Contains(rejected.Detail, "401") ||
		!strings.HasPrefix(rejected.Detail, "VIDALYTICS_API_KEY is set but the call failed: ") {
		t.Errorf("rejected = %+v", rejected)
	}
	odd := keyed(t, "vid", newFake(t, 200, `{"nope":true}`).srv.URL).Status(context.Background())
	if odd.State != connectors.StateError || !strings.Contains(odd.Detail, "unrecognised /video payload") {
		t.Errorf("odd = %+v", odd)
	}
	down := keyed(t, "vid", "http://127.0.0.1:1").Status(context.Background())
	if down.State != connectors.StateError {
		t.Errorf("unreachable = %+v", down)
	}
}

func TestListVideos(t *testing.T) {
	videos, err := keyed(t, "vid", newFake(t, 200, videosBody).srv.URL).ListVideos(context.Background())
	if err != nil || len(videos) != 3 {
		t.Fatalf("videos=%v err=%v", videos, err)
	}
	if _, err := keyed(t, "", "http://127.0.0.1:1").ListVideos(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("no key must be ErrNotConfigured, got %v", err)
	}
	if v, err := keyed(t, "vid", newFake(t, 500, `{}`).srv.URL).ListVideos(context.Background()); err == nil || v != nil {
		t.Errorf("a failure is an error, never an empty library: %v %v", v, err)
	}
}

func TestNeverTouchesStatsEndpoints(t *testing.T) {
	f := newFake(t, 200, videosBody)
	c := keyed(t, "vid", f.srv.URL)
	c.Status(context.Background())
	_, _ = c.ListVideos(context.Background())
	for _, h := range f.hits {
		if strings.Contains(h.path, "stats") {
			t.Fatalf("stats endpoint polled: %+v", h)
		}
	}
}
