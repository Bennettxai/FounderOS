package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/integrations/notion"
)

// Sentinel errors for proper error handling
var ErrNotionTokenExpired = errors.New("notion token expired")

func IsNotionTool(toolName string) bool {
	return strings.HasPrefix(toolName, "notion_")
}

// GetNotionTools returns the MCP tool definitions for Notion integration
func GetNotionTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "notion_list_databases",
			Description: "List all Notion databases the integration has access to. Returns database names, IDs, and URLs.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cursor": map[string]interface{}{
						"type":        "string",
						"description": "Cursor for pagination. Use the next_cursor from a previous response to get more results.",
					},
				},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_get_database",
			Description: "Get detailed information about a specific Notion database, including its schema/properties.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the database to retrieve. Required.",
					},
				},
				"required": []string{"database_id"},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_query_database",
			Description: "Query a Notion database to get its pages/entries. Supports filtering and sorting.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the database to query. Required.",
					},
					"page_size": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return (default 10, max 100)",
					},
					"cursor": map[string]interface{}{
						"type":        "string",
						"description": "Cursor for pagination",
					},
				},
				"required": []string{"database_id"},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_get_page",
			Description: "Get a specific Notion page by ID. Returns page properties and metadata.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the page to retrieve. Required.",
					},
				},
				"required": []string{"page_id"},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_create_page",
			Description: "Create a new page in a Notion database. Provide the database ID and property values.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"database_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the database to create the page in. Required.",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "The title of the new page. Required.",
					},
					"properties": map[string]interface{}{
						"type":        "object",
						"description": "Additional properties for the page. Keys are property names, values depend on property type.",
					},
				},
				"required": []string{"database_id", "title"},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_update_page",
			Description: "Update an existing Notion page's properties.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the page to update. Required.",
					},
					"properties": map[string]interface{}{
						"type":        "object",
						"description": "Properties to update. Keys are property names, values depend on property type. Required.",
					},
				},
				"required": []string{"page_id", "properties"},
			},
			Source: "builtin",
		},
		{
			Name:        "notion_search",
			Description: "Search for pages and databases in Notion. Can filter by type (page or database).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query text",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Filter results by type: 'page' or 'database'",
						"enum":        []string{"page", "database"},
					},
					"page_size": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return (default 10, max 100)",
					},
					"cursor": map[string]interface{}{
						"type":        "string",
						"description": "Cursor for pagination",
					},
				},
			},
			Source: "builtin",
		},
	}
}

// ExecuteNotionTool executes a Notion MCP tool
func (s *MCPService) ExecuteNotionTool(ctx context.Context, userID string, toolName string, arguments map[string]interface{}) (interface{}, error) {
	// Check if Notion is connected
	_, err := s.notionService.GetToken(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Notion is not connected. Please connect your Notion workspace in Settings first.")
		}
		return nil, fmt.Errorf("Notion access error: %w", err)
	}

	switch toolName {
	case "notion_list_databases":
		return s.executeNotionListDatabases(ctx, userID, arguments)
	case "notion_get_database":
		return s.executeNotionGetDatabase(ctx, userID, arguments)
	case "notion_query_database":
		return s.executeNotionQueryDatabase(ctx, userID, arguments)
	case "notion_get_page":
		return s.executeNotionGetPage(ctx, userID, arguments)
	case "notion_create_page":
		return s.executeNotionCreatePage(ctx, userID, arguments)
	case "notion_update_page":
		return s.executeNotionUpdatePage(ctx, userID, arguments)
	case "notion_search":
		return s.executeNotionSearch(ctx, userID, arguments)
	default:
		return nil, fmt.Errorf("unknown Notion tool: %s", toolName)
	}
}

func (s *MCPService) executeNotionListDatabases(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	cursor := ""
	if c, ok := arguments["cursor"].(string); ok {
		cursor = c
	}

	databases, nextCursor, hasMore, err := s.notionService.ListDatabases(ctx, userID, 100, cursor)
	if err != nil {
		// Check for token expiration using proper sentinel error
		if errors.Is(err, ErrNotionTokenExpired) {
			return nil, fmt.Errorf("Notion access expired. Please reconnect your Notion workspace in Settings.")
		}
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	// Format response
	var result []map[string]interface{}
	for _, db := range databases {
		result = append(result, map[string]interface{}{
			"id":               db.ID,
			"title":            notion.GetDatabaseTitle(&db),
			"url":              db.URL,
			"created_time":     db.CreatedTime,
			"last_edited_time": db.LastEditedTime,
		})
	}

	return map[string]interface{}{
		"databases":   result,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"count":       len(result),
	}, nil
}

func (s *MCPService) executeNotionGetDatabase(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	databaseID, _ := arguments["database_id"].(string)
	if databaseID == "" {
		return nil, fmt.Errorf("database_id is required")
	}

	database, err := s.notionService.GetDatabase(ctx, userID, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return map[string]interface{}{
		"id":               database.ID,
		"title":            notion.GetDatabaseTitle(database),
		"url":              database.URL,
		"properties":       database.Properties,
		"created_time":     database.CreatedTime,
		"last_edited_time": database.LastEditedTime,
	}, nil
}

func (s *MCPService) executeNotionQueryDatabase(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	databaseID, _ := arguments["database_id"].(string)
	if databaseID == "" {
		return nil, fmt.Errorf("database_id is required")
	}

	pageSize := 10
	if ps, ok := arguments["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	cursor := ""
	if c, ok := arguments["cursor"].(string); ok {
		cursor = c
	}

	queryResp, err := s.notionService.QueryDatabase(ctx, userID, databaseID, pageSize, cursor, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	// Format response
	var pages []map[string]interface{}
	for _, page := range queryResp.Results {
		pages = append(pages, map[string]interface{}{
			"id":               page.ID,
			"title":            notion.GetPageTitle(&page),
			"url":              page.URL,
			"properties":       page.Properties,
			"created_time":     page.CreatedTime,
			"last_edited_time": page.LastEditedTime,
			"archived":         page.Archived,
		})
	}

	nextCursor := ""
	if queryResp.NextCursor != nil {
		nextCursor = *queryResp.NextCursor
	}

	return map[string]interface{}{
		"pages":       pages,
		"next_cursor": nextCursor,
		"has_more":    queryResp.HasMore,
		"count":       len(pages),
	}, nil
}

func (s *MCPService) executeNotionGetPage(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	pageID, _ := arguments["page_id"].(string)
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	page, err := s.notionService.GetPage(ctx, userID, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	return map[string]interface{}{
		"id":               page.ID,
		"title":            notion.GetPageTitle(page),
		"url":              page.URL,
		"properties":       page.Properties,
		"created_time":     page.CreatedTime,
		"last_edited_time": page.LastEditedTime,
		"archived":         page.Archived,
		"parent":           page.Parent,
	}, nil
}

func (s *MCPService) executeNotionCreatePage(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	databaseID, _ := arguments["database_id"].(string)
	if databaseID == "" {
		return nil, fmt.Errorf("database_id is required")
	}

	title, _ := arguments["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
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

	// Merge additional properties if provided
	if additionalProps, ok := arguments["properties"].(map[string]interface{}); ok {
		for key, value := range additionalProps {
			properties[key] = value
		}
	}

	page, err := s.notionService.CreatePage(ctx, userID, databaseID, properties)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	return map[string]interface{}{
		"id":           page.ID,
		"title":        notion.GetPageTitle(page),
		"url":          page.URL,
		"created_time": page.CreatedTime,
		"message":      "Page created successfully",
	}, nil
}

func (s *MCPService) executeNotionUpdatePage(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	pageID, _ := arguments["page_id"].(string)
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	properties, ok := arguments["properties"].(map[string]interface{})
	if !ok || len(properties) == 0 {
		return nil, fmt.Errorf("properties is required")
	}

	page, err := s.notionService.UpdatePage(ctx, userID, pageID, properties)
	if err != nil {
		return nil, fmt.Errorf("failed to update page: %w", err)
	}

	return map[string]interface{}{
		"id":               page.ID,
		"title":            notion.GetPageTitle(page),
		"url":              page.URL,
		"last_edited_time": page.LastEditedTime,
		"message":          "Page updated successfully",
	}, nil
}

func (s *MCPService) executeNotionSearch(ctx context.Context, userID string, arguments map[string]interface{}) (interface{}, error) {
	query := ""
	if q, ok := arguments["query"].(string); ok {
		query = q
	}

	filter := ""
	if f, ok := arguments["filter"].(string); ok {
		filter = f
	}

	pageSize := 10
	if ps, ok := arguments["page_size"].(float64); ok {
		pageSize = int(ps)
		if pageSize > 100 {
			pageSize = 100
		}
	}

	cursor := ""
	if c, ok := arguments["cursor"].(string); ok {
		cursor = c
	}

	searchResp, err := s.notionService.Search(ctx, userID, query, filter, pageSize, cursor)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	nextCursor := ""
	if searchResp.NextCursor != nil {
		nextCursor = *searchResp.NextCursor
	}

	return map[string]interface{}{
		"results":     searchResp.Results,
		"next_cursor": nextCursor,
		"has_more":    searchResp.HasMore,
		"count":       len(searchResp.Results),
	}, nil
}
