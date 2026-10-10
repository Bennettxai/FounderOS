package trading

import (
	"math"
	"reflect"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/trading-view.test.ts, trading-volume.test.ts
// and the agentSummary half of trading-chart.test.ts.

var now = time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)

func snap(over func(*Snapshot)) Snapshot {
	s := Snapshot{CapturedAt: "2026-09-18T14:30:00.000Z", AccountID: "agentic", AccountLabel: "Agentic",
		AccountValueUSD: 592.28, BuyingPowerUSD: 556, CashUSD: 556, Source: "robinhood"}
	if over != nil {
		over(&s)
	}
	return s
}

func pos(symbol string, mv float64) Position {
	return Position{CapturedAt: "2026-09-18T14:30:00.000Z", AccountID: "agentic", Symbol: symbol, Quantity: 1, AvgCostUSD: mv, MarketValueUSD: mv}
}

func trade(at, symbol, status string) Activity {
	return Activity{ID: "t-" + at, At: at, AccountID: "agentic", Agent: "Markets Agent", Action: "buy", Symbol: symbol, Quantity: 1, PriceUSD: 100, Status: status}
}

func TestFreshness(t *testing.T) {
	if f := FreshnessOf(nil, now); f.State != "none" || f.Label != "no feed yet" {
		t.Fatalf("none: %+v", f)
	}
	if f := FreshnessOf([]Snapshot{snap(nil)}, now); f != (Freshness{"live", "synced 30m ago"}) {
		t.Fatalf("live: %+v", f)
	}
	old := snap(func(s *Snapshot) { s.AccountID = "individual"; s.CapturedAt = "2026-08-21T19:22:00.000Z" })
	if f := FreshnessOf([]Snapshot{old}, now); f != (Freshness{"stale", "synced 27d ago"}) {
		t.Fatalf("stale: %+v", f)
	}
	if f := FreshnessOf([]Snapshot{old, snap(nil)}, now); f.State != "live" {
		t.Fatalf("freshest decides: %+v", f)
	}
	if f := FreshnessOf([]Snapshot{snap(func(s *Snapshot) { s.Source = "seed" })}, now); f.State != "seeded" || f.Label != "seeded · example rows" {
		t.Fatalf("seeded: %+v", f)
	}
}

func TestSyncAge(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-18T14:59:40.000Z": "synced just now",
		"2026-09-18T14:30:00.000Z": "synced 30m ago",
		"2026-09-18T09:00:00.000Z": "synced 6h ago",
		"2026-09-11T15:00:00.000Z": "synced 7d ago",
	} {
		if got := SyncAge(in, now); got != want {
			t.Errorf("SyncAge(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestPositionSizesKeepsEveryBucket(t *testing.T) {
	got := PositionSizes([]Position{pos("BSX", 36), pos("QQQ", 180), pos("APLD", 982), pos("NVDA", 4000)})
	want := []Bucket{{"<$50", 1}, {"$50-250", 1}, {"$250-1k", 1}, {"$1k+", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	for _, b := range PositionSizes(nil) {
		if b.Count != 0 {
			t.Fatalf("empty book: %v", b)
		}
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestVolumeHeadlineChipsAndMeters(t *testing.T) {
	accounts := []Snapshot{
		snap(func(s *Snapshot) { s.AccountValueUSD = 600; s.DayPnlUSD = 4.5 }),
		snap(func(s *Snapshot) {
			s.AccountID, s.AccountLabel, s.AccountValueUSD, s.DayPnlUSD = "individual", "Individual", 1200, -1.25
		}),
	}
	price, value := 100.0, 200.0
	w := &Wallet{Address: "68SHabcdefghBST2", SOL: 2, USDPerSOL: &price, USDValue: &value}
	v := VolumeOf(accounts, []Position{pos("QQQ", 90), pos("SPY", 10)}, w)
	if v.Headline == nil || *v.Headline != 1800 {
		t.Fatalf("headline %v", v.Headline)
	}
	wantChips := []Chip{{"ok", "+$3.25 today"}, {"accent", "$100 in 2 positions"}}
	if !reflect.DeepEqual(v.Chips, wantChips) {
		t.Fatalf("chips %v", v.Chips)
	}
	if v.Caption != "brokerage, 2 accounts · $2,000.00 with the wallet" {
		t.Fatalf("caption %q", v.Caption)
	}
	labels := []string{"Agentic · agent may trade", "Individual · read-only to agents", "Phantom · 2 SOL · 68SH…BST2"}
	fracs := []float64{0.3, 0.6, 0.1}
	displays := []string{"$600.00", "$1,200.00", "$200.00"}
	hues := []string{HueAgentic, HueIndividual, HuePhantom}
	for i, m := range v.Meters {
		if m.Label != labels[i] || m.Frac == nil || !near(*m.Frac, fracs[i]) || m.Display != displays[i] || m.Hue != hues[i] {
			t.Fatalf("meter %d = %+v", i, m)
		}
	}
	if v.Foot != "only the agentic sleeve can be traded by an agent · $100.00 / SOL" {
		t.Fatalf("foot %q", v.Foot)
	}
}

func TestVolumeLosingDayAndUnreachableWalletIsAnEmptyMeter(t *testing.T) {
	r := VolumeOf([]Snapshot{snap(func(s *Snapshot) { s.DayPnlUSD = -3 })}, nil, nil)
	if r.Chips[0] != (Chip{"err", "-$3.00 today"}) || r.Chips[1] != (Chip{"accent", "all cash"}) {
		t.Fatalf("chips %v", r.Chips)
	}
	last := r.Meters[len(r.Meters)-1]
	// v1 lib/trading-view.ts: no wallet is an empty (0) meter reading "--", not an unknown track.
	if last.Label != "Phantom · wallet not reachable" || last.Frac == nil || *last.Frac != 0 || last.Display != "--" {
		t.Fatalf("wallet meter %+v", last)
	}
	if r.Foot != "only the agentic sleeve can be traded by an agent · wallet read from a public RPC" {
		t.Fatalf("foot %q", r.Foot)
	}
	if r.Caption != "brokerage, 1 account" {
		t.Fatalf("caption %q", r.Caption)
	}
}

func TestVolumeWalletWithNoPriceReadsUnavailable(t *testing.T) {
	r := VolumeOf([]Snapshot{snap(nil)}, nil, &Wallet{Address: "68SHabcdefghBST2", SOL: 2})
	last := r.Meters[len(r.Meters)-1]
	if last.Frac == nil || *last.Frac != 0 || last.Display != "price unavailable" {
		t.Fatalf("wallet meter %+v", last)
	}
}

func TestVolumeNothingFedIsUnknownNeverFabricated(t *testing.T) {
	e := VolumeOf(nil, nil, nil)
	if e.Headline != nil {
		t.Fatalf("headline must be unknown, got %v", *e.Headline)
	}
	// the only meter is v1's empty wallet track: a 0 fill reading "--", never a value
	if len(e.Meters) != 1 || e.Meters[0].Frac == nil || *e.Meters[0].Frac != 0 || e.Meters[0].Display != "--" {
		t.Fatalf("meters %+v", e.Meters)
	}
}

func TestShortAddress(t *testing.T) {
	if ShortAddress("68SHabcdefghBST2") != "68SH…BST2" || ShortAddress("short") != "short" {
		t.Fatal(ShortAddress("68SHabcdefghBST2"))
	}
}

func TestAgentSummary(t *testing.T) {
	funded := snap(func(s *Snapshot) { s.AccountValueUSD, s.CashUSD = 600, 600 })
	s := AgentSummaryOf(&funded, nil, nil)
	if s.HasActed || s.TradeCount != 0 || s.IdleCashUSD != 600 || s.DeployedUSD != 0 || s.LastTradeAt != nil {
		t.Fatalf("idle: %+v", s)
	}
	trades := []Activity{
		trade("2026-08-13T12:00:00.000Z", "SPY", "filled"),
		trade("2026-08-13T12:30:00.000Z", "QQQ", "cancelled"),
		trade("2026-08-13T12:45:00.000Z", "IWM", "filled"),
	}
	s = AgentSummaryOf(&funded, nil, trades)
	if !s.HasActed || s.TradeCount != 2 || *s.LastTradeAt != "2026-08-13T12:45:00.000Z" || s.LastTrade.Symbol != "IWM" {
		t.Fatalf("filled only: %+v", s)
	}
	p := Position{AccountID: "agentic", Symbol: "SPY", Quantity: 1, AvgCostUSD: 200, MarketValueUSD: 220.5, UnrealizedPnlUSD: 20.5}
	funded.CashUSD = 380
	s = AgentSummaryOf(&funded, []Position{p}, nil)
	if s.DeployedUSD != 220.5 || s.IdleCashUSD != 380 || s.UnrealizedPnlUSD != 20.5 {
		t.Fatalf("split: %+v", s)
	}
	s = AgentSummaryOf(nil, nil, nil)
	if s.HasActed || s.IdleCashUSD != 0 || s.TradeCount != 0 {
		t.Fatalf("never fed: %+v", s)
	}
}

func TestLimitsClampAndDefaults(t *testing.T) {
	v := LimitsViewOf(nil, "")
	if v.Source != "default" || v.Limits.MaxNotionalPerTradeUSD != 150 || v.Limits.MinSleeveValueUSD != 480 || v.Limits.Autopilot || len(v.Clamped) != 0 {
		t.Fatalf("defaults: %+v", v)
	}
	stored := DefaultLimits
	stored.MaxNotionalPerTradeUSD, stored.MinSleeveValueUSD, stored.Autopilot = 999, 100, true
	v = LimitsViewOf(&stored, "2026-09-20T10:00:00.000Z")
	if v.Source != "stored" || v.Limits.MaxNotionalPerTradeUSD != 300 || v.Limits.MinSleeveValueUSD != 450 || !v.Limits.Autopilot {
		t.Fatalf("clamped: %+v", v)
	}
	if !reflect.DeepEqual(v.Clamped, []string{"maxNotionalPerTradeUsd", "minSleeveValueUsd"}) || v.UpdatedAt == nil {
		t.Fatalf("clamped names: %+v", v)
	}
}
