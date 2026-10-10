-- ================================================
-- Migration 019: Document Upload & Management System (NO VECTOR)
-- ================================================

CREATE TABLE IF NOT EXISTS uploaded_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    filename VARCHAR(500) NOT NULL,
    original_filename VARCHAR(500) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    file_type VARCHAR(50) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    storage_path VARCHAR(1000) NOT NULL,
    storage_provider VARCHAR(50) DEFAULT 'local',
    extracted_text TEXT,
    page_count INTEGER,
    word_count INTEGER,
    context_profile_id UUID,
    project_id UUID,
    node_id UUID,
    document_type VARCHAR(100),
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    -- embedding TEXT,  -- Placeholder for vector(1536) when pgvector available
    processing_status VARCHAR(50) DEFAULT 'pending',
    processing_error TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uploaded_docs_user ON uploaded_documents(user_id);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_profile ON uploaded_documents(context_profile_id) WHERE context_profile_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_project ON uploaded_documents(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_node ON uploaded_documents(node_id) WHERE node_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_type ON uploaded_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_file_type ON uploaded_documents(file_type);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_status ON uploaded_documents(processing_status);
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_created ON uploaded_documents(created_at DESC);

CREATE TABLE IF NOT EXISTS document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER,
    page_number INTEGER,
    start_char INTEGER,
    end_char INTEGER,
    section_title VARCHAR(255),
    chunk_type VARCHAR(50) DEFAULT 'text',
    -- embedding TEXT,  -- Placeholder for vector(1536)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_chunks_document ON document_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_index ON document_chunks(document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_doc_chunks_page ON document_chunks(document_id, page_number) WHERE page_number IS NOT NULL;

CREATE TABLE IF NOT EXISTS document_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES uploaded_documents(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    reference_type VARCHAR(50) DEFAULT 'related',
    context TEXT,
    relevance_score DECIMAL(3,2) DEFAULT 0.5,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_refs_document ON document_references(document_id);
CREATE INDEX IF NOT EXISTS idx_doc_refs_entity ON document_references(entity_type, entity_id);

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

COMMENT ON TABLE uploaded_documents IS 'User-uploaded documents (PDFs, markdown, docx, etc.)';
COMMENT ON TABLE document_chunks IS 'Document chunks for RAG retrieval';
COMMENT ON TABLE document_references IS 'References between documents and other entities';
