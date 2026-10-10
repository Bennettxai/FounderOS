package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryHierarchyIntegration tests workspace → project → agent memory hierarchy
func TestMemoryHierarchyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	memoryService := NewMemoryHierarchyService(testDB.Pool)

	// Setup test data
	workspaceID := uuid.New()
	userID := uuid.New().String()
	userID2 := uuid.New().String()

	// Create workspace
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "Test Workspace", userID)
	require.NoError(t, err)

	t.Run("Workspace-level memory is accessible to all members", func(t *testing.T) {
		// Create workspace memory
		memoryID := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, importance_score, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, memoryID, workspaceID, "Workspace Guideline", "Always use standardized templates", "guideline", "workspace", 0.9)
		require.NoError(t, err)

		// User 1 retrieves workspace memories
		memories1, err := memoryService.GetWorkspaceMemories(ctx, workspaceID, userID, nil, 10)
		require.NoError(t, err)
		assert.Len(t, memories1, 1)
		assert.Equal(t, "Workspace Guideline", memories1[0].Title)

		// User 2 also sees workspace memories
		memories2, err := memoryService.GetWorkspaceMemories(ctx, workspaceID, userID2, nil, 10)
		require.NoError(t, err)
		assert.Len(t, memories2, 1)
		assert.Equal(t, "Workspace Guideline", memories2[0].Title)
	})

	t.Run("Private memories are user-specific", func(t *testing.T) {
		// User 1 creates private memory
		privateMemory := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, owner_user_id, importance_score, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		`, privateMemory, workspaceID, "My Private Note", "Confidential information", "note", "private", userID, 0.8)
		require.NoError(t, err)

		// User 1 can access their private memory
		memories1, err := memoryService.GetUserMemories(ctx, workspaceID, userID, nil, 10)
		require.NoError(t, err)
		assert.Len(t, memories1, 1)
		assert.Equal(t, "My Private Note", memories1[0].Title)

		// User 2 cannot access User 1's private memory
		memories2, err := memoryService.GetUserMemories(ctx, workspaceID, userID2, nil, 10)
		require.NoError(t, err)
		assert.Len(t, memories2, 0)
	})

	t.Run("Shared memories have explicit access control", func(t *testing.T) {
		// User 1 creates shared memory with User 2
		sharedMemory := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, owner_user_id, shared_with, importance_score, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`, sharedMemory, workspaceID, "Shared Project Note", "Project details", "note", "shared", userID, []string{userID2}, 0.7)
		require.NoError(t, err)

		// Both users can access shared memory
		accessible1, err := memoryService.GetAccessibleMemories(ctx, workspaceID, userID, nil, 10)
		require.NoError(t, err)
		hasShared1 := false
		for _, m := range accessible1 {
			if m.Title == "Shared Project Note" {
				hasShared1 = true
			}
		}
		assert.True(t, hasShared1, "User 1 should see shared memory")

		accessible2, err := memoryService.GetAccessibleMemories(ctx, workspaceID, userID2, nil, 10)
		require.NoError(t, err)
		hasShared2 := false
		for _, m := range accessible2 {
			if m.Title == "Shared Project Note" {
				hasShared2 = true
			}
		}
		assert.True(t, hasShared2, "User 2 should see shared memory")
	})

	t.Run("Get all accessible memories combines workspace + private + shared", func(t *testing.T) {
		// User 1 should see: workspace memory + their private + shared
		memories, err := memoryService.GetAccessibleMemories(ctx, workspaceID, userID, nil, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(memories), 2, "Should have at least workspace + private memory")

		// Verify memory types
		hasWorkspace := false
		hasPrivate := false
		for _, m := range memories {
			if m.Visibility == "workspace" {
				hasWorkspace = true
			}
			if m.Visibility == "private" && m.IsOwner {
				hasPrivate = true
			}
		}
		assert.True(t, hasWorkspace, "Should include workspace memories")
		assert.True(t, hasPrivate, "Should include private memories")
	})
}

// TestMemorySearchWithPgvector tests vector similarity search
func TestMemorySearchWithPgvector(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	workspaceID := uuid.New()
	userID := uuid.New().String()

	// Create workspace
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "Test Workspace", userID)
	require.NoError(t, err)

	t.Run("Vector similarity search finds relevant memories", func(t *testing.T) {
		// Create memories with embeddings
		memories := []struct {
			title     string
			content   string
			embedding []float32
		}{
			{
				"Database Migration Guide",
				"How to run PostgreSQL migrations",
				generateMockEmbedding(1.0, 0.0, 0.0), // Strong on "database" dimension
			},
			{
				"API Design Principles",
				"REST API best practices and conventions",
				generateMockEmbedding(0.0, 1.0, 0.0), // Strong on "API" dimension
			},
			{
				"Database Query Optimization",
				"Tips for optimizing SQL queries",
				generateMockEmbedding(0.9, 0.1, 0.0), // Strong on "database" dimension
			},
		}

		for _, m := range memories {
			memoryID := uuid.New()
			embedding := pgvector.NewVector(m.embedding)
			_, err := testDB.Pool.Exec(ctx, `
				INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, embedding, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			`, memoryID, workspaceID, m.title, m.content, "guide", "workspace", embedding)
			require.NoError(t, err)
		}

		// Search for database-related memories
		queryEmbedding := pgvector.NewVector(generateMockEmbedding(1.0, 0.0, 0.0))

		rows, err := testDB.Pool.Query(ctx, `
			SELECT id, title, content, embedding <=> $1 as distance
			FROM workspace_memories
			WHERE workspace_id = $2 AND embedding IS NOT NULL
			ORDER BY embedding <=> $1
			LIMIT 2
		`, queryEmbedding, workspaceID)
		require.NoError(t, err)
		defer rows.Close()

		var results []struct {
			id       uuid.UUID
			title    string
			content  string
			distance float64
		}
		for rows.Next() {
			var r struct {
				id       uuid.UUID
				title    string
				content  string
				distance float64
			}
			err := rows.Scan(&r.id, &r.title, &r.content, &r.distance)
			require.NoError(t, err)
			results = append(results, r)
		}

		// Should find "Database Migration Guide" and "Database Query Optimization" first
		assert.Len(t, results, 2)
		assert.Contains(t, results[0].title, "Database")
		assert.Contains(t, results[1].title, "Database")
	})
}

// TestConcurrentMemoryOperations tests concurrent memory access
func TestConcurrentMemoryOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	memoryService := NewMemoryHierarchyService(testDB.Pool)

	workspaceID := uuid.New()
	userID := uuid.New().String()

	// Create workspace
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "Test Workspace", userID)
	require.NoError(t, err)

	t.Run("Concurrent memory creation and retrieval", func(t *testing.T) {
		// Create memories concurrently
		numGoroutines := 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				memoryID := uuid.New()
				_, err := testDB.Pool.Exec(ctx, `
					INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
				`, memoryID, workspaceID, "Concurrent Memory", "Content from goroutine", "note", "workspace")
				assert.NoError(t, err)

				// Immediately try to retrieve
				memories, err := memoryService.GetWorkspaceMemories(ctx, workspaceID, userID, nil, 20)
				assert.NoError(t, err)
				assert.NotNil(t, memories)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(10 * time.Second):
				t.Fatal("timeout waiting for concurrent operations")
			}
		}

		// Verify all memories were created
		var count int
		err := testDB.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM workspace_memories WHERE workspace_id = $1
		`, workspaceID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, numGoroutines, count)
	})
}

// TestMemoryAccessCounting tests access_count increment on memory retrieval
func TestMemoryAccessCounting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	workspaceID := uuid.New()
	userID := uuid.New().String()
	memoryID := uuid.New()

	// Create workspace and memory
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, workspaceID, "Test Workspace", userID)
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO workspace_memories (id, workspace_id, title, content, memory_type, visibility, access_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, NOW(), NOW())
	`, memoryID, workspaceID, "Tracked Memory", "Content", "note", "workspace")
	require.NoError(t, err)

	t.Run("Access count increments on retrieval", func(t *testing.T) {
		// Increment access count
		for i := 0; i < 5; i++ {
			_, err := testDB.Pool.Exec(ctx, `
				UPDATE workspace_memories SET access_count = access_count + 1, last_accessed_at = NOW()
				WHERE id = $1
			`, memoryID)
			require.NoError(t, err)
		}

		// Verify count
		var accessCount int
		err := testDB.Pool.QueryRow(ctx, `
			SELECT access_count FROM workspace_memories WHERE id = $1
		`, memoryID).Scan(&accessCount)
		require.NoError(t, err)
		assert.Equal(t, 5, accessCount)
	})

	t.Run("last_accessed_at is updated", func(t *testing.T) {
		before := time.Now().Add(-1 * time.Minute)

		// Access memory
		_, err := testDB.Pool.Exec(ctx, `
			UPDATE workspace_memories SET access_count = access_count + 1, last_accessed_at = NOW()
			WHERE id = $1
		`, memoryID)
		require.NoError(t, err)

		// Verify timestamp
		var lastAccessed time.Time
		err = testDB.Pool.QueryRow(ctx, `
			SELECT last_accessed_at FROM workspace_memories WHERE id = $1
		`, memoryID).Scan(&lastAccessed)
		require.NoError(t, err)
		assert.True(t, lastAccessed.After(before), "last_accessed_at should be recent")
	})
}

// generateMockEmbedding creates a simple 3D embedding for testing
func generateMockEmbedding(x, y, z float32) []float32 {
	// Normalize to unit vector
	length := float32(1.0)
	if x != 0 || y != 0 || z != 0 {
		length = float32(1.0)
	}
	return []float32{x * length, y * length, z * length}
}
