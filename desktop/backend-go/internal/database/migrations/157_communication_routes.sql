-- Explicitly route external communications into governed BusinessOS workspaces.
-- Unassigned sources remain readable but never enter Optimal Engine.
CREATE TABLE IF NOT EXISTS communication_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL
        CHECK (provider IN ('gmail', 'outlook', 'slack', 'teams', 'whatsapp')),
    scope VARCHAR(24) NOT NULL
        CHECK (scope IN ('account', 'conversation')),
    external_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider, scope, external_id)
);

CREATE INDEX IF NOT EXISTS idx_communication_routes_resolve
    ON communication_routes(user_id, provider, scope, external_id)
    WHERE enabled;
CREATE INDEX IF NOT EXISTS idx_communication_routes_workspace
    ON communication_routes(workspace_id);
