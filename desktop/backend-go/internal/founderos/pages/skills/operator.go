package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoWorkspace means the workspace row is missing (bootstrap not run).
var ErrNoWorkspace = errors.New("workspace not found: run founderos-bootstrap")

var validStatus = map[string]bool{"live": true, "learning": true, "planned": true}

// LoadOperator reads the workspace's operator skills (founderos_skills, in
// order) and the agent id → name map their owners resolve through.
func LoadOperator(ctx context.Context, pool *pgxpool.Pool, workspaceSlug string) ([]OperatorSkill, map[string]string, error) {
	var ws string
	err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, workspaceSlug).Scan(&ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("%w (%s)", ErrNoWorkspace, workspaceSlug)
	}
	if err != nil {
		return nil, nil, err
	}
	rows, err := pool.Query(ctx, `SELECT id, name, category, description, owner_agent_id, status, tools, markdown
		FROM founderos_skills WHERE workspace_id = $1 ORDER BY ord, id`, ws)
	if err != nil {
		return nil, nil, err
	}
	skills := []OperatorSkill{}
	for rows.Next() {
		var s OperatorSkill
		var tools []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.Description, &s.OwnerAgentID, &s.Status, &tools, &s.Markdown); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if err := json.Unmarshal(tools, &s.Tools); err != nil {
			rows.Close()
			return nil, nil, fmt.Errorf("skill %q tools: %w", s.ID, err)
		}
		if s.Tools == nil {
			s.Tools = []string{}
		}
		if s.ID == "" || s.Name == "" || s.Category == "" || !validStatus[s.Status] {
			rows.Close()
			return nil, nil, fmt.Errorf("skill %q fails SkillSchema", s.ID)
		}
		skills = append(skills, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	names := map[string]string{}
	arows, err := pool.Query(ctx, `SELECT id, name FROM founderos_agents WHERE workspace_id = $1`, ws)
	if err != nil {
		return nil, nil, err
	}
	defer arows.Close()
	for arows.Next() {
		var id, name string
		if err := arows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		names[id] = name
	}
	return skills, names, arows.Err()
}
