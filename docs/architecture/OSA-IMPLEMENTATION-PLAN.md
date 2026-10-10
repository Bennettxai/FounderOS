# OSA Implementation Plan

> **Concrete implementation plan** for building the Optimal System Agent inside BusinessOS.
> Based on actual codebase audit — what exists, what's stubbed, what's missing, and the exact
> files + functions that need work.
>
> Created: 2026-02-23 | Last Updated: 2026-02-27
> Status: ACTIVE
> Related: [Build Roadmap](./MIOSA-BUILD-ROADMAP.md) | [Platform Architecture](./MIOSA-PLATFORM-ARCHITECTURE.md) | [Platform Flow & OSA](./MIOSA-PLATFORM-FLOW.md)

---

## Table of Contents

1. [Current State Summary](#1-current-state-summary)
2. [Sprint Blockers (Must Land First)](#2-sprint-blockers-must-land-first)
3. [OSA Build Phases](#3-osa-build-phases)
4. [Phase A: Fix the Pipeline (Current Sprint)](#4-phase-a-fix-the-pipeline)
5. [Phase B: Module Protection System](#5-phase-b-module-protection-system)
6. [Phase C: 5-Mode Router](#6-phase-c-5-mode-router)
7. [Phase D: Sandbox Edit Flow](#7-phase-d-sandbox-edit-flow)
8. [Phase E: SORX EXECUTE Mode](#8-phase-e-sorx-execute-mode)
9. [Phase F: ANALYZE & MAINTAIN Modes](#9-phase-f-analyze--maintain-modes)
10. [File-by-File Change Map](#10-file-by-file-change-map)
11. [Database Changes](#11-database-changes)
12. [Testing Strategy](#12-testing-strategy)
13. [Exit Criteria](#13-exit-criteria)

---

## Reactive vs Proactive — What This Plan Covers

> **Important context for reading this plan.**

This implementation plan covers the **reactive layer** of OSA — the L1-L2 autonomy tier that
lives in the open-source BOS repository. Reactive means: the system acts only in response to
user messages. Intent is classified, a mode handler is dispatched, and a response is returned.

The **proactive layer** (L3-L5 autonomy) — where OSA monitors signals, initiates workflows,
and acts without waiting for the user — is tracked separately and extracts to `osa-engine/`
as a premium Miosa.ai component. That extraction plan is documented in
[ARCHITECTURE-SEPARATION.md](../ARCHITECTURE-SEPARATION.md).

**Boundary line:** `internal/orchestration/osa_orchestrator.go` is the junction. Code above
it (mode router, handlers, context assembly) stays in BOS (reactive, open source). Code below
it that initiates autonomous action belongs in osa-engine/ (proactive, premium).

**SORX note:** The SORX Go engine (`internal/sorx/`) included here covers Tier 1-2 only and
stays in BOS. SORX as a complete premium skill engine (Tier 3-4) is NOT open sourced.
nano_sorx (Elixir/OTP) provides open-source reference patterns for proactive skill execution.

---

## Implementation Status (Updated 2026-02-27)

```
Phase A: Fix the Pipeline       ← ✅ COMPLETE (Sprint 2 — PR #5, #7, #10 fixed all blockers)
Phase B: Module Protection      ← ✅ COMPLETE (module_protection.go, 307 lines, 48 tests)
Phase C: 5-Mode Router          ← ✅ COMPLETE (osa_modes.go, 475+ lines, 56 tests + 45 E2E tests + integration tests)
Phase D: Sandbox Edit Flow      ← ✅ COMPLETE (Sprint 3 — PR #15, E2B integration, 42 tests)
Phase E: SORX EXECUTE Mode      ← ✅ COMPLETE (Sprint 3 — semantic skill matching via pgvector, carrier routing)
Phase F: ANALYZE & MAINTAIN     ← ✅ COMPLETE (Sprint 3 — PR #8, real data queries + health metrics)
```

> **All phases complete as of Sprint 3 (2026-02-27).** The E2E pipeline works: message → classify mode → dispatch handler → execute skill/generate module → stream response. CARRIER bridge connects Go (Tier 1-2) to Optimal/Elixir (Tier 3-4) for advanced reasoning.

> **Code location note:** The plan below proposed files in `internal/agents/`. The actual
> implementation lives in `internal/orchestration/osa_modes.go` and
> `internal/services/module_protection.go` — different paths, same functionality.

### Sprint 3C Additions (2026-02-27)

- **CARRIER AMQP bridge** (`internal/carrier/`) — connects Go backend to Optimal for Tier 3-4 advanced reasoning
- **LLM fallback** for ambiguous mode classification (0.4-0.7 confidence zone)
- **45 E2E mode classification tests** + integration pipeline tests
- **4 nil-guard fixes** (`pact.go`, `bmad.go`, `osa_orchestrator.go`)
- **Optimal repo created** (STEERSMAN + HOMEOSTAT)

---

## 1. Current State Summary

### What's Working (Can Build On)

| Component | File | Completeness | Notes |
|-----------|------|-------------|-------|
| Intent Router v2 | `internal/agents/intent_router_v2.go` (515 lines) | 100% | 4-layer classification: regex → semantic → context → LLM. Routes to 4 agent types + orchestrator. |
| SORX Engine | `internal/sorx/engine.go` (756 lines) | 75% | Skill tiers, temperature control, step execution. 28 action handlers (Gmail, Calendar, Slack, HubSpot, Linear, Notion, AI, BusinessOS). |
| SORX Agent Bridge | `internal/sorx/agent_bridge.go` (~300 lines) | 70% | Bidirectional skill ↔ agent connection. |
| Database Schema | Migrations 048, 054, 079, 089, 091 | 100% | All needed tables exist: `osa_modules`, `osa_workspaces`, `osa_generated_apps`, `custom_modules`, `custom_module_versions`, `app_generation_queue`, `sandbox_events`. |
| Container Security | `internal/container/container.go` | 70% | Drop-all-caps, seccomp, read-only rootfs, memory/CPU/PID limits. |
| Prompt Templates | `internal/services/osa_prompt_builder.go` (361 lines) | 85% | YAML templates embedded at compile time. 6 templates: crm-app-gen, feature-addition, bug-fix, dashboard, data-pipeline, placeholder. |
| Custom Modules CRUD | `internal/services/custom_module_service.go` (814 lines) | 60% | Create, versions, installs, shares. 15 handler endpoints. |
| Handlers/API | `internal/handlers/sorx.go` (457 lines), `osa_api.go` | 75% | SORX credential tickets, callbacks, decisions. OSA generate + status polling. |

### What's Broken (Sprint Blockers) — ALL FIXED

| Issue | File | Agent Fixing | Status |
|-------|------|-------------|--------|
| FK violation — `osa_generated_apps` not created before queueing | `internal/handlers/osa_apps.go` | Roberto A | FIXED (PR #5 — migration 095) |
| Prompt overwritten — rich prompt replaced with short description | `internal/services/osa_queue_worker.go` (line ~200) | Roberto A | FIXED (PR #5) |
| `OSAPromptBuilder` never called — uses raw string concat | `internal/services/app_generation_orchestrator.go` | Pedro F | FIXED (PR #7) |
| Hardcoded agent descriptions — all 4 agents get "Create Svelte frontend" | `internal/services/app_generation_orchestrator.go` (line ~179) | Pedro A | FIXED (PR #7) |
| `BuildEventBus` silent drops — buffer=100, drops `generation_complete` | `internal/services/build_event_bus.go` (line ~120) | Pedro D | FIXED (PR #7) |
| Hardcoded Anthropic URLs — 5 instances of `api.anthropic.com` | `internal/services/anthropic.go` | Roberto D | FIXED (PR #10) |
| SSE wrong route — frontend passes `appId` but backend expects `queueItemId` | `frontend/src/lib/stores/generatedAppsStore.ts` | Javaris B | FIXED (PR #16) |
| No "Open in Editor" button after generation | `frontend/src/lib/components/osa/AgentProgressPanel.svelte` | Javaris B | FIXED (PR #16) |
| Monaco save not wired — Ctrl+S doesn't persist | `frontend/src/routes/(app)/generated-apps/[id]/+page.svelte` | Javaris B | FIXED (PR #16) |
| PUT file save endpoint missing | `internal/handlers/osa_apps.go` | Pedro A | FIXED (PR #7) |

### What's Stubbed (Needs Implementation)

| Component | File | What's Missing |
|-----------|------|---------------|
| `getDBTemplate()` | `osa_prompt_builder.go` | Returns "not yet implemented" — no user/workspace-level templates |
| `OSAClientAdapter.GenerateApp()` | `osa_client_adapter.go` | Creates placeholder records, doesn't call real code generation |
| Sandbox → app gen pipeline | `sandbox_deployment.go` ↔ `app_generation_orchestrator.go` | Tables exist (migration 091) but not wired to app generation output |
| Module protection validation | ✅ COMPLETE | `module_protection.go`, 307 lines, 48 tests |
| OSA mode selection | ✅ COMPLETE | `osa_modes.go` with 3-layer classification (explicit → keyword → LLM fallback), 100+ tests |
| Cross-OS bridge | **Doesn't exist yet** | No `cross_os_bridge.go` — future (Phase 2+ of roadmap) |

---

## 2. Sprint Blockers (Must Land First)

These are the current sprint tasks. **OSA implementation cannot start until these land.**

### Merge Order

```
Roberto FK Fix (A)           ← EVERYTHING depends on this
    ↓
Pedro OSAPromptBuilder (F)   ← Mode router depends on prompt builder working
Pedro BuildEventBus (D)      ← Sandbox flow depends on reliable events
Roberto Ollama Cloud (D)     ← LLM provider config depends on configurable URLs
    ↓
Pedro Hardcoded Descs (A)    ← Orchestrator needs real agent descriptions
Pedro PUT File Save (A)      ← Sandbox edit flow needs file persistence
    ↓
Javaris SSE Route Fix (B)    ← End-to-end testing needs working SSE
Javaris Monaco Save (B)      ← Edit flow needs working editor
Javaris Open in Editor (B)   ← User flow needs navigation
```

### After Sprint Lands

The app generation pipeline will work end-to-end:
```
User creates app → osa_generated_apps row created → queue item added →
worker processes with enriched prompt → orchestrator generates with real agent descriptions →
files saved to DB → BuildEventBus emits generation_complete (no drops) →
SSE stream delivers to frontend (correct route) → "Open in Editor" button →
Monaco editor loads files → Ctrl+S saves via PUT endpoint → done
```

**This working pipeline is the foundation for all OSA phases below.**

---

## 3. OSA Build Phases

```
Phase A: Fix the Pipeline       ← Current sprint (agents working now)
    ↓
Phase B: Module Protection      ← Core manifest, validation algorithm
    ↓
Phase C: 5-Mode Router          ← Extend intent router with OSA modes
    ↓
Phase D: Sandbox Edit Flow      ← Fork → sandbox → preview → approve → apply
    ↓
Phase E: SORX EXECUTE Mode      ← Wire OSA → SORX skill execution
    ↓
Phase F: ANALYZE & MAINTAIN     ← Data queries, module versioning

Phases B, C can run in parallel.
Phase D depends on A + B.
Phase E depends on C.
Phase F depends on C + E.
```

---

## 4. Phase A: Fix the Pipeline

**Status:** In progress (current sprint agents)

**Owner:** Roberto (A, D), Pedro (A, D, F), Javaris (B)

**Details:** See [Section 2](#2-sprint-blockers-must-land-first) above.

**Exit Criteria:**
- [ ] `go build ./cmd/server` passes
- [ ] `go test ./...` passes
- [ ] App generation creates DB row before queueing
- [ ] Rich prompt flows through to orchestrator without overwrite
- [ ] OSAPromptBuilder called with real templates
- [ ] Agent descriptions are dynamic, not hardcoded
- [ ] BuildEventBus doesn't drop events
- [ ] SSE streams on correct route (`queueItemId`)
- [ ] "Open in Editor" navigates to Monaco
- [ ] Ctrl+S persists files via PUT endpoint
- [ ] Anthropic base URL configurable via env var

---

## 5. Phase B: Module Protection System — ✅ COMPLETE

**Goal:** Prevent OSA from breaking core module functionality. Core manifests define what's protected.

**Status:** BUILT. `internal/services/module_protection.go` (306 lines) + `module_protection_test.go` (48 tests, ALL PASS).

**What was actually built vs what was planned:**
- Planned: `ValidateChanges()` taking `osInstanceID + ModuleChangeset` → Built: `ValidateChanges()` taking `moduleID + []ChangeRequest`
- Planned: reads from `os_instances.core_module_manifest` → Built: reads from `custom_modules.manifest` JSONB column
- Added: 4 pattern types (exact, prefix wildcard, suffix wildcard, glob) for manifest matching
- Added: Per-change-type severity (schema/file deletes = error, updates = warning, route/operation modifications = error)
- Added: Batch validation via `ValidateChanges()` for multiple changes at once
- **Tests:** 48 tests covering all pattern types, change types, and edge cases

### Original Planned File: `internal/services/module_protection.go`

```go
package services

// ModuleProtectionService validates proposed changes against core manifests.
// Injected into OSA BUILD mode pipeline — runs AFTER code generation, BEFORE apply.

type ModuleProtectionService struct {
    db *database.Queries
}

// ValidateChanges checks proposed diff against the OS instance's core manifest.
// Returns nil if safe, returns ProtectionViolation if protected elements would be removed.
func (s *ModuleProtectionService) ValidateChanges(
    ctx context.Context,
    osInstanceID uuid.UUID,
    proposedChanges ModuleChangeset,
) (*ProtectionResult, error)

// ProtectionResult contains the validation outcome + suggestions if violated.
type ProtectionResult struct {
    Valid      bool
    Violations []ProtectionViolation
    Suggestions []string  // Safe alternatives OSA can propose
}

type ProtectionViolation struct {
    Type       string // "operation_removed", "column_removed", "route_removed"
    Module     string
    Element    string // The specific operation/column/route
    Manifest   string // Which manifest rule was violated
}

// ModuleChangeset represents proposed changes from OSA BUILD mode.
type ModuleChangeset struct {
    ModuleID        uuid.UUID
    Operations      []string          // API operations after change
    SchemaColumns   map[string][]string // table → columns after change
    UIRoutes        []string          // Frontend routes after change
    CodeDiff        string            // Raw diff for review
}
```

### Validation Algorithm

```
1. Load core_module_manifest from os_instances table
2. For the target module:
   a. Check every protected_operations still exists in proposedChanges.Operations
   b. Check every required_columns still exists in proposedChanges.SchemaColumns[table]
   c. Check every protected_ui_routes still exists in proposedChanges.UIRoutes
3. For each violation:
   a. Record what was removed and which manifest rule protects it
   b. Generate suggestion (e.g., "make it optional instead of removing")
4. Return ProtectionResult
```

### Core Manifest Format (Stored in `os_instances.core_module_manifest`)

Already defined in architecture doc (Section 5). Example:

```json
{
  "modules": {
    "crm": {
      "version": "1.0.0",
      "protected_operations": [
        "contacts.list", "contacts.create", "contacts.update", "contacts.delete"
      ],
      "protected_schemas": [
        {"table": "clients", "required_columns": ["id", "name", "email", "status"]}
      ],
      "protected_ui_routes": ["/clients", "/clients/[id]"],
      "customizable": {
        "add_operations": true,
        "add_columns": true,
        "add_routes": true,
        "remove_protected": false
      }
    }
  }
}
```

### Integration Points

| Where | What Changes |
|-------|-------------|
| `app_generation_orchestrator.go` | After code generation, before applying: call `ModuleProtectionService.ValidateChanges()` |
| `osa_prompt_builder.go` | Inject core manifest into OSA system prompt so it knows constraints upfront |
| `custom_module_service.go` | On module update: validate against manifest before saving |

### Files to Create/Modify

| Action | File | Changes |
|--------|------|---------|
| CREATE | `internal/services/module_protection.go` | ~200 lines — validation service |
| CREATE | `internal/services/module_protection_test.go` | ~150 lines — test all violation types |
| MODIFY | `internal/services/app_generation_orchestrator.go` | Add validation call after generation |
| MODIFY | `internal/services/osa_prompt_builder.go` | Add manifest injection into system prompt |
| MODIFY | `internal/services/custom_module_service.go` | Add protection check on module updates |

---

## 6. Phase C: 5-Mode Router — ✅ COMPLETE

**Goal:** Extend the existing Smart Intent Router to classify user messages into OSA's 5 modes.

**Status:** BUILT. `internal/orchestration/osa_modes.go` (349 lines) + `osa_modes_test.go` (56 tests + 3 benchmarks, ALL PASS).

**What was actually built vs what was planned:**
- Planned: `internal/agents/osa_mode_router.go` → Built: `internal/orchestration/osa_modes.go` (in orchestration package, not agents)
- Planned: 3-layer classification (regex → semantic → LLM) → Built: 6-priority chain (explicit → OSARouter → keyword scoring × 3 → default)
- Added: Weighted keyword scoring (primary 0.5, secondary 0.2-0.25, threshold 0.7) instead of simple regex
- Added: `ModeDisplayInfo()` for frontend labels/icons/descriptions
- Added: `AllModes()` for frontend dropdowns
- Added: Confidence scores for every classification
- Planned: separate `osa_chat.go` handler → Built: integrated into existing `osa_orchestrator.go` via `ProcessWithOSARouting()`
- **ASSIST is the safe default** — falls through to existing SmartIntentRouter, zero regression

**Also built (not in original plan):**
- Orchestrator refactored to ~550 lines with 5 mode handlers
- SORX engine wired from `main.go` → `handlers.go` → `chat_v2.go` → orchestrator
- `NewOSAOrchestratorWithConfig()` with full config struct (backward compatible)
- Nil-safety guard for missing dependencies
- `matchSkillFromMessage()` with 11 natural language patterns for EXECUTE mode
- `executeSorxSkill()` for SORX integration with SSE streaming

### Original Planned Approach

Don't replace the intent router — add an OSA mode layer that runs first. If the message is an OSA-mode action, handle it in OSA. Otherwise, fall through to the existing agent type routing.

### New File: `internal/agents/osa_mode_router.go`

```go
package agents

type OSAMode string

const (
    ModeBuild    OSAMode = "build"
    ModeAssist   OSAMode = "assist"
    ModeAnalyze  OSAMode = "analyze"
    ModeExecute  OSAMode = "execute"
    ModeMaintain OSAMode = "maintain"
    ModeNone     OSAMode = "none" // Fall through to intent router
)

type OSAModeRouter struct {
    intentRouter *SmartIntentRouter
    sorxEngine   *sorx.Engine
    protectionSvc *ModuleProtectionService
    promptBuilder *OSAPromptBuilder
}

// ClassifyMode determines which OSA mode handles this message.
// Uses the same 3-layer approach as SmartIntentRouter.
func (r *OSAModeRouter) ClassifyMode(
    ctx context.Context,
    message string,
    conversationHistory []Message,
    workspaceContext WorkspaceContext,
) (OSAMode, float64) // mode + confidence

// RouteMessage dispatches to the appropriate mode handler.
func (r *OSAModeRouter) RouteMessage(
    ctx context.Context,
    mode OSAMode,
    message string,
    wsCtx WorkspaceContext,
) (*OSAResponse, error)
```

### Mode Classification Rules

**Layer 1 — Pattern Matching (fast, high confidence):**

| Mode | Patterns |
|------|----------|
| BUILD | `create\s+(a\s+)?module`, `generate\s+(a\s+)?`, `build\s+(me\s+)?`, `add\s+feature`, `customize\s+module`, `fork\s+module` |
| ASSIST | `how\s+do\s+I`, `help\s+(me\s+)?with`, `explain`, `what\s+is`, `guide\s+me`, `show\s+me\s+how` |
| ANALYZE | `analyze`, `report\s+on`, `show\s+(me\s+)?trends`, `dashboard`, `what's\s+the\s+status`, `compare`, `metrics` |
| EXECUTE | `sync\s+my`, `send\s+(a\s+)?`, `import\s+`, `run\s+`, `connect\s+`, `trigger\s+`, `schedule` |
| MAINTAIN | `update\s+(the\s+)?module`, `upgrade`, `migrate`, `version`, `roll\s*back`, `fix\s+(the\s+)?module` |

**Layer 2 — Semantic Signals (weighted):**

| Signal | Weight | Modes |
|--------|--------|-------|
| Code generation terms (function, component, table, schema) | 0.3 | BUILD |
| Question markers (?, how, why, what) | 0.3 | ASSIST |
| Data terms (revenue, count, average, trend, growth) | 0.3 | ANALYZE |
| Action verbs (send, sync, import, connect, run) | 0.3 | EXECUTE |
| Maintenance terms (update, version, dependency, migrate) | 0.3 | MAINTAIN |
| Module reference ("the CRM module", "invoicing") | 0.2 | BUILD or MAINTAIN |
| Integration reference ("Gmail", "Slack", "calendar") | 0.2 | EXECUTE |

**Layer 3 — LLM Classification (fallback for ambiguous):**

```
System: You are an intent classifier for OSA. Given the user message, classify into exactly one mode:
- BUILD: User wants to create, generate, or customize a module/feature
- ASSIST: User wants help, guidance, or an explanation
- ANALYZE: User wants data insights, reports, or dashboards
- EXECUTE: User wants to run an action, sync data, or trigger an integration
- MAINTAIN: User wants to update, version, or maintain an existing module

Respond with just the mode name and confidence (0.0-1.0).
```

### Mode Handlers

Each mode delegates to existing infrastructure:

| Mode | Handler | Uses |
|------|---------|------|
| BUILD | `handleBuildMode()` | `app_generation_orchestrator.go` + `module_protection.go` + sandbox |
| ASSIST | `handleAssistMode()` | `osa_prompt_builder.go` + existing agent system (document/project agents) |
| ANALYZE | `handleAnalyzeMode()` | SORX `analysis.*` skills + database queries |
| EXECUTE | `handleExecuteMode()` | SORX engine (`engine.go`) + agent bridge |
| MAINTAIN | `handleMaintainMode()` | `custom_module_service.go` + version system |

### Integration: New Handler Endpoint

```go
// internal/handlers/osa_chat.go

// POST /api/v1/workspaces/:workspaceId/osa/chat
//
// The main OSA interaction endpoint. Replaces direct agent routing
// for OSA-enabled workspaces.
func (h *OSAChatHandler) HandleChat(c *gin.Context) {
    // 1. Parse message from request body
    // 2. Load workspace context (modules, integrations, history)
    // 3. Call OSAModeRouter.ClassifyMode()
    // 4. Call OSAModeRouter.RouteMessage()
    // 5. Stream response via SSE if BUILD/EXECUTE, return JSON if ASSIST/ANALYZE
}
```

### Files to Create/Modify

| Action | File | Changes |
|--------|------|---------|
| CREATE | `internal/agents/osa_mode_router.go` | ~350 lines — mode classification + routing |
| CREATE | `internal/agents/osa_mode_router_test.go` | ~200 lines — test classification accuracy |
| CREATE | `internal/handlers/osa_chat.go` | ~150 lines — main OSA chat endpoint |
| MODIFY | `internal/handlers/handlers.go` | Register `/osa/chat` route |
| MODIFY | `internal/agents/intent_router_v2.go` | Add fallthrough hook for when OSA mode is "none" |

---

## 7. Phase D: Sandbox Edit Flow

**Goal:** Full cycle — user asks OSA to modify a module → OSA forks to sandbox → generates changes → validates → user previews → user approves → apply to live → version snapshot.

### Current State

- `sandbox_deployment.go` exists — deploys Docker containers per-app
- `container.go` has security hardening (seccomp, drop caps, read-only rootfs)
- `sandbox_events` table tracks lifecycle
- `osa_generated_apps.sandbox_status` has states: pending/deploying/running/stopped/failed/error
- `custom_module_versions` table can store version snapshots with `manifest_snapshot`
- **Missing:** The orchestration that connects all of this

### New File: `internal/services/sandbox_edit_flow.go`

```go
package services

// SandboxEditFlow orchestrates the full module edit lifecycle:
// fork → sandbox → generate → validate → preview → approve → apply → version

type SandboxEditFlow struct {
    orchestrator   *AppGenerationOrchestrator
    protection     *ModuleProtectionService
    sandbox        *SandboxDeploymentService
    modules        *CustomModuleService
    eventBus       *BuildEventBus
}

// StartEdit begins the sandbox edit flow for a module.
func (f *SandboxEditFlow) StartEdit(
    ctx context.Context,
    workspaceID uuid.UUID,
    moduleID uuid.UUID,
    userRequest string,     // What the user asked OSA to do
    osaMode OSAMode,        // BUILD or MAINTAIN
) (*SandboxSession, error)

// PreviewChanges returns the diff and sandbox URL for user review.
func (f *SandboxEditFlow) PreviewChanges(
    ctx context.Context,
    sessionID uuid.UUID,
) (*SandboxPreview, error)

// ApproveAndApply applies sandbox changes to the live module.
func (f *SandboxEditFlow) ApproveAndApply(
    ctx context.Context,
    sessionID uuid.UUID,
) (*ModuleVersion, error)

// RejectChanges discards sandbox and cleans up.
func (f *SandboxEditFlow) RejectChanges(
    ctx context.Context,
    sessionID uuid.UUID,
) error
```

### Flow

```
1. StartEdit()
   ├── Fork current module state to sandbox
   ├── Deploy sandbox Docker container
   ├── Emit event: sandbox_started
   ├── Build OSA prompt with module context + core manifest
   ├── Call orchestrator.Generate() in sandbox environment
   ├── Run ModuleProtectionService.ValidateChanges()
   │   ├── PASS → Emit event: validation_passed
   │   └── FAIL → Emit event: validation_failed + violations + suggestions
   └── Return SandboxSession with status

2. PreviewChanges()
   ├── Read generated files from sandbox
   ├── Compute diff against current module
   ├── Return preview URL (sandbox container port)
   └── Return changeset summary

3. ApproveAndApply()
   ├── Copy sandbox files to live module
   ├── Run any schema migrations
   ├── Create custom_module_versions snapshot
   ├── Update custom_modules.version
   ├── Emit event: module_updated
   ├── Destroy sandbox container
   └── Return new ModuleVersion

4. RejectChanges()
   ├── Destroy sandbox container
   ├── Clean up temporary files
   └── Emit event: edit_rejected
```

### Frontend Integration

The existing `AgentProgressPanel.svelte` + SSE stream handles real-time feedback. Needs additions:

| Component | What to Add |
|-----------|------------|
| `AgentProgressPanel.svelte` | Show validation result (pass/fail + violations) |
| New: `SandboxPreview.svelte` | Diff viewer + sandbox iframe preview |
| New: `ApproveRejectBar.svelte` | "Apply Changes" / "Reject" buttons |
| `generatedAppsStore.ts` | Handle new event types: `sandbox_started`, `validation_passed/failed`, `module_updated` |

### Files to Create/Modify

| Action | File | Changes |
|--------|------|---------|
| CREATE | `internal/services/sandbox_edit_flow.go` | ~300 lines — full edit lifecycle |
| CREATE | `internal/services/sandbox_edit_flow_test.go` | ~200 lines |
| MODIFY | `internal/services/sandbox_deployment.go` | Add module-forking logic |
| MODIFY | `internal/services/custom_module_service.go` | Add `ApplyFromSandbox()` method |
| MODIFY | `internal/services/build_event_bus.go` | Add new event types for sandbox flow |
| CREATE | `frontend/src/lib/components/osa/SandboxPreview.svelte` | ~150 lines — diff + preview |
| CREATE | `frontend/src/lib/components/osa/ApproveRejectBar.svelte` | ~50 lines |
| MODIFY | `frontend/src/lib/stores/generatedAppsStore.ts` | Handle sandbox events |
| MODIFY | `frontend/src/lib/components/osa/AgentProgressPanel.svelte` | Show validation results |

---

## 8. Phase E: SORX EXECUTE Mode

**Goal:** When user says "sync my Gmail contacts" or "send this report", OSA routes to the SORX engine which executes the appropriate skill.

### Current State

SORX engine is 75% complete. What works:
- 28 action handlers (Gmail, Calendar, Slack, HubSpot, Linear, Notion, AI, BusinessOS)
- 21 built-in skill definitions
- Tier system (Deterministic → StructuredAI → ReasoningAI → GenerativeAI)
- Temperature control (Cold/Warm/Hot)
- Agent bridge for bidirectional skill ↔ agent communication

What's missing:
- OSA → SORX routing (mode router dispatches to SORX engine)
- Credential management for user integrations
- Callback system for long-running skills (polling exists, events don't)

### Integration: OSA EXECUTE → SORX

```go
// In osa_mode_router.go

func (r *OSAModeRouter) handleExecuteMode(
    ctx context.Context,
    message string,
    wsCtx WorkspaceContext,
) (*OSAResponse, error) {
    // 1. Classify which SORX skill matches the user's request
    skill, confidence := r.sorxEngine.MatchSkill(ctx, message)

    // 2. Check temperature — does this need approval?
    if wsCtx.SorxTemperature == "cold" {
        return &OSAResponse{
            Type: "approval_needed",
            Message: fmt.Sprintf("I can run '%s'. This will %s. Approve?", skill.Name, skill.Description),
            PendingSkill: skill,
        }, nil
    }

    // 3. Execute the skill
    execution, err := r.sorxEngine.ExecuteSkill(ctx, skill, message, wsCtx)
    if err != nil {
        return nil, err
    }

    // 4. Return results
    return &OSAResponse{
        Type: "execution_result",
        Message: formatExecutionResult(execution),
        SkillExecution: execution,
    }, nil
}
```

### Credential Flow

User integrations need OAuth tokens stored securely. SORX already has a credential ticket system:

```
POST /api/sorx/credential-ticket   → Request access to user's Gmail token
POST /api/sorx/redeem-credential   → Exchange ticket for actual credential
```

What's missing: the frontend UI where users connect integrations and the secure storage backend. For now, credentials can be stored in `os_instances.settings` JSONB (encrypted at rest).

### Files to Create/Modify

| Action | File | Changes |
|--------|------|---------|
| MODIFY | `internal/agents/osa_mode_router.go` | Add `handleExecuteMode()` |
| MODIFY | `internal/sorx/engine.go` | Add `MatchSkill()` — classify message to skill |
| MODIFY | `internal/sorx/types.go` | Add skill matching types |
| CREATE | `internal/services/integration_service.go` | ~200 lines — manage user integration credentials |
| CREATE | `frontend/src/lib/components/osa/ExecutionResult.svelte` | ~100 lines — show skill execution results |

---

## 9. Phase F: ANALYZE & MAINTAIN Modes

### ANALYZE Mode

**Goal:** OSA queries business data, generates insights, creates dashboard components.

```go
func (r *OSAModeRouter) handleAnalyzeMode(
    ctx context.Context,
    message string,
    wsCtx WorkspaceContext,
) (*OSAResponse, error) {
    // 1. Identify what data the user wants analyzed
    // 2. Build SQL query against workspace data (clients, projects, tasks, etc.)
    // 3. Execute query via repository layer
    // 4. Pass results + user question to LLM for insight generation
    // 5. Optionally generate a dashboard widget definition
    // 6. Return formatted analysis
}
```

**Key consideration:** ANALYZE mode queries the user's actual business data. Must use parameterized queries (never user input in SQL). The repository layer already handles this — ANALYZE mode just needs to know which tables to query based on the user's modules.

**Data sources per module:**

| Module | Tables | Example Queries |
|--------|--------|----------------|
| CRM | `clients`, client relationships | Revenue by client, client growth, pipeline status |
| Projects | `projects`, `tasks` | Task completion rate, overdue tasks, workload |
| Communication | `conversations`, `messages` | Response time, volume trends |
| Custom | Whatever tables the module defines | Schema-aware queries |

### MAINTAIN Mode

**Goal:** Module versioning, dependency updates, schema migrations.

```go
func (r *OSAModeRouter) handleMaintainMode(
    ctx context.Context,
    message string,
    wsCtx WorkspaceContext,
) (*OSAResponse, error) {
    // 1. Identify target module
    // 2. Determine maintenance action:
    //    - "update" → enter sandbox edit flow (Phase D) with MAINTAIN context
    //    - "version" → show version history from custom_module_versions
    //    - "rollback" → restore from previous version snapshot
    //    - "migrate" → generate and run schema migration
    //    - "dependencies" → check and update module dependencies
    // 3. Execute action
    // 4. Return result
}
```

**Rollback capability:** `custom_module_versions` stores `manifest_snapshot` JSONB. Rolling back means loading the snapshot and overwriting current module state. Must also handle schema rollback (store reverse migration in version snapshot).

### Files to Create/Modify

| Action | File | Changes |
|--------|------|---------|
| MODIFY | `internal/agents/osa_mode_router.go` | Add `handleAnalyzeMode()` + `handleMaintainMode()` |
| CREATE | `internal/services/osa_analyze_service.go` | ~250 lines — data query + insight generation |
| CREATE | `internal/services/osa_maintain_service.go` | ~200 lines — version management + rollback |
| MODIFY | `internal/services/custom_module_service.go` | Add `RollbackToVersion()`, `GetVersionHistory()` |
| CREATE | `frontend/src/lib/components/osa/AnalysisResult.svelte` | ~150 lines — charts + insights |
| CREATE | `frontend/src/lib/components/osa/VersionHistory.svelte` | ~100 lines — version timeline |

---

## 10. File-by-File Change Map

### Files to CREATE (New)

| File | Phase | Lines (est.) | Purpose |
|------|-------|-------------|---------|
| `internal/services/module_protection.go` | B | 200 | Core manifest validation |
| `internal/services/module_protection_test.go` | B | 150 | Validation tests |
| `internal/agents/osa_mode_router.go` | C | 350 | 5-mode classification + routing |
| `internal/agents/osa_mode_router_test.go` | C | 200 | Mode classification tests |
| `internal/handlers/osa_chat.go` | C | 150 | Main OSA chat endpoint |
| `internal/services/sandbox_edit_flow.go` | D | 300 | Full sandbox edit lifecycle |
| `internal/services/sandbox_edit_flow_test.go` | D | 200 | Sandbox flow tests |
| `internal/services/integration_service.go` | E | 200 | User integration credentials |
| `internal/services/osa_analyze_service.go` | F | 250 | Data analysis + insights |
| `internal/services/osa_maintain_service.go` | F | 200 | Module versioning + rollback |
| `frontend/src/lib/components/osa/SandboxPreview.svelte` | D | 150 | Diff viewer + sandbox preview |
| `frontend/src/lib/components/osa/ApproveRejectBar.svelte` | D | 50 | Approve/reject buttons |
| `frontend/src/lib/components/osa/ExecutionResult.svelte` | E | 100 | SORX execution results |
| `frontend/src/lib/components/osa/AnalysisResult.svelte` | F | 150 | Charts + insights display |
| `frontend/src/lib/components/osa/VersionHistory.svelte` | F | 100 | Module version timeline |
| **Total new files: 15** | | **~2,750** | |

### Files to MODIFY (Existing)

| File | Phase | What Changes |
|------|-------|-------------|
| `internal/services/app_generation_orchestrator.go` | B, D | Add protection validation call; connect to sandbox flow |
| `internal/services/osa_prompt_builder.go` | B | Inject core manifest into system prompt templates |
| `internal/services/custom_module_service.go` | B, D, F | Add protection check, `ApplyFromSandbox()`, `RollbackToVersion()` |
| `internal/agents/intent_router_v2.go` | C | Add fallthrough hook when OSA mode router returns "none" |
| `internal/handlers/handlers.go` | C | Register `/osa/chat` route |
| `internal/services/sandbox_deployment.go` | D | Add module-forking logic (fork current module to sandbox) |
| `internal/services/build_event_bus.go` | D | Add sandbox-specific event types |
| `internal/sorx/engine.go` | E | Add `MatchSkill()` for message-to-skill classification |
| `internal/sorx/types.go` | E | Add skill matching types |
| `frontend/src/lib/stores/generatedAppsStore.ts` | D | Handle new sandbox event types |
| `frontend/src/lib/components/osa/AgentProgressPanel.svelte` | D | Show validation results |
| **Total modified files: 11** | | |

---

## 11. Database Changes

### No New Migrations Needed for OSA

All required tables already exist:

| Table | Migration | Used By |
|-------|-----------|---------|
| `osa_modules` | 048 | Module registry |
| `osa_workspaces` | 048 | Workspace context |
| `osa_generated_apps` | 048 | Generated app tracking |
| `osa_build_events` | 048 | Real-time build events |
| `custom_modules` | 054 | Module CRUD + manifests |
| `custom_module_versions` | 054 | Version snapshots |
| `custom_module_installations` | 054 | Module installations |
| `osa_prompt_templates` | 079 | Template inheritance (user > workspace > system) |
| `app_templates` | 089 | 5 built-in templates |
| `app_generation_queue` | 089 | Async generation queue |
| `sandbox_events` | 091 | Sandbox lifecycle tracking |

### Optional: Add `osa_mode_logs` Table

For tracking OSA mode classification accuracy and tuning:

```sql
-- Optional (migration 095 if needed)
CREATE TABLE osa_mode_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id),
    message_preview VARCHAR(200),
    classified_mode VARCHAR(20) NOT NULL,
    confidence FLOAT NOT NULL,
    classification_layer VARCHAR(20), -- 'pattern', 'semantic', 'llm'
    user_corrected_mode VARCHAR(20), -- NULL if classification was correct
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_osa_mode_logs_workspace ON osa_mode_logs(workspace_id);
```

This lets us measure classification accuracy and improve the mode router over time.

---

## 12. Testing Strategy

### Unit Tests (Per Phase)

| Phase | Test File | What to Test |
|-------|-----------|-------------|
| B | `module_protection_test.go` | Validate: protected op removed → FAIL. Protected column removed → FAIL. New op added → PASS. New column added → PASS. |
| C | `osa_mode_router_test.go` | Classify: "Create a module" → BUILD. "How do I?" → ASSIST. "Show revenue" → ANALYZE. "Sync Gmail" → EXECUTE. "Update CRM" → MAINTAIN. |
| D | `sandbox_edit_flow_test.go` | Full lifecycle: start → preview → approve → apply → version created. And: start → validate fail → reject → cleanup. |
| E | (extend `sorx/engine_test.go`) | MatchSkill: "sync contacts" → gmail.contacts.sync. "send email" → gmail.send. |
| F | (extend `custom_module_service_test.go`) | Rollback: create version → modify → rollback → verify original state restored. |

### Integration Tests

| Test | What It Covers |
|------|---------------|
| Build mode E2E | User message → mode router → BUILD → orchestrator → sandbox → validate → preview → approve → apply |
| Execute mode E2E | User message → mode router → EXECUTE → SORX engine → skill execution → results |
| Protection violation | User asks to remove protected field → validation fails → OSA suggests alternative |
| Mode classification accuracy | 50+ example messages, verify >90% correct classification |

### Manual Smoke Tests

| Test | Steps |
|------|-------|
| Generate a module | Chat: "Create an invoicing module" → Watch sandbox → Preview → Approve → Module appears |
| Protected field | Chat: "Remove email from contacts" → See violation message + suggestions |
| SORX execution | Chat: "Sync my Gmail contacts" → See skill execute → See imported contacts |
| Data analysis | Chat: "Show revenue by client this month" → See chart + insights |
| Module rollback | Chat: "Roll back CRM to previous version" → Confirm → Version restored |

---

## 13. Exit Criteria

### OSA MVP — What "Done" Looks Like

- [ ] **BUILD mode works end-to-end:** User asks to create/modify module → sandbox → validate → preview → approve → live
- [ ] **Core protection enforced:** Cannot remove protected operations, columns, or routes
- [ ] **ASSIST mode answers with context:** Uses OS module knowledge, workspace data, conversation history
- [ ] **EXECUTE mode triggers SORX:** At least 3 skills work (Gmail sync, task creation, daily brief)
- [ ] **ANALYZE mode queries data:** At least 2 analysis types work (client revenue, task status)
- [ ] **MAINTAIN mode manages versions:** Can view version history and rollback
- [ ] **Mode router accurate:** >90% correct classification on test set
- [ ] **No regressions:** All existing tests pass
- [ ] **SSE streaming works:** Real-time progress for BUILD and EXECUTE modes
- [ ] **Frontend shows all modes:** Chat UI displays appropriate components per mode

### What's NOT in OSA MVP

- Cross-OS context (Phase 2 — MIOSA platform)
- Per-OS LLM provider config (follows Ollama Cloud wiring)
- CARRIER bridge to SorxMain Elixir engine (Tier 3-4 skills)
- Custom OS templates (Phase 3)
- User-created prompt templates via DB (getDBTemplate stub stays for now)
- Integration credential management UI (use config/env for now)

---

## Appendix: Quick Reference

### OSA Mode → Code Path

```
BUILD    → osa_mode_router.go → app_generation_orchestrator.go → sandbox_edit_flow.go → module_protection.go
ASSIST   → osa_mode_router.go → osa_prompt_builder.go → intent_router_v2.go (existing agents)
ANALYZE  → osa_mode_router.go → osa_analyze_service.go → database queries → LLM insight generation
EXECUTE  → osa_mode_router.go → sorx/engine.go → sorx/actions.go → external APIs
MAINTAIN → osa_mode_router.go → osa_maintain_service.go → custom_module_service.go → sandbox_edit_flow.go
```

### Key Existing Files to Know

```
internal/agents/intent_router_v2.go         515 lines  ← Mode router extends this
internal/services/osa_prompt_builder.go      361 lines  ← Template rendering
internal/services/app_generation_orchestrator.go        ← BUILD mode core
internal/services/custom_module_service.go   814 lines  ← Module CRUD
internal/services/build_event_bus.go                    ← SSE event streaming
internal/services/sandbox_deployment.go                 ← Docker sandbox
internal/container/container.go                         ← Container security
internal/sorx/engine.go                      756 lines  ← EXECUTE mode core
internal/sorx/agent_bridge.go               ~300 lines  ← Agent ↔ skill bridge
internal/sorx/actions.go                    1777 lines  ← 28 action handlers
internal/handlers/sorx.go                    457 lines  ← SORX API endpoints
```

---

**Last updated:** 2026-02-27
**Status:** ALL PHASES COMPLETE (A-F). E2E pipeline operational. CARRIER bridge to Optimal live.
**Code:** PRs #5, #7, #8, #10, #15, #16. 200+ tests pass. Build clean.
**Architecture note:** This plan covers the reactive layer (L1-L2, open source). Proactive extraction to osa-engine/ is tracked in ARCHITECTURE-SEPARATION.md.
