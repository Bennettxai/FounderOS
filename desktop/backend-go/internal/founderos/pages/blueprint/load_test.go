package blueprint

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func throwawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_blueprint_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// SeedRoster writes a small roster into two workspaces so the loader's
// workspace scope is exercised. Returns the founderos workspace id.
func seedRoster(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	var ws, other string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Other','other','u1') RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO founderos_departments (id, workspace_id, name, slug, tagline, color, ord) VALUES ('d-tech', $1, 'TECH', 'tech', 'the OS', '#fff', 2), ('d-sales', $1, 'Sales', 'sales', 'pipeline', '#fff', 1)`,
		`INSERT INTO founderos_agents (id, workspace_id, department_id, name, role, status, tier, description, model, tools) VALUES
		   ('conductor', $1, 'd-tech', 'Conductor', 'Super agent', 'active', 'lead', 'Routes', 'claude', '[]'),
		   ('comms-agent', $1, 'd-sales', 'Comms', 'Inbox', 'active', 'worker', 'Reads mail', 'claude', '["gmail","slack"]')`,
		`INSERT INTO founderos_people (id, workspace_id, department_id, name, role) VALUES ('koda', $1, 'd-sales', 'Koda', 'Setter')`,
		`INSERT INTO founderos_skills (id, workspace_id, name, category, description, ord) VALUES ('voice', $1, 'Voice', 'writing', 'writing voice', 1)`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatal(err)
		}
	}
	// a row in another workspace must never leak in
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_skills (id, workspace_id, name, category) VALUES ('leak', $1, 'Leak', 'x')`, other); err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestLoadRegistriesReadsTheWorkspaceRoster(t *testing.T) {
	pool := throwawayDB(t)
	ws := seedRoster(t, pool)
	reg, err := LoadRegistries(context.Background(), pool, ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Departments) != 2 || reg.Departments[0].Slug != "sales" {
		t.Fatalf("departments = %+v (want ord order)", reg.Departments)
	}
	if len(reg.Agents) != 2 || len(reg.People) != 1 || len(reg.Skills) != 1 {
		t.Fatalf("registries = %+v", reg)
	}
	var comms Agent
	for _, a := range reg.Agents {
		if a.ID == "comms-agent" {
			comms = a
		}
	}
	if len(comms.Tools) != 2 || comms.Tools[0] != "gmail" || comms.DepartmentID != "d-sales" {
		t.Fatalf("comms = %+v", comms)
	}
	g, err := Compile(reg, fixtureOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(g, "agent-comms-agent", "dept-sales", "member-of") || !hasEdge(g, "person-koda", "dept-sales", "member-of") {
		t.Fatal("roster membership edges missing")
	}
}

// FounderOS v1 reads the roster as SELECT * FROM agents ORDER BY tier, name
// (people: department_id, name; skills: ord, name) and the blueprint keeps
// that order: the Conductor's connection list in the inspector reads
// top-down the same way. SQLite compares bytes, so names sort in C order.
func TestLoadRegistriesKeepsProductionsRosterOrder(t *testing.T) {
	pool := throwawayDB(t)
	ws := seedRoster(t, pool)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO founderos_agents (id, workspace_id, department_id, name, role, status, tier, description, model, tools) VALUES
		   ('a-zed', $1, 'd-sales', 'Zed', 'r', 'active', 'lead', 'd', 'm', '[]'),
		   ('z-alpha', $1, 'd-sales', 'alpha', 'r', 'active', 'lead', 'd', 'm', '[]'),
		   ('b-beta', $1, 'd-sales', 'Beta', 'r', 'active', 'lead', 'd', 'm', '[]')`,
		`INSERT INTO founderos_people (id, workspace_id, department_id, name, role) VALUES ('a-zoe', $1, 'd-sales', 'Zoe', 'r'), ('z-ann', $1, 'd-sales', 'Ann', 'r'), ('m-tess', $1, 'd-tech', 'Tess', 'r')`,
		`INSERT INTO founderos_skills (id, workspace_id, name, category, description, ord) VALUES ('a-skill', $1, 'Zeta', 'x', 'd', 1)`,
	} {
		if _, err := pool.Exec(ctx, q, ws); err != nil {
			t.Fatal(err)
		}
	}
	reg, err := LoadRegistries(ctx, pool, ws)
	if err != nil {
		t.Fatal(err)
	}
	var agents, people, skills []string
	for _, a := range reg.Agents {
		agents = append(agents, a.Name)
	}
	for _, p := range reg.People {
		people = append(people, p.Name)
	}
	for _, s := range reg.Skills {
		skills = append(skills, s.Name)
	}
	if got := fmt.Sprint(agents); got != "[Beta Conductor Zed alpha Comms]" {
		t.Errorf("agents = %s, want tier then name in byte order", got)
	}
	if got := fmt.Sprint(people); got != "[Ann Koda Zoe Tess]" {
		t.Errorf("people = %s, want department then name", got)
	}
	if got := fmt.Sprint(skills); got != "[Voice Zeta]" {
		t.Errorf("skills = %s, want ord then name", got)
	}
}
