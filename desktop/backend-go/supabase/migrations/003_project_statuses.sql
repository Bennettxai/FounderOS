-- Migration 003: Custom project statuses
-- Date: 2025-12-21
-- Description: Adds customizable statuses per project for tasks

-- Create project_statuses table
CREATE TABLE IF NOT EXISTS project_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#6B7280',
    position INTEGER NOT NULL DEFAULT 0,
    is_done_state BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, name)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_project_statuses_project_id ON project_statuses(project_id);
CREATE INDEX IF NOT EXISTS idx_project_statuses_position ON project_statuses(project_id, position);

-- Add custom_status_id column to tasks
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS custom_status_id UUID REFERENCES project_statuses(id) ON DELETE SET NULL;

-- Create index for custom_status_id
CREATE INDEX IF NOT EXISTS idx_tasks_custom_status ON tasks(custom_status_id);

-- Function to create default statuses for a project
CREATE OR REPLACE FUNCTION create_default_project_statuses(p_project_id UUID) RETURNS VOID AS $$
BEGIN
    INSERT INTO project_statuses (project_id, name, color, position, is_done_state, is_default)
    VALUES
        (p_project_id, 'To Do', '#6B7280', 0, FALSE, TRUE),
        (p_project_id, 'In Progress', '#3B82F6', 1, FALSE, FALSE),
        (p_project_id, 'In Review', '#8B5CF6', 2, FALSE, FALSE),
        (p_project_id, 'Done', '#10B981', 3, TRUE, FALSE),
        (p_project_id, 'Blocked', '#F59E0B', 4, FALSE, FALSE)
    ON CONFLICT (project_id, name) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Trigger function to auto-create default statuses on project creation
CREATE OR REPLACE FUNCTION auto_create_project_statuses() RETURNS TRIGGER AS $$
BEGIN
    PERFORM create_default_project_statuses(NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for new projects
DROP TRIGGER IF EXISTS create_project_statuses_trigger ON projects;
CREATE TRIGGER create_project_statuses_trigger
    AFTER INSERT ON projects
    FOR EACH ROW EXECUTE FUNCTION auto_create_project_statuses();

-- Create default statuses for existing projects that don't have any
DO $$
DECLARE
    proj_id UUID;
BEGIN
    FOR proj_id IN
        SELECT p.id FROM projects p
        WHERE NOT EXISTS (
            SELECT 1 FROM project_statuses ps WHERE ps.project_id = p.id
        )
    LOOP
        PERFORM create_default_project_statuses(proj_id);
    END LOOP;
END;
$$;
