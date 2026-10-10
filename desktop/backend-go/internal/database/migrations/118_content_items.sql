-- 118_content_items.sql
-- The Content module: a per-workspace pipeline of content the business is making
-- (posts, reels, newsletters, podcasts, threads, articles) tracked through its
-- lifecycle (idea -> draft -> scheduled -> published). Workspace-scoped.

CREATE TABLE IF NOT EXISTS content_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    title VARCHAR(300) NOT NULL,
    content_type VARCHAR(40) DEFAULT 'post',   -- post|reel|newsletter|podcast|thread|article|other
    status VARCHAR(40) DEFAULT 'idea',         -- idea|draft|scheduled|published
    link TEXT DEFAULT '',
    body TEXT DEFAULT '',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_items_workspace ON content_items(workspace_id);
