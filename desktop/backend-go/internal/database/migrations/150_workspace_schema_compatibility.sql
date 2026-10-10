-- 150_workspace_schema_compatibility.sql
-- Brings older local databases onto the workspace schema required by the
-- current backend. Account provisioning remains application behavior so this
-- migration also works for users created after it runs.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS max_members INTEGER DEFAULT 5,
  ADD COLUMN IF NOT EXISTS max_projects INTEGER DEFAULT 10,
  ADD COLUMN IF NOT EXISTS max_storage_gb INTEGER DEFAULT 5;

ALTER TABLE projects
  ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_projects_workspace_id ON projects(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_workspace_id ON tasks(workspace_id);

CREATE TABLE IF NOT EXISTS content_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  title VARCHAR(300) NOT NULL,
  content_type VARCHAR(40) DEFAULT 'post',
  status VARCHAR(40) DEFAULT 'idea',
  hook TEXT DEFAULT '',
  body TEXT DEFAULT '',
  caption TEXT DEFAULT '',
  cta TEXT DEFAULT '',
  channel VARCHAR(60) DEFAULT '',
  link TEXT DEFAULT '',
  category VARCHAR(100) DEFAULT '',
  client VARCHAR(255) DEFAULT '',
  campaign VARCHAR(255) DEFAULT '',
  owner VARCHAR(255) DEFAULT '',
  editor VARCHAR(255) DEFAULT '',
  priority VARCHAR(40) DEFAULT 'normal',
  due_date TEXT DEFAULT '',
  publish_date TEXT DEFAULT '',
  asset_link TEXT DEFAULT '',
  review_link TEXT DEFAULT '',
  revision_notes TEXT DEFAULT '',
  notes TEXT DEFAULT '',
  created_by VARCHAR(255),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE content_items
  ADD COLUMN IF NOT EXISTS hook TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS body TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS cta TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS channel VARCHAR(60) DEFAULT '',
  ADD COLUMN IF NOT EXISTS caption TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS revision_notes TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS category VARCHAR(100) DEFAULT '',
  ADD COLUMN IF NOT EXISTS client VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS campaign VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS owner VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS editor VARCHAR(255) DEFAULT '',
  ADD COLUMN IF NOT EXISTS priority VARCHAR(40) DEFAULT 'normal',
  ADD COLUMN IF NOT EXISTS due_date TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS publish_date TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS asset_link TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS review_link TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS notes TEXT DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_content_items_workspace ON content_items(workspace_id);
CREATE INDEX IF NOT EXISTS idx_content_items_workspace_updated ON content_items(workspace_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS sync_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  module VARCHAR(100) NOT NULL,
  sync_mode VARCHAR(20) NOT NULL DEFAULT 'local'
    CHECK (sync_mode IN ('workspace', 'local')),
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(workspace_id, module)
);

CREATE INDEX IF NOT EXISTS idx_sync_policies_workspace ON sync_policies(workspace_id);

CREATE TABLE IF NOT EXISTS governance_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_type TEXT NOT NULL DEFAULT 'algedonic',
  source TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'INFO',
  description TEXT NOT NULL DEFAULT '',
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_governance_events_type ON governance_events(event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_governance_events_severity ON governance_events(severity, created_at DESC);

CREATE TABLE IF NOT EXISTS role_permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  role VARCHAR(100) NOT NULL,
  resource VARCHAR(100) NOT NULL,
  permission VARCHAR(100) NOT NULL,
  metadata JSONB DEFAULT '{}',
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
    workspace_id, name, display_name, description, color, icon,
    is_system, is_default, hierarchy_level, permissions
  )
  SELECT p_workspace_id, role_name, display_name, description, color, icon,
         TRUE, is_default, hierarchy_level, permissions
  FROM (VALUES
    ('owner', 'Owner', 'Full workspace access including billing and deletion', '#8B5CF6', 'crown', FALSE, 1,
      '{"projects":{"create":true,"read":true,"update":true,"delete":true,"manage_members":true},"tasks":{"create":true,"read":true,"update":true,"delete":true,"assign":true},"contexts":{"create":true,"read":true,"update":true,"delete":true,"share":true},"workspace":{"invite_members":true,"manage_roles":true,"manage_billing":true,"delete_workspace":true},"agents":{"use_all_agents":true,"create_custom_agents":true,"access_workspace_memory":true,"modify_workspace_memory":true}}'::jsonb),
    ('admin', 'Admin', 'Full access except billing and workspace deletion', '#3B82F6', 'shield', FALSE, 2,
      '{"projects":{"create":true,"read":true,"update":true,"delete":true,"manage_members":true},"tasks":{"create":true,"read":true,"update":true,"delete":true,"assign":true},"contexts":{"create":true,"read":true,"update":true,"delete":true,"share":true},"workspace":{"invite_members":true,"manage_roles":true,"manage_billing":false,"delete_workspace":false},"agents":{"use_all_agents":true,"create_custom_agents":true,"access_workspace_memory":true,"modify_workspace_memory":true}}'::jsonb),
    ('manager', 'Manager', 'Can manage projects, tasks, and team members', '#10B981', 'users', FALSE, 3,
      '{"projects":{"create":true,"read":true,"update":true,"delete":false,"manage_members":true},"tasks":{"create":true,"read":true,"update":true,"delete":true,"assign":true},"contexts":{"create":true,"read":true,"update":true,"delete":false,"share":true},"workspace":{"invite_members":true,"manage_roles":false,"manage_billing":false,"delete_workspace":false},"agents":{"use_all_agents":true,"create_custom_agents":false,"access_workspace_memory":true,"modify_workspace_memory":true}}'::jsonb),
    ('member', 'Member', 'Standard team member', '#6366F1', 'user', TRUE, 4,
      '{"projects":{"create":true,"read":true,"update":true,"delete":false,"manage_members":false},"tasks":{"create":true,"read":true,"update":true,"delete":false,"assign":false},"contexts":{"create":true,"read":true,"update":true,"delete":false,"share":true},"workspace":{"invite_members":false,"manage_roles":false,"manage_billing":false,"delete_workspace":false},"agents":{"use_all_agents":true,"create_custom_agents":false,"access_workspace_memory":true,"modify_workspace_memory":false}}'::jsonb),
    ('viewer', 'Viewer', 'Read-only workspace access', '#64748B', 'eye', FALSE, 5,
      '{"projects":{"create":false,"read":true,"update":false,"delete":false,"manage_members":false},"tasks":{"create":false,"read":true,"update":false,"delete":false,"assign":false},"contexts":{"create":false,"read":true,"update":false,"delete":false,"share":false},"workspace":{"invite_members":false,"manage_roles":false,"manage_billing":false,"delete_workspace":false},"agents":{"use_all_agents":true,"create_custom_agents":false,"access_workspace_memory":true,"modify_workspace_memory":false}}'::jsonb),
    ('guest', 'Guest', 'Limited project access', '#94A3B8', 'user-plus', FALSE, 6,
      '{"projects":{"create":false,"read":true,"update":false,"delete":false,"manage_members":false},"tasks":{"create":false,"read":true,"update":false,"delete":false,"assign":false},"contexts":{"create":false,"read":true,"update":false,"delete":false,"share":false},"workspace":{"invite_members":false,"manage_roles":false,"manage_billing":false,"delete_workspace":false},"agents":{"use_all_agents":false,"create_custom_agents":false,"access_workspace_memory":false,"modify_workspace_memory":false}}'::jsonb)
  ) AS defaults(role_name, display_name, description, color, icon, is_default, hierarchy_level, permissions)
  ON CONFLICT (workspace_id, name) DO UPDATE
  SET display_name = EXCLUDED.display_name,
      description = EXCLUDED.description,
      color = EXCLUDED.color,
      icon = EXCLUDED.icon,
      is_system = TRUE,
      is_default = EXCLUDED.is_default,
      hierarchy_level = EXCLUDED.hierarchy_level,
      permissions = EXCLUDED.permissions,
      updated_at = NOW();

  INSERT INTO role_permissions (workspace_id, role, resource, permission, metadata)
  SELECT wr.workspace_id, wr.name, resource.key, permission.key,
         jsonb_build_object('allowed', permission.value)
  FROM workspace_roles wr
  CROSS JOIN LATERAL jsonb_each(wr.permissions) resource
  CROSS JOIN LATERAL jsonb_each(resource.value) permission
  WHERE wr.workspace_id = p_workspace_id AND permission.value = 'true'::jsonb
  ON CONFLICT (workspace_id, role, resource, permission) DO UPDATE
  SET metadata = EXCLUDED.metadata, updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE workspace_record RECORD;
BEGIN
  FOR workspace_record IN SELECT id FROM workspaces LOOP
    PERFORM seed_default_workspace_roles(workspace_record.id);
  END LOOP;
END $$;

UPDATE workspace_members wm
SET role_id = wr.id,
    role_name = wr.name,
    role = wr.name,
    updated_at = NOW()
FROM workspace_roles wr
WHERE wr.workspace_id = wm.workspace_id
  AND wr.name = COALESCE(NULLIF(wm.role_name, ''), NULLIF(wm.role, ''), 'member')
  AND (wm.role_id IS DISTINCT FROM wr.id OR wm.role_name IS DISTINCT FROM wr.name OR wm.role IS DISTINCT FROM wr.name);
