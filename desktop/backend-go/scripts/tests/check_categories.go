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
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Check constraint on category
	var constraint string
	err = pool.QueryRow(ctx, `
		SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conname = 'app_templates_category_check'
	`).Scan(&constraint)

	if err != nil {
		log.Printf("Error getting constraint: %v", err)
	} else {
		fmt.Printf("Category constraint: %s\n", constraint)
	}

	// Check existing categories
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT category FROM app_templates ORDER BY category
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nExisting categories in app_templates:")
	for rows.Next() {
		var cat string
		rows.Scan(&cat)
		fmt.Printf("  - %s\n", cat)
	}
}
