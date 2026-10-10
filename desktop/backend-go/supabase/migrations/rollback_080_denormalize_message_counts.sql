-- =============================================================================
-- ROLLBACK: 080_denormalize_message_counts.sql
-- =============================================================================
-- Description: Safely rollback denormalized message counts
-- Date: 2026-01-19
-- Estimated Time: 2-5 minutes (depends on table size)
-- Risk Level: LOW (data not lost, computed from messages table)
--
-- IMPORTANT: Run this BEFORE rollback_079 if rolling back both migrations
--
-- What this rollback does:
-- 1. Drops triggers (stop automatic updates)
-- 2. Drops trigger functions
-- 3. Drops the message_count index
-- 4. Drops the message_count column from conversations
--
-- Impact:
-- - Conversation listing will use COUNT(*) again (slower, but functional)
-- - Performance degradation: ~50ms → 300-600ms per list query
-- - No data loss (message_count is computed, not source of truth)
--
-- Prerequisites:
-- - Database backup recommended (safety measure)
-- - No active user sessions preferred (minimize locking)
-- - Run during low-traffic window if possible
-- =============================================================================

-- =============================================================================
-- STEP 1: DROP TRIGGERS
-- =============================================================================
-- Drop triggers first to stop automatic updates
-- This prevents errors when we drop the functions

DO $$
BEGIN
    RAISE NOTICE 'Step 1/4: Dropping triggers...';
END $$;

DROP TRIGGER IF EXISTS trigger_increment_message_count ON messages;
DROP TRIGGER IF EXISTS trigger_decrement_message_count ON messages;

DO $$
BEGIN
    RAISE NOTICE '  ✓ Triggers dropped successfully';
END $$;

-- =============================================================================
-- STEP 2: DROP TRIGGER FUNCTIONS
-- =============================================================================
-- Drop the functions that were called by the triggers

DO $$
BEGIN
    RAISE NOTICE 'Step 2/4: Dropping trigger functions...';
END $$;

DROP FUNCTION IF EXISTS increment_conversation_message_count();
DROP FUNCTION IF EXISTS decrement_conversation_message_count();

DO $$
BEGIN
    RAISE NOTICE '  ✓ Functions dropped successfully';
END $$;

-- =============================================================================
-- STEP 3: DROP INDEX
-- =============================================================================
-- Drop the index on message_count
-- Using CONCURRENTLY to avoid table locks (safe for production)

DO $$
BEGIN
    RAISE NOTICE 'Step 3/4: Dropping index (CONCURRENTLY)...';
    RAISE NOTICE '  Note: This may take 30-60 seconds on large tables';
END $$;

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_message_count;

DO $$
BEGIN
    RAISE NOTICE '  ✓ Index dropped successfully';
END $$;

-- =============================================================================
-- STEP 4: DROP COLUMN
-- =============================================================================
-- Drop the message_count column from conversations table
-- This is a blocking operation but should be quick (<1 second)

DO $$
BEGIN
    RAISE NOTICE 'Step 4/4: Dropping message_count column...';
END $$;

ALTER TABLE conversations DROP COLUMN IF EXISTS message_count;

DO $$
BEGIN
    RAISE NOTICE '  ✓ Column dropped successfully';
END $$;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
-- Verify rollback was successful

DO $$
DECLARE
    column_exists BOOLEAN;
    trigger_count INTEGER;
    function_count INTEGER;
    index_count INTEGER;
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'ROLLBACK VERIFICATION';
    RAISE NOTICE '=============================================================================';

    -- Check column doesn't exist
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'conversations'
        AND column_name = 'message_count'
    ) INTO column_exists;

    IF column_exists THEN
        RAISE WARNING '  ✗ FAILED: message_count column still exists!';
    ELSE
        RAISE NOTICE '  ✓ Column removed: message_count';
    END IF;

    -- Check triggers don't exist
    SELECT COUNT(*)
    INTO trigger_count
    FROM information_schema.triggers
    WHERE event_object_table = 'messages'
    AND trigger_name IN ('trigger_increment_message_count', 'trigger_decrement_message_count');

    IF trigger_count > 0 THEN
        RAISE WARNING '  ✗ FAILED: % triggers still exist!', trigger_count;
    ELSE
        RAISE NOTICE '  ✓ Triggers removed: 2 triggers';
    END IF;

    -- Check functions don't exist
    SELECT COUNT(*)
    INTO function_count
    FROM pg_proc
    WHERE proname IN ('increment_conversation_message_count', 'decrement_conversation_message_count');

    IF function_count > 0 THEN
        RAISE WARNING '  ✗ FAILED: % functions still exist!', function_count;
    ELSE
        RAISE NOTICE '  ✓ Functions removed: 2 functions';
    END IF;

    -- Check index doesn't exist
    SELECT COUNT(*)
    INTO index_count
    FROM pg_indexes
    WHERE indexname = 'idx_conversations_message_count';

    IF index_count > 0 THEN
        RAISE WARNING '  ✗ FAILED: Index still exists!';
    ELSE
        RAISE NOTICE '  ✓ Index removed: idx_conversations_message_count';
    END IF;

    RAISE NOTICE '=============================================================================';

    IF NOT column_exists AND trigger_count = 0 AND function_count = 0 AND index_count = 0 THEN
        RAISE NOTICE '';
        RAISE NOTICE '✅ ROLLBACK COMPLETE: Migration 080 successfully rolled back';
        RAISE NOTICE '';
        RAISE NOTICE 'Next steps:';
        RAISE NOTICE '  1. Verify application is working (conversation listing should still work)';
        RAISE NOTICE '  2. Monitor performance (queries will be slower but functional)';
        RAISE NOTICE '  3. Consider re-applying migration if performance is critical';
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
-- Expected performance changes after rollback:
--
-- BEFORE ROLLBACK (with denormalized counts):
--   ListConversations: ~50ms
--   Uses: Simple column read (message_count)
--
-- AFTER ROLLBACK (with COUNT aggregation):
--   ListConversations: ~300-600ms
--   Uses: COUNT(*) with JOIN + GROUP BY
--
-- SQL Query Performance Comparison:
--
-- WITH denormalization (FAST):
-- SELECT id, title, message_count FROM conversations WHERE user_id = $1;
--
-- WITHOUT denormalization (SLOW):
-- SELECT c.id, c.title, COUNT(m.id) as message_count
-- FROM conversations c
-- LEFT JOIN messages m ON m.conversation_id = c.id
-- WHERE c.user_id = $1
-- GROUP BY c.id;
--
-- =============================================================================

-- =============================================================================
-- TROUBLESHOOTING
-- =============================================================================
--
-- If rollback fails:
--
-- 1. "cannot drop column message_count because other objects depend on it"
--    → Manually identify dependent objects:
--      SELECT * FROM pg_depend WHERE refobjid =
--        (SELECT oid FROM pg_class WHERE relname = 'conversations');
--
-- 2. "index idx_conversations_message_count does not exist"
--    → Safe to ignore, index already removed
--
-- 3. Rollback hangs on DROP INDEX CONCURRENTLY
--    → Check for long-running queries: SELECT * FROM pg_stat_activity;
--    → Wait for completion or cancel blocking queries
--
-- 4. Need to restore message_count data
--    → Re-run migration 080 to restore column and triggers
--    → Data will be backfilled from messages table
--
-- =============================================================================
-- ROLLBACK COMPLETION CHECKLIST
-- =============================================================================
--
-- After running this script:
--
-- [ ] Verify rollback completed successfully (check output above)
-- [ ] Test conversation listing in application
-- [ ] Monitor query performance (expect slower queries)
-- [ ] Check error logs for any issues
-- [ ] Update team that migration 080 has been rolled back
-- [ ] Document reason for rollback in runbook
--
-- =============================================================================
