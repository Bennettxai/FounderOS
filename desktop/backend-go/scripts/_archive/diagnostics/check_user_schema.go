//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Quick script to check user table schema
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

	fmt.Println("📋 USER TABLE SCHEMA")
	fmt.Println("═══════════════════════════════════════════════════════════")

	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = 'user'
		ORDER BY ordinal_position
	`)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Printf("%-20s %-20s %-12s %s\n", "COLUMN", "TYPE", "NULLABLE", "DEFAULT")
	fmt.Println("───────────────────────────────────────────────────────────")

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

		fmt.Printf("%-20s %-20s %-12s %s\n", colName, dataType, nullable, defaultStr)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
}
