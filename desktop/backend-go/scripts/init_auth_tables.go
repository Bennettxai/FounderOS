package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try reading from .env file
		data, err := os.ReadFile(".env")
		if err != nil {
			log.Fatal("DATABASE_URL not set and no .env file found")
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "DATABASE_URL=") {
				dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
				break
			}
		}
	}
	if dbURL == "" {
		log.Fatal("DATABASE_URL not found")
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse DB URL: %v", err)
	}

	if strings.Contains(dbURL, ":6543") || strings.Contains(dbURL, "pgbouncer=true") {
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}
	poolConfig.MaxConns = 3
	poolConfig.MinConns = 1

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping: %v", err)
	}
	fmt.Println("Connected to database!")

	// Create Better Auth tables
	queries := []struct {
		name string
		sql  string
	}{
		{
			name: "user table",
			sql: `CREATE TABLE IF NOT EXISTS "user" (
				id VARCHAR(255) PRIMARY KEY,
				name VARCHAR(255),
				email VARCHAR(255) UNIQUE NOT NULL,
				"emailVerified" BOOLEAN DEFAULT FALSE,
				image TEXT,
				"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)`,
		},
		{
			name: "session table",
			sql: `CREATE TABLE IF NOT EXISTS session (
				id VARCHAR(255) PRIMARY KEY,
				"userId" VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
				token VARCHAR(255) UNIQUE NOT NULL,
				"expiresAt" TIMESTAMP WITH TIME ZONE NOT NULL,
				"ipAddress" VARCHAR(45),
				"userAgent" TEXT,
				"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)`,
		},
		{
			name: "account table",
			sql: `CREATE TABLE IF NOT EXISTS account (
				id VARCHAR(255) PRIMARY KEY,
				"userId" VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
				"accountId" VARCHAR(255) NOT NULL,
				"providerId" VARCHAR(255) NOT NULL,
				"accessToken" TEXT,
				"refreshToken" TEXT,
				"accessTokenExpiresAt" TIMESTAMP WITH TIME ZONE,
				"refreshTokenExpiresAt" TIMESTAMP WITH TIME ZONE,
				scope TEXT,
				password TEXT,
				"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)`,
		},
		{
			name: "verification table",
			sql: `CREATE TABLE IF NOT EXISTS verification (
				id VARCHAR(255) PRIMARY KEY,
				identifier VARCHAR(255) NOT NULL,
				value VARCHAR(255) NOT NULL,
				"expiresAt" TIMESTAMP WITH TIME ZONE NOT NULL,
				"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)`,
		},
		{
			name: "session indexes",
			sql:  `CREATE INDEX IF NOT EXISTS idx_session_user ON session("userId")`,
		},
		{
			name: "session token index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_session_token ON session(token)`,
		},
		{
			name: "account user index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_account_user ON account("userId")`,
		},
		{
			name: "account provider index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_account_provider ON account("providerId", "accountId")`,
		},
	}

	for _, q := range queries {
		_, err := pool.Exec(ctx, q.sql)
		if err != nil {
			fmt.Printf("WARN: %s - %v\n", q.name, err)
		} else {
			fmt.Printf("OK: %s\n", q.name)
		}
	}

	// Verify tables exist
	var count int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_name IN ('user', 'session', 'account', 'verification')`).Scan(&count)
	if err != nil {
		log.Fatalf("Failed to verify tables: %v", err)
	}
	fmt.Printf("\nAuth tables found: %d/4\n", count)

	// Check user table columns
	rows, err := pool.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_name = 'user' ORDER BY ordinal_position`)
	if err != nil {
		log.Fatalf("Failed to query columns: %v", err)
	}
	defer rows.Close()

	fmt.Print("user columns: ")
	for rows.Next() {
		var col string
		rows.Scan(&col)
		fmt.Printf("%s, ", col)
	}
	fmt.Println()

	fmt.Println("\nDone! Better Auth tables are ready.")
}
