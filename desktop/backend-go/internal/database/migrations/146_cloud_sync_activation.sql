-- 146_cloud_sync_activation.sql
-- Opt-in cloud sync gate for the Knowledge module.
--
-- Model: LOCAL knowledge (the user's own Optimal Engine) is always on and free.
-- CLOUD SYNC is the opt-in, paid-later layer. Nothing syncs to the cloud until
-- the user explicitly ACTIVATES it in the Knowledge module. This flag is the
-- server-side enforcement of that opt-in: SyncToCloud refuses to write for a
-- workspace whose row has cloud_sync_activated = false.
--
-- Additive only. Idempotent (IF NOT EXISTS). Defaults to false so existing
-- workspaces stay opt-out until the user activates.
--
-- NOTE: numbered 146 (not 141) because 141-145 are already taken; migrations
-- apply in numeric order.
ALTER TABLE workspace_storage
    ADD COLUMN IF NOT EXISTS cloud_sync_activated BOOLEAN NOT NULL DEFAULT false;
