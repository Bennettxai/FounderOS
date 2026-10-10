-- ============================================================================
-- Session Token Rotation Script
-- ============================================================================
-- Purpose: Forces rotation of all session tokens by deleting active sessions
-- Impact: All users will be logged out and must re-authenticate
-- Usage: Run this script immediately after deploying token logging fix
-- Schedule: Run every 60 days as preventive measure (see cron job below)
-- ============================================================================

-- 1. Count active sessions before rotation
SELECT
    COUNT(*) as active_sessions,
    COUNT(DISTINCT "userId") as unique_users,
    MIN("expiresAt") as earliest_expiry,
    MAX("expiresAt") as latest_expiry
FROM session
WHERE "expiresAt" > NOW();

-- 2. Archive session metadata for audit trail (optional)
-- Uncomment if you want to keep audit records
-- CREATE TABLE IF NOT EXISTS session_rotation_audit (
--     rotation_id SERIAL PRIMARY KEY,
--     rotated_at TIMESTAMP DEFAULT NOW(),
--     sessions_deleted INTEGER,
--     users_affected INTEGER,
--     reason TEXT
-- );

-- 3. Delete all active sessions (forces re-login)
DELETE FROM session
WHERE "expiresAt" > NOW();

-- 4. Verify all sessions are cleared
SELECT
    COUNT(*) as remaining_active_sessions
FROM session
WHERE "expiresAt" > NOW();

-- 5. Log rotation event (optional audit record)
-- INSERT INTO session_rotation_audit (sessions_deleted, users_affected, reason)
-- SELECT
--     (SELECT COUNT(*) FROM session WHERE "expiresAt" <= NOW()),
--     0,
--     'Emergency rotation due to token logging vulnerability fix';

-- ============================================================================
-- CRON JOB SETUP (Optional - for automated 60-day rotation)
-- ============================================================================
-- To enable automated rotation every 60 days, run this in your PostgreSQL:
--
-- SELECT cron.schedule(
--     'session-token-rotation',
--     '0 2 1 */2 *',  -- At 2 AM on day 1 of every 2nd month
--     $$DELETE FROM session WHERE "expiresAt" > NOW()$$
-- );
--
-- Requires pg_cron extension. To install:
-- CREATE EXTENSION IF NOT EXISTS pg_cron;
-- ============================================================================

-- Expected Output:
-- - Row 1: Session counts BEFORE deletion
-- - Row 2: Number of deleted sessions
-- - Row 3: Remaining active sessions (should be 0)
