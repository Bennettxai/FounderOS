package branddeals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// A shape-accurate page object, as POST /v1/data_sources/{id}/query returns.
const acmePage = `{
 "object":"page","id":"abc-123","url":"https://www.notion.so/Acme-abc123","last_edited_time":"2026-08-10T12:00:00.000Z",
 "properties":{
  "Brand Name":{"type":"title","title":[{"plain_text":"Acme "},{"plain_text":"Tools"}]},
  "Status":{"type":"status","status":{"name":"Negotiating"}},
  "Brand Tier":{"type":"select","select":{"name":"A"}},
  "Deal Value":{"type":"number","number":4500},
  "Budget":{"type":"number","number":6000},
  "Amount Agreed":{"type":"number","number":null},
  "Suggested Rate":{"type":"number","number":5000},
  "Paid in Full":{"type":"checkbox","checkbox":false},
  "Deadline":{"type":"date","date":{"start":"2026-09-01"}},
  "Follow-up Date":{"type":"date","date":null},
  "Contact Name":{"type":"rich_text","rich_text":[{"plain_text":"Jo Doe"}]},
  "Contact Email":{"type":"email","email":"jo@acme.com"},
  "Main Channel":{"type":"select","select":{"name":"Instagram"}},
  "Video Type":{"type":"select","select":{"name":"Integration"}},
  "Source":{"type":"select","select":{"name":"Inbound"}},
  "ICP Fit":{"type":"select","select":{"name":"Strong"}}
 }}`

func page(id, brand, status string) string {
	return fmt.Sprintf(`{"object":"page","id":%q,"url":"https://www.notion.so/%s","last_edited_time":"2026-09-01T00:00:00.000Z",
	 "properties":{"Brand Name":{"type":"title","title":[{"plain_text":%q}]},"Status":{"type":"status","status":{"name":%q}}}}`, id, id, brand, status)
}

func resolver(t *testing.T, body string) connectors.Resolver {
	t.Helper()
	t.Setenv("NOTION_API_KEY", "")
	t.Setenv("NOTION_BRAND_DEALS_SOURCE", "")
	p := filepath.Join(t.TempDir(), "env.local")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return connectors.Resolver{EnvLocal: p}
}

type fake struct {
	mu      sync.Mutex
	hits    []string
	bodies  []map[string]any
	pages   [][]string // query result pages
	fail    string     // non-empty: answer every call with this Notion error code
	me      string
	version []string
}

func (f *fake) serve(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.hits = append(f.hits, r.Method+" "+r.URL.Path)
		f.version = append(f.version, r.Header.Get("Notion-Version"))
		var body map[string]any
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			json.Unmarshal(raw, &body)
		}
		f.bodies = append(f.bodies, body)
		if r.Header.Get("Authorization") != "Bearer ntn_test" {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}`)
			return
		}
		if f.fail != "" {
			w.WriteHeader(404)
			fmt.Fprintf(w, `{"object":"error","status":404,"code":%q,"message":"Could not find data_source with ID: x."}`, f.fail)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/users/me":
			fmt.Fprint(w, f.me)
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v1/data_sources/") && strings.HasSuffix(r.URL.Path, "/query"):
			idx := 0
			if c, _ := body["start_cursor"].(string); c != "" {
				fmt.Sscanf(c, "cur-%d", &idx)
			}
			more := idx+1 < len(f.pages)
			next := "null"
			if more {
				next = fmt.Sprintf(`"cur-%d"`, idx+1)
			}
			fmt.Fprintf(w, `{"object":"list","results":[%s],"has_more":%v,"next_cursor":%s}`, strings.Join(f.pages[idx], ","), more, next)
		case r.Method == "PATCH" && strings.HasPrefix(r.URL.Path, "/v1/pages/"):
			fmt.Fprint(w, `{"object":"page","id":"abc-123"}`)
		case r.Method == "POST" && r.URL.Path == "/v1/pages":
			fmt.Fprint(w, `{"object":"page","id":"new-1"}`)
		default:
			w.WriteHeader(404)
			fmt.Fprint(w, `{"object":"error","code":"invalid_request_url","message":"Invalid request URL."}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTest(t *testing.T, f *fake, env string) (*Connector, *time.Time) {
	c := New(resolver(t, env))
	c.baseURL = f.serve(t).URL + "/v1"
	now := time.UnixMilli(1_800_000_000_000)
	c.now = func() time.Time { return now }
	return c, &now
}

const keyed = "NOTION_API_KEY=ntn_test\n"

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta != (connectors.Meta{ID: "notion", Name: "Notion", Kind: connectors.KindNotion}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusThreeStates(t *testing.T) {
	f := &fake{me: `{"object":"user","id":"u1","name":"FounderOS v1","type":"bot"}`}
	c, _ := newTest(t, f, "")
	if s := c.Status(context.Background()); s.State != connectors.StateNotConfigured ||
		s.Detail != "Set NOTION_API_KEY (internal integration secret) in ~/.founderos/.env or under API keys, and share target pages with it." {
		t.Fatalf("unconfigured: %+v", s)
	}
	if len(f.hits) != 0 {
		t.Fatal("unconfigured must not call out")
	}

	c, _ = newTest(t, f, keyed)
	if s := c.Status(context.Background()); s.State != connectors.StateConnected || s.Detail != "Connected as FounderOS v1" {
		t.Fatalf("connected: %+v", s)
	}
	if f.version[len(f.version)-1] != "2025-09-03" {
		t.Fatalf("Notion-Version = %q", f.version[len(f.version)-1])
	}
	f.me = `{"object":"user","id":"u1","name":null,"type":"bot"}`
	if s := c.Status(context.Background()); s.Detail != "Connected as integration" {
		t.Fatalf("nameless bot: %+v", s)
	}

	c, _ = newTest(t, f, "NOTION_API_KEY=wrong\n")
	if s := c.Status(context.Background()); s.State != connectors.StateError || s.Detail != "Key set but auth failed: API token is invalid." {
		t.Fatalf("auth failure: %+v", s)
	}
	c.baseURL = "http://127.0.0.1:1/v1"
	if s := c.Status(context.Background()); s.State != connectors.StateError {
		t.Fatalf("unreachable: %+v", s)
	}
}

func TestParseDealMapsEveryProperty(t *testing.T) {
	d, ok := ParseDeal([]byte(acmePage))
	if !ok {
		t.Fatal("not parsed")
	}
	if d.ID != "abc-123" || d.Brand != "Acme Tools" || d.Status != "Negotiating" || *d.Tier != "A" ||
		*d.DealValueUSD != 4500 || *d.BudgetUSD != 6000 || d.AmountAgreedUSD != nil || *d.SuggestedRateUSD != 5000 ||
		d.PaidInFull || *d.Deadline != "2026-09-01" || d.FollowUpDate != nil || *d.ContactName != "Jo Doe" ||
		*d.ContactEmail != "jo@acme.com" || *d.MainChannel != "Instagram" || *d.VideoType != "Integration" ||
		*d.Source != "Inbound" || *d.ICPFit != "Strong" || d.NotionURL != "https://www.notion.so/Acme-abc123" ||
		d.LastEdited != "2026-08-10T12:00:00.000Z" || d.Seeded {
		raw, _ := json.Marshal(d)
		t.Fatalf("deal = %s", raw)
	}
	raw, _ := json.Marshal(d)
	for _, want := range []string{`"amountAgreedUsd":null`, `"dealValueUsd":4500`, `"icpFit":"Strong"`, `"seeded":false`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("JSON %s lacks %s", raw, want)
		}
	}
}

func TestParseDealToleratesMissingPropsAndDropsUntitled(t *testing.T) {
	d, ok := ParseDeal([]byte(`{"id":"x-y","last_edited_time":"2026-08-01T00:00:00.000Z","properties":{"Brand Name":{"title":[{"plain_text":"Mystery Co"}]}}}`))
	if !ok || d.Brand != "Mystery Co" || d.Status != "New" || d.DealValueUSD != nil || d.NotionURL != "https://www.notion.so/xy" {
		t.Fatalf("bare = %+v ok=%v", d, ok)
	}
	if _, ok := ParseDeal([]byte(`{"id":"y","url":"u","last_edited_time":"t","properties":{}}`)); ok {
		t.Fatal("untitled page must be dropped")
	}
	if _, ok := ParseDeal([]byte(`{"id":"y","properties":{"Brand Name":{"title":[{"plain_text":"  "}]}}}`)); ok {
		t.Fatal("blank title must be dropped")
	}
}

func TestGroupDealsPipelineOrderAndUnknownLanes(t *testing.T) {
	deals := []Deal{{Status: "Paid"}, {Status: "New"}, {Status: "Negotiating"}, {Status: "Weird Custom"}}
	lanes := GroupDeals(deals)
	var named []string
	for _, l := range lanes {
		if len(l.Deals) > 0 {
			named = append(named, l.Status)
		}
	}
	if fmt.Sprint(named) != "[New Negotiating Paid Weird Custom]" {
		t.Fatalf("lanes = %v", named)
	}
	if len(lanes) != len(PipelineOrder)+1 {
		t.Fatalf("lane count = %d", len(lanes))
	}
}

func TestSourceIDResolvesPerCallWithDefault(t *testing.T) {
	c, _ := newTest(t, &fake{}, keyed)
	if got := c.SourceID(); got != "3bd9c1cb-baa1-8171-9b0d-000bb2d72175" {
		t.Fatalf("default = %s", got)
	}
	os.WriteFile(c.res.EnvLocal, []byte(keyed+"NOTION_BRAND_DEALS_SOURCE=other\n"), 0o600)
	if got := c.SourceID(); got != "other" {
		t.Fatalf("override = %s", got)
	}
}

func TestSeededWithoutKey(t *testing.T) {
	f := &fake{}
	c, _ := newTest(t, f, "")
	r := c.FetchDeals(context.Background())
	if r.Mode != ModeSeeded || len(r.Deals) < 4 || r.SyncedAt != nil || len(f.hits) != 0 {
		t.Fatalf("result = %+v", r)
	}
	for _, d := range r.Deals {
		if !d.Seeded {
			t.Fatalf("seeded row not marked: %+v", d)
		}
	}
}

func TestFetchDealsPaginatesAndCaches(t *testing.T) {
	f := &fake{pages: [][]string{
		{acmePage, page("p2", "Beta", "Paid")},
		{page("p3", "", "New"), page("p4", "Gamma", "Filming")},
	}}
	c, now := newTest(t, f, keyed+"NOTION_BRAND_DEALS_SOURCE=src-1\n")
	r := c.FetchDeals(context.Background())
	if r.Mode != ModeLive || len(r.Deals) != 3 || r.Detail != "Brand Deals Hub · 3 deals" || r.SyncedAt == nil || *r.SyncedAt != 1_800_000_000_000 {
		t.Fatalf("result = %+v", r)
	}
	if len(f.hits) != 2 || f.hits[0] != "POST /v1/data_sources/src-1/query" {
		t.Fatalf("hits = %v", f.hits)
	}
	if f.bodies[0]["page_size"] != float64(100) || f.bodies[0]["start_cursor"] != nil || f.bodies[1]["start_cursor"] != "cur-1" {
		t.Fatalf("bodies = %v", f.bodies)
	}
	*now = now.Add(19 * time.Minute)
	if r := c.FetchDeals(context.Background()); len(f.hits) != 2 || len(r.Deals) != 3 || *r.SyncedAt != 1_800_000_000_000 {
		t.Fatal("20-minute cache missed")
	}
	*now = now.Add(2 * time.Minute)
	c.FetchDeals(context.Background())
	if len(f.hits) != 4 {
		t.Fatalf("expired cache must refetch, hits=%d", len(f.hits))
	}
}

func TestFetchDealsErrorAndStaleFallback(t *testing.T) {
	f := &fake{fail: "object_not_found"}
	c, now := newTest(t, f, keyed)
	r := c.FetchDeals(context.Background())
	if r.Mode != ModeError || r.Deals == nil || len(r.Deals) != 0 || r.SyncedAt != nil ||
		r.Detail != "Key set but the fetch failed: Could not find data_source with ID: x.. Is the integration shared with Brand Deals Hub?" {
		t.Fatalf("error result = %+v", r)
	}
	f.fail = ""
	f.pages = [][]string{{acmePage}}
	if r := c.FetchDeals(context.Background()); r.Mode != ModeLive {
		t.Fatalf("recovered = %+v", r)
	}
	*now = now.Add(time.Hour)
	f.fail = "service_unavailable"
	r = c.FetchDeals(context.Background())
	if r.Mode != ModeLive || len(r.Deals) != 1 || r.Detail != "Notion unreachable · showing the last good read" || *r.SyncedAt != 1_800_000_000_000 {
		t.Fatalf("stale fallback = %+v", r)
	}
}

func TestFetchDealsDoesNotCacheAnEmptyBoard(t *testing.T) {
	f := &fake{pages: [][]string{{}}}
	c, _ := newTest(t, f, keyed)
	c.FetchDeals(context.Background())
	c.FetchDeals(context.Background())
	if len(f.hits) != 2 {
		t.Fatalf("an empty read must not be cached, hits=%d", len(f.hits))
	}
}

func TestWritesRefusedWhenWritesOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := &fake{}
	c, _ := newTest(t, f, keyed)
	if err := c.UpdatePage(context.Background(), "abc-123", map[string]any{"Status": map[string]any{"status": map[string]any{"name": "Paid"}}}); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("update err = %v", err)
	}
	if _, err := c.CreatePage(context.Background(), map[string]any{}); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("create err = %v", err)
	}
	if len(f.hits) != 0 {
		t.Fatalf("a refused write reached Notion: %v", f.hits)
	}
	r := guard.Refused()
	if len(r) != 2 || r[0].Action != "notion.pages.update" || r[1].Action != "notion.pages.create" {
		t.Fatalf("refusals = %+v", r)
	}
}

func TestWritesWhenWritesOn(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := &fake{}
	c, _ := newTest(t, f, keyed+"NOTION_BRAND_DEALS_SOURCE=src-1\n")
	if err := c.UpdatePage(context.Background(), "abc-123", map[string]any{"Paid in Full": map[string]any{"checkbox": true}}); err != nil {
		t.Fatal(err)
	}
	id, err := c.CreatePage(context.Background(), map[string]any{"Brand Name": map[string]any{"title": []any{}}})
	if err != nil || id != "new-1" {
		t.Fatalf("create id=%q err=%v", id, err)
	}
	if f.hits[0] != "PATCH /v1/pages/abc-123" || f.hits[1] != "POST /v1/pages" {
		t.Fatalf("hits = %v", f.hits)
	}
	parent, _ := f.bodies[1]["parent"].(map[string]any)
	if parent["data_source_id"] != "src-1" || parent["type"] != "data_source_id" {
		t.Fatalf("parent = %v", f.bodies[1]["parent"])
	}
}

// The data-source query is a read that happens to be a POST: the guard must
// let it through to the real host while blocking page writes.
func TestReadPOSTsAllowOnlyTheQuery(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	client := &http.Client{Transport: guard.Transport(rewrite{srv.URL}, ReadPOSTs...)}
	req, _ := http.NewRequest("POST", "https://api.notion.com/v1/data_sources/src-1/query", strings.NewReader("{}"))
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("query refused: %v", err)
	}
	resp.Body.Close()
	req, _ = http.NewRequest("POST", "https://api.notion.com/v1/pages", strings.NewReader("{}"))
	if _, err := client.Do(req); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Fatalf("page create passed the guard: %v", err)
	}
}

type rewrite struct{ to string }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	u, _ := out.URL.Parse(r.to + req.URL.Path)
	out.URL, out.Host = u, u.Host
	return http.DefaultTransport.RoundTrip(out)
}
