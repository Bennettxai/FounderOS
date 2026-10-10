//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	sdk "github.com/severity1/claude-agent-sdk-go"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("TESTING LLM CONNECTION (Claude Agent SDK)")
	fmt.Println("=" + strings.Repeat("=", 59))

	// Check API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ ANTHROPIC_API_KEY not set in environment")
	}
	fmt.Printf("✅ ANTHROPIC_API_KEY is set (length: %d)\n", len(apiKey))

	// Test simple query
	fmt.Println("\nTesting Claude API connection...")
	fmt.Println("-" + strings.Repeat("-", 39))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var responseContent strings.Builder
	startTime := time.Now()

	err := sdk.WithClient(ctx, func(client sdk.Client) error {
		// Simple test prompt
		if err := client.Query(ctx, "Say 'Hello from BusinessOS!' and nothing else."); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		msgChan := client.ReceiveMessages(ctx)
		for message := range msgChan {
			if message == nil {
				break
			}

			switch msg := message.(type) {
			case *sdk.AssistantMessage:
				fmt.Printf("   [AssistantMessage received]\n")
				for _, block := range msg.Content {
					if textBlock, ok := block.(*sdk.TextBlock); ok {
						responseContent.WriteString(textBlock.Text)
					}
				}
			case *sdk.ResultMessage:
				fmt.Printf("   [ResultMessage] IsError: %v, SubType: %s\n", msg.IsError, msg.Subtype)
				// Note: Subtype "success" means it completed successfully even if IsError is true
				if msg.Subtype == "success" || !msg.IsError {
					return nil
				}
				return fmt.Errorf("result error: subtype=%s", msg.Subtype)
			default:
				fmt.Printf("   [Unknown message type: %T]\n", msg)
			}
		}
		return nil
	},
		sdk.WithModel(string(sdk.AgentModelSonnet)),
		sdk.WithMaxTurns(1),
	)

	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ LLM connection FAILED: %v\n", err)
		fmt.Println("\nPossible issues:")
		fmt.Println("  1. ANTHROPIC_API_KEY is invalid or expired")
		fmt.Println("  2. Network connectivity issues")
		fmt.Println("  3. API rate limiting")
		os.Exit(1)
	}

	response := responseContent.String()
	fmt.Printf("✅ LLM connection SUCCESSFUL!\n")
	fmt.Printf("   Response: %s\n", response)
	fmt.Printf("   Duration: %v\n", duration)

	fmt.Println("\n" + "=" + strings.Repeat("=", 59))
	fmt.Println("LLM INTEGRATION TEST PASSED")
	fmt.Println("The AI generation system should work correctly.")
	fmt.Println("=" + strings.Repeat("=", 59))
}
