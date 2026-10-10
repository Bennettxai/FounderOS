// Package personas serves /os/personas: the operator variants of the OS
// (FounderOS v1 db.personas + PersonaSchema), read from founderos_personas in
// the FounderOS workspace and validated on the way out.
package personas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoWorkspace means the workspace row is missing (bootstrap not run),
// which is not the same thing as a workspace with no personas.
var ErrNoWorkspace = errors.New("workspace not found: run founderos-bootstrap")

type Pillar struct {
	Name   string   `json:"name"`
	Focus  string   `json:"focus"`
	Agents []string `json:"agents"`
}

// Persona mirrors FounderOS v1 PersonaSchema.
type Persona struct {
	ID            string   `json:"id"`
	Order         int      `json:"order"`
	Name          string   `json:"name"`
	Archetype     string   `json:"archetype"`
	Tagline       string   `json:"tagline"`
	Summary       string   `json:"summary"`
	Accent        string   `json:"accent"`
	NorthStar     string   `json:"northStar"`
	Pillars       []Pillar `json:"pillars"`
	Connectors    []string `json:"connectors"`
	Metrics       []string `json:"metrics"`
	BrainUse      string   `json:"brainUse"`
	SignaturePlay string   `json:"signaturePlay"`
}

// Validate applies PersonaSchema: every text field non-empty, at least one
// pillar (each with a name, a focus and an agent), connector and metric.
func (p Persona) Validate() error {
	for field, v := range map[string]string{
		"id": p.ID, "name": p.Name, "archetype": p.Archetype, "tagline": p.Tagline, "summary": p.Summary,
		"accent": p.Accent, "northStar": p.NorthStar, "brainUse": p.BrainUse, "signaturePlay": p.SignaturePlay,
	} {
		if v == "" {
			return fmt.Errorf("persona %q: %s is empty", p.ID, field)
		}
	}
	if len(p.Pillars) == 0 {
		return fmt.Errorf("persona %q: no pillars", p.ID)
	}
	for _, pl := range p.Pillars {
		if pl.Name == "" || pl.Focus == "" || len(pl.Agents) == 0 {
			return fmt.Errorf("persona %q: pillar %q needs a name, a focus and an agent", p.ID, pl.Name)
		}
	}
	if len(p.Connectors) == 0 || len(p.Metrics) == 0 {
		return fmt.Errorf("persona %q: needs connectors and metrics", p.ID)
	}
	return nil
}

// Load reads the workspace's personas in order.
func Load(ctx context.Context, pool *pgxpool.Pool, workspaceSlug string) ([]Persona, error) {
	var ws string
	err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, workspaceSlug).Scan(&ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w (%s)", ErrNoWorkspace, workspaceSlug)
	}
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `SELECT id, ord, name, archetype, tagline, summary, accent, north_star,
		pillars, connectors, metrics, brain_use, signature_play
		FROM founderos_personas WHERE workspace_id = $1 ORDER BY ord, id`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Persona{}
	for rows.Next() {
		var p Persona
		var pillars, connectors, metrics []byte
		if err := rows.Scan(&p.ID, &p.Order, &p.Name, &p.Archetype, &p.Tagline, &p.Summary, &p.Accent, &p.NorthStar,
			&pillars, &connectors, &metrics, &p.BrainUse, &p.SignaturePlay); err != nil {
			return nil, err
		}
		for _, j := range []struct {
			raw []byte
			dst any
		}{{pillars, &p.Pillars}, {connectors, &p.Connectors}, {metrics, &p.Metrics}} {
			if err := json.Unmarshal(j.raw, j.dst); err != nil {
				return nil, fmt.Errorf("persona %q: %w", p.ID, err)
			}
		}
		if err := p.Validate(); err != nil {
			return nil, err
		}
		out = append(out, Rebrand(p))
	}
	return out, rows.Err()
}

// retired renames v1's G-Brain to the Optimal Engine's "Brain" in the seeded
// persona copy (G-Brain is gone in v2).
var retired = strings.NewReplacer("G-Brain", "Brain")

// Rebrand returns p with every retired product name replaced, in prose and in
// the pillar, connector and metric lists alike.
func Rebrand(p Persona) Persona {
	for _, f := range []*string{&p.Name, &p.Archetype, &p.Tagline, &p.Summary, &p.NorthStar, &p.BrainUse, &p.SignaturePlay} {
		*f = retired.Replace(*f)
	}
	list := func(in []string) []string {
		out := make([]string, len(in))
		for i, v := range in {
			out[i] = retired.Replace(v)
		}
		return out
	}
	pillars := make([]Pillar, len(p.Pillars))
	for i, pl := range p.Pillars {
		pillars[i] = Pillar{Name: retired.Replace(pl.Name), Focus: retired.Replace(pl.Focus), Agents: list(pl.Agents)}
	}
	p.Pillars, p.Connectors, p.Metrics = pillars, list(p.Connectors), list(p.Metrics)
	return p
}
