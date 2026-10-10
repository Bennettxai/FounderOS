# Database Migration Rollback Runbook

**Version:** 1.0.0
**Last Updated:** 2026-01-19
**Status:** ACTIVE

---

## Executive Summary

This runbook provides step-by-step instructions for rolling back database migrations 079 and 080. These migrations added performance optimizations (indexes and denormalized columns) that can be safely removed if issues arise.

**Risk Level:** LOW - Rollback removes optimizations but does not delete data.

**Performance Impact:** Query times will degrade by 70-90% after rollback, but all functionality remains intact.

---

## Table of Contents

1. [Pre-Rollback Checklist](#pre-rollback-checklist)
2. [When to Rollback](#when-to-rollback)
3. [Rollback Order](#rollback-order)
4. [Step-by-Step Procedures](#step-by-step-procedures)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)
7. [Post-Rollback Actions](#post-rollback-actions)
8. [Re-Application Guide](#re-application-guide)

---

## Pre-Rollback Checklist

### 🚨 Before You Begin

- [ ] **Identify the Problem:** Document why rollback is needed
- [ ] **Approval Obtained:** Get sign-off from tech lead/manager
- [ ] **Backup Created:** Full database backup completed
- [ ] **Backup Verified:** Test backup restoration process
- [ ] **Team Notified:** Inform team of rollback plan
- [ ] **Maintenance Window:** Schedule during low-traffic period
- [ ] **Rollback Scripts Ready:** Verify rollback scripts exist and are accessible
- [ ] **Database Access:** Confirm you have superuser/owner privileges
- [ ] **Monitoring Ready:** Ensure monitoring tools are active

### 📊 Required Information

| Item | Value | How to Check |
|------|-------|--------------|
| Database Version | PostgreSQL 14+ | `SELECT version();` |
| Current Migration | 079, 080 | Check `supabase_migrations` table |
| Table Sizes | artifacts, tasks, conversations | `SELECT pg_size_pretty(pg_total_relation_size('table_name'));` |
| Active Connections | Count | `SELECT count(*) FROM pg_stat_activity;` |
| Database Load | CPU, Memory | Check monitoring dashboard |

---

## When to Rollback

### Valid Reasons for Rollback

1. **Performance Issues**
   - Queries slower than expected
   - Index creation causing table locks
   - Database CPU/memory exhaustion

2. **Bugs Discovered**
   - Trigger errors affecting data integrity
   - Incorrect denormalized counts
   - Index corruption

3. **Application Errors**
   - Application crashes after migration
   - Timeouts on queries
   - OOM (Out of Memory) errors

4. **Business Requirements**
   - Need to revert for emergency hotfix
   - Rollback to match staging environment
   - Testing rollback procedures

### When NOT to Rollback

- ❌ Minor performance degradation (<20%)
- ❌ Isolated errors unrelated to migrations
- ❌ Panic/fear without clear issue
- ❌ Before investigating the root cause

---

## Rollback Order

### ⚠️ CRITICAL: Always Rollback in Reverse Order

```
Forward Migration Order:
  079_performance_indexes.sql → 080_denormalize_message_counts.sql

Rollback Order (REVERSE):
  rollback_080_denormalize_message_counts.sql → rollback_079_performance_indexes.sql
```

**Why?**
- Migration 080 depends on indexes from Migration 079
- Rolling back 079 first could break 080's triggers
- Always rollback dependent migrations first

### Scenarios

| Scenario | Rollback Scripts to Run | Order |
|----------|-------------------------|-------|
| **Rollback 080 Only** | `rollback_080` | 1. Run rollback_080 |
| **Rollback 079 Only** | `rollback_079` | 1. Run rollback_079 |
| **Rollback BOTH** | `rollback_080`, `rollback_079` | 1. Run rollback_080<br>2. Run rollback_079 |

---

## Step-by-Step Procedures

### Option A: Using Supabase CLI (Recommended)

```bash
# 1. Navigate to backend directory
cd desktop/backend-go

# 2. Connect to Supabase (if using Supabase hosted)
supabase db reset --linked  # This will reset to last migration

# OR manually rollback specific migrations:

# 3a. Rollback Migration 080 only
supabase db execute --file supabase/migrations/rollback_080_denormalize_message_counts.sql

# 3b. Rollback Migration 079 only
supabase db execute --file supabase/migrations/rollback_079_performance_indexes.sql

# 3c. Rollback BOTH (in correct order)
supabase db execute --file supabase/migrations/rollback_080_denormalize_message_counts.sql
supabase db execute --file supabase/migrations/rollback_079_performance_indexes.sql
```

### Option B: Using psql (Direct Database Connection)

```bash
# 1. Set environment variables
export PGHOST=db.your-project.supabase.co
export PGPORT=5432
export PGDATABASE=postgres
export PGUSER=postgres
export PGPASSWORD=your-password

# 2. Rollback Migration 080
psql -f desktop/backend-go/supabase/migrations/rollback_080_denormalize_message_counts.sql

# Expected output:
# NOTICE: Step 1/4: Dropping triggers...
# NOTICE:   ✓ Triggers dropped successfully
# ...
# NOTICE: ✅ ROLLBACK COMPLETE

# 3. Rollback Migration 079 (if needed)
psql -f desktop/backend-go/supabase/migrations/rollback_079_performance_indexes.sql

# Expected output:
# NOTICE: Step 1/8: Dropping monitoring views...
# NOTICE:   ✓ 2 views dropped successfully
# ...
# NOTICE: ✅ ROLLBACK COMPLETE
```

### Option C: Using Supabase Dashboard

1. Go to Supabase Dashboard → SQL Editor
2. Copy rollback script content
3. Paste into SQL Editor
4. Click "Run"
5. Review output for success/errors
6. Repeat for additional rollbacks if needed

---

## Verification

### Step 1: Check Rollback Output

**Success Indicators:**
```
✅ ROLLBACK COMPLETE: Migration XXX successfully rolled back
✓ All objects removed successfully
```

**Failure Indicators:**
```
⚠️ ROLLBACK INCOMPLETE: Some objects still exist
✗ FAILED: Index still exists
```

### Step 2: Verify Database Objects Removed

**For Migration 080:**
```sql
-- Check column doesn't exist
SELECT column_name
FROM information_schema.columns
WHERE table_name = 'conversations'
AND column_name = 'message_count';
-- Expected: 0 rows

-- Check triggers don't exist
SELECT trigger_name
FROM information_schema.triggers
WHERE event_object_table = 'messages'
AND trigger_name LIKE '%message_count%';
-- Expected: 0 rows

-- Check functions don't exist
SELECT proname
FROM pg_proc
WHERE proname LIKE '%message_count%';
-- Expected: 0 rows
```

**For Migration 079:**
```sql
-- Check indexes don't exist
SELECT indexname
FROM pg_indexes
WHERE indexname LIKE 'idx_%'
AND indexname IN (
    'idx_artifacts_user_updated',
    'idx_tasks_user_status_priority',
    'idx_conversations_user_updated'
    -- ... check key indexes
);
-- Expected: 0 rows

-- Check views don't exist
SELECT table_name
FROM information_schema.views
WHERE table_name IN ('v_index_usage_stats', 'v_slow_queries');
-- Expected: 0 rows
```

### Step 3: Test Application Functionality

**Critical Paths to Test:**

1. **Conversation Listing**
   ```
   Action: Navigate to conversations page
   Expected: List loads (slower, but functional)
   Check: No errors in console/logs
   ```

2. **Task Filtering**
   ```
   Action: Filter tasks by status/priority
   Expected: Filter works (slower, but accurate)
   Check: All tasks displayed correctly
   ```

3. **Search**
   ```
   Action: Search conversations by keyword
   Expected: Results returned (slower, but accurate)
   Check: Full-text search still works
   ```

4. **Artifact Operations**
   ```
   Action: Create/list/update artifacts
   Expected: CRUD operations work
   Check: Data persists correctly
   ```

### Step 4: Monitor Performance

**Query Performance (expect degradation):**
```sql
-- Check query execution times
EXPLAIN ANALYZE SELECT * FROM conversations WHERE user_id = 'test-user-id' ORDER BY updated_at DESC LIMIT 20;

-- Before rollback: ~50ms, Index Scan
-- After rollback: ~300ms, Sequential Scan
```

**Expected Performance Table:**

| Query Type | Before Rollback | After Rollback | Degradation |
|------------|-----------------|----------------|-------------|
| List Conversations | 50ms | 300ms | 6x slower |
| Filter Tasks | 40ms | 180ms | 4.5x slower |
| Search Messages | 100ms | 1-3s | 10-30x slower |
| List Artifacts | 50ms | 250ms | 5x slower |

---

## Troubleshooting

### Issue 1: Rollback Script Hangs

**Symptoms:**
- Script runs for >10 minutes with no output
- Database appears locked
- Other queries timing out

**Diagnosis:**
```sql
-- Check for blocking queries
SELECT pid, usename, state, query, wait_event_type
FROM pg_stat_activity
WHERE state = 'active'
AND query NOT LIKE '%pg_stat_activity%';

-- Check for locks
SELECT locktype, relation::regclass, mode, granted
FROM pg_locks
WHERE NOT granted;
```

**Solution:**
```sql
-- Option 1: Wait for completion (recommended)
-- DROP INDEX CONCURRENTLY can take 5-15 minutes on large tables

-- Option 2: Cancel blocking queries (if truly stuck)
SELECT pg_cancel_backend(pid)
FROM pg_stat_activity
WHERE state = 'active'
AND query LIKE '%your_table%';

-- Option 3: Terminate connections (last resort)
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'your_database'
AND pid != pg_backend_pid();
```

### Issue 2: "Cannot Drop Column" Error

**Error Message:**
```
ERROR: cannot drop column message_count because other objects depend on it
```

**Solution:**
```sql
-- 1. Identify dependent objects
SELECT * FROM pg_depend
WHERE refobjid = (
    SELECT oid FROM pg_attribute
    WHERE attrelid = 'conversations'::regclass
    AND attname = 'message_count'
);

-- 2. Drop dependent objects first
-- (manually drop each dependent object)

-- 3. Re-run rollback script
```

### Issue 3: Partial Rollback Completed

**Symptoms:**
- Some indexes dropped, others still exist
- Verification shows failures

**Solution:**
```bash
# 1. Check which objects still exist
psql -c "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_%';"

# 2. Re-run rollback script (safe to run multiple times)
psql -f rollback_079_performance_indexes.sql

# 3. Manually drop remaining objects if script fails again
psql -c "DROP INDEX CONCURRENTLY idx_remaining_index;"
```

### Issue 4: Application Errors After Rollback

**Symptoms:**
- Query timeouts
- Slow page loads
- "Out of memory" errors

**Solution:**
```sql
-- 1. Increase statement timeout temporarily
SET statement_timeout = '60s';

-- 2. Run VACUUM ANALYZE to update statistics
VACUUM ANALYZE conversations;
VACUUM ANALYZE messages;
VACUUM ANALYZE artifacts;
VACUUM ANALYZE tasks;

-- 3. Check for bloated tables
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 4. Consider re-applying migrations if performance is critical
```

### Issue 5: Trigger Errors After Rollback

**Error Message:**
```
ERROR: function increment_conversation_message_count() does not exist
```

**Cause:** Triggers still exist but functions were dropped

**Solution:**
```sql
-- Manually drop remaining triggers
DROP TRIGGER IF EXISTS trigger_increment_message_count ON messages;
DROP TRIGGER IF EXISTS trigger_decrement_message_count ON messages;

-- Verify no triggers remain
SELECT trigger_name, event_object_table
FROM information_schema.triggers
WHERE trigger_name LIKE '%message_count%';
```

---

## Post-Rollback Actions

### Immediate Actions (Within 1 Hour)

- [ ] **Verify Application Working:** Test all critical features
- [ ] **Monitor Error Logs:** Check for new errors in production
- [ ] **Update Team:** Notify team that rollback is complete
- [ ] **Document Incident:** Record reason for rollback
- [ ] **Monitor Performance:** Watch query performance metrics

### Short-Term Actions (Within 24 Hours)

- [ ] **Analyze Root Cause:** Investigate why rollback was needed
- [ ] **Update Monitoring:** Add alerts for similar issues
- [ ] **Performance Review:** Assess impact of degraded performance
- [ ] **User Communication:** Notify users if performance is noticeably worse
- [ ] **Plan Re-Application:** Decide if/when to re-apply migrations

### Long-Term Actions (Within 1 Week)

- [ ] **Fix Root Issue:** Address the problem that caused rollback
- [ ] **Test Fix:** Verify fix works in staging environment
- [ ] **Plan Migration:** Schedule re-application of migrations
- [ ] **Update Runbook:** Document lessons learned
- [ ] **Team Training:** Review rollback procedures with team

---

## Re-Application Guide

### When to Re-Apply Migrations

**Re-apply migrations when:**
- ✅ Root cause of rollback has been fixed
- ✅ Performance degradation is impacting users
- ✅ Database load is too high without indexes
- ✅ Search queries timing out (Migration 079)
- ✅ Conversation listing too slow (Migration 080)

### Re-Application Procedure

```bash
# 1. Ensure database is in good state
psql -c "VACUUM ANALYZE;"

# 2. Re-apply Migration 079 (indexes)
supabase db execute --file supabase/migrations/079_performance_indexes.sql

# Expected time: 10-20 minutes
# Monitor: Watch for index creation progress

# 3. Verify indexes created
psql -c "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_%';"

# 4. Re-apply Migration 080 (denormalization)
supabase db execute --file supabase/migrations/080_denormalize_message_counts.sql

# Expected time: 2-5 minutes
# Monitor: Watch for trigger creation and backfill

# 5. Verify performance improved
psql -c "EXPLAIN ANALYZE SELECT * FROM conversations WHERE user_id = 'test' ORDER BY updated_at DESC LIMIT 20;"

# Expected: <50ms execution time, Index Scan visible
```

### Monitoring After Re-Application

**Key Metrics to Watch:**

1. **Query Performance**
   - Conversation listing: Should be <50ms
   - Task filtering: Should be <40ms
   - Search queries: Should be <100ms

2. **Database Load**
   - CPU usage: Should decrease 30-40%
   - Memory usage: Should decrease 20-30%
   - Connection pool: Should have more available connections

3. **Application Metrics**
   - Page load times: Should improve 50-70%
   - API response times: Should improve 70-90%
   - Error rates: Should remain low

---

## Contact Information

### Escalation Path

| Level | Contact | When to Escalate |
|-------|---------|------------------|
| **L1** | Backend Team | Rollback questions, script errors |
| **L2** | Tech Lead | Rollback failures, data issues |
| **L3** | Database Admin | Database corruption, critical failures |
| **L4** | CTO | Production outage, data loss risk |

### Support Channels

- **Slack:** #backend-dev, #database-ops
- **Email:** backend-team@businessos.com
- **On-Call:** Check PagerDuty for current on-call engineer

---

## Appendix

### A. Quick Reference Commands

```bash
# Check current migration version
psql -c "SELECT * FROM supabase_migrations ORDER BY version DESC LIMIT 5;"

# List all performance indexes
psql -c "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_%' ORDER BY indexname;"

# Check table sizes
psql -c "SELECT tablename, pg_size_pretty(pg_total_relation_size('public.'||tablename)) AS size FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size('public.'||tablename) DESC;"

# Monitor active queries
psql -c "SELECT pid, usename, application_name, state, query_start, query FROM pg_stat_activity WHERE state = 'active';"

# Check database connections
psql -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"
```

### B. Rollback Script Locations

```
desktop/backend-go/supabase/migrations/
├── 079_performance_indexes.sql
├── 080_denormalize_message_counts.sql
├── rollback_079_performance_indexes.sql
├── rollback_080_denormalize_message_counts.sql
└── ROLLBACK_RUNBOOK.md (this file)
```

### C. Migration Timeline

| Date | Migration | Description | Rollback Script |
|------|-----------|-------------|-----------------|
| 2026-01-18 | 079 | Performance indexes (30 indexes) | rollback_079_performance_indexes.sql |
| 2026-01-18 | 080 | Denormalized message counts | rollback_080_denormalize_message_counts.sql |

---

**Document Version:** 1.0.0
**Last Reviewed:** 2026-01-19
**Next Review:** 2026-02-19
**Maintained By:** Backend Team
