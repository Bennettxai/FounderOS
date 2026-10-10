package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/foreplay"
	"github.com/rhl/businessos-backend/internal/founderos/pages/adpilot"
)

type adpilotFakeLLM struct {
	reply string
	err   error
	calls int
}

func (f *adpilotFakeLLM) Complete(context.Context, string) (string, error) {
	f.calls++
	return f.reply, f.err
}

// adpilotFakeAPI is a Foreplay stand-in: no test spends credits.
type adpilotFakeAPI struct {
	calls     int64
	discovery []foreplay.Ad
	domains   []foreplay.SpyderBrand
}

func (f *adpilotFakeAPI) CallCount() int64 { return f.calls }
func (f *adpilotFakeAPI) AdsByBrandID(context.Context, string, foreplay.AdFilters) ([]foreplay.Ad, error) {
	f.calls++
	return nil, nil
}
func (f *adpilotFakeAPI) BrandAnalytics(context.Context, string, string, string) ([]foreplay.AnalyticsDay, error) {
	f.calls++
	return nil, nil
}
func (f *adpilotFakeAPI) Usage(context.Context) (*foreplay.Usage, error) {
	f.calls++
	return &foreplay.Usage{TotalCredits: 10000, RemainingCredits: 9000}, nil
}
func (f *adpilotFakeAPI) BrandsByDomain(context.Context, string, int) ([]foreplay.SpyderBrand, error) {
	f.calls++
	return f.domains, nil
}
func (f *adpilotFakeAPI) DiscoveryAds(context.Context, string, foreplay.AdFilters) ([]foreplay.Ad, error) {
	f.calls++
	return f.discovery, nil
}

type adpilotHarness struct {
	env *adpilotEnv
	r   *gin.Engine
	api *adpilotFakeAPI
	llm *adpilotFakeLLM
}

func adpilotNewHarness(t *testing.T, configured bool) *adpilotHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	h := &adpilotHarness{api: &adpilotFakeAPI{}, llm: &adpilotFakeLLM{}}
	h.env = &adpilotEnv{
		Store:         adpilot.Store{Dir: filepath.Join(dir, "ad-intel")},
		CampaignsPath: filepath.Join(dir, "adpilot-campaigns.json"),
		LLM:           h.llm,
		Now:           func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) },
		API: func() (adpilot.API, error) {
			if !configured {
				return nil, foreplay.ErrNotConfigured
			}
			return h.api, nil
		},
	}
	h.r = gin.New()
	adpilotRoutes(h.r.Group("/api/founderos"), h.env)
	return h
}

func (h *adpilotHarness) do(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rd *bytes.Reader
	if s, ok := body.(string); ok {
		rd = bytes.NewReader([]byte(s))
	} else if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, "/api/founderos"+path, rd)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: not JSON: %s", method, path, w.Body)
	}
	return w.Code, out
}

func (h *adpilotHarness) seedStore(t *testing.T) {
	t.Helper()
	s := h.env.Store
	now := h.env.Now()
	if _, err := s.AddWatchEntry(adpilot.WatchEntry{ID: "br_1", Name: "Acme"}, now); err != nil {
		t.Fatal(err)
	}
	live := true
	var ads []foreplay.Ad
	for i := 0; i < 90; i++ {
		ads = append(ads, foreplay.Ad{ID: "ad" + string(rune('A'+i%26)) + string(rune('a'+i/26)), Live: &live,
			RunningDuration: &foreplay.RunningDuration{Seconds: float64(i) * 86400}})
	}
	if err := s.WriteBrandAds("br_1", ads); err != nil {
		t.Fatal(err)
	}
	var sigs []adpilot.Signal
	for i := 0; i < 70; i++ {
		sigs = append(sigs, adpilot.Signal{Type: "winner", BrandID: "br_1", BrandName: "Acme", Message: "m", At: "2026-09-29T00:00:00.000Z"})
	}
	if err := s.AppendSignals(sigs, 500); err != nil {
		t.Fatal(err)
	}
}

func TestAdpilotPageNeverSyncedIsHonest(t *testing.T) {
	h := adpilotNewHarness(t, false)
	code, body := h.do(t, http.MethodGet, "/pages/adpilot", nil)
	if code != 200 {
		t.Fatalf("code %d %v", code, body)
	}
	deck := body["deck"].(map[string]any)
	lib := body["library"].(map[string]any)
	if len(deck["campaigns"].([]any)) != 0 || deck["views"] != nil || deck["syncedAt"] != nil {
		t.Fatalf("deck = %v", deck)
	}
	if lib["credits"] != nil || lib["lastSyncAt"] != nil || lib["configured"] != false || len(lib["wall"].([]any)) != 0 {
		t.Fatalf("library = %v", lib)
	}
	if h.llm.calls != 0 || h.api.calls != 0 {
		t.Fatal("page load must not call a model or spend credits")
	}
}

func TestAdpilotPageCapsWallAndSignalsAndReadsCredits(t *testing.T) {
	h := adpilotNewHarness(t, true)
	h.seedStore(t)
	if err := h.env.Store.WriteUsage(foreplay.Usage{TotalCredits: 10000, RemainingCredits: 9640}); err != nil {
		t.Fatal(err)
	}
	at := "2026-09-30T11:00:00.000Z"
	_ = h.env.Store.WriteMeta(foreplay.StoreMeta{LastSyncAt: &at, LastSyncCalls: 5})
	_ = os.WriteFile(h.env.CampaignsPath, []byte(`{"campaigns":[{"id":"c1","name":"Vantage","objective":"bookings","status":"active","platform":"Meta","period":{"from":"a","to":"b"},"spend":100,"impressions":1000,"clicks":10,"leads":5,"bookings":2,"purchases":0,"revenue":300,"audience":{"age":{},"gender":{},"placements":{}},"geo":[]}],"syncedAt":"2026-09-30T00:00:00Z"}`), 0o644)

	code, body := h.do(t, http.MethodGet, "/pages/adpilot", nil)
	lib := body["library"].(map[string]any)
	deck := body["deck"].(map[string]any)
	if code != 200 || len(lib["wall"].([]any)) != 80 || len(lib["signals"].([]any)) != 60 || len(lib["watchlist"].([]any)) != 1 {
		t.Fatalf("code %d library = %v", code, lib)
	}
	first := lib["wall"].([]any)[0].(map[string]any)
	if first["daysRunning"].(float64) != 89 {
		t.Fatalf("wall not longevity ranked: %v", first)
	}
	cr := lib["credits"].(map[string]any)
	if cr["remaining"].(float64) != 9640 || cr["total"].(float64) != 10000 || lib["lastSyncAt"] != at || lib["configured"] != true {
		t.Fatalf("credits = %v", lib)
	}
	views := deck["views"].(map[string]any)
	if deck["liveCount"].(float64) != 1 || views["all"] == nil || views["c1"] == nil {
		t.Fatalf("deck = %v", deck)
	}
	if _, has := body["errors"].(map[string]any)["store"]; has {
		t.Fatal("no store error")
	}
}

func TestAdpilotPageSurfacesCorruptSources(t *testing.T) {
	h := adpilotNewHarness(t, true)
	_ = os.MkdirAll(h.env.Store.Dir, 0o755)
	_ = os.WriteFile(filepath.Join(h.env.Store.Dir, "watchlist.json"), []byte("{"), 0o644)
	_ = os.WriteFile(h.env.CampaignsPath, []byte("{"), 0o644)
	code, body := h.do(t, http.MethodGet, "/pages/adpilot", nil)
	errs := body["errors"].(map[string]any)
	if code != 200 || errs["store"] == nil || errs["campaigns"] == nil {
		t.Fatalf("errors = %v", body)
	}
}

func TestAdpilotSavedRoundTrip(t *testing.T) {
	h := adpilotNewHarness(t, false)
	w := adpilot.ToWallAd(foreplay.Ad{ID: "ad_1"}, "Acme")
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/saved", map[string]any{"id": "x"}); code != 400 {
		t.Fatalf("partial snapshot: %d", code)
	}
	code, body := h.do(t, http.MethodPost, "/pages/adscout/saved", w)
	if code != 200 || body["ok"] != true || len(body["saved"].([]any)) != 1 {
		t.Fatalf("save: %d %v", code, body)
	}
	_, body = h.do(t, http.MethodGet, "/pages/adscout/saved", nil)
	if len(body["saved"].([]any)) != 1 {
		t.Fatal("list")
	}
	if code, _ := h.do(t, http.MethodDelete, "/pages/adscout/saved", map[string]any{}); code != 400 {
		t.Fatal("adId required")
	}
	_, body = h.do(t, http.MethodDelete, "/pages/adscout/saved", map[string]any{"adId": "ad_1"})
	if len(body["saved"].([]any)) != 0 {
		t.Fatal("unsave")
	}
}

func TestAdpilotWatchlist(t *testing.T) {
	h := adpilotNewHarness(t, false)
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/watchlist", map[string]any{"nothing": 1}); code != 400 {
		t.Fatal("bad body")
	}
	code, body := h.do(t, http.MethodPost, "/pages/adscout/watchlist", map[string]any{"brandId": "br_1", "name": "Acme", "avatar": nil})
	if code != 200 || len(body["watchlist"].([]any)) != 1 {
		t.Fatalf("add by id: %d %v", code, body)
	}
	// by domain needs Foreplay: unconfigured is an honest 503
	if code, body := h.do(t, http.MethodPost, "/pages/adscout/watchlist", map[string]any{"domain": "skool.com"}); code != 503 || !strings.Contains(body["error"].(string), "FOREPLAY_API_KEY") {
		t.Fatalf("domain unconfigured: %d %v", code, body)
	}
	_, body = h.do(t, http.MethodGet, "/pages/adscout/watchlist", nil)
	if len(body["watchlist"].([]any)) != 1 {
		t.Fatal("get")
	}
	_, body = h.do(t, http.MethodDelete, "/pages/adscout/watchlist", map[string]any{"brandId": "br_1"})
	if len(body["watchlist"].([]any)) != 0 {
		t.Fatal("remove")
	}
}

func TestAdpilotWatchlistByDomain(t *testing.T) {
	h := adpilotNewHarness(t, true)
	n := 3
	h.api.domains = []foreplay.SpyderBrand{{ID: "br_9", Name: "Skool", AdsCount: &n}}
	code, body := h.do(t, http.MethodPost, "/pages/adscout/watchlist", map[string]any{"domain": "skool.com"})
	if code != 200 || body["added"].(map[string]any)["id"] != "br_9" {
		t.Fatalf("%d %v", code, body)
	}
	h.api.domains = nil
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/watchlist", map[string]any{"domain": "nobody.io"}); code != 404 {
		t.Fatalf("unknown domain: %d", code)
	}
}

func TestAdpilotSync(t *testing.T) {
	h := adpilotNewHarness(t, false)
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/sync", nil); code != 503 {
		t.Fatal("unconfigured sync is 503")
	}
	h = adpilotNewHarness(t, true)
	code, body := h.do(t, http.MethodPost, "/pages/adscout/sync", nil)
	if code != 200 || body["ok"] != true || body["brands"].(float64) != 0 || body["remainingCredits"].(float64) != 9000 {
		t.Fatalf("sync: %d %v", code, body)
	}
}

func TestAdpilotAsk(t *testing.T) {
	h := adpilotNewHarness(t, false)
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/ask", map[string]any{"question": "hi"}); code != 400 {
		t.Fatal("short question")
	}
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/ask", map[string]any{"question": "what works?"}); code != 409 {
		t.Fatal("empty store is 409")
	}
	if h.llm.calls != 0 {
		t.Fatal("no model call on an empty store")
	}
	h.seedStore(t)
	h.llm.reply = "Acme runs long hooks."
	code, body := h.do(t, http.MethodPost, "/pages/adscout/ask", map[string]any{"question": "what works?"})
	if code != 200 || body["answer"] != "Acme runs long hooks." {
		t.Fatalf("ask: %d %v", code, body)
	}
	h.llm.err = adpilot.ErrLLMUnavailable
	if code, body := h.do(t, http.MethodPost, "/pages/adscout/ask", map[string]any{"question": "what works?"}); code != 503 || body["unavailable"] != true {
		t.Fatalf("unavailable: %d %v", code, body)
	}
	h.llm.err = errors.New("exit 1")
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/ask", map[string]any{"question": "what works?"}); code != 502 {
		t.Fatal("model error is 502")
	}
}

func TestAdpilotMine(t *testing.T) {
	h := adpilotNewHarness(t, false)
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/mine", map[string]any{"concept": "ai"}); code != 400 {
		t.Fatal("short concept")
	}
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/mine", map[string]any{"concept": "ai agents", "minDays": 2.5}); code != 400 {
		t.Fatal("fractional minDays")
	}
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/mine", map[string]any{"concept": "ai agents", "format": "gif"}); code != 400 {
		t.Fatal("bad format")
	}
	if code, _ := h.do(t, http.MethodPost, "/pages/adscout/mine", map[string]any{"concept": "ai agents"}); code != 503 {
		t.Fatal("unconfigured mine is 503")
	}
	h = adpilotNewHarness(t, true)
	live := true
	h.api.discovery = []foreplay.Ad{{ID: "w1", Live: &live, RunningDuration: &foreplay.RunningDuration{Seconds: 40 * 86400}}}
	h.llm.err = adpilot.ErrLLMUnavailable
	code, body := h.do(t, http.MethodPost, "/pages/adscout/mine", map[string]any{"concept": "ai agents for founders", "minDays": 30, "format": "video"})
	if code != 200 || body["expandedBy"] != "fallback" || body["pooled"].(float64) != 1 || len(body["winners"].([]any)) != 1 {
		t.Fatalf("mine: %d %v", code, body)
	}
	if w := body["winners"].([]any)[0].(map[string]any); w["daysRunning"].(float64) != 40 || w["brand"] != "unknown" {
		t.Fatalf("winner = %v", w)
	}
}

func TestAdpilotRoutesMatchTheRouteMap(t *testing.T) {
	h := adpilotNewHarness(t, false)
	want := map[string]bool{
		"GET /api/founderos/pages/adpilot": true, "POST /api/founderos/pages/adscout/ask": true, "POST /api/founderos/pages/adscout/mine": true,
		"GET /api/founderos/pages/adscout/saved": true, "POST /api/founderos/pages/adscout/saved": true, "DELETE /api/founderos/pages/adscout/saved": true,
		"POST /api/founderos/pages/adscout/sync": true, "GET /api/founderos/pages/adscout/watchlist": true,
		"POST /api/founderos/pages/adscout/watchlist": true, "DELETE /api/founderos/pages/adscout/watchlist": true,
	}
	for _, r := range h.r.Routes() {
		delete(want, r.Method+" "+r.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing routes: %v", want)
	}
}
