package skills

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
)

func TestLoadOperatorReadsSkillsAndAgentNamesForTheWorkspace(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	other := catalogdb.Workspace(t, pool, "vantage")
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO founderos_departments (id, workspace_id, name, slug, color, ord) VALUES ('sales', $1, 'Sales', 'sales', '#fff', 1)`, ws)
	mustExec(`INSERT INTO founderos_agents (id, workspace_id, department_id, name, status, tier) VALUES ('closer', $1, 'sales', 'Closer Agent', 'active', 'worker')`, ws)
	mustExec(`INSERT INTO founderos_skills (id, workspace_id, name, category, description, owner_agent_id, status, tools, markdown, ord) VALUES
		('skill-b', $1, 'Objection map', 'Sales', 'd2', NULL, 'learning', '[]', '# B', 2),
		('skill-a', $1, 'Cold opener', 'Sales', 'd1', 'closer', 'live', '["gmail"]', '# A', 1)`, ws)
	mustExec(`INSERT INTO founderos_skills (id, workspace_id, name, category, status) VALUES ('skill-x', $1, 'Other', 'Ops', 'live')`, other)

	got, names, err := LoadOperator(ctx, pool, "founderos")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "skill-a" || got[1].ID != "skill-b" {
		t.Fatalf("got %+v, want skill-a then skill-b from founderos only", got)
	}
	a := got[0]
	if a.Name != "Cold opener" || a.Category != "Sales" || a.Description != "d1" || a.Status != "live" || a.OwnerAgentID == nil || *a.OwnerAgentID != "closer" || len(a.Tools) != 1 || a.Markdown != "# A" {
		t.Errorf("a = %+v", a)
	}
	if got[1].OwnerAgentID != nil || got[1].Tools == nil || len(got[1].Tools) != 0 {
		t.Errorf("b = %+v", got[1])
	}
	if names["closer"] != "Closer Agent" {
		t.Errorf("agent names %v", names)
	}
}

func TestLoadOperatorWithoutTheWorkspaceIsAnError(t *testing.T) {
	pool := catalogdb.TestDB(t)
	if _, _, err := LoadOperator(context.Background(), pool, "founderos"); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("err = %v", err)
	}
}
