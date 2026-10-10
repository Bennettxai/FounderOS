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

	// Reset processing items to pending
	fmt.Println("🔄 Resetting processing items to pending...")
	result, err := pool.Exec(ctx, `
		UPDATE app_generation_queue
		SET status = 'pending', started_at = NULL
		WHERE status = 'processing'
	`)
	if err != nil {
		log.Fatalf("Failed to reset: %v", err)
	}
	fmt.Printf("✅ Reset %d items to pending\n", result.RowsAffected())

	// Check status
	var pending, processing, completed int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'pending'").Scan(&pending)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'processing'").Scan(&processing)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_generation_queue WHERE status = 'completed'").Scan(&completed)
	fmt.Printf("📊 Queue: pending=%d, processing=%d, completed=%d\n", pending, processing, completed)
}
