-- ─────────────────────────────────────────────────────────────────────────────
-- 103: Organizations + Teams hierarchy (adaptive — partial org tables pre-existed)
--
--   Organization (the account)
--     ├── organization_members  (everyone in the account; org role)
--     ├── organization_invites   (invite people to the org)
--     └── Workspace (workspaces.organization_id)
--           ├── workspace_members  (GRANULAR access — which org people see this workspace)
--           └── Team (teams.workspace_id) · team_memberships (move users between teams)
--
-- Org membership != workspace access. Role-scoped module data stays on
-- workspace_roles.permissions JSONB (unchanged here).
-- ─────────────────────────────────────────────────────────────────────────────

-- 1. Organizations — bring the pre-existing (empty) table up to spec.
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS owner_id VARCHAR(255);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500);
CREATE INDEX IF NOT EXISTS idx_organizations_owner ON organizations(owner_id);

-- 2. Organization members — normalize column name to organization_id (table is empty).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='organization_members' AND column_name='org_id')
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='organization_members' AND column_name='organization_id') THEN
        ALTER TABLE organization_members RENAME COLUMN org_id TO organization_id;
    END IF;
END $$;
CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS invited_by VARCHAR(255);
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE organization_members ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
CREATE UNIQUE INDEX IF NOT EXISTS uq_org_members_org_user ON organization_members(organization_id, user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);

-- 3. Organization invites
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

-- 4. Link workspaces to their organization
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_workspaces_org ON workspaces(organization_id);

-- 5. Teams — inside a workspace
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(20),
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_teams_workspace ON teams(workspace_id);

-- 6. Team memberships — move users between teams
CREATE TABLE IF NOT EXISTS team_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'member',
    added_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_team_memberships_team ON team_memberships(team_id);
CREATE INDEX IF NOT EXISTS idx_team_memberships_user ON team_memberships(user_id);

-- 7. Backfill: each workspace owner gets a personal org; workspaces link to it;
--    existing workspace members become org members.
DO $$
DECLARE
    o RECORD;
    new_org_id UUID;
BEGIN
    FOR o IN SELECT DISTINCT owner_id FROM workspaces WHERE organization_id IS NULL LOOP
        INSERT INTO organizations (name, slug, owner_id)
        VALUES (
            'My Organization',
            'org-' || substr(md5(o.owner_id || clock_timestamp()::text), 1, 12),
            o.owner_id
        )
        RETURNING id INTO new_org_id;

        INSERT INTO organization_members (organization_id, user_id, role, status)
        VALUES (new_org_id, o.owner_id, 'owner', 'active')
        ON CONFLICT (organization_id, user_id) DO NOTHING;

        UPDATE workspaces
        SET organization_id = new_org_id
        WHERE owner_id = o.owner_id AND organization_id IS NULL;

        INSERT INTO organization_members (organization_id, user_id, role, status)
        SELECT DISTINCT new_org_id, wm.user_id, 'member', 'active'
        FROM workspace_members wm
        JOIN workspaces ws ON wm.workspace_id = ws.id
        WHERE ws.organization_id = new_org_id
          AND wm.user_id <> o.owner_id
          AND wm.status = 'active'
        ON CONFLICT (organization_id, user_id) DO NOTHING;
    END LOOP;
END $$;
