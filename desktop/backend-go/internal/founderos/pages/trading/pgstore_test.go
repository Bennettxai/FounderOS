package trading

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func tradingThrowawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_trading_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestPgStoreWithoutThePersonalWorkspaceIsAnError(t *testing.T) {
	pool := tradingThrowawayDB(t)
	_, err := NewPgStore(pool).LatestSnapshots(context.Background())
	if err == nil || !strings.Contains(err.Error(), "personal") {
		t.Fatalf("err = %v", err)
	}
}

func TestPgStoreReadsThePersonalWorkspaceOnly(t *testing.T) {
	pool := tradingThrowawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"personal", "founderos"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	snapSQL := `INSERT INTO founderos_trading_snapshots (account_id, captured_at, workspace_id, account_label, account_value_usd,
		buying_power_usd, cash_usd, day_pnl_usd, total_pnl_usd, source) VALUES ($1,$2,$3,$4,$5,0,$6,0,0,$7)`
	exec(snapSQL, "agentic", "2026-09-18T13:00:00Z", ws["personal"], "Agentic", 590.0, 500.0, "robinhood")
	exec(snapSQL, "agentic", "2026-09-18T14:30:00Z", ws["personal"], "Agentic", 600.0, 480.0, "robinhood")
	exec(snapSQL, "individual", "2026-09-18T14:00:00Z", ws["personal"], "Individual", 1200.0, 100.0, "robinhood")
	exec(snapSQL, "other", "2026-09-18T14:00:00Z", ws["founderos"], "Other", 9999.0, 0.0, "robinhood")
	posSQL := `INSERT INTO founderos_trading_positions (account_id, captured_at, symbol, workspace_id, quantity, avg_cost_usd,
		market_value_usd, unrealized_pnl_usd) VALUES ($1,$2,$3,$4,1,1,$5,0)`
	exec(posSQL, "agentic", "2026-09-18T13:00:00Z", "OLD", ws["personal"], 10.0)
	exec(posSQL, "agentic", "2026-09-18T14:30:00Z", "QQQ", ws["personal"], 120.0)
	exec(posSQL, "individual", "2026-09-18T14:00:00Z", "NVDA", ws["personal"], 900.0)
	exec(`INSERT INTO founderos_trading_activity (id, workspace_id, at, account_id, agent, action, symbol, quantity, price_usd, rationale, status)
		VALUES ('t1',$1,'2026-09-18T12:00:00Z','agentic','Markets Agent','buy','QQQ',1,100,'dip','filled'),
		       ('t2',$1,'2026-09-18T13:00:00Z','individual','the operator (manual)','sell','NVDA',1,900,'','rejected')`, ws["personal"])
	exec(`INSERT INTO founderos_trading_analysis (id, workspace_id, at, account_id, agent, examined, signals, notes, rows)
		VALUES ('a-old',$1,'2026-09-17T12:00:00Z','agentic','Markets Agent',3,0,'old','[]'),
		       ('a-new',$1,'2026-09-18T12:00:00Z','agentic','Markets Agent',5,1,'new','[{"ticker":"QQQ","score":null,"verdict":"signal","reason":"dip"}]')`, ws["personal"])
	exec(`INSERT INTO founderos_trading_orders (id, workspace_id, account_id, symbol, side, type, state, quantity, filled_quantity,
		dollar_amount_usd, limit_price_usd, placed_agent, created_at)
		VALUES ('o1',$1,'agentic','SPY','buy','market','queued',0,0,25,NULL,'agentic','2026-09-18T14:00:00Z')`, ws["personal"])

	st := NewPgStore(pool)
	accts, err := st.LatestSnapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(accts) != 2 || accts[0].AccountID != "individual" || accts[1].AccountValueUSD != 600 || accts[1].CapturedAt != "2026-09-18T14:30:00.000Z" {
		t.Fatalf("latest: %+v", accts)
	}
	h, err := st.History(ctx, "agentic", 500)
	if err != nil || len(h) != 2 || h[0].AccountValueUSD != 590 {
		t.Fatalf("history oldest first: %+v %v", h, err)
	}
	if h, _ = st.History(ctx, "agentic", 1); len(h) != 1 || h[0].AccountValueUSD != 600 {
		t.Fatalf("history keeps the newest: %+v", h)
	}
	ps, err := st.Positions(ctx)
	if err != nil || len(ps) != 2 || ps[0].Symbol != "NVDA" || ps[1].Symbol != "QQQ" {
		t.Fatalf("positions from the latest snapshot, largest first: %+v %v", ps, err)
	}
	act, err := st.Activity(ctx, 50)
	if err != nil || len(act) != 2 || act[0].ID != "t2" || act[1].Rationale != "dip" {
		t.Fatalf("activity newest first: %+v %v", act, err)
	}
	an, err := st.LatestAnalysis(ctx, "agentic")
	if err != nil || an == nil || an.ID != "a-new" || len(an.Rows) != 1 || an.Rows[0].Score != nil {
		t.Fatalf("analysis: %+v %v", an, err)
	}
	orders, err := st.OpenOrders(ctx)
	if err != nil || len(orders) != 1 || *orders[0].DollarAmountUSD != 25 || orders[0].LimitPriceUSD != nil {
		t.Fatalf("orders: %+v %v", orders, err)
	}
	l, _, err := st.Limits(ctx)
	if err != nil || l != nil {
		t.Fatalf("no stored limits: %+v %v", l, err)
	}
	exec(`INSERT INTO founderos_trading_limits (workspace_id, max_notional_per_trade_usd, max_position_pct_of_sleeve, max_risk_pct_per_trade,
		max_concurrent_positions, max_trades_per_day, min_sleeve_value_usd, max_deployed_capital_usd, autopilot, updated_at)
		VALUES ($1,100,30,2,4,8,480,500,true,'2026-09-19T10:00:00Z')`, ws["personal"])
	l, at, err := st.Limits(ctx)
	if err != nil || l == nil || l.MaxNotionalPerTradeUSD != 100 || !l.Autopilot || at != "2026-09-19T10:00:00.000Z" {
		t.Fatalf("stored limits: %+v %s %v", l, at, err)
	}
	if an, err := st.LatestAnalysis(ctx, "individual"); err != nil || an != nil {
		t.Fatalf("no analysis is nil, not an error: %+v %v", an, err)
	}
}
