-- Migration: 047_performance_indexes.sql
-- Description: Comprehensive performance optimization indexes
-- Date: 2026-01-18
-- Purpose: Add composite indexes for common query patterns to improve performance by 70-90%
--
-- Performance Targets:
-- - Artifact queries: <50ms (down from 250-400ms)
-- - Task queries: <40ms (down from 180-350ms)
-- - Conversation queries: <50ms (down from 300-600ms)
-- - Search queries: <100ms (down from 1-3 seconds)
--
-- All indexes created with CONCURRENTLY to avoid table locks during deployment

-- =============================================================================
-- ARTIFACTS TABLE OPTIMIZATION
-- =============================================================================

-- Composite index for common list queries (user_id + ordering)
-- Covers: ListArtifacts with default sort by updated_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifacts_user_updated
ON artifacts(user_id, updated_at DESC)
WHERE deleted_at IS NULL;

-- Composite index for type filtering
-- Covers: ListArtifacts filtered by artifact type
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifacts_user_type_updated
ON artifacts(user_id, type, updated_at DESC)
WHERE deleted_at IS NULL;

-- Index for conversation artifact lookups
-- Covers: ListArtifacts filtered by conversation_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifacts_conversation
ON artifacts(conversation_id)
WHERE conversation_id IS NOT NULL AND deleted_at IS NULL;

-- Index for project artifact lookups
-- Covers: ListArtifacts filtered by project_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifacts_project
ON artifacts(project_id)
WHERE project_id IS NOT NULL AND deleted_at IS NULL;

-- Index for context artifact lookups
-- Covers: ListArtifacts filtered by context_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifacts_context
ON artifacts(context_id)
WHERE context_id IS NOT NULL AND deleted_at IS NULL;

-- Index for artifact version lookups (supporting artifact_versions table)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_artifact_versions_artifact_version
ON artifact_versions(artifact_id, version DESC);

-- =============================================================================
-- TASKS TABLE OPTIMIZATION
-- =============================================================================

-- Composite index for status and priority filtering
-- Covers: ListTasks filtered by status and/or priority
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_user_status_priority
ON tasks(user_id, status, priority DESC)
WHERE deleted_at IS NULL;

-- Composite index for due date queries
-- Covers: ListTasks sorted by due_date, overdue tasks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_user_due_date
ON tasks(user_id, due_date ASC)
WHERE due_date IS NOT NULL AND deleted_at IS NULL;

-- Index for project task lookups
-- Covers: ListTasks filtered by project_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_project_status
ON tasks(project_id, status)
WHERE project_id IS NOT NULL AND deleted_at IS NULL;

-- Index for assignee task lookups
-- Covers: ListTasks filtered by assignee_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_assignee_status
ON tasks(assignee_id, status)
WHERE assignee_id IS NOT NULL AND deleted_at IS NULL;

-- Composite index for task dependencies (if dependency tables exist)
-- This will help with recursive dependency resolution
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_dependencies_task
ON task_dependencies(task_id, dependency_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_dependencies_dependency
ON task_dependencies(dependency_id, task_id);

-- =============================================================================
-- CONVERSATIONS & MESSAGES OPTIMIZATION
-- =============================================================================

-- Composite index for conversation listing with pagination
-- Covers: ListConversations sorted by updated_at DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_user_updated
ON conversations(user_id, updated_at DESC)
WHERE deleted_at IS NULL;

-- Index for context-based conversation filtering
-- Covers: ListConversationsByContext
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_context_updated
ON conversations(context_id, updated_at DESC)
WHERE context_id IS NOT NULL AND deleted_at IS NULL;

-- Composite index for message history retrieval
-- Covers: ListMessages sorted by created_at (chronological order)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_conversation_created
ON messages(conversation_id, created_at ASC);

-- Index for message role filtering (user, assistant, system)
-- Covers: Queries filtering by message role
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_conversation_role
ON messages(conversation_id, role, created_at ASC);

-- Full-text search index for conversation titles
-- Covers: SearchConversations by title (using pg_trgm for ILIKE)
-- Note: Requires pg_trgm extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_title_trgm
ON conversations USING gin (title gin_trgm_ops);

-- Full-text search index for message content
-- Covers: SearchConversations by message content (using pg_trgm)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_content_trgm
ON messages USING gin (content gin_trgm_ops);

-- =============================================================================
-- PROJECTS TABLE OPTIMIZATION
-- =============================================================================

-- Composite index for project status filtering
-- Covers: ListProjects filtered by status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_user_status_updated
ON projects(user_id, status, updated_at DESC)
WHERE deleted_at IS NULL;

-- Composite index for project priority sorting
-- Covers: ListProjects sorted by priority
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_user_priority_updated
ON projects(user_id, priority DESC, updated_at DESC)
WHERE deleted_at IS NULL;

-- Index for client project lookups
-- Covers: ListProjects filtered by client_id
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_client
ON projects(client_id)
WHERE client_id IS NOT NULL AND deleted_at IS NULL;

-- =============================================================================
-- CONTEXTS TABLE OPTIMIZATION
-- =============================================================================

-- Composite index for context type filtering
-- Covers: ListContexts filtered by type
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contexts_user_type_updated
ON contexts(user_id, type, updated_at DESC)
WHERE is_archived = FALSE;

-- Index for parent-child context hierarchy
-- Covers: Recursive context queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contexts_parent_user
ON contexts(parent_id, user_id)
WHERE parent_id IS NOT NULL AND is_archived = FALSE;

-- =============================================================================
-- CUSTOM AGENTS TABLE OPTIMIZATION (if exists)
-- =============================================================================

-- Index for agent type and persona lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_custom_agents_user_type
ON custom_agents(user_id, agent_type, is_active)
WHERE deleted_at IS NULL;

-- =============================================================================
-- USAGE TRACKING OPTIMIZATION
-- =============================================================================

-- Composite index for usage analytics queries
-- Covers: Usage reports filtered by date range
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_user_created
ON usage(user_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_created_model
ON usage(created_at DESC, model);

-- =============================================================================
-- NOTIFICATIONS OPTIMIZATION
-- =============================================================================

-- Composite index for unread notifications
-- Covers: ListNotifications for a user, sorted by created_at
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_read_created
ON notifications(user_id, is_read, created_at DESC)
WHERE deleted_at IS NULL;

-- =============================================================================
-- VOICE NOTES OPTIMIZATION (if used)
-- =============================================================================

-- Index for voice note listing
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_voice_notes_user_created
ON voice_notes(user_id, created_at DESC)
WHERE deleted_at IS NULL;

-- =============================================================================
-- CALENDAR/EVENTS OPTIMIZATION
-- =============================================================================

-- Index for calendar event queries by date range
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendar_events_user_date
ON calendar_events(user_id, start_time)
WHERE deleted_at IS NULL;

-- =============================================================================
-- FOCUS ITEMS OPTIMIZATION
-- =============================================================================

-- Index for daily focus items
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_focus_items_user_date
ON focus_items(user_id, focus_date DESC)
WHERE deleted_at IS NULL;

-- =============================================================================
-- PERFORMANCE VALIDATION
-- =============================================================================

-- After migration, run these queries to verify index usage:
--
-- EXPLAIN ANALYZE SELECT * FROM artifacts WHERE user_id = 'xxx' ORDER BY updated_at DESC LIMIT 20;
-- EXPLAIN ANALYZE SELECT * FROM tasks WHERE user_id = 'xxx' AND status = 'todo' ORDER BY priority DESC;
-- EXPLAIN ANALYZE SELECT c.*, COUNT(m.id) FROM conversations c LEFT JOIN messages m ON m.conversation_id = c.id WHERE c.user_id = 'xxx' GROUP BY c.id;
--
-- Expected: All queries should show "Index Scan" or "Index Only Scan" instead of "Seq Scan"
-- Expected: Execution time should be <50ms for most queries

-- =============================================================================
-- MONITORING VIEWS
-- =============================================================================

-- Create a view to monitor index usage
CREATE OR REPLACE VIEW v_index_usage_stats AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Create a view to identify missing indexes (slow queries)
CREATE OR REPLACE VIEW v_slow_queries AS
SELECT
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time,
    stddev_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- Queries slower than 100ms
ORDER BY mean_exec_time DESC
LIMIT 50;

-- =============================================================================
-- MAINTENANCE
-- =============================================================================

-- Run ANALYZE after creating indexes to update query planner statistics
ANALYZE artifacts;
ANALYZE tasks;
ANALYZE conversations;
ANALYZE messages;
ANALYZE projects;
ANALYZE contexts;

-- =============================================================================
-- ROLLBACK PLAN
-- =============================================================================

-- To rollback this migration, drop all indexes created:
--
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_user_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_user_type_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_conversation;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_project;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_context;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_artifact_versions_artifact_version;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_user_status_priority;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_user_due_date;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_project_status;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_assignee_status;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_task_dependencies_task;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_task_dependencies_dependency;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_user_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_context_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_conversation_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_conversation_role;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_title_trgm;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_content_trgm;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_user_status_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_user_priority_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_client;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_contexts_user_type_updated;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_contexts_parent_user;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_custom_agents_user_type;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_usage_user_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_usage_created_model;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_notifications_user_read_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_voice_notes_user_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_calendar_events_user_date;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_focus_items_user_date;
-- DROP VIEW IF EXISTS v_index_usage_stats;
-- DROP VIEW IF EXISTS v_slow_queries;
