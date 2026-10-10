package linear

import (
	"context"
	"encoding/json"
	"fmt"
)

// ============================================================================
// Team Types
// ============================================================================

// Team represents a Linear team.
type Team struct {
	ID          string `json:"id"`
	Key         string `json:"key"` // e.g., "ENG"
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IssueCount  int    `json:"issueCount"`
}

// ============================================================================
// Sync Methods
// ============================================================================

// syncTeams syncs teams from Linear to the local database.
func (p *Provider) syncTeams(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	query := `
		query {
			teams {
				nodes {
					id
					key
					name
					description
					issueCount
				}
			}
		}
	`

	resp, err := p.executeGraphQL(ctx, accessToken, query, nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		Teams struct {
			Nodes []struct {
				ID          string `json:"id"`
				Key         string `json:"key"`
				Name        string `json:"name"`
				Description string `json:"description"`
				IssueCount  int    `json:"issueCount"`
			} `json:"nodes"`
		} `json:"teams"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	stats := &syncStats{}

	for _, team := range data.Teams.Nodes {
		_, err := p.pool.Exec(ctx, `
			INSERT INTO linear_teams (
				user_id, external_id, key, name, description, issue_count, synced_at
			) VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (user_id, external_id) DO UPDATE SET
				key = EXCLUDED.key,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				issue_count = EXCLUDED.issue_count,
				synced_at = NOW(),
				updated_at = NOW()
		`, userID, team.ID, team.Key, team.Name, team.Description, team.IssueCount)

		if err != nil {
			return nil, fmt.Errorf("failed to save team %s: %w", team.Name, err)
		}
		stats.Created++
	}

	return stats, nil
}

// ============================================================================
// API Methods
// ============================================================================

// GetTeams returns teams from the local database.
func (p *Provider) GetTeams(ctx context.Context, userID string) ([]*Team, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT external_id, key, name, description, issue_count
		FROM linear_teams
		WHERE user_id = $1
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*Team
	for rows.Next() {
		var team Team
		err := rows.Scan(&team.ID, &team.Key, &team.Name, &team.Description, &team.IssueCount)
		if err != nil {
			return nil, err
		}
		teams = append(teams, &team)
	}

	return teams, nil
}
