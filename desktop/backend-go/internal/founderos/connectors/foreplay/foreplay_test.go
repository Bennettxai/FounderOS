package foreplay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Fixtures shaped like public.api.foreplay.co answers: a metadata envelope
// around data.
func envelope(data string) string {
	return `{"metadata":{"success":true,"status_code":200,"message":"ok","cursor":null,"count":1},"data":` + data + `}`
}

const usageData = `{"start_date":"2026-09-01","end_date":"2026-09-30","total_credits":10000,"remaining_credits":9640,"user":{"id":"u_1","email":"ops@example.com"}}`

const adData = `{"id":"ad_1","ad_id":"1234567890","name":"Vantage","brand_id":"br_1","description":"Stop losing leads after hours","headline":"AI intake that books calls","full_transcription":"If your front desk misses calls...","timestamped_transcription":[{"startTime":0,"endTime":2.5,"sentence":"If your front desk misses calls"}],"emotional_drivers":{"urgency":0.8,"trust":0.4},"display_format":"video","publisher_platform":["facebook","instagram"],"live":true,"started_running":1756684800000,"running_duration":{"seconds":2592000},"thumbnail":"https://cdn.example/t.jpg","video":"https://cdn.example/v.mp4","image":null,"cta_type":"LEARN_MORE","cta_title":"Learn more","link_url":"https://vantage.example","product_category":"software","market_target":"b2b","extra_field":"kept"}`

func brand(i int) string {
	return fmt.Sprintf(`{"id":"br_%d","name":"Brand %d","ad_library_id":"lib_%d","avatar":null,"category":"software","websites":["https://b%d.example"],"ads_count":%d,"active_ads_count":%d}`, i, i, i, i, 10+i, i)
}

func brands(from, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = brand(from + i)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

type harness struct {
	dir      string
	envLocal string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	t.Setenv("FOREPLAY_API_KEY", "")
	dir := t.TempDir()
	return &harness{dir: dir, envLocal: filepath.Join(dir, "env.local")}
}

func (h *harness) connector(t *testing.T, envLocal string) *Connector {
	t.Helper()
	os.WriteFile(h.envLocal, []byte(envLocal), 0o600)
	c := New(connectors.Resolver{EnvLocal: h.envLocal})
	c.Files = []string{filepath.Join(h.dir, "absent.env")}
	c.Store = Store{Dir: filepath.Join(h.dir, "ad-intel")}
	return c
}

func (h *harness) writeStore(t *testing.T, name, body string) {
	t.Helper()
	d := filepath.Join(h.dir, "ad-intel")
	os.MkdirAll(d, 0o755)
	if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

type failRT struct{ t *testing.T }

func (f failRT) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("status must not spend Foreplay credits, got %s", r.URL)
	return nil, errors.New("no network")
}

func noNetwork(t *testing.T) {
	orig := http.DefaultTransport
	http.DefaultTransport = failRT{t}
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "foreplay" || Meta.Name != "Foreplay" || Meta.Kind != connectors.KindCreative {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	noNetwork(t)
	h := newHarness(t)
	s := h.connector(t, "").Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "FOREPLAY_API_KEY") {
		t.Fatalf("status = %+v", s)
	}
}

func TestStatusKeyedReadsTheSyncedStoreWithoutSpending(t *testing.T) {
	noNetwork(t)
	h := newHarness(t)
	c := h.connector(t, "FOREPLAY_API_KEY=fp-key\n")
	h.writeStore(t, "usage.json", usageData)
	h.writeStore(t, "meta.json", `{"lastSyncAt":"2026-09-28T14:05:33.120Z","lastSyncCalls":42}`)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "9,640/10,000 credits · last sync 2026-09-28 14:05" {
		t.Fatalf("status = %+v", s)
	}
	if s.Meta["remaining"] != float64(9640) || s.Meta["total"] != float64(10000) {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusKeyedNeverSynced(t *testing.T) {
	noNetwork(t)
	h := newHarness(t)
	s := h.connector(t, "FOREPLAY_API_KEY=fp-key\n").Status(context.Background())
	if s.State != connectors.StateConnected || s.Detail != "credits unknown until first sync · never synced" || s.Meta != nil {
		t.Fatalf("status = %+v", s)
	}
	// A corrupt snapshot reads unknown too, never as zero credits.
	h.writeStore(t, "usage.json", `{"total_credits":`)
	if s := h.connector(t, "FOREPLAY_API_KEY=fp-key\n").Status(context.Background()); !strings.HasPrefix(s.Detail, "credits unknown") {
		t.Fatalf("status = %+v", s)
	}
}

func TestClientNotConfigured(t *testing.T) {
	h := newHarness(t)
	if _, err := h.connector(t, "").Client(); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}

type fakeAPI struct {
	t        *testing.T
	requests []*http.Request
	handle   func(w http.ResponseWriter, r *http.Request)
}

func (h *harness) client(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) (*Client, *fakeAPI) {
	f := &fakeAPI{t: t, handle: handle}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Foreplay is read-only, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "fp-key" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		f.requests = append(f.requests, r)
		f.handle(w, r)
	}))
	t.Cleanup(srv.Close)
	c := h.connector(t, "FOREPLAY_API_KEY=fp-key\n")
	c.BaseURL = srv.URL
	cl, err := c.Client()
	if err != nil {
		t.Fatal(err)
	}
	return cl, f
}

func TestUsage(t *testing.T) {
	h := newHarness(t)
	cl, f := h.client(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, envelope(usageData)) })
	u, err := cl.Usage(context.Background())
	if err != nil || u.TotalCredits != 10000 || u.RemainingCredits != 9640 || u.User == nil || u.User.Email != "ops@example.com" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
	if f.requests[0].URL.Path != "/api/usage" || cl.CallCount() != 1 {
		t.Errorf("path=%s calls=%d", f.requests[0].URL.Path, cl.CallCount())
	}
}

func TestSpyderBrandsPaginatesByOffset(t *testing.T) {
	h := newHarness(t)
	cl, f := h.client(t, func(w http.ResponseWriter, r *http.Request) {
		off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		n := 10
		if off == 20 {
			n = 3
		}
		io.WriteString(w, envelope(brands(off, n)))
	})
	got, err := cl.SpyderBrands(context.Background())
	if err != nil || len(got) != 23 {
		t.Fatalf("len=%d err=%v", len(got), err)
	}
	if got[22].ID != "br_22" || got[0].AdsCount == nil || *got[0].AdsCount != 10 {
		t.Errorf("brands = %+v", got[22])
	}
	for i, r := range f.requests {
		if r.URL.Path != "/api/spyder/brands" || r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("offset") != strconv.Itoa(i*10) {
			t.Errorf("request %d = %s", i, r.URL)
		}
	}
	if cl.CallCount() != 3 {
		t.Errorf("calls = %d", cl.CallCount())
	}
}

func TestSpyderBrandsStopsAtTenPages(t *testing.T) {
	h := newHarness(t)
	cl, _ := h.client(t, func(w http.ResponseWriter, r *http.Request) {
		off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		io.WriteString(w, envelope(brands(off, 10)))
	})
	got, err := cl.SpyderBrands(context.Background())
	if err != nil || len(got) != 100 || cl.CallCount() != 10 {
		t.Fatalf("len=%d calls=%d err=%v", len(got), cl.CallCount(), err)
	}
}

func TestAdReadsSendFiltersAndKeepUnknownFields(t *testing.T) {
	h := newHarness(t)
	cl, f := h.client(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/ad":
			io.WriteString(w, envelope(adData))
		default:
			io.WriteString(w, envelope("["+adData+"]"))
		}
	})
	ctx := context.Background()
	live, minDays, limit := true, 30, 25

	ads, err := cl.SpyderBrandAds(ctx, "br_1", AdFilters{Live: &live, Limit: &limit, Order: "oldest"})
	if err != nil || len(ads) != 1 {
		t.Fatalf("ads=%v err=%v", ads, err)
	}
	a := ads[0]
	if a.ID != "ad_1" || *a.Headline != "AI intake that books calls" || a.Live == nil || !*a.Live ||
		a.RunningDuration == nil || a.RunningDuration.Seconds != 2592000 || a.EmotionalDrivers["urgency"] != 0.8 ||
		len(a.TimestampedTranscription) != 1 || a.Image != nil || len(a.PublisherPlatform) != 2 {
		t.Fatalf("ad = %+v", a)
	}
	if !strings.Contains(string(a.Raw), `"extra_field":"kept"`) {
		t.Errorf("unknown fields must ride along: %s", a.Raw)
	}
	q := f.requests[0].URL.Query()
	if f.requests[0].URL.Path != "/api/spyder/brand/ads" || q.Get("brand_id") != "br_1" || q.Get("live") != "true" ||
		q.Get("limit") != "25" || q.Get("order") != "oldest" || q.Has("start_date") || q.Has("display_format") {
		t.Errorf("query = %s", f.requests[0].URL)
	}

	if _, err := cl.AdsByBrandID(ctx, "br_1,br_2", AdFilters{StartDate: "2026-09-01", DisplayFormat: "video", RunningDurationMinDays: &minDays}); err != nil {
		t.Fatal(err)
	}
	q = f.requests[1].URL.Query()
	if f.requests[1].URL.Path != "/api/brand/getAdsByBrandId" || q.Get("brand_ids") != "br_1,br_2" || q.Get("start_date") != "2026-09-01" ||
		q.Get("display_format") != "video" || q.Get("running_duration_min_days") != "30" {
		t.Errorf("query = %s", f.requests[1].URL)
	}

	if _, err := cl.DiscoveryAds(ctx, "ai receptionist", AdFilters{}); err != nil {
		t.Fatal(err)
	}
	if f.requests[2].URL.Path != "/api/discovery/ads" || f.requests[2].URL.Query().Get("query") != "ai receptionist" {
		t.Errorf("discovery = %s", f.requests[2].URL)
	}

	one, err := cl.AdDetails(ctx, "ad_1")
	if err != nil || one.ID != "ad_1" || f.requests[3].URL.Query().Get("ad_id") != "ad_1" {
		t.Fatalf("one=%+v err=%v url=%s", one, err, f.requests[3].URL)
	}

	if _, err := cl.AdDuplicates(ctx, "ad/1 x"); err != nil {
		t.Fatal(err)
	}
	if f.requests[4].URL.EscapedPath() != "/api/ad/duplicates/ad%2F1%20x" {
		t.Errorf("duplicates path = %s", f.requests[4].URL.EscapedPath())
	}
	if cl.CallCount() != 5 {
		t.Errorf("calls = %d", cl.CallCount())
	}
}

func TestBrandReads(t *testing.T) {
	h := newHarness(t)
	cl, f := h.client(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/brand/analytics" {
			io.WriteString(w, envelope(`[{"date":"2026-09-27","active_count":12,"inactive_count":3,"video":8,"image":4,"carousel":0},{"date":"2026-09-28","active_count":14}]`))
			return
		}
		io.WriteString(w, envelope(brands(1, 2)))
	})
	ctx := context.Background()
	days, err := cl.BrandAnalytics(ctx, "br_1", "2026-09-01", "")
	if err != nil || len(days) != 2 || days[0].ActiveCount != 12 || days[1].Video != nil || *days[0].InactiveCount != 3 {
		t.Fatalf("days=%+v err=%v", days, err)
	}
	q := f.requests[0].URL.Query()
	if q.Get("id") != "br_1" || q.Get("start_date") != "2026-09-01" || q.Has("end_date") {
		t.Errorf("analytics query = %s", f.requests[0].URL)
	}

	got, err := cl.BrandsByDomain(ctx, "vantage.example", 0)
	if err != nil || len(got) != 2 || got[0].Name != "Brand 1" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	q = f.requests[1].URL.Query()
	if f.requests[1].URL.Path != "/api/brand/getBrandsByDomain" || q.Get("domain") != "vantage.example" || q.Get("limit") != "10" {
		t.Errorf("by-domain query = %s", f.requests[1].URL)
	}
}

func TestErrorsAreHonest(t *testing.T) {
	h := newHarness(t)
	var reply func(w http.ResponseWriter)
	cl, _ := h.client(t, func(w http.ResponseWriter, r *http.Request) { reply(w) })
	ctx := context.Background()

	reply = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusPaymentRequired)
		io.WriteString(w, `{"message":"Insufficient credits"}`)
	}
	_, err := cl.Usage(ctx)
	var fe *Error
	if !errors.As(err, &fe) || fe.Status != 402 || err.Error() != "Foreplay /api/usage: Insufficient credits" {
		t.Fatalf("err = %v", err)
	}

	reply = func(w http.ResponseWriter) { w.WriteHeader(http.StatusInternalServerError) }
	if _, err := cl.Usage(ctx); err == nil || err.Error() != "Foreplay /api/usage: HTTP 500" {
		t.Errorf("err = %v", err)
	}

	reply = func(w http.ResponseWriter) { io.WriteString(w, `[1,2,3]`) }
	if _, err := cl.Usage(ctx); err == nil || err.Error() != "Foreplay /api/usage: unrecognized response shape" {
		t.Errorf("err = %v", err)
	}

	// A row missing a required field fails the call rather than yielding a
	// half-empty ad.
	reply = func(w http.ResponseWriter) { io.WriteString(w, envelope(`[{"name":"no id"}]`)) }
	if ads, err := cl.DiscoveryAds(ctx, "x", AdFilters{}); err == nil || ads != nil {
		t.Errorf("ads=%v err=%v", ads, err)
	}
	reply = func(w http.ResponseWriter) { io.WriteString(w, envelope(`{"start_date":"2026-09-01"}`)) }
	if _, err := cl.Usage(ctx); err == nil {
		t.Error("usage without credit fields must fail")
	}
	reply = func(w http.ResponseWriter) { io.WriteString(w, envelope(`[{"date":"2026-09-01"}]`)) }
	if _, err := cl.BrandAnalytics(ctx, "b", "", ""); err == nil {
		t.Error("an analytics day without active_count must fail")
	}
	reply = func(w http.ResponseWriter) { io.WriteString(w, envelope(`[{"id":"b"}]`)) }
	if _, err := cl.BrandsByDomain(ctx, "d", 5); err == nil {
		t.Error("a brand without a name must fail")
	}
}

func TestStoreReads(t *testing.T) {
	h := newHarness(t)
	c := h.connector(t, "")
	if w, err := c.Store.ReadWatchlist(); err != nil || len(w) != 0 {
		t.Fatalf("empty store: w=%v err=%v", w, err)
	}
	h.writeStore(t, "watchlist.json", brands(1, 2))
	h.writeStore(t, "ads-br_1.json", "["+adData+"]")
	h.writeStore(t, "ads-weird_id.json", "["+adData+"]")
	h.writeStore(t, "analytics-br_1.json", `[{"date":"2026-09-27","active_count":12}]`)

	w, err := c.Store.ReadWatchlist()
	if err != nil || len(w) != 2 || w[1].ID != "br_2" {
		t.Fatalf("w=%v err=%v", w, err)
	}
	ads, err := c.Store.ReadBrandAds("br_1")
	if err != nil || len(ads) != 1 || ads[0].ID != "ad_1" {
		t.Fatalf("ads=%v err=%v", ads, err)
	}
	// Brand ids become filenames through the same safe-charset rule as the TS.
	if ads, _ := c.Store.ReadBrandAds("weird/id"); len(ads) != 1 {
		t.Errorf("sanitised filename not used")
	}
	days, err := c.Store.ReadAnalytics("br_1")
	if err != nil || len(days) != 1 {
		t.Fatalf("days=%v err=%v", days, err)
	}
	h.writeStore(t, "watchlist.json", `{broken`)
	if _, err := c.Store.ReadWatchlist(); err == nil {
		t.Error("a corrupt snapshot is an error, not an empty watchlist")
	}
}

func TestFormatNumberMatchesToLocaleString(t *testing.T) {
	for in, want := range map[float64]string{0: "0", 999: "999", 1000: "1,000", 9640: "9,640", 1234567: "1,234,567", 1234.5: "1,234.5", 2.12345: "2.123", -1500: "-1,500"} {
		if got := formatNumber(in); got != want {
			t.Errorf("formatNumber(%v) = %q, want %q", in, got, want)
		}
	}
}
