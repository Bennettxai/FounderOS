package trakyo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Fixtures mirror the Trakyo API v1 envelopes (api.trakyo.io/openapi.json,
// checked 2026-09-23). Every figure, name and email is invented.
const metricsBody = `{
  "object": "metrics",
  "range": {"start": "2026-07-01", "end": "2026-07-31", "timezone": "UTC"},
  "attribution": "first_touch",
  "currency": "USD",
  "totals": {"clicks": 1200, "visits": 90, "form_submissions": 8, "bookings": 3, "transactions": 2, "revenue": "4500.50", "aov": "1500.17"}
}`

const leadsBody = `{
  "object": "list",
  "data": [
    {"object": "lead", "id": "lead_1",
     "identifiers": {"names": ["Ada Example"], "emails": ["ada@example.com"], "phones": []},
     "first_touch": {"type": "youtube", "name": "YouTube", "source_name": "YouTube",
       "content_item": {"name": "How the operator console works"}, "occurred_at": "2026-07-14T10:12:00+00:00"}},
    {"object": "lead", "id": "lead_2",
     "identifiers": {"names": [], "emails": ["no-name@example.com"], "phones": []},
     "first_touch": {"type": "referrer", "name": "instagram.com", "source_name": null, "content_item": null,
       "occurred_at": "2026-07-02T08:00:00+00:00"}},
    {"object": "lead", "id": "lead_3",
     "identifiers": {"names": ["No Attribution"], "emails": [], "phones": []}, "first_touch": null},
    {"object": "lead", "id": "lead_4",
     "identifiers": {"names": ["Paid Person"], "emails": ["paid@example.com"], "phones": []},
     "first_touch": {"type": "meta_ad", "name": "Meta Ads", "source_name": "Meta Ads",
       "content_item": {"name": "Cold traffic: stop selling hours"}, "occurred_at": "2026-07-20T09:00:00+00:00"}}
  ]
}`

func contentItem(id string, views string) string {
	return `{"id": "` + id + `", "type": "youtube", "name": "Example video", "source": "YouTube",
	  "views": ` + views + `, "tracked_since": null, "views_at_tracked_since": null, "published_at": null,
	  "clicks": 10, "visits": 2, "form_submissions": 2, "bookings": 1,
	  "revenue": {"first_touch": "19.99", "last_touch": "9.99"}}`
}

func envelope(hasMore bool, items ...string) string {
	more := "false"
	if hasMore {
		more = "true"
	}
	return `{"object": "list", "data": [` + strings.Join(items, ",") + `], "has_more": ` + more + `, "currency": "USD"}`
}

const funnelBody = `{"object": "funnel", "currency": "USD", "attribution": "first_touch",
  "range": {"start": "2026-08-25", "end": "2026-09-23", "timezone": "America/Chicago"},
  "counts": {"clicks": 10, "visits": 2, "form_submissions": 2, "bookings": 1, "closes": 1}, "revenue": "19.99"}`

type hit struct {
	method, path, rawQuery, auth, body string
}

type fake struct {
	mu   sync.Mutex
	hits []hit
	srv  *httptest.Server
}

// newFake answers with route(r) → (status, body).
func newFake(t *testing.T, route func(r *http.Request) (int, string)) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.hits = append(f.hits, hit{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), string(b)})
		f.mu.Unlock()
		status, body := route(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func fixed(status int, body string) func(*http.Request) (int, string) {
	return func(*http.Request) (int, string) { return status, body }
}

func (f *fake) calls() []hit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hit(nil), f.hits...)
}

func keyed(t *testing.T, key, base string) *Connector {
	t.Helper()
	dir := t.TempDir()
	envLocal := filepath.Join(dir, "env.local")
	if key != "" {
		if err := os.WriteFile(envLocal, []byte("TRAKYO_API_KEY="+key+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TRAKYO_API_KEY", "")
	c := New(connectors.Resolver{EnvLocal: envLocal})
	c.BaseURL = base
	c.CredFiles = []string{filepath.Join(dir, "absent.env")}
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "trakyo" || Meta.Name != "Trakyo" || Meta.Kind != connectors.KindCRM {
		t.Fatalf("meta = %+v", Meta)
	}
}

func TestParseMetrics(t *testing.T) {
	m, ok := ParseMetrics([]byte(metricsBody))
	if !ok {
		t.Fatal("metrics must parse")
	}
	want := Metrics{Clicks: 1200, Visits: 90, FormSubmissions: 8, Bookings: 3, Transactions: 2, RevenueUSD: 4500.5, RangeStart: "2026-07-01", RangeEnd: "2026-07-31"}
	if m != want {
		t.Errorf("got %+v want %+v", m, want)
	}
	// Omitted counters are zero for the range (Trakyo drops them), not unknown.
	sparse, ok := ParseMetrics([]byte(`{"totals": {"clicks": 5, "revenue": "0"}}`))
	if !ok || sparse.Clicks != 5 || sparse.Visits != 0 || sparse.RangeStart != "" {
		t.Errorf("sparse = %+v ok=%v", sparse, ok)
	}
	for _, body := range []string{`{"object":"metrics"}`, `null`, `[]`, `nope`} {
		if _, ok := ParseMetrics([]byte(body)); ok {
			t.Errorf("%q has no totals and must not read as zeros", body)
		}
	}
}

func TestStatusNotConfigured(t *testing.T) {
	f := newFake(t, fixed(200, metricsBody))
	s := keyed(t, "", f.srv.URL).Status(context.Background())
	if s.State != connectors.StateNotConfigured || !strings.Contains(s.Detail, "TRAKYO_API_KEY") {
		t.Fatalf("status = %+v", s)
	}
	if len(f.calls()) != 0 {
		t.Fatal("no call without a key")
	}
}

func TestStatusConnectedReportsLiveTotals(t *testing.T) {
	f := newFake(t, fixed(200, metricsBody))
	s := keyed(t, "tky_test", f.srv.URL).Status(context.Background())
	if s.State != connectors.StateConnected {
		t.Fatalf("status = %+v", s)
	}
	want := "1200 clicks · 90 visits · 8 form submissions · $4,500.5 revenue (2026-07-01 → 2026-07-31)"
	if s.Detail != want {
		t.Errorf("detail = %q\nwant     %q", s.Detail, want)
	}
	if s.Meta["clicks"] != float64(1200) || s.Meta["revenueUsd"] != 4500.5 || s.Meta["bookings"] != float64(3) {
		t.Errorf("meta = %+v", s.Meta)
	}
	c := f.calls()
	if len(c) != 1 || c[0].path != "/metrics" || c[0].auth != "Bearer tky_test" {
		t.Errorf("calls = %+v", c)
	}
}

func TestStatusErrorStates(t *testing.T) {
	forbidden := keyed(t, "k", newFake(t, fixed(403, `{"error":{"code":"insufficient_scope"}}`)).srv.URL).Status(context.Background())
	if forbidden.State != connectors.StateError || forbidden.Detail != "TRAKYO_API_KEY set but the Trakyo API call failed: HTTP 403" {
		t.Errorf("forbidden = %+v", forbidden)
	}
	empty := keyed(t, "k", newFake(t, fixed(200, `{"object":"metrics"}`)).srv.URL).Status(context.Background())
	if empty.State != connectors.StateError || !strings.Contains(empty.Detail, "metrics response carried no totals") {
		t.Errorf("no totals = %+v", empty)
	}
	down := keyed(t, "k", "http://127.0.0.1:1").Status(context.Background())
	if down.State != connectors.StateError {
		t.Errorf("down = %+v", down)
	}
}

func TestLeadsMapsAttributedFirstTouches(t *testing.T) {
	f := newFake(t, fixed(200, leadsBody))
	events, err := keyed(t, "k", f.srv.URL).Leads(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("lead without first_touch is skipped; got %d: %+v", len(events), events)
	}
	ada := events[0]
	if ada.Lead != "Ada Example" || ada.Label != "How the operator console works" || ada.Channel != "organic" ||
		ada.At != "2026-07-14" || ada.SourceType != "youtube" || ada.SourceName == nil || *ada.SourceName != "YouTube" {
		t.Errorf("ada = %+v", ada)
	}
	if events[1].Lead != "no-name@example.com" || events[1].Label != "instagram.com" || events[1].SourceName != nil {
		t.Errorf("email-only lead = %+v", events[1])
	}
	if events[2].Channel != "ads" {
		t.Errorf("meta_ad first touch is ads: %+v", events[2])
	}
	if c := f.calls(); c[0].path != "/leads" || c[0].rawQuery != "limit=100" {
		t.Errorf("calls = %+v", c)
	}
	if _, err := keyed(t, "k", newFake(t, fixed(500, `{}`)).srv.URL).Leads(context.Background()); err == nil {
		t.Error("a failure is an error, not an empty lead list")
	}
	if _, err := keyed(t, "", "http://127.0.0.1:1").Leads(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v", err)
	}
}

func TestAnalyticsPaginatesContent(t *testing.T) {
	f := newFake(t, func(r *http.Request) (int, string) {
		q := r.URL.Query()
		if q.Get("period") != "7d" || q.Get("currency") != "USD" || q.Get("attribution") != "first_touch" || q.Get("timezone") != "America/Chicago" {
			return 400, `{}`
		}
		if strings.HasSuffix(r.URL.Path, "/funnel") {
			return 200, funnelBody
		}
		if q.Get("offset") == "0" {
			return 200, envelope(true, contentItem("ci_one", "1000"))
		}
		return 200, envelope(false, contentItem("ci_two", "null"))
	})
	a := keyed(t, "k", f.srv.URL).Analytics(context.Background(), "7d")
	if a.Content.State != "ready" || len(a.Content.Rows) != 2 {
		t.Fatalf("content = %+v", a.Content)
	}
	if a.Funnel.State != "ready" || a.Funnel.Data == nil || a.Funnel.Data.Counts.Closes != 1 || a.Funnel.Data.Revenue != "19.99" {
		t.Fatalf("funnel = %+v", a.Funnel)
	}
	if a.Content.Rows[1].Views != nil {
		t.Error("null views stay unknown, never 0")
	}
	sawOffset1 := false
	for _, h := range f.calls() {
		if strings.Contains(h.rawQuery, "offset=1") && strings.Contains(h.rawQuery, "sort=revenue") && strings.Contains(h.rawQuery, "limit=100") {
			sawOffset1 = true
		}
	}
	if !sawOffset1 {
		t.Errorf("second page must ask for offset=1: %+v", f.calls())
	}
}

func TestAnalyticsPartialIsNeverShownComplete(t *testing.T) {
	f := newFake(t, func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/funnel") {
			return 200, funnelBody
		}
		if r.URL.Query().Get("offset") == "0" {
			return 200, envelope(true, contentItem("ci_one", "5"))
		}
		return 403, `{}`
	})
	a := keyed(t, "k", f.srv.URL).Analytics(context.Background(), "30d")
	if a.Content.State != "partial" || len(a.Content.Rows) != 1 || a.Content.Message == nil || !strings.Contains(*a.Content.Message, "403") {
		t.Errorf("content = %+v", a.Content)
	}
	if a.Funnel.State != "ready" {
		t.Errorf("funnel = %+v", a.Funnel)
	}
}

func TestAnalyticsFailsClosedOnInvalidPayloads(t *testing.T) {
	for _, body := range []string{
		envelope(false, `{"name": "broken"}`),
		strings.Replace(envelope(false, contentItem("ci_x", "1")), `"USD"`, `"EUR"`, 1),
		envelope(false, strings.Replace(contentItem("ci_x", "1"), `"clicks": 10`, `"clicks": -1`, 1)),
		envelope(false, strings.Replace(contentItem("ci_x", "1"), `"19.99"`, `"lots"`, 1)),
	} {
		a := keyed(t, "k", newFake(t, fixed(200, body)).srv.URL).Analytics(context.Background(), "30d")
		if a.Content.State != "error" || len(a.Content.Rows) != 0 {
			t.Errorf("body %s → %+v", body, a.Content)
		}
		if a.Funnel.State != "error" || a.Funnel.Data != nil {
			t.Errorf("a content envelope is not a funnel: %+v", a.Funnel)
		}
	}
}

func TestAnalyticsEmptyIsReadyAndNoKeyIsNotConfigured(t *testing.T) {
	f := newFake(t, func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/funnel") {
			return 200, funnelBody
		}
		return 200, envelope(false)
	})
	a := keyed(t, "k", f.srv.URL).Analytics(context.Background(), "90d")
	if a.Content.State != "ready" || len(a.Content.Rows) != 0 {
		t.Errorf("empty success = %+v", a.Content)
	}
	none := keyed(t, "", f.srv.URL)
	before := len(f.calls())
	n := none.Analytics(context.Background(), "30d")
	if n.Content.State != "not_configured" || n.Funnel.State != "not_configured" || len(f.calls()) != before {
		t.Errorf("no key = %+v", n)
	}
}

func TestChannelForContent(t *testing.T) {
	cases := [][2]string{
		{"Instagram bio", "instagram_bio"}, {"instagram.com", "instagram"}, {"LinkedIn", "linkedin"},
		{"X", "x"}, {"https://t.co/test", "x"}, {"Word of mouth", "word_of_mouth"}, {"Other", "unattributed"},
		{"Referrer Traffic", "unattributed"}, {"google.com", "search"},
	}
	for _, tc := range cases {
		if got := ChannelForContent("custom", "Campaign", tc[0]); got != tc[1] {
			t.Errorf("source %q → %q, want %q", tc[0], got, tc[1])
		}
	}
	if got := ChannelForContent("youtube", "My Instagram strategy", "YouTube"); got != "youtube" {
		t.Errorf("youtube type wins: %q", got)
	}
	if got := ChannelForContent("custom", "Example", "June webinar"); got != "unattributed" {
		t.Errorf("webinar: %q", got)
	}
	if got := ChannelForContent("custom", "instagram Manychat", "thefounderos-waitlist-launch"); got != "instagram" {
		t.Errorf("IG manychat: %q", got)
	}
	if got := ChannelForContent("custom", "Instagram strategy guide", "waitlist"); got != "forms" {
		t.Errorf("waitlist: %q", got)
	}
	if got := ChannelForContent("custom", "instagram Manychat", "LinkedIn"); got != "linkedin" {
		t.Errorf("linkedin: %q", got)
	}
}

func TestGroupChannelsKeepsUnknownViewsUnknown(t *testing.T) {
	page, err := parseContentPage([]byte(envelope(false, contentItem("ci_one", "1000"), contentItem("ci_two", "null"))))
	if err != nil {
		t.Fatal(err)
	}
	cards := GroupChannels(page.Data)
	var yt, bio *ChannelCard
	for i := range cards {
		switch cards[i].ID {
		case "youtube":
			yt = &cards[i]
		case "instagram_bio":
			bio = &cards[i]
		}
	}
	if yt.Clicks != 20 || yt.Views == nil || *yt.Views != 1000 || yt.VideosWithViews != 1 {
		t.Errorf("yt = %+v", yt)
	}
	if yt.Revenue.FirstTouch < 39.97 || yt.Revenue.FirstTouch > 39.99 || yt.Revenue.LastTouch < 19.97 || yt.Revenue.LastTouch > 19.99 {
		t.Errorf("revenue = %+v", yt.Revenue)
	}
	if bio.Views != nil {
		t.Error("no views known reads unknown, not 0")
	}
	if len(cards) != 12 {
		t.Errorf("12 channels, got %d", len(cards))
	}
}

func TestLinkWorkspace(t *testing.T) {
	f := newFake(t, func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/keys/self":
			return 200, `{"scopes": ["links:read"]}`
		case "/domains":
			return 200, `{"data": [{"id": "00000000-0000-0000-0000-000000000000", "hostname": "trakyo.link"}]}`
		case "/links":
			return 200, `{"data": [], "has_more": true}`
		}
		return 404, `{}`
	})
	w := keyed(t, "secret-key", f.srv.URL).LinkWorkspace(context.Background())
	if w.State != "ready" || len(w.Scopes) != 1 || !w.HasMore || len(w.Domains) != 1 || w.Domains[0].Hostname != "trakyo.link" {
		t.Fatalf("workspace = %+v", w)
	}
	raw, _ := json.Marshal(w)
	if strings.Contains(string(raw), "secret-key") {
		t.Error("key leaked into workspace")
	}
	if none := keyed(t, "", f.srv.URL).LinkWorkspace(context.Background()); none.State != "not_configured" {
		t.Errorf("none = %+v", none)
	}
	noScope := keyed(t, "k", newFake(t, fixed(200, `{"scopes": ["metrics:read"]}`)).srv.URL).LinkWorkspace(context.Background())
	if noScope.State != "partial" || len(noScope.Messages) != 1 || !strings.Contains(noScope.Messages[0], "links:read") {
		t.Errorf("noScope = %+v", noScope)
	}
	rejected := keyed(t, "k", newFake(t, fixed(401, `{}`)).srv.URL).LinkWorkspace(context.Background())
	if rejected.State != "error" || rejected.Messages[0] != "Trakyo rejected the API key. Reconnect in Integrations." {
		t.Errorf("rejected = %+v", rejected)
	}
}

func TestCreateRefusedWhileWritesDisabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	f := newFake(t, fixed(201, `{"id": "ci_test", "name": "IG Reel", "source": "Instagram", "type": "custom"}`))
	c := keyed(t, "k", f.srv.URL)
	if _, err := c.CreateContent(context.Background(), ContentInput{Source: "Instagram", Name: "IG Reel"}); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Errorf("content err = %v", err)
	}
	if _, err := c.CreateLink(context.Background(), LinkInput{URL: "https://example.com", ContentItemID: "ci_test"}); !errors.Is(err, guard.ErrWritesDisabled) {
		t.Errorf("link err = %v", err)
	}
	if len(f.calls()) != 0 {
		t.Fatalf("a refused write must not leave: %+v", f.calls())
	}
	refused := guard.Refused()
	if len(refused) != 2 || refused[0].Action != "trakyo.create_content" || refused[1].Action != "trakyo.create_link" {
		t.Errorf("refusals = %+v", refused)
	}
}

func TestCreateValidatesBeforeSpendingARequest(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := newFake(t, fixed(201, `{}`))
	c := keyed(t, "k", f.srv.URL)
	if _, err := c.CreateContent(context.Background(), ContentInput{Source: "YouTube", Name: "test"}); err == nil {
		t.Error("youtube is not a custom source")
	}
	if _, err := c.CreateContent(context.Background(), ContentInput{Source: "12345", Name: "x"}); err == nil {
		t.Error("numeric source rejected")
	}
	if _, err := c.CreateLink(context.Background(), LinkInput{URL: "javascript:alert(1)", ContentItemID: "ci_test"}); err == nil {
		t.Error("javascript: url rejected")
	}
	if _, err := c.CreateLink(context.Background(), LinkInput{URL: "https://u:p@example.com", ContentItemID: "ci_test"}); err == nil {
		t.Error("credentials in url rejected")
	}
	if _, err := c.CreateLink(context.Background(), LinkInput{URL: "https://example.com", ContentItemID: "raw-id"}); err == nil {
		t.Error("content id must be ci_")
	}
	if len(f.calls()) != 0 {
		t.Fatalf("validation spent a request: %+v", f.calls())
	}
}

func TestCreateWhenWritesEnabled(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := newFake(t, func(r *http.Request) (int, string) {
		if r.URL.Path == "/content" {
			return 201, `{"id": "ci_test", "name": "IG Reel", "source": "Instagram", "type": "custom"}`
		}
		return 403, `{"error": {"code": "insufficient_scope"}}`
	})
	c := keyed(t, "k", f.srv.URL)
	item, err := c.CreateContent(context.Background(), ContentInput{Source: " Instagram ", Name: "IG Reel"})
	if err != nil || item.ID != "ci_test" {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	_, err = c.CreateLink(context.Background(), LinkInput{URL: "https://example.com", ContentItemID: item.ID})
	var op *OperationError
	if !errors.As(err, &op) || op.Status != 403 || !strings.Contains(op.Message, "links:write") {
		t.Fatalf("link err = %#v", err)
	}
	calls := f.calls()
	if len(calls) != 2 || calls[0].method != "POST" || calls[0].path != "/content" || calls[1].path != "/links" {
		t.Fatalf("calls = %+v", calls)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(calls[0].body), &sent)
	if sent["source"] != "Instagram" {
		t.Errorf("source is trimmed before sending: %v", sent)
	}
	_ = json.Unmarshal([]byte(calls[1].body), &sent)
	if sent["append_tracking_param"] != true {
		t.Errorf("append_tracking_param defaults to true: %v", sent)
	}
}

func TestCreateUnconfirmedResultIsUncertain(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	c := keyed(t, "k", newFake(t, fixed(201, `{"weird": true}`)).srv.URL)
	_, err := c.CreateContent(context.Background(), ContentInput{Source: "Instagram", Name: "IG Reel"})
	var op *OperationError
	if !errors.As(err, &op) || !op.Uncertain || !strings.Contains(op.Message, "Check Trakyo") {
		t.Fatalf("err = %#v", err)
	}
}

// Ported from FounderOS v1 lib/funnel-analytics.ts (2026-09-24): name the real
// failure (HTTP status, transport, or an unrecognised payload), never one
// catch-all that hides which.
func TestAnalyticsMessageNamesTheFailure(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("Trakyo HTTP 401"), "Trakyo HTTP 401"},
		{errors.New("Invalid content payload"), "Trakyo answered with a payload this OS does not recognise."},
		{errors.New("Invalid funnel payload"), "Trakyo answered with a payload this OS does not recognise."},
		{errors.New("Pagination stalled"), "Trakyo answered with a payload this OS does not recognise."},
		{&url.Error{Op: "Get", URL: "https://x", Err: errors.New("dial tcp: no route to host")}, "Trakyo API unreachable from this machine (network or timeout). Try refreshing."},
		{context.DeadlineExceeded, "Trakyo API unreachable from this machine (network or timeout). Try refreshing."},
		{errors.New("something else"), "Trakyo data unavailable or invalid. Try refreshing."},
	}
	for _, c := range cases {
		if got := *analyticsMessage(c.err); got != c.want {
			t.Errorf("%v: got %q, want %q", c.err, got, c.want)
		}
	}
}
