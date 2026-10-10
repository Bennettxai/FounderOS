//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	fmt.Println("📋 DATABASE TABLES")
	fmt.Println("═══════════════════════════════════════════════════════════")

	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		count++
		fmt.Printf("%3d. %s\n", count, tableName)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Total: %d tables\n", count)
}
