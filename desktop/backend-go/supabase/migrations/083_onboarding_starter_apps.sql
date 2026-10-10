-- Migration: 083_onboarding_starter_apps
-- Create table for tracking starter app generation during OSA Build onboarding

CREATE TABLE IF NOT EXISTS onboarding_starter_apps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL,
    analysis_id UUID NOT NULL REFERENCES onboarding_user_analysis(id) ON DELETE CASCADE,

    -- App Details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    icon_emoji VARCHAR(10), -- Temporary emoji icon
    icon_url VARCHAR(500), -- URL to generated icon (if available)
    category VARCHAR(100), -- e.g., "tracker", "companion", "feedback", "daily"

    -- AI Customization
    reasoning TEXT, -- Why this app was recommended
    customization_prompt TEXT NOT NULL, -- Full prompt sent to AI for app generation
    based_on_interests JSONB DEFAULT '[]'::jsonb, -- Which user interests triggered this app
    based_on_tools JSONB DEFAULT '[]'::jsonb, -- Which tools triggered this app

    -- Generation Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, generating, ready, failed
    osa_workflow_id VARCHAR(255), -- OSA agent workflow ID for tracking
    error_message TEXT,

    -- Module Customization (if based on core module)
    base_module VARCHAR(100), -- e.g., "crm", "tasks", "projects" - null if completely new
    module_customizations JSONB DEFAULT '{}'::jsonb, -- Customizations applied to base module

    -- AI Provider Tracking
    generation_model VARCHAR(100), -- e.g., "llama-3.1-8b-instant"
    ai_provider VARCHAR(50), -- e.g., "groq", "anthropic"
    generation_tokens_used INTEGER DEFAULT 0,
    generation_duration_ms INTEGER,

    -- Display Order
    display_order INTEGER DEFAULT 0, -- Order shown in starter apps screen (1-4)

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Constraints
    UNIQUE(user_id, workspace_id, display_order)
);

-- Indexes for performance
CREATE INDEX idx_starter_apps_user ON onboarding_starter_apps(user_id);
CREATE INDEX idx_starter_apps_workspace ON onboarding_starter_apps(workspace_id);
CREATE INDEX idx_starter_apps_analysis ON onboarding_starter_apps(analysis_id);
CREATE INDEX idx_starter_apps_status ON onboarding_starter_apps(status);
CREATE INDEX idx_starter_apps_workflow ON onboarding_starter_apps(osa_workflow_id);
CREATE INDEX idx_starter_apps_display_order ON onboarding_starter_apps(workspace_id, display_order);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_starter_apps_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER starter_apps_updated_at
    BEFORE UPDATE ON onboarding_starter_apps
    FOR EACH ROW
    EXECUTE FUNCTION update_starter_apps_updated_at();

COMMENT ON TABLE onboarding_starter_apps IS 'Tracks personalized starter app generation during OSA Build onboarding';
COMMENT ON COLUMN onboarding_starter_apps.reasoning IS 'AI-generated explanation for why this app was recommended';
COMMENT ON COLUMN onboarding_starter_apps.customization_prompt IS 'Full AI prompt for generating this app';
COMMENT ON COLUMN onboarding_starter_apps.base_module IS 'Core BusinessOS module this app is based on (null for completely new apps)';
COMMENT ON COLUMN onboarding_starter_apps.display_order IS 'Order shown in starter apps screen (1-4 typically)';
