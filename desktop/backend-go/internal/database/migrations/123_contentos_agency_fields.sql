-- 123_contentos_agency_fields.sql
-- Expands the Content module into an agency-grade ContentOS board.
-- These fields keep the Notion-style production pipeline while adding the
-- client, campaign, editor, review, and asset context an agency needs.

ALTER TABLE content_items ADD COLUMN IF NOT EXISTS category     VARCHAR(80)  DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS client       VARCHAR(180) DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS campaign     VARCHAR(180) DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS owner        VARCHAR(180) DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS editor       VARCHAR(180) DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS priority     VARCHAR(40)  DEFAULT 'normal';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS due_date     VARCHAR(20)  DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS publish_date VARCHAR(20)  DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS asset_link   TEXT         DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS review_link  TEXT         DEFAULT '';
ALTER TABLE content_items ADD COLUMN IF NOT EXISTS notes        TEXT         DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_content_items_workspace_status
    ON content_items(workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_content_items_workspace_client
    ON content_items(workspace_id, client);
