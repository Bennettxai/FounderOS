-- Workspace Invitations (Magic Links)
-- Note: Table may already exist from migration 028_workspaces.sql
-- +goose Up
CREATE TABLE IF NOT EXISTS workspace_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    
    -- Invitation Details
    email VARCHAR(255) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,        -- Secure random token (32 bytes hex)
    
    -- Role to assign on accept
    role_id UUID REFERENCES workspace_roles(id) ON DELETE SET NULL,
    role_name VARCHAR(100) NOT NULL,          -- Denormalized for display
    
    -- Inviter Information
    invited_by_id VARCHAR(255) NOT NULL,
    invited_by_name VARCHAR(255),
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- 'pending', 'accepted', 'expired', 'revoked'
    
    -- Timestamps
    expires_at TIMESTAMPTZ NOT NULL,          -- Default: created_at + 7 days
    accepted_at TIMESTAMPTZ,
    accepted_by_user_id VARCHAR(255),         -- The user who accepted
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_token ON workspace_invitations(token);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_workspace ON workspace_invitations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_email ON workspace_invitations(email);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_status ON workspace_invitations(status) WHERE status = 'pending';

-- Partial unique index: only one pending invitation per email per workspace
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_invitations_pending_unique 
ON workspace_invitations(workspace_id, email) 
WHERE status = 'pending';

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_workspace_invitations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_workspace_invitations_updated_at ON workspace_invitations;
CREATE TRIGGER trigger_workspace_invitations_updated_at
    BEFORE UPDATE ON workspace_invitations
    FOR EACH ROW
    EXECUTE FUNCTION update_workspace_invitations_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_workspace_invitations_updated_at ON workspace_invitations;
DROP FUNCTION IF EXISTS update_workspace_invitations_updated_at();
DROP TABLE IF EXISTS workspace_invitations;
