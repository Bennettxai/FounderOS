package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ========== CONTEXT TOOLS (for AI Agent tree navigation) ==========

// ========== TreeSearchTool ==========

// TreeSearchTool searches the context tree (memories, documents, artifacts, voice notes,
// conversation summaries) by delegating to ContextService.
type TreeSearchTool struct {
	userID         string
	contextService ContextServiceInterface
}

func (t *TreeSearchTool) Name() string { return "tree_search" }
func (t *TreeSearchTool) Description() string {
	return "Search through the user's knowledge base including memories, documents, and artifacts. Use this to find relevant context before answering questions. Supports title search, content search, and semantic (meaning-based) search."
}
func (t *TreeSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query - what you're looking for",
			},
			"search_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"title", "content", "semantic"},
				"description": "Type of search: 'title' for name matching, 'content' for text search, 'semantic' for meaning-based search (default: semantic)",
			},
			"entity_types": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
					"enum": []string{"memories", "documents", "artifacts", "contexts"},
				},
				"description": "Types of items to search (default: all types)",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10, max: 25)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *TreeSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		Query       string   `json:"query"`
		SearchType  string   `json:"search_type"`
		EntityTypes []string `json:"entity_types"`
		MaxResults  int      `json:"max_results"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if params.SearchType == "" {
		params.SearchType = "semantic"
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 10
	}
	if params.MaxResults > 25 {
		params.MaxResults = 25
	}

	results, err := t.contextService.SearchTree(ctx, t.userID, TreeSearchParams{
		Query:       params.Query,
		SearchType:  params.SearchType,
		EntityTypes: params.EntityTypes,
		MaxResults:  params.MaxResults,
	})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results found for: \"%s\"", params.Query), nil
	}

	// Format results as markdown
	var output strings.Builder
	output.WriteString(fmt.Sprintf("## Search Results for: \"%s\" (%s search)\n\n", params.Query, params.SearchType))
	output.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results)))

	for i, r := range results {
		output.WriteString(fmt.Sprintf("### %d. %s\n", i+1, r.Title))
		output.WriteString(fmt.Sprintf("- **Type:** %s\n", r.Type))
		output.WriteString(fmt.Sprintf("- **ID:** %s\n", r.ID.String()))
		if r.Summary != "" {
			output.WriteString(fmt.Sprintf("- **Summary:** %s\n", r.Summary))
		}
		if params.SearchType == "semantic" {
			output.WriteString(fmt.Sprintf("- **Relevance:** %.2f\n", r.RelevanceScore))
		}
		output.WriteString("\n")
	}

	output.WriteString("\n*Use `load_context` tool with an ID to load the full content of any item.*")

	return output.String(), nil
}

// ========== BrowseTreeTool ==========

// BrowseTreeTool browses the context tree hierarchy
type BrowseTreeTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *BrowseTreeTool) Name() string { return "browse_tree" }
func (t *BrowseTreeTool) Description() string {
	return "Browse the user's knowledge tree to see what projects, memories, documents, and artifacts are available. Use this to understand the structure of available context before searching."
}
func (t *BrowseTreeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Filter to a specific project UUID",
			},
			"show_counts": map[string]interface{}{
				"type":        "boolean",
				"description": "Show item counts for each category (default: true)",
			},
		},
	}
}

func (t *BrowseTreeTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ProjectID  string `json:"project_id"`
		ShowCounts bool   `json:"show_counts"`
	}
	params.ShowCounts = true // default
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	var output strings.Builder
	output.WriteString("## Knowledge Tree\n\n")

	// Get overall statistics
	var memoryCount, docCount, artifactCount, projectCount int
	t.pool.QueryRow(ctx, `SELECT COUNT(*) FROM memories WHERE user_id = $1 AND is_active = true`, t.userID).Scan(&memoryCount)
	t.pool.QueryRow(ctx, `SELECT COUNT(*) FROM uploaded_documents WHERE user_id = $1`, t.userID).Scan(&docCount)
	t.pool.QueryRow(ctx, `SELECT COUNT(*) FROM artifacts WHERE user_id = $1`, t.userID).Scan(&artifactCount)
	t.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE user_id = $1 AND is_archived = false`, t.userID).Scan(&projectCount)

	if params.ShowCounts {
		output.WriteString("### Overview\n")
		output.WriteString(fmt.Sprintf("- **Projects:** %d\n", projectCount))
		output.WriteString(fmt.Sprintf("- **Memories:** %d\n", memoryCount))
		output.WriteString(fmt.Sprintf("- **Documents:** %d\n", docCount))
		output.WriteString(fmt.Sprintf("- **Artifacts:** %d\n", artifactCount))
		output.WriteString("\n")
	}

	// List projects
	output.WriteString("### Projects\n")
	query := `SELECT id, name, COALESCE(description, ''), status FROM projects WHERE user_id = $1 AND is_archived = false ORDER BY updated_at DESC LIMIT 20`
	args := []interface{}{t.userID}

	if params.ProjectID != "" {
		if projectUUID, err := uuid.Parse(params.ProjectID); err == nil {
			query = `SELECT id, name, COALESCE(description, ''), status FROM projects WHERE user_id = $1 AND id = $2`
			args = append(args, projectUUID)
		}
	}

	rows, err := t.pool.Query(ctx, query, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			var name, description, status string
			if rows.Scan(&id, &name, &description, &status) == nil {
				output.WriteString(fmt.Sprintf("- **%s** [%s] (ID: %s)\n", name, status, id.String()))
				if description != "" {
					output.WriteString(fmt.Sprintf("  %s\n", description))
				}
			}
		}
	}

	if projectCount == 0 {
		output.WriteString("No projects found.\n")
	}

	// Memory types breakdown
	output.WriteString("\n### Memory Types\n")
	typeRows, _ := t.pool.Query(ctx, `
		SELECT memory_type, COUNT(*)
		FROM memories
		WHERE user_id = $1 AND is_active = true
		GROUP BY memory_type
		ORDER BY COUNT(*) DESC
	`, t.userID)
	if typeRows != nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var memType string
			var count int
			if typeRows.Scan(&memType, &count) == nil {
				output.WriteString(fmt.Sprintf("- %s: %d\n", memType, count))
			}
		}
	}

	output.WriteString("\n*Use `tree_search` to find specific items, or `load_context` to load an item by ID.*")

	return output.String(), nil
}

// ========== LoadContextTool ==========

// LoadContextTool loads a specific context item by ID
type LoadContextTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *LoadContextTool) Name() string { return "load_context" }
func (t *LoadContextTool) Description() string {
	return "Load the full content of a specific memory, document, or artifact by its ID. Use this after finding items with tree_search or browse_tree to get the complete content."
}
func (t *LoadContextTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"item_id": map[string]interface{}{
				"type":        "string",
				"description": "The UUID of the item to load",
			},
			"item_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"memory", "document", "artifact"},
				"description": "The type of item to load",
			},
		},
		"required": []string{"item_id", "item_type"},
	}
}

func (t *LoadContextTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params struct {
		ItemID   string `json:"item_id"`
		ItemType string `json:"item_type"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.ItemID == "" {
		return "", fmt.Errorf("item_id is required")
	}
	if params.ItemType == "" {
		return "", fmt.Errorf("item_type is required")
	}

	itemUUID, err := uuid.Parse(params.ItemID)
	if err != nil {
		return "", fmt.Errorf("invalid item_id: %w", err)
	}

	var output strings.Builder

	switch params.ItemType {
	case "memory":
		var title, content, memType, summary string
		var importance int
		var createdAt time.Time
		err := t.pool.QueryRow(ctx, `
			SELECT COALESCE(title, 'Untitled'), content, memory_type, COALESCE(summary, ''), importance_score, created_at
			FROM memories
			WHERE id = $1 AND user_id = $2
		`, itemUUID, t.userID).Scan(&title, &content, &memType, &summary, &importance, &createdAt)
		if err != nil {
			return "", fmt.Errorf("memory not found: %w", err)
		}

		output.WriteString(fmt.Sprintf("## Memory: %s\n\n", title))
		output.WriteString(fmt.Sprintf("- **Type:** %s\n", memType))
		output.WriteString(fmt.Sprintf("- **Importance:** %d/10\n", importance))
		output.WriteString(fmt.Sprintf("- **Created:** %s\n", createdAt.Format("2006-01-02 15:04")))
		if summary != "" {
			output.WriteString(fmt.Sprintf("- **Summary:** %s\n", summary))
		}
		output.WriteString("\n### Content\n\n")
		output.WriteString(content)

	case "document":
		var displayName, filename, docType, description, extractedText string
		var createdAt time.Time
		err := t.pool.QueryRow(ctx, `
			SELECT COALESCE(display_name, filename), filename, COALESCE(document_type, 'document'),
			       COALESCE(description, ''), COALESCE(extracted_text, ''), created_at
			FROM uploaded_documents
			WHERE id = $1 AND user_id = $2
		`, itemUUID, t.userID).Scan(&displayName, &filename, &docType, &description, &extractedText, &createdAt)
		if err != nil {
			return "", fmt.Errorf("document not found: %w", err)
		}

		output.WriteString(fmt.Sprintf("## Document: %s\n\n", displayName))
		output.WriteString(fmt.Sprintf("- **Filename:** %s\n", filename))
		output.WriteString(fmt.Sprintf("- **Type:** %s\n", docType))
		output.WriteString(fmt.Sprintf("- **Uploaded:** %s\n", createdAt.Format("2006-01-02 15:04")))
		if description != "" {
			output.WriteString(fmt.Sprintf("- **Description:** %s\n", description))
		}
		output.WriteString("\n### Extracted Content\n\n")
		if extractedText != "" {
			// Limit content to avoid token overflow
			if len(extractedText) > 10000 {
				output.WriteString(extractedText[:10000])
				output.WriteString("\n\n*[Content truncated - document is very large]*")
			} else {
				output.WriteString(extractedText)
			}
		} else {
			output.WriteString("*No text extracted from this document*")
		}

	case "artifact":
		var title, content, artType string
		var createdAt time.Time
		err := t.pool.QueryRow(ctx, `
			SELECT title, content, type, created_at
			FROM artifacts
			WHERE id = $1 AND user_id = $2
		`, itemUUID, t.userID).Scan(&title, &content, &artType, &createdAt)
		if err != nil {
			return "", fmt.Errorf("artifact not found: %w", err)
		}

		output.WriteString(fmt.Sprintf("## Artifact: %s\n\n", title))
		output.WriteString(fmt.Sprintf("- **Type:** %s\n", artType))
		output.WriteString(fmt.Sprintf("- **Created:** %s\n", createdAt.Format("2006-01-02 15:04")))
		output.WriteString("\n### Content\n\n")
		output.WriteString(content)

	default:
		return "", fmt.Errorf("unknown item_type: %s", params.ItemType)
	}

	return output.String(), nil
}
