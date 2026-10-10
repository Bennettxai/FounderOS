package devicepush

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore keeps the latest push per device in founderos_device_pushes
// (migration 162), so readings survive a bridge restart.
type PgStore struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgStore(pool *pgxpool.Pool, workspaceID string) *PgStore {
	return &PgStore{pool: pool, workspaceID: workspaceID}
}

func (s *PgStore) Save(ctx context.Context, r Received) error {
	raw, err := json.Marshal(r.Payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO founderos_device_pushes (device, workspace_id, payload, received_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (device) DO UPDATE SET payload = EXCLUDED.payload, received_at = EXCLUDED.received_at`,
		r.Payload.Device, s.workspaceID, raw, r.ReceivedAt)
	return err
}

func (s *PgStore) Latest(ctx context.Context) ([]Received, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload, received_at FROM founderos_device_pushes ORDER BY device`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Received{}
	for rows.Next() {
		var raw []byte
		var r Received
		if err := rows.Scan(&raw, &r.ReceivedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &r.Payload); err != nil {
			return nil, fmt.Errorf("device push row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
