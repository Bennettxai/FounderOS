-- ================================================
-- Migration 019: Document Upload & Management System
-- Description: Uploaded documents and chunking for retrieval
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- UPLOADED DOCUMENTS TABLE
-- Stores uploaded files (PDFs, markdown, docx, etc.)
-- ================================================
CREATE TABLE IF NOT EXISTS uploaded_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Document identity
    filename VARCHAR(500) NOT NULL,
    original_filename VARCHAR(500) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,

    -- File info
    file_type VARCHAR(50) NOT NULL,          -- 'pdf', 'markdown', 'docx', 'txt', 'image'
    mime_type VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,

    -- Storage
    storage_path VARCHAR(1000) NOT NULL,     -- Path in storage (GCS, S3, local)
    storage_provider VARCHAR(50) DEFAULT 'local',  -- 'local', 'gcs', 's3'

    -- Extracted content
    extracted_text TEXT,                      -- Full text extraction from PDF/DOCX
    page_count INTEGER,                       -- For PDFs
    word_count INTEGER,

    -- Context links
    context_profile_id UUID,                  -- Will reference context_profiles after migration 017
    project_id UUID,                          -- Will reference projects
    node_id UUID,                             -- Will reference nodes

    -- Categorization
    document_type VARCHAR(100),               -- 'sop', 'framework', 'template', 'reference', 'report'
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',

    -- Semantic search
    embedding vector(1536),

    -- Processing status
    processing_status VARCHAR(50) DEFAULT 'pending',  -- 'pending', 'processing', 'completed', 'failed'
    processing_error TEXT,
    processed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for uploaded_documents
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_user ON uploaded_documents(user_id);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_profile ON uploaded_documents(context_profile_id) WHERE context_profile_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_project ON uploaded_documents(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_node ON uploaded_documents(node_id) WHERE node_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_type ON uploaded_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_file_type ON uploaded_documents(file_type);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_status ON uploaded_documents(processing_status);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_created ON uploaded_documents(created_at DESC);
-- Note: embedding column and index are created in migration 037_embedding_dimensions_768.sql
-- CREATE INDEX IF NOT EXISTS idx_uploaded_docs_embedding ON uploaded_documents USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- DOCUMENT CHUNKS TABLE
-- Split documents for better retrieval (RAG)
-- ================================================
CREATE TABLE IF NOT EXISTS document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,

    -- Chunk info
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER,

    -- Position in original document
    page_number INTEGER,                      -- For PDFs
    start_char INTEGER,                       -- Character offset start
    end_char INTEGER,                         -- Character offset end
    section_title VARCHAR(255),               -- If extractable

    -- Chunk metadata
    chunk_type VARCHAR(50) DEFAULT 'text',    -- 'text', 'code', 'table', 'heading'

    -- Embedding for semantic search
    embedding vector(1536),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for document_chunks
CREATE INDEX IF NOT EXISTS idx_doc_chunks_document ON document_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_index ON document_chunks(document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_page ON document_chunks(document_id, page_number) WHERE page_number IS NOT NULL;
-- Note: embedding column and index are created in migration 037_embedding_dimensions_768.sql
-- CREATE INDEX IF NOT EXISTS idx_doc_chunks_embedding ON document_chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- DOCUMENT REFERENCES TABLE
-- Track references between documents and other entities
-- ================================================
CREATE TABLE IF NOT EXISTS document_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,

    -- What this document references or is referenced by
    entity_type VARCHAR(50) NOT NULL,         -- 'memory', 'artifact', 'task', 'conversation', 'document'
    entity_id UUID NOT NULL,

    -- Reference type
    reference_type VARCHAR(50) DEFAULT 'related',  -- 'source', 'related', 'derived_from', 'cites'

    -- Metadata
    context TEXT,                              -- Why this reference exists
    relevance_score DECIMAL(3,2) DEFAULT 0.5,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for document_references
CREATE INDEX IF NOT EXISTS idx_doc_refs_document ON document_references(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_refs_entity ON document_references(entity_type, entity_id);

-- ================================================
-- TRIGGERS
-- ================================================
CREATE OR REPLACE FUNCTION update_uploaded_documents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_uploaded_docs_updated_at ON uploaded_documents;
CREATE TRIGGER trigger_uploaded_docs_updated_at
    BEFORE UPDATE ON uploaded_documents
    FOR EACH ROW
    EXECUTE FUNCTION update_uploaded_documents_updated_at();

-- Update document stats when chunks are added/removed
CREATE OR REPLACE FUNCTION update_document_chunk_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Could update word_count or other stats here
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_doc_chunk_stats ON document_chunks;
CREATE TRIGGER trigger_doc_chunk_stats
    AFTER INSERT OR DELETE ON document_chunks
    FOR EACH ROW
    EXECUTE FUNCTION update_document_chunk_stats();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE uploaded_documents IS 'User-uploaded documents (PDFs, markdown, docx, etc.)';
COMMENT ON TABLE document_chunks IS 'Document chunks for RAG retrieval';
COMMENT ON TABLE document_references IS 'References between documents and other entities';

COMMENT ON COLUMN uploaded_documents.file_type IS 'Type: pdf, markdown, docx, txt, image';
COMMENT ON COLUMN uploaded_documents.document_type IS 'Category: sop, framework, template, reference, report';
COMMENT ON COLUMN uploaded_documents.processing_status IS 'Status: pending, processing, completed, failed';
COMMENT ON COLUMN document_chunks.chunk_type IS 'Type: text, code, table, heading';
