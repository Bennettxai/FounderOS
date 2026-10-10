//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	envBytes, err := os.ReadFile(".env")
	if err != nil {
		log.Fatal("Failed to read .env:", err)
	}

	var dbURL string
	for _, line := range strings.Split(string(envBytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "DATABASE_URL=") && !strings.HasPrefix(line, "#") {
			dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
			break
		}
	}
	if dbURL == "" {
		log.Fatal("DATABASE_URL not found in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close(ctx)

	migration, err := os.ReadFile("internal/database/migrations/101_mcp_servers.sql")
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	_, err = conn.Exec(ctx, string(migration))
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Verify
	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'mcp_servers')").Scan(&exists)
	if err != nil {
		log.Fatal("Verification query failed:", err)
	}

	if exists {
		fmt.Println("✓ Migration 101_mcp_servers.sql applied successfully")
		fmt.Println("✓ Table 'mcp_servers' exists")
	} else {
		fmt.Println("✗ Table 'mcp_servers' was NOT created")
	}
}
