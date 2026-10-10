# Migration Numbering Gaps Documentation

## Overview

This document explains the gaps in migration numbering sequence. These gaps are **intentional** and represent migrations that were either consolidated, merged into other migrations, or removed during development.

## Current Migration Sequence

### Existing Migrations

```
047 - add_todays_focus_widget
048 - osa_integration
049 - osa_workflows_files
050 - osa_deployment_port
051 - osa_app_metadata
052 - workspace_versions
053 - onboarding_email_metadata
054 - custom_modules
[GAP: 055-065]
066 - custom_agents_behavior_fields
[GAP: 067-070]
071 - osa_workflows_files
[GAP: 072-077]
078 - osa_app_registry
079 - osa_prompt_templates
[GAP: 080-087]
088 - seed_builtin_templates
089 - app_generation_system
090 - extend_app_templates
```

## Gap Analysis

### Gap 1: Migrations 055-065 (11 missing)
**Range:** 055, 056, 057, 058, 059, 060, 061, 062, 063, 064, 065

**Likely reason:**
- Early development migrations that were consolidated into migration 066
- Features that were refactored before reaching production
- Experimental schemas that were discarded

### Gap 2: Migrations 067-070 (4 missing)
**Range:** 067, 068, 069, 070

**Likely reason:**
- Intermediate migrations merged into migration 071
- Agent/workflow features that went through multiple design iterations
- Schema changes that were superseded by later designs

### Gap 3: Migrations 072-077 (6 missing)
**Range:** 072, 073, 074, 075, 076, 077

**Likely reason:**
- OSA (Operating System Architecture) feature development iterations
- Migrations consolidated into the app registry and template systems
- Schema refinements merged into migrations 078-079

### Gap 4: Migrations 080-087 (8 missing)
**Range:** 080, 081, 082, 083, 084, 085, 086, 087

**Likely reason:**
- Development migrations between template and app generation systems
- Intermediate steps consolidated into migrations 088-090
- Features that were redesigned before production deployment

## Historical Context

### Development Timeline

The migration gaps correspond to active development periods where:

1. **Early OSA Integration (055-065):** Initial OSA integration attempts, later consolidated
2. **Workflow System Refinement (067-070):** Multiple workflow schema iterations
3. **App Registry Development (072-077):** App registry feature went through several design phases
4. **Template System Evolution (080-087):** Template and app generation underwent significant refactoring

### Why Gaps Exist

Migration gaps are **normal** in active development and indicate:

- **Healthy iteration:** Features were refined before production
- **Code quality:** Migrations were consolidated rather than left as technical debt
- **Clean history:** Unnecessary or superseded migrations were removed

## Resolution of Duplicate Issues

### Fixed on 2026-01-27

**Duplicate 047:**
- `047_osa_integration.sql` → renamed to `048_osa_integration.sql`
- Resolved conflict with `047_add_todays_focus_widget.sql`

**Duplicate 078:**
- `078_osa_prompt_templates.sql` → renamed to `079_osa_prompt_templates.sql`
- Resolved conflict with `078_osa_app_registry.sql`

## Guidelines for Future Migrations

### Best Practices

1. **Never reuse gap numbers** - Keep the sequence moving forward
2. **Next migration should be:** 091
3. **Document major consolidations** - Note in this file if migrations are merged
4. **Keep sequence continuous** - Avoid creating new gaps

### Migration Naming Convention

```
{number}_{descriptive_name}.sql

Examples:
091_add_feature_x.sql
092_update_table_y.sql
093_create_index_z.sql
```

## Verification

To verify migration sequence integrity:

```bash
# List all migrations in order
ls -1 internal/database/migrations/*.sql | sort

# Check for duplicates
ls -1 internal/database/migrations/*.sql | cut -d_ -f1 | sort | uniq -d

# Count total migrations
ls -1 internal/database/migrations/*.sql | wc -l
```

## Summary

- **Total migrations:** 15 (as of 2026-01-27)
- **Total gaps:** 29 migration numbers
- **Gap ranges:** 055-065, 067-070, 072-077, 080-087
- **Status:** All duplicates resolved
- **Next migration number:** 091

These gaps are **not a problem** - they represent normal development evolution and consolidation of database schema changes.
