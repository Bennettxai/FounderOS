package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// ========== SEARCH TOOLS ==========

// WebSearchTool performs web searches by delegating to WebSearchService.
type WebSearchTool struct {
	pool      *pgxpool.Pool
	userID    string
	searchSvc *services.WebSearchService
}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Search the web for current information. Use this when you need up-to-date information, facts, news, or data that might not be in your training data. Returns search results with titles, URLs, and snippets."
}
func (t *WebSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query. Be specific and use keywords for better results.",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 5, max: 10)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	if params.MaxResults <= 0 {
		params.MaxResults = 5
	}
	if params.MaxResults > 10 {
		params.MaxResults = 10
	}

	response, err := t.searchSvc.Search(ctx, params.Query, params.MaxResults)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(response.Results) == 0 {
		return fmt.Sprintf("No results found for: %s", params.Query), nil
	}

	// Format results as markdown.
	output := fmt.Sprintf("## Web Search Results for: \"%s\"\n\n", params.Query)
	for i, result := range response.Results {
		output += fmt.Sprintf("### %d. %s\n", i+1, result.Title)
		output += fmt.Sprintf("**URL:** %s\n", result.URL)
		output += fmt.Sprintf("%s\n\n", result.Snippet)
	}

	return output, nil
}
