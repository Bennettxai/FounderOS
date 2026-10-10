-- Migration 025: Image Embeddings for Multi-modal Search
-- Stores images with their CLIP embeddings for visual search

-- Create image_embeddings table
CREATE TABLE IF NOT EXISTS image_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Image data
    image_url TEXT,
    image_data BYTEA,  -- Store actual image bytes
    image_hash VARCHAR(64),  -- SHA-256 hash for deduplication

    -- Embedding
    embedding vector(512),  -- CLIP embeddings are typically 512 dimensions

    -- Metadata
    caption TEXT,
    description TEXT,
    metadata JSONB DEFAULT '{}',

    -- Associations
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,

    -- File info
    filename VARCHAR(255),
    mime_type VARCHAR(100),
    file_size BIGINT,

    -- Dimensions
    width INTEGER,
    height INTEGER,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Ensure user owns the image
    CONSTRAINT image_embeddings_user_check CHECK (user_id IS NOT NULL)
);

-- Indexes for efficient search
CREATE INDEX idx_image_embeddings_user_id ON image_embeddings(user_id);
CREATE INDEX idx_image_embeddings_context_id ON image_embeddings(context_id);
CREATE INDEX idx_image_embeddings_project_id ON image_embeddings(project_id);
CREATE INDEX idx_image_embeddings_conversation_id ON image_embeddings(conversation_id);
CREATE INDEX idx_image_embeddings_created_at ON image_embeddings(created_at DESC);
CREATE INDEX idx_image_embeddings_hash ON image_embeddings(image_hash);

-- Vector similarity search index (using cosine distance)
CREATE INDEX idx_image_embeddings_embedding ON image_embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- GIN index for metadata search
CREATE INDEX idx_image_embeddings_metadata ON image_embeddings USING gin(metadata);

-- Image tags for categorization
CREATE TABLE IF NOT EXISTS image_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES image_embeddings(id) ON DELETE CASCADE,
    tag VARCHAR(100) NOT NULL,
    confidence DECIMAL(3,2),  -- Auto-generated tag confidence (0.00-1.00)
    source VARCHAR(50) DEFAULT 'user',  -- 'user', 'auto', 'ai'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(image_id, tag)
);

CREATE INDEX idx_image_tags_image_id ON image_tags(image_id);
CREATE INDEX idx_image_tags_tag ON image_tags(tag);

-- Image collections for grouping related images
CREATE TABLE IF NOT EXISTS image_collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    thumbnail_image_id UUID REFERENCES image_embeddings(id) ON DELETE SET NULL,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_image_collections_user_id ON image_collections(user_id);

-- Many-to-many relationship for images in collections
CREATE TABLE IF NOT EXISTS image_collection_items (
    collection_id UUID NOT NULL REFERENCES image_collections(id) ON DELETE CASCADE,
    image_id UUID NOT NULL REFERENCES image_embeddings(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (collection_id, image_id)
);

CREATE INDEX idx_image_collection_items_collection ON image_collection_items(collection_id);
CREATE INDEX idx_image_collection_items_image ON image_collection_items(image_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_image_embeddings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER trigger_update_image_embeddings_updated_at
    BEFORE UPDATE ON image_embeddings
    FOR EACH ROW
    EXECUTE FUNCTION update_image_embeddings_updated_at();

CREATE TRIGGER trigger_update_image_collections_updated_at
    BEFORE UPDATE ON image_collections
    FOR EACH ROW
    EXECUTE FUNCTION update_image_embeddings_updated_at();

-- Comments for documentation
COMMENT ON TABLE image_embeddings IS 'Stores images with CLIP embeddings for multi-modal search';
COMMENT ON COLUMN image_embeddings.embedding IS 'CLIP embedding vector (512 dimensions)';
COMMENT ON COLUMN image_embeddings.image_hash IS 'SHA-256 hash for deduplication';
COMMENT ON TABLE image_tags IS 'Tags associated with images (user-defined or AI-generated)';
COMMENT ON TABLE image_collections IS 'Collections for organizing related images';
