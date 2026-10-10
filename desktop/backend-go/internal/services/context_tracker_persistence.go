package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// evictLRUIfNeeded removes least recently used contexts if cache is full
func (s *ContextTrackerService) evictLRUIfNeeded() {
	if len(s.activeContexts) <= s.maxContexts {
		return
	}

	var lruKey string
	var lruTime time.Time

	for key, cc := range s.activeContexts {
		if lruKey == "" || cc.LastAccessedAt.Before(lruTime) {
			lruKey = key
			lruTime = cc.LastAccessedAt
		}
	}

	if lruKey != "" {
		s.logger.Info("evicting LRU context from cache", "conversation_id", lruKey)
		delete(s.activeContexts, lruKey)
	}
}

// loadContextFromDB loads a context from the database
func (s *ContextTrackerService) loadContextFromDB(ctx context.Context, conversationID string) (*ConversationContext, error) {
	if s.pool == nil {
		return nil, nil
	}
	var cc ConversationContext
	var blocksJSON, metadataJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT conversation_id, user_id, total_tokens, max_tokens, reserve_tokens,
		        blocks, metadata, created_at, last_accessed_at, updated_at
		 FROM context_windows
		 WHERE conversation_id = $1`,
		conversationID).Scan(
		&cc.ConversationID, &cc.UserID, &cc.TotalTokens, &cc.MaxTokens, &cc.ReserveTokens,
		&blocksJSON, &metadataJSON, &cc.CreatedAt, &cc.LastAccessedAt, &cc.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(blocksJSON, &cc.Blocks); err != nil {
		cc.Blocks = make([]*ContextBlock, 0)
	}
	if err := json.Unmarshal(metadataJSON, &cc.Metadata); err != nil {
		cc.Metadata = make(map[string]interface{})
	}

	return &cc, nil
}

// saveContextToDB persists a context to the database
func (s *ContextTrackerService) saveContextToDB(ctx context.Context, cc *ConversationContext) error {
	if s.pool == nil {
		return nil
	}
	blocksJSON, err := json.Marshal(cc.Blocks)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(cc.Metadata)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO context_windows (conversation_id, user_id, total_tokens, max_tokens, reserve_tokens,
		                              blocks, metadata, created_at, last_accessed_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (conversation_id) DO UPDATE SET
		    total_tokens = EXCLUDED.total_tokens,
		    max_tokens = EXCLUDED.max_tokens,
		    reserve_tokens = EXCLUDED.reserve_tokens,
		    blocks = EXCLUDED.blocks,
		    metadata = EXCLUDED.metadata,
		    last_accessed_at = EXCLUDED.last_accessed_at,
		    updated_at = EXCLUDED.updated_at`,
		cc.ConversationID, cc.UserID, cc.TotalTokens, cc.MaxTokens, cc.ReserveTokens,
		blocksJSON, metadataJSON, cc.CreatedAt, cc.LastAccessedAt, cc.UpdatedAt)

	return err
}
