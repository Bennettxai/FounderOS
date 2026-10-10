-- Add foreign key constraint to user_dashboards.workspace_id
-- This migration runs after 028_workspaces.sql ensures the workspaces table exists

ALTER TABLE user_dashboards
    ADD CONSTRAINT fk_user_dashboards_workspace
    FOREIGN KEY (workspace_id)
    REFERENCES workspaces(id)
    ON DELETE CASCADE;
