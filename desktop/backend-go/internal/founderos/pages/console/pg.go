package console

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReadRoster reads the agent roster and runs of the FounderOS workspace (the
// table map puts agents and agent_runs there): the live count, the 40 newest
// runs and every run started in the last 15 days (one spare day so a local
// day window never loses its first morning to UTC).
func ReadRoster(ctx context.Context, pool *pgxpool.Pool, now time.Time) (*Roster, []Run, []Run, error) {
	if pool == nil {
		return nil, nil, nil, errors.New("no database")
	}
	var ws string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, errors.New("founderos workspace missing (bootstrap not run)")
		}
		return nil, nil, nil, err
	}
	var r Roster
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status = 'active'), count(*) FROM founderos_agents WHERE workspace_id = $1`, ws).Scan(&r.Active, &r.Total); err != nil {
		return nil, nil, nil, fmt.Errorf("agents: %w", err)
	}
	recent, err := runs(ctx, pool, `SELECT id, agent_id, started_at, finished_at, ok, summary FROM founderos_agent_runs WHERE workspace_id = $1 ORDER BY started_at DESC, id DESC LIMIT 40`, ws)
	if err != nil {
		return nil, nil, nil, err
	}
	window, err := runs(ctx, pool, `SELECT id, agent_id, started_at, finished_at, ok, summary FROM founderos_agent_runs WHERE workspace_id = $1 AND started_at >= $2 ORDER BY started_at DESC, id DESC`, ws, now.Add(-15*day))
	if err != nil {
		return nil, nil, nil, err
	}
	return &r, recent, window, nil
}

func runs(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) ([]Run, error) {
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("agent runs: %w", err)
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.AgentID, &r.StartedAt, &r.FinishedAt, &r.OK, &r.Summary); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
