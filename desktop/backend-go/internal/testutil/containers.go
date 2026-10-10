package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TestDatabase represents a test database instance
type TestDatabase struct {
	Pool    *pgxpool.Pool
	cleanup func()
}

// TestRedis represents a test Redis instance
type TestRedis struct {
	Client *redis.Client
	Server *miniredis.Miniredis
}

// SetupTestDatabase creates a PostgreSQL connection with migrations applied
// Requires TEST_DATABASE_URL environment variable or uses default test database
// Returns pool, cleanup function, and error
// Example:
//
//	db, err := testutil.SetupTestDatabase(t)
//	require.NoError(t, err)
//	defer db.Close()
func SetupTestDatabase(t *testing.T) (*TestDatabase, error) {
	// Skip if testing.Short() is enabled
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Get database URL from environment or use default test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable"
	}

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w (ensure TEST_DATABASE_URL is set or test database is running)", err)
	}

	// Apply migrations
	if err := applyMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Create cleanup function
	cleanup := func() {
		pool.Close()
	}

	return &TestDatabase{
		Pool:    pool,
		cleanup: cleanup,
	}, nil
}

// Close cleans up the test database
func (td *TestDatabase) Close() {
	if td.cleanup != nil {
		td.cleanup()
	}
}

// SetupTestRedis creates an in-memory Redis instance using miniredis
// Example:
//
//	redis, err := testutil.SetupTestRedis(t)
//	require.NoError(t, err)
//	defer redis.Close()
func SetupTestRedis(t *testing.T) (*TestRedis, error) {
	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start miniredis: %w", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Verify connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		mr.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &TestRedis{
		Client: client,
		Server: mr,
	}, nil
}

// Close cleans up the test Redis instance
func (tr *TestRedis) Close() {
	if tr.Client != nil {
		tr.Client.Close()
	}
	if tr.Server != nil {
		tr.Server.Close()
	}
}

// applyMigrations builds the canonical schema, then applies post-baseline
// repairs. This mirrors the production runner without importing database,
// which would create an import cycle in database package tests.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	schemaPath := filepath.Join(filepath.Dir(migrationsDir), "schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read canonical schema: %w", err)
	}
	if _, err := pool.Exec(ctx, string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to apply canonical schema: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		if entry.Name() <= "157_communication_routes.sql" {
			continue
		}

		migrationPath := filepath.Join(migrationsDir, entry.Name())
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", entry.Name(), err)
		}

		slog.Debug("Applied migration", "file", entry.Name())
	}

	return nil
}

// findMigrationsDir finds the migrations directory relative to the current working directory
func findMigrationsDir() string {
	// Try common paths
	candidates := []string{
		"internal/database/migrations",
		"../database/migrations",
		"../../database/migrations",
		"../../../database/migrations",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

// CleanupTestData removes all test data from the database
// This is useful for cleaning up between test runs
func CleanupTestData(ctx context.Context, pool *pgxpool.Pool) error {
	// Truncate all tables in reverse dependency order
	tables := []string{
		"module_shares",
		"module_installations",
		"custom_module_versions",
		"custom_modules",
		"conversation_messages",
		"conversations",
		"memories",
		"workspace_members",
		"workspaces",
		"sessions",
		`"user"`,
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Ignore errors for tables that don't exist
			slog.Debug("Failed to truncate table", "table", table, "error", err)
		}
	}

	return nil
}

// RequireTestDatabase is a helper that sets up a test database or skips the test
// Example:
//
//	db := testutil.RequireTestDatabase(t)
//	defer db.Close()
func RequireTestDatabase(t *testing.T) *TestDatabase {
	db, err := SetupTestDatabase(t)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	return db
}

// RequireTestRedis is a helper that sets up test Redis or skips the test
// Example:
//
//	redis := testutil.RequireTestRedis(t)
//	defer redis.Close()
func RequireTestRedis(t *testing.T) *TestRedis {
	redis, err := SetupTestRedis(t)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	return redis
}
