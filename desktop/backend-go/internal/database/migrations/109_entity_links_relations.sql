-- 109_entity_links_relations.sql
-- Make the generic relations layer (entity_links) workspace-aware so any module
-- can relate its core primitive to any other module's primitive by ID, scoped to
-- a workspace and shareable with the team. NULL workspace_id = personal/local link.
--
-- This is the one primitive that lets modules with DIFFERENT core primitives
-- (client / project / event / deal ...) compose by reference instead of by
-- duplicating data. Each module still owns its own table; this only stores pointers.

ALTER TABLE entity_links ADD COLUMN IF NOT EXISTS workspace_id UUID;

-- Lookups: "all links in this workspace touching entity X" (source or target).
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_source
    ON entity_links(workspace_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_entity_links_ws_target
    ON entity_links(workspace_id, target_type, target_id);
