package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoleContextService_GetUserRoleContext tests role context retrieval
func TestRoleContextService_GetUserRoleContext(t *testing.T) {
	// This is a template test - requires database setup
	// Run with: go test -v ./internal/services -run TestRoleContextService

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	service := NewRoleContextService(db.Pool)
	ctx := context.Background()

	// Create test workspace
	workspaceID := uuid.New()
	userID := "test-user-123"

	// Test: Get role context for owner
	roleCtx, err := service.GetUserRoleContext(ctx, userID, workspaceID)
	require.NoError(t, err)
	assert.Equal(t, "owner", roleCtx.RoleName)
	assert.Equal(t, 1, roleCtx.HierarchyLevel)
}

// TestUserRoleContext_HasPermission tests permission checking
func TestUserRoleContext_HasPermission(t *testing.T) {
	roleCtx := &UserRoleContext{
		UserID:          "user-123",
		WorkspaceID:     uuid.New(),
		RoleName:        "member",
		RoleDisplayName: "Member",
		HierarchyLevel:  4,
		Permissions: map[string]map[string]interface{}{
			"projects": {
				"create": true,
				"read":   true,
				"update": true,
				"delete": false,
			},
			"tasks": {
				"create": true,
				"read":   true,
			},
		},
	}

	tests := []struct {
		name       string
		resource   string
		permission string
		expected   bool
	}{
		{
			name:       "Has permission to create projects",
			resource:   "projects",
			permission: "create",
			expected:   true,
		},
		{
			name:       "Does not have permission to delete projects",
			resource:   "projects",
			permission: "delete",
			expected:   false,
		},
		{
			name:       "Missing resource returns false",
			resource:   "billing",
			permission: "manage",
			expected:   false,
		},
		{
			name:       "Missing permission returns false",
			resource:   "tasks",
			permission: "delete",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roleCtx.HasPermission(tt.resource, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyRolePermissionsUsesCanonicalRoleJSON(t *testing.T) {
	roleCtx := &UserRoleContext{Permissions: make(map[string]map[string]interface{})}

	applyRolePermissions(roleCtx, map[string]interface{}{
		"workspace": map[string]interface{}{
			"invite_members":   true,
			"delete_workspace": false,
		},
		"projects": []interface{}{"read", "create"},
	})

	assert.True(t, roleCtx.HasPermission("workspace", "invite_members"))
	assert.False(t, roleCtx.HasPermission("workspace", "delete_workspace"))
	assert.True(t, roleCtx.HasPermission("projects", "read"))
	assert.True(t, roleCtx.HasPermission("projects", "create"))
}

// TestUserRoleContext_IsAtLeastLevel tests hierarchy level checking
func TestUserRoleContext_IsAtLeastLevel(t *testing.T) {
	tests := []struct {
		name          string
		userLevel     int
		requiredLevel int
		expected      bool
	}{
		{
			name:          "Owner (1) meets admin requirement (2)",
			userLevel:     1,
			requiredLevel: 2,
			expected:      true,
		},
		{
			name:          "Admin (2) meets admin requirement (2)",
			userLevel:     2,
			requiredLevel: 2,
			expected:      true,
		},
		{
			name:          "Member (4) does not meet admin requirement (2)",
			userLevel:     4,
			requiredLevel: 2,
			expected:      false,
		},
		{
			name:          "Viewer (5) meets member requirement (4)",
			userLevel:     5,
			requiredLevel: 6,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleCtx := &UserRoleContext{
				HierarchyLevel: tt.userLevel,
			}
			result := roleCtx.IsAtLeastLevel(tt.requiredLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUserRoleContext_FormatCanDo tests permission formatting
func TestUserRoleContext_FormatCanDo(t *testing.T) {
	roleCtx := &UserRoleContext{
		Permissions: map[string]map[string]interface{}{
			"projects": {
				"create": true,
				"read":   true,
			},
			"tasks": {
				"create": true,
			},
		},
	}

	result := roleCtx.formatCanDo()
	assert.Contains(t, result, "projects")
	assert.Contains(t, result, "tasks")
	assert.Contains(t, result, "create")
	assert.Contains(t, result, "read")
}

// TestUserRoleContext_FormatCannotDo tests restriction formatting
func TestUserRoleContext_FormatCannotDo(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		expected []string
	}{
		{
			name:     "Viewer has many restrictions",
			roleName: "viewer",
			expected: []string{"Create or modify projects", "Delete any resources"},
		},
		{
			name:     "Owner has no restrictions",
			roleName: "owner",
			expected: []string{"None (full workspace access)"},
		},
		{
			name:     "Member has some restrictions",
			roleName: "member",
			expected: []string{"Delete workspace", "Modify role permissions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleCtx := &UserRoleContext{
				RoleName:    tt.roleName,
				Permissions: make(map[string]map[string]interface{}),
			}
			result := roleCtx.formatCannotDo()
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp)
			}
		})
	}
}

// TestUserRoleContext_GetRoleContextPrompt tests prompt generation
func TestUserRoleContext_GetRoleContextPrompt(t *testing.T) {
	roleCtx := &UserRoleContext{
		UserID:          "user-123",
		RoleName:        "member",
		RoleDisplayName: "Member",
		HierarchyLevel:  4,
		Title:           "Senior Developer",
		Department:      "Engineering",
		Permissions: map[string]map[string]interface{}{
			"projects": {
				"create": true,
				"read":   true,
			},
		},
	}

	prompt := roleCtx.GetRoleContextPrompt()

	// Verify all important elements are present
	assert.Contains(t, prompt, "user-123")
	assert.Contains(t, prompt, "Member")
	assert.Contains(t, prompt, "Senior Developer")
	assert.Contains(t, prompt, "Engineering")

	// Updated assertions for new format
	assert.Contains(t, prompt, "CRITICAL: USER ROLE & PERMISSIONS CONTEXT")
	assert.Contains(t, prompt, "PERMISSIONS GRANTED TO THIS USER")
	assert.Contains(t, prompt, "ACTIONS RESTRICTED FROM THIS USER")
	assert.Contains(t, prompt, "MANDATORY BEHAVIOR")
	assert.Contains(t, prompt, "When the user asks \"what can I do?\"")
	assert.Contains(t, prompt, "ALWAYS acknowledge their role")
	assert.Contains(t, prompt, "ONLY suggest actions that are within their permission set")

	// Verify the prompt is prominent (has visual markers)
	assert.Contains(t, prompt, "═══════════════════")
	assert.Contains(t, prompt, "🔐")
	assert.Contains(t, prompt, "🎯")

	// Print prompt for manual inspection during development
	t.Logf("Generated prompt:\n%s", prompt)
}

// TestUserRoleContext_GetProjectRole tests project-specific role retrieval
func TestUserRoleContext_GetProjectRole(t *testing.T) {
	projectID := uuid.New()
	roleCtx := &UserRoleContext{
		ProjectRoles: map[uuid.UUID]string{
			projectID: "lead",
		},
	}

	// Test: Project role exists
	role, ok := roleCtx.GetProjectRole(projectID)
	assert.True(t, ok)
	assert.Equal(t, "lead", role)

	// Test: Project role does not exist
	otherProjectID := uuid.New()
	role, ok = roleCtx.GetProjectRole(otherProjectID)
	assert.False(t, ok)
	assert.Empty(t, role)
}

// BenchmarkHasPermission benchmarks permission checking
func BenchmarkHasPermission(b *testing.B) {
	roleCtx := &UserRoleContext{
		Permissions: map[string]map[string]interface{}{
			"projects": {
				"create": true,
				"read":   true,
				"update": true,
				"delete": false,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roleCtx.HasPermission("projects", "create")
	}
}

// BenchmarkIsAtLeastLevel benchmarks hierarchy checking
func BenchmarkIsAtLeastLevel(b *testing.B) {
	roleCtx := &UserRoleContext{
		HierarchyLevel: 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roleCtx.IsAtLeastLevel(2)
	}
}
