//go:build ignore

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Load .env file
	if err := godotenv.Load("../../.env"); err != nil {
		slog.Warn("No .env file found, using environment variables")
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL not set")
		os.Exit(1)
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("Connected to database successfully")

	// Read migration file
	migrationPath := filepath.Join("..", "..", "internal", "database", "migrations", "036_background_jobs.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		slog.Error("Failed to read migration file", "error", err, "path", migrationPath)
		os.Exit(1)
	}

	slog.Info("Running migration 036_background_jobs.sql...")

	// Execute migration
	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		slog.Error("Failed to execute migration", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Migration 036 applied successfully!")

	// Verify tables created
	var tableCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_name IN ('background_jobs', 'scheduled_jobs')
	`).Scan(&tableCount)
	if err != nil {
		slog.Error("Failed to verify tables", "error", err)
		os.Exit(1)
	}

	if tableCount == 2 {
		slog.Info("✅ Tables verified: background_jobs, scheduled_jobs")
	} else {
		slog.Warn("⚠️  Expected 2 tables, found", "count", tableCount)
	}

	// Verify functions created
	var funcCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_proc
		WHERE proname IN ('acquire_background_job', 'calculate_retry_time', 'release_stuck_jobs')
	`).Scan(&funcCount)
	if err == nil && funcCount == 3 {
		slog.Info("✅ Functions verified: acquire_background_job, calculate_retry_time, release_stuck_jobs")
	}

	fmt.Println("\n🎉 Migration 036 completed successfully!")
}
