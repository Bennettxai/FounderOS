package funnel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgSeed reads the seeded funnel (db.funnel.journeys) from the venture
// workspaces' founderos_funnel_contacts and founderos_funnel_touches. The live
// lanes are composed at read time and never stored (table map #31).
type PgSeed struct {
	Pool       *pgxpool.Pool
	Workspaces map[string]string // slug → workspace UUID
}

// Journeys returns seeded journeys newest first, touches in seq order;
// venture "" means both.
func (s *PgSeed) Journeys(ctx context.Context, venture string) ([]Journey, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("funnel seed: founderos Postgres is not wired")
	}
	ids := []string{}
	for _, slug := range []string{"vantage", "launchpad-cohort"} {
		id := s.Workspaces[slug]
		if id == "" {
			return nil, fmt.Errorf("funnel seed: workspace %q is not resolved", slug)
		}
		ids = append(ids, id)
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, venture, status, product, amount_usd, relationship, likelihood,
			email, phone, person, company, role, linkedin, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')
		FROM founderos_funnel_contacts
		WHERE workspace_id = ANY($1::uuid[]) AND ($2 = '' OR venture = $2)
		ORDER BY created_at DESC, id`, ids, venture)
	if err != nil {
		return nil, fmt.Errorf("funnel seed: %w", err)
	}
	out := []Journey{}
	index := map[string]int{}
	for rows.Next() {
		var j Journey
		if err := rows.Scan(&j.ID, &j.Name, &j.Venture, &j.Status, &j.Product, &j.AmountUSD, &j.Relationship, &j.Likelihood,
			&j.Email, &j.Phone, &j.Person, &j.Company, &j.Role, &j.LinkedIn, &j.CreatedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("funnel seed: %w", err)
		}
		j.Touches = []Touch{}
		index[j.ID] = len(out)
		out = append(out, j)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("funnel seed: %w", err)
	}
	if len(out) == 0 {
		return out, nil
	}
	contactIDs := make([]string, 0, len(out))
	for _, j := range out {
		contactIDs = append(contactIDs, j.ID)
	}
	trows, err := s.Pool.Query(ctx, `SELECT id, contact_id, seq, stage, channel, label, source, to_char(at, 'YYYY-MM-DD')
		FROM founderos_funnel_touches WHERE contact_id = ANY($1) ORDER BY contact_id, seq`, contactIDs)
	if err != nil {
		return nil, fmt.Errorf("funnel seed touches: %w", err)
	}
	defer trows.Close()
	for trows.Next() {
		var t Touch
		if err := trows.Scan(&t.ID, &t.ContactID, &t.Seq, &t.Stage, &t.Channel, &t.Label, &t.Source, &t.At); err != nil {
			return nil, fmt.Errorf("funnel seed touches: %w", err)
		}
		if i, ok := index[t.ContactID]; ok {
			out[i].Touches = append(out[i].Touches, t)
		}
	}
	return out, trows.Err()
}

// PgArchive reads the retired CRM history (FounderOS v1 db.funnel.archivedJourneys):
// the 101 Attio + 87 GoHighLevel journeys restored 2026-09-26, each stored
// whole as schema-validated JSON in founderos_funnel_archive (the ETL target of
// FounderOS v1's funnel_archive table). No live CRM is polled.
type PgArchive struct {
	Pool       *pgxpool.Pool
	Workspaces map[string]string // slug → workspace UUID
}

// Journeys returns the archived journeys ordered by id; venture "" means both.
func (a *PgArchive) Journeys(ctx context.Context, venture string) ([]Journey, error) {
	if a == nil || a.Pool == nil {
		return nil, errors.New("funnel archive: founderos Postgres is not wired")
	}
	ids := []string{}
	for _, slug := range []string{"vantage", "launchpad-cohort"} {
		id := a.Workspaces[slug]
		if id == "" {
			return nil, fmt.Errorf("funnel archive: workspace %q is not resolved", slug)
		}
		ids = append(ids, id)
	}
	rows, err := a.Pool.Query(ctx, `SELECT journey FROM founderos_funnel_archive
		WHERE workspace_id = ANY($1::uuid[]) AND ($2 = '' OR venture = $2) ORDER BY id`, ids, venture)
	if err != nil {
		return nil, fmt.Errorf("funnel archive: %w", err)
	}
	defer rows.Close()
	out := []Journey{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("funnel archive: %w", err)
		}
		var j Journey
		if err := json.Unmarshal(raw, &j); err != nil {
			return nil, fmt.Errorf("funnel archive: %w", err)
		}
		if j.Touches == nil {
			j.Touches = []Touch{}
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
