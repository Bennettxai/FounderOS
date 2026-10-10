package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ListDatabases returns databases from Notion API.
func (s *DatabaseService) ListDatabases(ctx context.Context, userID string, pageSize int, cursor string) ([]NotionDatabaseAPI, string, bool, error) {
	requestBody := map[string]interface{}{
		"filter": map[string]interface{}{
			"property": "object",
			"value":    "database",
		},
		"page_size": pageSize,
	}
	if cursor != "" {
		requestBody["start_cursor"] = cursor
	}

	bodyJSON, _ := json.Marshal(requestBody)
	body, err := s.provider.APIRequest(ctx, userID, "POST", "/search", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, "", false, err
	}

	var resp struct {
		Results    []NotionDatabaseAPI `json:"results"`
		HasMore    bool                `json:"has_more"`
		NextCursor *string             `json:"next_cursor"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", false, err
	}

	nextCursor := ""
	if resp.NextCursor != nil {
		nextCursor = *resp.NextCursor
	}

	return resp.Results, nextCursor, resp.HasMore, nil
}

// GetDatabase returns a specific database from Notion API.
func (s *DatabaseService) GetDatabase(ctx context.Context, userID, databaseID string) (*NotionDatabaseAPI, error) {
	body, err := s.provider.APIRequest(ctx, userID, "GET", fmt.Sprintf("/databases/%s", databaseID), nil)
	if err != nil {
		return nil, err
	}

	var db NotionDatabaseAPI
	if err := json.Unmarshal(body, &db); err != nil {
		return nil, err
	}

	return &db, nil
}

// QueryDatabase queries a Notion database.
func (s *DatabaseService) QueryDatabase(ctx context.Context, userID, databaseID string, pageSize int, cursor string, filter, sorts interface{}) (*NotionQueryResponse, error) {
	requestBody := map[string]interface{}{
		"page_size": pageSize,
	}
	if cursor != "" {
		requestBody["start_cursor"] = cursor
	}
	if filter != nil {
		requestBody["filter"] = filter
	}
	if sorts != nil {
		requestBody["sorts"] = sorts
	}

	bodyJSON, _ := json.Marshal(requestBody)
	body, err := s.provider.APIRequest(ctx, userID, "POST", fmt.Sprintf("/databases/%s/query", databaseID), strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	var resp NotionQueryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Search searches Notion for pages and databases.
func (s *DatabaseService) Search(ctx context.Context, userID, query, filterType string, pageSize int, cursor string) (*NotionSearchResponse, error) {
	requestBody := map[string]interface{}{
		"page_size": pageSize,
	}
	if query != "" {
		requestBody["query"] = query
	}
	if filterType != "" {
		requestBody["filter"] = map[string]interface{}{
			"property": "object",
			"value":    filterType,
		}
	}
	if cursor != "" {
		requestBody["start_cursor"] = cursor
	}

	bodyJSON, _ := json.Marshal(requestBody)
	body, err := s.provider.APIRequest(ctx, userID, "POST", "/search", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	var resp NotionSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
