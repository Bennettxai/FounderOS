//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	// Add created_at column
	_, err = conn.Exec(ctx, `
		ALTER TABLE notification_batches 
		ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW()
	`)
	if err != nil {
		log.Printf("Error adding created_at: %v", err)
	} else {
		fmt.Println("✓ created_at column added")
	}

	// Add updated_at column
	_, err = conn.Exec(ctx, `
		ALTER TABLE notification_batches 
		ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW()
	`)
	if err != nil {
		log.Printf("Error adding updated_at: %v", err)
	} else {
		fmt.Println("✓ updated_at column added")
	}

	fmt.Println("Migration complete!")
}
