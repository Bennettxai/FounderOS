package blueprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadRegistries reads the roster (departments, agents, people, skills) of
// one workspace from the bridge's founderos_* tables.
func LoadRegistries(ctx context.Context, pool *pgxpool.Pool, workspaceID string) (Registries, error) {
	var reg Registries
	rows, err := pool.Query(ctx, `SELECT id, slug, name, tagline FROM founderos_departments WHERE workspace_id = $1 ORDER BY ord, id`, workspaceID)
	if err != nil {
		return reg, fmt.Errorf("departments: %w", err)
	}
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Tagline); err != nil {
			rows.Close()
			return reg, err
		}
		reg.Departments = append(reg.Departments, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return reg, err
	}

	rows, err = pool.Query(ctx, `SELECT id, department_id, name, role, status, tier, description, model, tools FROM founderos_agents WHERE workspace_id = $1 ORDER BY tier COLLATE "C", name COLLATE "C", id`, workspaceID)
	if err != nil {
		return reg, fmt.Errorf("agents: %w", err)
	}
	for rows.Next() {
		var a Agent
		var tools []byte
		if err := rows.Scan(&a.ID, &a.DepartmentID, &a.Name, &a.Role, &a.Status, &a.Tier, &a.Description, &a.Model, &tools); err != nil {
			rows.Close()
			return reg, err
		}
		if len(tools) > 0 {
			if err := json.Unmarshal(tools, &a.Tools); err != nil {
				rows.Close()
				return reg, fmt.Errorf("agent %s tools: %w", a.ID, err)
			}
		}
		reg.Agents = append(reg.Agents, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return reg, err
	}

	rows, err = pool.Query(ctx, `SELECT id, department_id, name, role FROM founderos_people WHERE workspace_id = $1 ORDER BY department_id COLLATE "C" NULLS FIRST, name COLLATE "C", id`, workspaceID)
	if err != nil {
		return reg, fmt.Errorf("people: %w", err)
	}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.DepartmentID, &p.Name, &p.Role); err != nil {
			rows.Close()
			return reg, err
		}
		reg.People = append(reg.People, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return reg, err
	}

	rows, err = pool.Query(ctx, `SELECT id, name, description FROM founderos_skills WHERE workspace_id = $1 ORDER BY ord, name COLLATE "C", id`, workspaceID)
	if err != nil {
		return reg, fmt.Errorf("skills: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.Description); err != nil {
			return reg, err
		}
		reg.Skills = append(reg.Skills, s)
	}
	return reg, rows.Err()
}
