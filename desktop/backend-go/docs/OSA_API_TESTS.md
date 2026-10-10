# OSA API Endpoint Testing Documentation

**Last Updated**: 2026-01-09
**Purpose**: Comprehensive testing documentation for OSA-5 integration endpoints

## Table of Contents
1. [Authentication](#authentication)
2. [Endpoint Testing](#endpoint-testing)
3. [Error Responses](#error-responses)
4. [Rate Limiting](#rate-limiting)
5. [Sample Test Data](#sample-test-data)

---

## Authentication

All OSA API endpoints require JWT authentication via session token.

### Authentication Headers

```bash
# Required header for all authenticated endpoints
Authorization: Bearer <session_token>
```

### Getting a Session Token

**Option 1: Email/Password Login**
```bash
curl -X POST http://localhost:8001/api/auth/sign-in/email \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your_password"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "email": "user@example.com",
    "full_name": "John Doe"
  }
}
```

**Option 2: Google OAuth**
```bash
# Navigate to:
http://localhost:8001/api/auth/google

# After callback, retrieve session via:
curl -X GET http://localhost:8001/api/auth/session \
  -H "Cookie: session=<cookie_value>"
```

### Testing Without Authentication (401 Expected)

```bash
curl -X GET http://localhost:8001/api/osa/workflows
# Response: {"error": "User not authenticated"}
```

---

## Endpoint Testing

### 1. GET /api/osa/workflows

**Description**: Retrieves all OSA workflows for the authenticated user

**Authentication**: Required

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/workflows \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "workflows": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "todo_app",
      "display_name": "To-Do Application",
      "description": "A full-stack task management application",
      "workflow_id": "wf_abc123xyz789",
      "status": "deployed",
      "files_created": 12,
      "build_status": "success",
      "created_at": "2026-01-09T10:30:00Z",
      "generated_at": "2026-01-09T10:35:00Z",
      "deployed_at": "2026-01-09T10:40:00Z",
      "workspace_name": "My Workspace"
    }
  ],
  "count": 1
}
```

**Expected Statuses:**
- `generating` - OSA is creating the app
- `generated` - App code generated
- `deploying` - Deployment in progress
- `deployed` - Live and running
- `failed` - Generation or deployment failed
- `running` - App is active
- `stopped` - App is inactive

**Curl Example:**
```bash
curl -X GET http://localhost:8001/api/osa/workflows \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Sample Response (Empty):**
```json
{
  "workflows": [],
  "count": 0
}
```

**Status Codes:**
- `200 OK` - Success
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Database error

---

### 2. GET /api/osa/workflows/:id

**Description**: Retrieves detailed information for a specific workflow

**Authentication**: Required

**URL Parameters:**
- `id` (required) - UUID or workflow ID prefix (e.g., `wf_abc123`)

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/workflows/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <your_token>"
```

**Alternative (using workflow ID prefix):**
```bash
curl -X GET http://localhost:8001/api/osa/workflows/wf_abc123 \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "todo_app",
  "display_name": "To-Do Application",
  "description": "A full-stack task management application with user authentication",
  "workflow_id": "wf_abc123xyz789",
  "status": "deployed",
  "files_created": 12,
  "build_status": "success",
  "metadata": {
    "analysis": "# Project Analysis\n\nThis application provides...",
    "architecture": "# Architecture Design\n\n## Database Schema\n...",
    "code": "package main\n\nimport (\n\t\"fmt\"\n)\n...",
    "quality": "# Quality Metrics\n\n- Code Coverage: 85%\n...",
    "deployment": "# Deployment Guide\n\n## Prerequisites\n...",
    "monitoring": "# Monitoring Setup\n\n## Metrics\n...",
    "strategy": "# Product Strategy\n\n## Market Analysis\n...",
    "recommendations": "# Recommendations\n\n## Next Steps\n..."
  },
  "error_message": null,
  "error_stack": null,
  "created_at": "2026-01-09T10:30:00Z",
  "generated_at": "2026-01-09T10:35:00Z",
  "deployed_at": "2026-01-09T10:40:00Z",
  "workspace_name": "My Workspace",
  "workspace_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

**Metadata Keys:**
- `analysis` - Project analysis markdown
- `architecture` - System architecture documentation
- `code` - Generated code files
- `quality` - Quality assurance reports
- `deployment` - Deployment instructions
- `monitoring` - Monitoring configuration
- `strategy` - Product strategy documentation
- `recommendations` - Implementation recommendations

**Status Codes:**
- `200 OK` - Workflow found
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Workflow doesn't exist or user doesn't have access
- `500 Internal Server Error` - Database error

---

### 3. GET /api/osa/workflows/:id/files

**Description**: Retrieves all files associated with a workflow

**Authentication**: Required

**URL Parameters:**
- `id` (required) - Workflow UUID or workflow ID prefix

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/workflows/550e8400-e29b-41d4-a716-446655440000/files \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "workflow_id": "wf_abc123xyz789",
  "files": [
    {
      "id": "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
      "name": "analysis.md",
      "type": "analysis",
      "size": 2048,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f2e3d4c5-b6a7-8901-bcde-f12345678901",
      "name": "architecture.md",
      "type": "architecture",
      "size": 4096,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f3e4d5c6-b7a8-9012-cdef-123456789012",
      "name": "code.go",
      "type": "code",
      "size": 8192,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f4e5d6c7-b8a9-0123-def1-234567890123",
      "name": "quality.md",
      "type": "quality",
      "size": 1536,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f5e6d7c8-b9a0-1234-ef12-345678901234",
      "name": "deployment.md",
      "type": "deployment",
      "size": 2560,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f6e7d8c9-b0a1-2345-f123-456789012345",
      "name": "monitoring.md",
      "type": "monitoring",
      "size": 1792,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f7e8d9c0-b1a2-3456-1234-567890123456",
      "name": "strategy.md",
      "type": "strategy",
      "size": 3072,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    },
    {
      "id": "f8e9d0c1-b2a3-4567-2345-678901234567",
      "name": "recommendations.md",
      "type": "recommendations",
      "size": 2304,
      "created_at": "2026-01-09T10:35:00Z",
      "updated_at": "2026-01-09T10:35:00Z"
    }
  ],
  "count": 8
}
```

**File Types:**
- `analysis` - Project analysis (markdown)
- `architecture` - Architecture documentation (markdown)
- `code` - Source code (Go)
- `quality` - Quality reports (markdown)
- `deployment` - Deployment guides (markdown)
- `monitoring` - Monitoring setup (markdown)
- `strategy` - Product strategy (markdown)
- `recommendations` - Implementation recommendations (markdown)

**File ID Generation:**
File IDs are deterministically generated using SHA-1 hash of `workflow_id:file_type`, ensuring consistent IDs across requests.

**Status Codes:**
- `200 OK` - Files retrieved successfully
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Workflow not found
- `500 Internal Server Error` - Database or parsing error

---

### 4. GET /api/osa/workflows/:id/files/:type

**Description**: Retrieves the content of a specific file by type

**Authentication**: Required

**URL Parameters:**
- `id` (required) - Workflow UUID or workflow ID prefix
- `type` (required) - File type (analysis, architecture, code, quality, deployment, monitoring, strategy, recommendations)

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/workflows/550e8400-e29b-41d4-a716-446655440000/files/analysis \
  -H "Authorization: Bearer <your_token>"
```

**Valid File Types:**
```bash
# All valid file types
/api/osa/workflows/:id/files/analysis
/api/osa/workflows/:id/files/architecture
/api/osa/workflows/:id/files/code
/api/osa/workflows/:id/files/quality
/api/osa/workflows/:id/files/deployment
/api/osa/workflows/:id/files/monitoring
/api/osa/workflows/:id/files/strategy
/api/osa/workflows/:id/files/recommendations
```

**Response Format:**
```json
{
  "type": "analysis",
  "content": "# Project Analysis\n\n## Overview\n\nThis application is a full-stack task management system designed to help users organize their daily tasks...\n\n## Key Features\n\n1. User Authentication\n2. Task CRUD Operations\n3. Real-time Updates\n4. Mobile Responsive Design\n\n## Technical Requirements\n\n- Go 1.21+\n- PostgreSQL 15+\n- Redis for caching\n- React 18 for frontend\n\n## Implementation Notes\n\n...",
  "size": 2048
}
```

**Example Requests:**
```bash
# Get architecture documentation
curl -X GET http://localhost:8001/api/osa/workflows/wf_abc123/files/architecture \
  -H "Authorization: Bearer <your_token>"

# Get generated code
curl -X GET http://localhost:8001/api/osa/workflows/wf_abc123/files/code \
  -H "Authorization: Bearer <your_token>"

# Get deployment guide
curl -X GET http://localhost:8001/api/osa/workflows/wf_abc123/files/deployment \
  -H "Authorization: Bearer <your_token>"
```

**Status Codes:**
- `200 OK` - File content retrieved
- `400 Bad Request` - Invalid file type
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Workflow or file not found
- `500 Internal Server Error` - Database or parsing error

---

### 5. GET /api/osa/files/:id/content

**Description**: Retrieves file content by file ID (returned from /workflows/:id/files)

**Authentication**: Required

**URL Parameters:**
- `id` (required) - File UUID

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/files/f1e2d3c4-b5a6-7890-abcd-ef1234567890/content \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "content": "# Project Analysis\n\n## Overview\n\nThis application is a full-stack task management system...",
  "file": {
    "id": "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
    "name": "analysis.md",
    "type": "analysis",
    "size": 2048,
    "created_at": "2026-01-09T10:35:00Z",
    "updated_at": "2026-01-09T10:35:00Z"
  }
}
```

**Workflow:**
1. First, call `/api/osa/workflows/:id/files` to get list of file IDs
2. Then call `/api/osa/files/:id/content` with specific file ID to get content

**Example Workflow:**
```bash
# Step 1: Get file list
WORKFLOW_ID="550e8400-e29b-41d4-a716-446655440000"
FILES=$(curl -s -X GET http://localhost:8001/api/osa/workflows/$WORKFLOW_ID/files \
  -H "Authorization: Bearer <your_token>")

# Step 2: Extract file ID (using jq)
FILE_ID=$(echo $FILES | jq -r '.files[0].id')

# Step 3: Get file content
curl -X GET http://localhost:8001/api/osa/files/$FILE_ID/content \
  -H "Authorization: Bearer <your_token>"
```

**Status Codes:**
- `200 OK` - File found
- `400 Bad Request` - Invalid file ID format
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - File doesn't exist
- `500 Internal Server Error` - Database error

---

### 6. POST /api/osa/modules/install

**Description**: Installs a workflow as a BusinessOS module

**Authentication**: Required

**Request Body:**
```json
{
  "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
  "module_name": "todo_module",
  "install_path": "/modules/todo",
  "file_ids": [
    "f1e2d3c4-b5a6-7890-abcd-ef1234567890",
    "f3e4d5c6-b7a8-9012-cdef-123456789012"
  ]
}
```

**Field Descriptions:**
- `workflow_id` (required) - Workflow UUID or workflow ID prefix
- `module_name` (optional) - Custom module name (defaults to workflow name)
- `install_path` (optional) - Installation path in BusinessOS
- `file_ids` (optional) - Specific file IDs to install (empty array installs all)

**Request:**
```bash
curl -X POST http://localhost:8001/api/osa/modules/install \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "550e8400-e29b-41d4-a716-446655440000",
    "module_name": "todo_module"
  }'
```

**Response Format (Success):**
```json
{
  "success": true,
  "module_id": "770e8400-e29b-41d4-a716-446655440099",
  "message": "Module installed successfully"
}
```

**Response Format (Partial Success):**
```json
{
  "success": true,
  "module_id": "770e8400-e29b-41d4-a716-446655440099",
  "message": "Module installed successfully (app update failed)"
}
```

**Installation Process:**
1. Validates workflow exists and belongs to user
2. Creates entry in `osa_modules` table
3. Links module to workflow in `osa_generated_apps`
4. Updates workflow status to `deployed`
5. Sets `deployed_at` timestamp

**What Gets Installed:**
- Schema definition (from architecture file)
- API definition (from code file)
- UI definition (from recommendations file)
- Complete metadata for module configuration

**Status Codes:**
- `200 OK` - Module installed (check `success` field)
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Missing or invalid token
- `404 Not Found` - Workflow not found
- `500 Internal Server Error` - Installation error

---

### 7. POST /api/osa/sync/trigger

**Description**: Manually triggers synchronization from OSA-5 workspace

**Authentication**: Required

**Request:**
```bash
curl -X POST http://localhost:8001/api/osa/sync/trigger \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "message": "Sync triggered",
  "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Background Process:**
This endpoint triggers an asynchronous sync operation that:
1. Scans OSA-5 workspace for new workflows
2. Polls for workflow status updates
3. Retrieves generated files
4. Updates database records
5. Triggers webhooks if configured

**Sync Frequency:**
- Manual: Via this endpoint
- Automatic: Every 30 seconds (configurable in OSAFileSyncService)

**Status Codes:**
- `200 OK` - Sync triggered
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Sync service error

---

### 8. GET /api/osa/webhooks

**Description**: Lists all webhooks configured for the user

**Authentication**: Required

**Request:**
```bash
curl -X GET http://localhost:8001/api/osa/webhooks \
  -H "Authorization: Bearer <your_token>"
```

**Response Format:**
```json
{
  "webhooks": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440100",
      "event_type": "app.generated",
      "webhook_url": "https://example.com/webhooks/osa",
      "enabled": true,
      "last_triggered_at": "2026-01-09T10:45:00Z",
      "success_count": 42,
      "failure_count": 2,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "count": 1
}
```

**Event Types:**
- `app.generated` - Workflow generation completed
- `app.deployed` - App successfully deployed
- `build.completed` - Build process finished
- `build.failed` - Build process failed

**Status Codes:**
- `200 OK` - Webhooks retrieved
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Database error

---

### 9. POST /api/osa/webhooks/register

**Description**: Registers a new webhook for OSA events

**Authentication**: Required

**Request Body:**
```json
{
  "workspace_id": "660e8400-e29b-41d4-a716-446655440001",
  "app_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "app.generated",
  "webhook_url": "https://example.com/webhooks/osa"
}
```

**Field Descriptions:**
- `workspace_id` (optional) - Specific workspace to monitor
- `app_id` (optional) - Specific app to monitor
- `event_type` (required) - Event to listen for
- `webhook_url` (required) - URL to receive webhook POST requests

**Request:**
```bash
curl -X POST http://localhost:8001/api/osa/webhooks/register \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "app.generated",
    "webhook_url": "https://example.com/webhooks/osa"
  }'
```

**Response Format:**
```json
{
  "id": "880e8400-e29b-41d4-a716-446655440100",
  "secret_key": "whs_abc123xyz789def456ghi",
  "message": "Webhook registered successfully",
  "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Webhook Security:**
The `secret_key` is used to verify webhook signatures using HMAC-SHA256:

```python
import hmac
import hashlib

def verify_webhook(body, signature, secret):
    expected = hmac.new(
        secret.encode(),
        body.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

**Webhook Payload Example:**
```json
{
  "event_type": "app.generated",
  "workflow_id": "wf_abc123xyz789",
  "timestamp": "2026-01-09T10:35:00Z",
  "status": "success",
  "data": {
    "files_created": 12,
    "build_status": "success"
  }
}
```

**Status Codes:**
- `201 Created` - Webhook registered
- `400 Bad Request` - Invalid request body
- `401 Unauthorized` - Missing or invalid token
- `500 Internal Server Error` - Registration error

---

## Error Responses

### Standard Error Format

All errors return consistent JSON structure:

```json
{
  "error": "Human-readable error message",
  "details": "Technical details (only in development)"
}
```

### Common Error Scenarios

**401 Unauthorized**
```json
{
  "error": "User not authenticated"
}
```

**404 Not Found**
```json
{
  "error": "Workflow not found"
}
```

**400 Bad Request**
```json
{
  "error": "Invalid request body"
}
```

**500 Internal Server Error**
```json
{
  "error": "Failed to fetch workflows",
  "details": "pq: connection refused"
}
```

**400 Invalid File Type**
```json
{
  "error": "Invalid file type"
}
```

**404 File Not Found**
```json
{
  "error": "File not found"
}
```

---

## Rate Limiting

### Current Implementation

**Status**: Not currently enforced for OSA endpoints

### Planned Limits

- **Authenticated endpoints**: 100 requests/minute per user
- **File content endpoints**: 50 requests/minute per user
- **Module installation**: 10 requests/minute per user
- **Webhook registration**: 20 requests/hour per user

### Rate Limit Headers (Planned)

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1736424000
```

### Rate Limit Exceeded Response

```json
{
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

---

## Sample Test Data

### Creating Test Workflow (Manual)

Since the OSA-5 integration is external, you can manually insert test data:

```sql
-- 1. Create workspace
INSERT INTO osa_workspaces (id, user_id, name, mode, template_type)
VALUES (
  '660e8400-e29b-41d4-a716-446655440001',
  '<your_user_id>',
  'Test Workspace',
  '2d',
  'business_os'
);

-- 2. Create test app
INSERT INTO osa_generated_apps (
  id,
  workspace_id,
  name,
  display_name,
  description,
  osa_workflow_id,
  status,
  files_created,
  build_status,
  metadata,
  created_at,
  generated_at,
  deployed_at
)
VALUES (
  '550e8400-e29b-41d4-a716-446655440000',
  '660e8400-e29b-41d4-a716-446655440001',
  'todo_app',
  'To-Do Application',
  'A full-stack task management application',
  'wf_abc123xyz789',
  'deployed',
  12,
  'success',
  '{
    "analysis": "# Project Analysis\n\nThis is a test application.",
    "architecture": "# Architecture\n\n## Database Schema\n\nTables: tasks, users",
    "code": "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
    "quality": "# Quality Report\n\nTests: Passing",
    "deployment": "# Deployment\n\nRun: go run main.go",
    "monitoring": "# Monitoring\n\nMetrics: Enabled",
    "strategy": "# Strategy\n\nTarget: Productivity apps",
    "recommendations": "# Recommendations\n\nNext: Add mobile app"
  }'::jsonb,
  NOW() - INTERVAL '1 hour',
  NOW() - INTERVAL '55 minutes',
  NOW() - INTERVAL '50 minutes'
);
```

### Test Script

```bash
#!/bin/bash

# Configuration
BASE_URL="http://localhost:8001"
TOKEN="<your_session_token>"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "=== OSA API Test Suite ==="
echo ""

# Test 1: List workflows
echo "Test 1: GET /api/osa/workflows"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/osa/workflows" \
  -H "Authorization: Bearer $TOKEN")
status=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$status" = "200" ]; then
  echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
  echo "Response: $body" | jq '.'
else
  echo -e "${RED}✗ FAIL${NC} (HTTP $status)"
  echo "Response: $body"
fi
echo ""

# Test 2: Get workflow by ID
WORKFLOW_ID=$(echo "$body" | jq -r '.workflows[0].id // empty')
if [ ! -z "$WORKFLOW_ID" ]; then
  echo "Test 2: GET /api/osa/workflows/:id"
  response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/osa/workflows/$WORKFLOW_ID" \
    -H "Authorization: Bearer $TOKEN")
  status=$(echo "$response" | tail -n1)
  body=$(echo "$response" | sed '$d')

  if [ "$status" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
    echo "Response: $body" | jq '.'
  else
    echo -e "${RED}✗ FAIL${NC} (HTTP $status)"
    echo "Response: $body"
  fi
  echo ""

  # Test 3: Get workflow files
  echo "Test 3: GET /api/osa/workflows/:id/files"
  response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/osa/workflows/$WORKFLOW_ID/files" \
    -H "Authorization: Bearer $TOKEN")
  status=$(echo "$response" | tail -n1)
  body=$(echo "$response" | sed '$d')

  if [ "$status" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
    echo "Response: $body" | jq '.'
  else
    echo -e "${RED}✗ FAIL${NC} (HTTP $status)"
    echo "Response: $body"
  fi
  echo ""

  # Test 4: Get file content by type
  echo "Test 4: GET /api/osa/workflows/:id/files/analysis"
  response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/osa/workflows/$WORKFLOW_ID/files/analysis" \
    -H "Authorization: Bearer $TOKEN")
  status=$(echo "$response" | tail -n1)
  body=$(echo "$response" | sed '$d')

  if [ "$status" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
    echo "Response: $body" | jq '.'
  else
    echo -e "${RED}✗ FAIL${NC} (HTTP $status)"
    echo "Response: $body"
  fi
  echo ""
fi

# Test 5: Unauthorized access
echo "Test 5: GET /api/osa/workflows (no auth)"
response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/osa/workflows")
status=$(echo "$response" | tail -n1)

if [ "$status" = "401" ]; then
  echo -e "${GREEN}✓ PASS${NC} (HTTP $status - Expected)"
else
  echo -e "${RED}✗ FAIL${NC} (HTTP $status)"
fi
echo ""

echo "=== Test Suite Complete ==="
```

### Running Tests

```bash
# 1. Save test script
cat > test_osa_api.sh << 'EOF'
# ... paste test script above ...
EOF

# 2. Make executable
chmod +x test_osa_api.sh

# 3. Get your token
TOKEN=$(curl -s -X POST http://localhost:8001/api/auth/sign-in/email \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' \
  | jq -r '.token')

# 4. Run tests
./test_osa_api.sh
```

---

## Integration Testing Checklist

- [ ] Authentication works with valid JWT token
- [ ] 401 returned for missing/invalid token
- [ ] GET /api/osa/workflows returns empty array for new users
- [ ] GET /api/osa/workflows returns workflows for users with data
- [ ] GET /api/osa/workflows/:id returns 404 for non-existent workflow
- [ ] GET /api/osa/workflows/:id works with UUID
- [ ] GET /api/osa/workflows/:id works with workflow ID prefix
- [ ] GET /api/osa/workflows/:id/files returns all file types
- [ ] GET /api/osa/workflows/:id/files/:type returns file content
- [ ] GET /api/osa/workflows/:id/files/:type returns 400 for invalid type
- [ ] GET /api/osa/files/:id/content returns correct file content
- [ ] POST /api/osa/modules/install creates module successfully
- [ ] POST /api/osa/modules/install returns 404 for invalid workflow
- [ ] POST /api/osa/sync/trigger returns success
- [ ] GET /api/osa/webhooks lists user's webhooks
- [ ] POST /api/osa/webhooks/register creates webhook with secret

---

## Performance Benchmarks

### Expected Response Times (Local)

- GET /api/osa/workflows: < 50ms
- GET /api/osa/workflows/:id: < 30ms
- GET /api/osa/workflows/:id/files: < 40ms
- GET /api/osa/workflows/:id/files/:type: < 25ms
- GET /api/osa/files/:id/content: < 100ms (searches all workflows)
- POST /api/osa/modules/install: < 150ms
- POST /api/osa/sync/trigger: < 20ms (async)
- GET /api/osa/webhooks: < 50ms
- POST /api/osa/webhooks/register: < 100ms

### Load Testing

```bash
# Using Apache Bench (ab)
ab -n 1000 -c 10 -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows

# Using wrk
wrk -t4 -c100 -d30s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8001/api/osa/workflows
```

---

## Troubleshooting

### Common Issues

**Issue**: "User not authenticated"
- **Cause**: Missing or expired token
- **Solution**: Re-authenticate and get new token

**Issue**: "Workflow not found"
- **Cause**: Invalid UUID or user doesn't have access
- **Solution**: Verify workflow ID and user ownership

**Issue**: "Failed to parse metadata"
- **Cause**: Invalid JSONB in database
- **Solution**: Check database for corrupted metadata

**Issue**: "Invalid file type"
- **Cause**: Requesting unsupported file type
- **Solution**: Use valid types: analysis, architecture, code, quality, deployment, monitoring, strategy, recommendations

### Debug Mode

Enable debug logging:

```bash
export GIN_MODE=debug
go run ./cmd/server
```

### Database Inspection

```sql
-- Check workflows
SELECT id, name, status, files_created, created_at
FROM osa_generated_apps
ORDER BY created_at DESC;

-- Check metadata keys
SELECT name, jsonb_object_keys(metadata) as key
FROM osa_generated_apps;

-- Check file sync status
SELECT * FROM osa_sync_status
WHERE entity_type = 'app';
```

---

## Additional Resources

- **OSA-5 Documentation**: [Internal Documentation]
- **API Changelog**: See CHANGELOG.md
- **Bug Reports**: GitHub Issues
- **Support**: Slack #osa-integration

---

**Document Version**: 1.0
**Last Reviewed**: 2026-01-09
**Next Review**: 2026-02-09
