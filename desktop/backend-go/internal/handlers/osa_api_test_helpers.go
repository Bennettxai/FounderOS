package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/require"
)

// createOSATestUser creates a test user and session for OSA API tests
func createOSATestUser(t *testing.T, ctx context.Context, testDB *testutil.TestDatabase) (string, string) {
	t.Helper()

	userID := uuid.New()
	sessionID := uuid.New()

	// Create user
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, username, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
	`, userID, "test-"+userID.String()+"@example.com", "hash", "testuser-"+userID.String()[:8])
	require.NoError(t, err)

	// Create session
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, session_token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '1 day', NOW(), NOW())
	`, sessionID, userID, "token-"+sessionID.String())
	require.NoError(t, err)

	return userID.String(), sessionID.String()
}

// createOSATestWorkspace creates a test workspace for OSA API tests
func createOSATestWorkspace(t *testing.T, ctx context.Context, testDB *testutil.TestDatabase, userIDStr string) uuid.UUID {
	t.Helper()

	userID, err := uuid.Parse(userIDStr)
	require.NoError(t, err)

	workspaceID := uuid.New()

	// Create workspace
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "Test Workspace", userID)
	require.NoError(t, err)

	// Add user as member
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, 'owner', NOW())
	`, workspaceID, userID)
	require.NoError(t, err)

	return workspaceID
}

// createOSATestApp creates a test OSA generated app for API tests
func createOSATestApp(t *testing.T, ctx context.Context, testDB *testutil.TestDatabase, workspaceID uuid.UUID, name string, status string) string {
	t.Helper()

	appID := uuid.New()

	metadata := map[string]interface{}{
		"test": true,
		"name": name,
	}
	metadataJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	// Create app
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO osa_module_instances (
			id, workspace_id, name, display_name, description,
			osa_workflow_id, status, files_created, metadata,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`, appID, workspaceID, name, name, "Test app: "+name,
		"wf-"+uuid.New().String()[:8], status, 5, metadataJSON)
	require.NoError(t, err)

	return appID.String()
}

// createOSATestTemplate creates a test template for API tests
func createOSATestTemplate(t *testing.T, ctx context.Context, testDB *testutil.TestDatabase, workspaceID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	templateID := uuid.New()

	config := map[string]interface{}{
		"framework": "react",
		"database":  "postgresql",
	}
	configJSON, err := json.Marshal(config)
	require.NoError(t, err)

	// Create template
	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO app_templates (
			id, workspace_id, name, description, category,
			config, is_builtin, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, false, NOW(), NOW())
	`, templateID, workspaceID, name, "Test template: "+name, "web",
		configJSON)
	require.NoError(t, err)

	return templateID
}

// createOSABuildEvent creates a test build event for API tests
func createOSABuildEvent(t *testing.T, ctx context.Context, testDB *testutil.TestDatabase, appIDStr string, eventType string, message string) {
	t.Helper()

	appID, err := uuid.Parse(appIDStr)
	require.NoError(t, err)

	eventID := uuid.New()

	eventData := map[string]interface{}{
		"message": message,
	}
	eventDataJSON, err := json.Marshal(eventData)
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO osa_build_events (
			id, app_id, event_type, phase, progress_percent,
			status_message, event_data, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, eventID, appID, eventType, "building", 50, message, eventDataJSON)
	require.NoError(t, err)
}

// setupOSARouter creates a test router with auth for OSA API tests
func setupOSARouter(userID, sessionID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to inject user context
	router.Use(func(c *gin.Context) {
		if userID != "" && sessionID != "" {
			c.Set("user_id", userID)
			c.Set("session_id", sessionID)

			// Parse UUID for cases that need it
			if uid, err := uuid.Parse(userID); err == nil {
				c.Set("user_id_uuid", uid)
			}
		}
		c.Next()
	})

	return router
}



// pgUUIDFromString creates a pgtype.UUID from string
func pgUUIDFromString(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	uid, err := uuid.Parse(s)
	require.NoError(t, err)

	var pgUUID pgtype.UUID
	err = pgUUID.Scan(uid)
	require.NoError(t, err)

	return pgUUID
}
