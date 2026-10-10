# Communications Module Sprint — Multi-Session Plan

**Branch:** `feat/communications-module` (off `main` @ `48286dc`)
**Goal:** Bring the communications module end-to-end alive — fix broken Gmail backend, type the channels page, wire engine sync for emails/messages, deliver unified inbox + realtime + compose polish.

## Team

| Codename | Role | Strengths |
|---|---|---|
| **Axis** (Claude Opus 4.7, 1M ctx) | Architect / coordinator / engineer | Spec authoring, cross-session integration, engine sync wiring, code review, merge orchestration |
| **Leah** | UI/UX designer (Claude Code session) | Email tab visual + interaction design, design-token enforcement |
| **Leon** | UI/UX designer (Claude Code session) | Channels tab visual + interaction design, design-token enforcement |
| **Ghost** | Engineer (Claude Code session) | Go backend — plumbing, route wiring, engine hooks |
| **Fantem** | Engineer (Claude Code session) | Frontend integration — types, API clients, realtime, error UX |

## Working agreement

- **Branch model:** every session works directly on `feat/communications-module`. No sub-branches. Sessions pull before they start, commit small, push frequently. Axis resolves any conflicts at integration time.
- **Required reading at start of each session** (paste these paths into the kickoff prompt):
  - `docs/COMMUNICATIONS_AUDIT.md` — what's broken, what works
  - `docs/COMMUNICATIONS_DESIGN_SYSTEM.md` — tokens + patterns
  - `docs/COMMUNICATIONS_SPRINT_PLAN.md` — this file
- **No new design tokens.** If you can't express it with `--bos-*`, raise to Axis before adding one.
- **No new CSS classes** unless they are component-scoped and prefixed `ch-` (communications namespace) or `cm-` (sub-feature).
- **Svelte 5 runes only.** No `createEventDispatcher`, no legacy reactive `$:`.
- **Type safety.** No `any` in any new code. Existing `any` is on the cleanup list.
- **Daily merge cadence.** Sessions push end-of-day; Axis reviews and integrates.

---

## Wave 1 — Unblock & stabilize (Days 1–3, all parallel)

The goal of wave 1: by Friday, the existing UI works for the existing happy paths.

### 🔧 Ghost — Wave 1 Task: Fix the broken Gmail backend

**Why this matters:** the Email tab is a UI talking to stubs. Without this, nothing else in email can work.

**Files:**
- `desktop/backend-go/internal/integrations/google/tool_handler.go` (lines 336–392 are stubs)
- `desktop/backend-go/internal/integrations/google/handler.go` (legacy with the real impl, mirror its delegation pattern)
- `desktop/backend-go/internal/integrations/google/gmail_actions.go`, `gmail_read.go`, `gmail_send.go` (the working services)
- `desktop/backend-go/internal/handlers/integrations_router.go` (lines 144–162 — register new routes here)

**Concrete deliverables:**
1. Replace stubs in `tool_handler.go` `GetEmails`, `GetEmail`, `SendEmail`, `SyncGmail` with real calls into `h.gmail.GetEmails(...)`, `h.gmail.GetEmail(...)`, `h.gmail.SendEmail(...)`, `h.gmail.SyncEmails(...)`. Mirror the calendar pattern at lines 239, 272, 296, 321 of the same file.
2. Add the missing routes on `/integrations/google_gmail/*` to match what the frontend already calls:
   - `POST /emails/:id/read` → `h.gmail.MarkAsRead(...)`
   - `POST /emails/:id/archive` → `h.gmail.ArchiveEmail(...)`
   - `DELETE /emails/:id` → `h.gmail.DeleteEmail(...)`
3. Add `GET /integrations/google_gmail/stats` returning `{has_access, total_emails, unread_count, last_sync}` from the `emails` table (`SELECT count(*), count(*) FILTER (WHERE NOT is_read), max(updated_at) FROM emails WHERE user_id = $1 AND provider='gmail'`).
4. Wire the new stats endpoint into `frontend/src/lib/api/gmail/gmail.ts:155-163` (replace the hardcoded zero return with a real fetch).

**Acceptance:**
- Connect Gmail in the Email tab → emails actually appear.
- Click an email → it loads.
- Click compose → email sends.
- Sidebar unread badge shows the real count.
- Mark-as-read updates both DB and Gmail (RemoveLabelIds: ["UNREAD"]).
- Run `go test ./...` clean.

**Independence:** Fully independent. Doesn't block Leah/Leon/Fantem.

---

### 🔧 Fantem — Wave 1 Task: Type & error-harden the Channels page

**Why this matters:** the page works but uses `any[]` everywhere. One unexpected Slack payload shape and it crashes silently.

**Files:**
- `frontend/src/routes/(app)/communication/channels/+page.svelte` (689 lines, lines 8–15 declare the `any[]`)
- `frontend/src/lib/api/index.ts` lines 226–234 (Slack methods on the umbrella `api` object)

**Concrete deliverables:**
1. Create `frontend/src/lib/api/slack/types.ts` with:
   - `SlackChannel` (id, name, is_private, member_count, last_activity, etc. — read from backend `slack_channels` schema in `internal/database/schema.sql:2484`)
   - `SlackMessage` (id, channel_id, slack_ts, sender, sender_id, content, sent_at, thread_ts, reactions, etc. — schema line 2518)
   - `SlackReaction` (icon, count, users)
2. Create `frontend/src/lib/api/slack/slack.ts` mirroring `gmail.ts` shape — typed wrappers for `getSlackChannels`, `getSlackMessages`, `sendSlackMessage`, `syncSlackChannels`, `syncSlackMessages`, `initiateSlackAuth`, `disconnectSlack`, `getSlackConnectionStatus`.
3. Update `channels/+page.svelte` to import from `$lib/api/slack` and replace `any[]` with `SlackChannel[]` / `SlackMessage[]`.
4. Replace `console.error` with a user-visible toast. Use the existing toast system (look at `frontend/src/lib/components/notifications/` for the pattern; if no toast helper exists, use `lib/components/ui/ErrorBoundary.svelte` shape and create `cm-toast` component scoped under comms).
5. Type-safe access: `message.reactions?.length > 0` → already handles undefined, but typed properly now.
6. Run `npm run check` (svelte-check) clean — zero TS errors.

**Acceptance:**
- `channels/+page.svelte` compiles with strict TS, no `any`.
- Network errors show a toast, not silent console.error.
- `npm run check` clean.

**Independence:** Independent of Ghost's work. Lives in different files.

---

### 🎨 Leah — Wave 1 Task: Email tab UX audit + redesign spec

**Why this matters:** before we add Outlook + threading + attachments, we need to know what the unified inbox SHOULD look like. Today's UI was built one provider at a time and shows it.

**Files (read-only audit):**
- `frontend/src/routes/(app)/communication/email/+page.svelte` (1,124 lines — the current state)
- `frontend/src/lib/components/calendar/*` (the architectural template — sidebar + toolbar + content + modals + utils — that worked)
- `docs/COMMUNICATIONS_DESIGN_SYSTEM.md` (the rules)

**Concrete deliverables (no code yet — design only):**
1. **Audit doc** at `docs/design/comms-email-audit.md`:
   - Screenshot or sketch annotations of what works and what doesn't in the current Email tab.
   - List every decorative/non-functional UI element (Paperclip, Archive button in preview actions, drafts folder, etc.).
   - Identify any token violations (hardcoded colors, off-scale spacing).
2. **Redesign spec** at `docs/design/comms-email-spec.md`:
   - Unified inbox with provider switcher (Gmail / Outlook / All).
   - Thread view (collapsed message stack like Gmail mobile / Linear).
   - Compose v2: rich text body, attachments, contact autocomplete, draft auto-save, send-later.
   - Empty states for: not connected, connecting, no emails in folder, search no-match.
   - Loading states: list skeleton, preview skeleton, sync in progress.
   - Every component referenced uses tokens from the design system doc — list them.
3. **Component breakdown** — what new components are needed and what gets reused from `lib/components/chat/messages/*` (for thread view) and `lib/components/forms/*` (for compose).

**Acceptance:**
- Spec is concrete enough that Fantem could implement from it without asking design questions.
- Every color, space, radius cited as a `--bos-*` or `--space-N` / `--radius-X` token.
- No new tokens proposed unless justified.

**Independence:** Pure design — independent of all engineers. Output is a doc Fantem reads in Wave 2.

---

### 🎨 Leon — Wave 1 Task: Channels tab UX audit + redesign spec

**Why this matters:** today the Channels tab is Slack-only. The schema and backend support multi-provider (Slack + Teams). We need to know what a clean multi-provider channels UI is before Fantem builds it.

**Files (read-only):**
- `frontend/src/routes/(app)/communication/channels/+page.svelte`
- `frontend/src/lib/components/chat/conversations/*` (existing patterns to mimic)
- `frontend/src/lib/components/chat/messages/*` (message bubble patterns)
- `desktop/backend-go/internal/integrations/microsoft/handler.go` (read what Teams endpoints look like — they don't exist yet but should mirror Slack)

**Concrete deliverables:**
1. **Audit doc** `docs/design/comms-channels-audit.md` — same shape as Leah's.
2. **Redesign spec** `docs/design/comms-channels-spec.md`:
   - Multi-provider sidebar (workspace selector at top: Slack / Teams; channels grouped under each).
   - Thread/reply view (Slack threads + Teams replies).
   - Message reactions (display works today; sending must be designed).
   - Presence indicators (online/away).
   - DMs separated from channels.
   - Empty states + loading + error.
   - Compose: file upload (image preview), @mentions autocomplete, emoji picker.
3. **Component breakdown** — what to reuse from chat module, what's new.

**Acceptance:** same as Leah's. Concrete, token-faithful, no design questions left.

**Independence:** Pure design — independent.

---

### 🧠 Axis — Wave 1 Task: Engine sync hooks spec + module-level wiring

**Why this matters:** the audit identified that calendar events flow into OptimalEngine via `OnEventCreated`, but emails and Slack messages don't. Without engine sync, agents can never read inbox/channels via RAG. This is the single biggest "module comes alive" lever.

**Concrete deliverables:**
1. **Spec doc** `docs/COMMUNICATIONS_ENGINE_SYNC.md` describing:
   - Add `ModuleEmail` and `ModuleMessage` to `internal/services/engine_sync.go:88-98`.
   - Mapping: email subject → Signal.Title, body_text + snippet → Signal.Body, sender + thread_id → Metadata.
   - Mapping: Slack message → Signal where Title = first 80 chars, Body = full text, Metadata = channel_id + sender.
   - Hook attachment points in `Gmail.SaveEmail` (gmail_read.go:79) and `Slack.SaveMessage` (in `slack/messages.go`).
2. **Implementation** (Axis owns this directly since it crosses concerns):
   - Add the two `Module` constants and entries to `moduleToNode` + `moduleDefaults` maps.
   - Add `OnEmailSaved` callback to `GmailService` mirroring the calendar `OnEventCreated` pattern.
   - Add `OnMessageSaved` callback to `slack.MessageService`.
   - Wire both in `handlers/routes_integrations.go` alongside the existing calendar hooks (lines 65–74).
   - Write a tiny smoke test: connect Gmail, sync, verify a Signal lands in the engine via the existing `/api/optimal/*` query path.

**Acceptance:**
- `engine_sync.go` has `ModuleEmail` and `ModuleMessage`.
- Synced Gmail and Slack messages appear in engine queries.
- Calendar engine sync still works (no regression).
- Spec doc explains the data shape so Ghost/Fantem can extend later without re-deriving.

**Dependency:** technically depends on Ghost's Wave 1 (Gmail backend must work to verify end-to-end). But the spec + the hook plumbing can be written in parallel and merged after Ghost.

---

## Wave 2 — Build on stable foundations (Days 4–7)

By end of wave 1: Gmail works, Channels is typed, designs are speced, engine sync hooks exist. Wave 2 implements the redesigns and wires the multi-provider story.

### 🔧 Ghost — Wave 2 Task: Outlook surface + unified inbox endpoint

**Files:**
- `desktop/backend-go/internal/integrations/microsoft/handler.go` (already has email routes at lines 60–63)
- New: `desktop/backend-go/internal/handlers/comms_inbox.go`
- `desktop/backend-go/internal/database/queries/` — new sqlc query for unified inbox

**Deliverables:**
1. Verify Outlook routes are reachable end-to-end (token exchange + list + send + sync). Fix any stubs same way as Gmail.
2. Add unified inbox endpoint: `GET /api/comms/inbox?providers=gmail,outlook&folder=inbox&limit=50`. Returns a merged list across `emails` (Gmail) + `microsoft_mail_messages` tables, normalized to a single `UnifiedEmail` shape, ordered by date desc.
3. Add `OnEmailSaved` hook for Outlook mirror of what Axis added for Gmail in Wave 1.
4. Add Teams channel routes (Microsoft Graph `/teams/{id}/channels`, `/channels/{id}/messages`) matching Slack's shape — register in `microsoft/handler.go` parallel to mail routes.

---

### 🔧 Fantem — Wave 2 Task: Email tab implementation against Leah's spec

**Inputs:**
- `docs/design/comms-email-spec.md` (Leah's output)
- Working backend (Ghost's wave 1 output, plus unified endpoint from Ghost's wave 2)
- Engine sync exists (Axis's wave 1)

**Deliverables:**
1. Refactor `email/+page.svelte` per spec.
2. Add `frontend/src/lib/api/comms/inbox.ts` typed client for the unified endpoint.
3. Build new components: `EmailThreadView`, `EmailComposeV2`, `EmailProviderSwitcher`, `EmailEmptyState`. Place in `frontend/src/lib/components/comms/email/`.
4. Wire compose attachments (multipart upload).
5. Wire drafts persistence (use the `is_draft` column already in `emails` table).
6. Replace decorative buttons with real handlers OR remove.
7. Add toasts on errors.

**Acceptance:** Leah signs off on visual parity with her spec. Connect Gmail + Outlook → unified inbox shows both.

---

### 🎨 Leah — Wave 2 Task: Implementation review + visual QA

**Deliverables:**
1. Review Fantem's commits against the spec, line-by-line.
2. Token compliance check using the checklist in `COMMUNICATIONS_DESIGN_SYSTEM.md`.
3. Cross-theme QA (light + dark).
4. Sketch any iteration deltas as comments + small fix-up commits.

---

### 🎨 Leon — Wave 2 Task: Channels tab implementation pairing with Fantem (or Ghost depending on capacity)

Mirror of Leah/Fantem pairing on the email side, but for channels. Spec → implementation → review.

---

### 🧠 Axis — Wave 2 Task: Realtime architecture + cross-session integration

**Deliverables:**
1. Spec `docs/COMMUNICATIONS_REALTIME.md`:
   - Single SSE endpoint `/api/comms/stream` multiplexing Gmail Pub/Sub watch + Slack Events API + Outlook Graph subscriptions.
   - Frontend: one `EventSource` per app session that fans out into per-tab stores.
   - Backwards plan: while webhook setup is happening, fall back to polling at 30s.
2. Daily integration on `feat/communications-module`. Resolve conflicts. Run integration tests.
3. Cross-session blockers triage: when Leah/Leon/Ghost/Fantem hit a question that crosses domains, Axis answers within the working day.

---

## Wave 3 — Polish + realtime (Week 2)

### Ghost — Realtime backend
- Gmail `users.watch` + Pub/Sub webhook receiver.
- Slack Events API webhook handler (push to `slack_messages`, broadcast on SSE).
- Outlook Graph subscription handler.
- The unified `/api/comms/stream` SSE endpoint.

### Fantem — Realtime frontend
- One `EventSource` connection from `+layout.svelte` (the comms hub).
- Per-tab subscriptions: email, channels listen for their event types and update local state.
- Optimistic UI for compose/send.
- Reconnect-with-backoff on stream drop.

### Leah — Final polish pass: email
- Loading state animations.
- Empty state illustrations (or icon arrangements — no new asset commitments without Axis approval).
- Microcopy review.
- Keyboard shortcuts (j/k navigate, e archive, # delete — match Gmail).

### Leon — Final polish pass: channels
- Same as Leah but for channels. Plus DM presence, typing indicators if realtime supports.

### Axis
- Final integration, cross-tab QA, ship-ready merge to `main`.
- Ensure engine-sync indices have backfilled all historical emails/messages so RAG works on day-one inbox content.
- Write changelog + handoff for Roberto.

---

## Coordination prompts (paste-ready)

When you spin up a new Claude Code session, paste the corresponding prompt below as the first message.

### For Ghost (engineer):

```
You are Ghost, a backend engineer working on BusinessOS. Read these three files first
in order, then await my task brief:

1. docs/COMMUNICATIONS_AUDIT.md
2. docs/COMMUNICATIONS_DESIGN_SYSTEM.md
3. docs/COMMUNICATIONS_SPRINT_PLAN.md (your specific tasks are listed under "Ghost")

You work directly on branch feat/communications-module — no sub-branches. Pull before
you start, commit small, push frequently. Do not start coding until you have read all
three docs and confirmed your wave 1 task scope back to me.
```

### For Fantem (engineer):

```
You are Fantem, a frontend integration engineer working on BusinessOS. Read these
three files first in order, then await my task brief:

1. docs/COMMUNICATIONS_AUDIT.md
2. docs/COMMUNICATIONS_DESIGN_SYSTEM.md
3. docs/COMMUNICATIONS_SPRINT_PLAN.md (your specific tasks are listed under "Fantem")

You work directly on branch feat/communications-module — no sub-branches. The codebase
uses Svelte 5 runes and TypeScript strict — no `any`, no createEventDispatcher. Do not
start coding until you have read all three docs and confirmed your wave 1 task scope
back to me.
```

### For Leah (UI/UX):

```
You are Leah, a UI/UX designer working on BusinessOS. Your wave 1 task is design
specification — no code yet. Read these in order:

1. docs/COMMUNICATIONS_AUDIT.md
2. docs/COMMUNICATIONS_DESIGN_SYSTEM.md  ← this defines every token you can use
3. docs/COMMUNICATIONS_SPRINT_PLAN.md (your task is under "Leah")

Then read frontend/src/routes/(app)/communication/email/+page.svelte to understand
the current state, and frontend/src/lib/components/calendar/* as the architectural
template that worked.

Your output for wave 1 is two markdown docs in docs/design/. Concrete enough that
Fantem can build from them without further design input. Every color/space/radius
must reference an existing --bos-* or scale token — no new tokens.
```

### For Leon (UI/UX):

```
You are Leon, a UI/UX designer working on BusinessOS. Your wave 1 task is design
specification — no code yet. Read these in order:

1. docs/COMMUNICATIONS_AUDIT.md
2. docs/COMMUNICATIONS_DESIGN_SYSTEM.md  ← this defines every token you can use
3. docs/COMMUNICATIONS_SPRINT_PLAN.md (your task is under "Leon")

Then read frontend/src/routes/(app)/communication/channels/+page.svelte and
frontend/src/lib/components/chat/* (the existing chat module — your reference
patterns).

Your output for wave 1 is two markdown docs in docs/design/. Concrete enough that
Fantem (or Ghost) can build from them without further design input. Every
color/space/radius must reference an existing --bos-* or scale token.
```

---

## Dependency graph (so Axis knows merge order)

```
Wave 1 (parallel, all start day 1):
  Ghost:  fix gmail tool-handler stubs + missing routes + stats   ─┐
  Fantem: type the channels page + error toasts                   ─┼─► merge order: Ghost → Fantem → Axis
  Leah:   email-redesign-spec doc                                 ─┤   (Leah/Leon are docs, no merge conflicts)
  Leon:   channels-redesign-spec doc                              ─┤
  Axis:   engine-sync hooks for ModuleEmail + ModuleMessage       ─┘   (depends on Ghost wave 1 for verify)

Wave 2 (starts day 4 once wave 1 lands):
  Ghost  → unified inbox + Outlook + Teams       (depends on Ghost wave 1)
  Fantem → Email tab v2                          (depends on Leah spec + Ghost wave 1+2 + Axis wave 1)
  Leon/Fantem → Channels tab v2                  (depends on Leon spec + Ghost wave 2)
  Leah   → email visual QA on Fantem's commits
  Leon   → channels visual QA
  Axis   → realtime spec + integration

Wave 3 (week 2):
  Ghost  → realtime backend
  Fantem → realtime frontend
  Leah   → email polish
  Leon   → channels polish
  Axis   → ship + handoff to Roberto
```

## Risk register

| Risk | Mitigation |
|---|---|
| Gmail stub fix has hidden auth-scope issues (the new tool-OAuth uses different scope tokens than legacy unified) | Ghost verifies token rows in `google_oauth_tokens` table during wave 1. Fall back to legacy unified path if scopes don't match. |
| OptimalEngine doesn't have `ModuleEmail` / `ModuleMessage` indexed | Axis confirms with engine maintainer before adding. Fallback: route through `ModuleChat` if engine resists schema changes. |
| Outlook surface has hidden bugs (300 lines of code never used in UI) | Ghost wave 2 starts with a smoke test — connect, sync, list 5 emails. Fix before building unified inbox. |
| Slack Events API needs public webhook (no localhost) | Wave 3 only. Use ngrok for dev. Production needs real domain — flag for Roberto. |
| Design tokens diverge as Leah/Leon work in parallel | Both must reference COMMUNICATIONS_DESIGN_SYSTEM.md. Axis reviews their specs day-by-day. |
| Two sessions touching the same file simultaneously on the shared branch | Coordinate in standup: Leah owns email design, Fantem owns email impl. Leon owns channels design, Fantem (or Ghost overflow) owns channels impl. Axis stays out of `+page.svelte` files. |
