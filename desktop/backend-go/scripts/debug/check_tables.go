// +build ignore

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

	fmt.Println("📋 Tables in database:")
	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf("  - %s\n", name)
	}

	// Check app_generation_queue structure
	fmt.Println("\n📊 app_generation_queue columns:")
	colRows, _ := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'app_generation_queue'
		ORDER BY ordinal_position
	`)
	defer colRows.Close()

	for colRows.Next() {
		var name, dtype, nullable string
		colRows.Scan(&name, &dtype, &nullable)
		fmt.Printf("  - %s (%s, nullable=%s)\n", name, dtype, nullable)
	}

	// Get existing workspace_id from queue
	fmt.Println("\n🔍 Getting workspace_id from existing queue item...")
	var wsID string
	err = pool.QueryRow(ctx, "SELECT workspace_id FROM app_generation_queue LIMIT 1").Scan(&wsID)
	if err != nil {
		fmt.Printf("   No items found: %v\n", err)
	} else {
		fmt.Printf("   Found workspace_id: %s\n", wsID)
	}
}
