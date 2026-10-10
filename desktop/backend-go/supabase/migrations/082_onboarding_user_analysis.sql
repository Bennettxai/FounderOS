-- Migration: 082_onboarding_user_analysis
-- Create table for storing user analysis during OSA Build onboarding

CREATE TABLE IF NOT EXISTS onboarding_user_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL,

    -- AI Analysis Results
    insights JSONB DEFAULT '[]'::jsonb, -- Array of conversational insight phrases
    interests JSONB DEFAULT '[]'::jsonb, -- Array of detected interests
    tools_used JSONB DEFAULT '[]'::jsonb, -- Array of tools/platforms user uses
    profile_summary TEXT, -- Full text summary of user profile

    -- Email Analysis Metadata
    email_metadata JSONB DEFAULT '{}'::jsonb, -- Structured data from email analysis
    total_emails_analyzed INTEGER DEFAULT 0,
    sender_domains JSONB DEFAULT '[]'::jsonb, -- Array of frequent sender domains
    detected_patterns JSONB DEFAULT '{}'::jsonb, -- Work patterns, communication style, etc.

    -- AI Provider Tracking
    analysis_model VARCHAR(100) NOT NULL, -- e.g., "llama-3.1-8b-instant"
    ai_provider VARCHAR(50) NOT NULL, -- e.g., "groq", "anthropic"
    analysis_tokens_used INTEGER DEFAULT 0,
    analysis_duration_ms INTEGER, -- Time taken for analysis

    -- Status
    status VARCHAR(50) DEFAULT 'analyzing', -- analyzing, completed, failed
    error_message TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Constraints
    UNIQUE(user_id, workspace_id)
);

-- Indexes for performance
CREATE INDEX idx_onboarding_analysis_user ON onboarding_user_analysis(user_id);
CREATE INDEX idx_onboarding_analysis_workspace ON onboarding_user_analysis(workspace_id);
CREATE INDEX idx_onboarding_analysis_status ON onboarding_user_analysis(status);
CREATE INDEX idx_onboarding_analysis_created ON onboarding_user_analysis(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_onboarding_analysis_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER onboarding_analysis_updated_at
    BEFORE UPDATE ON onboarding_user_analysis
    FOR EACH ROW
    EXECUTE FUNCTION update_onboarding_analysis_updated_at();

COMMENT ON TABLE onboarding_user_analysis IS 'Stores AI-generated analysis of user profile during OSA Build onboarding';
COMMENT ON COLUMN onboarding_user_analysis.insights IS 'Conversational insight phrases shown during analyzing screens (e.g., "No-code builder energy")';
COMMENT ON COLUMN onboarding_user_analysis.email_metadata IS 'Raw metadata extracted from Gmail: sender patterns, topics, tools mentioned';
COMMENT ON COLUMN onboarding_user_analysis.analysis_model IS 'Groq model used for analysis (e.g., llama-3.1-8b-instant)';
