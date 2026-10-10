package services

import (
	"context"
	"fmt"
	"time"
)

// GetTokenUsageStats returns statistics about token usage
func (s *ContextTrackerService) GetTokenUsageStats(ctx context.Context, conversationID string) (*TokenUsageStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation context not found: %s", conversationID)
	}

	availableTokens := cc.MaxTokens - cc.ReserveTokens - cc.TotalTokens
	usagePercentage := float64(cc.TotalTokens) / float64(cc.MaxTokens-cc.ReserveTokens) * 100

	pinnedTokens := 0
	evictableTokens := 0
	var oldestBlock *ContextBlock

	for _, b := range cc.Blocks {
		if b.Pinned {
			pinnedTokens += b.TokenCount
		} else {
			evictableTokens += b.TokenCount
		}
		if oldestBlock == nil || b.CreatedAt.Before(oldestBlock.CreatedAt) {
			oldestBlock = b
		}
	}

	oldestAge := "N/A"
	if oldestBlock != nil {
		oldestAge = time.Since(oldestBlock.CreatedAt).Round(time.Second).String()
	}

	estimatedTurns := 0
	if len(cc.Blocks) > 0 {
		avgTokensPerTurn := cc.TotalTokens / len(cc.Blocks)
		if avgTokensPerTurn > 0 {
			estimatedTurns = availableTokens / avgTokensPerTurn
		}
	}

	return &TokenUsageStats{
		ConversationID:  conversationID,
		TotalTokens:     cc.TotalTokens,
		MaxTokens:       cc.MaxTokens,
		UsagePercentage: usagePercentage,
		AvailableTokens: availableTokens,
		ReserveTokens:   cc.ReserveTokens,
		BlockCount:      len(cc.Blocks),
		PinnedTokens:    pinnedTokens,
		EvictableTokens: evictableTokens,
		OldestBlockAge:  oldestAge,
		EstimatedTurns:  estimatedTurns,
	}, nil
}

// GetBlocks returns all blocks in a context
func (s *ContextTrackerService) GetBlocks(ctx context.Context, conversationID string) ([]*ContextBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation context not found: %s", conversationID)
	}

	blocks := make([]*ContextBlock, len(cc.Blocks))
	copy(blocks, cc.Blocks)

	return blocks, nil
}

// GetBlocksByType returns blocks of a specific type
func (s *ContextTrackerService) GetBlocksByType(ctx context.Context, conversationID, blockType string) ([]*ContextBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation context not found: %s", conversationID)
	}

	blocks := make([]*ContextBlock, 0)
	for _, b := range cc.Blocks {
		if b.Type == blockType {
			blocks = append(blocks, b)
		}
	}

	return blocks, nil
}
