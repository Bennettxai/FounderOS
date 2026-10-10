package notion

import "time"

// Database represents a Notion database.
type Database struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	NotionID    string                 `json:"notion_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Cover       string                 `json:"cover,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	URL         string                 `json:"url,omitempty"`
	SyncedAt    *time.Time             `json:"synced_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Page represents a Notion page or database entry.
type Page struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	NotionID   string                 `json:"notion_id"`
	DatabaseID string                 `json:"database_id,omitempty"`
	Title      string                 `json:"title"`
	Icon       string                 `json:"icon,omitempty"`
	Cover      string                 `json:"cover,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Archived   bool                   `json:"archived"`
	SyncedAt   *time.Time             `json:"synced_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// NotionDatabase represents a database from the Notion API.
type NotionDatabase struct {
	Object      string                 `json:"object"`
	ID          string                 `json:"id"`
	Title       []Text                 `json:"title"`
	Description []Text                 `json:"description"`
	Icon        *Icon                  `json:"icon"`
	Cover       *Cover                 `json:"cover"`
	Properties  map[string]interface{} `json:"properties"`
	URL         string                 `json:"url"`
	Archived    bool                   `json:"archived"`
}

// Text represents a Notion rich text element.
type Text struct {
	Type      string `json:"type"`
	PlainText string `json:"plain_text"`
}

// Icon represents a Notion icon.
type Icon struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji,omitempty"`
	File  *struct {
		URL string `json:"url"`
	} `json:"file,omitempty"`
	External *struct {
		URL string `json:"url"`
	} `json:"external,omitempty"`
}

// Cover represents a Notion cover image.
type Cover struct {
	Type string `json:"type"`
	File *struct {
		URL string `json:"url"`
	} `json:"file,omitempty"`
	External *struct {
		URL string `json:"url"`
	} `json:"external,omitempty"`
}

// NotionPage represents a page from the Notion API.
type NotionPage struct {
	Object     string                 `json:"object"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
	Icon       *Icon                  `json:"icon"`
	Cover      *Cover                 `json:"cover"`
	URL        string                 `json:"url"`
	Archived   bool                   `json:"archived"`
}

// NotionDatabaseAPI represents a database from the Notion API (for MCP).
type NotionDatabaseAPI struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"`
	Title          []Text                 `json:"title"`
	URL            string                 `json:"url"`
	Properties     map[string]interface{} `json:"properties"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
}

// NotionPageAPI represents a page from the Notion API (for MCP).
type NotionPageAPI struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"`
	URL            string                 `json:"url"`
	Properties     map[string]interface{} `json:"properties"`
	Parent         interface{}            `json:"parent"`
	Archived       bool                   `json:"archived"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
}

// NotionQueryResponse represents a query response from Notion API.
type NotionQueryResponse struct {
	Results    []NotionPageAPI `json:"results"`
	HasMore    bool            `json:"has_more"`
	NextCursor *string         `json:"next_cursor"`
}

// NotionSearchResponse represents a search response from Notion API.
type NotionSearchResponse struct {
	Results    []interface{} `json:"results"`
	HasMore    bool          `json:"has_more"`
	NextCursor *string       `json:"next_cursor"`
}

// SyncDatabasesResult represents the result of a database sync.
type SyncDatabasesResult struct {
	TotalDatabases  int `json:"total_databases"`
	SyncedDatabases int `json:"synced_databases"`
	FailedDatabases int `json:"failed_databases"`
}

// SyncPagesResult represents the result of a page sync.
type SyncPagesResult struct {
	TotalPages  int `json:"total_pages"`
	SyncedPages int `json:"synced_pages"`
	FailedPages int `json:"failed_pages"`
}
