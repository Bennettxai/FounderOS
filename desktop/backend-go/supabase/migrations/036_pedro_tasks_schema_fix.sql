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
