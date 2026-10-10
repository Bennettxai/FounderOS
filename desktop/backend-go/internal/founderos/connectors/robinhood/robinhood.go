// Package robinhood ports FounderOS v1's lib/connectors/robinhood.ts: the
// status of the agentic Robinhood feed.
//
// There is no key and no network here. The Robinhood Trading MCP is
// agent-facing (OAuth, driven by an AI agent), so the OS is fed indirectly:
// the Markets agent pushes account snapshots and positions, and this
// connector reports whether that feed is live. Persistence sits behind the
// small Store interface so Postgres can be plugged in later; MemStore is the
// in-process default.
package robinhood

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "robinhood", Name: "Robinhood", Kind: connectors.KindPayments}

// Snapshot mirrors TradingAccountSnapshotSchema (lib/schemas.ts).
type Snapshot struct {
	CapturedAt      string  `json:"capturedAt"`
	AccountID       string  `json:"accountId"`
	AccountLabel    string  `json:"accountLabel"`
	AccountValueUSD float64 `json:"accountValueUsd"`
	BuyingPowerUSD  float64 `json:"buyingPowerUsd"`
	CashUSD         float64 `json:"cashUsd"`
	DayPnlUSD       float64 `json:"dayPnlUsd"`
	TotalPnlUSD     float64 `json:"totalPnlUsd"`
	Source          string  `json:"source"`
}

// Position mirrors TradingPositionSchema.
type Position struct {
	CapturedAt       string  `json:"capturedAt"`
	AccountID        string  `json:"accountId"`
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	AvgCostUSD       float64 `json:"avgCostUsd"`
	MarketValueUSD   float64 `json:"marketValueUsd"`
	UnrealizedPnlUSD float64 `json:"unrealizedPnlUsd"`
}

// MaxPositions caps one push, as POST /api/trading/snapshot does.
const MaxPositions = 200

// Store is where pushed snapshots live. Reads must return an error when the
// store is unreachable, never an empty list.
type Store interface {
	// LatestSnapshots is the newest snapshot of every account, richest first.
	LatestSnapshots(ctx context.Context) ([]Snapshot, error)
	// History is one account's snapshots oldest first; limit <= 0 means 500.
	History(ctx context.Context, accountID string, limit int) ([]Snapshot, error)
	// Positions come from each account's latest snapshot, largest first; an
	// empty accountID means every account.
	Positions(ctx context.Context, accountID string) ([]Position, error)
}

type Connector struct {
	Store Store
	now   func() time.Time
}

// New returns a connector over an in-memory store. There are no credentials
// to resolve: the feed is pushed to the bridge, not pulled.
func New(_ connectors.Resolver) *Connector {
	return &Connector{Store: NewMemStore(), now: time.Now}
}

func (c *Connector) LatestSnapshots(ctx context.Context) ([]Snapshot, error) {
	return c.Store.LatestSnapshots(ctx)
}

func (c *Connector) History(ctx context.Context, accountID string, limit int) ([]Snapshot, error) {
	return c.Store.History(ctx, accountID, limit)
}

func (c *Connector) Positions(ctx context.Context, accountID string) ([]Position, error) {
	return c.Store.Positions(ctx, accountID)
}

func (c *Connector) Status(ctx context.Context) connectors.Status {
	fed, err := c.Store.LatestSnapshots(ctx)
	if err != nil {
		return connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind, State: connectors.StateError,
			Detail: "Robinhood snapshot store unreadable: " + err.Error()}
	}
	// The Connections board passes FounderOS v1's trading.latestSnapshot(): the
	// richest account only (the trading payload keeps StatusFor(all)).
	if len(fed) > 1 {
		top := fed[0]
		for _, a := range fed[1:] {
			if a.AccountValueUSD > top.AccountValueUSD {
				top = a
			}
		}
		fed = []Snapshot{top}
	}
	return StatusFor(fed, c.now())
}

// StatusFor is the status of the latest fed snapshots: not_configured until
// the agent has pushed, connected afterwards with how long ago it was fed.
// Value and buying power are summed across accounts.
func StatusFor(accounts []Snapshot, now time.Time) connectors.Status {
	s := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if len(accounts) == 0 {
		s.State = connectors.StateNotConfigured
		s.Detail = "Robinhood not feeding the OS yet. Authenticate the Robinhood Trading MCP in an agent runtime, then push a snapshot to POST /api/trading/snapshot."
		return s
	}
	var value, bp float64
	freshest := accounts[0]
	for _, a := range accounts {
		value += a.AccountValueUSD
		bp += a.BuyingPowerUSD
		if a.CapturedAt > freshest.CapturedAt {
			freshest = a
		}
	}
	scope := fmt.Sprintf("%d accounts", len(accounts))
	if len(accounts) == 1 {
		scope = accounts[0].AccountLabel
	}
	s.State = connectors.StateConnected
	s.Detail = fmt.Sprintf("%s · updated %s", scope, ago(freshest.CapturedAt, now))
	s.Meta = map[string]any{"accountValueUsd": round2(value), "buyingPowerUsd": round2(bp)}
	return s
}

func round2(v float64) float64 { return math.Floor(v*100+0.5) / 100 }

func jsRound(v float64) int { return int(math.Floor(v + 0.5)) }

func parseISO(iso string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, iso)
}

// ago is the human "N min/hour/day ago" of the TS.
func ago(iso string, now time.Time) string {
	at, err := parseISO(iso)
	if err != nil {
		return "at an unknown time"
	}
	mins := jsRound(now.Sub(at).Minutes())
	if mins < 0 {
		mins = 0
	}
	if mins < 1 {
		return "just now"
	}
	if mins < 60 {
		return fmt.Sprintf("%d min ago", mins)
	}
	hrs := jsRound(float64(mins) / 60)
	if hrs < 24 {
		return fmt.Sprintf("%d hour%s ago", hrs, plural(hrs))
	}
	days := jsRound(float64(hrs) / 24)
	return fmt.Sprintf("%d day%s ago", days, plural(days))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ---- in-memory store --------------------------------------------------------

type snapKey struct{ account, at string }

type posKey struct{ account, at, symbol string }

// MemStore keeps pushed snapshots in process. Same keys as the SQLite tables:
// a snapshot per (account, capturedAt), a position per (account, capturedAt,
// symbol), and a re-push of either replaces it.
type MemStore struct {
	mu        sync.RWMutex
	snaps     map[snapKey]Snapshot
	positions map[posKey]Position
}

func NewMemStore() *MemStore {
	return &MemStore{snaps: map[snapKey]Snapshot{}, positions: map[posKey]Position{}}
}

func validSnapshot(s Snapshot) error {
	if s.CapturedAt == "" || s.AccountID == "" || s.AccountLabel == "" || s.Source == "" {
		return errors.New("snapshot needs capturedAt, accountId, accountLabel and source")
	}
	for _, v := range []float64{s.AccountValueUSD, s.BuyingPowerUSD, s.CashUSD, s.DayPnlUSD, s.TotalPnlUSD} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("snapshot money fields must be finite")
		}
	}
	return nil
}

func validPosition(p Position) error {
	if p.CapturedAt == "" || p.AccountID == "" || p.Symbol == "" {
		return errors.New("position needs capturedAt, accountId and symbol")
	}
	for _, v := range []float64{p.Quantity, p.AvgCostUSD, p.MarketValueUSD, p.UnrealizedPnlUSD} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("position fields must be finite")
		}
	}
	return nil
}

// PushSnapshot records what the Markets agent pushed. Nothing is written
// unless the whole push validates.
func (m *MemStore) PushSnapshot(s Snapshot, positions []Position) error {
	if err := validSnapshot(s); err != nil {
		return err
	}
	if len(positions) > MaxPositions {
		return fmt.Errorf("at most %d positions per push", MaxPositions)
	}
	for _, p := range positions {
		if err := validPosition(p); err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snaps[snapKey{s.AccountID, s.CapturedAt}] = s
	for _, p := range positions {
		m.positions[posKey{p.AccountID, p.CapturedAt, p.Symbol}] = p
	}
	return nil
}

func (m *MemStore) latestLocked() map[string]Snapshot {
	latest := map[string]Snapshot{}
	for _, s := range m.snaps {
		if cur, ok := latest[s.AccountID]; !ok || s.CapturedAt > cur.CapturedAt {
			latest[s.AccountID] = s
		}
	}
	return latest
}

func (m *MemStore) LatestSnapshots(context.Context) ([]Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Snapshot, 0)
	for _, s := range m.latestLocked() {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountValueUSD > out[j].AccountValueUSD })
	return out, nil
}

func (m *MemStore) History(_ context.Context, accountID string, limit int) ([]Snapshot, error) {
	if limit <= 0 {
		limit = 500
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Snapshot, 0)
	for _, s := range m.snaps {
		if s.AccountID == accountID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt < out[j].CapturedAt })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemStore) Positions(_ context.Context, accountID string) ([]Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	latest := m.latestLocked()
	out := make([]Position, 0)
	for _, p := range m.positions {
		s, ok := latest[p.AccountID]
		if !ok || s.CapturedAt != p.CapturedAt || (accountID != "" && p.AccountID != accountID) {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MarketValueUSD > out[j].MarketValueUSD })
	return out, nil
}
