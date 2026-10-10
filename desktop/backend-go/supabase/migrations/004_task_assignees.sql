-- Migration 004: Multiple assignees per task
-- Date: 2025-12-21
-- Description: Adds support for assigning multiple team members to a single task

-- Create task_assignees junction table
CREATE TABLE IF NOT EXISTS task_assignees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'assignee', -- 'assignee', 'reviewer', 'lead'
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assigned_by VARCHAR(255),
    UNIQUE(task_id, team_member_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_task_assignees_task ON task_assignees(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_member ON task_assignees(team_member_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_role ON task_assignees(role);

-- Migrate existing single assignee data to new junction table
-- This preserves existing assignments
INSERT INTO task_assignees (task_id, team_member_id, role, assigned_at)
SELECT id, assignee_id, 'assignee', updated_at
FROM tasks
WHERE assignee_id IS NOT NULL
ON CONFLICT (task_id, team_member_id) DO NOTHING;
