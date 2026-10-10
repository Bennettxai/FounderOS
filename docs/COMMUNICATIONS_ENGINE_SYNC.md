# Communications → OptimalEngine Sync Spec

**Owner:** Axis
**Branch:** `feat/communications-module`
**Status:** Workspace-governed routing implemented 2026-07-20
**Why:** today calendar events flow to the OptimalEngine knowledge graph (`OnEventCreated` → `engineSync.Enqueue(Signal{Module: ModuleCalendar})`). Emails and Slack messages don't. Without these signals, OSA agents can't read inbox/channels via RAG. This is the single biggest "module comes alive" lever.

---

## Mental model

The engine takes a `Signal{Module, ID, Title, Body, Genre, AuthorID, ModifiedAt, Metadata}` and:

1. Writes a markdown file to `OPTIMAL_NODES_ROOT/<node>/<id>.md` with frontmatter.
2. POSTs the signal to `/api/memory` for first-class memory storage with versioning + dedup.
3. Marks the node folder dirty; a debounced (5s) background reindex syncs the engine's SQLite to disk.

We add two new module folders. Every new email and every new Slack message becomes a signal.

---

## What gets added

### 1. `internal/services/engine_sync.go` — register two new modules

Add to the constants block (`engine_sync.go:88-98`):

```go
const (
    ...existing...
    ModuleEmail   Module = "email"   // → 19-email
    ModuleMessage Module = "message" // → 20-message
)
```

Add to `moduleToNode` (`engine_sync.go:102-112`):

```go
ModuleEmail:   "19-email",
ModuleMessage: "20-message",
```

Add to `moduleDefaults` (`engine_sync.go:120-130`):

```go
ModuleEmail:   {"email", "BusinessOS Email", "Synced emails from Gmail and Outlook surfaced through the BusinessOS Communications module."},
ModuleMessage: {"message", "BusinessOS Messages", "Synced channel and DM messages from Slack, Teams, and other communication providers."},
```

### 2. Signal mapping

**Email → Signal**

| Signal field | Source | Notes |
|---|---|---|
| `Module` | `ModuleEmail` | constant |
| `ID` | `email.ID` (DB UUID, not `external_id`) | so re-syncs hit the same MD file |
| `Title` | `email.Subject` (fallback `"(no subject)"` when empty) | |
| `Body` | `body_text` (fallback to plaintext-stripped `body_html`, fallback to `snippet`) | what the engine indexes for RAG |
| `Genre` | `"email"` | inherited from `moduleDefaults` |
| `AuthorID` | `email.UserID` | the BO user, NOT the email sender |
| `ModifiedAt` | `email.Date` (received time) | engine recency ranking |
| `Metadata` | see below | |

Metadata for email:
```go
{
    "provider":   "gmail" | "outlook",
    "from_email": email.FromEmail,
    "from_name":  email.FromName,
    "thread_id":  email.ThreadID,
    "is_sent":    "true" | "false",
    "labels":     comma-joined Gmail labels,
}
```

**Slack message → Signal**

| Signal field | Source | Notes |
|---|---|---|
| `Module` | `ModuleMessage` | constant |
| `ID` | `slack_messages.id` (DB UUID) | |
| `Title` | first 80 chars of `content` (collapsing whitespace), fallback `"<sender> in #<channel>"` | engine titles must be short |
| `Body` | full `content` text | |
| `Genre` | `"message"` | |
| `AuthorID` | `userID` (the BO user whose workspace this is) | NOT the Slack sender |
| `ModifiedAt` | `sent_at` | |
| `Metadata` | see below | |

Metadata for message:
```go
{
    "provider":   "slack",
    "channel_id": channelID,        // BO uuid, not slack_id
    "sender_id":  msg.SenderID,     // slack user id
    "sender_name": msg.SenderName,
    "thread_ts":  msg.ThreadTS,     // empty when not a thread reply
    "slack_ts":   msg.SlackTS,
}
```

### 3. Hook attachment points

**Gmail** — `internal/integrations/google/gmail_types.go` `GmailService` struct (line 79).
Add `OnEmailSaved EmailSavedHook` field. Hook fires from `gmail_read.go::saveEmail` AFTER the INSERT/UPSERT succeeds — we want the DB row (with its UUID) before the signal goes out. Refetch the row by `(user_id, provider, external_id)` to get the canonical ID.

```go
type EmailSavedHook func(ctx context.Context, email *Email, userID string)
```

**Slack** — `internal/integrations/slack/messages.go` `MessageService` struct (line 36).
Add `OnMessageSaved MessageSavedHook` field. Fires from `saveMessage` after upsert; refetch by `(user_id, channel_id, slack_ts)` to get the canonical ID.

```go
type MessageSavedHook func(ctx context.Context, msg *Message, userID string)
```

### 4. Wiring — `internal/handlers/comms_engine_sync.go` (new)

A small file matching the shape of `calendar_engine_sync.go`. Two factory functions: `newGmailEngineHook(sync)` and `newSlackMessageEngineHook(sync)`. Each returns a closure that builds a `Signal` and calls `sync.Enqueue`.

### 5. Wire-up in `routes_integrations.go`

Right next to the existing calendar hooks (lines 65-74). After the `gmailSvc` adapter wiring at line 31, attach the email hook. After the `slackProv` adapter wiring at line 35, attach the message hook through `integrationRouter.GetSlackMessageService()`.

---

## What is intentionally NOT in scope for this wave

- **Deletion sync.** When a user trashes an email or a Slack message gets deleted, we don't yet `EnqueueDelete`. Wave 3 cleanup.
- **Outlook hook.** Microsoft mail uses `microsoft_mail_messages`, separate codepath. Ghost wires this in Wave 2 alongside the unified inbox endpoint. Same hook shape, different table.
- **Backfill of existing rows.** This wave only catches new syncs. A separate one-shot backfill job will run before ship — Axis Wave 3.
- **Read-receipts flowing into engine.** Whether an email is read is metadata, but the Signal model doesn't have a "read" dimension; we'd just be re-emitting the same Signal. Skip.

## Local WhatsApp source

WhatsApp history uses a different lifecycle from OAuth providers.
The whatsapp-mcp bridge owns the canonical SQLite database and synchronizes source history into it.
BusinessOS opens that database in read-only query mode and exposes its chats and messages through the unified Communications channel API.

The UI reports three independent states:

1. `connection_status=connected` means the canonical local database is available.
2. `sync_status=history_synced` means the bridge has synchronized WhatsApp history into that database.
3. `routing_status=unassigned` means no chat has been assigned to a governed BusinessOS or Optimal Engine workspace.

BusinessOS must not treat history synchronization as workspace routing.
It must not copy, edit, send, or delete WhatsApp source records through this adapter.
Optimal Engine ingestion requires an explicit chat-to-workspace routing decision so private conversations cannot leak between organizations.

## Workspace routing contract

Every communication source is readable before it is routed, but it is excluded from Optimal Engine until a route resolves.
Routes are stored in `communication_routes` and are scoped to the authenticated user, provider, and either an account default or one conversation.
Conversation routes take precedence over account defaults.
Only an active member can route data into a BusinessOS workspace.
The resolved workspace UUID is written to every `services.Signal`, and the route scope is preserved in signal metadata.

The Communications channel view shows the effective destination and lets a user assign or clear the current conversation route.
Slack uses the internal BusinessOS channel UUID.
Teams resolves the Graph channel identifier to its internal BusinessOS channel UUID before route lookup.
Gmail uses its thread identifier, Outlook uses its conversation identifier, and WhatsApp uses its canonical local chat identifier.

WhatsApp remains read-only at the provider boundary.
Assigning a WhatsApp conversation can backfill up to 500 recent messages, and manual Communications sync refreshes up to 200 messages for every routed WhatsApp conversation.
Engine deduplication makes repeat syncs safe.
Account-level WhatsApp routing is intentionally unavailable in the UI because chats commonly belong to different organizations.

Set `WHATSAPP_MESSAGES_DB` to override the default desktop path at `~/.local/share/whatsapp-mcp/whatsapp-bridge/store/messages.db`.

---

## Smoke tests (verification step Day 3)

### Slack — verify standalone (does not depend on Ghost)
1. `go test ./internal/services -run TestEngineSync_Modules` (unit: constants registered)
2. With local engine running:
   - Connect Slack via UI, sync a channel
   - `curl localhost:8001/api/optimal/search?q=<recent-message-text>` returns the message
   - `ls $OPTIMAL_NODES_ROOT/20-message/` shows the synced MD files

### Gmail — depends on Ghost's Wave 1 landing
1. With Ghost's Gmail tool-handler fix merged: connect Gmail, sync 10 emails
2. `curl localhost:8001/api/optimal/search?q=<email-subject-keyword>` returns the email
3. `ls $OPTIMAL_NODES_ROOT/19-email/` shows MD files
4. Open one MD file → frontmatter has `module: email`, `metadata.provider: gmail`, `metadata.from_email: ...`

### No-regression
- Calendar event creation still produces `16-calendar/*.md` (unchanged hook).
- All existing tests pass: `go test ./...`.

---

## Notes for downstream sessions

- **Fantem (Wave 2 email v2):** the engine-sync exists at insert time. If you add a "favorite" toggle that updates `is_starred`, that does NOT re-fire the hook — the hook is only on insert/upsert. If we need starred-state in engine metadata, we add it then.
- **Ghost (Wave 2 Outlook):** mirror this exact spec. Add `OnEmailSaved` field to whatever Outlook service struct is the equivalent of `GmailService`. Reuse `ModuleEmail` — both providers share the module.
- **Leah/Leon:** when designing thread/conversation views, you can lean on the fact that emails and messages already have `thread_id`/`thread_ts` flowing into engine metadata. Agents can answer "show me everything in thread X" via RAG.
