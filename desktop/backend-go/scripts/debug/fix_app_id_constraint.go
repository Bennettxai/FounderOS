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

	// Drop the foreign key constraint on app_id
	fmt.Println("🔧 Dropping foreign key constraint on app_id...")
	_, err = pool.Exec(ctx, `ALTER TABLE osa_generated_files DROP CONSTRAINT IF EXISTS osa_generated_files_app_id_fkey`)
	if err != nil {
		log.Printf("Warning: %v", err)
	}
	fmt.Println("✅ Foreign key constraint dropped (app_id can now be any UUID)")

	// Verify columns
	rows, err := pool.Query(ctx, `
		SELECT column_name, is_nullable, data_type
		FROM information_schema.columns
		WHERE table_name = 'osa_generated_files'
		AND column_name IN ('workflow_id', 'app_id')
		ORDER BY column_name
	`)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 Column status:")
	for rows.Next() {
		var col, nullable, dataType string
		rows.Scan(&col, &nullable, &dataType)
		fmt.Printf("   %s: %s, nullable=%s\n", col, dataType, nullable)
	}
}
