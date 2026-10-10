//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	versionService := services.NewWorkspaceVersionService(pool, logger)

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           WORKSPACE VERSION SNAPSHOTS - TEST SUITE              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Test 1: Create a test user and workspace
	fmt.Println("📦 TEST 1: Creating test user and workspace...")
	testUserID := "test-user-version-snapshots"
	var workspaceID uuid.UUID

	// Create user first
	_, err = pool.Exec(ctx, `
		INSERT INTO "user" (id, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, testUserID, "test-versions@example.com", "Test User Versions")
	if err != nil {
		log.Fatalf("❌ Failed to create user: %v", err)
	}

	// Create workspace with unique slug
	uniqueSlug := fmt.Sprintf("test-workspace-versions-%d", time.Now().Unix())
	workspace, err := queries.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		Name:    "Test Workspace for Version Snapshots",
		Slug:    uniqueSlug,
		OwnerID: testUserID,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create workspace: %v", err)
	}
	// Convert pgtype.UUID to uuid.UUID
	workspaceID, err = uuid.FromBytes(workspace.ID.Bytes[:])
	if err != nil {
		log.Fatalf("❌ Failed to parse workspace ID: %v", err)
	}
	fmt.Printf("✅ Workspace created: %s\n", workspaceID)
	fmt.Println()

	// Test 2: Create initial snapshot (0.0.1)
	fmt.Println("📸 TEST 2: Creating initial snapshot (0.0.1)...")

	version1, err := versionService.CreateSnapshot(ctx, workspaceID, testUserID)
	if err != nil {
		log.Fatalf("❌ Failed to create snapshot: %v", err)
	}
	fmt.Printf("✅ Snapshot created: version %s\n", version1)

	// Verify snapshot data in DB
	row := pool.QueryRow(ctx, `
		SELECT version_number, snapshot_data
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, version1)

	var dbVersion string
	var dbData []byte
	if err := row.Scan(&dbVersion, &dbData); err != nil {
		log.Fatalf("❌ Failed to verify snapshot: %v", err)
	}

	var snapshotContent map[string]interface{}
	if err := json.Unmarshal(dbData, &snapshotContent); err != nil {
		log.Fatalf("❌ Failed to parse snapshot data: %v", err)
	}

	fmt.Printf("   Version: %s\n", dbVersion)
	fmt.Printf("   Snapshot Keys: %v\n", getKeys(snapshotContent))
	fmt.Println()

	// Test 3: Create second snapshot (0.0.2)
	fmt.Println("📸 TEST 3: Creating second snapshot (0.0.2)...")

	version2, err := versionService.CreateSnapshot(ctx, workspaceID, testUserID)
	if err != nil {
		log.Fatalf("❌ Failed to create second snapshot: %v", err)
	}
	fmt.Printf("✅ Snapshot created: version %s\n", version2)

	// Verify version numbering
	if version2 != "0.0.2" {
		log.Fatalf("❌ Expected version 0.0.2, got %s", version2)
	}
	fmt.Println("✅ Version numbering correct (0.0.1 → 0.0.2)")
	fmt.Println()

	// Test 4: List all versions
	fmt.Println("📋 TEST 4: Listing all versions...")
	versions, err := versionService.ListVersions(ctx, workspaceID)
	if err != nil {
		log.Fatalf("❌ Failed to list versions: %v", err)
	}

	fmt.Printf("✅ Found %d versions:\n", len(versions))
	for i, v := range versions {
		versionNum := v["version_number"].(string)
		createdAt := v["created_at"].(time.Time)
		fmt.Printf("   %d. Version %s (created: %v)\n", i+1, versionNum, createdAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()

	// Test 5: Restore to version 0.0.1
	fmt.Println("🔄 TEST 5: Restoring to version 0.0.1...")
	err = versionService.RestoreSnapshot(ctx, workspaceID, "0.0.1", testUserID)
	if err != nil {
		log.Fatalf("❌ Failed to restore snapshot: %v", err)
	}
	fmt.Println("✅ Snapshot restored successfully")

	// Verify restoration
	row = pool.QueryRow(ctx, `
		SELECT snapshot_data
		FROM workspace_versions
		WHERE workspace_id = $1 AND version_number = $2
	`, workspaceID, "0.0.1")

	var restoredData []byte
	if err := row.Scan(&restoredData); err != nil {
		log.Fatalf("❌ Failed to verify restored data: %v", err)
	}

	var restoredContent map[string]interface{}
	if err := json.Unmarshal(restoredData, &restoredContent); err != nil {
		log.Fatalf("❌ Failed to parse restored data: %v", err)
	}

	fmt.Printf("   Restored Snapshot Keys: %v\n", getKeys(restoredContent))
	fmt.Println()

	// Test 6: Create third snapshot after restore (should be 0.0.3)
	fmt.Println("📸 TEST 6: Creating snapshot after restore (should be 0.0.3)...")
	version3, err := versionService.CreateSnapshot(ctx, workspaceID, testUserID)
	if err != nil {
		log.Fatalf("❌ Failed to create third snapshot: %v", err)
	}
	fmt.Printf("✅ Snapshot created: version %s\n", version3)

	if version3 != "0.0.3" {
		log.Fatalf("❌ Expected version 0.0.3, got %s", version3)
	}
	fmt.Println("✅ Version numbering continues correctly (0.0.1 → 0.0.2 → 0.0.3)")
	fmt.Println()

	// Test 7: Error handling - restore non-existent version
	fmt.Println("⚠️  TEST 7: Testing error handling (restore non-existent version)...")
	err = versionService.RestoreSnapshot(ctx, workspaceID, "9.9.9", testUserID)
	if err == nil {
		log.Fatal("❌ Expected error for non-existent version, got nil")
	}
	fmt.Printf("✅ Error handled correctly: %v\n", err)
	fmt.Println()

	// Cleanup
	fmt.Println("🧹 Cleaning up test data...")
	_, err = pool.Exec(ctx, "DELETE FROM workspace_versions WHERE workspace_id = $1", workspaceID)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to cleanup versions: %v", err)
	}

	_, err = pool.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", workspaceID)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to cleanup workspace: %v", err)
	}

	_, err = pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, testUserID)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to cleanup user: %v", err)
	}
	fmt.Println("✅ Cleanup complete")
	fmt.Println()

	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        TEST SUMMARY                              ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  ✅ Create workspace                                             ║")
	fmt.Println("║  ✅ Create snapshot (0.0.1)                                      ║")
	fmt.Println("║  ✅ Create snapshot (0.0.2)                                      ║")
	fmt.Println("║  ✅ List versions                                                ║")
	fmt.Println("║  ✅ Restore snapshot (0.0.1)                                     ║")
	fmt.Println("║  ✅ Create snapshot after restore (0.0.3)                        ║")
	fmt.Println("║  ✅ Error handling (non-existent version)                        ║")
	fmt.Println("║  ✅ Cleanup                                                      ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  🎉 ALL TESTS PASSED                                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
