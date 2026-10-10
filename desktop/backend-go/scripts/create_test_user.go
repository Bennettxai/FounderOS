//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Create a test user for E2E testing
// Usage: go run scripts/create_test_user.go [email] [name]
// Example: go run scripts/create_test_user.go test@demo.com "Test User"
// Note: User table doesn't have password_hash - use OAuth flow for actual login

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     CREATE TEST USER                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Parse arguments
	email := "test@demo.com"
	name := "Test User"

	if len(os.Args) > 1 {
		email = os.Args[1]
	}
	if len(os.Args) > 2 {
		name = os.Args[2]
	}

	fmt.Printf("📋 Creating user:\n")
	fmt.Printf("   Email: %s\n", email)
	fmt.Printf("   Name:  %s\n", name)
	fmt.Println()
	fmt.Println("Note: User will need to authenticate via OAuth (no password in DB)")
	fmt.Println()

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: .env file not found, using environment variables")
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

	// Note: User table doesn't have password_hash column
	// For testing, create user without password (will use OAuth in real flow)

	// Create user
	userID := uuid.New().String()

	_, err = pool.Exec(ctx, `
		INSERT INTO "user" (id, email, name, "emailVerified")
		VALUES ($1, $2, $3, true)
		ON CONFLICT (email) DO UPDATE
		SET
			name = EXCLUDED.name,
			"emailVerified" = EXCLUDED."emailVerified"
	`, userID, email, name)

	if err != nil {
		fmt.Printf("❌ Failed to create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ User created successfully!")
	fmt.Println()

	// Create default workspace
	workspaceSlug := fmt.Sprintf("test-workspace-%d", time.Now().Unix())
	workspaceID := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, slug, owner_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (slug) DO NOTHING
	`, workspaceID, "Test Workspace", workspaceSlug, userID)

	if err != nil {
		fmt.Printf("⚠️  Failed to create workspace: %v\n", err)
		fmt.Println("   (User was created, but workspace creation failed)")
	} else {
		fmt.Println("✅ Default workspace created!")
		fmt.Printf("   Workspace ID: %s\n", workspaceID)
		fmt.Printf("   Workspace Slug: %s\n", workspaceSlug)
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    USER CREATED                                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("   Email:   %s\n", email)
	fmt.Printf("   User ID: %s\n", userID)
	fmt.Println()
	fmt.Println("⚠️  Authentication Note:")
	fmt.Println("   This user has NO PASSWORD in the database.")
	fmt.Println("   Use OAuth flow (Google) to authenticate:")
	fmt.Println()
	fmt.Println("🎯 Start onboarding:")
	fmt.Println("   http://localhost:5173/onboarding")
	fmt.Println()
	fmt.Println("   (User will authenticate via Google OAuth during onboarding)")
	fmt.Println()
}
