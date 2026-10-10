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
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_context_profiles_embedding ON context_profiles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

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
