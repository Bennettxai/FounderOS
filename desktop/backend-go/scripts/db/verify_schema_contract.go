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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rhl/businessos-backend/internal/database/schemahealth"
)

func main() {
	_ = godotenv.Load()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("parse DATABASE_URL: %v", err)
	}
	if strings.Contains(databaseURL, ":6543") || strings.Contains(databaseURL, "pgbouncer=true") {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	report, err := schemahealth.Check(ctx, pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "schema contract failed")
		for _, req := range report.Missing {
			fmt.Fprintln(os.Stderr, "missing:", req.String())
		}
		os.Exit(1)
	}

	fmt.Printf("schema contract passed: %d tables, %d column groups, %d functions\n",
		len(schemahealth.RequiredTables),
		len(schemahealth.RequiredColumns),
		len(schemahealth.RequiredFunctions),
	)
}
