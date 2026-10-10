-- Migration 028: Workspace Audit Logging
-- Description: Adds comprehensive audit logging for workspace activities

-- Create workspace_audit_logs table
CREATE TABLE IF NOT EXISTS workspace_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL, -- user who performed the action
    action TEXT NOT NULL, -- action type (e.g., create, update, delete, invite, remove_member)
    resource_type TEXT NOT NULL, -- what was affected (workspace, member, role, project, etc.)
    resource_id TEXT, -- ID of the affected resource
    details JSONB, -- additional context about the action
    ip_address TEXT, -- IP address of the user
    user_agent TEXT, -- User agent string
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace_id ON workspace_audit_logs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON workspace_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON workspace_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON workspace_audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_id ON workspace_audit_logs(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON workspace_audit_logs(created_at DESC);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace_user ON workspace_audit_logs(workspace_id, user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace_action ON workspace_audit_logs(workspace_id, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_workspace_resource ON workspace_audit_logs(workspace_id, resource_type, resource_id);

-- GIN index for JSONB details field for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_details ON workspace_audit_logs USING gin(details);

-- Function to easily create audit log entries
CREATE OR REPLACE FUNCTION log_workspace_action(
    p_workspace_id UUID,
    p_user_id TEXT,
    p_action TEXT,
    p_resource_type TEXT,
    p_resource_id TEXT DEFAULT NULL,
    p_details JSONB DEFAULT NULL,
    p_ip_address TEXT DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
BEGIN
    INSERT INTO workspace_audit_logs (
        workspace_id,
        user_id,
        action,
        resource_type,
        resource_id,
        details,
        ip_address,
        user_agent
    ) VALUES (
        p_workspace_id,
        p_user_id,
        p_action,
        p_resource_type,
        p_resource_id,
        p_details,
        p_ip_address,
        p_user_agent
    )
    RETURNING id INTO v_log_id;

    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;

-- Create trigger function to auto-log workspace changes
CREATE OR REPLACE FUNCTION audit_workspace_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM log_workspace_action(
            NEW.id,
            NEW.owner_id,
            'create',
            'workspace',
            NEW.id::TEXT,
            jsonb_build_object(
                'name', NEW.name,
                'plan_type', NEW.plan_type
            )
        );
    ELSIF TG_OP = 'UPDATE' THEN
        PERFORM log_workspace_action(
            NEW.id,
            NEW.owner_id,
            'update',
            'workspace',
            NEW.id::TEXT,
            jsonb_build_object(
                'old', jsonb_build_object('name', OLD.name, 'plan_type', OLD.plan_type),
                'new', jsonb_build_object('name', NEW.name, 'plan_type', NEW.plan_type)
            )
        );
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM log_workspace_action(
            OLD.id,
            OLD.owner_id,
            'delete',
            'workspace',
            OLD.id::TEXT,
            jsonb_build_object(
                'name', OLD.name
            )
        );
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for workspace changes
CREATE TRIGGER trigger_audit_workspace_changes
    AFTER INSERT OR UPDATE OR DELETE ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION audit_workspace_changes();

-- Create trigger function to auto-log member changes
CREATE OR REPLACE FUNCTION audit_member_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM log_workspace_action(
            NEW.workspace_id,
            COALESCE(NEW.invited_by, NEW.user_id),
            'add_member',
            'member',
            NEW.id::TEXT,
            jsonb_build_object(
                'user_id', NEW.user_id,
                'role', NEW.role,
                'status', NEW.status
            )
        );
    ELSIF TG_OP = 'UPDATE' THEN
        PERFORM log_workspace_action(
            NEW.workspace_id,
            NEW.user_id,
            'update_member',
            'member',
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
            OLD.user_id,
            'remove_member',
            'member',
            OLD.id::TEXT,
            jsonb_build_object(
                'user_id', OLD.user_id,
                'role', OLD.role
            )
        );
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for member changes
CREATE TRIGGER trigger_audit_member_changes
    AFTER INSERT OR UPDATE OR DELETE ON workspace_members
    FOR EACH ROW
    EXECUTE FUNCTION audit_member_changes();

COMMENT ON TABLE workspace_audit_logs IS 'Comprehensive audit trail for all workspace activities';
COMMENT ON COLUMN workspace_audit_logs.action IS 'Type of action performed (create, update, delete, invite, etc.)';
COMMENT ON COLUMN workspace_audit_logs.resource_type IS 'Type of resource affected (workspace, member, role, project, etc.)';
COMMENT ON COLUMN workspace_audit_logs.details IS 'JSONB field containing additional context and metadata';
COMMENT ON FUNCTION log_workspace_action IS 'Helper function to create audit log entries programmatically';
