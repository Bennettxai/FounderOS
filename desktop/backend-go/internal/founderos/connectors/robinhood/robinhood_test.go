package robinhood

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var now = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

func snap(account, label, at string, value, bp float64) Snapshot {
	return Snapshot{
		CapturedAt: at, AccountID: account, AccountLabel: label,
		AccountValueUSD: value, BuyingPowerUSD: bp, CashUSD: bp, Source: "markets-agent",
	}
}

func TestMeta(t *testing.T) {
	if Meta != (connectors.Meta{ID: "robinhood", Name: "Robinhood", Kind: connectors.KindPayments}) {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfiguredBeforeTheAgentPushes(t *testing.T) {
	c := New(connectors.Resolver{})
	s := c.Status(context.Background())
	if s.State != connectors.StateNotConfigured {
		t.Fatalf("state = %s", s.State)
	}
	if !strings.Contains(s.Detail, "POST /api/trading/snapshot") {
		t.Errorf("detail = %q", s.Detail)
	}
}

// StatusFor (the trading payload's status) sums every account; the board's
// connector Status passes only the richest, as FounderOS v1 does.
func TestStatusForSumsAccountsAndReportsFreshestPush(t *testing.T) {
	s := StatusFor([]Snapshot{
		snap("ind", "Individual", "2026-09-29T09:00:00Z", 1000.1, 10),
		snap("agentic", "Agentic", "2026-09-29T11:48:00Z", 250.52, 20.25),
	}, now)
	if s.State != connectors.StateConnected {
		t.Fatalf("state = %s", s.State)
	}
	if s.Detail != "2 accounts · updated 12 min ago" {
		t.Errorf("detail = %q", s.Detail)
	}
	if s.Meta["accountValueUsd"] != 1250.62 || s.Meta["buyingPowerUsd"] != 30.25 {
		t.Errorf("meta = %v", s.Meta)
	}
}

func TestStatusForOneAccountNamesIt(t *testing.T) {
	s := StatusFor([]Snapshot{snap("agentic", "Agentic sleeve", "2026-09-29T09:00:00Z", 1, 1)}, now)
	if s.Detail != "Agentic sleeve · updated 3 hours ago" {
		t.Errorf("detail = %q", s.Detail)
	}
}

func TestAgo(t *testing.T) {
	cases := map[string]string{
		"2026-09-29T12:00:20Z": "just now",
		"2026-09-29T11:59:00Z": "1 min ago",
		"2026-09-29T11:00:00Z": "1 hour ago",
		"2026-09-28T12:00:00Z": "1 day ago",
		"2026-09-26T12:00:00Z": "3 days ago",
		"2026-09-29T12:30:00Z": "just now", // clock skew never goes negative
	}
	for at, want := range cases {
		if got := ago(at, now); got != want {
			t.Errorf("ago(%s) = %q, want %q", at, got, want)
		}
	}
}

type brokenStore struct{ *MemStore }

func (brokenStore) LatestSnapshots(context.Context) ([]Snapshot, error) {
	return nil, errors.New("connection refused")
}

func TestStatusErrorWhenTheStoreIsUnreadable(t *testing.T) {
	c := New(connectors.Resolver{})
	c.Store = brokenStore{NewMemStore()}
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.Contains(s.Detail, "connection refused") {
		t.Fatalf("status = %+v", s)
	}
}

func TestMemStoreLatestHistoryAndPositions(t *testing.T) {
	st := NewMemStore()
	ctx := context.Background()
	must(t, st.PushSnapshot(snap("ind", "Individual", "2026-09-28T10:00:00Z", 900, 1), []Position{
		{CapturedAt: "2026-09-28T10:00:00Z", AccountID: "ind", Symbol: "OLD", Quantity: 1, MarketValueUSD: 5},
	}))
	must(t, st.PushSnapshot(snap("ind", "Individual", "2026-09-29T10:00:00Z", 1000, 1), []Position{
		{CapturedAt: "2026-09-29T10:00:00Z", AccountID: "ind", Symbol: "AAPL", Quantity: 2, MarketValueUSD: 400},
		{CapturedAt: "2026-09-29T10:00:00Z", AccountID: "ind", Symbol: "MSFT", Quantity: 1, MarketValueUSD: 450},
	}))
	must(t, st.PushSnapshot(snap("agentic", "Agentic", "2026-09-29T11:00:00Z", 200, 1), []Position{
		{CapturedAt: "2026-09-29T11:00:00Z", AccountID: "agentic", Symbol: "NVDA", Quantity: 1, MarketValueUSD: 100},
	}))
	// the same (account, capturedAt) replaces, like INSERT OR REPLACE
	must(t, st.PushSnapshot(snap("agentic", "Agentic", "2026-09-29T11:00:00Z", 210, 1), nil))

	latest, _ := st.LatestSnapshots(ctx)
	if len(latest) != 2 || latest[0].AccountID != "ind" || latest[0].AccountValueUSD != 1000 || latest[1].AccountValueUSD != 210 {
		t.Fatalf("latest = %+v", latest)
	}
	hist, _ := st.History(ctx, "ind", 0)
	if len(hist) != 2 || hist[0].CapturedAt != "2026-09-28T10:00:00Z" {
		t.Fatalf("history = %+v", hist)
	}
	if h, _ := st.History(ctx, "ind", 1); len(h) != 1 {
		t.Fatalf("history limit ignored: %+v", h)
	}
	pos, _ := st.Positions(ctx, "")
	if len(pos) != 3 || pos[0].Symbol != "MSFT" || pos[1].Symbol != "AAPL" || pos[2].Symbol != "NVDA" {
		t.Fatalf("positions = %+v", pos)
	}
	if p, _ := st.Positions(ctx, "agentic"); len(p) != 1 || p[0].Symbol != "NVDA" {
		t.Fatalf("agentic positions = %+v", p)
	}
}

func TestPushSnapshotValidates(t *testing.T) {
	st := NewMemStore()
	if err := st.PushSnapshot(Snapshot{AccountID: "x"}, nil); err == nil {
		t.Error("a snapshot with no capturedAt/label/source must be refused")
	}
	good := snap("a", "A", "2026-09-29T11:00:00Z", 1, 1)
	if err := st.PushSnapshot(good, []Position{{AccountID: "a", CapturedAt: "2026-09-29T11:00:00Z"}}); err == nil {
		t.Error("a position with no symbol must be refused")
	}
	many := make([]Position, 201)
	for i := range many {
		many[i] = Position{AccountID: "a", CapturedAt: "2026-09-29T11:00:00Z", Symbol: "S"}
	}
	if err := st.PushSnapshot(good, many); err == nil {
		t.Error("more than 200 positions must be refused")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// FounderOS v1's Connections board passes trading.latestSnapshot(): only the
// richest account. Its tile reads "Individual · updated …", not "2 accounts".
func TestStatusReportsOnlyTheRichestAccountLikeTheBoard(t *testing.T) {
	st := NewMemStore()
	ctx := context.Background()
	must(t, st.PushSnapshot(snap("agt", "Agentic", "2026-09-29T11:00:00Z", 100, 10), nil))
	must(t, st.PushSnapshot(snap("ind", "Individual", "2026-09-29T10:00:00Z", 2500, 50), nil))
	c := New(connectors.Resolver{})
	c.Store = st
	c.now = func() time.Time { return now }
	s := c.Status(ctx)
	if !strings.HasPrefix(s.Detail, "Individual · ") {
		t.Fatalf("detail = %q, want the richest account only", s.Detail)
	}
	if v, _ := s.Meta["accountValueUsd"].(float64); v != 2500 {
		t.Fatalf("meta = %+v", s.Meta)
	}
}
