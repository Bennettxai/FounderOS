package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	bd "github.com/rhl/businessos-backend/internal/founderos/connectors/branddeals"
)

type brandDealsFake struct {
	res   bd.Result
	calls int
}

func (f *brandDealsFake) FetchDeals(context.Context) bd.Result {
	f.calls++
	return f.res
}

func brandDealsServe(t *testing.T, fake *brandDealsFake, cookie bool) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	prevSrc, prevNow := brandDealsSource, brandDealsNow
	brandDealsSource = func(*Deps) brandDealsFetcher { return fake }
	brandDealsNow = func() time.Time { return time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { brandDealsSource, brandDealsNow = prevSrc, prevNow })

	r := router(t, &Deps{Board: connectors.NewRegistry()})
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/brand-deals", nil)
	if cookie {
		req.Header.Set("Cookie", "session=ok")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}

func brandDealsPtr[T any](v T) *T { return &v }

func TestBrandDealsNeedsASession(t *testing.T) {
	w, _ := brandDealsServe(t, &brandDealsFake{}, false)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
}

func TestBrandDealsLiveServesTheConnectorResultAndItsView(t *testing.T) {
	ms := int64(1758100000000)
	fake := &brandDealsFake{res: bd.Result{Mode: bd.ModeLive, Detail: "Brand Deals Hub · 2 deals", SyncedAt: &ms, Deals: []bd.Deal{
		{ID: "a", Brand: "Notion", Status: "Negotiating", DealValueUSD: brandDealsPtr(6000.0), FollowUpDate: brandDealsPtr("2026-09-18"),
			NotionURL: "https://www.notion.so/a", LastEdited: "2026-09-17T08:00:00.000Z"},
		{ID: "b", Brand: "Riverside", Status: "Paid", AmountAgreedUSD: brandDealsPtr(2500.0), NotionURL: "https://www.notion.so/b", LastEdited: "2026-09-10T08:00:00.000Z"},
	}}}
	w, body := brandDealsServe(t, fake, true)
	if w.Code != 200 || fake.calls != 1 {
		t.Fatalf("got %d (%d calls) %s", w.Code, fake.calls, w.Body)
	}
	// the source route's payload, flat
	if body["mode"] != "live" || body["detail"] != "Brand Deals Hub · 2 deals" || body["syncedAt"] != float64(ms) {
		t.Fatalf("payload = %v", body)
	}
	if deals, _ := body["deals"].([]any); len(deals) != 2 || deals[0].(map[string]any)["notionUrl"] != "https://www.notion.so/a" {
		t.Fatalf("deals = %v", body["deals"])
	}
	view, ok := body["view"].(map[string]any)
	if !ok {
		t.Fatalf("live board must carry its view: %v", body)
	}
	vol := view["volume"].(map[string]any)
	if vol["openUsd"] != 6000.0 || vol["paidUsd"] != 2500.0 {
		t.Fatalf("volume = %v", vol)
	}
	if view["hubUrl"] != "https://www.notion.so/a" || view["needsYou"] != 1.0 || len(view["activity"].([]any)) != 30 {
		t.Fatalf("view = %v", view)
	}
}

func TestBrandDealsSeededIsMarkedAndHasNoHub(t *testing.T) {
	fake := &brandDealsFake{res: bd.Result{Mode: bd.ModeSeeded, Detail: "Showing examples.", Deals: bd.SeededDeals()}}
	w, body := brandDealsServe(t, fake, true)
	if w.Code != 200 || body["mode"] != "seeded" || body["syncedAt"] != nil {
		t.Fatalf("got %d %v", w.Code, body)
	}
	view := body["view"].(map[string]any)
	if view["hubUrl"] != nil {
		t.Fatalf("examples must not deep-link a hub: %v", view["hubUrl"])
	}
}

func TestBrandDealsErrorIsUnknownNotZero(t *testing.T) {
	fake := &brandDealsFake{res: bd.Result{Mode: bd.ModeError, Detail: "Key set but the fetch failed: 401.", Deals: []bd.Deal{}}}
	w, body := brandDealsServe(t, fake, true)
	if w.Code != 200 || body["mode"] != "error" || body["detail"] != "Key set but the fetch failed: 401." {
		t.Fatalf("got %d %v", w.Code, body)
	}
	if v, present := body["view"]; !present || v != nil {
		t.Fatalf("an unreachable Notion has no figures (view must be null, never zeroes): %v", body["view"])
	}
	if deals, ok := body["deals"].([]any); !ok || len(deals) != 0 {
		t.Fatalf("deals must be an empty list, got %v", body["deals"])
	}
}
