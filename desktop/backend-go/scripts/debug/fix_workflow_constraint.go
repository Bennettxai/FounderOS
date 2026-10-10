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

	// Make workflow_id nullable
	fmt.Println("🔧 Making workflow_id nullable...")
	_, err = pool.Exec(ctx, `ALTER TABLE osa_generated_files ALTER COLUMN workflow_id DROP NOT NULL`)
	if err != nil {
		// Check if already nullable
		if err.Error() != "" {
			log.Printf("Note: %v", err)
		}
	}

	fmt.Println("✅ workflow_id is now nullable")

	// Verify
	var isNullable string
	err = pool.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_name = 'osa_generated_files' AND column_name = 'workflow_id'
	`).Scan(&isNullable)
	if err != nil {
		log.Printf("Failed to verify: %v", err)
	} else {
		fmt.Printf("📊 workflow_id is_nullable: %s\n", isNullable)
	}
}
