package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
)

func main() {
	godotenv.Load(".env")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	fmt.Println("✅ Connected to database")

	// Create queries
	queries := sqlc.New(pool)

	// Create event bus
	eventBus := services.NewBuildEventBus(slog.Default())

	// Create orchestrator
	orchestrator := services.NewAppGenerationOrchestrator(pool, queries, eventBus, "")

	fmt.Println("🚀 Starting direct orchestrator test...")
	fmt.Println()

	// Test request
	req := services.MultiAgentAppRequest{
		AppName:     "DirectTest",
		Description: "A simple test app to verify database persistence",
		Features:    []string{"Test feature 1", "Test feature 2"},
		QueueItemID: "direct-test-" + time.Now().Format("150405"),
	}

	fmt.Printf("📦 Request: AppName=%s\n", req.AppName)
	fmt.Println()

	// Run generation
	startTime := time.Now()
	result, err := orchestrator.Generate(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Generation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✅ Generation completed in %s\n", duration)
	if result != nil {
		fmt.Printf("📁 App generated: %s\n", result.AppName)
	}

	// Check database
	fmt.Println()
	fmt.Println("🔍 Checking database...")
	var fileCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM osa_generated_files").Scan(&fileCount)
	fmt.Printf("📊 Total files in osa_generated_files: %d\n", fileCount)

	if fileCount > 0 {
		fmt.Println("✅ SUCCESS! Files are being persisted to database!")
	} else {
		fmt.Println("⚠️  No files in database yet - check for errors above")
	}
}
