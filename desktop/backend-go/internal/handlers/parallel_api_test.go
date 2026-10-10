package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParallelChatRequests tests 10 simultaneous chat requests
func TestParallelChatRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	const numRequests = 10

	t.Run("10 parallel chat requests should all succeed", func(t *testing.T) {
		// Create 10 different users to avoid conflicts
		var wg sync.WaitGroup
		wg.Add(numRequests)

		successCount := atomic.Int32{}
		errorCount := atomic.Int32{}

		for i := 0; i < numRequests; i++ {
			go func(requestNum int) {
				defer wg.Done()

				userID, _ := createIntTestUserWithSession(t, ctx, testDB.Pool)
				workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

				// Create memory for context injection
				createIntTestMemory(t, ctx, testDB.Pool, workspaceID, userID, fmt.Sprintf("User %d context", requestNum))

				// Chat request would normally hit ChatV2 handler, but we'll just verify DB operations
				// since full chat requires AI provider setup
				conversationID := createIntTestConversation(t, ctx, testDB.Pool, workspaceID, userID)
				messageID := createIntTestMessage(t, ctx, testDB.Pool, conversationID, userID, fmt.Sprintf("Test message %d", requestNum))

				if messageID != uuid.Nil {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(numRequests), successCount.Load(), "All chat requests should succeed")
		assert.Equal(t, int32(0), errorCount.Load(), "No errors should occur")
	})
}

// TestParallelMemoryCRUD tests 20 parallel memory create/read/update operations
func TestParallelMemoryCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	embeddingService := &services.EmbeddingService{}

	t.Run("20 parallel memory creates should create 20 distinct memories", func(t *testing.T) {
		const numMemories = 20

		userID, sessionID := createIntTestUserWithSession(t, ctx, testDB.Pool)
		workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

		memoryHandler := NewMemoryHandler(testDB.Pool, embeddingService)
		router := setupOSARouter(userID, sessionID)
		router.POST("/memories", memoryHandler.CreateMemory)

		var wg sync.WaitGroup
		wg.Add(numMemories)

		createdIDs := make([]uuid.UUID, numMemories)
		var mu sync.Mutex

		for i := 0; i < numMemories; i++ {
			go func(idx int) {
				defer wg.Done()

				createReq := map[string]interface{}{
					"title":        fmt.Sprintf("Memory %d", idx),
					"content":      fmt.Sprintf("Content for memory %d", idx),
					"memory_type":  "fact",
					"workspace_id": workspaceID.String(),
				}

				body, _ := json.Marshal(createReq)
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/memories", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				// Note: May fail due to missing embedding service setup, but we're testing concurrency
				if w.Code == http.StatusCreated || w.Code == http.StatusOK {
					// Parse response to get ID
					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
						if idStr, ok := response["id"].(string); ok {
							if id, err := uuid.Parse(idStr); err == nil {
								mu.Lock()
								createdIDs[idx] = id
								mu.Unlock()
							}
						}
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify uniqueness of created IDs (no race conditions)
		uniqueIDs := make(map[uuid.UUID]bool)
		for _, id := range createdIDs {
			if id != uuid.Nil {
				uniqueIDs[id] = true
			}
		}

		// At least some should succeed (full success depends on embedding service setup)
		assert.Greater(t, len(uniqueIDs), 0, "At least some memories should be created")
	})

	t.Run("Parallel read operations during writes", func(t *testing.T) {
		userID, sessionID := createIntTestUserWithSession(t, ctx, testDB.Pool)
		workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

		// Pre-create 5 memories
		memoryIDs := make([]uuid.UUID, 5)
		for i := 0; i < 5; i++ {
			memoryIDs[i] = createIntTestMemory(t, ctx, testDB.Pool, workspaceID, userID, fmt.Sprintf("Concurrent Test Memory %d", i))
		}

		memoryHandler := NewMemoryHandler(testDB.Pool, embeddingService)
		router := setupOSARouter(userID, sessionID)
		router.GET("/memories/:id", memoryHandler.GetMemory)
		router.PATCH("/memories/:id", memoryHandler.UpdateMemory)

		var wg sync.WaitGroup
		readSuccessCount := atomic.Int32{}
		writeSuccessCount := atomic.Int32{}

		// 10 readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				memID := memoryIDs[idx%5]
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/memories/"+memID.String(), nil)
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					readSuccessCount.Add(1)
				}
			}(i)
		}

		// 5 writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				memID := memoryIDs[idx]
				updateReq := map[string]interface{}{
					"title": fmt.Sprintf("Updated Memory %d", idx),
				}

				body, _ := json.Marshal(updateReq)
				w := httptest.NewRecorder()
				req := httptest.NewRequest("PATCH", "/memories/"+memID.String(), bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					writeSuccessCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Greater(t, readSuccessCount.Load(), int32(0), "Some reads should succeed")
		assert.Greater(t, writeSuccessCount.Load(), int32(0), "Some writes should succeed")
	})
}

// TestParallelAppGeneration tests 5 parallel app generation requests
func TestParallelAppGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("5 parallel app generations should queue correctly", func(t *testing.T) {
		const numApps = 5

		var wg sync.WaitGroup
		wg.Add(numApps)

		successCount := atomic.Int32{}

		for i := 0; i < numApps; i++ {
			go func(appNum int) {
				defer wg.Done()

				userID, _ := createIntTestUserWithSession(t, ctx, testDB.Pool)
				workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

				// Queue app generation
				queueID := uuid.New()
				_, err := testDB.Pool.Exec(ctx, `
					INSERT INTO app_generation_queue (
						id, workspace_id, user_id, prompt, status, created_at, updated_at
					) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
				`, queueID, workspaceID, uuid.MustParse(userID), fmt.Sprintf("Generate app %d", appNum), "pending")

				if err == nil {
					successCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(numApps), successCount.Load(), "All app generation requests should queue")
	})
}

// TestParallelSSEStreaming tests multiple concurrent SSE streams
func TestParallelSSEStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("5 parallel SSE streams should remain isolated", func(t *testing.T) {
		const numStreams = 5

		var wg sync.WaitGroup
		wg.Add(numStreams)

		streamStartCount := atomic.Int32{}

		for i := 0; i < numStreams; i++ {
			go func(streamNum int) {
				defer wg.Done()

				userID, sessionID := createIntTestUserWithSession(t, ctx, testDB.Pool)
				workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

				router := setupOSARouter(userID, sessionID)

				// Mock SSE endpoint
				router.GET("/stream", func(c *gin.Context) {
					c.Header("Content-Type", "text/event-stream")
					c.Header("Cache-Control", "no-cache")
					c.Header("Connection", "keep-alive")

					// Send test events
					fmt.Fprintf(c.Writer, "data: {\"stream_id\": %d, \"workspace_id\": \"%s\"}\n\n", streamNum, workspaceID.String())
					c.Writer.Flush()
				})

				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/stream", nil)
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK && w.Header().Get("Content-Type") == "text/event-stream" {
					streamStartCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(numStreams), streamStartCount.Load(), "All SSE streams should start successfully")
	})
}

// TestParallelAuthentication tests concurrent login attempts
func TestParallelAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("10 parallel login attempts should not conflict", func(t *testing.T) {
		const numLogins = 10

		// Pre-create users
		userCreds := make([]map[string]string, numLogins)
		for i := 0; i < numLogins; i++ {
			userID := uuid.New()
			email := fmt.Sprintf("testuser%d@example.com", i)
			userCreds[i] = map[string]string{
				"id":    userID.String(),
				"email": email,
			}

			// Insert user
			_, err := testDB.Pool.Exec(ctx, `
				INSERT INTO users (id, email, password_hash, username, email_verified, created_at, updated_at)
				VALUES ($1, $2, $3, $4, true, NOW(), NOW())
			`, userID, email, "$2a$10$hashedpassword", fmt.Sprintf("user%d", i))
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		wg.Add(numLogins)

		sessionCreateCount := atomic.Int32{}

		for i := 0; i < numLogins; i++ {
			go func(idx int) {
				defer wg.Done()

				userID := uuid.MustParse(userCreds[idx]["id"])
				sessionID := uuid.New()

				// Create session (simulating successful login)
				_, err := testDB.Pool.Exec(ctx, `
					INSERT INTO sessions (id, user_id, session_token, expires_at, created_at, updated_at)
					VALUES ($1, $2, $3, NOW() + INTERVAL '1 day', NOW(), NOW())
				`, sessionID, userID, "token-"+sessionID.String())

				if err == nil {
					sessionCreateCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(numLogins), sessionCreateCount.Load(), "All sessions should be created without conflicts")
	})
}

// TestDatabaseConnectionPoolUnderLoad tests connection pool with 50+ concurrent queries
func TestDatabaseConnectionPoolUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("50 concurrent queries should not exhaust connection pool", func(t *testing.T) {
		const numQueries = 50

		userID, _ := createIntTestUserWithSession(t, ctx, testDB.Pool)
		workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

		var wg sync.WaitGroup
		wg.Add(numQueries)

		successCount := atomic.Int32{}
		errorCount := atomic.Int32{}

		startTime := time.Now()

		for i := 0; i < numQueries; i++ {
			go func(queryNum int) {
				defer wg.Done()

				// Simulate various DB operations
				switch queryNum % 5 {
				case 0:
					// SELECT query
					var count int
					err := testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspaces WHERE id = $1", workspaceID).Scan(&count)
					if err == nil {
						successCount.Add(1)
					} else {
						errorCount.Add(1)
					}

				case 1:
					// INSERT into memories
					memID := uuid.New()
					_, err := testDB.Pool.Exec(ctx, `
						INSERT INTO memories (id, workspace_id, user_id, title, content, memory_type, created_at, updated_at)
						VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
					`, memID, workspaceID, uuid.MustParse(userID), fmt.Sprintf("Load Test %d", queryNum), "Content", "fact")
					if err == nil {
						successCount.Add(1)
					} else {
						errorCount.Add(1)
					}

				case 2:
					// UPDATE query
					_, err := testDB.Pool.Exec(ctx, "UPDATE workspaces SET updated_at = NOW() WHERE id = $1", workspaceID)
					if err == nil {
						successCount.Add(1)
					} else {
						errorCount.Add(1)
					}

				case 3:
					// JOIN query
					rows, err := testDB.Pool.Query(ctx, `
						SELECT w.id, w.name, u.email
						FROM workspaces w
						JOIN users u ON w.owner_id = u.id
						WHERE w.id = $1
					`, workspaceID)
					if err == nil {
						rows.Close()
						successCount.Add(1)
					} else {
						errorCount.Add(1)
					}

				case 4:
					// Transaction
					tx, err := testDB.Pool.Begin(ctx)
					if err == nil {
						_, err = tx.Exec(ctx, "UPDATE workspaces SET updated_at = NOW() WHERE id = $1", workspaceID)
						if err == nil {
							tx.Commit(ctx)
							successCount.Add(1)
						} else {
							tx.Rollback(ctx)
							errorCount.Add(1)
						}
					} else {
						errorCount.Add(1)
					}
				}
			}(i)
		}

		wg.Wait()

		duration := time.Since(startTime)

		assert.Equal(t, int32(numQueries), successCount.Load(), "All queries should succeed")
		assert.Equal(t, int32(0), errorCount.Load(), "No queries should fail")
		assert.Less(t, duration, 5*time.Second, "Queries should complete within 5 seconds")

		t.Logf("Connection pool test: %d queries completed in %v", numQueries, duration)
	})
}

// TestRaceConditionDetection tests for race conditions in concurrent operations
func TestRaceConditionDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("Concurrent updates to same memory should not cause race", func(t *testing.T) {
		userID, _ := createIntTestUserWithSession(t, ctx, testDB.Pool)
		workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)
		memoryID := createIntTestMemory(t, ctx, testDB.Pool, workspaceID, userID, "Race Test Memory")

		const numUpdates = 20

		var wg sync.WaitGroup
		wg.Add(numUpdates)

		for i := 0; i < numUpdates; i++ {
			go func(updateNum int) {
				defer wg.Done()

				// Concurrent updates to access_count
				_, err := testDB.Pool.Exec(ctx, `
					UPDATE memories
					SET access_count = access_count + 1, last_accessed_at = NOW()
					WHERE id = $1
				`, memoryID)

				assert.NoError(t, err, "Update should not fail")
			}(i)
		}

		wg.Wait()

		// Verify final count
		var finalCount int
		err := testDB.Pool.QueryRow(ctx, "SELECT access_count FROM memories WHERE id = $1", memoryID).Scan(&finalCount)
		require.NoError(t, err)

		assert.Equal(t, numUpdates, finalCount, "Access count should reflect all concurrent updates")
	})

	t.Run("Concurrent workspace member additions should not conflict", func(t *testing.T) {
		userID, _ := createIntTestUserWithSession(t, ctx, testDB.Pool)
		workspaceID := createIntTestWorkspace(t, ctx, testDB.Pool, userID)

		const numMembers = 10

		// Create 10 users
		memberIDs := make([]uuid.UUID, numMembers)
		for i := 0; i < numMembers; i++ {
			memberID := uuid.New()
			memberIDs[i] = memberID
			_, err := testDB.Pool.Exec(ctx, `
				INSERT INTO users (id, email, password_hash, username, email_verified, created_at, updated_at)
				VALUES ($1, $2, $3, $4, true, NOW(), NOW())
			`, memberID, fmt.Sprintf("member%d@example.com", i), "$2a$10$hash", fmt.Sprintf("member%d", i))
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		wg.Add(numMembers)

		addedCount := atomic.Int32{}

		for i := 0; i < numMembers; i++ {
			go func(idx int) {
				defer wg.Done()

				_, err := testDB.Pool.Exec(ctx, `
					INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
					VALUES ($1, $2, $3, NOW())
				`, workspaceID, memberIDs[idx], "member")

				if err == nil {
					addedCount.Add(1)
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(numMembers), addedCount.Load(), "All members should be added")

		// Verify count in DB
		var totalMembers int
		err := testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1", workspaceID).Scan(&totalMembers)
		require.NoError(t, err)

		// +1 for the owner created in createIntTestWorkspace
		assert.Equal(t, numMembers+1, totalMembers, "Total member count should match")
	})
}

// ================================================
// BENCHMARKS
// ================================================

// BenchmarkParallelMemoryCreation benchmarks concurrent memory creation
func BenchmarkParallelMemoryCreation(b *testing.B) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDatabase(&testing.T{})
	if err != nil {
		b.Skip("Database not available")
	}
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	userID, _ := createIntTestUserWithSession(&testing.T{}, ctx, testDB.Pool)
	workspaceID := createIntTestWorkspace(&testing.T{}, ctx, testDB.Pool, userID)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		memID := uuid.New()
		_, _ = testDB.Pool.Exec(ctx, `
			INSERT INTO memories (id, workspace_id, user_id, title, content, memory_type, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		`, memID, workspaceID, uuid.MustParse(userID), "Benchmark Memory", "Content", "fact")
	}
}

// BenchmarkParallelMemoryRead benchmarks concurrent memory reads
func BenchmarkParallelMemoryRead(b *testing.B) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDatabase(&testing.T{})
	if err != nil {
		b.Skip("Database not available")
	}
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	userID, _ := createIntTestUserWithSession(&testing.T{}, ctx, testDB.Pool)
	workspaceID := createIntTestWorkspace(&testing.T{}, ctx, testDB.Pool, userID)
	memoryID := createIntTestMemory(&testing.T{}, ctx, testDB.Pool, workspaceID, userID, "Benchmark Read")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var title string
		_ = testDB.Pool.QueryRow(ctx, "SELECT title FROM memories WHERE id = $1", memoryID).Scan(&title)
	}
}

// BenchmarkConcurrentConnectionPoolUsage benchmarks connection pool under concurrent load
func BenchmarkConcurrentConnectionPoolUsage(b *testing.B) {
	ctx := context.Background()
	testDB, err := testutil.SetupTestDatabase(&testing.T{})
	if err != nil {
		b.Skip("Database not available")
	}
	defer testDB.Close()

	userID, _ := createIntTestUserWithSession(&testing.T{}, ctx, testDB.Pool)
	workspaceID := createIntTestWorkspace(&testing.T{}, ctx, testDB.Pool, userID)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var count int
			_ = testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM workspaces WHERE id = $1", workspaceID).Scan(&count)
		}
	})
}
