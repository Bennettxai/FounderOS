package trading

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeStore struct {
	accounts []Snapshot
	history  map[string][]Snapshot
	pos      []Position
	act      []Activity
	an       *Analysis
	orders   []Order
	limits   *Limits
	err      error
	asked    string
}

func (f *fakeStore) LatestSnapshots(context.Context) ([]Snapshot, error) { return f.accounts, f.err }
func (f *fakeStore) History(_ context.Context, id string, _ int) ([]Snapshot, error) {
	return f.history[id], f.err
}
func (f *fakeStore) Positions(context.Context) ([]Position, error)     { return f.pos, f.err }
func (f *fakeStore) Activity(context.Context, int) ([]Activity, error) { return f.act, f.err }
func (f *fakeStore) OpenOrders(context.Context) ([]Order, error)       { return f.orders, f.err }
func (f *fakeStore) Limits(context.Context) (*Limits, string, error)   { return f.limits, "", f.err }
func (f *fakeStore) LatestAnalysis(_ context.Context, id string) (*Analysis, error) {
	f.asked = id
	return f.an, f.err
}

func fixtureStore() *fakeStore {
	ag := snap(func(s *Snapshot) { s.AccountValueUSD = 600 })
	in := snap(func(s *Snapshot) { s.AccountID, s.AccountLabel, s.AccountValueUSD = "individual", "Individual", 1200 })
	return &fakeStore{
		accounts: []Snapshot{in, ag},
		history:  map[string][]Snapshot{"agentic": {ag}, "individual": {in}},
		pos:      []Position{pos("QQQ", 90)},
		act:      []Activity{trade("2026-09-18T14:00:00.000Z", "QQQ", "filled")},
		an:       &Analysis{ID: "a1", AccountID: "agentic", Signals: 2, Rows: []AnalysisRow{}},
		orders:   []Order{{ID: "o1", AccountID: "agentic", Symbol: "SPY", Side: "buy", State: "queued", PlacedAgent: "agentic"}},
	}
}

func TestBuildIsTheWholeTradingPayload(t *testing.T) {
	st := fixtureStore()
	price, usd := 100.0, 200.0
	wallet := func(context.Context) WalletRead {
		return WalletRead{Wallet: &Wallet{Address: "68SHabcdefghBST2", SOL: 2, USDPerSOL: &price, USDValue: &usd}, State: "connected"}
	}
	p, err := Build(context.Background(), st, wallet, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Accounts) != 2 || p.Snapshot == nil || p.Snapshot.AccountID != "individual" || *p.Source != "robinhood" {
		t.Fatalf("accounts/snapshot: %+v", p)
	}
	if len(p.History["agentic"]) != 1 || len(p.History["individual"]) != 1 {
		t.Fatalf("history per account: %+v", p.History)
	}
	if st.asked != AgenticID || p.Analysis == nil || len(p.OpenOrders) != 1 || len(p.Activity) != 1 || len(p.Positions) != 1 {
		t.Fatalf("analysis/orders/activity: %+v", p)
	}
	if p.Status.State != "connected" || !strings.Contains(p.Status.Detail, "2 accounts") {
		t.Fatalf("status %+v", p.Status)
	}
	if p.Phantom == nil || p.PhantomStatus.State != "connected" {
		t.Fatalf("phantom %+v", p.PhantomStatus)
	}
	if p.At != "2026-09-18T15:00:00.000Z" || p.View.Freshness.State != "live" || *p.View.Volume.Headline != 1800 {
		t.Fatalf("view %+v at %s", p.View, p.At)
	}
	if p.View.Agent.TradeCount != 1 || p.View.Agent.IdleCashUSD != 556 || len(p.View.Sizes) != 4 {
		t.Fatalf("agent %+v", p.View.Agent)
	}
	if p.Limits.Source != "default" {
		t.Fatalf("limits %+v", p.Limits)
	}
}

func TestBuildNothingFedIsHonest(t *testing.T) {
	wallet := func(context.Context) WalletRead {
		return WalletRead{State: "not_configured", Detail: "No PHANTOM_WALLET_ADDRESS set."}
	}
	p, err := Build(context.Background(), &fakeStore{}, wallet, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Snapshot != nil || p.Source != nil || p.Phantom != nil || p.View.Volume.Headline != nil {
		t.Fatalf("unknown must stay unknown: %+v", p)
	}
	if p.Status.State != "not_configured" || p.View.Freshness.State != "none" {
		t.Fatalf("status %+v", p.Status)
	}
	// JSON arrays, never null, so the board can map them.
	if p.Accounts == nil || p.Positions == nil || p.Activity == nil || p.OpenOrders == nil || p.History == nil {
		t.Fatalf("nil slices: %+v", p)
	}
	if p.PhantomStatus.State != "not_configured" {
		t.Fatalf("phantom %+v", p.PhantomStatus)
	}
}

func TestBuildFailsWhenTheStoreIsUnreadable(t *testing.T) {
	_, err := Build(context.Background(), &fakeStore{err: errors.New("db down")}, func(context.Context) WalletRead { return WalletRead{} }, now)
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("an unreadable store must fail the read, got %v", err)
	}
}
