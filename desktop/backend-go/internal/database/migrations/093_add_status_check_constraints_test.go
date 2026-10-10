package migrations

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration093_ValidStatusValues tests that valid status enum values are accepted
func TestMigration093_ValidStatusValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test workspace for FK constraint
	var workspaceID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// Define valid enum values based on migration 093
	validStatuses := []string{
		"generating", "generated", "deploying", "deployed",
		"running", "stopped", "failed", "archived",
	}
	validSandboxStatuses := []string{
		"none", "pending", "deploying", "running",
		"stopped", "failed", "removing",
	}
	validHealthStatuses := []string{
		"unknown", "healthy", "unhealthy", "degraded",
	}

	// Test valid app status values
	for _, status := range validStatuses {
		t.Run("ValidStatus_"+status, func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-"+status, "Test App", status).Scan(&appID)

			assert.NoError(t, err, "Valid status '%s' should be accepted", status)
			if err == nil {
				// Verify the status was saved correctly
				var savedStatus string
				err = db.Pool.QueryRow(ctx,
					"SELECT status FROM osa_module_instances WHERE id = $1", appID).Scan(&savedStatus)
				require.NoError(t, err)
				assert.Equal(t, status, savedStatus, "Status should be saved correctly")
			}
		})
	}

	// Test valid sandbox_status values
	for _, sandboxStatus := range validSandboxStatuses {
		t.Run("ValidSandboxStatus_"+sandboxStatus, func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, sandbox_status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-sandbox-"+sandboxStatus, "Test App", sandboxStatus).Scan(&appID)

			assert.NoError(t, err, "Valid sandbox_status '%s' should be accepted", sandboxStatus)
			if err == nil {
				// Verify the sandbox_status was saved correctly
				var savedStatus string
				err = db.Pool.QueryRow(ctx,
					"SELECT sandbox_status FROM osa_module_instances WHERE id = $1", appID).Scan(&savedStatus)
				require.NoError(t, err)
				assert.Equal(t, sandboxStatus, savedStatus, "Sandbox status should be saved correctly")
			}
		})
	}

	// Test valid health_status values
	for _, healthStatus := range validHealthStatuses {
		t.Run("ValidHealthStatus_"+healthStatus, func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, health_status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-health-"+healthStatus, "Test App", healthStatus).Scan(&appID)

			assert.NoError(t, err, "Valid health_status '%s' should be accepted", healthStatus)
			if err == nil {
				// Verify the health_status was saved correctly
				var savedStatus string
				err = db.Pool.QueryRow(ctx,
					"SELECT health_status FROM osa_module_instances WHERE id = $1", appID).Scan(&savedStatus)
				require.NoError(t, err)
				assert.Equal(t, healthStatus, savedStatus, "Health status should be saved correctly")
			}
		})
	}

	t.Logf("✅ All valid enum values tested successfully")
}

// TestMigration093_InvalidStatusValues tests that invalid status values are rejected
func TestMigration093_InvalidStatusValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test workspace for FK constraint
	var workspaceID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace-invalid', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// Define invalid enum values
	invalidStatuses := []string{
		"invalid", "GENERATING", "Generated", "deploye", // typos, wrong case
		"pending", "active", "deleted", "completed", // valid in other contexts
		"", "null", "undefined", // empty/null-like
	}
	invalidSandboxStatuses := []string{
		"invalid", "RUNNING", "None", "paused",
		"starting", "restarting", "destroyed",
		"", "N/A",
	}
	invalidHealthStatuses := []string{
		"invalid", "HEALTHY", "Unknown", "ok",
		"good", "bad", "critical", "warning",
		"", "n/a",
	}

	// Test invalid app status values
	for _, status := range invalidStatuses {
		t.Run("InvalidStatus_"+sanitizeTestName(status), func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-invalid", "Test App", status).Scan(&appID)

			assertConstraintViolation(t, err, "check_app_status",
				"Invalid status '%s' should be rejected by check_app_status constraint", status)
		})
	}

	// Test invalid sandbox_status values
	for _, sandboxStatus := range invalidSandboxStatuses {
		t.Run("InvalidSandboxStatus_"+sanitizeTestName(sandboxStatus), func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, sandbox_status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-invalid-sandbox", "Test App", sandboxStatus).Scan(&appID)

			assertConstraintViolation(t, err, "check_sandbox_status",
				"Invalid sandbox_status '%s' should be rejected by check_sandbox_status constraint", sandboxStatus)
		})
	}

	// Test invalid health_status values
	for _, healthStatus := range invalidHealthStatuses {
		t.Run("InvalidHealthStatus_"+sanitizeTestName(healthStatus), func(t *testing.T) {
			var appID string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO osa_module_instances (
					workspace_id, name, display_name, health_status
				) VALUES ($1, $2, $3, $4)
				RETURNING id
			`, workspaceID, "test-app-invalid-health", "Test App", healthStatus).Scan(&appID)

			assertConstraintViolation(t, err, "check_health_status",
				"Invalid health_status '%s' should be rejected by check_health_status constraint", healthStatus)
		})
	}

	t.Logf("✅ All invalid enum values correctly rejected")
}

// TestMigration093_UpdateInvalidValues tests that updates to invalid values are rejected
func TestMigration093_UpdateInvalidValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test workspace for FK constraint
	var workspaceID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace-update', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// Create a valid app
	var appID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name,
			status, sandbox_status, health_status
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, workspaceID, "test-app-update", "Test App",
		"generated", "none", "unknown").Scan(&appID)
	require.NoError(t, err, "Failed to create test app")

	// Test updating status to invalid value
	t.Run("UpdateInvalidStatus", func(t *testing.T) {
		_, err := db.Pool.Exec(ctx, `
			UPDATE osa_module_instances
			SET status = $1
			WHERE id = $2
		`, "invalid_status", appID)

		assertConstraintViolation(t, err, "check_app_status",
			"Updating to invalid status should be rejected")
	})

	// Test updating sandbox_status to invalid value
	t.Run("UpdateInvalidSandboxStatus", func(t *testing.T) {
		_, err := db.Pool.Exec(ctx, `
			UPDATE osa_module_instances
			SET sandbox_status = $1
			WHERE id = $2
		`, "invalid_sandbox", appID)

		assertConstraintViolation(t, err, "check_sandbox_status",
			"Updating to invalid sandbox_status should be rejected")
	})

	// Test updating health_status to invalid value
	t.Run("UpdateInvalidHealthStatus", func(t *testing.T) {
		_, err := db.Pool.Exec(ctx, `
			UPDATE osa_module_instances
			SET health_status = $1
			WHERE id = $2
		`, "invalid_health", appID)

		assertConstraintViolation(t, err, "check_health_status",
			"Updating to invalid health_status should be rejected")
	})

	// Verify original values are still intact
	var status, sandboxStatus, healthStatus string
	err = db.Pool.QueryRow(ctx, `
		SELECT status, sandbox_status, health_status
		FROM osa_module_instances
		WHERE id = $1
	`, appID).Scan(&status, &sandboxStatus, &healthStatus)
	require.NoError(t, err)
	assert.Equal(t, "generated", status, "Status should remain unchanged after failed update")
	assert.Equal(t, "none", sandboxStatus, "Sandbox status should remain unchanged after failed update")
	assert.Equal(t, "unknown", healthStatus, "Health status should remain unchanged after failed update")

	t.Logf("✅ Updates to invalid values correctly rejected")
}

// TestMigration093_ConstraintNames tests that constraint names are correct
func TestMigration093_ConstraintNames(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Query all constraints on osa_module_instances table
	rows, err := db.Pool.Query(ctx, `
		SELECT constraint_name, constraint_type
		FROM information_schema.table_constraints
		WHERE table_name = 'osa_module_instances'
		AND constraint_type = 'CHECK'
		AND constraint_name IN ('check_app_status', 'check_sandbox_status', 'check_health_status')
		ORDER BY constraint_name
	`)
	require.NoError(t, err, "Failed to query constraints")
	defer rows.Close()

	expectedConstraints := map[string]bool{
		"check_app_status":     false,
		"check_sandbox_status": false,
		"check_health_status":  false,
	}

	for rows.Next() {
		var name, constraintType string
		err := rows.Scan(&name, &constraintType)
		require.NoError(t, err)

		t.Logf("Found constraint: %s (type: %s)", name, constraintType)

		if _, exists := expectedConstraints[name]; exists {
			expectedConstraints[name] = true
			assert.Equal(t, "CHECK", constraintType, "Constraint %s should be CHECK type", name)
		}
	}

	// Verify all constraints were found
	for name, found := range expectedConstraints {
		assert.True(t, found, "Constraint %s should exist on osa_module_instances table", name)
	}

	t.Logf("✅ All constraint names verified")
}

// TestMigration093_RollbackRemovesConstraints tests that rollback properly removes constraints
func TestMigration093_RollbackRemovesConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Verify constraints exist before rollback
	constraintsBefore := getCheckConstraintCount(t, ctx, db)
	assert.GreaterOrEqual(t, constraintsBefore, 3,
		"Should have at least 3 CHECK constraints before rollback (check_app_status, check_sandbox_status, check_health_status)")

	// Execute rollback (down migration)
	rollbackSQL := `
		ALTER TABLE osa_module_instances
		DROP CONSTRAINT IF EXISTS check_health_status;

		ALTER TABLE osa_module_instances
		DROP CONSTRAINT IF EXISTS check_sandbox_status;

		ALTER TABLE osa_module_instances
		DROP CONSTRAINT IF EXISTS check_app_status;
	`

	_, err := db.Pool.Exec(ctx, rollbackSQL)
	require.NoError(t, err, "Rollback should succeed")

	// Verify constraints are removed
	constraintsAfter := getCheckConstraintCount(t, ctx, db)
	assert.Less(t, constraintsAfter, constraintsBefore,
		"Should have fewer CHECK constraints after rollback")

	// Verify specific constraints are gone
	constraintNames := []string{"check_app_status", "check_sandbox_status", "check_health_status"}
	for _, name := range constraintNames {
		exists := constraintExists(t, ctx, db, name)
		assert.False(t, exists, "Constraint %s should be removed after rollback", name)
	}

	// Create a test workspace for FK constraint
	var workspaceID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace-rollback', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// After rollback, invalid values should be accepted (no constraints)
	t.Run("InvalidValuesAcceptedAfterRollback", func(t *testing.T) {
		var appID string
		err := db.Pool.QueryRow(ctx, `
			INSERT INTO osa_module_instances (
				workspace_id, name, display_name,
				status, sandbox_status, health_status
			) VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, workspaceID, "test-app-rollback", "Test App",
			"invalid_status", "invalid_sandbox", "invalid_health").Scan(&appID)

		assert.NoError(t, err, "Invalid values should be accepted after rollback (no constraints)")

		if err == nil {
			// Verify values were saved
			var status, sandboxStatus, healthStatus string
			err = db.Pool.QueryRow(ctx, `
				SELECT status, sandbox_status, health_status
				FROM osa_module_instances WHERE id = $1
			`, appID).Scan(&status, &sandboxStatus, &healthStatus)
			require.NoError(t, err)
			assert.Equal(t, "invalid_status", status)
			assert.Equal(t, "invalid_sandbox", sandboxStatus)
			assert.Equal(t, "invalid_health", healthStatus)
		}
	})

	_, err = db.Pool.Exec(ctx, `
		DELETE FROM osa_module_instances
		WHERE workspace_id = $1
	`, workspaceID)
	require.NoError(t, err, "Rollback fixture cleanup should succeed")

	// Re-apply migration to restore constraints for other tests
	migrationSQL := `
		ALTER TABLE osa_module_instances
		ADD CONSTRAINT check_app_status
		CHECK (status IN (
			'generating', 'generated', 'deploying', 'deployed',
			'running', 'stopped', 'failed', 'archived'
		));

		ALTER TABLE osa_module_instances
		ADD CONSTRAINT check_sandbox_status
		CHECK (sandbox_status IN (
			'none', 'pending', 'deploying', 'running',
			'stopped', 'failed', 'removing'
		));

		ALTER TABLE osa_module_instances
		ADD CONSTRAINT check_health_status
		CHECK (health_status IN (
			'unknown', 'healthy', 'unhealthy', 'degraded'
		));
	`
	_, err = db.Pool.Exec(ctx, migrationSQL)
	require.NoError(t, err, "Re-applying migration should succeed")

	t.Logf("✅ Rollback correctly removes constraints")
}

// TestMigration093_DefaultValues tests that default values conform to constraints
func TestMigration093_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test workspace for FK constraint
	var workspaceID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace-defaults', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// Insert app without specifying status fields (use defaults)
	var appID string
	var status, sandboxStatus, healthStatus string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name
		) VALUES ($1, $2, $3)
		RETURNING id, status, sandbox_status, health_status
	`, workspaceID, "test-app-defaults", "Test App").Scan(&appID, &status, &sandboxStatus, &healthStatus)

	require.NoError(t, err, "Insert with default values should succeed")

	// Verify defaults match constraint values
	assert.Equal(t, "generated", status, "Default status should be 'generated' (valid in constraint)")
	assert.Equal(t, "none", sandboxStatus, "Default sandbox_status should be 'none' (valid in constraint)")
	assert.Equal(t, "unknown", healthStatus, "Default health_status should be 'unknown' (valid in constraint)")

	t.Logf("✅ Default values are valid and conform to constraints")
}

// TestMigration093_CombinedStatusChanges tests realistic status transitions
func TestMigration093_CombinedStatusChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test workspace for FK constraint
	var workspaceID string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id)
		VALUES ('test-workspace-transitions', gen_random_uuid()::text, gen_random_uuid()::text)
		RETURNING id
	`).Scan(&workspaceID)
	require.NoError(t, err, "Failed to create test workspace")

	// Create app in initial state
	var appID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO osa_module_instances (
			workspace_id, name, display_name,
			status, sandbox_status, health_status
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, workspaceID, "test-app-transitions", "Test App",
		"generating", "none", "unknown").Scan(&appID)
	require.NoError(t, err, "Failed to create test app")

	// Test realistic status transition: generating → generated → deploying → deployed → running
	transitions := []struct {
		name          string
		status        string
		sandboxStatus string
		healthStatus  string
	}{
		{"Generated", "generated", "none", "unknown"},
		{"Deploying", "deploying", "pending", "unknown"},
		{"DeployingToSandbox", "deploying", "deploying", "unknown"},
		{"Deployed", "deployed", "running", "unknown"},
		{"Running", "running", "running", "healthy"},
		{"Stopped", "stopped", "stopped", "unknown"},
		{"Failed", "failed", "failed", "unhealthy"},
		{"Archived", "archived", "none", "unknown"},
	}

	for _, trans := range transitions {
		t.Run("Transition_"+trans.name, func(t *testing.T) {
			_, err := db.Pool.Exec(ctx, `
				UPDATE osa_module_instances
				SET status = $1, sandbox_status = $2, health_status = $3
				WHERE id = $4
			`, trans.status, trans.sandboxStatus, trans.healthStatus, appID)

			assert.NoError(t, err, "Valid status transition to %s should succeed", trans.name)

			// Verify the update
			var status, sandboxStatus, healthStatus string
			err = db.Pool.QueryRow(ctx, `
				SELECT status, sandbox_status, health_status
				FROM osa_module_instances WHERE id = $1
			`, appID).Scan(&status, &sandboxStatus, &healthStatus)
			require.NoError(t, err)
			assert.Equal(t, trans.status, status)
			assert.Equal(t, trans.sandboxStatus, sandboxStatus)
			assert.Equal(t, trans.healthStatus, healthStatus)
		})
	}

	t.Logf("✅ Realistic status transitions work correctly")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// assertConstraintViolation verifies that an error is a constraint violation
func assertConstraintViolation(t *testing.T, err error, constraintName, msgFormat string, args ...interface{}) {
	t.Helper()

	if len(args) > 0 {
		require.Errorf(t, err, msgFormat, args...)
	} else {
		require.Error(t, err, msgFormat)
	}

	// Check if error is a pgconn.PgError (PostgreSQL error)
	var pgErr *pgconn.PgError
	if assert.ErrorAs(t, err, &pgErr, "Error should be a PostgreSQL error") {
		// PostgreSQL error code 23514 = check_violation
		assert.Equal(t, "23514", pgErr.Code, "Error should be a CHECK constraint violation (23514)")

		// Verify constraint name is mentioned in the error
		assert.Contains(t, strings.ToLower(pgErr.Message), strings.ToLower(constraintName),
			"Error message should mention constraint '%s'", constraintName)

		t.Logf("✓ Constraint violation correctly detected: %s", pgErr.Message)
	}
}

// getCheckConstraintCount returns the number of CHECK constraints on osa_module_instances
func getCheckConstraintCount(t *testing.T, ctx context.Context, db *testutil.TestDatabase) int {
	t.Helper()

	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.table_constraints
		WHERE table_name = 'osa_module_instances'
		AND constraint_type = 'CHECK'
	`).Scan(&count)
	require.NoError(t, err)

	return count
}

// constraintExists checks if a specific constraint exists on osa_module_instances
func constraintExists(t *testing.T, ctx context.Context, db *testutil.TestDatabase, constraintName string) bool {
	t.Helper()

	var exists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_name = 'osa_module_instances'
			AND constraint_name = $1
		)
	`, constraintName).Scan(&exists)
	require.NoError(t, err)

	return exists
}

// sanitizeTestName sanitizes strings for use in test names
func sanitizeTestName(s string) string {
	if s == "" {
		return "empty"
	}
	// Replace problematic characters
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
