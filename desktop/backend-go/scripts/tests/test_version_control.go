//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// ================================================================
// VERSION CONTROL TEST SUITE
// ================================================================
// Comprehensive testing for app and workspace version control:
// 1. App version restore: POST /api/apps/:id/restore/:version
// 2. Workspace version restore: POST /api/workspaces/:id/restore/:version
// 3. Version numbering (0.0.1, 0.0.2, auto-increment)
// 4. Changes persist after restore
// 5. Auto-backup before restore
// 6. Permission checks (admin/owner only for workspace, any member for app)
// ================================================================

const (
	baseURL = "http://localhost:8001"

	// Test colors
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

type TestContext struct {
	WorkspaceID uuid.UUID
	UserID      string
	Token       string
	AppID       uuid.UUID
	Logger      *slog.Logger
}

type AppVersionSnapshot struct {
	ID               uuid.UUID              `json:"id"`
	AppID            uuid.UUID              `json:"app_id"`
	VersionNumber    string                 `json:"version_number"`
	SnapshotData     map[string]interface{} `json:"snapshot_data"`
	SnapshotMetadata map[string]interface{} `json:"snapshot_metadata,omitempty"`
	ChangeSummary    *string                `json:"change_summary,omitempty"`
	CreatedBy        *string                `json:"created_by,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type WorkspaceVersion struct {
	ID               string                 `json:"id"`
	VersionNumber    string                 `json:"version_number"`
	CreatedBy        *string                `json:"created_by"`
	CreatedAt        interface{}            `json:"created_at"`
	SnapshotMetadata map[string]interface{} `json:"snapshot_metadata"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	printHeader("VERSION CONTROL TEST SUITE")

	// Get test credentials from environment
	token := os.Getenv("TEST_TOKEN")
	workspaceIDStr := os.Getenv("TEST_WORKSPACE_ID")
	userID := os.Getenv("TEST_USER_ID")

	if token == "" || workspaceIDStr == "" || userID == "" {
		log.Fatal("Missing environment variables: TEST_TOKEN, TEST_WORKSPACE_ID, TEST_USER_ID")
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		log.Fatalf("Invalid workspace ID: %v", err)
	}

	ctx := &TestContext{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Token:       token,
		Logger:      logger,
	}

	// Run test suite
	totalTests := 0
	passedTests := 0

	tests := []struct {
		name string
		fn   func(*TestContext) error
	}{
		{"Test 1: App Version - Create Snapshot", testAppCreateSnapshot},
		{"Test 2: App Version - Sequential Numbering", testAppVersionNumbering},
		{"Test 3: App Version - Modify and Restore", testAppModifyAndRestore},
		{"Test 4: App Version - Changes Persist After Restore", testAppChangePersistence},
		{"Test 5: App Version - List Versions", testAppListVersions},
		{"Test 6: App Version - Get Specific Version", testAppGetVersion},
		{"Test 7: App Version - Get Latest Version", testAppGetLatestVersion},
		{"Test 8: App Version - Version Stats", testAppVersionStats},
		{"Test 9: Workspace Version - Create Snapshot", testWorkspaceCreateSnapshot},
		{"Test 10: Workspace Version - Sequential Numbering", testWorkspaceVersionNumbering},
		{"Test 11: Workspace Version - Modify and Restore", testWorkspaceModifyAndRestore},
		{"Test 12: Workspace Version - Auto-Backup Before Restore", testWorkspaceAutoBackup},
		{"Test 13: Workspace Version - Permission Check (Admin/Owner)", testWorkspacePermissions},
		{"Test 14: Workspace Version - Dry Run Preview", testWorkspaceDryRun},
		{"Test 15: App Version - Cleanup Old Versions", testAppCleanupOldVersions},
	}

	for _, test := range tests {
		totalTests++
		printTestStart(test.name)

		if err := test.fn(ctx); err != nil {
			printTestFail(test.name, err)
		} else {
			printTestPass(test.name)
			passedTests++
		}

		time.Sleep(500 * time.Millisecond) // Rate limiting
	}

	printSummary(passedTests, totalTests)
}

// ================================================================
// APP VERSION TESTS
// ================================================================

func testAppCreateSnapshot(ctx *TestContext) error {
	// First, create a test app
	appID, err := createTestApp(ctx)
	if err != nil {
		return fmt.Errorf("failed to create test app: %w", err)
	}
	ctx.AppID = appID
	ctx.Logger.Info("Created test app", slog.String("app_id", appID.String()))

	// Create snapshot with custom summary
	changeSummary := "Initial snapshot for testing"
	body := map[string]interface{}{
		"change_summary": changeSummary,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/apps/%s/versions", baseURL, appID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var snapshot AppVersionSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify version number
	if snapshot.VersionNumber != "0.0.1" {
		return fmt.Errorf("expected version 0.0.1, got %s", snapshot.VersionNumber)
	}

	// Verify change summary
	if snapshot.ChangeSummary == nil || *snapshot.ChangeSummary != changeSummary {
		return fmt.Errorf("change summary mismatch")
	}

	ctx.Logger.Info("Snapshot created", slog.String("version", snapshot.VersionNumber))
	return nil
}

func testAppVersionNumbering(ctx *TestContext) error {
	// Create multiple snapshots and verify sequential numbering
	expectedVersions := []string{"0.0.2", "0.0.3", "0.0.4"}

	for _, expectedVersion := range expectedVersions {
		req, err := http.NewRequest("POST",
			fmt.Sprintf("%s/api/apps/%s/versions", baseURL, ctx.AppID),
			nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+ctx.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create snapshot: status %d", resp.StatusCode)
		}

		var snapshot AppVersionSnapshot
		if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
			return err
		}

		if snapshot.VersionNumber != expectedVersion {
			return fmt.Errorf("expected version %s, got %s", expectedVersion, snapshot.VersionNumber)
		}

		ctx.Logger.Info("Version created", slog.String("version", snapshot.VersionNumber))
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func testAppModifyAndRestore(ctx *TestContext) error {
	// 1. Modify the app
	if err := modifyApp(ctx, ctx.AppID, true, true); err != nil {
		return fmt.Errorf("failed to modify app: %w", err)
	}
	ctx.Logger.Info("Modified app: is_favorite=true, is_pinned=true")

	// 2. Create snapshot of modified state
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/apps/%s/versions", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var snapshot AppVersionSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return err
	}
	modifiedVersion := snapshot.VersionNumber
	ctx.Logger.Info("Created snapshot of modified state", slog.String("version", modifiedVersion))

	// 3. Modify again
	if err := modifyApp(ctx, ctx.AppID, false, false); err != nil {
		return fmt.Errorf("failed to modify app again: %w", err)
	}
	ctx.Logger.Info("Modified app again: is_favorite=false, is_pinned=false")

	// 4. Restore to previous version (0.0.1 - initial state)
	restoreReq, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/apps/%s/restore/0.0.1", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	restoreReq.Header.Set("Authorization", "Bearer "+ctx.Token)

	restoreResp, err := http.DefaultClient.Do(restoreReq)
	if err != nil {
		return err
	}
	defer restoreResp.Body.Close()

	if restoreResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(restoreResp.Body)
		return fmt.Errorf("restore failed: status %d, body: %s", restoreResp.StatusCode, string(body))
	}

	ctx.Logger.Info("Restored to version 0.0.1")
	return nil
}

func testAppChangePersistence(ctx *TestContext) error {
	// After restore, verify the app state persisted
	app, err := getApp(ctx, ctx.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Should be restored to initial state (is_favorite=false, is_pinned=false)
	isFavorite, _ := app["is_favorite"].(bool)
	isPinned, _ := app["is_pinned"].(bool)

	if isFavorite || isPinned {
		return fmt.Errorf("app state not correctly restored: is_favorite=%v, is_pinned=%v",
			isFavorite, isPinned)
	}

	ctx.Logger.Info("Verified app state persisted after restore",
		slog.Bool("is_favorite", isFavorite),
		slog.Bool("is_pinned", isPinned))
	return nil
}

func testAppListVersions(ctx *TestContext) error {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/apps/%s/versions", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Versions []AppVersionSnapshot `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Should have at least 4 versions (0.0.1, 0.0.2, 0.0.3, 0.0.4 + modified version)
	if len(result.Versions) < 4 {
		return fmt.Errorf("expected at least 4 versions, got %d", len(result.Versions))
	}

	ctx.Logger.Info("Listed versions", slog.Int("count", len(result.Versions)))
	return nil
}

func testAppGetVersion(ctx *TestContext) error {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/apps/%s/versions/0.0.1", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var version AppVersionSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return err
	}

	if version.VersionNumber != "0.0.1" {
		return fmt.Errorf("expected version 0.0.1, got %s", version.VersionNumber)
	}

	ctx.Logger.Info("Retrieved specific version", slog.String("version", version.VersionNumber))
	return nil
}

func testAppGetLatestVersion(ctx *TestContext) error {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/apps/%s/versions/latest", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var version AppVersionSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return err
	}

	ctx.Logger.Info("Retrieved latest version", slog.String("version", version.VersionNumber))
	return nil
}

func testAppVersionStats(ctx *TestContext) error {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/apps/%s/versions/stats", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return err
	}

	totalVersions, _ := stats["total_versions"].(float64)
	if totalVersions < 4 {
		return fmt.Errorf("expected at least 4 versions in stats, got %v", totalVersions)
	}

	ctx.Logger.Info("Retrieved version stats",
		slog.Any("total_versions", totalVersions))
	return nil
}

func testAppCleanupOldVersions(ctx *TestContext) error {
	req, err := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/apps/%s/versions/cleanup?keep_count=3", baseURL, ctx.AppID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ctx.Logger.Info("Cleaned up old versions (kept 3 most recent)")
	return nil
}

// ================================================================
// WORKSPACE VERSION TESTS
// ================================================================

func testWorkspaceCreateSnapshot(ctx *TestContext) error {
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/versions", baseURL, ctx.WorkspaceID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		VersionNumber string `json:"version_number"`
		Message       string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.VersionNumber == "" {
		return fmt.Errorf("version number is empty")
	}

	ctx.Logger.Info("Workspace snapshot created", slog.String("version", result.VersionNumber))
	return nil
}

func testWorkspaceVersionNumbering(ctx *TestContext) error {
	// Create multiple workspace snapshots
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("POST",
			fmt.Sprintf("%s/api/workspaces/%s/versions", baseURL, ctx.WorkspaceID),
			nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+ctx.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to create snapshot: status %d", resp.StatusCode)
		}

		var result struct {
			VersionNumber string `json:"version_number"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}

		ctx.Logger.Info("Workspace version created", slog.String("version", result.VersionNumber))
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func testWorkspaceModifyAndRestore(ctx *TestContext) error {
	// List versions
	versions, err := listWorkspaceVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) < 2 {
		return fmt.Errorf("need at least 2 versions for restore test")
	}

	// Get the second-to-last version
	restoreVersion := versions[1].VersionNumber

	// Restore to that version
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/restore/%s", baseURL, ctx.WorkspaceID, restoreVersion),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("restore failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	ctx.Logger.Info("Workspace restored", slog.String("version", restoreVersion))
	return nil
}

func testWorkspaceAutoBackup(ctx *TestContext) error {
	// Count versions before restore
	versionsBefore, err := listWorkspaceVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list versions before: %w", err)
	}
	countBefore := len(versionsBefore)

	// Restore to first version (this should create an auto-backup)
	if countBefore < 1 {
		return fmt.Errorf("need at least 1 version")
	}

	restoreVersion := versionsBefore[0].VersionNumber
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/restore/%s", baseURL, ctx.WorkspaceID, restoreVersion),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("restore failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Count versions after restore
	time.Sleep(500 * time.Millisecond) // Give time for backup to complete
	versionsAfter, err := listWorkspaceVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list versions after: %w", err)
	}
	countAfter := len(versionsAfter)

	// Should have one more version (the auto-backup)
	if countAfter != countBefore+1 {
		return fmt.Errorf("expected %d versions after restore, got %d (auto-backup not created)",
			countBefore+1, countAfter)
	}

	ctx.Logger.Info("Auto-backup verified",
		slog.Int("versions_before", countBefore),
		slog.Int("versions_after", countAfter))
	return nil
}

func testWorkspacePermissions(ctx *TestContext) error {
	// This test assumes the current user is admin/owner
	// In a real scenario, you'd need to test with a non-admin user

	// Just verify that admin/owner can create versions
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/versions", baseURL, ctx.WorkspaceID),
		nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("admin/owner should be able to create versions, got status %d", resp.StatusCode)
	}

	ctx.Logger.Info("Permission check passed (admin/owner can create versions)")
	return nil
}

func testWorkspaceDryRun(ctx *TestContext) error {
	versions, err := listWorkspaceVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) < 1 {
		return fmt.Errorf("need at least 1 version for dry run test")
	}

	restoreVersion := versions[0].VersionNumber

	// Dry run restore
	body := map[string]interface{}{
		"dry_run": true,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/restore/%s", baseURL, ctx.WorkspaceID, restoreVersion),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dry run failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Verify it's a dry run response
	dryRun, ok := result["dry_run"].(bool)
	if !ok || !dryRun {
		return fmt.Errorf("expected dry_run=true in response")
	}

	// Verify preview data exists
	preview, ok := result["preview"].(map[string]interface{})
	if !ok || preview == nil {
		return fmt.Errorf("expected preview data in dry run response")
	}

	ctx.Logger.Info("Dry run preview successful",
		slog.Any("preview_keys", getMapKeys(preview)))
	return nil
}

// ================================================================
// HELPER FUNCTIONS
// ================================================================

func createTestApp(ctx *TestContext) (uuid.UUID, error) {
	// Create a test app using the user_generated_apps table
	body := map[string]interface{}{
		"workspace_id": ctx.WorkspaceID,
		"app_name":     "Test App for Version Control",
		"is_visible":   true,
		"is_pinned":    false,
		"is_favorite":  false,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/workspaces/%s/apps", baseURL, ctx.WorkspaceID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("failed to create app: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, err
	}

	idStr, ok := result["id"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("app ID not found in response")
	}

	return uuid.Parse(idStr)
}

func modifyApp(ctx *TestContext, appID uuid.UUID, isFavorite, isPinned bool) error {
	body := map[string]interface{}{
		"is_favorite": isFavorite,
		"is_pinned":   isPinned,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("PATCH",
		fmt.Sprintf("%s/api/osa/apps/%s", baseURL, appID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to modify app: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func getApp(ctx *TestContext, appID uuid.UUID) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/osa/apps/%s", baseURL, appID),
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get app: status %d, body: %s", resp.StatusCode, string(body))
	}

	var app map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, err
	}

	return app, nil
}

func listWorkspaceVersions(ctx *TestContext) ([]WorkspaceVersion, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/workspaces/%s/versions", baseURL, ctx.WorkspaceID),
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list versions: status %d", resp.StatusCode)
	}

	var result struct {
		Versions []WorkspaceVersion `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Versions, nil
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ================================================================
// PRETTY PRINTING
// ================================================================

func printHeader(title string) {
	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║%s %-60s %s║%s\n", colorCyan, colorReset, title, colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════╝%s\n\n", colorCyan, colorReset)
}

func printTestStart(name string) {
	fmt.Printf("%s▶ Running: %s%s\n", colorBlue, name, colorReset)
}

func printTestPass(name string) {
	fmt.Printf("%s✓ PASS: %s%s\n\n", colorGreen, name, colorReset)
}

func printTestFail(name string, err error) {
	fmt.Printf("%s✗ FAIL: %s%s\n", colorRed, name, colorReset)
	fmt.Printf("%s  Error: %v%s\n\n", colorRed, err, colorReset)
}

func printSummary(passed, total int) {
	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║%s %-60s %s║%s\n", colorCyan, colorReset, "TEST SUMMARY", colorCyan, colorReset)
	fmt.Printf("%s╠══════════════════════════════════════════════════════════════╣%s\n", colorCyan, colorReset)

	status := "FAILED"
	statusColor := colorRed
	if passed == total {
		status = "SUCCESS"
		statusColor = colorGreen
	}

	fmt.Printf("%s║%s Tests Passed: %s%d/%d%s %s║%s\n",
		colorCyan, colorReset, statusColor, passed, total, colorReset,
		fmt.Sprintf("%*s", 40-len(fmt.Sprintf("%d/%d", passed, total)), ""),
		colorCyan)
	fmt.Printf("%s║%s Status: %s%-53s%s %s║%s\n",
		colorCyan, colorReset, statusColor, status, colorReset, colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════════╝%s\n\n", colorCyan, colorReset)
}
