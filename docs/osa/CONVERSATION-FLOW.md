# OSA Conversation Flows

> What the user sees and what happens behind the scenes for each mode.

---

## Reactive vs Proactive Conversation Flows

Conversation flows in this codebase differ fundamentally depending on whether you are in the reactive (open source) or proactive (premium) layer.

**Reactive flows (BusinessOS / ASSIST mode — L1-L2):**
The user sends a message. OSA classifies it. If no proactive mode matches, the message falls through to ASSIST — the SmartIntentRouter delegates to a specialist agent (project, client, document, task, analyst), the agent queries context, and a response streams back. The user always initiates. The agent only responds. This is the open-source baseline.

**Proactive flows (OSA premium / BUILD, EXECUTE, MAINTAIN — L3-L5):**
The interaction is more complex. The agent may propose a plan, show checkboxes, run a background job, surface a completion card, or ask for approval before executing. In some cases (Sprint 5 proactive worker), OSA initiates the conversation — the user does not need to ask first. These flows are premium (Miosa.ai) and are documented in the sections below.

> **See:** `docs/osa/ARCHITECTURE-SEPARATION.md` for the full reactive/proactive split and open-source boundary.

---

## BUILD Mode — "Build me a feedback form module"

### What the User Sees

```
┌──────────────────────────────────────────────────────────┐
│ User: Build me a feedback form module with star           │
│       ratings and comments                                │
│                                                          │
│ ┌── OSA [BUILD mode detected • confidence 0.92] ──────┐ │
│ │                                                       │ │
│ │ 🔨 I'll generate a feedback form module for you.     │ │
│ │                                                       │ │
│ │ This will include:                                    │ │
│ │ • Database table: feedback_submissions                │ │
│ │ • API endpoints: POST/GET /api/feedback               │ │
│ │ • UI component: FeedbackForm.svelte                   │ │
│ │ • Star rating widget (1-5 stars)                      │ │
│ │ • Text comment field                                  │ │
│ │                                                       │ │
│ │ Generating...                                         │ │
│ │ ████████████████████░░░░░░░░ 65%                     │ │
│ │                                                       │ │
│ │ ✅ migration/091_feedback.sql                          │ │
│ │ ✅ internal/handlers/feedback.go                       │ │
│ │ ✅ internal/services/feedback.go                       │ │
│ │ 🔄 components/FeedbackForm.svelte (generating...)     │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
│                                                          │
│ [After generation completes:]                             │
│                                                          │
│ ┌── Generated Module Preview ──────────────────────────┐ │
│ │ 📁 feedback-module/                                   │ │
│ │ ├── manifest.json                                     │ │
│ │ ├── migration/091_feedback.sql                        │ │
│ │ ├── internal/handlers/feedback.go                     │ │
│ │ ├── internal/services/feedback.go                     │ │
│ │ └── components/FeedbackForm.svelte                    │ │
│ │                                                       │ │
│ │ [ Open in Editor ]  [ Install Module ]  [ Discard ]   │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### What Happens Behind the Scenes

```
1. POST /api/v2/chat/send { message: "Build me a feedback form..." }
2. OSA Mode Router: BUILD detected (pattern "build" + "module", confidence 0.92)
3. Module Protection: No conflicts with protected manifests
4. OSA Orchestrator → handleBuildMode()
5. PACT/BMAD generation pipeline:
   a. Plan: Determine files needed, architecture
   b. Generate: LLM generates each file (streamed via SSE)
   c. Validate: Check generated SQL is safe, code compiles conceptually
6. Files saved to osa_generated_apps + osa_generated_files tables
7. SSE events stream to frontend:
   - { event: "progress", data: { file: "feedback.go", status: "generating" } }
   - { event: "file_complete", data: { file: "feedback.go", content: "..." } }
   - { event: "generation_complete", data: { appId: "abc-123" } }
8. User clicks "Open in Editor" → Monaco editor loads generated files
9. User edits (optional) → saves via PUT /api/workspaces/:id/apps/:id/files
10. User clicks "Install Module" → Module Protection validates → applies to workspace
```

---

## ASSIST Mode — "What's the status of Project Alpha?"

### What the User Sees

```
┌──────────────────────────────────────────────────────────┐
│ User: What's the status of Project Alpha?                 │
│                                                          │
│ ┌── OSA [ASSIST mode] ────────────────────────────────┐ │
│ │                                                       │ │
│ │ **Project Alpha** is currently in progress.           │ │
│ │                                                       │ │
│ │ • **Status:** Active (started Jan 15)                 │ │
│ │ • **Completed tasks:** 12 of 18 (67%)                │ │
│ │ • **Next milestone:** MVP demo (Feb 28)               │ │
│ │ • **Blockers:** Waiting on design review from Sarah    │ │
│ │ • **Budget:** $12,500 of $20,000 used (63%)          │ │
│ │                                                       │ │
│ │ Would you like me to analyze the timeline risks,      │ │
│ │ or should I create a task to follow up with Sarah?    │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### What Happens Behind the Scenes

```
1. POST /api/v2/chat/send { message: "What's the status of Project Alpha?" }
2. OSA Mode Router: No keywords for BUILD/EXECUTE/ANALYZE/MAINTAIN
   → Default ASSIST (confidence 0.6)
3. SmartIntentRouter v2 classifies → Project Agent
4. Project Agent:
   a. Query: SELECT * FROM projects WHERE name ILIKE '%alpha%'
   b. Query: SELECT count(*) FROM tasks WHERE project_id = ? AND status = 'done'
   c. Query: SELECT * FROM tasks WHERE project_id = ? AND status = 'blocked'
   d. Query: SELECT SUM(amount) FROM expenses WHERE project_id = ?
5. Context assembled → LLM generates natural language response
6. Response streamed to frontend
7. Agent suggests mode switches (analyze timeline, execute task creation)
```

---

## EXECUTE Mode — "Sync my Gmail contacts"

### What the User Sees

```
┌──────────────────────────────────────────────────────────┐
│ User: Sync my Gmail contacts with the CRM                 │
│                                                          │
│ ┌── OSA [EXECUTE mode • gmail.sync] ──────────────────┐ │
│ │                                                       │ │
│ │ 🔄 I'll sync your Gmail contacts into the CRM.       │ │
│ │                                                       │ │
│ │ ┌── Approval Required ────────────────────────────┐  │ │
│ │ │ This will:                                       │  │ │
│ │ │ • Read contacts from your Gmail account          │  │ │
│ │ │ • Create new CRM entries for unknown contacts    │  │ │
│ │ │ • Update existing entries with new info          │  │ │
│ │ │                                                  │  │ │
│ │ │ [ Approve ]    [ Cancel ]                        │  │ │
│ │ └──────────────────────────────────────────────────┘  │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
│                                                          │
│ [After user clicks Approve:]                              │
│                                                          │
│ ┌── OSA [executing...] ──────────────────────────────┐  │
│ │                                                       │ │
│ │ Step 1/3: Reading Gmail contacts...                   │ │
│ │ ✅ Found 247 contacts                                  │ │
│ │                                                       │ │
│ │ Step 2/3: Importing to CRM...                         │ │
│ │ ✅ 12 new contacts created                             │ │
│ │ ✅ 3 existing contacts updated                         │ │
│ │ ⏭️ 232 contacts already up to date                    │ │
│ │                                                       │ │
│ │ Step 3/3: Summary                                     │ │
│ │ ✅ Sync complete! 247 contacts processed.              │ │
│ │    12 new • 3 updated • 232 unchanged                 │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### What Happens Behind the Scenes

```
1. POST /api/v2/chat/send { message: "Sync my Gmail contacts..." }
2. OSA Mode Router: EXECUTE (keywords: "sync" 0.5 + "gmail" 0.2 + "contacts" 0.2 = 0.9)
3. Skill matcher: "gmail" + "sync" → "gmail.sync" skill
4. Integration check: User has Gmail OAuth token? → Yes
5. Temperature check: WARM → write operation → show decision card
6. SSE event: { event: "decision_required", data: { skill: "gmail.sync", actions: [...] } }
7. Frontend shows decision card → user clicks "Approve"
8. POST /api/sorx/execute { skill: "gmail.sync", approved: true }
9. SORX Engine executes:
   Step 1: gmail.list_contacts(token) → 247 contacts
     SSE: { event: "step_progress", data: { step: 1, status: "complete", count: 247 } }
   Step 2: businessos.upsert_clients(contacts) → 12 new, 3 updated
     SSE: { event: "step_progress", data: { step: 2, status: "complete", new: 12, updated: 3 } }
   Step 3: agent.summarize(results) → "Sync complete! 247 contacts..."
     SSE: { event: "step_progress", data: { step: 3, status: "complete", summary: "..." } }
10. Execution logged in skill_executions table
```

---

## ANALYZE Mode — "Show me revenue by client this quarter"

### What the User Sees

```
┌──────────────────────────────────────────────────────────┐
│ User: Show me revenue by client this quarter              │
│                                                          │
│ ┌── OSA [ANALYZE mode] ──────────────────────────────┐  │
│ │                                                       │ │
│ │ 📊 Revenue by Client — Q1 2026                       │ │
│ │                                                       │ │
│ │ Client          │ Revenue   │ Change     │ Share      │ │
│ │ ─────────────── │ ───────── │ ────────── │ ────────── │ │
│ │ Acme Corp       │ $45,200   │ ↑ 40% YoY  │ 35%       │ │
│ │ TechCorp        │ $28,100   │ ↑ 12% YoY  │ 22%       │ │
│ │ StartupX        │ $22,500   │ NEW         │ 17%       │ │
│ │ GlobalCo        │ $18,900   │ ↓ 5% YoY   │ 15%       │ │
│ │ Others (3)      │ $14,200   │ ↑ 8% YoY   │ 11%       │ │
│ │ ─────────────── │ ───────── │            │           │ │
│ │ **Total**       │ $128,900  │ ↑ 23% YoY  │           │ │
│ │                                                       │ │
│ │ **Key Insights:**                                     │ │
│ │ 1. Acme Corp drives 35% of revenue — high dependency │ │
│ │ 2. StartupX is a promising new client ($22.5K in Q1) │ │
│ │ 3. GlobalCo declining — review relationship           │ │
│ │                                                       │ │
│ │ **Recommendation:** Diversify — reduce Acme           │ │
│ │ dependency below 30% by growing other accounts.       │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### What Happens Behind the Scenes

```
1. POST /api/v2/chat/send { message: "Show me revenue by client..." }
2. OSA Mode Router: ANALYZE (keywords: "revenue" 0.25 + "client" 0.2 + "quarter" 0.2 = 0.65)
   Hmm, below 0.7 threshold — LLM fallback confirms ANALYZE
3. Data source identification: invoices table + clients table
4. Build parameterized SQL:
   SELECT c.name, SUM(i.amount) as revenue
   FROM invoices i JOIN clients c ON c.id = i.client_id
   WHERE i.created_at >= date_trunc('quarter', CURRENT_DATE)
   GROUP BY c.name ORDER BY revenue DESC
5. Execute query → results
6. LLM prompt: "Given this revenue data: [...], provide business insights"
7. LLM generates table + insights + recommendation
8. Response streamed to frontend
```

---

## MAINTAIN Mode — "Check if anything needs updating"

### What the User Sees

```
┌──────────────────────────────────────────────────────────┐
│ User: Check if anything needs updating                    │
│                                                          │
│ ┌── OSA [MAINTAIN mode] ─────────────────────────────┐  │
│ │                                                       │ │
│ │ 🔧 System Health Report                              │ │
│ │                                                       │ │
│ │ Module Status:                                        │ │
│ │ ✅ CRM Module — v1.2.0 (up to date)                   │ │
│ │ ✅ Invoicing Module — v2.0.1 (up to date)             │ │
│ │ ⚠️ Feedback Module — v1.0.0 → v1.1.0 available       │ │
│ │    Changelog: Bug fix for star rating on mobile       │ │
│ │                                                       │ │
│ │ System Health:                                        │ │
│ │ ✅ Database: 45 active connections (of 100 max)        │ │
│ │ ✅ Redis: 128MB used (of 512MB)                        │ │
│ │ ✅ API: p99 latency 450ms (target <2s)                │ │
│ │ ✅ No security vulnerabilities detected                │ │
│ │                                                       │ │
│ │ Would you like to update the Feedback Module           │ │
│ │ to v1.1.0?                                            │ │
│ │                                                       │ │
│ │ [ Update Feedback Module ]    [ Skip for Now ]        │ │
│ │                                                       │ │
│ └───────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

---

## Mode Switching

Users can switch modes mid-conversation. OSA suggests switches when appropriate:

```
User: "What clients do we have?" → ASSIST mode
OSA: "You have 47 active clients. Acme Corp is the largest..."
OSA: "Would you like me to analyze revenue trends? I can switch to ANALYZE mode."

User: "Yes, analyze revenue" → ANALYZE mode (explicit switch)
OSA: [switches to ANALYZE mode, generates data table]
```

The mode badge in the chat UI always shows the current mode, and users can manually override via the mode selector dropdown.

---

## BUILD Mode — Full Natural Language Flow

### The Vision: No Commands, Just Conversation

The user never types `/build` or clicks a "Generate" button. They just describe what they want in natural language, and OSA handles the rest. This is like Claude Code's plan mode — the agent shows a plan with checkboxes, the user confirms, and work happens in the background.

### Complete Flow: Chat → Plan → Confirm → Generate → Preview → Install

```
┌─────────────────────────────────────────────────────────────────┐
│ STEP 1: NATURAL LANGUAGE INPUT                                   │
│                                                                  │
│ User types in chat (no special commands):                        │
│ "I need a way for clients to leave feedback with star ratings"   │
│                                                                  │
│ OSA Mode Router classifies → BUILD (confidence 0.91)             │
│ User sees: [BUILD mode detected] badge in chat                   │
├─────────────────────────────────────────────────────────────────┤
│ STEP 2: OSA SHOWS PLAN (checkboxes like Claude Code)             │
│                                                                  │
│ ┌──────────────────────────────────────────────────────────┐    │
│ │ 🔨 I'll build a Feedback Module. Here's what I'll create: │    │
│ │                                                           │    │
│ │ ☐ Database: feedback_submissions table                    │    │
│ │   (id, user_id, rating 1-5, comment, anonymous, created) │    │
│ │                                                           │    │
│ │ ☐ Backend: Go API endpoints                               │    │
│ │   POST /api/feedback — submit feedback                    │    │
│ │   GET  /api/feedback — list feedback (with pagination)    │    │
│ │                                                           │    │
│ │ ☐ Frontend: FeedbackForm.svelte                           │    │
│ │   Star rating widget (1-5 stars, hover preview)           │    │
│ │   Comment text area                                       │    │
│ │   Anonymous toggle                                        │    │
│ │                                                           │    │
│ │ ☐ Tests: Handler + service test stubs                     │    │
│ │                                                           │    │
│ │ Want to modify anything before I start?                    │    │
│ │                                                           │    │
│ │ [ Generate ]    [ Edit Plan ]    [ Cancel ]               │    │
│ └──────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│ STEP 3: USER CONFIRMS (or modifies)                              │
│                                                                  │
│ Option A: User clicks "Generate" → proceed immediately           │
│ Option B: User types "add an email notification when submitted"  │
│   → OSA updates plan, shows again with new item                  │
│ Option C: User clicks "Cancel" → plan discarded                  │
├─────────────────────────────────────────────────────────────────┤
│ STEP 4: BACKGROUND GENERATION (user keeps chatting)              │
│                                                                  │
│ Generation starts in background queue:                           │
│ POST /api/osa/apps/generate-osa → returns { queue_item_id }     │
│                                                                  │
│ User sees progress in chat:                                      │
│ ┌──────────────────────────────────────────┐                    │
│ │ 🔨 Generating Feedback Module...          │                    │
│ │ ████████████████░░░░░░░░ 65%             │                    │
│ │ ✅ feedback_submissions.sql               │                    │
│ │ ✅ handlers/feedback.go                   │                    │
│ │ 🔄 FeedbackForm.svelte (generating...)   │                    │
│ └──────────────────────────────────────────┘                    │
│                                                                  │
│ Meanwhile, user can keep chatting:                               │
│ User: "While that builds, what's the status of Project Alpha?"   │
│ OSA: [switches to ASSIST mode, answers normally]                 │
├─────────────────────────────────────────────────────────────────┤
│ STEP 5: COMPLETION NOTIFICATION                                  │
│                                                                  │
│ SSE event: { event: "generation_complete", data: { appId } }    │
│                                                                  │
│ ┌──────────────────────────────────────────┐                    │
│ │ ✅ Feedback Module is ready!              │                    │
│ │                                          │                    │
│ │ Generated 4 files:                       │                    │
│ │ • feedback_submissions.sql (migration)   │                    │
│ │ • handlers/feedback.go (API)             │                    │
│ │ • services/feedback.go (logic)           │                    │
│ │ • FeedbackForm.svelte (UI)               │                    │
│ │                                          │                    │
│ │ [ Preview ]  [ View Code ]  [ Install ]  │                    │
│ └──────────────────────────────────────────┘                    │
├─────────────────────────────────────────────────────────────────┤
│ STEP 6: SANDBOX PREVIEW                                          │
│                                                                  │
│ User clicks "Preview"                                            │
│ → App renders in sandboxed iframe (read-only, no write access)   │
│ → User sees the actual UI running                                │
│ → Can interact with the form, click buttons, see layout          │
│                                                                  │
│ User clicks "View Code"                                          │
│ → Monaco editor opens with all generated files                   │
│ → User can edit code before installing                           │
│ → Changes saved back to osa_generated_files                      │
├─────────────────────────────────────────────────────────────────┤
│ STEP 7: INSTALL TO OS                                            │
│                                                                  │
│ User clicks "Install"                                            │
│ → Module Protection validates manifest                           │
│ → Confirmation dialog: "Install Feedback Module?"                │
│ → On confirm:                                                    │
│   • App status → 'installed' in DB                               │
│   • Module routes registered in workspace config                 │
│   • Icon appears on desktop (2D window manager)                  │
│   • Nav item appears in sidebar (dynamic modules section)        │
│   • App is LIVE in their OS instance                             │
│                                                                  │
│ OSA: "Feedback Module installed! You'll find it on your          │
│       desktop and in the sidebar. Your clients can start         │
│       leaving feedback immediately."                             │
└─────────────────────────────────────────────────────────────────┘
```

### Background Queue — How It Works

The generation queue lets users keep chatting while apps build:

```
Timeline:
────────────────────────────────────────────────────────

T+0s   User: "Build me a feedback form"
       → POST /api/osa/apps/generate-osa
       → Returns { queue_item_id: "abc-123" }
       → Frontend subscribes to SSE: /osa/apps/generate/abc-123/stream

T+5s   User: "What's the status of Project Alpha?"
       → Normal ASSIST mode (separate request, different mode)
       → OSA answers immediately, generation continues in background

T+15s  SSE event: { status: "generating", progress: 30, file: "migration.sql" }
       → Progress bar updates in chat

T+45s  SSE event: { status: "generating", progress: 65, file: "FeedbackForm.svelte" }
       → Progress bar updates

T+60s  SSE event: { status: "complete", app_id: "abc-123" }
       → Completion card appears in chat with Preview/Install buttons

T+75s  User clicks "Install"
       → Module installed to workspace
       → Icon added to desktop + sidebar
```

**Backend components:**
- `osa_queue_worker.go` — polls queue, picks up jobs, runs agent pipeline
- `app_generation_orchestrator.go` — orchestrates agent execution, saves files
- `build_event_bus.go` — in-memory pub/sub for SSE events
- `osa_apps.go:StreamAppGeneration()` — SSE endpoint, subscribes to event bus

**Frontend components:**
- `generatedAppsStore.ts` — manages app state, SSE subscription, auto-reconnect
- `osa.ts` — API client (generateApp, streamAppGeneration, cancelGeneration)

### Where Apps Display

Generated apps appear in TWO places after install:

**1. Sidebar (Static → Dynamic)**

Currently the sidebar (`+layout.svelte:91-177`) has 17 hardcoded nav items. Sprint 3 will add a dynamic "My Apps" section below the static items that queries installed apps from the database.

**2. Desktop (Already Working)**

The 2D desktop (`/window/+page.svelte`) is a full macOS-like window manager with icons, windows, dock, and spotlight search. It ALREADY handles `osa-app-*` modules:

```svelte
// Lines 934-953 of /window/+page.svelte
{:else if win.module.startsWith('osa-app-')}
    {@const appId = win.module.replace('osa-app-', '')}
    {@const deployedApp = $deployedAppsStore.apps.find(app => app.id === appId)}
    {#if deployedApp && deployedApp.status === 'running'}
        <iframe
            src={deployedApp.url}
            title={deployedApp.name}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
        ></iframe>
    {/if}
```

After install:
- Desktop icon appears with app name + icon
- Double-click → opens window with app running in sandboxed iframe
- App can be pinned to dock for quick access
- `deployedAppsStore.startDiscovery(workspaceId)` keeps app list updated

---

**Last Updated:** 2026-02-26
