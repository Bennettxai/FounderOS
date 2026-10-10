# OSA Interface Architecture — How Users Talk to the 5-Mode System

**Date:** 2026-02-23
**Status:** Analysis + Proposal
**Depends on:** OSA 5-Mode Router (DONE), Frontend mode selector (NOT DONE)
**Last Updated:** 2026-02-26

---

## Interface Modes: Reactive (Free) vs Proactive (Premium)

The chat interface serves two fundamentally different modes of operation depending on whether
the user is on the open-source/free tier or the Miosa.ai premium tier.

**Reactive mode (local LLM, free tier)**
The interface is a conventional request-response chat. The user sends a message, OSA
classifies the intent into one of the 5 modes, dispatches to the appropriate handler, and
streams a response. The LLM provider can be a local Ollama instance. No Miosa.ai subscription
is required. All described in this document applies to this mode.

**Proactive mode (MIOSA API, premium tier)**
The interface gains ambient intelligence. OSA initiates conversations — surfacing anomalies,
flagging deadlines, triggering briefings — without the user sending a message first. The
frontend must handle server-initiated SSE streams and present them appropriately. This mode
requires the `osa-engine/` proactive layer and a Miosa.ai API subscription.

The frontend `FocusModeSelector` and the underlying `SendMessageRequest` struct are the same
for both modes. The difference is entirely in what happens on the backend after routing:
reactive mode dispatches a handler and returns; proactive mode maintains persistent context
and monitors signals between sessions.

See [ARCHITECTURE-SEPARATION.md](../ARCHITECTURE-SEPARATION.md) for the boundary details.

---

## Current State: What Exists

### 1. Chat Interface (PRIMARY — `/chat`)

**Path:** `frontend/src/routes/(app)/chat/+page.svelte`

This is the main interface. Users type messages. The backend auto-classifies into BUILD/ASSIST/ANALYZE/EXECUTE/MAINTAIN.

**Components already built:**
- `FocusModeSelector.svelte` — 6 focus modes (Research, Deep Research, Analyze, Write, Build, General)
- `ChatInput.svelte` — message input with file attachments
- `AssistantMessage.svelte` — streaming response with thinking blocks
- `@agent` mentions — `@analysis`, `@planning`, `@document`
- `/slash` commands — command autocomplete

**What's missing:** No way to explicitly select an OSA mode. The focus modes are close but don't pass `osa_mode` to the backend.

### 2. Terminal Interface (`/terminal`)

**Path:** `frontend/src/routes/(app)/terminal/+page.svelte`

WebSocket-based Docker terminal. Currently used for code execution, not for OSA interaction.

**NOT suitable as primary OSA interface** — terminal is for technical users running commands, not for business users asking "sync my contacts."

### 3. Popup Chat (`/popup-chat`)

Lightweight chat modal. Same backend, smaller UI surface.

### 4. Generated Apps Page (`/generated-apps/`)

SSE-based progress tracking during BUILD mode. Already works with the pipeline (once FK bug is fixed).

### 5. SORX HTTP API (`/api/sorx/*`)

Direct API for skill execution. Currently no frontend UI calls this — it's for programmatic access.

---

## The Interface Problem

The 5-mode system is a backend routing layer. Users don't know or care about "modes." They care about getting things done:

| What the user says | What they mean | OSA Mode |
|---|---|---|
| "Build me an invoicing module" | Create new code | BUILD |
| "How many deals closed this quarter?" | Get an answer | ASSIST or ANALYZE |
| "Show me a sales dashboard" | Analytics view | ANALYZE |
| "Sync my Gmail contacts" | Run an automation | EXECUTE |
| "Send a Slack message to #team" | Trigger an action | EXECUTE |
| "Check if the system is healthy" | Status check | MAINTAIN |
| "What's my schedule today?" | Get information | ASSIST or EXECUTE |

**The mode detection mostly works** via keyword scoring. But there are ambiguous cases where the wrong mode fires.

---

## Proposed Interface Architecture

### For Technical Users (Developers, Admins)

**Use the Chat module** with:
1. **Auto-detected mode badge** — shows which mode was selected (BUILD, ASSIST, etc.) with confidence score
2. **Mode override dropdown** — lets user force a mode if auto-detect is wrong
3. **Slash commands** — `/build`, `/execute`, `/analyze`, `/maintain` to force mode
4. **Terminal integration** — for BUILD mode, show terminal output alongside chat

**Implementation:** Extend `FocusModeSelector.svelte` to map to OSA modes:
```
Existing "Build" focus   -> osa_mode: "BUILD"
Existing "Analyze" focus -> osa_mode: "ANALYZE"
Existing "General" focus -> osa_mode: "" (auto-detect)
NEW "Execute" focus      -> osa_mode: "EXECUTE"
NEW "Maintain" focus     -> osa_mode: "MAINTAIN"
```

Frontend passes `osa_mode` via the request's `focus_options` map. Backend already reads it from `FocusModeOpts["osa_mode"]`.

### For Non-Technical Users (Business Users)

**Use the Chat module** with:
1. **Guided skill catalog** — visual cards showing available EXECUTE skills ("Sync Gmail", "Send Slack Message", "Generate Report")
2. **Natural language only** — no slash commands, no mode selector
3. **Decision UI** — when SORX pauses for human-in-the-loop, show decision cards with options
4. **Progress indicators** — SSE streaming shows what's happening ("Connecting to Gmail...", "Found 47 contacts...", "Creating tasks...")

**Implementation:** New component `SkillCatalog.svelte` that:
- Calls `GET /api/sorx/skills` to list available skills
- Shows grouped cards (Email, Calendar, CRM, Tasks)
- Clicking a card pre-fills the chat with the right prompt
- Shows required integrations (grayed out if not connected)

### Interface Flow

```
                    +------------------+
                    |   Chat Module    |
                    |   (Primary UI)   |
                    +--------+---------+
                             |
                    User sends message
                             |
            +----------------+----------------+
            |                |                |
     Focus mode set?   @agent mention?   /slash command?
            |                |                |
            v                v                v
    FocusModeOpts      Agent routing    Command routing
    osa_mode="BUILD"   (existing)       /execute -> EXECUTE
            |                |                |
            +----------------+----------------+
                             |
                    OSA Mode Router
                    (backend, auto-detect)
                             |
           +---------+------+------+---------+
           |         |      |      |         |
         BUILD    ASSIST  ANALYZE  EXECUTE  MAINTAIN
           |         |      |      |         |
    PACT/BMAD    Smart   Analyst  SORX     Health
    Orchestrator  Intent  Agent   Engine   Stub
                  Router
```

### What the Frontend Needs to Send

```typescript
// In SendMessageRequest (existing)
interface SendMessageRequest {
  message: string;
  conversation_id?: string;
  focus_mode?: string;       // "build", "analyze", "execute", etc.
  focus_options?: {          // Key-value pairs
    osa_mode?: string;       // Explicit mode override
    depth?: string;          // For ANALYZE
    skill_id?: string;       // For EXECUTE (direct skill trigger)
  };
}
```

### What the Frontend Receives Back (SSE)

```typescript
// Mode classification event (NEW — needs frontend handler)
{ type: "thinking", data: { step: "classifying", content: "Mode: EXECUTE (0.85)", agent: "osa_mode_router" }}

// EXECUTE mode progress
{ type: "thinking", data: { step: "executing", content: "Running skill: gmail.sync", agent: "sorx" }}
{ type: "tool_call", data: { tool_name: "gmail.list_messages", status: "calling" }}
{ type: "tool_result", data: { tool_name: "gmail.list_messages", status: "success", result: "47 contacts found" }}

// Decision required (SORX human-in-the-loop)
{ type: "blocks", data: { type: "decision", question: "Import all 47 contacts?", options: ["Yes", "No", "Select specific"] }}

// BUILD mode progress (existing)
{ type: "artifact_start", data: { type: "code", title: "schema.sql" }}
{ type: "token", content: "CREATE TABLE invoices..." }
{ type: "artifact_complete", data: { ... }}
```

---

## Terminal Module — When to Use It

The terminal IS useful but for different purposes:

| Use case | Interface | Why |
|---|---|---|
| Asking OSA to build something | Chat | Natural language, SSE progress |
| Running generated code | Terminal | Docker container, real shell |
| Debugging a deployment | Terminal | Direct container access |
| Managing SORX skills | Chat (admin mode) | Guided UI better than CLI |
| Viewing analytics | Chat + Dashboard | ANALYZE mode returns structured data |
| Running automations | Chat | EXECUTE mode with progress UI |

**The terminal becomes relevant in BUILD mode's "edit and iterate" flow:**
1. User asks "Build an invoicing module" -> Chat (BUILD mode)
2. PACT/BMAD generates code -> Chat shows progress via SSE
3. User opens generated app in Monaco editor -> Generated Apps page
4. User wants to test the generated code -> Terminal (Docker sandbox)
5. User sees bug, asks "Fix the validation on line 42" -> Chat (BUILD mode)

---

## Implementation Priority

| Phase | What | Interface | Effort |
|---|---|---|---|
| **Phase 1** (NOW) | Fix FK + prompt + routes blockers | No UI change needed | Backend only |
| **Phase 2** (Next) | Add mode badge to chat UI | `ModeIndicator.svelte` | 2-3 hours |
| **Phase 2** (Next) | Map focus modes to OSA modes | `FocusModeSelector.svelte` edit | 1-2 hours |
| **Phase 2** (Next) | Decision UI for SORX human-in-the-loop | `DecisionCard.svelte` | 3-4 hours |
| **Phase 3** | Skill catalog for non-tech users | `SkillCatalog.svelte` | 6-8 hours |
| **Phase 3** | EXECUTE mode progress streaming | Extend `AssistantMessage.svelte` | 3-4 hours |
| **Phase 4** | Terminal integration with BUILD mode | Wire sandbox to generated apps | Already partially exists |

---

## Key Files

### Backend (Done)
- `internal/orchestration/osa_modes.go` — Mode router
- `internal/orchestration/osa_orchestrator.go` — Mode dispatch
- `internal/services/module_protection.go` — BUILD safety
- `internal/sorx/engine.go` — SORX execution
- `internal/handlers/sorx.go` — SORX HTTP API

### Frontend (Needs Work)
- `frontend/src/lib/components/chat/FocusModeSelector.svelte` — Add OSA mode mapping
- `frontend/src/lib/components/chat/focusModes.ts` — Define OSA mode options
- `frontend/src/routes/(app)/chat/+page.svelte` — Wire mode indicator
- `frontend/src/lib/components/chat/AssistantMessage.svelte` — Handle decision events

### SORX Skills (Available Now)
- 10+ integration skills (Gmail, HubSpot, Linear, Slack, Notion, Calendar)
- 15+ command skills (analyze, summarize, brainstorm, report, email draft)
- 30+ action handlers (real API calls to external services)
- Decision workflow (human-in-the-loop pausing)
