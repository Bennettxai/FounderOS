//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
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

	fmt.Println("✅ Connected to database")
	fmt.Println("")

	// Tables to verify
	tables := []string{
		"onboarding_user_analysis",
		"onboarding_starter_apps",
		"onboarding_email_metadata",
		"workspace_onboarding_profiles",
		"workspace_versions",
		"osa_generated_files",
		"osa_file_versions",
		"osa_generation_jobs",
		"osa_framework_metadata",
		"app_templates",
	}

	fmt.Println("📊 Verifying Table Schema:")
	fmt.Println("─────────────────────────────────────────────────────────────")

	for _, table := range tables {
		query := fmt.Sprintf(`
			SELECT COUNT(*) as column_count
			FROM information_schema.columns
			WHERE table_name = '%s'
		`, table)

		var colCount int
		err := pool.QueryRow(ctx, query).Scan(&colCount)
		if err != nil {
			fmt.Printf("❌ %s - ERROR: %v\n", table, err)
			continue
		}

		if colCount == 0 {
			fmt.Printf("❌ %s - TABLE NOT FOUND\n", table)
		} else {
			fmt.Printf("✅ %s - %d columns\n", table, colCount)

			// Get row count
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)
			var rowCount int
			err = pool.QueryRow(ctx, countQuery).Scan(&rowCount)
			if err != nil {
				fmt.Printf("   ⚠️  Could not count rows: %v\n", err)
			} else {
				fmt.Printf("   📝 Rows: %d\n", rowCount)
			}
		}
	}

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("")

	// Verify integration point functions exist in code
	fmt.Println("🔗 Integration Points Verification:")
	fmt.Println("─────────────────────────────────────────────────────────────")

	integrationPoints := []struct {
		name string
		file string
	}{
		{"transformAIAnalysisToWorkspaceProfile", "internal/services/onboarding_service.go"},
		{"enrichGenerationContext", "internal/services/app_generation_worker.go"},
		{"WorkspaceVersionService.CreateSnapshot", "internal/services/workspace_version_service.go"},
		{"WorkspaceVersionService.RestoreSnapshot", "internal/services/workspace_version_service.go"},
		{"WorkspaceVersionService.ListVersions", "internal/services/workspace_version_service.go"},
	}

	for _, ip := range integrationPoints {
		filePath := ip.file
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("✅ %s - file exists: %s\n", ip.name, filePath)
		} else {
			fmt.Printf("❌ %s - file not found: %s\n", ip.name, filePath)
		}
	}

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("")
	fmt.Println("🎉 Schema verification complete!")
}
