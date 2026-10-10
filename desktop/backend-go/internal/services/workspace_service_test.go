package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TEST HELPERS
// =============================================================================

// setupTestWorkspace creates a test workspace with roles and returns cleanup function
func setupTestWorkspace(t *testing.T, pool *pgxpool.Pool, ownerID string) (*Workspace, func()) {
	service := NewWorkspaceService(pool)
	ctx := context.Background()

	// Create workspace
	req := CreateWorkspaceRequest{
		Name:     "Test Workspace",
		PlanType: "professional",
	}

	workspace, err := service.CreateWorkspace(ctx, req, ownerID)
	require.NoError(t, err, "Failed to create test workspace")
	require.NotNil(t, workspace)

	// Cleanup function
	cleanup := func() {
		// Delete workspace (cascade will delete roles and members)
		err := service.DeleteWorkspace(ctx, workspace.ID, ownerID)
		if err != nil {
			t.Logf("Warning: failed to cleanup workspace: %v", err)
		}
	}

	return workspace, cleanup
}

// setupTestUsers returns test user IDs
func setupTestUsers() (owner, admin, member, viewer string) {
	return "test-owner-" + uuid.New().String()[:8],
		"test-admin-" + uuid.New().String()[:8],
		"test-member-" + uuid.New().String()[:8],
		"test-viewer-" + uuid.New().String()[:8]
}

// =============================================================================
// WORKSPACE CREATION TESTS
// =============================================================================

// TestWorkspaceCreation tests workspace creation with default roles
func TestWorkspaceCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	service := NewWorkspaceService(db.Pool)
	ctx := context.Background()

	ownerID := "test-user-" + uuid.New().String()[:8]

	// Test: Create workspace with valid data
	t.Run("Create workspace successfully", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:        "Acme Corporation",
			Slug:        "acme-corp",
			Description: stringPtrTest("A test workspace"),
			PlanType:    "professional",
		}

		workspace, err := service.CreateWorkspace(ctx, req, ownerID)
		require.NoError(t, err)
		require.NotNil(t, workspace)

		// Verify workspace fields
		assert.Equal(t, "Acme Corporation", workspace.Name)
		assert.Equal(t, "acme-corp", workspace.Slug)
		assert.Equal(t, "professional", workspace.PlanType)
		assert.Equal(t, ownerID, workspace.OwnerID)
		assert.Equal(t, 50, workspace.MaxMembers)    // Professional plan
		assert.Equal(t, 200, workspace.MaxProjects)  // Professional plan
		assert.Equal(t, 200, workspace.MaxStorageGB) // Professional plan

		// Verify 6 default roles were created
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)
		assert.Len(t, roles, 6, "Should have 6 default system roles")

		// Verify role names
		roleNames := make(map[string]bool)
		for _, role := range roles {
			roleNames[role.Name] = true
		}
		assert.True(t, roleNames["owner"], "Should have owner role")
		assert.True(t, roleNames["admin"], "Should have admin role")
		assert.True(t, roleNames["manager"], "Should have manager role")
		assert.True(t, roleNames["member"], "Should have member role")
		assert.True(t, roleNames["viewer"], "Should have viewer role")
		assert.True(t, roleNames["guest"], "Should have guest role")

		// Verify owner was added as member
		members, err := service.ListMembers(ctx, workspace.ID)
		require.NoError(t, err)
		assert.Len(t, members, 1, "Should have 1 member (owner)")
		assert.Equal(t, ownerID, members[0].UserID)
		assert.Equal(t, "owner", members[0].Role)
		assert.Equal(t, "active", members[0].Status)

		// Cleanup
		err = service.DeleteWorkspace(ctx, workspace.ID, ownerID)
		require.NoError(t, err)
	})

	// Test: Auto-generate slug if not provided
	t.Run("Auto-generate slug", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:     "My Test Workspace",
			PlanType: "free",
		}

		workspace, err := service.CreateWorkspace(ctx, req, ownerID)
		require.NoError(t, err)
		require.NotNil(t, workspace)

		// Verify slug was auto-generated
		assert.Equal(t, "my-test-workspace", workspace.Slug)

		// Cleanup
		_ = service.DeleteWorkspace(ctx, workspace.ID, ownerID)
	})

	// Test: Validation - empty name
	t.Run("Reject empty workspace name", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:     "",
			PlanType: "free",
		}

		workspace, err := service.CreateWorkspace(ctx, req, ownerID)
		assert.Error(t, err)
		assert.Nil(t, workspace)
		assert.Contains(t, err.Error(), "workspace name is required")
	})

	// Test: Validation - invalid slug
	t.Run("Reject invalid slug", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:     "Test",
			Slug:     "Invalid Slug!",
			PlanType: "free",
		}

		workspace, err := service.CreateWorkspace(ctx, req, ownerID)
		assert.Error(t, err)
		assert.Nil(t, workspace)
		assert.Contains(t, err.Error(), "invalid slug")
	})

	// Test: Plan limits
	t.Run("Set correct limits for free plan", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:     "Free Workspace",
			PlanType: "free",
		}

		workspace, err := service.CreateWorkspace(ctx, req, ownerID)
		require.NoError(t, err)

		assert.Equal(t, 5, workspace.MaxMembers)
		assert.Equal(t, 10, workspace.MaxProjects)
		assert.Equal(t, 5, workspace.MaxStorageGB)

		// Cleanup
		_ = service.DeleteWorkspace(ctx, workspace.ID, ownerID)
	})
}

// =============================================================================
// MEMBER MANAGEMENT TESTS
// =============================================================================

// TestMemberManagement tests adding, updating, and removing members
func TestMemberManagement(t *testing.T) {
	t.Skip("Requires database - implement with testcontainers or run with live DB")

	var pool *pgxpool.Pool // Replace with actual pool
	service := NewWorkspaceService(pool)
	ctx := context.Background()

	owner, admin, member, viewer := setupTestUsers()
	workspace, cleanup := setupTestWorkspace(t, pool, owner)
	defer cleanup()

	// Test: Add member with default role
	t.Run("Add member with default role", func(t *testing.T) {
		req := AddMemberRequest{
			UserID: member,
			Role:   "member",
		}

		newMember, err := service.AddMember(ctx, workspace.ID, req, owner)
		require.NoError(t, err)
		assert.Equal(t, member, newMember.UserID)
		assert.Equal(t, "member", newMember.Role)
		assert.Equal(t, "active", newMember.Status)
		assert.NotNil(t, newMember.InvitedBy)
		assert.Equal(t, owner, *newMember.InvitedBy)
	})

	// Test: Add admin
	t.Run("Add admin member", func(t *testing.T) {
		req := AddMemberRequest{
			UserID: admin,
			Role:   "admin",
		}

		newMember, err := service.AddMember(ctx, workspace.ID, req, owner)
		require.NoError(t, err)
		assert.Equal(t, "admin", newMember.Role)
	})

	// Test: Update member role
	t.Run("Update member role", func(t *testing.T) {
		// Upgrade member to manager
		updatedMember, err := service.UpdateMemberRole(ctx, workspace.ID, member, "manager")
		require.NoError(t, err)
		assert.Equal(t, "manager", updatedMember.Role)

		// Verify role was updated
		role, err := service.GetUserRole(ctx, workspace.ID, member)
		require.NoError(t, err)
		assert.Equal(t, "manager", role)
	})

	// Test: Cannot change owner role
	t.Run("Cannot change owner role", func(t *testing.T) {
		_, err := service.UpdateMemberRole(ctx, workspace.ID, owner, "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot change owner role")
	})

	// Test: Cannot remove owner
	t.Run("Cannot remove owner", func(t *testing.T) {
		err := service.RemoveMember(ctx, workspace.ID, owner)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove workspace owner")
	})

	// Test: Remove member
	t.Run("Remove member", func(t *testing.T) {
		// Add viewer first
		req := AddMemberRequest{
			UserID: viewer,
			Role:   "viewer",
		}
		_, err := service.AddMember(ctx, workspace.ID, req, owner)
		require.NoError(t, err)

		// Remove viewer
		err = service.RemoveMember(ctx, workspace.ID, viewer)
		require.NoError(t, err)

		// Verify viewer is removed
		_, err = service.GetUserRole(ctx, workspace.ID, viewer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a member")
	})

	// Test: Member limit enforcement
	t.Run("Enforce member limit", func(t *testing.T) {
		// Create free workspace (5 member limit)
		freeReq := CreateWorkspaceRequest{
			Name:     "Free Workspace",
			PlanType: "free",
		}
		freeWorkspace, err := service.CreateWorkspace(ctx, freeReq, owner)
		require.NoError(t, err)
		defer service.DeleteWorkspace(ctx, freeWorkspace.ID, owner)

		// Try to add 5 more members (total 6, exceeds limit)
		for i := 0; i < 5; i++ {
			userID := "user-" + uuid.New().String()[:8]
			req := AddMemberRequest{
				UserID: userID,
				Role:   "member",
			}
			_, err := service.AddMember(ctx, freeWorkspace.ID, req, owner)
			if i < 4 {
				require.NoError(t, err, "Should allow members up to limit")
			} else {
				assert.Error(t, err, "Should reject member exceeding limit")
				assert.Contains(t, err.Error(), "maximum member limit")
			}
		}
	})

	// Test: Invalid role
	t.Run("Reject invalid role", func(t *testing.T) {
		req := AddMemberRequest{
			UserID: "new-user",
			Role:   "invalid-role",
		}

		_, err := service.AddMember(ctx, workspace.ID, req, owner)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

// =============================================================================
// ROLE HIERARCHY TESTS
// =============================================================================

// TestRoleHierarchy tests role hierarchy levels and permissions
func TestRoleHierarchy(t *testing.T) {
	t.Skip("Requires database - implement with testcontainers or run with live DB")

	var pool *pgxpool.Pool // Replace with actual pool
	service := NewWorkspaceService(pool)
	ctx := context.Background()

	owner := "test-owner-" + uuid.New().String()[:8]
	workspace, cleanup := setupTestWorkspace(t, pool, owner)
	defer cleanup()

	// Test: Verify hierarchy levels
	t.Run("Verify hierarchy levels", func(t *testing.T) {
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)

		hierarchyLevels := make(map[string]int)
		for _, role := range roles {
			hierarchyLevels[role.Name] = role.HierarchyLevel
		}

		assert.Equal(t, 1, hierarchyLevels["owner"], "Owner should be level 1")
		assert.Equal(t, 2, hierarchyLevels["admin"], "Admin should be level 2")
		assert.Equal(t, 3, hierarchyLevels["manager"], "Manager should be level 3")
		assert.Equal(t, 4, hierarchyLevels["member"], "Member should be level 4")
		assert.Equal(t, 5, hierarchyLevels["viewer"], "Viewer should be level 5")
		assert.Equal(t, 6, hierarchyLevels["guest"], "Guest should be level 6")
	})

	// Test: System roles cannot be deleted
	t.Run("System roles are protected", func(t *testing.T) {
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)

		for _, role := range roles {
			assert.True(t, role.IsSystem, "All default roles should be system roles")
		}
	})

	// Test: Member is default role
	t.Run("Member is default role", func(t *testing.T) {
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)

		hasDefaultRole := false
		for _, role := range roles {
			if role.IsDefault {
				assert.Equal(t, "member", role.Name, "Member should be the default role")
				hasDefaultRole = true
			}
		}
		assert.True(t, hasDefaultRole, "Should have a default role")
	})

	// Test: Permission structure
	t.Run("Verify owner permissions", func(t *testing.T) {
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)

		var ownerRole *WorkspaceRole
		for i := range roles {
			if roles[i].Name == "owner" {
				ownerRole = &roles[i]
				break
			}
		}
		require.NotNil(t, ownerRole)

		// Check workspace permissions
		workspacePerms, ok := ownerRole.Permissions["workspace"].(map[string]interface{})
		require.True(t, ok, "Should have workspace permissions")
		assert.Equal(t, true, workspacePerms["delete_workspace"], "Owner can delete workspace")
		assert.Equal(t, true, workspacePerms["manage_billing"], "Owner can manage billing")

		// Check agent permissions
		agentPerms, ok := ownerRole.Permissions["agents"].(map[string]interface{})
		require.True(t, ok, "Should have agent permissions")
		assert.Equal(t, true, agentPerms["modify_workspace_memory"], "Owner can modify workspace memory")
	})

	// Test: Viewer restrictions
	t.Run("Verify viewer restrictions", func(t *testing.T) {
		roles, err := service.ListRoles(ctx, workspace.ID)
		require.NoError(t, err)

		var viewerRole *WorkspaceRole
		for i := range roles {
			if roles[i].Name == "viewer" {
				viewerRole = &roles[i]
				break
			}
		}
		require.NotNil(t, viewerRole)

		// Verify read-only project permissions
		projectPerms, ok := viewerRole.Permissions["projects"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, projectPerms["read"], "Viewer can read projects")
		assert.Equal(t, false, projectPerms["create"], "Viewer cannot create projects")
		assert.Equal(t, false, projectPerms["update"], "Viewer cannot update projects")
		assert.Equal(t, false, projectPerms["delete"], "Viewer cannot delete projects")
	})
}

// =============================================================================
// WORKSPACE CRUD TESTS
// =============================================================================

// TestWorkspaceCRUD tests workspace CRUD operations
func TestWorkspaceCRUD(t *testing.T) {
	t.Skip("Requires database - implement with testcontainers or run with live DB")

	var pool *pgxpool.Pool // Replace with actual pool
	service := NewWorkspaceService(pool)
	ctx := context.Background()

	owner := "test-owner-" + uuid.New().String()[:8]

	// Test: Get workspace by ID
	t.Run("Get workspace by ID", func(t *testing.T) {
		workspace, cleanup := setupTestWorkspace(t, pool, owner)
		defer cleanup()

		retrieved, err := service.GetWorkspace(ctx, workspace.ID)
		require.NoError(t, err)
		assert.Equal(t, workspace.ID, retrieved.ID)
		assert.Equal(t, workspace.Name, retrieved.Name)
	})

	// Test: Get workspace by slug
	t.Run("Get workspace by slug", func(t *testing.T) {
		req := CreateWorkspaceRequest{
			Name:     "Slug Test",
			Slug:     "slug-test",
			PlanType: "free",
		}
		workspace, err := service.CreateWorkspace(ctx, req, owner)
		require.NoError(t, err)
		defer service.DeleteWorkspace(ctx, workspace.ID, owner)

		retrieved, err := service.GetWorkspaceBySlug(ctx, "slug-test")
		require.NoError(t, err)
		assert.Equal(t, workspace.ID, retrieved.ID)
	})

	// Test: Update workspace
	t.Run("Update workspace", func(t *testing.T) {
		workspace, cleanup := setupTestWorkspace(t, pool, owner)
		defer cleanup()

		newName := "Updated Name"
		newDesc := "Updated description"
		updateReq := UpdateWorkspaceRequest{
			Name:        &newName,
			Description: &newDesc,
		}

		updated, err := service.UpdateWorkspace(ctx, workspace.ID, updateReq)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newDesc, *updated.Description)
	})

	// Test: List user workspaces
	t.Run("List user workspaces", func(t *testing.T) {
		// Create 3 workspaces for same user
		ws1, cleanup1 := setupTestWorkspace(t, pool, owner)
		defer cleanup1()
		ws2, cleanup2 := setupTestWorkspace(t, pool, owner)
		defer cleanup2()
		ws3, cleanup3 := setupTestWorkspace(t, pool, owner)
		defer cleanup3()

		workspaces, err := service.ListUserWorkspaces(ctx, owner)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(workspaces), 3, "Should list at least 3 workspaces")

		// Verify our workspaces are in the list
		ids := make(map[uuid.UUID]bool)
		for _, ws := range workspaces {
			ids[ws.ID] = true
		}
		assert.True(t, ids[ws1.ID])
		assert.True(t, ids[ws2.ID])
		assert.True(t, ids[ws3.ID])
	})

	// Test: Delete workspace (only owner)
	t.Run("Delete workspace as owner", func(t *testing.T) {
		workspace, _ := setupTestWorkspace(t, pool, owner)

		err := service.DeleteWorkspace(ctx, workspace.ID, owner)
		require.NoError(t, err)

		// Verify deletion
		_, err = service.GetWorkspace(ctx, workspace.ID)
		assert.Error(t, err)
	})

	// Test: Non-owner cannot delete
	t.Run("Non-owner cannot delete workspace", func(t *testing.T) {
		workspace, cleanup := setupTestWorkspace(t, pool, owner)
		defer cleanup()

		otherUser := "other-user-" + uuid.New().String()[:8]
		err := service.DeleteWorkspace(ctx, workspace.ID, otherUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only workspace owner")
	})
}

// =============================================================================
// PERMISSION ENFORCEMENT TESTS
// =============================================================================

// TestPermissionEnforcement tests permission checks
func TestPermissionEnforcement(t *testing.T) {
	t.Skip("Requires database - implement with testcontainers or run with live DB")

	var pool *pgxpool.Pool // Replace with actual pool
	service := NewWorkspaceService(pool)
	ctx := context.Background()

	owner, _, member, viewer := setupTestUsers()
	workspace, cleanup := setupTestWorkspace(t, pool, owner)
	defer cleanup()

	// Add member and viewer
	_, _ = service.AddMember(ctx, workspace.ID, AddMemberRequest{UserID: member, Role: "member"}, owner)
	_, _ = service.AddMember(ctx, workspace.ID, AddMemberRequest{UserID: viewer, Role: "viewer"}, owner)

	// Test: Get role permissions
	t.Run("Get user role and permissions", func(t *testing.T) {
		// Get owner role
		role, err := service.GetUserRole(ctx, workspace.ID, owner)
		require.NoError(t, err)
		assert.Equal(t, "owner", role)

		// Get member role
		role, err = service.GetUserRole(ctx, workspace.ID, member)
		require.NoError(t, err)
		assert.Equal(t, "member", role)

		// Get viewer role
		role, err = service.GetUserRole(ctx, workspace.ID, viewer)
		require.NoError(t, err)
		assert.Equal(t, "viewer", role)

		// Non-member has no role
		_, err = service.GetUserRole(ctx, workspace.ID, "non-member")
		assert.Error(t, err)
	})

	// Test: List members
	t.Run("List all workspace members", func(t *testing.T) {
		members, err := service.ListMembers(ctx, workspace.ID)
		require.NoError(t, err)
		assert.Len(t, members, 3, "Should have owner, member, and viewer")

		userRoles := make(map[string]string)
		for _, m := range members {
			userRoles[m.UserID] = m.Role
		}

		assert.Equal(t, "owner", userRoles[owner])
		assert.Equal(t, "member", userRoles[member])
		assert.Equal(t, "viewer", userRoles[viewer])
	})
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// stringPtrTest is a test helper for creating string pointers
func stringPtrTest(s string) *string {
	return &s
}
