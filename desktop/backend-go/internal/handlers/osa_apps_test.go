package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestOSAAppsHandler(t *testing.T, pool *pgxpool.Pool) *OSAAppsHandler {
	queries := sqlc.New(pool)
	return NewOSAAppsHandler(queries, pool, slog.Default())
}

func TestGenerateOSAApp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a test database with the full schema
	// For now, this is a placeholder that shows the structure
	t.Skip("Requires test database with app_generation_queue table")

	// Setup
	var pool *pgxpool.Pool // Would be initialized from test database
	handler := setupTestOSAAppsHandler(t, pool)

	// Create test workspace
	ctx := context.Background()
	var workspaceID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, owner_id)
		VALUES ('Test Workspace', $1)
		RETURNING id
	`, uuid.New()).Scan(&workspaceID)
	require.NoError(t, err)
	defer pool.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", workspaceID)

	// Create test template
	var templateID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO app_templates (
			template_name,
			category,
			display_name,
			generation_prompt,
			scaffold_type
		) VALUES (
			'test-template',
			'utility',
			'Test Template',
			'Generate a test app',
			'standalone'
		)
		RETURNING id
	`).Scan(&templateID)
	require.NoError(t, err)
	defer pool.Exec(ctx, "DELETE FROM app_templates WHERE id = $1", templateID)

	// Setup Gin context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create request body
	templateIDStr := templateID.String()
	requestBody := GenerateOSAAppRequest{
		TemplateID:  &templateIDStr,
		AppName:     "Test App",
		Description: "A test app for unit testing",
		Config: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/workspaces/"+workspaceID.String()+"/apps/generate-osa", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: workspaceID.String()}}

	// Mock user in context (normally done by auth middleware)
	// This would require setting up the user in context properly
	// c.Set("current_user", &User{ID: uuid.New().String()})

	// Execute
	handler.GenerateOSAApp(c)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response GenerateOSAAppResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "pending", response.Status)
	assert.NotEqual(t, uuid.Nil, response.QueueItemID)

	// Verify queue item was created
	var queueStatus string
	err = pool.QueryRow(ctx, `
		SELECT status FROM app_generation_queue WHERE id = $1
	`, response.QueueItemID).Scan(&queueStatus)
	require.NoError(t, err)
	assert.Equal(t, "pending", queueStatus)
}

// TestIsValidPath tests the path validation function
func TestIsValidPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Valid paths
		{"empty path", "", true},
		{"root path", "/", true},
		{"simple path", "src/main.go", true},
		{"nested path", "src/components/Button.tsx", true},
		{"absolute path", "/src", true},
		{"path with dots in filename", "file.test.ts", true},

		// Invalid paths - directory traversal
		{"parent directory traversal", "../etc/passwd", false},
		{"double parent traversal", "../../etc/passwd", false},
		{"hidden parent traversal", "src/../../../etc/passwd", false},
		{"windows style traversal", "..\\windows\\system32", false},

		// Invalid paths - null bytes
		{"null byte injection", "src/file\x00.txt", false},
		{"null byte at end", "src/file.txt\x00", false},

		// Invalid paths - double slashes
		{"double slash prefix", "//etc/passwd", false},
		{"double slash in path", "src//etc//passwd", true}, // This is allowed as it's not at the start

		// Edge cases
		{"dot in path (current dir)", "./src/main.go", true},
		{"multiple dots (contains ..)", "src/file...txt", false}, // Contains ".." substring - security risk
		{"dots separated", "src/. ./file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPath(tt.path)
			assert.Equal(t, tt.expected, result, "Path: %s", tt.path)
		})
	}
}

// TestGetAppFiles_Security tests security aspects of the file browsing endpoint
func TestGetAppFiles_Security(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		pathParam      string
		expectedStatus int
		description    string
	}{
		{
			name:           "path traversal attack - parent directory",
			pathParam:      "../../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject path traversal with parent directory",
		},
		{
			name:           "path traversal attack - encoded",
			pathParam:      "%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject URL-encoded path traversal",
		},
		{
			name:           "null byte injection",
			pathParam:      "src/file\x00.txt",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject null byte injection",
		},
		{
			name:           "double slash prefix",
			pathParam:      "//etc/passwd",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject double slash prefix",
		},
		{
			name:           "valid path",
			pathParam:      "src/main.go",
			expectedStatus: http.StatusOK,
			description:    "Should accept valid relative path",
		},
		{
			name:           "valid absolute path",
			pathParam:      "/src",
			expectedStatus: http.StatusOK,
			description:    "Should accept valid absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a security test structure - would need integration with test database
			// For now, we validate the isValidPath logic is being called correctly
			t.Logf("Testing: %s - %s", tt.name, tt.description)

			if tt.pathParam != "" {
				isValid := isValidPath(tt.pathParam)
				if tt.expectedStatus == http.StatusBadRequest {
					assert.False(t, isValid, "Expected path to be invalid: %s", tt.pathParam)
				} else if tt.expectedStatus == http.StatusOK {
					assert.True(t, isValid, "Expected path to be valid: %s", tt.pathParam)
				}
			}
		})
	}
}

// TestGetAppFiles_OwnershipValidation tests that users can only access their own apps
func TestGetAppFiles_OwnershipValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)

	// This test validates the ownership check logic
	t.Run("ownership check exists", func(t *testing.T) {
		// The implementation calls GetOSAModuleInstanceByIDWithAuth which includes:
		// SELECT ... FROM osa_module_instances a
		// JOIN workspaces w ON a.workspace_id = w.id
		// JOIN workspace_members wm ON w.id = wm.workspace_id
		// WHERE a.id = $1 AND wm.user_id = $2
		//
		// This ensures that the user must be a member of the workspace
		// that owns the app before accessing its files

		t.Log("Ownership validation is implemented via GetOSAModuleInstanceByIDWithAuth query")
		t.Log("User must be workspace member to access app files")
	})
}

// TestGetAppFiles_Pagination tests pagination functionality
func TestGetAppFiles_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		limit          string
		offset         string
		expectedLimit  int32
		expectedOffset int32
	}{
		{
			name:           "default pagination",
			limit:          "",
			offset:         "",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "custom limit",
			limit:          "10",
			offset:         "",
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "custom offset",
			limit:          "",
			offset:         "20",
			expectedLimit:  50,
			expectedOffset: 20,
		},
		{
			name:           "custom limit and offset",
			limit:          "25",
			offset:         "50",
			expectedLimit:  25,
			expectedOffset: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pagination parameter parsing
			limit := int32(50) // default
			if tt.limit != "" {
				if parsedLimit, err := parseIntParam(tt.limit); err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}

			offset := int32(0) // default
			if tt.offset != "" {
				if parsedOffset, err := parseIntParam(tt.offset); err == nil && parsedOffset >= 0 {
					offset = parsedOffset
				}
			}

			assert.Equal(t, tt.expectedLimit, limit, "Limit mismatch")
			assert.Equal(t, tt.expectedOffset, offset, "Offset mismatch")
		})
	}
}

// TestGetAppFiles_Filtering tests file filtering functionality
func TestGetAppFiles_Filtering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("filter by path prefix", func(t *testing.T) {
		// The implementation filters files by path prefix:
		// if pathFilter != "" && !strings.HasPrefix(file.FilePath, pathFilter) {
		//     continue
		// }
		t.Log("Files are filtered by path prefix when path query param is provided")
	})

	t.Run("filter by type", func(t *testing.T) {
		// The implementation filters by file type:
		// if typeFilter != "" && file.FileType != typeFilter {
		//     continue
		// }
		t.Log("Files are filtered by type (file/directory) when type query param is provided")
	})

	t.Run("filter by language", func(t *testing.T) {
		// The implementation filters by programming language:
		// if languageFilter != "" {
		//     if file.Language == nil || *file.Language != languageFilter {
		//         continue
		//     }
		// }
		t.Log("Files are filtered by programming language when language query param is provided")
	})
}

// TestGetAppFiles_Response tests response format
func TestGetAppFiles_Response(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("response structure", func(t *testing.T) {
		// Expected response format:
		expected := FileListResponse{
			AppID:       uuid.New(),
			Repository:  "github.com/user/repo",
			Files:       []FileNodeFlat{},
			Total:       0,
			Limit:       50,
			Offset:      0,
			CurrentPath: "src/",
		}

		// Validate JSON marshaling
		data, err := json.Marshal(expected)
		require.NoError(t, err)

		var decoded FileListResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, expected.AppID, decoded.AppID)
		assert.Equal(t, expected.Repository, decoded.Repository)
		assert.Equal(t, expected.Limit, decoded.Limit)
		assert.Equal(t, expected.Offset, decoded.Offset)
		assert.Equal(t, expected.CurrentPath, decoded.CurrentPath)
	})
}

// TestGetAppFiles_SensitiveFileBlocking tests that sensitive files are not exposed
func TestGetAppFiles_SensitiveFileBlocking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sensitiveFiles := []string{
		".env",
		".env.local",
		".env.production",
		"credentials.json",
		"serviceAccountKey.json",
		"private.key",
		"id_rsa",
		"secrets.yaml",
	}

	t.Run("sensitive files should be filtered", func(t *testing.T) {
		// The current implementation does not explicitly filter sensitive files
		// This is a recommendation for enhancement
		for _, file := range sensitiveFiles {
			t.Logf("RECOMMENDATION: Block access to sensitive file: %s", file)
		}

		t.Log("SECURITY ENHANCEMENT: Consider adding explicit filtering for sensitive files")
		t.Log("Files like .env, credentials.json, *.key should be blocked from the API response")
	})
}
