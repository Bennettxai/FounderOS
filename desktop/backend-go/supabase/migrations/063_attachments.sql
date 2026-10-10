-- Migration 038: Universal Attachments System
-- File attachments that can be linked to any entity in BusinessOS
-- Supports local storage, cloud storage, and external links

-- ============================================================================
-- ATTACHMENTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What entity is this attached to
    entity_type VARCHAR(100) NOT NULL,  -- 'task', 'project', 'client', 'custom_record', etc.
    entity_id UUID NOT NULL,

    -- File metadata
    file_name VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,               -- Size in bytes
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),

    -- Storage location
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'local',  -- 'local', 'gcs', 's3', 'external'
    storage_path TEXT NOT NULL,                             -- Path or URL
    storage_bucket VARCHAR(255),                            -- For cloud storage

    -- Display
    thumbnail_url TEXT,
    preview_url TEXT,

    -- For images
    width INT,
    height INT,

    -- For documents
    page_count INT,

    -- For audio/video
    duration_seconds INT,

    -- Processing status
    processing_status VARCHAR(50) DEFAULT 'ready',  -- 'pending', 'processing', 'ready', 'failed'
    processing_error TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    -- Examples: extracted text, EXIF data, AI-generated description

    -- Who uploaded
    uploaded_by VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Find attachments for an entity
CREATE INDEX IF NOT EXISTS idx_attachments_entity
    ON attachments(entity_type, entity_id);

-- User's attachments
CREATE INDEX IF NOT EXISTS idx_attachments_user
    ON attachments(user_id, created_at DESC);

-- By file type
CREATE INDEX IF NOT EXISTS idx_attachments_mime
    ON attachments(user_id, mime_type);

-- By processing status (for background workers)
CREATE INDEX IF NOT EXISTS idx_attachments_processing
    ON attachments(processing_status)
    WHERE processing_status IN ('pending', 'processing');

-- ============================================================================
-- ATTACHMENT VERSIONS (For version history)
-- ============================================================================

CREATE TABLE IF NOT EXISTS attachment_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attachment_id UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,

    -- Version info
    version_number INT NOT NULL,
    version_label VARCHAR(100),

    -- File metadata (same as parent)
    file_size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),

    -- Who created this version
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(attachment_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_attachment_versions_attachment
    ON attachment_versions(attachment_id);

-- ============================================================================
-- ATTACHMENT FOLDERS (Optional organization)
-- ============================================================================

CREATE TABLE IF NOT EXISTS attachment_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Hierarchy
    parent_id UUID REFERENCES attachment_folders(id) ON DELETE CASCADE,

    -- Folder metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(50),

    -- Linked to entity (optional)
    entity_type VARCHAR(100),
    entity_id UUID,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attachment_folders_user
    ON attachment_folders(user_id);

CREATE INDEX IF NOT EXISTS idx_attachment_folders_parent
    ON attachment_folders(parent_id)
    WHERE parent_id IS NOT NULL;

-- Add folder reference to attachments
ALTER TABLE attachments
    ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES attachment_folders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_attachments_folder
    ON attachments(folder_id)
    WHERE folder_id IS NOT NULL;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE OR REPLACE FUNCTION update_attachment_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS attachments_updated_at ON attachments;
CREATE TRIGGER attachments_updated_at
    BEFORE UPDATE ON attachments
    FOR EACH ROW
    EXECUTE FUNCTION update_attachment_updated_at();

DROP TRIGGER IF EXISTS attachment_folders_updated_at ON attachment_folders;
CREATE TRIGGER attachment_folders_updated_at
    BEFORE UPDATE ON attachment_folders
    FOR EACH ROW
    EXECUTE FUNCTION update_attachment_updated_at();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Get total storage used by a user
CREATE OR REPLACE FUNCTION get_user_storage_usage(p_user_id VARCHAR(255))
RETURNS BIGINT AS $$
BEGIN
    RETURN COALESCE((
        SELECT SUM(file_size)
        FROM attachments
        WHERE user_id = p_user_id
    ), 0);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE attachments IS 'Universal file attachments for any entity';
COMMENT ON TABLE attachment_versions IS 'Version history for attachments';
COMMENT ON TABLE attachment_folders IS 'Folders to organize attachments';
COMMENT ON COLUMN attachments.storage_provider IS 'Storage backend: local, gcs, s3, external';
COMMENT ON COLUMN attachments.processing_status IS 'For async processing: thumbnail generation, text extraction, etc.';
