// Package sorx provides built-in action handlers for the skill execution engine.
package sorx

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	// Register all built-in actions
	RegisterAction("gmail.list_messages", gmailListMessages)
	RegisterAction("gmail.send_email", gmailSendEmail)
	RegisterAction("gmail.search", gmailSearch)

	RegisterAction("google_calendar.list_events", googleCalendarListEvents)
	RegisterAction("google_calendar.create_event", googleCalendarCreateEvent)

	RegisterAction("hubspot.list_contacts", hubspotListContacts)
	RegisterAction("hubspot.create_contact", hubspotCreateContact)

	RegisterAction("linear.list_issues", linearListIssues)
	RegisterAction("linear.create_issue", linearCreateIssue)

	RegisterAction("slack.send_message", slackSendMessage)
	RegisterAction("slack.list_channels", slackListChannels)

	RegisterAction("notion.search", notionSearch)
	RegisterAction("notion.create_page", notionCreatePage)

	RegisterAction("ai.extract_actions", aiExtractActions)
	RegisterAction("ai.summarize", aiSummarize)
	RegisterAction("ai.classify", aiClassify)

	RegisterAction("transform.map_fields", transformMapFields)
	RegisterAction("transform.filter", transformFilter)

	RegisterAction("businessos.create_tasks", businessOSCreateTasks)
	RegisterAction("businessos.upsert_clients", businessOSUpsertClients)
	RegisterAction("businessos.create_daily_log", businessOSCreateDailyLog)
	RegisterAction("businessos.import_tasks", businessOSImportTasks)
	RegisterAction("businessos.create_nodes", businessOSCreateNodes)
	RegisterAction("businessos.list_pending_tasks", businessOSListPendingTasks)
	RegisterAction("businessos.get_client_summary", businessOSGetClientSummary)
	RegisterAction("businessos.get_pipeline_summary", businessOSGetPipelineSummary)
	RegisterAction("businessos.get_meeting_context", businessOSGetMeetingContext)

	RegisterAction("google_calendar.get_event", googleCalendarGetEvent)

	// Context gathering actions for command-based skills
	RegisterAction("businessos.gather_context", businessOSGatherContext)
}

// ============================================================================
// Helper Functions - Credential Loading & Retry Logic
// ============================================================================

// loadCredentials loads OAuth credentials for a user and service.
func loadCredentials(ctx context.Context, pool *pgxpool.Pool, userID, providerID string) (*Credentials, error) {
	var creds Credentials
	var accessTokenEnc, refreshTokenEnc []byte
	var expiresAt time.Time
	var scopes []string

	err := pool.QueryRow(ctx, `
		SELECT access_token_encrypted, refresh_token_encrypted, token_expires_at, scopes
		FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
	`, userID, providerID).Scan(&accessTokenEnc, &refreshTokenEnc, &expiresAt, &scopes)

	if err != nil {
		return nil, fmt.Errorf("credentials not found for provider %s: %w", providerID, err)
	}

	creds.Provider = providerID
	creds.AccessTokenEncrypted = accessTokenEnc
	creds.RefreshTokenEncrypted = refreshTokenEnc
	creds.ExpiresAt = &expiresAt
	creds.Scopes = scopes

	return &creds, nil
}

// retryWithBackoff executes a function with exponential backoff retry logic.
func retryWithBackoff(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Don't retry on last attempt
		if i == attempts-1 {
			break
		}

		// Exponential backoff: 1s, 2s, 4s
		backoff := time.Duration(1<<uint(i)) * time.Second
		slog.Warn("action failed, retrying", "attempt", i+1, "backoff", backoff, "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return fmt.Errorf("action failed after %d attempts: %w", attempts, err)
}

// getPoolFromContext extracts pgxpool.Pool from execution context.
func getPoolFromContext(ac ActionContext) (*pgxpool.Pool, error) {
	// The pool should be injected into execution context during setup
	pool, ok := ac.Execution.Context["_pool"].(*pgxpool.Pool)
	if !ok || pool == nil {
		return nil, fmt.Errorf("database pool not available in execution context")
	}
	return pool, nil
}

// skillTierFromContext reads the _skill_tier value injected by runExecution.
// Returns TierStructuredAI as a safe default when the value is absent.
func skillTierFromContext(ac ActionContext) SkillTier {
	v, ok := ac.Execution.Context["_skill_tier"]
	if !ok {
		return TierStructuredAI
	}
	switch t := v.(type) {
	case int:
		return SkillTier(t)
	case SkillTier:
		return t
	default:
		return TierStructuredAI
	}
}
