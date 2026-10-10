package handlers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createWhatsAppFixture(t *testing.T, statements string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "messages.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE chats (jid TEXT PRIMARY KEY, name TEXT, last_message_time TIMESTAMP);
		CREATE TABLE messages (
			id TEXT, chat_jid TEXT, sender TEXT, content TEXT, timestamp TIMESTAMP,
			is_from_me BOOLEAN, media_type TEXT, filename TEXT,
			PRIMARY KEY (id, chat_jid)
		);
	` + statements)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	return path
}

func TestWhatsAppStoreReadsCanonicalHistoryWithoutMutatingIt(t *testing.T) {
	path := createWhatsAppFixture(t, `
		INSERT INTO chats VALUES
			('person@s.whatsapp.net', 'Greice', '2026-07-20 10:30:00-05:00'),
			('team@g.us', 'Delivery team', '2026-07-20 10:31:00-05:00');
		INSERT INTO messages VALUES
			('m1', 'person@s.whatsapp.net', 'person@s.whatsapp.net', 'Can we meet Tuesday?', '2026-07-20 10:30:00-05:00', 0, '', ''),
			('m2', 'team@g.us', 'me@s.whatsapp.net', 'I will review it.', '2026-07-20 10:31:00-05:00', 1, '', '');
	`)

	store := &whatsappStore{path: path}
	ctx := context.Background()
	status, err := store.status(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, status.ChatCount)
	require.Equal(t, 2, status.MessageCount)
	require.NotNil(t, status.LastMessage)

	channels, err := store.channels(ctx)
	require.NoError(t, err)
	require.Len(t, channels, 2)
	require.Equal(t, whatsappProvider, channels[0].Provider)
	require.True(t, channels[0].ReadOnly)
	require.False(t, channels[0].IsDM)
	require.Equal(t, "Delivery team", channels[0].Name)

	dmID := encodeWhatsAppChannelID("person@s.whatsapp.net")
	messages, more, err := store.messages(ctx, dmID, 50, nil)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, messages, 1)
	require.Equal(t, "Can we meet Tuesday?", messages[0].Content)
	require.Equal(t, "Greice", messages[0].SenderName)

	readOnly, err := store.open(ctx)
	require.NoError(t, err)
	_, err = readOnly.ExecContext(ctx, `INSERT INTO chats (jid, name) VALUES ('forbidden', 'Forbidden')`)
	require.Error(t, err)
	require.NoError(t, readOnly.Close())
}

func TestWhatsAppMessagesSupportsBeforePagination(t *testing.T) {
	path := createWhatsAppFixture(t, `
		INSERT INTO chats VALUES ('person@s.whatsapp.net', 'Person', '2026-07-20 10:31:00-05:00');
		INSERT INTO messages VALUES
			('m1', 'person@s.whatsapp.net', 'person@s.whatsapp.net', 'Older', '2026-07-20 10:30:00-05:00', 0, '', ''),
			('m2', 'person@s.whatsapp.net', 'person@s.whatsapp.net', 'Newer', '2026-07-20 10:31:00-05:00', 0, '', '');
	`)

	store := &whatsappStore{path: path}
	before := time.Date(2026, 7, 20, 10, 31, 0, 0, time.FixedZone("CDT", -5*60*60))
	messages, more, err := store.messages(context.Background(), encodeWhatsAppChannelID("person@s.whatsapp.net"), 1, &before)
	require.NoError(t, err)
	require.False(t, more)
	require.Len(t, messages, 1)
	require.Equal(t, "Older", messages[0].Content)
}

func TestWhatsAppCanonicalStoreIntegration(t *testing.T) {
	path := os.Getenv("WHATSAPP_INTEGRATION_TEST_DB")
	if path == "" {
		t.Skip("WHATSAPP_INTEGRATION_TEST_DB is not set")
	}
	store := &whatsappStore{path: path}
	status, err := store.status(context.Background())
	require.NoError(t, err)
	require.Positive(t, status.ChatCount)
	require.Positive(t, status.MessageCount)

	channels, err := store.channels(context.Background())
	require.NoError(t, err)
	require.Len(t, channels, status.ChatCount)
	require.True(t, channels[0].ReadOnly)
}
