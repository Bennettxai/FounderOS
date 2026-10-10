-- 096_sorx_skills.sql
-- Move SORX skill definitions from hardcoded Go maps to a database-driven table.
-- Skills have pgvector embeddings (nomic-embed-text, 768 dims) for semantic matching.
-- Admin can add/edit skills via DB without code redeploy.

-- Enable pgvector extension if not already enabled
CREATE EXTENSION IF NOT EXISTS vector;

-- sorx_skills: canonical skill registry for the SORX execution engine
CREATE TABLE IF NOT EXISTS sorx_skills (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,                          -- e.g. "gmail.sync"
    tier        TEXT        NOT NULL DEFAULT 'free'
                            CHECK (tier IN ('free', 'pro', 'enterprise')),
    description TEXT        NOT NULL,                                 -- human-readable, also used for embedding
    embedding   vector(768),                                          -- nomic-embed-text 768 dims
    config      JSONB       NOT NULL DEFAULT '{}',                   -- skill-specific config (auth scopes, params)
    enabled     BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- IVFFlat index for fast cosine similarity search (lists = 10 is fine for <1000 skills)
CREATE INDEX IF NOT EXISTS sorx_skills_embedding_idx
    ON sorx_skills USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);

-- Fast filter for enabled skills only
CREATE INDEX IF NOT EXISTS sorx_skills_enabled_name_idx
    ON sorx_skills (name) WHERE enabled = true;

-- Seed with the 11 built-in skills (embeddings populated later by SkillLoaderService)
-- embedding is NULL on insert — SkillLoaderService generates and updates on first load
INSERT INTO sorx_skills (name, tier, description, config) VALUES
    ('gmail.sync',       'free',       'Sync Gmail emails and import messages into the workspace', '{"scopes": ["gmail.readonly"]}'),
    ('gmail.send',       'free',       'Send an email via Gmail on behalf of the user',            '{"scopes": ["gmail.send"]}'),
    ('slack.send',       'free',       'Send a Slack message to a channel or user',                '{"scopes": ["chat:write"]}'),
    ('contacts.sync',    'free',       'Sync contacts from Google Contacts or CRM into workspace', '{"scopes": ["contacts.readonly"]}'),
    ('calendar.sync',    'free',       'Sync Google Calendar events into the workspace',            '{"scopes": ["calendar.readonly"]}'),
    ('notion.sync',      'pro',        'Import pages and databases from Notion into the workspace', '{}'),
    ('linear.sync',      'pro',        'Sync Linear issues and projects into the workspace',        '{}'),
    ('hubspot.sync',     'pro',        'Sync HubSpot CRM contacts and deals into the workspace',   '{}'),
    ('airtable.sync',    'pro',        'Import Airtable base records into the workspace',           '{}'),
    ('sheets.export',    'free',       'Export workspace data to a Google Sheets spreadsheet',     '{"scopes": ["spreadsheets"]}'),
    ('webhook.trigger',  'enterprise', 'Trigger a custom webhook URL with workspace event data',   '{}')
ON CONFLICT (name) DO NOTHING;

-- updated_at trigger
CREATE OR REPLACE FUNCTION update_sorx_skills_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sorx_skills_updated_at_trigger ON sorx_skills;
CREATE TRIGGER sorx_skills_updated_at_trigger
    BEFORE UPDATE ON sorx_skills
    FOR EACH ROW EXECUTE FUNCTION update_sorx_skills_updated_at();
