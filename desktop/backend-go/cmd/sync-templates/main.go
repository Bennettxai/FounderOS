package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database"
	"github.com/rhl/businessos-backend/internal/services"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Info("Note: No .env file found (this is fine in production)")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
	}

	// Connect to database
	pool, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
	}
	defer pool.Close()

	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Determine templates directory
	templatesDir := getTemplatesDir()
	if templatesDir == "" {
		slog.Error("Failed to locate templates directory")
	}

	logger.Info("starting template sync",
		"templates_dir", templatesDir,
		"database", cfg.DatabaseURL != "",
	)

	// Create sync service
	syncService := services.NewTemplateSyncService(pool, logger, templatesDir)

	// Run sync
	result, err := syncService.SyncTemplates(ctx)
	if err != nil {
		slog.Error("Template sync failed", "error", err)
	}

	// Print results
	printResults(result)

	// Exit with error if there were any sync errors
	if len(result.Errors) > 0 {
		os.Exit(1)
	}

	logger.Info("template sync completed successfully")
}

// getTemplatesDir finds the templates directory
func getTemplatesDir() string {
	// Check common locations
	possiblePaths := []string{
		"internal/prompts/templates/osa",                       // From project root
		"../internal/prompts/templates/osa",                    // From cmd directory
		"../../internal/prompts/templates/osa",                 // From cmd/sync-templates
		"desktop/backend-go/internal/prompts/templates/osa",    // From monorepo root
		"../desktop/backend-go/internal/prompts/templates/osa", // From sibling directory
		os.Getenv("TEMPLATES_DIR"),                             // Environment variable
	}

	for _, path := range possiblePaths {
		if path == "" {
			continue
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath
		}
	}

	// Try to find based on executable location
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// Try relative to executable
		templatePath := filepath.Join(execDir, "../../internal/prompts/templates/osa")
		if info, err := os.Stat(templatePath); err == nil && info.IsDir() {
			absPath, _ := filepath.Abs(templatePath)
			return absPath
		}
	}

	return ""
}

// printResults prints the sync results in a readable format
func printResults(result *services.SyncResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEMPLATE SYNC RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Inserted: %d\n", result.Inserted)
	fmt.Printf("Updated:  %d\n", result.Updated)
	fmt.Printf("Skipped:  %d\n", result.Skipped)
	fmt.Printf("Errors:   %d\n", len(result.Errors))

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}
