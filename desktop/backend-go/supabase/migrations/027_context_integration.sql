-- ================================================
-- Migration 020: Context Integration
-- Description: Integrate conversations, voice notes, and profiles as context sources
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- PART 1: CONVERSATIONS AS CONTEXT SOURCE
-- Add context-related columns to conversations table
-- ================================================

-- Check if conversations table exists before altering
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'conversations') THEN
        -- Add context source flag
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'is_context_source') THEN
            ALTER TABLE conversations ADD COLUMN is_context_source BOOLEAN DEFAULT TRUE;
        END IF;

        -- Add extracted memories reference
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'extracted_memories') THEN
            ALTER TABLE conversations ADD COLUMN extracted_memories UUID[] DEFAULT '{}';
        END IF;

        -- Add summary
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'summary') THEN
            ALTER TABLE conversations ADD COLUMN summary TEXT;
        END IF;

        -- Add key topics
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'key_topics') THEN
            ALTER TABLE conversations ADD COLUMN key_topics TEXT[] DEFAULT '{}';
        END IF;

        -- Add embedding for semantic search
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'conversations' AND column_name = 'embedding') THEN
            ALTER TABLE conversations ADD COLUMN embedding vector(1536);
        END IF;

        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_conversations_context ON conversations(user_id, is_context_source);
-- Note: embedding index created in migration 037
--         CREATE INDEX IF NOT EXISTS idx_conversations_embedding ON conversations USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
    END IF;
END $$;

-- ================================================
-- CONVERSATION SUMMARIES TABLE
-- AI-generated summaries for efficient context loading
-- ================================================
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,

    -- Summary content
    summary TEXT NOT NULL,
    key_points TEXT[] DEFAULT '{}',
    decisions_made TEXT[] DEFAULT '{}',
    action_items TEXT[] DEFAULT '{}',

    -- Topics and entities mentioned
    topics TEXT[] DEFAULT '{}',
    mentioned_entities JSONB DEFAULT '{}',    -- {"projects": [], "clients": [], "tasks": [], "people": []}

    -- Embedding for semantic search
    embedding vector(1536),

    -- Metadata
    message_count INTEGER,
    time_range_start TIMESTAMPTZ,
    time_range_end TIMESTAMPTZ,
    summarized_at TIMESTAMPTZ DEFAULT NOW(),

    -- Summary quality
    summary_version INTEGER DEFAULT 1,
    is_complete BOOLEAN DEFAULT FALSE,        -- True if conversation is complete

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for conversation_summaries
CREATE INDEX IF NOT EXISTS idx_conv_summaries_conv ON conversation_summaries(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_user ON conversation_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_conv_summaries_time ON conversation_summaries(summarized_at DESC);
-- Note: embedding index created in migration 037
-- CREATE INDEX IF NOT EXISTS idx_conv_summaries_embedding ON conversation_summaries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ================================================
-- PART 2: VOICE NOTES AS CONTEXT SOURCE
-- Add context-related columns to voice_notes table
-- ================================================

DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'voice_notes') THEN
        -- Add project reference
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'project_id') THEN
            ALTER TABLE voice_notes ADD COLUMN project_id UUID;
        END IF;

        -- Add node reference
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'node_id') THEN
            ALTER TABLE voice_notes ADD COLUMN node_id UUID;
        END IF;

        -- Add context source flag
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'is_context_source') THEN
            ALTER TABLE voice_notes ADD COLUMN is_context_source BOOLEAN DEFAULT TRUE;
        END IF;

        -- Add extracted memories reference
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'extracted_memories') THEN
            ALTER TABLE voice_notes ADD COLUMN extracted_memories UUID[] DEFAULT '{}';
        END IF;

        -- Add embedding for semantic search
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'embedding') THEN
            ALTER TABLE voice_notes ADD COLUMN embedding vector(1536);
        END IF;

        -- Add key topics
        IF NOT EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'voice_notes' AND column_name = 'key_topics') THEN
            ALTER TABLE voice_notes ADD COLUMN key_topics TEXT[] DEFAULT '{}';
        END IF;

        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_voice_notes_project ON voice_notes(project_id) WHERE project_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_voice_notes_node ON voice_notes(node_id) WHERE node_id IS NOT NULL;
        CREATE INDEX IF NOT EXISTS idx_voice_notes_context ON voice_notes(user_id, is_context_source);
-- Note: embedding index created in migration 037
--         CREATE INDEX IF NOT EXISTS idx_voice_notes_embedding ON voice_notes USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
    END IF;
END $$;

-- ================================================
-- PART 3: CONTEXT PROFILE ITEMS TABLE
-- Links various content types to context profiles
-- ================================================
CREATE TABLE IF NOT EXISTS context_profile_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    context_profile_id UUID NOT NULL,         -- References context_profiles from migration 017
    user_id VARCHAR(255) NOT NULL,

    -- Item reference
    item_type VARCHAR(50) NOT NULL,           -- 'document', 'artifact', 'memory', 'conversation', 'voice_note', 'kb_context'
    item_id UUID NOT NULL,

    -- Display info
    display_name VARCHAR(255),
    description TEXT,

    -- Item metadata
    token_estimate INTEGER,                   -- Estimated tokens if loaded
    last_accessed_at TIMESTAMPTZ,
    access_count INTEGER DEFAULT 0,

    -- Ordering and pinning
    sort_order INTEGER DEFAULT 0,
    is_pinned BOOLEAN DEFAULT FALSE,
    is_auto_added BOOLEAN DEFAULT FALSE,      -- True if added by rule, false if manual

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Prevent duplicate items in same profile
    UNIQUE(context_profile_id, item_type, item_id)
);

-- Indexes for context_profile_items
CREATE INDEX IF NOT EXISTS idx_profile_items_profile ON context_profile_items(context_profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_type ON context_profile_items(item_type, item_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_user ON context_profile_items(user_id);
CREATE INDEX IF NOT EXISTS idx_profile_items_pinned ON context_profile_items(context_profile_id, is_pinned) WHERE is_pinned = TRUE;
CREATE INDEX IF NOT EXISTS idx_profile_items_sort ON context_profile_items(context_profile_id, sort_order);

-- ================================================
-- TRIGGERS
-- ================================================

-- Update updated_at on conversation_summaries
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

-- Update updated_at on context_profile_items
DROP TRIGGER IF EXISTS trigger_profile_items_updated_at ON context_profile_items;
CREATE TRIGGER trigger_profile_items_updated_at
    BEFORE UPDATE ON context_profile_items
    FOR EACH ROW
    EXECUTE FUNCTION update_conv_summaries_updated_at();

-- Update access tracking on context_profile_items
CREATE OR REPLACE FUNCTION update_profile_item_access()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.last_accessed_at IS DISTINCT FROM OLD.last_accessed_at THEN
        NEW.access_count = COALESCE(OLD.access_count, 0) + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_profile_item_access ON context_profile_items;
CREATE TRIGGER trigger_profile_item_access
    BEFORE UPDATE ON context_profile_items
    FOR EACH ROW
    EXECUTE FUNCTION update_profile_item_access();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE conversation_summaries IS 'AI-generated summaries of conversations for context loading';
COMMENT ON TABLE context_profile_items IS 'Links documents, artifacts, memories to context profiles';

COMMENT ON COLUMN conversation_summaries.mentioned_entities IS 'JSON: {"projects": [], "clients": [], "tasks": [], "people": []}';
COMMENT ON COLUMN context_profile_items.item_type IS 'Type: document, artifact, memory, conversation, voice_note, kb_context';
COMMENT ON COLUMN context_profile_items.is_auto_added IS 'True if added by loading rule, false if manually added';
