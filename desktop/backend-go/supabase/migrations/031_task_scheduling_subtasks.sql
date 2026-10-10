-- Migration: Add task scheduling and subtask columns
-- These columns were added in schema.sql but missing from the original supabase_migration.sql

-- Task scheduling fields
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS start_date TIMESTAMP;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS custom_status_id UUID;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS position INT DEFAULT 0;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(user_id, position);
