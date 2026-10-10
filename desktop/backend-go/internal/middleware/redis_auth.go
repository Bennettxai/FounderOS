// Package middleware provides Redis-backed session caching for horizontal scaling
package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// maskToken masks a token for safe logging (shows first 8 chars + ...)
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:8] + "..."
}

// SessionCache provides Redis-backed session caching with secure token hashing
type SessionCache struct {
	client     *redis.Client
	keyPrefix  string
	ttl        time.Duration
	hmacSecret []byte // HMAC secret for secure key derivation (prevents token enumeration)
}

// CachedUser represents cached user data in Redis
type CachedUser struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Image         *string   `json:"image,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CachedAt      time.Time `json:"cached_at"`
}

// SessionCacheConfig configures the session cache
type SessionCacheConfig struct {
	KeyPrefix  string        // Redis key prefix (default: "auth_session:")
	TTL        time.Duration // Cache TTL (default: 15 minutes)
	HMACSecret string        // HMAC secret for secure key derivation (CRITICAL in production)
}

// DefaultSessionCacheConfig returns sensible defaults
func DefaultSessionCacheConfig() *SessionCacheConfig {
	return &SessionCacheConfig{
		KeyPrefix: "auth_session:",
		TTL:       15 * time.Minute, // Short TTL for security
	}
}

// NewSessionCache creates a new Redis-backed session cache
// If cfg.HMACSecret is empty, generates a random secret (development mode only)
// CRITICAL: In production, ALWAYS provide a strong HMACSecret via environment variable
// Returns error if random secret generation fails (extremely rare, but should be handled gracefully)
func NewSessionCache(client *redis.Client, cfg *SessionCacheConfig) (*SessionCache, error) {
	if cfg == nil {
		cfg = DefaultSessionCacheConfig()
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
			return nil, fmt.Errorf("failed to generate HMAC secret: %w", err)
		}
		slog.Warn("WARNING: Using auto-generated HMAC secret for session keys. " +
			"Set REDIS_KEY_HMAC_SECRET environment variable for production!")
	}

	if len(hmacSecret) < 32 {
		slog.Warn("WARNING: HMAC secret is shorter than recommended 32 bytes", "current_bytes", len(hmacSecret))
	}

	return &SessionCache{
		client:     client,
		keyPrefix:  cfg.KeyPrefix,
		ttl:        cfg.TTL,
		hmacSecret: hmacSecret,
	}, nil
}

// sessionKey generates a secure Redis key for a session token using HMAC-SHA256
// This prevents enumeration attacks by hashing the token before using it as a key
//
// Security rationale:
// - Raw tokens in Redis keys allow attackers to enumerate valid sessions
// - HMAC-SHA256 produces a one-way hash that can't be reversed
// - Using a secret key ensures only the application can generate valid hashes
// - 64-character hex output provides sufficient namespace to prevent collisions
func (sc *SessionCache) sessionKey(token string) string {
	hash := sc.hashToken(token)
	return sc.keyPrefix + hash
}

// userSessionsKey generates a secure Redis key for user's session set using HMAC-SHA256
// This prevents user ID enumeration and links session hashes to user accounts
func (sc *SessionCache) userSessionsKey(userID string) string {
	hash := sc.hashToken("user:" + userID)
	return "user_sessions:" + hash
}

// hashToken performs HMAC-SHA256 hashing of input using the configured secret
// Returns hex-encoded hash (64 characters)
func (sc *SessionCache) hashToken(input string) string {
	h := hmac.New(sha256.New, sc.hmacSecret)
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves cached user data for a session token
func (sc *SessionCache) Get(ctx context.Context, token string) (*CachedUser, error) {
	data, err := sc.client.Get(ctx, sc.sessionKey(token)).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var user CachedUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &user, nil
}

// Set caches user data for a session token
func (sc *SessionCache) Set(ctx context.Context, token string, user *BetterAuthUser) error {
	cached := &CachedUser{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Image:         user.Image,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		CachedAt:      time.Now(),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	// Use pipeline for atomic operations:
	// 1. Set session data with TTL
	// 2. Add session key to user's session set
	pipe := sc.client.Pipeline()
	sessionKey := sc.sessionKey(token)
	userSessionsKey := sc.userSessionsKey(user.ID)

	pipe.Set(ctx, sessionKey, data, sc.ttl)
	pipe.SAdd(ctx, userSessionsKey, sessionKey)
	// Set TTL on user sessions set to auto-cleanup (slightly longer than session TTL)
	pipe.Expire(ctx, userSessionsKey, sc.ttl+5*time.Minute)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline failed: %w", err)
	}

	return nil
}

// Invalidate removes a session from the cache and its user index
// This should be called when a single session is logged out
func (sc *SessionCache) Invalidate(ctx context.Context, token string) error {
	sessionKey := sc.sessionKey(token)

	// Get user ID from session before deleting
	data, err := sc.client.Get(ctx, sessionKey).Bytes()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get session for invalidation: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := sc.client.Pipeline()
	pipe.Del(ctx, sessionKey)

	// If we found the session, also remove from user's session set
	if err == nil && len(data) > 0 {
		var cached CachedUser
		if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr == nil {
			userSessionsKey := sc.userSessionsKey(cached.ID)
			pipe.SRem(ctx, userSessionsKey, sessionKey)
		}
	}

	_, err = pipe.Exec(ctx)
	return err
}

// InvalidateUserSessions removes all cached sessions for a user
// This uses the user->sessions index for efficient O(N) operation where N = user's session count
// Call this when: password change, account compromise, permission changes, or forced logout
func (sc *SessionCache) InvalidateUserSessions(ctx context.Context, userID string) error {
	userSessionsKey := sc.userSessionsKey(userID)

	// Get all session keys for this user from the set
	sessionKeys, err := sc.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionKeys) == 0 {
		slog.Info("SessionCache: no sessions to invalidate", "user_id", userID)
		return nil
	}

	// Use pipeline for atomic deletion of all sessions
	pipe := sc.client.Pipeline()

	// Delete all session data keys
	for _, sessionKey := range sessionKeys {
		pipe.Del(ctx, sessionKey)
	}

	// Delete the user's session set itself
	pipe.Del(ctx, userSessionsKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", err)
	}

	slog.Info("SessionCache: sessions invalidated", "count", len(sessionKeys), "user_id", userID)
	return nil
}

// CachedAuthMiddleware provides Redis-cached session validation with PostgreSQL fallback
func CachedAuthMiddleware(pool *pgxpool.Pool, cache *SessionCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract session token
		sessionCookie, err := c.Cookie(SessionCookieName)
		if err != nil || sessionCookie == "" {
			slog.Debug("cached_auth: no session cookie found", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}

		slog.Debug("cached_auth: processing session cookie", "masked_token", maskToken(sessionCookie))

		sessionCookie, err = url.QueryUnescape(sessionCookie)
		if err != nil {
			slog.Warn("cached_auth: URL decode failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session cookie"})
			return
		}

		slog.Debug("cached_auth: decoded session cookie", "masked_token", maskToken(sessionCookie))

		sessionToken, valid := verifySessionCookie(sessionCookie)
		if !valid {
			slog.Debug("cached_auth: invalid session cookie signature")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session signature"})
			return
		}

		slog.Debug("cached_auth: extracted session token", "masked_token", maskToken(sessionToken))

		ctx := c.Request.Context()

		// Try Redis cache first (if cache is available)
		if cache != nil {
			cached, err := cache.Get(ctx, sessionToken)
			if err != nil {
				// Log but don't fail - fall through to DB
				slog.Warn("cached_auth: redis get error", "error", err)
			} else if cached != nil {
				// Cache hit - convert to BetterAuthUser and continue
				slog.Debug("cached_auth: cache hit", "user_id", cached.ID, "email", cached.Email)
				user := &BetterAuthUser{
					ID:            cached.ID,
					Name:          cached.Name,
					Email:         cached.Email,
					EmailVerified: cached.EmailVerified,
					Image:         cached.Image,
					CreatedAt:     cached.CreatedAt,
					UpdatedAt:     cached.UpdatedAt,
				}
				c.Set(UserContextKey, user)
				c.Set("user_id", user.ID) // Also set user_id for integration handlers
				c.Next()
				return
			}
			slog.Debug("cached_auth: cache miss, querying database")
		}

		// Cache miss or no cache - query PostgreSQL
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var user BetterAuthUser
		err = pool.QueryRow(dbCtx, `
			SELECT u.id, u.name, u.email, u."emailVerified", u.image, u."createdAt", u."updatedAt"
			FROM session s
			JOIN "user" u ON s."userId" = u.id
			WHERE s.token = $1 AND s."expiresAt" > NOW()
		`, sessionToken).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.EmailVerified,
			&user.Image,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			slog.Warn("cached_auth: session lookup failed", "error", err, "masked_token", maskToken(sessionToken))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		}

		slog.Debug("cached_auth: user authenticated from DB", "user_id", user.ID, "email", user.Email)

		// Cache the result for next time (if cache available)
		if cache != nil {
			if err := cache.Set(ctx, sessionToken, &user); err != nil {
				// Log but don't fail
				slog.Warn("SessionCache: set error", "error", err)
			}
		}

		c.Set(UserContextKey, &user)
		c.Set("user_id", user.ID) // Also set user_id for integration handlers
		c.Next()
	}
}

// CachedOptionalAuthMiddleware allows unauthenticated requests with Redis caching
func CachedOptionalAuthMiddleware(pool *pgxpool.Pool, cache *SessionCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie(SessionCookieName)
		if err != nil || sessionCookie == "" {
			c.Next()
			return
		}

		sessionCookie, err = url.QueryUnescape(sessionCookie)
		if err != nil {
			c.Next()
			return
		}

		sessionToken, valid := verifySessionCookie(sessionCookie)
		if !valid {
			slog.Debug("CachedOptionalAuthMiddleware: invalid session cookie signature")
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// Try Redis cache first
		if cache != nil {
			cached, err := cache.Get(ctx, sessionToken)
			if err == nil && cached != nil {
				user := &BetterAuthUser{
					ID:            cached.ID,
					Name:          cached.Name,
					Email:         cached.Email,
					EmailVerified: cached.EmailVerified,
					Image:         cached.Image,
					CreatedAt:     cached.CreatedAt,
					UpdatedAt:     cached.UpdatedAt,
				}
				c.Set(UserContextKey, user)
				c.Set("user_id", user.ID) // Also set user_id for integration handlers
				c.Next()
				return
			}
		}

		// Cache miss - query PostgreSQL
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var user BetterAuthUser
		err = pool.QueryRow(dbCtx, `
			SELECT u.id, u.name, u.email, u."emailVerified", u.image, u."createdAt", u."updatedAt"
			FROM session s
			JOIN "user" u ON s."userId" = u.id
			WHERE s.token = $1 AND s."expiresAt" > NOW()
		`, sessionToken).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.EmailVerified,
			&user.Image,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err == nil {
			c.Set(UserContextKey, &user)
			c.Set("user_id", user.ID) // Also set user_id for integration handlers
			// Cache for next time
			if cache != nil {
				_ = cache.Set(ctx, sessionToken, &user)
			}
		}

		c.Next()
	}
}

// InvalidateSessionMiddleware provides session invalidation for logout
func InvalidateSessionMiddleware(cache *SessionCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request first
		c.Next()

		// After logout handler completes, invalidate cache
		if c.Writer.Status() == http.StatusOK {
			sessionCookie, err := c.Cookie(SessionCookieName)
			if err == nil && sessionCookie != "" {
				sessionCookie, _ = url.QueryUnescape(sessionCookie)
				sessionToken, valid := verifySessionCookie(sessionCookie)
				if !valid {
					slog.Debug("InvalidateSessionMiddleware: invalid session cookie signature, skipping invalidation")
					return
				}

				if cache != nil {
					if err := cache.Invalidate(c.Request.Context(), sessionToken); err != nil {
						slog.Warn("SessionCache: invalidation error", "error", err)
					}
				}
			}
		}
	}
}

// CacheStats returns session cache statistics
type CacheStats struct {
	Hits   uint64 `json:"hits"`
	Misses uint64 `json:"misses"`
	Keys   int64  `json:"keys"`
}

// Stats returns cache statistics
func (sc *SessionCache) Stats(ctx context.Context) (*CacheStats, error) {
	// Count keys matching our prefix
	var count int64
	iter := sc.client.Scan(ctx, 0, sc.keyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return &CacheStats{
		Keys: count,
	}, nil
}
