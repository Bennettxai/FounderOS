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
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

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
