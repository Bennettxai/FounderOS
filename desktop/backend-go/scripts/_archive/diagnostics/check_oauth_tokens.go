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

// Check OAuth tokens in database (for debugging OAuth issues)
// Usage: go run scripts/check_oauth_tokens.go [user_email]
// Example: go run scripts/check_oauth_tokens.go test@example.com

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                CHECK OAUTH TOKENS IN DATABASE                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
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

	fmt.Println("✅ Connected to database")
	fmt.Println()

	// Get user email from args (optional)
	var userEmail string
	if len(os.Args) > 1 {
		userEmail = os.Args[1]
		fmt.Printf("🔍 Filtering for user: %s\n", userEmail)
		fmt.Println()
	} else {
		fmt.Println("🔍 Showing all OAuth integrations (no filter)")
		fmt.Println("   Tip: Use 'go run scripts/check_oauth_tokens.go user@example.com' to filter")
		fmt.Println()
	}

	// Query integrations
	query := `
		SELECT
			i.id,
			i.user_id,
			u.email,
			i.provider,
			i.status,
			i.token_expiry,
			i.created_at,
			i.updated_at,
			CASE
				WHEN i.access_token IS NOT NULL THEN true
				ELSE false
			END as has_access_token,
			CASE
				WHEN i.refresh_token IS NOT NULL THEN true
				ELSE false
			END as has_refresh_token
		FROM integrations i
		LEFT JOIN "user" u ON i.user_id = u.id
		WHERE ($1 = '' OR u.email = $1)
		ORDER BY i.created_at DESC
	`

	rows, err := pool.Query(ctx, query, userEmail)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id              uuid.UUID
			userID          string
			email           string
			provider        string
			status          string
			tokenExpiry     *time.Time
			createdAt       time.Time
			updatedAt       time.Time
			hasAccessToken  bool
			hasRefreshToken bool
		)

		err := rows.Scan(&id, &userID, &email, &provider, &status, &tokenExpiry,
			&createdAt, &updatedAt, &hasAccessToken, &hasRefreshToken)
		if err != nil {
			fmt.Printf("⚠️  Error scanning row: %v\n", err)
			continue
		}

		count++

		fmt.Println("─────────────────────────────────────────────────────────────")
		fmt.Printf("Integration #%d\n", count)
		fmt.Println("─────────────────────────────────────────────────────────────")
		fmt.Printf("  ID:              %s\n", id)
		fmt.Printf("  User ID:         %s\n", userID)
		fmt.Printf("  Email:           %s\n", email)
		fmt.Printf("  Provider:        %s\n", provider)
		fmt.Printf("  Status:          %s\n", status)

		if hasAccessToken {
			fmt.Println("  Access Token:    ✅ Present (encrypted)")
		} else {
			fmt.Println("  Access Token:    ❌ Missing")
		}

		if hasRefreshToken {
			fmt.Println("  Refresh Token:   ✅ Present (encrypted)")
		} else {
			fmt.Println("  Refresh Token:   ❌ Missing")
		}

		if tokenExpiry != nil {
			now := time.Now()
			if tokenExpiry.Before(now) {
				fmt.Printf("  Token Expiry:    ⚠️  EXPIRED (%s)\n", tokenExpiry.Format(time.RFC3339))
				fmt.Printf("                   Expired %d minutes ago\n", int(now.Sub(*tokenExpiry).Minutes()))
			} else {
				fmt.Printf("  Token Expiry:    ✅ Valid until %s\n", tokenExpiry.Format(time.RFC3339))
				fmt.Printf("                   Expires in %d minutes\n", int(tokenExpiry.Sub(now).Minutes()))
			}
		} else {
			fmt.Println("  Token Expiry:    ⚠️  Not set")
		}

		fmt.Printf("  Created:         %s\n", createdAt.Format(time.RFC3339))
		fmt.Printf("  Updated:         %s\n", updatedAt.Format(time.RFC3339))
		fmt.Println()

		// Warnings
		if status != "active" {
			fmt.Println("  ⚠️  WARNING: Integration status is not 'active'")
			fmt.Println("     User may need to reconnect OAuth")
			fmt.Println()
		}

		if !hasAccessToken {
			fmt.Println("  ❌ ERROR: Missing access token")
			fmt.Println("     OAuth flow did not complete successfully")
			fmt.Println("     User needs to reconnect")
			fmt.Println()
		}

		if tokenExpiry != nil && tokenExpiry.Before(time.Now()) && !hasRefreshToken {
			fmt.Println("  ❌ ERROR: Token expired and no refresh token")
			fmt.Println("     User MUST reconnect OAuth")
			fmt.Println()
		}
	}

	if count == 0 {
		fmt.Println("📊 No OAuth integrations found")
		if userEmail != "" {
			fmt.Printf("   No integrations for user: %s\n", userEmail)
		} else {
			fmt.Println("   Database has no OAuth integrations yet")
		}
		fmt.Println()
		fmt.Println("This is normal if:")
		fmt.Println("  • No user has completed Gmail OAuth yet")
		fmt.Println("  • Testing hasn't started")
		fmt.Println("  • Database was recently reset")
	} else {
		fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                         SUMMARY                                  ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Printf("  Total integrations: %d\n", count)
		fmt.Println()

		// Count by status
		var activeCount, expiredCount, missingTokenCount int
		rows2, _ := pool.Query(ctx, `
			SELECT
				status,
				COUNT(*),
				SUM(CASE WHEN access_token IS NULL THEN 1 ELSE 0 END) as missing_tokens,
				SUM(CASE WHEN token_expiry < NOW() THEN 1 ELSE 0 END) as expired
			FROM integrations
			WHERE ($1 = '' OR user_id IN (SELECT id FROM "user" WHERE email = $1))
			GROUP BY status
		`, userEmail)
		defer rows2.Close()

		for rows2.Next() {
			var status string
			var count, missing, expired int
			rows2.Scan(&status, &count, &missing, &expired)

			if status == "active" {
				activeCount = count
			}
			missingTokenCount += missing
			expiredCount += expired
		}

		fmt.Printf("  Active:         %d\n", activeCount)
		fmt.Printf("  Missing tokens: %d\n", missingTokenCount)
		fmt.Printf("  Expired:        %d\n", expiredCount)
		fmt.Println()

		if missingTokenCount > 0 || expiredCount > 0 {
			fmt.Println("  ⚠️  ACTION REQUIRED:")
			if missingTokenCount > 0 {
				fmt.Printf("     • %d integration(s) missing tokens - users need to reconnect\n", missingTokenCount)
			}
			if expiredCount > 0 {
				fmt.Printf("     • %d integration(s) expired - check if refresh tokens work\n", expiredCount)
			}
		} else {
			fmt.Println("  ✅ All integrations look healthy!")
		}
	}

	fmt.Println()
	fmt.Println("🎉 OAuth token check complete")
}
