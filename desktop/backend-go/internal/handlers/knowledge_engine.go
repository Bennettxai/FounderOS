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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// =====================================================================
// KNOWLEDGE ENGINE VIEW (Plane 2: the workspace's Optimal Engine KB)
//
// Each BusinessOS workspace links to exactly ONE Optimal Engine workspace
// through its EngineConfig (settings->'optimal_engine': enabled, base_url,
// api_key, workspace-slug). The Knowledge module's engine view surfaces ONLY
// that engine workspace's knowledge base - never any other workspace's.
//
// Gating is strict: the caller must be authenticated AND an active member of
// the X-Workspace-ID workspace. Non-members get 403. When the workspace has no
// engine configured (or it is disabled), the endpoint returns an empty list
// with engine_linked=false so the frontend can render a "connect an engine"
// state instead of an error.
// =====================================================================

// KnowledgeEngineHandler serves the engine-backed Knowledge view for the
// active workspace. It owns no engine config of its own: it reads each
// workspace's stored EngineConfig and talks to that engine directly, mirroring
// the per-workspace resolution used by the optimal proxy and the write path.
type KnowledgeEngineHandler struct {
	pool *pgxpool.Pool
}

// NewKnowledgeEngineHandler constructs the handler.
func NewKnowledgeEngineHandler(pool *pgxpool.Pool) *KnowledgeEngineHandler {
	return &KnowledgeEngineHandler{pool: pool}
}

// knowledgeEngineItem is one row in the Knowledge canonical view. Fields the
// engine does not natively provide are filled with sensible defaults so the
// frontend always receives a complete shape (Req 3 canonical, Req 7 promotion
// targets, Req 8 internal/external labels).
type knowledgeEngineItem struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Abstract     string  `json:"abstract"`
	Source       string  `json:"source"`
	Status       string  `json:"status"`
	TargetModule *string `json:"target_module"`
	Owner        *string `json:"owner"`
	Canonical    bool    `json:"canonical"`
	IsExternal   bool    `json:"is_external"`
	UpdatedAt    string  `json:"updated_at"`
}

// knowledgeEngineResponse is the exact GET /api/knowledge/engine contract.
type knowledgeEngineResponse struct {
	Items        []knowledgeEngineItem `json:"items"`
	EngineLinked bool                  `json:"engine_linked"`
}

// knowledgeEngineConfig is the per-workspace engine link, read from
// settings->'optimal_engine'. Mirrors workspace_engine.go's stored shape; the
// only fields this read path needs are enabled, base_url, api_key, and the
// engine-side workspace slug.
type knowledgeEngineConfig struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Workspace string `json:"workspace"` // engine-side workspace slug
}

// linked reports whether this config can actually serve a KB: enabled, with a
// usable base URL.
func (cfg knowledgeEngineConfig) linked() bool {
	return cfg.Enabled && strings.TrimSpace(cfg.BaseURL) != ""
}

// GetEngineKnowledge returns the active workspace's engine-backed knowledge
// base, mapped to the canonical Knowledge view contract.
//
// GET /api/knowledge/engine?q=<optional>
//
//	Headers: X-Workspace-ID (required) - the active BusinessOS workspace.
//	Auth:    required; caller must be an active member of that workspace.
//
// Response: { "items": [ ... ], "engine_linked": bool }
//
// Behavior:
//   - 401 when unauthenticated.
//   - 403 when X-Workspace-ID is missing/invalid or the caller is not an
//     active member.
//   - 200 with { items: [], engine_linked: false } when the workspace has no
//     engine configured or it is disabled.
//   - 200 with mapped items otherwise (only this engine workspace's KB).
func (h *KnowledgeEngineHandler) GetEngineKnowledge(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}

	// Require an active membership of the X-Workspace-ID workspace.
	wsID, ok := h.activeMembership(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace you are a member of"})
		return
	}

	// Load this workspace's engine link. Not linked -> empty + engine_linked:false.
	cfg, err := h.readEngineConfig(c.Request.Context(), wsID)
	if err != nil {
		slog.Error("[KnowledgeEngine] failed to read engine config", "workspace_id", wsID.String(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load engine config"})
		return
	}
	if !cfg.linked() {
		c.JSON(http.StatusOK, knowledgeEngineResponse{Items: []knowledgeEngineItem{}, EngineLinked: false})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	items, err := h.fetchEngineItems(c.Request.Context(), cfg, q)
	if err != nil {
		// The engine being unreachable is not a client error; surface a clean
		// linked-but-empty state rather than a 5xx that breaks the module.
		slog.Warn("[KnowledgeEngine] engine query failed", "workspace_id", wsID.String(), "error", err)
		c.JSON(http.StatusOK, knowledgeEngineResponse{Items: []knowledgeEngineItem{}, EngineLinked: true})
		return
	}

	c.JSON(http.StatusOK, knowledgeEngineResponse{Items: items, EngineLinked: true})
}

// activeMembership resolves X-Workspace-ID and confirms the user is an active
// member. Returns (uuid.Nil, false) on any miss so the caller can 403.
func (h *KnowledgeEngineHandler) activeMembership(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := strings.TrimSpace(c.GetHeader("X-Workspace-ID"))
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// readEngineConfig loads settings->'optimal_engine' for a workspace. A missing
// config yields a zero-value (disabled) config, not an error.
func (h *KnowledgeEngineHandler) readEngineConfig(ctx context.Context, workspaceID uuid.UUID) (knowledgeEngineConfig, error) {
	var raw []byte
	err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(settings->'optimal_engine', '{}'::jsonb) FROM workspaces WHERE id = $1`,
		workspaceID).Scan(&raw)
	if err != nil {
		return knowledgeEngineConfig{}, err
	}
	cfg := knowledgeEngineConfig{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &cfg)
	}
	return cfg, nil
}

// engineSearchResponse is the subset of the engine's GET /api/search shape we
// consume. The engine returns more (pagination, query echo); we ignore it.
type engineSearchResponse struct {
	Results []engineSearchResult `json:"results"`
}

// engineSearchResult mirrors the engine's per-hit shape
// (format_search_results/2 in the Elixir router).
type engineSearchResult struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Node       string  `json:"node"`
	Genre      string  `json:"genre"`
	URI        string  `json:"uri"`
	L0Abstract string  `json:"l0_abstract"`
	SNRatio    float64 `json:"sn_ratio"`
}

// fetchEngineItems calls the engine's /api/search for ONLY this workspace's
// engine slug and maps hits to the contract. The engine's /api/search returns
// nothing for an empty query, so when no q is given we pass the workspace slug
// as a broad anchor to surface that workspace's documents.
func (h *KnowledgeEngineHandler) fetchEngineItems(ctx context.Context, cfg knowledgeEngineConfig, q string) ([]knowledgeEngineItem, error) {
	engineWS := strings.TrimSpace(cfg.Workspace)
	if engineWS == "" {
		engineWS = "default"
	}

	query := q
	if query == "" {
		// /api/search ignores empty q; anchor to the workspace slug so the
		// caller still gets that workspace's canonical docs.
		query = engineWorkspaceSearchAnchor(engineWS)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("workspace", engineWS) // hard scope: this engine workspace only
	params.Set("limit", "50")

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/") + "/api/search?" + params.Encode()

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("X-API-Key", key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &engineHTTPError{status: resp.StatusCode}
	}

	var parsed engineSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	items := make([]knowledgeEngineItem, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		source := r.URI
		if source == "" {
			source = r.Node
		}
		items = append(items, knowledgeEngineItem{
			ID:       r.ID,
			Title:    r.Title,
			Abstract: r.L0Abstract,
			Source:   source,
			// Defaults for fields the engine does not yet provide. These map to
			// the canonical-source (Req 3), promotion (Req 7), and
			// internal/external (Req 8) requirements; promotion/labels are
			// owned by a later write path.
			Status:       "active",
			TargetModule: nil,
			Owner:        nil,
			Canonical:    true,
			IsExternal:   false,
			UpdatedAt:    "",
		})
	}
	return items, nil
}

func engineWorkspaceSearchAnchor(workspace string) string {
	anchor := strings.TrimSpace(workspace)
	replacer := strings.NewReplacer("-", " ", "_", " ", ":", " ")
	anchor = strings.Join(strings.Fields(replacer.Replace(anchor)), " ")
	if anchor == "" {
		return "default"
	}
	return anchor
}

// engineHTTPError is a small typed error so logs can carry the upstream status.
type engineHTTPError struct{ status int }

func (e *engineHTTPError) Error() string {
	return "optimal engine returned non-2xx status: " + http.StatusText(e.status)
}
