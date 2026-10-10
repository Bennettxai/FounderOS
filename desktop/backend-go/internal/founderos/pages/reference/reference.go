// Package reference serves /os/reference: the reference model's operating
// domains (FounderOS v1 app/reference/page.tsx + db.domains + DomainSchema),
// read from founderos_domains in the FounderOS workspace (migration 165) and
// validated on the way out.
package reference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/roadmap"
)

// ErrNoWorkspace means the workspace row is missing (bootstrap not run),
// which is not the same thing as a workspace with no domains.
var ErrNoWorkspace = errors.New("workspace not found: run founderos-bootstrap")

// Domain mirrors v1 DomainSchema.
type Domain struct {
	ID     string   `json:"id"`
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Color  string   `json:"color"`
	Items  []string `json:"items"`
}

// Validate applies DomainSchema: id, title and color non-empty.
func (d Domain) Validate() error {
	if d.ID == "" || d.Title == "" || d.Color == "" {
		return fmt.Errorf("domain %q: id, title and color are required", d.ID)
	}
	return nil
}

// Load reads the workspace's domains by number (v1 ORDER BY number).
func Load(ctx context.Context, pool *pgxpool.Pool, workspaceSlug string) ([]Domain, error) {
	var ws string
	err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, workspaceSlug).Scan(&ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w (%s)", ErrNoWorkspace, workspaceSlug)
	}
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `SELECT id, number, title, color, items FROM founderos_domains
		WHERE workspace_id = $1 ORDER BY number, id`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Domain{}
	for rows.Next() {
		var d Domain
		var raw []byte
		if err := rows.Scan(&d.ID, &d.Number, &d.Title, &d.Color, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &d.Items); err != nil {
			return nil, fmt.Errorf("domain %q items: %w", d.ID, err)
		}
		if d.Items == nil {
			d.Items = []string{}
		}
		d.Title = roadmap.RetireGBrain(d.Title)
		for i := range d.Items {
			d.Items[i] = roadmap.RetireGBrain(d.Items[i])
		}
		if err := d.Validate(); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
