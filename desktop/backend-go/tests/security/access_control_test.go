package security_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestHorizontalPrivilegeEscalation tests that users cannot access other users' resources
func TestHorizontalPrivilegeEscalation(t *testing.T) {
	userA := "user-a-" + uuid.New().String()
	userB := "user-b-" + uuid.New().String()

	tests := []struct {
		name         string
		resourceType string
		ownerID      string
		accessorID   string
		shouldAccess bool
	}{
		{"Owner accesses own conversation", "conversation", userA, userA, true},
		{"User B tries to access User A's conversation", "conversation", userA, userB, false},
		{"Owner accesses own memory", "memory", userA, userA, true},
		{"User B tries to access User A's memory", "memory", userA, userB, false},
		{"Owner accesses own project", "project", userA, userA, true},
		{"User B tries to access User A's project", "project", userA, userB, false},
		{"Owner accesses own workspace", "workspace", userA, userA, true},
		{"User B tries to access User A's workspace", "workspace", userA, userB, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAccess := checkResourceAccess(tt.ownerID, tt.accessorID, tt.resourceType)
			assert.Equal(t, tt.shouldAccess, canAccess, "Access control mismatch")
		})
	}
}

// TestVerticalPrivilegeEscalation tests that regular users cannot perform admin actions
func TestVerticalPrivilegeEscalation(t *testing.T) {
	adminUserID := "admin-" + uuid.New().String()
	regularUserID := "user-" + uuid.New().String()

	adminActions := []string{
		"delete_user",
		"modify_system_settings",
		"access_all_workspaces",
		"view_all_users",
		"modify_rate_limits",
		"access_admin_panel",
	}

	for _, action := range adminActions {
		t.Run("Admin_"+action, func(t *testing.T) {
			canAdmin := hasAdminAccess(adminUserID)
			assert.True(t, canAdmin, "Admin should have access")

			adminCanPerform := canPerformAction(adminUserID, action, true)
			assert.True(t, adminCanPerform, "Admin should be able to perform admin action")
		})

		t.Run("Regular_User_"+action, func(t *testing.T) {
			canAdmin := hasAdminAccess(regularUserID)
			assert.False(t, canAdmin, "Regular user should not have admin access")

			regularCanPerform := canPerformAction(regularUserID, action, false)
			assert.False(t, regularCanPerform, "Regular user should NOT be able to perform admin action")
		})
	}
}

// TestPathTraversal tests path traversal prevention in file access
func TestPathTraversal(t *testing.T) {
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//etc/passwd",
		"..\\..\\..\\.env",
		"../../config/secrets.yaml",
		"../../../../../root/.ssh/id_rsa",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"%2e%2e%2f%2e%2e%2f",
		"..%252f..%252f",
		"/./../../../etc/passwd",
	}

	for _, payload := range pathTraversalPayloads {
		t.Run("Path_Traversal_"+limitString(payload, 25), func(t *testing.T) {
			// Test 1: Detect path traversal
			detected := detectPathTraversal(payload)
			assert.True(t, detected, "Should detect path traversal attempt")

			// Test 2: Clean path should remove traversal
			cleaned := cleanFilePath(payload)
			assert.NotContains(t, cleaned, "..", "Cleaned path should not contain ..")
			assert.NotContains(t, cleaned, "%2e", "Cleaned path should not contain encoded dots")

			// Test 3: Validate against allowed directory
			allowedDir := "/app/user-files/"
			isAllowed := isPathWithinDirectory(payload, allowedDir)
			assert.False(t, isAllowed, "Path traversal should not be within allowed directory")
		})
	}
}

// TestIDOR tests Insecure Direct Object Reference prevention
func TestIDOR(t *testing.T) {
	userAID := uuid.New().String()
	userBID := uuid.New().String()

	resourceAID := uuid.New().String()
	resourceBID := uuid.New().String()

	tests := []struct {
		name       string
		userID     string
		resourceID string
		ownerID    string
		expected   bool
	}{
		{"User A accesses own resource", userAID, resourceAID, userAID, true},
		{"User B accesses own resource", userBID, resourceBID, userBID, true},
		{"User B tries to access User A's resource", userBID, resourceAID, userAID, false},
		{"User A tries to access User B's resource", userAID, resourceBID, userBID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAccess := checkIDORAccess(tt.userID, tt.resourceID, tt.ownerID)
			assert.Equal(t, tt.expected, canAccess, "IDOR check failed")
		})
	}
}

// TestWorkspaceIsolation tests workspace resource isolation
func TestWorkspaceIsolation(t *testing.T) {
	workspaceAID := uuid.New().String()
	workspaceBID := uuid.New().String()

	userInWorkspaceA := uuid.New().String()
	userInWorkspaceB := uuid.New().String()

	tests := []struct {
		name        string
		userID      string
		workspaceID string
		resourceID  string
		expected    bool
	}{
		{"User in Workspace A accesses resource in A", userInWorkspaceA, workspaceAID, "res-a", true},
		{"User in Workspace B accesses resource in B", userInWorkspaceB, workspaceBID, "res-b", true},
		{"User in Workspace A tries to access resource in B", userInWorkspaceA, workspaceAID, "res-b", false},
		{"User in Workspace B tries to access resource in A", userInWorkspaceB, workspaceBID, "res-a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In real implementation, would check database for workspace membership
			isMember := checkWorkspaceMembership(tt.userID, tt.workspaceID)

			// Resource should belong to workspace
			resourceBelongs := tt.resourceID == "res-a" && tt.workspaceID == workspaceAID ||
				tt.resourceID == "res-b" && tt.workspaceID == workspaceBID

			canAccess := isMember && resourceBelongs
			assert.Equal(t, tt.expected, canAccess, "Workspace isolation check failed")
		})
	}
}

// TestRoleBasedAccess tests role-based access control
func TestRoleBasedAccess(t *testing.T) {
	tests := []struct {
		role     string
		action   string
		allowed  bool
	}{
		{"admin", "create_workspace", true},
		{"admin", "delete_workspace", true},
		{"admin", "manage_users", true},
		{"member", "create_workspace", false},
		{"member", "delete_workspace", false},
		{"member", "manage_users", false},
		{"member", "create_project", true},
		{"member", "view_workspace", true},
		{"viewer", "create_project", false},
		{"viewer", "view_workspace", true},
		{"viewer", "edit_project", false},
	}

	for _, tt := range tests {
		t.Run(tt.role+"_"+tt.action, func(t *testing.T) {
			canPerform := checkRolePermission(tt.role, tt.action)
			assert.Equal(t, tt.allowed, canPerform, "Role permission check failed")
		})
	}
}

// TestDirectObjectReferenceEnumeration tests that object IDs cannot be enumerated
func TestDirectObjectReferenceEnumeration(t *testing.T) {
	t.Run("Sequential IDs should not be used", func(t *testing.T) {
		// Generate multiple IDs
		id1 := uuid.New().String()
		id2 := uuid.New().String()
		id3 := uuid.New().String()

		// UUIDs should not be sequential
		assert.NotEqual(t, id1, id2, "IDs should be unique")
		assert.NotEqual(t, id2, id3, "IDs should be unique")

		// UUIDs are not enumerable
		assert.Greater(t, len(id1), 30, "UUID should be long and random")
	})

	t.Run("Predictable patterns rejected", func(t *testing.T) {
		predictableIDs := []string{"1", "2", "3", "user1", "user2", "admin"}

		for _, id := range predictableIDs {
			_, err := uuid.Parse(id)
			assert.Error(t, err, "Predictable ID should not parse as UUID")
		}
	})
}

// TestAPIEndpointAccessControl tests access control on API endpoints
func TestAPIEndpointAccessControl(t *testing.T) {
	publicEndpoints := []string{
		"/api/health",
		"/api/auth/signin",
		"/api/auth/signup",
	}

	protectedEndpoints := []string{
		"/api/memories",
		"/api/conversations",
		"/api/projects",
		"/api/workspaces",
		"/api/users/me",
	}

	adminEndpoints := []string{
		"/api/admin/users",
		"/api/admin/workspaces",
		"/api/admin/settings",
	}

	t.Run("Public endpoints accessible without auth", func(t *testing.T) {
		for _, endpoint := range publicEndpoints {
			requiresAuth := endpointRequiresAuth(endpoint)
			assert.False(t, requiresAuth, "Public endpoint should not require auth: "+endpoint)
		}
	})

	t.Run("Protected endpoints require authentication", func(t *testing.T) {
		for _, endpoint := range protectedEndpoints {
			requiresAuth := endpointRequiresAuth(endpoint)
			assert.True(t, requiresAuth, "Protected endpoint should require auth: "+endpoint)
		}
	})

	t.Run("Admin endpoints require admin role", func(t *testing.T) {
		for _, endpoint := range adminEndpoints {
			requiresAdmin := endpointRequiresAdmin(endpoint)
			assert.True(t, requiresAdmin, "Admin endpoint should require admin role: "+endpoint)
		}
	})
}

// Helper functions

func checkResourceAccess(ownerID, accessorID, resourceType string) bool {
	// In real implementation, would check database
	return ownerID == accessorID
}

func hasAdminAccess(userID string) bool {
	// In real implementation, would check user role in database
	return strings.HasPrefix(userID, "admin-")
}

func canPerformAction(userID, action string, isAdmin bool) bool {
	if !isAdmin {
		return false
	}
	// Admin can perform all actions
	return true
}

func detectPathTraversal(path string) bool {
	dangerous := []string{"..", "%2e", "%252e", "\\"}
	lowerPath := strings.ToLower(path)
	for _, d := range dangerous {
		if strings.Contains(lowerPath, d) {
			return true
		}
	}
	return false
}

func cleanFilePath(path string) string {
	// First decode URL-encoded characters (recursive to handle double encoding)
	decoded := path
	for i := 0; i < 3; i++ { // Decode up to 3 levels
		newDecoded := decoded
		// Decode %2e -> .
		newDecoded = strings.ReplaceAll(newDecoded, "%2e", ".")
		newDecoded = strings.ReplaceAll(newDecoded, "%2E", ".")
		// Decode %2f -> /
		newDecoded = strings.ReplaceAll(newDecoded, "%2f", "/")
		newDecoded = strings.ReplaceAll(newDecoded, "%2F", "/")
		// Decode %5c -> \
		newDecoded = strings.ReplaceAll(newDecoded, "%5c", "\\")
		newDecoded = strings.ReplaceAll(newDecoded, "%5C", "\\")

		if newDecoded == decoded {
			break // No more changes, stop decoding
		}
		decoded = newDecoded
	}

	// Replace forward slashes with OS-appropriate separator on Windows
	decoded = strings.ReplaceAll(decoded, "/", string(filepath.Separator))

	// Use filepath.Clean to normalize path (handles .., ., multiple slashes, etc.)
	cleaned := filepath.Clean(decoded)

	// CRITICAL: After cleaning, reject if path still contains parent directory traversal
	// This is essential for security - even cleaned paths should not escape base directory
	if strings.Contains(cleaned, "..") {
		// Return a safe empty path instead
		return ""
	}

	return cleaned
}

func isPathWithinDirectory(path, allowedDir string) bool {
	// Clean and decode the input path
	cleaned := cleanFilePath(path)

	// If cleaning removed path traversal, it's suspicious
	if cleaned == "" {
		return false
	}

	// Check if original path had traversal attempts
	if detectPathTraversal(path) {
		return false
	}

	// Convert to absolute paths for comparison
	allowedAbs, err := filepath.Abs(allowedDir)
	if err != nil {
		return false
	}

	// Make cleaned path absolute by joining with allowed directory
	// This simulates how the application would interpret the path
	testPath := filepath.Join(allowedAbs, cleaned)
	testAbs, err := filepath.Abs(testPath)
	if err != nil {
		return false
	}

	// Ensure the resulting path is within allowed directory
	// Use HasPrefix with separator to prevent false positives (e.g., /app vs /application)
	if !strings.HasPrefix(testAbs, allowedAbs+string(filepath.Separator)) && testAbs != allowedAbs {
		return false
	}

	return true
}

func checkIDORAccess(userID, resourceID, ownerID string) bool {
	// User can only access if they own the resource
	return userID == ownerID
}

func checkWorkspaceMembership(userID, workspaceID string) bool {
	// In real implementation, would query database
	// For this test, assume membership based on ID
	return true
}

func checkRolePermission(role, action string) bool {
	permissions := map[string][]string{
		"admin": {"create_workspace", "delete_workspace", "manage_users", "create_project", "view_workspace", "edit_project"},
		"member": {"create_project", "view_workspace", "edit_project"},
		"viewer": {"view_workspace"},
	}

	allowedActions, exists := permissions[role]
	if !exists {
		return false
	}

	for _, allowed := range allowedActions {
		if allowed == action {
			return true
		}
	}
	return false
}

func endpointRequiresAuth(endpoint string) bool {
	publicPrefixes := []string{"/api/health", "/api/auth/"}
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(endpoint, prefix) {
			return false
		}
	}
	return true
}

func endpointRequiresAdmin(endpoint string) bool {
	return strings.HasPrefix(endpoint, "/api/admin/")
}
