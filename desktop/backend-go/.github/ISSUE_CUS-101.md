# CUS-101: Implement OSA Sync API Endpoints in OSA-5 Service

## 📋 Summary

The BusinessOS backend has outbox processor code ready to sync entities to OSA, but the OSA-5 service doesn't have the necessary API endpoints yet.

## 🎯 Background

The backend outbox processor (`internal/sync/outbox_processor.go`) is designed to sync user data, workspaces, apps, projects, and tasks to the OSA-5 service for distributed orchestration. However, the OSA integration client (`internal/integrations/osa/client.go`) is missing the required sync methods because the OSA-5 API endpoints don't exist yet.

## ✅ Required Endpoints

The OSA-5 service needs to implement the following REST API endpoints:

### 1. User Sync
```http
POST /api/sync/user
Content-Type: application/json
Authorization: Bearer <OSA_SHARED_SECRET>

{
  "user_id": "uuid",
  "email": "user@example.com",
  "full_name": "John Doe",
  "timezone": "America/New_York"
}

Response: 200 OK
{
  "success": true,
  "synced_at": "2026-01-19T16:30:00Z"
}
```

### 2. Workspace Sync
```http
POST /api/sync/workspace
Content-Type: application/json
Authorization: Bearer <OSA_SHARED_SECRET>

{
  "workspace_id": "uuid",
  "name": "My Workspace",
  "owner_id": "uuid",
  "settings": {...}
}

Response: 200 OK
{
  "success": true,
  "synced_at": "2026-01-19T16:30:00Z"
}
```

### 3. App Sync
```http
POST /api/sync/app
Content-Type: application/json
Authorization: Bearer <OSA_SHARED_SECRET>

{
  "app_id": "uuid",
  "workspace_id": "uuid",
  "name": "Generated App",
  "type": "full-stack",
  "status": "deployed"
}

Response: 200 OK
{
  "success": true,
  "synced_at": "2026-01-19T16:30:00Z"
}
```

### 4. Project Sync
```http
POST /api/sync/project
Content-Type: application/json
Authorization: Bearer <OSA_SHARED_SECRET>

{
  "project_id": "uuid",
  "workspace_id": "uuid",
  "name": "Project Name",
  "description": "Project description"
}

Response: 200 OK
{
  "success": true,
  "synced_at": "2026-01-19T16:30:00Z"
}
```

### 5. Task Sync
```http
POST /api/sync/task
Content-Type: application/json
Authorization: Bearer <OSA_SHARED_SECRET>

{
  "task_id": "uuid",
  "project_id": "uuid",
  "title": "Task title",
  "status": "in_progress",
  "assignee_id": "uuid"
}

Response: 200 OK
{
  "success": true,
  "synced_at": "2026-01-19T16:30:00Z"
}
```

## 🔒 Authentication

All sync endpoints must use shared secret authentication:
- Header: `Authorization: Bearer <OSA_SHARED_SECRET>`
- Secret configured via environment variable `OSA_SHARED_SECRET`
- Validate signature using HMAC-SHA256

## 📦 Implementation Notes

### OSA-5 Service (Python/Flask or Go/Gin)

1. **Add routes:**
```python
# Python/Flask example
@app.route('/api/sync/user', methods=['POST'])
@require_auth
def sync_user():
    data = request.get_json()
    # Validate payload
    # Store in OSA database
    # Return success response
    return jsonify({"success": True, "synced_at": datetime.utcnow()})
```

2. **Store sync data** in OSA database (PostgreSQL/MongoDB)

3. **Return sync confirmation** with timestamp

4. **Error handling:**
   - 400 Bad Request - Invalid payload
   - 401 Unauthorized - Missing/invalid auth
   - 500 Internal Server Error - Sync failed

### BusinessOS Backend Client Update

Once OSA-5 endpoints are ready, update `internal/integrations/osa/client.go`:

```go
func (c *Client) SyncUser(ctx context.Context, req *UserSyncRequest) (*SyncResponse, error) {
    url := fmt.Sprintf("%s/api/sync/user", c.baseURL)

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.sharedSecret))

    resp, err := c.httpClient.Do(httpReq)
    // ... handle response
}
```

Then uncomment the outbox processor code in `internal/sync/outbox_processor.go`:
- Lines 334-355 (User sync)
- Lines 367-383 (Workspace sync)
- Lines 385-400 (App sync)
- Lines 402-417 (Project sync)
- Lines 419-434 (Task sync)

## 🚫 Blocks

This issue blocks the following TODOs in the BusinessOS backend:

1. `internal/sync/outbox_processor.go:334` - User sync implementation
2. `internal/sync/outbox_processor.go:367` - Workspace sync implementation
3. `internal/sync/outbox_processor.go:385` - App sync implementation
4. `internal/sync/outbox_processor.go:402` - Project sync implementation
5. `internal/sync/outbox_processor.go:419` - Task sync implementation
6. `internal/handlers/chat_v2.go:299` - OSA routing enablement
7. `internal/services/osa_sync_service_stub.go:23` - OSA sync service completion
8. `internal/integrations/osa/resilient_client.go:312` - Request processing

## 📊 Estimated Effort

**OSA-5 Team:**
- API endpoint implementation: 12 hours
- Authentication setup: 2 hours
- Database schema: 3 hours
- Testing: 3 hours
- **Total: 20 hours**

**BusinessOS Team (after OSA-5 complete):**
- Client method implementation: 4 hours
- Outbox processor uncomment/test: 2 hours
- Integration testing: 2 hours
- **Total: 8 hours**

## 🎯 Success Criteria

- [ ] All 5 sync endpoints implemented in OSA-5
- [ ] Authentication working with shared secret
- [ ] Data persisted in OSA database
- [ ] Error handling implemented
- [ ] BusinessOS client methods implemented
- [ ] Outbox processor code uncommented and working
- [ ] Integration tests passing
- [ ] Documentation updated

## 🔗 Related

- **Blocks:** CUS-94 (TODO resolution)
- **Related:** OSA-5 roadmap
- **Reference:** `desktop/backend-go/CUS-94_COMPLETION_SUMMARY.md`

## 📅 Timeline

**Priority:** High
**Estimated Start:** TBD (OSA team availability)
**Estimated Completion:** TBD

---

**Created:** 2026-01-19
**Assignee:** OSA Team
**Labels:** backend, osa-5, api, sync, blocked
