package cache

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// InvalidationService handles cache invalidation strategies
type InvalidationService struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewInvalidationService creates a new cache invalidation service
func NewInvalidationService(redis *redis.Client, logger *slog.Logger) *InvalidationService {
	return &InvalidationService{
		redis:  redis,
		logger: logger,
	}
}

// =============================================================================
// CONVERSATION CACHE INVALIDATION
// =============================================================================

// InvalidateConversation invalidates all caches related to a conversation
func (s *InvalidationService) InvalidateConversation(ctx context.Context, conversationID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("conv:%s:*", conversationID),                    // All conversation data
		fmt.Sprintf("conv:%s:messages", conversationID),             // Message history
		fmt.Sprintf("conv:%s:messages:page:*", conversationID),      // Paginated messages
		// REMOVED: fmt.Sprintf("conversations:*") - overly-broad wildcard causing cache stampede
		// Only invalidate conversation-specific data, not all users' conversation lists
	}

	return s.invalidatePatterns(ctx, patterns)
}

// InvalidateConversationList invalidates conversation list cache for a user
func (s *InvalidationService) InvalidateConversationList(ctx context.Context, userID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("conversations:%s:*", userID),                   // All conversation list pages
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// MEMORY CACHE INVALIDATION
// =============================================================================

// InvalidateMemory invalidates caches for a specific memory
func (s *InvalidationService) InvalidateMemory(ctx context.Context, memoryID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("memory:%s", memoryID),                          // Single memory
		// REMOVED: fmt.Sprintf("memories:*") - overly-broad wildcard causing cache stampede
		// Only invalidate memory-specific data, not all users' memory lists
		fmt.Sprintf("embed:memory:%s", memoryID),                    // Memory embedding
	}

	return s.invalidatePatterns(ctx, patterns)
}

// InvalidateMemoryList invalidates memory list cache
func (s *InvalidationService) InvalidateMemoryList(ctx context.Context, userID uuid.UUID, workspaceID *uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("memories:%s:*", userID),
	}

	if workspaceID != nil {
		patterns = append(patterns, fmt.Sprintf("memories:%s:workspace:%s:*", userID, *workspaceID))
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// ARTIFACT CACHE INVALIDATION
// =============================================================================

// InvalidateArtifact invalidates artifact caches
func (s *InvalidationService) InvalidateArtifact(ctx context.Context, artifactID uuid.UUID, userID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("artifact:%s", artifactID),
		fmt.Sprintf("artifacts:%s:*", userID),                       // User's artifact lists
	}

	return s.invalidatePatterns(ctx, patterns)
}

// InvalidateArtifactsByConversation invalidates artifacts for a conversation
func (s *InvalidationService) InvalidateArtifactsByConversation(ctx context.Context, conversationID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("artifacts:conv:%s", conversationID),
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// TASK CACHE INVALIDATION
// =============================================================================

// InvalidateTask invalidates task caches
func (s *InvalidationService) InvalidateTask(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("task:%s", taskID),
		fmt.Sprintf("tasks:%s:*", userID),                           // User's task lists
	}

	return s.invalidatePatterns(ctx, patterns)
}

// InvalidateTasksByProject invalidates tasks for a project
func (s *InvalidationService) InvalidateTasksByProject(ctx context.Context, projectID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("tasks:project:%s:*", projectID),
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// AGENT STATUS CACHE INVALIDATION
// =============================================================================

// InvalidateAgentStatus invalidates agent status cache
func (s *InvalidationService) InvalidateAgentStatus(ctx context.Context, agentID string) error {
	patterns := []string{
		fmt.Sprintf("agent:%s:status", agentID),
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// EMBEDDING CACHE INVALIDATION
// =============================================================================

// InvalidateEmbedding invalidates a specific embedding cache
func (s *InvalidationService) InvalidateEmbedding(ctx context.Context, textHash string) error {
	key := fmt.Sprintf("embed:%s", textHash)

	if err := s.redis.Del(ctx, key).Err(); err != nil {
		s.logger.Error("Failed to invalidate embedding cache",
			"key", key,
			"error", err)
		return fmt.Errorf("invalidate embedding: %w", err)
	}

	return nil
}

// InvalidateAllEmbeddings clears all embedding caches (use with caution)
func (s *InvalidationService) InvalidateAllEmbeddings(ctx context.Context) error {
	patterns := []string{"embed:*"}
	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// WORKSPACE CACHE INVALIDATION
// =============================================================================

// InvalidateWorkspace invalidates workspace-related caches
func (s *InvalidationService) InvalidateWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("workspace:%s:*", workspaceID),
		// REMOVED: fmt.Sprintf("workspaces:*") - overly-broad wildcard causing cache stampede
		// Only invalidate workspace-specific data, not all users' workspace lists
	}

	return s.invalidatePatterns(ctx, patterns)
}

// =============================================================================
// BULK INVALIDATION
// =============================================================================

// InvalidateAll clears all application caches (nuclear option)
func (s *InvalidationService) InvalidateAll(ctx context.Context) error {
	// PRODUCTION SAFETY: Require explicit environment variable
	if os.Getenv("REDIS_ALLOW_FLUSH") != "true" {
		return fmt.Errorf("FlushDB disabled - set REDIS_ALLOW_FLUSH=true to enable")
	}

	// AUDIT LOG with critical severity
	s.logger.Error("CRITICAL: Flushing entire Redis database",
		"timestamp", time.Now(),
		"operation", "FlushDB")

	if err := s.redis.FlushDB(ctx).Err(); err != nil {
		s.logger.Error("Failed to flush all caches", "error", err)
		return fmt.Errorf("flush all caches: %w", err)
	}

	s.logger.Info("All caches cleared successfully")
	return nil
}

// InvalidateByPrefix clears all keys matching a prefix
func (s *InvalidationService) InvalidateByPrefix(ctx context.Context, prefix string) error {
	pattern := prefix + "*"
	return s.invalidatePatterns(ctx, []string{pattern})
}

// =============================================================================
// TIME-BASED INVALIDATION (TTL Management)
// =============================================================================

// SetTTL sets a TTL on a cache key
func (s *InvalidationService) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	if err := s.redis.Expire(ctx, key, ttl).Err(); err != nil {
		s.logger.Error("Failed to set TTL",
			"key", key,
			"ttl", ttl,
			"error", err)
		return fmt.Errorf("set TTL: %w", err)
	}

	return nil
}

// =============================================================================
// INTERNAL HELPERS
// =============================================================================

// invalidatePatterns deletes all keys matching the given patterns
func (s *InvalidationService) invalidatePatterns(ctx context.Context, patterns []string) error {
	var allKeys []string

	for _, pattern := range patterns {
		keys, err := s.scanPattern(ctx, pattern)
		if err != nil {
			s.logger.Error("Failed to scan pattern",
				"pattern", pattern,
				"error", err)
			continue
		}
		allKeys = append(allKeys, keys...)
	}

	if len(allKeys) == 0 {
		return nil
	}

	// Delete all keys in a pipeline for efficiency
	pipe := s.redis.Pipeline()
	for _, key := range allKeys {
		pipe.Del(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Error("Failed to delete cache keys",
			"count", len(allKeys),
			"error", err)
		return fmt.Errorf("delete cache keys: %w", err)
	}

	s.logger.Debug("Invalidated cache keys",
		"patterns", patterns,
		"count", len(allKeys))

	return nil
}

// scanPattern scans for all keys matching a pattern
func (s *InvalidationService) scanPattern(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	iter := s.redis.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan pattern: %w", err)
	}

	return keys, nil
}

// =============================================================================
// INVALIDATION EVENTS (for pub/sub)
// =============================================================================

// InvalidationEvent represents a cache invalidation event
type InvalidationEvent struct {
	Type      string    `json:"type"`       // conversation, memory, artifact, task, etc.
	EntityID  string    `json:"entity_id"`  // UUID of the entity
	UserID    string    `json:"user_id"`    // User who triggered invalidation
	Timestamp time.Time `json:"timestamp"`
}

// PublishInvalidationEvent publishes an invalidation event for distributed systems
func (s *InvalidationService) PublishInvalidationEvent(ctx context.Context, event InvalidationEvent) error {
	event.Timestamp = time.Now()

	channel := "cache:invalidation"

	if err := s.redis.Publish(ctx, channel, event).Err(); err != nil {
		s.logger.Error("Failed to publish invalidation event",
			"channel", channel,
			"event", event,
			"error", err)
		return fmt.Errorf("publish invalidation: %w", err)
	}

	return nil
}
