package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/security"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// MCPServersHandler handles CRUD operations for user MCP server connections
type MCPServersHandler struct {
	pool *pgxpool.Pool
}

// NewMCPServersHandler creates a new MCPServersHandler
func NewMCPServersHandler(pool *pgxpool.Pool) *MCPServersHandler {
	return &MCPServersHandler{pool: pool}
}

// CreateMCPServerRequest is the request body for creating an MCP server connection
type CreateMCPServerRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	ServerURL   string            `json:"server_url" binding:"required"`
	Transport   string            `json:"transport"`
	AuthType    string            `json:"auth_type"`
	APIKey      string            `json:"api_key"`
	Headers     map[string]string `json:"headers"`
}

// UpdateMCPServerRequest is the request body for updating an MCP server connection
type UpdateMCPServerRequest struct {
	Name        *string            `json:"name"`
	Description *string            `json:"description"`
	ServerURL   *string            `json:"server_url"`
	Transport   *string            `json:"transport"`
	AuthType    *string            `json:"auth_type"`
	APIKey      *string            `json:"api_key"`
	Headers     *map[string]string `json:"headers"`
	Enabled     *bool              `json:"enabled"`
}

// mcpServerResponse is the sanitized response (no encrypted tokens)
type mcpServerResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	ServerURL     string                 `json:"server_url"`
	Transport     string                 `json:"transport"`
	AuthType      string                 `json:"auth_type"`
	HasAuth       bool                   `json:"has_auth"`
	Headers       map[string]string      `json:"headers"`
	Enabled       bool                   `json:"enabled"`
	ToolsCache    []services.MCPClientTool `json:"tools_cache"`
	Status        string                 `json:"status"`
	LastError     *string                `json:"last_error"`
	LastConnected *string                `json:"last_connected"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

const maxMCPServersPerUser = 20

// toResponse converts a DB model to a sanitized API response
func toMCPServerResponse(s sqlc.McpServer) mcpServerResponse {
	resp := mcpServerResponse{
		ID:          uuidToString(s.ID),
		Name:        s.Name,
		Description: ptrStringVal(s.Description),
		ServerURL:   s.ServerUrl,
		Transport:   s.Transport,
		AuthType:    s.AuthType,
		HasAuth:     s.AuthTokenEnc != nil && *s.AuthTokenEnc != "",
		Enabled:     s.Enabled,
		Status:      s.Status,
		LastError:   s.LastError,
	}

	// Parse headers from JSON
	resp.Headers = map[string]string{}
	if len(s.CustomHeaders) > 0 {
		_ = json.Unmarshal(s.CustomHeaders, &resp.Headers)
	}

	// Parse tools cache from JSON
	resp.ToolsCache = []services.MCPClientTool{}
	if len(s.ToolsCache) > 0 {
		_ = json.Unmarshal(s.ToolsCache, &resp.ToolsCache)
	}

	// Timestamps
	if s.LastConnected.Valid {
		t := s.LastConnected.Time.Format("2006-01-02T15:04:05Z")
		resp.LastConnected = &t
	}
	if s.CreatedAt.Valid {
		resp.CreatedAt = s.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	}
	if s.UpdatedAt.Valid {
		resp.UpdatedAt = s.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return resp
}

// ListMCPServers returns all MCP servers for the authenticated user
func (h *MCPServersHandler) ListMCPServers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	servers, err := queries.ListMCPServers(ctx, user.ID)
	if err != nil {
		slog.Error("Failed to list MCP servers", "user_id", user.ID, "error", err)
		utils.RespondInternalError(c, slog.Default(), "list MCP servers", err)
		return
	}

	result := make([]mcpServerResponse, len(servers))
	for i, s := range servers {
		result[i] = toMCPServerResponse(s)
	}

	c.JSON(http.StatusOK, gin.H{"connectors": result})
}

// GetMCPServer returns a specific MCP server
func (h *MCPServersHandler) GetMCPServer(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "connector_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	server, err := queries.GetMCPServer(ctx, sqlc.GetMCPServerParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "MCP server")
		return
	}

	c.JSON(http.StatusOK, gin.H{"connector": toMCPServerResponse(server)})
}

// CreateMCPServer adds a new MCP server connection
func (h *MCPServersHandler) CreateMCPServer(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate server name
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if len(name) == 0 || len(name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be 1-100 characters"})
		return
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name can only contain lowercase letters, numbers, hyphens, and underscores"})
			return
		}
	}

	// Validate URL (SSRF protection)
	if err := services.ValidateMCPServerURL(req.ServerURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server URL: " + err.Error()})
		return
	}

	// Validate transport
	transport := "sse"
	if req.Transport != "" {
		if req.Transport != "sse" && req.Transport != "streamable_http" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transport must be 'sse' or 'streamable_http'"})
			return
		}
		transport = req.Transport
	}

	// Validate auth type
	authType := "none"
	if req.AuthType != "" {
		if req.AuthType != "none" && req.AuthType != "api_key" && req.AuthType != "bearer" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Auth type must be 'none', 'api_key', or 'bearer'"})
			return
		}
		authType = req.AuthType
	}

	// Validate header count
	if len(req.Headers) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 custom headers allowed"})
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	// Check server count limit
	count, err := queries.CountUserMCPServers(ctx, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "count MCP servers", err)
		return
	}
	if count >= maxMCPServersPerUser {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 20 MCP servers allowed per user. Please remove unused servers.",
		})
		return
	}

	// Encrypt auth token if provided
	var authTokenEnc *string
	if req.APIKey != "" && authType != "none" {
		enc := security.GetGlobalEncryption()
		if enc != nil {
			encrypted, encErr := enc.Encrypt(req.APIKey)
			if encErr != nil {
				slog.Error("Failed to encrypt MCP auth token", "error", encErr)
				utils.RespondInternalError(c, slog.Default(), "encrypt auth token", encErr)
				return
			}
			authTokenEnc = &encrypted
		} else {
			// If encryption not available, store a warning
			slog.Warn("Encryption not initialized, storing MCP auth token as-is")
			authTokenEnc = &req.APIKey
		}
	}

	// Marshal headers to JSON
	headersJSON, _ := json.Marshal(req.Headers)

	desc := req.Description

	server, err := queries.CreateMCPServer(ctx, sqlc.CreateMCPServerParams{
		UserID:        user.ID,
		Name:          name,
		Description:   &desc,
		ServerUrl:     req.ServerURL,
		Transport:     transport,
		AuthType:      authType,
		AuthTokenEnc:  authTokenEnc,
		CustomHeaders: headersJSON,
		Enabled:       true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "An MCP server with this name already exists"})
			return
		}
		slog.Error("Failed to create MCP server", "error", err)
		utils.RespondInternalError(c, slog.Default(), "create MCP server", err)
		return
	}

	slog.Info("MCP server created", "id", uuidToString(server.ID), "name", name, "user_id", user.ID)

	c.JSON(http.StatusCreated, gin.H{"connector": toMCPServerResponse(server)})
}

// UpdateMCPServer updates an existing MCP server connection
func (h *MCPServersHandler) UpdateMCPServer(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "connector_id")
		return
	}

	var req UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate name if provided
	if req.Name != nil {
		name := strings.ToLower(strings.TrimSpace(*req.Name))
		if len(name) == 0 || len(name) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be 1-100 characters"})
			return
		}
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Name can only contain lowercase letters, numbers, hyphens, and underscores"})
				return
			}
		}
		req.Name = &name
	}

	// Validate URL if provided
	if req.ServerURL != nil {
		if err := services.ValidateMCPServerURL(*req.ServerURL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server URL: " + err.Error()})
			return
		}
	}

	// Validate transport if provided
	if req.Transport != nil {
		if *req.Transport != "sse" && *req.Transport != "streamable_http" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transport must be 'sse' or 'streamable_http'"})
			return
		}
	}

	// Validate auth type if provided
	if req.AuthType != nil {
		if *req.AuthType != "none" && *req.AuthType != "api_key" && *req.AuthType != "bearer" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Auth type must be 'none', 'api_key', or 'bearer'"})
			return
		}
	}

	// Encrypt auth token if provided
	var authTokenEnc *string
	if req.APIKey != nil && *req.APIKey != "" {
		enc := security.GetGlobalEncryption()
		if enc != nil {
			encrypted, encErr := enc.Encrypt(*req.APIKey)
			if encErr != nil {
				slog.Error("Failed to encrypt MCP auth token", "error", encErr)
				utils.RespondInternalError(c, slog.Default(), "encrypt auth token", encErr)
				return
			}
			authTokenEnc = &encrypted
		}
	}

	// Marshal headers if provided
	var headersJSON []byte
	if req.Headers != nil {
		if len(*req.Headers) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 custom headers allowed"})
			return
		}
		headersJSON, _ = json.Marshal(*req.Headers)
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	params := sqlc.UpdateMCPServerParams{
		ID:            pgtype.UUID{Bytes: id, Valid: true},
		UserID:        user.ID,
		Name:          req.Name,
		Description:   req.Description,
		ServerUrl:     req.ServerURL,
		Transport:     req.Transport,
		AuthType:      req.AuthType,
		AuthTokenEnc:  authTokenEnc,
		CustomHeaders: headersJSON,
		Enabled:       req.Enabled,
	}

	server, err := queries.UpdateMCPServer(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			utils.RespondNotFound(c, slog.Default(), "MCP server")
			return
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "An MCP server with this name already exists"})
			return
		}
		slog.Error("Failed to update MCP server", "error", err)
		utils.RespondInternalError(c, slog.Default(), "update MCP server", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"connector": toMCPServerResponse(server)})
}

// DeleteMCPServer removes an MCP server connection
func (h *MCPServersHandler) DeleteMCPServer(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "connector_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	err = queries.DeleteMCPServer(ctx, sqlc.DeleteMCPServerParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		slog.Error("Failed to delete MCP server", "error", err)
		utils.RespondInternalError(c, slog.Default(), "delete MCP server", err)
		return
	}

	slog.Info("MCP server deleted", "id", c.Param("id"), "user_id", user.ID)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestMCPServer tests the connection to an MCP server and discovers tools
func (h *MCPServersHandler) TestMCPServer(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "connector_id")
		return
	}

	ctx := context.Background()
	queries := sqlc.New(h.pool)

	server, err := queries.GetMCPServer(ctx, sqlc.GetMCPServerParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "MCP server")
		return
	}

	// Decrypt auth token
	authToken := ""
	if server.AuthTokenEnc != nil && *server.AuthTokenEnc != "" {
		enc := security.GetGlobalEncryption()
		if enc != nil {
			decrypted, decErr := enc.Decrypt(*server.AuthTokenEnc)
			if decErr != nil {
				slog.Error("Failed to decrypt MCP auth token", "error", decErr)
			} else {
				authToken = decrypted
			}
		}
	}

	// Parse custom headers
	headers := map[string]string{}
	if len(server.CustomHeaders) > 0 {
		_ = json.Unmarshal(server.CustomHeaders, &headers)
	}

	// Create MCP client and test
	client := services.NewMCPClient(server.ServerUrl, server.AuthType, authToken, headers)

	tools, discoverErr := client.DiscoverTools(ctx)
	if discoverErr != nil {
		// Update status to error
		errMsg := discoverErr.Error()
		_ = queries.UpdateMCPServerStatus(ctx, sqlc.UpdateMCPServerStatusParams{
			ID:        pgtype.UUID{Bytes: id, Valid: true},
			UserID:    user.ID,
			Status:    "error",
			LastError: &errMsg,
		})

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Connection failed: " + errMsg,
		})
		return
	}

	// Cache discovered tools
	toolsJSON, _ := json.Marshal(tools)
	_ = queries.UpdateMCPServerToolsCache(ctx, sqlc.UpdateMCPServerToolsCacheParams{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		UserID:     user.ID,
		ToolsCache: toolsJSON,
	})

	slog.Info("MCP server test successful", "id", c.Param("id"), "tools_found", len(tools))

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Connection successful",
		"tool_count": len(tools),
		"tools":      tools,
	})
}

// DiscoverMCPTools forces tool re-discovery on an MCP server
func (h *MCPServersHandler) DiscoverMCPTools(c *gin.Context) {
	// Same logic as test — discover tools and cache them
	h.TestMCPServer(c)
}

// helper: dereference string pointer with empty default
func ptrStringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
