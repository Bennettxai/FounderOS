package content

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Workspace homes (docs/founderos/table-map.md): the agent roster and its runs
// live in founderos; lead magnets in personal.
const (
	AgentsWorkspace      = "founderos"
	LeadMagnetsWorkspace = "personal"
)

// ErrNotFound: no lead magnet with that id in the workspace.
var ErrNotFound = errors.New("lead magnet not found")

// Agents reads the Marketing/Growth pillar's founderos_agents rows.
func Agents(ctx context.Context, pool *pgxpool.Pool, ws string) ([]Agent, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, role, status, tier, description, model, tools::text, parent_id, department_id
		FROM founderos_agents WHERE workspace_id = $1 AND department_id = $2`, ws, DeptID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Agent, error) {
		var a Agent
		var tools string
		if err := r.Scan(&a.ID, &a.Name, &a.Role, &a.Status, &a.Tier, &a.Description, &a.Model, &tools, &a.ParentID, &a.DepartmentID); err != nil {
			return a, err
		}
		a.Tools = []string{}
		_ = json.Unmarshal([]byte(tools), &a.Tools)
		return a, nil
	})
}

// RunsSince reads founderos_agent_runs for the given agents started at or after since.
func RunsSince(ctx context.Context, pool *pgxpool.Pool, ws string, since time.Time, ids []string) ([]Run, error) {
	rows, err := pool.Query(ctx, `
		SELECT agent_id, started_at, ok FROM founderos_agent_runs
		WHERE workspace_id = $1 AND started_at >= $2 AND agent_id = ANY($3)
		ORDER BY started_at DESC`, ws, since, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Run, error) {
		var run Run
		var at time.Time
		err := r.Scan(&run.AgentID, &at, &run.OK)
		run.StartedAt = at.UTC().Format(time.RFC3339Nano)
		return run, err
	})
}

const lmCols = `id, name, offer, url, status, captures, destination, source, launched_at, notes, origin`

func scanLM(r pgx.Row) (LeadMagnet, error) {
	var m LeadMagnet
	err := r.Scan(&m.ID, &m.Name, &m.Offer, &m.URL, &m.Status, &m.Captures, &m.Destination, &m.Source, &m.LaunchedAt, &m.Notes, &m.Origin)
	return m, err
}

// LeadMagnets is leadMagnets.all(): newest launch first, then by name.
func LeadMagnets(ctx context.Context, pool *pgxpool.Pool, ws string) ([]LeadMagnet, error) {
	rows, err := pool.Query(ctx, `SELECT `+lmCols+` FROM founderos_lead_magnets WHERE workspace_id = $1 ORDER BY launched_at DESC, name`, ws)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (LeadMagnet, error) { return scanLM(r) })
}

// LeadMagnetByID returns ErrNotFound when the id is not in the workspace.
func LeadMagnetByID(ctx context.Context, pool *pgxpool.Pool, ws, id string) (LeadMagnet, error) {
	m, err := scanLM(pool.QueryRow(ctx, `SELECT `+lmCols+` FROM founderos_lead_magnets WHERE workspace_id = $1 AND id = $2`, ws, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

// TakenIDs is every lead magnet id (the primary key is global, so all
// workspaces count when picking a new one).
func TakenIDs(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM founderos_lead_magnets`)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	taken := map[string]bool{}
	for _, id := range ids {
		taken[id] = true
	}
	return taken, err
}

// InsertLeadMagnet creates a row (the create path picks a free id first).
func InsertLeadMagnet(ctx context.Context, pool *pgxpool.Pool, ws string, m LeadMagnet) error {
	_, err := pool.Exec(ctx, `INSERT INTO founderos_lead_magnets (`+lmCols+`, workspace_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		m.ID, m.Name, m.Offer, m.URL, m.Status, m.Captures, m.Destination, m.Source, m.LaunchedAt, m.Notes, m.Origin, ws)
	return err
}

// UpdateLeadMagnet rewrites the editable columns of an existing row.
func UpdateLeadMagnet(ctx context.Context, pool *pgxpool.Pool, ws string, m LeadMagnet) error {
	tag, err := pool.Exec(ctx, `UPDATE founderos_lead_magnets SET name=$3, offer=$4, url=$5, status=$6, captures=$7, destination=$8, source=$9, launched_at=$10, notes=$11
		WHERE workspace_id = $1 AND id = $2`, ws, m.ID, m.Name, m.Offer, m.URL, m.Status, m.Captures, m.Destination, m.Source, m.LaunchedAt, m.Notes)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteLeadMagnet returns ErrNotFound when nothing was removed.
func DeleteLeadMagnet(ctx context.Context, pool *pgxpool.Pool, ws, id string) error {
	tag, err := pool.Exec(ctx, `DELETE FROM founderos_lead_magnets WHERE workspace_id = $1 AND id = $2`, ws, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
