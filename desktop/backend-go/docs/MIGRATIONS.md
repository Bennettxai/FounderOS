# Database Migrations Guide

## Overview

BusinessOS uses PostgreSQL with a custom migration system. This document covers migration creation, testing, and deployment strategies.

---

## Table of Contents

1. [Migration System](#migration-system)
2. [Numbering Scheme](#numbering-scheme)
3. [Creating Migrations](#creating-migrations)
4. [Testing Migrations](#testing-migrations)
5. [Rollback Strategy](#rollback-strategy)
6. [Staging Deployment](#staging-deployment)
7. [Production Deployment](#production-deployment)
8. [Current Migrations (052-054, 088-089)](#current-migrations)
9. [Troubleshooting](#troubleshooting)

---

## Migration System

### Directory Structure

```
desktop/backend-go/
├── internal/database/
│   ├── migrations/           # Migration files (numbered)
│   │   ├── 052_workspace_versions.sql
│   │   ├── 053_onboarding_email_metadata.sql
│   │   ├── 054_custom_modules.sql
│   │   ├── 088_seed_builtin_templates.sql
│   │   └── 089_app_generation_system.sql
│   ├── schema.sql            # Consolidated schema (for sqlc)
│   └── queries/              # sqlc query files
├── scripts/migrations/       # Migration testing scripts
└── docs/                     # Documentation (this file)
```

### Migration File Format

All migration files follow the `migrate` format with UP and DOWN sections:

```sql
-- +migrate Up
-- Description of changes

CREATE TABLE IF NOT EXISTS example_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_example_name ON example_table(name);

COMMENT ON TABLE example_table IS 'Descriptive comment';

-- +migrate Down
DROP TABLE IF EXISTS example_table CASCADE;
```

#### Best Practices for Migration Files

- **Always include UP and DOWN sections**
- **Use `IF NOT EXISTS` for CREATE operations**
- **Use `IF EXISTS` for DROP operations**
- **Create indexes AFTER table creation**
- **Use CASCADE for DROP operations** (be careful!)
- **Add comments for complex tables**
- **Use TIMESTAMPTZ for timestamps** (not TIMESTAMP)
- **Use gen_random_uuid() for UUIDs** (not uuid_generate_v4())

---

## Numbering Scheme

### Convention

- **Format**: `NNN_descriptive_name.sql`
- **NNN**: 3-digit number (001-999)
- **descriptive_name**: snake_case description

### Current Number Ranges

| Range | Purpose | Example |
|-------|---------|---------|
| 001-050 | Core system tables | `001_auth_schema.sql` |
| 051-070 | Feature additions | `052_workspace_versions.sql` |
| 071-085 | Integration systems | `078_osa_app_metadata.sql` |
| 086-099 | App generation system | `089_app_generation_system.sql` |
| 100+ | Future expansions | - |

### Gaps in Numbering

There are intentional gaps (e.g., 055-087) to allow for backfilling migrations developed in parallel branches.

**Rule**: Always use the next available number. Do not reuse or skip numbers arbitrarily.

---

## Creating Migrations

### Step 1: Design Schema

Before writing SQL, document your schema:

```markdown
## New Feature: User Preferences

**Tables:**
- user_preferences: Stores user-specific settings

**Columns:**
- id: UUID (primary key)
- user_id: UUID (foreign key to users)
- preferences: JSONB (settings blob)
- created_at: TIMESTAMPTZ
- updated_at: TIMESTAMPTZ

**Indexes:**
- idx_user_preferences_user_id ON user_preferences(user_id)

**Foreign Keys:**
- user_id REFERENCES users(id) ON DELETE CASCADE
```

### Step 2: Create Migration File

```bash
cd desktop/backend-go/internal/database/migrations

# Find next available number
ls -1 *.sql | tail -1
# Output: 089_app_generation_system.sql

# Create new migration
touch 090_user_preferences.sql
```

### Step 3: Write Migration

```sql
-- +migrate Up
-- User Preferences System
-- Stores user-specific application settings

CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    preferences JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);

COMMENT ON TABLE user_preferences IS 'User-specific application settings';
COMMENT ON COLUMN user_preferences.preferences IS 'JSON blob with theme, notifications, etc.';

-- +migrate Down
DROP TABLE IF EXISTS user_preferences CASCADE;
```

### Step 4: Update schema.sql

After creating the migration, update `internal/database/schema.sql`:

```sql
-- Add your new table definition to schema.sql
-- This is required for sqlc code generation
```

### Step 5: Create sqlc Queries (if needed)

If your migration requires CRUD operations, create queries:

```sql
-- File: internal/database/queries/user_preferences.sql

-- name: GetUserPreferences :one
SELECT * FROM user_preferences
WHERE user_id = $1 LIMIT 1;

-- name: UpsertUserPreferences :one
INSERT INTO user_preferences (user_id, preferences)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET preferences = EXCLUDED.preferences,
    updated_at = NOW()
RETURNING *;

-- name: DeleteUserPreferences :exec
DELETE FROM user_preferences WHERE user_id = $1;
```

### Step 6: Regenerate sqlc Code

```bash
cd desktop/backend-go
sqlc generate
```

This generates Go code in `internal/database/sqlc/`.

---

## Testing Migrations

### Local Testing (Recommended)

Use the automated test script:

```bash
cd desktop/backend-go/scripts/migrations

# Make script executable
chmod +x test_migrations.sh

# Test all migrations (052-054, 088-089)
./test_migrations.sh

# Test specific migration
./test_migrations.sh --specific 052

# Test rollback capability
./test_migrations.sh --rollback
```

**What the script does:**
1. Creates a temporary test database
2. Applies all migrations in order
3. Verifies schema integrity
4. Tests CRUD operations
5. Checks for conflicts
6. Cleans up test database

### Manual Testing

If you prefer manual testing:

```bash
# 1. Create test database
psql -h localhost -U postgres -c "CREATE DATABASE migration_test;"

# 2. Apply migration
psql -h localhost -U postgres -d migration_test -f internal/database/migrations/052_workspace_versions.sql

# 3. Verify tables created
psql -h localhost -U postgres -d migration_test -c "\dt"

# 4. Test rollback (extract DOWN section)
sed -n '/+migrate Down/,$p' internal/database/migrations/052_workspace_versions.sql | sed '1d' | psql -h localhost -U postgres -d migration_test

# 5. Clean up
psql -h localhost -U postgres -c "DROP DATABASE migration_test;"
```

### Integration Testing

Test migrations with your Go application:

```bash
# Run application with test database
DATABASE_URL="postgres://user:pass@localhost:5432/migration_test" go run cmd/server/main.go
```

---

## Rollback Strategy

### When to Rollback

- Migration causes production errors
- Performance degradation detected
- Data corruption or loss
- Breaking changes not backward compatible

### How to Rollback

#### Automated Rollback (using script)

```bash
cd desktop/backend-go/scripts/migrations
./test_migrations.sh --rollback
```

#### Manual Rollback

```bash
# Extract DOWN migration from file
migration_file="internal/database/migrations/089_app_generation_system.sql"

# Get DOWN SQL
down_sql=$(sed -n '/+migrate Down/,$p' "$migration_file" | sed '1d')

# Apply to production (BE CAREFUL!)
echo "$down_sql" | psql "$DATABASE_URL"
```

### Rollback Safety

Not all migrations can be safely rolled back:

| Migration Type | Rollback Safety | Notes |
|----------------|-----------------|-------|
| CREATE TABLE | ✅ Safe | Data is deleted |
| ALTER TABLE ADD COLUMN | ✅ Safe | Column is dropped |
| ALTER TABLE DROP COLUMN | ⚠️ Risky | Data loss permanent |
| CREATE INDEX | ✅ Safe | Index rebuilt on reapply |
| Data migrations | ❌ Dangerous | May lose data transformations |

**Rule**: Always backup database before rollback.

---

## Staging Deployment

### Pre-Deployment Checklist

- [ ] All migrations tested locally
- [ ] sqlc code regenerated (`sqlc generate`)
- [ ] Application compiles without errors
- [ ] Integration tests pass
- [ ] Rollback plan documented
- [ ] Backup strategy confirmed

### Staging Deployment Steps

#### 1. Backup Staging Database

```bash
# Backup staging database
pg_dump "$STAGING_DATABASE_URL" > backup_$(date +%Y%m%d_%H%M%S).sql
```

#### 2. Apply Migrations

**Option A: Using test script (recommended)**

```bash
cd desktop/backend-go/scripts/migrations

# Run on staging
./test_migrations.sh --staging
```

**Option B: Manual application**

```bash
# Apply each migration
psql "$STAGING_DATABASE_URL" -f internal/database/migrations/052_workspace_versions.sql
psql "$STAGING_DATABASE_URL" -f internal/database/migrations/053_onboarding_email_metadata.sql
psql "$STAGING_DATABASE_URL" -f internal/database/migrations/054_custom_modules.sql
psql "$STAGING_DATABASE_URL" -f internal/database/migrations/088_seed_builtin_templates.sql
psql "$STAGING_DATABASE_URL" -f internal/database/migrations/089_app_generation_system.sql
```

#### 3. Verify Deployment

```bash
# Check tables created
psql "$STAGING_DATABASE_URL" -c "\dt" | grep -E "workspace_versions|onboarding_sessions|custom_modules|app_templates"

# Check indexes
psql "$STAGING_DATABASE_URL" -c "\di" | grep -E "workspace_versions|onboarding_sessions|custom_modules|app_templates"

# Check foreign keys
psql "$STAGING_DATABASE_URL" -c "
  SELECT tc.table_name, kcu.column_name, ccu.table_name AS foreign_table_name
  FROM information_schema.table_constraints AS tc
  JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name
  JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name
  WHERE tc.constraint_type = 'FOREIGN KEY'
    AND tc.table_name IN ('workspace_versions', 'onboarding_sessions', 'custom_modules', 'app_templates', 'app_generation_queue')
  ORDER BY tc.table_name;
"
```

#### 4. Run Smoke Tests

```bash
# Test basic operations
psql "$STAGING_DATABASE_URL" <<EOF
-- Test workspace_versions
INSERT INTO workspace_versions (workspace_id, version_number, snapshot_data, created_by)
VALUES (gen_random_uuid(), 'v1.0.0', '{"test": true}'::jsonb, 'test_user');

SELECT COUNT(*) FROM workspace_versions;

-- Test app_templates
SELECT COUNT(*) FROM app_templates WHERE template_name IN ('saas_dashboard', 'api_backend', 'landing_page', 'crm_module', 'task_manager');

-- Clean up test data
DELETE FROM workspace_versions WHERE created_by = 'test_user';
EOF
```

#### 5. Deploy Application

After database migrations succeed:

```bash
# Deploy updated Go application to staging
# This depends on your deployment method (Docker, Cloud Run, etc.)

# Example with Docker
cd desktop/backend-go
docker build -t businessos-backend:staging .
docker push businessos-backend:staging

# Deploy to Cloud Run (example)
gcloud run deploy businessos-backend-staging \
  --image businessos-backend:staging \
  --platform managed \
  --region us-central1
```

### Post-Deployment Verification

- [ ] Application starts without errors
- [ ] Health check passes
- [ ] API endpoints respond correctly
- [ ] No database connection errors in logs
- [ ] Frontend can interact with new features

---

## Production Deployment

### Pre-Production Checklist

- [ ] Staging deployment successful for 24+ hours
- [ ] No errors in staging logs
- [ ] Performance metrics acceptable
- [ ] Rollback plan tested on staging
- [ ] Database backup completed
- [ ] Maintenance window scheduled (if needed)
- [ ] Rollback automation ready

### Production Deployment Steps

#### 1. Schedule Maintenance Window (if needed)

For large migrations or breaking changes:

```bash
# Notify users 24-48 hours in advance
# Set status page to "maintenance scheduled"
```

#### 2. Create Production Backup

```bash
# Full database backup
pg_dump "$PRODUCTION_DATABASE_URL" | gzip > backup_prod_$(date +%Y%m%d_%H%M%S).sql.gz

# Verify backup
gunzip -c backup_prod_*.sql.gz | head -100

# Upload to secure storage
gsutil cp backup_prod_*.sql.gz gs://businessos-backups/$(date +%Y-%m)/
```

#### 3. Apply Migrations

**Use transaction wrapper for safety:**

```sql
BEGIN;

-- Apply all migrations
\i internal/database/migrations/052_workspace_versions.sql
\i internal/database/migrations/053_onboarding_email_metadata.sql
\i internal/database/migrations/054_custom_modules.sql
\i internal/database/migrations/088_seed_builtin_templates.sql
\i internal/database/migrations/089_app_generation_system.sql

-- Verify critical tables
SELECT COUNT(*) FROM workspace_versions; -- Should be 0 (new table)
SELECT COUNT(*) FROM app_templates; -- Should be 5 (seeded templates)

-- If everything looks good, commit
COMMIT;

-- If issues found, rollback
-- ROLLBACK;
```

#### 4. Deploy Application

```bash
# Deploy new application version
cd desktop/backend-go
docker build -t businessos-backend:prod .
docker push businessos-backend:prod

# Deploy to production
gcloud run deploy businessos-backend-prod \
  --image businessos-backend:prod \
  --platform managed \
  --region us-central1 \
  --no-traffic  # Deploy without traffic first

# Smoke test new version
curl https://businessos-backend-prod-canary.run.app/health

# If healthy, route traffic
gcloud run services update-traffic businessos-backend-prod --to-latest
```

#### 5. Monitor

```bash
# Watch logs for errors
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=businessos-backend-prod" --limit 100 --format json

# Check metrics
# - Response times
# - Error rates
# - Database connection pool usage
```

### Rollback Plan

If issues arise in production:

```bash
# 1. Route traffic back to old version
gcloud run services update-traffic businessos-backend-prod --to-revisions=PREVIOUS_REVISION=100

# 2. Rollback database (if needed)
psql "$PRODUCTION_DATABASE_URL" <<EOF
BEGIN;

-- Apply DOWN migrations in reverse order
\i internal/database/migrations/089_app_generation_system.sql  -- DOWN section
\i internal/database/migrations/088_seed_builtin_templates.sql  -- DOWN section
\i internal/database/migrations/054_custom_modules.sql  -- DOWN section
\i internal/database/migrations/053_onboarding_email_metadata.sql  -- DOWN section
\i internal/database/migrations/052_workspace_versions.sql  -- DOWN section

COMMIT;
EOF

# 3. Verify old version works
curl https://businessos.com/api/health
```

---

## Current Migrations (052-054, 088-089)

### Migration 052: Workspace Versions

**Purpose**: Enables workspace snapshot and restore functionality.

**Tables**:
- `workspace_versions`: Stores workspace snapshots with version numbers

**Key Features**:
- Version numbering (semantic versioning)
- Snapshot data (JSONB)
- Metadata (app count, member count, etc.)

**Dependencies**:
- Requires `workspaces` table

**Rollback**: Safe (no data loss concern for new table)

---

### Migration 053: Onboarding Email Metadata

**Purpose**: Stores email analysis data for AI-powered onboarding.

**Tables**:
- `onboarding_sessions`: Tracks onboarding progress
- `onboarding_conversation_history`: Stores Q&A flow
- `onboarding_email_metadata`: Per-email extracted metadata

**Key Features**:
- Email sentiment analysis
- Tool detection (Slack, Notion, etc.)
- Confidence scoring
- Fallback triggers

**Dependencies**:
- Requires `"user"` table
- Requires `workspaces` table

**Rollback**: Safe (new tables, cascade delete)

---

### Migration 054: Custom Modules

**Purpose**: User-created modules and marketplace system.

**Tables**:
- `custom_modules`: Main module registry
- `custom_module_versions`: Version history
- `custom_module_installations`: Workspace installations
- `custom_module_shares`: Sharing permissions

**Key Features**:
- Module marketplace (public/private)
- Version control
- Installation tracking
- Granular sharing (user/workspace/email)

**Dependencies**:
- None (uses UUIDs without FK validation)

**Rollback**: Safe (cascade delete)

---

### Migration 088: Seed Builtin Templates

**Purpose**: Inserts 5 default app templates (SaaS, API, Landing, CRM, Task Manager).

**Data Inserted**:
- `saas_dashboard`: Full-featured dashboard
- `api_backend`: Go REST API
- `landing_page`: Marketing site
- `crm_module`: Sales pipeline
- `task_manager`: Kanban board

**Key Features**:
- Upsert logic (ON CONFLICT DO UPDATE)
- Priority scoring for recommendations
- Target business types and team sizes

**Dependencies**:
- Requires `app_templates` table (from migration 089)

**Rollback**: Deletes seeded data (not reversible if users customized)

**⚠️ Important**: This migration depends on 089 running first. Apply in correct order: 089 → 088.

---

### Migration 089: App Generation System

**Purpose**: Creates tables for personalized app generation.

**Tables**:
- `app_templates`: Template definitions
- `user_generated_apps`: User-created apps
- `app_generation_queue`: Async generation queue

**Key Features**:
- Template recommendation system
- OSA integration (`osa_app_id` reference)
- Generation queue with retries
- Priority-based processing

**Dependencies**:
- Requires `workspaces` table
- Loosely references `osa_apps` (SET NULL on delete)

**Rollback**: Safe (cascade delete)

---

## Troubleshooting

### Common Issues

#### Issue: "relation does not exist"

**Cause**: Migration applied out of order or dependency missing.

**Solution**:
```bash
# Check if dependency table exists
psql "$DATABASE_URL" -c "\dt workspaces"

# If missing, apply earlier migrations first
# Then reapply failed migration
```

#### Issue: "duplicate key value violates unique constraint"

**Cause**: Data already exists (e.g., rerunning seed migration).

**Solution**:
```sql
-- Use ON CONFLICT for idempotent migrations
INSERT INTO app_templates (template_name, ...)
VALUES ('saas_dashboard', ...)
ON CONFLICT (template_name) DO UPDATE SET ...;
```

#### Issue: "column does not exist"

**Cause**: sqlc queries not regenerated after schema change.

**Solution**:
```bash
cd desktop/backend-go
sqlc generate
go build ./...
```

#### Issue: "foreign key violation"

**Cause**: Referenced table/row does not exist.

**Solution**:
```sql
-- Check referenced table
SELECT * FROM workspaces LIMIT 1;

-- If empty, insert test data or adjust FK constraints
ALTER TABLE workspace_versions ALTER COLUMN workspace_id DROP NOT NULL;
```

### Performance Issues

#### Slow Migration Execution

**Cause**: Large table, missing indexes, or locking.

**Solution**:
```sql
-- Use CONCURRENTLY for large indexes (cannot be in transaction)
CREATE INDEX CONCURRENTLY idx_large_table_column ON large_table(column);

-- Add batching for data migrations
DO $$
DECLARE
    batch_size INT := 10000;
    offset_val INT := 0;
BEGIN
    LOOP
        UPDATE large_table SET ...
        WHERE id IN (SELECT id FROM large_table LIMIT batch_size OFFSET offset_val);

        IF NOT FOUND THEN EXIT; END IF;
        offset_val := offset_val + batch_size;
        COMMIT;
    END LOOP;
END $$;
```

### Debugging Tips

```bash
# Enable query logging
export PGOPTIONS="-c log_statement=all"

# Run migration with verbose output
psql -v ON_ERROR_STOP=1 -e "$DATABASE_URL" -f migration_file.sql

# Check PostgreSQL logs
tail -f /var/log/postgresql/postgresql-*.log
```

---

## Best Practices Summary

### DO ✅

- **Always test locally first**
- **Use transactions for multiple migrations**
- **Backup before production deployment**
- **Use IF NOT EXISTS for idempotent migrations**
- **Add indexes AFTER table creation**
- **Use CASCADE for cleanup**
- **Document rollback strategy**
- **Regenerate sqlc code after schema changes**

### DON'T ❌

- **Don't skip testing on staging**
- **Don't apply migrations directly to production without backup**
- **Don't use ALTER TABLE DROP COLUMN without data migration**
- **Don't forget foreign key ON DELETE actions**
- **Don't use TIMESTAMP (use TIMESTAMPTZ)**
- **Don't hardcode UUIDs (use gen_random_uuid())**
- **Don't ignore migration conflicts**

---

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [sqlc Documentation](https://docs.sqlc.dev/)
- [migrate Tool](https://github.com/golang-migrate/migrate)
- [BusinessOS Architecture Docs](../../../docs/ARCHITECTURE.md)

---

**Last Updated**: 2026-01-26
**Maintained By**: BusinessOS Database Team
**Related Docs**: [Schema Design](./SCHEMA_DESIGN.md), [Performance Tuning](./PERFORMANCE.md)
