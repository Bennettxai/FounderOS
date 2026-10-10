-- 127_clients_workspace.sql
-- Clients (the Relationships module's company/account records) were purely
-- user_id-scoped, so every client a user owns appeared in EVERY workspace.
-- Wrong model: a client belongs to a specific business context. Add workspace
-- scoping so account records never leak across unrelated workspaces.
--
-- NULL workspace_id = unassigned/personal (kept for backfill safety; the list
-- endpoint shows only the active workspace's clients when X-Workspace-ID is
-- present, so unassigned clients no longer leak into every workspace).
ALTER TABLE clients ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_clients_workspace ON clients(workspace_id);
