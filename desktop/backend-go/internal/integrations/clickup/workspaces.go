package clickup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// GetWorkspaces retrieves all authorized workspaces (teams) for the user.
func (p *Provider) GetWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", APIURL+"/team", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
	}

	var result struct {
		Teams []Workspace `json:"teams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Teams, nil
}

// GetSpaces retrieves all spaces for a given workspace (team).
func (p *Provider) GetSpaces(ctx context.Context, userID, teamID string) ([]Space, error) {
	token, err := p.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	url := fmt.Sprintf("%s/team/%s/space", APIURL, teamID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Add query parameter to include archived spaces if needed
	query := req.URL.Query()
	query.Add("archived", "false")
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get spaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
	}

	var result struct {
		Spaces []Space `json:"spaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Spaces, nil
}

// SyncWorkspaces fetches workspaces from ClickUp API and persists them to the database.
func (p *Provider) SyncWorkspaces(ctx context.Context, userID string) (int, error) {
	workspaces, err := p.GetWorkspaces(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch workspaces: %w", err)
	}

	queries := sqlc.New(p.pool)
	synced := 0

	for _, workspace := range workspaces {
		memberCount := int32(len(workspace.Members))

		strPtr := func(s string) *string {
			if s == "" {
				return nil
			}
			return &s
		}

		_, err := queries.UpsertClickUpWorkspace(ctx, sqlc.UpsertClickUpWorkspaceParams{
			UserID:      userID,
			WorkspaceID: workspace.ID,
			Name:        workspace.Name,
			Color:       strPtr(workspace.Color),
			Avatar:      strPtr(workspace.Avatar),
			MemberCount: &memberCount,
		})
		if err != nil {
			slog.Warn("Failed to upsert workspace", "workspace_id", workspace.ID, "error", err)
			continue
		}
		synced++
	}

	return synced, nil
}
