package subconscious

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockBlockStore is an in-memory BlockStore for testing. Thread-safe.
type MockBlockStore struct {
	mu     sync.RWMutex
	blocks map[string]Block // key: "userID:blockType"
}

func NewMockBlockStore() *MockBlockStore {
	return &MockBlockStore{blocks: make(map[string]Block)}
}

func (m *MockBlockStore) key(userID string, blockType BlockType) string {
	return fmt.Sprintf("%s:%s", userID, blockType)
}

func (m *MockBlockStore) GetActiveBlocks(_ context.Context, userID string) ([]Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Block
	now := time.Now().UTC()
	for _, b := range m.blocks {
		if b.UserID == userID {
			if b.ExpiresAt == nil || b.ExpiresAt.After(now) {
				result = append(result, b)
			}
		}
	}
	return result, nil
}

func (m *MockBlockStore) GetBlock(_ context.Context, userID string, blockType BlockType) (*Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.blocks[m.key(userID, blockType)]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &b, nil
}

func (m *MockBlockStore) UpsertBlock(_ context.Context, block Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	content := block.Content
	if len(content) > MaxBlockContentLength {
		content = content[:MaxBlockContentLength]
	}
	existing, exists := m.blocks[m.key(block.UserID, block.BlockType)]
	version := 1
	if exists {
		version = existing.Version + 1
	}
	m.blocks[m.key(block.UserID, block.BlockType)] = Block{
		ID:        fmt.Sprintf("mock-%d", version),
		UserID:    block.UserID,
		BlockType: block.BlockType,
		Content:   content,
		Weight:    block.Weight,
		Version:   version,
		ExpiresAt: block.ExpiresAt,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	return nil
}

func (m *MockBlockStore) DeleteExpired(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var deleted int64
	now := time.Now().UTC()
	for key, b := range m.blocks {
		if b.ExpiresAt != nil && b.ExpiresAt.Before(now) {
			delete(m.blocks, key)
			deleted++
		}
	}
	return deleted, nil
}

func TestMockBlockStoreUpsert(t *testing.T) {
	store := NewMockBlockStore()
	ctx := context.Background()

	err := store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockUserPreferences,
		Content:   "I prefer dark mode",
		Weight:    0.7,
	})
	if err != nil {
		t.Fatal(err)
	}

	block, err := store.GetBlock(ctx, "user1", BlockUserPreferences)
	if err != nil {
		t.Fatal(err)
	}
	if block.Content != "I prefer dark mode" {
		t.Errorf("expected 'I prefer dark mode', got %q", block.Content)
	}
	if block.Version != 1 {
		t.Errorf("expected version 1, got %d", block.Version)
	}

	// Update
	err = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockUserPreferences,
		Content:   "I prefer dark mode\nI always want TypeScript",
		Weight:    0.7,
	})
	if err != nil {
		t.Fatal(err)
	}

	block, err = store.GetBlock(ctx, "user1", BlockUserPreferences)
	if err != nil {
		t.Fatal(err)
	}
	if block.Version != 2 {
		t.Errorf("expected version 2, got %d", block.Version)
	}
}

func TestMockBlockStoreActiveBlocks(t *testing.T) {
	store := NewMockBlockStore()
	ctx := context.Background()

	// Add a permanent block
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockUserPreferences,
		Content:   "pref",
		Weight:    0.7,
	})

	// Add an expired block
	expired := time.Now().UTC().Add(-1 * time.Hour)
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockSessionPatterns,
		Content:   "old pattern",
		Weight:    0.5,
		ExpiresAt: &expired,
	})

	blocks, err := store.GetActiveBlocks(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}

	// Only the permanent block should be active
	if len(blocks) != 1 {
		t.Fatalf("expected 1 active block, got %d", len(blocks))
	}
	if blocks[0].BlockType != BlockUserPreferences {
		t.Errorf("expected user_preferences block, got %s", blocks[0].BlockType)
	}
}

func TestBlockContentTruncation(t *testing.T) {
	store := NewMockBlockStore()
	ctx := context.Background()

	// Create content longer than MaxBlockContentLength
	longContent := ""
	for i := 0; i < MaxBlockContentLength+500; i++ {
		longContent += "x"
	}

	err := store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockGuidance,
		Content:   longContent,
		Weight:    0.5,
	})
	if err != nil {
		t.Fatal(err)
	}

	block, err := store.GetBlock(ctx, "user1", BlockGuidance)
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Content) > MaxBlockContentLength {
		t.Errorf("content should be truncated to %d, got %d",
			MaxBlockContentLength, len(block.Content))
	}
}

func TestBlockTypes(t *testing.T) {
	// Verify all 8 block types are defined
	if len(AllBlockTypes) != 8 {
		t.Fatalf("expected 8 block types, got %d", len(AllBlockTypes))
	}
}
