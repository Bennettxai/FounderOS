package notion

import (
	"context"
	"encoding/json"
)

// DatabaseService handles Notion database operations.
type DatabaseService struct {
	provider *Provider
}

// NewDatabaseService creates a new database service.
func NewDatabaseService(provider *Provider) *DatabaseService {
	return &DatabaseService{provider: provider}
}

// saveDatabase saves a Notion database to our database.
func (s *DatabaseService) saveDatabase(ctx context.Context, userID string, db NotionDatabase) error {
	title := ""
	if len(db.Title) > 0 {
		title = db.Title[0].PlainText
	}

	description := ""
	if len(db.Description) > 0 {
		description = db.Description[0].PlainText
	}

	icon := ""
	if db.Icon != nil {
		if db.Icon.Type == "emoji" {
			icon = db.Icon.Emoji
		} else if db.Icon.File != nil {
			icon = db.Icon.File.URL
		} else if db.Icon.External != nil {
			icon = db.Icon.External.URL
		}
	}

	cover := ""
	if db.Cover != nil {
		if db.Cover.File != nil {
			cover = db.Cover.File.URL
		} else if db.Cover.External != nil {
			cover = db.Cover.External.URL
		}
	}

	propertiesJSON, _ := json.Marshal(db.Properties)

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO notion_databases (
			user_id, notion_id, title, description, icon, cover,
			properties, url, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id, notion_id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			icon = EXCLUDED.icon,
			cover = EXCLUDED.cover,
			properties = EXCLUDED.properties,
			url = EXCLUDED.url,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, db.ID, title, description, icon, cover, propertiesJSON, db.URL)

	return err
}

// savePage saves a Notion page to our database.
func (s *DatabaseService) savePage(ctx context.Context, userID, databaseID string, page NotionPage) error {
	title := extractTitle(page.Properties)

	icon := ""
	if page.Icon != nil {
		if page.Icon.Type == "emoji" {
			icon = page.Icon.Emoji
		} else if page.Icon.File != nil {
			icon = page.Icon.File.URL
		} else if page.Icon.External != nil {
			icon = page.Icon.External.URL
		}
	}

	cover := ""
	if page.Cover != nil {
		if page.Cover.File != nil {
			cover = page.Cover.File.URL
		} else if page.Cover.External != nil {
			cover = page.Cover.External.URL
		}
	}

	propertiesJSON, _ := json.Marshal(page.Properties)

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO notion_pages (
			user_id, notion_id, database_id, title, icon, cover,
			properties, url, archived, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (user_id, notion_id) DO UPDATE SET
			database_id = EXCLUDED.database_id,
			title = EXCLUDED.title,
			icon = EXCLUDED.icon,
			cover = EXCLUDED.cover,
			properties = EXCLUDED.properties,
			url = EXCLUDED.url,
			archived = EXCLUDED.archived,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, page.ID, databaseID, title, icon, cover, propertiesJSON, page.URL, page.Archived)

	return err
}

// extractTitle extracts the title from page properties.
func extractTitle(properties map[string]interface{}) string {
	for _, key := range []string{"title", "Title", "name", "Name"} {
		if prop, ok := properties[key]; ok {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if titleArr, ok := propMap["title"].([]interface{}); ok && len(titleArr) > 0 {
					if textObj, ok := titleArr[0].(map[string]interface{}); ok {
						if plainText, ok := textObj["plain_text"].(string); ok {
							return plainText
						}
					}
				}
			}
		}
	}
	return "Untitled"
}

// GetDatabaseTitle extracts the title from a NotionDatabaseAPI.
func GetDatabaseTitle(db *NotionDatabaseAPI) string {
	if len(db.Title) > 0 {
		return db.Title[0].PlainText
	}
	return "Untitled"
}

// GetPageTitle extracts the title from a NotionPageAPI.
func GetPageTitle(page *NotionPageAPI) string {
	for _, key := range []string{"title", "Title", "name", "Name"} {
		if prop, ok := page.Properties[key]; ok {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if titleArr, ok := propMap["title"].([]interface{}); ok && len(titleArr) > 0 {
					if textObj, ok := titleArr[0].(map[string]interface{}); ok {
						if plainText, ok := textObj["plain_text"].(string); ok {
							return plainText
						}
					}
				}
			}
		}
	}
	return "Untitled"
}
