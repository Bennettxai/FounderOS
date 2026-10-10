package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// parseIntParam
// ============================================================================

func TestParseIntParam_ValidPositive(t *testing.T) {
	result, err := parseIntParam("42")
	require.NoError(t, err)
	assert.Equal(t, int32(42), result)
}

func TestParseIntParam_ValidZero(t *testing.T) {
	result, err := parseIntParam("0")
	require.NoError(t, err)
	assert.Equal(t, int32(0), result)
}

func TestParseIntParam_ValidLargeNumber(t *testing.T) {
	result, err := parseIntParam("1000000")
	require.NoError(t, err)
	assert.Equal(t, int32(1000000), result)
}

func TestParseIntParam_NegativeNumber(t *testing.T) {
	result, err := parseIntParam("-5")
	require.NoError(t, err)
	assert.Equal(t, int32(-5), result)
}

func TestParseIntParam_InvalidString(t *testing.T) {
	_, err := parseIntParam("not-a-number")
	assert.Error(t, err)
}

func TestParseIntParam_EmptyString(t *testing.T) {
	_, err := parseIntParam("")
	assert.Error(t, err)
}

func TestParseIntParam_Float(t *testing.T) {
	// Sscanf stops at '.' so "1.5" parses as 1
	result, err := parseIntParam("1.5")
	require.NoError(t, err)
	assert.Equal(t, int32(1), result)
}

func TestParseIntParam_LeadingWhitespace(t *testing.T) {
	// Sscanf handles leading whitespace
	result, err := parseIntParam(" 10")
	require.NoError(t, err)
	assert.Equal(t, int32(10), result)
}

// ============================================================================
// mapStatusForFrontend
// ============================================================================

func TestMapStatusForFrontend_StartingVariants(t *testing.T) {
	assert.Equal(t, "starting", mapStatusForFrontend("started"))
	assert.Equal(t, "starting", mapStatusForFrontend("starting"))
}

func TestMapStatusForFrontend_InProgressVariants(t *testing.T) {
	cases := []string{"planning", "generating", "running", "processing", "in_progress"}
	for _, status := range cases {
		t.Run(status, func(t *testing.T) {
			assert.Equal(t, "in_progress", mapStatusForFrontend(status))
		})
	}
}

func TestMapStatusForFrontend_CompletedVariants(t *testing.T) {
	assert.Equal(t, "completed", mapStatusForFrontend("completed"))
	assert.Equal(t, "completed", mapStatusForFrontend("done"))
	assert.Equal(t, "completed", mapStatusForFrontend("success"))
}

func TestMapStatusForFrontend_FailedVariants(t *testing.T) {
	assert.Equal(t, "failed", mapStatusForFrontend("failed"))
	assert.Equal(t, "failed", mapStatusForFrontend("error"))
}

func TestMapStatusForFrontend_PendingVariants(t *testing.T) {
	assert.Equal(t, "pending", mapStatusForFrontend("pending"))
	assert.Equal(t, "pending", mapStatusForFrontend("queued"))
	assert.Equal(t, "pending", mapStatusForFrontend("waiting"))
}

func TestMapStatusForFrontend_UnknownDefaultsToInProgress(t *testing.T) {
	assert.Equal(t, "in_progress", mapStatusForFrontend(""))
	assert.Equal(t, "in_progress", mapStatusForFrontend("some_unknown_status"))
	assert.Equal(t, "in_progress", mapStatusForFrontend("COMPLETED")) // case-sensitive
}

// ============================================================================
// isValidPath
// ============================================================================

func TestIsValidPath_EmptyPath(t *testing.T) {
	assert.True(t, isValidPath(""))
}

func TestIsValidPath_RootPath(t *testing.T) {
	assert.True(t, isValidPath("/"))
}

func TestIsValidPath_SimpleRelativePath(t *testing.T) {
	assert.True(t, isValidPath("src/main.go"))
}

func TestIsValidPath_NestedPath(t *testing.T) {
	assert.True(t, isValidPath("src/components/Button.tsx"))
}

func TestIsValidPath_AbsolutePath(t *testing.T) {
	assert.True(t, isValidPath("/src"))
}

func TestIsValidPath_PathWithDotsInFilename(t *testing.T) {
	assert.True(t, isValidPath("file.test.ts"))
}

func TestIsValidPath_DotSlashPrefix(t *testing.T) {
	// "./src/main.go" does not contain ".." substring
	assert.True(t, isValidPath("./src/main.go"))
}

func TestIsValidPath_ParentDirectoryTraversal(t *testing.T) {
	assert.False(t, isValidPath("../etc/passwd"))
}

func TestIsValidPath_DoubleParentTraversal(t *testing.T) {
	assert.False(t, isValidPath("../../etc/passwd"))
}

func TestIsValidPath_HiddenTraversal(t *testing.T) {
	assert.False(t, isValidPath("src/../../../etc/passwd"))
}

func TestIsValidPath_WindowsStyleTraversal(t *testing.T) {
	assert.False(t, isValidPath("..\\windows\\system32"))
}

func TestIsValidPath_BackslashInPath(t *testing.T) {
	assert.False(t, isValidPath("src\\file.go"))
}

func TestIsValidPath_NullByteInjection(t *testing.T) {
	assert.False(t, isValidPath("src/file\x00.txt"))
}

func TestIsValidPath_NullByteAtEnd(t *testing.T) {
	assert.False(t, isValidPath("src/file.txt\x00"))
}

func TestIsValidPath_DoubleSlashPrefix(t *testing.T) {
	assert.False(t, isValidPath("//etc/passwd"))
}

func TestIsValidPath_EncodedTraversal(t *testing.T) {
	// URL-encoded ".." (%2e%2e)
	assert.False(t, isValidPath("%2e%2e%2fetc%2fpasswd"))
}

func TestIsValidPath_TripleDotInFilename(t *testing.T) {
	// "src/file...txt" contains ".." substring — rejected
	assert.False(t, isValidPath("src/file...txt"))
}

func TestIsValidPath_InternalDoubleSlash(t *testing.T) {
	// "src//etc//passwd" — double slash not at start, allowed
	assert.True(t, isValidPath("src//etc//passwd"))
}

func TestIsValidPath_SpaceSeparatedDots(t *testing.T) {
	// "src/. ./file.txt" — the ". ." substring doesn't contain ".."
	assert.True(t, isValidPath("src/. ./file.txt"))
}

// ============================================================================
// formatTimestamp
// ============================================================================

func TestFormatTimestamp_ValidTimestamp(t *testing.T) {
	ts := pgtype.Timestamptz{
		Time:  time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC),
		Valid: true,
	}
	result := formatTimestamp(ts)
	require.NotNil(t, result)
	assert.Equal(t, "2025-06-15T12:30:00Z", *result)
}

func TestFormatTimestamp_InvalidTimestamp(t *testing.T) {
	ts := pgtype.Timestamptz{Valid: false}
	result := formatTimestamp(ts)
	assert.Nil(t, result)
}

func TestFormatTimestamp_ZeroTime(t *testing.T) {
	ts := pgtype.Timestamptz{
		Time:  time.Time{},
		Valid: true,
	}
	result := formatTimestamp(ts)
	require.NotNil(t, result)
	// Just verify it returns a string — zero time is still valid
	assert.NotEmpty(t, *result)
}

// ============================================================================
// getInstallationStatus
// ============================================================================

func TestGetInstallationStatus_NilReturnsDefault(t *testing.T) {
	result := getInstallationStatus(nil)
	assert.Equal(t, "pending", result)
}

func TestGetInstallationStatus_NonNilReturnsValue(t *testing.T) {
	status := "installed"
	result := getInstallationStatus(&status)
	assert.Equal(t, "installed", result)
}

func TestGetInstallationStatus_EmptyStringReturnsEmpty(t *testing.T) {
	status := ""
	result := getInstallationStatus(&status)
	assert.Equal(t, "", result)
}

func TestGetInstallationStatus_ArbitraryValue(t *testing.T) {
	status := "uninstalling"
	result := getInstallationStatus(&status)
	assert.Equal(t, "uninstalling", result)
}

// ============================================================================
// uuidPtrFromPgtype
// ============================================================================

func TestUUIDPtrFromPgtype_ValidUUID(t *testing.T) {
	id := uuid.New()
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	result := uuidPtrFromPgtype(pgUUID)
	require.NotNil(t, result)
	assert.Equal(t, id, *result)
}

func TestUUIDPtrFromPgtype_InvalidUUID(t *testing.T) {
	pgUUID := pgtype.UUID{Valid: false}
	result := uuidPtrFromPgtype(pgUUID)
	assert.Nil(t, result)
}

func TestUUIDPtrFromPgtype_ZeroUUID(t *testing.T) {
	pgUUID := pgtype.UUID{Bytes: [16]byte{}, Valid: true}
	result := uuidPtrFromPgtype(pgUUID)
	require.NotNil(t, result)
	assert.Equal(t, uuid.Nil, *result)
}

// ============================================================================
// timestampPtrToString
// ============================================================================

func TestTimestampPtrToString_ValidTimestamp(t *testing.T) {
	ts := pgtype.Timestamptz{
		Time:  time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Valid: true,
	}
	result := timestampPtrToString(ts)
	require.NotNil(t, result)
	assert.Equal(t, "2025-01-15T09:00:00Z", *result)
}

func TestTimestampPtrToString_InvalidTimestamp(t *testing.T) {
	ts := pgtype.Timestamptz{Valid: false}
	result := timestampPtrToString(ts)
	assert.Nil(t, result)
}

// ============================================================================
// convertToAppDetailFromGeneratedApp
// ============================================================================

func TestConvertToAppDetailFromGeneratedApp_BasicFields(t *testing.T) {
	id := uuid.New()
	workspaceID := uuid.New()
	now := pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Second), Valid: true}

	app := sqlc.OsaModuleInstance{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		WorkspaceID: pgtype.UUID{Bytes: workspaceID, Valid: true},
		Name:        "test-app",
		DisplayName: "Test App",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result := convertToAppDetailFromGeneratedApp(app)

	assert.Equal(t, id, result.ID)
	assert.Equal(t, workspaceID, result.WorkspaceID)
	assert.Equal(t, "test-app", result.Name)
	assert.Equal(t, "Test App", result.DisplayName)
	assert.Nil(t, result.ModuleID)    // ModuleID not set
	assert.Nil(t, result.Description) // Description not set
	assert.Nil(t, result.GeneratedAt) // GeneratedAt not set
	assert.Nil(t, result.DeployedAt)  // DeployedAt not set
	assert.Nil(t, result.LastBuildAt) // LastBuildAt not set
}

func TestConvertToAppDetailFromGeneratedApp_WithOptionalFields(t *testing.T) {
	id := uuid.New()
	workspaceID := uuid.New()
	moduleID := uuid.New()
	now := pgtype.Timestamptz{Time: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	desc := "My description"
	status := "deployed"
	repo := "github.com/user/repo"
	deployURL := "https://app.example.com"

	app := sqlc.OsaModuleInstance{
		ID:             pgtype.UUID{Bytes: id, Valid: true},
		WorkspaceID:    pgtype.UUID{Bytes: workspaceID, Valid: true},
		ModuleID:       pgtype.UUID{Bytes: moduleID, Valid: true},
		Name:           "my-app",
		DisplayName:    "My App",
		Description:    &desc,
		Status:         &status,
		CodeRepository: &repo,
		DeploymentUrl:  &deployURL,
		CreatedAt:      now,
		UpdatedAt:      now,
		GeneratedAt:    now,
		DeployedAt:     now,
	}

	result := convertToAppDetailFromGeneratedApp(app)

	assert.Equal(t, id, result.ID)
	assert.Equal(t, workspaceID, result.WorkspaceID)
	require.NotNil(t, result.ModuleID)
	assert.Equal(t, moduleID, *result.ModuleID)
	require.NotNil(t, result.Description)
	assert.Equal(t, "My description", *result.Description)
	require.NotNil(t, result.Status)
	assert.Equal(t, "deployed", *result.Status)
	require.NotNil(t, result.CodeRepository)
	assert.Equal(t, "github.com/user/repo", *result.CodeRepository)
	require.NotNil(t, result.DeploymentURL)
	assert.Equal(t, "https://app.example.com", *result.DeploymentURL)
	require.NotNil(t, result.GeneratedAt)
	require.NotNil(t, result.DeployedAt)
	assert.Equal(t, "2025-03-01T00:00:00Z", *result.GeneratedAt)
}

func TestConvertToAppDetailFromGeneratedApp_TimestampFormat(t *testing.T) {
	specificTime := time.Date(2025, 12, 25, 18, 30, 45, 0, time.UTC)
	now := pgtype.Timestamptz{Time: specificTime, Valid: true}

	app := sqlc.OsaModuleInstance{
		ID:          pgtype.UUID{Bytes: uuid.New(), Valid: true},
		WorkspaceID: pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Name:        "time-test",
		DisplayName: "Time Test",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result := convertToAppDetailFromGeneratedApp(app)

	assert.Equal(t, "2025-12-25T18:30:45Z", result.CreatedAt)
	assert.Equal(t, "2025-12-25T18:30:45Z", result.UpdatedAt)
}

// ============================================================================
// AppDetail JSON serialization
// ============================================================================

func TestAppDetailJSONSerialization_OmitsEmptyOptionals(t *testing.T) {
	detail := AppDetail{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Name:        "test",
		DisplayName: "Test",
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(detail)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Optional fields should be omitted when nil
	_, hasModuleID := raw["module_id"]
	assert.False(t, hasModuleID, "module_id should be omitted when nil")

	_, hasDescription := raw["description"]
	assert.False(t, hasDescription, "description should be omitted when nil")

	_, hasStatus := raw["status"]
	assert.False(t, hasStatus, "status should be omitted when nil")

	_, hasGeneratedAt := raw["generated_at"]
	assert.False(t, hasGeneratedAt, "generated_at should be omitted when nil")
}

func TestAppDetailJSONSerialization_IncludesSetOptionals(t *testing.T) {
	desc := "My app description"
	status := "deployed"
	detail := AppDetail{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Name:        "test",
		DisplayName: "Test",
		Description: &desc,
		Status:      &status,
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(detail)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "My app description", raw["description"])
	assert.Equal(t, "deployed", raw["status"])
}

func TestAppListResponseJSONSerialization(t *testing.T) {
	response := AppListResponse{
		Apps:       []AppDetail{},
		TotalCount: 0,
		Limit:      20,
		Offset:     0,
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded AppListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, int64(0), decoded.TotalCount)
	assert.Equal(t, int32(20), decoded.Limit)
	assert.Equal(t, int32(0), decoded.Offset)
	assert.Empty(t, decoded.Apps)
}

// ============================================================================
// FileListResponse JSON serialization
// ============================================================================

func TestFileListResponseJSONSerialization_RoundTrip(t *testing.T) {
	appID := uuid.New()
	lang := "go"
	modTime := "2025-06-01T10:00:00Z"

	original := FileListResponse{
		AppID:      appID,
		Repository: "github.com/user/repo",
		Files: []FileNodeFlat{
			{
				ID:       uuid.New(),
				Path:     "src/main.go",
				Name:     "main.go",
				Type:     "file",
				Language: &lang,
				Size:     1024,
				Modified: &modTime,
				Hash:     "abc123",
				Status:   "installed",
			},
		},
		Total:       1,
		Limit:       50,
		Offset:      0,
		CurrentPath: "src/",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded FileListResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.AppID, decoded.AppID)
	assert.Equal(t, original.Repository, decoded.Repository)
	assert.Equal(t, original.Total, decoded.Total)
	assert.Equal(t, original.Limit, decoded.Limit)
	assert.Equal(t, original.Offset, decoded.Offset)
	assert.Equal(t, original.CurrentPath, decoded.CurrentPath)
	require.Len(t, decoded.Files, 1)
	assert.Equal(t, "src/main.go", decoded.Files[0].Path)
	assert.Equal(t, "main.go", decoded.Files[0].Name)
	assert.Equal(t, int64(1024), decoded.Files[0].Size)
}

// ============================================================================
// UpdateAppMetadataRequest JSON deserialization
// ============================================================================

func TestUpdateAppMetadataRequest_AllFields(t *testing.T) {
	raw := `{
		"display_name": "New Name",
		"description": "New description",
		"metadata": {"key": "value", "count": 42}
	}`

	var req UpdateAppMetadataRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "New Name", *req.DisplayName)

	require.NotNil(t, req.Description)
	assert.Equal(t, "New description", *req.Description)

	require.NotNil(t, req.Metadata)
	assert.Equal(t, "value", (*req.Metadata)["key"])
}

func TestUpdateAppMetadataRequest_PartialFields(t *testing.T) {
	raw := `{"display_name": "Partial Update"}`

	var req UpdateAppMetadataRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	require.NotNil(t, req.DisplayName)
	assert.Equal(t, "Partial Update", *req.DisplayName)
	assert.Nil(t, req.Description)
	assert.Nil(t, req.Metadata)
}

func TestUpdateAppMetadataRequest_EmptyBody(t *testing.T) {
	raw := `{}`

	var req UpdateAppMetadataRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Nil(t, req.DisplayName)
	assert.Nil(t, req.Description)
	assert.Nil(t, req.Metadata)
}

// ============================================================================
// SaveFileRequest JSON deserialization
// ============================================================================

func TestSaveFileRequest_ValidRequest(t *testing.T) {
	raw := `{"file_path": "src/main.go", "content": "package main\n\nfunc main() {}"}`

	var req SaveFileRequest
	err := json.Unmarshal([]byte(raw), &req)
	require.NoError(t, err)

	assert.Equal(t, "src/main.go", req.FilePath)
	assert.Equal(t, "package main\n\nfunc main() {}", req.Content)
}
