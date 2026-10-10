#!/bin/bash
# Backup database before E2E testing
# Usage: ./scripts/backup_database.sh [backup_name]

BACKUP_NAME="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
BACKUP_DIR="./backups"

echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                    DATABASE BACKUP TOOL                          ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""

# Load .env
if [ -f ".env" ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo "✅ Loaded .env file"
else
    echo "❌ .env file not found"
    exit 1
fi

# Check DATABASE_URL
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL not set"
    exit 1
fi

echo ""
echo "📊 Backup Configuration:"
echo "   Backup name: $BACKUP_NAME"
echo "   Backup dir:  $BACKUP_DIR"
echo ""

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Parse DATABASE_URL to extract host, port, database, user
# Format: postgresql://user:pass@host:port/database
DB_HOST=$(echo $DATABASE_URL | sed -n 's/.*@\([^:]*\):.*/\1/p')
DB_PORT=$(echo $DATABASE_URL | sed -n 's/.*:\([0-9]*\)\/.*/\1/p')
DB_NAME=$(echo $DATABASE_URL | sed -n 's/.*\/\([^?]*\).*/\1/p')
DB_USER=$(echo $DATABASE_URL | sed -n 's/.*\/\/\([^:]*\):.*/\1/p')

echo "🔍 Database Info:"
echo "   Host:     $DB_HOST"
echo "   Port:     $DB_PORT"
echo "   Database: $DB_NAME"
echo "   User:     $DB_USER"
echo ""

# Backup strategy: Export specific tables only (not full dump)
echo "📦 Creating backup..."
echo ""

# Tables to backup
TABLES=(
    "user"
    "workspace"
    "workspace_onboarding_profiles"
    "onboarding_user_analysis"
    "app_templates"
    "user_generated_apps"
    "app_generation_queue"
    "workspace_versions"
)

BACKUP_FILE="$BACKUP_DIR/$BACKUP_NAME.sql"

echo "-- BusinessOS Database Backup" > "$BACKUP_FILE"
echo "-- Created: $(date)" >> "$BACKUP_FILE"
echo "-- Database: $DB_NAME" >> "$BACKUP_FILE"
echo "" >> "$BACKUP_FILE"

for table in "${TABLES[@]}"; do
    echo "   Backing up: $table"

    # Export table structure
    echo "-- Table: $table (structure)" >> "$BACKUP_FILE"
    psql "$DATABASE_URL" -c "\d $table" >> "$BACKUP_FILE" 2>&1
    echo "" >> "$BACKUP_FILE"

    # Export table data (if exists)
    echo "-- Table: $table (data)" >> "$BACKUP_FILE"
    psql "$DATABASE_URL" -c "COPY (SELECT * FROM \"$table\") TO STDOUT WITH CSV HEADER" >> "$BACKUP_FILE.${table}.csv" 2>/dev/null

    if [ -f "$BACKUP_FILE.${table}.csv" ]; then
        echo "   ✅ $table: $(wc -l < "$BACKUP_FILE.${table}.csv") rows"
    else
        echo "   ⚠️  $table: table not found or empty"
    fi
done

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║                    BACKUP COMPLETE                               ║"
echo "╚══════════════════════════════════════════════════════════════════╝"
echo ""
echo "✅ Backup saved to: $BACKUP_FILE"
echo "✅ CSV exports in: $BACKUP_DIR/$BACKUP_NAME.*.csv"
echo ""
echo "📊 Backup size:"
du -h "$BACKUP_DIR/$BACKUP_NAME"* | awk '{print "   "$0}'
echo ""
echo "💡 To restore:"
echo "   1. Drop and recreate tables (DANGEROUS!)"
echo "   2. Import CSV files using COPY command"
echo "   3. Or use: psql \$DATABASE_URL < $BACKUP_FILE"
echo ""
echo "⚠️  Note: This is a logical backup (SQL + CSV)"
echo "   For production, use pg_dump or Supabase backups"
