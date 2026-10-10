package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrTokenExpired indicates that the access token has expired
var ErrTokenExpired = errors.New("token expired")

// GetSlackTools returns the MCP tool definitions for Slack integration
func GetSlackTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "slack_list_channels",
			Description: "List Slack channels the bot has access to. Returns channel names, IDs, and member counts.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of channels to return (default 20, max 100)",
					},
					"types": map[string]interface{}{
						"type":        "string",
						"description": "Channel types to include: 'public', 'private', or 'all' (default 'all')",
						"enum":        []string{"public", "private", "all"},
					},
				},
			},
			Source: "builtin",
		},
		{
			Name:        "slack_send_message",
			Description: "Send a message to a Slack channel or direct message. Can also reply to threads.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel": map[string]interface{}{
						"type":        "string",
						"description": "Channel ID or name (e.g., 'C1234567890' or '#general'). Required.",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Message text to send. Supports Slack markdown formatting. Required.",
					},
					"thread_ts": map[string]interface{}{
						"type":        "string",
						"description": "Thread timestamp to reply to. If provided, message will be posted as a thread reply.",
					},
				},
				"required": []string{"channel", "text"},
			},
			Source: "builtin",
		},
		{
			Name:        "slack_get_channel_history",
			Description: "Get recent messages from a Slack channel. Returns message text, authors, and timestamps.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel": map[string]interface{}{
						"type":        "string",
						"description": "Channel ID (e.g., 'C1234567890'). Required.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of messages to retrieve (default 10, max 100)",
					},
				},
				"required": []string{"channel"},
			},
			Source: "builtin",
		},
		{
			Name:        "slack_search_messages",
			Description: "Search for messages across the Slack workspace. Requires user authorization with search:read scope.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query. Supports Slack search modifiers like 'from:@user', 'in:#channel', 'has:link'. Required.",
					},
					"count": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return (default 10, max 100)",
					},
				},
				"required": []string{"query"},
			},
			Source: "builtin",
		},
		{
			Name:        "slack_list_users",
			Description: "List users in the Slack workspace. Returns names, emails, and status.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of users to return (default 20, max 100)",
					},
				},
			},
			Source: "builtin",
		},
		{
			Name:        "slack_get_user_info",
			Description: "Get detailed information about a specific Slack user by their user ID.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "Slack user ID (e.g., 'U1234567890'). Required.",
					},
				},
				"required": []string{"user_id"},
			},
			Source: "builtin",
		},
	}
}

// IsSlackTool checks if a tool name is a Slack tool
func IsSlackTool(toolName string) bool {
	return strings.HasPrefix(toolName, "slack_")
}

// SlackChannelForAI represents a channel in AI-friendly format
type SlackChannelForAI struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsPrivate   bool   `json:"is_private"`
	MemberCount int    `json:"member_count"`
	Topic       string `json:"topic,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
}

// SlackMessageForAI represents a message in AI-friendly format
type SlackMessageForAI struct {
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	UserName  string `json:"user_name,omitempty"`
	Text      string `json:"text"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	ReplyCount int   `json:"reply_count,omitempty"`
}

// SlackUserForAI represents a user in AI-friendly format
type SlackUserForAI struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Email    string `json:"email,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	IsAdmin  bool   `json:"is_admin"`
}

// ExecuteSlackTool executes a Slack MCP tool
func (m *MCPService) ExecuteSlackTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	if m.slackService == nil {
		return nil, fmt.Errorf("Slack is not configured. Please connect your Slack workspace first.")
	}

	switch toolName {
	case "slack_list_channels":
		limit := 20
		if l, ok := arguments["limit"].(float64); ok {
			limit = int(l)
			if limit > 100 {
				limit = 100
			}
		}

		channels, err := m.slackService.ListChannels(ctx, m.userID, limit)
		if err != nil {
			// Check for token expiration using proper sentinel error
			if errors.Is(err, ErrTokenExpired) {
				return nil, fmt.Errorf("Slack access expired. Please reconnect your Slack workspace.")
			}
			return nil, fmt.Errorf("failed to list channels: %w", err)
		}

		// Format for AI
		formatted := make([]SlackChannelForAI, len(channels))
		for i, ch := range channels {
			formatted[i] = SlackChannelForAI{
				ID:          ch.ID,
				Name:        ch.Name,
				IsPrivate:   ch.IsPrivate,
				MemberCount: ch.NumMembers,
				Topic:       ch.Topic.Value,
				Purpose:     ch.Purpose.Value,
			}
		}

		return map[string]interface{}{
			"channels": formatted,
			"count":    len(formatted),
		}, nil

	case "slack_send_message":
		channel, _ := arguments["channel"].(string)
		text, _ := arguments["text"].(string)
		threadTS, _ := arguments["thread_ts"].(string)

		if channel == "" || text == "" {
			return nil, fmt.Errorf("channel and text are required")
		}

		timestamp, err := m.slackService.SendMessage(ctx, m.userID, channel, text, threadTS)
		if err != nil {
			return nil, fmt.Errorf("failed to send message: %w", err)
		}

		response := map[string]interface{}{
			"success":   true,
			"channel":   channel,
			"timestamp": timestamp,
			"message":   "Message sent successfully",
		}

		if threadTS != "" {
			response["thread_ts"] = threadTS
			response["is_reply"] = true
		}

		return response, nil

	case "slack_get_channel_history":
		channel, _ := arguments["channel"].(string)
		if channel == "" {
			return nil, fmt.Errorf("channel is required")
		}

		limit := 10
		if l, ok := arguments["limit"].(float64); ok {
			limit = int(l)
			if limit > 100 {
				limit = 100
			}
		}

		messages, err := m.slackService.GetChannelHistory(ctx, m.userID, channel, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get channel history: %w", err)
		}

		// Format for AI
		formatted := make([]SlackMessageForAI, len(messages))
		for i, msg := range messages {
			formatted[i] = SlackMessageForAI{
				Timestamp:  msg.Timestamp,
				User:       msg.User,
				Text:       msg.Text,
				ThreadTS:   msg.ThreadTimestamp,
				ReplyCount: msg.ReplyCount,
			}
		}

		return map[string]interface{}{
			"messages": formatted,
			"count":    len(formatted),
			"channel":  channel,
		}, nil

	case "slack_search_messages":
		query, _ := arguments["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query is required")
		}

		count := 10
		if c, ok := arguments["count"].(float64); ok {
			count = int(c)
			if count > 100 {
				count = 100
			}
		}

		results, err := m.slackService.SearchMessages(ctx, m.userID, query, count)
		if err != nil {
			return nil, fmt.Errorf("failed to search messages: %w", err)
		}

		// Format matches for AI
		var formatted []SlackMessageForAI
		for _, match := range results.Matches {
			formatted = append(formatted, SlackMessageForAI{
				Timestamp: match.Timestamp,
				User:      match.User,
				UserName:  match.Username,
				Text:      match.Text,
			})
		}

		return map[string]interface{}{
			"matches": formatted,
			"total":   results.Total,
			"query":   query,
		}, nil

	case "slack_list_users":
		limit := 20
		if l, ok := arguments["limit"].(float64); ok {
			limit = int(l)
			if limit > 100 {
				limit = 100
			}
		}

		users, err := m.slackService.ListUsers(ctx, m.userID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}

		// Format for AI
		formatted := make([]SlackUserForAI, len(users))
		for i, u := range users {
			formatted[i] = SlackUserForAI{
				ID:       u.ID,
				Name:     u.Name,
				RealName: u.RealName,
				Email:    u.Profile.Email,
				Title:    u.Profile.Title,
				Status:   u.Profile.StatusText,
				IsAdmin:  u.IsAdmin,
			}
		}

		return map[string]interface{}{
			"users": formatted,
			"count": len(formatted),
		}, nil

	case "slack_get_user_info":
		slackUserID, _ := arguments["user_id"].(string)
		if slackUserID == "" {
			return nil, fmt.Errorf("user_id is required")
		}

		user, err := m.slackService.GetUserInfo(ctx, m.userID, slackUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}

		return SlackUserForAI{
			ID:       user.ID,
			Name:     user.Name,
			RealName: user.RealName,
			Email:    user.Profile.Email,
			Title:    user.Profile.Title,
			Status:   user.Profile.StatusText,
			IsAdmin:  user.IsAdmin,
		}, nil

	default:
		return nil, fmt.Errorf("unknown slack tool: %s", toolName)
	}
}
