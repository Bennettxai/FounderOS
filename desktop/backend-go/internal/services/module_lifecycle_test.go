package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Helpers (unit tests — no DB required)
// ============================================================================

// mockActionRegistry captures registered/unregistered action keys for assertions.
type mockActionRegistry struct {
	registered   map[string]interface{}
	unregistered []string
	failErr      error
}

func newMockRegistry() *mockActionRegistry {
	return &mockActionRegistry{registered: make(map[string]interface{})}
}

func (m *mockActionRegistry) RegisterModuleAction(key string, handler interface{}) error {
	if m.failErr != nil {
		return m.failErr
	}
	m.registered[key] = handler
	return nil
}

func (m *mockActionRegistry) UnregisterModuleAction(key string) error {
	if m.failErr != nil {
		return m.failErr
	}
	m.unregistered = append(m.unregistered, key)
	delete(m.registered, key)
	return nil
}

// newLifecycleInstallSvc builds a ModuleInstallationService with nil pool (pure action tests).
func newLifecycleInstallSvc(reg ModuleActionRegistry) *ModuleInstallationService {
	svc := NewModuleInstallationService(nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	svc.SetActionRegistry(reg)
	return svc
}

// makeModule returns a CustomModule whose manifest contains the given action names.
func makeModule(slug string, actionNames ...string) *CustomModule {
	actions := make([]interface{}, 0, len(actionNames))
	for _, name := range actionNames {
		actions = append(actions, map[string]interface{}{
			"name":        name,
			"type":        "function",
			"description": "lifecycle test action",
		})
	}
	return &CustomModule{
		ID:      uuid.New(),
		Slug:    slug,
		Name:    slug,
		Version: "1.0.0",
		Manifest: map[string]interface{}{
			"actions": actions,
		},
	}
}

// buildTestZIP creates a valid ZIP archive containing a module.json entry.
func buildTestZIP(t *testing.T, manifest ModuleManifest) []byte {
	t.Helper()
	data, err := json.Marshal(manifest)
	require.NoError(t, err)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("module.json")
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// ============================================================================
// Integration Helpers (DB-backed tests)
// ============================================================================

// loadFixtureManifest reads a JSON module manifest from testdata/modules/.
// Uses relative path — go test sets CWD to the package directory.
func loadFixtureManifest(t *testing.T, name string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile("testdata/modules/" + name + ".json")
	require.NoError(t, err, "fixture %s not found", name)

	var manifest map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &manifest))
	return manifest
}

// seedUserAndWorkspace inserts a minimal user and workspace so FK constraints
// on custom_modules / module_installations are satisfied.
func seedUserAndWorkspace(t *testing.T, ctx context.Context, db *testutil.TestDatabase) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	workspaceID := uuid.New()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, created_at, updated_at)
		VALUES ($1, $2, 'hash', 'Test User', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, userID, userID.String()+"@test.com")
	require.NoError(t, err, "seed user")

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, slug, owner_id, created_at, updated_at)
		VALUES ($1, 'Test Workspace', $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, workspaceID, "test-ws-"+workspaceID.String()[:8], userID.String())
	require.NoError(t, err, "seed workspace")

	return userID, workspaceID
}

// createModuleFromFixture creates a module in the DB using a named fixture file.
func createModuleFromFixture(
	t *testing.T,
	ctx context.Context,
	svc *CustomModuleService,
	workspaceID, userID uuid.UUID,
	fixtureName string,
) *CustomModule {
	t.Helper()
	manifest := loadFixtureManifest(t, fixtureName)
	name, _ := manifest["name"].(string)
	desc, _ := manifest["description"].(string)
	category, _ := manifest["category"].(string)

	module, err := svc.CreateModule(ctx, workspaceID, userID, CreateModuleRequest{
		Name:        name,
		Description: desc,
		Category:    category,
		Manifest:    manifest,
		Config:      map[string]interface{}{},
		Icon:        "\xf0\x9f\x93\xa6",
		Tags:        []string{category},
		Keywords:    []string{fixtureName},
	})
	require.NoError(t, err, "create module from fixture %s", fixtureName)
	return module
}

// ============================================================================
// Unit Tests — Action Registration (no DB)
// ============================================================================

func TestRegisterActionsActionKeys(t *testing.T) {
	reg := newMockRegistry()
	svc := newLifecycleInstallSvc(reg)
	ctx := context.Background()

	module := makeModule("crm", "create_contact", "list_contacts", "delete_contact")
	err := svc.RegisterActions(ctx, module)
	require.NoError(t, err)

	assert.Contains(t, reg.registered, "crm.create_contact")
	assert.Contains(t, reg.registered, "crm.list_contacts")
	assert.Contains(t, reg.registered, "crm.delete_contact")
	assert.Len(t, reg.registered, 3)
}

func TestAddFeatureRegistersNewAction(t *testing.T) {
	reg := newMockRegistry()
	svc := newLifecycleInstallSvc(reg)
	ctx := context.Background()

	module := makeModule("crm", "create_contact", "list_contacts")
	require.NoError(t, svc.RegisterActions(ctx, module))
	require.Len(t, reg.registered, 2)

	module.Manifest["actions"] = append(
		module.Manifest["actions"].([]interface{}),
		map[string]interface{}{
			"name":        "bulk_import",
			"type":        "workflow",
			"description": "batch contact import",
		},
	)

	require.NoError(t, svc.RegisterActions(ctx, module))

	assert.Contains(t, reg.registered, "crm.bulk_import", "newly added action must be registered")
	assert.Contains(t, reg.registered, "crm.create_contact", "existing actions must remain")
	assert.Contains(t, reg.registered, "crm.list_contacts", "existing actions must remain")
	assert.Len(t, reg.registered, 3)
}

func TestRemoveFeatureUnregistersActions(t *testing.T) {
	reg := newMockRegistry()
	svc := newLifecycleInstallSvc(reg)
	ctx := context.Background()

	module := makeModule("crm", "create_contact", "list_contacts", "delete_contact")
	require.NoError(t, svc.RegisterActions(ctx, module))
	require.Len(t, reg.registered, 3)

	module.Manifest["actions"] = []interface{}{
		map[string]interface{}{"name": "delete_contact", "type": "function"},
	}

	require.NoError(t, svc.UnregisterActions(module))

	assert.NotContains(t, reg.registered, "crm.delete_contact", "removed action must be unregistered")
	assert.Contains(t, reg.registered, "crm.create_contact", "retained action must stay registered")
	assert.Contains(t, reg.registered, "crm.list_contacts", "retained action must stay registered")
	assert.Len(t, reg.registered, 2)
	assert.Equal(t, []string{"crm.delete_contact"}, reg.unregistered)
}

func TestRegisterActionsWithNilRegistry(t *testing.T) {
	svc := NewModuleInstallationService(nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	module := makeModule("crm", "create_contact")
	err := svc.RegisterActions(context.Background(), module)
	assert.NoError(t, err)
}

func TestRegisterActionsInvalidManifestActions(t *testing.T) {
	reg := newMockRegistry()
	svc := newLifecycleInstallSvc(reg)
	ctx := context.Background()

	module := &CustomModule{
		ID:   uuid.New(),
		Slug: "test",
		Name: "test",
		Manifest: map[string]interface{}{
			"actions": []interface{}{
				"not-a-map",
				map[string]interface{}{"name": "valid_action", "type": "function"},
			},
		},
	}

	err := svc.RegisterActions(ctx, module)
	assert.NoError(t, err, "partial success should not error")
	assert.Contains(t, reg.registered, "test.valid_action")
	assert.Len(t, reg.registered, 1)
}

// ============================================================================
// Unit Tests — Export/Import (no DB)
// ============================================================================

func TestExportManifestRoundtrip(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	exportSvc := NewModuleExportService(nil, logger)

	moduleID := uuid.MustParse("00000000-0000-0000-0001-000000000001")
	now := time.Now().UTC().Truncate(time.Second)
	description := "Customer relationship management"
	original := &CustomModule{
		ID:          moduleID,
		Name:        "CRM Module",
		Slug:        "crm",
		Description: &description,
		Category:    "business",
		Version:     "1.2.3",
		UpdatedAt:   now,
		Manifest: map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{"name": "create_contact", "type": "function"},
			},
		},
	}

	manifest := exportSvc.GenerateManifestJSON(original)

	assert.Equal(t, moduleID.String(), manifest.ID)
	assert.Equal(t, "CRM Module", manifest.Name)
	assert.Equal(t, "crm", manifest.Slug)
	assert.Equal(t, "1.2.3", manifest.Version)
	assert.Equal(t, "business", manifest.Category)
	assert.Equal(t, now.Format("2006-01-02T15:04:05Z"), manifest.ExportedAt, "ExportedAt must match module UpdatedAt")

	actions, ok := manifest.Manifest["actions"].([]interface{})
	require.True(t, ok, "actions must be preserved in manifest")
	require.Len(t, actions, 1)
	action := actions[0].(map[string]interface{})
	assert.Equal(t, "create_contact", action["name"])
}

func TestImportParseZIP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	importSvc := NewModuleImportService(nil, logger)

	sourceManifest := ModuleManifest{
		ID:          uuid.MustParse("00000000-0000-0000-0001-000000000001").String(),
		Name:        "CRM Module",
		Slug:        "crm",
		Version:     "1.2.3",
		Description: "Customer relationship management",
		Category:    "business",
		Manifest: map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{"name": "create_contact", "type": "function"},
				map[string]interface{}{"name": "list_contacts", "type": "function"},
			},
		},
	}

	zipData := buildTestZIP(t, sourceManifest)
	require.NotEmpty(t, zipData)

	imported, err := importSvc.ParseZIP(zipData)
	require.NoError(t, err)

	assert.Equal(t, "crm", imported.Manifest.Slug)
	assert.Equal(t, "1.2.3", imported.Manifest.Version)
	assert.Equal(t, "CRM Module", imported.Manifest.Name)
	assert.Equal(t, "business", imported.Manifest.Category)
}

func TestImportExportCycle(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	exportSvc := NewModuleExportService(nil, logger)
	importSvc := NewModuleImportService(nil, logger)

	original := &CustomModule{
		ID:       uuid.MustParse("00000000-0000-0000-0001-000000000002"),
		Name:     "Projects Module",
		Slug:     "projects",
		Version:  "2.0.0",
		Category: "business",
		Manifest: map[string]interface{}{
			"actions": []interface{}{
				map[string]interface{}{"name": "create_project", "type": "function"},
				map[string]interface{}{"name": "archive_project", "type": "function"},
			},
			"protected_tables": []interface{}{"projects", "project_members"},
		},
	}

	manifest := exportSvc.GenerateManifestJSON(original)
	zipData := buildTestZIP(t, manifest)
	require.NotEmpty(t, zipData)

	imported, err := importSvc.ParseZIP(zipData)
	require.NoError(t, err)

	assert.Equal(t, original.Slug, imported.Manifest.Slug)
	assert.Equal(t, original.Version, imported.Manifest.Version)
	assert.Equal(t, original.Name, imported.Manifest.Name)
	assert.Equal(t, original.Category, imported.Manifest.Category)

	protectedTables, ok := imported.Manifest.Manifest["protected_tables"].([]interface{})
	require.True(t, ok, "protected_tables should survive the export/import cycle")
	assert.ElementsMatch(t, []interface{}{"projects", "project_members"}, protectedTables)

	actions, ok := imported.Manifest.Manifest["actions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, actions, 2)
}

// ============================================================================
// Fixture Validation — proves all 5 manifests pass validateManifest()
// ============================================================================

func TestFixtureManifests_AreValid(t *testing.T) {
	fixtures := []string{"crm", "projects", "tables", "documents", "notifications"}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			manifest := loadFixtureManifest(t, name)
			err := validateManifest(manifest)
			assert.NoError(t, err, "fixture %s should pass validation", name)

			assert.NotEmpty(t, manifest["name"], "must have name")
			assert.NotEmpty(t, manifest["version"], "must have version")
			assert.NotEmpty(t, manifest["migration_up"], "must have migration_up")
			assert.NotEmpty(t, manifest["migration_down"], "must have migration_down")
			assert.NotEmpty(t, manifest["routes"], "must have routes")
			assert.NotEmpty(t, manifest["protected_schemas"], "must have protected_schemas")
		})
	}
}

// ============================================================================
// Integration Tests — Full Lifecycle (DB required, skipped in short mode)
// ============================================================================

func TestModuleLifecycle_SingleInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "crm")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)
	assert.Equal(t, module.ID, installations[0].ModuleID)
	assert.True(t, installations[0].IsEnabled)

	updated, err := moduleSvc.GetModule(ctx, module.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.InstallCount)
}

func TestModuleLifecycle_InstallTwiceFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "projects")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	err = installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already installed")

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)
}

func TestModuleLifecycle_Uninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "tables")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	err = installSvc.UninstallModule(ctx, module.ID, workspaceID)
	require.NoError(t, err)

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 0)

	updated, err := moduleSvc.GetModule(ctx, module.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.InstallCount)
}

func TestModuleLifecycle_ReinstallAfterUninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "documents")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	err = installSvc.UninstallModule(ctx, module.ID, workspaceID)
	require.NoError(t, err)

	err = installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)
	assert.True(t, installations[0].IsEnabled)
}

func TestModuleLifecycle_AddFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "crm")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	originalActions := module.Manifest["actions"].([]interface{})
	originalCount := len(originalActions)

	newAction := map[string]interface{}{
		"name":        "export_contacts_csv",
		"type":        "function",
		"description": "Export all contacts as CSV file",
	}
	updatedActions := append(originalActions, newAction)
	updatedManifest := make(map[string]interface{})
	for k, v := range module.Manifest {
		updatedManifest[k] = v
	}
	updatedManifest["actions"] = updatedActions

	updated, err := moduleSvc.UpdateModule(ctx, module.ID, userID, UpdateModuleRequest{
		Manifest: &updatedManifest,
	})
	require.NoError(t, err)

	newActions := updated.Manifest["actions"].([]interface{})
	assert.Len(t, newActions, originalCount+1)

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)
	assert.True(t, installations[0].IsEnabled)
}

func TestModuleLifecycle_RemoveFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "notifications")

	err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
	require.NoError(t, err)

	originalActions := module.Manifest["actions"].([]interface{})
	require.True(t, len(originalActions) >= 2, "need at least 2 actions to remove one")

	reducedActions := originalActions[:len(originalActions)-1]
	updatedManifest := make(map[string]interface{})
	for k, v := range module.Manifest {
		updatedManifest[k] = v
	}
	updatedManifest["actions"] = reducedActions

	updated, err := moduleSvc.UpdateModule(ctx, module.ID, userID, UpdateModuleRequest{
		Manifest: &updatedManifest,
	})
	require.NoError(t, err)

	newActions := updated.Manifest["actions"].([]interface{})
	assert.Len(t, newActions, len(originalActions)-1)

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)
}

func TestModuleLifecycle_ProtectionEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	protectionSvc := NewModuleProtectionService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "crm")

	// Delete protected schema — must be blocked
	result, err := protectionSvc.ValidateChange(ctx, module.ID, ChangeRequest{
		ChangeType: "schema",
		Target:     "crm_contacts",
		Operation:  "delete",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed, "deleting protected schema must be blocked")
	assert.NotEmpty(t, result.Violations)
	assert.Equal(t, "error", result.Violations[0].Severity)
	assert.Contains(t, result.Violations[0].Message, "crm_contacts")

	// Delete protected route — must be blocked
	result, err = protectionSvc.ValidateChange(ctx, module.ID, ChangeRequest{
		ChangeType: "route",
		Target:     "api/crm/contacts",
		Operation:  "delete",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed, "modifying protected route must be blocked")

	// Delete protected operation — must be blocked
	result, err = protectionSvc.ValidateChange(ctx, module.ID, ChangeRequest{
		ChangeType: "operation",
		Target:     "crm.delete_all_contacts",
		Operation:  "delete",
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed, "protected operation must be blocked")

	// Update protected schema — should WARN but allow
	result, err = protectionSvc.ValidateChange(ctx, module.ID, ChangeRequest{
		ChangeType: "schema",
		Target:     "crm_contacts",
		Operation:  "update",
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed, "update should be allowed with warning")
	assert.NotEmpty(t, result.Warnings)

	// Non-protected target — fully allowed
	result, err = protectionSvc.ValidateChange(ctx, module.ID, ChangeRequest{
		ChangeType: "schema",
		Target:     "some_unrelated_table",
		Operation:  "delete",
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed, "non-protected target should be allowed")
	assert.Empty(t, result.Violations)
}

func TestModuleLifecycle_ExportImportCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	exportSvc := NewModuleExportService(db.Pool, logger)
	importSvc := NewModuleImportService(db.Pool, logger)

	original := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "crm")

	zipData, err := exportSvc.ExportModule(ctx, original.ID)
	require.NoError(t, err)
	require.NotEmpty(t, zipData)

	userID2, workspaceID2 := seedUserAndWorkspace(t, ctx, db)

	imported, err := importSvc.ImportModule(ctx, zipData, workspaceID2, userID2)
	require.NoError(t, err)
	require.NotNil(t, imported)

	assert.Equal(t, original.Name, imported.Name)
	assert.Equal(t, original.Category, imported.Category)

	origActions := original.Manifest["actions"].([]interface{})
	impActions := imported.Manifest["actions"].([]interface{})
	assert.Equal(t, len(origActions), len(impActions), "action count must match after import")
}

func TestModuleLifecycle_MultiModuleCoexistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	fixtures := []string{"crm", "projects", "tables", "documents", "notifications"}
	moduleIDs := make([]uuid.UUID, 0, len(fixtures))

	for _, name := range fixtures {
		module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, name)
		err := installSvc.InstallModule(ctx, module.ID, workspaceID, userID)
		require.NoError(t, err, "install %s", name)
		moduleIDs = append(moduleIDs, module.ID)
	}

	installations, err := installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 5)

	for _, inst := range installations {
		assert.True(t, inst.IsEnabled)
	}

	err = installSvc.UninstallModule(ctx, moduleIDs[2], workspaceID) // tables
	require.NoError(t, err)

	installations, err = installSvc.ListInstalledModules(ctx, workspaceID)
	require.NoError(t, err)
	assert.Len(t, installations, 4)
}

func TestModuleLifecycle_UninstallNotInstalled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.RequireTestDatabase(t)
	ctx := context.Background()
	t.Cleanup(func() {
		testutil.CleanupTestData(ctx, db.Pool)
		db.Close()
	})

	logger := slog.Default()
	userID, workspaceID := seedUserAndWorkspace(t, ctx, db)

	moduleSvc := NewCustomModuleService(db.Pool, logger)
	installSvc := NewModuleInstallationService(db.Pool, logger)

	module := createModuleFromFixture(t, ctx, moduleSvc, workspaceID, userID, "crm")

	err := installSvc.UninstallModule(ctx, module.ID, workspaceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}
