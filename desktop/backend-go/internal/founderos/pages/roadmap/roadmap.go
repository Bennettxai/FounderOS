// Package roadmap serves /os/roadmap: the build plan as one board (FounderOS
// v1 app/roadmap/page.tsx + lib/roadmap.ts + components/RoadmapBoard.tsx),
// read from founderos_phases, founderos_roadmap_items and founderos_departments
// in the FounderOS workspace (migration 165), validated on the way out.
package roadmap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNoWorkspace means the workspace row is missing (bootstrap not run),
	// which is not the same thing as a workspace with an empty roadmap.
	ErrNoWorkspace = errors.New("workspace not found: run founderos-bootstrap")
	ErrNotFound    = errors.New("no such roadmap item")
	ErrBadStatus   = errors.New("id and a roadmap status are required")
)

// Statuses is v1 RoadmapStatusSchema.
var Statuses = []string{"done", "now", "next", "later"}

var quarterRe = regexp.MustCompile(`^\d{4}-Q[1-4]$`)

// Phase mirrors v1 PhaseSchema.
type Phase struct {
	ID     string   `json:"id"`
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Items  []string `json:"items"`
}

// Item mirrors v1 RoadmapItemSchema.
type Item struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Quarter      string  `json:"quarter"`
	Status       string  `json:"status"`
	DepartmentID *string `json:"departmentId"`
	Description  string  `json:"description"`
	PhaseID      *string `json:"phaseId"`
}

// Progress is v1 PhaseProgress: a phase and the roadmap rows it owns.
type Progress struct {
	Phase Phase  `json:"phase"`
	Items []Item `json:"items"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Pct   int    `json:"pct"`
}

// Page is the /os/roadmap model.
type Page struct {
	Phases      []Progress        `json:"phases"`
	Items       []Item            `json:"items"`
	Departments map[string]string `json:"departments"`
	Shipped     int               `json:"shipped"`
	Total       int               `json:"total"`
}

func validStatus(s string) bool {
	for _, v := range Statuses {
		if v == s {
			return true
		}
	}
	return false
}

// Validate applies RoadmapItemSchema.
func (i Item) Validate() error {
	switch {
	case i.ID == "" || i.Title == "":
		return fmt.Errorf("roadmap item %q: id and title are required", i.ID)
	case !quarterRe.MatchString(i.Quarter):
		return fmt.Errorf("roadmap item %q: quarter %q must look like 2026-Q2", i.ID, i.Quarter)
	case !validStatus(i.Status):
		return fmt.Errorf("roadmap item %q: status %q", i.ID, i.Status)
	}
	return nil
}

// Validate applies PhaseSchema.
func (p Phase) Validate() error {
	if p.ID == "" || p.Title == "" {
		return fmt.Errorf("phase %q: id and title are required", p.ID)
	}
	return nil
}

// gbrain retires G-Brain from v1's seeded copy: the knowledge core is the
// Optimal Engine / Brain in v2. Longer patterns first (strings.Replacer
// prefers the earlier argument at the same position).
var gbrain = strings.NewReplacer(
	"G-Brain (gbrain CLI)", "Optimal Engine",
	"gbrain CLI", "Optimal Engine",
	"into G-Brain", "into the Brain",
	"G-Brain", "Brain",
	"gbrain", "Brain",
)

// RetireGBrain rewrites v1's G-Brain mentions to v2's Brain.
func RetireGBrain(s string) string { return gbrain.Replace(s) }

// PhaseProgressOf is v1 phaseProgress: done/total of the rows each phase owns;
// a phase nobody has filed work under reads 0, never NaN.
func PhaseProgressOf(phases []Phase, items []Item) []Progress {
	out := make([]Progress, 0, len(phases))
	for _, ph := range phases {
		owned := []Item{}
		done := 0
		for _, it := range items {
			if it.PhaseID != nil && *it.PhaseID == ph.ID {
				owned = append(owned, it)
				if it.Status == "done" {
					done++
				}
			}
		}
		pct := 0
		if len(owned) > 0 {
			pct = int(math.Round(float64(done) / float64(len(owned)) * 100))
		}
		out = append(out, Progress{Phase: ph, Items: owned, Done: done, Total: len(owned), Pct: pct})
	}
	return out
}

func workspaceID(ctx context.Context, pool *pgxpool.Pool, slug string) (string, error) {
	var ws string
	err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, slug).Scan(&ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w (%s)", ErrNoWorkspace, slug)
	}
	return ws, err
}

const itemCols = `id, title, quarter, status, department_id, description, phase_id`

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	if err := row.Scan(&it.ID, &it.Title, &it.Quarter, &it.Status, &it.DepartmentID, &it.Description, &it.PhaseID); err != nil {
		return it, err
	}
	it.Title = RetireGBrain(it.Title)
	it.Description = RetireGBrain(it.Description)
	return it, it.Validate()
}

// Load reads the workspace's board. Items keep v1's order (quarter, then the
// raw title in byte order, as SQLite sorts it).
func Load(ctx context.Context, pool *pgxpool.Pool, workspaceSlug string) (*Page, error) {
	ws, err := workspaceID(ctx, pool, workspaceSlug)
	if err != nil {
		return nil, err
	}

	phases := []Phase{}
	rows, err := pool.Query(ctx, `SELECT id, number, title, items FROM founderos_phases
		WHERE workspace_id = $1 ORDER BY number, id`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p Phase
		var raw []byte
		if err := rows.Scan(&p.ID, &p.Number, &p.Title, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &p.Items); err != nil {
			rows.Close()
			return nil, fmt.Errorf("phase %q items: %w", p.ID, err)
		}
		if p.Items == nil {
			p.Items = []string{}
		}
		p.Title = RetireGBrain(p.Title)
		for i := range p.Items {
			p.Items[i] = RetireGBrain(p.Items[i])
		}
		if err := p.Validate(); err != nil {
			rows.Close()
			return nil, err
		}
		phases = append(phases, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := []Item{}
	rows, err = pool.Query(ctx, `SELECT `+itemCols+` FROM founderos_roadmap_items
		WHERE workspace_id = $1 ORDER BY quarter, title COLLATE "C", id`, ws)
	if err != nil {
		return nil, err
	}
	shipped := 0
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		if it.Status == "done" {
			shipped++
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	depts := map[string]string{}
	rows, err = pool.Query(ctx, `SELECT id, name FROM founderos_departments WHERE workspace_id = $1`, ws)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, err
		}
		depts[id] = name
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Page{Phases: PhaseProgressOf(phases, items), Items: items, Departments: depts, Shipped: shipped, Total: len(items)}, nil
}

// SetStatus is v1 PATCH /api/roadmap: mark a row done from the board (or push
// it back). Returns the updated row.
func SetStatus(ctx context.Context, pool *pgxpool.Pool, workspaceSlug, id, status string) (*Item, error) {
	if id == "" || !validStatus(status) {
		return nil, ErrBadStatus
	}
	ws, err := workspaceID(ctx, pool, workspaceSlug)
	if err != nil {
		return nil, err
	}
	it, err := scanItem(pool.QueryRow(ctx, `UPDATE founderos_roadmap_items SET status = $3
		WHERE workspace_id = $1 AND id = $2 RETURNING `+itemCols, ws, id, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}
