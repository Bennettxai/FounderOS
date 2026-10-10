package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	godotenv.Load(".env")

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

	fmt.Println("✅ Connected to database")

	// Check queue status
	var pending, processing, completed int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'pending'").Scan(&pending)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'processing'").Scan(&processing)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'completed'").Scan(&completed)
	fmt.Printf("📊 Queue: pending=%d, processing=%d, completed=%d\n", pending, processing, completed)

	// Check generated files count
	var fileCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM osa_generated_files").Scan(&fileCount)
	fmt.Printf("📁 Generated files in DB: %d\n", fileCount)

	// Show recent files if any
	if fileCount > 0 {
		fmt.Println("\n📄 Recent generated files:")
		rows, _ := pool.Query(ctx, `
			SELECT file_path, language, file_size_bytes, created_at
			FROM osa_generated_files
			ORDER BY created_at DESC
			LIMIT 5
		`)
		defer rows.Close()
		for rows.Next() {
			var path string
			var lang *string
			var size int32
			var createdAt time.Time
			rows.Scan(&path, &lang, &size, &createdAt)
			langStr := "unknown"
			if lang != nil {
				langStr = *lang
			}
			fmt.Printf("   - %s (%s, %d bytes) @ %s\n", path, langStr, size, createdAt.Format("15:04:05"))
		}
	}

	// If no pending items, insert a test one
	if pending == 0 && processing == 0 {
		fmt.Println("\n⚡ Inserting test queue item...")

		// Get valid workspace_id from existing data
		var workspaceID string
		err := pool.QueryRow(ctx, "SELECT workspace_id FROM app_generation_queue LIMIT 1").Scan(&workspaceID)
		if err != nil {
			workspaceID = "00000000-0000-0000-0000-000000000001"
			fmt.Printf("   Using default workspace_id: %s\n", workspaceID)
		}

		genContext := map[string]interface{}{
			"app_name":    "TestCounter",
			"description": "A simple counter app with increment and decrement",
			"features": []string{
				"Increment counter",
				"Decrement counter",
				"Reset to zero",
			},
		}
		contextJSON, _ := json.Marshal(genContext)

		_, err = pool.Exec(ctx, `
			INSERT INTO app_generation_queue (workspace_id, generation_context, status, created_at)
			VALUES ($1, $2::jsonb, 'pending', NOW())
		`, workspaceID, string(contextJSON))

		if err != nil {
			log.Fatalf("Failed to insert: %v", err)
		}
		fmt.Println("✅ Test queue item inserted!")
		fmt.Println("\n🔄 The server will pick it up within 5 seconds...")
		fmt.Println("   Monitor logs for: 'processing queue item'")
	}
}
