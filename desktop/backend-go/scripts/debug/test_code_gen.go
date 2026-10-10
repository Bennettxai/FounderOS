// +build ignore

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
	// Load .env - try multiple paths
	envPaths := []string{".env", "../../.env", "../../../.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from: %s", path)
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

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}
	fmt.Println("✅ Connected to database")

	// Check queue status
	var pendingCount, processingCount, completedCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'pending'").Scan(&pendingCount)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'processing'").Scan(&processingCount)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'completed'").Scan(&completedCount)

	fmt.Printf("📊 Queue status: pending=%d, processing=%d, completed=%d\n", pendingCount, processingCount, completedCount)

	// Check if we need to insert a test item
	if pendingCount == 0 && processingCount == 0 {
		fmt.Println("⚡ Inserting test queue item...")

		genContext := map[string]interface{}{
			"app_name":    "TestTodoApp",
			"description": "A simple todo list application with add, edit, delete functionality",
			"features": []string{
				"Add new todos",
				"Mark todos as complete",
				"Delete todos",
			},
		}
		contextJSON, _ := json.Marshal(genContext)

		// Need a valid workspace_id - get one from users table
		var workspaceID string
		err := pool.QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&workspaceID)
		if err != nil {
			log.Fatalf("No users found: %v", err)
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO app_generation_queue (workspace_id, generation_context, status, created_at)
			VALUES ($1, $2, 'pending', NOW())
		`, workspaceID, contextJSON)

		if err != nil {
			log.Fatalf("Failed to insert: %v", err)
		}
		fmt.Println("✅ Test queue item inserted")
	}

	// Show recent items
	fmt.Println("\n📋 Recent queue items:")
	rows, _ := pool.Query(ctx, `
		SELECT id, status, created_at, started_at, completed_at, error_message
		FROM app_generation_queue
		ORDER BY created_at DESC
		LIMIT 5
	`)
	defer rows.Close()

	for rows.Next() {
		var id, status string
		var createdAt time.Time
		var startedAt, completedAt *time.Time
		var errorMsg *string
		rows.Scan(&id, &status, &createdAt, &startedAt, &completedAt, &errorMsg)

		errStr := ""
		if errorMsg != nil {
			errStr = fmt.Sprintf(" (error: %s)", *errorMsg)
		}
		fmt.Printf("  - %s: %s @ %v%s\n", id[:8], status, createdAt.Format("15:04:05"), errStr)
	}

	// Check generated files
	var fileCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM osa_generated_files").Scan(&fileCount)
	fmt.Printf("\n📁 Generated files in DB: %d\n", fileCount)

	if fileCount > 0 {
		fmt.Println("\n📄 Recent generated files:")
		fileRows, _ := pool.Query(ctx, `
			SELECT file_path, file_name, language, file_size_bytes
			FROM osa_generated_files
			ORDER BY created_at DESC
			LIMIT 10
		`)
		defer fileRows.Close()

		for fileRows.Next() {
			var filePath, fileName string
			var language *string
			var size int32
			fileRows.Scan(&filePath, &fileName, &language, &size)
			lang := "unknown"
			if language != nil {
				lang = *language
			}
			fmt.Printf("  - %s (%s, %d bytes)\n", filePath, lang, size)
		}
	}
}
