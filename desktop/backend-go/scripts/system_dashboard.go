//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Live system dashboard showing real-time metrics
// Usage: go run scripts/system_dashboard.go [--watch]
// Use --watch to continuously refresh every 5 seconds

func main() {
	watchMode := false
	if len(os.Args) > 1 && os.Args[1] == "--watch" {
		watchMode = true
	}

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: .env file not found")
	}

	ctx := context.Background()

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL not set")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	for {
		// Clear screen (works on most terminals)
		if watchMode {
			fmt.Print("\033[H\033[2J")
		}

		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║              BUSINESSOS SYSTEM DASHBOARD                         ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Printf("   Updated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		if watchMode {
			fmt.Println("   Press Ctrl+C to exit watch mode")
		}
		fmt.Println()

		// Users
		var userCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM "user"`).Scan(&userCount)
		fmt.Printf("👥 USERS:                   %d\n", userCount)

		// Workspaces
		var workspaceCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace`).Scan(&workspaceCount)
		fmt.Printf("🏢 WORKSPACES:              %d\n", workspaceCount)

		// Onboarding profiles
		var profileCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace_onboarding_profiles`).Scan(&profileCount)
		fmt.Printf("📋 ONBOARDING PROFILES:     %d\n", profileCount)

		// AI analyses
		var analysisCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM onboarding_user_analysis`).Scan(&analysisCount)
		fmt.Printf("🤖 AI ANALYSES:             %d\n", analysisCount)

		// App templates
		var templateCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_templates`).Scan(&templateCount)
		fmt.Printf("📦 APP TEMPLATES:           %d\n", templateCount)

		// Generated apps
		var generatedAppCount int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_generated_apps`).Scan(&generatedAppCount)
		fmt.Printf("🎨 GENERATED APPS:          %d\n", generatedAppCount)

		// Generation queue
		fmt.Println()
		fmt.Println("📊 APP GENERATION QUEUE:")
		fmt.Println("─────────────────────────────────────────────────────────────")

		rows, err := pool.Query(ctx, `
			SELECT status, COUNT(*)
			FROM app_generation_queue
			GROUP BY status
			ORDER BY status
		`)
		if err == nil {
			defer rows.Close()
			queueEmpty := true
			for rows.Next() {
				var status string
				var count int
				rows.Scan(&status, &count)
				queueEmpty = false

				statusIcon := "⏳"
				if status == "completed" {
					statusIcon = "✅"
				} else if status == "failed" {
					statusIcon = "❌"
				} else if status == "processing" {
					statusIcon = "🔄"
				}

				fmt.Printf("   %s %-12s: %d\n", statusIcon, status, count)
			}

			if queueEmpty {
				fmt.Println("   (empty)")
			}
		}

		// OAuth integrations
		fmt.Println()
		fmt.Println("🔐 OAUTH INTEGRATIONS:")
		fmt.Println("─────────────────────────────────────────────────────────────")

		rows, err = pool.Query(ctx, `
			SELECT provider, status, COUNT(*)
			FROM integrations
			GROUP BY provider, status
			ORDER BY provider, status
		`)
		if err == nil {
			defer rows.Close()
			integrationsEmpty := true
			for rows.Next() {
				var provider, status string
				var count int
				rows.Scan(&provider, &status, &count)
				integrationsEmpty = false

				statusIcon := "✅"
				if status != "active" {
					statusIcon = "⚠️"
				}

				fmt.Printf("   %s %-10s (%s): %d\n", statusIcon, provider, status, count)
			}

			if integrationsEmpty {
				fmt.Println("   (none)")
			}
		}

		// Version snapshots
		fmt.Println()
		fmt.Println("📸 VERSION SNAPSHOTS:")
		fmt.Println("─────────────────────────────────────────────────────────────")

		var snapshotCount, workspacesWithSnapshots int
		pool.QueryRow(ctx, `SELECT COUNT(*), COUNT(DISTINCT workspace_id) FROM workspace_versions`).
			Scan(&snapshotCount, &workspacesWithSnapshots)

		if snapshotCount > 0 {
			fmt.Printf("   Total snapshots:    %d\n", snapshotCount)
			fmt.Printf("   Workspaces tracked: %d\n", workspacesWithSnapshots)

			// Latest snapshot
			var latestVersion string
			var latestCreated time.Time
			err := pool.QueryRow(ctx, `
				SELECT version, created_at
				FROM workspace_versions
				ORDER BY created_at DESC
				LIMIT 1
			`).Scan(&latestVersion, &latestCreated)
			if err == nil {
				fmt.Printf("   Latest version:     %s (%s ago)\n",
					latestVersion,
					formatDuration(time.Since(latestCreated)))
			}
		} else {
			fmt.Println("   (none)")
		}

		// Recent activity
		fmt.Println()
		fmt.Println("🕐 RECENT ACTIVITY (Last 24h):")
		fmt.Println("─────────────────────────────────────────────────────────────")

		var recentUsers, recentWorkspaces, recentProfiles, recentApps int
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM "user" WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&recentUsers)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&recentWorkspaces)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace_onboarding_profiles WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&recentProfiles)
		pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_generated_apps WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&recentApps)

		fmt.Printf("   New users:          %d\n", recentUsers)
		fmt.Printf("   New workspaces:     %d\n", recentWorkspaces)
		fmt.Printf("   New profiles:       %d\n", recentProfiles)
		fmt.Printf("   New apps:           %d\n", recentApps)

		fmt.Println()
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")

		if !watchMode {
			break
		}

		fmt.Println()
		fmt.Println("Refreshing in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
