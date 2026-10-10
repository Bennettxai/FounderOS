-- Migration: 084_onboarding_email_metadata
-- Create table for storing extracted email metadata used in onboarding analysis

CREATE TABLE IF NOT EXISTS onboarding_email_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    analysis_id UUID NOT NULL REFERENCES onboarding_user_analysis(id) ON DELETE CASCADE,

    -- Email Reference
    email_id UUID, -- Reference to emails table if available
    external_id VARCHAR(255), -- Gmail message ID

    -- Extracted Metadata
    sender_domain VARCHAR(255),
    sender_email VARCHAR(255),
    subject_keywords JSONB DEFAULT '[]'::jsonb, -- Extracted keywords from subject
    body_keywords JSONB DEFAULT '[]'::jsonb, -- Extracted keywords from body
    detected_tools JSONB DEFAULT '[]'::jsonb, -- Tools/platforms mentioned (Figma, Notion, etc.)
    detected_topics JSONB DEFAULT '[]'::jsonb, -- Topics/themes (design, development, marketing)

    -- Classification
    category VARCHAR(100), -- work, personal, marketing, newsletter, etc.
    sentiment VARCHAR(50), -- positive, neutral, negative
    importance_score DECIMAL(3, 2), -- 0.00 to 1.00

    -- Timestamps
    email_date TIMESTAMPTZ, -- When email was sent
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_email_metadata_user ON onboarding_email_metadata(user_id);
CREATE INDEX idx_email_metadata_analysis ON onboarding_email_metadata(analysis_id);
CREATE INDEX idx_email_metadata_sender_domain ON onboarding_email_metadata(sender_domain);
CREATE INDEX idx_email_metadata_category ON onboarding_email_metadata(category);
CREATE INDEX idx_email_metadata_email_date ON onboarding_email_metadata(email_date DESC);

COMMENT ON TABLE onboarding_email_metadata IS 'Stores extracted metadata from emails analyzed during onboarding';
COMMENT ON COLUMN onboarding_email_metadata.detected_tools IS 'Tools/platforms mentioned in email (e.g., ["Figma", "Notion", "GitHub"])';
COMMENT ON COLUMN onboarding_email_metadata.importance_score IS 'AI-calculated importance score from 0.00 to 1.00';
