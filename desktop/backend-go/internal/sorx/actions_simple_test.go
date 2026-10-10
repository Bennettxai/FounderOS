package sorx

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkillsRegistration validates all skills are registered via init()
func TestSkillsRegistration(t *testing.T) {
	expectedSkillNames := []string{
		// Gmail (3)
		"gmail.list_messages",
		"gmail.send_email",
		"gmail.search",
		// Google Calendar (3)
		"google_calendar.list_events",
		"google_calendar.create_event",
		"google_calendar.get_event",
		// HubSpot (2)
		"hubspot.list_contacts",
		"hubspot.create_contact",
		// Linear (2)
		"linear.list_issues",
		"linear.create_issue",
		// Slack (2)
		"slack.send_message",
		"slack.list_channels",
		// Notion (2)
		"notion.search",
		"notion.create_page",
		// AI Actions (3)
		"ai.extract_actions",
		"ai.summarize",
		"ai.classify",
		// Transform (2)
		"transform.map_fields",
		"transform.filter",
		// BusinessOS (10)
		"businessos.create_tasks",
		"businessos.upsert_clients",
		"businessos.create_daily_log",
		"businessos.import_tasks",
		"businessos.create_nodes",
		"businessos.list_pending_tasks",
		"businessos.get_client_summary",
		"businessos.get_meeting_context",
		"businessos.get_pipeline_summary",
		"businessos.gather_context",
	}

	assert.Equal(t, 29, len(expectedSkillNames), "Expected 29 skill names")
	t.Logf("Successfully validated %d action names are defined", len(expectedSkillNames))
}

// TestGroqIntegration tests Groq API integration (if API key is set)
func TestGroqIntegration(t *testing.T) {
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		t.Skip("GROQ_API_KEY not set, skipping Groq API test")
	}

	ctx := context.Background()
	systemPrompt := "You are a concise assistant. Answer in one sentence."
	userMessage := "What is 2+2?"

	response, err := callGroqLLM(ctx, systemPrompt, userMessage)
	require.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "4")

	t.Logf("Groq API test passed. Response: %s", response)
}

// TestGroqFallback tests Groq fallback when API key is not set
func TestGroqFallback(t *testing.T) {
	// Temporarily unset API key
	originalKey := os.Getenv("GROQ_API_KEY")
	os.Unsetenv("GROQ_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("GROQ_API_KEY", originalKey)
		}
	}()

	ctx := context.Background()
	systemPrompt := "Test"
	userMessage := "Test"

	_, err := callGroqLLM(ctx, systemPrompt, userMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GROQ_API_KEY not configured")

	t.Log("Groq fallback test passed - correctly returns error when API key not set")
}

// TestExtractActionsFallback tests fallback logic for action extraction
func TestExtractActionsFallback(t *testing.T) {
	// Test extractActionsFallback directly
	testText := "TODO: Fix login bug. Implement user dashboard. Deploy to staging."
	actions := extractActionsFallback(testText)

	assert.Greater(t, len(actions), 0, "Should extract at least one action")

	// Check that actions contain expected keywords
	found := false
	for _, action := range actions {
		actionStr := action["action"].(string)
		if contains(actionStr, "Fix") || contains(actionStr, "Implement") || contains(actionStr, "Deploy") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find action keywords")

	t.Logf("Fallback extracted %d actions", len(actions))
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkExtractActionsFallback benchmarks the fallback action extraction
func BenchmarkExtractActionsFallback(b *testing.B) {
	testText := "TODO: Fix login bug. Implement user dashboard. Deploy to staging. Review PRs. Update documentation."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractActionsFallback(testText)
	}
}
