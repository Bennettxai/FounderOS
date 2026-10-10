-- Combined Migrations for Supabase
-- Generated: 2026-01-02T18:17:09.476Z
-- Apply this via Supabase SQL Editor if direct connection fails


-- ================================================
-- Migration: 016_memories.sql
-- ================================================

-- ================================================
-- Migration 016: Episodic Memory System
-- Description: Core memory tables for storing and retrieving user memories
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- Ensure pgvector extension is available
CREATE EXTENSION IF NOT EXISTS vector;

-- ================================================
-- MEMORIES TABLE
-- Core memory entries - stores individual memories
-- ================================================
CREATE TABLE IF NOT EXISTS memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Memory Identity
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    content TEXT NOT NULL,

    -- Memory Type: 'fact', 'preference', 'decision', 'pattern', 'insight', 'interaction', 'learning'
    memory_type VARCHAR(50) NOT NULL,
    category VARCHAR(100),

    -- Source Tracking
    source_type VARCHAR(50) NOT NULL,  -- 'conversation', 'voice_note', 'document', 'task', 'project', 'manual', 'inferred'
    source_id UUID,
    source_context TEXT,

    -- Hierarchy Links
    project_id UUID,
    node_id UUID,

    -- Relevance & Retrieval
    importance_score DECIMAL(3,2) DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,

    -- Embeddings for Semantic Search (OpenAI ada-002 = 1536 dimensions)
    embedding vector(1536),
    embedding_model VARCHAR(100),

    -- Memory Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ,

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for memories
CREATE INDEX IF NOT EXISTS idx_memories_user ON memories(user_id);
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(memory_type);
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id);
CREATE INDEX IF NOT EXISTS idx_memories_node ON memories(node_id);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance_score DESC);
CREATE INDEX IF NOT EXISTS idx_memories_active ON memories(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at DESC);

-- Vector index for semantic search (IVFFlat for approximate nearest neighbor)
CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- MEMORY ASSOCIATIONS TABLE
-- Links memories to other entities
-- ================================================
CREATE TABLE IF NOT EXISTS memory_associations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,

    -- What is this memory associated with?
    entity_type VARCHAR(50) NOT NULL,  -- 'project', 'node', 'task', 'client', 'artifact', 'context', 'conversation'
    entity_id UUID NOT NULL,

    -- Association strength
    relevance_score DECIMAL(3,2) DEFAULT 0.5,
    association_type VARCHAR(50),  -- 'about', 'created_from', 'related_to', 'derived_from'

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for memory_associations
CREATE INDEX IF NOT EXISTS idx_memory_assoc_memory ON memory_associations(memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_assoc_entity ON memory_associations(entity_type, entity_id);

-- ================================================
-- MEMORY ACCESS LOG TABLE
-- Track when and how memories are accessed
-- ================================================
CREATE TABLE IF NOT EXISTS memory_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Access context
    access_type VARCHAR(50) NOT NULL,  -- 'agent_retrieval', 'user_view', 'search_result', 'auto_inject'
    accessing_agent VARCHAR(100),
    conversation_id UUID,

    -- What triggered the access?
    trigger_query TEXT,
    relevance_score DECIMAL(3,2),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for memory_access_log
CREATE INDEX IF NOT EXISTS idx_memory_access_memory ON memory_access_log(memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_access_time ON memory_access_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_access_user ON memory_access_log(user_id);

-- ================================================
-- USER FACTS TABLE
-- Quick-access important user info (preferences, facts)
-- ================================================
CREATE TABLE IF NOT EXISTS user_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Fact details
    fact_key VARCHAR(255) NOT NULL,      -- e.g., 'preferred_name', 'timezone', 'communication_style'
    fact_value TEXT NOT NULL,
    fact_type VARCHAR(50) NOT NULL,      -- 'preference', 'fact', 'style', 'context'

    -- Source
    source_memory_id UUID REFERENCES memories(id) ON DELETE SET NULL,
    confidence_score DECIMAL(3,2) DEFAULT 1.0,

    -- Metadata
    is_active BOOLEAN DEFAULT TRUE,
    last_confirmed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, fact_key)
);

-- Indexes for user_facts
CREATE INDEX IF NOT EXISTS idx_user_facts_user ON user_facts(user_id);
CREATE INDEX IF NOT EXISTS idx_user_facts_type ON user_facts(fact_type);
CREATE INDEX IF NOT EXISTS idx_user_facts_active ON user_facts(user_id, is_active);

-- ================================================
-- TRIGGERS
-- ================================================

-- Update updated_at on memories
CREATE OR REPLACE FUNCTION update_memories_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_memories_updated_at ON memories;
CREATE TRIGGER trigger_memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW
    EXECUTE FUNCTION update_memories_updated_at();

-- Update updated_at on user_facts
DROP TRIGGER IF EXISTS trigger_user_facts_updated_at ON user_facts;
CREATE TRIGGER trigger_user_facts_updated_at
    BEFORE UPDATE ON user_facts
    FOR EACH ROW
    EXECUTE FUNCTION update_memories_updated_at();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE memories IS 'Episodic memory storage for user context and learnings';
COMMENT ON TABLE memory_associations IS 'Links memories to projects, nodes, tasks, and other entities';
COMMENT ON TABLE memory_access_log IS 'Tracks memory retrieval for learning and optimization';
COMMENT ON TABLE user_facts IS 'Quick-access user preferences and facts';

COMMENT ON COLUMN memories.memory_type IS 'Type: fact, preference, decision, pattern, insight, interaction, learning';
COMMENT ON COLUMN memories.source_type IS 'Source: conversation, voice_note, document, task, project, manual, inferred';
COMMENT ON COLUMN memories.embedding IS 'OpenAI ada-002 embedding (1536 dimensions) for semantic search';



-- ================================================
-- Migration: 017_context_system.sql
-- ================================================

-- ================================================
-- Migration 017: Intelligent Context System
-- Description: Context profiles, loading rules, and session tracking
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- CONTEXT PROFILES TABLE
-- Describes the full context tree for an entity (project, node, application)
-- ================================================
CREATE TABLE IF NOT EXISTS context_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What this profile describes
    entity_type VARCHAR(50) NOT NULL,         -- 'project', 'node', 'application'
    entity_id UUID NOT NULL,

    -- Profile content
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Tree structure (JSON representation of context tree)
    context_tree JSONB NOT NULL DEFAULT '{}',

    -- Summary for quick loading
    summary TEXT,
    key_facts TEXT[] DEFAULT '{}',

    -- Document tracking
    document_types TEXT[] DEFAULT '{}',
    total_documents INTEGER DEFAULT 0,
    total_file_size_bytes BIGINT DEFAULT 0,

    -- Statistics
    total_contexts INTEGER DEFAULT 0,
    total_memories INTEGER DEFAULT 0,
    total_artifacts INTEGER DEFAULT 0,
    total_tasks INTEGER DEFAULT 0,

    -- Embedding for profile matching
    embedding vector(1536),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, entity_type, entity_id)
);

-- Indexes for context_profiles
CREATE INDEX IF NOT EXISTS idx_context_profiles_user ON context_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_context_profiles_entity ON context_profiles(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_context_profiles_embedding ON context_profiles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- CONTEXT LOADING RULES TABLE
-- Defines what context to load automatically based on triggers
-- ================================================
CREATE TABLE IF NOT EXISTS context_loading_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Rule definition
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- When to apply this rule
    trigger_type VARCHAR(50) NOT NULL,        -- 'project_select', 'node_select', 'agent_type', 'keyword', 'always'
    trigger_value VARCHAR(255),               -- The specific trigger (project_id, agent name, keyword)

    -- What to load: Memories
    load_memories BOOLEAN DEFAULT TRUE,
    memory_types TEXT[] DEFAULT '{}',         -- Which memory types to load
    memory_limit INTEGER DEFAULT 10,

    -- What to load: Contexts/KB
    load_contexts BOOLEAN DEFAULT TRUE,
    context_categories TEXT[] DEFAULT '{}',   -- Which KB categories to load
    context_limit INTEGER DEFAULT 5,

    -- What to load: Artifacts
    load_artifacts BOOLEAN DEFAULT FALSE,
    artifact_types TEXT[] DEFAULT '{}',
    artifact_limit INTEGER DEFAULT 3,

    -- What to load: Conversations
    load_recent_conversations BOOLEAN DEFAULT TRUE,
    conversation_limit INTEGER DEFAULT 3,

    -- Priority (higher = loaded first)
    priority INTEGER DEFAULT 0,

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for context_loading_rules
CREATE INDEX IF NOT EXISTS idx_context_rules_user ON context_loading_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_context_rules_trigger ON context_loading_rules(trigger_type, trigger_value);
CREATE INDEX IF NOT EXISTS idx_context_rules_active ON context_loading_rules(user_id, is_active);

-- ================================================
-- AGENT CONTEXT SESSIONS TABLE
-- Tracks context for each agent session (what's loaded, token usage)
-- ================================================
CREATE TABLE IF NOT EXISTS agent_context_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    conversation_id UUID NOT NULL,

    -- Agent info
    agent_type VARCHAR(100) NOT NULL,
    agent_id UUID,                            -- If using custom agent

    -- Context window tracking
    max_context_tokens INTEGER NOT NULL,      -- Model's max context
    used_context_tokens INTEGER DEFAULT 0,
    available_tokens INTEGER,

    -- What's loaded in context
    loaded_memories UUID[] DEFAULT '{}',
    loaded_contexts UUID[] DEFAULT '{}',
    loaded_artifacts UUID[] DEFAULT '{}',
    loaded_documents UUID[] DEFAULT '{}',

    -- System prompt components
    base_system_prompt TEXT,
    injected_context TEXT,
    total_system_prompt_tokens INTEGER,

    -- Session state
    project_id UUID,
    node_id UUID,
    focus_mode VARCHAR(50),

    -- Timestamps
    started_at TIMESTAMPTZ DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

-- Indexes for agent_context_sessions
CREATE INDEX IF NOT EXISTS idx_agent_sessions_user ON agent_context_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_conversation ON agent_context_sessions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_active ON agent_context_sessions(user_id, ended_at) WHERE ended_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_agent_sessions_project ON agent_context_sessions(project_id) WHERE project_id IS NOT NULL;

-- ================================================
-- CONTEXT RETRIEVAL LOG TABLE
-- Tracks what context was retrieved and used in each session
-- ================================================
CREATE TABLE IF NOT EXISTS context_retrieval_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES agent_context_sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- What was retrieved
    retrieval_type VARCHAR(50) NOT NULL,      -- 'memory', 'context', 'artifact', 'document'
    item_id UUID NOT NULL,
    item_title VARCHAR(255),

    -- How it was retrieved
    retrieval_method VARCHAR(50) NOT NULL,    -- 'semantic_search', 'tree_search', 'rule_based', 'manual'
    query_used TEXT,
    relevance_score DECIMAL(3,2),

    -- Token tracking
    token_count INTEGER,
    was_truncated BOOLEAN DEFAULT FALSE,

    -- Usage
    was_used_in_response BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for context_retrieval_log
CREATE INDEX IF NOT EXISTS idx_context_retrieval_session ON context_retrieval_log(session_id);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_item ON context_retrieval_log(item_id);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_user ON context_retrieval_log(user_id);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_type ON context_retrieval_log(retrieval_type);
CREATE INDEX IF NOT EXISTS idx_context_retrieval_time ON context_retrieval_log(created_at DESC);

-- ================================================
-- TRIGGERS
-- ================================================

-- Update updated_at on context_profiles
CREATE OR REPLACE FUNCTION update_context_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_context_profiles_updated_at ON context_profiles;
CREATE TRIGGER trigger_context_profiles_updated_at
    BEFORE UPDATE ON context_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_context_profiles_updated_at();

-- Update updated_at on context_loading_rules
DROP TRIGGER IF EXISTS trigger_context_rules_updated_at ON context_loading_rules;
CREATE TRIGGER trigger_context_rules_updated_at
    BEFORE UPDATE ON context_loading_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_context_profiles_updated_at();

-- Update last_activity_at on agent_context_sessions
CREATE OR REPLACE FUNCTION update_agent_session_activity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_activity_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_agent_session_activity ON agent_context_sessions;
CREATE TRIGGER trigger_agent_session_activity
    BEFORE UPDATE ON agent_context_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_agent_session_activity();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE context_profiles IS 'Context tree profiles for projects, nodes, and applications';
COMMENT ON TABLE context_loading_rules IS 'Rules for automatically loading context based on triggers';
COMMENT ON TABLE agent_context_sessions IS 'Tracks what context is loaded in each agent session';
COMMENT ON TABLE context_retrieval_log IS 'Log of context retrieval for analytics and optimization';

COMMENT ON COLUMN context_profiles.entity_type IS 'Type: project, node, application';
COMMENT ON COLUMN context_profiles.context_tree IS 'JSON tree structure of all context items';
COMMENT ON COLUMN context_loading_rules.trigger_type IS 'Trigger: project_select, node_select, agent_type, keyword, always';
COMMENT ON COLUMN agent_context_sessions.max_context_tokens IS 'Maximum tokens allowed for this model';
COMMENT ON COLUMN context_retrieval_log.retrieval_method IS 'Method: semantic_search, tree_search, rule_based, manual';



-- ================================================
-- Migration: 018_output_styles.sql
-- ================================================

-- ================================================
-- Migration 018: Output Styles System
-- Description: Output style templates and user preferences
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- OUTPUT STYLES TABLE
-- Style templates for AI responses
-- ================================================
CREATE TABLE IF NOT EXISTS output_styles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Style Identity
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),

    -- Style Configuration
    style_type VARCHAR(50) NOT NULL,          -- 'prose', 'structured', 'code', 'mixed'

    -- Formatting Rules
    use_headers BOOLEAN DEFAULT TRUE,
    use_bullets BOOLEAN DEFAULT TRUE,
    use_numbered_lists BOOLEAN DEFAULT FALSE,
    use_paragraphs BOOLEAN DEFAULT TRUE,
    use_code_blocks BOOLEAN DEFAULT FALSE,
    use_tables BOOLEAN DEFAULT FALSE,
    use_blockquotes BOOLEAN DEFAULT FALSE,

    -- Length & Density
    verbosity VARCHAR(20) DEFAULT 'balanced',  -- 'concise', 'balanced', 'detailed', 'comprehensive'
    max_paragraphs INTEGER,
    max_bullets_per_section INTEGER,

    -- Tone
    tone VARCHAR(50) DEFAULT 'professional',   -- 'casual', 'professional', 'formal', 'friendly', 'technical'

    -- System Prompt Addition
    style_instructions TEXT NOT NULL,

    -- Block Mapping (how to convert output to blocks)
    block_mapping JSONB DEFAULT '{}',

    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for output_styles
CREATE INDEX IF NOT EXISTS idx_output_styles_name ON output_styles(name);
CREATE INDEX IF NOT EXISTS idx_output_styles_active ON output_styles(is_active, sort_order);

-- ================================================
-- USER OUTPUT PREFERENCES TABLE
-- User-specific style preferences
-- ================================================
CREATE TABLE IF NOT EXISTS user_output_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Default style
    default_style_id UUID REFERENCES output_styles(id),

    -- Context-specific overrides
    style_overrides JSONB DEFAULT '{}',       -- {"focus_mode:deep": "style_id", "agent:analyst": "style_id"}

    -- Custom instructions
    custom_instructions TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id)
);

-- Indexes for user_output_preferences
CREATE INDEX IF NOT EXISTS idx_user_output_prefs ON user_output_preferences(user_id);

-- ================================================
-- SEED DATA: 8 Pre-defined Output Styles
-- ================================================
INSERT INTO output_styles (name, display_name, description, icon, style_type, use_headers, use_bullets, use_numbered_lists, use_paragraphs, use_code_blocks, use_tables, use_blockquotes, verbosity, tone, style_instructions, block_mapping, is_system, sort_order)
VALUES
    -- 1. Conversational (ChatGPT-like casual)
    ('conversational', 'Conversational', 'Natural, flowing conversation like talking to a friend', 'message-circle', 'prose',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'balanced', 'friendly',
     'Respond in a natural, conversational way. Use flowing paragraphs instead of bullet points or lists. Write as if having a friendly conversation. Avoid formal structure - just communicate naturally and clearly. Use "I" and "you" freely. Keep responses warm and approachable.',
     '{"paragraph": "text", "emphasis": "text"}',
     TRUE, 1),

    -- 2. Professional (Business communication)
    ('professional', 'Professional', 'Clear, structured business communication', 'briefcase', 'structured',
     TRUE, TRUE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'balanced', 'professional',
     'Respond in a professional business style. Use clear structure with headers for main sections. Use bullet points for lists of items or key points. Keep language formal but accessible. Be direct and actionable.',
     '{"h2": "heading", "bullet": "bullet_list", "paragraph": "text"}',
     TRUE, 2),

    -- 3. Technical (Developer documentation)
    ('technical', 'Technical', 'Precise technical documentation with code examples', 'code', 'code',
     TRUE, TRUE, FALSE, TRUE, TRUE, TRUE, FALSE,
     'detailed', 'technical',
     'Respond in a technical documentation style. Include code examples where relevant. Use precise technical terminology. Structure with clear headers. Use tables for comparisons or specifications. Include type definitions and API signatures when applicable.',
     '{"h2": "heading", "code": "code", "table": "table", "bullet": "bullet_list"}',
     TRUE, 3),

    -- 4. Executive Summary (Brief, high-level)
    ('executive', 'Executive Summary', 'Brief, high-level summaries for quick consumption', 'zap', 'structured',
     FALSE, TRUE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'concise', 'formal',
     'Provide brief, executive-level summaries. Lead with the key takeaway or recommendation. Use 3-5 bullet points maximum for supporting details. Avoid technical jargon. Focus on business impact and actionable insights. Keep total response under 200 words.',
     '{"bullet": "bullet_list", "paragraph": "text"}',
     TRUE, 4),

    -- 5. Detailed Analysis (Comprehensive reports)
    ('detailed', 'Detailed Analysis', 'Comprehensive, in-depth analysis with full context', 'file-text', 'structured',
     TRUE, TRUE, TRUE, TRUE, FALSE, TRUE, FALSE,
     'comprehensive', 'professional',
     'Provide comprehensive, detailed analysis. Use clear section headers. Include numbered lists for sequential steps or rankings. Use tables for data comparisons. Provide context and background. Include considerations, trade-offs, and recommendations. Be thorough but organized.',
     '{"h2": "heading", "h3": "subheading", "numbered": "numbered_list", "bullet": "bullet_list", "table": "table", "paragraph": "text"}',
     TRUE, 5),

    -- 6. Creative (Engaging, story-like)
    ('creative', 'Creative', 'Engaging, narrative style for creative content', 'sparkles', 'prose',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, TRUE,
     'detailed', 'casual',
     'Write in an engaging, creative style. Use narrative techniques - metaphors, vivid descriptions, storytelling. Create flow and rhythm in the writing. Engage the reader emotionally. Avoid dry, factual presentation. Make the content memorable and enjoyable to read.',
     '{"paragraph": "text", "blockquote": "quote"}',
     TRUE, 6),

    -- 7. Step-by-Step (Tutorial/how-to)
    ('tutorial', 'Step-by-Step', 'Clear tutorial style with numbered steps', 'list-ordered', 'structured',
     TRUE, FALSE, TRUE, TRUE, TRUE, FALSE, FALSE,
     'detailed', 'friendly',
     'Write in a clear tutorial style. Use numbered steps for any process or procedure. Include code examples with explanations. Add tips or notes for important points. Anticipate common questions or issues. Make sure each step is clear and actionable.',
     '{"h2": "heading", "numbered": "numbered_list", "code": "code", "note": "callout", "paragraph": "text"}',
     TRUE, 7),

    -- 8. Q&A (Direct answers)
    ('qa', 'Q&A', 'Direct question-and-answer format', 'help-circle', 'mixed',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'concise', 'friendly',
     'Answer directly and concisely. Start with the direct answer to the question. Then provide brief supporting context if needed. Avoid unnecessary preamble. If multiple questions, address each clearly. Use simple, clear language.',
     '{"paragraph": "text"}',
     TRUE, 8)

ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    style_type = EXCLUDED.style_type,
    use_headers = EXCLUDED.use_headers,
    use_bullets = EXCLUDED.use_bullets,
    use_numbered_lists = EXCLUDED.use_numbered_lists,
    use_paragraphs = EXCLUDED.use_paragraphs,
    use_code_blocks = EXCLUDED.use_code_blocks,
    use_tables = EXCLUDED.use_tables,
    use_blockquotes = EXCLUDED.use_blockquotes,
    verbosity = EXCLUDED.verbosity,
    tone = EXCLUDED.tone,
    style_instructions = EXCLUDED.style_instructions,
    block_mapping = EXCLUDED.block_mapping,
    updated_at = NOW();

-- ================================================
-- TRIGGERS
-- ================================================
CREATE OR REPLACE FUNCTION update_output_styles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_output_styles_updated_at ON output_styles;
CREATE TRIGGER trigger_output_styles_updated_at
    BEFORE UPDATE ON output_styles
    FOR EACH ROW
    EXECUTE FUNCTION update_output_styles_updated_at();

DROP TRIGGER IF EXISTS trigger_user_output_prefs_updated_at ON user_output_preferences;
CREATE TRIGGER trigger_user_output_prefs_updated_at
    BEFORE UPDATE ON user_output_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_output_styles_updated_at();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE output_styles IS 'AI response style templates';
COMMENT ON TABLE user_output_preferences IS 'User-specific output style preferences';
COMMENT ON COLUMN output_styles.style_type IS 'Type: prose, structured, code, mixed';
COMMENT ON COLUMN output_styles.verbosity IS 'Level: concise, balanced, detailed, comprehensive';
COMMENT ON COLUMN output_styles.block_mapping IS 'Maps markdown elements to block types for document integration';



-- ================================================
-- Migration: 019_documents_no_vector.sql
-- ================================================

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



-- ================================================
-- Migration: 020_context_integration_no_vector.sql
-- ================================================

-- ================================================
-- Migration 020: Context Integration (NO VECTOR)
-- ================================================

-- Add context-related columns to conversations table
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'conversations') THEN
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'is_context_source') THEN
            ALTER TABLE conversations ADD COLUMN is_context_source BOOLEAN DEFAULT TRUE;
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'extracted_memories') THEN
            ALTER TABLE conversations ADD COLUMN extracted_memories UUID[] DEFAULT '{}';
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'summary') THEN
            ALTER TABLE conversations ADD COLUMN summary TEXT;
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'key_topics') THEN
            ALTER TABLE conversations ADD COLUMN key_topics TEXT[] DEFAULT '{}';
        END IF;
        -- Skip vector embedding column
        CREATE INDEX IF NOT EXISTS idx_conversations_context ON conversations(user_id, is_context_source);
    END IF;
END $$;

-- Conversation summaries table
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    key_points TEXT[] DEFAULT '{}',
    decisions_made TEXT[] DEFAULT '{}',
    action_items TEXT[] DEFAULT '{}',
    topics TEXT[] DEFAULT '{}',
    mentioned_entities JSONB DEFAULT '{}',
    -- embedding TEXT,  -- Placeholder for vector(1536)
    message_count INTEGER,
    time_range_start TIMESTAMPTZ,
    time_range_end TIMESTAMPTZ,
    summarized_at TIMESTAMPTZ DEFAULT NOW(),
    summary_version INTEGER DEFAULT 1,
    is_complete BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conv_summaries_conv ON conversation_summaries(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_user ON conversation_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_time ON conversation_summaries(summarized_at DESC);

-- Add context columns to voice_notes if exists
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'voice_notes') THEN
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'project_id') THEN
            ALTER TABLE voice_notes ADD COLUMN project_id UUID;
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'node_id') THEN
            ALTER TABLE voice_notes ADD COLUMN node_id UUID;
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'is_context_source') THEN
            ALTER TABLE voice_notes ADD COLUMN is_context_source BOOLEAN DEFAULT TRUE;
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'extracted_memories') THEN
            ALTER TABLE voice_notes ADD COLUMN extracted_memories UUID[] DEFAULT '{}';
        END IF;
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'key_topics') THEN
            ALTER TABLE voice_notes ADD COLUMN key_topics TEXT[] DEFAULT '{}';
        END IF;
        CREATE INDEX IF NOT EXISTS idx_voice_notes_project ON voice_notes(project_id) WHERE project_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_voice_notes_node ON voice_notes(node_id) WHERE node_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_voice_notes_context ON voice_notes(user_id, is_context_source);
    END IF;
END $$;

-- Context profile items (if not already exists from 017)
CREATE TABLE IF NOT EXISTS context_profile_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_profile_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    item_id UUID NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    token_estimate INTEGER,
    last_accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,
    sort_order INTEGER DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_auto_added BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(context_profile_id, item_type, item_id)
);

CREATE INDEX IF NOT EXISTS idx_profile_items_profile ON context_profile_items(context_profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_type ON context_profile_items(item_type, item_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_user ON context_profile_items(user_id);

-- Triggers
CREATE OR REPLACE FUNCTION update_conv_summaries_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_conv_summaries_updated_at ON conversation_summaries;
CREATE TRIGGER trigger_conv_summaries_updated_at
    BEFORE UPDATE ON conversation_summaries
    FOR EACH ROW
    EXECUTE FUNCTION update_conv_summaries_updated_at();

COMMENT ON TABLE conversation_summaries IS 'AI-generated summaries of conversations for context loading';



-- ================================================
-- Migration: 021_learning_system.sql
-- ================================================

-- ================================================
-- Migration 021: Self-Learning System
-- Description: Track learnings, behavior patterns, and user feedback
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- LEARNING EVENTS TABLE
-- Tracks what the system learns from interactions
-- ================================================
CREATE TABLE IF NOT EXISTS learning_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What was learned
    learning_type VARCHAR(50) NOT NULL,       -- 'correction', 'preference', 'pattern', 'feedback', 'behavior', 'fact'
    learning_content TEXT NOT NULL,
    learning_summary VARCHAR(500),            -- Brief summary

    -- Source of learning
    source_type VARCHAR(50) NOT NULL,         -- 'explicit_feedback', 'implicit_behavior', 'correction', 'conversation', 'voice_note'
    source_id UUID,
    source_context TEXT,                      -- Additional context about the source

    -- Confidence & Application
    confidence_score DECIMAL(3,2) DEFAULT 0.5,
    times_applied INTEGER DEFAULT 0,
    last_applied_at TIMESTAMPTZ,
    successful_applications INTEGER DEFAULT 0,

    -- Whether it resulted in a memory or fact
    created_memory_id UUID,                   -- References memories table
    created_fact_key VARCHAR(255),            -- References user_facts

    -- Categorization
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',

    -- Validation
    was_validated BOOLEAN DEFAULT FALSE,
    validated_at TIMESTAMPTZ,
    validation_result VARCHAR(50),            -- 'confirmed', 'rejected', 'modified'
    validation_notes TEXT,

    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    superseded_by UUID,                       -- If this learning was replaced by a newer one

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for learning_events
CREATE INDEX IF NOT EXISTS idx_learning_events_user ON learning_events(user_id);
CREATE INDEX IF NOT EXISTS idx_learning_events_type ON learning_events(learning_type);
CREATE INDEX IF NOT EXISTS idx_learning_events_source ON learning_events(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_learning_events_active ON learning_events(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_learning_events_confidence ON learning_events(user_id, confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_learning_events_created ON learning_events(created_at DESC);

-- ================================================
-- USER BEHAVIOR PATTERNS TABLE
-- Observed patterns in user behavior over time
-- ================================================
CREATE TABLE IF NOT EXISTS user_behavior_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Pattern details
    pattern_type VARCHAR(100) NOT NULL,       -- 'time_preference', 'topic_interest', 'communication_style', 'tool_usage', 'workflow'
    pattern_key VARCHAR(255) NOT NULL,        -- Specific pattern identifier
    pattern_value TEXT NOT NULL,              -- The observed pattern value
    pattern_description TEXT,                 -- Human-readable description

    -- Evidence tracking
    observation_count INTEGER DEFAULT 1,
    first_observed_at TIMESTAMPTZ DEFAULT NOW(),
    last_observed_at TIMESTAMPTZ DEFAULT NOW(),
    evidence_ids UUID[] DEFAULT '{}',         -- References to supporting data

    -- Confidence
    confidence_score DECIMAL(3,2) DEFAULT 0.5,
    min_observations_for_confidence INTEGER DEFAULT 3,

    -- Application
    is_applied BOOLEAN DEFAULT FALSE,         -- Whether this pattern is actively used
    applied_in_prompt BOOLEAN DEFAULT FALSE,  -- Whether included in system prompts

    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    deactivated_at TIMESTAMPTZ,
    deactivation_reason TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, pattern_type, pattern_key)
);

-- Indexes for user_behavior_patterns
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_user ON user_behavior_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_type ON user_behavior_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_active ON user_behavior_patterns(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_confidence ON user_behavior_patterns(user_id, confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_applied ON user_behavior_patterns(user_id, is_applied) WHERE is_applied = TRUE;

-- ================================================
-- FEEDBACK LOG TABLE
-- Explicit user feedback on AI outputs
-- ================================================
CREATE TABLE IF NOT EXISTS feedback_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What feedback is about
    target_type VARCHAR(50) NOT NULL,         -- 'message', 'artifact', 'memory', 'suggestion', 'agent_response'
    target_id UUID NOT NULL,

    -- Feedback content
    feedback_type VARCHAR(50) NOT NULL,       -- 'thumbs_up', 'thumbs_down', 'correction', 'comment', 'rating'
    feedback_value TEXT,                      -- The actual feedback (correction text, comment, rating value)
    rating INTEGER,                           -- 1-5 if rating type

    -- Context
    conversation_id UUID,
    agent_type VARCHAR(100),
    focus_mode VARCHAR(50),

    -- What the AI produced (for learning)
    original_content TEXT,                    -- What the AI said/generated
    expected_content TEXT,                    -- What user expected (if correction)

    -- Processing
    was_processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    resulting_learning_id UUID,               -- References learning_events

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for feedback_log
CREATE INDEX IF NOT EXISTS idx_feedback_log_user ON feedback_log(user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_log_target ON feedback_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_feedback_log_type ON feedback_log(feedback_type);
CREATE INDEX IF NOT EXISTS idx_feedback_log_unprocessed ON feedback_log(was_processed) WHERE was_processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_feedback_log_created ON feedback_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_log_conversation ON feedback_log(conversation_id) WHERE conversation_id IS NOT NULL;

-- ================================================
-- PERSONALIZATION PROFILES TABLE
-- Aggregated user personalization settings
-- ================================================
CREATE TABLE IF NOT EXISTS personalization_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,

    -- Communication preferences
    preferred_tone VARCHAR(50) DEFAULT 'professional',      -- 'formal', 'casual', 'professional', 'friendly'
    preferred_verbosity VARCHAR(50) DEFAULT 'balanced',     -- 'concise', 'balanced', 'detailed'
    preferred_format VARCHAR(50) DEFAULT 'structured',      -- 'prose', 'bullets', 'structured', 'mixed'

    -- Content preferences
    prefers_examples BOOLEAN DEFAULT TRUE,
    prefers_analogies BOOLEAN DEFAULT FALSE,
    prefers_code_samples BOOLEAN DEFAULT FALSE,
    prefers_visual_aids BOOLEAN DEFAULT FALSE,

    -- Expertise tracking
    expertise_areas TEXT[] DEFAULT '{}',      -- Topics user knows well
    learning_areas TEXT[] DEFAULT '{}',       -- Topics user is learning
    common_topics TEXT[] DEFAULT '{}',        -- Frequently discussed topics

    -- Time preferences
    timezone VARCHAR(100),
    preferred_working_hours JSONB DEFAULT '{}',  -- {"start": "09:00", "end": "18:00", "days": [1,2,3,4,5]}
    most_active_hours INTEGER[] DEFAULT '{}',    -- Hours when user is most active

    -- Interaction stats
    total_conversations INTEGER DEFAULT 0,
    total_feedback_given INTEGER DEFAULT 0,
    positive_feedback_ratio DECIMAL(3,2) DEFAULT 0.5,

    -- Profile completeness
    profile_completeness DECIMAL(3,2) DEFAULT 0.0,  -- 0.0 to 1.0
    last_profile_update TIMESTAMPTZ DEFAULT NOW(),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for personalization_profiles
CREATE INDEX IF NOT EXISTS idx_personalization_user ON personalization_profiles(user_id);

-- ================================================
-- TRIGGERS
-- ================================================

-- Update updated_at on learning_events
CREATE OR REPLACE FUNCTION update_learning_events_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_learning_events_updated_at ON learning_events;
CREATE TRIGGER trigger_learning_events_updated_at
    BEFORE UPDATE ON learning_events
    FOR EACH ROW
    EXECUTE FUNCTION update_learning_events_updated_at();

-- Update updated_at on user_behavior_patterns
DROP TRIGGER IF EXISTS trigger_behavior_patterns_updated_at ON user_behavior_patterns;
CREATE TRIGGER trigger_behavior_patterns_updated_at
    BEFORE UPDATE ON user_behavior_patterns
    FOR EACH ROW
    EXECUTE FUNCTION update_learning_events_updated_at();

-- Update updated_at on personalization_profiles
DROP TRIGGER IF EXISTS trigger_personalization_updated_at ON personalization_profiles;
CREATE TRIGGER trigger_personalization_updated_at
    BEFORE UPDATE ON personalization_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_learning_events_updated_at();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE learning_events IS 'Tracks what the system learns from user interactions';
COMMENT ON TABLE user_behavior_patterns IS 'Observed patterns in user behavior over time';
COMMENT ON TABLE feedback_log IS 'Explicit user feedback on AI outputs';
COMMENT ON TABLE personalization_profiles IS 'Aggregated user personalization settings';

COMMENT ON COLUMN learning_events.learning_type IS 'Type: correction, preference, pattern, feedback, behavior, fact';
COMMENT ON COLUMN learning_events.source_type IS 'Source: explicit_feedback, implicit_behavior, correction, conversation, voice_note';
COMMENT ON COLUMN user_behavior_patterns.pattern_type IS 'Type: time_preference, topic_interest, communication_style, tool_usage, workflow';
COMMENT ON COLUMN feedback_log.feedback_type IS 'Type: thumbs_up, thumbs_down, correction, comment, rating';
COMMENT ON COLUMN feedback_log.target_type IS 'Target: message, artifact, memory, suggestion, agent_response';



-- ================================================
-- Migration: 022_application_profiles_no_vector.sql
-- ================================================

-- ================================================
-- Migration 022: Application Profiles (NO VECTOR)
-- ================================================

CREATE TABLE IF NOT EXISTS application_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    app_type VARCHAR(100),
    version VARCHAR(50),
    tech_stack JSONB DEFAULT '{}',
    languages TEXT[] DEFAULT '{}',
    frameworks TEXT[] DEFAULT '{}',
    structure_tree JSONB NOT NULL DEFAULT '{}',
    root_path VARCHAR(1000),
    components JSONB DEFAULT '[]',
    total_components INTEGER DEFAULT 0,
    modules JSONB DEFAULT '[]',
    total_modules INTEGER DEFAULT 0,
    icons JSONB DEFAULT '[]',
    assets JSONB DEFAULT '[]',
    api_endpoints JSONB DEFAULT '[]',
    total_endpoints INTEGER DEFAULT 0,
    database_schema JSONB DEFAULT '{}',
    total_tables INTEGER DEFAULT 0,
    conventions JSONB DEFAULT '{}',
    coding_standards TEXT,
    integration_points JSONB DEFAULT '[]',
    readme_summary TEXT,
    documentation_urls TEXT[] DEFAULT '{}',
    -- embedding TEXT,  -- Placeholder for vector(1536)
    last_synced_at TIMESTAMPTZ,
    sync_source VARCHAR(255),
    sync_branch VARCHAR(100),
    sync_commit VARCHAR(100),
    auto_sync_enabled BOOLEAN DEFAULT FALSE,
    last_analyzed_at TIMESTAMPTZ,
    analysis_version INTEGER DEFAULT 1,
    lines_of_code INTEGER,
    file_count INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_profiles_user ON application_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_app_profiles_type ON application_profiles(app_type);
CREATE INDEX IF NOT EXISTS idx_app_profiles_name ON application_profiles(user_id, name);

CREATE TABLE IF NOT EXISTS application_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    component_type VARCHAR(100),
    description TEXT,
    props JSONB DEFAULT '[]',
    events JSONB DEFAULT '[]',
    slots JSONB DEFAULT '[]',
    imports TEXT[] DEFAULT '{}',
    exported_as VARCHAR(255),
    usage_examples JSONB DEFAULT '[]',
    used_in TEXT[] DEFAULT '{}',
    lines_of_code INTEGER,
    last_modified_at TIMESTAMPTZ,
    -- embedding TEXT,  -- Placeholder for vector(1536)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_app_components_profile ON application_components(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_components_type ON application_components(component_type);
CREATE INDEX IF NOT EXISTS idx_app_components_name ON application_components(app_profile_id, name);

CREATE TABLE IF NOT EXISTS application_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_profile_id UUID NOT NULL REFERENCES application_profiles(id) ON DELETE CASCADE,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    handler_path VARCHAR(1000),
    description TEXT,
    summary VARCHAR(255),
    path_params JSONB DEFAULT '[]',
    query_params JSONB DEFAULT '[]',
    body_schema JSONB DEFAULT '{}',
    response_schema JSONB DEFAULT '{}',
    auth_required BOOLEAN DEFAULT FALSE,
    required_permissions TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    deprecated BOOLEAN DEFAULT FALSE,
    -- embedding TEXT,  -- Placeholder for vector(1536)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(app_profile_id, method, path)
);

CREATE INDEX IF NOT EXISTS idx_app_endpoints_profile ON application_api_endpoints(app_profile_id);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_method ON application_api_endpoints(method);
CREATE INDEX IF NOT EXISTS idx_app_endpoints_path ON application_api_endpoints(app_profile_id, path);

-- Triggers
CREATE OR REPLACE FUNCTION update_app_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_app_profiles_updated_at ON application_profiles;
CREATE TRIGGER trigger_app_profiles_updated_at
    BEFORE UPDATE ON application_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

DROP TRIGGER IF EXISTS trigger_app_components_updated_at ON application_components;
CREATE TRIGGER trigger_app_components_updated_at
    BEFORE UPDATE ON application_components
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

DROP TRIGGER IF EXISTS trigger_app_endpoints_updated_at ON application_api_endpoints;
CREATE TRIGGER trigger_app_endpoints_updated_at
    BEFORE UPDATE ON application_api_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profiles_updated_at();

-- Update counts trigger
CREATE OR REPLACE FUNCTION update_app_profile_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'application_components' THEN
        UPDATE application_profiles
        SET total_components = (SELECT COUNT(*) FROM application_components WHERE app_profile_id = COALESCE(NEW.app_profile_id, OLD.app_profile_id))
        WHERE id = COALESCE(NEW.app_profile_id, OLD.app_profile_id);
    ELSIF TG_TABLE_NAME = 'application_api_endpoints' THEN
        UPDATE application_profiles
        SET total_endpoints = (SELECT COUNT(*) FROM application_api_endpoints WHERE app_profile_id = COALESCE(NEW.app_profile_id, OLD.app_profile_id))
        WHERE id = COALESCE(NEW.app_profile_id, OLD.app_profile_id);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_component_count ON application_components;
CREATE TRIGGER trigger_update_component_count
    AFTER INSERT OR DELETE ON application_components
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profile_counts();

DROP TRIGGER IF EXISTS trigger_update_endpoint_count ON application_api_endpoints;
CREATE TRIGGER trigger_update_endpoint_count
    AFTER INSERT OR DELETE ON application_api_endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_app_profile_counts();

COMMENT ON TABLE application_profiles IS 'Context profiles for applications and codebases';
COMMENT ON TABLE application_components IS 'Detailed component registry for applications';
COMMENT ON TABLE application_api_endpoints IS 'API endpoint registry for applications';



-- ================================================
-- Migration: 023_pedro_tasks_schema_fix.sql
-- ================================================

-- ================================================
-- Migration 023: Pedro Tasks Schema Fix
-- Description: Add missing columns for Pedro Tasks services compatibility
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- PART 1: FIX conversation_summaries TABLE
-- Add columns expected by ConversationIntelligenceService
-- ================================================

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'conversation_summaries') THEN
        -- Add title column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'title') THEN
            ALTER TABLE conversation_summaries ADD COLUMN title VARCHAR(500);
        END IF;

        -- Add sentiment column (JSONB for flexibility)
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'sentiment') THEN
            ALTER TABLE conversation_summaries ADD COLUMN sentiment JSONB DEFAULT '{}';
        END IF;

        -- Rename mentioned_entities to entities if needed (or add entities)
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'entities') THEN
            ALTER TABLE conversation_summaries ADD COLUMN entities JSONB DEFAULT '{}';
        END IF;

        -- Add questions column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'questions') THEN
            ALTER TABLE conversation_summaries ADD COLUMN questions TEXT[] DEFAULT '{}';
        END IF;

        -- Add decisions column (separate from decisions_made for service compatibility)
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'decisions') THEN
            ALTER TABLE conversation_summaries ADD COLUMN decisions TEXT[] DEFAULT '{}';
        END IF;

        -- Add code_mentions column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'code_mentions') THEN
            ALTER TABLE conversation_summaries ADD COLUMN code_mentions JSONB DEFAULT '[]';
        END IF;

        -- Add token_count column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'token_count') THEN
            ALTER TABLE conversation_summaries ADD COLUMN token_count INTEGER DEFAULT 0;
        END IF;

        -- Add duration column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'duration') THEN
            ALTER TABLE conversation_summaries ADD COLUMN duration VARCHAR(50);
        END IF;

        -- Add metadata column
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversation_summaries' AND column_name = 'metadata') THEN
            ALTER TABLE conversation_summaries ADD COLUMN metadata JSONB DEFAULT '{}';
        END IF;
    END IF;
END $$;

-- Add unique constraint on conversation_id if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'conversation_summaries_conversation_id_key'
    ) THEN
        ALTER TABLE conversation_summaries ADD CONSTRAINT conversation_summaries_conversation_id_key UNIQUE (conversation_id);
    END IF;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

-- ================================================
-- PART 2: FIX memories TABLE
-- Add alias columns for MemoryExtractorService compatibility
-- ================================================

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memories') THEN
        -- The service uses 'type' but migration has 'memory_type'
        -- The service uses 'importance' but migration has 'importance_score'
        -- The service uses 'source' but migration has 'source_type'

        -- We'll create a view OR add compatibility columns
        -- Option: Add computed columns via triggers or just use correct names in service

        -- For now, let's make title and summary nullable (service doesn't always provide them)
        ALTER TABLE memories ALTER COLUMN title DROP NOT NULL;
        ALTER TABLE memories ALTER COLUMN summary DROP NOT NULL;
    END IF;
EXCEPTION WHEN others THEN
    NULL;
END $$;

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON COLUMN conversation_summaries.title IS 'AI-generated title for the conversation';
COMMENT ON COLUMN conversation_summaries.sentiment IS 'Sentiment analysis: {"overall": "positive", "scores": {...}}';
COMMENT ON COLUMN conversation_summaries.entities IS 'Extracted entities: {"people": [], "tools": [], "concepts": []}';
COMMENT ON COLUMN conversation_summaries.questions IS 'Questions identified in the conversation';
COMMENT ON COLUMN conversation_summaries.decisions IS 'Decisions made during the conversation';
COMMENT ON COLUMN conversation_summaries.code_mentions IS 'Code/file references: [{"file": "...", "line": ...}]';
COMMENT ON COLUMN conversation_summaries.token_count IS 'Total tokens in the conversation';
COMMENT ON COLUMN conversation_summaries.duration IS 'Conversation duration string';



-- ================================================
-- Migration: 024_embedding_dimensions_768.sql
-- ================================================

-- ================================================
-- Migration 024: Align embedding dimensions to 768
-- Description: Several tables were created with vector(1536) but the runtime embedding model is nomic-embed-text (768).
--              This migration resets embedding columns to vector(768) so semantic search/indexing works consistently.
-- Notes: Embeddings are derived data; this migration drops existing embedding columns (and indexes) and recreates them.
-- Date: 2026-01-02
-- ================================================

-- Ensure pgvector exists
CREATE EXTENSION IF NOT EXISTS vector;

-- Helper: reset a table's embedding column to vector(768)
-- We intentionally DROP COLUMN to avoid cast failures when old embeddings (1536) exist.

-- Memories
ALTER TABLE memories DROP COLUMN IF EXISTS embedding;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_memories_embedding;
CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING hnsw (embedding vector_cosine_ops);

-- Uploaded documents
ALTER TABLE uploaded_documents DROP COLUMN IF EXISTS embedding;
ALTER TABLE uploaded_documents ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_uploaded_docs_embedding;
CREATE INDEX IF NOT EXISTS idx_uploaded_docs_embedding ON uploaded_documents USING hnsw (embedding vector_cosine_ops);

-- Document chunks
ALTER TABLE document_chunks DROP COLUMN IF EXISTS embedding;
ALTER TABLE document_chunks ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_doc_chunks_embedding;
CREATE INDEX IF NOT EXISTS idx_doc_chunks_embedding ON document_chunks USING hnsw (embedding vector_cosine_ops);

-- Conversations (context integration)
ALTER TABLE conversations DROP COLUMN IF EXISTS embedding;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_conversations_embedding;
CREATE INDEX IF NOT EXISTS idx_conversations_embedding ON conversations USING hnsw (embedding vector_cosine_ops);

-- Conversation summaries
ALTER TABLE conversation_summaries DROP COLUMN IF EXISTS embedding;
ALTER TABLE conversation_summaries ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_conv_summaries_embedding;
CREATE INDEX IF NOT EXISTS idx_conv_summaries_embedding ON conversation_summaries USING hnsw (embedding vector_cosine_ops);

-- Voice notes
ALTER TABLE voice_notes DROP COLUMN IF EXISTS embedding;
ALTER TABLE voice_notes ADD COLUMN IF NOT EXISTS embedding vector(768);
DROP INDEX IF EXISTS idx_voice_notes_embedding;
CREATE INDEX IF NOT EXISTS idx_voice_notes_embedding ON voice_notes USING hnsw (embedding vector_cosine_ops);

-- Optional tables (some environments may not have these yet)
DO $$
BEGIN
	IF to_regclass('public.context_profiles') IS NOT NULL THEN
		ALTER TABLE context_profiles DROP COLUMN IF EXISTS embedding;
		ALTER TABLE context_profiles ADD COLUMN IF NOT EXISTS embedding vector(768);
		DROP INDEX IF EXISTS idx_context_profiles_embedding;
		CREATE INDEX IF NOT EXISTS idx_context_profiles_embedding ON context_profiles USING hnsw (embedding vector_cosine_ops);
	END IF;

	IF to_regclass('public.application_profiles') IS NOT NULL THEN
		ALTER TABLE application_profiles DROP COLUMN IF EXISTS embedding;
		ALTER TABLE application_profiles ADD COLUMN IF NOT EXISTS embedding vector(768);
		DROP INDEX IF EXISTS idx_app_profiles_embedding;
		CREATE INDEX IF NOT EXISTS idx_app_profiles_embedding ON application_profiles USING hnsw (embedding vector_cosine_ops);
	END IF;

	IF to_regclass('public.application_components') IS NOT NULL THEN
		ALTER TABLE application_components DROP COLUMN IF EXISTS embedding;
		ALTER TABLE application_components ADD COLUMN IF NOT EXISTS embedding vector(768);
		DROP INDEX IF EXISTS idx_app_components_embedding;
		CREATE INDEX IF NOT EXISTS idx_app_components_embedding ON application_components USING hnsw (embedding vector_cosine_ops);
	END IF;

	IF to_regclass('public.application_api_endpoints') IS NOT NULL THEN
		ALTER TABLE application_api_endpoints DROP COLUMN IF EXISTS embedding;
		ALTER TABLE application_api_endpoints ADD COLUMN IF NOT EXISTS embedding vector(768);
		DROP INDEX IF EXISTS idx_app_endpoints_embedding;
		CREATE INDEX IF NOT EXISTS idx_app_endpoints_embedding ON application_api_endpoints USING hnsw (embedding vector_cosine_ops);
	END IF;

	IF to_regclass('public.code_patterns') IS NOT NULL THEN
		ALTER TABLE code_patterns DROP COLUMN IF EXISTS embedding;
		ALTER TABLE code_patterns ADD COLUMN IF NOT EXISTS embedding vector(768);
		DROP INDEX IF EXISTS idx_code_patterns_embedding;
		CREATE INDEX IF NOT EXISTS idx_code_patterns_embedding ON code_patterns USING hnsw (embedding vector_cosine_ops);
	END IF;
END $$;


