-- Migration: 103_platform_admin.sql
-- Description: Platform-level admin tables for BusinessOS multi-tenant SaaS.
--              Adds superadmin role, global settings, cross-tenant computer tracking,
--              Stripe subscriptions, platform audit log, and API key management.
-- Created: 2026-04-11

-- ── Platform role on the user table ──────────────────────────────────────────
-- Three-tier platform RBAC: superadmin → admin (workspace owner) → user.
-- Applied with IF NOT EXISTS guard so re-runs are safe.
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS platform_role VARCHAR(20) DEFAULT 'user'
    CHECK (platform_role IN ('superadmin', 'admin', 'user'));

-- ── Platform settings ────────────────────────────────────────────────────────
-- Global configuration key-value store. Single source of truth for platform-wide
-- flags, limits, and feature switches.
CREATE TABLE IF NOT EXISTS platform_settings (
    key        VARCHAR(255) PRIMARY KEY,
    value      JSONB        NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by VARCHAR(255) REFERENCES "user"(id) ON DELETE SET NULL
);

-- ── Platform computers ───────────────────────────────────────────────────────
-- Tracks every cloud VM across all workspaces. Single queryable surface for the
-- admin dashboard without scattering across per-workspace tables.
CREATE TABLE IF NOT EXISTS platform_computers (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    owner_user_id    VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    miosa_computer_id VARCHAR(255),              -- MIOSA VM ID (nullable until provisioned)
    miosa_slug       VARCHAR(255),
    plan             VARCHAR(50)  NOT NULL DEFAULT 'free',
    status           VARCHAR(50)  NOT NULL DEFAULT 'inactive',
    ram_mb           INTEGER      NOT NULL DEFAULT 0,
    vcpus            INTEGER      NOT NULL DEFAULT 0,
    storage_gb       INTEGER      NOT NULL DEFAULT 0,
    auto_stop        BOOLEAN      NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_active_at   TIMESTAMPTZ,
    metadata         JSONB        NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_platform_computers_workspace ON platform_computers(workspace_id);
CREATE INDEX IF NOT EXISTS idx_platform_computers_owner     ON platform_computers(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_platform_computers_status    ON platform_computers(status);

-- ── Platform subscriptions ───────────────────────────────────────────────────
-- One row per workspace. Stripe customer/subscription IDs + plan state.
-- seats_limit = 0 means unlimited.
CREATE TABLE IF NOT EXISTS platform_subscriptions (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id         UUID         NOT NULL UNIQUE REFERENCES workspaces(id) ON DELETE CASCADE,
    stripe_customer_id   VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    plan                 VARCHAR(50)  NOT NULL DEFAULT 'free',
    status               VARCHAR(50)  NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'past_due', 'cancelled', 'trialing')),
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,
    credits_total        INTEGER      NOT NULL DEFAULT 0,
    credits_used         INTEGER      NOT NULL DEFAULT 0,
    seats_limit          INTEGER      NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_subscriptions_stripe
    ON platform_subscriptions(stripe_customer_id);

-- ── Platform audit log ───────────────────────────────────────────────────────
-- Append-only record of all superadmin/admin actions. Never updated, never deleted.
-- target_type/target_id reference the affected entity by convention (no FK, intentional —
-- audit entries must survive the deletion of the target).
CREATE TABLE IF NOT EXISTS platform_audit_log (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    actor_role  VARCHAR(20)  NOT NULL,
    action      VARCHAR(255) NOT NULL,
    target_type VARCHAR(100),                    -- 'user' | 'workspace' | 'computer' | 'subscription'
    target_id   VARCHAR(255),
    details     JSONB,
    ip_address  VARCHAR(45),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_audit_log_actor
    ON platform_audit_log(actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_audit_log_target
    ON platform_audit_log(target_type, target_id);

-- ── Platform API keys ────────────────────────────────────────────────────────
-- Per-workspace API keys for programmatic access to BusinessOS.
-- key_hash stores a SHA-256 hash of the raw key — the plaintext is never persisted.
-- key_prefix ("bos_xxxx") is stored to help users identify keys without exposing them.
CREATE TABLE IF NOT EXISTS platform_api_keys (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id        VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    key_hash       VARCHAR(255) NOT NULL UNIQUE,
    key_prefix     VARCHAR(20)  NOT NULL,
    name           VARCHAR(255) NOT NULL DEFAULT 'API Key',
    scopes         TEXT[]       NOT NULL DEFAULT '{}',
    rate_limit_rpm INTEGER      NOT NULL DEFAULT 60,
    status         VARCHAR(20)  NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked')),
    last_used_at   TIMESTAMPTZ,
    expires_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_api_keys_hash
    ON platform_api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_platform_api_keys_workspace
    ON platform_api_keys(workspace_id);
