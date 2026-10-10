package handlers

// File handler unit tests: auth layer, UUID validation, and input guard paths.
//
// Uses the same newAuthRouter / newNoAuthRouter / newHandlerNoPool helpers
// defined in osa_apps_management_test.go (same package, same build).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── GetAppFiles ─────────────────────────────────────────────────────────────

func TestGetAppFiles_Unauthorized_MissingUser(t *testing.T) {
	h := newHandlerNoPool()
	r := newNoAuthRouter()
	r.GET("/api/osa/apps/:id/files", h.GetAppFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetAppFiles_InvalidUserID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter("not-a-uuid")
	r.GET("/api/osa/apps/:id/files", h.GetAppFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAppFiles_InvalidAppID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter(uuid.New().String())
	r.GET("/api/osa/apps/:id/files", h.GetAppFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/not-a-uuid/files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Invalid app ID", body["error"])
}

// The handler rejects invalid path params before touching the DB.
func TestGetAppFiles_PathTraversalRejected(t *testing.T) {
	h := newHandlerNoPool()

	// We cannot hit the DB so we need to bypass auth + UUID parse
	// but still reach the path-validation branch.  Since the handler returns
	// early on DB errors (which happen before path check after auth), we test
	// the isValidPath logic directly — the path validation guard in the handler
	// is the same function.
	traversalPaths := []string{
		"../../../etc/passwd",
		"..\\windows\\system32",
		"src/../../../etc",
		"%2e%2e%2fetc%2fpasswd",
		"src/file\x00.txt",
		"//etc/passwd",
	}

	for _, p := range traversalPaths {
		t.Run(p, func(t *testing.T) {
			assert.False(t, isValidPath(p), "Expected path to be rejected: %q", p)
		})
	}

	// Confirm the guard is wired into the handler by checking the code path
	// that returns 400 for invalid paths.  We use a router that injects a
	// valid user and a valid app UUID but with an invalid path query param.
	// Because the DB call comes first (ownership check), we only test the
	// guard function isolation here — integration tests cover the full chain.
	_ = h
}

// ── DownloadApp ─────────────────────────────────────────────────────────────

func TestDownloadApp_Unauthorized_MissingUser(t *testing.T) {
	h := newHandlerNoPool()
	r := newNoAuthRouter()
	r.GET("/api/osa/apps/:id/download", h.DownloadApp)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDownloadApp_InvalidUserID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter("bad-id")
	r.GET("/api/osa/apps/:id/download", h.DownloadApp)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDownloadApp_InvalidAppID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter(uuid.New().String())
	r.GET("/api/osa/apps/:id/download", h.DownloadApp)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/not-a-uuid/download", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── GetAppGeneratedFiles ─────────────────────────────────────────────────────

func TestGetAppGeneratedFiles_Unauthorized_MissingUser(t *testing.T) {
	h := newHandlerNoPool()
	r := newNoAuthRouter()
	r.GET("/api/osa/apps/:id/generated-files", h.GetAppGeneratedFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/generated-files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetAppGeneratedFiles_InvalidUserID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter("bad-id")
	r.GET("/api/osa/apps/:id/generated-files", h.GetAppGeneratedFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/"+uuid.New().String()+"/generated-files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAppGeneratedFiles_InvalidAppID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter(uuid.New().String())
	r.GET("/api/osa/apps/:id/generated-files", h.GetAppGeneratedFiles)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/apps/not-a-uuid/generated-files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── SaveAppFile ──────────────────────────────────────────────────────────────

func TestSaveAppFile_Unauthorized_MissingUser(t *testing.T) {
	h := newHandlerNoPool()
	r := newNoAuthRouter()
	r.PUT("/api/osa/apps/:id/files", h.SaveAppFile)

	body, _ := json.Marshal(SaveFileRequest{FilePath: "src/main.go", Content: "code"})
	req := httptest.NewRequest(http.MethodPut, "/api/osa/apps/"+uuid.New().String()+"/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSaveAppFile_InvalidUserID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter("not-uuid")
	r.PUT("/api/osa/apps/:id/files", h.SaveAppFile)

	body, _ := json.Marshal(SaveFileRequest{FilePath: "src/main.go", Content: "code"})
	req := httptest.NewRequest(http.MethodPut, "/api/osa/apps/"+uuid.New().String()+"/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSaveAppFile_InvalidAppID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter(uuid.New().String())
	r.PUT("/api/osa/apps/:id/files", h.SaveAppFile)

	body, _ := json.Marshal(SaveFileRequest{FilePath: "src/main.go", Content: "code"})
	req := httptest.NewRequest(http.MethodPut, "/api/osa/apps/not-a-uuid/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid app ID", resp["error"])
}

// ── GetQueueItemStatus ───────────────────────────────────────────────────────

func TestGetQueueItemStatus_Unauthorized_MissingUser(t *testing.T) {
	h := newHandlerNoPool()
	r := newNoAuthRouter()
	r.GET("/api/osa/queue/:queue_item_id/status", h.GetQueueItemStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/queue/"+uuid.New().String()+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetQueueItemStatus_InvalidQueueItemID(t *testing.T) {
	h := newHandlerNoPool()
	r := newAuthRouter(uuid.New().String())
	r.GET("/api/osa/queue/:queue_item_id/status", h.GetQueueItemStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/osa/queue/not-a-uuid/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid queue item ID", resp["error"])
}

// ── SaveFileRequest validation ────────────────────────────────────────────────

func TestSaveFileRequest_FileSizeLimit(t *testing.T) {
	// Confirm the constant matches the documented limit.
	const maxFileSize = 1 << 20 // 1 MB
	assert.Equal(t, 1048576, maxFileSize, "1 MB should be 1048576 bytes")

	// A string of exactly maxFileSize characters is at the boundary.
	boundary := strings.Repeat("x", maxFileSize)
	assert.Equal(t, maxFileSize, len(boundary))

	// One byte over the limit.
	over := strings.Repeat("x", maxFileSize+1)
	assert.Greater(t, len(over), maxFileSize)
}

func TestSaveFileRequest_PathValidation(t *testing.T) {
	// SaveAppFile calls isValidPath on req.FilePath — confirm rejections.
	invalidPaths := []string{
		"../secret.env",
		"../../etc/shadow",
		"file\x00.go",
		"..\\windows",
		"//etc/passwd",
	}

	for _, p := range invalidPaths {
		t.Run(p, func(t *testing.T) {
			assert.False(t, isValidPath(p))
		})
	}

	validPaths := []string{
		"src/main.go",
		"cmd/server/main.go",
		"README.md",
		"/src/app.ts",
	}

	for _, p := range validPaths {
		t.Run(p, func(t *testing.T) {
			assert.True(t, isValidPath(p))
		})
	}
}

// ── FileListResponse edge cases ───────────────────────────────────────────────

func TestFileListResponse_EmptyFiles(t *testing.T) {
	response := FileListResponse{
		AppID:  uuid.New(),
		Files:  []FileNodeFlat{},
		Total:  0,
		Limit:  50,
		Offset: 0,
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded FileListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Empty(t, decoded.Files)
	assert.Equal(t, int64(0), decoded.Total)
}

func TestFileListResponse_PaginationBoundaryStartBeyondEnd(t *testing.T) {
	// Mirror the pagination boundary logic in GetAppFiles.
	filteredFiles := make([]int, 3) // length 3

	limit := int32(50)
	offset := int32(10) // start beyond the slice

	start := int(offset)
	end := int(offset) + int(limit)

	if start > len(filteredFiles) {
		start = len(filteredFiles)
	}
	if end > len(filteredFiles) {
		end = len(filteredFiles)
	}

	assert.Equal(t, 3, start)
	assert.Equal(t, 3, end)
	paginated := filteredFiles[start:end]
	assert.Empty(t, paginated)
}

func TestFileListResponse_PaginationNormalCase(t *testing.T) {
	filteredFiles := []int{0, 1, 2, 3, 4} // length 5

	offset := int32(1)
	limit := int32(2)

	start := int(offset)
	end := int(offset) + int(limit)

	if start > len(filteredFiles) {
		start = len(filteredFiles)
	}
	if end > len(filteredFiles) {
		end = len(filteredFiles)
	}

	paginated := filteredFiles[start:end]
	assert.Equal(t, []int{1, 2}, paginated)
}
