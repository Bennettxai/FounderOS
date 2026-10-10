package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// AssetsHandler manages the per-workspace brand/media library: images, logos,
// videos, documents and links shared across projects and campaigns. Files are
// stored on disk and served back through /assets/:id/raw; "link" assets just
// carry an external URL. Workspace-scoped via X-Workspace-ID.
type AssetsHandler struct {
	pool *pgxpool.Pool
}

func NewAssetsHandler(pool *pgxpool.Pool) *AssetsHandler {
	return &AssetsHandler{pool: pool}
}

const (
	assetsFilesDir   = "./uploads/assets"
	maxAssetFileSize = 100 << 20 // 100 MB
)

type asset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	Tags      string    `json:"tags"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *AssetsHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
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

// normalizeAssetKind keeps kind within the allowed set.
func normalizeAssetKind(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "image":
		return "image"
	case "logo":
		return "logo"
	case "video":
		return "video"
	case "document":
		return "document"
	case "link":
		return "link"
	default:
		return "file"
	}
}

// ListAssets returns the workspace's assets. GET /api/v1/assets
func (h *AssetsHandler) ListAssets(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	var err error
	if q != "" {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, url, mime_type, size_bytes, tags, created_by, created_at, updated_at
			FROM   assets
			WHERE  workspace_id = $1
			AND   (name ILIKE '%'||$2||'%' OR tags ILIKE '%'||$2||'%' OR kind ILIKE '%'||$2||'%')
			ORDER  BY created_at DESC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, name, kind, url, mime_type, size_bytes, tags, created_by, created_at, updated_at
			FROM   assets
			WHERE  workspace_id = $1
			ORDER  BY created_at DESC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list assets", err)
		return
	}
	defer rows.Close()

	assets := make([]asset, 0)
	for rows.Next() {
		var a asset
		var id uuid.UUID
		if err := rows.Scan(&id, &a.Name, &a.Kind, &a.URL, &a.MimeType, &a.SizeBytes, &a.Tags, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan asset", err)
			return
		}
		a.ID = id.String()
		assets = append(assets, a)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate assets", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"assets": assets, "count": len(assets)})
}

type assetLinkInput struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
	Tags string `json:"tags"`
}

// CreateAssetLink adds a link-style asset (external URL, no file upload).
// POST /api/v1/assets
func (h *AssetsHandler) CreateAssetLink(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	var in assetLinkInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.URL) == "" {
		RespondBadRequestErr(c, "name and url are required")
		return
	}

	kind := normalizeAssetKind(in.Kind)
	if kind == "file" {
		kind = "link"
	}

	a, err := h.insertAsset(c, wsID, user.ID, strings.TrimSpace(in.Name), kind, strings.TrimSpace(in.URL), "", "", 0, strings.TrimSpace(in.Tags))
	if err != nil {
		RespondInternalErr(c, "create asset", err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

// UploadAsset saves an uploaded file to disk and records it.
// POST /api/v1/assets/upload  (multipart: file, optional name/kind/tags)
func (h *AssetsHandler) UploadAsset(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAssetFileSize)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		RespondBadRequestErr(c, "file is required")
		return
	}
	defer file.Close()

	// Sanitize filename, prefix with a uuid so uploads never collide.
	origName := filepath.Base(header.Filename)
	safeName := strings.ReplaceAll(origName, " ", "_")
	fileID := uuid.New().String()
	dir := filepath.Join(assetsFilesDir, wsID.String())
	if err := os.MkdirAll(dir, 0755); err != nil {
		RespondInternalErr(c, "create upload dir", err)
		return
	}
	storagePath := filepath.Join(dir, fileID+"_"+safeName)
	out, err := os.Create(storagePath)
	if err != nil {
		RespondInternalErr(c, "create file", err)
		return
	}
	defer out.Close()
	written, err := io.Copy(out, file)
	if err != nil {
		RespondInternalErr(c, "write file", err)
		return
	}

	mime := header.Header.Get("Content-Type")
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		name = origName
	}
	kind := c.PostForm("kind")
	if kind == "" {
		kind = kindFromMime(mime)
	} else {
		kind = normalizeAssetKind(kind)
	}
	tags := strings.TrimSpace(c.PostForm("tags"))

	a, err := h.insertAsset(c, wsID, user.ID, name, kind, "", storagePath, mime, written, tags)
	if err != nil {
		_ = os.Remove(storagePath)
		RespondInternalErr(c, "record asset", err)
		return
	}
	// url points at the raw-serving route so the client can render it. Use the
	// versioned path because the dev proxy only forwards /api/v1/*.
	a.URL = fmt.Sprintf("/api/v1/assets/%s/raw", a.ID)
	if _, err := h.pool.Exec(c.Request.Context(),
		`UPDATE assets SET url=$2 WHERE id=$1`, a.ID, a.URL); err != nil {
		RespondInternalErr(c, "finalize asset url", err)
		return
	}
	c.JSON(http.StatusCreated, a)
}

// insertAsset writes one asset row and returns it.
func (h *AssetsHandler) insertAsset(c *gin.Context, wsID uuid.UUID, userID, name, kind, url, storagePath, mime string, size int64, tags string) (asset, error) {
	var a asset
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO assets (workspace_id, name, kind, url, storage_path, mime_type, size_bytes, tags, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, kind, url, mime_type, size_bytes, tags, created_by, created_at, updated_at
	`, wsID, name, kind, url, storagePath, mime, size, tags, userID).
		Scan(&id, &a.Name, &a.Kind, &a.URL, &a.MimeType, &a.SizeBytes, &a.Tags, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return asset{}, err
	}
	a.ID = id.String()
	return a, nil
}

type assetUpdateInput struct {
	Name string `json:"name"`
	Tags string `json:"tags"`
	Kind string `json:"kind"`
}

// UpdateAsset edits an asset's name/tags/kind. PUT /api/v1/assets/:id
func (h *AssetsHandler) UpdateAsset(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid asset id")
		return
	}
	var in assetUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var a asset
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE assets
		SET    name       = COALESCE(NULLIF($3,''), name),
		       tags       = $4,
		       kind       = COALESCE(NULLIF($5,''), kind),
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, kind, url, mime_type, size_bytes, tags, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Tags), normalizeAssetKind(in.Kind)).
		Scan(&rid, &a.Name, &a.Kind, &a.URL, &a.MimeType, &a.SizeBytes, &a.Tags, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "asset")
		return
	}
	a.ID = rid.String()
	c.JSON(http.StatusOK, a)
}

// DeleteAsset removes an asset and its file. DELETE /api/v1/assets/:id
func (h *AssetsHandler) DeleteAsset(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid asset id")
		return
	}
	var storagePath string
	err = h.pool.QueryRow(c.Request.Context(),
		`DELETE FROM assets WHERE id=$1 AND workspace_id=$2 RETURNING storage_path`, id, wsID).Scan(&storagePath)
	if err != nil {
		RespondNotFoundErr(c, "asset")
		return
	}
	if storagePath != "" {
		_ = os.Remove(storagePath)
	}
	c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
}

// ServeAssetRaw streams an uploaded asset file. GET /api/v1/assets/:id/raw
// Authenticated by session cookie ONLY (no X-Workspace-ID) so <img>/<video>
// tags — which cannot send custom headers — can load it. The workspace is
// derived from the asset row and membership is verified against it.
func (h *AssetsHandler) ServeAssetRaw(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid asset id")
		return
	}
	var storagePath, mime string
	var wsID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT a.storage_path, a.mime_type, a.workspace_id
		   FROM assets a
		   JOIN workspace_members m ON m.workspace_id = a.workspace_id
		  WHERE a.id = $1 AND m.user_id = $2 AND m.status = 'active'`,
		id, user.ID).Scan(&storagePath, &mime, &wsID)
	if err != nil || storagePath == "" {
		RespondNotFoundErr(c, "asset file")
		return
	}
	if _, err := os.Stat(storagePath); err != nil {
		RespondNotFoundErr(c, "asset file")
		return
	}
	if mime != "" {
		c.Header("Content-Type", mime)
	}
	c.File(storagePath)
}

// kindFromMime infers an asset kind from a content type.
func kindFromMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "application/pdf"),
		strings.HasPrefix(mime, "application/msword"),
		strings.HasPrefix(mime, "application/vnd"):
		return "document"
	default:
		return "file"
	}
}
