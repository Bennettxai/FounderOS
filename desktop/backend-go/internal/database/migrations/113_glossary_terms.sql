-- 113_glossary_terms.sql
-- The Glossary module: a per-workspace dictionary of the business's own terms
-- (e.g. what "proposal" means for us), so humans AND AI agents decode language
-- consistently. Workspace-scoped; one canonical term per workspace.

CREATE TABLE IF NOT EXISTS glossary_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    term VARCHAR(300) NOT NULL,
    definition TEXT NOT NULL DEFAULT '',
    category VARCHAR(120),
    aliases TEXT,                       -- comma-separated alt names / variants
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, term)
);

CREATE INDEX IF NOT EXISTS idx_glossary_terms_workspace ON glossary_terms(workspace_id);
CREATE INDEX IF NOT EXISTS idx_glossary_terms_term ON glossary_terms(workspace_id, term);
