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

// Automatic troubleshooter that diagnoses common issues
// Usage: go run scripts/troubleshooter.go

type Issue struct {
	category string
	problem  string
	severity string // "critical", "warning", "info"
	fix      string
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               BUSINESSOS TROUBLESHOOTER                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Scanning for common issues...")
	fmt.Println()

	// Load environment
	godotenv.Load()

	ctx := context.Background()
	var issues []Issue

	// Check 1: Environment variables
	fmt.Println("🔍 Checking environment variables...")
	issues = append(issues, checkEnvironment()...)

	// Check 2: Database
	fmt.Println("🔍 Checking database...")
	issues = append(issues, checkDatabase(ctx)...)

	// Check 3: Services
	fmt.Println("🔍 Checking services...")
	issues = append(issues, checkServices()...)

	// Check 4: Dependencies
	fmt.Println("🔍 Checking dependencies...")
	issues = append(issues, checkDependencies()...)

	fmt.Println()

	// Report issues
	if len(issues) == 0 {
		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    NO ISSUES FOUND! 🎉                           ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("✅ System looks healthy!")
		fmt.Println("   All checks passed. Ready for testing.")
		return
	}

	// Categorize issues
	critical := 0
	warnings := 0
	info := 0

	for _, issue := range issues {
		switch issue.severity {
		case "critical":
			critical++
		case "warning":
			warnings++
		case "info":
			info++
		}
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ISSUES DETECTED                               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Printf("   🔴 Critical: %d\n", critical)
	fmt.Printf("   🟠 Warnings: %d\n", warnings)
	fmt.Printf("   🔵 Info:     %d\n", info)
	fmt.Println()

	// Print issues by severity
	if critical > 0 {
		fmt.Println("🔴 CRITICAL ISSUES (Must Fix):")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for i, issue := range issues {
			if issue.severity == "critical" {
				fmt.Printf("%d. [%s] %s\n", i+1, issue.category, issue.problem)
				fmt.Printf("   Fix: %s\n", issue.fix)
				fmt.Println()
			}
		}
	}

	if warnings > 0 {
		fmt.Println("🟠 WARNINGS (Should Fix):")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for i, issue := range issues {
			if issue.severity == "warning" {
				fmt.Printf("%d. [%s] %s\n", i+1, issue.category, issue.problem)
				fmt.Printf("   Fix: %s\n", issue.fix)
				fmt.Println()
			}
		}
	}

	if info > 0 {
		fmt.Println("🔵 INFORMATIONAL:")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for i, issue := range issues {
			if issue.severity == "info" {
				fmt.Printf("%d. [%s] %s\n", i+1, issue.category, issue.problem)
				if issue.fix != "" {
					fmt.Printf("   Tip: %s\n", issue.fix)
				}
				fmt.Println()
			}
		}
	}

	// Exit code
	if critical > 0 {
		fmt.Println("❌ System has critical issues. Fix them before testing.")
		os.Exit(1)
	} else if warnings > 0 {
		fmt.Println("⚠️  System has warnings. Testing may work but could have issues.")
	}
}

func checkEnvironment() []Issue {
	var issues []Issue

	// Critical env vars
	criticalVars := []struct {
		name string
		desc string
	}{
		{"DATABASE_URL", "Database connection string"},
		{"SECRET_KEY", "JWT signing key"},
		{"GROQ_API_KEY", "AI analysis"},
		{"GOOGLE_CLIENT_ID", "Google OAuth"},
		{"GOOGLE_CLIENT_SECRET", "Google OAuth"},
	}

	for _, v := range criticalVars {
		if os.Getenv(v.name) == "" {
			issues = append(issues, Issue{
				category: "Environment",
				problem:  fmt.Sprintf("%s is not set (%s)", v.name, v.desc),
				severity: "critical",
				fix:      fmt.Sprintf("Set %s in desktop/backend-go/.env", v.name),
			})
		}
	}

	// Optional but recommended
	if os.Getenv("OSA_BASE_URL") == "" {
		issues = append(issues, Issue{
			category: "Environment",
			problem:  "OSA_BASE_URL not set (app generation will fail)",
			severity: "warning",
			fix:      "Set OSA_BASE_URL if you want to test app generation",
		})
	}

	return issues
}

func checkDatabase(ctx context.Context) []Issue {
	var issues []Issue

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return issues // Already caught by checkEnvironment
	}

	// Try to connect
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctxTimeout, dbURL)
	if err != nil {
		issues = append(issues, Issue{
			category: "Database",
			problem:  fmt.Sprintf("Cannot connect: %v", err),
			severity: "critical",
			fix:      "Check DATABASE_URL and ensure database is accessible",
		})
		return issues
	}
	defer pool.Close()

	// Check if we can query
	var result int
	err = pool.QueryRow(ctxTimeout, "SELECT 1").Scan(&result)
	if err != nil {
		issues = append(issues, Issue{
			category: "Database",
			problem:  fmt.Sprintf("Connection OK but query failed: %v", err),
			severity: "critical",
			fix:      "Check database status and permissions",
		})
		return issues
	}

	// Check critical tables
	requiredTables := []string{
		"app_templates",
		"user_generated_apps",
		"app_generation_queue",
		"workspace_onboarding_profiles",
		"onboarding_user_analysis",
	}

	for _, table := range requiredTables {
		var exists bool
		err := pool.QueryRow(ctxTimeout, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil || !exists {
			issues = append(issues, Issue{
				category: "Database",
				problem:  fmt.Sprintf("Table missing: %s", table),
				severity: "critical",
				fix:      "Run: go run scripts/apply_missing_migrations.go",
			})
		}
	}

	// Check if app_templates is seeded
	var templateCount int
	pool.QueryRow(ctxTimeout, "SELECT COUNT(*) FROM app_templates").Scan(&templateCount)
	if templateCount < 10 {
		issues = append(issues, Issue{
			category: "Database",
			problem:  fmt.Sprintf("Only %d app templates (expected 10)", templateCount),
			severity: "warning",
			fix:      "Run migration 081 to seed templates",
		})
	}

	return issues
}

func checkServices() []Issue {
	var issues []Issue
	client := http.Client{Timeout: 2 * time.Second}

	// Backend
	resp, err := client.Get("http://localhost:8001/api/onboarding/status")
	if err != nil {
		issues = append(issues, Issue{
			category: "Services",
			problem:  "Backend server not responding on port 8001",
			severity: "critical",
			fix:      "Start backend: cd desktop/backend-go && go run cmd/server/main.go",
		})
	} else {
		resp.Body.Close()
	}

	// Frontend
	resp, err = client.Get("http://localhost:5173")
	if err != nil {
		issues = append(issues, Issue{
			category: "Services",
			problem:  "Frontend server not responding on port 5173",
			severity: "warning",
			fix:      "Start frontend: cd frontend && npm run dev",
		})
	} else {
		resp.Body.Close()
	}

	return issues
}

func checkDependencies() []Issue {
	var issues []Issue

	// Check if .env exists
	if _, err := os.Stat("desktop/backend-go/.env"); os.IsNotExist(err) {
		issues = append(issues, Issue{
			category: "Dependencies",
			problem:  ".env file not found",
			severity: "warning",
			fix:      "Copy .env.example to .env and fill in values",
		})
	}

	// Check Go version (simplified - just check if go is available)
	// In production, you'd want to parse `go version` output

	return issues
}
