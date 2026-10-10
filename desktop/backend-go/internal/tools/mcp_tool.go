package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/security"
	"github.com/rhl/businessos-backend/internal/services"
)

// MCPToolAdapter wraps a remote MCP tool as an AgentTool so it can be used
// by agents via the standard tool registry.
type MCPToolAdapter struct {
	// Tool metadata (from the MCP server's tools/list response)
	name        string
	description string
	inputSchema map[string]interface{}

	// Connection details for calling the remote MCP server
	serverURL string
	authType  string
	authToken string // decrypted
	headers   map[string]string
	source    string // "mcp:<server-name>" for attribution
}

func (t *MCPToolAdapter) Name() string                       { return t.name }
func (t *MCPToolAdapter) Description() string                { return t.description }
func (t *MCPToolAdapter) InputSchema() map[string]interface{} { return t.inputSchema }

// Execute calls the remote MCP server to run this tool
func (t *MCPToolAdapter) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	client := services.NewMCPClient(t.serverURL, t.authType, t.authToken, t.headers)
	result, err := client.ExecuteTool(ctx, t.name, args)
	if err != nil {
		slog.Warn("MCP tool execution failed",
			"tool", t.name,
			"source", t.source,
			"error", err,
		)
		return "", fmt.Errorf("MCP tool %s failed: %w", t.name, err)
	}

	out, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result), nil
	}
	return string(out), nil
}

// RegisterMCPTools loads the user's enabled MCP servers from the database and
// registers their cached tools into the registry. Tools are namespaced as
// "<server_name>.<tool_name>" to avoid collisions with built-in tools.
//
// This is best-effort: if a server has no cached tools or decryption fails, it
// is skipped with a warning log. The agent can still use all built-in tools.
func (r *AgentToolRegistry) RegisterMCPTools(ctx context.Context) {
	if r.pool == nil || r.userID == "" {
		return
	}

	queries := sqlc.New(r.pool)
	servers, err := queries.ListEnabledMCPServers(ctx, r.userID)
	if err != nil {
		slog.Warn("Failed to load MCP servers for tool registry", "user_id", r.userID, "error", err)
		return
	}

	enc := security.GetGlobalEncryption()
	registered := 0

	for _, srv := range servers {
		// Parse cached tools
		if srv.ToolsCache == nil {
			continue
		}
		var cachedTools []services.MCPClientTool
		if err := json.Unmarshal(srv.ToolsCache, &cachedTools); err != nil {
			slog.Warn("Failed to parse MCP tools cache", "server", srv.Name, "error", err)
			continue
		}
		if len(cachedTools) == 0 {
			continue
		}

		// Decrypt auth token if present
		var authToken string
		if srv.AuthTokenEnc != nil && *srv.AuthTokenEnc != "" && enc != nil {
			decrypted, err := enc.Decrypt(*srv.AuthTokenEnc)
			if err != nil {
				slog.Warn("Failed to decrypt MCP server token", "server", srv.Name, "error", err)
				continue
			}
			authToken = decrypted
		}

		// Parse custom headers
		var headers map[string]string
		if srv.CustomHeaders != nil {
			_ = json.Unmarshal(srv.CustomHeaders, &headers)
		}

		sourceName := fmt.Sprintf("mcp:%s", srv.Name)

		for _, tool := range cachedTools {
			namespacedName := fmt.Sprintf("%s.%s", srv.Name, tool.Name)

			// Don't override built-in tools
			if _, exists := r.tools[namespacedName]; exists {
				continue
			}

			desc := tool.Description
			if desc == "" {
				desc = fmt.Sprintf("Tool from MCP server %s", srv.Name)
			}

			r.tools[namespacedName] = &MCPToolAdapter{
				name:        namespacedName,
				description: desc,
				inputSchema: tool.InputSchema,
				serverURL:   srv.ServerUrl,
				authType:    srv.AuthType,
				authToken:   authToken,
				headers:     headers,
				source:      sourceName,
			}
			registered++
		}
	}

	if registered > 0 {
		slog.Info("Registered MCP tools", "user_id", r.userID, "tool_count", registered, "server_count", len(servers))
	}
}
