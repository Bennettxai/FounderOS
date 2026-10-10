package beehiiv

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

const pubBody = `{"data":{"id":"pub_1","name":"the operator Weekly","stats":{"active_subscriptions":18266,"total_subscriptions":20001,"average_open_rate":48.2}}}`

const postsBody = `{"data":[
 {"id":"post_2","title":"Second","status":"confirmed","publish_date":1727000000,"web_url":"https://b.beehiiv.com/p/second",
  "stats":{"email":{"recipients":1000,"delivered":990,"unique_opens":500,"open_rate":50.506,"unique_clicks":40,"click_rate":4.044,"unsubscribes":3,"spam_reports":1},"web":{"views":20}}},
 {"id":"post_draft","title":"Draft","status":"draft","publish_date":null},
 {"id":"post_1","status":"published","displayed_date":"2026-09-01T12:00:00Z",
  "stats":{"email":{"total_sent":10,"total_delivered":0,"total_unique_opened":0}}}
],"limit":50,"page":1,"total_results":3,"total_pages":1}`

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv("BEEHIIV_API_KEY", "")
	t.Setenv("BEEHIIV_PUBLICATION_ID", "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

const keyed = "BEEHIIV_API_KEY=bh-test\nBEEHIIV_PUBLICATION_ID=pub_1\n"

type fake struct {
	hits   atomic.Int32
	status atomic.Int32 // HTTP status to answer with; 0 = 200
	pub    atomic.Value
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	f.pub.Store(pubBody)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		if r.Header.Get("Authorization") != "Bearer bh-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if s := f.status.Load(); s != 0 {
			w.WriteHeader(int(s))
			return
		}
		if r.URL.Query().Get("expand[]") != "stats" {
			t.Errorf("missing expand[]=stats on %s", r.URL)
		}
		switch r.URL.Path {
		case "/v2/publications/pub_1":
			fmt.Fprint(w, f.pub.Load().(string))
		case "/v2/publications/pub_1/posts":
			q := r.URL.Query()
			if q.Get("limit") != "50" || q.Get("order_by") != "publish_date" || q.Get("direction") != "desc" {
				t.Errorf("posts query = %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, postsBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTest(t *testing.T, f *fake, env string) (*Connector, *time.Time) {
	c := New(resolver(t, env))
	c.baseURL = f.serve(t).URL + "/v2"
	now := time.Unix(1_800_000_000, 0)
	c.now = func() time.Time { return now }
	return c, &now
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta != (connectors.Meta{ID: "beehiiv", Name: "Beehiiv (Email List)", Kind: connectors.KindSocial}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfiguredNeedsBothKeyAndPublication(t *testing.T) {
	for _, env := range []string{"", "BEEHIIV_API_KEY=bh-test\n", "BEEHIIV_PUBLICATION_ID=pub_1\n"} {
		f := &fake{}
		c, _ := newTest(t, f, env)
		s := c.Status(context.Background())
		if s.State != connectors.StateNotConfigured || s.Detail != "Set BEEHIIV_API_KEY and BEEHIIV_PUBLICATION_ID in ~/.founderos/.env or under API keys." {
			t.Fatalf("env %q: %+v", env, s)
		}
		if f.hits.Load() != 0 {
			t.Fatal("unconfigured must not call out")
		}
	}
}

func TestStatusConnected(t *testing.T) {
	c, _ := newTest(t, &fake{}, keyed)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "18,266 subscribers" || s.Meta["subscribers"] != 18266 {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusErrors(t *testing.T) {
	f := &fake{}
	c, _ := newTest(t, f, keyed)
	f.status.Store(401)
	if s := c.Status(context.Background()); s.State != connectors.StateError || s.Detail != "Key set but API check failed: HTTP 401" {
		t.Fatalf("auth failure: %+v", s)
	}
	f.status.Store(0)
	f.pub.Store(`{"data":{"id":"pub_1","stats":{}}}`)
	if s := c.Status(context.Background()); s.State != connectors.StateError || s.Detail != "Key set but API check failed: no subscriber stats in response" {
		t.Fatalf("no stats must be an error, never 0: %+v", s)
	}
	c.baseURL = "http://127.0.0.1:1/v2"
	if s := c.Status(context.Background()); s.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", s)
	}
}

func TestParseStatsPrefersActiveAndNamesTheMetric(t *testing.T) {
	cases := []struct {
		body   string
		ok     bool
		n      int
		metric Metric
	}{
		{`{"data":{"stats":{"active_subscriptions":5,"total_subscriptions":9}}}`, true, 5, MetricActive},
		{`{"stats":{"total_subscriptions":9}}`, true, 9, MetricTotal},
		{`{"data":{"stats":{"active_subscriptions":0,"total_subscriptions":9}}}`, true, 0, MetricActive}, // 0 is a real count
		{`{"data":{"stats":{"active_subscriptions":-1,"total_subscriptions":"9"}}}`, false, 0, ""},
		{`{"data":{}}`, false, 0, ""},
		{`not json`, false, 0, ""},
	}
	for _, tc := range cases {
		n, m, ok := ParseStats([]byte(tc.body))
		if ok != tc.ok || n != tc.n || m != tc.metric {
			t.Errorf("%s: got %d %q %v", tc.body, n, m, ok)
		}
	}
}

func TestReadingCachesAndMarksStaleOnFailure(t *testing.T) {
	f := &fake{}
	c, now := newTest(t, f, keyed)
	r := c.Reading(context.Background())
	if r == nil || r.Subscribers != 18266 || r.Metric != MetricActive || !r.Fresh || r.PublicationID != "pub_1" {
		t.Fatalf("reading = %+v", r)
	}
	c.Reading(context.Background())
	if f.hits.Load() != 1 {
		t.Fatalf("60s cache missed: hits=%d", f.hits.Load())
	}
	*now = now.Add(61 * time.Second)
	f.status.Store(500)
	r = c.Reading(context.Background())
	if r == nil || r.Subscribers != 18266 || r.Fresh {
		t.Fatalf("a failed refresh must serve the last good count marked stale: %+v", r)
	}
	if n, ok := c.Subscribers(context.Background()); !ok || n != 18266 {
		t.Fatalf("subscribers = %d %v", n, ok)
	}
}

func TestReadingUnknownWhenNeverFetched(t *testing.T) {
	f := &fake{}
	c, _ := newTest(t, f, keyed)
	f.status.Store(500)
	if r := c.Reading(context.Background()); r != nil {
		t.Fatalf("never fetched must read unknown (nil), got %+v", r)
	}
	if _, ok := c.Subscribers(context.Background()); ok {
		t.Fatal("unknown must not read as a number")
	}
	c2, _ := newTest(t, &fake{}, "")
	if r := c2.Reading(context.Background()); r != nil {
		t.Fatal("unconfigured must read unknown")
	}
}

func TestPostsParsesSentPostsOnly(t *testing.T) {
	f := &fake{}
	c, _ := newTest(t, f, keyed)
	got := c.Posts(context.Background())
	if len(got) != 2 {
		t.Fatalf("got %d posts, want 2 (draft dropped)", len(got))
	}
	p := got[0]
	want := Newsletter{
		ID: "post_2", Title: "Second", PublishedAt: "2024-09-22T10:13:20.000Z", WebURL: "https://b.beehiiv.com/p/second",
		Recipients: 1000, Delivered: 990, DeliveryRate: 99, Opens: 500, OpenRate: 50.51, Clicks: 40, ClickRate: 4.04,
		Unsubscribes: 3, UnsubscribeRate: 0.3, SpamReports: 1, WebViews: 20,
	}
	if p != want {
		t.Fatalf("post = %+v\nwant   %+v", p, want)
	}
	q := got[1]
	if q.Title != "Untitled" || q.Recipients != 10 || q.DeliveryRate != 0 || q.UnsubscribeRate != 0 || q.WebURL != "" || q.PublishedAt != "2026-09-01T12:00:00.000Z" {
		t.Fatalf("aliases/defaults = %+v", q)
	}
	c.Posts(context.Background())
	if f.hits.Load() != 1 {
		t.Fatal("posts must be cached for 60s")
	}
}

func TestPostsUnknownWhenUnconfiguredOrFailing(t *testing.T) {
	c, _ := newTest(t, &fake{}, "")
	if got := c.Posts(context.Background()); got != nil {
		t.Fatalf("unconfigured = %+v", got)
	}
	f := &fake{}
	c, _ = newTest(t, f, keyed)
	f.status.Store(503)
	if got := c.Posts(context.Background()); got != nil {
		t.Fatalf("failing = %+v", got)
	}
}

func TestParsePostsNeverPanicsOnJunk(t *testing.T) {
	for _, body := range []string{``, `null`, `{"data":"x"}`, `{"data":[null,1,"x",{"id":"a","status":"confirmed","stats":"x"}]}`} {
		_ = ParsePosts([]byte(body))
	}
	if got := ParsePosts([]byte(`{"data":[{"id":"a","status":"confirmed","publish_date":1}]}`)); len(got) != 1 || !strings.HasPrefix(got[0].PublishedAt, "1970") {
		t.Fatalf("got %+v", got)
	}
}
