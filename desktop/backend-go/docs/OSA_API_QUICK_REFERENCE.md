# OSA API Quick Reference

**Base URL**: `http://localhost:8001`

## Authentication

All endpoints require JWT Bearer token:
```bash
Authorization: Bearer <your_token>
```

Get token via:
```bash
curl -X POST http://localhost:8001/api/auth/sign-in/email \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' 
```

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/osa/workflows | List all workflows |
| GET | /api/osa/workflows/:id | Get workflow details |
| GET | /api/osa/workflows/:id/files | List workflow files |
| GET | /api/osa/workflows/:id/files/:type | Get file content by type |
| GET | /api/osa/files/:id/content | Get file content by ID |
| POST | /api/osa/modules/install | Install workflow as module |
| POST | /api/osa/sync/trigger | Trigger manual sync |
| GET | /api/osa/webhooks | List webhooks |
| POST | /api/osa/webhooks/register | Register new webhook |

## Quick Examples

### List workflows
```bash
curl -X GET http://localhost:8001/api/osa/workflows \
  -H "Authorization: Bearer <token>"
```

### Get workflow details
```bash
curl -X GET http://localhost:8001/api/osa/workflows/<workflow_id> \
  -H "Authorization: Bearer <token>"
```

### Get file content
```bash
curl -X GET http://localhost:8001/api/osa/workflows/<id>/files/analysis \
  -H "Authorization: Bearer <token>"
```

### Install module
```bash
curl -X POST http://localhost:8001/api/osa/modules/install \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"workflow_id":"<id>","module_name":"my_module"}'
```

## Response Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Server Error

## File Types

Valid file types for `/workflows/:id/files/:type`:
- `analysis` - Project analysis
- `architecture` - Architecture docs
- `code` - Source code (Go)
- `quality` - Quality reports
- `deployment` - Deployment guides
- `monitoring` - Monitoring setup
- `strategy` - Product strategy
- `recommendations` - Implementation recommendations

## Status Values

Workflow statuses:
- `generating` - Creating app
- `generated` - Code ready
- `deploying` - Deployment in progress
- `deployed` - Live
- `running` - Active
- `stopped` - Inactive
- `failed` - Error occurred

## Test Data Setup

```sql
-- Create test workspace
INSERT INTO osa_workspaces (user_id, name, mode, template_type)
VALUES ('<user_id>', 'Test Workspace', '2d', 'business_os');

-- Create test app
INSERT INTO osa_generated_apps (
  workspace_id, name, display_name, description,
  osa_workflow_id, status, files_created, build_status,
  metadata
)
VALUES (
  '<workspace_id>', 'todo_app', 'To-Do App', 'Task manager',
  'wf_test123', 'deployed', 12, 'success',
  '{"analysis":"# Test Analysis","code":"package main"}'::jsonb
);
```

See [OSA_API_TESTS.md](./OSA_API_TESTS.md) for detailed documentation.
