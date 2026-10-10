package instagram

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations"
	"github.com/rhl/businessos-backend/internal/security"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceCredentialsAreEncryptedAndIsolated(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	userID := createInstagramTestUser(t, db, "instagram-owner")
	workspaceA := createInstagramTestWorkspace(t, db, userID, "instagram-a", "active")
	workspaceB := createInstagramTestWorkspace(t, db, userID, "instagram-b", "active")
	provider := newTestProvider(t, db)

	tokenA := testToken("secret-a", "account-a")
	tokenB := testToken("secret-b", "account-b")
	require.NoError(t, provider.SaveToken(ctx, userID, workspaceA, tokenA))
	require.NoError(t, provider.SaveToken(ctx, userID, workspaceB, tokenB))

	var encryptedA []byte
	require.NoError(t, db.QueryRow(ctx, `
		SELECT encrypted_data FROM instagram_workspace_credentials WHERE workspace_id = $1
	`, workspaceA).Scan(&encryptedA))
	require.NotEqual(t, []byte(tokenA.AccessToken), encryptedA)
	require.NotContains(t, string(encryptedA), tokenA.AccessToken)

	gotA, err := provider.GetToken(ctx, userID, workspaceA)
	require.NoError(t, err)
	require.Equal(t, tokenA.AccessToken, gotA.AccessToken)
	gotB, err := provider.GetToken(ctx, userID, workspaceB)
	require.NoError(t, err)
	require.Equal(t, tokenB.AccessToken, gotB.AccessToken)
}

func TestWorkspaceCredentialAccessRequiresActiveMembership(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	ownerID := createInstagramTestUser(t, db, "instagram-owner-access")
	otherID := createInstagramTestUser(t, db, "instagram-other-access")
	workspaceID := createInstagramTestWorkspace(t, db, ownerID, "instagram-access", "active")
	provider := newTestProvider(t, db)
	require.NoError(t, provider.SaveToken(ctx, ownerID, workspaceID, testToken("secret", "account")))

	_, err := provider.GetToken(ctx, otherID, workspaceID)
	require.ErrorIs(t, err, ErrWorkspaceAccess)
	require.ErrorIs(t, provider.Disconnect(ctx, otherID, workspaceID), ErrWorkspaceManagement)

	_, err = db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, status)
		VALUES ($1, $2, 'member', 'inactive')
	`, workspaceID, otherID)
	require.NoError(t, err)
	_, err = provider.GetConnectionStatus(ctx, otherID, workspaceID)
	require.ErrorIs(t, err, ErrWorkspaceAccess)
}

func TestDisconnectOnlyRemovesActiveWorkspaceCredential(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	userID := createInstagramTestUser(t, db, "instagram-disconnect")
	workspaceA := createInstagramTestWorkspace(t, db, userID, "instagram-disconnect-a", "active")
	workspaceB := createInstagramTestWorkspace(t, db, userID, "instagram-disconnect-b", "active")
	provider := newTestProvider(t, db)
	require.NoError(t, provider.SaveToken(ctx, userID, workspaceA, testToken("secret-a", "account-a")))
	require.NoError(t, provider.SaveToken(ctx, userID, workspaceB, testToken("secret-b", "account-b")))

	require.NoError(t, provider.Disconnect(ctx, userID, workspaceA))
	statusA, err := provider.GetConnectionStatus(ctx, userID, workspaceA)
	require.NoError(t, err)
	require.False(t, statusA.Connected)
	statusB, err := provider.GetConnectionStatus(ctx, userID, workspaceB)
	require.NoError(t, err)
	require.True(t, statusB.Connected)
}

func TestActiveMemberCanReadButCannotManageWorkspaceCredential(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	ownerID := createInstagramTestUser(t, db, "instagram-manager-owner")
	memberID := createInstagramTestUser(t, db, "instagram-manager-member")
	workspaceID := createInstagramTestWorkspace(t, db, ownerID, "instagram-manager", "active")
	_, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_name, status)
		VALUES ($1, $2, 'member', 'member', 'active')
	`, workspaceID, memberID)
	require.NoError(t, err)
	provider := newTestProvider(t, db)
	require.NoError(t, provider.SaveToken(ctx, ownerID, workspaceID, testToken("secret", "account")))

	status, err := provider.GetConnectionStatus(ctx, memberID, workspaceID)
	require.NoError(t, err)
	require.True(t, status.Connected)
	_, err = provider.CreateOAuthState(ctx, memberID, workspaceID)
	require.ErrorIs(t, err, ErrWorkspaceManagement)
	require.ErrorIs(t, provider.Disconnect(ctx, memberID, workspaceID), ErrWorkspaceManagement)
}

func TestOAuthStateIsOpaqueOneTimeAndWorkspaceBound(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	userID := createInstagramTestUser(t, db, "instagram-state")
	workspaceID := createInstagramTestWorkspace(t, db, userID, "instagram-state", "active")
	provider := newTestProvider(t, db)

	state, err := provider.CreateOAuthState(ctx, userID, workspaceID)
	require.NoError(t, err)
	require.NotContains(t, state, userID)
	require.NotContains(t, state, workspaceID)

	gotUserID, gotWorkspaceID, err := provider.ConsumeOAuthState(ctx, state)
	require.NoError(t, err)
	require.Equal(t, userID, gotUserID)
	require.Equal(t, workspaceID, gotWorkspaceID)

	_, _, err = provider.ConsumeOAuthState(ctx, state)
	require.ErrorIs(t, err, ErrInvalidOAuthState)
}

func TestInstagramMigrationPurgesLegacyPlaintextRows(t *testing.T) {
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	userID := createInstagramTestUser(t, db, "instagram-migration")
	_, err := db.Exec(ctx, `
		INSERT INTO credential_vault (user_id, provider_id, encrypted_data)
		VALUES ($1, 'instagram', 'plaintext-token'), ($1, 'google', 'encrypted-google')
	`, userID)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `
		INSERT INTO user_integrations (user_id, provider_id)
		VALUES ($1, 'instagram'), ($1, 'google')
	`, userID)
	require.NoError(t, err)

	migration, err := os.ReadFile("../../database/migrations/148_instagram_workspace_credentials.sql")
	require.NoError(t, err)
	_, err = db.Exec(ctx, string(migration))
	require.NoError(t, err)

	var instagramVaultRows, instagramIntegrationRows, googleVaultRows int
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM credential_vault WHERE provider_id = 'instagram'`).Scan(&instagramVaultRows))
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM user_integrations WHERE provider_id = 'instagram'`).Scan(&instagramIntegrationRows))
	require.NoError(t, db.QueryRow(ctx, `SELECT COUNT(*) FROM credential_vault WHERE provider_id = 'google'`).Scan(&googleVaultRows))
	require.Zero(t, instagramVaultRows)
	require.Zero(t, instagramIntegrationRows)
	require.Equal(t, 1, googleVaultRows)
}

func TestHandlerRejectsMissingAndCrossWorkspaceAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := requireInstagramTestDB(t)
	ctx := context.Background()
	ownerID := createInstagramTestUser(t, db, "instagram-handler-owner")
	otherID := createInstagramTestUser(t, db, "instagram-handler-other")
	workspaceID := createInstagramTestWorkspace(t, db, ownerID, "instagram-handler", "active")
	provider := newTestProvider(t, db)
	require.NoError(t, provider.SaveToken(ctx, ownerID, workspaceID, testToken("secret", "account")))
	handler := NewHandler(provider)

	missingWorkspace := httptest.NewRecorder()
	missingContext, _ := gin.CreateTestContext(missingWorkspace)
	missingContext.Request = httptest.NewRequest(http.MethodGet, "/status", nil)
	missingContext.Set("user_id", ownerID)
	handler.GetStatus(missingContext)
	require.Equal(t, http.StatusBadRequest, missingWorkspace.Code)

	invalidWorkspace := httptest.NewRecorder()
	invalidContext, _ := gin.CreateTestContext(invalidWorkspace)
	invalidContext.Request = httptest.NewRequest(http.MethodGet, "/status", nil)
	invalidContext.Request.Header.Set("X-Workspace-ID", "not-a-workspace-id")
	invalidContext.Set("user_id", ownerID)
	handler.GetStatus(invalidContext)
	require.Equal(t, http.StatusBadRequest, invalidWorkspace.Code)

	crossWorkspace := httptest.NewRecorder()
	crossContext, _ := gin.CreateTestContext(crossWorkspace)
	crossContext.Request = httptest.NewRequest(http.MethodPost, "/disconnect", nil)
	crossContext.Request.Header.Set("X-Workspace-ID", workspaceID)
	crossContext.Set("user_id", otherID)
	handler.Disconnect(crossContext)
	require.Equal(t, http.StatusForbidden, crossWorkspace.Code)

	status, err := provider.GetConnectionStatus(ctx, ownerID, workspaceID)
	require.NoError(t, err)
	require.True(t, status.Connected)
}

func requireInstagramTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable"
	}
	admin, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("PostgreSQL test database unavailable: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("PostgreSQL test database unavailable: %v", err)
	}

	schema := "instagram_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %s`, schema))
	require.NoError(t, err)

	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))

	_, err = pool.Exec(ctx, `
		CREATE TABLE "user" (
			id VARCHAR(255) PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL
		);
		CREATE TABLE workspaces (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			owner_id VARCHAR(255) NOT NULL
		);
		CREATE TABLE workspace_members (
			workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			role_name TEXT,
			status TEXT NOT NULL,
			UNIQUE (workspace_id, user_id)
		);
		CREATE TABLE credential_vault (
			user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			provider_id VARCHAR(50) NOT NULL,
			encrypted_data BYTEA NOT NULL,
			UNIQUE (user_id, provider_id)
		);
		CREATE TABLE user_integrations (
			user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			provider_id VARCHAR(50) NOT NULL,
			UNIQUE (user_id, provider_id)
		);
		CREATE TABLE instagram_workspace_credentials (
			workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
			connected_by_user_id VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
			encrypted_data BYTEA NOT NULL,
			encryption_version INTEGER NOT NULL DEFAULT 1,
			expires_at TIMESTAMPTZ,
			external_account_id VARCHAR(255),
			external_account_name VARCHAR(255),
			scopes TEXT[] NOT NULL DEFAULT '{}',
			metadata JSONB NOT NULL DEFAULT '{}',
			connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE instagram_oauth_states (
			state_hash BYTEA PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema))
		admin.Close()
	})
	return pool
}

func newTestProvider(t *testing.T, pool *pgxpool.Pool) *Provider {
	t.Helper()
	key, err := security.GenerateKey()
	require.NoError(t, err)
	encryption, err := security.NewTokenEncryption(key)
	require.NoError(t, err)
	return &Provider{pool: pool, encryption: encryption}
}

func createInstagramTestUser(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	id := prefix + "-" + uuid.NewString()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO "user" (id, email, password_hash, name)
		VALUES ($1, $2, 'test', 'Instagram Test')
	`, id, id+"@example.test")
	require.NoError(t, err)
	return id
}

func createInstagramTestWorkspace(t *testing.T, pool *pgxpool.Pool, userID, prefix, status string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO workspaces (id, name, slug, owner_id)
		VALUES ($1, $2, $3, $4)
	`, id, prefix, prefix+"-"+uuid.NewString(), userID)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
		INSERT INTO workspace_members (workspace_id, user_id, role, role_name, status)
		VALUES ($1, $2, 'owner', 'owner', $3)
	`, id, userID, status)
	require.NoError(t, err)
	return id
}

func testToken(accessToken, accountID string) *integrations.TokenResponse {
	return &integrations.TokenResponse{
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(time.Hour),
		Scopes:      []string{"instagram_basic"},
		AccountID:   accountID,
		AccountName: accountID,
		Metadata:    map[string]interface{}{"account": accountID},
	}
}
