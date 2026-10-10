package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer pool.Close()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         BusinessOS - Database Migration Verification          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	// 1. Check applied migrations
	fmt.Println("\n📋 APPLIED MIGRATIONS:")
	fmt.Println("   " + strings.Repeat("─", 60))
	rows, err := pool.Query(context.Background(), "SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		log.Fatal("Failed to query migrations:", err)
	}
	defer rows.Close()

	migrationCount := 0
	for rows.Next() {
		var version string
		var appliedAt interface{}
		if err := rows.Scan(&version, &appliedAt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("   ✓ Migration %s - Applied: %v\n", version, appliedAt)
		migrationCount++
	}
	fmt.Printf("   " + strings.Repeat("─", 60) + "\n")
	fmt.Printf("   Total: %d migrations\n", migrationCount)

	// 2. Verify all expected tables
	fmt.Println("\n🗄️  TABLE VERIFICATION:")
	fmt.Println("   " + strings.Repeat("─", 60))

	tableGroups := map[string][]string{
		"Image Embeddings (Migration 025)": {
			"image_embeddings",
			"image_tags",
			"image_collections",
			"image_collection_items",
		},
		"Workspaces & Roles (Migration 026)": {
			"workspaces",
			"workspace_roles",
			"workspace_members",
			"user_workspace_profiles",
			"workspace_memories",
			"role_permissions",
		},
		"Workspace Invites (Migration 027)": {
			"workspace_invites",
		},
		"Audit Logs (Migration 028)": {
			"workspace_audit_logs",
		},
		"Project Members (Migration 029)": {
			"project_members",
			"project_role_definitions",
		},
	}

	allTablesExist := true
	for groupName, tables := range tableGroups {
		fmt.Printf("\n   %s:\n", groupName)
		for _, table := range tables {
			var exists bool
			query := `SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)`
			err := pool.QueryRow(context.Background(), query, table).Scan(&exists)
			if err != nil {
				log.Printf("      ❌ Error checking %s: %v", table, err)
				allTablesExist = false
				continue
			}
			if exists {
				// Get row count
				var count int64
				_ = pool.QueryRow(context.Background(), fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
				fmt.Printf("      ✓ %-30s (%d rows)\n", table, count)
			} else {
				fmt.Printf("      ✗ %-30s MISSING\n", table)
				allTablesExist = false
			}
		}
	}

	// 3. Verify key functions
	fmt.Println("\n🔧 FUNCTION VERIFICATION:")
	fmt.Println("   " + strings.Repeat("─", 60))

	functions := []string{
		"seed_default_workspace_roles",
		"can_access_memory",
		"get_workspace_memories",
		"get_user_memories",
		"get_accessible_memories",
		"share_memory",
		"unshare_memory",
		"track_memory_access",
		"has_project_access",
		"get_project_role",
		"get_project_permissions",
		"log_workspace_action",
	}

	allFunctionsExist := true
	for _, funcName := range functions {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM pg_proc
			WHERE proname = $1
		)`
		err := pool.QueryRow(context.Background(), query, funcName).Scan(&exists)
		if err != nil {
			log.Printf("   ❌ Error checking function %s: %v", funcName, err)
			allFunctionsExist = false
			continue
		}
		if exists {
			fmt.Printf("   ✓ %s\n", funcName)
		} else {
			fmt.Printf("   ✗ %s MISSING\n", funcName)
			allFunctionsExist = false
		}
	}

	// 4. Verify vector extension
	fmt.Println("\n🧩 EXTENSION VERIFICATION:")
	fmt.Println("   " + strings.Repeat("─", 60))

	var vectorExists bool
	err = pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&vectorExists)
	if err != nil {
		log.Printf("   ❌ Error checking vector extension: %v", err)
	} else if vectorExists {
		fmt.Println("   ✓ pgvector extension installed")
	} else {
		fmt.Println("   ✗ pgvector extension NOT installed")
	}

	// 5. Check for vector indexes
	fmt.Println("\n📊 VECTOR INDEX VERIFICATION:")
	fmt.Println("   " + strings.Repeat("─", 60))

	vectorIndexes := []struct {
		table string
		index string
	}{
		{"image_embeddings", "idx_image_embeddings_embedding"},
		{"workspace_memories", "idx_workspace_memories_embedding"},
	}

	for _, vi := range vectorIndexes {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE tablename = $1
			AND indexname = $2
		)`
		err := pool.QueryRow(context.Background(), query, vi.table, vi.index).Scan(&exists)
		if err != nil {
			log.Printf("   ❌ Error checking index %s: %v", vi.index, err)
			continue
		}
		if exists {
			fmt.Printf("   ✓ %s on %s\n", vi.index, vi.table)
		} else {
			fmt.Printf("   ✗ %s on %s MISSING\n", vi.index, vi.table)
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("═", 64))
	fmt.Println("📊 VERIFICATION SUMMARY:")
	fmt.Println(strings.Repeat("═", 64))

	if migrationCount >= 6 && allTablesExist && allFunctionsExist && vectorExists {
		fmt.Println("✅ ALL VERIFICATIONS PASSED!")
		fmt.Println("   • All migrations applied successfully")
		fmt.Println("   • All required tables exist")
		fmt.Println("   • All required functions exist")
		fmt.Println("   • Vector extension installed")
		fmt.Println("\n🎉 Database is ready for use!")
	} else {
		fmt.Println("⚠️  SOME VERIFICATIONS FAILED:")
		if migrationCount < 6 {
			fmt.Printf("   • Missing migrations: %d/6 applied\n", migrationCount)
		}
		if !allTablesExist {
			fmt.Println("   • Some tables are missing")
		}
		if !allFunctionsExist {
			fmt.Println("   • Some functions are missing")
		}
		if !vectorExists {
			fmt.Println("   • Vector extension not installed")
		}
		fmt.Println("\n⚠️  Database may not be fully functional!")
	}

	fmt.Println(strings.Repeat("═", 64))
}
