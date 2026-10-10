-- +migrate Up
-- Onboarding email analysis and metadata tables

-- Onboarding sessions table
CREATE TABLE IF NOT EXISTS onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress', -- in_progress, completed, abandoned, expired
    current_step VARCHAR(100) NOT NULL DEFAULT 'company_name',
    steps_completed TEXT[] DEFAULT '{}',
    extracted_data JSONB DEFAULT '{}',
    low_confidence_count INT DEFAULT 0,
    fallback_triggered BOOLEAN DEFAULT FALSE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    analysis_completed BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_onboarding_sessions_user_id ON onboarding_sessions(user_id);
CREATE INDEX idx_onboarding_sessions_status ON onboarding_sessions(status);
CREATE INDEX idx_onboarding_sessions_expires_at ON onboarding_sessions(expires_at);

-- Onboarding conversation history
CREATE TABLE IF NOT EXISTS onboarding_conversation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- user, agent, system
    content TEXT NOT NULL,
    confidence_score DECIMAL(3,2),
    extracted_fields JSONB,
    question_type VARCHAR(100),
    sequence_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_onboarding_conversation_session_id ON onboarding_conversation_history(session_id);
CREATE INDEX idx_onboarding_conversation_sequence ON onboarding_conversation_history(session_id, sequence_number);

-- Onboarding email metadata (per-email extracted data)
CREATE TABLE IF NOT EXISTS onboarding_email_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES onboarding_sessions(id) ON DELETE CASCADE,
    email_id VARCHAR(255) NOT NULL, -- Gmail message ID
    sender_domain VARCHAR(255),
    subject_keywords TEXT[], -- Top keywords from subject
    body_keywords TEXT[], -- Top keywords from body
    detected_tools JSONB DEFAULT '{}', -- {"Slack": 5, "Notion": 3}
    topics JSONB DEFAULT '{}', -- {"collaboration": 10, "development": 5}
    sentiment VARCHAR(50), -- positive, negative, neutral
    importance_score DECIMAL(3,2), -- 0.00 - 1.00
    category VARCHAR(100), -- work, personal, marketing, etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, email_id)
);

CREATE INDEX idx_onboarding_email_metadata_session_id ON onboarding_email_metadata(session_id);
CREATE INDEX idx_onboarding_email_metadata_email_id ON onboarding_email_metadata(email_id);
CREATE INDEX idx_onboarding_email_metadata_sender_domain ON onboarding_email_metadata(sender_domain);

-- +migrate Down
DROP TABLE IF EXISTS onboarding_email_metadata CASCADE;
DROP TABLE IF EXISTS onboarding_conversation_history CASCADE;
DROP TABLE IF EXISTS onboarding_sessions CASCADE;
