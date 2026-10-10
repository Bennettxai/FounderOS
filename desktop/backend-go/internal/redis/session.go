// Package redis provides session storage backed by Redis for horizontal scaling
package redis

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Session represents a user session stored in Redis
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Email     string                 `json:"email,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Image     string                 `json:"image,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionStore provides session management operations with secure key hashing
type SessionStore struct {
	client     *redis.Client
	keyPrefix  string
	ttl        time.Duration
	hmacSecret []byte // HMAC secret for secure key derivation (prevents enumeration attacks)
	workerPool *CacheWorkerPool // Optional worker pool for bounded concurrency
}

// SessionStoreConfig configures the session store
type SessionStoreConfig struct {
	KeyPrefix        string        // Prefix for session keys (default: "session:")
	TTL              time.Duration // Session TTL (default: 24 hours)
	HMACSecret       string        // HMAC secret for secure key derivation (CRITICAL in production)
	UseWorkerPool    bool          // Enable worker pool for bounded concurrency (default: false)
	WorkerPoolSize   int           // Worker pool size (default: 10, only used if UseWorkerPool=true)
	WorkerQueueSize  int           // Worker queue size (default: 100, only used if UseWorkerPool=true)
}

// DefaultSessionStoreConfig returns default configuration
func DefaultSessionStoreConfig() *SessionStoreConfig {
	return &SessionStoreConfig{
		KeyPrefix:       "session:",
		TTL:             24 * time.Hour,
		UseWorkerPool:   false, // Disabled by default for backward compatibility
		WorkerPoolSize:  10,
		WorkerQueueSize: 100,
	}
}

// NewSessionStore creates a new Redis-backed session store with secure key hashing
// If cfg.HMACSecret is empty, generates a random secret (development mode only)
// CRITICAL: In production, ALWAYS provide a strong HMACSecret via environment variable
func NewSessionStore(cfg *SessionStoreConfig) (*SessionStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized - call Connect first")
	}

	if cfg == nil {
		cfg = DefaultSessionStoreConfig()
	}

	// Convert HMAC secret to bytes
	var hmacSecret []byte
	if cfg.HMACSecret != "" {
		hmacSecret = []byte(cfg.HMACSecret)
	} else {
		// Generate random secret for development (NOT for production)
		// In production, this will cause issues across server restarts/instances
		hmacSecret = make([]byte, 32)
		if _, err := rand.Read(hmacSecret); err != nil {
			return nil, fmt.Errorf("CRITICAL: Failed to generate HMAC secret: %w", err)
		}
		slog.Warn("SessionStore using auto-generated HMAC secret - Set REDIS_KEY_HMAC_SECRET environment variable for production")
	}

	if len(hmacSecret) < 32 {
		slog.Warn("HMAC secret shorter than recommended 32 bytes", "current_length", len(hmacSecret))
	}

	store := &SessionStore{
		client:     client,
		keyPrefix:  cfg.KeyPrefix,
		ttl:        cfg.TTL,
		hmacSecret: hmacSecret,
	}

	// Initialize worker pool if enabled
	if cfg.UseWorkerPool {
		pool := NewCacheWorkerPool(cfg.WorkerPoolSize, cfg.WorkerQueueSize)
		if pool != nil {
			pool.Start()
			store.workerPool = pool
			slog.Info("SessionStore worker pool enabled", "workers", cfg.WorkerPoolSize, "queue_size", cfg.WorkerQueueSize)
		} else {
			slog.Warn("Worker pool requested but Redis client not available - falling back to direct access")
		}
	}

	return store, nil
}

// sessionKey generates a secure Redis key for a session using HMAC-SHA256
// This prevents enumeration attacks by hashing the session ID before using it as a key
//
// Security rationale:
// - Raw session IDs in Redis keys allow attackers to enumerate valid sessions
// - HMAC-SHA256 produces a one-way hash that can't be reversed
// - Using a secret key ensures only the application can generate valid hashes
// - 64-character hex output provides sufficient namespace to prevent collisions
func (s *SessionStore) sessionKey(sessionID string) string {
	hash := s.hashKey(sessionID)
	return s.keyPrefix + hash
}

// userSessionsKey generates a secure Redis key for user's session set using HMAC-SHA256
// This prevents user ID enumeration and links session hashes to user accounts
func (s *SessionStore) userSessionsKey(userID string) string {
	hash := s.hashKey("user:" + userID)
	return "user_sessions:" + hash
}

// hashKey performs HMAC-SHA256 hashing of input using the configured secret
// Returns hex-encoded hash (64 characters)
func (s *SessionStore) hashKey(input string) string {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// Create stores a new session
func (s *SessionStore) Create(ctx context.Context, session *Session) error {
	if session.ID == "" {
		return fmt.Errorf("session ID is required")
	}
	if session.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	// Set timestamps
	now := time.Now()
	session.CreatedAt = now
	session.ExpiresAt = now.Add(s.ttl)

	// Serialize session
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := s.client.Pipeline()

	// Store session with TTL
	pipe.Set(ctx, s.sessionKey(session.ID), data, s.ttl)

	// Add to user's session set (for listing/invalidating all user sessions)
	pipe.SAdd(ctx, s.userSessionsKey(session.UserID), session.ID)
	pipe.Expire(ctx, s.userSessionsKey(session.UserID), s.ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	data, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil // Session not found (expired or deleted)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		_ = s.Delete(ctx, sessionID)
		return nil, nil
	}

	return &session, nil
}

// Refresh extends the session TTL
func (s *SessionStore) Refresh(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Update expiration
	session.ExpiresAt = time.Now().Add(s.ttl)

	// Serialize and store
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := s.client.Pipeline()
	pipe.Set(ctx, s.sessionKey(sessionID), data, s.ttl)
	pipe.Expire(ctx, s.userSessionsKey(session.UserID), s.ttl)

	_, err = pipe.Exec(ctx)
	return err
}

// Delete removes a session
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	// Get session first to remove from user's set
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, s.sessionKey(sessionID))

	if session != nil {
		pipe.SRem(ctx, s.userSessionsKey(session.UserID), sessionID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteUserSessions removes all sessions for a user (logout from all devices)
func (s *SessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	// Get all session IDs for user
	sessionIDs, err := s.client.SMembers(ctx, s.userSessionsKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	// Build keys to delete
	keys := make([]string, 0, len(sessionIDs)+1)
	for _, sid := range sessionIDs {
		keys = append(keys, s.sessionKey(sid))
	}
	keys = append(keys, s.userSessionsKey(userID))

	// Delete all keys
	return s.client.Del(ctx, keys...).Err()
}

// ListUserSessions returns all active sessions for a user
func (s *SessionStore) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	sessionIDs, err := s.client.SMembers(ctx, s.userSessionsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user session IDs: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*Session{}, nil
	}

	// Build keys
	keys := make([]string, len(sessionIDs))
	for i, sid := range sessionIDs {
		keys[i] = s.sessionKey(sid)
	}

	// Get all sessions
	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	sessions := make([]*Session, 0, len(results))
	now := time.Now()

	for _, result := range results {
		if result == nil {
			continue
		}

		var session Session
		if err := json.Unmarshal([]byte(result.(string)), &session); err != nil {
			continue
		}

		// Skip expired sessions
		if now.After(session.ExpiresAt) {
			continue
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// UpdateMetadata updates session metadata without changing other fields
func (s *SessionStore) UpdateMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Merge metadata
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}
	for k, v := range metadata {
		session.Metadata[k] = v
	}

	// Calculate remaining TTL
	remainingTTL := time.Until(session.ExpiresAt)
	if remainingTTL <= 0 {
		return fmt.Errorf("session expired")
	}

	// Serialize and store
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return s.client.Set(ctx, s.sessionKey(sessionID), data, remainingTTL).Err()
}

// Count returns the total number of active sessions
func (s *SessionStore) Count(ctx context.Context) (int64, error) {
	// Use SCAN to count session keys (doesn't block Redis)
	var count int64
	iter := s.client.Scan(ctx, 0, s.keyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return count, nil
}

// Close gracefully shuts down the session store and its worker pool (if enabled)
func (s *SessionStore) Close(timeout time.Duration) error {
	if s.workerPool != nil {
		return s.workerPool.Shutdown(timeout)
	}
	return nil
}

// GetWorkerPoolMetrics returns worker pool metrics if worker pool is enabled
// Returns nil if worker pool is not enabled
func (s *SessionStore) GetWorkerPoolMetrics() *MetricsSnapshot {
	if s.workerPool == nil {
		return nil
	}
	snapshot := s.workerPool.GetMetrics().Snapshot()
	return &snapshot
}
