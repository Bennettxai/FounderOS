package subconscious

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BlockType enumerates the 8 memory block categories.
type BlockType string

const (
	BlockCoreDirectives  BlockType = "core_directives"
	BlockGuidance        BlockType = "guidance"
	BlockSessionPatterns BlockType = "session_patterns"
	BlockUserPreferences BlockType = "user_preferences"
	BlockToolGuidelines  BlockType = "tool_guidelines"
	BlockSelfImprovement BlockType = "self_improvement"
	BlockProjectContext  BlockType = "project_context"
	BlockPendingItems    BlockType = "pending_items"
)

// AllBlockTypes is the ordered list for injection (highest priority first).
var AllBlockTypes = []BlockType{
	BlockCoreDirectives,
	BlockGuidance,
	BlockSessionPatterns,
	BlockUserPreferences,
	BlockToolGuidelines,
	BlockSelfImprovement,
	BlockProjectContext,
	BlockPendingItems,
}

// Block is a single memory block stored in subconscious_blocks.
type Block struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	BlockType BlockType  `json:"block_type"`
	Content   string     `json:"content"`
	Weight    float64    `json:"weight"`
	Version   int        `json:"version"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// MaxBlockContentLength is the maximum characters per block.
const MaxBlockContentLength = 2000

// BlockStore is the interface for subconscious block persistence.
type BlockStore interface {
	// GetActiveBlocks returns all non-expired blocks for a user, ordered by weight DESC.
	GetActiveBlocks(ctx context.Context, userID string) ([]Block, error)
	// GetBlock returns a single block by user and type.
	GetBlock(ctx context.Context, userID string, blockType BlockType) (*Block, error)
	// UpsertBlock creates or updates a block. Uses optimistic concurrency via version.
	UpsertBlock(ctx context.Context, block Block) error
	// DeleteExpired removes blocks past their expiry time.
	DeleteExpired(ctx context.Context) (int64, error)
}

// PostgresBlockStore implements BlockStore backed by the subconscious_blocks table.
type PostgresBlockStore struct {
	pool *pgxpool.Pool
}

// NewPostgresBlockStore creates a new PostgresBlockStore.
func NewPostgresBlockStore(pool *pgxpool.Pool) *PostgresBlockStore {
	return &PostgresBlockStore{pool: pool}
}

func (s *PostgresBlockStore) GetActiveBlocks(ctx context.Context, userID string) ([]Block, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, block_type, content, weight, version, expires_at, created_at, updated_at
		FROM subconscious_blocks
		WHERE user_id = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY weight DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("subconscious blocks: get active: %w", err)
	}
	defer rows.Close()

	var blocks []Block
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.ID, &b.UserID, &b.BlockType, &b.Content, &b.Weight,
			&b.Version, &b.ExpiresAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("subconscious blocks: scan: %w", err)
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

func (s *PostgresBlockStore) GetBlock(ctx context.Context, userID string, blockType BlockType) (*Block, error) {
	var b Block
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, block_type, content, weight, version, expires_at, created_at, updated_at
		FROM subconscious_blocks
		WHERE user_id = $1 AND block_type = $2
	`, userID, string(blockType)).Scan(
		&b.ID, &b.UserID, &b.BlockType, &b.Content, &b.Weight,
		&b.Version, &b.ExpiresAt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("subconscious blocks: get: %w", err)
	}
	return &b, nil
}

func (s *PostgresBlockStore) UpsertBlock(ctx context.Context, block Block) error {
	// Truncate content to max length.
	content := block.Content
	if len(content) > MaxBlockContentLength {
		content = content[:MaxBlockContentLength]
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO subconscious_blocks (user_id, block_type, content, weight, version, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, block_type)
		DO UPDATE SET
			content = EXCLUDED.content,
			weight = EXCLUDED.weight,
			version = subconscious_blocks.version + 1,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, block.UserID, string(block.BlockType), content, block.Weight, block.Version, block.ExpiresAt)
	if err != nil {
		return fmt.Errorf("subconscious blocks: upsert: %w", err)
	}
	return nil
}

func (s *PostgresBlockStore) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM subconscious_blocks WHERE expires_at IS NOT NULL AND expires_at <= NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("subconscious blocks: delete expired: %w", err)
	}
	return tag.RowsAffected(), nil
}
