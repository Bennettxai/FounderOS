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

	rows, err := pool.Query(ctx, `
		SELECT id, template_name, display_name, category
		FROM app_templates
		ORDER BY priority_score DESC
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("App Templates in Database:")
	fmt.Println("==========================")
	count := 0
	for rows.Next() {
		var id, name, displayName, category string
		if err := rows.Scan(&id, &name, &displayName, &category); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		count++
		fmt.Printf("%d. %s\n", count, displayName)
		fmt.Printf("   ID: %s\n", id)
		fmt.Printf("   Name: %s\n", name)
		fmt.Printf("   Category: %s\n\n", category)
	}

	if count == 0 {
		fmt.Println("No templates found. Run migration 088 to seed templates.")
	}
}
