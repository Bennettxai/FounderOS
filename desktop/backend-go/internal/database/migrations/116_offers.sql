-- 116_offers.sql
-- The Offers module: a per-workspace catalog of the business's productized offers
-- (name, price, status, what's included), so the team and AI agents share one
-- source of truth for what we sell. Workspace-scoped.

CREATE TABLE IF NOT EXISTS offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    price VARCHAR(120) DEFAULT '',
    status VARCHAR(40) DEFAULT 'active',   -- active | draft | archived
    description TEXT DEFAULT '',
    includes TEXT DEFAULT '',              -- what's included (free text / list)
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_offers_workspace ON offers(workspace_id);
