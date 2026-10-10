# OSA-5 Integration - Developer Setup & Testing Guide

Complete guide for setting up, testing, and troubleshooting the OSA-5 integration with BusinessOS backend.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Setup](#initial-setup)
3. [Configuration](#configuration)
4. [Starting Services](#starting-services)
5. [Testing the Integration](#testing-the-integration)
6. [API Reference](#api-reference)
7. [Troubleshooting](#troubleshooting)
8. [Architecture Overview](#architecture-overview)

---

## Prerequisites

### Required Software

1. **Go 1.21+**
   ```bash
   # Verify installation
   go version
   ```

2. **PostgreSQL 14+** (with pgvector extension)
   ```bash
   # Option 1: Supabase (recommended for development)
   # Sign up at https://supabase.com

   # Option 2: Local PostgreSQL
   brew install postgresql@14
   brew services start postgresql@14
   ```

3. **Redis 7+** (for session caching)
   ```bash
   brew install redis
   brew services start redis

   # Or use Docker
   docker run -d --name redis -p 6379:6379 redis:7-alpine
   ```

4. **OSA-5** (AI Agent Orchestration System)
   ```bash
   # Clone OSA-5 repository
   git clone https://github.com/your-org/OSA-5
   cd OSA-5

   # Follow OSA-5 setup instructions
   # Default location: /Users/ososerious/OSA-5
   ```

5. **Ollama** (for AI embeddings - optional but recommended)
   ```bash
   # Install Ollama
   brew install ollama

   # Start Ollama service
   ollama serve

   # Pull embedding model
   ollama pull nomic-embed-text
   ```

### Environment Variables

Create a `.env` file in the project root (copy from `.env.example`):

```bash
cp .env.example .env
```

**Required OSA Variables:**

```bash
# OSA-5 Integration
OSA_ENABLED=true
OSA_BASE_URL=http://localhost:8089
OSA_SHARED_SECRET=your-shared-secret-here
OSA_WORKSPACE_PATH=/Users/ososerious/OSA-5/miosa-backend/generated

# Database (Supabase or local)
DATABASE_URL=postgres://postgres.PROJECT_ID:[PASSWORD]@aws-0-us-east-1.pooler.supabase.com:6543/postgres?pgbouncer=true

# Redis (for session caching)
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=changeme_insecure_dev_password
REDIS_TLS_ENABLED=false

# Security Keys (CRITICAL: Generate unique keys for production)
SECRET_KEY=$(openssl rand -base64 64)
TOKEN_ENCRYPTION_KEY=$(openssl rand -base64 32)
REDIS_KEY_HMAC_SECRET=$(openssl rand -base64 32)

# Server
SERVER_PORT=8001
ENVIRONMENT=development
```

---

## Initial Setup

### 1. Clone and Navigate to Backend

```bash
cd /Users/ososerious/BusinessOS-1/desktop/backend-go
```

### 2. Install Go Dependencies

```bash
go mod download
go mod verify
```

### 3. Database Setup

#### Run Migrations

```bash
# Run all migrations (including OSA integration)
go run ./cmd/migrate

# Expected output:
# Running migration 042_osa_integration.sql...
# ✓ Created osa_modules table
# ✓ Created osa_workspaces table
# ✓ Created osa_generated_apps table
# ✓ Created osa_execution_history table
# ✓ Created osa_sync_status table
# ✓ Created osa_build_events table
# ✓ Created osa_webhooks table
```

#### Verify Tables

```bash
psql $DATABASE_URL -c "\dt osa_*"
```

Expected tables:
- `osa_modules` - Module registry
- `osa_workspaces` - User workspaces (2D/3D)
- `osa_generated_apps` - Generated applications
- `osa_execution_history` - Terminal command history
- `osa_sync_status` - Sync state tracking
- `osa_build_events` - Build progress events
- `osa_webhooks` - Webhook configurations

### 4. Create Test Workspace

```bash
psql $DATABASE_URL << 'EOF'
-- Create a test workspace for development
INSERT INTO osa_workspaces (user_id, name, template_type, mode)
SELECT id, 'Development Workspace', 'business_os', '2d'
FROM "user"
LIMIT 1
ON CONFLICT DO NOTHING;
EOF
```

---

## Configuration

### OSA-5 Configuration

Ensure OSA-5 is configured to generate files to the workspace path:

```bash
# In OSA-5 config (e.g., config.yaml or .env)
GENERATED_FILES_PATH=/Users/ososerious/OSA-5/miosa-backend/generated
BUSINESSOS_WEBHOOK_URL=http://localhost:8001/api/osa/webhooks/workflow-complete
BUSINESSOS_SHARED_SECRET=your-shared-secret-here
```

### Directory Structure for File Sync

OSA-5 should generate files in this structure:

```
/Users/ososerious/OSA-5/miosa-backend/generated/
├── analysis/
│   ├── analysis_11af0132.md
│   └── analysis_22bf0243.md
├── architecture/
│   ├── architecture_11af0132.md
│   └── architecture_22bf0243.md
├── code/
│   ├── code_11af0132.go
│   └── code_22bf0243.go
├── quality/
│   ├── quality_11af0132.md
│   └── quality_22bf0243.md
├── deployment/
│   ├── deployment_11af0132.md
│   └── deployment_22bf0243.md
├── monitoring/
│   ├── monitoring_11af0132.md
│   └── monitoring_22bf0243.md
├── strategy/
│   ├── strategy_11af0132.md
│   └── strategy_22bf0243.md
└── recommendations/
    ├── recommendations_11af0132.md
    └── recommendations_22bf0243.md
```

**File Naming Convention:**
- Format: `{type}_{workflowID}.{ext}`
- `workflowID` is the first 8 characters of the OSA workflow UUID
- Example: `analysis_11af0132.md` for workflow `11af0132-abcd-1234-5678-90abcdef1234`

---

## Starting Services

### 1. Start Backend Server

```bash
# Development mode with hot reload (if using air)
air

# Or standard go run
go run ./cmd/server

# Expected output:
# Server instance ID: a1b2c3d4
# Database connected successfully
# Redis connected successfully
# ✅ OSA client initialized (base_url=http://localhost:8089)
# ✅ OSA sync service initialized (transactional outbox pattern)
# ✅ OSA file sync service initialized (workspace=/Users/ososerious/OSA-5/miosa-backend/generated)
# ✅ OSA file sync service started (polling every 30s)
# Server starting on port 8001
```

### 2. Verify Service Health

```bash
# Basic health check
curl http://localhost:8001/health

# Detailed health check
curl http://localhost:8001/health/detailed

# OSA-specific health check
curl http://localhost:8001/api/osa/health
```

Expected response:
```json
{
  "status": "healthy",
  "osa": {
    "connected": true,
    "base_url": "http://localhost:8089",
    "file_sync": "running",
    "last_sync": "2026-01-09T10:30:00Z"
  }
}
```

### 3. Start OSA-5 Service

```bash
cd /Users/ososerious/OSA-5

# Follow OSA-5 specific instructions
# Typically:
npm start
# or
python main.py
```

---

## Testing the Integration

### Test 1: Generate a Workflow in OSA-5

1. **Trigger a workflow generation in OSA-5:**
   ```bash
   # Example: Via OSA-5 CLI or UI
   osa generate --type crud --name "Task Manager" --features auth,api,ui
   ```

2. **Monitor file sync logs:**
   ```bash
   # In backend terminal, watch for:
   # New workflow discovered workflow_id=11af0132
   # Workflow synced to database workflow_id=11af0132 files_created=8
   ```

3. **Verify in database:**
   ```bash
   psql $DATABASE_URL << 'EOF'
   SELECT id, name, display_name, status, files_created, created_at
   FROM osa_generated_apps
   ORDER BY created_at DESC
   LIMIT 5;
   EOF
   ```

### Test 2: API Endpoint - List Workflows

```bash
# Get authentication token first
TOKEN=$(curl -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"your-email@example.com","password":"your-password"}' \
  | jq -r '.access_token')

# List all workflows
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows \
  | jq .
```

Expected response:
```json
{
  "workflows": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "task-manager",
      "display_name": "Task Manager",
      "workflow_id": "11af0132",
      "status": "generated",
      "files_created": 8,
      "created_at": "2026-01-09T10:30:00Z"
    }
  ],
  "count": 1
}
```

### Test 3: API Endpoint - Get Workflow Details

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows/11af0132 \
  | jq .
```

### Test 4: API Endpoint - List Workflow Files

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows/11af0132/files \
  | jq .
```

Expected response:
```json
{
  "workflow_id": "11af0132",
  "files": [
    {
      "id": "f1e2d3c4-b5a6-7890-1234-567890abcdef",
      "name": "analysis.md",
      "type": "analysis",
      "size": 2048,
      "created_at": "2026-01-09T10:30:00Z"
    },
    {
      "id": "a2b3c4d5-e6f7-8901-2345-67890abcdef1",
      "name": "architecture.md",
      "type": "architecture",
      "size": 4096,
      "created_at": "2026-01-09T10:30:00Z"
    }
  ],
  "count": 8
}
```

### Test 5: API Endpoint - Get File Content

```bash
# By workflow ID and file type
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows/11af0132/files/analysis \
  | jq .

# By file ID
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/files/f1e2d3c4-b5a6-7890-1234-567890abcdef/content \
  | jq .
```

### Test 6: Frontend UI Testing

1. **Start the frontend:**
   ```bash
   cd /Users/ososerious/BusinessOS-1/frontend
   npm run dev
   ```

2. **Navigate to OSA page:**
   ```
   http://localhost:5173/window/osa
   ```

3. **Verify UI functionality:**
   - Workflows list loads
   - Workflow cards display correctly
   - File viewer opens
   - File content renders (markdown/code)
   - Install module button works

### Test 7: Module Installation

```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "11af0132",
    "module_name": "task-manager",
    "install_path": "/modules/task-manager"
  }' \
  http://localhost:8001/api/osa/modules/install \
  | jq .
```

Expected response:
```json
{
  "success": true,
  "module_id": "b2c3d4e5-f6a7-8901-2345-67890abcdef1",
  "message": "Module installed successfully"
}
```

Verify installation:
```bash
psql $DATABASE_URL << 'EOF'
SELECT id, name, display_name, module_type, status
FROM osa_modules
WHERE status = 'installed'
ORDER BY created_at DESC;
EOF
```

### Test 8: Webhook Testing (Optional)

Send a test webhook from OSA-5:

```bash
# Generate HMAC signature
SECRET="your-shared-secret-here"
PAYLOAD='{"event_type":"workflow.completed","workflow_id":"11af0132","timestamp":"2026-01-09T10:30:00Z","status":"success","data":{"build_status":"success"}}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)

# Send webhook
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-OSA-Signature: $SIGNATURE" \
  -d "$PAYLOAD" \
  http://localhost:8001/api/osa/webhooks/workflow-complete \
  | jq .
```

---

## API Reference

### Authentication

All endpoints require Bearer token authentication:

```bash
Authorization: Bearer YOUR_JWT_TOKEN
```

### Endpoints

#### GET `/api/osa/health`
Public health check for OSA integration.

**Response:**
```json
{
  "status": "healthy",
  "osa": {
    "connected": true,
    "base_url": "http://localhost:8089",
    "file_sync": "running"
  }
}
```

#### GET `/api/osa/workflows`
List all workflows for authenticated user.

**Response:**
```json
{
  "workflows": [...],
  "count": 10
}
```

#### GET `/api/osa/workflows/:id`
Get workflow details by ID or workflow_id prefix.

**Parameters:**
- `id` - UUID or workflow ID prefix (e.g., `11af0132`)

**Response:**
```json
{
  "id": "uuid",
  "name": "task-manager",
  "display_name": "Task Manager",
  "workflow_id": "11af0132",
  "status": "generated",
  "files_created": 8,
  "metadata": {...}
}
```

#### GET `/api/osa/workflows/:id/files`
List all files for a workflow.

**Response:**
```json
{
  "workflow_id": "11af0132",
  "files": [...],
  "count": 8
}
```

#### GET `/api/osa/workflows/:id/files/:type`
Get file content by workflow ID and type.

**Parameters:**
- `type` - One of: `analysis`, `architecture`, `code`, `quality`, `deployment`, `monitoring`, `strategy`, `recommendations`

**Response:**
```json
{
  "type": "analysis",
  "content": "# Analysis...",
  "size": 2048
}
```

#### GET `/api/osa/files/:id/content`
Get file content by file ID.

**Response:**
```json
{
  "content": "...",
  "file": {
    "id": "uuid",
    "name": "analysis.md",
    "type": "analysis",
    "size": 2048
  }
}
```

#### POST `/api/osa/modules/install`
Install a workflow as a BusinessOS module.

**Request:**
```json
{
  "workflow_id": "11af0132",
  "module_name": "task-manager",
  "install_path": "/modules/task-manager"
}
```

**Response:**
```json
{
  "success": true,
  "module_id": "uuid",
  "message": "Module installed successfully"
}
```

#### POST `/api/osa/webhooks/workflow-complete`
Webhook endpoint for OSA-5 workflow completion (public).

**Headers:**
- `X-OSA-Signature` - HMAC-SHA256 signature

**Request:**
```json
{
  "event_type": "workflow.completed",
  "workflow_id": "11af0132",
  "timestamp": "2026-01-09T10:30:00Z",
  "status": "success",
  "data": {...}
}
```

---

## Troubleshooting

### Common Errors

#### 1. "OSA workspace path does not exist"

**Error:**
```
OSA workspace path does not exist path=/Users/ososerious/OSA-5/miosa-backend/generated
```

**Solution:**
```bash
# Create the directory structure
mkdir -p /Users/ososerious/OSA-5/miosa-backend/generated/{analysis,architecture,code,quality,deployment,monitoring,strategy,recommendations}

# Or update .env with correct path
OSA_WORKSPACE_PATH=/path/to/your/osa/workspace
```

#### 2. "Failed to connect to OSA"

**Error:**
```
Failed to create OSA client: connection refused
```

**Solution:**
```bash
# Verify OSA-5 is running
curl http://localhost:8089/health

# Check OSA_BASE_URL in .env
OSA_BASE_URL=http://localhost:8089

# Restart backend after fixing
```

#### 3. "No workspace found"

**Error:**
```
No workspace found - workflow will be processed when workspace is created
```

**Solution:**
```bash
# Create a default workspace
psql $DATABASE_URL << 'EOF'
INSERT INTO osa_workspaces (user_id, name, template_type, mode)
SELECT id, 'Default Workspace', 'business_os', '2d'
FROM "user"
LIMIT 1;
EOF
```

#### 4. "Webhook signature verification failed"

**Error:**
```
Invalid signature
```

**Solution:**
```bash
# Verify shared secret matches in both systems
# BusinessOS .env:
OSA_SHARED_SECRET=your-secret

# OSA-5 config:
BUSINESSOS_SHARED_SECRET=your-secret

# Generate new secret if needed:
openssl rand -base64 32
```

#### 5. "Database migration failed"

**Error:**
```
Migration 042_osa_integration.sql failed
```

**Solution:**
```bash
# Check if tables already exist
psql $DATABASE_URL -c "\dt osa_*"

# Drop and recreate if needed (CAUTION: loses data)
psql $DATABASE_URL << 'EOF'
DROP TABLE IF EXISTS osa_webhooks CASCADE;
DROP TABLE IF EXISTS osa_build_events CASCADE;
DROP TABLE IF EXISTS osa_sync_status CASCADE;
DROP TABLE IF EXISTS osa_execution_history CASCADE;
DROP TABLE IF EXISTS osa_generated_apps CASCADE;
DROP TABLE IF EXISTS osa_modules CASCADE;
DROP TABLE IF EXISTS osa_workspaces CASCADE;
EOF

# Re-run migration
go run ./cmd/migrate
```

### Debugging Tips

#### Enable Debug Logging

```bash
# In .env
LOG_LEVEL=debug

# Or via environment variable
LOG_LEVEL=debug go run ./cmd/server
```

#### Monitor File Sync

```bash
# Watch for file changes
watch -n 1 'ls -lR /Users/ososerious/OSA-5/miosa-backend/generated/'

# Tail backend logs
tail -f backend.log | grep OSA
```

#### Check Database State

```bash
# View recent workflows
psql $DATABASE_URL << 'EOF'
SELECT
  id,
  name,
  osa_workflow_id,
  status,
  files_created,
  created_at
FROM osa_generated_apps
ORDER BY created_at DESC
LIMIT 10;
EOF

# Check sync status
psql $DATABASE_URL << 'EOF'
SELECT
  entity_type,
  entity_id,
  osa_entity_id,
  sync_status,
  last_sync_at
FROM osa_sync_status
ORDER BY last_sync_at DESC;
EOF
```

#### Test OSA Connectivity

```bash
# From backend server
curl http://localhost:8089/health

# Test with credentials
curl -H "Authorization: Bearer $OSA_SHARED_SECRET" \
  http://localhost:8089/api/workflows
```

### Log Locations

- **Backend logs:** `./logs/server.log` or stdout
- **OSA-5 logs:** Check OSA-5 documentation
- **Database logs:** Check PostgreSQL logs
- **Redis logs:** Check Redis logs or `redis-cli monitor`

---

## Architecture Overview

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (SvelteKit)                     │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Workflow     │  │ File         │  │ Module       │         │
│  │ List View    │  │ Viewer       │  │ Installer    │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└─────────────────────┬────────────────────────────────────────────┘
                      │ HTTP/REST
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    BusinessOS Backend (Go)                       │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   API Handlers                            │  │
│  │  • OSAWorkflowsHandler                                    │  │
│  │  • OSAWebhooksHandler                                     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│  ┌──────────────────────────┼────────────────────────────────┐ │
│  │        Services          │                                 │ │
│  │  • OSASyncService        │  • OSAFileSyncService           │ │
│  │    (API sync)            │    (File polling)               │ │
│  └──────────────────────────┼────────────────────────────────┘ │
│                             │                                    │
│  ┌──────────────────────────┼────────────────────────────────┐ │
│  │       Integrations       │                                 │ │
│  │  • ResilientClient ──────┼─────> OSA-5 API                │ │
│  │    (Circuit breaker)     │                                 │ │
│  └──────────────────────────┼────────────────────────────────┘ │
│                             │                                    │
│  ┌──────────────────────────▼────────────────────────────────┐ │
│  │                   Database (PostgreSQL)                    │ │
│  │  • osa_workspaces         • osa_generated_apps            │ │
│  │  • osa_modules            • osa_sync_status               │ │
│  │  • osa_execution_history  • osa_build_events              │ │
│  └────────────────────────────────────────────────────────────┘ │
└────────────┬─────────────────────────────────────────┬──────────┘
             │                                          │
             │ Webhook                                  │ File System
             │ (HTTP POST)                              │ Polling
             ▼                                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                         OSA-5 System                             │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Workflow     │  │ Code         │  │ File         │         │
│  │ Engine       │  │ Generator    │  │ Output       │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                  │
│  Output Directory: /OSA-5/miosa-backend/generated/              │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

#### 1. Workflow Generation Flow

```
1. User triggers workflow in OSA-5
   ↓
2. OSA-5 generates files to workspace directory
   /generated/analysis/analysis_11af0132.md
   /generated/architecture/architecture_11af0132.md
   /generated/code/code_11af0132.go
   ...
   ↓
3. BusinessOS File Sync polls directory (every 30s)
   ↓
4. New workflow detected (workflow_id: 11af0132)
   ↓
5. Files read and stored in database
   - osa_generated_apps (metadata)
   - osa_sync_status (sync state)
   ↓
6. Frontend polls API and displays new workflow
```

#### 2. API Query Flow

```
1. User opens OSA page in frontend
   ↓
2. Frontend calls GET /api/osa/workflows
   ↓
3. Backend queries osa_generated_apps table
   ↓
4. Returns workflow list with metadata
   ↓
5. User clicks workflow to view files
   ↓
6. Frontend calls GET /api/osa/workflows/:id/files
   ↓
7. Backend extracts files from metadata JSONB
   ↓
8. Returns file list
   ↓
9. User clicks file to view content
   ↓
10. Frontend calls GET /api/osa/files/:id/content
    ↓
11. Backend returns file content
    ↓
12. Frontend renders markdown/code
```

#### 3. Module Installation Flow

```
1. User clicks "Install as Module"
   ↓
2. Frontend calls POST /api/osa/modules/install
   ↓
3. Backend creates entry in osa_modules table
   - Links to osa_generated_apps
   - Stores schema, API, UI definitions
   ↓
4. Updates app status to 'deployed'
   ↓
5. Returns module_id
   ↓
6. Frontend shows success notification
```

### Database Schema

#### Core Tables

**osa_workspaces**
- User workspace configurations
- 2D/3D mode settings
- Active module references

**osa_generated_apps**
- Generated application metadata
- Workflow ID reference
- File contents (JSONB)
- Status tracking

**osa_modules**
- Installed module registry
- Schema/API/UI definitions
- Version tracking

**osa_sync_status**
- Bidirectional sync state
- Last sync timestamps
- Error tracking

**osa_build_events**
- Real-time build progress
- Event stream for UI updates

### Security

#### Authentication
- JWT Bearer tokens required for all API endpoints
- Session validation via Redis cache
- User-scoped data access

#### Webhook Security
- HMAC-SHA256 signature verification
- Shared secret between BusinessOS and OSA-5
- Replay attack prevention (timestamp validation)

#### File System Security
- Restricted to configured workspace path
- No arbitrary file system access
- Deterministic file ID generation (UUID v5)

---

## Next Steps

After successful setup and testing:

1. **Production Deployment**
   - Generate strong secrets for all keys
   - Enable TLS for Redis
   - Configure production database (Supabase)
   - Set up monitoring and alerting

2. **Advanced Features**
   - Configure webhooks for real-time updates
   - Set up continuous deployment integration
   - Enable multi-workspace support
   - Implement module versioning

3. **Performance Optimization**
   - Enable Redis caching for workflows
   - Implement pagination for large datasets
   - Optimize database queries with indexes
   - Configure CDN for file delivery

---

## Support

For issues or questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review backend logs: `tail -f backend.log`
3. Check database state with provided SQL queries
4. Verify OSA-5 connectivity and configuration
5. Open an issue in the repository with:
   - Error messages
   - Configuration (sanitized)
   - Steps to reproduce

---

## Changelog

**2026-01-09** - Initial version
- Complete setup instructions
- API reference
- Testing procedures
- Troubleshooting guide
- Architecture documentation
