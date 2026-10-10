# OSA Modes — The 5 Operating Modes

> Each mode has different capabilities, prompts, safety levels, and response formats.
>
> **Note (2026-02-28):** The 5-mode classification now runs inside OSA (Elixir/OTP), not inside BOS. The mode definitions below are still accurate — they just execute in OSA and are accessed via the backend-owned HTTP runtime adapter. See [OSA-SDK-INTEGRATION.md](OSA-SDK-INTEGRATION.md) for how BOS connects.

---

## Open Source vs Premium — Mode Availability

Not all modes are equal in terms of open-source availability. The split follows the reactive/proactive boundary:

| Mode | Autonomy | Open Source? | Notes |
|------|----------|-------------|-------|
| **ASSIST** | L1-L2 (reactive) | Yes — free tier | Default mode; user asks, agent answers |
| **ANALYZE** (read-only) | L1-L2 (reactive) | Yes — free tier | Read-only queries, no write operations |
| **ANALYZE** (full) | L3 (proactive) | Premium | Scheduled reports, push insights |
| **BUILD** | L3-L4 (proactive) | Premium | Code generation, sandbox, module install |
| **EXECUTE** | L3-L4 (proactive) | Premium | Skill execution, integrations, SORX |
| **MAINTAIN** | L4-L5 (proactive) | Premium | Autonomous health checks, updates |

**Reactive modes (L1-L2):** ASSIST and read-only ANALYZE operate exactly like BusinessOS reactive agents — the user initiates, the agent responds.

**Proactive modes (L3-L5):** BUILD, EXECUTE, MAINTAIN, and full ANALYZE go beyond responding. They monitor, generate, and act. These modes require OSA premium (Miosa.ai) and are not included in the open-source release.

> **See:** `docs/osa/ARCHITECTURE-SEPARATION.md` for the full open-source split specification.

---

## Mode Overview

| Mode | Purpose | Trigger | Safety | Response |
|------|---------|---------|--------|----------|
| **BUILD** | Create/modify modules and apps | "build", "create", "generate" | High (sandbox + approval) | Code generation + preview |
| **ASSIST** | Answer questions with context | Default (anything not matched) | Low (read-only) | Text + references |
| **ANALYZE** | Data insights and reports | "analyze", "report", "metrics" | Low (read-only queries) | Data tables + LLM insights |
| **EXECUTE** | Run integrations and skills | "sync", "send", "import", "run" | Medium (temperature gating) | Skill execution + results |
| **MAINTAIN** | System health and updates | "update", "upgrade", "version" | Medium (changes system state) | Health report + actions |

---

## Mode Classification Pipeline

> **Updated 2026-02-27** — Classification is now a 3-layer pipeline, not just keyword matching.

The mode router (`internal/orchestration/osa_modes.go`) classifies messages through this pipeline:

```
Message
  │
  ▼
1. Explicit mode check ─── Frontend sends {"explicit_mode": "BUILD"} → use it directly
  │
  ▼
2. Keyword scoring ─── Primary patterns (0.75) + secondary patterns (0.20)
  │                     Threshold: 0.70 → confident match
  │                     Priority: BUILD > EXECUTE > MAINTAIN > ANALYZE
  │
  ├── score >= 0.70 ──→ Use keyword result (fast path, ~0ms)
  │
  ├── score [0.4, 0.7) ─→ 3. LLM fallback (ambiguous zone)
  │                          Sends structured JSON prompt to LLM
  │                          Returns: {mode, confidence, reasoning, sub_intent}
  │                          Falls back to ASSIST on any failure
  │
  └── score < 0.4 ────→ 4. Default ASSIST (conversational, 0.6 confidence)
```

**Why the LLM fallback?** Messages like "Show me revenue trends for Q4" only hit secondary keywords (score ~0.20) but are clearly ANALYZE. The LLM catches these ambiguous cases that keyword matching misses. In testing, this catches ~15% of messages that would otherwise default to ASSIST incorrectly.

**Test coverage:** 56 unit tests + 45 E2E tests + integration tests covering all 5 modes, priority collisions, and keyword gaps.

---

### Mode Classification Architecture (Updated 2026-02-27)

The mode classifier uses a 3-layer approach:

```
User Message
    │
    ▼
Layer 1: Explicit Mode Override
    Frontend can set metadata.explicit_mode = "BUILD"|"ANALYZE"|etc.
    If set → use immediately, confidence 1.0
    │
    ▼
Layer 2: Keyword Scoring (fast path, <1ms)
    Primary keywords: 0.75 score (e.g., "build", "create", "analyze", "send")
    Secondary keywords: 0.20 score (e.g., "chart", "trending", "contacts")
    Priority order: BUILD > EXECUTE > MAINTAIN > ANALYZE
    Threshold: 0.70
    If score >= 0.70 → use keyword result
    │
    ▼
Layer 3: LLM Fallback (ambiguous zone, 1-3s)
    If highest keyword score is in [0.4, 0.7) AND LLM is available:
    → Call LLM with structured prompt, expects JSON response
    → {"mode":"BUILD","confidence":0.85,"reasoning":"...","sub_intent":"..."}
    → Falls back to ASSIST on any LLM failure
    │
    ▼
Default: ASSIST (confidence 0.6)
    Anything scoring < 0.4 on keywords → ASSIST
    Most conversational messages land here
```

**Test coverage:** 56 unit tests + 45 E2E tests + integration pipeline tests = 100+ tests covering all modes.

---

## BUILD Mode

**What it does:** Generates new modules, customizes existing ones, creates apps in a sandboxed environment. The user sees a preview before anything is applied.

**Flow:**
```
User: "Build me a feedback form module"
  │
  ▼
1. Mode Router classifies as BUILD (pattern: "build" + "module")
2. Module Protection checks: does this conflict with protected manifests?
3. PACT/BMAD orchestrator generates code:
   - Database migration SQL
   - Go handler + service + repository
   - Svelte UI components
   - Module manifest (routes, schemas, operations)
4. Generated code appears in sandbox (not live)
5. User reviews in Monaco editor
6. User approves → module installed to workspace
   User rejects → sandbox discarded
```

**Key properties:**
- Always sandboxed — generated code never runs in production without approval
- Module Protection validates before install
- SSE streams generation progress
- Files persist to `osa_generated_apps` + `osa_generated_files` tables

**Current status:** Working — 3 cascading blockers fixed (FK violation, SSE params, prompt overwrite). Module protection validates BUILD intent before PACT/BMAD dispatch. LLM fallback handles ambiguous BUILD requests (e.g., 'I need something for project tracking').

---

## ASSIST Mode

**What it does:** Answers questions using the full context of the user's workspace — their data, modules, integrations, conversation history, and memories.

**Flow:**
```
User: "What's the status of the Johnson project?"
  │
  ▼
1. Mode Router classifies as ASSIST (default — no other mode matched)
2. Routes to SmartIntentRouter v2
3. Intent Router dispatches to appropriate agent:
   - Document Agent → for document questions
   - Project Agent → for project status (this case)
   - Client Agent → for client information
   - Analyst Agent → for data analysis
   - Task Agent → for task management
4. Agent queries workspace data + LLM generates contextual response
5. Response streamed to user
```

**Key properties:**
- Read-only — never modifies data
- Has access to all workspace context (memory hierarchy: workspace → project → agent)
- Falls through to this mode when nothing else matches (confidence 0.6)
- Most common mode (majority of messages are questions)

**Current status:** Fully working — delegates to existing intent router. Graceful degradation when agent registry unavailable.

---

## ANALYZE Mode

**What it does:** Queries business data and generates insights with LLM analysis. Think "ask your data questions in natural language."

**Flow:**
```
User: "Show me revenue by client this quarter"
  │
  ▼
1. Mode Router classifies as ANALYZE (keywords: "revenue", "client", "quarter")
2. Identify data source (CRM, projects, invoices, etc.)
3. Build parameterized SQL query (safe — no SQL injection)
4. Execute query against workspace data
5. Pass results to LLM: "Given this data: [...], provide business insights"
6. Stream formatted response: data table + LLM analysis
```

**Key properties:**
- Read-only queries — never modifies data
- Uses parameterized SQL (no string concatenation — injection-safe)
- LLM generates insights, not just raw data
- Can produce charts/visualizations (Sprint 4+)

**Current status:** Fully working — queries real workspace data (app counts, queue stats, active modules) then passes to analyst agent. Nil-guard added for degraded deployments.

---

## EXECUTE Mode

**What it does:** Runs integrations and SORX skills. Connects to Gmail, Slack, HubSpot, Linear, Notion, and internal BusinessOS operations.

**Flow:**
```
User: "Sync my Gmail contacts"
  │
  ▼
1. Mode Router classifies as EXECUTE (keywords: "sync" + "gmail")
2. Pattern matcher finds skill: "gmail.sync"
3. Check: Does user have Gmail integration connected?
4. Check temperature:
   - COLD (0.0-0.3): Auto-execute without asking
   - WARM (0.3-0.7): Auto-execute read-only, ask for write ops
   - HOT (0.7-1.0): Always ask before executing
5. SORX Engine executes skill steps:
   Step 1: gmail.list_contacts() → 247 contacts
   Step 2: businessos.upsert_clients() → 12 new, 3 updated
   Step 3: agent.summarize() → "Imported 247 contacts"
6. Stream results to user
```

**Key properties:**
- Temperature gating controls what auto-executes vs what needs approval
- Skills are multi-step workflows (not single API calls)
- SORX Tier 1-2 run locally, Tier 3-4 route to Elixir (Sprint 5)
- Credential system uses encrypted per-user storage

**21 Built-in Skills:**

| Category | Skills |
|----------|--------|
| Gmail | list_messages, send_email, search |
| Calendar | list_events, create_event |
| Slack | send_message |
| HubSpot | list_contacts, create_contact |
| Linear | list_issues, create_issue |
| Notion | create_page |
| BusinessOS | create_tasks, upsert_clients, create_daily_log, list_pending_tasks |
| AI Ops | extract_actions, classify, summarize |
| Transform | map_fields, filter |

**Current status:** Working — semantic skill matching via pgvector. CARRIER routes Tier 3-4 to Optimal. Proactive scheduler for cron-like execution.

---

## MAINTAIN Mode

**What it does:** System health checks, module updates, version management, dependency auditing, and rollback capabilities.

**Flow:**
```
User: "Check if any modules need updating"
  │
  ▼
1. Mode Router classifies as MAINTAIN (keywords: "check", "updating", "modules")
2. Query module registry for installed modules
3. Compare installed versions against registry latest versions
4. Check dependencies for conflicts
5. Generate health report:
   - 3 modules up to date
   - 1 module has update available (v1.2.0 → v1.3.0)
   - 0 security vulnerabilities
6. Offer to apply updates (with user approval)
```

**Key properties:**
- Can modify system state (installs updates, rolls back versions)
- Requires user approval for any changes
- Reports real health data (DB connections, API status, module health)

**Current status:** Working — returns real DB health metrics. Nil-guard fix for graceful degradation.

---

## Mode Classification Details

### Keyword Weights

Each mode has primary keywords (weight 0.5) and secondary keywords (weight 0.2-0.25). The sum must reach threshold ≥0.7 to trigger the mode.

**EXECUTE mode keywords:**
```
Primary (0.5 each):  sync, send, import, export, run, trigger, execute, connect
Secondary (0.2-0.25): gmail, slack, hubspot, linear, calendar, notion, email, contacts
```

**ANALYZE mode keywords:**
```
Primary (0.5 each):  analyze, report, metrics, dashboard, statistics, insights
Secondary (0.2-0.25): revenue, clients, projects, growth, compare, trend
```

**MAINTAIN mode keywords:**
```
Primary (0.5 each):  update, upgrade, version, rollback, health, diagnose
Secondary (0.2-0.25): module, dependency, outdated, fix, repair, check
```

**BUILD mode** uses pattern matching + LLM fallback (not keyword scoring) because:
- "Build" is too common as a word
- False positive cost is high (generating code when user wanted information)
- LLM check adds ~200ms but prevents expensive mistakes

### Why ASSIST is Default

ASSIST is the safest default because:
1. It never modifies data (read-only)
2. It handles any message gracefully (general conversation)
3. It delegates to specialized agents that already work
4. Misclassifying as ASSIST is low-cost (user gets a helpful answer, just not the automated action)

---

## Tuning Guide

To adjust mode behavior, modify these constants in `osa_modes.go`:

| Parameter | Default | Effect of Increasing | Effect of Decreasing |
|-----------|---------|---------------------|---------------------|
| EXECUTE threshold | 0.7 | Fewer EXECUTE triggers, more ASSIST fallback | More EXECUTE triggers, risk of false positives |
| ANALYZE threshold | 0.7 | Fewer ANALYZE triggers | More ANALYZE triggers |
| MAINTAIN threshold | 0.7 | Fewer MAINTAIN triggers | More MAINTAIN triggers |
| BUILD LLM confidence | 0.85 | Fewer BUILD triggers from LLM | More BUILD triggers, risk of unwanted code gen |
| Primary keyword weight | 0.5 | Single keyword closer to threshold | Single keyword further from threshold |
| Secondary keyword weight | 0.2-0.25 | Secondary keywords matter more | Secondary keywords matter less |

**Recommendation:** Start with defaults. After Sprint 2 production deployment, collect real user messages and measure classification accuracy. Adjust weights based on data, not intuition.

---

## Mode Classification — How to Test

### Unit tests (keyword-only)
```bash
cd desktop/backend-go
go test ./internal/orchestration/ -run TestClassifyMode -v
```

### E2E tests (realistic messages)
```bash
go test ./internal/orchestration/ -run TestE2E -v
```

### Integration tests (classify → dispatch pipeline)
```bash
go test ./internal/orchestration/ -run TestFullPipeline -v
```

### Live API test (requires auth session)
```bash
# Get CSRF token
curl -s -b "better-auth.session_token=$TOKEN" http://localhost:8001/api/auth/csrf

# Classify a message
curl -s http://localhost:8001/api/osa/modes -X POST \
  -H "Content-Type: application/json" \
  -H "Cookie: better-auth.session_token=$TOKEN; csrf_token=$CSRF" \
  -H "X-CSRF-Token: $CSRF" \
  -d '{"message":"Build me a CRM dashboard"}'
```

Response:
```json
{
  "success": true,
  "mode": "BUILD",
  "confidence": 0.95,
  "reasoning": "Build keywords matched",
  "sub_intent": "keyword_match",
  "explicit": false,
  "alternatives": [...]
}
```

**Last Updated:** 2026-02-27
