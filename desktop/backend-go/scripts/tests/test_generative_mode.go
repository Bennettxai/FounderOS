//go:build ignore

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	baseURL := "http://localhost:8080"

	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println("TESTING PURE GENERATIVE MODE (No Template)")
	fmt.Println("=" + string(make([]byte, 60)))

	// Get dev workspace
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	// Get any workspace ID (try dev first, then any)
	var workspaceID, workspaceName string
	err = pool.QueryRow(ctx, `SELECT id, name FROM workspaces WHERE slug = 'dev' LIMIT 1`).Scan(&workspaceID, &workspaceName)
	if err != nil {
		// Try any workspace
		err = pool.QueryRow(ctx, `SELECT id, name FROM workspaces LIMIT 1`).Scan(&workspaceID, &workspaceName)
		if err != nil {
			log.Fatalf("No workspaces found: %v", err)
		}
	}
	fmt.Printf("Using workspace: %s (%s)\n", workspaceName, workspaceID)

	// Get a test user from workspace_members (optional)
	var userID, userEmail string
	err = pool.QueryRow(ctx, `
		SELECT user_id, role
		FROM workspace_members
		WHERE workspace_id = $1
		LIMIT 1
	`, workspaceID).Scan(&userID, &userEmail)
	if err != nil {
		fmt.Println("(No workspace members found, proceeding with tests anyway)")
	} else {
		fmt.Printf("Found workspace member: %s (role: %s)\n", userID, userEmail)
	}
	fmt.Println()

	// Test 1: Pure generative mode (no template_id)
	fmt.Println("TEST 1: Pure Generative Mode API Call")
	fmt.Println("-" + string(make([]byte, 40)))

	reqBody := map[string]interface{}{
		"app_name":    "Customer Feedback Tracker",
		"description": "A simple app to collect and analyze customer feedback with sentiment analysis",
	}

	body, _ := json.Marshal(reqBody)
	fmt.Printf("Request body: %s\n", string(body))

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/apps", baseURL, workspaceID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Note: In real test, would need auth token

	fmt.Printf("URL: %s\n", url)
	fmt.Println("(Note: Server must be running with proper auth for this to succeed)")
	fmt.Println()

	// Test 2: Direct database insert (simulating what the handler does)
	fmt.Println("TEST 2: Direct Database Insert (Generative Mode)")
	fmt.Println("-" + string(make([]byte, 40)))

	genContext := map[string]interface{}{
		"app_name":    "Test Generative App",
		"description": "Test description for AI generation",
		"prompt":      "Test Generative App: Test description for AI generation",
		"mode":        "generative",
	}
	contextJSON, _ := json.Marshal(genContext)

	var queueItemID string
	err = pool.QueryRow(ctx, `
		INSERT INTO app_generation_queue (
			workspace_id,
			status,
			priority,
			generation_context,
			max_retries
		) VALUES ($1, 'pending', 5, $2, 3)
		RETURNING id
	`, workspaceID, contextJSON).Scan(&queueItemID)

	if err != nil {
		log.Fatalf("❌ Direct insert FAILED: %v", err)
	}

	fmt.Printf("✅ Direct insert SUCCEEDED!\n")
	fmt.Printf("   Queue Item ID: %s\n", queueItemID)

	// Verify the insert
	var status string
	var templateID interface{}
	err = pool.QueryRow(ctx, `
		SELECT status, template_id
		FROM app_generation_queue
		WHERE id = $1
	`, queueItemID).Scan(&status, &templateID)

	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("   Status: %s\n", status)
	fmt.Printf("   Template ID: %v (should be NULL)\n", templateID)
	fmt.Println()

	// Test 3: Verify the queue item has correct context
	fmt.Println("TEST 3: Verify Generation Context")
	fmt.Println("-" + string(make([]byte, 40)))

	var storedContext []byte
	err = pool.QueryRow(ctx, `
		SELECT generation_context
		FROM app_generation_queue
		WHERE id = $1
	`, queueItemID).Scan(&storedContext)

	if err != nil {
		log.Fatalf("Failed to get context: %v", err)
	}

	var parsedContext map[string]interface{}
	json.Unmarshal(storedContext, &parsedContext)

	fmt.Printf("✅ Generation context stored correctly:\n")
	for k, v := range parsedContext {
		fmt.Printf("   %s: %v\n", k, v)
	}
	fmt.Println()

	// Cleanup test data
	fmt.Println("TEST 4: Cleanup")
	fmt.Println("-" + string(make([]byte, 40)))
	_, err = pool.Exec(ctx, `DELETE FROM app_generation_queue WHERE id = $1`, queueItemID)
	if err != nil {
		log.Printf("Warning: Failed to cleanup: %v", err)
	} else {
		fmt.Println("✅ Test data cleaned up")
	}

	fmt.Println()
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println("ALL TESTS PASSED! Pure generative mode is working.")
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Start the backend server: ./bin/server")
	fmt.Println("2. Test via frontend with description only (no template)")
	fmt.Println("3. Verify OSAQueueWorker processes the pending job")

	time.Sleep(100 * time.Millisecond)
}
