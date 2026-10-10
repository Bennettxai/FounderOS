package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSessionCache_Set verifies session creation adds to user index
func TestSessionCache_Set(t *testing.T) {
	// Skip if Redis not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	// Create session cache
	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "test-secret-key-for-testing-only",
	}
	cache, err := NewSessionCache(client, cfg)
	require.NoError(t, err)

	// Clean up test data
	defer func() {
		// Clean up all test keys
		iter := client.Scan(ctx, 0, "test_session:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
		iter = client.Scan(ctx, 0, "user_sessions:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
	}()

	// Test user
	user := &BetterAuthUser{
		ID:            "user123",
		Name:          "Test User",
		Email:         "test@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set session
	token := "test-token-123"
	err = cache.Set(ctx, token, user)
	require.NoError(t, err, "Set should succeed")

	// Verify session data exists
	sessionKey := cache.sessionKey(token)
	exists := client.Exists(ctx, sessionKey).Val()
	assert.Equal(t, int64(1), exists, "Session key should exist")

	// Verify session in user's set
	userSessionsKey := cache.userSessionsKey(user.ID)
	isMember := client.SIsMember(ctx, userSessionsKey, sessionKey).Val()
	assert.True(t, isMember, "Session key should be in user's session set")

	// Verify user set has TTL
	ttl := client.TTL(ctx, userSessionsKey).Val()
	assert.Greater(t, ttl.Seconds(), 0.0, "User session set should have TTL")
}

// TestSessionCache_Invalidate verifies single session invalidation
func TestSessionCache_Invalidate(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "test-secret-key-for-testing-only",
	}
	cache, err := NewSessionCache(client, cfg)
	require.NoError(t, err)

	defer func() {
		iter := client.Scan(ctx, 0, "test_session:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
		iter = client.Scan(ctx, 0, "user_sessions:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
	}()

	user := &BetterAuthUser{
		ID:            "user456",
		Name:          "Test User 2",
		Email:         "test2@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create session
	token := "test-token-456"
	err = cache.Set(ctx, token, user)
	require.NoError(t, err)

	// Verify session exists
	cached, err := cache.Get(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, user.ID, cached.ID)

	// Invalidate session
	err = cache.Invalidate(ctx, token)
	require.NoError(t, err, "Invalidate should succeed")

	// Verify session deleted
	sessionKey := cache.sessionKey(token)
	exists := client.Exists(ctx, sessionKey).Val()
	assert.Equal(t, int64(0), exists, "Session key should be deleted")

	// Verify session removed from user's set
	userSessionsKey := cache.userSessionsKey(user.ID)
	isMember := client.SIsMember(ctx, userSessionsKey, sessionKey).Val()
	assert.False(t, isMember, "Session key should be removed from user's set")
}

// TestSessionCache_InvalidateUserSessions verifies bulk session invalidation
func TestSessionCache_InvalidateUserSessions(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "test-secret-key-for-testing-only",
	}
	cache, err := NewSessionCache(client, cfg)
	require.NoError(t, err)

	defer func() {
		iter := client.Scan(ctx, 0, "test_session:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
		iter = client.Scan(ctx, 0, "user_sessions:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
	}()

	user := &BetterAuthUser{
		ID:            "user789",
		Name:          "Test User 3",
		Email:         "test3@example.com",
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create 3 sessions for the same user
	tokens := []string{"token-1", "token-2", "token-3"}
	for _, token := range tokens {
		err = cache.Set(ctx, token, user)
		require.NoError(t, err, "Should create session")
	}

	// Verify all 3 sessions exist
	userSessionsKey := cache.userSessionsKey(user.ID)
	count := client.SCard(ctx, userSessionsKey).Val()
	assert.Equal(t, int64(3), count, "User should have 3 sessions")

	// Invalidate all user sessions
	err = cache.InvalidateUserSessions(ctx, user.ID)
	require.NoError(t, err, "InvalidateUserSessions should succeed")

	// Verify all sessions deleted
	for _, token := range tokens {
		sessionKey := cache.sessionKey(token)
		exists := client.Exists(ctx, sessionKey).Val()
		assert.Equal(t, int64(0), exists, "Session %s should be deleted", token)
	}

	// Verify user's session set deleted
	exists := client.Exists(ctx, userSessionsKey).Val()
	assert.Equal(t, int64(0), exists, "User's session set should be deleted")
}

// TestSessionCache_InvalidateUserSessions_NoSessions verifies graceful handling of no sessions
func TestSessionCache_InvalidateUserSessions_NoSessions(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "test-secret-key-for-testing-only",
	}
	cache, err := NewSessionCache(client, cfg)
	require.NoError(t, err)

	// Try to invalidate sessions for user with no sessions
	err = cache.InvalidateUserSessions(ctx, "nonexistent-user")
	assert.NoError(t, err, "Should handle no sessions gracefully")
}

// TestSessionCache_MultipleUsers verifies session isolation between users
func TestSessionCache_MultipleUsers(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}

	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "test-secret-key-for-testing-only",
	}
	cache, err := NewSessionCache(client, cfg)
	require.NoError(t, err)

	defer func() {
		iter := client.Scan(ctx, 0, "test_session:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
		iter = client.Scan(ctx, 0, "user_sessions:*", 0).Iterator()
		for iter.Next(ctx) {
			client.Del(ctx, iter.Val())
		}
	}()

	// Create sessions for user1
	user1 := &BetterAuthUser{
		ID:        "user-aaa",
		Email:     "user1@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = cache.Set(ctx, "user1-token1", user1)
	_ = cache.Set(ctx, "user1-token2", user1)

	// Create sessions for user2
	user2 := &BetterAuthUser{
		ID:        "user-bbb",
		Email:     "user2@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = cache.Set(ctx, "user2-token1", user2)
	_ = cache.Set(ctx, "user2-token2", user2)

	// Verify user1 has 2 sessions
	user1SessionsKey := cache.userSessionsKey(user1.ID)
	count1 := client.SCard(ctx, user1SessionsKey).Val()
	assert.Equal(t, int64(2), count1, "User1 should have 2 sessions")

	// Verify user2 has 2 sessions
	user2SessionsKey := cache.userSessionsKey(user2.ID)
	count2 := client.SCard(ctx, user2SessionsKey).Val()
	assert.Equal(t, int64(2), count2, "User2 should have 2 sessions")

	// Invalidate user1's sessions
	err = cache.InvalidateUserSessions(ctx, user1.ID)
	require.NoError(t, err)

	// Verify user1's sessions deleted
	exists := client.Exists(ctx, user1SessionsKey).Val()
	assert.Equal(t, int64(0), exists, "User1's sessions should be deleted")

	// Verify user2's sessions still exist
	count2After := client.SCard(ctx, user2SessionsKey).Val()
	assert.Equal(t, int64(2), count2After, "User2's sessions should still exist")

	// Verify user2 can still access their sessions
	cached, err := cache.Get(ctx, "user2-token1")
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, user2.ID, cached.ID)
}

// TestSessionCache_HMACConsistency verifies HMAC key generation is consistent
func TestSessionCache_HMACConsistency(t *testing.T) {
	cfg := &SessionCacheConfig{
		KeyPrefix:  "test_session:",
		TTL:        1 * time.Minute,
		HMACSecret: "consistent-secret-key",
	}
	cache, err := NewSessionCache(nil, cfg)
	require.NoError(t, err)

	token := "test-token"
	userID := "user123"

	// Generate keys multiple times
	key1 := cache.sessionKey(token)
	key2 := cache.sessionKey(token)
	userKey1 := cache.userSessionsKey(userID)
	userKey2 := cache.userSessionsKey(userID)

	// Verify consistency
	assert.Equal(t, key1, key2, "Session keys should be consistent")
	assert.Equal(t, userKey1, userKey2, "User session keys should be consistent")

	// Verify different tokens produce different keys
	key3 := cache.sessionKey("different-token")
	assert.NotEqual(t, key1, key3, "Different tokens should produce different keys")
}

// TestSessionCache_HMACSecretRequired verifies HMAC secret warning in development
func TestSessionCache_HMACSecretRequired(t *testing.T) {
	// Create cache without HMAC secret (development mode)
	cfg := &SessionCacheConfig{
		KeyPrefix: "test_session:",
		TTL:       1 * time.Minute,
		// HMACSecret is empty - should auto-generate
	}
	cache, err := NewSessionCache(nil, cfg)
	require.NoError(t, err)

	// Verify HMAC secret was auto-generated
	assert.NotNil(t, cache.hmacSecret, "HMAC secret should be auto-generated")
	assert.Greater(t, len(cache.hmacSecret), 0, "HMAC secret should not be empty")
}
