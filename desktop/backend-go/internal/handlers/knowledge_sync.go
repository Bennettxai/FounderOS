package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// knowledgeDocsDDL bootstraps the cloud-side table on first sync so a separate
// migration run is not required on the cloud DB. Matches migration 125.
const knowledgeDocsDDL = `
CREATE TABLE IF NOT EXISTS knowledge_documents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_slug TEXT NOT NULL,
    workspace_id   UUID,
    path           TEXT NOT NULL,
    title          TEXT NOT NULL DEFAULT '',
    body           TEXT NOT NULL DEFAULT '',
    section        TEXT NOT NULL DEFAULT 'docs',
    synced_by      VARCHAR(255),
    synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_slug, path)
);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_slug ON knowledge_documents(workspace_slug);`

// workspaceStorageDDL bootstraps the cloud-side storage-accounting table on
// first sync so a separate migration run is not required on the cloud DB.
// Matches migration 140. Idempotent.
const workspaceStorageDDL = `
CREATE TABLE IF NOT EXISTS workspace_storage (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    bytes_used   BIGINT NOT NULL DEFAULT 0,
    bytes_limit  BIGINT NOT NULL DEFAULT 1073741824,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE workspace_storage ADD COLUMN IF NOT EXISTS cloud_sync_activated BOOLEAN NOT NULL DEFAULT false;`

// defaultStorageLimitBytes is the free-tier quota (1 GB) applied when a
// workspace has no explicit workspace_storage row yet.
const defaultStorageLimitBytes int64 = 1073741824

// kbDoc is one collected markdown document.
type kbDoc struct {
	Path    string
	Title   string
	Body    string
	Section string
}

// sectionOf derives the module section (top-level folder) from a relative path.
func sectionOf(rel string) string {
	if i := strings.Index(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return "docs"
}

// collectLocalDocs walks a workspace dir and returns every .md file (skipping dotdirs).
func (h *KnowledgeHandler) collectLocalDocs(dir string) []kbDoc {
	var docs []kbDoc
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, e := filepath.Rel(dir, p)
		if e != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		data, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		docs = append(docs, kbDoc{Path: rel, Title: titleFromName(d.Name()), Body: string(data), Section: sectionOf(rel)})
		return nil
	})
	return docs
}

type syncKnowledgeReq struct {
	Workspace   string `json:"workspace"`
	WorkspaceID string `json:"workspace_id"`
}

// SyncToCloud pushes a workspace's local knowledge docs up to the cloud copy so
// teammates without their own engine can view them in the Knowledge module, and
// teammates with an engine can pull them. Owner-only: requires CLOUD_DATABASE_URL
// (only the owner's machine has it). Full replace of that workspace's cloud copy.
//
//	POST /api/knowledge/sync-to-cloud  { "workspace": "agency-miosa", "workspace_id": "<uuid>" }
func (h *KnowledgeHandler) SyncToCloud(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	var req syncKnowledgeReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Workspace) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace required"})
		return
	}
	dir, ok := h.workspaceDir(req.Workspace)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no local files for this workspace to sync"})
		return
	}
	cloudURL := services.CloudDatabaseURL()
	if cloudURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CLOUD_DATABASE_URL not set on this machine (only the owner can sync up)"})
		return
	}

	docs := h.collectLocalDocs(dir)
	ctx := c.Request.Context()

	cfg, err := pgxpool.ParseConfig(cloudURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse cloud url: " + err.Error()})
		return
	}
	// Supabase transaction pooler (:6543) doesn't support prepared statements.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cloud, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "connect cloud: " + err.Error()})
		return
	}
	defer cloud.Close()

	if _, err := cloud.Exec(ctx, knowledgeDocsDDL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ensure cloud table: " + err.Error()})
		return
	}
	if _, err := cloud.Exec(ctx, workspaceStorageDDL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ensure storage table: " + err.Error()})
		return
	}

	// Resolve the workspace UUID (storage + activation are keyed by it) and the
	// quota that applies. A sync fully replaces this workspace's docs, so the
	// incoming payload size IS the new bytes_used -> we can enforce the limit up
	// front, before the expensive delete/re-insert. Prefer the id the client
	// supplied; fall back to resolving it from the workspaces table by slug so
	// activation can be checked even before anything has ever synced.
	wsIDStr := strings.TrimSpace(req.WorkspaceID)
	if wsIDStr == "" {
		if resolved, ok := resolveWorkspaceIDBySlug(ctx, cloud, req.Workspace); ok {
			wsIDStr = resolved
		}
	}
	var wsID interface{}
	if wsIDStr != "" {
		wsID = wsIDStr
	}

	// MULTI-TENANT GATE: only a member of this workspace may push its knowledge
	// to the shared cloud copy. Requires the slug to resolve to a workspace_id.
	if wsIDStr == "" || !userIsWorkspaceMember(ctx, cloud, wsIDStr, user.ID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
		return
	}

	// OPT-IN GATE: nothing syncs to the cloud until the user has explicitly
	// activated cloud sync for this workspace in the Knowledge module. Enforced
	// server-side so a stale/rogue client cannot push before opt-in.
	if !isCloudSyncActivated(ctx, cloud, wsIDStr) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "NOT_ACTIVATED",
			"message": "Cloud sync is not activated for this workspace",
		}})
		return
	}
	var incomingBytes int64
	for _, d := range docs {
		incomingBytes += int64(len(d.Body))
	}
	bytesLimit := defaultStorageLimitBytes
	if wsID != nil {
		var existing int64
		if err := cloud.QueryRow(ctx, `SELECT bytes_limit FROM workspace_storage WHERE workspace_id=$1`, wsID).Scan(&existing); err == nil && existing > 0 {
			bytesLimit = existing
		}
	}
	// SOFT limit (billing not live yet): never block a sync. If the workspace is
	// over its free quota we still write everything and flag over_limit in the
	// response so the UI can nudge "upgrade coming soon". Flip this to a hard
	// 413 block once billing is wired.
	overLimit := incomingBytes > bytesLimit

	tx, err := cloud.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "begin: " + err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM knowledge_documents WHERE workspace_slug=$1`, req.Workspace); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "clear: " + err.Error()})
		return
	}

	count := 0
	for _, d := range docs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO knowledge_documents (workspace_slug, workspace_id, path, title, body, section, synced_by)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			req.Workspace, wsID, d.Path, d.Title, d.Body, d.Section, user.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "insert " + d.Path + ": " + err.Error()})
			return
		}
		count++
	}

	// Authoritative usage from what we just wrote (octet_length = byte length,
	// same units as the incoming pre-check). Persist + re-enforce the quota
	// inside the tx so a race or a body larger on disk than measured can't slip
	// past. When no workspace_id was supplied we cannot key a storage row, so we
	// fall back to the incoming byte total for the response only.
	bytesUsed := incomingBytes
	if wsID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(SUM(octet_length(body)),0) FROM knowledge_documents WHERE workspace_id=$1`, wsID).Scan(&bytesUsed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "measure storage: " + err.Error()})
			return
		}
		// Soft limit: record the true usage and flag over_limit, but commit the
		// sync regardless (no hard block while billing is not live).
		if bytesUsed > bytesLimit {
			overLimit = true
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO workspace_storage (workspace_id, bytes_used, bytes_limit, updated_at)
			 VALUES ($1,$2,$3,now())
			 ON CONFLICT (workspace_id) DO UPDATE SET bytes_used=EXCLUDED.bytes_used, updated_at=now()`,
			wsID, bytesUsed, bytesLimit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "record storage: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": req.Workspace,
		"documents": count,
		"synced_by": user.Email,
		"storage": gin.H{
			"bytes_used":  bytesUsed,
			"bytes_limit": bytesLimit,
			"over_limit":  overLimit,
		},
	})
}

// pullDoc is one document handed to a teammate pulling into their own engine.
type pullDoc struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Section  string    `json:"section"`
	SyncedAt time.Time `json:"synced_at"`
}

// PullFromCloud returns every synced document for a workspace so a teammate can
// pull the workspace's source-of-truth knowledge INTO their own local Optimal
// Engine. The pull is additive by contract: the caller (the teammate's engine
// ingest) must ADD/UPDATE these docs without deleting or overriding whatever
// already exists in their engine. The cloud copy is the shared source of truth;
// each engine keeps its own superset.
//
//	GET /api/knowledge/pull?workspace=<slug>
func (h *KnowledgeHandler) PullFromCloud(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	slug := c.Query("workspace")
	if strings.TrimSpace(slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace required"})
		return
	}
	if h.pool == nil {
		c.JSON(http.StatusOK, gin.H{"workspace": slug, "documents": []pullDoc{}, "count": 0})
		return
	}
	// Multi-tenant gate: the cloud copy is shared, so only members of this
	// workspace may pull it. (Local file reads elsewhere are single-tenant.)
	if !h.callerMemberOfSlug(c, h.pool, slug) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT path, title, body, section, synced_at FROM knowledge_documents WHERE workspace_slug=$1 ORDER BY path`, slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read cloud copy: " + err.Error()})
		return
	}
	defer rows.Close()

	docs := []pullDoc{}
	for rows.Next() {
		var d pullDoc
		if err := rows.Scan(&d.Path, &d.Title, &d.Body, &d.Section, &d.SyncedAt); err == nil {
			docs = append(docs, d)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"workspace": slug,
		"documents": docs,
		"count":     len(docs),
		"contract":  "additive: add/update into your engine, never delete or override existing content",
	})
}

// KnowledgeSources returns, per document path, where it lives: in the local
// engine (this machine's workspace files), in the shared cloud copy, or both
// ("synced"). This powers the single-source-of-truth view where you see engine
// and cloud docs together, tagged, and decide what to sync - rather than the
// two being separated.
//
//	GET /api/knowledge/sources?workspace=<slug>
//	-> { workspace, sources: { "<path>": "engine"|"cloud"|"synced" },
//	     counts: { engine, cloud, synced } }
func (h *KnowledgeHandler) KnowledgeSources(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	slug := c.Query("workspace")
	if strings.TrimSpace(slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace required"})
		return
	}
	ctx := c.Request.Context()

	// Local engine docs (this machine's workspace files).
	localSet := map[string]bool{}
	if dir, ok := h.workspaceDir(slug); ok {
		for _, d := range h.collectLocalDocs(dir) {
			localSet[d.Path] = true
		}
	}

	// Cloud copy: query the cloud DB directly if this machine owns the sync
	// (CLOUD_DATABASE_URL set); otherwise this IS the cloud backend, use its pool.
	cloudSet := map[string]bool{}
	queryCloud := func(pool *pgxpool.Pool) {
		if pool == nil {
			return
		}
		// Multi-tenant gate: only fold in the shared cloud copy for members of
		// this workspace. Non-members see local docs only, never cloud paths.
		if !h.callerMemberOfSlug(c, pool, slug) {
			return
		}
		rows, err := pool.Query(ctx, `SELECT path FROM knowledge_documents WHERE workspace_slug=$1`, slug)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil {
				cloudSet[p] = true
			}
		}
	}
	if cloudURL := services.CloudDatabaseURL(); cloudURL != "" {
		if cfg, err := pgxpool.ParseConfig(cloudURL); err == nil {
			cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
			if cloud, err := pgxpool.NewWithConfig(ctx, cfg); err == nil {
				defer cloud.Close()
				queryCloud(cloud)
			}
		}
	} else {
		queryCloud(h.pool)
	}

	sources := map[string]string{}
	var engine, cloud, synced int
	for p := range localSet {
		if cloudSet[p] {
			sources[p] = "synced"
			synced++
		} else {
			sources[p] = "engine"
			engine++
		}
	}
	for p := range cloudSet {
		if !localSet[p] {
			sources[p] = "cloud"
			cloud++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": slug,
		"sources":   sources,
		"counts":    gin.H{"engine": engine, "cloud": cloud, "synced": synced},
	})
}

// GetStorage returns the cloud-sync storage usage and quota for a workspace so
// the Knowledge module can show a usage meter and warn before a sync is
// rejected. Reads the same DB the sync writes to: the cloud DB when this
// machine owns the sync (CLOUD_DATABASE_URL set), otherwise this backend's
// pool. Workspace_storage is keyed by workspace_id, so we bridge the folder
// slug -> workspace_id via the synced knowledge_documents. When nothing has
// been synced yet (or the table does not exist), returns the free-tier defaults.
//
//	GET /api/knowledge/storage?workspace=<slug>
//	-> { workspace, bytes_used, bytes_limit }
func (h *KnowledgeHandler) GetStorage(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	slug := c.Query("workspace")
	if strings.TrimSpace(slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace required"})
		return
	}
	ctx := c.Request.Context()

	bytesUsed := int64(0)
	bytesLimit := defaultStorageLimitBytes
	activated := false
	read := func(pool *pgxpool.Pool) {
		if pool == nil {
			return
		}
		// Resolve the workspace UUID. Prefer a synced doc's workspace_id, but fall
		// back to the workspaces table by slug so activation status is known even
		// before anything has ever synced (knowledge_documents is empty pre-sync).
		var wsID string
		if err := pool.QueryRow(ctx,
			`SELECT workspace_id::text FROM knowledge_documents WHERE workspace_slug=$1 AND workspace_id IS NOT NULL LIMIT 1`, slug).Scan(&wsID); err != nil {
			if resolved, ok := resolveWorkspaceIDBySlug(ctx, pool, slug); ok {
				wsID = resolved
			}
		}
		if wsID == "" {
			return
		}
		// Multi-tenant gate: never report another workspace's usage to a
		// non-member. Leaves the safe defaults in place if not a member.
		if !userIsWorkspaceMember(ctx, pool, wsID, user.ID) {
			return
		}
		var bu, bl int64
		var act bool
		if err := pool.QueryRow(ctx,
			`SELECT bytes_used, bytes_limit, COALESCE(cloud_sync_activated, false) FROM workspace_storage WHERE workspace_id=$1`, wsID).Scan(&bu, &bl, &act); err == nil {
			bytesUsed, bytesLimit, activated = bu, bl, act
		}
	}
	if cloudURL := services.CloudDatabaseURL(); cloudURL != "" {
		if cfg, err := pgxpool.ParseConfig(cloudURL); err == nil {
			cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
			if cloud, err := pgxpool.NewWithConfig(ctx, cfg); err == nil {
				defer cloud.Close()
				read(cloud)
			}
		}
	} else {
		read(h.pool)
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace":   slug,
		"bytes_used":  bytesUsed,
		"bytes_limit": bytesLimit,
		"activated":   activated,
		"over_limit":  bytesUsed > bytesLimit,
	})
}

// ---- cloud-copy fallback readers (used when local files are absent) ----

// dbWorkspaces lists workspaces that exist only as a synced cloud copy.
func (h *KnowledgeHandler) dbWorkspaces(ctx context.Context) []gin.H {
	if h.pool == nil {
		return nil
	}
	rows, err := h.pool.Query(ctx, `SELECT DISTINCT workspace_slug FROM knowledge_documents ORDER BY workspace_slug`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var s string
		if rows.Scan(&s) == nil {
			out = append(out, gin.H{"slug": s, "name": titleFromName(s)})
		}
	}
	return out
}

// dbTree builds the file tree for a workspace from its synced cloud copy.
func (h *KnowledgeHandler) dbTree(ctx context.Context, slug string) ([]kbNode, bool) {
	if h.pool == nil {
		return nil, false
	}
	rows, err := h.pool.Query(ctx, `SELECT path, title FROM knowledge_documents WHERE workspace_slug=$1 ORDER BY path`, slug)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	var items []kbDoc
	for rows.Next() {
		var p, t string
		if rows.Scan(&p, &t) == nil {
			items = append(items, kbDoc{Path: p, Title: t})
		}
	}
	if len(items) == 0 {
		return nil, false
	}
	return treeFromPaths(items), true
}

// dbFile returns the raw markdown for one document from the synced cloud copy.
func (h *KnowledgeHandler) dbFile(ctx context.Context, slug, path string) (string, bool) {
	if h.pool == nil || path == "" {
		return "", false
	}
	var body string
	err := h.pool.QueryRow(ctx, `SELECT body FROM knowledge_documents WHERE workspace_slug=$1 AND path=$2`, slug, path).Scan(&body)
	if err != nil {
		return "", false
	}
	return body, true
}

// treeFromPaths assembles a nested kbNode tree from flat document paths,
// matching the shape produced by walk() so the frontend renders identically.
func treeFromPaths(items []kbDoc) []kbNode {
	type tnode struct {
		name, path, typ, title string
		children               map[string]*tnode
		order                  []string
	}
	root := &tnode{children: map[string]*tnode{}}
	for _, it := range items {
		parts := strings.Split(it.Path, "/")
		cur := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			child, ok := cur.children[part]
			if !ok {
				child = &tnode{name: part, path: strings.Join(parts[:i+1], "/"), children: map[string]*tnode{}}
				if i == len(parts)-1 {
					child.typ = "file"
					child.title = it.Title
				} else {
					child.typ = "dir"
				}
				cur.children[part] = child
				cur.order = append(cur.order, part)
			}
			cur = child
		}
	}
	var conv func(n *tnode) []kbNode
	conv = func(n *tnode) []kbNode {
		var nodes []kbNode
		for _, k := range n.order {
			ch := n.children[k]
			kn := kbNode{Name: ch.name, Path: ch.path, Type: ch.typ, Title: ch.title}
			if ch.typ == "dir" {
				kn.Children = conv(ch)
			}
			nodes = append(nodes, kn)
		}
		sort.SliceStable(nodes, func(i, j int) bool {
			if nodes[i].Type != nodes[j].Type {
				return nodes[i].Type == "dir"
			}
			return nodes[i].Name < nodes[j].Name
		})
		return nodes
	}
	return conv(root)
}
