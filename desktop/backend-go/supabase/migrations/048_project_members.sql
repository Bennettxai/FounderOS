-- Migration 029: Project-Level Access Control
-- Description: Adds project-level member assignments and permissions

-- Create project_members table for project-level access control
CREATE TABLE IF NOT EXISTS project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- lead, contributor, reviewer, viewer
    can_edit BOOLEAN NOT NULL DEFAULT true,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    can_invite BOOLEAN NOT NULL DEFAULT false,
    assigned_by TEXT NOT NULL, -- user_id of who assigned
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    removed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active', -- active, inactive, removed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique membership per project
    CONSTRAINT unique_project_member UNIQUE (project_id, user_id),

    -- Validate role values
    CONSTRAINT check_project_role CHECK (role IN ('lead', 'contributor', 'reviewer', 'viewer')),

    -- Validate status values
    CONSTRAINT check_member_status CHECK (status IN ('active', 'inactive', 'removed'))
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_project_members_project_id ON project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_workspace_id ON project_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_project_members_status ON project_members(status);
CREATE INDEX IF NOT EXISTS idx_project_members_role ON project_members(role);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_project_members_project_user ON project_members(project_id, user_id);
CREATE INDEX IF NOT EXISTS idx_project_members_workspace_user ON project_members(workspace_id, user_id);

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_project_members_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-update updated_at
CREATE TRIGGER trigger_update_project_members_updated_at
    BEFORE UPDATE ON project_members
    FOR EACH ROW
    EXECUTE FUNCTION update_project_members_updated_at();

-- Create default project roles reference table (for documentation/validation)
CREATE TABLE IF NOT EXISTS project_role_definitions (
    role TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT,
    hierarchy_level INT NOT NULL, -- lower = more power
    default_can_edit BOOLEAN NOT NULL DEFAULT false,
    default_can_delete BOOLEAN NOT NULL DEFAULT false,
    default_can_invite BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default project roles
INSERT INTO project_role_definitions (role, display_name, description, hierarchy_level, default_can_edit, default_can_delete, default_can_invite)
VALUES
    ('lead', 'Project Lead', 'Full project control, can manage members and settings', 1, true, true, true),
    ('contributor', 'Contributor', 'Can edit and contribute to project', 2, true, false, false),
    ('reviewer', 'Reviewer', 'Can review and comment, limited editing', 3, false, false, false),
    ('viewer', 'Viewer', 'Read-only access to project', 4, false, false, false)
ON CONFLICT (role) DO NOTHING;

-- Function to check if user has project access
CREATE OR REPLACE FUNCTION has_project_access(
    p_user_id TEXT,
    p_project_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM project_members
        WHERE user_id = p_user_id
        AND project_id = p_project_id
        AND status = 'active'
        AND (removed_at IS NULL OR removed_at > NOW())
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get user's project role
CREATE OR REPLACE FUNCTION get_project_role(
    p_user_id TEXT,
    p_project_id UUID
)
RETURNS TEXT AS $$
DECLARE
    v_role TEXT;
BEGIN
    SELECT role INTO v_role
    FROM project_members
    WHERE user_id = p_user_id
    AND project_id = p_project_id
    AND status = 'active'
    AND (removed_at IS NULL OR removed_at > NOW());

    RETURN v_role;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's project permissions
CREATE OR REPLACE FUNCTION get_project_permissions(
    p_user_id TEXT,
    p_project_id UUID
)
RETURNS TABLE (
    can_edit BOOLEAN,
    can_delete BOOLEAN,
    can_invite BOOLEAN,
    role TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT pm.can_edit, pm.can_delete, pm.can_invite, pm.role
    FROM project_members pm
    WHERE pm.user_id = p_user_id
    AND pm.project_id = p_project_id
    AND pm.status = 'active'
    AND (pm.removed_at IS NULL OR pm.removed_at > NOW());
END;
$$ LANGUAGE plpgsql;

-- Trigger to log project member changes to audit log (if audit function exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'log_workspace_action') THEN
        CREATE OR REPLACE FUNCTION audit_project_member_changes()
        RETURNS TRIGGER AS $inner$
        BEGIN
            IF TG_OP = 'INSERT' THEN
                PERFORM log_workspace_action(
                    NEW.workspace_id,
                    NEW.assigned_by,
                    'add_project_member',
                    'project_member',
                    NEW.id::TEXT,
                    jsonb_build_object(
                        'project_id', NEW.project_id,
                        'user_id', NEW.user_id,
                        'role', NEW.role,
                        'can_edit', NEW.can_edit,
                        'can_delete', NEW.can_delete,
                        'can_invite', NEW.can_invite
                    )
                );
            ELSIF TG_OP = 'UPDATE' THEN
                PERFORM log_workspace_action(
                    NEW.workspace_id,
                    NEW.assigned_by,
                    'update_project_member',
                    'project_member',
                    NEW.id::TEXT,
                    jsonb_build_object(
                        'old_role', OLD.role,
                        'new_role', NEW.role,
                        'old_status', OLD.status,
                        'new_status', NEW.status
                    )
                );
            ELSIF TG_OP = 'DELETE' THEN
                PERFORM log_workspace_action(
                    OLD.workspace_id,
                    OLD.assigned_by,
                    'remove_project_member',
                    'project_member',
                    OLD.id::TEXT,
                    jsonb_build_object(
                        'project_id', OLD.project_id,
                        'user_id', OLD.user_id,
                        'role', OLD.role
                    )
                );
            END IF;
            RETURN NULL;
        END;
        $inner$ LANGUAGE plpgsql;

        -- Create trigger for audit logging
        DROP TRIGGER IF EXISTS trigger_audit_project_member_changes ON project_members;
        CREATE TRIGGER trigger_audit_project_member_changes
            AFTER INSERT OR UPDATE OR DELETE ON project_members
            FOR EACH ROW
            EXECUTE FUNCTION audit_project_member_changes();
    END IF;
END $$;

COMMENT ON TABLE project_members IS 'Project-level access control and member assignments';
COMMENT ON TABLE project_role_definitions IS 'Defines available project roles and their default permissions';
COMMENT ON COLUMN project_members.role IS 'Project role: lead, contributor, reviewer, viewer';
COMMENT ON COLUMN project_members.status IS 'Membership status: active, inactive, removed';
COMMENT ON FUNCTION has_project_access IS 'Check if user has access to a specific project';
COMMENT ON FUNCTION get_project_role IS 'Get user''s role in a specific project';
COMMENT ON FUNCTION get_project_permissions IS 'Get user''s permissions in a specific project';
