//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Check if workspace exists
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspaces").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to check workspaces: %v", err)
	}

	if count > 0 {
		fmt.Printf("✅ Workspaces exist: %d found\n", count)

		// Show first workspace
		var id uuid.UUID
		var name string
		err = pool.QueryRow(ctx, "SELECT id, name FROM workspaces LIMIT 1").Scan(&id, &name)
		if err == nil {
			fmt.Printf("   First workspace: %s (%s)\n", name, id)
		}
		return
	}

	// Create test workspace
	fmt.Println("No workspaces found. Creating test workspace...")

	workspaceID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, slug, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, workspaceID, "Test Workspace", "test-workspace", "Created for OSA worker testing")

	if err != nil {
		log.Fatalf("Failed to create workspace: %v", err)
	}

	fmt.Printf("✅ Created test workspace: %s\n", workspaceID)
}
