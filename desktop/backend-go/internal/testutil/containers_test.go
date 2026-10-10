package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupTestDatabase tests the database setup
// This test requires a running PostgreSQL test database
// Set TEST_DATABASE_URL to point to your test database
func TestSetupTestDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if test database is configured
	if os.Getenv("TEST_DATABASE_URL") == "" && !isDatabaseAvailable() {
		t.Skip("Test database not available - set TEST_DATABASE_URL or run local postgres on default port")
	}

	db, err := SetupTestDatabase(t)
	require.NoError(t, err)
	defer db.Close()

	// Verify connection
	ctx := context.Background()
	err = db.Pool.Ping(ctx)
	require.NoError(t, err)

	// Verify we can execute queries
	var version string
	err = db.Pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	require.NoError(t, err)
	assert.Contains(t, version, "PostgreSQL")

	// Verify migrations were applied by checking for the Better Auth user table.
	var tableExists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'user'
		)
	`).Scan(&tableExists)
	require.NoError(t, err)
	assert.True(t, tableExists, "user table should exist after migrations")
}

// TestSetupTestRedis tests the Redis setup
func TestSetupTestRedis(t *testing.T) {
	redis, err := SetupTestRedis(t)
	require.NoError(t, err)
	defer redis.Close()

	// Verify connection
	ctx := context.Background()
	err = redis.Client.Ping(ctx).Err()
	require.NoError(t, err)

	// Test basic operations
	err = redis.Client.Set(ctx, "test_key", "test_value", 0).Err()
	require.NoError(t, err)

	val, err := redis.Client.Get(ctx, "test_key").Result()
	require.NoError(t, err)
	assert.Equal(t, "test_value", val)
}

// TestRequireTestDatabase tests the helper function
func TestRequireTestDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("TEST_DATABASE_URL") == "" && !isDatabaseAvailable() {
		t.Skip("Test database not available")
	}

	db := RequireTestDatabase(t)
	defer db.Close()

	// Should have a valid pool
	assert.NotNil(t, db.Pool)

	// Should be able to ping
	ctx := context.Background()
	err := db.Pool.Ping(ctx)
	assert.NoError(t, err)
}

// TestRequireTestRedis tests the helper function
func TestRequireTestRedis(t *testing.T) {
	redis := RequireTestRedis(t)
	defer redis.Close()

	// Should have a valid client
	assert.NotNil(t, redis.Client)

	// Should be able to ping
	ctx := context.Background()
	err := redis.Client.Ping(ctx).Err()
	assert.NoError(t, err)
}

// TestCleanupTestData tests the cleanup function
func TestCleanupTestData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("TEST_DATABASE_URL") == "" && !isDatabaseAvailable() {
		t.Skip("Test database not available")
	}

	db := RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Insert some test data
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO "user" (id, email, name, "createdAt", "updatedAt")
		VALUES ('test-user-cleanup', 'cleanup@example.com', 'Cleanup User', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err)

	// Verify data exists
	var count int
	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM "user" WHERE id = 'test-user-cleanup'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Cleanup
	err = CleanupTestData(ctx, db.Pool)
	require.NoError(t, err)

	// Verify data is gone
	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM "user" WHERE id = 'test-user-cleanup'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// isDatabaseAvailable checks if the default test database is available
func isDatabaseAvailable() bool {
	// Try to connect to default test database
	ctx := context.Background()
	dbURL := "postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable"

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return false
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return false
	}

	return true
}
