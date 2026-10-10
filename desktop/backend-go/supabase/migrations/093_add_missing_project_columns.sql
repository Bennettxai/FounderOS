-- Migration 093: Add missing project columns
-- The sqlc queries reference start_date, due_date, completed_at, visibility, owner_id
-- but these columns don't exist in the projects table.

ALTER TABLE projects ADD COLUMN IF NOT EXISTS start_date date;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS due_date date;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS completed_at timestamp without time zone;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS visibility varchar(50) DEFAULT 'private';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS owner_id varchar(255);
