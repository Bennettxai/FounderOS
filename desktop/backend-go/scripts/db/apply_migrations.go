//go:build ignore

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Remove pgbouncer parameter and use direct connection
	dbURL = strings.Replace(dbURL, "?pgbouncer=true", "", 1)
	dbURL = strings.Replace(dbURL, "&pgbouncer=true", "", 1)
	// Change from pooler to direct database host if SUPABASE_DIRECT_HOST is set
	if directHost := os.Getenv("SUPABASE_DIRECT_HOST"); directHost != "" {
		dbURL = strings.Replace(dbURL, "aws-0-us-east-1.pooler.supabase.com:6543", directHost, 1)
	}

	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("✅ Connected to database")

	// Check existing migrations
	applied, err := getAppliedMigrations(ctx, pool)
	if err != nil {
		log.Printf("Warning: Could not check applied migrations: %v", err)
		applied = make(map[string]bool)
	}

	// Get migrations to apply
	migrationsToApply := []string{
		"082_onboarding_user_analysis.sql",
		"083_onboarding_starter_apps.sql",
		"084_onboarding_email_metadata.sql",
		"085_workspace_versions.sql",
	}

	migrationsDir := "../supabase/migrations"

	for _, migrationFile := range migrationsToApply {
		if applied[migrationFile] {
			log.Printf("⏭️  Skipping %s (already applied)", migrationFile)
			continue
		}

		migrationPath := filepath.Join(migrationsDir, migrationFile)

		log.Printf("📝 Applying migration: %s", migrationFile)

		// Read migration
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			log.Fatalf("Failed to read migration %s: %v", migrationFile, err)
		}

		// Apply migration in transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("Failed to begin transaction: %v", err)
		}

		_, err = tx.Exec(ctx, string(migrationSQL))
		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to execute migration %s: %v", migrationFile, err)
		}

		// Record migration in schema_migrations table (if exists)
		_, err = tx.Exec(ctx, `
			INSERT INTO schema_migrations (version, applied_at)
			VALUES ($1, NOW())
			ON CONFLICT (version) DO NOTHING
		`, migrationFile)
		if err != nil {
			// Table might not exist, that's OK
			log.Printf("Warning: Could not record migration in schema_migrations: %v", err)
		}

		err = tx.Commit(ctx)
		if err != nil {
			log.Fatalf("Failed to commit migration %s: %v", migrationFile, err)
		}

		log.Printf("✅ Applied: %s", migrationFile)
	}

	log.Println("\n🎉 All migrations applied successfully!")

	// Verify tables created
	tables := []string{
		"onboarding_user_analysis",
		"onboarding_starter_apps",
		"onboarding_email_metadata",
		"workspace_versions",
	}

	log.Println("\n🔍 Verifying tables...")
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			log.Printf("⚠️  Error checking table %s: %v", table, err)
		} else if exists {
			log.Printf("✅ Table exists: %s", table)
		} else {
			log.Printf("❌ Table NOT found: %s", table)
		}
	}
}

func getAppliedMigrations(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	// Check if schema_migrations table exists
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil || !exists {
		return make(map[string]bool), nil
	}

	// Get applied migrations
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			continue
		}
		applied[version] = true
	}

	return applied, nil
}
