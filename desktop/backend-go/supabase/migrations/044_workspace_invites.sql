-- Migration 027: Workspace Email Invitations
-- Description: Adds email invitation system for workspace members

-- Create workspace_invites table
CREATE TABLE IF NOT EXISTS workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    invited_by TEXT NOT NULL, -- user_id of inviter
    token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, accepted, expired, revoked
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace_id ON workspace_invites(workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_email ON workspace_invites(email);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_token ON workspace_invites(token);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_status ON workspace_invites(status);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_expires_at ON workspace_invites(expires_at);

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_workspace_invites_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-update updated_at
CREATE TRIGGER trigger_update_workspace_invites_updated_at
    BEFORE UPDATE ON workspace_invites
    FOR EACH ROW
    EXECUTE FUNCTION update_workspace_invites_updated_at();

-- Add constraint to validate status values
ALTER TABLE workspace_invites
ADD CONSTRAINT check_invite_status
CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'));

-- Add constraint to validate role values
ALTER TABLE workspace_invites
ADD CONSTRAINT check_invite_role
CHECK (role IN ('owner', 'admin', 'manager', 'member', 'viewer', 'guest'));

COMMENT ON TABLE workspace_invites IS 'Stores email invitations for workspace members';
COMMENT ON COLUMN workspace_invites.token IS 'Unique secure token for accepting invitation';
COMMENT ON COLUMN workspace_invites.status IS 'Invitation status: pending, accepted, expired, revoked';
COMMENT ON COLUMN workspace_invites.expires_at IS 'Invitation expiration timestamp (default 7 days)';
