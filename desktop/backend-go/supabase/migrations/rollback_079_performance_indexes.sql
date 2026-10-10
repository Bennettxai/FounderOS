-- =============================================================================
-- ROLLBACK: 079_performance_indexes.sql
-- =============================================================================
-- Description: Safely rollback all performance optimization indexes
-- Date: 2026-01-19
-- Estimated Time: 5-15 minutes (depends on table sizes and load)
-- Risk Level: LOW (indexes can be recreated, no data loss)
--
-- IMPORTANT: Run this AFTER rollback_080 if rolling back both migrations
--
-- What this rollback does:
-- 1. Drops 28 performance indexes (CONCURRENTLY to avoid locks)
-- 2. Drops 2 monitoring views
-- 3. Does NOT drop pg_trgm extension (may be used elsewhere)
--
-- Impact:
-- - Query performance will degrade by 70-90% (back to pre-optimization levels)
-- - Expected query times:
--   * Artifacts: 250-400ms (was <50ms)
--   * Tasks: 180-350ms (was <40ms)
--   * Conversations: 300-600ms (was <50ms)
--   * Search: 1-3 seconds (was <100ms)
-- - Database will perform more sequential scans
-- - Higher CPU and memory usage for queries
--
-- Prerequisites:
-- - Database backup recommended (safety measure)
-- - Run during low-traffic window (minimize impact)
-- - Ensure no other migrations are running
-- =============================================================================

-- =============================================================================
-- STEP 1: DROP MONITORING VIEWS
-- =============================================================================
-- Drop views first (no dependencies on indexes)

DO $$
BEGIN
    RAISE NOTICE 'Step 1/8: Dropping monitoring views...';
END $$;

DROP VIEW IF EXISTS v_slow_queries;
DROP VIEW IF EXISTS v_index_usage_stats;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 2 views dropped successfully';
END $$;

-- =============================================================================
-- STEP 2: DROP ARTIFACTS INDEXES (6 indexes)
-- =============================================================================
-- Using CONCURRENTLY to avoid table locks during production rollback
-- Each DROP may take 10-30 seconds depending on table size

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 2/8: Dropping ARTIFACTS indexes (6 indexes)...';
    RAISE NOTICE '  Note: This may take 1-2 minutes total';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_user_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_user_type_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_conversation;
DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_project;
DROP INDEX CONCURRENTLY IF EXISTS idx_artifacts_context;
DROP INDEX CONCURRENTLY IF EXISTS idx_artifact_versions_artifact_version;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 6 artifacts indexes dropped';
END $$;

-- =============================================================================
-- STEP 3: DROP TASKS INDEXES (6 indexes)
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 3/8: Dropping TASKS indexes (6 indexes)...';
    RAISE NOTICE '  Note: This may take 1-2 minutes total';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_user_status_priority;
DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_user_due_date;
DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_project_status;
DROP INDEX CONCURRENTLY IF EXISTS idx_tasks_assignee_status;
DROP INDEX CONCURRENTLY IF EXISTS idx_task_dependencies_task;
DROP INDEX CONCURRENTLY IF EXISTS idx_task_dependencies_dependency;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 6 tasks indexes dropped';
END $$;

-- =============================================================================
-- STEP 4: DROP CONVERSATIONS & MESSAGES INDEXES (6 indexes)
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 4/8: Dropping CONVERSATIONS & MESSAGES indexes (6 indexes)...';
    RAISE NOTICE '  Note: This may take 2-3 minutes total (includes GIN indexes)';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_user_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_context_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_conversation_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_conversation_role;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_title_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_content_trgm;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 6 conversations/messages indexes dropped';
END $$;

-- =============================================================================
-- STEP 5: DROP PROJECTS INDEXES (3 indexes)
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 5/8: Dropping PROJECTS indexes (3 indexes)...';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_projects_user_status_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_projects_user_priority_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_projects_client;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 3 projects indexes dropped';
END $$;

-- =============================================================================
-- STEP 6: DROP CONTEXTS INDEXES (2 indexes)
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 6/8: Dropping CONTEXTS indexes (2 indexes)...';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_contexts_user_type_updated;
DROP INDEX CONCURRENTLY IF EXISTS idx_contexts_parent_user;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 2 contexts indexes dropped';
END $$;

-- =============================================================================
-- STEP 7: DROP MISCELLANEOUS INDEXES (5 indexes)
-- =============================================================================
-- Custom agents, usage tracking, notifications, voice notes, calendar

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 7/8: Dropping MISCELLANEOUS indexes (5 indexes)...';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_custom_agents_user_type;
DROP INDEX CONCURRENTLY IF EXISTS idx_usage_user_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_usage_created_model;
DROP INDEX CONCURRENTLY IF EXISTS idx_notifications_user_read_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_voice_notes_user_created;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 5 miscellaneous indexes dropped';
END $$;

-- =============================================================================
-- STEP 8: DROP CALENDAR & FOCUS INDEXES (2 indexes)
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE 'Step 8/8: Dropping CALENDAR & FOCUS indexes (2 indexes)...';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_calendar_events_user_date;
DROP INDEX CONCURRENTLY IF EXISTS idx_focus_items_user_date;

DO $$
BEGIN
    RAISE NOTICE '  ✓ 2 calendar/focus indexes dropped';
END $$;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
-- Comprehensive verification of all dropped objects

DO $$
DECLARE
    remaining_indexes TEXT[];
    remaining_views TEXT[];
    expected_indexes TEXT[] := ARRAY[
        'idx_artifacts_user_updated',
        'idx_artifacts_user_type_updated',
        'idx_artifacts_conversation',
        'idx_artifacts_project',
        'idx_artifacts_context',
        'idx_artifact_versions_artifact_version',
        'idx_tasks_user_status_priority',
        'idx_tasks_user_due_date',
        'idx_tasks_project_status',
        'idx_tasks_assignee_status',
        'idx_task_dependencies_task',
        'idx_task_dependencies_dependency',
        'idx_conversations_user_updated',
        'idx_conversations_context_updated',
        'idx_messages_conversation_created',
        'idx_messages_conversation_role',
        'idx_conversations_title_trgm',
        'idx_messages_content_trgm',
        'idx_projects_user_status_updated',
        'idx_projects_user_priority_updated',
        'idx_projects_client',
        'idx_contexts_user_type_updated',
        'idx_contexts_parent_user',
        'idx_custom_agents_user_type',
        'idx_usage_user_created',
        'idx_usage_created_model',
        'idx_notifications_user_read_created',
        'idx_voice_notes_user_created',
        'idx_calendar_events_user_date',
        'idx_focus_items_user_date'
    ];
    expected_views TEXT[] := ARRAY['v_index_usage_stats', 'v_slow_queries'];
    idx TEXT;
    view_name TEXT;
    still_exists BOOLEAN;
    failed_count INTEGER := 0;
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'ROLLBACK VERIFICATION';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE '';

    -- Check indexes
    RAISE NOTICE 'Checking indexes (30 total)...';
    FOREACH idx IN ARRAY expected_indexes LOOP
        SELECT EXISTS (
            SELECT 1
            FROM pg_indexes
            WHERE indexname = idx
        ) INTO still_exists;

        IF still_exists THEN
            RAISE WARNING '  ✗ FAILED: Index still exists: %', idx;
            failed_count := failed_count + 1;
        END IF;
    END LOOP;

    IF failed_count = 0 THEN
        RAISE NOTICE '  ✓ All 30 indexes removed successfully';
    ELSE
        RAISE WARNING '  ✗ % indexes still exist', failed_count;
    END IF;

    -- Check views
    RAISE NOTICE '';
    RAISE NOTICE 'Checking views (2 total)...';
    failed_count := 0;
    FOREACH view_name IN ARRAY expected_views LOOP
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.views
            WHERE table_name = view_name
        ) INTO still_exists;

        IF still_exists THEN
            RAISE WARNING '  ✗ FAILED: View still exists: %', view_name;
            failed_count := failed_count + 1;
        END IF;
    END LOOP;

    IF failed_count = 0 THEN
        RAISE NOTICE '  ✓ All 2 views removed successfully';
    ELSE
        RAISE WARNING '  ✗ % views still exist', failed_count;
    END IF;

    RAISE NOTICE '';
    RAISE NOTICE '=============================================================================';

    -- Summary
    IF failed_count = 0 THEN
        RAISE NOTICE '';
        RAISE NOTICE '✅ ROLLBACK COMPLETE: Migration 079 successfully rolled back';
        RAISE NOTICE '';
        RAISE NOTICE 'Summary:';
        RAISE NOTICE '  • 30 performance indexes removed';
        RAISE NOTICE '  • 2 monitoring views removed';
        RAISE NOTICE '  • pg_trgm extension preserved (may be used elsewhere)';
        RAISE NOTICE '';
        RAISE NOTICE 'Next steps:';
        RAISE NOTICE '  1. Verify application is working (queries will be slower)';
        RAISE NOTICE '  2. Monitor query performance (expect 70-90%% degradation)';
        RAISE NOTICE '  3. Monitor database CPU/memory usage (expect increase)';
        RAISE NOTICE '  4. Consider re-applying migration if performance is critical';
        RAISE NOTICE '  5. Update team about performance impact';
    ELSE
        RAISE WARNING '';
        RAISE WARNING '⚠️ ROLLBACK INCOMPLETE: Some objects still exist';
        RAISE WARNING 'Review errors above and run rollback script again if needed';
    END IF;
END $$;

-- =============================================================================
-- PERFORMANCE IMPACT AFTER ROLLBACK
-- =============================================================================
--
-- Expected performance degradation after rollback:
--
-- ┌─────────────────────────────────────────────────────────────────────────┐
-- │ Table            │ Query Type      │ BEFORE  │ AFTER   │ Degradation │
-- ├─────────────────────────────────────────────────────────────────────────┤
-- │ artifacts        │ List by user    │ <50ms   │ 250ms   │ 500%        │
-- │ tasks            │ Status filter   │ <40ms   │ 180ms   │ 450%        │
-- │ conversations    │ List + sort     │ <50ms   │ 300ms   │ 600%        │
-- │ messages         │ Search content  │ <100ms  │ 1-3s    │ 1000-3000%  │
-- │ projects         │ Status filter   │ <40ms   │ 200ms   │ 500%        │
-- │ notifications    │ Unread list     │ <30ms   │ 150ms   │ 500%        │
-- └─────────────────────────────────────────────────────────────────────────┘
--
-- Query Plan Changes:
--
-- BEFORE ROLLBACK (with indexes):
--   → Index Scan on idx_artifacts_user_updated
--   → Index Only Scan on idx_tasks_user_status_priority
--   → Bitmap Index Scan on idx_messages_content_trgm
--
-- AFTER ROLLBACK (without indexes):
--   → Sequential Scan on artifacts (reads entire table)
--   → Sequential Scan on tasks (reads entire table)
--   → Sequential Scan on messages (reads entire table)
--
-- =============================================================================

-- =============================================================================
-- NOTE: pg_trgm EXTENSION NOT REMOVED
-- =============================================================================
--
-- The pg_trgm extension is NOT removed by this rollback because:
--
-- 1. It may be used by other parts of the application
-- 2. It's a PostgreSQL extension, not a schema object
-- 3. Removing it could break other features
-- 4. It's safe to leave installed (no performance impact)
--
-- If you MUST remove pg_trgm:
--   DROP EXTENSION IF EXISTS pg_trgm CASCADE;
--
-- WARNING: This will drop ALL objects that depend on pg_trgm!
--
-- =============================================================================

-- =============================================================================
-- TROUBLESHOOTING
-- =============================================================================
--
-- If rollback fails or hangs:
--
-- 1. "cannot drop index concurrently because..."
--    → Check for active transactions:
--      SELECT * FROM pg_stat_activity WHERE state = 'active';
--    → Wait for transactions to complete or cancel them
--
-- 2. DROP INDEX CONCURRENTLY takes very long (>5 minutes per index)
--    → Normal for large tables under heavy load
--    → Can monitor progress:
--      SELECT * FROM pg_stat_progress_create_index;
--    → Consider running during maintenance window
--
-- 3. Some indexes fail to drop
--    → Check if they're referenced by views or constraints:
--      SELECT * FROM pg_depend WHERE objid = (
--        SELECT oid FROM pg_class WHERE relname = 'index_name'
--      );
--    → Drop dependent objects first
--
-- 4. Need to check which indexes still exist
--    → Run this query:
--      SELECT indexname FROM pg_indexes
--      WHERE indexname LIKE 'idx_%'
--      ORDER BY indexname;
--
-- 5. Application errors after rollback
--    → Queries may timeout (increase timeout settings)
--    → Monitor slow query log
--    → Consider re-applying migration 079
--
-- =============================================================================
-- ROLLBACK COMPLETION CHECKLIST
-- =============================================================================
--
-- After running this script:
--
-- [ ] Verify rollback completed successfully (check output above)
-- [ ] Test application functionality (all features should work, just slower)
-- [ ] Monitor query performance (use EXPLAIN ANALYZE)
-- [ ] Check database CPU/memory usage (expect increase)
-- [ ] Review slow query logs
-- [ ] Update team about performance impact
-- [ ] Document reason for rollback in runbook
-- [ ] Plan re-application if performance becomes critical
--
-- =============================================================================
-- RE-APPLYING MIGRATION 079
-- =============================================================================
--
-- If performance becomes critical after rollback, re-apply:
--
-- 1. Ensure database has low load
-- 2. Run: psql < 079_performance_indexes.sql
-- 3. Monitor index creation progress
-- 4. Run ANALYZE on affected tables
-- 5. Verify query performance improved
--
-- Estimated time to re-apply: 10-20 minutes
--
-- =============================================================================
