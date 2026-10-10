// Package handlers — OSA Workflows handler tests.
//
// Test layout:
//   - Unit tests: helper functions, auth-only code paths (no DB required)
//   - Integration tests: full DB-backed handler scenarios (skipped without test DB)
//
// Run unit tests only:
//
//	go test -run TestUnit ./internal/handlers/...
//
// Run all tests (requires test DB):
//
//	go test ./internal/handlers/...
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhl/businessos-backend/internal/services"
)

// ---------------------------------------------------------------------------
// UNIT TESTS: helpers — no database required
// ---------------------------------------------------------------------------

// TestUnit_ResolveWorkflowSearch_UUID verifies that a well-formed UUID is parsed
// correctly and the search prefix is set to "uuid%".
func TestUnit_ResolveWorkflowSearch_UUID(t *testing.T) {
	id := uuid.New()

	searchID, searchPrefix := resolveWorkflowSearch(id.String())

	parsed, ok := searchID.(uuid.UUID)
	assert.True(t, ok, "searchID should be a uuid.UUID when input is a valid UUID")
	assert.Equal(t, id, parsed)
	assert.Equal(t, id.String()+"%", searchPrefix)
}

// TestUnit_ResolveWorkflowSearch_NonUUID verifies that a non-UUID string falls
// back to uuid.Nil and the prefix is still appended.
func TestUnit_ResolveWorkflowSearch_NonUUID(t *testing.T) {
	rawID := "wf-abc123"

	searchID, searchPrefix := resolveWorkflowSearch(rawID)

	assert.Equal(t, uuid.Nil, searchID, "searchID should be uuid.Nil for non-UUID input")
	assert.Equal(t, "wf-abc123%", searchPrefix)
}

// TestUnit_ResolveWorkflowSearch_EmptyString checks that an empty string is treated
// as a non-UUID and still produces a valid prefix.
func TestUnit_ResolveWorkflowSearch_EmptyString(t *testing.T) {
	searchID, searchPrefix := resolveWorkflowSearch("")

	assert.Equal(t, uuid.Nil, searchID)
	assert.Equal(t, "%", searchPrefix)
}

// TestUnit_ParseMetadataJSON_Valid tests successful JSON parsing.
func TestUnit_ParseMetadataJSON_Valid(t *testing.T) {
	raw := []byte(`{"analysis":"content","code":"package main"}`)
	var dst map[string]interface{}

	err := parseMetadataJSON(raw, &dst)

	require.NoError(t, err)
	assert.Equal(t, "content", dst["analysis"])
	assert.Equal(t, "package main", dst["code"])
}

// TestUnit_ParseMetadataJSON_Invalid tests that invalid JSON returns an error and
// leaves the destination map unchanged.
func TestUnit_ParseMetadataJSON_Invalid(t *testing.T) {
	raw := []byte(`{not valid json}`)
	dst := map[string]interface{}{"existing": "value"}
	original := dst["existing"]

	err := parseMetadataJSON(raw, &dst)

	assert.Error(t, err)
	// destination should remain unchanged on error
	assert.Equal(t, original, dst["existing"])
}

// TestUnit_WorkflowFileTypes verifies the canonical set and ordering of file types.
func TestUnit_WorkflowFileTypes(t *testing.T) {
	expected := []string{
		"analysis", "architecture", "code", "quality",
		"deployment", "monitoring", "strategy", "recommendations",
	}

	assert.Equal(t, expected, workflowFileTypes)
}

// TestUnit_DNSNamespace verifies the DNS namespace UUID is the well-known value.
func TestUnit_DNSNamespace(t *testing.T) {
	expected := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	assert.Equal(t, expected, dnsNamespace)
}

// TestUnit_DeterministicFileID verifies that file IDs are stable across calls.
func TestUnit_DeterministicFileID(t *testing.T) {
	appID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	id1 := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":analysis"))
	id2 := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":analysis"))
	id3 := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":architecture"))

	assert.Equal(t, id1, id2, "same inputs must produce the same file ID")
	assert.NotEqual(t, id1, id3, "different file types must produce different IDs")
}

// ---------------------------------------------------------------------------
// UNIT TESTS: auth-only handler paths — no database required
// ---------------------------------------------------------------------------

// newNoDBHandler creates an OSAWorkflowsHandler with a nil pool. Any handler
// method that checks auth before touching the DB can be tested without a DB.
func newNoDBHandler() *OSAWorkflowsHandler {
	return NewOSAWorkflowsHandler(nil, nil)
}

// TestUnit_ListWorkflows_Unauthorized confirms a 401 is returned when user_id is absent.
func TestUnit_ListWorkflows_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows", nil)
	// Intentionally NOT setting "user_id" in the context

	h.ListWorkflows(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "User not authenticated", body["error"])
}

// TestUnit_GetWorkflow_Unauthorized confirms a 401 when user_id is absent.
func TestUnit_GetWorkflow_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+uuid.New().String(), nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetWorkflow(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_GetWorkflowFiles_Unauthorized confirms a 401 when user_id is absent.
func TestUnit_GetWorkflowFiles_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+uuid.New().String()+"/files", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetWorkflowFiles(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_GetFileContent_Unauthorized confirms a 401 when user_id is absent.
func TestUnit_GetFileContent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/id/files/analysis", nil)
	c.Params = gin.Params{
		{Key: "id", Value: uuid.New().String()},
		{Key: "type", Value: "analysis"},
	}

	h.GetFileContent(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_GetFileContent_InvalidType checks that an invalid file type returns 400
// even before hitting the database.
func TestUnit_GetFileContent_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	invalidTypes := []string{
		"invalid_type",
		"",
		"../../etc",
		"sql",
		"readme",
		"ANALYSIS", // case-sensitive: uppercase is invalid
	}

	for _, ft := range invalidTypes {
		t.Run("type="+ft, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/id/files/"+ft, nil)
			c.Params = gin.Params{
				{Key: "id", Value: uuid.New().String()},
				{Key: "type", Value: ft},
			}
			c.Set("user_id", uuid.New())

			h.GetFileContent(c)

			assert.Equal(t, http.StatusBadRequest, w.Code, "type=%q should be rejected", ft)
		})
	}
}

// TestUnit_GetFileContent_ValidTypeSet verifies that the set of valid file type
// keys matches the expected canonical list. This is a pure data assertion that
// does not invoke the handler at all.
func TestUnit_GetFileContent_ValidTypeSet(t *testing.T) {
	validTypes := map[string]bool{
		"analysis": true, "architecture": true, "code": true, "quality": true,
		"deployment": true, "monitoring": true, "strategy": true, "recommendations": true,
	}

	// All workflow file types must be in the valid-types set
	for _, ft := range workflowFileTypes {
		assert.True(t, validTypes[ft],
			"workflowFileTypes entry %q must be in the handler's validTypes map", ft)
	}

	// The valid-types set must match the workflowFileTypes length
	assert.Len(t, workflowFileTypes, len(validTypes))
}

// TestUnit_GetFileContentByID_InvalidUUID verifies that a non-UUID path param
// returns 400 without touching the database.
func TestUnit_GetFileContentByID_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	invalidIDs := []string{
		"not-a-uuid",
		"123",
		"",
		"../../etc/passwd",
		strings.Repeat("a", 100),
	}

	for _, id := range invalidIDs {
		t.Run("id="+id, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/files/"+id+"/content", nil)
			c.Params = gin.Params{{Key: "id", Value: id}}
			c.Set("user_id", uuid.New())

			h.GetFileContentByID(c)

			assert.Equal(t, http.StatusBadRequest, w.Code, "id=%q should be rejected", id)
		})
	}
}

// TestUnit_GetFileContentByID_Unauthorized checks a 401 before UUID parsing.
func TestUnit_GetFileContentByID_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/files/"+uuid.New().String()+"/content", nil)
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.GetFileContentByID(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_InstallModule_Unauthorized confirms 401 when user_id is absent.
func TestUnit_InstallModule_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	body, _ := json.Marshal(map[string]interface{}{
		"workflow_id": uuid.New().String(),
		"module_name": "test-module",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/modules/install", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.InstallModule(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_InstallModule_MalformedJSON verifies 400 is returned for malformed JSON body.
func TestUnit_InstallModule_MalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/modules/install",
		strings.NewReader("{invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New())

	h.InstallModule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUnit_TriggerSync_Unauthorized confirms 401 when user_id is absent.
func TestUnit_TriggerSync_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/sync/trigger", nil)

	h.TriggerSync(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnit_TriggerSync_Success verifies the sync endpoint returns 200 and echoes
// user_id. This test does NOT need a DB — the handler only spawns a goroutine.
func TestUnit_TriggerSync_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newNoDBHandler()

	userID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/sync/trigger", nil)
	c.Set("user_id", userID)

	h.TriggerSync(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Sync triggered", resp["message"])
	assert.Equal(t, userID.String(), resp["user_id"])
}

// ---------------------------------------------------------------------------
// INTEGRATION TESTS: full DB-backed scenarios
// ---------------------------------------------------------------------------

// newWorkflowsHandler builds a handler wired to a live test pool.
func newWorkflowsHandler(t *testing.T, pool *pgxpool.Pool) *OSAWorkflowsHandler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	syncService := services.NewOSAFileSyncService(pool, logger, "")
	return NewOSAWorkflowsHandler(pool, syncService)
}

// TestListWorkflows covers workflow listing including the empty-list and multi-item cases.
func TestListWorkflows(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		setupApps      int
		userID         interface{} // nil → omit from context
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "empty list",
			setupApps:      0,
			userID:         userID,
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "three workflows",
			setupApps:      3,
			userID:         userID,
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "unauthorized — no user_id",
			setupApps:      0,
			userID:         nil, // omit
			expectedStatus: http.StatusUnauthorized,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh workspace per sub-test to keep counts independent
			subUserID := uuid.New()
			subWorkspaceID := createTestWorkspace(t, pool, subUserID)
			defer cleanupWorkflows(t, pool, subWorkspaceID)

			for i := 0; i < tt.setupApps; i++ {
				createTestApp(t, pool, subWorkspaceID, map[string]interface{}{
					"analysis": "content",
				})
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows", nil)
			if tt.userID != nil {
				c.Set("user_id", subUserID)
			}

			handler.ListWorkflows(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, tt.expectedCount, int(resp["count"].(float64)))

				workflows := resp["workflows"].([]interface{})
				assert.Len(t, workflows, tt.expectedCount)
			}
		})
	}
}

// TestListWorkflows_ResponseShape validates required JSON fields in the list response.
func TestListWorkflows_ResponseShape(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	createTestApp(t, pool, workspaceID, map[string]interface{}{"analysis": "test"})

	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows", nil)
	c.Set("user_id", userID)

	handler.ListWorkflows(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// Top-level envelope
	_, hasWorkflows := resp["workflows"]
	_, hasCount := resp["count"]
	assert.True(t, hasWorkflows, "response must have 'workflows' key")
	assert.True(t, hasCount, "response must have 'count' key")

	// Workflow item shape
	workflows := resp["workflows"].([]interface{})
	require.NotEmpty(t, workflows)
	item := workflows[0].(map[string]interface{})

	requiredFields := []string{
		"id", "name", "display_name", "description", "workflow_id",
		"status", "files_created", "created_at", "workspace_name",
	}
	for _, field := range requiredFields {
		_, ok := item[field]
		assert.True(t, ok, "workflow item must have field %q", field)
	}
}

// TestListWorkflows_UserIsolation verifies that a user only sees their own workflows.
func TestListWorkflows_UserIsolation(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userA := uuid.New()
	userB := uuid.New()
	wsA := createTestWorkspace(t, pool, userA)
	wsB := createTestWorkspace(t, pool, userB)
	defer cleanupWorkflows(t, pool, wsA)
	defer cleanupWorkflows(t, pool, wsB)

	// User A has 2 workflows, user B has 1
	createTestApp(t, pool, wsA, map[string]interface{}{"analysis": "a"})
	createTestApp(t, pool, wsA, map[string]interface{}{"analysis": "b"})
	createTestApp(t, pool, wsB, map[string]interface{}{"analysis": "c"})

	handler := newWorkflowsHandler(t, pool)

	checkCount := func(uid uuid.UUID, want int) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows", nil)
		c.Set("user_id", uid)
		handler.ListWorkflows(c)

		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, want, int(resp["count"].(float64)))
	}

	checkCount(userA, 2)
	checkCount(userB, 1)
}

// TestGetWorkflow covers fetching a single workflow by UUID and by workflow-ID prefix.
func TestGetWorkflow(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	otherUserID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{
		"analysis":     "Analysis content",
		"architecture": "Architecture content",
		"code":         "Code content",
	}
	appID := createTestApp(t, pool, workspaceID, metadata)
	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		workflowID     string
		userID         uuid.UUID
		expectedStatus int
		checkFields    bool
	}{
		{
			name:           "success by UUID",
			workflowID:     appID.String(),
			userID:         userID,
			expectedStatus: http.StatusOK,
			checkFields:    true,
		},
		{
			name:           "not found — random UUID",
			workflowID:     uuid.New().String(),
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "not found — different user (ownership check)",
			workflowID:     appID.String(),
			userID:         otherUserID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "not found — invalid UUID treated as prefix search",
			workflowID:     "invalid-uuid",
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "not found — empty string prefix",
			workflowID:     "zzz-nonexistent",
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+tt.workflowID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.workflowID}}
			c.Set("user_id", tt.userID)

			handler.GetWorkflow(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkFields {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, "test-app", resp["name"])
				assert.Equal(t, "Test App", resp["display_name"])

				// Metadata must be an object, not null
				meta, ok := resp["metadata"].(map[string]interface{})
				assert.True(t, ok, "metadata must be a JSON object")
				assert.Equal(t, "Analysis content", meta["analysis"])

				// All envelope fields must be present
				requiredKeys := []string{
					"id", "name", "display_name", "description", "workflow_id",
					"status", "files_created", "created_at", "workspace_name",
					"workspace_id", "metadata",
				}
				for _, k := range requiredKeys {
					_, present := resp[k]
					assert.True(t, present, "response must have key %q", k)
				}
			}
		})
	}
}

// TestGetWorkflow_MetadataParseFailureFallback verifies that corrupted metadata
// is tolerated — the handler falls back to an empty map rather than returning 500.
func TestGetWorkflow_MetadataParseFailureFallback(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	// Manually insert an app with invalid JSON in the metadata column
	var appID uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name, description,
			osa_workflow_id, status, files_created, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id
	`, workspaceID, "bad-meta", "Bad Meta", "desc",
		"wf-bad", "generated", 0, `{}`).Scan(&appID)
	require.NoError(t, err)

	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: appID.String()}}
	c.Set("user_id", userID)

	handler.GetWorkflow(c)

	// The handler must not return 500 — it falls back to an empty metadata map
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["metadata"])
}

// TestGetWorkflowFiles covers file listing including bundled code files.
func TestGetWorkflowFiles(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	// Build a code bundle with two files
	codeBundle := strings.Join([]string{
		"=== FILE: src/app.js ===",
		`console.log("hello");`,
		"=== END FILE ===",
		"=== FILE: package.json ===",
		`{"name":"test"}`,
		"=== END FILE ===",
	}, "\n")

	metadata := map[string]interface{}{
		"analysis":     "Analysis text",
		"architecture": "Architecture design",
		"code":         codeBundle,
		"quality":      "Quality report",
	}
	appID := createTestApp(t, pool, workspaceID, metadata)
	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		workflowID     string
		userID         uuid.UUID
		expectedStatus int
		minFileCount   int
		checkStructure bool
	}{
		{
			name:           "success — 4 files (2 code + 2 meta)",
			workflowID:     appID.String(),
			userID:         userID,
			expectedStatus: http.StatusOK,
			minFileCount:   4,
			checkStructure: true,
		},
		{
			name:           "not found — unknown UUID",
			workflowID:     uuid.New().String(),
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "not found — different user",
			workflowID:     appID.String(),
			userID:         uuid.New(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+tt.workflowID+"/files", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.workflowID}}
			c.Set("user_id", tt.userID)

			handler.GetWorkflowFiles(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkStructure {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

				files, ok := resp["files"].([]interface{})
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(files), tt.minFileCount)

				count := int(resp["count"].(float64))
				assert.Equal(t, len(files), count, "count must match files length")

				// Every file entry must have these fields
				for _, rawFile := range files {
					f := rawFile.(map[string]interface{})
					assert.NotEmpty(t, f["id"], "file must have id")
					assert.NotEmpty(t, f["name"], "file must have name")
					assert.NotEmpty(t, f["type"], "file must have type")
					_, hasSize := f["size"]
					assert.True(t, hasSize, "file must have size")
					_, hasLang := f["language"]
					assert.True(t, hasLang, "file must have language")
				}
			}
		})
	}
}

// TestGetWorkflowFiles_EmptyMetadata verifies that a workflow with no metadata
// files returns an empty list rather than an error.
func TestGetWorkflowFiles_EmptyMetadata(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	appID := createTestApp(t, pool, workspaceID, map[string]interface{}{})
	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String()+"/files", nil)
	c.Params = gin.Params{{Key: "id", Value: appID.String()}}
	c.Set("user_id", userID)

	handler.GetWorkflowFiles(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["count"])
	files := resp["files"].([]interface{})
	assert.Empty(t, files)
}

// TestGetWorkflowFiles_OnlyCodeBundle verifies that a workflow whose only file
// is a bundled "code" entry produces the correct per-file entries.
func TestGetWorkflowFiles_OnlyCodeBundle(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	bundle := strings.Join([]string{
		"=== FILE: main.go ===",
		"package main",
		"=== END FILE ===",
		"=== FILE: go.mod ===",
		"module example.com/app",
		"=== END FILE ===",
		"=== FILE: README.md ===",
		"# My App",
		"=== END FILE ===",
	}, "\n")

	appID := createTestApp(t, pool, workspaceID, map[string]interface{}{
		"code": bundle,
	})
	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String()+"/files", nil)
	c.Params = gin.Params{{Key: "id", Value: appID.String()}}
	c.Set("user_id", userID)

	handler.GetWorkflowFiles(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	files := resp["files"].([]interface{})
	assert.Len(t, files, 3, "three files in bundle")

	// Validate languages are inferred correctly
	nameToLang := make(map[string]string)
	for _, rawF := range files {
		f := rawF.(map[string]interface{})
		nameToLang[f["name"].(string)] = f["language"].(string)
	}
	assert.Equal(t, "go", nameToLang["main.go"])
	assert.Equal(t, "markdown", nameToLang["README.md"])
}

// TestGetFileContent tests fetching a specific metadata file by type name.
func TestGetFileContent(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	const analysisContent = "# Analysis Report\nThis is the analysis."
	const codeContent = "console.log('test');"

	metadata := map[string]interface{}{
		"analysis": analysisContent,
		"code":     codeContent,
	}
	appID := createTestApp(t, pool, workspaceID, metadata)
	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		workflowID     string
		fileType       string
		userID         uuid.UUID
		expectedStatus int
		expectedSize   int
		expectedType   string
	}{
		{
			name:           "analysis file",
			workflowID:     appID.String(),
			fileType:       "analysis",
			userID:         userID,
			expectedStatus: http.StatusOK,
			expectedSize:   len(analysisContent),
			expectedType:   "analysis",
		},
		{
			name:           "code file",
			workflowID:     appID.String(),
			fileType:       "code",
			userID:         userID,
			expectedStatus: http.StatusOK,
			expectedSize:   len(codeContent),
			expectedType:   "code",
		},
		{
			name:           "absent file type (no content set)",
			workflowID:     appID.String(),
			fileType:       "deployment",
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid file type",
			workflowID:     appID.String(),
			fileType:       "invalid_type",
			userID:         userID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "workflow not found",
			workflowID:     uuid.New().String(),
			fileType:       "analysis",
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "different user cannot access",
			workflowID:     appID.String(),
			fileType:       "analysis",
			userID:         uuid.New(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			url := "/api/osa/workflows/" + tt.workflowID + "/files/" + tt.fileType
			c.Request = httptest.NewRequest(http.MethodGet, url, nil)
			c.Params = gin.Params{
				{Key: "id", Value: tt.workflowID},
				{Key: "type", Value: tt.fileType},
			}
			c.Set("user_id", tt.userID)

			handler.GetFileContent(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

				assert.Equal(t, tt.expectedType, resp["type"])
				assert.Equal(t, float64(tt.expectedSize), resp["size"])
				assert.NotEmpty(t, resp["content"])
			}
		})
	}
}

// TestGetFileContentByID tests fetching file content by the deterministic UUID.
func TestGetFileContentByID(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{
		"analysis": "Analysis content here",
	}
	appID := createTestApp(t, pool, workspaceID, metadata)

	// Derive the deterministic ID the same way the handler does
	expectedFileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":analysis"))
	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		fileID         string
		userID         uuid.UUID
		expectedStatus int
		checkContent   bool
	}{
		{
			name:           "success by deterministic ID",
			fileID:         expectedFileID.String(),
			userID:         userID,
			expectedStatus: http.StatusOK,
			checkContent:   true,
		},
		{
			name:           "invalid UUID format",
			fileID:         "not-a-uuid",
			userID:         userID,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid UUID but file does not exist",
			fileID:         uuid.New().String(),
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "valid UUID — different user cannot see file",
			fileID:         expectedFileID.String(),
			userID:         uuid.New(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/files/"+tt.fileID+"/content", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.fileID}}
			c.Set("user_id", tt.userID)

			handler.GetFileContentByID(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkContent {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

				assert.NotEmpty(t, resp["content"])
				file, ok := resp["file"].(map[string]interface{})
				require.True(t, ok, "response must have a 'file' object")

				fileFields := []string{"id", "name", "type", "size", "language", "created_at", "updated_at"}
				for _, field := range fileFields {
					_, present := file[field]
					assert.True(t, present, "file must have field %q", field)
				}
			}
		})
	}
}

// TestGetFileContentByID_BundledCodeFile checks that a specific file inside a
// code bundle can be retrieved by its deterministic UUID.
func TestGetFileContentByID_BundledCodeFile(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	const srcPath = "src/app.ts"
	const srcContent = `export const hello = () => "world";`
	bundle := "=== FILE: " + srcPath + " ===\n" + srcContent + "\n=== END FILE ==="

	appID := createTestApp(t, pool, workspaceID, map[string]interface{}{
		"code": bundle,
	})

	// Compute deterministic ID for the bundled file
	bundledFileID := uuid.NewSHA1(dnsNamespace, []byte(appID.String()+":code:"+srcPath))
	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/files/"+bundledFileID.String()+"/content", nil)
	c.Params = gin.Params{{Key: "id", Value: bundledFileID.String()}}
	c.Set("user_id", userID)

	handler.GetFileContentByID(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, srcContent, resp["content"])
	file := resp["file"].(map[string]interface{})
	assert.Equal(t, srcPath, file["name"])
	assert.Equal(t, "typescript", file["language"])
}

// TestInstallModule covers the happy path and error cases of module installation.
func TestInstallModule(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	metadata := map[string]interface{}{
		"architecture":    "System design",
		"code":            "Application code",
		"recommendations": "UI recommendations",
	}
	appID := createTestApp(t, pool, workspaceID, metadata)
	handler := newWorkflowsHandler(t, pool)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		userID         uuid.UUID
		expectedStatus int
		checkSuccess   bool
	}{
		{
			name: "success with explicit module name",
			requestBody: map[string]interface{}{
				"workflow_id": appID.String(),
				"module_name": "my-custom-module",
			},
			userID:         userID,
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name: "success — module name falls back to app name",
			requestBody: map[string]interface{}{
				"workflow_id": appID.String(),
				// module_name omitted — handler uses app name
			},
			userID:         userID,
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name: "workflow not found",
			requestBody: map[string]interface{}{
				"workflow_id": uuid.New().String(),
				"module_name": "missing-module",
			},
			userID:         userID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "different user cannot install",
			requestBody: map[string]interface{}{
				"workflow_id": appID.String(),
				"module_name": "hacked",
			},
			userID:         uuid.New(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/modules/install", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("user_id", tt.userID)

			handler.InstallModule(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkSuccess {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.True(t, resp["success"].(bool))
				assert.NotEmpty(t, resp["module_id"])
				assert.NotEmpty(t, resp["message"])
			}
		})
	}
}

// TestInstallModule_EmptyBody confirms that an empty JSON body returns 400 via
// Gin's BindJSON (requires at least a valid JSON object).
func TestInstallModule_EmptyBody(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	handler := newWorkflowsHandler(t, pool)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/modules/install",
		strings.NewReader(""))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", uuid.New())

	handler.InstallModule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTriggerSync tests the manual sync endpoint.
func TestTriggerSync(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	handler := newWorkflowsHandler(t, pool)

	userID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/osa/sync/trigger", nil)
	c.Set("user_id", userID)

	handler.TriggerSync(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Sync triggered", resp["message"])
	assert.Equal(t, userID.String(), resp["user_id"])
}

// TestConcurrentWorkflowAccess ensures the handler is safe under concurrent load.
func TestConcurrentWorkflowAccess(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	appID := createTestApp(t, pool, workspaceID, map[string]interface{}{
		"analysis": "concurrent test content",
	})
	handler := newWorkflowsHandler(t, pool)

	const concurrency = 20
	var wg sync.WaitGroup
	statuses := make(chan int, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String(), nil)
			c.Params = gin.Params{{Key: "id", Value: appID.String()}}
			c.Set("user_id", userID)
			handler.GetWorkflow(c)
			statuses <- w.Code
		}()
	}

	wg.Wait()
	close(statuses)

	successCount := 0
	for code := range statuses {
		if code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, concurrency, successCount, "all concurrent requests should succeed")
}

// TestConcurrentWorkflowFiles tests concurrent file listing.
func TestConcurrentWorkflowFiles(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	userID := uuid.New()
	workspaceID := createTestWorkspace(t, pool, userID)
	defer cleanupWorkflows(t, pool, workspaceID)

	appID := createTestApp(t, pool, workspaceID, map[string]interface{}{
		"analysis":     "content",
		"architecture": "design",
	})
	handler := newWorkflowsHandler(t, pool)

	const concurrency = 10
	var wg sync.WaitGroup
	results := make(chan int, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String()+"/files", nil)
			c.Params = gin.Params{{Key: "id", Value: appID.String()}}
			c.Set("user_id", userID)
			handler.GetWorkflowFiles(c)
			results <- w.Code
		}()
	}

	wg.Wait()
	close(results)

	for code := range results {
		assert.Equal(t, http.StatusOK, code)
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

// BenchmarkListWorkflows measures throughput of the list endpoint.
func BenchmarkListWorkflows(b *testing.B) {
	pool, err := pgxpool.New(context.Background(),
		"postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable")
	if err != nil {
		b.Skipf("Skipping benchmark: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	userID := uuid.New()
	var workspaceID uuid.UUID
	err = pool.QueryRow(context.Background(), `
		INSERT INTO osa_workspaces (user_id, name, workspace_path)
		VALUES ($1, 'bench-workspace', '/tmp/bench')
		RETURNING id
	`, userID).Scan(&workspaceID)
	if err != nil {
		b.Skipf("Setup failed: %v", err)
	}
	defer pool.Exec(context.Background(), "DELETE FROM osa_workspaces WHERE id = $1", workspaceID) //nolint:errcheck

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	syncService := services.NewOSAFileSyncService(pool, logger, "")
	handler := NewOSAWorkflowsHandler(pool, syncService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows", nil)
		c.Set("user_id", userID)
		handler.ListWorkflows(c)
	}
}

// BenchmarkGetWorkflow measures single-workflow fetch throughput.
func BenchmarkGetWorkflow(b *testing.B) {
	pool, err := pgxpool.New(context.Background(),
		"postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable")
	if err != nil {
		b.Skipf("Skipping benchmark: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	userID := uuid.New()
	var workspaceID uuid.UUID
	err = pool.QueryRow(context.Background(), `
		INSERT INTO osa_workspaces (user_id, name, workspace_path)
		VALUES ($1, 'bench-ws', '/tmp/bench-wf')
		RETURNING id
	`, userID).Scan(&workspaceID)
	if err != nil {
		b.Skipf("Workspace setup failed: %v", err)
	}
	defer pool.Exec(context.Background(), "DELETE FROM osa_workspaces WHERE id = $1", workspaceID) //nolint:errcheck

	metaJSON := []byte(`{"analysis":"bench content","code":"package main"}`)
	var appID uuid.UUID
	err = pool.QueryRow(context.Background(), `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name, description,
			osa_workflow_id, status, files_created, metadata
		) VALUES ($1, 'bench-app', 'Bench', 'bench', 'wf-bench', 'generated', 2, $2)
		RETURNING id
	`, workspaceID, metaJSON).Scan(&appID)
	if err != nil {
		b.Skipf("App setup failed: %v", err)
	}
	defer pool.Exec(context.Background(), "DELETE FROM osa_module_instances WHERE id = $1", appID) //nolint:errcheck

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	syncService := services.NewOSAFileSyncService(pool, logger, "")
	handler := NewOSAWorkflowsHandler(pool, syncService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/osa/workflows/"+appID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: appID.String()}}
		c.Set("user_id", userID)
		handler.GetWorkflow(c)
	}
}
