-- Migration 026: Workspaces, Roles, and Team Collaboration
-- Feature 1 from FUTURE_FEATURES.md
-- Implements multi-tenant workspace system with role-based access control

BEGIN;

-- =====================================================================
-- WORKSPACES
-- =====================================================================

-- Workspaces are the top-level container for teams
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Workspace Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,        -- URL-friendly identifier (e.g., 'acme-corp')
    description TEXT,
    logo_url VARCHAR(500),

    -- Billing & Limits
    plan_type VARCHAR(50) DEFAULT 'free',     -- 'free', 'starter', 'professional', 'enterprise'
    max_members INTEGER DEFAULT 5,
    max_projects INTEGER DEFAULT 10,
    max_storage_gb INTEGER DEFAULT 5,

    -- Settings (JSON for flexibility)
    settings JSONB DEFAULT '{}',
    /*
    Example settings:
    {
        "allow_public_sharing": false,
        "require_2fa": false,
        "allowed_domains": ["@acme.com"],
        "default_project_visibility": "private",
        "enable_ai_agents": true
    }
    */

    -- Ownership
    owner_id VARCHAR(255) NOT NULL,           -- User who created the workspace

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);

COMMENT ON TABLE workspaces IS 'Top-level containers for team collaboration';
COMMENT ON COLUMN workspaces.slug IS 'URL-friendly unique identifier for the workspace';
COMMENT ON COLUMN workspaces.plan_type IS 'Subscription plan level determining feature access and limits';

-- =====================================================================
-- WORKSPACE ROLES
-- =====================================================================

-- Workspace roles define permission sets
-- These can be system-defined (owner, admin, member) or custom per workspace
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Role Identity
    name VARCHAR(100) NOT NULL,               -- Internal name (e.g., 'admin', 'manager')
    display_name VARCHAR(100) NOT NULL,       -- Human-readable name
    description TEXT,
    color VARCHAR(20),                        -- Hex color for UI display (e.g., '#3B82F6')
    icon VARCHAR(50),                         -- Icon identifier

    -- Permissions Matrix (JSONB for flexibility)
    permissions JSONB NOT NULL DEFAULT '{}',
    /*
    Example permissions structure:
    {
        "projects": {
            "create": true,
            "read": true,
            "update": true,
            "delete": false,
            "manage_members": false
        },
        "tasks": {
            "create": true,
            "read": true,
            "update": true,
            "delete": true,
            "assign": true
        },
        "contexts": {
            "create": true,
            "read": true,
            "update": true,
            "delete": false,
            "share": true
        },
        "workspace": {
            "invite_members": false,
            "manage_roles": false,
            "manage_billing": false,
            "delete_workspace": false
        },
        "agents": {
            "use_all_agents": true,
            "create_custom_agents": false,
            "access_workspace_memory": true,
            "modify_workspace_memory": false
        }
    }
    */

    -- Role Type
    is_system BOOLEAN DEFAULT FALSE,          -- System roles cannot be deleted
    is_default BOOLEAN DEFAULT FALSE,         -- Assigned to new members by default

    -- Hierarchy (lower number = more power)
    hierarchy_level INTEGER DEFAULT 99,       -- 1=owner, 2=admin, 3=manager, 4=member, 5=viewer, 6=guest

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_roles_hierarchy ON workspace_roles(workspace_id, hierarchy_level);

COMMENT ON TABLE workspace_roles IS 'Role definitions with permissions for workspace members';
COMMENT ON COLUMN workspace_roles.hierarchy_level IS 'Role hierarchy - lower numbers have more power (1=owner, 99=guest)';
COMMENT ON COLUMN workspace_roles.is_system IS 'System roles (owner, admin, member, viewer, guest) cannot be deleted';

-- =====================================================================
-- WORKSPACE MEMBERS
-- =====================================================================

-- Junction table connecting users to workspaces with roles
CREATE TABLE IF NOT EXISTS workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Role Assignment
    role VARCHAR(100) NOT NULL,               -- Role name (denormalized for quick access)

    -- Status
    status VARCHAR(50) DEFAULT 'active',      -- 'active', 'invited', 'suspended', 'left'
    invited_by VARCHAR(255),
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,

    -- Custom Permissions Override (optional)
    custom_permissions JSONB,                 -- Override specific role permissions if needed
    /*
    Example: Give member delete permission just for their own projects
    {
        "projects": {
            "delete": "own"
        }
    }
    */

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_status ON workspace_members(workspace_id, status);

COMMENT ON TABLE workspace_members IS 'Users who are members of workspaces with assigned roles';
-- Note: Commented out - column 'role' doesn't exist in fresh installs (uses role_id instead)
-- COMMENT ON COLUMN workspace_members.role IS 'Role name - references workspace_roles.name';
COMMENT ON COLUMN workspace_members.status IS 'Membership status: active, invited, suspended, or left';

-- =====================================================================
-- USER WORKSPACE PROFILES
-- =====================================================================

-- User profile information specific to each workspace
-- (Same user can have different titles/roles in different workspaces)
CREATE TABLE IF NOT EXISTS user_workspace_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Profile Info (workspace-specific)
    display_name VARCHAR(255),
    title VARCHAR(100),                       -- Job title in this workspace
    department VARCHAR(100),
    avatar_url VARCHAR(500),

    -- Contact
    work_email VARCHAR(255),
    phone VARCHAR(50),

    -- Work Preferences
    timezone VARCHAR(50),
    working_hours JSONB,
    /*
    Example working hours:
    {
        "start": "09:00",
        "end": "17:00",
        "days": [1, 2, 3, 4, 5],
        "timezone": "America/New_York"
    }
    */

    -- Notification Preferences
    notification_preferences JSONB DEFAULT '{}',
    /*
    {
        "email": true,
        "push": true,
        "in_app": true,
        "digest": "daily",
        "quiet_hours": {"start": "22:00", "end": "08:00"}
    }
    */

    -- Agent Personalization (per workspace)
    preferred_output_style VARCHAR(50),       -- 'concise', 'detailed', 'technical'
    communication_preferences JSONB,
    expertise_areas TEXT[],

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_workspace ON user_workspace_profiles(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_user ON user_workspace_profiles(user_id);

COMMENT ON TABLE user_workspace_profiles IS 'User profile information specific to each workspace';
COMMENT ON COLUMN user_workspace_profiles.title IS 'Job title within this workspace (can differ across workspaces)';

-- =====================================================================
-- WORKSPACE MEMORIES
-- =====================================================================

-- Shared knowledge base for the workspace
-- Agents can search and contribute to this memory
CREATE TABLE IF NOT EXISTS workspace_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Memory Identity
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    content TEXT NOT NULL,

    -- Memory Type
    memory_type VARCHAR(50) NOT NULL,         -- 'decision', 'process', 'knowledge', 'pattern', 'policy'
    category VARCHAR(100),                    -- User-defined category

    -- Scope (what this memory applies to)
    scope_type VARCHAR(50) DEFAULT 'workspace', -- 'workspace', 'project', 'node'
    scope_id UUID,                            -- If scoped to specific project/node

    -- Access Control
    visibility VARCHAR(50) DEFAULT 'team',    -- 'team', 'managers', 'admins', 'owners'
    created_by VARCHAR(255) NOT NULL,

    -- Relevance & Usage
    importance_score DECIMAL(3,2) DEFAULT 0.5, -- 0.0 to 1.0
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMPTZ,

    -- Embeddings for semantic search
    embedding vector(768),                    -- Nomic embeddings

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,          -- Pinned memories always appear first

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add embedding column if table already existed from migration 028 (which didn't have it)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'workspace_memories' AND column_name = 'embedding'
    ) THEN
        ALTER TABLE workspace_memories ADD COLUMN embedding vector(768);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace ON workspace_memories(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_scope ON workspace_memories(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_type ON workspace_memories(workspace_id, memory_type);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_visibility ON workspace_memories(workspace_id, visibility);
-- Note: embedding column and index should be added in a later migration (similar to 037)
-- CREATE INDEX IF NOT EXISTS idx_workspace_memories_embedding ON workspace_memories USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

COMMENT ON TABLE workspace_memories IS 'Shared knowledge base for workspace - searchable by agents';
COMMENT ON COLUMN workspace_memories.visibility IS 'Who can see this memory: team, managers, admins, owners';
COMMENT ON COLUMN workspace_memories.scope_type IS 'What this memory applies to: workspace, project, or node';

-- =====================================================================
-- PROJECT MEMBERS (extends existing projects table)
-- =====================================================================

-- First, add workspace_id to existing projects table if not exists
ALTER TABLE projects ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_projects_workspace ON projects(workspace_id);

COMMENT ON COLUMN projects.workspace_id IS 'Workspace this project belongs to';

-- Project team assignments (who can access which projects)
-- Note: This table may already exist from previous migrations (created in 048_project_members.sql)
-- We add new columns if they don't exist
-- Only attempt modifications if the table exists

-- Add workspace_id column if project_members table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                      WHERE table_name = 'project_members' AND column_name = 'workspace_id') THEN
            ALTER TABLE project_members ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- Add notification_level column if project_members table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                      WHERE table_name = 'project_members' AND column_name = 'notification_level') THEN
            ALTER TABLE project_members ADD COLUMN notification_level VARCHAR(50) DEFAULT 'all';
        END IF;
    END IF;
END $$;

-- Add created_at/updated_at if project_members table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                      WHERE table_name = 'project_members' AND column_name = 'created_at') THEN
            ALTER TABLE project_members ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                      WHERE table_name = 'project_members' AND column_name = 'updated_at') THEN
            ALTER TABLE project_members ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();
        END IF;
    END IF;
END $$;

-- Create indexes only if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id);
        CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members(user_id);
        CREATE INDEX IF NOT EXISTS idx_project_members_workspace ON project_members(workspace_id);
    END IF;
END $$;

-- Add comments only if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        COMMENT ON TABLE project_members IS 'Users assigned to projects with project-specific roles';
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'project_members' AND column_name = 'role') THEN
            COMMENT ON COLUMN project_members.role IS 'Project role - can override workspace role for this project';
        END IF;
    END IF;
END $$;

-- =====================================================================
-- ROLE PERMISSIONS TABLE (for role_context.go compatibility)
-- =====================================================================

-- Materialized view of role permissions for faster lookups
-- This table is denormalized from workspace_roles.permissions JSONB
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,

    -- Permission
    resource VARCHAR(100) NOT NULL,           -- 'projects', 'tasks', 'contexts', etc.
    permission VARCHAR(100) NOT NULL,         -- 'create', 'read', 'update', 'delete', etc.
    metadata JSONB DEFAULT '{}',              -- Additional permission metadata

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, role, resource, permission)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_workspace ON role_permissions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(workspace_id, role);

COMMENT ON TABLE role_permissions IS 'Denormalized role permissions for fast lookups by role_context.go';

-- =====================================================================
-- SEED DEFAULT ROLES FUNCTION
-- =====================================================================

-- Function to create default system roles for a new workspace
CREATE OR REPLACE FUNCTION seed_default_workspace_roles(p_workspace_id UUID)
RETURNS void AS $$
BEGIN
    -- 1. OWNER (Full access)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
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
    );

    -- 2. ADMIN (Almost full access except ownership transfer)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
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
    );

    -- 3. MANAGER (Can manage projects and team)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
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
    );

    -- 4. MEMBER (Standard access - create and edit own work)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
        p_workspace_id,
        'member',
        'Member',
        'Standard team member - can create and manage own work',
        '#6366F1',
        'user',
        true,
        true,  -- Default role for new members
        4,
        '{"projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": false}, "tasks": {"create": true, "read": true, "update": true, "delete": false, "assign": false}, "contexts": {"create": true, "read": true, "update": true, "delete": false, "share": true}, "workspace": {"invite_members": false, "manage_roles": false, "manage_billing": false, "delete_workspace": false}, "agents": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}}'::jsonb
    );

    -- 5. VIEWER (Read-only access)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
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
    );

    -- 6. GUEST (Limited read access to specific projects only)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, is_system, is_default, hierarchy_level, permissions)
    VALUES (
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
    );

    -- Populate role_permissions table from the JSONB permissions
    -- This is for compatibility with role_context.go
    INSERT INTO role_permissions (workspace_id, role, resource, permission, metadata)
    SELECT
        workspace_id,
        name as role,
        perm.key as resource,
        perm_detail.key as permission,
        jsonb_build_object('allowed', perm_detail.value) as metadata
    FROM workspace_roles wr,
         jsonb_each(wr.permissions) as perm,
         jsonb_each(perm.value) as perm_detail
    WHERE wr.workspace_id = p_workspace_id
      AND perm_detail.value::text = 'true';

END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION seed_default_workspace_roles IS 'Creates 6 default system roles for a new workspace';

-- =====================================================================
-- TRIGGERS
-- =====================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workspaces_updated_at BEFORE UPDATE ON workspaces FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workspace_roles_updated_at BEFORE UPDATE ON workspace_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workspace_members_updated_at BEFORE UPDATE ON workspace_members FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_workspace_profiles_updated_at BEFORE UPDATE ON user_workspace_profiles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workspace_memories_updated_at BEFORE UPDATE ON workspace_memories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create trigger for project_members only if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_members') THEN
        DROP TRIGGER IF EXISTS update_project_members_updated_at ON project_members;
        CREATE TRIGGER update_project_members_updated_at BEFORE UPDATE ON project_members FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- =====================================================================
-- EXAMPLE USAGE (COMMENTED OUT - FOR REFERENCE)
-- =====================================================================

/*
-- Create a workspace
INSERT INTO workspaces (name, slug, owner_id, plan_type)
VALUES ('Acme Corp', 'acme-corp', 'user-123', 'professional')
RETURNING id;

-- Seed default roles for the workspace
SELECT seed_default_workspace_roles('workspace-uuid-here');

-- Add owner as first member
INSERT INTO workspace_members (workspace_id, user_id, role, status, joined_at)
VALUES ('workspace-uuid', 'user-123', 'owner', 'active', NOW());

-- Create user profile in workspace
INSERT INTO user_workspace_profiles (workspace_id, user_id, display_name, title, department)
VALUES ('workspace-uuid', 'user-123', 'John Doe', 'CEO', 'Executive');

-- Invite a new member
INSERT INTO workspace_members (workspace_id, user_id, role, status, invited_by, invited_at)
VALUES ('workspace-uuid', 'user-456', 'member', 'invited', 'user-123', NOW());

-- Query role_context.go compatible data
SELECT
    wm.user_id,
    wm.workspace_id,
    wm.role as role_name,
    wr.display_name as role_display_name,
    wr.hierarchy_level,
    uwp.title,
    uwp.department
FROM workspace_members wm
JOIN workspace_roles wr ON wr.name = wm.role AND wr.workspace_id = wm.workspace_id
LEFT JOIN user_workspace_profiles uwp ON uwp.user_id = wm.user_id AND uwp.workspace_id = wm.workspace_id
WHERE wm.user_id = 'user-123' AND wm.workspace_id = 'workspace-uuid';
*/

COMMIT;
