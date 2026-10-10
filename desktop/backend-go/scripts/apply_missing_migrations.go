//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

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

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║             APPLYING MISSING MIGRATIONS                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// List of migrations to apply
	migrations := []string{
		"068_osa_integration.sql",
		"074_osa_workflows_files.sql",
		"076_osa_deployment_port.sql",
		"078_osa_app_metadata.sql",
		"081_app_templates_system.sql",
		"082_onboarding_user_analysis.sql",
		"084_onboarding_email_metadata.sql",
		"086_performance_composite_indexes.sql",
	}

	sort.Strings(migrations)

	for _, migrationFile := range migrations {
		// Check if already applied
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)
		`, migrationFile).Scan(&exists)

		if err != nil {
			log.Printf("❌ Error checking migration %s: %v\n", migrationFile, err)
			continue
		}

		if exists {
			fmt.Printf("⏭️  %s - already applied\n", migrationFile)
			continue
		}

		// Read migration file
		migrationPath := filepath.Join("supabase", "migrations", migrationFile)
		content, err := os.ReadFile(migrationPath)
		if err != nil {
			log.Printf("❌ Error reading %s: %v\n", migrationFile, err)
			continue
		}

		// Execute migration in transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Printf("❌ Error starting transaction for %s: %v\n", migrationFile, err)
			continue
		}

		// Execute SQL
		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			tx.Rollback(ctx)
			log.Printf("❌ Error applying %s: %v\n", migrationFile, err)
			continue
		}

		// Record in schema_migrations
		_, err = tx.Exec(ctx, `
			INSERT INTO schema_migrations (version) VALUES ($1)
			ON CONFLICT (version) DO NOTHING
		`, migrationFile)
		if err != nil {
			tx.Rollback(ctx)
			log.Printf("❌ Error recording %s: %v\n", migrationFile, err)
			continue
		}

		// Commit
		err = tx.Commit(ctx)
		if err != nil {
			log.Printf("❌ Error committing %s: %v\n", migrationFile, err)
			continue
		}

		fmt.Printf("✅ %s - applied successfully\n", migrationFile)
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    MIGRATIONS COMPLETE                           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}
