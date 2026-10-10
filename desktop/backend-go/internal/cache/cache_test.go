package cache

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a mini-redis instance for testing
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

// setupTestLogger creates a test logger
func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors during tests
	}))
}

func TestConversationHistoryCaching(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	conversationID := "conv-123"
	messages := []*ConversationMessage{
		{
			ID:        "msg-1",
			Role:      "user",
			Content:   "Hello",
			CreatedAt: time.Now(),
		},
		{
			ID:        "msg-2",
			Role:      "assistant",
			Content:   "Hi there!",
			CreatedAt: time.Now(),
		},
	}

	// Test cache miss
	_, err := cache.GetConversationHistory(ctx, conversationID)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Equal(t, uint64(1), cache.stats.Misses.Load())

	// Test cache set
	err = cache.SetConversationHistory(ctx, conversationID, messages)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), cache.stats.Sets.Load())

	// Test cache hit
	cached, err := cache.GetConversationHistory(ctx, conversationID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(cached))
	assert.Equal(t, "msg-1", cached[0].ID)
	assert.Equal(t, "user", cached[0].Role)
	assert.Equal(t, uint64(1), cache.stats.Hits.Load())

	// Test cache invalidation
	err = cache.InvalidateConversationHistory(ctx, conversationID)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), cache.stats.Deletes.Load())

	// Verify invalidation worked
	_, err = cache.GetConversationHistory(ctx, conversationID)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Equal(t, uint64(2), cache.stats.Misses.Load())
}

func TestEmbeddingCaching(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	text := "This is a test document for embedding"
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	// Test cache miss
	_, err := cache.GetEmbedding(ctx, text)
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Test cache set
	err = cache.SetEmbedding(ctx, text, embedding)
	require.NoError(t, err)

	// Test cache hit
	cached, err := cache.GetEmbedding(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 5, len(cached))
	assert.Equal(t, float32(0.1), cached[0])
	assert.Equal(t, float32(0.5), cached[4])

	// Test that same text produces same hash (deterministic)
	cached2, err := cache.GetEmbedding(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, cached, cached2)

	// Test that different text produces cache miss
	_, err = cache.GetEmbedding(ctx, "Different text")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestAgentStatusCaching(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	agentID := "agent-456"
	status := &AgentStatus{
		AgentID:     agentID,
		Status:      "active",
		CurrentTask: "Processing document",
		LastUpdate:  time.Now(),
		Metadata: map[string]interface{}{
			"cpu":    50.5,
			"memory": 1024,
		},
	}

	// Test cache miss
	_, err := cache.GetAgentStatus(ctx, agentID)
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Test cache set
	err = cache.SetAgentStatus(ctx, status)
	require.NoError(t, err)

	// Test cache hit
	cached, err := cache.GetAgentStatus(ctx, agentID)
	require.NoError(t, err)
	assert.Equal(t, "agent-456", cached.AgentID)
	assert.Equal(t, "active", cached.Status)
	assert.Equal(t, "Processing document", cached.CurrentTask)
	assert.Equal(t, float64(50.5), cached.Metadata["cpu"])

	// Test cache invalidation
	err = cache.InvalidateAgentStatus(ctx, agentID)
	require.NoError(t, err)

	// Verify invalidation
	_, err = cache.GetAgentStatus(ctx, agentID)
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestArtifactListCaching(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	userID := "user-789"
	page := 1
	filters := map[string]interface{}{
		"type":   "CODE",
		"status": "active",
	}

	key := ArtifactListKey(userID, page, filters)
	artifacts := []map[string]interface{}{
		{"id": "art-1", "title": "Document 1"},
		{"id": "art-2", "title": "Document 2"},
	}

	// Test cache miss
	_, err := cache.GetArtifactList(ctx, key)
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Test cache set
	err = cache.SetArtifactList(ctx, key, artifacts)
	require.NoError(t, err)

	// Test cache hit
	cached, err := cache.GetArtifactList(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, cached)

	// Test invalidation by user
	err = cache.InvalidateArtifactListsByUser(ctx, userID)
	require.NoError(t, err)

	// Verify invalidation
	_, err = cache.GetArtifactList(ctx, key)
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestGenericCaching(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	// Test string value
	err := cache.Set(ctx, "test:string", "hello world", 1*time.Minute)
	require.NoError(t, err)

	val, err := cache.Get(ctx, "test:string")
	require.NoError(t, err)
	assert.Equal(t, "hello world", val)

	// Test struct value
	type TestStruct struct {
		Name  string
		Count int
	}
	testData := TestStruct{Name: "test", Count: 42}

	err = cache.Set(ctx, "test:struct", testData, 1*time.Minute)
	require.NoError(t, err)

	// Test delete
	err = cache.Delete(ctx, "test:string")
	require.NoError(t, err)

	_, err = cache.Get(ctx, "test:string")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestCacheStats(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	// Perform various operations
	cache.Get(ctx, "nonexistent")                              // miss
	cache.Set(ctx, "key1", "value1", 1*time.Minute)            // set
	cache.Get(ctx, "key1")                                     // hit
	cache.Get(ctx, "key2")                                     // miss
	cache.Set(ctx, "key2", "value2", 1*time.Minute)            // set
	cache.Get(ctx, "key2")                                     // hit
	cache.Delete(ctx, "key1")                                  // delete

	stats := cache.GetStats()
	assert.Equal(t, uint64(2), stats.Hits.Load())
	assert.Equal(t, uint64(2), stats.Misses.Load())
	assert.Equal(t, uint64(2), stats.Sets.Load())
	assert.Equal(t, uint64(1), stats.Deletes.Load())
	assert.Equal(t, uint64(4), stats.TotalRequests.Load())

	hitRate := cache.GetHitRate()
	assert.Equal(t, 50.0, hitRate) // 2 hits out of 4 total = 50%

	// Test stats reset
	cache.ResetStats()
	stats = cache.GetStats()
	assert.Equal(t, uint64(0), stats.Hits.Load())
	assert.Equal(t, uint64(0), stats.Misses.Load())
}

func TestCachePing(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	err := cache.Ping(ctx)
	assert.NoError(t, err)
}

func TestCacheFlush(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	// Set some data
	cache.Set(ctx, "key1", "value1", 1*time.Minute)
	cache.Set(ctx, "key2", "value2", 1*time.Minute)

	// Verify data exists
	_, err := cache.Get(ctx, "key1")
	assert.NoError(t, err)

	// Enable FlushDB for testing
	os.Setenv("REDIS_ALLOW_FLUSH", "true")
	defer os.Unsetenv("REDIS_ALLOW_FLUSH")

	// Flush cache
	err = cache.FlushAll(ctx)
	require.NoError(t, err)

	// Verify data is gone
	_, err = cache.Get(ctx, "key1")
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Verify stats were reset
	stats := cache.GetStats()
	assert.Equal(t, uint64(0), stats.Hits.Load())
}

func TestHashDeterminism(t *testing.T) {
	// Test that hashing is deterministic
	text1 := "test document"
	text2 := "test document"
	text3 := "different document"

	hash1 := hashText(text1)
	hash2 := hashText(text2)
	hash3 := hashText(text3)

	assert.Equal(t, hash1, hash2, "Same text should produce same hash")
	assert.NotEqual(t, hash1, hash3, "Different text should produce different hash")

	// Test filter hashing
	filters1 := map[string]interface{}{"type": "CODE", "status": "active"}
	filters2 := map[string]interface{}{"type": "CODE", "status": "active"}
	filters3 := map[string]interface{}{"type": "DOC", "status": "active"}

	filterHash1 := hashFilters(filters1)
	filterHash2 := hashFilters(filters2)
	filterHash3 := hashFilters(filters3)

	assert.Equal(t, filterHash1, filterHash2, "Same filters should produce same hash")
	assert.NotEqual(t, filterHash1, filterHash3, "Different filters should produce different hash")
}

func BenchmarkConversationHistoryCaching(b *testing.B) {
	mr, client := setupTestRedis(&testing.T{})
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	messages := make([]*ConversationMessage, 50)
	for i := 0; i < 50; i++ {
		messages[i] = &ConversationMessage{
			ID:        string(rune(i)),
			Role:      "user",
			Content:   "Test message content",
			CreatedAt: time.Now(),
		}
	}

	conversationID := "bench-conv-123"
	cache.SetConversationHistory(ctx, conversationID, messages)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetConversationHistory(ctx, conversationID)
	}
}

func BenchmarkEmbeddingCaching(b *testing.B) {
	mr, client := setupTestRedis(&testing.T{})
	defer mr.Close()
	defer client.Close()

	cache := NewCacheService(client, setupTestLogger())
	ctx := context.Background()

	text := "This is a benchmark test document for embedding caching"
	embedding := make([]float32, 1536) // OpenAI embedding size
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	cache.SetEmbedding(ctx, text, embedding)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetEmbedding(ctx, text)
	}
}

// TestQueryCacheDelete tests the Delete method
func TestQueryCacheDelete(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	logger := setupTestLogger()
	cache := NewQueryCache(client, logger)
	ctx := context.Background()

	// Set a cache entry
	key := "test:delete:key"
	value := map[string]string{"test": "value"}
	err := cache.Set(ctx, key, value, 1*time.Hour)
	require.NoError(t, err)

	// Verify it exists
	var retrieved map[string]string
	hit, err := cache.Get(ctx, key, &retrieved)
	require.NoError(t, err)
	assert.True(t, hit, "Cache entry should exist before delete")

	// Delete the entry
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's deleted
	hit, err = cache.Get(ctx, key, &retrieved)
	require.NoError(t, err)
	assert.False(t, hit, "Cache entry should not exist after delete")
}

// TestQueryCacheDeleteByPattern tests the DeleteByPattern method
func TestQueryCacheDeleteByPattern(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	logger := setupTestLogger()
	cache := NewQueryCache(client, logger)
	ctx := context.Background()

	// Set multiple cache entries with same pattern
	pattern := "test:pattern:*"
	keys := []string{
		"test:pattern:key1",
		"test:pattern:key2",
		"test:pattern:key3",
	}
	value := map[string]string{"test": "value"}

	for _, key := range keys {
		err := cache.Set(ctx, key, value, 1*time.Hour)
		require.NoError(t, err)
	}

	// Set an entry that doesn't match the pattern
	otherKey := "other:key"
	err := cache.Set(ctx, otherKey, value, 1*time.Hour)
	require.NoError(t, err)

	// Verify all entries exist
	for _, key := range keys {
		var retrieved map[string]string
		hit, err := cache.Get(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.True(t, hit, "Cache entry should exist before delete: %s", key)
	}

	// Delete by pattern
	deletedCount, err := cache.DeleteByPattern(ctx, pattern)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deletedCount, "Should have deleted 3 entries")

	// Verify pattern entries are deleted
	for _, key := range keys {
		var retrieved map[string]string
		hit, err := cache.Get(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.False(t, hit, "Cache entry should be deleted after pattern delete: %s", key)
	}

	// Verify other entry still exists
	var retrieved map[string]string
	hit, err := cache.Get(ctx, otherKey, &retrieved)
	require.NoError(t, err)
	assert.True(t, hit, "Entry with different pattern should not be deleted")
}

// TestQueryCacheMDel tests the MDel method
func TestQueryCacheMDel(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	logger := setupTestLogger()
	cache := NewQueryCache(client, logger)
	ctx := context.Background()

	// Set multiple cache entries
	keys := []string{
		"test:mdel:key1",
		"test:mdel:key2",
		"test:mdel:key3",
	}
	value := map[string]string{"test": "value"}

	for _, key := range keys {
		err := cache.Set(ctx, key, value, 1*time.Hour)
		require.NoError(t, err)
	}

	// Verify entries exist
	for _, key := range keys {
		var retrieved map[string]string
		hit, err := cache.Get(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.True(t, hit, "Cache entry should exist before mdel: %s", key)
	}

	// Delete multiple entries at once
	deletedCount, err := cache.MDel(ctx, keys)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deletedCount, "Should have deleted 3 entries")

	// Verify entries are deleted
	for _, key := range keys {
		var retrieved map[string]string
		hit, err := cache.Get(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.False(t, hit, "Cache entry should be deleted after mdel: %s", key)
	}
}

// TestQueryCacheMDelEmpty tests MDel with empty list
func TestQueryCacheMDelEmpty(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	logger := setupTestLogger()
	cache := NewQueryCache(client, logger)
	ctx := context.Background()

	// Call MDel with empty list
	deletedCount, err := cache.MDel(ctx, []string{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), deletedCount, "MDel with empty list should return 0")
}
