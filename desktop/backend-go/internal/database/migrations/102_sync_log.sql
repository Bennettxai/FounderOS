-- Migration: 102_sync_log.sql
-- Description: Sync log table for tracking bidirectional data sync between
--              local BusinessOS clients and cloud/device instances.
-- Created: 2026-04-11

CREATE TABLE IF NOT EXISTS sync_log (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id  VARCHAR(255) NOT NULL,
    direction  VARCHAR(10)  NOT NULL CHECK (direction IN ('push', 'pull')),
    table_name VARCHAR(255) NOT NULL,
    record_id  VARCHAR(255),
    action     VARCHAR(10)  NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    synced_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    metadata   JSONB
);

CREATE INDEX IF NOT EXISTS idx_sync_log_device ON sync_log(device_id, synced_at);
