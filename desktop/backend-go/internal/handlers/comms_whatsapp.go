package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const whatsappProvider = "whatsapp"

type whatsappStore struct {
	path string
}

type whatsappStatus struct {
	ChatCount    int
	MessageCount int
	LastMessage  *time.Time
}

func newWhatsAppStore() *whatsappStore {
	path := strings.TrimSpace(os.Getenv("WHATSAPP_MESSAGES_DB"))
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, ".local", "share", "whatsapp-mcp", "whatsapp-bridge", "store", "messages.db")
		}
	}
	return &whatsappStore{path: path}
}

func (s *whatsappStore) open(ctx context.Context) (*sql.DB, error) {
	if s == nil || s.path == "" {
		return nil, fmt.Errorf("whatsapp database path is not configured")
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return nil, fmt.Errorf("whatsapp database unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("whatsapp database path is not a regular file")
	}

	dsn := (&url.URL{Scheme: "file", Path: s.path}).String() + "?mode=ro&_pragma=query_only%3D1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func (s *whatsappStore) status(ctx context.Context) (whatsappStatus, error) {
	db, err := s.open(ctx)
	if err != nil {
		return whatsappStatus{}, err
	}
	defer db.Close()

	var status whatsappStatus
	var last any
	if err := db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM chats),
			(SELECT COUNT(*) FROM messages),
			(SELECT MAX(timestamp) FROM messages)
	`).Scan(&status.ChatCount, &status.MessageCount, &last); err != nil {
		return whatsappStatus{}, err
	}
	status.LastMessage = parseWhatsAppTime(last)
	return status, nil
}

func (s *whatsappStore) channels(ctx context.Context) ([]unifiedChannel, error) {
	db, err := s.open(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT c.jid, COALESCE(NULLIF(c.name, ''), c.jid), c.last_message_time,
			COUNT(m.id)
		FROM chats c
		LEFT JOIN messages m ON m.chat_jid = c.jid
		GROUP BY c.jid, c.name, c.last_message_time
		ORDER BY c.last_message_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]unifiedChannel, 0)
	for rows.Next() {
		var jid, name string
		var last any
		var messageCount int
		if err := rows.Scan(&jid, &name, &last, &messageCount); err != nil {
			return nil, err
		}
		isDM := !strings.HasSuffix(jid, "@g.us")
		memberCount := 0
		if isDM {
			memberCount = 2
		}
		out = append(out, unifiedChannel{
			ID:            encodeWhatsAppChannelID(jid),
			Provider:      whatsappProvider,
			WorkspaceID:   "whatsapp-local",
			WorkspaceName: "WhatsApp on this Mac",
			ExternalID:    jid,
			Name:          name,
			IsPrivate:     true,
			IsDM:          isDM,
			MemberCount:   memberCount,
			MessageCount:  messageCount,
			ReadOnly:      true,
			LastActivity:  parseWhatsAppTime(last),
		})
	}
	return out, rows.Err()
}

func (s *whatsappStore) messages(ctx context.Context, channelID string, limit int, before *time.Time) ([]unifiedMessage, bool, error) {
	jid, err := decodeWhatsAppChannelID(channelID)
	if err != nil {
		return nil, false, err
	}
	db, err := s.open(ctx)
	if err != nil {
		return nil, false, err
	}
	defer db.Close()

	query := `
		SELECT m.id, COALESCE(m.sender, ''), m.content, m.timestamp, COALESCE(m.is_from_me, 0),
			m.media_type, m.filename, COALESCE(NULLIF(sender_chat.name, ''), m.sender)
		FROM messages m
		LEFT JOIN chats sender_chat ON sender_chat.jid = m.sender
		WHERE m.chat_jid = ?`
	args := []any{jid}
	if before != nil {
		query += " AND julianday(m.timestamp) < julianday(?)"
		args = append(args, before.Format(time.RFC3339Nano))
	}
	query += " ORDER BY m.timestamp DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	out := make([]unifiedMessage, 0, limit+1)
	for rows.Next() {
		var id, sender string
		var content, mediaType, filename, senderName sql.NullString
		var sent any
		var fromMe bool
		if err := rows.Scan(&id, &sender, &content, &sent, &fromMe, &mediaType, &filename, &senderName); err != nil {
			return nil, false, err
		}
		body := strings.TrimSpace(content.String)
		if body == "" && mediaType.String != "" {
			body = "[" + mediaType.String + "]"
			if filename.String != "" {
				body += " " + filename.String
			}
		}
		name := senderName.String
		if fromMe {
			sender = "me"
			name = "You"
		}
		out = append(out, unifiedMessage{
			ID:         encodeWhatsAppMessageID(jid, id),
			Provider:   whatsappProvider,
			ChannelID:  channelID,
			ExternalID: id,
			SenderID:   sender,
			SenderName: name,
			Content:    body,
			SentAt:     parseWhatsAppTime(sent),
		})
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, rows.Err()
}

func encodeWhatsAppChannelID(jid string) string {
	return "wa_" + base64.RawURLEncoding.EncodeToString([]byte(jid))
}

func decodeWhatsAppChannelID(id string) (string, error) {
	if !strings.HasPrefix(id, "wa_") {
		return "", fmt.Errorf("invalid WhatsApp channel id")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, "wa_"))
	if err != nil || len(raw) == 0 {
		return "", fmt.Errorf("invalid WhatsApp channel id")
	}
	return string(raw), nil
}

func encodeWhatsAppMessageID(jid, id string) string {
	return "wam_" + base64.RawURLEncoding.EncodeToString([]byte(jid+"\x00"+id))
}

func parseWhatsAppTime(value any) *time.Time {
	if value == nil {
		return nil
	}
	if parsed, ok := value.(time.Time); ok {
		return &parsed
	}
	raw := strings.TrimSpace(fmt.Sprint(value))
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return &parsed
		}
	}
	return nil
}
