package services

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestContextTracker(maxContexts int) *ContextTrackerService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &ContextTrackerConfig{
		MaxActiveContexts:   maxContexts,
		DefaultMaxTokens:    10000,
		DefaultReserveRatio: 0.2,
	}
	return &ContextTrackerService{
		pool:                nil, // DB not used by pure-logic tests
		logger:              logger,
		activeContexts:      make(map[string]*ConversationContext),
		maxContexts:         cfg.MaxActiveContexts,
		defaultMaxTokens:    cfg.DefaultMaxTokens,
		defaultReserveRatio: cfg.DefaultReserveRatio,
	}
}

// seedContext injects a ConversationContext directly into the in-memory cache.
func seedContext(svc *ContextTrackerService, cc *ConversationContext) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.activeContexts[cc.ConversationID] = cc
}

// newContext returns a minimal ConversationContext for a given conversation.
func newConvContext(convID string, maxTokens, reserveTokens int) *ConversationContext {
	return &ConversationContext{
		ConversationID: convID,
		UserID:         "user-test",
		TotalTokens:    0,
		MaxTokens:      maxTokens,
		ReserveTokens:  reserveTokens,
		Blocks:         make([]*ContextBlock, 0),
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// newBlock creates a test ContextBlock.
func newBlock(id, blockType string, tokens, priority int, pinned bool) *ContextBlock {
	return &ContextBlock{
		ID:          id,
		Type:        blockType,
		Content:     "test content for " + id,
		TokenCount:  tokens,
		Priority:    priority,
		Pinned:      pinned,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		AccessCount: 1,
	}
}

// ---------------------------------------------------------------------------
// EstimateTokens — pure function, no dependencies
// ---------------------------------------------------------------------------

func TestEstimateTokens_EmptyString(t *testing.T) {
	result := EstimateTokens("")

	assert.Equal(t, 0, result, "empty string should estimate 0 tokens")
}

func TestEstimateTokens_FourCharactersPerToken(t *testing.T) {
	// "test" = 4 chars → 1 token
	result := EstimateTokens("test")

	assert.Equal(t, 1, result)
}

func TestEstimateTokens_LargeInput(t *testing.T) {
	// 400 chars → 100 tokens
	text := make([]byte, 400)
	for i := range text {
		text[i] = 'a'
	}
	result := EstimateTokens(string(text))

	assert.Equal(t, 100, result)
}

func TestEstimateTokens_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"4 chars", "abcd", 1},
		{"8 chars", "abcdefgh", 2},
		{"3 chars rounds down", "abc", 0},
		{"12 chars", "abcdefghijkl", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, EstimateTokens(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// evictBlocks — priority-ordered eviction
// ---------------------------------------------------------------------------

func TestEvictBlocks_EvictsLowestPriorityFirst(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-1", 10000, 2000)

	// priority 1 = lowest → evicted first
	cc.Blocks = []*ContextBlock{
		newBlock("b-high", "user", 500, 5, false),
		newBlock("b-low", "user", 500, 1, false),
		newBlock("b-mid", "user", 500, 3, false),
	}
	cc.TotalTokens = 1500

	result := svc.evictBlocks(cc, 500)

	require.Equal(t, 1, result.BlocksRemoved, "exactly one block should be evicted")
	assert.Equal(t, "b-low", result.EvictedBlocks[0], "lowest priority block should be evicted first")
	assert.Equal(t, 500, result.TokensFreed)
}

func TestEvictBlocks_SkipsPinnedBlocks(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-pin", 10000, 2000)

	cc.Blocks = []*ContextBlock{
		newBlock("pinned-block", "system", 1000, 1, true), // pinned, should NOT be evicted
		newBlock("evictable", "user", 500, 2, false),
	}
	cc.TotalTokens = 1500

	result := svc.evictBlocks(cc, 500)

	require.Equal(t, 1, result.BlocksRemoved)
	assert.Equal(t, "evictable", result.EvictedBlocks[0])

	// Pinned block must still be in cc.Blocks.
	require.Len(t, cc.Blocks, 1)
	assert.True(t, cc.Blocks[0].Pinned)
}

func TestEvictBlocks_EvictsMultipleBlocksWhenNeeded(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-multi", 10000, 2000)

	cc.Blocks = []*ContextBlock{
		newBlock("b1", "user", 300, 1, false),
		newBlock("b2", "user", 300, 2, false),
		newBlock("b3", "user", 300, 3, false),
	}
	cc.TotalTokens = 900

	// Need 700 tokens freed: will evict b1+b2 (600 tokens) first, then b3 if still needed.
	result := svc.evictBlocks(cc, 700)

	assert.GreaterOrEqual(t, result.TokensFreed, 700,
		"eviction should free at least the requested tokens")
	assert.GreaterOrEqual(t, result.BlocksRemoved, 2)
}

func TestEvictBlocks_NoPinnedBlocksAvailable_EvictsNothing(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-all-pinned", 10000, 2000)

	cc.Blocks = []*ContextBlock{
		newBlock("p1", "system", 500, 1, true),
		newBlock("p2", "system", 500, 2, true),
	}
	cc.TotalTokens = 1000

	result := svc.evictBlocks(cc, 500)

	assert.Equal(t, 0, result.BlocksRemoved, "no unpinned blocks to evict")
	assert.Equal(t, 0, result.TokensFreed)
	assert.Len(t, cc.Blocks, 2, "pinned blocks remain untouched")
}

func TestEvictBlocks_TotalTokensUpdatedAfterEviction(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-token-update", 10000, 2000)

	cc.Blocks = []*ContextBlock{
		newBlock("b1", "user", 400, 1, false),
		newBlock("b2", "user", 600, 5, false),
	}
	cc.TotalTokens = 1000

	result := svc.evictBlocks(cc, 400)

	assert.Equal(t, result.NewTotalTokens, cc.TotalTokens,
		"EvictionResult.NewTotalTokens should match the context's updated TotalTokens")
}

// ---------------------------------------------------------------------------
// evictLRUIfNeeded — cache management
// ---------------------------------------------------------------------------

func TestEvictLRUIfNeeded_DoesNotEvictWhenUnderLimit(t *testing.T) {
	svc := newTestContextTracker(3)

	seedContext(svc, newConvContext("c1", 10000, 2000))
	seedContext(svc, newConvContext("c2", 10000, 2000))

	svc.mu.Lock()
	svc.evictLRUIfNeeded()
	count := len(svc.activeContexts)
	svc.mu.Unlock()

	assert.Equal(t, 2, count, "nothing should be evicted when under the limit")
}

func TestEvictLRUIfNeeded_EvictsLeastRecentlyUsedContext(t *testing.T) {
	svc := newTestContextTracker(2) // limit = 2

	oldest := newConvContext("conv-old", 10000, 2000)
	oldest.LastAccessedAt = time.Now().Add(-10 * time.Minute)

	recent := newConvContext("conv-new", 10000, 2000)
	recent.LastAccessedAt = time.Now()

	seedContext(svc, oldest)
	seedContext(svc, recent)

	// Inserting a third entry triggers LRU eviction.
	newest := newConvContext("conv-newest", 10000, 2000)
	newest.LastAccessedAt = time.Now().Add(1 * time.Second)
	seedContext(svc, newest)

	svc.mu.Lock()
	svc.evictLRUIfNeeded()
	remaining := make([]string, 0, len(svc.activeContexts))
	for k := range svc.activeContexts {
		remaining = append(remaining, k)
	}
	svc.mu.Unlock()

	assert.NotContains(t, remaining, "conv-old",
		"the least recently used context should have been evicted")
	assert.Len(t, remaining, 2)
}

func TestEvictLRUIfNeeded_ExactlyAtLimit_NoEviction(t *testing.T) {
	svc := newTestContextTracker(2)

	seedContext(svc, newConvContext("c1", 10000, 2000))
	seedContext(svc, newConvContext("c2", 10000, 2000))

	svc.mu.Lock()
	svc.evictLRUIfNeeded()
	count := len(svc.activeContexts)
	svc.mu.Unlock()

	assert.Equal(t, 2, count, "exactly at limit should not trigger eviction")
}

// ---------------------------------------------------------------------------
// GetTokenUsageStats — in-memory computation
// ---------------------------------------------------------------------------

func TestGetTokenUsageStats_CalculatesUsagePercentage(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-stats", 10000, 2000)
	cc.TotalTokens = 4000
	// Usage = 4000 / (10000 - 2000) * 100 = 50%
	seedContext(svc, cc)

	stats, err := svc.GetTokenUsageStats(context.Background(), "conv-stats")

	require.NoError(t, err)
	assert.InDelta(t, 50.0, stats.UsagePercentage, 0.01)
	assert.Equal(t, 4000, stats.TotalTokens)
	assert.Equal(t, 4000, stats.AvailableTokens) // 10000 - 2000 - 4000
}

func TestGetTokenUsageStats_CountsPinnedAndEvictableTokens(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-pinned-stats", 10000, 2000)
	cc.Blocks = []*ContextBlock{
		newBlock("pinned", "system", 300, 5, true),
		newBlock("evict1", "user", 200, 1, false),
		newBlock("evict2", "user", 100, 2, false),
	}
	cc.TotalTokens = 600
	seedContext(svc, cc)

	stats, err := svc.GetTokenUsageStats(context.Background(), "conv-pinned-stats")

	require.NoError(t, err)
	assert.Equal(t, 300, stats.PinnedTokens)
	assert.Equal(t, 300, stats.EvictableTokens)
	assert.Equal(t, 3, stats.BlockCount)
}

func TestGetTokenUsageStats_ReturnsErrorForUnknownConversation(t *testing.T) {
	svc := newTestContextTracker(10)

	_, err := svc.GetTokenUsageStats(context.Background(), "does-not-exist")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestGetTokenUsageStats_EstimatedTurns(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-turns", 10000, 2000)
	// 2 blocks × 500 tokens each = 1000 total; avg = 500/block
	// Available = 10000 - 2000 - 1000 = 7000 → estimated turns = 7000/500 = 14
	cc.Blocks = []*ContextBlock{
		newBlock("b1", "user", 500, 1, false),
		newBlock("b2", "user", 500, 2, false),
	}
	cc.TotalTokens = 1000
	seedContext(svc, cc)

	stats, err := svc.GetTokenUsageStats(context.Background(), "conv-turns")

	require.NoError(t, err)
	assert.Equal(t, 14, stats.EstimatedTurns)
}

func TestGetTokenUsageStats_NoBlocks_EstimatedTurnsIsZero(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-no-blocks", 10000, 2000)
	seedContext(svc, cc)

	stats, err := svc.GetTokenUsageStats(context.Background(), "conv-no-blocks")

	require.NoError(t, err)
	assert.Equal(t, 0, stats.EstimatedTurns)
}

// ---------------------------------------------------------------------------
// GetActiveContextCount
// ---------------------------------------------------------------------------

func TestGetActiveContextCount_ReflectsCurrentCacheSize(t *testing.T) {
	svc := newTestContextTracker(10)

	assert.Equal(t, 0, svc.GetActiveContextCount())

	seedContext(svc, newConvContext("c1", 10000, 2000))
	assert.Equal(t, 1, svc.GetActiveContextCount())

	seedContext(svc, newConvContext("c2", 10000, 2000))
	assert.Equal(t, 2, svc.GetActiveContextCount())
}

// ---------------------------------------------------------------------------
// IncrementBlockAccess — in-memory only
// ---------------------------------------------------------------------------

func TestIncrementBlockAccess_IncrementsCountForKnownBlock(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-access", 10000, 2000)
	cc.Blocks = []*ContextBlock{newBlock("block-1", "user", 100, 1, false)}
	cc.Blocks[0].AccessCount = 1
	seedContext(svc, cc)

	err := svc.IncrementBlockAccess(context.Background(), "conv-access", "block-1")

	require.NoError(t, err)
	assert.Equal(t, 2, cc.Blocks[0].AccessCount)
}

func TestIncrementBlockAccess_ReturnsErrorForUnknownConversation(t *testing.T) {
	svc := newTestContextTracker(10)

	err := svc.IncrementBlockAccess(context.Background(), "conv-missing", "block-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conv-missing")
}

func TestIncrementBlockAccess_ReturnsErrorForUnknownBlock(t *testing.T) {
	svc := newTestContextTracker(10)
	seedContext(svc, newConvContext("conv-1", 10000, 2000))

	err := svc.IncrementBlockAccess(context.Background(), "conv-1", "no-such-block")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-block")
}

// ---------------------------------------------------------------------------
// GetBlocks / GetBlocksByType — in-memory only
// ---------------------------------------------------------------------------

func TestGetBlocks_ReturnsAllBlocks(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-get", 10000, 2000)
	cc.Blocks = []*ContextBlock{
		newBlock("b1", "user", 100, 1, false),
		newBlock("b2", "system", 200, 5, true),
	}
	seedContext(svc, cc)

	blocks, err := svc.GetBlocks(context.Background(), "conv-get")

	require.NoError(t, err)
	assert.Len(t, blocks, 2)
}

func TestGetBlocks_ReturnsErrorForUnknownConversation(t *testing.T) {
	svc := newTestContextTracker(10)

	_, err := svc.GetBlocks(context.Background(), "nope")

	require.Error(t, err)
}

func TestGetBlocksByType_FiltersCorrectly(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-typed", 10000, 2000)
	cc.Blocks = []*ContextBlock{
		newBlock("b-user-1", "user", 100, 1, false),
		newBlock("b-system", "system", 200, 5, true),
		newBlock("b-user-2", "user", 150, 2, false),
	}
	seedContext(svc, cc)

	userBlocks, err := svc.GetBlocksByType(context.Background(), "conv-typed", "user")

	require.NoError(t, err)
	assert.Len(t, userBlocks, 2)
	for _, b := range userBlocks {
		assert.Equal(t, "user", b.Type)
	}
}

func TestGetBlocksByType_ReturnsEmptySliceForNoMatch(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-nomatch", 10000, 2000)
	cc.Blocks = []*ContextBlock{
		newBlock("b1", "user", 100, 1, false),
	}
	seedContext(svc, cc)

	blocks, err := svc.GetBlocksByType(context.Background(), "conv-nomatch", "assistant")

	require.NoError(t, err)
	assert.Empty(t, blocks)
}

// ---------------------------------------------------------------------------
// ClearContext — keeps pinned, removes unpinned
// ---------------------------------------------------------------------------

func TestClearContext_RemovesUnpinnedBlocks(t *testing.T) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-clear", 10000, 2000)
	cc.Blocks = []*ContextBlock{
		newBlock("pinned", "system", 300, 5, true),
		newBlock("regular", "user", 200, 1, false),
	}
	cc.TotalTokens = 500
	seedContext(svc, cc)

	// ClearContext calls saveContextToDB which uses the pool → will error on nil.
	// That error is returned alongside the result.
	result, _ := svc.ClearContext(context.Background(), "conv-clear")

	require.NotNil(t, result)
	assert.Equal(t, 1, result.BlocksRemoved)
	assert.Equal(t, 200, result.TokensFreed)
	assert.Equal(t, 300, cc.TotalTokens, "only pinned tokens should remain")
}

func TestClearContext_ReturnsErrorForUnknownConversation(t *testing.T) {
	svc := newTestContextTracker(10)

	_, err := svc.ClearContext(context.Background(), "nope")

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// min helper
// ---------------------------------------------------------------------------

func TestMin_ReturnsLesserValue(t *testing.T) {
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, 0, min(0, 0))
	assert.Equal(t, 0, min(0, 100))
}

// ---------------------------------------------------------------------------
// NewContextTrackerService config defaults
// ---------------------------------------------------------------------------

func TestNewContextTrackerService_NilConfigAppliesDefaults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := NewContextTrackerService(nil, logger, nil)

	assert.Equal(t, 100, svc.maxContexts)
	assert.Equal(t, 128000, svc.defaultMaxTokens)
	assert.InDelta(t, 0.2, svc.defaultReserveRatio, 0.0001)
}

func TestNewContextTrackerService_CustomConfigApplied(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &ContextTrackerConfig{
		MaxActiveContexts:   50,
		DefaultMaxTokens:    200000,
		DefaultReserveRatio: 0.1,
	}

	svc := NewContextTrackerService(nil, logger, cfg)

	assert.Equal(t, 50, svc.maxContexts)
	assert.Equal(t, 200000, svc.defaultMaxTokens)
	assert.InDelta(t, 0.1, svc.defaultReserveRatio, 0.0001)
}

// ---------------------------------------------------------------------------
// Concurrency — GetActiveContextCount under parallel seeding
// ---------------------------------------------------------------------------

func TestContextTracker_ConcurrentGetActiveContextCount(t *testing.T) {
	svc := newTestContextTracker(100)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cc := newConvContext("conv-concurrent-"+string(rune('A'+n)), 10000, 2000)
			seedContext(svc, cc)
			_ = svc.GetActiveContextCount()
		}(i)
	}
	wg.Wait()

	assert.LessOrEqual(t, svc.GetActiveContextCount(), 20)
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkEstimateTokens(b *testing.B) {
	text := "the quick brown fox jumps over the lazy dog "
	for i := 0; i < b.N; i++ {
		_ = EstimateTokens(text)
	}
}

func BenchmarkEvictBlocks(b *testing.B) {
	svc := newTestContextTracker(10)
	cc := newConvContext("conv-bench", 10000, 2000)
	cc.TotalTokens = 9000

	// 30 evictable blocks
	for i := 0; i < 30; i++ {
		cc.Blocks = append(cc.Blocks, newBlock(
			"b"+string(rune('a'+i%26)), "user", 300, i%5+1, false,
		))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Restore blocks each iteration to keep benchmark meaningful.
		for _, blk := range cc.Blocks {
			blk.AccessCount = 1
		}
		_ = svc.evictBlocks(cc, 1500)
	}
}
