-- ============================================================================
-- Migration: 020_workspaces.sql
-- Description: Workspace & Team Collaboration System
-- Created: 2026-01-07
-- Author: Javaris
-- ============================================================================

-- ============================================================================
-- WORKSPACES TABLE
-- Multi-tenant containers for team collaboration
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Workspace Identity
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    logo_url VARCHAR(500),

    -- Billing & Limits
    plan_type VARCHAR(50) DEFAULT 'free' CHECK (plan_type IN ('free', 'starter', 'professional', 'enterprise')),
    max_members INTEGER DEFAULT 5,
    max_projects INTEGER DEFAULT 10,
    max_storage_gb INTEGER DEFAULT 5,

    -- Settings
    settings JSONB DEFAULT '{}',

    -- Owner
    owner_id VARCHAR(255) NOT NULL,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);

-- ============================================================================
-- WORKSPACE ROLES TABLE
-- Customizable roles per workspace with permissions matrix
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Role Identity
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(20),
    icon VARCHAR(50),

    -- Permissions Matrix (JSONB)
    permissions JSONB NOT NULL DEFAULT '{}',

    -- Role Type
    is_system BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,

    -- Hierarchy (higher = more permissions)
    hierarchy_level INTEGER DEFAULT 0 CHECK (hierarchy_level >= 0 AND hierarchy_level <= 100),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);

-- ============================================================================
-- WORKSPACE MEMBERS TABLE
-- Links users to workspaces with role assignments
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Role Assignment
    role_id UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role_name VARCHAR(100),

    -- Status
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'invited', 'suspended')),
    invited_by VARCHAR(255),
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,

    -- Permissions Override
    custom_permissions JSONB,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_status ON workspace_members(status);

-- ============================================================================
-- WORKSPACE INVITATIONS TABLE
-- Magic link invitations for workspace membership
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Invitation Details
    email VARCHAR(255) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,

    -- Role to assign on accept
    role_id UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role_name VARCHAR(100) NOT NULL,

    -- Inviter Information
    invited_by_id VARCHAR(255) NOT NULL,
    invited_by_name VARCHAR(255),

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),

    -- Timestamps
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    accepted_by_user_id VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_invitations_token ON workspace_invitations(token);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_workspace ON workspace_invitations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_email ON workspace_invitations(email);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_status ON workspace_invitations(status) WHERE status = 'pending';

-- Partial unique index: only one pending invitation per email per workspace
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_invitations_pending_unique 
ON workspace_invitations(workspace_id, email) 
WHERE status = 'pending';

-- ============================================================================
-- WORKSPACE MEMORIES TABLE
-- Shared knowledge/context across team
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Memory Identity
    title VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    content TEXT NOT NULL,

    -- Memory Type
    memory_type VARCHAR(50) NOT NULL CHECK (memory_type IN ('decision', 'process', 'knowledge', 'pattern', 'policy')),
    category VARCHAR(100),

    -- Scope
    scope_type VARCHAR(50) DEFAULT 'workspace' CHECK (scope_type IN ('workspace', 'project', 'node')),
    scope_id UUID,

    -- Access Control
    visibility VARCHAR(50) DEFAULT 'team' CHECK (visibility IN ('team', 'managers', 'admins')),
    created_by VARCHAR(255) NOT NULL,

    -- Relevance
    importance_score DECIMAL(3,2) DEFAULT 0.5 CHECK (importance_score >= 0 AND importance_score <= 1),
    access_count INTEGER DEFAULT 0,

    -- Embeddings (nullable for Phase 2)
    -- embedding vector(1536),

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    is_active BOOLEAN DEFAULT TRUE,
    is_pinned BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace ON workspace_memories(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_scope ON workspace_memories(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_type ON workspace_memories(memory_type);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_active ON workspace_memories(workspace_id, is_active) WHERE is_active = TRUE;

-- ============================================================================
-- USER WORKSPACE PROFILES TABLE
-- Per-workspace user profiles (can differ between workspaces)
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_workspace_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Profile Info
    display_name VARCHAR(255),
    title VARCHAR(100),
    department VARCHAR(100),
    avatar_url VARCHAR(500),

    -- Contact
    work_email VARCHAR(255),
    phone VARCHAR(50),

    -- Work Preferences
    timezone VARCHAR(50),
    working_hours JSONB,
    notification_preferences JSONB,

    -- Agent Personalization
    preferred_output_style VARCHAR(50),
    communication_preferences JSONB,
    expertise_areas TEXT[] DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_workspace ON user_workspace_profiles(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_user ON user_workspace_profiles(user_id);

-- ============================================================================
-- PROJECT MEMBERS TABLE (WORKSPACE-AWARE)
-- Team assignment to projects with workspace context
-- ============================================================================
CREATE TABLE IF NOT EXISTS workspace_project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Project-specific role
    project_role VARCHAR(50) DEFAULT 'contributor' CHECK (project_role IN ('lead', 'contributor', 'reviewer', 'viewer')),

    -- Assignment details
    assigned_by VARCHAR(255),
    assigned_at TIMESTAMPTZ DEFAULT NOW(),

    -- Notification preferences
    notification_level VARCHAR(50) DEFAULT 'all' CHECK (notification_level IN ('all', 'mentions', 'none')),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(project_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_project_members_project ON workspace_project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_user ON workspace_project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_workspace ON workspace_project_members(workspace_id);

-- ============================================================================
-- FUNCTION: Seed default roles for a new workspace
-- ============================================================================
CREATE OR REPLACE FUNCTION seed_workspace_default_roles(p_workspace_id UUID)
RETURNS void AS $$
BEGIN
    -- Owner (level 100)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, permissions)
    VALUES (p_workspace_id, 'owner', 'Owner', 'Full access to everything including billing and ownership transfer', '#ef4444', 'crown', 100, TRUE, 
    '{
        "projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true},
        "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true},
        "clients": {"create": true, "read": true, "update": true, "delete": true},
        "artifacts": {"create": true, "read": true, "update": true, "delete": true},
        "settings": {"view": true, "manage": true},
        "team": {"view": true, "invite": true, "manage_roles": true},
        "billing": {"view": true, "manage": true},
        "agent": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}
    }'::jsonb);

    -- Admin (level 90)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, permissions)
    VALUES (p_workspace_id, 'admin', 'Administrator', 'Full access except billing and ownership transfer', '#f97316', 'shield', 90, TRUE,
    '{
        "projects": {"create": true, "read": true, "update": true, "delete": true, "manage_members": true},
        "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true},
        "clients": {"create": true, "read": true, "update": true, "delete": true},
        "artifacts": {"create": true, "read": true, "update": true, "delete": true},
        "settings": {"view": true, "manage": true},
        "team": {"view": true, "invite": true, "manage_roles": true},
        "billing": {"view": true, "manage": false},
        "agent": {"use_all_agents": true, "create_custom_agents": true, "access_workspace_memory": true, "modify_workspace_memory": true}
    }'::jsonb);

    -- Manager (level 70)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, permissions)
    VALUES (p_workspace_id, 'manager', 'Manager', 'Can manage projects, tasks, and team members', '#eab308', 'users', 70, TRUE,
    '{
        "projects": {"create": true, "read": true, "update": true, "delete": false, "manage_members": true},
        "tasks": {"create": true, "read": true, "update": true, "delete": true, "assign": true},
        "clients": {"create": true, "read": true, "update": true, "delete": false},
        "artifacts": {"create": true, "read": true, "update": true, "delete": false},
        "settings": {"view": true, "manage": false},
        "team": {"view": true, "invite": true, "manage_roles": false},
        "billing": {"view": false, "manage": false},
        "agent": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": true}
    }'::jsonb);

    -- Member (level 50) - Default role
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, is_default, permissions)
    VALUES (p_workspace_id, 'member', 'Member', 'Standard access - create and edit own work', '#22c55e', 'user', 50, TRUE, TRUE,
    '{
        "projects": {"create": true, "read": true, "update": "own", "delete": false, "manage_members": false},
        "tasks": {"create": true, "read": true, "update": "own", "delete": "own", "assign": false},
        "clients": {"create": false, "read": true, "update": false, "delete": false},
        "artifacts": {"create": true, "read": true, "update": "own", "delete": "own"},
        "settings": {"view": false, "manage": false},
        "team": {"view": true, "invite": false, "manage_roles": false},
        "billing": {"view": false, "manage": false},
        "agent": {"use_all_agents": true, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}
    }'::jsonb);

    -- Viewer (level 20)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, permissions)
    VALUES (p_workspace_id, 'viewer', 'Viewer', 'Read-only access', '#3b82f6', 'eye', 20, TRUE,
    '{
        "projects": {"create": false, "read": true, "update": false, "delete": false, "manage_members": false},
        "tasks": {"create": false, "read": true, "update": false, "delete": false, "assign": false},
        "clients": {"create": false, "read": true, "update": false, "delete": false},
        "artifacts": {"create": false, "read": true, "update": false, "delete": false},
        "settings": {"view": false, "manage": false},
        "team": {"view": true, "invite": false, "manage_roles": false},
        "billing": {"view": false, "manage": false},
        "agent": {"use_all_agents": false, "create_custom_agents": false, "access_workspace_memory": true, "modify_workspace_memory": false}
    }'::jsonb);

    -- Guest (level 10)
    INSERT INTO workspace_roles (workspace_id, name, display_name, description, color, icon, hierarchy_level, is_system, permissions)
    VALUES (p_workspace_id, 'guest', 'Guest', 'Limited read access to specific projects', '#6b7280', 'user-x', 10, TRUE,
    '{
        "projects": {"create": false, "read": "assigned", "update": false, "delete": false, "manage_members": false},
        "tasks": {"create": false, "read": "assigned", "update": false, "delete": false, "assign": false},
        "clients": {"create": false, "read": false, "update": false, "delete": false},
        "artifacts": {"create": false, "read": "assigned", "update": false, "delete": false},
        "settings": {"view": false, "manage": false},
        "team": {"view": false, "invite": false, "manage_roles": false},
        "billing": {"view": false, "manage": false},
        "agent": {"use_all_agents": false, "create_custom_agents": false, "access_workspace_memory": false, "modify_workspace_memory": false}
    }'::jsonb);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGER: Auto-update updated_at timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_workspace_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_workspaces_updated_at') THEN
        CREATE TRIGGER trigger_workspaces_updated_at
            BEFORE UPDATE ON workspaces
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_workspace_roles_updated_at') THEN
        CREATE TRIGGER trigger_workspace_roles_updated_at
            BEFORE UPDATE ON workspace_roles
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_workspace_members_updated_at') THEN
        CREATE TRIGGER trigger_workspace_members_updated_at
            BEFORE UPDATE ON workspace_members
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_workspace_invitations_updated_at') THEN
        CREATE TRIGGER trigger_workspace_invitations_updated_at
            BEFORE UPDATE ON workspace_invitations
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_workspace_memories_updated_at') THEN
        CREATE TRIGGER trigger_workspace_memories_updated_at
            BEFORE UPDATE ON workspace_memories
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_user_workspace_profiles_updated_at') THEN
        CREATE TRIGGER trigger_user_workspace_profiles_updated_at
            BEFORE UPDATE ON user_workspace_profiles
            FOR EACH ROW EXECUTE FUNCTION update_workspace_timestamp();
    END IF;
END $$;

-- ============================================================================
-- ROLLBACK SCRIPT (for reference)
-- ============================================================================
-- DROP TRIGGER IF EXISTS trigger_workspaces_updated_at ON workspaces;
-- DROP TRIGGER IF EXISTS trigger_workspace_roles_updated_at ON workspace_roles;
-- DROP TRIGGER IF EXISTS trigger_workspace_members_updated_at ON workspace_members;
-- DROP TRIGGER IF EXISTS trigger_workspace_invitations_updated_at ON workspace_invitations;
-- DROP TRIGGER IF EXISTS trigger_workspace_memories_updated_at ON workspace_memories;
-- DROP TRIGGER IF EXISTS trigger_user_workspace_profiles_updated_at ON user_workspace_profiles;
-- DROP FUNCTION IF EXISTS update_workspace_timestamp();
-- DROP FUNCTION IF EXISTS seed_workspace_default_roles(UUID);
-- DROP TABLE IF EXISTS workspace_project_members;
-- DROP TABLE IF EXISTS user_workspace_profiles;
-- DROP TABLE IF EXISTS workspace_memories;
-- DROP TABLE IF EXISTS workspace_invitations;
-- DROP TABLE IF EXISTS workspace_members;
-- DROP TABLE IF EXISTS workspace_roles;
-- DROP TABLE IF EXISTS workspaces;
