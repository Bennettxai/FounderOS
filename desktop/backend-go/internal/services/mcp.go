package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/integrations/notion"
	"github.com/rhl/businessos-backend/internal/integrations/slack"
	"github.com/rhl/businessos-backend/internal/security"
)

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Source      string                 `json:"source"`
}

type MCPService struct {
	pool            *pgxpool.Pool
	userID          string
	calendarService *google.CalendarService
	slackService    *slack.ChannelService
	notionService   *notion.DatabaseService
}

func NewMCPService(pool *pgxpool.Pool, userID string, calendarService *google.CalendarService, slackService *slack.ChannelService, notionService *notion.DatabaseService) *MCPService {
	return &MCPService{
		pool:            pool,
		userID:          userID,
		calendarService: calendarService,
		slackService:    slackService,
		notionService:   notionService,
	}
}

func (m *MCPService) GetBuiltinTools() []MCPTool {
	tools := []MCPTool{
		{
			Name:        "search_conversations",
			Description: "Search through past conversations",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results (default 10)",
					},
				},
				"required": []string{"query"},
			},
			Source: "builtin",
		},
		{
			Name:        "get_project_context",
			Description: "Get project details and context",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
				},
				"required": []string{"project_name"},
			},
			Source: "builtin",
		},
		{
			Name:        "create_artifact",
			Description: "Create an artifact (code, document, etc.)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Artifact title",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Artifact type",
						"enum":        []string{"code", "document", "markdown", "html", "proposal", "sop", "framework", "report", "plan"},
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Artifact content",
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Programming language (for code)",
					},
				},
				"required": []string{"title", "type", "content"},
			},
			Source: "builtin",
		},
		{
			Name:        "add_to_daily_log",
			Description: "Add an entry to the daily log",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Log entry content",
					},
				},
				"required": []string{"content"},
			},
			Source: "builtin",
		},
		{
			Name:        "get_context_profile",
			Description: "Get a context profile (person, business, project)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the context",
					},
				},
				"required": []string{"name"},
			},
			Source: "builtin",
		},
		{
			Name:        "list_resources",
			Description: "List available resources (contexts, projects, etc.)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"resource_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of resource to list",
						"enum":        []string{"contexts", "projects", "artifacts", "team"},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results",
					},
				},
			},
			Source: "builtin",
		},
	}

	tools = append(tools, GetCalendarTools()...)
	tools = append(tools, GetSlackTools()...)
	tools = append(tools, GetNotionTools()...)

	return tools
}

func (m *MCPService) GetAllTools() []MCPTool {
	tools := m.GetBuiltinTools()

	// Load dynamic MCP server tools
	dynamicTools := m.getDynamicMCPTools(context.Background())
	tools = append(tools, dynamicTools...)

	return tools
}

// getDynamicMCPTools loads the user's enabled MCP servers and returns their
// cached tools as MCPTool entries with "mcp:<server>" source attribution.
func (m *MCPService) getDynamicMCPTools(ctx context.Context) []MCPTool {
	if m.pool == nil || m.userID == "" {
		return nil
	}

	queries := sqlc.New(m.pool)
	servers, err := queries.ListEnabledMCPServers(ctx, m.userID)
	if err != nil {
		return nil
	}

	var tools []MCPTool
	for _, srv := range servers {
		if srv.ToolsCache == nil {
			continue
		}
		var cachedTools []MCPClientTool
		if err := json.Unmarshal(srv.ToolsCache, &cachedTools); err != nil {
			continue
		}
		for _, ct := range cachedTools {
			tools = append(tools, MCPTool{
				Name:        fmt.Sprintf("%s.%s", srv.Name, ct.Name),
				Description: ct.Description,
				Parameters:  ct.InputSchema,
				Source:       fmt.Sprintf("mcp:%s", srv.Name),
			})
		}
	}
	return tools
}

func (m *MCPService) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	if IsCalendarTool(toolName) {
		return m.ExecuteCalendarTool(ctx, toolName, arguments)
	}

	if IsSlackTool(toolName) {
		return m.ExecuteSlackTool(ctx, toolName, arguments)
	}

	if IsNotionTool(toolName) {
		return m.ExecuteNotionTool(ctx, m.userID, toolName, arguments)
	}

	// Route namespaced MCP tools (e.g. "github.create_issue") to the correct server
	if parts := strings.SplitN(toolName, ".", 2); len(parts) == 2 {
		result, err := m.executeDynamicMCPTool(ctx, parts[0], parts[1], arguments)
		if err == nil {
			return result, nil
		}
		// If the server wasn't found, fall through to built-in tools
		if !strings.Contains(err.Error(), "MCP server not found") {
			return nil, err
		}
	}

	queries := sqlc.New(m.pool)

	switch toolName {
	case "search_conversations":
		query, _ := arguments["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query is required")
		}
		conversations, err := queries.SearchConversations(ctx, sqlc.SearchConversationsParams{
			UserID:  m.userID,
			Column2: &query,
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"conversations": conversations}, nil

	case "get_project_context":
		projectName, _ := arguments["project_name"].(string)
		if projectName == "" {
			return nil, fmt.Errorf("project_name is required")
		}
		projects, err := queries.ListProjects(ctx, sqlc.ListProjectsParams{
			UserID: m.userID,
		})
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Name == projectName {
				return p, nil
			}
		}
		return nil, fmt.Errorf("project not found: %s", projectName)

	case "create_artifact":
		title, _ := arguments["title"].(string)
		content, _ := arguments["content"].(string)
		artifactType, _ := arguments["type"].(string)
		language, _ := arguments["language"].(string)

		if title == "" || content == "" || artifactType == "" {
			return nil, fmt.Errorf("title, content, and type are required")
		}

		var lang *string
		if language != "" {
			lang = &language
		}

		typeMap := map[string]sqlc.Artifacttype{
			"code":      sqlc.ArtifacttypeCODE,
			"document":  sqlc.ArtifacttypeDOCUMENT,
			"markdown":  sqlc.ArtifacttypeMARKDOWN,
			"html":      sqlc.ArtifacttypeHTML,
			"react":     sqlc.ArtifacttypeREACT,
			"svg":       sqlc.ArtifacttypeSVG,
			"proposal":  sqlc.ArtifacttypeDOCUMENT,
			"sop":       sqlc.ArtifacttypeDOCUMENT,
			"framework": sqlc.ArtifacttypeDOCUMENT,
			"report":    sqlc.ArtifacttypeDOCUMENT,
			"plan":      sqlc.ArtifacttypeDOCUMENT,
		}

		aType, ok := typeMap[artifactType]
		if !ok {
			aType = sqlc.ArtifacttypeDOCUMENT
		}

		artifact, err := queries.CreateArtifact(ctx, sqlc.CreateArtifactParams{
			UserID:   m.userID,
			Title:    title,
			Type:     aType,
			Content:  content,
			Language: lang,
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":      uuid.UUID(artifact.ID.Bytes).String(),
			"title":   artifact.Title,
			"type":    artifact.Type,
			"created": true,
		}, nil

	case "add_to_daily_log":
		content, _ := arguments["content"].(string)
		if content == "" {
			return nil, fmt.Errorf("content is required")
		}

		log, err := queries.GetTodayLog(ctx, m.userID)
		if err != nil {
			log, err = queries.CreateDailyLog(ctx, sqlc.CreateDailyLogParams{
				UserID:  m.userID,
				Content: content,
			})
			if err != nil {
				return nil, err
			}
		} else {
			newContent := log.Content + "\n\n" + content
			log, err = queries.UpdateDailyLog(ctx, sqlc.UpdateDailyLogParams{
				ID:      log.ID,
				Content: newContent,
			})
			if err != nil {
				return nil, err
			}
		}
		return map[string]interface{}{"success": true, "log_id": uuid.UUID(log.ID.Bytes).String()}, nil

	case "get_context_profile":
		name, _ := arguments["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("name is required")
		}

		contexts, err := queries.ListContexts(ctx, sqlc.ListContextsParams{
			UserID: m.userID,
			Search: &name,
		})
		if err != nil {
			return nil, err
		}
		if len(contexts) > 0 {
			return contexts[0], nil
		}
		return nil, fmt.Errorf("context not found: %s", name)

	case "list_resources":
		resourceType, _ := arguments["resource_type"].(string)
		limit := 10
		if l, ok := arguments["limit"].(float64); ok {
			limit = int(l)
		}

		switch resourceType {
		case "contexts":
			contexts, err := queries.ListContexts(ctx, sqlc.ListContextsParams{
				UserID: m.userID,
			})
			if err != nil {
				return nil, err
			}
			if len(contexts) > limit {
				contexts = contexts[:limit]
			}
			return map[string]interface{}{"contexts": contexts}, nil

		case "projects":
			projects, err := queries.ListProjects(ctx, sqlc.ListProjectsParams{
				UserID: m.userID,
			})
			if err != nil {
				return nil, err
			}
			if len(projects) > limit {
				projects = projects[:limit]
			}
			return map[string]interface{}{"projects": projects}, nil

		case "artifacts":
			artifacts, err := queries.ListArtifacts(ctx, sqlc.ListArtifactsParams{
				UserID: m.userID,
			})
			if err != nil {
				return nil, err
			}
			if len(artifacts) > limit {
				artifacts = artifacts[:limit]
			}
			return map[string]interface{}{"artifacts": artifacts}, nil

		case "team":
			members, err := queries.ListTeamMembers(ctx, m.userID)
			if err != nil {
				return nil, err
			}
			if len(members) > limit {
				members = members[:limit]
			}
			return map[string]interface{}{"team_members": members}, nil

		default:
			return nil, fmt.Errorf("unknown resource type: %s", resourceType)
		}

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

type ToolResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (m *MCPService) ExecuteToolJSON(ctx context.Context, toolName string, argumentsJSON string) ([]byte, error) {
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &arguments); err != nil {
		return json.Marshal(ToolResponse{Success: false, Error: "invalid arguments JSON"})
	}

	result, err := m.ExecuteTool(ctx, toolName, arguments)
	if err != nil {
		return json.Marshal(ToolResponse{Success: false, Error: err.Error()})
	}

	return json.Marshal(ToolResponse{Success: true, Result: result})
}

func pgtypeToUUID(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return uuid.UUID(p.Bytes)
}

// executeDynamicMCPTool finds the user's MCP server by name and executes a tool on it.
func (m *MCPService) executeDynamicMCPTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (interface{}, error) {
	if m.pool == nil || m.userID == "" {
		return nil, fmt.Errorf("MCP server not found: %s", serverName)
	}

	queries := sqlc.New(m.pool)
	servers, err := queries.ListEnabledMCPServers(ctx, m.userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query MCP servers: %w", err)
	}

	for _, srv := range servers {
		if srv.Name != serverName {
			continue
		}

		// Decrypt auth token
		var authToken string
		if srv.AuthTokenEnc != nil && *srv.AuthTokenEnc != "" {
			enc := security.GetGlobalEncryption()
			if enc != nil {
				decrypted, err := enc.Decrypt(*srv.AuthTokenEnc)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt MCP server credentials: %w", err)
				}
				authToken = decrypted
			}
		}

		// Parse custom headers
		var headers map[string]string
		if srv.CustomHeaders != nil {
			_ = json.Unmarshal(srv.CustomHeaders, &headers)
		}

		client := NewMCPClient(srv.ServerUrl, srv.AuthType, authToken, headers)
		return client.ExecuteTool(ctx, toolName, arguments)
	}

	return nil, fmt.Errorf("MCP server not found: %s", serverName)
}
