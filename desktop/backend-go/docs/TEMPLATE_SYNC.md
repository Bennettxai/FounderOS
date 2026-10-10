# Template Sync System

## Overview

The Template Sync System synchronizes YAML template definitions from `internal/prompts/templates/osa/` to the PostgreSQL database (`app_templates` table). YAML files are the **source of truth**.

## Architecture

```
YAML Templates (Source of Truth)
       ↓
Template Sync Service
       ↓
PostgreSQL Database (app_templates table)
       ↓
App Template Service (API reads from DB)
```

## Components

### 1. Template Sync Service
**Location**: `internal/services/template_sync_service.go`

**Key Functions**:
- `SyncTemplates()` - Syncs all YAML templates to database
- `MapYAMLToDB()` - Maps YAML structure to DB columns
- `GetTemplateByName()` - Retrieves synced template from DB

**Features**:
- Auto-detects category-based metadata (icon, scaffold type)
- Calculates priority scores based on tags and category
- Extracts business types, challenges, and team sizes
- Stores template variables as JSONB for validation

### 2. Migration
**Location**: `internal/database/migrations/090_extend_app_templates.sql`

**Adds**:
- `yaml_template_name` - Reference to YAML file
- `yaml_version` - Template version
- `template_variables` - JSONB of variable definitions

### 3. CLI Sync Tool
**Location**: `cmd/sync-templates/main.go`

**Usage**:
```bash
# From project root
go run cmd/sync-templates/main.go

# Or build and run
go build -o bin/sync-templates ./cmd/sync-templates
./bin/sync-templates
```

**Environment Variables**:
- `TEMPLATES_DIR` - Override templates directory (default: `internal/prompts/templates/osa`)
- `DATABASE_URL` - PostgreSQL connection string
- `TEST_DATABASE_URL` - Test database for integration tests

### 4. Auto-Sync on Server Startup
**Location**: `cmd/server/main.go` (line ~112)

**Enable**:
```bash
export SYNC_TEMPLATES_ON_STARTUP=true
./bin/server
```

**Configuration**:
- Runs with 30-second timeout
- Non-blocking (warns on failure, doesn't fail startup)
- Only runs if database is connected

## Usage

### Manual Sync

```bash
cd desktop/backend-go

# Sync templates
go run cmd/sync-templates/main.go
```

**Output**:
```
============================================================
TEMPLATE SYNC RESULTS
============================================================
Inserted: 2
Updated:  3
Skipped:  0
Errors:   0
============================================================
```

### Auto-Sync on Startup

Add to `.env`:
```
SYNC_TEMPLATES_ON_STARTUP=true
TEMPLATES_DIR=internal/prompts/templates/osa  # optional
```

Start server:
```bash
go run cmd/server/main.go
```

Logs:
```
INFO syncing YAML templates to database
INFO template sync completed inserted=2 updated=3 errors=0
```

### Programmatic Usage

```go
import "github.com/rhl/businessos-backend/internal/services"

// Initialize service
syncService := services.NewTemplateSyncService(
    pool,
    slog.Default(),
    "internal/prompts/templates/osa",
)

// Sync templates
result, err := syncService.SyncTemplates(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Synced: %d inserted, %d updated\n", result.Inserted, result.Updated)
```

## YAML Template Format

### Example Template

```yaml
name: "my-template"
display_name: "My Template"
description: "Description of what this template does"
category: "app-generation"  # or: data-visualization, maintenance, feature
version: "1.0.0"
tags:
  - "tag1"
  - "tag2"

variables:
  - name: "RequiredVar"
    type: "string"
    required: true
    description: "A required variable"

  - name: "OptionalVar"
    type: "string"
    required: false
    default: "default-value"
    description: "An optional variable"

template: |
  # Generated Prompt

  This is the template content with {{.RequiredVar}} and {{.OptionalVar}}.
```

### Category to Icon Mapping

| Category | Icon | Scaffold Type | Priority Base |
|----------|------|---------------|---------------|
| `app-generation` | `users` | `full-stack` | 85 |
| `data-visualization` | `chart` | `svelte` | 90 |
| `maintenance` | `wrench` | `go` | 75 |
| `feature` | `plus` | `svelte` | 80 |
| `operations` | `server` | `svelte` | 70 |
| `marketing` | `globe` | `svelte` | 70 |

### Priority Score Calculation

**Base Score** (by category):
- App Generation: 85
- Data Visualization: 90
- Maintenance: 75
- Feature: 80

**Adjustments** (by tags):
- `+5`: full-stack, crm, dashboard
- `-5`: bug

**Range**: 1-100

### Target Extraction

**Business Types** (from category + tags):
- `app-generation`, `crm` → saas, startup, enterprise, small_business
- `data-visualization` → saas, enterprise, agency
- `maintenance` → all types

**Challenges** (from category + tags):
- `app-generation` → rapid_prototyping, scalability, time_to_market
- `data-visualization` → analytics, reporting, data_insights
- `maintenance` → bug_fixing, code_quality, stability
- Tag `crm` → client_relationships
- Tag `analytics` → analytics
- Tag `bug` → bug_fixing

**Team Sizes**:
- Default: solo, small, medium, large (suitable for all)

## Database Schema

### app_templates Table

```sql
CREATE TABLE app_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(100) UNIQUE NOT NULL,
    category VARCHAR(50) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    icon_type VARCHAR(50) NOT NULL,
    target_business_types TEXT[] NOT NULL,
    target_challenges TEXT[] NOT NULL,
    target_team_sizes TEXT[] NOT NULL,
    priority_score INTEGER NOT NULL,
    template_config JSONB,
    required_modules TEXT[],
    optional_features TEXT[],
    generation_prompt TEXT NOT NULL,
    scaffold_type VARCHAR(50),

    -- YAML metadata (Migration 090)
    yaml_template_name VARCHAR(100),
    yaml_version VARCHAR(20),
    template_variables JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_templates_yaml_name ON app_templates(yaml_template_name);
```

## Testing

### Unit Tests

```bash
# Test mapping logic
go test ./internal/services -run TestTemplateSyncService_MapYAMLToDB -v

# Test category mapping
go test ./internal/services -run TestTemplateSyncService_CategoryToIcon -v

# Test priority calculation
go test ./internal/services -run TestTemplateSyncService_CalculatePriorityScore -v

# Test all sync service tests
go test ./internal/services -run TestTemplateSyncService -v
```

### Integration Tests

```bash
# Set test database
export TEST_DATABASE_URL="postgresql://user:pass@localhost:5432/businessos_test"

# Run integration tests
go test ./internal/services -run TestTemplateSyncService_Integration -v
```

**Tests**:
- `SyncTemplates_Integration` - Syncs templates to test DB
- `GetTemplateByName_Integration` - Retrieves synced templates
- `IdempotentSync_Integration` - Verifies sync can run multiple times

## Development Workflow

### Adding a New Template

1. **Create YAML file**:
   ```bash
   touch internal/prompts/templates/osa/my-new-template.yaml
   ```

2. **Define template structure** (see format above)

3. **Sync to database**:
   ```bash
   go run cmd/sync-templates/main.go
   ```

4. **Verify in database**:
   ```sql
   SELECT * FROM app_templates WHERE yaml_template_name = 'my-new-template';
   ```

### Updating an Existing Template

1. **Edit YAML file**: Modify `internal/prompts/templates/osa/<name>.yaml`

2. **Re-sync**:
   ```bash
   go run cmd/sync-templates/main.go
   ```

3. **Verify update**:
   - Check `updated_at` timestamp in database
   - Verify new fields are updated

### Removing a Template

**Option 1: Keep in DB (recommended)**
- Move YAML file to archive or delete it
- Template remains in DB for historical data
- Users can still reference it

**Option 2: Full Removal**
- Delete YAML file
- Manually delete from DB:
  ```sql
  DELETE FROM app_templates WHERE yaml_template_name = 'old-template';
  ```

## Troubleshooting

### Sync Fails

**Error**: `Failed to find YAML files`
```bash
# Check templates directory exists
ls internal/prompts/templates/osa/

# Set explicit path
export TEMPLATES_DIR=/absolute/path/to/templates
go run cmd/sync-templates/main.go
```

**Error**: `Database connection failed`
```bash
# Verify DATABASE_URL is set
echo $DATABASE_URL

# Test database connection
psql $DATABASE_URL -c "SELECT 1;"
```

### Template Not Syncing

**Check YAML syntax**:
```bash
# Install yq (YAML parser)
brew install yq  # macOS
# or
apt install yq   # Linux

# Validate YAML
yq eval internal/prompts/templates/osa/my-template.yaml
```

**Check logs**:
```bash
# Run with verbose logging
go run cmd/sync-templates/main.go 2>&1 | grep ERROR
```

### Priority Score Unexpected

```bash
# Check calculated score
go test ./internal/services -run TestTemplateSyncService_CalculatePriorityScore -v
```

**Adjust**:
- Change category (affects base score)
- Add/remove tags (affects adjustments)

## API Integration

Templates synced to DB are accessible via:

```
GET /api/osa/templates
```

**Response**:
```json
{
  "templates": [
    {
      "id": "uuid",
      "template_name": "crm-app-generation",
      "display_name": "CRM Application Generation",
      "category": "app-generation",
      "icon_type": "users",
      "priority_score": 90,
      "yaml_version": "1.0.0"
    }
  ]
}
```

## Best Practices

1. **YAML as Source of Truth**
   - Always edit YAML files, not database directly
   - Re-sync after every YAML change

2. **Version Control**
   - Commit YAML files to Git
   - Don't commit database state

3. **Testing**
   - Test YAML changes locally before deploying
   - Run integration tests after sync

4. **Production Deployment**
   - Run sync as part of deployment pipeline
   - Use `SYNC_TEMPLATES_ON_STARTUP=true` in production
   - Monitor sync results in logs

5. **Backwards Compatibility**
   - Don't remove variables from existing templates
   - Add new variables as optional with defaults
   - Update `version` field when making changes

## Migration Guide

### From Hardcoded Templates (Migration 088)

**Before** (hardcoded SQL):
```sql
INSERT INTO app_templates (template_name, ...) VALUES ('crm_module', ...);
```

**After** (YAML → DB sync):
```yaml
# internal/prompts/templates/osa/crm-app-generation.yaml
name: "crm-app-generation"
display_name: "CRM Application Generation"
...
```

**Migration Steps**:
1. Run Migration 090 (adds YAML metadata columns)
2. Create YAML files for existing templates
3. Run sync to populate `yaml_template_name` field
4. Verify `yaml_template_name IS NOT NULL` for all templates

## Performance

### Sync Performance

- **5 templates**: ~100ms
- **50 templates**: ~500ms
- **500 templates**: ~5s

### Optimization

- Sync runs in transaction (atomic)
- Indexes on `yaml_template_name` for fast lookups
- Template variables stored as JSONB (efficient queries)

## Security

- Sync service requires database connection (authenticated)
- No user input validation needed (internal YAML files)
- Template content is trusted (not user-generated)

## Monitoring

**Metrics to Track**:
- Sync success/failure rate
- Number of templates synced
- Sync duration
- Template usage (which templates are most used)

**Logs**:
```
INFO template sync completed inserted=X updated=Y errors=Z
WARN template sync failed error="connection timeout"
ERROR failed to sync template file=bug-fix.yaml error="invalid YAML"
```

## Future Enhancements

- [ ] Template versioning (track history of changes)
- [ ] Template dependencies (template A requires template B)
- [ ] Template testing framework (validate generated output)
- [ ] Template analytics (usage metrics, popularity)
- [ ] Multi-language templates (i18n support)
- [ ] Template marketplace (community templates)

## Related Documentation

- [OSA Architecture](./OSA_ARCHITECTURE.md)
- [Database Migrations](./MIGRATIONS.md)
- [Testing Guide](./TESTING.md)
- [API Documentation](./API.md)
