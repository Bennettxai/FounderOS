package sorx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/integrations/hubspot"
)

// ============================================================================
// HubSpot Actions
// ============================================================================

func hubspotListContacts(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("hubspotListContacts", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "hubspot")
	if err != nil {
		return nil, fmt.Errorf("HubSpot not connected: %w", err)
	}

	provider := hubspot.NewProvider(pool)

	limit := int32(10)
	if val, ok := ac.Params["limit"].(float64); ok {
		limit = int32(val)
	}

	var contactCount int
	err = retryWithBackoff(ctx, 3, func() error {
		// Use HubSpot provider to fetch contacts
		contacts, err := provider.GetContacts(ctx, ac.Execution.UserID, limit, 0)
		if err != nil {
			return err
		}
		contactCount = len(contacts)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}

	slog.Info("hubspotListContacts success", "count", contactCount)
	return map[string]interface{}{
		"contacts": []interface{}{}, // TODO: Convert contacts to generic interface
		"count":    contactCount,
		"provider": creds.Provider,
	}, nil
}

func hubspotCreateContact(ctx context.Context, ac ActionContext) (interface{}, error) {
	email, _ := ac.Params["email"].(string)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	slog.Info("hubspotCreateContact", "user_id", ac.Execution.UserID, "email", email)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	creds, err := loadCredentials(ctx, pool, ac.Execution.UserID, "hubspot")
	if err != nil {
		return nil, fmt.Errorf("HubSpot not connected: %w", err)
	}

	provider := hubspot.NewProvider(pool)

	firstName, _ := ac.Params["first_name"].(string)
	lastName, _ := ac.Params["last_name"].(string)

	err = retryWithBackoff(ctx, 3, func() error {
		return provider.CreateContact(ctx, ac.Execution.UserID, email, firstName, lastName)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	slog.Info("hubspotCreateContact success", "email", email)
	return map[string]interface{}{
		"created":  true,
		"email":    email,
		"provider": creds.Provider,
	}, nil
}
