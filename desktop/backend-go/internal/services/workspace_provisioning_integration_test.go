package services

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test targets the same isolated database used by installation verification.
// It is opt-in locally because it creates and deletes account records.
func TestEnsureDefaultWorkspaceAgainstInstallationDatabase(t *testing.T) {
	databaseURL := os.Getenv("WORKSPACE_PROVISIONING_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("WORKSPACE_PROVISIONING_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()

	ownerID := "install-owner-" + uuid.NewString()
	memberID := "install-member-" + uuid.NewString()
	for _, user := range []struct{ id, email string }{
		{ownerID, ownerID + "@example.test"},
		{memberID, memberID + "@example.test"},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO "user" (id, name, email, "createdAt", "updatedAt")
			VALUES ($1, $1, $2, NOW(), NOW())
		`, user.id, user.email)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE owner_id = $1`, ownerID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user" WHERE id = ANY($1)`, []string{ownerID, memberID})
	})

	service := NewWorkspaceService(pool)
	first, err := service.EnsureDefaultWorkspace(ctx, ownerID)
	require.NoError(t, err)
	second, err := service.EnsureDefaultWorkspace(ctx, ownerID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	var roleCount, ownerLevel int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspace_roles WHERE workspace_id = $1`, first.ID).Scan(&roleCount))
	require.NoError(t, pool.QueryRow(ctx, `SELECT hierarchy_level FROM workspace_roles WHERE workspace_id = $1 AND name = 'owner'`, first.ID).Scan(&ownerLevel))
	assert.Equal(t, 6, roleCount)
	assert.Equal(t, 1, ownerLevel)

	_, err = pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role_id, role_name, role, status)
		SELECT $1, $2, id, 'member', 'member', 'active'
		FROM workspace_roles WHERE workspace_id = $1 AND name = 'member'
	`, first.ID, memberID)
	require.NoError(t, err)

	memberWorkspace, err := service.EnsureDefaultWorkspace(ctx, memberID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, memberWorkspace.ID)
	var memberRole string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id = $1 AND user_id = $2
	`, first.ID, memberID).Scan(&memberRole))
	assert.Equal(t, "member", memberRole)
}
