// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	envPaths := []string{".env", "../../.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	fmt.Println("🔄 Resetting stuck 'processing' items to 'pending'...")
	result, err := pool.Exec(ctx, `
		UPDATE app_generation_queue
		SET status = 'pending',
		    started_at = NULL,
		    retry_count = 0,
		    error_message = NULL
		WHERE status = 'processing'
	`)
	if err != nil {
		log.Fatalf("Reset failed: %v", err)
	}
	fmt.Printf("   Reset %d items\n", result.RowsAffected())

	// Insert a fresh test item
	fmt.Println("\n⚡ Inserting fresh test queue item...")

	genContext := map[string]interface{}{
		"app_name":    "SimpleCounter",
		"description": "A minimal counter app with increment and decrement buttons",
		"features": []string{
			"Increment counter",
			"Decrement counter",
			"Reset to zero",
		},
	}
	contextJSON, _ := json.Marshal(genContext)

	// Get a valid workspace_id from existing queue items
	var workspaceID string
	err = pool.QueryRow(ctx, "SELECT workspace_id FROM app_generation_queue LIMIT 1").Scan(&workspaceID)
	if err != nil {
		// Use default dev workspace ID
		workspaceID = "00000000-0000-0000-0000-000000000001"
		log.Printf("Using default workspace_id: %s", workspaceID)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO app_generation_queue (workspace_id, generation_context, status, created_at)
		VALUES ($1, $2::jsonb, 'pending', NOW())
	`, workspaceID, string(contextJSON))

	if err != nil {
		log.Fatalf("Failed to insert: %v", err)
	}
	fmt.Println("✅ Test queue item inserted")

	// Show status
	var pendingCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'pending'").Scan(&pendingCount)
	fmt.Printf("\n📊 Pending items ready to process: %d\n", pendingCount)
	fmt.Println("\n🚀 Now run the server with: go run ./cmd/server")
	fmt.Println("   Watch the logs for DEBUG output to see model responses")
}
