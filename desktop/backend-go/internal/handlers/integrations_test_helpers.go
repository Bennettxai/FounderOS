package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// createIntTestUserWithSession creates a user and session for integration tests
func createIntTestUserWithSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, string) {
	t.Helper()

	userID := uuid.New()
	sessionID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, username, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
	`, userID, "inttest-"+userID.String()+"@example.com", "$2a$10$hashedpassword", "inttest-"+userID.String()[:8])
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, session_token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '1 day', NOW(), NOW())
	`, sessionID, userID, "token-"+sessionID.String())
	require.NoError(t, err)

	return userID.String(), sessionID.String()
}

// createIntTestWorkspace creates a workspace for integration tests
func createIntTestWorkspace(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userIDStr string) uuid.UUID {
	t.Helper()

	userID, err := uuid.Parse(userIDStr)
	require.NoError(t, err)

	workspaceID := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "IntTest Workspace", userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, 'owner', NOW())
	`, workspaceID, userID)
	require.NoError(t, err)

	return workspaceID
}

// createIntTestMemory creates a memory for integration tests
func createIntTestMemory(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workspaceID uuid.UUID, userIDStr, title string) uuid.UUID {
	t.Helper()

	userID, err := uuid.Parse(userIDStr)
	require.NoError(t, err)

	memoryID := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO memories (
			id, workspace_id, user_id, title, content, memory_type,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, memoryID, workspaceID, userID, title, "Test content for "+title, "fact")
	require.NoError(t, err)

	return memoryID
}

// createIntTestConversation creates a conversation for integration tests
func createIntTestConversation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workspaceID uuid.UUID, userIDStr string) uuid.UUID {
	t.Helper()

	userID, err := uuid.Parse(userIDStr)
	require.NoError(t, err)

	conversationID := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO conversations (
			id, workspace_id, user_id, title, message_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, conversationID, workspaceID, userID, "IntTest Conversation", 0)
	require.NoError(t, err)

	return conversationID
}

// createIntTestMessage creates a message for integration tests
func createIntTestMessage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, conversationID uuid.UUID, userIDStr, content string) uuid.UUID {
	t.Helper()

	userID, err := uuid.Parse(userIDStr)
	require.NoError(t, err)

	messageID := uuid.New()

	_, err = pool.Exec(ctx, `
		INSERT INTO conversation_messages (
			id, conversation_id, user_id, role, content, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
	`, messageID, conversationID, userID, "user", content)
	require.NoError(t, err)

	return messageID
}
