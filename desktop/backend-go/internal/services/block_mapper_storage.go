package services

import (
	"context"
	"encoding/json"
)

// SaveBlockDocument persists a block document to the database
func (s *BlockMapperService) SaveBlockDocument(ctx context.Context, doc *BlockDocument) error {
	blocksJSON, err := json.Marshal(doc.Blocks)
	if err != nil {
		return err
	}
	outlineJSON, err := json.Marshal(doc.Outline)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO block_documents (id, source_id, title, blocks, outline, metadata, hash, total_blocks, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO UPDATE SET
		    blocks = EXCLUDED.blocks,
		    outline = EXCLUDED.outline,
		    metadata = EXCLUDED.metadata,
		    hash = EXCLUDED.hash,
		    total_blocks = EXCLUDED.total_blocks`,
		doc.ID, doc.SourceID, doc.Title, blocksJSON, outlineJSON, metadataJSON,
		doc.Hash, doc.TotalBlocks, doc.CreatedAt)

	return err
}

// GetBlockDocument retrieves a block document by ID
func (s *BlockMapperService) GetBlockDocument(ctx context.Context, id string) (*BlockDocument, error) {
	var doc BlockDocument
	var blocksJSON, outlineJSON, metadataJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, source_id, title, blocks, outline, metadata, hash, total_blocks, created_at
		 FROM block_documents WHERE id = $1`,
		id).Scan(&doc.ID, &doc.SourceID, &doc.Title, &blocksJSON, &outlineJSON, &metadataJSON,
		&doc.Hash, &doc.TotalBlocks, &doc.CreatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(blocksJSON, &doc.Blocks)
	json.Unmarshal(outlineJSON, &doc.Outline)
	json.Unmarshal(metadataJSON, &doc.Metadata)

	return &doc, nil
}
