-- ═══════════════════════════════════════════════════════════════════════════════
-- Tenant / Organization Foundation Migration
-- ═══════════════════════════════════════════════════════════════════════════════
-- Created: 2026-02-23
-- Description: Establishes the multi-tenant organization layer.
--              Creates organizations and organization_members tables,
--              then links osa_workspaces and custom_modules via a nullable
--              organization_id FK so both tables remain backward-compatible.
--
-- Design notes:
--   • No RLS policies are applied here; enforcement is left to the
--     application layer or a follow-up migration.
--   • organization_id is nullable on both linked tables so existing rows
--     continue to work without backfill.
--   • plan is a free-form text column; a CHECK constraint can be added
--     in a later migration once the plan list stabilizes.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: organizations
-- Top-level tenant entity. One organization can contain many workspaces/modules.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS organizations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identity
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL,   -- URL-safe handle, e.g. "acme-corp"

    -- Billing / feature tier
    plan        TEXT        NOT NULL DEFAULT 'free',  -- 'free', 'pro', 'business', 'enterprise'

    -- Flexible org-level configuration (theme, feature flags, limits, etc.)
    settings    JSONB       NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT organizations_slug_unique UNIQUE (slug),
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9][a-z0-9_-]*[a-z0-9]$')
);

-- Indexes for organizations
CREATE INDEX IF NOT EXISTS idx_organizations_slug       ON organizations (slug);
CREATE INDEX IF NOT EXISTS idx_organizations_plan       ON organizations (plan);
CREATE INDEX IF NOT EXISTS idx_organizations_created_at ON organizations (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_organizations_settings   ON organizations USING GIN (settings);

-- ───────────────────────────────────────────────────────────────────────────────
-- Trigger: keep organizations.updated_at current
-- ───────────────────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION update_organizations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_organizations_updated_at();

-- ───────────────────────────────────────────────────────────────────────────────
-- Table: organization_members
-- Maps users to organizations with a role.
-- user_id is TEXT to match the "user".id column type used elsewhere.
-- ───────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS organization_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    org_id      UUID        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL,  -- matches "user".id VARCHAR(255) / TEXT type

    -- Access control
    role        TEXT        NOT NULL DEFAULT 'member',  -- 'owner', 'admin', 'member', 'viewer'

    -- Timestamps
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT organization_members_unique UNIQUE (org_id, user_id),
    CONSTRAINT organization_members_role_check CHECK (
        role IN ('owner', 'admin', 'member', 'viewer')
    )
);

-- Indexes for organization_members
CREATE INDEX IF NOT EXISTS idx_org_members_org_id  ON organization_members (org_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON organization_members (user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_role    ON organization_members (org_id, role);

-- ───────────────────────────────────────────────────────────────────────────────
-- Alter: osa_workspaces — add organization_id (nullable)
-- Allows a workspace to be scoped to an organization while keeping all
-- existing rows intact (no backfill required).
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE osa_workspaces
    ADD COLUMN IF NOT EXISTS organization_id UUID
        REFERENCES organizations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_osa_workspaces_org
    ON osa_workspaces (organization_id)
    WHERE organization_id IS NOT NULL;

-- ───────────────────────────────────────────────────────────────────────────────
-- Alter: custom_modules — add organization_id (nullable)
-- Allows modules to be shared across a whole organization, not just a workspace.
-- ───────────────────────────────────────────────────────────────────────────────
ALTER TABLE custom_modules
    ADD COLUMN IF NOT EXISTS organization_id UUID
        REFERENCES organizations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_custom_modules_org
    ON custom_modules (organization_id)
    WHERE organization_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- Comments for documentation
-- ═══════════════════════════════════════════════════════════════════════════════

COMMENT ON TABLE organizations IS
    'Top-level tenant entity. Groups users, workspaces, and modules under a shared billing/plan context.';

COMMENT ON COLUMN organizations.slug IS
    'URL-safe, lowercase handle unique to each organization (e.g. "acme-corp").';

COMMENT ON COLUMN organizations.plan IS
    'Billing tier: free | pro | business | enterprise. Enforced by application layer.';

COMMENT ON COLUMN organizations.settings IS
    'Flexible JSONB bag for org-level config: feature flags, theme overrides, usage limits, etc.';

COMMENT ON TABLE organization_members IS
    'Many-to-many join between users and organizations with role-based access control.';

COMMENT ON COLUMN organization_members.user_id IS
    'References "user".id. Stored as TEXT to match the existing user ID column type.';

COMMENT ON COLUMN organization_members.role IS
    'Access tier: owner (full control), admin (manage members/settings), member (standard), viewer (read-only).';

COMMENT ON COLUMN osa_workspaces.organization_id IS
    'Optional FK to organizations. NULL means the workspace belongs to an individual user only.';

COMMENT ON COLUMN custom_modules.organization_id IS
    'Optional FK to organizations. NULL means the module is scoped to its workspace only.';

-- ═══════════════════════════════════════════════════════════════════════════════
-- END OF MIGRATION
-- ═══════════════════════════════════════════════════════════════════════════════
