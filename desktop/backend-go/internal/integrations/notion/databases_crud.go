package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetDatabases retrieves databases for a user.
func (s *DatabaseService) GetDatabases(ctx context.Context, userID string) ([]*Database, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, notion_id, title, description, icon, cover,
			properties, url, synced_at, created_at, updated_at
		FROM notion_databases
		WHERE user_id = $1
		ORDER BY title
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []*Database
	for rows.Next() {
		var db Database
		var propertiesJSON []byte

		err := rows.Scan(
			&db.ID, &db.UserID, &db.NotionID, &db.Title, &db.Description,
			&db.Icon, &db.Cover, &propertiesJSON, &db.URL, &db.SyncedAt,
			&db.CreatedAt, &db.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(propertiesJSON) > 0 {
			json.Unmarshal(propertiesJSON, &db.Properties)
		}

		databases = append(databases, &db)
	}

	return databases, nil
}

// GetPages retrieves pages for a database.
func (s *DatabaseService) GetPages(ctx context.Context, userID, databaseID string, limit, offset int) ([]*Page, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, notion_id, database_id, title, icon, cover,
			properties, url, archived, synced_at, created_at, updated_at
		FROM notion_pages
		WHERE user_id = $1 AND database_id = $2 AND archived = false
		ORDER BY title
		LIMIT $3 OFFSET $4
	`, userID, databaseID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*Page
	for rows.Next() {
		var p Page
		var propertiesJSON []byte

		err := rows.Scan(
			&p.ID, &p.UserID, &p.NotionID, &p.DatabaseID, &p.Title,
			&p.Icon, &p.Cover, &propertiesJSON, &p.URL, &p.Archived,
			&p.SyncedAt, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(propertiesJSON) > 0 {
			json.Unmarshal(propertiesJSON, &p.Properties)
		}

		pages = append(pages, &p)
	}

	return pages, nil
}

// GetPage returns a specific page from Notion API.
func (s *DatabaseService) GetPage(ctx context.Context, userID, pageID string) (*NotionPageAPI, error) {
	body, err := s.provider.APIRequest(ctx, userID, "GET", fmt.Sprintf("/pages/%s", pageID), nil)
	if err != nil {
		return nil, err
	}

	var page NotionPageAPI
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// CreatePage creates a new page in a Notion database.
func (s *DatabaseService) CreatePage(ctx context.Context, userID, databaseID string, properties map[string]interface{}) (*NotionPageAPI, error) {
	requestBody := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": databaseID,
		},
		"properties": properties,
	}

	bodyJSON, _ := json.Marshal(requestBody)
	body, err := s.provider.APIRequest(ctx, userID, "POST", "/pages", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	var page NotionPageAPI
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// UpdatePage updates an existing Notion page.
func (s *DatabaseService) UpdatePage(ctx context.Context, userID, pageID string, properties map[string]interface{}) (*NotionPageAPI, error) {
	requestBody := map[string]interface{}{
		"properties": properties,
	}

	bodyJSON, _ := json.Marshal(requestBody)
	body, err := s.provider.APIRequest(ctx, userID, "PATCH", fmt.Sprintf("/pages/%s", pageID), strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}

	var page NotionPageAPI
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	return &page, nil
}

// GetToken returns the user's Notion token (delegates to provider).
func (s *DatabaseService) GetToken(ctx context.Context, userID string) (interface{}, error) {
	return s.provider.GetToken(ctx, userID)
}
