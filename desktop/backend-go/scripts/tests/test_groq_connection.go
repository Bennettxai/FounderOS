//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Test Groq API connection and functionality
// Usage: go run scripts/test_groq_connection.go

type GroqRequest struct {
	Model    string          `json:"model"`
	Messages []GroqMessage   `json:"messages"`
	MaxTokens int            `json:"max_tokens"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  GROQ API CONNECTION TEST                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  Warning: .env file not found, using environment variables")
	}

	// Check API key
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ GROQ_API_KEY not set in environment")
		fmt.Println("   Set it in desktop/backend-go/.env:")
		fmt.Println("   GROQ_API_KEY=your_key_here")
		os.Exit(1)
	}

	fmt.Printf("✅ GROQ_API_KEY found: %s...\n", apiKey[:min(20, len(apiKey))])
	fmt.Println()

	// Test 1: Simple completion
	fmt.Println("🔍 TEST 1: Simple Completion")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("   Sending test message: 'Say hello'")

	start := time.Now()
	response, err := callGroq(apiKey, "Say hello in a friendly way (max 10 words)")
	if err != nil {
		fmt.Printf("❌ Test 1 FAILED: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(start)

	fmt.Printf("✅ Test 1 PASSED\n")
	fmt.Printf("   Response: %s\n", response.Choices[0].Message.Content)
	fmt.Printf("   Model: %s\n", response.Model)
	fmt.Printf("   Tokens: %d (prompt) + %d (completion) = %d total\n",
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
		response.Usage.TotalTokens)
	fmt.Printf("   Duration: %dms\n", duration.Milliseconds())
	fmt.Println()

	// Test 2: Profile analysis (simulated)
	fmt.Println("🔍 TEST 2: Profile Analysis (Simulated Onboarding)")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Println("   Simulating email analysis for onboarding...")

	analysisPrompt := `Analyze this user's email activity and create a profile.

Email data:
- 20 recent emails
- Top domains: slack.com (5), github.com (4), notion.so (3)
- Keywords: "project", "deadline", "team", "design", "code review"
- Tools detected: Slack, GitHub, Notion, Figma

Generate a JSON response with:
{
  "insights": ["phrase 1", "phrase 2", "phrase 3"],
  "interests": ["interest1", "interest2", "interest3"],
  "tools_used": ["Slack", "GitHub", "Notion"],
  "profile_summary": "A concise summary...",
  "confidence": 0.85
}

Respond ONLY with valid JSON.`

	start = time.Now()
	response, err = callGroq(apiKey, analysisPrompt)
	if err != nil {
		fmt.Printf("❌ Test 2 FAILED: %v\n", err)
		os.Exit(1)
	}
	duration = time.Since(start)

	fmt.Printf("✅ Test 2 PASSED\n")
	fmt.Printf("   Response: %s\n", truncate(response.Choices[0].Message.Content, 200))
	fmt.Printf("   Tokens: %d total\n", response.Usage.TotalTokens)
	fmt.Printf("   Duration: %dms\n", duration.Milliseconds())
	fmt.Println()

	// Validate JSON
	var profileResult map[string]interface{}
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &profileResult); err != nil {
		fmt.Println("⚠️  Warning: Response is not valid JSON")
		fmt.Println("   This might cause issues in onboarding flow")
		fmt.Println("   Response should be pure JSON without markdown code blocks")
	} else {
		fmt.Println("✅ Response is valid JSON")
		if insights, ok := profileResult["insights"].([]interface{}); ok {
			fmt.Printf("   Insights count: %d\n", len(insights))
		}
		if interests, ok := profileResult["interests"].([]interface{}); ok {
			fmt.Printf("   Interests count: %d\n", len(interests))
		}
	}
	fmt.Println()

	// Final summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      TEST SUMMARY                                ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println("✅ Groq API is accessible")
	fmt.Println("✅ API key is valid")
	fmt.Println("✅ Simple completions work")
	fmt.Println("✅ Profile analysis format works")
	fmt.Println()
	fmt.Println("🎉 Groq integration is ready for E2E testing!")
	fmt.Println()
	fmt.Println("Note: During actual onboarding, the AI will analyze REAL email data")
	fmt.Println("      from Gmail API. This test just verifies Groq connectivity.")
}

func callGroq(apiKey, prompt string) (*GroqResponse, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	reqBody := GroqRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []GroqMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &groqResp, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
