-- Migration 098: Drop orphaned and dead tables
-- These tables have ZERO references in Go handler/service code.
-- Verified via full codebase grep: no .go file queries these tables.
--
-- Categories:
--   1. Feature stubs never implemented (20 tables)
--   2. Deleted sync module remnants (2 tables)
--   3. SQLC-only tables with no handler/service callers (2 tables)
--
-- Integration tables are intentionally KEPT (ClickUp, Airtable, HubSpot, Fathom, etc.)

BEGIN;

-- ============================================================================
-- 1. FULLY ORPHANED: Zero references in any .go file
-- ============================================================================

-- Backup table from migration, no longer needed
DROP TABLE IF EXISTS embeddings_backup_20260129 CASCADE;

-- Exact duplicate of workspace_invites (which has 19 active refs)
DROP TABLE IF EXISTS workspace_invitations CASCADE;

-- Superseded by agent_context_sessions
DROP TABLE IF EXISTS context_sessions CASCADE;

-- Feature never implemented
DROP TABLE IF EXISTS meeting_recordings CASCADE;

-- Voice command audit log never wired
DROP TABLE IF EXISTS voice_commands_log CASCADE;

-- Node telemetry never implemented
DROP TABLE IF EXISTS node_metrics CASCADE;

-- Tenant/org system — no code exists yet
DROP TABLE IF EXISTS organization_members CASCADE;

-- Project-chat linking never built
DROP TABLE IF EXISTS project_conversations CASCADE;

-- Feature stubs (only in init.sql/schema.sql, no migration, no code)
DROP TABLE IF EXISTS project_documents CASCADE;
DROP TABLE IF EXISTS project_templates CASCADE;

-- Old tag systems superseded by universal tags + tag_assignments
DROP TABLE IF EXISTS project_tag_assignments CASCADE;
DROP TABLE IF EXISTS project_tags CASCADE;
DROP TABLE IF EXISTS conversation_tags CASCADE;

-- Focus system stubs, never wired to any handler
DROP TABLE IF EXISTS focus_configuration_presets CASCADE;
DROP TABLE IF EXISTS focus_context_presets CASCADE;

-- Integration OAuth pending state, never used
DROP TABLE IF EXISTS integration_pending_connections CASCADE;

-- Memory graph feature never built
DROP TABLE IF EXISTS memory_associations CASCADE;

-- Document cross-referencing never implemented
DROP TABLE IF EXISTS document_references CASCADE;

-- Context retrieval logging never implemented
DROP TABLE IF EXISTS context_retrieval_log CASCADE;

-- Module dependency graph never wired
DROP TABLE IF EXISTS osa_module_dependencies CASCADE;

-- ============================================================================
-- 2. DELETED SYNC MODULE: internal/sync/ was removed, SQLC wrappers are dead
--    NOTE: sync_outbox is KEPT — still used by osa_sync_service_*.go
-- ============================================================================

DROP TABLE IF EXISTS sync_conflicts CASCADE;
DROP TABLE IF EXISTS sync_dlq CASCADE;

-- ============================================================================
-- 3. SQLC-ONLY: Generated query wrappers exist but no handler/service imports them
-- ============================================================================

-- Entity linking system — SQLC generated, zero handler imports
DROP TABLE IF EXISTS entity_mentions CASCADE;
DROP TABLE IF EXISTS entity_links CASCADE;

COMMIT;
