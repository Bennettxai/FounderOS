-- 162: the latest device-collector push per Mac, so a bridge restart keeps
-- WhatsApp/Wispr/Obsidian/local-stack/usage readings instead of reading
-- "not configured" until the next push. Bridge-native (not an ETL target).
CREATE TABLE IF NOT EXISTS founderos_device_pushes (
    device TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_founderos_device_pushes_workspace ON founderos_device_pushes(workspace_id);
