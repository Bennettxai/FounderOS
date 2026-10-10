//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Quick diagnostic script to verify system health before E2E testing
// Usage: go run scripts/quick_diagnostic.go

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         BUSINESSOS - QUICK SYSTEM DIAGNOSTIC                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: .env file not found, using environment variables")
	}

	ctx := context.Background()

	// Check 1: Database Connection
	fmt.Println("🔍 CHECK 1: Database Connection")
	fmt.Println("─────────────────────────────────────────────────────────────")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL not set in environment")
		os.Exit(1)
	}
	fmt.Printf("   Database URL: %s\n", maskPassword(dbURL))

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Test connection
	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Database connection OK")
	fmt.Println()

	// Check 2: Critical Tables
	fmt.Println("🔍 CHECK 2: Critical Tables")
	fmt.Println("─────────────────────────────────────────────────────────────")

	criticalTables := []struct {
		name        string
		minRows     int
		description string
	}{
		{"app_templates", 10, "App generation templates"},
		{"workspace_onboarding_profiles", 0, "Onboarding profiles"},
		{"onboarding_user_analysis", 0, "AI analysis results"},
		{"app_generation_queue", 0, "Generation queue"},
		{"user_generated_apps", 0, "Generated apps"},
		{"workspace_versions", 0, "Version snapshots"},
	}

	allTablesOK := true
	for _, table := range criticalTables {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = $1
			)
		`, table.name).Scan(&exists)

		if err != nil {
			fmt.Printf("❌ Error checking %s: %v\n", table.name, err)
			allTablesOK = false
			continue
		}

		if !exists {
			fmt.Printf("❌ Table missing: %s\n", table.name)
			allTablesOK = false
			continue
		}

		// Count rows
		var rowCount int
		err = pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table.name)).Scan(&rowCount)
		if err != nil {
			fmt.Printf("⚠️  %s exists but query failed: %v\n", table.name, err)
			continue
		}

		if rowCount < table.minRows {
			fmt.Printf("⚠️  %s: %d rows (expected ≥%d) - %s\n",
				table.name, rowCount, table.minRows, table.description)
			if table.minRows > 0 {
				allTablesOK = false
			}
		} else {
			fmt.Printf("✅ %s: %d rows - %s\n", table.name, rowCount, table.description)
		}
	}

	if !allTablesOK {
		fmt.Println("\n⚠️  Some tables missing or have insufficient data")
		fmt.Println("   Run: go run scripts/apply_missing_migrations.go")
	}
	fmt.Println()

	// Check 3: Integration Points
	fmt.Println("🔍 CHECK 3: Integration Point Files")
	fmt.Println("─────────────────────────────────────────────────────────────")

	integrationFiles := []struct {
		path        string
		function    string
		description string
	}{
		{"internal/services/onboarding_service.go", "transformAIAnalysisToWorkspaceProfile", "AI → Profile"},
		{"internal/services/app_generation_worker.go", "enrichGenerationContext", "Profile → OSA"},
		{"internal/services/workspace_version_service.go", "CreateSnapshot", "Version snapshots"},
		{"internal/services/post_onboarding_service.go", "QueueAppsForWorkspace", "Template matching"},
	}

	allFilesOK := true
	for _, file := range integrationFiles {
		if _, err := os.Stat(file.path); os.IsNotExist(err) {
			fmt.Printf("❌ Missing: %s\n", file.path)
			allFilesOK = false
		} else {
			fmt.Printf("✅ %s - %s (%s)\n", file.description, file.function, file.path)
		}
	}

	if !allFilesOK {
		fmt.Println("\n❌ Some integration files are missing!")
		os.Exit(1)
	}
	fmt.Println()

	// Check 4: Environment Variables
	fmt.Println("🔍 CHECK 4: Required Environment Variables")
	fmt.Println("─────────────────────────────────────────────────────────────")

	envVars := []struct {
		name        string
		required    bool
		description string
	}{
		{"DATABASE_URL", true, "PostgreSQL connection string"},
		{"SECRET_KEY", true, "JWT signing key"},
		{"GROQ_API_KEY", true, "Groq AI for analysis"},
		{"GOOGLE_CLIENT_ID", true, "Google OAuth"},
		{"GOOGLE_CLIENT_SECRET", true, "Google OAuth"},
		{"OSA_BASE_URL", false, "OSA service URL"},
		{"OSA_API_KEY", false, "OSA authentication"},
		{"REDIS_URL", false, "Redis for sessions (optional)"},
	}

	allEnvOK := true
	for _, env := range envVars {
		value := os.Getenv(env.name)
		if value == "" {
			if env.required {
				fmt.Printf("❌ MISSING (required): %s - %s\n", env.name, env.description)
				allEnvOK = false
			} else {
				fmt.Printf("⚠️  NOT SET (optional): %s - %s\n", env.name, env.description)
			}
		} else {
			if env.name == "SECRET_KEY" || env.name == "GROQ_API_KEY" ||
			   env.name == "GOOGLE_CLIENT_SECRET" || env.name == "OSA_API_KEY" {
				fmt.Printf("✅ %s: ****** (hidden) - %s\n", env.name, env.description)
			} else {
				fmt.Printf("✅ %s: %s - %s\n", env.name, truncate(value, 40), env.description)
			}
		}
	}

	if !allEnvOK {
		fmt.Println("\n❌ Missing required environment variables!")
		fmt.Println("   Check desktop/backend-go/.env file")
		os.Exit(1)
	}
	fmt.Println()

	// Check 5: Sample App Templates
	fmt.Println("🔍 CHECK 5: Sample App Templates")
	fmt.Println("─────────────────────────────────────────────────────────────")

	rows, err := pool.Query(ctx, `
		SELECT template_name, display_name, category, priority_score
		FROM app_templates
		ORDER BY priority_score DESC
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var templateName, displayName, category string
			var priorityScore int
			if err := rows.Scan(&templateName, &displayName, &category, &priorityScore); err != nil {
				continue
			}
			fmt.Printf("   • %s (%s) - %s [score: %d]\n",
				displayName, templateName, category, priorityScore)
			count++
		}

		if count == 0 {
			fmt.Println("❌ No app templates found!")
			fmt.Println("   Run: cd desktop/backend-go && psql $DATABASE_URL < migrations/081_app_templates_system.sql")
		} else {
			fmt.Printf("✅ %d app templates available\n", count)
		}
	}
	fmt.Println()

	// Check 6: Services Health
	fmt.Println("🔍 CHECK 6: Local Services")
	fmt.Println("─────────────────────────────────────────────────────────────")

	// We can't easily check if services are running from this script without importing net/http
	// So we just provide instructions
	fmt.Println("   To verify services are running, run these commands:")
	fmt.Println("   • Backend:  curl http://localhost:8001/api/onboarding/status")
	fmt.Println("   • Frontend: curl -s http://localhost:5173 | grep sveltekit")
	fmt.Println()

	// Final Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      DIAGNOSTIC SUMMARY                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

	if allTablesOK && allFilesOK && allEnvOK {
		fmt.Println("✅ ALL CHECKS PASSED")
		fmt.Println("   System is ready for E2E testing!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Ensure backend is running: cd desktop/backend-go && go run cmd/server/main.go")
		fmt.Println("2. Ensure frontend is running: cd frontend && npm run dev")
		fmt.Println("3. Open browser: http://localhost:5173/onboarding")
		fmt.Println("4. Follow: DAY3_MANUAL_TESTING_CHECKLIST.md")
	} else {
		fmt.Println("❌ SOME CHECKS FAILED")
		fmt.Println("   Review errors above before starting E2E testing")
		os.Exit(1)
	}
}

func maskPassword(url string) string {
	// Simple password masking for display
	// postgres://user:password@host:port/db → postgres://user:****@host:port/db
	start := 0
	for i, c := range url {
		if c == ':' && i > 10 {
			start = i + 1
			break
		}
	}

	if start == 0 {
		return url
	}

	end := start
	for i := start; i < len(url); i++ {
		if url[i] == '@' {
			end = i
			break
		}
	}

	if end == start {
		return url
	}

	return url[:start] + "****" + url[end:]
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
