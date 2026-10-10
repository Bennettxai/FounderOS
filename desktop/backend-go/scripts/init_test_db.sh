#!/bin/bash
# Initialize test database for CI/CD
# This script sets up the full BusinessOS schema in the test database

set -e

# Database connection from environment or default
DATABASE_URL=${DATABASE_URL:-"postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable"}

echo "🔧 Initializing test database..."
echo "📍 Connection: $DATABASE_URL"

# Extract connection parameters for psql
DB_HOST=$(echo $DATABASE_URL | sed -n 's/.*@\([^:]*\):.*/\1/p')
DB_PORT=$(echo $DATABASE_URL | sed -n 's/.*:\([0-9]*\)\/.*/\1/p')
DB_NAME=$(echo $DATABASE_URL | sed -n 's/.*\/\([^?]*\).*/\1/p')
DB_USER=$(echo $DATABASE_URL | sed -n 's/.*\/\/\([^:]*\):.*/\1/p')
DB_PASS=$(echo $DATABASE_URL | sed -n 's/.*:\/\/[^:]*:\([^@]*\)@.*/\1/p')

export PGPASSWORD="$DB_PASS"

echo "📦 Step 1: Creating database if not exists..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "   Database already exists"

echo "📦 Step 2: Applying init.sql (base schema)..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f internal/database/init.sql

echo "📦 Step 3: Creating schema_migrations tracking table..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
EOF

echo "📦 Step 4: Applying migrations..."
# Apply migrations in order (numbered 002-046)
for migration_file in internal/database/migrations/0*.sql internal/database/migrations/supabase_migration.sql; do
    if [ -f "$migration_file" ]; then
        echo "   Applying: $migration_file"
        psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$migration_file" 2>&1 | grep -v "already exists" | grep -v "duplicate key" || true
    fi
done

echo "✅ Test database initialized successfully!"
echo "📊 Verifying table count..."
TABLE_COUNT=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo "   Tables created: $TABLE_COUNT"

echo "✅ Database ready for tests!"
