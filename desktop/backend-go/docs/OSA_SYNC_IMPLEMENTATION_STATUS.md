# OSA Sync + Webhooks Implementation Status

## Phase 1: OSA SYNC SERVICE - COMPLETED

### 1. OSA Client Sync Methods ✅
**File:** `internal/integrations/osa/client.go`

Implemented 5 new sync methods:
- `SyncUser(ctx, userID)` - Synchronizes user data with OSA-5
- `SyncWorkspace(ctx, workspaceID, userID)` - Synchronizes workspace data
- `SyncApp(ctx, appID, workspaceID, userID)` - Synchronizes app data
- `SyncProject(ctx, projectID, workspaceID, userID)` - Synchronizes project data
- `SyncTask(ctx, taskID, projectID, workspaceID, userID)` - Synchronizes task data

**Pattern:** All methods use the existing `makeRequest()` helper with HMAC authentication.

### 2. OSA Sync Service ✅
**File:** `internal/services/osa_sync_service.go` (renamed from `*_stub.go`)

**Implemented:**
- Full service with OSA client initialization
- 5 public sync methods (SyncUser, SyncWorkspace, SyncApp, SyncProject, SyncTask)
- ProcessOutbox() method for batch processing
- 5 private process*Event() methods for payload handling
- Proper error handling with context wrapping
- Structured logging with slog

**Key Features:**
- Proper context propagation
- Error wrapping with detailed messages
- OSA client lifecycle management (initialization + Close())

### 3. Outbox Processor Methods ✅
**File:** `internal/sync/outbox_processor.go`

**Implemented:**
- Added `OSASyncServiceInterface` for dependency injection
- Updated constructor to accept `osaSyncService` parameter
- Implemented 5 TODO methods:
  - `processUserEvent()` - Calls osaSyncService.SyncUser()
  - `processWorkspaceEvent()` - Calls osaSyncService.SyncWorkspace()
  - `processAppEvent()` - Calls osaSyncService.SyncApp()
  - `processProjectEvent()` - Calls osaSyncService.SyncProject()
  - `processTaskEvent()` - Calls osaSyncService.SyncTask()

**Improvements:**
- Updated payload structs to include required IDs (UserID, WorkspaceID)
- Removed TODOs and stubbed code
- Proper error propagation

---

## Phase 2: WEBHOOK SECURITY - COMPLETED

### 4. Webhook Timestamp Validation ✅
**File:** `internal/handlers/osa_webhooks.go`

**Implemented:**
- `verifyTimestamp(timestamp)` method - validates ±5 minute window
- Added timestamp checks to both webhook handlers:
  - `HandleWorkflowComplete()`
  - `HandleBuildEvent()`

**Security Improvements:**
- Prevents replay attacks
- Returns 401 Unauthorized for invalid timestamps
- Configurable time window (currently 5 minutes)

**Dev Mode Change:**
- Changed signature bypass from `return true` to `return false`
- Added TODO comment to remove bypass in production

### 5. Linear Webhook Signature Verification ✅
**File:** `internal/webhooks/handler.go`

**Implemented:**
- `verifyLinearSignature(body, signature)` method
- HMAC-SHA256 signature verification
- Constant-time comparison to prevent timing attacks
- Integrated into `LinearWebhook()` handler

**Security Features:**
- Uses `LinearWebhookSecret` from config
- Skips verification only in development (logs warning)
- Returns 401 for invalid signatures

---

## Phase 3: CONFLICT DETECTION - TODO

### 6. Implement Conflict Detector
**File:** `internal/sync/conflicts/detector.go` (currently stub)

**To Implement:**
- Choose resolution strategy (recommendation: last-write-wins for v1)
- `DetectConflict()` - compares timestamps, detects concurrent modifications
- `ResolveConflict()` - applies resolution strategy
- `ListConflicts()` - retrieves unresolved conflicts

**Database:**
- Table `sync_conflicts` already migrated (migration 043)
- Fields: entity_type, entity_id, local/remote data, resolution_strategy

### 7. Integrate Conflict Detection
**Locations:**
- Outbox processor: check for conflicts before sync
- Store conflicts in database if detected
- Retry with resolution

---

## Phase 4: NATS MESSAGING - TODO (Optional)

### 8. Implement NATS Client
**File:** `internal/sync/messaging/nats.go`

**To Implement:**
- `Connect()` - establishes NATS connection
- `Publish()` - publishes sync events
- `Subscribe()` - listens for events
- `Close()` - cleanup

**Decision Needed:**
- Q1-Q3: Regional vs global NATS deployment
- Recommendation: Single global NATS for v1

### 9. Switch from Polling to Event-Driven
- Replace outbox polling with NATS publish on insert
- Subscribe to NATS for processing
- Keep polling as fallback

---

## Database Schema Status ✅

### Tables Created (migrations 042-043)
- `sync_outbox` - Transactional outbox pattern
- `sync_dlq` - Dead letter queue
- `sync_conflicts` - Conflict tracking
- `osa_webhooks` - Webhook configurations
- `osa_build_events` - Build event tracking

### SQLC Queries Available ✅
**File:** `internal/database/queries/sync_outbox.sql`

All queries implemented:
- CreateOutboxEvent, GetPendingOutboxEvents
- MarkOutboxEventProcessing, MarkOutboxEventCompleted, MarkOutboxEventFailed
- GetOutboxEventStats, ListFailedOutboxEvents, ListDLQReadyEvents
- MoveEventToDLQ, DeleteOutboxEvent
- CleanupOldCompletedEvents, ResetStuckProcessingEvents
- ListDLQEvents, ResolveDLQEvent, RetryDLQEvent

---

## Testing Requirements

### Unit Tests TODO
- [ ] OSA client sync methods
- [ ] OSA sync service methods
- [ ] Outbox processor with mock service
- [ ] Webhook signature verification
- [ ] Timestamp validation

### Integration Tests TODO
- [ ] End-to-end outbox processing
- [ ] Webhook delivery simulation
- [ ] Conflict detection scenarios
- [ ] NATS pub/sub (when implemented)

---

## Configuration Required

### Environment Variables
```bash
# OSA Integration
OSA_BASE_URL=https://osa-api.example.com
OSA_SHARED_SECRET=your-secret-key

# Webhook Security
OSA_WEBHOOK_SECRET=webhook-signing-secret
LINEAR_WEBHOOK_SECRET=linear-signing-secret

# Outbox Processor
OUTBOX_WORKERS=4
OUTBOX_POLL_INTERVAL=5s

# NATS (optional)
NATS_URL=nats://localhost:4222
```

---

## Next Steps (Priority Order)

1. **Conflict Detection (Phase 3)** - Critical for data consistency
   - Implement last-write-wins strategy
   - Add conflict resolution UI

2. **Testing** - Critical for reliability
   - Unit tests for all new methods
   - Integration tests for sync flow

3. **Monitoring** - Important for production
   - Add metrics for sync success/failure rates
   - Dashboard for outbox queue status

4. **NATS Messaging (Phase 4)** - Nice to have
   - Improves scalability
   - Reduces database polling load

---

## Code Quality Checklist ✅

- [x] No `panic` in production code
- [x] Context propagation everywhere
- [x] Structured logging with `slog`
- [x] Error wrapping with context
- [x] Proper error handling (no ignored errors)
- [x] Code compiles without errors
- [x] Follows Handler → Service → Repository pattern
- [x] Interface for testability (OSASyncServiceInterface)

---

## Files Modified

### Created/Renamed:
- `internal/services/osa_sync_service.go` (renamed from *_stub.go)

### Modified:
- `internal/integrations/osa/client.go` - Added 5 sync methods
- `internal/sync/outbox_processor.go` - Implemented 5 TODO methods, added interface
- `internal/handlers/osa_webhooks.go` - Added timestamp validation
- `internal/webhooks/handler.go` - Added Linear signature verification

### To Create:
- `internal/sync/conflicts/detector.go` - Conflict resolution logic
- Unit test files for all new functionality

---

**Version:** 1.0.0
**Last Updated:** 2026-01-18
**Status:** Phase 1 & 2 Complete | Phase 3 & 4 Pending
