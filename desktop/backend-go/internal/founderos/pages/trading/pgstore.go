package trading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Workspace is where the trading tables live (docs/founderos/table-map.md:
// trading is personal, all FIN). The compat /api/trading/* routes write here.
const Workspace = "personal"

// PgStore reads founderos_trading_* in the personal workspace.
type PgStore struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	ws   string
}

func NewPgStore(pool *pgxpool.Pool) *PgStore { return &PgStore{pool: pool} }

func (s *PgStore) workspace(ctx context.Context) (string, error) {
	if s.pool == nil {
		return "", errors.New("trading store: no database")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != "" {
		return s.ws, nil
	}
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, Workspace).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("trading store: no %q workspace (run founderos-bootstrap)", Workspace)
	}
	if err != nil {
		return "", err
	}
	s.ws = id
	return id, nil
}

// iso is the instant as FounderOS v1 stores it: UTC with milliseconds, so string
// comparison orders it the same as time.
func iso(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

const snapCols = `account_id, account_label, captured_at, account_value_usd, buying_power_usd, cash_usd, day_pnl_usd, total_pnl_usd, source`

func scanSnapshots(rows pgx.Rows) ([]Snapshot, error) {
	defer rows.Close()
	out := []Snapshot{}
	for rows.Next() {
		var s Snapshot
		var at time.Time
		if err := rows.Scan(&s.AccountID, &s.AccountLabel, &at, &s.AccountValueUSD, &s.BuyingPowerUSD, &s.CashUSD, &s.DayPnlUSD, &s.TotalPnlUSD, &s.Source); err != nil {
			return nil, err
		}
		s.CapturedAt = iso(at)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (s *PgStore) LatestSnapshots(ctx context.Context) ([]Snapshot, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON (account_id) `+snapCols+` FROM founderos_trading_snapshots
		WHERE workspace_id = $1 ORDER BY account_id, captured_at DESC`, ws)
	if err != nil {
		return nil, err
	}
	out, err := scanSnapshots(rows)
	sort.SliceStable(out, func(i, j int) bool { return out[i].AccountValueUSD > out[j].AccountValueUSD })
	return out, err
}

func (s *PgStore) History(ctx context.Context, accountID string, limit int) ([]Snapshot, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = historyLimit
	}
	rows, err := s.pool.Query(ctx, `SELECT * FROM (SELECT `+snapCols+` FROM founderos_trading_snapshots
		WHERE workspace_id = $1 AND account_id = $2 ORDER BY captured_at DESC LIMIT $3) h ORDER BY captured_at ASC`, ws, accountID, limit)
	if err != nil {
		return nil, err
	}
	return scanSnapshots(rows)
}

func (s *PgStore) Positions(ctx context.Context) ([]Position, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT p.account_id, p.captured_at, p.symbol, p.quantity, p.avg_cost_usd, p.market_value_usd, p.unrealized_pnl_usd
		FROM founderos_trading_positions p
		JOIN (SELECT account_id, max(captured_at) AS at FROM founderos_trading_snapshots WHERE workspace_id = $1 GROUP BY account_id) l
		  ON l.account_id = p.account_id AND l.at = p.captured_at
		WHERE p.workspace_id = $1 ORDER BY p.market_value_usd DESC`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Position{}
	for rows.Next() {
		var p Position
		var at time.Time
		if err := rows.Scan(&p.AccountID, &at, &p.Symbol, &p.Quantity, &p.AvgCostUSD, &p.MarketValueUSD, &p.UnrealizedPnlUSD); err != nil {
			return nil, err
		}
		p.CapturedAt = iso(at)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *PgStore) Activity(ctx context.Context, limit int) ([]Activity, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, at, account_id, agent, action, symbol, quantity, price_usd, rationale, status
		FROM founderos_trading_activity WHERE workspace_id = $1 ORDER BY at DESC LIMIT $2`, ws, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Activity{}
	for rows.Next() {
		var a Activity
		var at time.Time
		if err := rows.Scan(&a.ID, &at, &a.AccountID, &a.Agent, &a.Action, &a.Symbol, &a.Quantity, &a.PriceUSD, &a.Rationale, &a.Status); err != nil {
			return nil, err
		}
		a.At = iso(at)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *PgStore) LatestAnalysis(ctx context.Context, accountID string) (*Analysis, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	var a Analysis
	var at time.Time
	var raw []byte
	err = s.pool.QueryRow(ctx, `SELECT id, at, account_id, agent, examined, signals, notes, rows FROM founderos_trading_analysis
		WHERE workspace_id = $1 AND account_id = $2 ORDER BY at DESC LIMIT 1`, ws, accountID).
		Scan(&a.ID, &at, &a.AccountID, &a.Agent, &a.Examined, &a.Signals, &a.Notes, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.At = iso(at)
	if err := json.Unmarshal(raw, &a.Rows); err != nil {
		return nil, fmt.Errorf("trading analysis %s rows: %w", a.ID, err)
	}
	if a.Rows == nil {
		a.Rows = []AnalysisRow{}
	}
	return &a, nil
}

func (s *PgStore) OpenOrders(ctx context.Context) ([]Order, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, account_id, symbol, side, type, state, quantity, filled_quantity,
		dollar_amount_usd, limit_price_usd, placed_agent, created_at
		FROM founderos_trading_orders WHERE workspace_id = $1 ORDER BY created_at DESC`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var o Order
		var at time.Time
		if err := rows.Scan(&o.ID, &o.AccountID, &o.Symbol, &o.Side, &o.Type, &o.State, &o.Quantity, &o.FilledQuantity,
			&o.DollarAmountUSD, &o.LimitPriceUSD, &o.PlacedAgent, &at); err != nil {
			return nil, err
		}
		o.CreatedAt = iso(at)
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PgStore) Limits(ctx context.Context) (*Limits, string, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, "", err
	}
	var l Limits
	var at time.Time
	var conc, perDay int
	err = s.pool.QueryRow(ctx, `SELECT max_notional_per_trade_usd, max_position_pct_of_sleeve, max_risk_pct_per_trade,
		max_concurrent_positions, max_trades_per_day, min_sleeve_value_usd, max_deployed_capital_usd, autopilot, updated_at
		FROM founderos_trading_limits WHERE workspace_id = $1`, ws).
		Scan(&l.MaxNotionalPerTradeUSD, &l.MaxPositionPctOfSleeve, &l.MaxRiskPctPerTrade, &conc, &perDay,
			&l.MinSleeveValueUSD, &l.MaxDeployedCapitalUSD, &l.Autopilot, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	l.MaxConcurrentPositions, l.MaxTradesPerDay = float64(conc), float64(perDay)
	return &l, iso(at), nil
}
