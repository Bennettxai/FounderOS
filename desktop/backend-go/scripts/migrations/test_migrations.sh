#!/usr/bin/env bash
# ============================================================================
# BusinessOS Migration Testing Script
# ============================================================================
# Tests database migrations (052-054, 088-089) for conflicts and integrity
#
# Usage:
#   ./test_migrations.sh                    # Test all migrations
#   ./test_migrations.sh --rollback         # Test rollback capability
#   ./test_migrations.sh --staging          # Run on staging DB
#   ./test_migrations.sh --specific 052     # Test specific migration
#
# Prerequisites:
#   - PostgreSQL client (psql)
#   - Database credentials in .env
#   - golang-migrate CLI (optional, for rollback testing)
# ============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/internal/database/migrations"

# Load environment variables
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

# Default configuration
TEST_MODE="${1:-all}"
SPECIFIC_MIGRATION="${2:-}"
DB_URL="${DATABASE_URL:-}"
TEST_DB_NAME="businessos_migration_test"

# ============================================================================
# Helper Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_banner() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║         BusinessOS Migration Testing Tool                     ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if psql is installed
    if ! command -v psql &> /dev/null; then
        log_error "psql (PostgreSQL client) is not installed"
        exit 1
    fi

    # Check if DATABASE_URL is set
    if [ -z "$DB_URL" ]; then
        log_error "DATABASE_URL not set in environment"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

create_test_database() {
    log_info "Creating test database: $TEST_DB_NAME"

    # Extract connection parameters from DATABASE_URL
    DB_HOST=$(echo "$DB_URL" | sed -n 's|.*@\([^:]*\):.*|\1|p')
    DB_PORT=$(echo "$DB_URL" | sed -n 's|.*:\([0-9]*\)/.*|\1|p')
    DB_USER=$(echo "$DB_URL" | sed -n 's|.*://\([^:]*\):.*|\1|p')
    DB_PASSWORD=$(echo "$DB_URL" | sed -n 's|.*://[^:]*:\([^@]*\)@.*|\1|p')

    # Create test database
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $TEST_DB_NAME;"

    log_success "Test database created"
}

drop_test_database() {
    log_info "Dropping test database: $TEST_DB_NAME"

    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $TEST_DB_NAME;" 2>/dev/null || true

    log_success "Test database dropped"
}

apply_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file")

    log_info "Applying migration: $migration_name"

    # Extract UP migration (everything between +migrate Up and +migrate Down)
    local up_sql=$(sed -n '/+migrate Up/,/+migrate Down/p' "$migration_file" | sed '1d;$d')

    if [ -z "$up_sql" ]; then
        log_error "No UP migration found in $migration_name"
        return 1
    fi

    # Apply migration
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -c "$up_sql" 2>&1 | tee /tmp/migration_output.log

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "Migration $migration_name applied successfully"
        return 0
    else
        log_error "Migration $migration_name failed"
        cat /tmp/migration_output.log
        return 1
    fi
}

rollback_migration() {
    local migration_file="$1"
    local migration_name=$(basename "$migration_file")

    log_info "Rolling back migration: $migration_name"

    # Extract DOWN migration (everything after +migrate Down)
    local down_sql=$(sed -n '/+migrate Down/,$p' "$migration_file" | sed '1d')

    if [ -z "$down_sql" ]; then
        log_warning "No DOWN migration found in $migration_name (some migrations don't have rollback)"
        return 0
    fi

    # Apply rollback
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -c "$down_sql" 2>&1 | tee /tmp/migration_rollback.log

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log_success "Migration $migration_name rolled back successfully"
        return 0
    else
        log_error "Rollback of $migration_name failed"
        cat /tmp/migration_rollback.log
        return 1
    fi
}

verify_schema_integrity() {
    log_info "Verifying schema integrity..."

    # Check for table existence
    local tables_query="
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name IN ('workspace_versions', 'onboarding_sessions', 'onboarding_email_metadata',
                             'custom_modules', 'app_templates', 'user_generated_apps', 'app_generation_queue')
        ORDER BY table_name;
    "

    local tables=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -t -c "$tables_query")

    echo "Tables created:"
    echo "$tables"

    # Check for foreign key constraints
    log_info "Checking foreign key constraints..."
    local fk_query="
        SELECT
            tc.table_name,
            kcu.column_name,
            ccu.table_name AS foreign_table_name,
            ccu.column_name AS foreign_column_name
        FROM information_schema.table_constraints AS tc
        JOIN information_schema.key_column_usage AS kcu
          ON tc.constraint_name = kcu.constraint_name
          AND tc.table_schema = kcu.table_schema
        JOIN information_schema.constraint_column_usage AS ccu
          ON ccu.constraint_name = tc.constraint_name
          AND ccu.table_schema = tc.table_schema
        WHERE tc.constraint_type = 'FOREIGN KEY'
          AND tc.table_name IN ('workspace_versions', 'onboarding_sessions', 'onboarding_email_metadata',
                                'custom_modules', 'custom_module_versions', 'custom_module_installations',
                                'user_generated_apps', 'app_generation_queue')
        ORDER BY tc.table_name;
    "

    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -c "$fk_query"

    # Check for indexes
    log_info "Checking indexes..."
    local idx_query="
        SELECT
            tablename,
            indexname,
            indexdef
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename IN ('workspace_versions', 'onboarding_sessions', 'onboarding_email_metadata',
                           'custom_modules', 'app_templates', 'user_generated_apps', 'app_generation_queue')
        ORDER BY tablename, indexname;
    "

    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -c "$idx_query"

    log_success "Schema integrity verification complete"
}

test_data_operations() {
    log_info "Testing basic CRUD operations..."

    # Test workspace_versions
    log_info "Testing workspace_versions table..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" <<EOF
-- Insert test workspace version
INSERT INTO workspace_versions (workspace_id, version_number, snapshot_data, created_by)
VALUES (
    gen_random_uuid(),
    'v1.0.0',
    '{"apps": [], "members": []}'::jsonb,
    'test_user'
);

-- Query to verify
SELECT COUNT(*) as workspace_version_count FROM workspace_versions;
EOF

    # Test app_templates
    log_info "Testing app_templates table..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" <<EOF
-- Insert test template
INSERT INTO app_templates (template_name, category, display_name, description, scaffold_type)
VALUES (
    'test_template',
    'operations',
    'Test Template',
    'A test template for validation',
    'svelte'
);

-- Query to verify
SELECT COUNT(*) as template_count FROM app_templates;
EOF

    # Test custom_modules
    log_info "Testing custom_modules table..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" <<EOF
-- Insert test module
INSERT INTO custom_modules (created_by, workspace_id, name, slug, category, manifest)
VALUES (
    gen_random_uuid(),
    gen_random_uuid(),
    'Test Module',
    'test-module',
    'utility',
    '{}'::jsonb
);

-- Query to verify
SELECT COUNT(*) as module_count FROM custom_modules;
EOF

    log_success "CRUD operations test complete"
}

check_migration_conflicts() {
    log_info "Checking for migration conflicts..."

    # Check for duplicate table definitions
    local duplicate_tables=$(grep -r "CREATE TABLE" "$MIGRATIONS_DIR" | grep -E "(052|053|054|088|089)" | awk '{print $4}' | sort | uniq -d)

    if [ -n "$duplicate_tables" ]; then
        log_error "Duplicate table definitions found:"
        echo "$duplicate_tables"
        return 1
    fi

    # Check for missing dependencies
    log_info "Checking foreign key dependencies..."

    # Ensure workspaces table exists (referenced by workspace_versions)
    local has_workspaces=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$TEST_DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'workspaces';")

    if [ "$has_workspaces" -eq 0 ]; then
        log_warning "workspaces table not found (required by workspace_versions)"
    fi

    log_success "No migration conflicts detected"
}

# ============================================================================
# Main Test Functions
# ============================================================================

test_all_migrations() {
    log_info "Testing migrations: 052-054, 088-089"

    # Array of migrations to test
    migrations=(
        "052_workspace_versions.sql"
        "053_onboarding_email_metadata.sql"
        "054_custom_modules.sql"
        "088_seed_builtin_templates.sql"
        "089_app_generation_system.sql"
    )

    local failed=0

    for migration in "${migrations[@]}"; do
        local migration_path="$MIGRATIONS_DIR/$migration"

        if [ ! -f "$migration_path" ]; then
            log_error "Migration file not found: $migration"
            failed=$((failed + 1))
            continue
        fi

        if ! apply_migration "$migration_path"; then
            failed=$((failed + 1))
        fi
    done

    if [ $failed -eq 0 ]; then
        log_success "All migrations applied successfully"

        # Run integrity checks
        verify_schema_integrity
        test_data_operations
        check_migration_conflicts

        return 0
    else
        log_error "$failed migration(s) failed"
        return 1
    fi
}

test_rollback() {
    log_info "Testing rollback capability..."

    # Apply all migrations first
    test_all_migrations

    # Rollback in reverse order
    migrations=(
        "089_app_generation_system.sql"
        "088_seed_builtin_templates.sql"
        "054_custom_modules.sql"
        "053_onboarding_email_metadata.sql"
        "052_workspace_versions.sql"
    )

    local failed=0

    for migration in "${migrations[@]}"; do
        local migration_path="$MIGRATIONS_DIR/$migration"

        if [ ! -f "$migration_path" ]; then
            log_error "Migration file not found: $migration"
            failed=$((failed + 1))
            continue
        fi

        if ! rollback_migration "$migration_path"; then
            failed=$((failed + 1))
        fi
    done

    if [ $failed -eq 0 ]; then
        log_success "All rollbacks completed successfully"
        return 0
    else
        log_error "$failed rollback(s) failed"
        return 1
    fi
}

test_specific_migration() {
    local migration_num="$1"
    log_info "Testing specific migration: $migration_num"

    # Find migration file
    local migration_file=$(find "$MIGRATIONS_DIR" -name "${migration_num}_*.sql" | head -1)

    if [ -z "$migration_file" ]; then
        log_error "Migration $migration_num not found"
        return 1
    fi

    apply_migration "$migration_file"
    verify_schema_integrity
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    print_banner
    check_prerequisites

    # Create test database
    create_test_database

    # Trap to ensure cleanup
    trap drop_test_database EXIT

    # Run tests based on mode
    case "$TEST_MODE" in
        all)
            test_all_migrations
            ;;
        --rollback)
            test_rollback
            ;;
        --specific)
            if [ -z "$SPECIFIC_MIGRATION" ]; then
                log_error "Please specify migration number: ./test_migrations.sh --specific 052"
                exit 1
            fi
            test_specific_migration "$SPECIFIC_MIGRATION"
            ;;
        --staging)
            log_warning "Running on staging database (BE CAREFUL!)"
            read -p "Are you sure you want to run on staging? (yes/no): " confirm
            if [ "$confirm" == "yes" ]; then
                TEST_DB_NAME="postgres"  # Use actual staging DB
                test_all_migrations
            else
                log_info "Staging test cancelled"
            fi
            ;;
        *)
            log_error "Unknown test mode: $TEST_MODE"
            echo "Usage: $0 [all|--rollback|--specific NUM|--staging]"
            exit 1
            ;;
    esac

    log_success "Migration testing complete!"
}

# Run main function
main "$@"
