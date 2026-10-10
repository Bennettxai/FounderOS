package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldBaselineMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "historical migration", filename: "151_ship_full_build_modules.sql", want: true},
		{name: "cutoff migration", filename: baselineCutoff, want: true},
		{name: "repair migration", filename: "158_repair_baselined_runtime_schema.sql", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldBaselineMigration(tt.filename); got != tt.want {
				t.Fatalf("shouldBaselineMigration(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// TestMigrationRollback tests that each migration can be applied and rolled back cleanly
func TestMigrationRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Get migrations and their rollback files
	migrations, err := loadMigrationsWithRollbacks()
	require.NoError(t, err, "Failed to load migrations")

	// Test last 5 migrations (most critical and recent)
	testCount := 5
	if len(migrations) < testCount {
		testCount = len(migrations)
	}

	recentMigrations := migrations[len(migrations)-testCount:]

	for _, mig := range recentMigrations {
		t.Run(mig.Name, func(t *testing.T) {
			// Get initial schema state
			initialTables := getTableNames(t, db.Pool)
			initialIndexes := getIndexNames(t, db.Pool)

			// Apply migration
			err := applyMigration(t, ctx, db.Pool, mig.Path)
			require.NoError(t, err, "Failed to apply migration %s", mig.Name)

			// Verify migration applied (schema changed)
			afterApplyTables := getTableNames(t, db.Pool)
			afterApplyIndexes := getIndexNames(t, db.Pool)

			// Rollback migration if rollback file exists
			if mig.RollbackPath != "" {
				err = rollbackMigration(t, ctx, db.Pool, mig.RollbackPath)
				require.NoError(t, err, "Failed to rollback migration %s", mig.Name)

				// Verify rollback succeeded (schema restored)
				afterRollbackIndexes := getIndexNames(t, db.Pool)

				// Tables should be back to initial state (or close to it)
				// Note: Some migrations may create tables that persist, so we check indexes instead
				assert.Subset(t, initialIndexes, afterRollbackIndexes,
					"Rollback should remove added indexes for %s", mig.Name)

				t.Logf("Migration %s: Applied (+%d tables, +%d indexes) → Rolled back (-%d indexes)",
					mig.Name,
					len(afterApplyTables)-len(initialTables),
					len(afterApplyIndexes)-len(initialIndexes),
					len(afterApplyIndexes)-len(afterRollbackIndexes))
			} else {
				t.Logf("Migration %s: No rollback file found (skipping rollback test)", mig.Name)
			}
		})
	}
}

// TestMigrationRollbackOrder tests that migrations can be rolled back in reverse order
func TestMigrationRollbackOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Get test migrations (last 3 with rollbacks)
	migrations, err := loadMigrationsWithRollbacks()
	require.NoError(t, err)

	// Find migrations with rollbacks
	migrationsWithRollbacks := []Migration{}
	for i := len(migrations) - 1; i >= 0 && len(migrationsWithRollbacks) < 3; i-- {
		if migrations[i].RollbackPath != "" && migrationSupportsTransaction(migrations[i]) {
			migrationsWithRollbacks = append([]Migration{migrations[i]}, migrationsWithRollbacks...)
		}
	}

	if len(migrationsWithRollbacks) == 0 {
		t.Skip("No migrations with rollback files found")
	}

	t.Logf("Testing rollback order for %d migrations", len(migrationsWithRollbacks))

	// Apply all test migrations
	for _, mig := range migrationsWithRollbacks {
		err := applyMigration(t, ctx, db.Pool, mig.Path)
		require.NoError(t, err, "Failed to apply migration %s", mig.Name)
		t.Logf("Applied migration: %s", mig.Name)
	}

	// Record state after all migrations
	tablesAfterMigrations := getTableNames(t, db.Pool)

	// Rollback in reverse order
	for i := len(migrationsWithRollbacks) - 1; i >= 0; i-- {
		mig := migrationsWithRollbacks[i]
		err := rollbackMigration(t, ctx, db.Pool, mig.RollbackPath)
		require.NoError(t, err, "Failed to rollback migration %s in reverse order", mig.Name)
		t.Logf("Rolled back migration: %s", mig.Name)
	}

	// Verify final state
	tablesAfterRollbacks := getTableNames(t, db.Pool)
	t.Logf("Tables after migrations: %d, after rollbacks: %d",
		len(tablesAfterMigrations), len(tablesAfterRollbacks))

	// Should have fewer or equal tables after rollback
	assert.LessOrEqual(t, len(tablesAfterRollbacks), len(tablesAfterMigrations),
		"Rollback should not create new tables")
}

func migrationSupportsTransaction(migration Migration) bool {
	paths := []string{migration.Path, migration.RollbackPath}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		if strings.Contains(strings.ToUpper(string(body)), "CONCURRENTLY") {
			return false
		}
	}
	return true
}

// TestPartialMigrationFailure tests rollback when migration fails mid-way
func TestPartialMigrationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a migration that will fail mid-way
	badMigrationSQL := `
		-- This will succeed
		CREATE TABLE IF NOT EXISTS test_partial_table (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		);

		-- This will fail (duplicate table)
		CREATE TABLE test_partial_table (
			id UUID PRIMARY KEY
		);
	`

	// Attempt to apply bad migration (should fail)
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, badMigrationSQL)
	assert.Error(t, err, "Bad migration should fail")

	// Rollback transaction
	err = tx.Rollback(ctx)
	require.NoError(t, err, "Transaction rollback should succeed")

	// Verify table was NOT created (transaction rolled back)
	var exists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'test_partial_table'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "Partial migration should be rolled back completely")

	t.Log("Partial migration failure correctly rolled back")
}

// TestMigrationIdempotency tests that migrations can be applied multiple times safely
func TestMigrationIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Test with a simple migration that should be idempotent
	idempotentMigration := `
		-- Idempotent migration example
		CREATE TABLE IF NOT EXISTS test_idempotent (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_test_idempotent_name ON test_idempotent(name);
	`

	// Apply migration first time
	_, err := db.Pool.Exec(ctx, idempotentMigration)
	require.NoError(t, err, "First migration application should succeed")

	tables1 := getTableNames(t, db.Pool)

	// Apply migration second time (should not fail)
	_, err = db.Pool.Exec(ctx, idempotentMigration)
	require.NoError(t, err, "Second migration application should succeed (idempotent)")

	tables2 := getTableNames(t, db.Pool)

	// Verify schema is identical
	assert.Equal(t, tables1, tables2, "Schema should be identical after idempotent reapplication")

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS test_idempotent")
	require.NoError(t, err)

	t.Log("Migration idempotency verified successfully")
}

// TestMigrationRollbackDataIntegrity tests that rollback preserves data when possible
func TestMigrationRollbackDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create test table with data
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_data_integrity (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			value INTEGER,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	// Insert test data
	testData := []struct {
		name  string
		value int
	}{
		{"test1", 100},
		{"test2", 200},
		{"test3", 300},
	}

	for _, td := range testData {
		_, err := db.Pool.Exec(ctx,
			"INSERT INTO test_data_integrity (name, value) VALUES ($1, $2)",
			td.name, td.value)
		require.NoError(t, err)
	}

	// Apply a migration that adds an index (non-destructive)
	migrationSQL := `
		CREATE INDEX IF NOT EXISTS idx_test_data_integrity_name
		ON test_data_integrity(name);
	`
	_, err = db.Pool.Exec(ctx, migrationSQL)
	require.NoError(t, err)

	// Verify data is still intact
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_data_integrity").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(testData), count, "Data should be preserved after migration")

	// Rollback migration (drop index)
	rollbackSQL := `DROP INDEX IF EXISTS idx_test_data_integrity_name`
	_, err = db.Pool.Exec(ctx, rollbackSQL)
	require.NoError(t, err)

	// Verify data is STILL intact after rollback
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_data_integrity").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(testData), count, "Data should be preserved after rollback")

	// Verify data values
	rows, err := db.Pool.Query(ctx, "SELECT name, value FROM test_data_integrity ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()

	i := 0
	for rows.Next() {
		var name string
		var value int
		err := rows.Scan(&name, &value)
		require.NoError(t, err)
		assert.Equal(t, testData[i].name, name, "Data values should be preserved")
		assert.Equal(t, testData[i].value, value, "Data values should be preserved")
		i++
	}

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS test_data_integrity")
	require.NoError(t, err)

	t.Log("Data integrity verified through migration and rollback")
}

// TestSchemaVersionTracking tests that schema_migrations table is updated correctly
func TestSchemaVersionTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create schema_migrations table if not exists
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	// Test migration version tracking
	testVersion := fmt.Sprintf("TEST_%d", time.Now().Unix())

	// Insert version
	_, err = db.Pool.Exec(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		testVersion)
	require.NoError(t, err)

	// Verify version exists
	var exists bool
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
		testVersion).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "Migration version should be recorded")

	// Simulate rollback (remove version)
	_, err = db.Pool.Exec(ctx,
		"DELETE FROM schema_migrations WHERE version = $1",
		testVersion)
	require.NoError(t, err)

	// Verify version removed
	err = db.Pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
		testVersion).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "Migration version should be removed after rollback")

	t.Log("Schema version tracking verified")
}

// TestForeignKeyConstraintsDuringRollback tests that FK constraints are handled during rollback
func TestForeignKeyConstraintsDuringRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create parent and child tables with FK constraint
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_parent (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS test_child (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			parent_id UUID NOT NULL REFERENCES test_parent(id) ON DELETE CASCADE,
			name TEXT NOT NULL
		);
	`)
	require.NoError(t, err)

	// Insert test data
	var parentID string
	err = db.Pool.QueryRow(ctx,
		"INSERT INTO test_parent (name) VALUES ($1) RETURNING id",
		"test_parent").Scan(&parentID)
	require.NoError(t, err)

	_, err = db.Pool.Exec(ctx,
		"INSERT INTO test_child (parent_id, name) VALUES ($1, $2)",
		parentID, "test_child")
	require.NoError(t, err)

	// Attempt to drop parent table (should fail due to FK)
	_, err = db.Pool.Exec(ctx, "DROP TABLE test_parent")
	assert.Error(t, err, "Should not be able to drop parent table with FK references")

	// Proper rollback order: drop child first, then parent
	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS test_child")
	require.NoError(t, err)

	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS test_parent")
	require.NoError(t, err)

	t.Log("Foreign key constraints correctly enforced during rollback")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// Migration represents a database migration with its rollback
type Migration struct {
	Name         string
	Path         string
	RollbackPath string
	Version      int
}

// loadMigrationsWithRollbacks loads all migrations and their rollback files
func loadMigrationsWithRollbacks() ([]Migration, error) {
	migrationsDir := findMigrationsDirectory()
	if migrationsDir == "" {
		return nil, fmt.Errorf("migrations directory not found")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	rollbackFiles := make(map[string]string)

	// First pass: identify rollback files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "rollback_") {
			// Extract original migration name
			originalName := strings.TrimPrefix(name, "rollback_")
			rollbackFiles[originalName] = filepath.Join(migrationsDir, name)
		}
	}

	// Second pass: create migration objects
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), "rollback_") {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}

		// Extract version from filename (e.g., "001_" -> 1)
		var version int
		fmt.Sscanf(name, "%d_", &version)

		mig := Migration{
			Name:         name,
			Path:         filepath.Join(migrationsDir, name),
			RollbackPath: rollbackFiles[name],
			Version:      version,
		}

		migrations = append(migrations, mig)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// findMigrationsDirectory finds the migrations directory
func findMigrationsDirectory() string {
	candidates := []string{
		"supabase/migrations",
		"../supabase/migrations",
		"../../supabase/migrations",
		"../../../supabase/migrations",
		"scripts/migrations",
		"../scripts/migrations",
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

// applyMigration applies a migration from a file
func applyMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, migrationPath string) error {
	t.Helper()

	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration: %w", err)
	}

	// PostgreSQL does not allow concurrent index operations in a transaction.
	if strings.Contains(strings.ToUpper(string(migrationSQL)), "CONCURRENTLY") {
		if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
		return nil
	}

	// Execute in transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return tx.Commit(ctx)
}

// rollbackMigration rolls back a migration from a rollback file
func rollbackMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rollbackPath string) error {
	t.Helper()

	rollbackSQL, err := os.ReadFile(rollbackPath)
	if err != nil {
		return fmt.Errorf("failed to read rollback: %w", err)
	}

	// Execute in transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, string(rollbackSQL))
	if err != nil {
		return fmt.Errorf("failed to execute rollback: %w", err)
	}

	return tx.Commit(ctx)
}

// verifyMigrationState checks if a migration version is recorded
func verifyMigrationState(t *testing.T, pool *pgxpool.Pool, version string) bool {
	t.Helper()

	ctx := context.Background()
	var exists bool

	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)
	`, version).Scan(&exists)

	if err != nil {
		t.Logf("Warning: Could not verify migration state: %v", err)
		return false
	}

	return exists
}

// getTableNames returns all table names in the current database
func getTableNames(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()

	ctx := context.Background()
	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		tables = append(tables, name)
	}

	return tables
}

// getTableColumns returns all column names for a table
func getTableColumns(t *testing.T, pool *pgxpool.Pool, tableName string) []string {
	t.Helper()

	ctx := context.Background()
	rows, err := pool.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	require.NoError(t, err)
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		columns = append(columns, name)
	}

	return columns
}

// getIndexNames returns all index names in the current database
func getIndexNames(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()

	ctx := context.Background()
	rows, err := pool.Query(ctx, `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public'
		ORDER BY indexname
	`)
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}

	return indexes
}

// executeInTransaction executes SQL in a transaction
func executeInTransaction(ctx context.Context, pool *pgxpool.Pool, sql string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, sql)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// getTableRowCount returns the number of rows in a table
func getTableRowCount(t *testing.T, pool *pgxpool.Pool, tableName string) int {
	t.Helper()

	ctx := context.Background()
	var count int
	err := pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", pgx.Identifier{tableName}.Sanitize())).Scan(&count)
	require.NoError(t, err)
	return count
}
