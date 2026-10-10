-- ============================================================================
-- BusinessOS Production Migration
-- Aligns production DB (setup-database.sql / workspace-centric) with the
-- Go backend's sqlc-generated queries (schema.sql / user-centric + enums).
--
-- Safe to run on a fresh database. Existing auth tables (user, session,
-- account, verification) are NOT dropped — only columns are added.
-- All CREATE TABLE / DROP TABLE statements use IF NOT EXISTS / IF EXISTS.
-- ============================================================================

BEGIN;

-- ============================================================================
-- EXTENSIONS
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================================
-- SECTION 1: ENUM TYPES
-- The Go sqlc code expects these PostgreSQL enums. Production has none.
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE messagerole AS ENUM ('USER', 'ASSISTANT', 'SYSTEM', 'user', 'assistant', 'system');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE artifacttype AS ENUM ('CODE', 'DOCUMENT', 'MARKDOWN', 'REACT', 'HTML', 'SVG');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE contexttype AS ENUM ('PERSON', 'BUSINESS', 'PROJECT', 'CUSTOM', 'document', 'DOCUMENT');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE projectstatus AS ENUM ('ACTIVE', 'PAUSED', 'COMPLETED', 'ARCHIVED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE projectpriority AS ENUM ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE nodetype AS ENUM ('BUSINESS', 'PROJECT', 'LEARNING', 'OPERATIONAL');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE nodehealth AS ENUM ('HEALTHY', 'NEEDS_ATTENTION', 'CRITICAL', 'NOT_STARTED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE taskstatus AS ENUM ('todo', 'in_progress', 'done', 'cancelled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE taskpriority AS ENUM ('critical', 'high', 'medium', 'low');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE memberstatus AS ENUM ('AVAILABLE', 'BUSY', 'OVERLOADED', 'OOO');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE clienttype AS ENUM ('company', 'individual');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE clientstatus AS ENUM ('lead', 'prospect', 'active', 'inactive', 'churned');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE interactiontype AS ENUM ('call', 'email', 'meeting', 'note');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE dependencytype AS ENUM ('finish_to_start', 'start_to_start', 'finish_to_finish', 'start_to_finish');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE meetingtype AS ENUM (
        'team', 'sales', 'onboarding', 'kickoff', 'implementation',
        'standup', 'retrospective', 'planning', 'review', 'one_on_one',
        'client', 'internal', 'external', 'other'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ============================================================================
-- SECTION 2: USER TABLE — add Better Auth camelCase columns if missing
-- Auth handlers INSERT with: id, name, email, "emailVerified", image, "createdAt", "updatedAt"
-- Auth handlers SELECT: id, name, email, "emailVerified", image, "createdAt"
-- ============================================================================
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "emailVerified" BOOLEAN DEFAULT FALSE;
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "createdAt"     TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS "updatedAt"     TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS image           TEXT;

-- ============================================================================
-- SECTION 3: DROP MISMATCHED TABLES (workspace-centric → user-centric)
-- These tables exist but have the wrong schema for the Go backend.
-- Safe: production has no real business data yet.
-- Order matters: dependents first.
-- ============================================================================

-- Remove FK constraints first where needed
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_workspace_id_fkey;
ALTER TABLE tasks    DROP CONSTRAINT IF EXISTS tasks_workspace_id_fkey;

DROP TABLE IF EXISTS messages              CASCADE;
DROP TABLE IF EXISTS conversations         CASCADE;
DROP TABLE IF EXISTS tasks                 CASCADE;
DROP TABLE IF EXISTS projects              CASCADE;
DROP TABLE IF EXISTS clients               CASCADE;
DROP TABLE IF EXISTS team_members          CASCADE;

-- workspace_roles in prod is a pivot table (workspace_id, user_id) — not the
-- role-definition table the Go queries expect. Drop it; we recreate below.
DROP TABLE IF EXISTS workspace_roles CASCADE;

-- ============================================================================
-- SECTION 4: CORE BUSINESS TABLES (user-centric, with enums)
-- Matches schema.sql exactly.
-- ============================================================================

-- 4a. Contexts (needed before projects, conversations, nodes, calendar_events)
CREATE TABLE IF NOT EXISTS contexts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               VARCHAR(255) NOT NULL,
    name                  VARCHAR(255) NOT NULL,
    type                  contexttype DEFAULT 'CUSTOM',
    content               TEXT,
    structured_data       JSONB,
    system_prompt_template TEXT,
    blocks                JSONB DEFAULT '[]',
    cover_image           VARCHAR(500),
    icon                  VARCHAR(50),
    parent_id             UUID REFERENCES contexts(id) ON DELETE SET NULL,
    is_template           BOOLEAN DEFAULT FALSE,
    is_archived           BOOLEAN DEFAULT FALSE,
    last_edited_at        TIMESTAMP,
    word_count            INTEGER DEFAULT 0,
    is_public             BOOLEAN DEFAULT FALSE,
    share_id              VARCHAR(32) UNIQUE,
    property_schema       JSONB DEFAULT '[]',
    properties            JSONB DEFAULT '{}',
    client_id             UUID,
    created_at            TIMESTAMP DEFAULT NOW(),
    updated_at            TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contexts_user_id    ON contexts(user_id);
CREATE INDEX IF NOT EXISTS idx_contexts_parent_id  ON contexts(parent_id);
CREATE INDEX IF NOT EXISTS idx_contexts_is_archived ON contexts(is_archived);
CREATE INDEX IF NOT EXISTS idx_contexts_share_id   ON contexts(share_id);

-- 4b. Clients (user-centric, rich CRM fields)
CREATE TABLE IF NOT EXISTS clients (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          VARCHAR(255) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    type             clienttype DEFAULT 'company',
    email            VARCHAR(255),
    phone            VARCHAR(50),
    website          VARCHAR(255),
    industry         VARCHAR(100),
    company_size     VARCHAR(50),
    address          VARCHAR(255),
    city             VARCHAR(100),
    state            VARCHAR(100),
    zip_code         VARCHAR(20),
    country          VARCHAR(100),
    status           clientstatus DEFAULT 'lead',
    source           VARCHAR(100),
    assigned_to      VARCHAR(255),
    lifetime_value   NUMERIC(12, 2),
    tags             JSONB DEFAULT '[]',
    custom_fields    JSONB DEFAULT '{}',
    notes            TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    last_contacted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_clients_user_id        ON clients(user_id);
CREATE INDEX IF NOT EXISTS ix_clients_user_status     ON clients(user_id, status);
CREATE INDEX IF NOT EXISTS ix_clients_user_type       ON clients(user_id, type);

-- Add FK from contexts to clients now that both exist
DO $$ BEGIN
    ALTER TABLE contexts
        ADD CONSTRAINT fk_contexts_client_id
        FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
CREATE INDEX IF NOT EXISTS idx_contexts_client_id ON contexts(client_id);

-- 4c. Projects (user-centric, enum status/priority)
CREATE TABLE IF NOT EXISTS projects (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          VARCHAR(255) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    status           projectstatus DEFAULT 'ACTIVE',
    priority         projectpriority DEFAULT 'MEDIUM',
    client_name      VARCHAR(255),
    client_id        UUID REFERENCES clients(id) ON DELETE SET NULL,
    project_type     VARCHAR(100) DEFAULT 'internal',
    project_metadata JSONB,
    start_date       DATE,
    due_date         DATE,
    completed_at     TIMESTAMPTZ,
    visibility       VARCHAR(20) DEFAULT 'private',
    owner_id         VARCHAR(255),
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_projects_user_id  ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_client   ON projects(client_id);
CREATE INDEX IF NOT EXISTS idx_projects_status   ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_due_date ON projects(due_date);

-- 4d. Project notes
CREATE TABLE IF NOT EXISTS project_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 4e. Conversations (user-centric, context-linked)
CREATE TABLE IF NOT EXISTS conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(255) NOT NULL,
    title      VARCHAR(255) DEFAULT 'New Conversation',
    context_id UUID REFERENCES contexts(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);

-- 4f. Messages (enum role)
CREATE TABLE IF NOT EXISTS messages (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id  UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role             messagerole NOT NULL,
    content          TEXT NOT NULL,
    created_at       TIMESTAMP DEFAULT NOW(),
    message_metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);

-- 4g. Team members (user-centric, enum status)
CREATE TABLE IF NOT EXISTS team_members (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    email             VARCHAR(255) NOT NULL,
    role              VARCHAR(255) NOT NULL,
    avatar_url        TEXT,
    status            memberstatus DEFAULT 'AVAILABLE',
    capacity          INTEGER DEFAULT 0,
    manager_id        UUID REFERENCES team_members(id) ON DELETE SET NULL,
    skills            JSONB,
    hourly_rate       NUMERIC(10, 2),
    share_calendar    BOOLEAN DEFAULT FALSE,
    calendar_user_id  VARCHAR(255),
    joined_at         TIMESTAMP DEFAULT NOW(),
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);

-- 4h. Tasks (user-centric, enum status/priority, FK to team_members for assignee)
CREATE TABLE IF NOT EXISTS tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        VARCHAR(255) NOT NULL,
    title          VARCHAR(500) NOT NULL,
    description    TEXT,
    status         taskstatus DEFAULT 'todo',
    priority       taskpriority DEFAULT 'medium',
    due_date       TIMESTAMP,
    start_date     TIMESTAMP,
    completed_at   TIMESTAMP,
    project_id     UUID REFERENCES projects(id) ON DELETE SET NULL,
    assignee_id    UUID REFERENCES team_members(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    custom_status_id UUID,
    position       INT DEFAULT 0,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id  ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent   ON tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_position ON tasks(user_id, position);

-- ============================================================================
-- SECTION 5: WORKSPACE SUBSYSTEM TABLES
-- Required by workspaces.sql queries: workspace_members, workspace_roles,
-- workspace_invites, workspace_memories, user_workspace_profiles,
-- workspace_project_members.
-- The existing "workspaces" table structure is compatible — keep it, extend it.
-- ============================================================================

-- Add missing columns to existing workspaces table
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS logo_url  TEXT;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS plan_type VARCHAR(50) DEFAULT 'free';

-- Workspace roles (role definitions per workspace)
CREATE TABLE IF NOT EXISTS workspace_roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(100),
    description     TEXT,
    color           VARCHAR(20) DEFAULT '#6B7280',
    icon            VARCHAR(50),
    hierarchy_level INTEGER DEFAULT 0,
    is_system       BOOLEAN DEFAULT FALSE,
    is_default      BOOLEAN DEFAULT FALSE,
    permissions     JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, name)
);
CREATE INDEX IF NOT EXISTS idx_workspace_roles_workspace ON workspace_roles(workspace_id);

-- Workspace members: drop old minimal version and recreate with full schema
DROP TABLE IF EXISTS workspace_members CASCADE;
CREATE TABLE IF NOT EXISTS workspace_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    role_id      UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role_name    VARCHAR(100) NOT NULL DEFAULT 'member',
    status       VARCHAR(50) NOT NULL DEFAULT 'active',
    invited_by   VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    invited_at   TIMESTAMPTZ,
    joined_at    TIMESTAMPTZ DEFAULT NOW(),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_user      ON workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_status    ON workspace_members(workspace_id, status);

-- Workspace invitations
CREATE TABLE IF NOT EXISTS workspace_invites (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        VARCHAR(255) NOT NULL,
    token        VARCHAR(255) NOT NULL UNIQUE,
    role         VARCHAR(100) DEFAULT 'member',
    invited_by   VARCHAR(255) NOT NULL REFERENCES "user"(id),
    status       VARCHAR(50) DEFAULT 'pending',
    expires_at   TIMESTAMPTZ NOT NULL,
    accepted_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace ON workspace_invites(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_email     ON workspace_invites(email);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_token     ON workspace_invites(token);

-- Workspace memories
CREATE TABLE IF NOT EXISTS workspace_memories (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title             VARCHAR(500) NOT NULL,
    summary           TEXT,
    content           TEXT,
    memory_type       VARCHAR(100) DEFAULT 'general',
    category          VARCHAR(100),
    scope_type        VARCHAR(100),
    scope_id          UUID,
    visibility        VARCHAR(50) DEFAULT 'workspace',
    created_by        VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    importance_score  FLOAT DEFAULT 0.5,
    tags              JSONB DEFAULT '[]',
    metadata          JSONB DEFAULT '{}',
    is_pinned         BOOLEAN DEFAULT FALSE,
    is_active         BOOLEAN DEFAULT TRUE,
    access_count      INTEGER DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_workspace ON workspace_memories(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_type      ON workspace_memories(workspace_id, memory_type);
CREATE INDEX IF NOT EXISTS idx_workspace_memories_scope     ON workspace_memories(workspace_id, scope_type, scope_id);

-- User workspace profiles
CREATE TABLE IF NOT EXISTS user_workspace_profiles (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id             UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id                  VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    display_name             VARCHAR(255),
    title                    VARCHAR(255),
    department               VARCHAR(255),
    avatar_url               TEXT,
    work_email               VARCHAR(255),
    phone                    VARCHAR(50),
    timezone                 VARCHAR(100),
    working_hours            JSONB DEFAULT '{}',
    notification_preferences JSONB DEFAULT '{}',
    expertise_areas          JSONB DEFAULT '[]',
    created_at               TIMESTAMPTZ DEFAULT NOW(),
    updated_at               TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_workspace ON user_workspace_profiles(workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_workspace_profiles_user      ON user_workspace_profiles(user_id);

-- Workspace project members (workspace-aware project assignments)
CREATE TABLE IF NOT EXISTS workspace_project_members (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id            VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    workspace_id       UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_role       VARCHAR(100) DEFAULT 'contributor',
    assigned_by        VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    notification_level VARCHAR(50) DEFAULT 'all',
    assigned_at        TIMESTAMPTZ DEFAULT NOW(),
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_project   ON workspace_project_members(project_id);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_user      ON workspace_project_members(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_project_members_workspace ON workspace_project_members(workspace_id);

-- ============================================================================
-- SECTION 6: NODES (knowledge graph)
-- ============================================================================
CREATE TABLE IF NOT EXISTS nodes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          VARCHAR(255) NOT NULL,
    parent_id        UUID REFERENCES nodes(id) ON DELETE SET NULL,
    context_id       UUID REFERENCES contexts(id) ON DELETE SET NULL,
    name             VARCHAR(255) NOT NULL,
    type             nodetype NOT NULL,
    health           nodehealth DEFAULT 'NOT_STARTED',
    purpose          TEXT,
    current_status   TEXT,
    this_week_focus  JSONB,
    decision_queue   JSONB,
    delegation_ready JSONB,
    is_active        BOOLEAN DEFAULT FALSE,
    is_archived      BOOLEAN DEFAULT FALSE,
    sort_order       INTEGER DEFAULT 0,
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_nodes_user_id ON nodes(user_id);

CREATE TABLE IF NOT EXISTS node_metrics (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id      UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    metric_name  VARCHAR(255) NOT NULL,
    metric_value VARCHAR(500) NOT NULL,
    recorded_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS node_projects (
    node_id    UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    linked_at  TIMESTAMPTZ DEFAULT NOW(),
    linked_by  VARCHAR(255),
    PRIMARY KEY (node_id, project_id)
);
CREATE INDEX IF NOT EXISTS idx_node_projects_node    ON node_projects(node_id);
CREATE INDEX IF NOT EXISTS idx_node_projects_project ON node_projects(project_id);

CREATE TABLE IF NOT EXISTS node_contexts (
    node_id    UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    context_id UUID NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    linked_at  TIMESTAMPTZ DEFAULT NOW(),
    linked_by  VARCHAR(255),
    PRIMARY KEY (node_id, context_id)
);
CREATE INDEX IF NOT EXISTS idx_node_contexts_node    ON node_contexts(node_id);
CREATE INDEX IF NOT EXISTS idx_node_contexts_context ON node_contexts(context_id);

CREATE TABLE IF NOT EXISTS node_conversations (
    node_id         UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    linked_at       TIMESTAMPTZ DEFAULT NOW(),
    linked_by       VARCHAR(255),
    PRIMARY KEY (node_id, conversation_id)
);
CREATE INDEX IF NOT EXISTS idx_node_conversations_node         ON node_conversations(node_id);
CREATE INDEX IF NOT EXISTS idx_node_conversations_conversation ON node_conversations(conversation_id);

-- ============================================================================
-- SECTION 7: TASKS SUBSYSTEM (project statuses, assignees, dependencies)
-- ============================================================================

CREATE TABLE IF NOT EXISTS project_statuses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    color        VARCHAR(7) DEFAULT '#6B7280',
    position     INT DEFAULT 0,
    is_done_state BOOLEAN DEFAULT FALSE,
    is_default   BOOLEAN DEFAULT FALSE,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, name)
);
CREATE INDEX IF NOT EXISTS idx_project_statuses_project ON project_statuses(project_id);

-- Add FK from tasks to project_statuses
DO $$ BEGIN
    ALTER TABLE tasks ADD CONSTRAINT fk_tasks_custom_status
        FOREIGN KEY (custom_status_id) REFERENCES project_statuses(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS task_assignees (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    role           VARCHAR(50) DEFAULT 'assignee',
    assigned_at    TIMESTAMPTZ DEFAULT NOW(),
    assigned_by    VARCHAR(255),
    UNIQUE(task_id, team_member_id)
);
CREATE INDEX IF NOT EXISTS idx_task_assignees_task   ON task_assignees(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignees_member ON task_assignees(team_member_id);

CREATE TABLE IF NOT EXISTS task_dependencies (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    predecessor_id   UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    successor_id     UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type  dependencytype DEFAULT 'finish_to_start',
    lag_days         INT DEFAULT 0,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(predecessor_id, successor_id)
);
CREATE INDEX IF NOT EXISTS idx_task_deps_predecessor ON task_dependencies(predecessor_id);
CREATE INDEX IF NOT EXISTS idx_task_deps_successor   ON task_dependencies(successor_id);

-- ============================================================================
-- SECTION 8: ARTIFACTS
-- ============================================================================
CREATE TABLE IF NOT EXISTS artifacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) NOT NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id      UUID REFERENCES messages(id) ON DELETE SET NULL,
    project_id      UUID REFERENCES projects(id) ON DELETE SET NULL,
    context_id      UUID REFERENCES contexts(id) ON DELETE SET NULL,
    title           VARCHAR(255) NOT NULL,
    type            artifacttype NOT NULL,
    language        VARCHAR(50),
    content         TEXT NOT NULL,
    summary         VARCHAR(500),
    version         INTEGER DEFAULT 1,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_artifacts_user_id         ON artifacts(user_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_conversation_id ON artifacts(conversation_id);

CREATE TABLE IF NOT EXISTS artifact_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version     INTEGER NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- SECTION 9: PRODUCTIVITY TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS focus_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(255) NOT NULL,
    text       VARCHAR(500) NOT NULL,
    completed  BOOLEAN DEFAULT FALSE,
    focus_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_focus_items_user_id ON focus_items(user_id);

CREATE TABLE IF NOT EXISTS daily_logs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              VARCHAR(255) NOT NULL,
    date                 DATE NOT NULL,
    content              TEXT NOT NULL,
    transcription_source VARCHAR(50),
    extracted_actions    JSONB,
    extracted_patterns   JSONB,
    energy_level         INTEGER,
    created_at           TIMESTAMP DEFAULT NOW(),
    updated_at           TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, date)
);
CREATE INDEX IF NOT EXISTS idx_daily_logs_user_id ON daily_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_daily_logs_date    ON daily_logs(date);

CREATE TABLE IF NOT EXISTS user_settings (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                    VARCHAR(255) UNIQUE NOT NULL,
    default_model              VARCHAR(100),
    email_notifications        BOOLEAN DEFAULT TRUE,
    daily_summary              BOOLEAN DEFAULT FALSE,
    theme                      VARCHAR(20) DEFAULT 'light',
    sidebar_collapsed           BOOLEAN DEFAULT FALSE,
    share_analytics            BOOLEAN DEFAULT TRUE,
    custom_settings            JSONB,
    thinking_enabled           BOOLEAN DEFAULT FALSE,
    thinking_show_in_ui        BOOLEAN DEFAULT TRUE,
    thinking_save_traces       BOOLEAN DEFAULT TRUE,
    thinking_default_template_id UUID,
    thinking_max_tokens        INT DEFAULT 4096,
    created_at                 TIMESTAMP DEFAULT NOW(),
    updated_at                 TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_settings_user_id ON user_settings(user_id);

-- ============================================================================
-- SECTION 10: CRM EXTENSIONS (client contacts, interactions, deals)
-- ============================================================================

CREATE TABLE IF NOT EXISTS client_contacts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id  UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    email      VARCHAR(255),
    phone      VARCHAR(50),
    role       VARCHAR(100),
    is_primary BOOLEAN DEFAULT FALSE,
    notes      TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_client_contacts_client ON client_contacts(client_id);

CREATE TABLE IF NOT EXISTS client_interactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    contact_id  UUID REFERENCES client_contacts(id) ON DELETE SET NULL,
    type        interactiontype NOT NULL,
    subject     VARCHAR(255) NOT NULL,
    description TEXT,
    outcome     VARCHAR(255),
    occurred_at TIMESTAMPTZ DEFAULT NOW(),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS ix_client_interactions_client   ON client_interactions(client_id);
CREATE INDEX IF NOT EXISTS ix_client_interactions_occurred ON client_interactions(occurred_at);

-- CRM Pipelines & Deals (referenced by clients.sql queries)
CREATE TABLE IF NOT EXISTS pipelines (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pipelines_user ON pipelines(user_id);

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    position    INT DEFAULT 0,
    probability INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id);

CREATE TABLE IF NOT EXISTS deals (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id          UUID REFERENCES clients(id) ON DELETE SET NULL,
    user_id            VARCHAR(255) NOT NULL,
    pipeline_id        UUID REFERENCES pipelines(id) ON DELETE SET NULL,
    stage_id           UUID REFERENCES pipeline_stages(id) ON DELETE SET NULL,
    name               VARCHAR(255) NOT NULL,
    amount             NUMERIC(12, 2) DEFAULT 0,
    probability        INTEGER DEFAULT 0,
    expected_close_date DATE,
    actual_close_date  DATE,
    description        TEXT,
    status             VARCHAR(50) DEFAULT 'open',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_deals_client   ON deals(client_id);
CREATE INDEX IF NOT EXISTS idx_deals_user     ON deals(user_id);
CREATE INDEX IF NOT EXISTS idx_deals_pipeline ON deals(pipeline_id, stage_id);

-- ============================================================================
-- SECTION 11: PROJECT CONVERSATIONS (many-to-many)
-- ============================================================================
CREATE TABLE IF NOT EXISTS project_conversations (
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    linked_at       TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (project_id, conversation_id)
);

CREATE TABLE IF NOT EXISTS conversation_tags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tag             VARCHAR(100) NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- SECTION 12: TEAM MEMBER ACTIVITIES
-- ============================================================================
CREATE TABLE IF NOT EXISTS team_member_activities (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id      UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    activity_type  VARCHAR(100) NOT NULL,
    description    TEXT NOT NULL,
    created_at     TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- SECTION 13: OSA WORKSPACES & MODULES
-- Required by osa_workspace_init.go (CreateOSAWorkspace, ListOSAWorkspacesByUser)
-- ============================================================================

CREATE TABLE IF NOT EXISTS osa_workspaces (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    mode            VARCHAR(20) DEFAULT '2d',
    layout          JSONB DEFAULT '{}',
    active_modules  UUID[] DEFAULT '{}',
    template_type   VARCHAR(50) DEFAULT 'business_os',
    settings        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_osa_workspaces_user     ON osa_workspaces(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_workspaces_template ON osa_workspaces(template_type);

CREATE TABLE IF NOT EXISTS osa_modules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(255) NOT NULL,
    display_name      VARCHAR(255) NOT NULL,
    description       TEXT,
    module_type       VARCHAR(50) NOT NULL,
    schema_definition JSONB,
    api_definition    JSONB,
    ui_definition     JSONB,
    created_by        VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL,
    workspace_id      UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    status            VARCHAR(50) DEFAULT 'draft',
    version           VARCHAR(50) DEFAULT '1.0.0',
    metadata          JSONB DEFAULT '{}',
    tags              TEXT[],
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    deployed_at       TIMESTAMPTZ,
    UNIQUE(name, workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_osa_modules_workspace  ON osa_modules(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_modules_status     ON osa_modules(status);
CREATE INDEX IF NOT EXISTS idx_osa_modules_created_by ON osa_modules(created_by);

CREATE TABLE IF NOT EXISTS osa_module_instances (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    module_id        UUID REFERENCES osa_modules(id) ON DELETE SET NULL,
    name             VARCHAR(255) NOT NULL,
    display_name     VARCHAR(255) NOT NULL,
    description      TEXT,
    osa_workflow_id  VARCHAR(255),
    osa_sandbox_id   VARCHAR(255),
    code_repository  TEXT,
    deployment_url   TEXT,
    status           VARCHAR(50) DEFAULT 'generated',
    files_created    INTEGER DEFAULT 0,
    tests_passed     BOOLEAN DEFAULT FALSE,
    build_status     VARCHAR(50),
    metadata         JSONB DEFAULT '{}',
    error_message    TEXT,
    error_stack      TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    generated_at     TIMESTAMPTZ,
    deployed_at      TIMESTAMPTZ,
    last_build_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_osa_module_instances_workspace ON osa_module_instances(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_module_instances_status    ON osa_module_instances(status);
CREATE INDEX IF NOT EXISTS idx_osa_module_instances_workflow  ON osa_module_instances(osa_workflow_id);

CREATE TABLE IF NOT EXISTS osa_build_events (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_instance_id UUID NOT NULL REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    workspace_id       UUID REFERENCES osa_workspaces(id) ON DELETE SET NULL,
    event_type         VARCHAR(50) NOT NULL,
    event_data         JSONB DEFAULT '{}',
    build_id           VARCHAR(255),
    phase              VARCHAR(50),
    progress_percent   INTEGER DEFAULT 0,
    status_message     TEXT,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osa_build_events_instance ON osa_build_events(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_build_events_type     ON osa_build_events(event_type);
CREATE INDEX IF NOT EXISTS idx_osa_build_events_build_id ON osa_build_events(build_id);
CREATE INDEX IF NOT EXISTS idx_osa_build_events_created  ON osa_build_events(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_execution_history (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE SET NULL,
    workspace_id       UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    command            TEXT NOT NULL,
    working_directory  TEXT,
    environment_vars   JSONB DEFAULT '{}',
    output             TEXT,
    error_output       TEXT,
    exit_code          INTEGER,
    duration_ms        INTEGER,
    triggered_by       VARCHAR(50),
    metadata           JSONB DEFAULT '{}',
    created_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osa_exec_user      ON osa_execution_history(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_instance  ON osa_execution_history(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_workspace ON osa_execution_history(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_exec_created   ON osa_execution_history(created_at DESC);

CREATE TABLE IF NOT EXISTS osa_sync_status (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type      VARCHAR(50) NOT NULL,
    entity_id        UUID NOT NULL,
    osa_entity_id    VARCHAR(255),
    osa_entity_type  VARCHAR(50),
    sync_status      VARCHAR(50) DEFAULT 'pending',
    last_sync_at     TIMESTAMPTZ,
    next_sync_at     TIMESTAMPTZ,
    sync_direction   VARCHAR(50) DEFAULT 'bidirectional',
    error_count      INTEGER DEFAULT 0,
    last_error       TEXT,
    local_snapshot   JSONB,
    remote_snapshot  JSONB,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(entity_type, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_osa_sync_entity ON osa_sync_status(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_osa_sync_status ON osa_sync_status(sync_status);

CREATE TABLE IF NOT EXISTS osa_webhooks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id       UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    event_type         VARCHAR(50) NOT NULL,
    webhook_url        TEXT NOT NULL,
    secret_key         VARCHAR(255),
    enabled            BOOLEAN DEFAULT TRUE,
    last_triggered_at  TIMESTAMPTZ,
    success_count      INTEGER DEFAULT 0,
    failure_count      INTEGER DEFAULT 0,
    metadata           JSONB DEFAULT '{}',
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_workspace ON osa_webhooks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_instance  ON osa_webhooks(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_webhooks_event     ON osa_webhooks(event_type);

-- ============================================================================
-- SECTION 14: GOOGLE OAUTH & INTEGRATION TABLES
-- ============================================================================

CREATE TABLE IF NOT EXISTS google_oauth_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       VARCHAR(255) UNIQUE NOT NULL,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type    VARCHAR(50) DEFAULT 'Bearer',
    expiry        TIMESTAMPTZ NOT NULL,
    scopes        TEXT[],
    google_email  VARCHAR(255),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_google_oauth_user_id ON google_oauth_tokens(user_id);

CREATE TABLE IF NOT EXISTS slack_oauth_tokens (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  VARCHAR(255) UNIQUE NOT NULL,
    workspace_id             VARCHAR(255) NOT NULL,
    workspace_name           VARCHAR(255),
    bot_token                TEXT NOT NULL,
    user_token               TEXT,
    bot_user_id              VARCHAR(255),
    authed_user_id           VARCHAR(255),
    bot_scopes               TEXT[],
    user_scopes              TEXT[],
    incoming_webhook_url     TEXT,
    incoming_webhook_channel VARCHAR(255),
    created_at               TIMESTAMPTZ DEFAULT NOW(),
    updated_at               TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_slack_oauth_user_id  ON slack_oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_slack_oauth_workspace ON slack_oauth_tokens(workspace_id);

CREATE TABLE IF NOT EXISTS notion_oauth_tokens (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           VARCHAR(255) UNIQUE NOT NULL,
    workspace_id      VARCHAR(255) NOT NULL,
    workspace_name    VARCHAR(255),
    workspace_icon    TEXT,
    access_token      TEXT NOT NULL,
    bot_id            VARCHAR(255),
    owner_type        VARCHAR(50),
    owner_user_id     VARCHAR(255),
    owner_user_name   VARCHAR(255),
    owner_user_email  VARCHAR(255),
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notion_oauth_user_id  ON notion_oauth_tokens(user_id);

-- ============================================================================
-- SECTION 15: CALENDAR EVENTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS calendar_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) NOT NULL,
    google_event_id VARCHAR(255),
    calendar_id     VARCHAR(255) DEFAULT 'primary',
    title           VARCHAR(500),
    description     TEXT,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    all_day         BOOLEAN DEFAULT FALSE,
    location        VARCHAR(500),
    attendees       JSONB DEFAULT '[]',
    status          VARCHAR(50) DEFAULT 'confirmed',
    visibility      VARCHAR(50) DEFAULT 'default',
    html_link       TEXT,
    source          VARCHAR(50) DEFAULT 'google',
    meeting_type    meetingtype DEFAULT 'other',
    context_id      UUID REFERENCES contexts(id) ON DELETE SET NULL,
    project_id      UUID REFERENCES projects(id) ON DELETE SET NULL,
    client_id       UUID REFERENCES clients(id) ON DELETE SET NULL,
    recording_url   TEXT,
    meeting_link    TEXT,
    external_links  JSONB DEFAULT '[]',
    meeting_notes   TEXT,
    action_items    JSONB DEFAULT '[]',
    synced_at       TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, google_event_id)
);
CREATE INDEX IF NOT EXISTS idx_calendar_events_user_id ON calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_time    ON calendar_events(user_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_calendar_events_source  ON calendar_events(source);
CREATE INDEX IF NOT EXISTS idx_calendar_events_type    ON calendar_events(meeting_type);

-- ============================================================================
-- SECTION 16: USAGE ANALYTICS
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           VARCHAR(255) NOT NULL,
    conversation_id   UUID REFERENCES conversations(id) ON DELETE SET NULL,
    provider          VARCHAR(50) NOT NULL,
    model             VARCHAR(100) NOT NULL,
    input_tokens      INTEGER DEFAULT 0,
    output_tokens     INTEGER DEFAULT 0,
    total_tokens      INTEGER DEFAULT 0,
    thinking_tokens   INTEGER DEFAULT 0,
    agent_name        VARCHAR(100),
    delegated_to      VARCHAR(100),
    parent_request_id UUID REFERENCES ai_usage_logs(id) ON DELETE SET NULL,
    request_type      VARCHAR(50),
    context_ids       UUID[],
    node_id           UUID REFERENCES nodes(id) ON DELETE SET NULL,
    project_id        UUID REFERENCES projects(id) ON DELETE SET NULL,
    duration_ms       INTEGER DEFAULT 0,
    started_at        TIMESTAMPTZ DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    estimated_cost    NUMERIC(10, 6),
    created_at        TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_usage_user_id      ON ai_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_conversation ON ai_usage_logs(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_provider     ON ai_usage_logs(provider);
CREATE INDEX IF NOT EXISTS idx_ai_usage_date         ON ai_usage_logs(started_at);

-- ============================================================================
-- SECTION 17: NOTIFICATIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(255) NOT NULL,
    type       VARCHAR(100) NOT NULL,
    title      VARCHAR(500) NOT NULL,
    body       TEXT,
    data       JSONB DEFAULT '{}',
    is_read    BOOLEAN DEFAULT FALSE,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id  ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read  ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_created  ON notifications(user_id, created_at DESC);

-- ============================================================================
-- SECTION 18: ONBOARDING
-- ============================================================================

CREATE TABLE IF NOT EXISTS onboarding_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(255) UNIQUE NOT NULL,
    current_step    INTEGER DEFAULT 0,
    completed_steps JSONB DEFAULT '[]',
    skipped         BOOLEAN DEFAULT FALSE,
    completed_at    TIMESTAMPTZ,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_onboarding_state_user_id ON onboarding_state(user_id);

-- ============================================================================
-- SECTION 19: ACTIVITY LOG
-- ============================================================================

CREATE TABLE IF NOT EXISTS activity_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(255) NOT NULL,
    action      VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100),
    entity_id   VARCHAR(255),
    details     JSONB DEFAULT '{}',
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_activity_log_user_id ON activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_entity  ON activity_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_created ON activity_log(user_id, created_at DESC);

-- ============================================================================
-- SECTION 20: OSA MODULE FILES & WORKFLOWS (referenced by later migrations)
-- ============================================================================

CREATE TABLE IF NOT EXISTS osa_module_files (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_instance_id UUID NOT NULL REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    file_path          TEXT NOT NULL,
    file_content       TEXT,
    file_type          VARCHAR(50),
    is_generated       BOOLEAN DEFAULT TRUE,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osa_module_files_instance ON osa_module_files(module_instance_id);

CREATE TABLE IF NOT EXISTS osa_workflows (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_instance_id UUID REFERENCES osa_module_instances(id) ON DELETE CASCADE,
    workspace_id       UUID REFERENCES osa_workspaces(id) ON DELETE CASCADE,
    user_id            VARCHAR(255) REFERENCES "user"(id) ON DELETE CASCADE,
    name               VARCHAR(255) NOT NULL,
    status             VARCHAR(50) DEFAULT 'pending',
    steps              JSONB DEFAULT '[]',
    current_step       INTEGER DEFAULT 0,
    metadata           JSONB DEFAULT '{}',
    error_message      TEXT,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW(),
    completed_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_instance  ON osa_workflows(module_instance_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_workspace ON osa_workflows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_user      ON osa_workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_workflows_status    ON osa_workflows(status);

-- ============================================================================
-- SECTION 21: MCP SERVERS (migration 101)
-- ============================================================================

CREATE TABLE IF NOT EXISTS mcp_servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    url         TEXT,
    command     TEXT,
    args        JSONB DEFAULT '[]',
    env         JSONB DEFAULT '{}',
    is_enabled  BOOLEAN DEFAULT TRUE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_user ON mcp_servers(user_id);

-- ============================================================================
-- SECTION 22: CREDENTIAL VAULT
-- ============================================================================

CREATE TABLE IF NOT EXISTS credential_vault (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        VARCHAR(255) NOT NULL,
    name           VARCHAR(255) NOT NULL,
    service        VARCHAR(100) NOT NULL,
    credential_type VARCHAR(50) DEFAULT 'api_key',
    encrypted_data TEXT NOT NULL,
    is_active      BOOLEAN DEFAULT TRUE,
    last_used_at   TIMESTAMPTZ,
    expires_at     TIMESTAMPTZ,
    metadata       JSONB DEFAULT '{}',
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credential_vault_user    ON credential_vault(user_id);
CREATE INDEX IF NOT EXISTS idx_credential_vault_service ON credential_vault(user_id, service);

-- ============================================================================
-- SECTION 23: OSA SYNC LOG (migration 102)
-- ============================================================================

CREATE TABLE IF NOT EXISTS osa_sync_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id   UUID,
    action      VARCHAR(50) NOT NULL,
    status      VARCHAR(50) DEFAULT 'success',
    details     JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_user   ON osa_sync_log(user_id);
CREATE INDEX IF NOT EXISTS idx_osa_sync_log_entity ON osa_sync_log(entity_type, entity_id);

-- ============================================================================
-- SECTION 24: PLATFORM ADMIN (migration 103/104) — add missing columns
-- ============================================================================

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS platform_role VARCHAR(20) DEFAULT 'user'
    CHECK (platform_role IN ('superadmin', 'admin', 'user'));

-- ============================================================================
-- DONE
-- ============================================================================

COMMIT;
