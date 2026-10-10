package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Project Types
// ============================================================================

// Project represents a Linear project.
type Project struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state"`
	Progress    float64    `json:"progress"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	TargetDate  *time.Time `json:"targetDate,omitempty"`
	Team        string     `json:"team"`
}

// ============================================================================
// Sync Methods
// ============================================================================

// syncProjects syncs projects from Linear to the local database.
func (p *Provider) syncProjects(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	query := `
		query($first: Int) {
			projects(first: $first) {
				nodes {
					id
					name
					description
					state
					progress
					startDate
					targetDate
					teams {
						nodes {
							name
						}
					}
				}
			}
		}
	`

	resp, err := p.executeGraphQL(ctx, accessToken, query, map[string]interface{}{
		"first": 50,
	})
	if err != nil {
		return nil, err
	}

	var data struct {
		Projects struct {
			Nodes []struct {
				ID          string  `json:"id"`
				Name        string  `json:"name"`
				Description string  `json:"description"`
				State       string  `json:"state"`
				Progress    float64 `json:"progress"`
				StartDate   *string `json:"startDate"`
				TargetDate  *string `json:"targetDate"`
				Teams       struct {
					Nodes []struct {
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"teams"`
			} `json:"nodes"`
		} `json:"projects"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	stats := &syncStats{}

	for _, proj := range data.Projects.Nodes {
		team := ""
		if len(proj.Teams.Nodes) > 0 {
			team = proj.Teams.Nodes[0].Name
		}

		_, err := p.pool.Exec(ctx, `
			INSERT INTO linear_projects (
				user_id, external_id, name, description,
				state, progress, start_date, target_date, team, synced_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			ON CONFLICT (user_id, external_id) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				state = EXCLUDED.state,
				progress = EXCLUDED.progress,
				start_date = EXCLUDED.start_date,
				target_date = EXCLUDED.target_date,
				team = EXCLUDED.team,
				synced_at = NOW(),
				updated_at = NOW()
		`, userID, proj.ID, proj.Name, proj.Description,
			proj.State, proj.Progress, proj.StartDate, proj.TargetDate, team)

		if err != nil {
			return nil, fmt.Errorf("failed to save project %s: %w", proj.Name, err)
		}
		stats.Created++
	}

	return stats, nil
}

// ============================================================================
// API Methods
// ============================================================================

// GetProjects returns projects from the local database.
func (p *Provider) GetProjects(ctx context.Context, userID string) ([]*Project, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT external_id, name, description, state, progress,
			start_date, target_date, team
		FROM linear_projects
		WHERE user_id = $1
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID, &project.Name, &project.Description,
			&project.State, &project.Progress, &project.StartDate,
			&project.TargetDate, &project.Team,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, &project)
	}

	return projects, nil
}
