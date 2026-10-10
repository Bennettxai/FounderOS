package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// In production, this would use testcontainers, but for now we'll use a simple in-memory approach
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Use DATABASE_TEST_URL environment variable or skip
	dbURL := "postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("Skipping test: cannot ping test database: %v", err)
		return nil
	}

	return pool
}

// cleanupWorkflows removes test workflows from database
func cleanupWorkflows(t *testing.T, pool *pgxpool.Pool, workspaceID uuid.UUID) {
	t.Helper()
	_, _ = pool.Exec(context.Background(),
		"DELETE FROM osa_module_instances WHERE workspace_id = $1", workspaceID)
	_, _ = pool.Exec(context.Background(),
		"DELETE FROM osa_workspaces WHERE id = $1", workspaceID)
}

// createTestWorkspace creates a test workspace in the database
func createTestWorkspace(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) uuid.UUID {
	t.Helper()

	var workspaceID uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO osa_workspaces (user_id, name, workspace_path)
		VALUES ($1, 'test-workspace', '/tmp/test-workspace')
		RETURNING id
	`, userID).Scan(&workspaceID)

	require.NoError(t, err)
	return workspaceID
}

// createTestApp creates a test generated app in the database
func createTestApp(t *testing.T, pool *pgxpool.Pool, workspaceID uuid.UUID, metadata map[string]interface{}) uuid.UUID {
	t.Helper()

	metadataJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	var appID uuid.UUID
	err = pool.QueryRow(context.Background(), `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name, description,
			osa_workflow_id, status, files_created, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, workspaceID, "test-app", "Test App", "Test Description",
		"wf-"+uuid.New().String()[:8], "generated", 3, metadataJSON).Scan(&appID)

	require.NoError(t, err)
	return appID
}

// setupRouter creates a test router with middleware
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}
