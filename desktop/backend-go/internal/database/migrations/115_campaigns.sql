-- 115_campaigns.sql
-- The Campaigns module: a per-workspace registry of marketing/outreach campaigns
-- (email, ads, sms, organic) with lifecycle status. Workspace-scoped.

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(300) NOT NULL,
    channel VARCHAR(40) DEFAULT 'email',     -- email|ads|sms|organic|other
    status VARCHAR(40) DEFAULT 'draft',      -- draft|active|paused|done
    description TEXT DEFAULT '',
    start_date DATE,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_workspace ON campaigns(workspace_id);
