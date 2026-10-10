//go:build ignore

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Comprehensive environment validation before E2E testing
// Usage: go run scripts/validate_environment.go

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           COMPREHENSIVE ENVIRONMENT VALIDATION                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	allChecks := []check{
		{"Load .env file", checkDotEnv},
		{"Database connection", checkDatabase},
		{"Database tables", checkTables},
		{"App templates seeded", checkAppTemplates},
		{"Groq API key", checkGroqAPIKey},
		{"Google OAuth credentials", checkGoogleOAuth},
		{"Backend server", checkBackendServer},
		{"Frontend server", checkFrontendServer},
		{"Required ports available", checkPorts},
	}

	passed := 0
	warnings := 0
	failed := 0

	for i, c := range allChecks {
		fmt.Printf("[%d/%d] %s...\n", i+1, len(allChecks), c.name)
		result := c.fn()

		if result.status == "pass" {
			fmt.Printf("      ✅ %s\n", result.message)
			passed++
		} else if result.status == "warn" {
			fmt.Printf("      ⚠️  %s\n", result.message)
			warnings++
		} else {
			fmt.Printf("      ❌ %s\n", result.message)
			if result.remedy != "" {
				fmt.Printf("         Fix: %s\n", result.remedy)
			}
			failed++
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      VALIDATION SUMMARY                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Printf("   ✅ Passed:   %d\n", passed)
	fmt.Printf("   ⚠️  Warnings: %d\n", warnings)
	fmt.Printf("   ❌ Failed:   %d\n", failed)
	fmt.Println()

	if failed > 0 {
		fmt.Println("❌ Environment validation FAILED")
		fmt.Println("   Fix the failed checks above before starting E2E tests")
		os.Exit(1)
	} else if warnings > 0 {
		fmt.Println("⚠️  Environment validation passed with warnings")
		fmt.Println("   You can proceed, but some features may not work")
	} else {
		fmt.Println("🎉 Environment validation PASSED")
		fmt.Println("   System is fully ready for E2E testing!")
	}
}

type check struct {
	name string
	fn   func() checkResult
}

type checkResult struct {
	status  string // "pass", "warn", "fail"
	message string
	remedy  string
}

func checkDotEnv() checkResult {
	if err := godotenv.Load(); err != nil {
		return checkResult{"warn", ".env file not found, using system environment", "Create desktop/backend-go/.env from .env.example"}
	}
	return checkResult{"pass", ".env file loaded", ""}
}

func checkDatabase() checkResult {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return checkResult{"fail", "DATABASE_URL not set", "Set DATABASE_URL in .env"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return checkResult{"fail", fmt.Sprintf("Connection failed: %v", err), "Check DATABASE_URL and database status"}
	}
	defer pool.Close()

	var result int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
		return checkResult{"fail", fmt.Sprintf("Query failed: %v", err), "Check database is accessible"}
	}

	return checkResult{"pass", "Connected to database", ""}
}

func checkTables() checkResult {
	dbURL := os.Getenv("DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, _ := pgxpool.New(ctx, dbURL)
	if pool == nil {
		return checkResult{"fail", "Cannot connect to database", ""}
	}
	defer pool.Close()

	requiredTables := []string{
		"app_templates",
		"workspace_onboarding_profiles",
		"onboarding_user_analysis",
		"app_generation_queue",
		"user_generated_apps",
		"workspace_versions",
	}

	for _, table := range requiredTables {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil || !exists {
			return checkResult{"fail", fmt.Sprintf("Table missing: %s", table), "Run migrations: go run scripts/apply_missing_migrations.go"}
		}
	}

	return checkResult{"pass", fmt.Sprintf("All %d required tables exist", len(requiredTables)), ""}
}

func checkAppTemplates() checkResult {
	dbURL := os.Getenv("DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, _ := pgxpool.New(ctx, dbURL)
	if pool == nil {
		return checkResult{"fail", "Cannot connect to database", ""}
	}
	defer pool.Close()

	var count int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_templates").Scan(&count)

	if count < 10 {
		return checkResult{"fail", fmt.Sprintf("Only %d app templates (expected 10)", count), "Run migration 081 to seed templates"}
	}

	return checkResult{"pass", fmt.Sprintf("%d app templates seeded", count), ""}
}

func checkGroqAPIKey() checkResult {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return checkResult{"fail", "GROQ_API_KEY not set", "Set GROQ_API_KEY in .env (get from console.groq.com)"}
	}

	if len(key) < 20 {
		return checkResult{"warn", "GROQ_API_KEY looks too short", "Verify key is correct"}
	}

	return checkResult{"pass", fmt.Sprintf("GROQ_API_KEY set (%d chars)", len(key)), ""}
}

func checkGoogleOAuth() checkResult {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" {
		return checkResult{"fail", "GOOGLE_CLIENT_ID not set", "Set GOOGLE_CLIENT_ID in .env (from Google Cloud Console)"}
	}

	if clientSecret == "" {
		return checkResult{"fail", "GOOGLE_CLIENT_SECRET not set", "Set GOOGLE_CLIENT_SECRET in .env"}
	}

	return checkResult{"pass", "Google OAuth credentials set", ""}
}

func checkBackendServer() checkResult {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8001/api/onboarding/status")
	if err != nil {
		return checkResult{"warn", "Backend server not responding", "Start backend: cd desktop/backend-go && go run cmd/server/main.go"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return checkResult{"pass", "Backend server running (auth required)", ""}
	}

	return checkResult{"pass", fmt.Sprintf("Backend server running (status %d)", resp.StatusCode), ""}
}

func checkFrontendServer() checkResult {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:5173")
	if err != nil {
		return checkResult{"warn", "Frontend server not responding", "Start frontend: cd frontend && npm run dev"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return checkResult{"pass", "Frontend server running", ""}
	}

	return checkResult{"warn", fmt.Sprintf("Frontend returned status %d", resp.StatusCode), "Check frontend build"}
}

func checkPorts() checkResult {
	// This is a simplified check - just verifies backend/frontend are accessible
	// More comprehensive port checking would require OS-specific commands

	backendOK := false
	frontendOK := false

	client := http.Client{Timeout: 1 * time.Second}

	if resp, err := client.Get("http://localhost:8001/api/onboarding/status"); err == nil {
		resp.Body.Close()
		backendOK = true
	}

	if resp, err := client.Get("http://localhost:5173"); err == nil {
		resp.Body.Close()
		frontendOK = true
	}

	if !backendOK && !frontendOK {
		return checkResult{"warn", "Ports 8001 and 5173 not in use", "Start backend and frontend servers"}
	}

	if !backendOK {
		return checkResult{"warn", "Port 8001 not in use", "Start backend server"}
	}

	if !frontendOK {
		return checkResult{"warn", "Port 5173 not in use", "Start frontend server"}
	}

	return checkResult{"pass", "Ports 8001 and 5173 responding", ""}
}
