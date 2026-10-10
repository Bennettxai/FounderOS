package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	// Check thinking_traces schema
	fmt.Println("Checking thinking_traces table schema...")
	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'thinking_traces'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("Failed to query schema: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nthinking_traces columns:")
	fmt.Println("Column Name          | Type           | Nullable")
	fmt.Println("---------------------|----------------|----------")
	for rows.Next() {
		var colName, dataType, nullable string
		if err := rows.Scan(&colName, &dataType, &nullable); err != nil {
			log.Fatalf("Failed to scan: %v", err)
		}
		fmt.Printf("%-20s | %-14s | %s\n", colName, dataType, nullable)
	}

	// Check reasoning_templates schema
	fmt.Println("\nChecking reasoning_templates table schema...")
	rows2, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'reasoning_templates'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("Failed to query schema: %v", err)
	}
	defer rows2.Close()

	fmt.Println("\nreasoning_templates columns:")
	fmt.Println("Column Name          | Type           | Nullable")
	fmt.Println("---------------------|----------------|----------")
	for rows2.Next() {
		var colName, dataType, nullable string
		if err := rows2.Scan(&colName, &dataType, &nullable); err != nil {
			log.Fatalf("Failed to scan: %v", err)
		}
		fmt.Printf("%-20s | %-14s | %s\n", colName, dataType, nullable)
	}

	fmt.Println("\n✅ Schema verification complete!")
}
