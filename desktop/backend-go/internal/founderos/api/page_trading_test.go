package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/trading"
)

type tradingFakeStore struct{ err error }

func (f tradingFakeStore) LatestSnapshots(context.Context) ([]trading.Snapshot, error) {
	return []trading.Snapshot{{CapturedAt: "2026-09-18T14:30:00.000Z", AccountID: "agentic", AccountLabel: "Agentic", AccountValueUSD: 600, CashUSD: 600, Source: "seed"}}, f.err
}
func (f tradingFakeStore) History(context.Context, string, int) ([]trading.Snapshot, error) {
	return nil, f.err
}
func (f tradingFakeStore) Positions(context.Context) ([]trading.Position, error) { return nil, f.err }
func (f tradingFakeStore) Activity(context.Context, int) ([]trading.Activity, error) {
	return nil, f.err
}
func (f tradingFakeStore) LatestAnalysis(context.Context, string) (*trading.Analysis, error) {
	return nil, f.err
}
func (f tradingFakeStore) OpenOrders(context.Context) ([]trading.Order, error) { return nil, f.err }
func (f tradingFakeStore) Limits(context.Context) (*trading.Limits, string, error) {
	return nil, "", f.err
}

func tradingTestRouter(src tradingSources) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tradingMount(r.Group("/api/founderos"), src)
	return r
}

func tradingGet(r http.Handler, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/trading", nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTradingPageServesThePayload(t *testing.T) {
	walletCalls := 0
	src := tradingSources{
		store: func() trading.Store { return tradingFakeStore{} },
		wallet: func(context.Context) trading.WalletRead {
			walletCalls++
			return trading.WalletRead{State: "error", Detail: "Solana RPC did not return a balance: timeout"}
		},
		now: func() time.Time { return time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC) },
	}
	w := tradingGet(tradingTestRouter(src), "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["phantom"] != nil || walletCalls != 1 {
		t.Fatalf("an unreachable wallet is null, not zero: %v", body["phantom"])
	}
	if ps := body["phantomStatus"].(map[string]any); ps["state"] != "error" {
		t.Fatalf("phantomStatus %v", ps)
	}
	view := body["view"].(map[string]any)
	if view["freshness"].(map[string]any)["state"] != "seeded" {
		t.Fatalf("freshness %v", view["freshness"])
	}
	for _, k := range []string{"accounts", "history", "snapshot", "positions", "activity", "analysis", "openOrders", "status", "source", "limits", "at"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("payload missing %q: %v", k, body)
		}
	}
}

func TestTradingPageSurfacesAnUnreadableStore(t *testing.T) {
	src := tradingSources{
		store:  func() trading.Store { return tradingFakeStore{err: errors.New("connection refused")} },
		wallet: func(context.Context) trading.WalletRead { return trading.WalletRead{} },
		now:    time.Now,
	}
	w := tradingGet(tradingTestRouter(src), "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("an unreachable store must not read as empty: %d %s", w.Code, w.Body)
	}
}

func TestTradingPageIsRegisteredBehindTheSession(t *testing.T) {
	// No pool: the route exists, needs the session, and says the store is down.
	t.Setenv("PHANTOM_WALLET_ADDRESS", "") // never reach an RPC from a test
	r := router(t, &Deps{})
	if w := tradingGet(r, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	w := tradingGet(r, "session=ok")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no database: %d %s", w.Code, w.Body)
	}
}
