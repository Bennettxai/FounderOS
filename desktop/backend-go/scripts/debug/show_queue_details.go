package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
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

	fmt.Println("📊 Queue Items (last 10):")
	fmt.Println("=" + string(make([]byte, 80)))

	rows, err := pool.Query(ctx, `
		SELECT id, status, created_at, started_at, completed_at, error_message
		FROM app_generation_queue
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var status string
		var createdAt time.Time
		var startedAt, completedAt *time.Time
		var errorMsg *string

		rows.Scan(&id, &status, &createdAt, &startedAt, &completedAt, &errorMsg)

		shortID := id[:8]
		created := createdAt.Format("15:04:05")
		started := "-"
		completed := "-"
		errStr := "-"

		if startedAt != nil {
			started = startedAt.Format("15:04:05")
		}
		if completedAt != nil {
			completed = completedAt.Format("15:04:05")
		}
		if errorMsg != nil && *errorMsg != "" {
			if len(*errorMsg) > 30 {
				errStr = (*errorMsg)[:30] + "..."
			} else {
				errStr = *errorMsg
			}
		}

		fmt.Printf("ID: %s | status: %-10s | created: %s | started: %s | completed: %s | error: %s\n",
			shortID, status, created, started, completed, errStr)
	}
}
