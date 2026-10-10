# OSA Architecture (ARCHIVED)

> **SUPERSEDED (2026-02-28):** This doc describes the old integrated architecture where OSA's orchestration lived inside BOS at `internal/orchestration/`. That code has been removed. OSA is now a separate Elixir/OTP service, and BOS communicates with it via a backend-owned HTTP runtime adapter.
>
> **Signal Theory Status (2026-03-05):** All 7 layers are now live in production. FastClassifier produces SignalEnvelope on every message. GenreEnricher provides mode-agnostic structure + writing style annotations. No-gate architecture: additive metadata only, LLM self-routes. Frontend receives signal_classified SSE events. Commit d33a890, 40 files. See OSA-ARCHITECTURE.md section 13 for implementation details.
>
> **Current source of truth:** [OSA-SDK-INTEGRATION.md](OSA-SDK-INTEGRATION.md)

> How the Optimal System Agent works — from user message to response.

---

## Where OSA Sits — Reactive vs Proactive

OSA does not operate in isolation. It sits on top of the **BusinessOS reactive agent system** and extends it with proactive capabilities. Understanding this layering is essential.

```
┌─────────────────────────────────────────────────┐
│              OSA (Proactive Layer)               │
│   BUILD | EXECUTE | MAINTAIN | ANALYZE (full)    │
│   L3-L5 autonomy — Premium (Miosa.ai)           │
├─────────────────────────────────────────────────┤
│          BusinessOS (Reactive Layer)             │
│  ASSIST mode + read-only ANALYZE                │
│  Specialist agents: analyst, client, document,  │
│  project, task — SmartIntentRouter — RAG        │
│  L1-L2 autonomy — Open Source                   │
└─────────────────────────────────────────────────┘
         ▲
         │  every message enters here
         │
  osa_orchestrator.go  ← junction point
```

**Reactive (BusinessOS, L1-L2):** The user encodes intent into a message. The agent processes and responds. User is always the initiating actor. This is the open-source base layer.

**Proactive (OSA, L3-L5):** OSA encodes context, generates output, and surfaces it for human review — without waiting for a prompt. This is the premium layer.

> **See:** `docs/osa/ARCHITECTURE-SEPARATION.md` for the full reactive/proactive split specification, including what gets open sourced vs what stays proprietary.

---

## Overview

OSA processes every user message through a classification → routing → execution → response pipeline. The entire flow happens in the Go backend, with LLM calls made to external providers (Anthropic, Groq, Ollama).

## Pipeline

```
1. USER MESSAGE arrives via POST /api/v2/chat/send
       │
2. MODE CLASSIFICATION (osa_modes.go)
   │   6-priority chain evaluates the message:
   │   P1: Explicit mode (user selected from dropdown)     → confidence 1.0
   │   P2: BUILD detection (pattern + LLM)                 → confidence 0.85-0.95
   │   P3: EXECUTE keyword scoring                         → threshold ≥0.7
   │   P4: MAINTAIN keyword scoring                        → threshold ≥0.7
   │   P5: ANALYZE keyword scoring                         → threshold ≥0.7
   │   P6: DEFAULT → ASSIST                                → confidence 0.6
   │
3. MODE HANDLER executes (osa_orchestrator.go)
   │   Each mode has its own handler with different:
   │   - System prompt (defines agent personality and capabilities)
   │   - Tool access (which SORX skills are available)
   │   - Safety level (what requires human approval)
   │   - Response format (text, code, data table, progress stream)
   │
4. LLM CALL (services/anthropic.go)
   │   Mode-specific prompt + user context → LLM provider
   │   Streaming response via SSE to frontend
   │
5. RESPONSE delivered to user via SSE stream
```

## Key Components

### Mode Router (`internal/orchestration/osa_modes.go`)

The brain that decides what to do with a message. Uses a 6-priority classification chain:

```go
// Priority 1: User explicitly chose a mode (dropdown)
if req.ExplicitMode != "" {
    return ModeResult{Mode: req.ExplicitMode, Confidence: 1.0}
}

// Priority 2: BUILD detection (expensive — uses LLM if pattern match is ambiguous)
if score := buildPatternScore(message); score > 0.7 {
    return ModeResult{Mode: BUILD, Confidence: score}
}

// Priority 3-5: Keyword scoring for EXECUTE, MAINTAIN, ANALYZE
// Each mode has primary keywords (weight 0.5) and secondary (weight 0.2-0.25)
// Sum must reach threshold ≥0.7 to trigger

// Priority 6: Default to ASSIST (preserves all existing behavior)
return ModeResult{Mode: ASSIST, Confidence: 0.6}
```

**Design decision:** BUILD uses expensive LLM fallback because misclassifying "build me an app" as ASSIST is costly (user gets a text answer instead of generated code). Other modes use cheap keyword patterns because the cost of misclassification is low (ASSIST handles everything gracefully).

### Orchestrator (`internal/orchestration/osa_orchestrator.go`)

Routes the classified mode to the correct handler:

```go
func (o *OSAOrchestrator) HandleMessage(ctx context.Context, msg Message) Response {
    mode := o.modeRouter.Classify(msg)

    switch mode.Mode {
    case BUILD:    return o.handleBuildMode(ctx, msg)
    case ASSIST:   return o.handleAssistMode(ctx, msg)
    case ANALYZE:  return o.handleAnalyzeMode(ctx, msg)
    case EXECUTE:  return o.handleExecuteMode(ctx, msg)
    case MAINTAIN: return o.handleMaintainMode(ctx, msg)
    }
}
```

### Module Protection (`internal/services/module_protection.go`)

Safety layer that prevents users from breaking core functionality:

```
4 Protection Pattern Types:
1. PROTECTED_ROUTES    — Core UI routes that can't be removed
2. PROTECTED_SCHEMAS   — Core DB tables that can't be dropped
3. PROTECTED_OPS       — Core operations that can't be disabled
4. PROTECTED_CONFIG    — Core settings that can't be overwritten

Example: User says "remove the dashboard page"
→ Module Protection checks: is "dashboard" a protected route?
→ Yes → Block the operation, explain why
→ No → Allow BUILD mode to proceed
```

### SORX Engine (`internal/sorx/engine.go`)

Skill execution layer. When EXECUTE mode triggers, SORX handles the actual work:

```
SORX has 4 tiers:
  Tier 1: Deterministic (no AI)  — direct API calls, DB inserts
  Tier 2: Structured AI (Haiku)  — extract, classify, summarize
  Tier 3: Reasoning AI (Sonnet)  — MCTS, Boardroom (via CARRIER)
  Tier 4: Generative AI (Opus)   — code generation, verification (via CARRIER)

Current: Tier 1-2 run locally in Go. Tier 3-4 fallback to direct LLM call.
Future:  Tier 3-4 route through CARRIER to Elixir SorxMain (Sprint 5).
```

## Data Flow Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                        Go Backend                              │
│                                                                │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐                │
│  │ Handler  │───▶│ OSA Mode │───▶│ Orchest- │                │
│  │ (HTTP)   │    │ Router   │    │ rator    │                │
│  └──────────┘    └──────────┘    └────┬─────┘                │
│                                       │                       │
│       ┌───────────┬──────────┬────────┼────────┐             │
│       ▼           ▼          ▼        ▼        ▼             │
│  ┌─────────┐ ┌────────┐ ┌────────┐ ┌──────┐ ┌────────┐     │
│  │  BUILD  │ │ ASSIST │ │ANALYZE │ │EXEC  │ │MAINTAIN│     │
│  │         │ │        │ │        │ │      │ │        │     │
│  │ PACT/   │ │ Intent │ │ SQL    │ │ SORX │ │ Health │     │
│  │ BMAD    │ │ Router │ │ Query  │ │Engine│ │ Check  │     │
│  └────┬────┘ └───┬────┘ └───┬────┘ └──┬───┘ └───┬────┘     │
│       │          │          │         │         │            │
│       └──────────┴──────────┴─────────┴─────────┘            │
│                          │                                    │
│                    ┌─────▼─────┐                              │
│                    │ LLM Call  │                              │
│                    │ (Anthropic│                              │
│                    │  / Groq)  │                              │
│                    └─────┬─────┘                              │
│                          │                                    │
│                    ┌─────▼─────┐                              │
│                    │ SSE Stream│──────────────────────────┐   │
│                    └───────────┘                          │   │
│                                                          │   │
└──────────────────────────────────────────────────────────┼───┘
                                                           │
                                                           ▼
                                                    ┌──────────┐
                                                    │ Frontend  │
                                                    │ (Svelte)  │
                                                    └──────────┘
```

## Configuration Points

These are the knobs you can tune to change OSA behavior:

| Config | Where | What It Controls |
|--------|-------|-----------------|
| Mode keyword weights | `osa_modes.go` constants | Which messages trigger which modes |
| Confidence thresholds | `osa_modes.go` constants | How certain OSA must be to select a non-default mode |
| System prompts per mode | `osa_orchestrator.go` | The personality and instructions for each mode |
| SORX skill registry | `sorx/engine.go` (currently), DB (Sprint 3) | Which skills are available for EXECUTE mode |
| Temperature gating | `sorx/engine.go` | Which skills auto-execute vs require approval |
| Module protection rules | `module_protection.go` | What core functionality is protected from modification |
| LLM provider | `services/anthropic.go` + env vars | Which LLM provider and model to use |

## Key Files

| File | Purpose | Lines |
|------|---------|-------|
| `internal/orchestration/osa_modes.go` | Mode classification engine | 349 |
| `internal/orchestration/osa_modes_test.go` | Mode classification tests | 56 tests |
| `internal/orchestration/osa_orchestrator.go` | Mode routing and handlers | ~550 |
| `internal/services/module_protection.go` | Core protection service | 306 |
| `internal/services/module_protection_test.go` | Protection tests | 48 tests |
| `internal/sorx/engine.go` | SORX skill execution | — |
| `internal/sorx/actions.go` | 30 action handlers | 1,777 |
| `internal/services/sorx.go` | SORX service + credentials | 703 |
| `internal/handlers/sorx.go` | SORX HTTP API | 457 |
| `internal/agents/intent_router_v2.go` | ASSIST mode routing | — |
| `internal/services/anthropic.go` | LLM provider interface | — |
| `internal/services/build_event_bus.go` | SSE streaming for BUILD | — |

---

**Last Updated:** 2026-02-26
