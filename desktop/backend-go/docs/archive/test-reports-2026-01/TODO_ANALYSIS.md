# TODO Analysis - Backend Go

**Total TODOs Found:** 36
**Date:** 2026-01-15
**Analyzed by:** Backend Expert (Claude)

---

## 📊 Summary

| Category | Count | Action |
|----------|-------|--------|
| **TRIVIAL** (Can resolve now) | 8 | ✅ Resolve immediately |
| **MEDIUM** (Config/Settings) | 7 | 📋 Document & create tickets |
| **COMPLEX** (Features) | 14 | 📋 Create Linear tickets |
| **TEST ONLY** | 4 | ℹ️ Keep for documentation |
| **STUBS** (Intentional) | 3 | ℹ️ Keep as placeholders |

---

## ✅ TRIVIAL TODOs (Resolve Now)

### 1. **OSA Webhooks - Configurable timeout** ✅ SIMPLE
**File:** `internal/handlers/osa_webhooks.go:123`
```go
// TODO: Make this configurable via env var
```
**Resolution:** Add env var `OSA_WEBHOOK_TIMEOUT` with default value

---

### 2. **Webhooks - Signing secret** ✅ SIMPLE
**File:** `internal/webhooks/handler.go:198`
```go
signingSecret := "" // TODO: Get from config
```
**Resolution:** Add config field for webhook signing secret

---

### 3. **Reranker - Context expansion** ✅ SIMPLE
**File:** `internal/services/reranker.go:329`
```go
// TODO: Could expand with task context, workspace context, etc.
```
**Resolution:** Document as future enhancement, not blocking

---

### 4. **Mobile Handlers - User settings** ✅ SIMPLE
**File:** `internal/handlers/mobile_handlers.go:56`
```go
// TODO: Implement user settings lookup
```
**Resolution:** Check if already implemented elsewhere or add basic implementation

---

### 5-8. **Sync Outbox - Node ID configuration** ✅ SIMPLE
**Files:**
- `internal/sync/outbox/writer.go:102`
- `internal/terminal/osa_command.go:56,57,85`

```go
nodeID := "businessos" // TODO: Make this configurable for multi-instance deployments
```
**Resolution:** Add `NODE_ID` env var with default "businessos"

---

## 📋 MEDIUM TODOs (Document & Create Tickets)

### M1. **OSA API - Default workspace lookup** 🎫 MEDIUM
**Files:**
- `internal/handlers/osa_api.go:60`
- `internal/handlers/osa_internal.go:63`

```go
workspaceID = uuid.New() // TODO: Get user's default workspace
```
**Action:** Create ticket for implementing default workspace service
**Priority:** MEDIUM
**Estimate:** 2-3 hours

---

### M2. **Chat V2 - OSA Routing** 🎫 MEDIUM
**File:** `internal/handlers/chat_v2.go:299`
```go
// TODO: Enable OSA routing when orchestration package is implemented
```
**Action:** Create ticket (blocked by orchestration package)
**Priority:** MEDIUM
**Estimate:** 4 hours
**Dependency:** Orchestration package

---

### M3. **Linear Integration - Update Issue** 🎫 MEDIUM
**File:** `internal/integrations/linear/handler.go:247`
```go
// TODO: Implement update issue in provider
```
**Action:** Create ticket for Linear update functionality
**Priority:** LOW
**Estimate:** 3 hours

---

### M4. **Prompt Personalizer - Semantic Search** 🎫 MEDIUM
**File:** `internal/services/prompt_personalizer.go:87`
```go
// TODO: Implement semantic search for relevant memories
```
**Action:** Create ticket for RAG integration
**Priority:** MEDIUM
**Estimate:** 6 hours

---

### M5-M7. **Webhook Handlers - Database Integration** 🎫 MEDIUM
**Files:**
- `internal/webhooks/handler.go:91,193,282,366`

```go
// TODO: Implement actual sync queue
// TODO: Save message to database or trigger notification
// TODO: Update local database
// TODO: Fetch updated contact and sync to local database
```
**Action:** Create epic ticket for webhook persistence layer
**Priority:** MEDIUM
**Estimate:** 2 days

---

## 🎯 COMPLEX TODOs (Create Linear Tickets)

### C1. **OSA Sync Service - Full Implementation** 🎫 HIGH
**File:** `internal/services/osa_sync_service_stub.go:23`
```go
// TODO: Implement sync methods when database schema is ready:
```
**Action:** Create epic ticket for OSA sync implementation
**Priority:** HIGH
**Estimate:** 1-2 weeks
**Status:** Blocked by database schema finalization

---

### C2. **Outbox Processor - OSA Sync Methods** 🎫 HIGH
**Files:**
- `internal/sync/outbox_processor.go:334,367,385,402,419`

```go
// TODO: Call OSA API to sync user
// TODO: Implement workspace sync with OSA
// TODO: Implement app sync with OSA
// TODO: Implement project sync with OSA
// TODO: Implement task sync with OSA
```
**Action:** Create tickets for each sync method
**Priority:** HIGH
**Estimate:** 1 week (all combined)

---

### C3. **Airtable Integration - Persistence** 🎫 LOW
**Files:**
- `internal/integrations/airtable/handler.go:218,514`

```go
// TODO: Store bases in database
// TODO: Store records in database
```
**Action:** Create ticket for Airtable caching layer
**Priority:** LOW
**Estimate:** 4 hours

---

### C4. **ClickUp Integration - Persistence** 🎫 LOW
**Files:**
- `internal/integrations/clickup/handler.go:180,423`

```go
// TODO: Save workspaces to database if needed
// TODO: Save tasks to database if needed
```
**Action:** Create ticket for ClickUp caching layer
**Priority:** LOW
**Estimate:** 4 hours

---

### C5. **Webhook Signature Verification** 🎫 MEDIUM
**File:** `internal/webhooks/handler.go:231`
```go
// TODO: Verify Linear webhook signature
```
**Action:** Create security ticket for webhook verification
**Priority:** MEDIUM (Security)
**Estimate:** 2 hours

---

### C6. **Notion Webhooks** 🎫 LOW
**File:** `internal/webhooks/handler.go:394`
```go
// TODO: Parse and handle Notion webhook events when they become available
```
**Action:** Create ticket (blocked by Notion API availability)
**Priority:** LOW
**Estimate:** 4 hours
**Dependency:** Notion webhook API release

---

### C7. **Sync Conflicts Detection** 🎫 HIGH
**File:** `internal/sync/conflicts/detector.go:13`
```go
// TODO: Implement after specification decision on:
```
**Action:** Create architectural decision ticket
**Priority:** HIGH
**Estimate:** 3 days

---

### C8. **NATS Messaging** 🎫 MEDIUM
**File:** `internal/sync/messaging/nats.go:14`
```go
// TODO: Implement after specification decisions on:
```
**Action:** Create architectural decision ticket
**Priority:** MEDIUM
**Estimate:** 2 days

---

### C9. **OSA Resilient Client** 🎫 LOW
**File:** `internal/integrations/osa/resilient_client.go:312`
```go
// TODO: Implement actual request processing
```
**Action:** Review if already implemented, create ticket if needed
**Priority:** LOW
**Estimate:** TBD

---

## ℹ️ TEST ONLY (Keep for Documentation)

### T1-T4. **Test Setup TODOs**
**Files:**
- `internal/services/background_jobs_integration_test.go:21`
- `internal/services/role_context_test.go:17`
- `internal/services/workspace_service_test.go:60`
- `internal/services/memory_hierarchy_service_test.go:361`

```go
// TODO: Setup test database connection
// TODO: Setup test database with testcontainers
// TODO: This test would verify:
```
**Action:** Keep as documentation of test setup needs
**Note:** These are intentional TODOs in test files for future test improvements

---

## 🔄 STUBS (Intentional Placeholders)

### S1-S3. **Service Stubs**
**Files:**
- `internal/services/osa_sync_service_stub.go`
- `internal/sync/conflicts/detector.go`
- `internal/sync/messaging/nats.go`

**Action:** Keep as intentional stubs until specification is finalized
**Note:** These files are placeholders waiting for architectural decisions

---

## 📋 Recommended Actions

### Immediate (2 hours)
1. ✅ Resolve 8 trivial TODOs
2. ✅ Add configuration options (env vars)
3. ✅ Remove completed TODOs

### Short-term (1 week)
1. 📋 Create 7 Medium priority Linear tickets
2. 📋 Document all Complex TODOs with ticket references
3. 📋 Update CLAUDE.md with TODO resolution policy

### Long-term (Sprint planning)
1. 📋 Create 14 Complex feature tickets in Linear
2. 📋 Prioritize based on business value
3. 📋 Add to backlog for sprint planning

---

## 🎯 Success Metrics

- [x] All 36 TODOs cataloged
- [ ] 8 Trivial TODOs resolved (0/8)
- [ ] 21 Complex TODOs documented with ticket references (0/21)
- [ ] 7 TODOs kept as intentional placeholders (7/7)
- [ ] Build passes after cleanup
- [ ] No functionality broken

---

**Next Steps:**
1. Resolve trivial TODOs
2. Create Linear tickets for complex TODOs
3. Update code with ticket references
4. Verify build and tests
5. Commit changes
