package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Issue Types
// ============================================================================

// Issue represents a Linear issue.
type Issue struct {
	ID          string     `json:"id"`
	Identifier  string     `json:"identifier"` // e.g., "ENG-123"
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state"`
	Priority    int        `json:"priority"`
	Assignee    string     `json:"assignee,omitempty"`
	Project     string     `json:"project,omitempty"`
	Team        string     `json:"team"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// CreateIssueInput contains the data for creating a new issue.
type CreateIssueInput struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	TeamID      string `json:"team_id"`
	Priority    int    `json:"priority,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
}

// ============================================================================
// Sync Methods
// ============================================================================

// syncIssues syncs issues from Linear to the local database.
func (p *Provider) syncIssues(ctx context.Context, userID, accessToken string) (*syncStats, error) {
	query := `
		query($first: Int) {
			issues(first: $first, orderBy: updatedAt) {
				nodes {
					id
					identifier
					title
					description
					priority
					state {
						name
					}
					assignee {
						name
					}
					project {
						name
					}
					team {
						name
					}
					dueDate
					createdAt
					updatedAt
				}
			}
		}
	`

	resp, err := p.executeGraphQL(ctx, accessToken, query, map[string]interface{}{
		"first": 100,
	})
	if err != nil {
		return nil, err
	}

	var data struct {
		Issues struct {
			Nodes []struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Priority    int    `json:"priority"`
				State       struct {
					Name string `json:"name"`
				} `json:"state"`
				Assignee *struct {
					Name string `json:"name"`
				} `json:"assignee"`
				Project *struct {
					Name string `json:"name"`
				} `json:"project"`
				Team struct {
					Name string `json:"name"`
				} `json:"team"`
				DueDate   *string `json:"dueDate"`
				CreatedAt string  `json:"createdAt"`
				UpdatedAt string  `json:"updatedAt"`
			} `json:"nodes"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	stats := &syncStats{}

	for _, issue := range data.Issues.Nodes {
		assignee := ""
		if issue.Assignee != nil {
			assignee = issue.Assignee.Name
		}
		project := ""
		if issue.Project != nil {
			project = issue.Project.Name
		}

		_, err := p.pool.Exec(ctx, `
			INSERT INTO linear_issues (
				user_id, external_id, identifier, title, description,
				state, priority, assignee, project, team,
				due_date, external_created_at, external_updated_at, synced_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
			ON CONFLICT (user_id, external_id) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				state = EXCLUDED.state,
				priority = EXCLUDED.priority,
				assignee = EXCLUDED.assignee,
				project = EXCLUDED.project,
				due_date = EXCLUDED.due_date,
				external_updated_at = EXCLUDED.external_updated_at,
				synced_at = NOW(),
				updated_at = NOW()
		`, userID, issue.ID, issue.Identifier, issue.Title, issue.Description,
			issue.State.Name, issue.Priority, assignee, project, issue.Team.Name,
			issue.DueDate, issue.CreatedAt, issue.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to save issue %s: %w", issue.Identifier, err)
		}
		stats.Created++
	}

	return stats, nil
}

// ============================================================================
// API Methods
// ============================================================================

// GetIssues returns issues from the local database.
func (p *Provider) GetIssues(ctx context.Context, userID string, limit int) ([]*Issue, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT external_id, identifier, title, description, state, priority,
			assignee, project, team, due_date, external_created_at, external_updated_at
		FROM linear_issues
		WHERE user_id = $1
		ORDER BY external_updated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []*Issue
	for rows.Next() {
		var issue Issue
		var dueDate, createdAt, updatedAt *time.Time
		err := rows.Scan(
			&issue.ID, &issue.Identifier, &issue.Title, &issue.Description,
			&issue.State, &issue.Priority, &issue.Assignee, &issue.Project,
			&issue.Team, &dueDate, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		if dueDate != nil {
			issue.DueDate = dueDate
		}
		if createdAt != nil {
			issue.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			issue.UpdatedAt = *updatedAt
		}
		issues = append(issues, &issue)
	}

	return issues, nil
}

// CreateIssue creates a new issue in Linear.
func (p *Provider) CreateIssue(ctx context.Context, userID string, input CreateIssueInput) (*Issue, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	mutation := `
		mutation($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					state {
						name
					}
					team {
						name
					}
					createdAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"title":       input.Title,
			"description": input.Description,
			"teamId":      input.TeamID,
			"priority":    input.Priority,
		},
	}

	if input.AssigneeID != "" {
		variables["input"].(map[string]interface{})["assigneeId"] = input.AssigneeID
	}
	if input.ProjectID != "" {
		variables["input"].(map[string]interface{})["projectId"] = input.ProjectID
	}

	resp, err := p.executeGraphQL(ctx, token.AccessToken, mutation, variables)
	if err != nil {
		return nil, err
	}

	var data struct {
		IssueCreate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID          string `json:"id"`
				Identifier  string `json:"identifier"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Priority    int    `json:"priority"`
				State       struct {
					Name string `json:"name"`
				} `json:"state"`
				Team struct {
					Name string `json:"name"`
				} `json:"team"`
				CreatedAt string `json:"createdAt"`
			} `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	if !data.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	issue := &Issue{
		ID:         data.IssueCreate.Issue.ID,
		Identifier: data.IssueCreate.Issue.Identifier,
		Title:      data.IssueCreate.Issue.Title,
		State:      data.IssueCreate.Issue.State.Name,
		Team:       data.IssueCreate.Issue.Team.Name,
		Priority:   data.IssueCreate.Issue.Priority,
	}

	return issue, nil
}
