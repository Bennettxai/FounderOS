# Staging Migration Test Plan

## Overview

This document provides a step-by-step checklist for testing migrations 052-054 and 088-089 on the staging environment.

---

## Pre-Migration Checklist

- [ ] **Review Migration Files**
  - [ ] Read all SQL files (052, 053, 054, 088, 089)
  - [ ] Verify no syntax errors
  - [ ] Confirm DOWN migrations exist
  - [ ] Check for dependency conflicts

- [ ] **Local Testing Complete**
  - [ ] Ran `test_migrations.sh` successfully
  - [ ] Verified schema integrity locally
  - [ ] Tested CRUD operations
  - [ ] Tested rollback locally

- [ ] **Code Preparation**
  - [ ] sqlc code regenerated (`sqlc generate`)
  - [ ] Go application compiles (`go build ./...`)
  - [ ] No linter errors
  - [ ] Integration tests pass

- [ ] **Environment Setup**
  - [ ] Staging database URL confirmed
  - [ ] Database credentials secured
  - [ ] Backup storage accessible
  - [ ] Monitoring tools ready

- [ ] **Communication**
  - [ ] Team notified of staging deployment
  - [ ] Rollback owner assigned
  - [ ] Incident response plan reviewed

---

## Phase 1: Backup & Preparation

### Step 1.1: Create Database Backup

```bash
# Set staging database URL
export STAGING_DB_URL="postgres://user:pass@staging.db.host:6543/businessos_staging"

# Create backup with timestamp
BACKUP_FILE="backup_staging_$(date +%Y%m%d_%H%M%S).sql"
pg_dump "$STAGING_DB_URL" > "$BACKUP_FILE"

# Verify backup file
ls -lh "$BACKUP_FILE"
head -100 "$BACKUP_FILE"

# Compress and store
gzip "$BACKUP_FILE"
gsutil cp "${BACKUP_FILE}.gz" gs://businessos-backups/staging/
```

**Verification:**
- [ ] Backup file created successfully
- [ ] Backup file size is reasonable (> 1MB expected)
- [ ] Backup uploaded to cloud storage
- [ ] Backup can be downloaded from cloud storage

---

### Step 1.2: Verify Current Schema State

```bash
# Check existing tables
psql "$STAGING_DB_URL" -c "\dt" | tee current_tables.txt

# Check if migration tables already exist (should be 0)
psql "$STAGING_DB_URL" -c "
  SELECT COUNT(*) as existing_tables
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_name IN (
      'workspace_versions',
      'onboarding_sessions',
      'onboarding_email_metadata',
      'custom_modules',
      'custom_module_versions',
      'custom_module_installations',
      'app_templates',
      'user_generated_apps',
      'app_generation_queue'
    );
"
```

**Expected Result**: `existing_tables = 0` (these are new tables)

**Verification:**
- [ ] Current schema documented
- [ ] No conflicting tables exist
- [ ] Required dependency tables exist (workspaces, "user")

---

### Step 1.3: Check Database Connections

```bash
# Test connection
psql "$STAGING_DB_URL" -c "SELECT version();"

# Check active connections
psql "$STAGING_DB_URL" -c "
  SELECT count(*) as active_connections
  FROM pg_stat_activity
  WHERE datname = current_database();
"

# Check database size
psql "$STAGING_DB_URL" -c "
  SELECT pg_size_pretty(pg_database_size(current_database())) as db_size;
"
```

**Verification:**
- [ ] Connection successful
- [ ] Active connections < 50 (safe to proceed)
- [ ] Database size documented

---

## Phase 2: Migration Application

### Step 2.1: Apply Migration 052 (workspace_versions)

```bash
psql "$STAGING_DB_URL" -f internal/database/migrations/052_workspace_versions.sql
```

**Verification Queries:**

```sql
-- Check table created
SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'workspace_versions';
-- Expected: 1

-- Check columns
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'workspace_versions'
ORDER BY ordinal_position;
-- Expected: id, workspace_id, version_number, snapshot_data, snapshot_metadata, created_by, created_at

-- Check indexes
SELECT indexname FROM pg_indexes WHERE tablename = 'workspace_versions';
-- Expected: idx_workspace_versions_workspace, idx_workspace_versions_created_at

-- Test insert
INSERT INTO workspace_versions (workspace_id, version_number, snapshot_data, created_by)
VALUES (gen_random_uuid(), 'v1.0.0-test', '{"test": true}'::jsonb, 'staging_test')
RETURNING id;

-- Verify and clean up
SELECT * FROM workspace_versions WHERE created_by = 'staging_test';
DELETE FROM workspace_versions WHERE created_by = 'staging_test';
```

**Verification:**
- [ ] Table `workspace_versions` created
- [ ] All columns present with correct types
- [ ] Indexes created successfully
- [ ] Foreign key to `workspaces` works
- [ ] Test insert/delete successful

---

### Step 2.2: Apply Migration 053 (onboarding_email_metadata)

```bash
psql "$STAGING_DB_URL" -f internal/database/migrations/053_onboarding_email_metadata.sql
```

**Verification Queries:**

```sql
-- Check tables created
SELECT table_name FROM information_schema.tables
WHERE table_name IN ('onboarding_sessions', 'onboarding_conversation_history', 'onboarding_email_metadata')
ORDER BY table_name;
-- Expected: 3 rows

-- Check foreign keys
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_name IN ('onboarding_sessions', 'onboarding_conversation_history', 'onboarding_email_metadata');

-- Test insert (requires valid user_id)
INSERT INTO onboarding_sessions (user_id, status, current_step, extracted_data)
VALUES (
  (SELECT id FROM "user" LIMIT 1),
  'in_progress',
  'company_name',
  '{"company": "Test Corp"}'::jsonb
)
RETURNING id;

-- Clean up test data
DELETE FROM onboarding_sessions WHERE extracted_data @> '{"company": "Test Corp"}'::jsonb;
```

**Verification:**
- [ ] All 3 tables created
- [ ] Foreign keys correctly reference `"user"` and `workspaces`
- [ ] Indexes created on session_id, user_id
- [ ] Test insert successful
- [ ] CASCADE delete works

---

### Step 2.3: Apply Migration 054 (custom_modules)

```bash
psql "$STAGING_DB_URL" -f internal/database/migrations/054_custom_modules.sql
```

**Verification Queries:**

```sql
-- Check tables
SELECT table_name FROM information_schema.tables
WHERE table_name LIKE 'custom_module%'
ORDER BY table_name;
-- Expected: custom_modules, custom_module_versions, custom_module_installations, custom_module_shares

-- Check GIN indexes for arrays
SELECT indexname, indexdef FROM pg_indexes
WHERE tablename = 'custom_modules'
  AND indexdef LIKE '%GIN%';
-- Expected: idx_custom_modules_tags, idx_custom_modules_keywords, idx_custom_modules_manifest

-- Test insert
INSERT INTO custom_modules (created_by, workspace_id, name, slug, category, manifest, tags)
VALUES (
  gen_random_uuid(),
  gen_random_uuid(),
  'Test Module',
  'test-module-staging',
  'utility',
  '{"version": "1.0"}'::jsonb,
  ARRAY['test', 'staging']
)
RETURNING id;

-- Test GIN index search
SELECT * FROM custom_modules WHERE 'test' = ANY(tags);

-- Clean up
DELETE FROM custom_modules WHERE slug = 'test-module-staging';
```

**Verification:**
- [ ] All 4 tables created
- [ ] GIN indexes created for JSONB and arrays
- [ ] UNIQUE constraint on (workspace_id, slug) works
- [ ] Array column queries work
- [ ] Test module insert/delete successful

---

### Step 2.4: Apply Migration 089 (app_generation_system)

**Note**: Must be applied BEFORE 088 due to data dependency.

```bash
psql "$STAGING_DB_URL" -f internal/database/migrations/089_app_generation_system.sql
```

**Verification Queries:**

```sql
-- Check tables
SELECT table_name FROM information_schema.tables
WHERE table_name IN ('app_templates', 'user_generated_apps', 'app_generation_queue')
ORDER BY table_name;
-- Expected: 3 rows

-- Check GIN indexes for targeting arrays
SELECT indexname FROM pg_indexes
WHERE tablename = 'app_templates'
  AND indexdef LIKE '%GIN%';
-- Expected: idx_app_templates_business_types, idx_app_templates_challenges, idx_app_templates_team_sizes

-- Verify constraints
SELECT conname, contype FROM pg_constraint
WHERE conrelid = 'app_generation_queue'::regclass;
-- Expected: CHECK constraint on status, CHECK constraint on priority

-- Test insert
INSERT INTO app_templates (template_name, category, display_name, scaffold_type)
VALUES ('test_template_staging', 'operations', 'Test Template', 'svelte')
RETURNING id;

-- Clean up
DELETE FROM app_templates WHERE template_name = 'test_template_staging';
```

**Verification:**
- [ ] All 3 tables created
- [ ] GIN indexes for array columns work
- [ ] CHECK constraints on status and priority work
- [ ] Foreign keys to workspaces work
- [ ] Test template insert/delete successful

---

### Step 2.5: Apply Migration 088 (seed_builtin_templates)

**Note**: This is a DATA migration (seeding). Applied after 089.

```bash
psql "$STAGING_DB_URL" -f internal/database/migrations/088_seed_builtin_templates.sql
```

**Verification Queries:**

```sql
-- Check seeded templates
SELECT template_name, display_name, category, priority_score
FROM app_templates
ORDER BY priority_score DESC;
-- Expected: 5 rows (saas_dashboard, api_backend, landing_page, crm_module, task_manager)

-- Verify template details
SELECT
    template_name,
    array_length(target_business_types, 1) as business_types_count,
    array_length(required_modules, 1) as required_modules_count,
    array_length(optional_features, 1) as optional_features_count
FROM app_templates;

-- Test upsert (should update, not error)
INSERT INTO app_templates (template_name, category, display_name, scaffold_type, priority_score)
VALUES ('saas_dashboard', 'operations', 'Updated Dashboard', 'svelte', 95)
ON CONFLICT (template_name) DO UPDATE
SET priority_score = EXCLUDED.priority_score
RETURNING template_name, priority_score;
-- Expected: saas_dashboard with priority_score = 95

-- Restore original priority
UPDATE app_templates SET priority_score = 90 WHERE template_name = 'saas_dashboard';
```

**Verification:**
- [ ] All 5 templates seeded
- [ ] Priority scores correct (saas=90, api=85, landing=80, crm=85, task=80)
- [ ] Target arrays populated
- [ ] ON CONFLICT upsert works
- [ ] Generation prompts present

---

## Phase 3: Integration Testing

### Step 3.1: Schema Integrity Check

```bash
# Run comprehensive integrity check
psql "$STAGING_DB_URL" <<'EOF'
-- Check for orphaned foreign keys
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_name IN (
    'workspace_versions',
    'onboarding_sessions',
    'custom_modules',
    'app_templates',
    'user_generated_apps',
    'app_generation_queue'
  )
ORDER BY tc.table_name;

-- Check for missing indexes (tables without any indexes)
SELECT tablename
FROM pg_tables
WHERE schemaname = 'public'
  AND tablename IN (
    'workspace_versions',
    'onboarding_sessions',
    'custom_modules',
    'app_templates',
    'user_generated_apps'
  )
  AND tablename NOT IN (SELECT DISTINCT tablename FROM pg_indexes WHERE schemaname = 'public');
-- Expected: 0 rows (all tables should have indexes)

-- Check table comments
SELECT
    c.relname AS table_name,
    pg_catalog.obj_description(c.oid, 'pg_class') AS table_comment
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname = 'public'
  AND c.relname IN (
    'workspace_versions',
    'onboarding_sessions',
    'custom_modules',
    'app_templates',
    'app_generation_queue'
  )
ORDER BY c.relname;
EOF
```

**Verification:**
- [ ] All foreign keys valid
- [ ] All tables have at least one index
- [ ] Table comments present
- [ ] No orphaned constraints

---

### Step 3.2: Application Integration Test

**Deploy updated Go application to staging:**

```bash
cd desktop/backend-go

# Build application
go build -o bin/server-staging ./cmd/server

# Run locally against staging DB
export DATABASE_URL="$STAGING_DB_URL"
export SERVER_PORT=8002
./bin/server-staging &

# Store PID for cleanup
SERVER_PID=$!
sleep 5

# Test health endpoint
curl http://localhost:8002/health

# Test database connectivity
curl http://localhost:8002/api/status

# Stop server
kill $SERVER_PID
```

**Verification:**
- [ ] Application starts without errors
- [ ] No database connection errors in logs
- [ ] Health check passes
- [ ] API endpoints respond

---

### Step 3.3: End-to-End Feature Tests

**Test workspace versions:**

```bash
# Create workspace version via API (if endpoint exists)
# Or test directly via psql

psql "$STAGING_DB_URL" <<'EOF'
-- Test workspace snapshot creation
INSERT INTO workspace_versions (
  workspace_id,
  version_number,
  snapshot_data,
  snapshot_metadata,
  created_by
)
SELECT
  id,
  'v1.0.0',
  '{"apps": [], "members": [], "settings": {}}'::jsonb,
  '{"app_count": 0, "member_count": 1}'::jsonb,
  'staging_e2e_test'
FROM workspaces
LIMIT 1
RETURNING id, workspace_id, version_number;

-- Verify snapshot retrieval
SELECT
  version_number,
  snapshot_metadata->>'app_count' as app_count,
  created_at
FROM workspace_versions
WHERE created_by = 'staging_e2e_test';

-- Clean up
DELETE FROM workspace_versions WHERE created_by = 'staging_e2e_test';
EOF
```

**Test app generation queue:**

```bash
psql "$STAGING_DB_URL" <<'EOF'
-- Test queue entry creation
INSERT INTO app_generation_queue (
  workspace_id,
  template_id,
  status,
  priority,
  generation_context
)
SELECT
  w.id,
  t.id,
  'pending',
  8,
  '{"app_name": "Test App", "description": "E2E test"}'::jsonb
FROM workspaces w
CROSS JOIN app_templates t
WHERE t.template_name = 'saas_dashboard'
LIMIT 1
RETURNING id, status, priority;

-- Verify queue processing logic
UPDATE app_generation_queue
SET status = 'processing', started_at = NOW()
WHERE status = 'pending'
  AND generation_context @> '{"app_name": "Test App"}'::jsonb
RETURNING id, status;

-- Mark complete
UPDATE app_generation_queue
SET status = 'completed', completed_at = NOW()
WHERE generation_context @> '{"app_name": "Test App"}'::jsonb;

-- Verify final state
SELECT status, completed_at IS NOT NULL as has_completion_time
FROM app_generation_queue
WHERE generation_context @> '{"app_name": "Test App"}'::jsonb;

-- Clean up
DELETE FROM app_generation_queue WHERE generation_context @> '{"app_name": "Test App"}'::jsonb;
EOF
```

**Verification:**
- [ ] Workspace snapshots can be created
- [ ] App generation queue works (pending → processing → completed)
- [ ] Template recommendations query works
- [ ] No errors in application logs

---

## Phase 4: Performance & Monitoring

### Step 4.1: Query Performance Check

```bash
# Enable query timing
psql "$STAGING_DB_URL" -c "\timing on"

# Test common queries
psql "$STAGING_DB_URL" <<'EOF'
-- Test workspace versions query (with index)
EXPLAIN ANALYZE
SELECT * FROM workspace_versions
WHERE workspace_id = (SELECT id FROM workspaces LIMIT 1)
ORDER BY created_at DESC
LIMIT 10;

-- Test app template recommendation (GIN index)
EXPLAIN ANALYZE
SELECT * FROM app_templates
WHERE 'saas' = ANY(target_business_types)
ORDER BY priority_score DESC
LIMIT 5;

-- Test custom module search (GIN index)
EXPLAIN ANALYZE
SELECT * FROM custom_modules
WHERE 'productivity' = ANY(tags)
  AND is_public = TRUE;
EOF
```

**Expected Performance:**
- Query times < 100ms for indexed queries
- Index scans used (not sequential scans)
- No full table scans on large tables

**Verification:**
- [ ] All queries use indexes
- [ ] Query times acceptable (< 100ms)
- [ ] No sequential scans on large tables
- [ ] EXPLAIN plans show efficient execution

---

### Step 4.2: Monitor Database Metrics

```bash
# Check table sizes
psql "$STAGING_DB_URL" -c "
  SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
  FROM pg_tables
  WHERE schemaname = 'public'
    AND tablename IN (
      'workspace_versions',
      'onboarding_sessions',
      'custom_modules',
      'app_templates',
      'user_generated_apps',
      'app_generation_queue'
    )
  ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"

# Check index usage
psql "$STAGING_DB_URL" -c "
  SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
  FROM pg_stat_user_indexes
  WHERE schemaname = 'public'
    AND tablename IN (
      'workspace_versions',
      'onboarding_sessions',
      'custom_modules',
      'app_templates'
    )
  ORDER BY tablename, indexname;
"
```

**Verification:**
- [ ] Table sizes documented
- [ ] Index usage statistics recorded
- [ ] No bloat detected

---

## Phase 5: Rollback Testing

### Step 5.1: Test Rollback Capability

**⚠️ CAUTION**: This will delete tables and data. Only proceed if staging is safe to reset.

```bash
# Apply DOWN migrations in reverse order
psql "$STAGING_DB_URL" <<'EOF'
BEGIN;

-- Rollback 088 (data deletion)
DELETE FROM app_templates
WHERE template_name IN ('saas_dashboard', 'api_backend', 'landing_page', 'crm_module', 'task_manager');

-- Rollback 089
DROP TABLE IF EXISTS app_generation_queue CASCADE;
DROP TABLE IF EXISTS user_generated_apps CASCADE;
DROP TABLE IF EXISTS app_templates CASCADE;

-- Rollback 054
DROP TABLE IF EXISTS custom_module_shares CASCADE;
DROP TABLE IF EXISTS custom_module_installations CASCADE;
DROP TABLE IF EXISTS custom_module_versions CASCADE;
DROP TABLE IF EXISTS custom_modules CASCADE;

-- Rollback 053
DROP TABLE IF EXISTS onboarding_email_metadata CASCADE;
DROP TABLE IF EXISTS onboarding_conversation_history CASCADE;
DROP TABLE IF EXISTS onboarding_sessions CASCADE;

-- Rollback 052
DROP TABLE IF EXISTS workspace_versions CASCADE;

-- Verify all tables removed
SELECT COUNT(*) as remaining_tables
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
    'workspace_versions',
    'onboarding_sessions',
    'custom_modules',
    'app_templates',
    'user_generated_apps',
    'app_generation_queue'
  );
-- Expected: 0

COMMIT;
EOF
```

**Verification:**
- [ ] All tables successfully dropped
- [ ] No orphaned foreign key constraints
- [ ] No errors during rollback
- [ ] Remaining_tables = 0

---

### Step 5.2: Re-apply Migrations

After successful rollback test, re-apply migrations:

```bash
# Apply in correct order
psql "$STAGING_DB_URL" -f internal/database/migrations/052_workspace_versions.sql
psql "$STAGING_DB_URL" -f internal/database/migrations/053_onboarding_email_metadata.sql
psql "$STAGING_DB_URL" -f internal/database/migrations/054_custom_modules.sql
psql "$STAGING_DB_URL" -f internal/database/migrations/089_app_generation_system.sql
psql "$STAGING_DB_URL" -f internal/database/migrations/088_seed_builtin_templates.sql

# Verify all tables restored
psql "$STAGING_DB_URL" -c "
  SELECT COUNT(*) as restored_tables
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_name IN (
      'workspace_versions',
      'onboarding_sessions',
      'onboarding_email_metadata',
      'custom_modules',
      'app_templates',
      'user_generated_apps',
      'app_generation_queue'
    );
"
-- Expected: 9 tables
```

**Verification:**
- [ ] All migrations re-applied successfully
- [ ] All tables restored
- [ ] Seeded data present (5 templates)
- [ ] Application still works

---

## Phase 6: Documentation & Sign-off

### Step 6.1: Document Results

Create migration report:

```markdown
# Staging Migration Report
**Date**: YYYY-MM-DD
**Migrations**: 052-054, 088-089
**Database**: Staging (businessos_staging)
**Applied By**: [Your Name]

## Results

### Tables Created
- workspace_versions ✅
- onboarding_sessions ✅
- onboarding_email_metadata ✅
- onboarding_conversation_history ✅
- custom_modules ✅
- custom_module_versions ✅
- custom_module_installations ✅
- custom_module_shares ✅
- app_templates ✅
- user_generated_apps ✅
- app_generation_queue ✅

### Data Seeded
- 5 app templates (saas_dashboard, api_backend, landing_page, crm_module, task_manager) ✅

### Performance
- Query times: < 50ms (avg)
- Index usage: 100% (all queries using indexes)
- Table sizes: < 1MB each (new tables)

### Issues Encountered
- None

### Rollback Test
- Rollback executed: ✅
- Re-apply executed: ✅
- Data integrity verified: ✅

## Recommendation
✅ **Approve for production deployment**
```

**Verification:**
- [ ] Report created and saved
- [ ] All metrics documented
- [ ] Screenshots/logs attached (if needed)

---

### Step 6.2: Staging Sign-off Checklist

- [ ] **All migrations applied successfully**
- [ ] **Schema integrity verified**
- [ ] **Application integration tested**
- [ ] **Performance metrics acceptable**
- [ ] **Rollback capability confirmed**
- [ ] **No errors in logs for 24 hours**
- [ ] **Team notified of successful staging deployment**
- [ ] **Documentation updated**
- [ ] **Production deployment plan reviewed**
- [ ] **Approved by**: ___________________

---

## Rollback Plan (Emergency)

If critical issues arise during staging testing:

### Immediate Rollback Steps

```bash
# 1. Restore from backup
gunzip -c backup_staging_*.sql.gz | psql "$STAGING_DB_URL"

# 2. Verify restoration
psql "$STAGING_DB_URL" -c "SELECT COUNT(*) FROM workspaces;"

# 3. Restart application
# (Depends on your deployment method)

# 4. Verify health
curl https://staging.businessos.com/api/health
```

### Rollback Decision Matrix

| Issue | Severity | Action |
|-------|----------|--------|
| Table creation fails | High | Stop immediately, investigate, restore backup |
| Foreign key violation | Medium | Investigate dependencies, may proceed with caution |
| Index creation slow | Low | Proceed, optimize later |
| Application won't start | High | Restore backup, investigate locally |
| Query performance degraded | Medium | Proceed, add indexes, monitor |
| Data corruption detected | Critical | **IMMEDIATE ROLLBACK**, restore backup |

---

## Contact Information

**Migration Lead**: [Name]
**Database Admin**: [Name]
**On-call Engineer**: [Name]
**Incident Channel**: #incidents-staging

---

**Last Updated**: 2026-01-26
**Next Review**: Before production deployment
