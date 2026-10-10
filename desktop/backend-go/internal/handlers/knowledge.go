package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// KnowledgeHandler serves the workspace markdown file tree (Plane 1: source of
// truth) for the Knowledge module. Markdown files on disk are the canonical KB
// content; this handler exposes the tree and file bodies, workspace-scoped.
type KnowledgeHandler struct {
	pool *pgxpool.Pool
	root string
}

// NewKnowledgeHandler constructs a KnowledgeHandler.
// KNOWLEDGE_WORKSPACES_ROOT should point at the parent folder containing workspace
// folders.
// If only OPTIMAL_ENGINE_ROOT is set and it points at one workspace, use its
// parent so workspace context stays segmented by slug.
func NewKnowledgeHandler(pool *pgxpool.Pool) *KnowledgeHandler {
	root := os.Getenv("KNOWLEDGE_WORKSPACES_ROOT")
	if root == "" {
		root = os.Getenv("OPTIMAL_ENGINE_ROOT")
		if looksLikeWorkspaceDir(root) {
			root = filepath.Dir(root)
		}
	}
	if root == "" {
		if home, err := os.UserHomeDir(); err == nil {
			root = filepath.Join(home, "code", "OptimalOS", "workspaces")
		}
	}
	return &KnowledgeHandler{pool: pool, root: root}
}

func looksLikeWorkspaceDir(dir string) bool {
	if dir == "" {
		return false
	}
	for _, marker := range []string{".optimal", ".wiki", "operating-system", "nodes", "rhythm"} {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// isWorkspaceProjection only accepts directories that have the durable
// workspace projection markers. The Engine can create temporary asset folders
// during diagnostics, but an arbitrary directory must never become a selectable
// BusinessOS workspace simply because it exists below the workspace root.
func isWorkspaceProjection(dir string) bool {
	for _, marker := range []string{".wiki", ".optimal", "workspace.yaml", "organization.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

// isInternalWorkspaceProjection identifies Engine routing and benchmark scopes
// that are storage details, not operating contexts a person should promote into
// BusinessOS.
func isInternalWorkspaceProjection(slug string) bool {
	if slug == "inbox" || slug == "team" || slug == "knowledge-intake" {
		return true
	}
	return strings.HasPrefix(slug, "default-") || strings.HasPrefix(slug, "benchmark-")
}

// RegisterKnowledgeRoutes wires /api/knowledge routes.
func RegisterKnowledgeRoutes(api *gin.RouterGroup, h *KnowledgeHandler, auth gin.HandlerFunc) {
	g := api.Group("/knowledge")
	g.Use(auth, middleware.RequireAuth())
	{
		g.GET("/workspaces", h.GetWorkspaces)
		g.GET("/tree", h.GetTree)
		g.GET("/file", h.GetFile)
		g.PUT("/file", h.PutFile)
		g.POST("/sync-to-cloud", h.SyncToCloud)
		g.GET("/pull", h.PullFromCloud)
		g.GET("/sources", h.KnowledgeSources)
		g.GET("/storage", h.GetStorage)
		g.POST("/cloud/activate", h.ActivateCloudSync) // opt-in gate: turn on cloud sync
		g.DELETE("/cloud", h.DeleteCloudCopy)          // right-to-be-forgotten: wipe synced cloud copy
	}
}

// putFileReq is the body for creating/updating a markdown file.
type putFileReq struct {
	Workspace string `json:"workspace"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

// PutFile writes (creates or overwrites) a markdown file in a workspace,
// creating parent folders as needed. Path-traversal safe, .md only.
func (h *KnowledgeHandler) PutFile(c *gin.Context) {
	var req putFileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	dir, ok := h.workspaceDir(req.Workspace)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace"})
		return
	}
	rel := req.Path
	if rel == "" || strings.Contains(rel, "..") || !strings.HasSuffix(strings.ToLower(rel), ".md") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	abs, err := filepath.Abs(full)
	if err != nil || !strings.HasPrefix(abs, dir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create folder"})
		return
	}
	if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not write file"})
		return
	}
	info, _ := os.Stat(abs)
	c.JSON(http.StatusOK, gin.H{"ok": true, "path": filepath.ToSlash(rel), "modified": info.ModTime()})
}

// GetWorkspaces lists the workspace folders available on disk.
func (h *KnowledgeHandler) GetWorkspaces(c *gin.Context) {
	ws := []gin.H{}
	if entries, err := os.ReadDir(h.root); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if isInternalWorkspaceProjection(e.Name()) {
				continue
			}
			// Accept real directories AND symlinks that resolve to a directory,
			// so a workspace attached via symlink (e.g. dany-os -> agency-miosa)
			// shows as mounted, not just directly-created folders.
			isDir := e.IsDir()
			if !isDir && e.Type()&os.ModeSymlink != 0 {
				if info, err := os.Stat(filepath.Join(h.root, e.Name())); err == nil && info.IsDir() {
					isDir = true
				}
			}
			if isDir && isWorkspaceProjection(filepath.Join(h.root, e.Name())) {
				ws = append(ws, gin.H{"slug": e.Name(), "name": titleFromName(e.Name())})
			}
		}
	}
	// Merge cloud-synced workspaces the caller is a MEMBER of (teammates without
	// local files). Non-member cloud workspaces are hidden — multi-tenant gate.
	for _, d := range h.dbWorkspaces(c.Request.Context()) {
		slug, _ := d["slug"].(string)
		if slug == "" || !h.callerMemberOfSlug(c, h.pool, slug) {
			continue
		}
		found := false
		for _, w := range ws {
			if w["slug"] == d["slug"] {
				found = true
				break
			}
		}
		if !found {
			ws = append(ws, d)
		}
	}
	sort.SliceStable(ws, func(i, j int) bool { return ws[i]["slug"].(string) < ws[j]["slug"].(string) })
	c.JSON(http.StatusOK, gin.H{"workspaces": ws})
}

// kbNode is a node in the markdown file tree.
type kbNode struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"` // workspace-relative, forward-slash
	Type     string   `json:"type"` // "dir" | "file"
	Title    string   `json:"title,omitempty"`
	Children []kbNode `json:"children,omitempty"`
}

// workspaceDir resolves and validates the directory for a workspace slug,
// guarding against path traversal.
func (h *KnowledgeHandler) workspaceDir(slug string) (string, bool) {
	if slug == "" || strings.ContainsAny(slug, "/\\") || strings.Contains(slug, "..") {
		return "", false
	}
	dir := filepath.Join(h.root, slug)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	rootAbs, _ := filepath.Abs(h.root)
	if !strings.HasPrefix(abs, rootAbs) {
		return "", false
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", false
	}
	return abs, true
}

// GetTree returns the markdown file tree for ?workspace=<slug>.
func (h *KnowledgeHandler) GetTree(c *gin.Context) {
	slug := c.Query("workspace")
	dir, ok := h.workspaceDir(slug)
	if !ok {
		// No local files: serve the synced cloud copy — but only to members of
		// that workspace (multi-tenant gate; the local branch below is ungated).
		if !h.callerMemberOfSlug(c, h.pool, slug) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
			return
		}
		if tree, found := h.dbTree(c.Request.Context(), slug); found {
			c.JSON(http.StatusOK, gin.H{"workspace": slug, "tree": tree})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace"})
		return
	}
	children := h.walk(dir, dir)
	c.JSON(http.StatusOK, gin.H{"workspace": slug, "tree": children})
}

// walk builds the tree under dir, keeping only .md files and folders that
// contain markdown. Hidden entries (dotfiles) are skipped.
func (h *KnowledgeHandler) walk(base, dir string) []kbNode {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var nodes []kbNode
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		rel, _ := filepath.Rel(base, full)
		rel = filepath.ToSlash(rel)
		if e.IsDir() {
			kids := h.walk(base, full)
			if len(kids) > 0 {
				nodes = append(nodes, kbNode{Name: name, Path: rel, Type: "dir", Children: kids})
			}
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		nodes = append(nodes, kbNode{Name: name, Path: rel, Type: "file", Title: titleFromName(name)})
	}
	// Dirs first, then files, each alphabetical.
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == "dir"
		}
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}

// GetFile returns the raw markdown for ?workspace=<slug>&path=<rel>.
func (h *KnowledgeHandler) GetFile(c *gin.Context) {
	slug := c.Query("workspace")
	dir, ok := h.workspaceDir(slug)
	if !ok {
		// No local files: serve the synced cloud copy — members only.
		if !h.callerMemberOfSlug(c, h.pool, slug) {
			c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
			return
		}
		rel := c.Query("path")
		if body, found := h.dbFile(c.Request.Context(), slug, rel); found {
			c.JSON(http.StatusOK, gin.H{"workspace": slug, "path": rel, "content": body})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace"})
		return
	}
	rel := c.Query("path")
	if rel == "" || strings.Contains(rel, "..") || !strings.HasSuffix(strings.ToLower(rel), ".md") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	abs, err := filepath.Abs(full)
	if err != nil || !strings.HasPrefix(abs, dir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	info, _ := os.Stat(abs)
	c.JSON(http.StatusOK, gin.H{
		"workspace": slug,
		"path":      filepath.ToSlash(rel),
		"content":   string(data),
		"modified":  info.ModTime(),
	})
}

// titleFromName derives a readable title from a filename, dropping a numeric
// prefix and the .md extension: "00-executive-layer.md" -> "Executive Layer".
func titleFromName(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if i := strings.IndexByte(base, '-'); i > 0 && isAllDigits(base[:i]) {
		base = base[i+1:]
	}
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	return strings.Title(base)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
