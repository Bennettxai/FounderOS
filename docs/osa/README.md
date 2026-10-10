# OSA — Optimal System Agent

> **The core intelligence layer of BusinessOS and the MIOSA platform.**
> OSA is a separate Elixir/OTP system that BOS communicates with via a backend-owned HTTP runtime adapter.

---

## Architecture (Updated Feb 28, 2026)

OSA is a **separate system** — it runs as its own Elixir/OTP service. BOS connects to it over HTTP through `desktop/backend-go/internal/integrations/osa`. All orchestration intelligence (5-mode classification, agent routing, PACT/BMAD, wave execution) lives in OSA, not in BOS.

> **Source of truth for the integration:** [OSA-SDK-INTEGRATION.md](OSA-SDK-INTEGRATION.md)

### How It Works

```
User Message
    │
    ▼
┌──────────────────────────────────────┐
│  BOS (Go Backend)                    │
│  chat_streaming.go                   │
│       │                              │
│       ├─ OSA_ENABLED=true?           │
│       │   ├─ YES → SDK.Stream()     │
│       │   │        SDK.Orchestrate() │
│       │   │        ↓                │
│       │   │   SSE events → frontend  │
│       │   │                          │
│       │   └─ FAIL → fallback ↓      │
│       │                              │
│       └─ NO → Local agents           │
│           SmartIntentRouter → COT    │
│           @mentions → specialists    │
└──────────────┬───────────────────────┘
               │ HTTPS + JWT
               ▼
┌──────────────────────────────────────┐
│  OSA (Elixir/OTP)                    │
│                                      │
│  Mode Router → BUILD / ASSIST /      │
│                ANALYZE / EXECUTE /    │
│                MAINTAIN              │
│                                      │
│  22+ Agents (tier-aware routing)     │
│  Wave Execution (5 waves)            │
│  PACT/BMAD Orchestrators             │
│  SORX Skill Engine                   │
│  Fleet Management                    │
│  Signal Theory Classification        │
└──────────────────────────────────────┘
```

### Two Layers

| Layer | System | Where | Autonomy | Open Source? |
|-------|--------|-------|----------|-------------|
| Base | **BusinessOS** | BOS Go backend | Reactive (L1-L2) | Yes — free tier |
| Premium | **OSA** | Separate Elixir service | Proactive (L3-L5) | Partial |

**Reactive (BusinessOS):** User sends message → local agents respond. SmartIntentRouter, COT, specialist agents (analyst, client, document, project, task), RAG, tiered context.

**Proactive (OSA):** OSA monitors, classifies into 5 modes, orchestrates 22+ agents, executes skills, generates apps. BOS delegates to OSA via SDK when `OSA_ENABLED=true`.

**Fallback:** When OSA is unavailable (circuit breaker open, service down), BOS automatically falls back to local agent routing. Zero downtime for users.

---

## Documentation Map

| Document | What It Covers |
|----------|---------------|
| **[OSA-SDK-INTEGRATION.md](OSA-SDK-INTEGRATION.md)** | **SOURCE OF TRUTH** — How BOS talks to OSA, event mapping, SDK methods, config, what was removed |
| [MODES.md](MODES.md) | The 5 modes — BUILD, ASSIST, ANALYZE, EXECUTE, MAINTAIN (now live in OSA) |
| [SORX-INTEGRATION.md](SORX-INTEGRATION.md) | How OSA connects to the SORX skill execution engine |
| [CONVERSATION-FLOW.md](CONVERSATION-FLOW.md) | User interaction patterns |
| [PROMPTS-AND-CONFIG.md](PROMPTS-AND-CONFIG.md) | Prompt templates per mode (now managed by OSA) |

### Archived (pre-SDK, when orchestration was in BOS)

These docs describe the old integrated architecture where OSA's orchestration code lived inside BOS. That code was removed on Feb 28, 2026 (~7,950 LOC deleted). They remain for historical reference:

| Document | Why Archived |
|----------|-------------|
| [OSA-DEFINITIVE-ARCHITECTURE.md](OSA-DEFINITIVE-ARCHITECTURE.md) | Describes `internal/orchestration/` which no longer exists in BOS |
| [ARCHITECTURE.md](ARCHITECTURE.md) | References `osa_orchestrator.go` junction point (now in OSA) |
| [BUILD-STRATEGY.md](BUILD-STRATEGY.md) | "Integrated, not separate" decision — reversed |

---

## Key Decisions

| Date | Decision | Details |
|------|----------|---------|
| 2026-02-23 | Integrated in Go backend | OSA orchestration built inside BOS |
| 2026-02-28 | **Reversed: Separated via SDK** | OSA moved to Elixir/OTP, BOS stripped of 7,950 LOC orchestration, SDK wires the two systems |

The separation was driven by:
- Code duplication (~7,950 LOC in BOS duplicating OSA)
- OSA's Elixir/OTP architecture (22 agents, wave execution, fleet mgmt) is more capable
- Clean separation of concerns: BOS handles business modules, OSA handles intelligence
- SDK provides circuit breaker + graceful fallback

---

**Last Updated:** 2026-02-28
