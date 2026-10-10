# Supabase Migrations

This directory contains database migrations for the BusinessOS backend, managed using Supabase CLI.

**Migration Count:** 80 migrations
**Latest Migration:** 080_denormalize_message_counts.sql

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Migration Structure](#migration-structure)
3. [Creating Migrations](#creating-migrations)
4. [Running Migrations](#running-migrations)
5. [Rollback Procedures](#rollback-procedures)
6. [Performance Migrations](#performance-migrations)
7. [Testing Migrations](#testing-migrations)
8. [Best Practices](#best-practices)

---

## Quick Start

```bash
# Initialize Supabase (if not already done)
supabase init

# Start local Supabase instance
supabase start

# Apply migrations
supabase db push

# Reset database (WARNING: destroys data)
supabase db reset
```

---

## Migration Structure

```
supabase/
├── config.toml                          # Supabase configuration
├── migrations/                          # Migration files
│   ├── 001_auth_schema.sql              # Better Auth schema
│   ├── 002_subtasks.sql                 # Subtasks feature
│   ├── ...
│   ├── 079_performance_indexes.sql      # Performance optimization (30 indexes)
│   ├── 080_denormalize_message_counts.sql  # Denormalized counts
│   ├── rollback_079_performance_indexes.sql  # Rollback for 079
│   ├── rollback_080_denormalize_message_counts.sql  # Rollback for 080
│   └── ROLLBACK_RUNBOOK.md              # Rollback procedures
├── .gitignore                           # Ignore local dev files
└── README.md                            # This file
```

### Migration Naming Convention

```
{number}_{description}.sql

Examples:
- 001_auth_schema.sql
- 037_embedding_dimensions_768.sql
- 079_performance_indexes.sql

Rollback files:
- rollback_{number}_{description}.sql
```

---

## Creating Migrations

### Option 1: Auto-Generate from Database Changes

```bash
# Make changes in Supabase Dashboard SQL Editor or local instance
# Then generate migration file from diff
supabase db diff --schema public --file {migration_name}
```

### Option 2: Create Manually

```bash
# Create new migration file
supabase migration new {description}

# Edit the generated file
# Add your SQL statements
```

### Option 3: Copy from Internal Migrations

```bash
# If you have migrations in internal/database/migrations/
cp internal/database/migrations/XXX_name.sql supabase/migrations/
```

---

## Running Migrations

### Local Development

```bash
# Start local Supabase
supabase start

# Apply pending migrations
supabase db push

# Or reset and reapply all
supabase db reset
```

### Remote (Production/Staging)

```bash
# Link to remote project
supabase link --project-ref {project-ref}

# Push migrations to remote
supabase db push --linked

# Or use direct database connection
psql -h db.your-project.supabase.co -U postgres -d postgres -f migrations/080_example.sql
```

### CI/CD (GitHub Actions)

Migrations are automatically applied in GitHub Actions workflow:

```yaml
# .github/workflows/backend-tests.yml
- name: Apply Supabase Migrations
  run: |
    supabase start
    # Migrations applied automatically on start
```

---

## Rollback Procedures

### 🚨 Important: Always Rollback in Reverse Order

**Forward Order:**
```
079_performance_indexes.sql → 080_denormalize_message_counts.sql
```

**Rollback Order (REVERSE):**
```
rollback_080_denormalize_message_counts.sql → rollback_079_performance_indexes.sql
```

### Quick Rollback Commands

```bash
# Rollback Migration 080 only
supabase db execute --file migrations/rollback_080_denormalize_message_counts.sql

# Rollback Migration 079 only
supabase db execute --file migrations/rollback_079_performance_indexes.sql

# Rollback BOTH (correct order)
supabase db execute --file migrations/rollback_080_denormalize_message_counts.sql
supabase db execute --file migrations/rollback_079_performance_indexes.sql
```

### Full Rollback Documentation

**See:** `migrations/ROLLBACK_RUNBOOK.md` for complete rollback procedures, troubleshooting, and team training materials.

---

## Performance Migrations

### Migration 079: Performance Indexes

**File:** `079_performance_indexes.sql`
**Created:** 2026-01-18
**Impact:** 70-90% query performance improvement

**What it adds:**
- 30 composite indexes for common query patterns
- 2 GIN indexes for full-text search (pg_trgm)
- 2 monitoring views (v_index_usage_stats, v_slow_queries)

**Query Improvements:**
- Artifact queries: 250-400ms → <50ms
- Task queries: 180-350ms → <40ms
- Conversation queries: 300-600ms → <50ms
- Search queries: 1-3s → <100ms

**Rollback:** `rollback_079_performance_indexes.sql`

### Migration 080: Denormalized Message Counts

**File:** `080_denormalize_message_counts.sql`
**Created:** 2026-01-18
**Impact:** 90% reduction in conversation listing query time

**What it adds:**
- `message_count` column on `conversations` table
- 2 trigger functions (increment/decrement)
- 2 triggers (on INSERT/DELETE messages)
- 1 composite index on message_count
- Automatic backfill of existing counts

**Query Improvements:**
- ListConversations: 300-600ms → <50ms (eliminates COUNT aggregation)

**Rollback:** `rollback_080_denormalize_message_counts.sql`

---

## Testing Migrations

### Test in Local Supabase

```bash
# 1. Start fresh local instance
supabase db reset

# 2. Verify migrations applied
supabase db diff

# 3. Run application tests
go test ./...

# 4. Check query performance
psql -h localhost -p 54322 -U postgres -d postgres
```

### Query Performance Testing

```sql
-- Test conversation listing (should use indexes)
EXPLAIN ANALYZE
SELECT * FROM conversations
WHERE user_id = 'test-user-id'
ORDER BY updated_at DESC
LIMIT 20;

-- Expected after migration 079:
-- Index Scan using idx_conversations_user_updated
-- Execution time: <50ms

-- Test message count (should use denormalized column)
EXPLAIN ANALYZE
SELECT id, title, message_count
FROM conversations
WHERE user_id = 'test-user-id';

-- Expected after migration 080:
-- No JOIN, no COUNT(*), just column read
-- Execution time: <30ms
```

### Verify Index Usage

```sql
-- Check if indexes are being used
SELECT * FROM v_index_usage_stats
ORDER BY index_scans DESC;

-- Check for slow queries
SELECT * FROM v_slow_queries;

-- Check index sizes
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

---

## Best Practices

### Migration Guidelines

1. **Sequential Numbering**
   - Always use next available number
   - Zero-pad to 3 digits (001, 002, ..., 100)
   - Never reuse numbers

2. **Descriptive Names**
   ```
   ✅ GOOD: 079_performance_indexes.sql
   ❌ BAD:  079_stuff.sql
   ```

3. **Idempotent Migrations**
   - Use `IF NOT EXISTS` for CREATE statements
   - Use `IF EXISTS` for DROP statements
   - Migrations should be safe to run multiple times

4. **CONCURRENTLY for Production**
   ```sql
   -- ✅ Good (no table locks)
   CREATE INDEX CONCURRENTLY idx_name ON table(column);

   -- ❌ Bad (locks table)
   CREATE INDEX idx_name ON table(column);
   ```

5. **Always Include Rollback**
   - Every migration should have a rollback script
   - Test rollback before production deployment
   - Document rollback in ROLLBACK_RUNBOOK.md

6. **Performance Considerations**
   - Test on production-size datasets
   - Monitor index creation time
   - Check for blocking operations
   - Run ANALYZE after schema changes

7. **Documentation**
   - Add comments explaining purpose
   - Document expected performance impact
   - Include verification queries
   - Reference related features/tickets

### Example Migration Template

```sql
-- Migration: XXX_feature_name.sql
-- Description: What this migration does
-- Date: YYYY-MM-DD
-- Purpose: Why we need this change
--
-- Performance Impact:
-- - Expected query improvements
-- - Index sizes
-- - Estimated migration time
--
-- Prerequisites:
-- - Any required extensions
-- - Minimum PostgreSQL version
-- - Dependent migrations

-- =============================================================================
-- MAIN CHANGES
-- =============================================================================

-- Your SQL here
CREATE TABLE IF NOT EXISTS new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_new_table_created
ON new_table(created_at DESC);

-- =============================================================================
-- VERIFICATION
-- =============================================================================

-- Queries to verify migration succeeded
-- SELECT COUNT(*) FROM new_table;

-- =============================================================================
-- ROLLBACK PLAN
-- =============================================================================

-- Document how to rollback this migration
-- DROP INDEX CONCURRENTLY IF EXISTS idx_new_table_created;
-- DROP TABLE IF EXISTS new_table;
```

---

## Troubleshooting

### Common Issues

**Issue:** Migration fails with "relation already exists"
```bash
# Solution: Check if migration was partially applied
psql -c "SELECT * FROM pg_tables WHERE tablename = 'your_table';"

# If exists, either:
# 1. Skip migration (if intentional)
# 2. Drop manually and re-run
# 3. Add IF NOT EXISTS to migration
```

**Issue:** Supabase CLI not found
```bash
# Install Supabase CLI
npm install -g supabase

# Or using Homebrew (macOS)
brew install supabase/tap/supabase
```

**Issue:** Local Supabase won't start
```bash
# Check Docker is running
docker ps

# Reset Supabase
supabase stop
supabase start

# Check logs
supabase status
```

**Issue:** Migration takes too long (>5 minutes)
```sql
-- Check for blocking queries
SELECT * FROM pg_stat_activity WHERE state = 'active';

-- Check index creation progress
SELECT * FROM pg_stat_progress_create_index;
```

---

## Migration History

### Recent Migrations

| Number | Name | Date | Description | Rollback Available |
|--------|------|------|-------------|-------------------|
| 080 | denormalize_message_counts | 2026-01-18 | Denormalized counts for performance | ✅ Yes |
| 079 | performance_indexes | 2026-01-18 | 30 composite indexes for queries | ✅ Yes |
| 046 | osa_app_metadata | 2026-01-16 | OSA application metadata | ❌ No |
| 037 | embedding_dimensions_768 | 2026-01-14 | Increased vector dimensions | ❌ No |

### Statistics

- **Total Migrations:** 80
- **With Rollback Scripts:** 2 (079, 080)
- **Schema Size:** ~14,000 lines SQL
- **Tables:** 70+ tables
- **Indexes:** 100+ indexes (30 from migration 079)
- **Extensions:** pg_trgm, pgvector, uuid-ossp

---

## Resources

### Documentation
- [Supabase Migrations Docs](https://supabase.com/docs/guides/cli/local-development#database-migrations)
- [PostgreSQL CREATE INDEX](https://www.postgresql.org/docs/current/sql-createindex.html)
- [Migration Best Practices](https://supabase.com/docs/guides/database/migrations)

### Tools
- [Supabase CLI](https://github.com/supabase/cli)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)
- [pgAdmin](https://www.pgadmin.org/)

### Internal Docs
- `ROLLBACK_RUNBOOK.md` - Complete rollback procedures
- `../docs/QUERY_OPTIMIZATION_REPORT.md` - Performance analysis
- `../SUPABASE_CI_CD_MIGRATION.md` - CI/CD setup

---

## Contact

**Questions?**
- Slack: #backend-dev, #database-ops
- Email: backend-team@businessos.com
- Documentation: `docs/` folder

**Emergencies:**
- Check PagerDuty for on-call engineer
- Escalate to Tech Lead if rollback needed

---

**Last Updated:** 2026-01-19
**Maintained By:** Backend Team
**Next Review:** 2026-02-19
