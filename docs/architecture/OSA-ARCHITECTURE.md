# OSA Architecture — Your Own Claude Code Inside Your OS

> **How OSA (Optimal System Agent) integrates with CustomOS.**
> OSA is not a chatbot feature. It's the operating system's brain — like having Claude Code
> with full access to your entire system. This document defines how it's built, where it runs,
> and what it has access to.
>
> Created: 2026-02-23
> Status: ACTIVE — Partially Implemented (Phase B+C complete)
> Related: [Platform Master](./MIOSA-PLATFORM-MASTER.md) | [Implementation Plan](./OSA-IMPLEMENTATION-PLAN.md) | [System Status](./OSA-SYSTEM-STATUS.md)
>
> ### Implementation Reality (2026-02-23)
> ```
> Section 2 (Built-In Architecture):     ✅ CONFIRMED — OSA runs inside CustomOS backend
> Section 3 (Access Model):              ✅ CONFIRMED — Direct pgxpool + Redis + SORX
> Section 4 (Three Layers):              ✅ LAYER 1 BUILT — Mode router + classification
>                                        ✅ LAYER 2 PARTIAL — 5 mode handlers exist, handlers delegate to existing agents
>                                        ✅ LAYER 3 EXISTS — Database, Redis, SORX, streaming all present
> Section 5 (Code Mapping):              ✅ UPDATED — See "Implementation Status" table below
> Section 6 (Core vs Module):            ✅ BUILT — CustomModuleService (814 lines), ModuleInstallationService, 15 API endpoints
> Section 7 (Request Flow):              ✅ MOSTLY ACCURATE — Flow works, mode router in place
> Section 8 (Desktop Experience):        ✅ EXISTS — Desktop shell, dock, windows all functional
> Section 9 (Module Backend):            ✅ BUILT — Dynamic module install/uninstall with action registry
> Section 10 (Module Frontend):          📋 ARCHITECTURE ONLY — Dynamic navigation not built
> Section 11 (SORX Integration):         ✅ BUILT — SORX engine works, Tier 1-4 (Tier 3-4 via CARRIER bridge to Optimal)
> Section 12 (Cross-OS):                 ✅ BUILT — CARRIER bridge implemented (`internal/carrier/`)
> Section 13 (Future Scaling):           📋 ARCHITECTURE ONLY — Currently Phase 1 (built-in)
> ```

---

## Dual Architecture: Reactive vs Proactive

> **This is the most important architectural distinction in the codebase.** OSA spans two
> fundamentally different execution models. Understanding which layer you are working in
> determines everything about how code should be written and where it should live.

### The Two Layers

| Layer | System | Autonomy | Distribution | Status |
|-------|--------|----------|-------------|--------|
| **Reactive** | BusinessOS (this repo) | L1-L2 | Open source | Shipped |
| **Proactive** | Optimal (Miosa-osa/Optimal) | L3-L5 | Premium — Miosa.ai | Built — Elixir/OTP |

**Reactive (BusinessOS):** Responds to user messages. User types → intent classified → agent
dispatched → response streamed. The system does nothing until a user acts. All five OSA modes
in their current form are reactive — they classify intent and dispatch to handlers. This layer
is open source and forms the encoding surface: the interface through which users communicate
intent to the system.

**Proactive (OSA):** Acts on behalf of users without waiting to be asked. Monitors signals,
detects patterns, initiates workflows, surfaces insights before the user knows to request them.
This is the premium intelligence layer, distributed as Miosa.ai. It requires L3-L5 autonomy
(shared decision-making through full autonomous operation) and is not included in the open
source release.

### Codebase Boundaries

```
BOS (open source — reactive, L1-L2)
  internal/orchestration/osa_modes.go        ← Mode router: classifies intent
  internal/orchestration/osa_orchestrator.go ← Mode dispatch: routes to handlers
  internal/services/module_protection.go     ← BUILD mode safety
  internal/sorx/engine.go                    ← EXECUTE mode: skill execution
  internal/agents/                           ← ASSIST/ANALYZE handlers
  internal/carrier/                          ← CARRIER AMQP bridge connecting to Optimal
    messages.go                              ← Full MessageContext propagation
    proactive_consumer.go                    ← Proactive command consumer
    registration.go                          ← Registration + heartbeat

Optimal (Miosa-osa/Optimal — Elixir/OTP)    ← BUILT
  Proactive intelligence layer (L3-L5)
  Contains STEERSMAN (unified cybernetic reasoning)
  Contains HOMEOSTAT (VSM viability monitor)
  Monitors signals, initiates workflows
  Runs autonomously between user sessions
  Requires MIOSA API subscription

nano_sorx (Elixir/OTP)                       ← Reference implementation
  Proactive patterns in portable YAML skills
  Open source reference for the proactive model
  Not the production runtime

Junction (reactive ↔ proactive handoff)
  internal/orchestration/osa_orchestrator.go ← The seam between both worlds
  internal/sorx/carrier_routing.go           ← Tier-based routing (Tier 3-4 → CARRIER → Optimal)
```

**See also:** [ARCHITECTURE-SEPARATION.md](../ARCHITECTURE-SEPARATION.md) for the complete
file-by-file mapping of what belongs to each layer and the extraction plan.

### SORX and nano_sorx

**SORX** is the premium skill engine. It is NOT open sourced. The Go SORX engine
(`internal/sorx/`) included in this repository provides Tier 1-2 deterministic and
structured-AI execution. Tier 3-4 reasoning (MCTS, Boardroom, Critic) routes through
the CARRIER bridge (`internal/carrier/` + `internal/sorx/carrier_routing.go`) to
Optimal (Miosa-osa/Optimal) — a separate Elixir/OTP service that is also premium.
The CARRIER bridge is **implemented**: tier-based routing in `carrier_routing.go`
sends Tier 3-4 requests over AMQP to Optimal, which contains Engine.MCTS,
Engine.Boardroom, Engine.Critic, and STEERSMAN. If CARRIER or Optimal is unavailable,
the Go engine handles locally with a direct LLM call as fallback.

**nano_sorx** is the Elixir/OTP reference implementation of proactive skill patterns.
It expresses skills as portable YAML definitions and demonstrates how proactive workflows
can be structured. It is the open-source alternative pattern — not a replacement for the
production SORX runtime, but a reference for contributors building on top of the platform.

---

## Table of Contents

1. [The Claude Code Analogy](#1-the-claude-code-analogy)
2. [Architecture Decision: Built-In, Not External](#2-architecture-decision-built-in-not-external)
3. [What OSA Has Access To](#3-what-osa-has-access-to)
4. [The Three Layers](#4-the-three-layers)
5. [How OSA Maps to Existing Code](#5-how-osa-maps-to-existing-code)
6. [CustomOS Core vs Module Layer](#6-customos-core-vs-module-layer)
7. [The Request Flow — User Message to OSA Action](#7-the-request-flow--user-message-to-osa-action)
8. [The Desktop Experience](#8-the-desktop-experience)
9. [How Modules Change the Backend](#9-how-modules-change-the-backend)
10. [How Modules Change the Frontend](#10-how-modules-change-the-frontend)
11. [Skills & SORX — OSA's Hands](#11-skills--sorx--osas-hands)
12. [Cross-OS Context — OSA Reaching Beyond](#12-cross-os-context--osa-reaching-beyond)
13. [Signal Theory: 7-Layer Implementation](#13-signal-theory-7-layer-implementation-live)
14. [Future: OSA as Standalone Service](#14-future-osa-as-standalone-service)

---

## 1. The Claude Code Analogy

Think about how Claude Code works:

```
CLAUDE CODE                              OSA
─────────────────────────────────────────────────────────────
Runs on your machine                     Runs inside your OS instance
Has access to your filesystem            Has access to your database
Can read/write any file                  Can read/write any module's data
Can run terminal commands                Can execute SORX skills
Can edit code                            Can build/modify modules
Can search your codebase                 Can search your OS data
Can dispatch sub-agents                  Can delegate to specialized agents
Has conversation context                 Has conversation + OS context
Remembers across sessions (memory)       Has memory hierarchy
Works in your terminal                   Works in the OS chat interface
You can ask it anything                  Users can ask it anything
It figures out what tool to use          It figures out which mode to use
```

**The key insight:** Claude Code is not "part of" your computer's OS. It's a powerful agent that has **full access** to your system. OSA works the same way — it's an agent with full access to the CustomOS.

But unlike Claude Code (which is external), OSA runs **inside** the OS for a critical reason: **speed and direct access**. OSA needs to query the database, execute skills, modify modules, and stream responses — all with zero network overhead.

---

## 2. Architecture Decision: Built-In, Not External

### The Options We Considered

```
OPTION A: OSA as External Service (Like Claude Code)
┌──────────┐     HTTP/WS      ┌──────────┐
│   OSA    │ ◄──────────────► │ CustomOS │
│ Service  │                  │ Backend  │
└──────────┘                  └──────────┘
Separate container            Separate container

Pros: Independent scaling, independent updates
Cons: Network latency on EVERY action, auth complexity, 2x containers cost

OPTION B: OSA Built Into CustomOS (What We're Doing)
┌─────────────────────────────────┐
│         CustomOS Backend        │
│                                 │
│  ┌───────────┐  ┌───────────┐  │
│  │    OSA    │  │  Business  │  │
│  │  Engine   │──│  Logic     │  │
│  │           │  │  (modules) │  │
│  └───────────┘  └───────────┘  │
│         │              │        │
│         ▼              ▼        │
│  ┌─────────────────────────┐   │
│  │     Database + Redis     │   │
│  └─────────────────────────┘   │
└─────────────────────────────────┘
Same container, direct access

Pros: Zero latency, direct DB access, shared context, simpler auth
Cons: Coupled (but that's fine — OSA IS the OS)

OPTION C: Hybrid — OSA Core Built In, LLM Calls External (ACTUAL APPROACH)
┌─────────────────────────────────┐
│         CustomOS Backend        │
│                                 │
│  ┌───────────────────────────┐  │
│  │        OSA Engine         │  │
│  │  ┌──────┐  ┌──────────┐  │  │
│  │  │Router│  │ Executor  │  │  │
│  │  │(mode │  │ (actions) │  │  │
│  │  │select│  │           │  │  │
│  │  └──┬───┘  └─────┬────┘  │  │
│  │     │             │       │  │
│  │     ▼             ▼       │  │
│  │  ┌──────────────────┐     │  │
│  │  │ Context Manager  │     │  │
│  │  │ (DB, modules,    │     │  │
│  │  │  memory, skills) │     │  │
│  │  └────────┬─────────┘     │  │
│  └───────────┼───────────────┘  │
│              │                   │
│  ┌───────────▼───────────────┐  │
│  │  Database + Redis + SORX  │  │
│  └───────────────────────────┘  │
└──────────────┬──────────────────┘
               │ API calls (LLM only)
               ▼
   ┌─────────────────────┐
   │  LLM Providers      │
   │  (Anthropic, Groq,  │
   │   Ollama, etc.)     │
   └─────────────────────┘
```

### Why Built-In Is Right

1. **OSA IS the OS** — It's not an add-on. It's the brain. Separating it creates artificial boundaries.

2. **Direct database access** — OSA needs to query contacts, projects, tasks, modules — all directly. No API round-trips.

3. **SORX skill execution** — OSA triggers SORX skills that run in the same process. Shared execution context.

4. **Module inspection** — OSA needs to know what modules are installed, what schemas exist, what operations are available. Direct Go struct access.

5. **Streaming** — SSE streaming from LLM → user works best when OSA is in the same process as the HTTP handler.

6. **Cost** — One container per OS instance, not two. At scale (10,000 OS instances), this halves infrastructure cost.

**The only external call is to LLM providers** (Anthropic, Groq, etc.) — and that's already how the current agent system works.

---

## 3. What OSA Has Access To

OSA has the **same access level as a system administrator** within its OS instance:

### Full Read Access

| System | What OSA Can Read | How |
|--------|------------------|-----|
| **Database** | All tables in the OS instance | Direct `pgxpool` queries |
| **Module registry** | Installed modules, configs, schemas | `custom_modules` table |
| **Module data** | All data in all module tables | SQL queries against module tables |
| **Conversations** | All conversation history for this OS | `conversations` + `messages` tables |
| **Memories** | Workspace memories, shared knowledge | `workspace_memories` table |
| **User profiles** | Team members, roles, permissions | `user` + `team_members` + `workspace_roles` |
| **Integrations** | Connected services, OAuth tokens | `integration_credentials` table |
| **Module versions** | Version history for all modules | `custom_module_versions` table |
| **Core manifest** | What's protected, what's customizable | `os_instances.core_module_manifest` |
| **OS config** | Feature flags, settings, template | `os_instances.settings` |
| **SORX skills** | Available skills, execution history | `sorx_skills` + `sorx_executions` |

### Full Write Access (With Protection)

| System | What OSA Can Write | Protection |
|--------|-------------------|------------|
| **Module data** | CRUD on any module table | Standard DB constraints |
| **Custom modules** | Create, modify, version modules | **Module Protection Service** validates against core manifest |
| **Conversations** | Save messages, update metadata | None — OSA manages its own conversations |
| **Memories** | Create, update workspace memories | Workspace scope only |
| **SORX executions** | Trigger skill executions | Skill availability + user permissions |
| **Module configs** | Enable/disable features | Feature flag validation |
| **Sandbox** | Fork modules to sandbox containers | Container isolation |

### What OSA CANNOT Do

| Action | Why Not |
|--------|---------|
| Remove protected core module operations | Module Protection Service blocks it |
| Remove protected schema columns | Core manifest enforcement |
| Remove protected UI routes | Manifest validation |
| Access other OS instances | Cross-OS requires explicit grants |
| Access platform-level data | Tenant/billing data is in MIOSA DB, not OS DB |
| Modify its own system prompt | System prompt is frozen per OS template |
| Bypass user permissions | OSA respects RBAC — won't show admin data to members |

---

## 4. The Three Layers

```
┌─────────────────────────────────────────────────────────┐
│                    LAYER 1: OSA BRAIN                   │
│                                                         │
│  Mode Router → Classifies user intent into 5 modes     │
│  Context Manager → Assembles relevant context          │
│  LLM Interface → Calls AI providers for reasoning      │
│  Response Formatter → Structures output for UI         │
│                                                         │
│  This is the "thinking" layer. Pure intelligence.       │
├─────────────────────────────────────────────────────────┤
│                    LAYER 2: OSA HANDS                   │
│                                                         │
│  Module Builder → Creates/modifies modules (BUILD)     │
│  Data Querier → Queries any table (ANALYZE)            │
│  Skill Executor → Triggers SORX skills (EXECUTE)      │
│  Version Manager → Manages module versions (MAINTAIN)  │
│  Knowledge Engine → Answers with OS context (ASSIST)   │
│  Sandbox Controller → Forks, validates, deploys        │
│                                                         │
│  This is the "doing" layer. Takes actions.             │
├─────────────────────────────────────────────────────────┤
│                    LAYER 3: OS PLATFORM                 │
│                                                         │
│  Database (pgxpool) → Direct SQL access                │
│  Redis (cache) → Session, pub/sub, caching             │
│  SORX Engine → Skill registration + execution          │
│  Container Manager → Docker sandbox deployment         │
│  Auth System → BetterAuth sessions                     │
│  SSE Streaming → Real-time responses to UI             │
│  Module System → custom_modules CRUD + versioning      │
│                                                         │
│  This is the "infrastructure" layer. Always there.     │
└─────────────────────────────────────────────────────────┘
```

### Layer 1: OSA Brain (New Code)

```go
// internal/agents/osa_mode_router.go
type OSAModeRouter struct {
    classifier *ModeClassifier     // 3-layer classification
    handlers   map[OSAMode]ModeHandler
    logger     *slog.Logger
}

type OSAMode string
const (
    ModeBuild    OSAMode = "BUILD"
    ModeAssist   OSAMode = "ASSIST"
    ModeAnalyze  OSAMode = "ANALYZE"
    ModeExecute  OSAMode = "EXECUTE"
    ModeMaintain OSAMode = "MAINTAIN"
)

func (r *OSAModeRouter) Route(ctx context.Context, msg string, osCtx *OSContext) (*ModeResult, error) {
    // 1. Classify the message
    mode, confidence := r.classifier.Classify(msg, osCtx)

    // 2. Get the handler for this mode
    handler := r.handlers[mode]

    // 3. Execute
    return handler.Handle(ctx, msg, osCtx)
}
```

### Layer 2: OSA Hands (New Code)

```go
// Each mode has a handler that takes actions

// BUILD handler
type BuildModeHandler struct {
    sandbox    *SandboxEditFlow
    protection *ModuleProtectionService
    modules    *CustomModuleService
}

// ANALYZE handler
type AnalyzeModeHandler struct {
    db      *database.Queries
    llm     services.LLMService
}

// EXECUTE handler
type ExecuteModeHandler struct {
    sorx    *sorx.Engine
    matcher *SkillMatcher
}

// MAINTAIN handler
type MaintainModeHandler struct {
    versions  *ModuleVersionService
    rollback  *RollbackService
}

// ASSIST handler
type AssistModeHandler struct {
    memory    *MemoryHierarchyService
    knowledge *KnowledgeService
    llm       services.LLMService
}
```

### Layer 3: OS Platform (Existing Code — Unchanged)

Everything that exists today in the BOS codebase. OSA's handlers use these services directly.

---

## 5. How OSA Maps to Existing Code

The current BOS agent system already does most of what OSA needs. OSA is a **wrapper + upgrade**, not a rewrite.

### What Exists → What It Becomes

```
EXISTING CODE                        OSA EQUIVALENT
──────────────────────────────────────────────────────────────
SmartIntentRouter (6 agent types)  → OSA Mode Router (5 modes)
  Pattern matching (regex)           Same patterns, new mode categories
  Semantic signals                   Same approach
  LLM fallback                      Same fallback

AgentRegistryV2 (6 agent types)    → Mode Handler Registry
  OrchestratorV2                     → Handles multi-mode requests
  DocumentAgentV2                    → Part of BUILD mode
  ProjectAgentV2                     → Part of ASSIST mode
  TaskAgentV2                        → Part of ASSIST mode
  ClientAgentV2                      → Part of ASSIST mode
  AnalystAgentV2                     → Part of ANALYZE mode

BaseAgentV2                         → ModeHandler base
  buildMessages()                    Same message building
  Run() streaming                    Same streaming
  Context injection                  Same + module context

LLMService (Anthropic/Groq/Ollama) → Same (unchanged)
  StreamChat()                       Same streaming
  ChatComplete()                     Same completion

SORX Engine                         → EXECUTE mode handler
  ExecuteSkill()                     Called by ExecuteModeHandler
  MatchSkill() (NEW)                 NL → skill matching

TieredContextService                → OSA Context Manager (extended)
  Projects, tasks, clients            Same + module context
  Memory injection                    Same
  Role context                        Same

Streaming + Artifact Detection      → Same (unchanged)
  SSE events                          Same event types
  ArtifactDetector                    Same detection

OSA Integration (osa_api.go)        → BUILD mode handler
  HandleGenerateApp()                 Wrapped by BuildModeHandler
  ProcessWithOSARouting()             Replaced by OSA Mode Router
```

### What's New — Implementation Status (Updated 2026-02-23)

> **Note:** File paths differ from original plan. Code was placed in `internal/orchestration/`
> and `internal/services/` rather than `internal/agents/` — this is the correct location since
> mode routing is orchestration logic, not agent logic.

| Component | Planned File | Actual File | Status |
|-----------|-------------|-------------|--------|
| `OSAModeRouter` | `internal/agents/osa_mode_router.go` | `internal/orchestration/osa_modes.go` | **BUILT** (349 lines, 56 tests + 3 benchmarks) |
| `ModuleProtectionService` | `internal/services/module_protection.go` | `internal/services/module_protection.go` | **BUILT** (306 lines, 48 tests) |
| Mode Handlers (all 5) | `internal/agents/osa_*_handler.go` | `internal/orchestration/osa_orchestrator.go` | **BUILT** (5 handlers in orchestrator, ~550 lines) |
| SORX→OSA Wiring | (not specified) | `cmd/server/main.go` + `handlers.go` + `chat_v2.go` | **BUILT** (init + config struct + setters) |
| Tenant/Org Foundation | (not specified) | `supabase/migrations/090_tenant_org_foundation.sql` | **BUILT** (155 lines, organizations + org_members) |
| `SandboxEditFlow` | `internal/services/sandbox_edit_flow.go` | — | **NOT BUILT** (depends on FK fix, Phase D) |
| `OSAContextManager` | `internal/agents/osa_context.go` | — | **NOT BUILT** (Phase F, uses existing TieredContextService for now) |
| `SkillMatcher` | `internal/sorx/skill_matcher.go` | — | **NOT BUILT** (SORX engine does basic pattern matching, full NL matching is Phase E) |
| `CrossOSBridge` | `internal/services/cross_os_bridge.go` | — | **NOT BUILT** (Phase 2, needs tenant isolation first) |

### What Was Built But Not In Original Plan

| Component | File | What It Does |
|-----------|------|-------------|
| 6-Priority Classification Chain | `osa_modes.go` | Explicit → OSARouter → EXECUTE → MAINTAIN → ANALYZE → ASSIST fallback |
| Weighted Keyword Scoring | `osa_modes.go` | Primary patterns (0.5), secondary (0.2-0.25), threshold ≥ 0.7 |
| OSAOrchestratorConfig | `osa_orchestrator.go` | Full config struct with OSAClient, Registry, LLM, SorxEngine, Protection |
| Nil-safety Guards | `osa_orchestrator.go` | Prevents panics when optional services not injected |
| Handler Wiring | `handlers.go` | `SetSorxEngine()` + `SetProtectionService()` setter methods |
| NewOSAOrchestratorWithConfig | `chat_v2.go` | Config-based construction replacing inline init |

### Architecture vs Code: Key Differences

```
ARCHITECTURE DOC SAID                    CODE ACTUALLY DOES
────────────────────────────────────────────────────────────────
Mode handlers in internal/agents/     →  Mode handlers in internal/orchestration/
  (separate files per mode)               (5 methods on OSAOrchestrator struct)

3-layer classification                →  6-priority classification chain
  (regex → semantic → LLM fallback)      (explicit → OSARouter → keywords → default)

SkillMatcher for NL → skill           →  SORX engine.MatchSkill() with basic patterns
  (custom semantic matching)              (11 built-in skill patterns, threshold matching)

AnalyzeModeHandler queries DB         →  Analyst agent via SmartIntentRouter
  (direct SQL generation)                 (LLM-based analysis, not direct SQL yet)

OSA as external client killed         →  osa.Client still exists (internal/integrations/osa/)
  (all local dispatch)                    (kept for future cross-OS remote calls)
```

---

## 6. CustomOS Core vs Module Layer

### What Is "Core" (Always There, Never Changes)

```
CORE INFRASTRUCTURE — The engine that runs every OS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Backend:
  ├── Auth system (BetterAuth sessions, RBAC)
  ├── Workspace management
  ├── OSA Engine (mode router, all 5 mode handlers)
  ├── LLM service layer (Anthropic, Groq, Ollama)
  ├── SORX skill engine (execution framework)
  ├── Module system (install, configure, version, protect)
  ├── Sandbox system (Docker fork, validate, deploy)
  ├── Memory hierarchy (workspace, private, shared)
  ├── SSE streaming
  ├── Redis caching
  ├── Container management
  ├── File storage
  └── Background jobs

Frontend:
  ├── Desktop shell (window manager, dock, menu bar, spotlight)
  ├── Chat interface (conversations, streaming, artifacts, thinking)
  ├── Dashboard (widgets, analytics)
  ├── Settings
  ├── Team management
  ├── Module manager UI (install, configure, remove)
  ├── Workspace switcher
  └── OS Selector (MIOSA platform integration)

Database:
  ├── user, session (auth)
  ├── workspaces, team_members, workspace_roles (RBAC)
  ├── conversations, messages (chat)
  ├── workspace_memories (knowledge)
  ├── custom_modules, custom_module_versions (module system)
  ├── os_instances (OS config + manifest)
  ├── contexts (tiered context)
  └── reasoning_templates (thinking config)
```

### What Is "Module" (Varies Per OS Instance)

```
MODULE LAYER — What changes based on template and user choices
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Each module registers:
  ├── Backend handlers (CRUD endpoints)
  ├── Backend services (business logic)
  ├── Database migrations (module tables)
  ├── Frontend routes (pages)
  ├── Frontend components (UI)
  ├── SORX skills (integrations)
  └── Module manifest (protection rules)

Example: CRM Module
  Backend:
    ├── handler/clients.go → CRUD for clients
    ├── handler/deals.go → Pipeline management
    ├── service/crm_service.go → Business logic
    └── repository/client_repository.go → Data access

  Database:
    ├── clients table
    ├── deals table
    ├── deal_stages table
    └── contact_activities table

  Frontend:
    ├── routes/(app)/clients/+page.svelte
    ├── routes/(app)/clients/[id]/+page.svelte
    ├── routes/(app)/crm/+page.svelte
    ├── components/crm/ClientCard.svelte
    └── components/crm/DealPipeline.svelte

  SORX Skills:
    ├── crm.import_contacts
    ├── crm.export_contacts
    └── crm.enrich_contacts

  Manifest:
    ├── protected_operations: [contacts.list, contacts.create, ...]
    ├── protected_schemas: [{table: "clients", required_columns: [...]}]
    └── protected_ui_routes: ["/clients", "/clients/[id]"]
```

### How Modules Register

```go
// Module registration at startup
type ModuleRegistration struct {
    ID          string
    Name        string
    Version     string
    Routes      []RouteRegistration     // HTTP routes to register
    Migrations  []MigrationRegistration // DB migrations to run
    Skills      []SkillRegistration     // SORX skills to register
    Manifest    *ModuleManifest         // Protection rules
    FrontendDir string                  // Path to frontend assets
}

// At OS startup:
func (os *CustomOS) RegisterModules(installed []ModuleRegistration) {
    for _, mod := range installed {
        // 1. Register routes
        os.router.RegisterModuleRoutes(mod.Routes)

        // 2. Run migrations (if new version)
        os.db.RunModuleMigrations(mod.Migrations)

        // 3. Register SORX skills
        os.sorx.RegisterModuleSkills(mod.Skills)

        // 4. Load manifest for protection
        os.protection.LoadManifest(mod.Manifest)
    }
}
```

---

## 7. The Request Flow — User Message to OSA Action

### Full Flow Diagram

```
USER types: "Show me revenue by client this quarter"
    │
    ▼
┌────────────────────────────────────────────────────┐
│ FRONTEND (Chat Interface)                          │
│                                                    │
│ 1. User types message in chat input               │
│ 2. POST /api/v2/chat/send                         │
│    { message, workspace_id, conversation_id }      │
│ 3. Opens SSE stream for response                  │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ BACKEND (Chat Handler)                             │
│                                                    │
│ 4. AuthMiddleware → validate session               │
│ 5. Load conversation history                       │
│ 6. Build context (projects, tasks, memories)       │
│ 7. Hand off to OSA Mode Router                     │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ OSA MODE ROUTER (Layer 1: Brain)                   │
│                                                    │
│ 8. Classify message:                               │
│    Layer 1 (regex): "revenue" → ANALYZE signal     │
│    Layer 2 (semantic): "by client" → data query    │
│    Result: ANALYZE mode (confidence: 0.92)         │
│                                                    │
│ 9. Select handler: AnalyzeModeHandler              │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ ANALYZE MODE HANDLER (Layer 2: Hands)              │
│                                                    │
│ 10. Parse intent: "revenue by client, this quarter"│
│ 11. Check available modules → CRM module installed │
│ 12. Build SQL query:                               │
│     SELECT c.name, SUM(d.value) as revenue         │
│     FROM clients c JOIN deals d ON ...             │
│     WHERE d.closed_at >= quarter_start             │
│     GROUP BY c.name ORDER BY revenue DESC          │
│ 13. Execute query → get results                    │
│ 14. Send results + question to LLM:               │
│     "Analyze this revenue data and provide         │
│      insights about trends and opportunities"      │
│ 15. Stream LLM response                           │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ LLM PROVIDER (External)                            │
│                                                    │
│ 16. Anthropic API → claude-sonnet-4                │
│     Input: data + context + system prompt          │
│     Output: streaming insight text                 │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ RESPONSE STREAM (SSE to Frontend)                  │
│                                                    │
│ 17. data: {"type":"text","content":"Here's your    │
│            revenue breakdown this quarter..."}     │
│ 18. data: {"type":"artifact","artifact":{          │
│            "type":"chart","data":[...]}}           │
│ 19. data: {"type":"text","content":"Key insight:   │
│            Acme Corp grew 40% QoQ..."}             │
│ 20. data: {"type":"done"}                          │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│ FRONTEND (Renders Response)                        │
│                                                    │
│ 21. Display insight text in chat                   │
│ 22. Render chart artifact in artifact panel        │
│ 23. Save assistant message to conversation         │
│ 24. Update token usage display                     │
└────────────────────────────────────────────────────┘
```

---

## 8. The Desktop Experience

### How OSA Fits in the Desktop

```
┌─────────────────────────────────────────────────────────┐
│ CUSTOMOS DESKTOP                                        │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ MENU BAR: [OS Name] | Clock | Search | Settings  │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────┐  ┌───────────────────────┐ │
│  │ OSA CHAT WINDOW         │  │ MODULE WINDOW         │ │
│  │ (Always Available)      │  │ (CRM / Projects / etc)│ │
│  │                         │  │                       │ │
│  │ User: Show revenue...   │  │ [Module-specific UI]  │ │
│  │                         │  │                       │ │
│  │ OSA: Here's your...     │  │                       │ │
│  │ [chart artifact]        │  │                       │ │
│  │                         │  │                       │ │
│  │ [Input: Ask anything]   │  │                       │ │
│  └─────────────────────────┘  └───────────────────────┘ │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │ DOCK: [OSA] [Dashboard] [CRM] [Projects] [Tasks] │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  DESKTOP ICONS:                                         │
│  [Module icons based on installed modules]              │
└─────────────────────────────────────────────────────────┘
```

### OSA Is Always Available

- **Dock icon** — Always pinned, one click to open
- **Spotlight search** (Cmd+Space) — Type anything, OSA handles it
- **Inline in any module** — Context menu → "Ask OSA about this"
- **Quick action bar** — Available from any window

### Desktop Adapts to Installed Modules

When modules are installed/removed:
- Desktop icons update (new module → new icon appears)
- Sidebar navigation updates (new route added)
- Dock suggestions update
- OSA's context updates (knows about new module)
- Dashboard widgets update (new data sources)

---

## 9. How Modules Change the Backend

### Module Installation Flow

```
User: "Install the Campaign module"
  │
  ▼
OSA (BUILD mode):
  1. Find module in marketplace → campaign module definition
  2. Validate compatibility (check dependencies)
  3. Run module migrations → create campaign tables
  4. Register handlers → /api/v1/campaigns CRUD endpoints
  5. Register SORX skills → campaigns.create, campaigns.schedule
  6. Load manifest → protection rules
  7. Notify frontend → new routes available
  │
  ▼
Backend changes (automatic):
  - New tables: campaigns, campaign_metrics, campaign_templates
  - New routes: /api/v1/campaigns, /api/v1/campaigns/:id, ...
  - New skills: campaigns.create, campaigns.schedule, campaigns.analyze
  - New module in custom_modules table
  │
  ▼
Frontend changes (automatic):
  - New routes: /campaigns, /campaigns/[id]
  - New components loaded from module package
  - New desktop icon appears
  - New sidebar nav item
  - OSA now knows about campaigns
```

### Module Removal Flow

```
User: "Remove the Campaign module"
  │
  ▼
OSA:
  1. Check core manifest → is this module protected?
     - If YES → "I can't remove this. It's a core module."
     - If NO → proceed
  2. Warn user: "This will delete all campaign data. Are you sure?"
  3. User confirms
  4. Deregister handlers (remove routes)
  5. Deregister skills
  6. Optionally archive tables (don't delete data immediately)
  7. Remove from module registry
  8. Notify frontend → routes removed
```

### Module-Specific Backend Code

Each module is a self-contained package:

```
modules/
├── crm/
│   ├── manifest.json           ← Protection rules
│   ├── migrations/
│   │   ├── 001_create_clients.sql
│   │   └── 002_add_deals.sql
│   ├── handler/
│   │   ├── clients.go
│   │   └── deals.go
│   ├── service/
│   │   └── crm_service.go
│   ├── repository/
│   │   └── client_repo.go
│   ├── skills/
│   │   ├── import_contacts.go
│   │   └── export_contacts.go
│   └── frontend/              ← Svelte components + routes
│       ├── routes/
│       └── components/
│
├── campaigns/
│   ├── manifest.json
│   ├── migrations/
│   ├── handler/
│   ├── service/
│   ├── skills/
│   └── frontend/
│
└── ... (any module)
```

---

## 10. How Modules Change the Frontend

### Dynamic Navigation

```typescript
// The sidebar generates navigation from installed modules
const navigation = [
  // Core (always present)
  { name: 'Dashboard', route: '/dashboard', icon: 'home' },
  { name: 'OSA Chat', route: '/chat', icon: 'message-circle' },
  { name: 'Settings', route: '/settings', icon: 'cog' },

  // Module-specific (dynamic)
  ...installedModules.map(mod => ({
    name: mod.display_name,
    route: mod.base_route,
    icon: mod.icon,
  }))
];
```

### Dynamic Desktop Icons

```typescript
// Desktop icons generated from installed modules
const desktopIcons = [
  // Core icons (always present)
  { id: 'osa', label: 'OSA', module: 'chat', icon: 'brain' },
  { id: 'dashboard', label: 'Dashboard', module: 'dashboard', icon: 'layout-dashboard' },

  // Module icons (dynamic)
  ...installedModules.map(mod => ({
    id: mod.id,
    label: mod.display_name,
    module: mod.id,
    icon: mod.icon,
  }))
];
```

### Dynamic Window Content

```svelte
<!-- Window renders module content based on module ID -->
{#if window.module === 'chat'}
  <ChatPage />
{:else if window.module === 'dashboard'}
  <DashboardPage />
{:else}
  <!-- Dynamic module page loaded from module package -->
  <ModuleRenderer moduleId={window.module} />
{/if}
```

---

## 11. Skills & SORX — OSA's Hands

### How OSA Uses SORX

```
User: "Sync my Gmail contacts"
  │
  ▼
OSA Mode Router → EXECUTE mode
  │
  ▼
ExecuteModeHandler:
  1. Parse intent: "sync Gmail contacts"
  2. SkillMatcher.Match("sync Gmail contacts")
     → Returns: gmail.contacts.sync (confidence: 0.95)
  3. Check integration: Does user have Gmail OAuth token?
     → Yes: proceed
     → No: "You need to connect Gmail first. Go to Settings → Integrations."
  4. sorx.ExecuteSkill("gmail.contacts.sync", params)
  5. Stream progress events to user
  6. Report results: "Imported 247 contacts from Gmail"
```

### Skill Tiers

```
TIER 1: Deterministic (No AI)
  businessos.create_task → Direct DB insert
  gmail.contacts.list → Gmail API call
  calendar.events.today → Calendar API call

TIER 2: Structured AI (Haiku-level)
  crm.enrich_contacts → Extract company info from web
  email.summarize → Summarize email thread

TIER 3: Reasoning AI (Sonnet-level)
  analyze.revenue_trends → Query + interpret business data
  build.custom_report → Generate custom report structure

TIER 4: Generative AI (Opus-level)
  build.module → Generate entire module code
  build.migration → Generate database migration
```

---

## 12. Cross-OS Context — OSA Reaching Beyond

When OSA needs data from another OS instance:

```
User (in BusinessOS): "What campaigns did we run last month?"

OSA in BusinessOS:
  1. ANALYZE mode selected
  2. Check local modules → no campaigns module
  3. Check org's other OS instances:
     → AgencyOS has campaigns module
  4. Check cross_os_access_grants:
     → Grant exists: AgencyOS → BusinessOS, scope: campaigns.read
  5. Call Cross-OS Bridge:
     POST https://<agencyos-url>/api/v1/internal/cross-os-query
     Authorization: Bearer <platform-service-token>
     Body: { query: "campaigns last month", scope: "campaigns.read" }
  6. AgencyOS responds with campaign data
  7. OSA presents results in BusinessOS conversation

If NO grant exists:
  OSA: "I can see your AgencyOS has campaign data, but I don't have
        access to it yet. Would you like to grant me read access?"
  User: "Yes"
  → System creates cross_os_access_grant
  → OSA retries the query
```

---

## 13. Signal Theory: 7-Layer Implementation (Live)

Signal Theory is the governing framework for output quality. All 7 layers are implemented and live in production:

```
LAYER 1: FastClassifier (Mode Detection)
  ✅ Deployed in internal/orchestration/osa_modes.go
     Every message → signal classification
     6-priority chain: explicit → BUILD → keywords → default
     Confidence scoring per mode

LAYER 2: GenreEnricher (Model-Agnostic Annotations)
  ✅ Deployed in internal/handlers/chat_streaming.go
     Produces SignalEnvelope on every message
     Annotates structure (genre) + writing style
     No-gate architecture: additive metadata only

LAYER 3-7: Response Routing & Feedback
  ✅ Deployed in internal/handlers/response_builder.go
     LLM self-routes based on signal metadata
     Frontend receives signal_classified SSE events
     Metrics tracking signal quality + S/N ratio

IMPLEMENTATION DETAILS:
  • FastClassifier produces: mode, confidence, intent_type
  • GenreEnricher produces: genre, writing_style, structure_hints, complexity
  • Combined output: SignalEnvelope{mode, genre, confidence, hints}
  • Frontend uses signal_classified events for adaptive rendering
  • No hard gates: LLM model receives hints, makes routing decision
  • All 7 layers verified working in commit d33a890 (40 files changed)

SIGNAL QUALITY METRICS:
  • S/N tracking enabled across all modes
  • Per-genre entropy analysis
  • Cross-layer redundancy elimination
  • Bandwidth-aware response compression
  • Noise floor measurement per mode (ANALYZE, BUILD, ASSIST, EXECUTE, MAINTAIN)
```

**Commit:** d33a890 (Signal Theory 7-Layer Impl)
**Files changed:** 40
**Status:** Production (all SSE events include signal_classified metadata)

---

## 14. Future: OSA as Standalone Service

In the future, if we need to scale OSA independently:

```
PHASE 1 (NOW): OSA built into CustomOS
  - Same container, direct access
  - Simplest, fastest, cheapest

PHASE 2 (FUTURE): OSA as sidecar
  - Separate process, same machine
  - Communicates via Unix socket (near-zero latency)
  - Can be updated independently
  - Share database connection

PHASE 3 (FAR FUTURE): OSA as service
  - Separate container
  - Communicates via gRPC (low latency)
  - Independent scaling
  - Own database connection pool
  - Useful at 10,000+ OS instances
```

The architecture is designed so Layer 2 (Hands) uses interfaces. Moving from built-in to sidecar to service only changes how the interfaces are connected — not the interfaces themselves.

```go
// These interfaces work whether OSA is in-process or remote
type ModuleService interface {
    ListModules(ctx context.Context) ([]Module, error)
    InstallModule(ctx context.Context, moduleID string) error
    RemoveModule(ctx context.Context, moduleID string) error
}

type DataQuerier interface {
    Query(ctx context.Context, sql string, args ...interface{}) ([]Row, error)
}

type SkillExecutor interface {
    Execute(ctx context.Context, skillID string, params map[string]interface{}) (*Result, error)
}
```

---

## Summary

```
OSA = Your own Claude Code inside your OS

Built INTO CustomOS (not external)
  → Direct database access (no API overhead)
  → Direct SORX access (same process)
  → Direct module access (shared structs)
  → Only LLM calls go external

Three layers:
  1. Brain (mode router + LLM) → thinks
  2. Hands (mode handlers) → acts
  3. Platform (DB + Redis + SORX) → infrastructure

Existing code does 80% of what's needed:
  → Intent router → becomes mode router
  → Agent registry → becomes mode handlers
  → LLM service → unchanged
  → SORX engine → unchanged
  → Streaming → unchanged

New code (~2,750 lines):
  → Mode router + 5 mode handlers
  → Module protection service
  → Sandbox edit flow
  → Skill matcher
  → Cross-OS bridge

Desktop adapts to modules:
  → Icons, navigation, windows all dynamic
  → Install module → new UI appears
  → Remove module → UI disappears
  → OSA always knows what's installed
```

---

**Last updated:** 2026-02-27 — Implementation status updated: module system BUILT (CustomModuleService + install/uninstall), SORX Tier 1-4 via CARRIER bridge, CARRIER bridge BUILT (`internal/carrier/`), Optimal (Elixir/OTP) replaces osa-engine extraction target, STEERSMAN + HOMEOSTAT documented
