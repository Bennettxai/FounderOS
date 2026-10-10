//go:build ignore

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
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	fmt.Println("Making template_id nullable in app_generation_queue...")
	fmt.Println("This allows pure AI generative mode (without templates)...")

	// Check current constraint
	var isNullable string
	err = pool.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_name = 'app_generation_queue'
		AND column_name = 'template_id'
	`).Scan(&isNullable)
	if err != nil {
		log.Fatalf("Failed to check column: %v", err)
	}

	fmt.Printf("Current template_id nullable: %s\n", isNullable)

	if isNullable == "YES" {
		fmt.Println("✅ template_id is already nullable. No changes needed.")
		return
	}

	// Alter the column to allow NULL
	_, err = pool.Exec(ctx, `
		ALTER TABLE app_generation_queue
		ALTER COLUMN template_id DROP NOT NULL
	`)
	if err != nil {
		log.Fatalf("Failed to alter column: %v", err)
	}

	fmt.Println("✅ Successfully made template_id nullable!")
	fmt.Println("Pure AI generative mode is now supported.")

	// Verify
	err = pool.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_name = 'app_generation_queue'
		AND column_name = 'template_id'
	`).Scan(&isNullable)
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}
	fmt.Printf("Verified template_id nullable: %s\n", isNullable)
}
