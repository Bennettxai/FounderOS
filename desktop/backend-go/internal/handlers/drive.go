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

// DriveHandler manages the per-workspace file store: files organized into simple
// folder paths (e.g. "Contracts" or "Contracts/2026"). Files are stored on disk
// and served back through /drive/:id/raw. Workspace-scoped via X-Workspace-ID.
type DriveHandler struct {
	pool *pgxpool.Pool
}

func NewDriveHandler(pool *pgxpool.Pool) *DriveHandler {
	return &DriveHandler{pool: pool}
}

const (
	driveFilesDir    = "./uploads/drive"
	maxDriveFileSize = 100 << 20 // 100 MB
)

type driveFile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Folder    string    `json:"folder"`
	MimeType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	URL       string    `json:"url"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *DriveHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// normalizeFolder trims a folder path into a simple, slash-separated form with no
// leading/trailing slashes and no empty segments. Returns "" for the root.
func normalizeFolder(s string) string {
	segs := strings.Split(s, "/")
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		out = append(out, seg)
	}
	return strings.Join(out, "/")
}

// ListDriveFiles returns the workspace's files. GET /api/v1/drive
// Optional filters: ?folder=<path> (exact folder) and ?q=<search>.
func (h *DriveHandler) ListDriveFiles(c *gin.Context) {
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
	folder := normalizeFolder(c.Query("folder"))
	hasFolder := c.Query("folder") != ""

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, name, folder, mime_type, size_bytes, url, created_by, created_at, updated_at
		FROM   drive_files
		WHERE  workspace_id = $1
		AND   ($2 = '' OR name ILIKE '%'||$2||'%' OR folder ILIKE '%'||$2||'%')
		AND   (NOT $3 OR folder = $4)
		ORDER  BY folder ASC, name ASC
	`, wsID, q, hasFolder, folder)
	if err != nil {
		RespondInternalErr(c, "list drive files", err)
		return
	}
	defer rows.Close()

	files := make([]driveFile, 0)
	for rows.Next() {
		var f driveFile
		var id uuid.UUID
		if err := rows.Scan(&id, &f.Name, &f.Folder, &f.MimeType, &f.SizeBytes, &f.URL, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt); err != nil {
			RespondInternalErr(c, "scan drive file", err)
			return
		}
		f.ID = id.String()
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate drive files", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": files, "count": len(files)})
}

// ListDriveFolders returns the distinct non-empty folders in the workspace.
// GET /api/v1/drive/folders
func (h *DriveHandler) ListDriveFolders(c *gin.Context) {
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

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT DISTINCT folder
		FROM   drive_files
		WHERE  workspace_id = $1 AND folder <> ''
		ORDER  BY folder ASC
	`, wsID)
	if err != nil {
		RespondInternalErr(c, "list drive folders", err)
		return
	}
	defer rows.Close()

	folders := make([]string, 0)
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			RespondInternalErr(c, "scan folder", err)
			return
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate folders", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"folders": folders, "count": len(folders)})
}

// UploadDriveFile saves an uploaded file to disk and records it.
// POST /api/v1/drive/upload  (multipart: file, optional folder/name)
func (h *DriveHandler) UploadDriveFile(c *gin.Context) {
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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDriveFileSize)
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
	dir := filepath.Join(driveFilesDir, wsID.String())
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
	folder := normalizeFolder(c.PostForm("folder"))

	f, err := h.insertDriveFile(c, wsID, user.ID, name, folder, mime, storagePath, written)
	if err != nil {
		_ = os.Remove(storagePath)
		RespondInternalErr(c, "record drive file", err)
		return
	}
	// url points at the raw-serving route so the client can download it. Use the
	// versioned path because the dev proxy only forwards /api/v1/*.
	f.URL = fmt.Sprintf("/api/v1/drive/%s/raw", f.ID)
	if _, err := h.pool.Exec(c.Request.Context(),
		`UPDATE drive_files SET url=$2 WHERE id=$1`, f.ID, f.URL); err != nil {
		RespondInternalErr(c, "finalize drive url", err)
		return
	}
	c.JSON(http.StatusCreated, f)
}

// insertDriveFile writes one drive_files row and returns it.
func (h *DriveHandler) insertDriveFile(c *gin.Context, wsID uuid.UUID, userID, name, folder, mime, storagePath string, size int64) (driveFile, error) {
	var f driveFile
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO drive_files (workspace_id, name, folder, mime_type, size_bytes, storage_path, url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, '', $7)
		RETURNING id, name, folder, mime_type, size_bytes, url, created_by, created_at, updated_at
	`, wsID, name, folder, mime, size, storagePath, userID).
		Scan(&id, &f.Name, &f.Folder, &f.MimeType, &f.SizeBytes, &f.URL, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return driveFile{}, err
	}
	f.ID = id.String()
	return f, nil
}

type driveUpdateInput struct {
	Name   string `json:"name"`
	Folder string `json:"folder"`
}

// UpdateDriveFile renames and/or moves a file. PUT /api/v1/drive/:id
func (h *DriveHandler) UpdateDriveFile(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid file id")
		return
	}
	var in driveUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var f driveFile
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE drive_files
		SET    name       = COALESCE(NULLIF($3,''), name),
		       folder     = $4,
		       updated_at = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, name, folder, mime_type, size_bytes, url, created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Name), normalizeFolder(in.Folder)).
		Scan(&rid, &f.Name, &f.Folder, &f.MimeType, &f.SizeBytes, &f.URL, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		RespondNotFoundErr(c, "file")
		return
	}
	f.ID = rid.String()
	c.JSON(http.StatusOK, f)
}

// DeleteDriveFile removes a file and its row. DELETE /api/v1/drive/:id
func (h *DriveHandler) DeleteDriveFile(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid file id")
		return
	}
	var storagePath string
	err = h.pool.QueryRow(c.Request.Context(),
		`DELETE FROM drive_files WHERE id=$1 AND workspace_id=$2 RETURNING storage_path`, id, wsID).Scan(&storagePath)
	if err != nil {
		RespondNotFoundErr(c, "file")
		return
	}
	if storagePath != "" {
		_ = os.Remove(storagePath)
	}
	c.JSON(http.StatusOK, gin.H{"message": "file deleted"})
}

// ServeDriveRaw streams an uploaded file. GET /api/v1/drive/:id/raw
// Authenticated by session cookie ONLY (no X-Workspace-ID) so <img> tags and
// direct-download links — which cannot send custom headers — can load it. The
// workspace is derived from the file row and membership is verified against it.
func (h *DriveHandler) ServeDriveRaw(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid file id")
		return
	}
	var storagePath, mime string
	var wsID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT d.storage_path, d.mime_type, d.workspace_id
		   FROM drive_files d
		   JOIN workspace_members m ON m.workspace_id = d.workspace_id
		  WHERE d.id = $1 AND m.user_id = $2 AND m.status = 'active'`,
		id, user.ID).Scan(&storagePath, &mime, &wsID)
	if err != nil || storagePath == "" {
		RespondNotFoundErr(c, "file")
		return
	}
	if _, err := os.Stat(storagePath); err != nil {
		RespondNotFoundErr(c, "file")
		return
	}
	if mime != "" {
		c.Header("Content-Type", mime)
	}
	c.File(storagePath)
}
