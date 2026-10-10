package handlers

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/integrations/microsoft"
	"github.com/rhl/businessos-backend/internal/integrations/slack"
	"github.com/rhl/businessos-backend/internal/services"
)

// newGmailEngineHook returns a closure that mirrors saved Gmail emails
// into the OptimalEngine knowledge graph as Signal{Module: "email"}.
// nil when the engine isn't wired so the underlying service stays a
// no-op caller.
//
// Mapping (see docs/COMMUNICATIONS_ENGINE_SYNC.md):
//
//	Title     ← email.Subject (fallback "(no subject)")
//	Body      ← body_text → strip(body_html) → snippet
//	Genre     ← "email"
//	AuthorID  ← BO user (NOT the email sender)
//	Metadata  ← provider, from_email, from_name, thread_id, is_sent, labels
func newGmailEngineHook(sync *services.EngineSync, pool *pgxpool.Pool) google.EmailSavedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, email *google.Email, userID string) {
		if email == nil {
			return
		}
		route, _ := resolveCommunicationRoute(ctx, pool, userID, "gmail", email.ThreadID)
		if route == nil {
			return
		}
		title := email.Subject
		if title == "" {
			title = "(no subject)"
		}

		body := email.BodyText
		if body == "" {
			body = stripHTML(email.BodyHTML)
		}
		if body == "" {
			body = email.Snippet
		}

		meta := map[string]string{
			"provider":      "gmail",
			"from_email":    email.FromEmail,
			"routing_scope": route.Scope,
		}
		if email.FromName != "" {
			meta["from_name"] = email.FromName
		}
		if email.ThreadID != "" {
			meta["thread_id"] = email.ThreadID
		}
		if email.IsSent {
			meta["is_sent"] = "true"
		}
		if len(email.Labels) > 0 {
			meta["labels"] = strings.Join(email.Labels, ",")
		}

		modifiedAt := email.Date
		if modifiedAt.IsZero() {
			modifiedAt = email.ReceivedAt
		}

		sync.Enqueue(ctx, services.Signal{
			Module:      services.ModuleEmail,
			ID:          email.ID,
			AuthorID:    userID,
			WorkspaceID: route.WorkspaceID,
			Title:       title,
			Body:        body,
			Genre:       "email",
			ModifiedAt:  modifiedAt,
			Metadata:    meta,
		})
	}
}

// newOutlookEngineHook returns a closure that mirrors saved Outlook
// emails into the engine as Signal{Module: "email", metadata.provider:
// "outlook"}. Mirror of newGmailEngineHook for the Microsoft data path
// — same module, different concrete row type.
//
// Mapping (see docs/COMMUNICATIONS_ENGINE_SYNC.md):
//
//	Title     ← msg.Subject (fallback "(no subject)")
//	Body      ← body_content (text) → strip(body_content) (html) → body_preview
//	Genre     ← "email"
//	AuthorID  ← BO user (NOT the email sender)
//	Metadata  ← provider="outlook", from_email, from_name, thread_id (conversation_id),
//	            categories (comma-joined), importance
func newOutlookEngineHook(sync *services.EngineSync, pool *pgxpool.Pool) microsoft.EmailSavedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, msg *microsoft.OutlookMessage, userID string) {
		if msg == nil {
			return
		}
		route, _ := resolveCommunicationRoute(ctx, pool, userID, "outlook", msg.ConversationID)
		if route == nil {
			return
		}
		title := msg.Subject
		if title == "" {
			title = "(no subject)"
		}

		// Derive body — prefer plain text, fall back to stripped HTML, then
		// preview. Mirrors the Gmail hook's fallback chain.
		body := ""
		if msg.BodyContent != "" {
			if strings.EqualFold(msg.BodyContentType, "html") {
				body = stripHTML(msg.BodyContent)
			} else {
				body = msg.BodyContent
			}
		}
		if body == "" {
			body = msg.BodyPreview
		}

		meta := map[string]string{
			"provider":      "outlook",
			"from_email":    msg.FromEmail,
			"routing_scope": route.Scope,
		}
		if msg.FromName != "" {
			meta["from_name"] = msg.FromName
		}
		if msg.ConversationID != "" {
			meta["thread_id"] = msg.ConversationID
		}
		if msg.Importance != "" {
			meta["importance"] = msg.Importance
		}
		if len(msg.Categories) > 0 {
			meta["labels"] = strings.Join(msg.Categories, ",")
		}

		modifiedAt := msg.ReceivedDateTime
		if modifiedAt.IsZero() {
			modifiedAt = msg.SentDateTime
		}

		sync.Enqueue(ctx, services.Signal{
			Module:      services.ModuleEmail,
			ID:          msg.ID,
			AuthorID:    userID,
			WorkspaceID: route.WorkspaceID,
			Title:       title,
			Body:        body,
			Genre:       "email",
			ModifiedAt:  modifiedAt,
			Metadata:    meta,
		})
	}
}

// newSlackMessageEngineHook returns a closure that mirrors saved Slack
// messages into the engine as Signal{Module: "message"}.
//
// Mapping:
//
//	Title     ← first 80 chars of content (whitespace collapsed),
//	            fallback "<sender> in channel"
//	Body      ← full content
//	Genre     ← "message"
//	AuthorID  ← BO user (NOT the Slack sender)
//	Metadata  ← provider, channel_id, sender_id, sender_name, thread_ts, slack_ts
func newSlackMessageEngineHook(sync *services.EngineSync, pool *pgxpool.Pool) slack.MessageSavedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, msg *slack.Message, userID string) {
		if msg == nil {
			return
		}
		route, _ := resolveCommunicationRoute(ctx, pool, userID, "slack", msg.ChannelID)
		if route == nil {
			return
		}

		title := truncateForTitle(msg.Content, 80)
		if title == "" {
			sender := msg.SenderName
			if sender == "" {
				sender = msg.SenderID
			}
			if sender == "" {
				sender = "unknown"
			}
			title = sender + " in channel"
		}

		meta := map[string]string{
			"provider":      "slack",
			"channel_id":    msg.ChannelID,
			"slack_ts":      msg.SlackTS,
			"routing_scope": route.Scope,
		}
		if msg.SenderID != "" {
			meta["sender_id"] = msg.SenderID
		}
		if msg.SenderName != "" {
			meta["sender_name"] = msg.SenderName
		}
		if msg.ThreadTS != "" {
			meta["thread_ts"] = msg.ThreadTS
		}

		sync.Enqueue(ctx, services.Signal{
			Module:      services.ModuleMessage,
			ID:          msg.ID,
			AuthorID:    userID,
			WorkspaceID: route.WorkspaceID,
			Title:       title,
			Body:        msg.Content,
			Genre:       "message",
			ModifiedAt:  msg.SentAt,
			Metadata:    meta,
		})
	}
}

// newTeamsMessageEngineHook returns a closure that mirrors saved Teams
// messages into the engine as Signal{Module: "message", metadata.provider:
// "teams"}. Mirror of newSlackMessageEngineHook for the Microsoft data
// path — same module, different concrete row type.
//
// Mapping (see docs/COMMUNICATIONS_ENGINE_SYNC.md):
//
//	Title     ← first 80 chars of body_content (whitespace collapsed),
//	            fallback "<sender> in channel"
//	Body      ← body_content (text) → strip(body_content) (html) → subject
//	Genre     ← "message"
//	AuthorID  ← BO user (NOT the Teams sender)
//	Metadata  ← provider="teams", team_id, channel_id, sender_id, sender_name,
//	            reply_to_id (when this is a thread reply), message_id (Graph)
func newTeamsMessageEngineHook(sync *services.EngineSync, pool *pgxpool.Pool) microsoft.MessageSavedHook {
	if sync == nil {
		return nil
	}
	return func(ctx context.Context, msg *microsoft.TeamsMessage, userID string) {
		if msg == nil {
			return
		}
		conversationID := msg.ChannelID
		if pool != nil {
			_ = pool.QueryRow(ctx, `SELECT id::text FROM microsoft_teams_channels
				WHERE user_id=$1 AND team_id=$2 AND channel_id=$3`, userID, msg.TeamID, msg.ChannelID).Scan(&conversationID)
		}
		route, _ := resolveCommunicationRoute(ctx, pool, userID, "teams", conversationID)
		if route == nil {
			return
		}

		// Body — prefer plain text, fall back to stripped HTML.
		body := ""
		if msg.BodyContent != "" {
			if strings.EqualFold(msg.BodyContentType, "html") {
				body = stripHTML(msg.BodyContent)
			} else {
				body = msg.BodyContent
			}
		}
		if body == "" {
			body = msg.Subject
		}

		title := truncateForTitle(body, 80)
		if title == "" {
			sender := msg.SenderName
			if sender == "" {
				sender = msg.SenderID
			}
			if sender == "" {
				sender = "unknown"
			}
			title = sender + " in channel"
		}

		meta := map[string]string{
			"provider":      "teams",
			"team_id":       msg.TeamID,
			"channel_id":    msg.ChannelID,
			"message_id":    msg.MessageID,
			"routing_scope": route.Scope,
		}
		if msg.SenderID != "" {
			meta["sender_id"] = msg.SenderID
		}
		if msg.SenderName != "" {
			meta["sender_name"] = msg.SenderName
		}
		if msg.ReplyToID != "" {
			meta["reply_to_id"] = msg.ReplyToID
		}

		modifiedAt := time.Time{}
		if msg.CreatedDateTime != nil {
			modifiedAt = *msg.CreatedDateTime
		}

		sync.Enqueue(ctx, services.Signal{
			Module:      services.ModuleMessage,
			ID:          msg.ID,
			AuthorID:    userID,
			WorkspaceID: route.WorkspaceID,
			Title:       title,
			Body:        body,
			Genre:       "message",
			ModifiedAt:  modifiedAt,
			Metadata:    meta,
		})
	}
}

// truncateForTitle collapses whitespace and trims to maxLen runes.
// Engine titles are short identifiers, not the full message body.
func truncateForTitle(s string, maxLen int) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if unicode.IsSpace(r) {
			if prevSpace {
				continue
			}
			b.WriteRune(' ')
			prevSpace = true
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	out := strings.TrimSpace(b.String())
	if len([]rune(out)) <= maxLen {
		return out
	}
	rs := []rune(out)
	return strings.TrimSpace(string(rs[:maxLen])) + "…"
}

// stripHTML is a minimal tag stripper for the engine indexing fallback
// when an email has only body_html. It removes tags and collapses
// whitespace; the engine's own classifier does proper HTML processing,
// this is just to give it readable plain text.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
