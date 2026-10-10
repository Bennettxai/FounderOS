-- Migration 002: Add subtask support to tasks table
-- Date: 2025-12-21
-- Description: Adds parent_task_id, position, start_date, and estimated_hours columns to tasks table

-- Add new columns to tasks table
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS position INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS start_date TIMESTAMP,
ADD COLUMN IF NOT EXISTS estimated_hours NUMERIC(10, 2);

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Create trigger function to prevent nested subtasks (max 1 level deep)
CREATE OR REPLACE FUNCTION check_subtask_nesting() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_task_id IS NOT NULL THEN
        -- Check if the parent task is itself a subtask
        IF EXISTS (SELECT 1 FROM tasks WHERE id = NEW.parent_task_id AND parent_task_id IS NOT NULL) THEN
            RAISE EXCEPTION 'Subtasks cannot have nested subtasks (max 1 level deep)';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger (drop first if exists to allow re-running migration)
DROP TRIGGER IF EXISTS enforce_subtask_nesting ON tasks;
CREATE TRIGGER enforce_subtask_nesting
    BEFORE INSERT OR UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION check_subtask_nesting();

-- Update existing tasks to have position 0 where null
UPDATE tasks SET position = 0 WHERE position IS NULL;
