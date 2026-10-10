# OSA User Connection — How Users Talk to the Agent

> **The missing piece.** This documents the complete flow from user keystroke to agent response.
> How the interface works, what the user sees, how the system connects, and what's broken.
>
> Created: 2026-02-23
> Status: ACTIVE
> Related: [OSA System Status](./OSA-SYSTEM-STATUS.md) | [Interface Architecture](./OSA-INTERFACE-ARCHITECTURE.md) | [Production Roadmap](./MIOSA-PRODUCTION-ROADMAP.md)

---

## Two Message Paths: Reactive vs Proactive

A user message takes one of two fundamentally different paths depending on which layer is
handling it. Understanding this split is essential before reading the detailed flow below.

### Path 1 — Reactive (BusinessOS, open source, L1-L2)

This is the path documented in full detail below. The user types a message. The backend
receives it, classifies intent using the COT (Chain of Thought) engine and the 5-mode router,
selects a mode handler, and streams a response. The system is purely reactive: it does nothing
until the user initiates.

```
User types → POST /api/v2/chat/send → OSA Mode Router → mode handler → LLM → SSE response
```

**LLM provider:** Local Ollama, Groq, or Anthropic (configurable, no Miosa.ai required)
**Cost:** Free tier, open source

### Path 2 — Proactive (OSA engine, premium, L3-L5)

The proactive path inverts the flow. OSA monitors signals — calendar events, CRM activity,
message patterns, anomalies — and initiates outbound SSE streams to the frontend without the
user sending anything. When the user does respond, their reply enters the same 5-mode
classification pipeline, but the conversation was started by the system.

```
Signal detected → osa-engine/ evaluates → proactive trigger → SSE push to client
User responds → (same reactive path above)
```

**LLM provider:** MIOSA API (premium, Miosa.ai subscription required)
**Cost:** Premium tier, requires osa-engine/ which is NOT in this repository

The junction between these paths is `internal/orchestration/osa_orchestrator.go`.
See [ARCHITECTURE-SEPARATION.md](../ARCHITECTURE-SEPARATION.md) for the full boundary mapping.

---

## The Connection: User → Agent → LLM → Response

### Complete Flow (What Exists Today)

```
USER TYPES MESSAGE
    │
    ▼
┌─────────────────────────────────────────────────────┐
│ FRONTEND (SvelteKit)                                 │
│                                                      │
│  Chat Page (+page.svelte)                            │
│    │                                                 │
│    ▼                                                 │
│  chat.ts store → sendMessage()                       │
│    │  Builds SendMessageRequest:                     │
│    │  { message, conversation_id, workspace_id,      │
│    │    focus_mode, focus_options, model_override }   │
│    │                                                 │
│    ▼                                                 │
│  conversations.ts API client                         │
│    │  POST /api/chat/message                         │
│    │  Returns: ReadableStream (SSE tokens)           │
│    │                                                 │
│    ▼                                                 │
│  AssistantMessage.svelte renders tokens as they      │
│  arrive via ReadableStream reader                    │
└─────────────────────────────────────────────────────┘
    │
    │  HTTP POST with streaming response
    │
    ▼
┌─────────────────────────────────────────────────────┐
│ BACKEND (Go/Gin)                                     │
│                                                      │
│  POST /api/v2/chat/send                              │
│    │  AuthMiddleware → validate session               │
│    │                                                 │
│    ▼                                                 │
│  SendMessageV2() handler (chat_v2.go)                │
│    │  Extracts: message, conversation_id, model,     │
│    │    workspace_id, focus_mode, focus_options       │
│    │                                                 │
│    ▼                                                 │
│  OSA ORCHESTRATOR (osa_orchestrator.go)               │
│    │                                                 │
│    ├─→ Step 1: OSAModeRouter.ClassifyMode()          │
│    │   6-priority chain:                             │
│    │   1. Explicit mode (from UI dropdown)           │
│    │   2. BUILD detection (OSARouter pattern+LLM)    │
│    │   3. EXECUTE keywords (weighted scoring ≥0.7)   │
│    │   4. MAINTAIN keywords (weighted scoring ≥0.7)  │
│    │   5. ANALYZE keywords (weighted scoring ≥0.7)   │
│    │   6. Default → ASSIST (confidence 0.6)          │
│    │                                                 │
│    ├─→ Step 2: Route to mode handler                 │
│    │   BUILD   → ModuleProtection → PACT/BMAD       │
│    │   ASSIST  → SmartIntentRouter → 5 agent types   │
│    │   ANALYZE → Analyst agent + analytics prompt    │
│    │   EXECUTE → SORX skill matching → TriggerSkill  │
│    │   MAINTAIN → Health stub (Phase 2)              │
│    │                                                 │
│    ├─→ Step 3: Agent dispatched with enriched prompt │
│    │   BaseAgentV2 builds system prompt:             │
│    │   - Role-specific instructions                  │
│    │   - Conversation history                        │
│    │   - Workspace context (via TieredContextService)│
│    │   - Module/project/client data                  │
│    │                                                 │
│    └─→ Step 4: LLM call                             │
│        AnthropicService.StreamChat()                 │
│        - Model: claude-sonnet-4-20250514 (default)    │
│        - Max tokens: 8192                            │
│        - Temperature: 0.7                            │
│        - Streaming: token-by-token SSE               │
│                                                      │
└─────────────────────────────────────────────────────┘
    │
    │  HTTPS to Anthropic API (or configured provider)
    │
    ▼
┌─────────────────────────────────────────────────────┐
│ LLM PROVIDER                                         │
│                                                      │
│  Anthropic API (api.anthropic.com/v1/messages)       │
│  OR Ollama Cloud (configurable via ANTHROPIC_BASE_URL│
│     — Sprint 1 Phase A task, not yet wired)          │
│  OR Groq, OpenAI, xAI, Ollama Local                 │
│     (via AI_PROVIDER env var, LLMService factory)    │
│                                                      │
│  6 providers supported:                              │
│  - anthropic (default)                               │
│  - ollama_cloud                                      │
│  - groq                                              │
│  - openai                                            │
│  - xai                                               │
│  - ollama_local                                      │
│                                                      │
│  Returns: streaming token chunks                     │
└─────────────────────────────────────────────────────┘
    │
    │  Token chunks streamed back
    │
    ▼
┌─────────────────────────────────────────────────────┐
│ RESPONSE DISPLAYED                                   │
│                                                      │
│  Backend streams tokens via SSE/ReadableStream       │
│  Frontend AssistantMessage.svelte renders:            │
│  - Markdown formatted text                           │
│  - Code blocks with syntax highlighting              │
│  - Thinking blocks (if COT enabled)                  │
│  - Tool use blocks (future)                          │
│                                                      │
│  User sees: conversational response in chat UI       │
└─────────────────────────────────────────────────────┘
```

---

## What the User Sees Today

### The Chat Interface (Primary OSA Surface)

The main way users interact with OSA is through the chat module at `/chat`.

**Current UI elements:**
- **Message input** — text area with file attachment support
- **Focus mode selector** — 6 modes (Research, Deep Research, Analyze, Write, Build, General)
- **@agent mentions** — `@analysis`, `@planning`, `@document` to force-route to specific agents
- **Slash commands** — `/build`, `/analyze`, etc. for command shortcuts
- **Streaming responses** — token-by-token display as the LLM generates
- **Thinking blocks** — Chain-of-Thought reasoning displayed when enabled
- **Conversation sidebar** — history of past conversations

**What's NOT there yet:**
- No OSA mode badge (user can't see which mode was selected)
- No confidence indicator (user can't see how sure the classification was)
- No SORX decision cards (no approve/reject UI for skill execution)
- No skill catalog (no visual way to browse available integrations)
- No module protection violation UI (no feedback when BUILD is blocked)

### The App Generation Interface (BUILD Mode Surface)

When a user asks OSA to build something, a separate flow kicks in via `/generated-apps/`.

**Current flow:**
1. User opens CreateAppModal from the OSA gallery or chat
2. Fills in: name, description, features, template type
3. Submits → `POST /api/workspaces/:id/apps/generate-osa`
4. Backend queues the generation request
5. Frontend opens EventSource SSE on `/api/osa/apps/generate/:queueItemId/stream`
6. AgentProgressPanel shows real-time progress (agent steps, file generation)
7. When complete, user sees generated files and can "Open in Editor"

**What's broken (Phase A blockers):**
- FK violation: files don't persist because `osa_generated_apps` row is never created before `osa_generated_files` INSERT
- Prompt overwrite: rich generation prompt is replaced by short user description
- Wrong SSE route: frontend uses `appId` but backend expects `queueItemId`

### The SORX Skill Interface (EXECUTE Mode Surface)

Users trigger skills by typing natural language in chat. There is NO separate SORX UI today.

**Current flow:**
1. User types "sync my Gmail contacts" in chat
2. OSA classifies as EXECUTE mode (score ≥ 0.7)
3. Pattern matcher finds "gmail" → skill ID "gmail.sync"
4. SORX engine triggers the skill
5. Result streamed back as chat response

**What's missing:**
- No visual skill catalog (user has to know what to ask for)
- No approval cards (SORX temperature control exists but no UI for decisions)
- No skill execution history UI
- No credential connection UI (integrations must be pre-configured)

---

## System Prompt Architecture

### How Prompts Are Built

Each agent type has a different system prompt. The prompt is built in layers:

```
Layer 1: ROLE DEFINITION
  "You are a [document/project/client/analyst/task] specialist for BusinessOS."

Layer 2: WORKSPACE CONTEXT (from TieredContextService)
  - Workspace name and description
  - User's role and permissions
  - Installed modules and their status
  - Recent conversation history

Layer 3: MODE-SPECIFIC INSTRUCTIONS
  BUILD:    "Generate code following Go handler→service→repository pattern..."
  ASSIST:   "Help the user with their request using available OS context..."
  ANALYZE:  "Query business data and generate actionable insights..."
  EXECUTE:  "Match the user's request to available SORX skills..."
  MAINTAIN: "Check system health and module versions..."

Layer 4: TOOL DEFINITIONS (future — not yet implemented)
  Available functions the LLM can call
```

### OSA Prompt Templates (Existing but NOT Wired)

The `OSAPromptBuilder` service exists at `internal/services/osa_prompt_builder.go` with embedded YAML templates in `internal/services/templates/osa/`:

| Template | Purpose |
|----------|---------|
| `crm-app-generation.yaml` | CRM module generation prompt |
| `feature-addition.yaml` | Adding features to existing modules |
| `bug-fix.yaml` | Bug fix generation prompt |
| (more planned) | Module-specific prompts |

**CRITICAL ISSUE:** The `OSAPromptBuilder` is injected into the handler but **never called** in the generation pipeline. The `AppGenerationOrchestrator` builds its own prompts instead. This means the carefully crafted YAML templates are unused. **Pedro Agent F's task** is to wire this.

### LLM Configuration

| Setting | Value | Source |
|---------|-------|--------|
| Default model | `claude-sonnet-4-20250514` | `config.go` → `ANTHROPIC_MODEL` env |
| Max tokens | 8192 | Hardcoded in `anthropic.go` |
| Temperature | 0.7 | Hardcoded in `anthropic.go` |
| Top P | 0.9 | Hardcoded in `anthropic.go` |
| Timeout | 120 seconds | Hardcoded in `anthropic.go` |
| Extended thinking | Supported | `StreamChatWithThinking()` method |
| API base URL | `https://api.anthropic.com` | Hardcoded 5x (Sprint 1 fix target) |

---

## What's Working vs What's Broken

### WORKING (Verified)

| Component | Evidence |
|-----------|----------|
| 5-Mode classification engine | 56 tests pass, 3 benchmarks |
| Module protection validation | 48 tests pass, 4 pattern types |
| OSA orchestrator with 5 mode handlers | 36 tests pass |
| SORX engine (30 actions, 21 skills) | Engine builds, skills registered |
| SmartIntentRouter (4-layer classification) | Existing, 515 lines |
| Agent system (6 types) | All agents functional |
| SSE streaming infrastructure | BuildEventBus exists |
| Chat UI with streaming | Frontend renders tokens |
| Focus mode selector | 6 modes in dropdown |
| Conversation management | Full CRUD + history |

### BROKEN (Blocks E2E)

| Component | Bug | Fix Owner |
|-----------|-----|-----------|
| File persistence | FK violation — `osa_generated_apps` not created before `osa_generated_files` | Roberto Agent A |
| Rich prompts | Variable shadowing overwrites prompt with short description | Roberto Agent A |
| SSE route | Frontend uses `appId`, backend expects `queueItemId` | Javaris Agent B |
| LLM URL | 5 hardcoded Anthropic URLs, can't switch providers | Roberto Agent D |
| Prompt templates | OSAPromptBuilder injected but never called | Pedro Agent F |
| Event drops | BuildEventBus silently drops events under backpressure | Pedro Agent D |
| File save | No PUT endpoint for Monaco editor saves | Pedro Agent A |

### NOT BUILT (Architecture Only)

| Component | Sprint | Why It Matters |
|-----------|--------|---------------|
| Frontend mode badge | Sprint 2 | Users can't see which mode OSA picked |
| Decision cards | Sprint 2 | SORX can't ask for human approval |
| Skill catalog | Sprint 3 | Non-tech users can't discover integrations |
| Sandbox edit flow | Sprint 2 | Can't fork→edit→preview→apply modules |
| CARRIER bridge | Sprint 5 | Can't reach Elixir SorxMain for Tier 3-4 |
| Per-OS LLM config | Sprint 2 | Each OS locked to same LLM provider |

---

## For the Team: How to Test the Connection

### Quick Smoke Test (Local Dev)

```bash
# 1. Start backend
cd desktop/backend-go
go build ./cmd/server && ./bin/server

# 2. Start frontend
cd frontend
npm run dev

# 3. Open browser
# http://localhost:5173/chat

# 4. Type a message
# "What can you help me with?"
# → Should stream a response (ASSIST mode)

# 5. Test BUILD mode
# "Build me a todo app"
# → Should attempt generation (will hit FK bug currently)

# 6. Test EXECUTE mode
# "Sync my Gmail contacts"
# → Should attempt SORX skill (needs Gmail integration configured)
```

### What to Look For

- **Token streaming**: Response appears word-by-word, not all at once
- **Mode classification**: Check backend logs for `"classified mode"` entries
- **Agent routing**: Check logs for which agent was dispatched
- **Error handling**: Bad requests should return clear error messages, not 500s

---

**Last Updated:** 2026-02-26
**Status:** Documents current connection flow. Phase A blockers prevent E2E testing.
**Architecture note:** Two message paths documented above — reactive (open source, local LLM) and proactive (premium, MIOSA API). This document covers the reactive path. See ARCHITECTURE-SEPARATION.md for the full split.
