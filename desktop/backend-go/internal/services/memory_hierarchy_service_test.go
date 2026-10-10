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
// MEMORY HIERARCHY TESTS
// =============================================================================

// TestMemoryHierarchy tests workspace vs user memory isolation
func TestMemoryHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	db := testutil.RequireTestDatabase(t)
	defer db.Close()

	memoryService := NewMemoryHierarchyService(db.Pool)
	workspaceService := NewWorkspaceService(db.Pool)
	ctx := context.Background()

	// Setup test workspace
	owner := "test-owner-" + uuid.New().String()[:8]
	member := "test-member-" + uuid.New().String()[:8]
	viewer := "test-viewer-" + uuid.New().String()[:8]

	workspace, err := workspaceService.CreateWorkspace(ctx, CreateWorkspaceRequest{
		Name:     "Memory Test Workspace",
		PlanType: "free",
	}, owner)
	require.NoError(t, err)
	defer workspaceService.DeleteWorkspace(ctx, workspace.ID, owner)

	// Add members
	workspaceService.AddMember(ctx, workspace.ID, AddMemberRequest{UserID: member, Role: "member"}, owner)
	workspaceService.AddMember(ctx, workspace.ID, AddMemberRequest{UserID: viewer, Role: "viewer"}, owner)

	// Test: Create workspace memory (visible to all)
	t.Run("Workspace memory visible to all members", func(t *testing.T) {
		// Owner creates workspace memory
		wsMemory, err := memoryService.CreateWorkspaceMemory(
			ctx,
			workspace.ID,
			"Team Process",
			"Our team uses agile methodology",
			"process",
			owner,
			[]string{"agile", "process"},
			map[string]interface{}{"importance": "high"},
		)
		require.NoError(t, err)
		assert.Equal(t, "workspace", wsMemory.Visibility)

		// Verify all members can see it
		ownerMemories, err := memoryService.GetWorkspaceMemories(ctx, workspace.ID, owner, nil, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(ownerMemories), 1, "Owner should see workspace memory")

		memberMemories, err := memoryService.GetWorkspaceMemories(ctx, workspace.ID, member, nil, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(memberMemories), 1, "Member should see workspace memory")

		viewerMemories, err := memoryService.GetWorkspaceMemories(ctx, workspace.ID, viewer, nil, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(viewerMemories), 1, "Viewer should see workspace memory")
	})

	// Test: Create private memory (owner only)
	t.Run("Private memory only visible to owner", func(t *testing.T) {
		// Owner creates private memory
		privateMemory, err := memoryService.CreatePrivateMemory(
			ctx,
			workspace.ID,
			owner,
			"Personal Note",
			"My private thoughts about the project",
			"note",
			[]string{"private"},
			map[string]interface{}{},
		)
		require.NoError(t, err)
		assert.Equal(t, "private", privateMemory.Visibility)

		// Owner can see it
		ownerUserMemories, err := memoryService.GetUserMemories(ctx, workspace.ID, owner, nil, 100)
		require.NoError(t, err)
		foundPrivate := false
		for _, mem := range ownerUserMemories {
			if mem.ID == privateMemory.ID {
				foundPrivate = true
				break
			}
		}
		assert.True(t, foundPrivate, "Owner should see their private memory")

		// Member cannot see it
		memberUserMemories, err := memoryService.GetUserMemories(ctx, workspace.ID, member, nil, 100)
		require.NoError(t, err)
		foundByMember := false
		for _, mem := range memberUserMemories {
			if mem.ID == privateMemory.ID {
				foundByMember = true
				break
			}
		}
		assert.False(t, foundByMember, "Member should NOT see owner's private memory")
	})

	// Test: Share memory with specific users
	t.Run("Share memory with specific users", func(t *testing.T) {
		// Owner creates private memory
		privateMemory, err := memoryService.CreatePrivateMemory(
			ctx,
			workspace.ID,
			owner,
			"Shared Secret",
			"Confidential information for specific users",
			"knowledge",
			[]string{"confidential"},
			map[string]interface{}{},
		)
		require.NoError(t, err)

		// Share with member (but not viewer)
		err = memoryService.ShareMemory(ctx, privateMemory.ID, owner, []string{member})
		require.NoError(t, err)

		// Member should see it
		canAccess, err := memoryService.CanAccessMemory(ctx, member, privateMemory.ID)
		require.NoError(t, err)
		assert.True(t, canAccess, "Member should access shared memory")

		// Viewer should not see it
		canAccessViewer, err := memoryService.CanAccessMemory(ctx, viewer, privateMemory.ID)
		require.NoError(t, err)
		assert.False(t, canAccessViewer, "Viewer should NOT access memory not shared with them")
	})

	// Test: Unshare memory
	t.Run("Unshare memory", func(t *testing.T) {
		// Owner creates and shares memory
		privateMemory, err := memoryService.CreatePrivateMemory(
			ctx,
			workspace.ID,
			owner,
			"Temporarily Shared",
			"This will be unshared",
			"knowledge",
			[]string{},
			map[string]interface{}{},
		)
		require.NoError(t, err)

		// Share with member
		err = memoryService.ShareMemory(ctx, privateMemory.ID, owner, []string{member})
		require.NoError(t, err)

		// Verify member can access
		canAccess, err := memoryService.CanAccessMemory(ctx, member, privateMemory.ID)
		require.NoError(t, err)
		assert.True(t, canAccess)

		// Unshare
		err = memoryService.UnshareMemory(ctx, privateMemory.ID, owner)
		require.NoError(t, err)

		// Verify member can no longer access
		canAccessAfter, err := memoryService.CanAccessMemory(ctx, member, privateMemory.ID)
		require.NoError(t, err)
		assert.False(t, canAccessAfter, "Member should not access unshared memory")
	})

	// Test: Get accessible memories (workspace + private + shared)
	t.Run("Get all accessible memories", func(t *testing.T) {
		// Create various memory types
		wsMemory, _ := memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "WS1", "Workspace memory 1", "knowledge", owner, []string{}, nil)
		privateMemory, _ := memoryService.CreatePrivateMemory(ctx, workspace.ID, owner, "Private1", "Private memory 1", "note", []string{}, nil)
		sharedMemory, _ := memoryService.CreatePrivateMemory(ctx, workspace.ID, owner, "Shared1", "Shared memory 1", "knowledge", []string{}, nil)
		memoryService.ShareMemory(ctx, sharedMemory.ID, owner, []string{member})

		// Owner should see all their memories
		ownerAccessible, err := memoryService.GetAccessibleMemories(ctx, workspace.ID, owner, nil, 100)
		require.NoError(t, err)

		foundWS := false
		foundPrivate := false
		foundShared := false
		for _, mem := range ownerAccessible {
			if mem.ID == wsMemory.ID {
				foundWS = true
			}
			if mem.ID == privateMemory.ID {
				foundPrivate = true
			}
			if mem.ID == sharedMemory.ID {
				foundShared = true
			}
		}
		assert.True(t, foundWS, "Owner should see workspace memory")
		assert.True(t, foundPrivate, "Owner should see private memory")
		assert.True(t, foundShared, "Owner should see shared memory")

		// Member should see workspace and shared (not private)
		memberAccessible, err := memoryService.GetAccessibleMemories(ctx, workspace.ID, member, nil, 100)
		require.NoError(t, err)

		foundWSMember := false
		foundPrivateMember := false
		foundSharedMember := false
		for _, mem := range memberAccessible {
			if mem.ID == wsMemory.ID {
				foundWSMember = true
			}
			if mem.ID == privateMemory.ID {
				foundPrivateMember = true
			}
			if mem.ID == sharedMemory.ID {
				foundSharedMember = true
			}
		}
		assert.True(t, foundWSMember, "Member should see workspace memory")
		assert.False(t, foundPrivateMember, "Member should NOT see owner's private memory")
		assert.True(t, foundSharedMember, "Member should see shared memory")
	})

	// Test: Track memory access
	t.Run("Track memory access", func(t *testing.T) {
		wsMemory, _ := memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Access Test", "Test access tracking", "knowledge", owner, []string{}, nil)

		// Track multiple accesses
		for i := 0; i < 5; i++ {
			err := memoryService.TrackAccess(ctx, wsMemory.ID)
			require.NoError(t, err)
		}

		// Retrieve and verify access count
		memories, err := memoryService.GetWorkspaceMemories(ctx, workspace.ID, owner, nil, 100)
		require.NoError(t, err)

		for _, mem := range memories {
			if mem.ID == wsMemory.ID {
				assert.GreaterOrEqual(t, mem.AccessCount, 5, "Access count should be tracked")
				break
			}
		}
	})

	// Test: Filter by memory type
	t.Run("Filter memories by type", func(t *testing.T) {
		// Create memories of different types
		memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Decision 1", "A decision", "decision", owner, []string{}, nil)
		memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Process 1", "A process", "process", owner, []string{}, nil)
		memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Knowledge 1", "Some knowledge", "knowledge", owner, []string{}, nil)

		// Filter for decisions only
		decisionType := "decision"
		decisions, err := memoryService.GetWorkspaceMemories(ctx, workspace.ID, owner, &decisionType, 100)
		require.NoError(t, err)

		// All returned memories should be decisions
		for _, mem := range decisions {
			assert.Equal(t, "decision", mem.MemoryType, "Should only return decision memories")
		}
	})
}

// =============================================================================
// ACCESS CONTROL TESTS
// =============================================================================

// TestMemoryAccessControl tests memory access control enforcement
func TestMemoryAccessControl(t *testing.T) {
	t.Skip("Requires database - implement with testcontainers or run with live DB")

	var pool *pgxpool.Pool // Replace with actual pool
	memoryService := NewMemoryHierarchyService(pool)
	workspaceService := NewWorkspaceService(pool)
	ctx := context.Background()

	// Setup
	owner := "test-owner-" + uuid.New().String()[:8]
	member := "test-member-" + uuid.New().String()[:8]
	outsider := "test-outsider-" + uuid.New().String()[:8]

	workspace, err := workspaceService.CreateWorkspace(ctx, CreateWorkspaceRequest{
		Name:     "Access Control Test",
		PlanType: "free",
	}, owner)
	require.NoError(t, err)
	defer workspaceService.DeleteWorkspace(ctx, workspace.ID, owner)

	workspaceService.AddMember(ctx, workspace.ID, AddMemberRequest{UserID: member, Role: "member"}, owner)

	// Test: Workspace member can access workspace memory
	t.Run("Workspace member can access workspace memory", func(t *testing.T) {
		wsMemory, _ := memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Public Info", "Public information", "knowledge", owner, []string{}, nil)

		canAccess, err := memoryService.CanAccessMemory(ctx, member, wsMemory.ID)
		require.NoError(t, err)
		assert.True(t, canAccess, "Workspace member should access workspace memory")
	})

	// Test: Non-member cannot access workspace memory
	t.Run("Non-member cannot access workspace memory", func(t *testing.T) {
		wsMemory, _ := memoryService.CreateWorkspaceMemory(ctx, workspace.ID, "Public Info", "Public information", "knowledge", owner, []string{}, nil)

		canAccess, err := memoryService.CanAccessMemory(ctx, outsider, wsMemory.ID)
		require.NoError(t, err)
		assert.False(t, canAccess, "Non-member should NOT access workspace memory")
	})

	// Test: Only owner can access private memory
	t.Run("Only owner can access private memory", func(t *testing.T) {
		privateMemory, _ := memoryService.CreatePrivateMemory(ctx, workspace.ID, owner, "Secret", "Secret information", "knowledge", []string{}, nil)

		// Owner can access
		ownerCanAccess, err := memoryService.CanAccessMemory(ctx, owner, privateMemory.ID)
		require.NoError(t, err)
		assert.True(t, ownerCanAccess, "Owner should access their private memory")

		// Member cannot access
		memberCanAccess, err := memoryService.CanAccessMemory(ctx, member, privateMemory.ID)
		require.NoError(t, err)
		assert.False(t, memberCanAccess, "Other members should NOT access private memory")
	})

	// Test: Shared memory access control
	t.Run("Shared memory access control", func(t *testing.T) {
		sharedMemory, _ := memoryService.CreatePrivateMemory(ctx, workspace.ID, owner, "Selective Share", "Shared with specific users", "knowledge", []string{}, nil)
		memoryService.ShareMemory(ctx, sharedMemory.ID, owner, []string{member})

		// Owner can access (creator)
		ownerCanAccess, err := memoryService.CanAccessMemory(ctx, owner, sharedMemory.ID)
		require.NoError(t, err)
		assert.True(t, ownerCanAccess)

		// Member can access (shared with)
		memberCanAccess, err := memoryService.CanAccessMemory(ctx, member, sharedMemory.ID)
		require.NoError(t, err)
		assert.True(t, memberCanAccess)

		// Outsider cannot access
		outsiderCanAccess, err := memoryService.CanAccessMemory(ctx, outsider, sharedMemory.ID)
		require.NoError(t, err)
		assert.False(t, outsiderCanAccess)
	})
}

// =============================================================================
// INTEGRATION WITH CHAT AGENT TESTS
// =============================================================================

// TestMemoryWithChatAgent tests memory integration with chat agents
func TestMemoryWithChatAgent(t *testing.T) {
	t.Skip("Requires database and agent integration - implement separately")

	// TODO: This test would verify:
	// 1. Agent retrieves workspace memories for context
	// 2. Agent respects memory visibility (private vs shared)
	// 3. Agent can create new workspace memories
	// 4. Agent cannot access memories outside workspace
	// 5. Role-based memory access (viewer vs member)

	// Example test flow:
	// - Create workspace with owner and member
	// - Create workspace memory about "coding standards"
	// - Create private memory (owner only)
	// - Send chat message from member
	// - Verify agent uses workspace memory in response
	// - Verify agent doesn't expose private memory
	// - Send chat from owner
	// - Verify agent can reference private memory for owner
}
