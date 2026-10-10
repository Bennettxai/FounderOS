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

	// Get workspace schema
	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = 'workspaces'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Workspaces table schema:")
	fmt.Println("Column Name          | Type         | Nullable | Default")
	fmt.Println("---------------------|--------------|----------|--------")
	for rows.Next() {
		var name, dataType, nullable string
		var defaultVal *string
		rows.Scan(&name, &dataType, &nullable, &defaultVal)
		def := "NULL"
		if defaultVal != nil {
			def = *defaultVal
		}
		fmt.Printf("%-20s | %-12s | %-8s | %s\n", name, dataType, nullable, def)
	}

	// Check existing workspaces
	fmt.Println("\nExisting workspaces:")
	rows2, err := pool.Query(ctx, "SELECT id, name, slug FROM workspaces LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		var id, name, slug string
		rows2.Scan(&id, &name, &slug)
		fmt.Printf("  %s | %s | %s\n", id, name, slug)
		count++
	}
	if count == 0 {
		fmt.Println("  (No workspaces found)")
	}
}
