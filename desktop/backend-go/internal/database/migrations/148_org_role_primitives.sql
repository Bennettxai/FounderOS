-- 148_org_role_primitives.sql
-- Canonical backend-side org + role primitives that older Supabase migrations
-- created outside internal/database/migrations. Fresh CI/dev databases need
-- these before the runtime schema contract can pass.

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    owner_id VARCHAR(255),
    logo_url VARCHAR(500),
    settings JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS owner_id VARCHAR(255);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500);
CREATE INDEX IF NOT EXISTS idx_organizations_owner ON organizations(owner_id);

CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    invited_by VARCHAR(255),
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'organization_members'
          AND column_name = 'org_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'organization_members'
          AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE organization_members RENAME COLUMN org_id TO organization_id;
    END IF;
END $$;

ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS invited_by VARCHAR(255);
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
CREATE UNIQUE INDEX IF NOT EXISTS uq_org_members_org_user ON organization_members(organization_id, user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);

CREATE TABLE IF NOT EXISTS organization_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    invited_by VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_invites_org ON organization_invites(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_invites_token ON organization_invites(token);

ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_workspaces_org ON workspaces(organization_id);

CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    permission VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, role, resource, permission)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_workspace ON role_permissions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(workspace_id, role);

CREATE OR REPLACE FUNCTION seed_default_workspace_roles(p_workspace_id UUID)
RETURNS void AS $$
BEGIN
    INSERT INTO workspace_roles (
        workspace_id,
        name,
        display_name,
        description,
        color,
        icon,
        is_system,
        is_default,
        hierarchy_level,
        permissions
    )
    VALUES
        (
            p_workspace_id,
            'owner',
            'Owner',
            'Full workspace access including billing and deletion',
            '#8B5CF6',
            'crown',
            true,
            false,
            1,
            '{"projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": true, "share": true}, "workspace": {"invite_members": true, "manage_roles": true, "manage_billing": true, "delete_workspace": true}, "agents": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
        ),
        (
            p_workspace_id,
            'admin',
            'Admin',
            'Full access except billing and workspace deletion',
            '#3B82F6',
            'shield',
            true,
            false,
            2,
            '{"projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": true, "share": true}, "workspace": {"invite_members": true, "manage_roles": true, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
        ),
        (
            p_workspace_id,
            'manager',
            'Manager',
            'Can manage projects, tasks, and team members',
            '#10B981',
            'users',
            true,
            false,
            3,
            '{"projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": true}, "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true}, "contexts": {"create": true, "read": true, "update": true, "delete": false, "share": true}, "workspace": {"invite_members": true, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": true}}'::jsonb
        ),
        (
            p_workspace_id,
            'member',
            'Member',
            'Standard team member - can create and manage own work',
            '#6366F1',
            'user',
            true,
            true,
            4,
            '{"projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": false}, "tasks": {"create": true, "read": true, "update": true, "delete": false, "assign": false}, "contexts": {"create": true, "read": true, "update": true, "delete": false, "share": true}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}}'::jsonb
        ),
        (
            p_workspace_id,
            'viewer',
            'Viewer',
            'Read-only access to workspace resources',
            '#64748B',
            'eye',
            true,
            false,
            5,
            '{"projects": {"create": false, "read": true, "update": false, "delete": false, "manage_members": false}, "tasks": {"create": false, "read": true, "update": false, "delete": false, "assign": false}, "contexts": {"create": false, "read": true, "update": false, "delete": false, "share": false}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}}'::jsonb
        ),
        (
            p_workspace_id,
            'guest',
            'Guest',
            'Limited access to specific projects only',
            '#94A3B8',
            'user-plus',
            true,
            false,
            6,
            '{"projects": {"create": false, "read": true, "update": false, "delete": false, "manage_members": false}, "tasks": {"create": false, "read": true, "update": false, "delete": false, "assign": false}, "contexts": {"create": false, "read": true, "update": false, "delete": false, "share": false}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": false, "create_custom_agents": false, "access_workspace_memory": false, "modify_workspace_memory": false}}'::jsonb
        )
    ON CONFLICT (workspace_id, name)
    DO UPDATE SET
        display_name = EXCLUDED.display_name,
        description = EXCLUDED.description,
        color = EXCLUDED.color,
        icon = EXCLUDED.icon,
        is_system = EXCLUDED.is_system,
        is_default = EXCLUDED.is_default,
        hierarchy_level = EXCLUDED.hierarchy_level,
        permissions = EXCLUDED.permissions,
        updated_at = NOW();

    DELETE FROM role_permissions WHERE workspace_id = p_workspace_id;

    INSERT INTO role_permissions (workspace_id, role, resource, permission, metadata)
    SELECT
        wr.workspace_id,
        wr.name AS role,
        perm.key AS resource,
        perm_detail.key AS permission,
        jsonb_build_object('allowed', perm_detail.value) AS metadata
    FROM workspace_roles wr,
         jsonb_each(wr.permissions) AS perm,
         jsonb_each(perm.value) AS perm_detail
    WHERE wr.workspace_id = p_workspace_id
      AND perm_detail.value::text = 'true'
    ON CONFLICT (workspace_id, role, resource, permission)
    DO UPDATE SET metadata = EXCLUDED.metadata, updated_at = NOW();
END;
$$ LANGUAGE plpgsql;
