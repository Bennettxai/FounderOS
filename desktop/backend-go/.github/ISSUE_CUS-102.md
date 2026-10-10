# CUS-102: Enhance Webhook Persistence (Optional)

## 📋 Summary

Store webhook data in the database for historical querying, offline access, and analytics.

## 🎯 Background

Currently, all webhooks (Google Calendar, Slack, Linear, HubSpot) are processed successfully and logged, but the data isn't persisted to the database. This enhancement would enable:

- **Historical querying** - Search past webhook events
- **Offline access** - View integration data without API calls
- **Analytics** - Track webhook patterns and usage
- **Search functionality** - Full-text search across webhook data
- **Audit trail** - Complete history of integration events

## ✅ Current Behavior

All webhooks work correctly and log events:
```
✅ Google Calendar: Syncs calendars, processes events
✅ Slack: Receives messages, processes commands
✅ Linear: Tracks issue updates, status changes
✅ HubSpot: Receives contact updates, deal changes
✅ Notion: (Blocked - Notion API doesn't support webhooks yet)
```

## 🚀 Enhanced Behavior

Webhook data would be stored in database tables for later querying.

## 📦 Implementation Tasks

### 1. Google Calendar Sync Queue (3 hours)

**File:** `internal/webhooks/handler.go:96`

**Current Code:**
```go
// TODO: Queue sync job instead of just logging
slog.Info("Received Google Calendar webhook", "watch_id", watchID)
```

**Enhanced Implementation:**
```go
// Queue background sync job
job := &jobs.GoogleCalendarSyncJob{
    UserID:       userID,
    CalendarID:   calendarID,
    ResourceID:   resourceID,
    ChannelID:    channelID,
    ReceivedAt:   time.Now(),
}

err := h.jobQueue.Enqueue(ctx, job)
if err != nil {
    slog.Error("Failed to enqueue calendar sync", "error", err)
    return
}

slog.Info("Queued Google Calendar sync job", "user_id", userID, "calendar_id", calendarID)
```

**Database Schema (if needed):**
```sql
CREATE TABLE google_calendar_sync_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    resource_id TEXT,
    channel_id TEXT,
    status TEXT DEFAULT 'pending', -- pending, processing, completed, failed
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendar_sync_status ON google_calendar_sync_queue(status, received_at);
CREATE INDEX idx_calendar_sync_user ON google_calendar_sync_queue(user_id);
```

### 2. Slack Message Storage (4 hours)

**File:** `internal/webhooks/handler.go:198`

**Current Code:**
```go
// TODO: Store message in database for offline access
slog.Info("Received Slack message", "channel", channelID, "user", userID, "text", text)
```

**Enhanced Implementation:**
```go
// Store Slack message in database
message := &sqlc.CreateSlackMessageParams{
    MessageID:   messageID,
    ChannelID:   channelID,
    UserID:      userID,
    Text:        text,
    Timestamp:   slackTimestamp,
    ThreadTS:    threadTS, // if reply
    ReceivedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
}

err := h.queries.CreateSlackMessage(ctx, message)
if err != nil {
    slog.Error("Failed to store Slack message", "error", err)
    // Don't fail webhook - just log error
}

slog.Info("Stored Slack message", "message_id", messageID, "channel", channelID)
```

**Database Schema:**
```sql
CREATE TABLE slack_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id TEXT NOT NULL UNIQUE,
    channel_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    text TEXT NOT NULL,
    timestamp TEXT NOT NULL, -- Slack timestamp
    thread_ts TEXT, -- If part of thread
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_slack_messages_channel ON slack_messages(channel_id, timestamp DESC);
CREATE INDEX idx_slack_messages_user ON slack_messages(user_id, timestamp DESC);
CREATE INDEX idx_slack_messages_thread ON slack_messages(thread_ts) WHERE thread_ts IS NOT NULL;
```

**SQLC Query:**
```sql
-- name: CreateSlackMessage :exec
INSERT INTO slack_messages (
    message_id, channel_id, user_id, text, timestamp, thread_ts, received_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) ON CONFLICT (message_id) DO UPDATE SET
    text = EXCLUDED.text,
    received_at = EXCLUDED.received_at;
```

### 3. Linear Database Updates (4 hours)

**File:** `internal/webhooks/handler.go:287`

**Current Code:**
```go
// TODO: Update Linear issue in database
slog.Info("Linear issue updated", "issue_id", issueID, "action", action)
```

**Enhanced Implementation:**
```go
// Update Linear issue in database
issueUpdate := &sqlc.UpdateLinearIssueParams{
    IssueID:     issueID,
    Title:       issue.Title,
    Description: issue.Description,
    State:       issue.State.Name,
    Priority:    issue.Priority,
    UpdatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
}

err := h.queries.UpdateLinearIssue(ctx, issueUpdate)
if err != nil {
    slog.Error("Failed to update Linear issue", "issue_id", issueID, "error", err)
}

slog.Info("Updated Linear issue in database", "issue_id", issueID, "action", action)
```

**Database Schema:**
```sql
CREATE TABLE linear_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id TEXT NOT NULL UNIQUE,
    team_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    state TEXT,
    priority INT,
    assignee_id TEXT,
    creator_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_linear_issues_team ON linear_issues(team_id, updated_at DESC);
CREATE INDEX idx_linear_issues_assignee ON linear_issues(assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_linear_issues_state ON linear_issues(state);
```

**SQLC Queries:**
```sql
-- name: UpdateLinearIssue :exec
INSERT INTO linear_issues (
    issue_id, title, description, state, priority, updated_at, synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW()
) ON CONFLICT (issue_id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    state = EXCLUDED.state,
    priority = EXCLUDED.priority,
    updated_at = EXCLUDED.updated_at,
    synced_at = NOW();

-- name: GetLinearIssue :one
SELECT * FROM linear_issues WHERE issue_id = $1;

-- name: ListLinearIssues :many
SELECT * FROM linear_issues ORDER BY updated_at DESC LIMIT $1;
```

### 4. HubSpot Contact Sync (4 hours)

**File:** `internal/webhooks/handler.go:371`

**Current Code:**
```go
// TODO: Fetch and store HubSpot contact details
slog.Info("HubSpot contact updated", "contact_id", contactID)
```

**Enhanced Implementation:**
```go
// Fetch full contact details from HubSpot API
contact, err := h.hubspotProvider.GetContact(ctx, userID, contactID)
if err != nil {
    slog.Error("Failed to fetch HubSpot contact", "contact_id", contactID, "error", err)
    c.JSON(http.StatusOK, gin.H{"status": "webhook_received"})
    return
}

// Store in database
contactData := &sqlc.UpsertHubSpotContactParams{
    ContactID:   contactID,
    UserID:      userID,
    Email:       contact.Email,
    FirstName:   contact.FirstName,
    LastName:    contact.LastName,
    Company:     contact.Company,
    Properties:  contact.PropertiesJSON, // JSONB
    SyncedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
}

err = h.queries.UpsertHubSpotContact(ctx, contactData)
if err != nil {
    slog.Error("Failed to store HubSpot contact", "contact_id", contactID, "error", err)
}

slog.Info("Synced HubSpot contact to database", "contact_id", contactID)
```

**Database Schema:**
```sql
CREATE TABLE hubspot_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    company TEXT,
    properties JSONB, -- All HubSpot properties
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, contact_id)
);

CREATE INDEX idx_hubspot_contacts_user ON hubspot_contacts(user_id, synced_at DESC);
CREATE INDEX idx_hubspot_contacts_email ON hubspot_contacts(email) WHERE email IS NOT NULL;
CREATE INDEX idx_hubspot_properties ON hubspot_contacts USING GIN (properties);
```

### 5. Notion Webhooks (4 hours) - BLOCKED

**File:** `internal/webhooks/handler.go:399`

**Status:** ⏸️ **BLOCKED** - Notion API doesn't support webhooks yet

**Blocker:** Notion API limitation (as of 2026-01-19)

**Future Implementation:**
Once Notion adds webhook support, implement similar to other webhooks:
1. Parse Notion webhook payload
2. Store page/database updates in `notion_pages` table
3. Enable offline access to Notion content

### 6. NATS Messaging Architecture (20 hours) - BLOCKED

**File:** `internal/sync/messaging/nats.go:14`

**Status:** ⏸️ **BLOCKED** - Architecture decision needed

**Blocker:** NATS architecture not finalized (Q1-Q3 specification decisions needed)

**Scope:**
- Distributed messaging between services
- Event streaming architecture
- Message routing and filtering
- Delivery guarantees

**Timeline:** Requires team architecture decision meeting

## 📊 Estimated Effort

| Task | Hours | Priority |
|------|-------|----------|
| Google Calendar Sync Queue | 3 | Medium |
| Slack Message Storage | 4 | Medium |
| Linear Database Updates | 4 | Medium |
| HubSpot Contact Sync | 4 | Medium |
| Notion Webhooks | 4 | Low (blocked) |
| NATS Messaging | 20 | Low (blocked) |
| **Total (unblocked)** | **15** | - |
| **Total (all)** | **39** | - |

## 🎯 Benefits

### Short Term
- Historical webhook data queryable
- Offline access to integration data
- Reduced API calls to external services

### Long Term
- Full-text search across all integrations
- Analytics dashboards showing integration usage
- Audit trail for compliance
- Faster UI (local data instead of API calls)

## ⚠️ Trade-offs

### Pros
- ✅ Better offline experience
- ✅ Historical data access
- ✅ Reduced external API calls
- ✅ Search functionality
- ✅ Analytics capabilities

### Cons
- ❌ Increased database storage
- ❌ Additional maintenance (data cleanup)
- ❌ Potential data staleness
- ❌ Sync complexity

## 🚀 Implementation Strategy

### Phase 1: Core Webhooks (11 hours)
1. Slack message storage (4h)
2. Linear issue updates (4h)
3. Google Calendar sync queue (3h)

### Phase 2: CRM Integration (4 hours)
4. HubSpot contact sync (4h)

### Phase 3: Future (When Unblocked)
5. Notion webhooks (when API available)
6. NATS messaging (when architecture decided)

## ✅ Success Criteria

- [ ] Database schemas created
- [ ] SQLC queries generated
- [ ] Webhook handlers updated
- [ ] Data persisted correctly
- [ ] Migration scripts tested
- [ ] No performance degradation
- [ ] Documentation updated
- [ ] Can query historical webhook data

## 🔗 Related

- **Blocks:** None (low priority enhancement)
- **Related:** CUS-94 (TODO resolution)
- **Reference:** `desktop/backend-go/CUS-94_COMPLETION_SUMMARY.md`

## 📅 Timeline

**Priority:** Low (optional enhancement)
**Can Start:** Anytime
**Can Implement:** Incrementally (one integration at a time)
**Estimated Completion:** 2-3 weeks (if prioritized)

---

**Created:** 2026-01-19
**Assignee:** Backend Team
**Labels:** backend, webhooks, enhancement, low-priority, database
