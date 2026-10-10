// +build integration

package sorx

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveActions is the main entry point for live API testing
// Run with: go test -tags=integration -v ./internal/sorx -run TestLiveActions
func TestLiveActions(t *testing.T) {
	// Load configuration from .env.test.local or .env.test
	config, err := LoadLiveTestConfig()
	require.NoError(t, err, "Failed to load test configuration")

	// Create test runner
	runner := NewLiveTestRunner(config)

	// Run all test categories
	t.Run("Gmail", func(t *testing.T) {
		testGmailActions(t, runner, config)
	})

	t.Run("GoogleCalendar", func(t *testing.T) {
		testGoogleCalendarActions(t, runner, config)
	})

	t.Run("Slack", func(t *testing.T) {
		testSlackActions(t, runner, config)
	})

	t.Run("Notion", func(t *testing.T) {
		testNotionActions(t, runner, config)
	})

	t.Run("Linear", func(t *testing.T) {
		testLinearActions(t, runner, config)
	})

	t.Run("HubSpot", func(t *testing.T) {
		testHubSpotActions(t, runner, config)
	})

	t.Run("AI", func(t *testing.T) {
		testAIActions(t, runner, config)
	})

	t.Run("AIProviders", func(t *testing.T) {
		testAIProviderActions(t, runner, config)
	})

	t.Run("Transform", func(t *testing.T) {
		testTransformActions(t, runner, config)
	})

	t.Run("BusinessOS", func(t *testing.T) {
		testBusinessOSActions(t, runner, config)
	})

	// Finalize and generate report
	err = runner.Finalize()
	require.NoError(t, err, "Failed to finalize test run")
}

// ============================================================================
// GMAIL ACTIONS
// ============================================================================

func testGmailActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "google")

	t.Run("gmail.list_messages", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"gmail.list_messages",
			"gmail",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"max_results": 5,
					},
				), nil
			},
			func(result interface{}) error {
				messages, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				messagesList, ok := messages["messages"].([]interface{})
				if !ok {
					return fmt.Errorf("expected messages array")
				}
				assert.True(t, len(messagesList) <= 5, "Should return at most 5 messages")
				return nil
			},
			nil, // No cleanup needed for read operation
		)
	})

	t.Run("gmail.send_email", func(t *testing.T) {
		var messageID string

		runner.RunActionTest(
			t,
			"gmail.send_email",
			"gmail",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"to":      config.GoogleTestEmail,
						"subject": "[BusinessOS Test] Automated Test Email",
						"body":    fmt.Sprintf("This is an automated test email sent at %s", time.Now().Format(time.RFC3339)),
					},
				), nil
			},
			func(result interface{}) error {
				response, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := response["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing message id")
				}
				messageID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Delete test email if configured
				if messageID != "" && config.TestCleanupMode != "never" {
					// Note: Gmail API doesn't support permanent deletion via regular API
					// Messages go to trash instead
					return nil
				}
				return nil
			},
		)
	})

	t.Run("gmail.search", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"gmail.search",
			"gmail",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"query":       "subject:[BusinessOS Test]",
						"max_results": 10,
					},
				), nil
			},
			func(result interface{}) error {
				messages, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// Search may return 0 results if no test emails exist yet
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// GOOGLE CALENDAR ACTIONS
// ============================================================================

func testGoogleCalendarActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "google")

	t.Run("google_calendar.list_events", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"google_calendar.list_events",
			"calendar",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"max_results": 10,
						"time_min":    time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
						"time_max":    time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
					},
				), nil
			},
			func(result interface{}) error {
				events, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 events if calendar is empty
				return nil
			},
			nil,
		)
	})

	t.Run("google_calendar.create_event", func(t *testing.T) {
		var eventID string

		runner.RunActionTest(
			t,
			"google_calendar.create_event",
			"calendar",
			func() (ActionContext, error) {
				startTime := time.Now().Add(24 * time.Hour).Truncate(time.Hour)
				endTime := startTime.Add(1 * time.Hour)

				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"summary":     "[BusinessOS Test] Automated Test Event",
						"description": "This is an automated test event",
						"start_time":  startTime.Format(time.RFC3339),
						"end_time":    endTime.Format(time.RFC3339),
					},
				), nil
			},
			func(result interface{}) error {
				event, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := event["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing event id")
				}
				eventID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Delete test event
				if eventID != "" {
					// Note: Would need google_calendar.delete_event action
					// For now, test events will remain in calendar
					return nil
				}
				return nil
			},
		)
	})

	t.Run("google_calendar.get_event", func(t *testing.T) {
		// First create an event to retrieve
		startTime := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
		endTime := startTime.Add(1 * time.Hour)

		createCtx := CreateTestActionContext(
			GenerateTestID(),
			creds,
			map[string]interface{}{
				"summary":     "[BusinessOS Test] Event for Get Test",
				"description": "Test event for get operation",
				"start_time":  startTime.Format(time.RFC3339),
				"end_time":    endTime.Format(time.RFC3339),
			},
		)

		handler, _ := GetActionHandler("google_calendar.create_event")
		createResult, err := handler(context.Background(), createCtx)
		if err != nil {
			t.Skipf("Cannot test get_event: failed to create test event: %v", err)
			return
		}

		event := createResult.(map[string]interface{})
		eventID := event["id"].(string)

		runner.RunActionTest(
			t,
			"google_calendar.get_event",
			"calendar",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"event_id": eventID,
					},
				), nil
			},
			func(result interface{}) error {
				event, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := event["id"].(string)
				if !ok || id != eventID {
					return fmt.Errorf("event id mismatch: expected %s, got %v", eventID, id)
				}
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// SLACK ACTIONS
// ============================================================================

func testSlackActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "slack")

	t.Run("slack.list_channels", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"slack.list_channels",
			"slack",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"limit": 100,
					},
				), nil
			},
			func(result interface{}) error {
				channels, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				channelsList, ok := channels["channels"].([]interface{})
				if !ok {
					return fmt.Errorf("expected channels array")
				}
				assert.True(t, len(channelsList) > 0, "Should have at least one channel")
				return nil
			},
			nil,
		)
	})

	t.Run("slack.send_message", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"slack.send_message",
			"slack",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"channel": config.SlackTestChannelID,
						"text":    fmt.Sprintf("[BusinessOS Test] Automated test message at %s", time.Now().Format(time.RFC3339)),
					},
				), nil
			},
			func(result interface{}) error {
				response, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				ok, _ = response["ok"].(bool)
				if !ok {
					return fmt.Errorf("message send failed")
				}
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Slack messages cannot be deleted via bot token
				// They remain in the test channel
				return nil
			},
		)
	})
}

// ============================================================================
// NOTION ACTIONS
// ============================================================================

func testNotionActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "notion")

	t.Run("notion.search", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"notion.search",
			"notion",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"query": "test",
					},
				), nil
			},
			func(result interface{}) error {
				response, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 results if nothing matches
				return nil
			},
			nil,
		)
	})

	t.Run("notion.create_page", func(t *testing.T) {
		var pageID string

		runner.RunActionTest(
			t,
			"notion.create_page",
			"notion",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"database_id": config.NotionTestDatabaseID,
						"title":       fmt.Sprintf("[BusinessOS Test] Page %s", GenerateTestID()),
						"properties": map[string]interface{}{
							"Status": map[string]interface{}{
								"select": map[string]interface{}{
									"name": "In Progress",
								},
							},
						},
					},
				), nil
			},
			func(result interface{}) error {
				page, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := page["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing page id")
				}
				pageID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Archive test page
				if pageID != "" {
					// Note: Would need notion.archive_page action
					// For now, test pages remain in database
					return nil
				}
				return nil
			},
		)
	})
}

// ============================================================================
// LINEAR ACTIONS
// ============================================================================

func testLinearActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "linear")

	t.Run("linear.list_issues", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"linear.list_issues",
			"linear",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"team_id": config.LinearTestTeamID,
						"limit":   10,
					},
				), nil
			},
			func(result interface{}) error {
				issues, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 issues if team is empty
				return nil
			},
			nil,
		)
	})

	t.Run("linear.create_issue", func(t *testing.T) {
		var issueID string

		runner.RunActionTest(
			t,
			"linear.create_issue",
			"linear",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"team_id":     config.LinearTestTeamID,
						"title":       fmt.Sprintf("[BusinessOS Test] Issue %s", GenerateTestID()),
						"description": "This is an automated test issue",
						"priority":    2, // Normal priority
					},
				), nil
			},
			func(result interface{}) error {
				issue, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := issue["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing issue id")
				}
				issueID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Archive or delete test issue
				if issueID != "" {
					// Note: Would need linear.archive_issue action
					// For now, test issues remain in team
					return nil
				}
				return nil
			},
		)
	})

	t.Run("linear.get_issue", func(t *testing.T) {
		// First create an issue to retrieve
		createCtx := CreateTestActionContext(
			GenerateTestID(),
			creds,
			map[string]interface{}{
				"team_id":     config.LinearTestTeamID,
				"title":       fmt.Sprintf("[BusinessOS Test] Issue for Get Test %s", GenerateTestID()),
				"description": "Test issue for get operation",
				"priority":    2,
			},
		)

		handler, _ := GetActionHandler("linear.create_issue")
		createResult, err := handler(context.Background(), createCtx)
		if err != nil {
			t.Skipf("Cannot test get_issue: failed to create test issue: %v", err)
			return
		}

		issue := createResult.(map[string]interface{})
		issueID := issue["id"].(string)

		runner.RunActionTest(
			t,
			"linear.get_issue",
			"linear",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"issue_id": issueID,
					},
				), nil
			},
			func(result interface{}) error {
				issue, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := issue["id"].(string)
				if !ok || id != issueID {
					return fmt.Errorf("issue id mismatch: expected %s, got %v", issueID, id)
				}
				return nil
			},
			nil,
		)
	})

	t.Run("linear.update_issue", func(t *testing.T) {
		// First create an issue to update
		createCtx := CreateTestActionContext(
			GenerateTestID(),
			creds,
			map[string]interface{}{
				"team_id":     config.LinearTestTeamID,
				"title":       fmt.Sprintf("[BusinessOS Test] Issue for Update Test %s", GenerateTestID()),
				"description": "Initial description",
				"priority":    2,
			},
		)

		handler, _ := GetActionHandler("linear.create_issue")
		createResult, err := handler(context.Background(), createCtx)
		if err != nil {
			t.Skipf("Cannot test update_issue: failed to create test issue: %v", err)
			return
		}

		issue := createResult.(map[string]interface{})
		issueID := issue["id"].(string)

		runner.RunActionTest(
			t,
			"linear.update_issue",
			"linear",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"issue_id":    issueID,
						"title":       "[BusinessOS Test] Updated Title",
						"description": "Updated description via test",
						"priority":    1, // High priority
					},
				), nil
			},
			func(result interface{}) error {
				issue, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				updated, ok := issue["updated"].(bool)
				if !ok || !updated {
					return fmt.Errorf("issue update failed")
				}
				return nil
			},
			nil,
		)
	})

	t.Run("linear.search_issues", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"linear.search_issues",
			"linear",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"query":   "[BusinessOS Test]",
						"team_id": config.LinearTestTeamID,
						"limit":   20,
					},
				), nil
			},
			func(result interface{}) error {
				issues, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 issues if no test issues exist
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// HUBSPOT ACTIONS
// ============================================================================

func testHubSpotActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	creds := CreateTestCredentials(config, "hubspot")

	t.Run("hubspot.list_contacts", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"hubspot.list_contacts",
			"hubspot",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"limit": 10,
					},
				), nil
			},
			func(result interface{}) error {
				contacts, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 contacts if HubSpot is empty
				return nil
			},
			nil,
		)
	})

	t.Run("hubspot.create_contact", func(t *testing.T) {
		var contactID string

		runner.RunActionTest(
			t,
			"hubspot.create_contact",
			"hubspot",
			func() (ActionContext, error) {
				testID := GenerateTestID()
				return CreateTestActionContext(
					testID,
					creds,
					map[string]interface{}{
						"email":      fmt.Sprintf("test-%s@businessos.test", testID),
						"firstname":  "BusinessOS",
						"lastname":   "Test",
						"company":    "BusinessOS Testing",
						"phone":      "+1-555-TEST-001",
					},
				), nil
			},
			func(result interface{}) error {
				contact, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := contact["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing contact id")
				}
				contactID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Delete test contact
				if contactID != "" {
					// Note: Would need hubspot.delete_contact action
					// For now, test contacts remain in HubSpot
					return nil
				}
				return nil
			},
		)
	})

	t.Run("hubspot.get_contact", func(t *testing.T) {
		// First create a contact to retrieve
		testID := GenerateTestID()
		createCtx := CreateTestActionContext(
			testID,
			creds,
			map[string]interface{}{
				"email":     fmt.Sprintf("get-test-%s@businessos.test", testID),
				"firstname": "GetTest",
				"lastname":  "Contact",
			},
		)

		handler, _ := GetActionHandler("hubspot.create_contact")
		createResult, err := handler(context.Background(), createCtx)
		if err != nil {
			t.Skipf("Cannot test get_contact: failed to create test contact: %v", err)
			return
		}

		contact := createResult.(map[string]interface{})
		contactID := contact["id"].(string)

		runner.RunActionTest(
			t,
			"hubspot.get_contact",
			"hubspot",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"contact_id": contactID,
					},
				), nil
			},
			func(result interface{}) error {
				contact, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := contact["id"].(string)
				if !ok || id != contactID {
					return fmt.Errorf("contact id mismatch: expected %s, got %v", contactID, id)
				}
				return nil
			},
			nil,
		)
	})

	t.Run("hubspot.update_contact", func(t *testing.T) {
		// First create a contact to update
		testID := GenerateTestID()
		createCtx := CreateTestActionContext(
			testID,
			creds,
			map[string]interface{}{
				"email":     fmt.Sprintf("update-test-%s@businessos.test", testID),
				"firstname": "UpdateTest",
				"lastname":  "Contact",
			},
		)

		handler, _ := GetActionHandler("hubspot.create_contact")
		createResult, err := handler(context.Background(), createCtx)
		if err != nil {
			t.Skipf("Cannot test update_contact: failed to create test contact: %v", err)
			return
		}

		contact := createResult.(map[string]interface{})
		contactID := contact["id"].(string)

		runner.RunActionTest(
			t,
			"hubspot.update_contact",
			"hubspot",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"contact_id": contactID,
						"firstname":  "Updated",
						"lastname":   "Name",
						"company":    "Updated Company",
					},
				), nil
			},
			func(result interface{}) error {
				contact, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				updated, ok := contact["updated"].(bool)
				if !ok || !updated {
					return fmt.Errorf("contact update failed")
				}
				return nil
			},
			nil,
		)
	})

	t.Run("hubspot.search_contacts", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"hubspot.search_contacts",
			"hubspot",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"query": "BusinessOS",
						"limit": 20,
					},
				), nil
			},
			func(result interface{}) error {
				contacts, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 contacts if no matches
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// AI ACTIONS
// ============================================================================

func testAIActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	// Test with available AI providers
	providers := config.GetAvailableProviders()
	if len(providers) == 0 {
		t.Skip("No AI providers configured")
		return
	}

	// Use first available provider
	provider := providers[0]
	var creds map[string]string

	switch provider {
	case "anthropic":
		creds = CreateTestCredentials(config, "anthropic")
	case "openai":
		creds = CreateTestCredentials(config, "openai")
	case "groq":
		creds = CreateTestCredentials(config, "groq")
	}

	t.Run("ai.extract_actions", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"ai.extract_actions",
			"ai",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"text": "Send an email to john@example.com about the project status. Also, create a calendar event for tomorrow at 2pm for the team meeting.",
					},
				), nil
			},
			func(result interface{}) error {
				actions, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				actionsList, ok := actions["actions"].([]interface{})
				if !ok {
					return fmt.Errorf("expected actions array")
				}
				assert.True(t, len(actionsList) > 0, "Should extract at least one action")
				return nil
			},
			nil,
		)
	})

	t.Run("ai.summarize", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"ai.summarize",
			"ai",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"text": "The quarterly review meeting covered three main topics: revenue growth exceeded targets by 15%, the new product launch is scheduled for Q2, and the team will be expanded by 5 new hires. Action items include updating the budget forecast, finalizing the product roadmap, and posting the job openings by end of week.",
					},
				), nil
			},
			func(result interface{}) error {
				summary, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				text, ok := summary["summary"].(string)
				if !ok || text == "" {
					return fmt.Errorf("missing summary text")
				}
				return nil
			},
			nil,
		)
	})

	t.Run("ai.classify", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"ai.classify",
			"ai",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					creds,
					map[string]interface{}{
						"text": "Urgent: The production server is down and customers are reporting errors!",
						"categories": []string{
							"urgent",
							"normal",
							"low_priority",
						},
					},
				), nil
			},
			func(result interface{}) error {
				classification, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				category, ok := classification["category"].(string)
				if !ok || category == "" {
					return fmt.Errorf("missing classification category")
				}
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// TRANSFORM ACTIONS
// ============================================================================

func testTransformActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	t.Run("transform.map_fields", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"transform.map_fields",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"data": map[string]interface{}{
							"first_name": "John",
							"last_name":  "Doe",
							"email_addr": "john@example.com",
						},
						"mapping": map[string]interface{}{
							"firstName": "first_name",
							"lastName":  "last_name",
							"email":     "email_addr",
						},
					},
				), nil
			},
			func(result interface{}) error {
				mapped, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				assert.Equal(t, "John", mapped["firstName"], "firstName should be mapped")
				assert.Equal(t, "Doe", mapped["lastName"], "lastName should be mapped")
				assert.Equal(t, "john@example.com", mapped["email"], "email should be mapped")
				return nil
			},
			nil,
		)
	})

	t.Run("transform.filter", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"transform.filter",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"data": []interface{}{
							map[string]interface{}{"name": "Task 1", "status": "completed"},
							map[string]interface{}{"name": "Task 2", "status": "pending"},
							map[string]interface{}{"name": "Task 3", "status": "completed"},
						},
						"filter": map[string]interface{}{
							"field": "status",
							"value": "completed",
						},
					},
				), nil
			},
			func(result interface{}) error {
				filtered, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				items, ok := filtered["items"].([]interface{})
				if !ok {
					return fmt.Errorf("expected items array")
				}
				assert.Equal(t, 2, len(items), "Should filter to 2 completed tasks")
				return nil
			},
			nil,
		)
	})

	t.Run("json.parse", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"json.parse",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"json_string": `{"name":"John","age":30,"active":true}`,
					},
				), nil
			},
			func(result interface{}) error {
				parsed, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				assert.Equal(t, "John", parsed["name"], "name should be John")
				assert.Equal(t, float64(30), parsed["age"], "age should be 30")
				assert.Equal(t, true, parsed["active"], "active should be true")
				return nil
			},
			nil,
		)
	})

	t.Run("json.stringify", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"json.stringify",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"data": map[string]interface{}{
							"name":   "Jane",
							"email":  "jane@example.com",
							"active": true,
							"score":  95.5,
						},
					},
				), nil
			},
			func(result interface{}) error {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				jsonStr, ok := resultMap["json_string"].(string)
				if !ok || jsonStr == "" {
					return fmt.Errorf("missing json_string in result")
				}
				// Validate it's valid JSON by parsing
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
					return fmt.Errorf("invalid JSON output: %v", err)
				}
				return nil
			},
			nil,
		)
	})

	t.Run("base64.encode", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"base64.encode",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"data": "Hello, BusinessOS!",
					},
				), nil
			},
			func(result interface{}) error {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				encoded, ok := resultMap["encoded"].(string)
				if !ok || encoded == "" {
					return fmt.Errorf("missing encoded data")
				}
				// Expected: SGVsbG8sIEJ1c2luZXNzT1Mh
				assert.NotEmpty(t, encoded, "encoded should not be empty")
				return nil
			},
			nil,
		)
	})

	t.Run("base64.decode", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"base64.decode",
			"transform",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"encoded": "SGVsbG8sIEJ1c2luZXNzT1Mh",
					},
				), nil
			},
			func(result interface{}) error {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				decoded, ok := resultMap["decoded"].(string)
				if !ok {
					return fmt.Errorf("missing decoded data")
				}
				assert.Equal(t, "Hello, BusinessOS!", decoded, "decoded should match original")
				return nil
			},
			nil,
		)
	})
}

// ============================================================================
// AI PROVIDER ACTIONS (Anthropic, OpenAI, Groq)
// ============================================================================

func testAIProviderActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	// Test each available provider
	providers := config.GetAvailableProviders()
	if len(providers) == 0 {
		t.Skip("No AI providers configured")
		return
	}

	for _, provider := range providers {
		providerCreds := CreateTestCredentials(config, provider)
		testName := fmt.Sprintf("%s_complete", provider)

		t.Run(testName, func(t *testing.T) {
			runner.RunActionTest(
				t,
				testName,
				"ai",
				func() (ActionContext, error) {
					return CreateTestActionContext(
						GenerateTestID(),
						providerCreds,
						map[string]interface{}{
							"prompt":      "What is 2+2? Answer with just the number.",
							"max_tokens":  50,
							"temperature": 0.1,
						},
					), nil
				},
				func(result interface{}) error {
					response, ok := result.(map[string]interface{})
					if !ok {
						return fmt.Errorf("expected map, got %T", result)
					}
					text, ok := response["text"].(string)
					if !ok || text == "" {
						return fmt.Errorf("missing response text")
					}
					assert.NotEmpty(t, text, "response text should not be empty")
					return nil
				},
				nil,
			)
		})
	}
}

// ============================================================================
// BUSINESSOS ACTIONS
// ============================================================================

func testBusinessOSActions(t *testing.T, runner *LiveTestRunner, config *LiveTestConfig) {
	// These tests require database connection
	if config.DatabaseURL == "" {
		t.Skip("DATABASE_URL not configured")
		return
	}

	t.Run("businessos.create_project", func(t *testing.T) {
		var projectID string

		runner.RunActionTest(
			t,
			"businessos.create_project",
			"businessos",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"name":        fmt.Sprintf("Test Project %s", GenerateTestID()),
						"description": "Automated test project",
						"status":      "active",
					},
				), nil
			},
			func(result interface{}) error {
				project, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				id, ok := project["id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("missing project id")
				}
				projectID = id
				return nil
			},
			func(ctx ActionContext, result interface{}) error {
				// Cleanup: Delete test project
				if projectID != "" {
					// Would need businessos.delete_project action
					return nil
				}
				return nil
			},
		)
	})

	t.Run("businessos.list_projects", func(t *testing.T) {
		runner.RunActionTest(
			t,
			"businessos.list_projects",
			"businessos",
			func() (ActionContext, error) {
				return CreateTestActionContext(
					GenerateTestID(),
					nil,
					map[string]interface{}{
						"limit":  20,
						"offset": 0,
					},
				), nil
			},
			func(result interface{}) error {
				projects, ok := result.(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected map, got %T", result)
				}
				// May return 0 projects if none exist
				return nil
			},
			nil,
		)
	})
}
