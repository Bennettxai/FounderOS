//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Cleanup test data from database after E2E testing
// Usage: go run scripts/cleanup_test_data.go [--dry-run]
// Use --dry-run to see what would be deleted without actually deleting

func main() {
	dryRun := false
	if len(os.Args) > 1 && os.Args[1] == "--dry-run" {
		dryRun = true
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              CLEANUP TEST DATA FROM DATABASE                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	if dryRun {
		fmt.Println("🔍 DRY RUN MODE - No data will be deleted")
		fmt.Println()
	} else {
		fmt.Println("⚠️  WARNING: This will DELETE test data from database!")
		fmt.Println("   Press Ctrl+C within 5 seconds to cancel...")
		fmt.Println()
		// time.Sleep(5 * time.Second)
		// Commented out to make it safer - user must remove comment to enable
		fmt.Println("❌ Safety check failed: Uncomment the sleep line to enable deletion")
		fmt.Println("   Edit scripts/cleanup_test_data.go and uncomment time.Sleep")
		os.Exit(1)
	}

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: .env file not found, using environment variables")
	}

	ctx := context.Background()

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL not set")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	fmt.Println("✅ Connected to database")
	fmt.Println()

	// Define test data patterns
	testPatterns := []struct {
		table       string
		description string
		condition   string
	}{
		{
			table:       "\"user\"",
			description: "Test users",
			condition:   "email LIKE 'test%@%' OR email LIKE 'demo%@%' OR name LIKE 'Test %'",
		},
		{
			table:       "workspace",
			description: "Test workspaces",
			condition:   "slug LIKE 'test-%' OR name LIKE 'Test %'",
		},
		{
			table:       "workspace_versions",
			description: "Test workspace versions",
			condition:   "workspace_id IN (SELECT id FROM workspace WHERE slug LIKE 'test-%')",
		},
		{
			table:       "onboarding_user_analysis",
			description: "Test onboarding analysis",
			condition:   "workspace_id IN (SELECT id FROM workspace WHERE slug LIKE 'test-%')",
		},
		{
			table:       "workspace_onboarding_profiles",
			description: "Test onboarding profiles",
			condition:   "workspace_id IN (SELECT id FROM workspace WHERE slug LIKE 'test-%')",
		},
		{
			table:       "app_generation_queue",
			description: "Test generation queue items",
			condition:   "workspace_id IN (SELECT id FROM workspace WHERE slug LIKE 'test-%')",
		},
		{
			table:       "user_generated_apps",
			description: "Test generated apps",
			condition:   "workspace_id IN (SELECT id FROM workspace WHERE slug LIKE 'test-%')",
		},
		{
			table:       "onboarding_email_metadata",
			description: "Test email metadata",
			condition:   "user_id IN (SELECT id FROM \"user\" WHERE email LIKE 'test%@%')",
		},
	}

	totalAffected := 0

	// Process each table
	for _, pattern := range testPatterns {
		fmt.Printf("🔍 Checking %s: %s\n", pattern.table, pattern.description)

		// Count rows that would be affected
		var count int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", pattern.table, pattern.condition)
		err := pool.QueryRow(ctx, countQuery).Scan(&count)
		if err != nil {
			fmt.Printf("   ⚠️  Error counting: %v\n", err)
			continue
		}

		if count == 0 {
			fmt.Printf("   ✅ No test data found\n")
			continue
		}

		fmt.Printf("   📊 Found %d test records\n", count)
		totalAffected += count

		if !dryRun {
			// Delete test data
			deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE %s", pattern.table, pattern.condition)
			result, err := pool.Exec(ctx, deleteQuery)
			if err != nil {
				fmt.Printf("   ❌ Error deleting: %v\n", err)
				continue
			}
			fmt.Printf("   ✅ Deleted %d records\n", result.RowsAffected())
		} else {
			fmt.Printf("   🔍 Would delete %d records (dry run)\n", count)
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      CLEANUP SUMMARY                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

	if dryRun {
		fmt.Printf("🔍 DRY RUN: Would delete %d total records\n", totalAffected)
		fmt.Println()
		fmt.Println("To actually delete, run:")
		fmt.Println("  go run scripts/cleanup_test_data.go")
		fmt.Println()
		fmt.Println("⚠️  Remember to uncomment the safety sleep in the script first!")
	} else {
		fmt.Printf("✅ Deleted %d total test records\n", totalAffected)
		fmt.Println()
		fmt.Println("🎉 Cleanup complete!")
	}
}
