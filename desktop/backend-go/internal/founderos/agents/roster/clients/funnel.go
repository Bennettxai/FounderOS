package clients

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FunnelContact is the slice of a funnel journey the Client Roster counts
// (founderos_funnel_contacts). AmountUSD nil is unknown.
type FunnelContact struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Venture   string   `json:"venture"`
	Status    string   `json:"status"`
	AmountUSD *float64 `json:"amountUsd"`
}

// FunnelStore reads the funnel contacts, newest first (db.funnel.journeys).
type FunnelStore interface {
	Contacts(ctx context.Context) ([]FunnelContact, error)
}

// Ventures are the workspaces the funnel lives in (table map: per-row from
// venture).
var Ventures = []string{"vantage", "launchpad-cohort"}

// PgFunnel reads founderos_funnel_contacts across both venture workspaces.
type PgFunnel struct {
	Pool       *pgxpool.Pool
	Workspaces map[string]string // slug → workspace UUID
}

func (f *PgFunnel) Contacts(ctx context.Context) ([]FunnelContact, error) {
	if f == nil || f.Pool == nil {
		return nil, errors.New("funnel: founderos Postgres is not wired")
	}
	ids := make([]string, 0, len(Ventures))
	for _, slug := range Ventures {
		id := f.Workspaces[slug]
		if id == "" {
			return nil, fmt.Errorf("funnel: workspace %q is not resolved", slug)
		}
		ids = append(ids, id)
	}
	rows, err := f.Pool.Query(ctx, `SELECT id, name, venture, status, amount_usd
		FROM founderos_funnel_contacts
		WHERE workspace_id = ANY($1::uuid[])
		ORDER BY created_at DESC, id`, ids)
	if err != nil {
		return nil, fmt.Errorf("funnel: %w", err)
	}
	defer rows.Close()
	out := []FunnelContact{}
	for rows.Next() {
		var c FunnelContact
		if err := rows.Scan(&c.ID, &c.Name, &c.Venture, &c.Status, &c.AmountUSD); err != nil {
			return nil, fmt.Errorf("funnel: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("funnel: %w", err)
	}
	return out, nil
}
