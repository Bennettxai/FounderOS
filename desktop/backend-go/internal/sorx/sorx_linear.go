package sorx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/integrations/linear"
)

// ============================================================================
// Linear Actions
// ============================================================================

func linearListIssues(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("linearListIssues", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "linear"); err != nil {
		return nil, fmt.Errorf("Linear not connected: %w", err)
	}

	provider := linear.NewProvider(pool)

	limit := 10
	if val, ok := ac.Params["limit"].(float64); ok {
		limit = int(val)
	}

	issues, err := provider.GetIssues(ctx, ac.Execution.UserID, limit)
	if err != nil {
		slog.Warn("linearListIssues: GetIssues failed, attempting sync", "error", err)
		// Try syncing first, then retry
		if _, syncErr := provider.Sync(ctx, ac.Execution.UserID, integrations.SyncOptions{}); syncErr != nil {
			return nil, fmt.Errorf("failed to sync Linear issues: %w", syncErr)
		}
		issues, err = provider.GetIssues(ctx, ac.Execution.UserID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get Linear issues: %w", err)
		}
	}

	slog.Info("linearListIssues complete", "count", len(issues))
	return map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	}, nil
}

func linearCreateIssue(ctx context.Context, ac ActionContext) (interface{}, error) {
	title, _ := ac.Params["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	slog.Info("linearCreateIssue", "user_id", ac.Execution.UserID, "title", title)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "linear"); err != nil {
		return nil, fmt.Errorf("Linear not connected: %w", err)
	}

	provider := linear.NewProvider(pool)

	description, _ := ac.Params["description"].(string)
	teamID, _ := ac.Params["team_id"].(string)
	priority := 0
	if val, ok := ac.Params["priority"].(float64); ok {
		priority = int(val)
	}

	input := linear.CreateIssueInput{
		Title:       title,
		Description: description,
		TeamID:      teamID,
		Priority:    priority,
	}

	issue, err := provider.CreateIssue(ctx, ac.Execution.UserID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create Linear issue: %w", err)
	}

	slog.Info("linearCreateIssue complete", "issue_id", issue.ID, "identifier", issue.Identifier)
	return map[string]interface{}{
		"created":    true,
		"issue_id":   issue.ID,
		"identifier": issue.Identifier,
		"title":      issue.Title,
		"state":      issue.State,
	}, nil
}
