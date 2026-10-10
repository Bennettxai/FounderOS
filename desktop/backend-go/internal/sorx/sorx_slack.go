package sorx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/integrations/slack"
)

// ============================================================================
// Slack Actions
// ============================================================================

func slackSendMessage(ctx context.Context, ac ActionContext) (interface{}, error) {
	channel, _ := ac.Params["channel"].(string)
	text, _ := ac.Params["text"].(string)

	if channel == "" || text == "" {
		return nil, fmt.Errorf("channel and text are required")
	}

	slog.Info("slackSendMessage", "user_id", ac.Execution.UserID, "channel", channel)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "slack"); err != nil {
		return nil, fmt.Errorf("Slack not connected: %w", err)
	}

	provider := slack.NewProvider(pool)
	channelService := slack.NewChannelService(provider)

	threadTS, _ := ac.Params["thread_ts"].(string)

	ts, err := channelService.SendMessage(ctx, ac.Execution.UserID, channel, text, threadTS)
	if err != nil {
		return nil, fmt.Errorf("failed to send Slack message: %w", err)
	}

	slog.Info("slackSendMessage complete", "channel", channel, "timestamp", ts)
	return map[string]interface{}{
		"sent":      true,
		"channel":   channel,
		"timestamp": ts,
	}, nil
}

func slackListChannels(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("slackListChannels", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	if _, err := loadCredentials(ctx, pool, ac.Execution.UserID, "slack"); err != nil {
		return nil, fmt.Errorf("Slack not connected: %w", err)
	}

	provider := slack.NewProvider(pool)
	channelService := slack.NewChannelService(provider)

	limit := 100
	if val, ok := ac.Params["limit"].(float64); ok {
		limit = int(val)
	}

	channels, err := channelService.ListChannels(ctx, ac.Execution.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list Slack channels: %w", err)
	}

	slog.Info("slackListChannels complete", "count", len(channels))
	return map[string]interface{}{
		"channels": channels,
		"count":    len(channels),
	}, nil
}
