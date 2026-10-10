-- 140_workspace_storage.sql
-- Per-workspace cloud-sync storage accounting for the Knowledge module.
-- When a workspace's local knowledge is synced up (knowledge_documents), we
-- record how many bytes that workspace occupies and the quota it is allowed.
-- Free tier = 1 GB. Enforced at sync time: a sync that would push bytes_used
-- past bytes_limit is rejected (HTTP 413) and rolled back.
--
-- Additive only. Keyed by BusinessOS workspace_id (UUID), matching the
-- workspace_id retained on knowledge_documents. Idempotent (IF NOT EXISTS).
--
-- NOTE: numbered 140 (not 130) because 130_workspace_apps.sql already exists;
-- migrations apply in numeric order and 130-139 are taken.
CREATE TABLE IF NOT EXISTS workspace_storage (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    bytes_used   BIGINT NOT NULL DEFAULT 0,
    bytes_limit  BIGINT NOT NULL DEFAULT 1073741824,   -- 1 GB free tier
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
