# Communications Module — Deep Audit

**Repo:** BusinessOS
**Branch:** `main`
**HEAD at audit time:** `48286dc` — *feat: integrate OptimalEngine Svelte SDK across all modules*
**Audit date:** 2026-05-01

---

## 🚨 The biggest finding upfront

**The Email tab is a UI talking to a stubbed backend.** Two parallel Gmail backends exist; the frontend points at the broken one.

`frontend/src/lib/api/gmail/gmail.ts:18` sets `GMAIL_BASE = "/integrations/google_gmail"`. That path is served by `ToolHandler` in `desktop/backend-go/internal/integrations/google/tool_handler.go`. Look at what those handlers actually do:

| Endpoint frontend calls | Backend method | Reality |
|---|---|---|
| `GET /emails` | `ToolHandler.GetEmails` (`tool_handler.go:336-359`) | **Returns `[]interface{}{}` always** — comment says "Would call h.gmail.GetEmails(...)" |
| `GET /emails/:id` | `ToolHandler.GetEmail` (`tool_handler.go:362-370`) | **`501 Not Implemented`** |
| `POST /send` | `ToolHandler.SendEmail` (`tool_handler.go:373-381`) | **`501 Not Implemented`** |
| `POST /sync` | `ToolHandler.SyncGmail` (`tool_handler.go:384-392`) | **`501 Not Implemented`** |
| `POST /emails/:id/read` | — | **Route doesn't exist** (`tool_handler.go:81-89`) |
| `POST /emails/:id/archive` | — | **Route doesn't exist** |
| `DELETE /emails/:id` | — | **Route doesn't exist** |

Meanwhile, the **fully working** Gmail implementation lives on the legacy path `/integrations/google/gmail/*` via `Handler` (`google/handler.go:55-67`) — it has working sync (`gmail_read.go:32`, 295 lines), send (`gmail_send.go`), mark-read (`gmail_actions.go:10`), archive (`:37`), trash (`:64`). The frontend just isn't pointed at it.

So today, when you connect Gmail and click the email tab, you see an empty inbox forever. That's the "is the comms module alive?" answer for email: **no**.

---

## Tab-by-tab audit

### Calendar — ✅ working

**Frontend:** `frontend/src/routes/(app)/communication/calendar/+page.svelte` (566 lines, the heaviest of the three).
- Uses `$lib/api/calendar` (`getCalendarConnectionStatus`, `getCalendarAuthUrl`) and a base `api.getCalendarEvents`/`api.syncCalendar`.
- Calls **two** backend surfaces, mixed:
  - Local DB calendar at `/calendar/*` (`handlers/calendar.go`) for stats, upcoming, today, events CRUD.
  - Google calendar at `/integrations/google/calendar/*` for OAuth + sync.
- Views: week, day, month, agenda. Components in `lib/components/calendar/*`. Uses Svelte 5 runes (`$state`, `$derived`, `$effect`).

**Backend:**
- `internal/handlers/calendar.go` (~80 lines visible) + `calendar_engine_sync.go` for OptimalEngine hook.
- `internal/integrations/google/calendar.go` (407 lines) — full sync, create, delete.
- Engine sync wired (`calendar_engine_sync.go:14-47`): every Google or Microsoft calendar event created is pushed to the OptimalEngine knowledge graph as a Signal with module `ModuleCalendar` (`engine_sync.go:95`).

**Gaps even here:**
- Two backends speaking calendar (local `/calendar/*` and `/integrations/google/calendar/*`) with overlapping concerns. Stats endpoint is local-only — Google sync count comes via a different shape.
- `loadEvents()` swallows errors silently (line 190: `catch { events = [] }`). No retry, no toast.
- No Outlook surface here even though `microsoft/outlook_calendar.go` exists.
- "Sync" is manual-only. No background polling, no push from Google (would need watch channels).

### Email — 🟥 broken end-to-end (UI exists, backend stubbed)

**Frontend:** `communication/email/+page.svelte` (1,124 lines).
- Folders sidebar (inbox/sent/drafts/starred/archive/trash) — six folders, all routed to the same broken `getEmails({folder})` endpoint. Backend ignores `folder` (it's stubbed).
- Three-pane layout (folders / list / preview).
- Compose modal with To/Cc/Subject/Body. Reply/Forward populate compose state.
- HTML rendering via DOMPurify with tight allowlist (`+page.svelte:399`) — **good**, this is the one safety-aware piece.
- `getGmailStats()` is **also stubbed on the frontend** (`gmail.ts:155-163`) — returns hardcoded zeros. Sidebar unread count + "X emails synced" footer always show 0.
- Decorative-only: the `<Paperclip>` button (`+page.svelte:521`), the Archive button in preview actions (`:425`), drafts folder.

**Backend gaps (recap):**
- `ToolHandler` Gmail methods stubbed. `Handler` (legacy unified) has the real logic but isn't called.
- No engine sync for inbound emails — they would never reach OSA/agents even if sync worked.
- No `/stats` endpoint defined for Gmail in either path.

**Outlook (Microsoft) backend exists and is real:**
- `microsoft/handler.go:60-63` — `GET /mail/emails`, `GET /mail/emails/:id`, `POST /mail/send`, `POST /mail/sync`, all wired to `outlook_mail.go` (300 lines).
- `microsoft_mail_messages` table populated. Just not surfaced in comm UI.

### Channels — 🟨 working (Slack only) but thin

**Frontend:** `communication/channels/+page.svelte` (689 lines).
- Uses `any[]` for channels and messages (`+page.svelte:8-13`) — types not formalized.
- Slack-only. Connect → list → sync → view → send. No threads, no DMs surfaced separately, no reactions sending (only display).
- Decorative-only: search button, more-options button, attach button, message reactions UI (display only).
- Errors only `console.error` — no user-visible toast.
- `member_count` rendered without checking it exists.

**Backend (Slack):** `slack/handler.go` (255 lines), `messages.go` (279), `channels.go` (559) — full, working.
- OAuth, channels GET/sync, messages GET/POST/sync.
- Schema: `slack_channels`, `slack_messages`, `slack_oauth_tokens` (`schema.sql:498, 2484, 2518`).

**Gaps:**
- No Teams surface even though `microsoft_handler` exists for Teams via OneDrive provider; Teams fetcher wired into the engine adapters but **no UI route**.
- No Discord. The generic `channels` table exists (`schema.sql:2336`) but only Slack writes to it.
- No Slack engine sync — messages don't flow to OptimalEngine.
- No realtime — polling only. Slack has Events API and Socket Mode; neither is wired.
- Type `any` is a real bug surface — e.g., `message.reactions` (line 268) crashes if absent on some Slack message shapes.

---

## Architectural map (the actual end-to-end as it stands)

```
┌──────────────────────────────────────────────────────────────────────┐
│  FRONTEND  (SvelteKit / Svelte 5 runes)                              │
│                                                                      │
│  /communication                                                      │
│    ├─ +layout.svelte   (3 tabs: Calendar / Email / Channels)         │
│    └─ +page.svelte     (redirects to /communication/calendar)        │
│                                                                      │
│  /communication/calendar/+page.svelte                                │
│    └─ uses $lib/api/calendar + api.getCalendarEvents/syncCalendar    │
│                                                                      │
│  /communication/email/+page.svelte                                   │
│    └─ uses $lib/api/gmail (POINTS AT BROKEN BACKEND)                 │
│                                                                      │
│  /communication/channels/+page.svelte                                │
│    └─ uses api.getSlack* (works, but `any` types)                    │
└──────────────────────────────────────────────────────────────────────┘
                                  │  Vite proxy → :8001
                                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│  GO BACKEND (Gin)                                                    │
│                                                                      │
│  /api/calendar/*                        ← local DB calendar (works)  │
│    └─ handlers/calendar.go + calendar_engine_sync.go                 │
│                                                                      │
│  /api/integrations/google_calendar/*    ← tool-OAuth (works)         │
│    └─ google/tool_handler.go (calendar methods properly delegate)    │
│                                                                      │
│  /api/integrations/google_gmail/*       ← tool-OAuth (BROKEN)        │
│    └─ google/tool_handler.go            ← STUBBED / 501              │
│                                                                      │
│  /api/integrations/google/*             ← legacy unified (works)     │
│    └─ google/handler.go → gmail_*.go    ← real Gmail impl lives here │
│                                                                      │
│  /api/integrations/microsoft/*          ← Outlook + Teams (works,    │
│    └─ microsoft/handler.go                no UI surfaces it)         │
│                                                                      │
│  /api/integrations/slack/*              ← works                      │
│    └─ slack/handler.go → channels.go + messages.go                   │
│                                                                      │
│  IntegrationRouter.RegisterRoutes wires all of the above             │
│  (handlers/integrations_router.go)                                   │
│                                                                      │
│  ─────────── Engine wiring (handlers/routes_integrations.go) ────────│
│  • adapters.SetGmailFetcher(...)    ← engine PULLS from Gmail        │
│  • adapters.SetSlackFetcher(...)    ← engine PULLS from Slack        │
│  • OnEventCreated hook              ← BO PUSHES calendar events      │
│                                       only — nothing for emails/msgs│
└──────────────────────────────────────────────────────────────────────┘
                                  │
                ┌─────────────────┼─────────────────┐
                ▼                 ▼                 ▼
         ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
         │  PostgreSQL  │  │ OptimalEngine│  │ External APIs│
         │  emails,     │  │ (signals,    │  │ Gmail, Slack,│
         │  slack_*,    │  │  knowledge   │  │ Outlook,     │
         │  microsoft_* │  │  graph)      │  │ Calendar     │
         └──────────────┘  └──────────────┘  └──────────────┘
```

---

## The "alive" question — what does this module need to actually run?

Right now the comm module is three half-disconnected things. To make it the home for agent-driven communication BusinessOS promises in the README, four layers need to come together:

### 1. Fix the broken plumbing first (the unblocker, no scope expansion)

These are the sub-day fixes that turn "demo UI" into "actually works":

- **Gmail tool handler — wire it to the real services.** In `tool_handler.go:336-392`, replace the stubs with calls to `h.gmail.GetEmails`, `h.gmail.GetEmail`, `h.gmail.SendEmail`, `h.gmail.SyncEmails` — same pattern that calendar uses on `:239, :272, :296, :321`. Add the missing routes for read/archive/delete.
- **Or alternative:** point the frontend at the legacy `/integrations/google/gmail/*` and delete the tool-handler Gmail stubs. The "tool-OAuth scoping" benefit can be kept if the legacy provider is feature-flagged (it already is — `provider.HasFeature("gmail")`).
- **Stop lying about stats.** Add `GET /integrations/.../gmail/stats` returning real counts from the `emails` table. Wire `getGmailStats()` to it.
- **Type the Slack page.** Define `SlackChannel` and `SlackMessage` types and replace `any[]`. Adds zero behavior but prevents the next runtime crash.

### 2. Make the module polyglot (one tab, multiple providers)

The schema already supports it — `emails.provider` column exists, `microsoft_mail_messages` is a parallel table. The UI just doesn't.

- **Email tab → unified inbox.** Provider switcher (Gmail / Outlook), or merged view with a provider tag per row. Backend exposes a `GET /communication/inbox?providers=gmail,outlook` that unions `emails` + `microsoft_mail_messages` ordered by date.
- **Channels tab → Slack + Teams.** Microsoft handler already has `outlook_calendar.go`; add a Teams channel surface (Microsoft Graph supports it). Use the generic `channels` table (`schema.sql:2336`) as the destination so the UI can be provider-agnostic.

### 3. Wire it into OptimalEngine so agents can act

This is the part that makes it a *MIOSA template* and not a webmail clone.

Currently calendar events fire `OnEventCreated → engineSync.Enqueue(Signal{Module: ModuleCalendar})` (`calendar_engine_sync.go:36`). Nothing equivalent exists for emails or messages.

Add two hooks paralleling the calendar one:
- `OnEmailReceived` in `Gmail`/`Outlook` services — every saved email becomes a Signal with new `ModuleEmail` (add to `engine_sync.go:88-98`). Body = subject + snippet + body_text.
- `OnMessageReceived` in `Slack`/`Teams` — same shape, `ModuleMessage`.

Once those signals land in the engine, OSA can:
- Answer "what did Sarah email me about Q3?" via RAG
- Triage inbox via Signal Theory (the noise filter is already designed for this)
- Auto-draft replies using context from across modules (project, CRM, prior emails)

### 4. Make it real-time (the perception layer)

Polling every "Sync" click is the floor. To feel alive:
- **Gmail** — Push notifications via Pub/Sub watch (`users.watch`), fronted by an SSE channel from backend → frontend. The backend already has SSE infra in `internal/streaming/`.
- **Slack** — Events API or Socket Mode. Events API needs a public webhook; Socket Mode works locally. Either way, push into the `slack_messages` table and broadcast over SSE.
- **Outlook** — Microsoft Graph webhooks (subscriptions).

A single `/api/comms/stream` SSE endpoint multiplexing all three providers → frontend `EventSource` updates the lists in place. No more "click sync to see new mail."

### 5. Compose-side polish (the credibility layer)

- Attachments (multipart upload → Gmail/Outlook MIME).
- Drafts persistence (the `drafts` folder is purely cosmetic right now — `is_draft` column exists in `emails` but no API).
- Send-later / scheduled send.
- Rich text editor for body (you already have a block editor in `lib/components/editor`).
- Contact autocomplete from CRM `crm_contacts` + `microsoft_contacts` + `hubspot_contacts` — all three tables are in schema.

---

## Suggested order of work

If I were Roberto's reviewer, this is the order that compounds best:

1. **Day 1**: fix `ToolHandler` Gmail stubs + wire `markAsRead`/`archive`/`delete` routes + real `getGmailStats`. Email tab goes from broken to working.
2. **Day 2**: type the Slack page, surface real errors, fix the `any` types. Channels tab goes from fragile to solid.
3. **Day 3-4**: add `ModuleEmail` and `ModuleMessage` to engine sync. Now agents can read inbox/channels via RAG. **This is the single biggest "comes alive" moment.**
4. **Week 2**: unified inbox (Gmail + Outlook in the Email tab; Slack + Teams in Channels). Schema is ready; this is mostly a frontend + a union endpoint.
5. **Week 2-3**: realtime via SSE — Gmail push, Slack Events API, Outlook subscriptions.
6. **Week 3+**: compose polish (attachments, drafts, scheduling, rich text, contact autocomplete).

**Quick reality check before committing to a path**: is the goal "make the existing UI work" (steps 1-2, ~3 days) or "make BusinessOS the place agents live in your inbox" (steps 1-4, ~2 weeks)? The first is plumbing; the second is the actual product story.

---

## Reference: file inventory

### Frontend
- `frontend/src/routes/(app)/communication/+layout.svelte` — Hub shell, 3 tabs
- `frontend/src/routes/(app)/communication/+page.svelte` — Redirect to /calendar
- `frontend/src/routes/(app)/communication/calendar/+page.svelte` — 566 lines, working
- `frontend/src/routes/(app)/communication/email/+page.svelte` — 1,124 lines, broken backend
- `frontend/src/routes/(app)/communication/channels/+page.svelte` — 689 lines, Slack only
- `frontend/src/lib/api/gmail/{gmail.ts,types.ts,index.ts}`
- `frontend/src/lib/api/calendar/{calendar.ts,types.ts,index.ts}`
- `frontend/src/lib/api/index.ts:226-234` — Slack methods on the umbrella `api` object

### Backend
- `desktop/backend-go/internal/handlers/integrations_router.go` — wires all integration routes
- `desktop/backend-go/internal/handlers/routes_integrations.go` — engine adapter wiring + hooks
- `desktop/backend-go/internal/handlers/calendar.go` + `calendar_engine_sync.go`
- `desktop/backend-go/internal/integrations/google/handler.go` — legacy unified (works)
- `desktop/backend-go/internal/integrations/google/tool_handler.go` — new tool-OAuth (Gmail stubbed)
- `desktop/backend-go/internal/integrations/google/gmail_{read,send,actions,helpers,types}.go`
- `desktop/backend-go/internal/integrations/google/calendar.go` — 407 lines
- `desktop/backend-go/internal/integrations/microsoft/handler.go` + `outlook_mail.go` (300 lines) + `outlook_calendar.go` + `outlook_calendar_sync.go`
- `desktop/backend-go/internal/integrations/slack/handler.go` + `channels.go` (559) + `messages.go` (279) + `provider.go`
- `desktop/backend-go/internal/services/engine_sync.go` — Signal struct, Module enum, push-to-engine

### Database (in `internal/database/schema.sql`)
- `emails` (line 2287) — Gmail-shaped, supports `provider` column
- `slack_channels` (2484), `slack_messages` (2518), `slack_oauth_tokens` (498)
- `microsoft_mail_messages` (1924), `microsoft_calendar_events` (1970), `microsoft_contacts`
- `channels` (2336) — generic Slack/Discord/Teams table, currently Slack-only
