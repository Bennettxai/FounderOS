//go:build ignore

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
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	fmt.Println("=== Database Connection Test ===")
	fmt.Printf("Connecting to: %s\n", maskPassword(dbURL))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection successful!")

	// Check migration status
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'app_generation_queue'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Printf("❌ Failed to check migration status: %v", err)
		return
	}

	if tableExists {
		fmt.Println("✅ Migration 089 tables exist")

		// Check queue status
		var pending, processing, completed, failed int
		err = pool.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE status = 'pending') as pending,
				COUNT(*) FILTER (WHERE status = 'processing') as processing,
				COUNT(*) FILTER (WHERE status = 'completed') as completed,
				COUNT(*) FILTER (WHERE status = 'failed') as failed
			FROM app_generation_queue
		`).Scan(&pending, &processing, &completed, &failed)
		if err != nil {
			log.Printf("❌ Failed to query queue status: %v", err)
			return
		}

		fmt.Printf("📊 Queue Status: pending=%d, processing=%d, completed=%d, failed=%d\n",
			pending, processing, completed, failed)
	} else {
		fmt.Println("⚠️  Migration 089 tables NOT found - migration needs to be run")
	}
}

func maskPassword(dbURL string) string {
	// Simple masking for display
	return dbURL[:20] + "..." + dbURL[len(dbURL)-20:]
}
