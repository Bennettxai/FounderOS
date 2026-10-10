package services

import (
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextTrackerService monitors token usage and manages context window eviction
type ContextTrackerService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	mu     sync.RWMutex

	// In-memory LRU cache for active conversations
	activeContexts map[string]*ConversationContext
	maxContexts    int

	// Token budget defaults
	defaultMaxTokens    int
	defaultReserveRatio float64
}

// ConversationContext tracks token usage for a conversation
type ConversationContext struct {
	ConversationID string                 `json:"conversation_id"`
	UserID         string                 `json:"user_id"`
	TotalTokens    int                    `json:"total_tokens"`
	MaxTokens      int                    `json:"max_tokens"`
	ReserveTokens  int                    `json:"reserve_tokens"`
	Blocks         []*ContextBlock        `json:"blocks"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	LastAccessedAt time.Time              `json:"last_accessed_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ContextBlock represents a block of content in the context window
type ContextBlock struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // system, user, assistant, memory, document
	Content     string                 `json:"content"`
	TokenCount  int                    `json:"token_count"`
	Priority    int                    `json:"priority"` // Higher = more important
	Pinned      bool                   `json:"pinned"`   // Never evict pinned blocks
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	AccessCount int                    `json:"access_count"`
}

// TokenUsageStats provides statistics about token usage
type TokenUsageStats struct {
	ConversationID  string  `json:"conversation_id"`
	TotalTokens     int     `json:"total_tokens"`
	MaxTokens       int     `json:"max_tokens"`
	UsagePercentage float64 `json:"usage_percentage"`
	AvailableTokens int     `json:"available_tokens"`
	ReserveTokens   int     `json:"reserve_tokens"`
	BlockCount      int     `json:"block_count"`
	PinnedTokens    int     `json:"pinned_tokens"`
	EvictableTokens int     `json:"evictable_tokens"`
	OldestBlockAge  string  `json:"oldest_block_age"`
	EstimatedTurns  int     `json:"estimated_turns"`
}

// EvictionResult contains results of a context eviction
type EvictionResult struct {
	EvictedBlocks  []string `json:"evicted_blocks"`
	TokensFreed    int      `json:"tokens_freed"`
	BlocksRemoved  int      `json:"blocks_removed"`
	NewTotalTokens int      `json:"new_total_tokens"`
}

// ContextTrackerConfig configuration for the service
type ContextTrackerConfig struct {
	MaxActiveContexts   int
	DefaultMaxTokens    int
	DefaultReserveRatio float64
}

// NewContextTrackerService creates a new context tracker service
func NewContextTrackerService(pool *pgxpool.Pool, logger *slog.Logger, config *ContextTrackerConfig) *ContextTrackerService {
	if config == nil {
		config = &ContextTrackerConfig{
			MaxActiveContexts:   100,
			DefaultMaxTokens:    128000, // Claude's context window
			DefaultReserveRatio: 0.2,    // Reserve 20% for response
		}
	}

	return &ContextTrackerService{
		pool:                pool,
		logger:              logger,
		activeContexts:      make(map[string]*ConversationContext),
		maxContexts:         config.MaxActiveContexts,
		defaultMaxTokens:    config.DefaultMaxTokens,
		defaultReserveRatio: config.DefaultReserveRatio,
	}
}
