package sorx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/integrations/google"
	"google.golang.org/api/gmail/v1"
)

// ============================================================================
// Gmail Actions
// ============================================================================

func gmailListMessages(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("gmailListMessages", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	maxResults := int64(50)
	if val, ok := ac.Params["max_results"].(float64); ok {
		maxResults = int64(val)
	}

	// Create Gmail API client
	provider := google.NewProvider(pool, nil) // oauthConfig can be nil for token-based access
	gmailSvc := google.NewGmailService(provider)
	srv, err := gmailSvc.GetGmailAPI(ctx, ac.Execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	var messages []*gmail.Message
	err = retryWithBackoff(ctx, 3, func() error {
		req := srv.Users.Messages.List("me").MaxResults(maxResults)
		if query, ok := ac.Params["query"].(string); ok && query != "" {
			req = req.Q(query)
		}

		resp, err := req.Do()
		if err != nil {
			return err
		}
		messages = resp.Messages
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Convert to simple format
	results := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		results = append(results, map[string]interface{}{
			"id":       msg.Id,
			"threadId": msg.ThreadId,
			"snippet":  msg.Snippet,
		})
	}

	slog.Info("gmailListMessages success", "count", len(results))
	return map[string]interface{}{
		"messages": results,
		"count":    len(results),
		"provider": creds.Provider,
	}, nil
}

func gmailSendEmail(ctx context.Context, ac ActionContext) (interface{}, error) {
	to, _ := ac.Params["to"].(string)
	subject, _ := ac.Params["subject"].(string)
	body, _ := ac.Params["body"].(string)

	if to == "" || subject == "" {
		return nil, fmt.Errorf("to and subject are required")
	}

	slog.Info("gmailSendEmail", "user_id", ac.Execution.UserID, "to", to, "subject", subject)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	provider := google.NewProvider(pool, nil)
	gmailSvc := google.NewGmailService(provider)

	email := &google.ComposeEmail{
		To:      []string{to},
		Subject: subject,
		Body:    body,
		IsHTML:  false,
	}

	err = retryWithBackoff(ctx, 3, func() error {
		return gmailSvc.SendEmail(ctx, ac.Execution.UserID, email)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("gmailSendEmail success", "to", to)
	return map[string]interface{}{
		"sent":     true,
		"to":       to,
		"subject":  subject,
		"provider": creds.Provider,
	}, nil
}

func gmailSearch(ctx context.Context, ac ActionContext) (interface{}, error) {
	query, _ := ac.Params["query"].(string)

	slog.Info("gmailSearch", "user_id", ac.Execution.UserID, "query", query)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to load Google credentials: %w", err)
	}

	provider := google.NewProvider(pool, nil)
	gmailSvc := google.NewGmailService(provider)
	srv, err := gmailSvc.GetGmailAPI(ctx, ac.Execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	var messages []*gmail.Message
	err = retryWithBackoff(ctx, 3, func() error {
		resp, err := srv.Users.Messages.List("me").Q(query).MaxResults(50).Do()
		if err != nil {
			return err
		}
		messages = resp.Messages
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		results = append(results, map[string]interface{}{
			"id":      msg.Id,
			"snippet": msg.Snippet,
		})
	}

	slog.Info("gmailSearch success", "count", len(results))
	return map[string]interface{}{
		"query":    query,
		"messages": results,
		"count":    len(results),
		"provider": creds.Provider,
	}, nil
}
