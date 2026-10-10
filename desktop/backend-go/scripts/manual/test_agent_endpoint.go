package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
)

func main() {
	godotenv.Load()

	sessionToken := os.Getenv("BUSINESSOS_SESSION_TOKEN")
	if len(sessionToken) < 20 {
		fmt.Fprintln(os.Stderr, "BUSINESSOS_SESSION_TOKEN is required")
		os.Exit(2)
	}

	// Simulate the exact request
	payload := map[string]interface{}{
		"test_message": "Hello from Go test script!",
	}

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest(
		"POST",
		"http://localhost:8001/api/ai/custom-agents/32814bf2-5ac8-4926-b534-ad5f2168f8be/test",
		bytes.NewBuffer(jsonData),
	)

	req.Header.Set("Content-Type", "application/json")

	// Add the session cookie
	cookie := &http.Cookie{
		Name:  "better-auth.session_token",
		Value: sessionToken,
	}
	req.AddCookie(cookie)

	fmt.Println("🚀 Testing agent endpoint WITH authentication cookie...")
	fmt.Printf("📋 Cookie: %s\n", sessionToken[:20]+"...")
	fmt.Println()

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("❌ Request error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("📊 Response Status: %d\n", resp.StatusCode)
	if resp.StatusCode == 200 {
		fmt.Println("✅ SUCCESS! Agent test endpoint is working with authentication!")
	} else if resp.StatusCode == 401 {
		fmt.Println("❌ STILL 401 - Cookie not being accepted by backend")
	} else {
		fmt.Printf("⚠️ Unexpected status: %d\n", resp.StatusCode)
	}
	fmt.Printf("📄 Response Body:\n%s\n", string(body))
}
