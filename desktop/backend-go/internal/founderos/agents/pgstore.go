package agents

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore keeps runs and broadcasts in the founderos_* tables of one
// BusinessOS workspace (agent runs belong to FounderOS by the table map).
type PgStore struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgStore(pool *pgxpool.Pool, workspaceID string) *PgStore {
	return &PgStore{pool: pool, workspaceID: workspaceID}
}

func (s *PgStore) InsertRun(ctx context.Context, r Run) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO founderos_agent_runs (id, workspace_id, agent_id, started_at, finished_at, ok, summary, model, tokens_in, tokens_out, cost_usd)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		r.ID, s.workspaceID, r.AgentID, r.StartedAt, r.FinishedAt, r.OK, r.Summary, r.Model, r.TokensIn, r.TokensOut, r.CostUSD)
	return err
}

func (s *PgStore) Recent(ctx context.Context, agentID string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, started_at, finished_at, ok, summary, model, tokens_in, tokens_out, cost_usd
		FROM founderos_agent_runs
		WHERE workspace_id = $1 AND ($2 = '' OR agent_id = $2)
		ORDER BY started_at DESC
		LIMIT $3`, s.workspaceID, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.AgentID, &r.StartedAt, &r.FinishedAt, &r.OK, &r.Summary, &r.Model, &r.TokensIn, &r.TokensOut, &r.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *PgStore) InsertBroadcast(ctx context.Context, b Broadcast) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO founderos_broadcasts (id, workspace_id, message, created_at) VALUES ($1, $2, $3, $4)`,
		b.ID, s.workspaceID, b.Message, b.CreatedAt); err != nil {
		return err
	}
	for _, r := range b.Replies {
		if _, err := tx.Exec(ctx, `INSERT INTO founderos_broadcast_replies (id, workspace_id, broadcast_id, agent_id, ok, reply, finished_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, b.ID+":"+r.AgentID, s.workspaceID, b.ID, r.AgentID, r.OK, r.Reply, r.FinishedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// PgCronSource reads founderos_agent_crons with each cron's latest run from
// founderos_cron_runs, and records new cron runs there.
type PgCronSource struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgCronSource(pool *pgxpool.Pool, workspaceID string) *PgCronSource {
	return &PgCronSource{pool: pool, workspaceID: workspaceID}
}

func (s *PgCronSource) Crons(ctx context.Context) ([]Cron, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.agent_id, c.schedule, c.description, c.enabled,
		       (SELECT max(r.started_at) FROM founderos_cron_runs r WHERE r.cron_id = c.id AND r.workspace_id = c.workspace_id)
		FROM founderos_agent_crons c
		WHERE c.workspace_id = $1
		ORDER BY c.id`, s.workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cron
	for rows.Next() {
		var c Cron
		if err := rows.Scan(&c.ID, &c.AgentID, &c.Schedule, &c.Description, &c.Enabled, &c.LastRunAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PgCronSource) RecordCronRun(ctx context.Context, c Cron, r Run) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO founderos_cron_runs (id, workspace_id, cron_id, agent_id, started_at, finished_at, ok, summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		r.ID, s.workspaceID, c.ID, r.AgentID, r.StartedAt, r.FinishedAt, r.OK, r.Summary)
	return err
}
