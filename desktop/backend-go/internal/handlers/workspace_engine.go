package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// =====================================================================
// OPTIMAL ENGINE CONNECTION (per-workspace, shared)
//
// The Optimal Engine connection is configured once per workspace and shared
// by the whole team: base URL, API key, and the engine-side workspace slug
// that backs this BusinessOS workspace as its knowledge base. The config is
// stored in workspaces.settings->'optimal_engine' (JSONB) so it needs no extra
// table, and it travels with the workspace.
// =====================================================================

// engineStoredConfig is the shape persisted under settings.optimal_engine.
// The api_key is kept here in storage but NEVER returned to clients directly.
type engineStoredConfig struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Workspace string `json:"workspace"` // engine-side workspace slug
	UpdatedAt string `json:"updated_at,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// engineClientResponse is what clients receive from GET and UPDATE endpoints.
// HasAPIKey signals whether a key is stored without exposing the secret.
type engineClientResponse struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	HasAPIKey bool   `json:"has_api_key"`
	Workspace string `json:"workspace"`
	UpdatedAt string `json:"updated_at,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// engineUpdateRequest is the shape accepted by the UPDATE endpoint.
// If api_key is omitted or empty and one already exists, the stored key is kept.
type engineUpdateRequest struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Workspace string `json:"workspace"`
}

// toClientResponse converts stored config to the safe client shape.
func (s *engineStoredConfig) toClientResponse() engineClientResponse {
	return engineClientResponse{
		Enabled:   s.Enabled,
		BaseURL:   s.BaseURL,
		HasAPIKey: s.APIKey != "",
		Workspace: s.Workspace,
		UpdatedAt: s.UpdatedAt,
		UpdatedBy: s.UpdatedBy,
	}
}

// GetWorkspaceEngineConfig returns the workspace's Optimal Engine connection.
// The api_key is never included; has_api_key indicates whether one is stored.
//
// GET /api/workspaces/:id/engine  (any member)
//
// Response: { "engine": { enabled, base_url, has_api_key, workspace, updated_at, updated_by } }
func (h *WorkspaceHandler) GetWorkspaceEngineConfig(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	// Must be a member to read the connection.
	if _, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	stored, err := h.readEngineConfig(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"engine": stored.toClientResponse()})
}

// UpdateWorkspaceEngineConfig saves the workspace's Optimal Engine connection.
//
// PUT /api/workspaces/:id/engine  (owner/admin/manager)
//
// Request:  { enabled, base_url, api_key (optional), workspace }
// Response: { "engine": { enabled, base_url, has_api_key, workspace, updated_at, updated_by } }
//
// If api_key is omitted or empty and one is already stored, the stored key is kept.
// base_url must be a well-formed http(s) URL; it is trimmed and lowercased.
func (h *WorkspaceHandler) UpdateWorkspaceEngineConfig(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}

	role, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	if role != "owner" && role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners, admins, and managers can configure the Optimal Engine"})
		return
	}

	var req engineUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Validate and normalize base_url.
	trimmedURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if trimmedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url is required"})
		return
	}
	parsed, err := url.ParseRequestURI(trimmedURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_url must be a well-formed http or https URL"})
		return
	}

	// Load the existing config so we can preserve the API key if the caller did
	// not supply a new one.
	existing, err := h.readEngineConfig(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load current engine config"})
		return
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		// No new key supplied - keep whatever is stored.
		apiKey = existing.APIKey
	}

	stored := engineStoredConfig{
		Enabled:   req.Enabled,
		BaseURL:   trimmedURL,
		APIKey:    apiKey,
		Workspace: strings.TrimSpace(req.Workspace),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedBy: user.ID,
	}

	payload, err := json.Marshal(stored)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode engine config"})
		return
	}

	// Merge into settings JSONB without clobbering other settings keys.
	_, err = h.pool.Exec(c.Request.Context(),
		`UPDATE workspaces
		 SET settings = jsonb_set(COALESCE(settings, '{}'::jsonb), '{optimal_engine}', $2::jsonb, true),
		     updated_at = NOW()
		 WHERE id = $1`,
		workspaceID, string(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"engine": stored.toClientResponse()})
}

// TestWorkspaceEngineConnection pings the configured engine and reports
// reachability. The API key is loaded from storage; it is never logged or
// echoed back in the response.
//
// POST /api/workspaces/:id/engine/test  (any member)
//
// Response: { reachable bool, status int, message string }
func (h *WorkspaceHandler) TestWorkspaceEngineConnection(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "workspace_id")
		return
	}
	if _, err := h.workspaceService.GetUserRole(c.Request.Context(), workspaceID, user.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	// Load the saved config - the key lives in storage, not in the request.
	saved, err := h.readEngineConfig(c.Request.Context(), workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load engine config"})
		return
	}

	if saved.BaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"reachable": false, "message": "No engine URL configured"})
		return
	}

	reachable, status, message := pingEngine(c.Request.Context(), saved.BaseURL, saved.APIKey)
	c.JSON(http.StatusOK, gin.H{
		"reachable": reachable,
		"status":    status,
		"message":   message,
	})
}

// readEngineConfig loads settings.optimal_engine for a workspace. Returns a
// zero-value (disabled) config when nothing is set yet.
func (h *WorkspaceHandler) readEngineConfig(ctx context.Context, workspaceID uuid.UUID) (*engineStoredConfig, error) {
	var raw []byte
	err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(settings->'optimal_engine', '{}'::jsonb) FROM workspaces WHERE id = $1`,
		workspaceID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	cfg := &engineStoredConfig{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, cfg)
	}
	return cfg, nil
}

// pingEngine does a short GET against the engine health endpoint.
// The api_key is used for auth but never logged.
func pingEngine(ctx context.Context, baseURL, apiKey string) (bool, int, string) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	for _, path := range []string{"/api/health", "/health"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			return false, 0, "Invalid engine URL"
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("X-API-Key", apiKey)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, 0, "Could not reach engine: " + err.Error()
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, resp.StatusCode, "Connected"
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return false, resp.StatusCode, "Reached the engine but the API key was rejected"
		}
		if resp.StatusCode != http.StatusNotFound {
			return false, resp.StatusCode, "Engine responded with an error"
		}
	}
	return false, http.StatusNotFound, "Engine responded with an error"
}
