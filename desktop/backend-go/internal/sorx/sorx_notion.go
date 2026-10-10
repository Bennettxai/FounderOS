package sorx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/integrations/notion"
)

// ============================================================================
// Notion Actions
// ============================================================================

func notionSearch(ctx context.Context, ac ActionContext) (interface{}, error) {
	query, _ := ac.Params["query"].(string)

	slog.Info("notionSearch", "user_id", ac.Execution.UserID, "query", query)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "notion"); err != nil {
		return nil, fmt.Errorf("Notion not connected: %w", err)
	}

	provider := notion.NewProvider(pool)
	dbService := notion.NewDatabaseService(provider)

	filterType, _ := ac.Params["filter_type"].(string) // "page" or "database"
	pageSize := 10
	if val, ok := ac.Params["page_size"].(float64); ok {
		pageSize = int(val)
	}
	cursor, _ := ac.Params["cursor"].(string)

	searchResp, err := dbService.Search(ctx, ac.Execution.UserID, query, filterType, pageSize, cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to search Notion: %w", err)
	}

	slog.Info("notionSearch complete", "results", len(searchResp.Results), "has_more", searchResp.HasMore)
	return map[string]interface{}{
		"query":       query,
		"results":     searchResp.Results,
		"count":       len(searchResp.Results),
		"has_more":    searchResp.HasMore,
		"next_cursor": searchResp.NextCursor,
	}, nil
}

func notionCreatePage(ctx context.Context, ac ActionContext) (interface{}, error) {
	title, _ := ac.Params["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	slog.Info("notionCreatePage", "user_id", ac.Execution.UserID, "title", title)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "notion"); err != nil {
		return nil, fmt.Errorf("Notion not connected: %w", err)
	}

	provider := notion.NewProvider(pool)
	dbService := notion.NewDatabaseService(provider)

	databaseID, _ := ac.Params["database_id"].(string)
	if databaseID == "" {
		return nil, fmt.Errorf("database_id is required")
	}

	// Build properties with title
	properties := map[string]interface{}{
		"Name": map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"text": map[string]interface{}{
						"content": title,
					},
				},
			},
		},
	}

	// Merge additional properties from params
	if extraProps, ok := ac.Params["properties"].(map[string]interface{}); ok {
		for k, v := range extraProps {
			properties[k] = v
		}
	}

	page, err := dbService.CreatePage(ctx, ac.Execution.UserID, databaseID, properties)
	if err != nil {
		return nil, fmt.Errorf("failed to create Notion page: %w", err)
	}

	slog.Info("notionCreatePage complete", "page_id", page.ID, "url", page.URL)
	return map[string]interface{}{
		"created": true,
		"page_id": page.ID,
		"url":     page.URL,
		"title":   title,
	}, nil
}
