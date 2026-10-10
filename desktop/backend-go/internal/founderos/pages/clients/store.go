package clients

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the clientWork repo. Reads fail loudly when the table is
// unreachable; an unreachable store is never an empty board.
type Store interface {
	All(ctx context.Context) ([]Work, error)
	Get(ctx context.Context, id string) (*Work, error)
	Add(ctx context.Context, w Work) error
	Claim(ctx context.Context, id string) (bool, error)
	Finish(ctx context.Context, id string, workspaceID *string) error
}

// Detail lines written by Finish (lib/db.ts clientWork.finish).
const (
	DetailLaunched  = "Agent launched. Review the workspace for results; completion is not yet synced."
	DetailUncertain = "Launch could not be confirmed. Check Superset before creating another request to avoid duplicate work."
)

// WorkspaceSlug is client_work's home (table map #35).
const WorkspaceSlug = "founderos"

// PgStore is founderos_client_work scoped to the founderos workspace.
type PgStore struct {
	Pool *pgxpool.Pool
	mu   sync.Mutex
	ws   string
}

func NewPgStore(pool *pgxpool.Pool) *PgStore { return &PgStore{Pool: pool} }

func (s *PgStore) workspace(ctx context.Context) (string, error) {
	if s == nil || s.Pool == nil {
		return "", errors.New("client work store: no database")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != "" {
		return s.ws, nil
	}
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, WorkspaceSlug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("client work store: workspace %q is missing (run cmd/founderos-bootstrap)", WorkspaceSlug)
	}
	if err != nil {
		return "", err
	}
	s.ws = id
	return id, nil
}

const cols = `id, client_id, brief, status, created_at, superset_workspace_id, detail`

func scanWork(row pgx.Row) (Work, error) {
	var w Work
	var at time.Time
	var status string
	if err := row.Scan(&w.ID, &w.ClientID, &w.Brief, &status, &at, &w.WorkspaceID, &w.Detail); err != nil {
		return w, err
	}
	w.Status = Status(status)
	w.CreatedAt = at.UTC().Format("2006-01-02T15:04:05.000Z")
	return w, nil
}

func (s *PgStore) All(ctx context.Context) ([]Work, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM founderos_client_work WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT 100`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Work{}
	for rows.Next() {
		w, err := scanWork(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *PgStore) Get(ctx context.Context, id string) (*Work, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return nil, err
	}
	w, err := scanWork(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM founderos_client_work WHERE workspace_id = $1 AND id = $2`, ws, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *PgStore) Add(ctx context.Context, w Work) error {
	ws, err := s.workspace(ctx)
	if err != nil {
		return err
	}
	at, ok := ParseInstant(w.CreatedAt)
	if !ok || !w.Status.Valid() || w.ID == "" || w.ClientID == "" {
		return errors.New("client work: invalid row")
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO founderos_client_work (id, workspace_id, client_id, brief, status, created_at, superset_workspace_id, detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, w.ID, ws, w.ClientID, w.Brief, string(w.Status), at, w.WorkspaceID, w.Detail)
	return err
}

// Claim moves a saved request to launching; false when it was already claimed.
func (s *PgStore) Claim(ctx context.Context, id string) (bool, error) {
	ws, err := s.workspace(ctx)
	if err != nil {
		return false, err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE founderos_client_work SET status = 'launching' WHERE workspace_id = $1 AND id = $2 AND status = 'saved'`, ws, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Finish settles a launching request: launched with its workspace, or
// needs_attention when the launch could not be confirmed.
func (s *PgStore) Finish(ctx context.Context, id string, workspaceID *string) error {
	ws, err := s.workspace(ctx)
	if err != nil {
		return err
	}
	status, detail := StatusNeedsAttention, DetailUncertain
	if workspaceID != nil {
		status, detail = StatusLaunched, DetailLaunched
	}
	_, err = s.Pool.Exec(ctx, `UPDATE founderos_client_work SET status = $1, superset_workspace_id = $2, detail = $3
		WHERE workspace_id = $4 AND id = $5 AND status = 'launching'`, string(status), workspaceID, detail, ws, id)
	return err
}
