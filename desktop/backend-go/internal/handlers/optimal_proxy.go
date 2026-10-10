package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// OptimalProxy forwards every /api/optimal/* request to a live Elixir
// OptimalEngine HTTP server. It resolves the target PER WORKSPACE: a request
// carrying X-Workspace-ID is proxied to that workspace's configured engine
// (settings->'optimal_engine': base_url + api_key), so each workspace's reads
// hit its own second brain - consistent with the write path. Requests without a
// configured workspace engine fall back to the global OPTIMAL_ENGINE_URL.
type OptimalProxy struct {
	defaultTarget *url.URL // global fallback (OPTIMAL_ENGINE_URL), optional
	pool          *pgxpool.Pool
	rp            *httputil.ReverseProxy
}

type proxyTargetKey struct{}

type proxyTarget struct {
	url       *url.URL
	apiKey    string
	workspace string
}

// NewOptimalProxy builds the proxy. engineURL is the optional global fallback;
// pool enables per-workspace resolution. At least one must be present.
func NewOptimalProxy(engineURL string, pool *pgxpool.Pool) (*OptimalProxy, error) {
	var def *url.URL
	if s := strings.TrimSpace(engineURL); s != "" {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, errors.New("optimal proxy: engine URL must include scheme + host")
		}
		def = u
	}
	if def == nil && pool == nil {
		return nil, errors.New("optimal proxy: needs an engine URL or a DB pool")
	}

	p := &OptimalProxy{defaultTarget: def, pool: pool}
	p.rp = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := def
			apiKey := ""
			workspace := ""
			if t, ok := r.In.Context().Value(proxyTargetKey{}).(*proxyTarget); ok && t != nil {
				target = t.url
				apiKey = t.apiKey
				workspace = t.workspace
			}
			r.SetURL(target)
			// Rewrite path: /api/v1/optimal/nodes → /api/nodes
			//               /api/optimal/graph   → /api/graph
			path := r.Out.URL.Path
			if i := strings.Index(path, "/optimal/"); i >= 0 {
				r.Out.URL.Path = "/api" + path[i+len("/optimal"):]
			} else if strings.HasSuffix(path, "/optimal") {
				r.Out.URL.Path = "/api"
			}
			// BusinessOS clients historically POST JSON to /optimal/search.
			// The Elixir Optimal Engine exposes search as GET /api/search?q=...
			// Translate the local compatibility shape at the proxy boundary.
			if r.Out.Method == http.MethodPost && r.Out.URL.Path == "/api/search" {
				var body struct {
					Query     string      `json:"query"`
					Q         string      `json:"q"`
					Workspace string      `json:"workspace"`
					Limit     interface{} `json:"limit"`
				}
				if r.Out.Body != nil {
					raw, _ := io.ReadAll(r.Out.Body)
					_ = r.Out.Body.Close()
					_ = json.Unmarshal(raw, &body)
				}
				q := r.Out.URL.Query()
				if strings.TrimSpace(body.Query) != "" {
					q.Set("q", strings.TrimSpace(body.Query))
				} else if strings.TrimSpace(body.Q) != "" {
					q.Set("q", strings.TrimSpace(body.Q))
				}
				if strings.TrimSpace(workspace) != "" {
					q.Set("workspace", strings.TrimSpace(workspace))
				} else if strings.TrimSpace(body.Workspace) != "" {
					q.Set("workspace", strings.TrimSpace(body.Workspace))
				}
				switch v := body.Limit.(type) {
				case float64:
					if v == float64(int64(v)) {
						q.Set("limit", fmt.Sprintf("%d", int64(v)))
					} else {
						q.Set("limit", fmt.Sprintf("%g", v))
					}
				case string:
					if strings.TrimSpace(v) != "" {
						q.Set("limit", strings.TrimSpace(v))
					}
				}
				r.Out.URL.RawQuery = q.Encode()
				r.Out.Method = http.MethodGet
				r.Out.Body = nil
				r.Out.ContentLength = 0
				r.Out.Header.Del("Content-Type")
				r.Out.Header.Del("Content-Length")
			} else if strings.TrimSpace(workspace) != "" && r.Out.URL.Query().Get("workspace") == "" && isWorkspaceScopedOptimalPath(r.Out.URL.Path) {
				q := r.Out.URL.Query()
				q.Set("workspace", strings.TrimSpace(workspace))
				r.Out.URL.RawQuery = q.Encode()
			}
			r.Out.Host = target.Host
			if apiKey != "" {
				r.Out.Header.Set("Authorization", "Bearer "+apiKey)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "optimal proxy: upstream call failed",
				"path", r.URL.Path, "method", r.Method, "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, `{"error":"optimal engine unreachable"}`)
		},
		Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second},
	}
	return p, nil
}

// resolveWorkspaceEngine reads a workspace's configured engine (base_url + key).
// The bool reports whether a concrete workspace was selected. A selected but
// unconfigured workspace must not fall back to the global engine, because that
// would leak the default engine workspace into another BusinessOS workspace.
func (p *OptimalProxy) resolveWorkspaceEngine(ctx context.Context, wsID string) (*proxyTarget, bool) {
	if p.pool == nil || strings.TrimSpace(wsID) == "" {
		return nil, false
	}
	var raw []byte
	if err := p.pool.QueryRow(ctx,
		`SELECT COALESCE(settings->'optimal_engine', '{}'::jsonb)::text
		   FROM workspaces WHERE id = $1::uuid`, wsID).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, true
		}
		slog.WarnContext(ctx, "optimal proxy: workspace engine lookup failed", "workspace_id", wsID, "error", err)
		return nil, true
	}
	var cfg struct {
		Enabled   bool   `json:"enabled"`
		BaseURL   string `json:"base_url"`
		APIKey    string `json:"api_key"`
		Workspace string `json:"workspace"`
	}
	if json.Unmarshal(raw, &cfg) != nil || !cfg.Enabled || strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, true
	}
	u, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, true
	}
	return &proxyTarget{url: u, apiKey: cfg.APIKey, workspace: cfg.Workspace}, true
}

func (p *OptimalProxy) currentUserCanAccessWorkspace(ctx context.Context, userID, wsID string) bool {
	if p.pool == nil || strings.TrimSpace(userID) == "" || strings.TrimSpace(wsID) == "" {
		return false
	}

	var member bool
	err := p.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1
			  FROM workspace_members
			 WHERE workspace_id = $1::uuid
			   AND user_id = $2
			   AND status = 'active'
		)`, wsID, userID).Scan(&member)
	return err == nil && member
}

func isWorkspaceScopedOptimalPath(path string) bool {
	switch path {
	case "/api/search", "/api/grep", "/api/rag", "/api/wiki", "/api/memory", "/api/graph":
		return true
	default:
		return strings.HasPrefix(path, "/api/wiki/") ||
			strings.HasPrefix(path, "/api/memory/") ||
			strings.HasPrefix(path, "/api/recall/")
	}
}

// RegisterOptimalProxyRoutes mounts the catch-all proxy at /api/optimal/*path.
func RegisterOptimalProxyRoutes(api *gin.RouterGroup, p *OptimalProxy) {
	api.Any("/optimal/*path", p.Forward)
}

// Forward resolves the per-workspace engine (falling back to the global one),
// then reverse-proxies the request.
func (p *OptimalProxy) Forward(c *gin.Context) {
	wsID := strings.TrimSpace(c.GetHeader("X-Workspace-ID"))
	if user := middleware.GetCurrentUser(c); user != nil && wsID != "" && !p.currentUserCanAccessWorkspace(c.Request.Context(), user.ID, wsID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}

	target, workspaceSelected := p.resolveWorkspaceEngine(c.Request.Context(), wsID)
	if target == nil && (workspaceSelected || p.defaultTarget == nil) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "no Optimal Engine configured for this workspace",
			"hint":  "connect one in Settings > Optimal Engine",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	if target != nil {
		ctx = context.WithValue(ctx, proxyTargetKey{}, target)
	}
	p.rp.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}
