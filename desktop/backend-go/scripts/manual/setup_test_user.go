package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Generate password hash
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// First check if user exists
	var userID string
	err = db.QueryRow(`SELECT id FROM "user" WHERE email = $1`, "testuser@businessos.dev").Scan(&userID)
	if err == sql.ErrNoRows {
		// Create user
		userID = "test-user-001"
		_, err = db.Exec(`INSERT INTO "user" (id, name, email, "emailVerified") VALUES ($1, $2, $3, true)`,
			userID, "Test User", "testuser@businessos.dev")
		if err != nil {
			log.Fatal("Failed to create user:", err)
		}
		fmt.Println("Created user:", userID)
	} else if err != nil {
		log.Fatal("Query error:", err)
	} else {
		fmt.Println("Found existing user:", userID)
	}

	// Check if account exists
	var accountID string
	err = db.QueryRow(`SELECT id FROM account WHERE "userId" = $1 AND "providerId" = 'credential'`, userID).Scan(&accountID)
	if err == sql.ErrNoRows {
		// Create account with password
		accountID = "acc-" + userID
		_, err = db.Exec(`INSERT INTO account (id, "userId", "accountId", "providerId", password) VALUES ($1, $2, $3, 'credential', $4)`,
			accountID, userID, userID, string(hash))
		if err != nil {
			log.Fatal("Failed to create account:", err)
		}
		fmt.Println("Created account with password")
	} else if err != nil {
		log.Fatal("Account query error:", err)
	} else {
		// Update password
		_, err = db.Exec(`UPDATE account SET password = $1 WHERE id = $2`, string(hash), accountID)
		if err != nil {
			log.Fatal("Failed to update password:", err)
		}
		fmt.Println("Updated password for existing account")
	}

	fmt.Println("\n✅ Test user ready!")
	fmt.Println("   Email: testuser@businessos.dev")
	fmt.Println("   Password: password123")
}
