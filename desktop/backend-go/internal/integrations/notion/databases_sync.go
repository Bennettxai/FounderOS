package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// SyncDatabases syncs databases from Notion.
func (s *DatabaseService) SyncDatabases(ctx context.Context, userID string) (*SyncDatabasesResult, error) {
	slog.Info("Notion database sync starting for user", "user_id", userID)

	result := &SyncDatabasesResult{}

	searchBody := `{"filter":{"property":"object","value":"database"}}`
	body, err := s.provider.APIRequest(ctx, userID, "POST", "/search", strings.NewReader(searchBody))
	if err != nil {
		return nil, fmt.Errorf("failed to search databases: %w", err)
	}

	var searchResp struct {
		Results []NotionDatabase `json:"results"`
		HasMore bool             `json:"has_more"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	result.TotalDatabases = len(searchResp.Results)

	for _, db := range searchResp.Results {
		if err := s.saveDatabase(ctx, userID, db); err != nil {
			slog.Info("Failed to save database", "id", db.ID, "error", err)
			result.FailedDatabases++
		} else {
			result.SyncedDatabases++
		}
	}

	slog.Info("Notion database sync complete for user : synced / databases", "user_id", userID, "synced", result.SyncedDatabases, "total", result.TotalDatabases)

	return result, nil
}

// SyncPages syncs pages from a Notion database.
func (s *DatabaseService) SyncPages(ctx context.Context, userID, databaseID string) (*SyncPagesResult, error) {
	slog.Info("Notion page sync starting for user , database", "value0", userID, "value1", databaseID)

	var notionDatabaseID string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT notion_id FROM notion_databases WHERE id = $1 AND user_id = $2
	`, databaseID, userID).Scan(&notionDatabaseID)
	if err != nil {
		return nil, fmt.Errorf("database not found: %w", err)
	}

	result := &SyncPagesResult{}

	body, err := s.provider.APIRequest(ctx, userID, "POST",
		fmt.Sprintf("/databases/%s/query", notionDatabaseID),
		strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var queryResp struct {
		Results []NotionPage `json:"results"`
		HasMore bool         `json:"has_more"`
	}

	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	result.TotalPages = len(queryResp.Results)

	for _, page := range queryResp.Results {
		if err := s.savePage(ctx, userID, databaseID, page); err != nil {
			slog.Info("Failed to save page", "id", page.ID, "error", err)
			result.FailedPages++
		} else {
			result.SyncedPages++
		}
	}

	slog.Info("Notion page sync complete: synced / pages", "value0", result.SyncedPages, "value1", result.TotalPages)

	return result, nil
}
