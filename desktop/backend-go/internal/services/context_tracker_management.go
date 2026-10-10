package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// GetOrCreateContext gets or creates a context for a conversation
func (s *ContextTrackerService) GetOrCreateContext(ctx context.Context, conversationID, userID string) (*ConversationContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cc, exists := s.activeContexts[conversationID]; exists {
		cc.LastAccessedAt = time.Now()
		return cc, nil
	}

	cc, err := s.loadContextFromDB(ctx, conversationID)
	if err == nil && cc != nil {
		cc.LastAccessedAt = time.Now()
		s.activeContexts[conversationID] = cc
		s.evictLRUIfNeeded()
		return cc, nil
	}

	reserveTokens := int(float64(s.defaultMaxTokens) * s.defaultReserveRatio)
	cc = &ConversationContext{
		ConversationID: conversationID,
		UserID:         userID,
		TotalTokens:    0,
		MaxTokens:      s.defaultMaxTokens,
		ReserveTokens:  reserveTokens,
		Blocks:         make([]*ContextBlock, 0),
		Metadata:       make(map[string]interface{}),
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		UpdatedAt:      time.Now(),
	}

	s.activeContexts[conversationID] = cc
	s.evictLRUIfNeeded()

	if err := s.saveContextToDB(ctx, cc); err != nil {
		s.logger.Warn("failed to save context to DB", "error", err)
	}

	return cc, nil
}

// AddBlock adds a new block to the context
func (s *ContextTrackerService) AddBlock(ctx context.Context, conversationID string, block *ContextBlock) (*EvictionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation context not found: %s", conversationID)
	}

	if block.ID == "" {
		block.ID = uuid.New().String()
	}
	block.CreatedAt = time.Now()
	block.AccessCount = 1

	availableTokens := cc.MaxTokens - cc.ReserveTokens - cc.TotalTokens
	var evictionResult *EvictionResult

	if block.TokenCount > availableTokens {
		tokensNeeded := block.TokenCount - availableTokens + 1000
		evictionResult = s.evictBlocks(cc, tokensNeeded)
	}

	cc.Blocks = append(cc.Blocks, block)
	cc.TotalTokens += block.TokenCount
	cc.UpdatedAt = time.Now()
	cc.LastAccessedAt = time.Now()

	if err := s.saveContextToDB(ctx, cc); err != nil {
		s.logger.Warn("failed to save context after adding block", "error", err)
	}

	return evictionResult, nil
}

// evictBlocks removes lowest priority blocks until tokensNeeded are freed
func (s *ContextTrackerService) evictBlocks(cc *ConversationContext, tokensNeeded int) *EvictionResult {
	result := &EvictionResult{
		EvictedBlocks: make([]string, 0),
		TokensFreed:   0,
		BlocksRemoved: 0,
	}

	type evictionCandidate struct {
		index    int
		block    *ContextBlock
		priority int
	}

	candidates := make([]evictionCandidate, 0)
	for i, b := range cc.Blocks {
		if !b.Pinned {
			priorityScore := b.Priority*1000 + b.AccessCount
			candidates = append(candidates, evictionCandidate{
				index:    i,
				block:    b,
				priority: priorityScore,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority < candidates[j].priority
	})

	evictedIndices := make(map[int]bool)
	for _, candidate := range candidates {
		if result.TokensFreed >= tokensNeeded {
			break
		}

		result.EvictedBlocks = append(result.EvictedBlocks, candidate.block.ID)
		result.TokensFreed += candidate.block.TokenCount
		result.BlocksRemoved++
		evictedIndices[candidate.index] = true
	}

	newBlocks := make([]*ContextBlock, 0, len(cc.Blocks)-result.BlocksRemoved)
	for i, b := range cc.Blocks {
		if !evictedIndices[i] {
			newBlocks = append(newBlocks, b)
		}
	}
	cc.Blocks = newBlocks
	cc.TotalTokens -= result.TokensFreed
	result.NewTotalTokens = cc.TotalTokens

	s.logger.Info("evicted blocks from context",
		"conversation_id", cc.ConversationID,
		"blocks_removed", result.BlocksRemoved,
		"tokens_freed", result.TokensFreed)

	return result
}

// PinBlock marks a block as pinned (never evict)
func (s *ContextTrackerService) PinBlock(ctx context.Context, conversationID, blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	for _, b := range cc.Blocks {
		if b.ID == blockID {
			b.Pinned = true
			cc.UpdatedAt = time.Now()
			return s.saveContextToDB(ctx, cc)
		}
	}

	return fmt.Errorf("block not found: %s", blockID)
}

// UnpinBlock removes the pinned status from a block
func (s *ContextTrackerService) UnpinBlock(ctx context.Context, conversationID, blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	for _, b := range cc.Blocks {
		if b.ID == blockID {
			b.Pinned = false
			cc.UpdatedAt = time.Now()
			return s.saveContextToDB(ctx, cc)
		}
	}

	return fmt.Errorf("block not found: %s", blockID)
}

// SetBlockPriority updates the priority of a block
func (s *ContextTrackerService) SetBlockPriority(ctx context.Context, conversationID, blockID string, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	for _, b := range cc.Blocks {
		if b.ID == blockID {
			b.Priority = priority
			cc.UpdatedAt = time.Now()
			return s.saveContextToDB(ctx, cc)
		}
	}

	return fmt.Errorf("block not found: %s", blockID)
}

// RemoveBlock removes a specific block from the context
func (s *ContextTrackerService) RemoveBlock(ctx context.Context, conversationID, blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	newBlocks := make([]*ContextBlock, 0, len(cc.Blocks))
	var removedTokens int
	found := false

	for _, b := range cc.Blocks {
		if b.ID == blockID {
			removedTokens = b.TokenCount
			found = true
		} else {
			newBlocks = append(newBlocks, b)
		}
	}

	if !found {
		return fmt.Errorf("block not found: %s", blockID)
	}

	cc.Blocks = newBlocks
	cc.TotalTokens -= removedTokens
	cc.UpdatedAt = time.Now()

	return s.saveContextToDB(ctx, cc)
}

// SetMaxTokens updates the max tokens for a conversation
func (s *ContextTrackerService) SetMaxTokens(ctx context.Context, conversationID string, maxTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	cc.MaxTokens = maxTokens
	cc.ReserveTokens = int(float64(maxTokens) * s.defaultReserveRatio)
	cc.UpdatedAt = time.Now()

	return s.saveContextToDB(ctx, cc)
}

// ClearContext removes all non-pinned blocks from a context
func (s *ContextTrackerService) ClearContext(ctx context.Context, conversationID string) (*EvictionResult, error) {
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

	newBlocks := make([]*ContextBlock, 0)
	for _, b := range cc.Blocks {
		if b.Pinned {
			newBlocks = append(newBlocks, b)
		} else {
			result.EvictedBlocks = append(result.EvictedBlocks, b.ID)
			result.TokensFreed += b.TokenCount
			result.BlocksRemoved++
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

// DeleteContext removes a context entirely
func (s *ContextTrackerService) DeleteContext(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeContexts, conversationID)

	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM context_windows WHERE conversation_id = $1`,
		conversationID)

	return err
}

// IncrementBlockAccess increments the access count for a block
func (s *ContextTrackerService) IncrementBlockAccess(ctx context.Context, conversationID, blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, exists := s.activeContexts[conversationID]
	if !exists {
		return fmt.Errorf("conversation context not found: %s", conversationID)
	}

	for _, b := range cc.Blocks {
		if b.ID == blockID {
			b.AccessCount++
			return nil
		}
	}

	return fmt.Errorf("block not found: %s", blockID)
}
