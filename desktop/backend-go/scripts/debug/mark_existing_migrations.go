package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer pool.Close()

	fmt.Println("✅ Database connection successful")

	// Check which migrations are already applied based on table existence
	migrationsToMark := []struct {
		version    string
		name       string
		checkTable string
	}{
		{"025", "Image Embeddings", "image_embeddings"},
		{"026", "Workspaces and Roles", "workspaces"},
		{"027", "Workspace Invites", "workspace_invites"},
		{"028", "Workspace Audit Logs", "workspace_audit_logs"},
		{"029", "Project Members", "project_members"},
	}

	fmt.Println("\n🔍 Checking which migrations are already applied...")

	for _, mig := range migrationsToMark {
		// Check if table exists
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
		err := pool.QueryRow(context.Background(), query, mig.checkTable).Scan(&exists)
		if err != nil {
			log.Printf("❌ Error checking table %s: %v", mig.checkTable, err)
			continue
		}

		if exists {
			// Check if already marked in schema_migrations
			var alreadyMarked bool
			err := pool.QueryRow(context.Background(),
				"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
				mig.version).Scan(&alreadyMarked)
			if err != nil {
				log.Printf("❌ Error checking migration status for %s: %v", mig.version, err)
				continue
			}

			if !alreadyMarked {
				// Mark as applied
				_, err = pool.Exec(context.Background(),
					"INSERT INTO schema_migrations (version) VALUES ($1)",
					mig.version)
				if err != nil {
					log.Printf("❌ Failed to mark migration %s: %v", mig.version, err)
					continue
				}
				fmt.Printf("✅ Marked migration %s (%s) as applied\n", mig.version, mig.name)
			} else {
				fmt.Printf("⏭️  Migration %s (%s) already marked\n", mig.version, mig.name)
			}
		} else {
			fmt.Printf("⚠️  Table %s doesn't exist - migration %s not yet applied\n", mig.checkTable, mig.version)
		}
	}

	// Show all applied migrations
	fmt.Println("\n📋 Final migration status:")
	rows, err := pool.Query(context.Background(), "SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		log.Fatal("Failed to query migrations:", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var version string
		var appliedAt interface{}
		if err := rows.Scan(&version, &appliedAt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("   ✓ %s (applied: %v)\n", version, appliedAt)
		count++
	}

	fmt.Printf("\n✅ Total migrations applied: %d\n", count)
}
