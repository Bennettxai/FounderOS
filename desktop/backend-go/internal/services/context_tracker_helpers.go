package services

import (
	"context"
	"fmt"
	"time"
)

// GetActiveContextCount returns the number of active contexts in memory
func (s *ContextTrackerService) GetActiveContextCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.activeContexts)
}

// EstimateTokens estimates the token count for a string (approximation)
func EstimateTokens(text string) int {
	// Rough estimation: ~4 characters per token for English
	return len(text) / 4
}

// CompactContext optimizes the context by merging similar blocks
func (s *ContextTrackerService) CompactContext(ctx context.Context, conversationID string) (*EvictionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation context not found: %s", conversationID)
	}

	result := &EvictionResult{
		EvictedBlocks: make([]string, 0),
		TokensFreed:   0,
		BlocksRemoved: 0,
	}

	seenContent := make(map[string]bool)
	newBlocks := make([]*ContextBlock, 0)

	for _, b := range cc.Blocks {
		if b.Pinned {
			newBlocks = append(newBlocks, b)
			continue
		}

		prefixLen := len(b.Content)
		if prefixLen > 100 {
			prefixLen = 100
		}
		contentHash := fmt.Sprintf("%s:%s", b.Type, b.Content[:prefixLen])
		if seenContent[contentHash] {
			result.EvictedBlocks = append(result.EvictedBlocks, b.ID)
			result.TokensFreed += b.TokenCount
			result.BlocksRemoved++
		} else {
			seenContent[contentHash] = true
			newBlocks = append(newBlocks, b)
		}
	}

	cc.Blocks = newBlocks
	cc.TotalTokens -= result.TokensFreed
	result.NewTotalTokens = cc.TotalTokens
	cc.UpdatedAt = time.Now()

	if err := s.saveContextToDB(ctx, cc); err != nil {
		return result, err
	}

	return result, nil
}
