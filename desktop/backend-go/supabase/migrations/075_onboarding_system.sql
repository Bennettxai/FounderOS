-- ============================================================================
-- Migration: 045_onboarding_system.sql
-- Description: Conversational AI Onboarding System
-- Created: 2026-01-12
-- Author: BusinessOS Team
-- ============================================================================

-- ============================================================================
-- ONBOARDING SESSIONS TABLE
-- Tracks individual onboarding sessions per user
-- ============================================================================
CREATE TABLE IF NOT EXISTS onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- User reference
    user_id VARCHAR(255) NOT NULL,
    
    -- Session status
    status VARCHAR(50) DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'completed', 'abandoned', 'expired')),
    
    -- Current step tracking
    current_step VARCHAR(50) DEFAULT 'company_name',
    steps_completed JSONB DEFAULT '[]',
    
    -- Extracted data from conversation
    extracted_data JSONB DEFAULT '{}',
    -- Expected structure:
    -- {
    --   "workspace_name": "string",
    --   "business_type": "agency|startup|freelance|ecommerce|consulting|other",
    --   "team_size": "solo|2-5|6-15|16-50|50+",
    --   "role": "string",
    --   "challenge": "string",
    --   "integrations": ["google", "slack", ...]
    -- }
    
    -- AI tracking
    low_confidence_count INTEGER DEFAULT 0,
    fallback_triggered BOOLEAN DEFAULT FALSE,
    
    -- Workspace created from this session
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    
    -- Timestamps
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_user ON onboarding_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_status ON onboarding_sessions(status);
CREATE INDEX IF NOT EXISTS idx_onboarding_sessions_expires ON onboarding_sessions(expires_at) WHERE status = 'in_progress';

-- ============================================================================
-- ONBOARDING CONVERSATION HISTORY TABLE
-- Stores the chat messages for each session
-- ============================================================================
CREATE TABLE IF NOT EXISTS onboarding_conversation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    
    -- Message details
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'agent', 'system')),
    content TEXT NOT NULL,
    
    -- AI metadata
    confidence_score FLOAT,
    extracted_fields JSONB DEFAULT '{}',
    -- Expected structure: { "field_name": "value", ... }
    
    -- For agent messages: what question/step was this
    question_type VARCHAR(50),
    
    -- Message order
    sequence_number INTEGER NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversation_history_session ON onboarding_conversation_history(session_id);
CREATE INDEX IF NOT EXISTS idx_conversation_history_sequence ON onboarding_conversation_history(session_id, sequence_number);

-- ============================================================================
-- WORKSPACE ONBOARDING PROFILES TABLE
-- Stores onboarding answers and recommendations for workspaces
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_onboarding_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Answers from onboarding
    business_type VARCHAR(50),
    team_size VARCHAR(20),
    owner_role VARCHAR(255),
    main_challenge TEXT,
    
    -- Computed recommendations
    recommended_integrations JSONB DEFAULT '[]',
    -- ["google", "slack", "notion"]
    
    -- Feature flags based on answers
    feature_recommendations JSONB DEFAULT '{}',
    -- { "enable_crm": true, "enable_project_management": true, ... }
    
    -- Onboarding source
    onboarding_session_id UUID REFERENCES onboarding_sessions(id) ON DELETE SET NULL,
    onboarding_method VARCHAR(50) DEFAULT 'conversational' CHECK (onboarding_method IN ('conversational', 'fallback_form', 'skipped', 'invited')),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_onboarding_workspace ON workspace_onboarding_profiles(workspace_id);

-- ============================================================================
-- INTEGRATION PENDING CONNECTIONS TABLE
-- Tracks OAuth connections initiated during onboarding
-- ============================================================================
CREATE TABLE IF NOT EXISTS integration_pending_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    
    -- Integration details
    provider VARCHAR(50) NOT NULL,
    -- "google", "microsoft", "slack", "notion", "linear", "hubspot", "airtable", "clickup", "fathom"
    
    -- OAuth state
    oauth_state VARCHAR(255) UNIQUE,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'connecting', 'connected', 'failed')),
    
    -- After connection, link to actual token
    oauth_token_id UUID,
    
    -- Error tracking
    error_message TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    connected_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_connections_session ON integration_pending_connections(session_id);
CREATE INDEX IF NOT EXISTS idx_pending_connections_user ON integration_pending_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_pending_connections_state ON integration_pending_connections(oauth_state);

-- ============================================================================
-- ADD ONBOARDING FIELDS TO WORKSPACES TABLE
-- ============================================================================
DO $$
BEGIN
    -- Add onboarding_completed_at column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'workspaces' AND column_name = 'onboarding_completed_at'
    ) THEN
        ALTER TABLE workspaces ADD COLUMN onboarding_completed_at TIMESTAMPTZ;
    END IF;
    
    -- Add onboarding_data column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'workspaces' AND column_name = 'onboarding_data'
    ) THEN
        ALTER TABLE workspaces ADD COLUMN onboarding_data JSONB DEFAULT '{}';
    END IF;
END $$;

-- ============================================================================
-- ADD ONBOARDING STATUS TO USERS (via user_settings or profiles)
-- ============================================================================
-- Note: This assumes you have a user_settings or profiles table
-- If not, the onboarding status is tracked via workspace membership

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to get or create an onboarding session for a user
CREATE OR REPLACE FUNCTION get_or_create_onboarding_session(p_user_id VARCHAR(255))
RETURNS UUID AS $$
DECLARE
    v_session_id UUID;
BEGIN
    -- Check for existing in-progress session that hasn't expired
    SELECT id INTO v_session_id
    FROM onboarding_sessions
    WHERE user_id = p_user_id
      AND status = 'in_progress'
      AND expires_at > NOW()
    ORDER BY created_at DESC
    LIMIT 1;
    
    -- If found, return it
    IF v_session_id IS NOT NULL THEN
        RETURN v_session_id;
    END IF;
    
    -- Otherwise, create a new session
    INSERT INTO onboarding_sessions (user_id)
    VALUES (p_user_id)
    RETURNING id INTO v_session_id;
    
    RETURN v_session_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check if user needs onboarding
CREATE OR REPLACE FUNCTION user_needs_onboarding(p_user_id VARCHAR(255))
RETURNS BOOLEAN AS $$
DECLARE
    v_workspace_count INTEGER;
    v_completed_session BOOLEAN;
BEGIN
    -- Check if user is a member of any workspace
    SELECT COUNT(*) INTO v_workspace_count
    FROM workspace_members
    WHERE user_id = p_user_id AND status = 'active';
    
    -- If user has workspaces, they don't need onboarding
    IF v_workspace_count > 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Check if user has a completed onboarding session
    SELECT EXISTS (
        SELECT 1 FROM onboarding_sessions
        WHERE user_id = p_user_id AND status = 'completed'
    ) INTO v_completed_session;
    
    -- If completed session exists but no workspace, something went wrong
    -- Let them redo onboarding
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to compute integration recommendations based on answers
CREATE OR REPLACE FUNCTION compute_integration_recommendations(
    p_challenge TEXT,
    p_business_type VARCHAR(50)
)
RETURNS JSONB AS $$
DECLARE
    v_challenge_lower TEXT;
    v_recommendations JSONB;
BEGIN
    v_challenge_lower := LOWER(COALESCE(p_challenge, ''));
    
    -- Challenge-based recommendations
    IF v_challenge_lower LIKE '%organiz%' OR v_challenge_lower LIKE '%chaos%' OR v_challenge_lower LIKE '%mess%' THEN
        v_recommendations := '["notion", "google", "linear"]'::JSONB;
    ELSIF v_challenge_lower LIKE '%scale%' OR v_challenge_lower LIKE '%grow%' OR v_challenge_lower LIKE '%automat%' THEN
        v_recommendations := '["linear", "slack", "airtable"]'::JSONB;
    ELSIF v_challenge_lower LIKE '%client%' OR v_challenge_lower LIKE '%customer%' OR v_challenge_lower LIKE '%crm%' THEN
        v_recommendations := '["hubspot", "slack", "google"]'::JSONB;
    ELSIF v_challenge_lower LIKE '%team%' OR v_challenge_lower LIKE '%collaborat%' OR v_challenge_lower LIKE '%communic%' THEN
        v_recommendations := '["slack", "notion", "linear"]'::JSONB;
    ELSIF v_challenge_lower LIKE '%time%' OR v_challenge_lower LIKE '%busy%' OR v_challenge_lower LIKE '%meeting%' THEN
        v_recommendations := '["google", "fathom", "slack"]'::JSONB;
    ELSE
        -- Default by business type
        CASE p_business_type
            WHEN 'agency' THEN v_recommendations := '["hubspot", "slack", "notion"]'::JSONB;
            WHEN 'consulting' THEN v_recommendations := '["hubspot", "slack", "notion"]'::JSONB;
            WHEN 'startup' THEN v_recommendations := '["linear", "slack", "notion"]'::JSONB;
            WHEN 'freelance' THEN v_recommendations := '["google", "notion", "fathom"]'::JSONB;
            ELSE v_recommendations := '["google", "slack", "notion"]'::JSONB;
        END CASE;
    END IF;
    
    RETURN v_recommendations;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update timestamps trigger
CREATE OR REPLACE FUNCTION update_onboarding_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_onboarding_sessions_updated
    BEFORE UPDATE ON onboarding_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_onboarding_timestamps();

CREATE TRIGGER trigger_workspace_onboarding_profiles_updated
    BEFORE UPDATE ON workspace_onboarding_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_onboarding_timestamps();

-- ============================================================================
-- EXPIRE OLD SESSIONS (can be called by a cron job)
-- ============================================================================
CREATE OR REPLACE FUNCTION expire_old_onboarding_sessions()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    UPDATE onboarding_sessions
    SET status = 'expired', updated_at = NOW()
    WHERE status = 'in_progress' AND expires_at < NOW();
    
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- GRANT PERMISSIONS (adjust based on your role setup)
-- ============================================================================
-- GRANT ALL ON onboarding_sessions TO your_app_role;
-- GRANT ALL ON onboarding_conversation_history TO your_app_role;
-- GRANT ALL ON workspace_onboarding_profiles TO your_app_role;
-- GRANT ALL ON integration_pending_connections TO your_app_role;
