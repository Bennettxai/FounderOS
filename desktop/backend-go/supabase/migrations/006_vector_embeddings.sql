-- Migration 006: Vector Embeddings for Knowledge Base RAG
-- Requires pgvector extension to be installed in PostgreSQL

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Document embeddings table (block-level granularity)
CREATE TABLE IF NOT EXISTS context_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_id UUID NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    block_id TEXT NOT NULL,                    -- Editor block ID
    block_type TEXT NOT NULL,                  -- paragraph, heading, list, etc.
    content TEXT NOT NULL,                     -- Raw text content
    embedding vector(768),                     -- nomic-embed-text dimension
    metadata JSONB DEFAULT '{}',               -- Additional block metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(context_id, block_id)
);

-- Create HNSW index for fast cosine similarity search
-- HNSW provides ~10x faster queries than IVFFlat with similar recall
CREATE INDEX IF NOT EXISTS idx_context_embeddings_vector
ON context_embeddings USING hnsw (embedding vector_cosine_ops);

-- Index for context lookups
CREATE INDEX IF NOT EXISTS idx_context_embeddings_context_id
ON context_embeddings(context_id);

-- Index for block type filtering
CREATE INDEX IF NOT EXISTS idx_context_embeddings_block_type
ON context_embeddings(block_type);

-- Add embedding tracking columns to contexts table
ALTER TABLE contexts ADD COLUMN IF NOT EXISTS embedding_status TEXT DEFAULT 'pending';
ALTER TABLE contexts ADD COLUMN IF NOT EXISTS last_embedded_at TIMESTAMPTZ;
ALTER TABLE contexts ADD COLUMN IF NOT EXISTS embedding_count INTEGER DEFAULT 0;

-- Create index on embedding status for batch operations
CREATE INDEX IF NOT EXISTS idx_contexts_embedding_status
ON contexts(embedding_status) WHERE embedding_status != 'indexed';
