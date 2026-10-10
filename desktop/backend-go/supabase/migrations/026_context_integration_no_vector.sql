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
