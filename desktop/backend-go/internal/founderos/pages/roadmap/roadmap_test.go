package roadmap

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
)

const insertItem = `INSERT INTO founderos_roadmap_items (id, workspace_id, title, quarter, status, department_id, description, phase_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

func TestLoadBuildsTheBoardFromTheWorkspaceRows(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	other := catalogdb.Workspace(t, pool, "vantage")
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('dept-tech', $1, 'TECH', 'tech', '#fff', 1)`, []any{ws}},
		{`INSERT INTO founderos_phases (id, workspace_id, number, title, items) VALUES ('phase-2', $1, 2, 'Real Agents', '["Runtime"]')`, []any{ws}},
		{`INSERT INTO founderos_phases (id, workspace_id, number, title, items) VALUES ('phase-1', $1, 1, 'Real Connections', '["G-Brain"]')`, []any{ws}},
		{`INSERT INTO founderos_phases (id, workspace_id, number, title, items) VALUES ('phase-x', $1, 1, 'Elsewhere', '[]')`, []any{other}},
		{insertItem, []any{"rm-b", ws, "Monochrome rebuild", "2026-Q2", "done", "dept-tech", "IMAP, Slack, gbrain wired.", "phase-1"}},
		{insertItem, []any{"rm-a", ws, "G-Brain provider live", "2026-Q2", "done", "dept-tech", "gbrain CLI doctor/query.", "phase-1"}},
		{insertItem, []any{"rm-c", ws, "Connect Slack", "2026-Q2", "now", nil, "", "phase-1"}},
		{insertItem, []any{"rm-d", ws, "Auth", "2026-Q4", "next", "dept-tech", "", nil}},
		{insertItem, []any{"rm-e", ws, "Scheduler", "2026-Q3", "later", "dept-tech", "", "phase-2"}},
		{insertItem, []any{"rm-x", other, "Not ours", "2026-Q2", "done", nil, "", "phase-x"}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(ctx, pool, "founderos")
	if err != nil {
		t.Fatal(err)
	}
	// v1 db.roadmap.all(): ORDER BY quarter, title (byte order), from the raw title.
	var ids []string
	for _, it := range p.Items {
		ids = append(ids, it.ID)
	}
	if got := join(ids); got != "rm-c rm-a rm-b rm-e rm-d" {
		t.Fatalf("item order = %s", got)
	}
	if p.Shipped != 2 || p.Total != 5 {
		t.Fatalf("shipped %d/%d, want 2/5", p.Shipped, p.Total)
	}
	// G-Brain is gone: the rows read Brain.
	a := p.Items[1]
	if a.Title != "Brain provider live" || a.Description != "Optimal Engine doctor/query." {
		t.Fatalf("G-Brain not retired: %+v", a)
	}
	if p.Items[2].Description != "IMAP, Slack, Brain wired." {
		t.Fatalf("gbrain not retired: %q", p.Items[2].Description)
	}
	if p.Items[0].DepartmentID != nil || p.Items[4].PhaseID != nil {
		t.Fatal("NULL refs must stay null")
	}
	if *a.PhaseID != "phase-1" || *a.DepartmentID != "dept-tech" {
		t.Fatalf("refs not mapped: %+v", a)
	}
	if p.Departments["dept-tech"] != "TECH" {
		t.Fatalf("departments = %v", p.Departments)
	}
	// Phases in number order, each with the rows it owns (v1 phaseProgress).
	if len(p.Phases) != 2 || p.Phases[0].Phase.ID != "phase-1" || p.Phases[1].Phase.ID != "phase-2" {
		t.Fatalf("phases = %+v", p.Phases)
	}
	ph := p.Phases[0]
	if ph.Done != 2 || ph.Total != 3 || ph.Pct != 67 || len(ph.Items) != 3 {
		t.Fatalf("phase-1 progress = %d/%d %d%% (%d items)", ph.Done, ph.Total, ph.Pct, len(ph.Items))
	}
	if ph.Phase.Items[0] != "Brain" {
		t.Fatalf("phase items not retired: %v", ph.Phase.Items)
	}
	if p.Phases[1].Pct != 0 || p.Phases[1].Total != 1 {
		t.Fatalf("phase-2 = %+v", p.Phases[1])
	}
}

func TestPhaseProgressWithNoRowsReadsZeroNotNaN(t *testing.T) {
	got := PhaseProgressOf([]Phase{{ID: "p", Number: 1, Title: "Empty", Items: []string{}}}, nil)
	if got[0].Pct != 0 || got[0].Total != 0 || got[0].Items == nil {
		t.Fatalf("got %+v", got[0])
	}
}

func TestLoadWithoutTheWorkspaceIsAnError(t *testing.T) {
	pool := catalogdb.TestDB(t)
	if _, err := Load(context.Background(), pool, "founderos"); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("err = %v, want ErrNoWorkspace", err)
	}
}

func TestSetStatusMovesARowAndRefusesBadInput(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	if _, err := pool.Exec(ctx, insertItem, "rm-a", ws, "Auth", "2026-Q4", "next", nil, "", nil); err != nil {
		t.Fatal(err)
	}
	it, err := SetStatus(ctx, pool, "founderos", "rm-a", "done")
	if err != nil || it.Status != "done" || it.ID != "rm-a" {
		t.Fatalf("SetStatus = %+v, %v", it, err)
	}
	p, err := Load(ctx, pool, "founderos")
	if err != nil || p.Items[0].Status != "done" {
		t.Fatalf("status not persisted: %+v %v", p, err)
	}
	if _, err := SetStatus(ctx, pool, "founderos", "rm-a", "planned"); !errors.Is(err, ErrBadStatus) {
		t.Fatalf("bad status: %v", err)
	}
	if _, err := SetStatus(ctx, pool, "founderos", "nope", "done"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
}

func TestRetireGBrain(t *testing.T) {
	for in, want := range map[string]string{
		"G-Brain (gbrain CLI)":                      "Optimal Engine",
		"G-Brain provider live":                     "Brain provider live",
		"transcripts file themselves into G-Brain.": "transcripts file themselves into the Brain.",
		"App, gbrain and agents run on the host":    "App, Brain and agents run on the host",
		"gbrain CLI doctor/query + brain-store":     "Optimal Engine doctor/query + brain-store",
		"Free-tier project unpaused; gbrain hybrid": "Free-tier project unpaused; Brain hybrid",
		"Slack": "Slack",
	} {
		if got := RetireGBrain(in); got != want {
			t.Errorf("RetireGBrain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestItemValidateMirrorsRoadmapItemSchema(t *testing.T) {
	ok := Item{ID: "a", Title: "t", Quarter: "2026-Q2", Status: "now"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*Item){
		"no title":    func(i *Item) { i.Title = "" },
		"bad quarter": func(i *Item) { i.Quarter = "Q2" },
		"bad status":  func(i *Item) { i.Status = "planned" },
	} {
		it := ok
		mut(&it)
		if it.Validate() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func join(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out
}
