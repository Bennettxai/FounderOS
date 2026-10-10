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
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run check_table_schema.go <table_name>")
		os.Exit(1)
	}

	tableName := os.Args[1]

	godotenv.Load()
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	fmt.Printf("📋 TABLE SCHEMA: %s\n", tableName)
	fmt.Println("═══════════════════════════════════════════════════════════")

	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Printf("%-30s %-25s %-10s %s\n", "COLUMN", "TYPE", "NULLABLE", "DEFAULT")
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")

	for rows.Next() {
		var colName, dataType, nullable string
		var defaultVal *string
		rows.Scan(&colName, &dataType, &nullable, &defaultVal)

		defaultStr := "NULL"
		if defaultVal != nil {
			defaultStr = *defaultVal
			if len(defaultStr) > 30 {
				defaultStr = defaultStr[:27] + "..."
			}
		}

		fmt.Printf("%-30s %-25s %-10s %s\n", colName, dataType, nullable, defaultStr)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}
