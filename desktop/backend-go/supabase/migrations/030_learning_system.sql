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
