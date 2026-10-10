# OSA Runtime Integration — How BOS Talks to OSA

> **Status:** ACTIVE — Source of truth for BOS ↔ OSA communication.
> **Last Updated:** 2026-02-28

---

## Architecture Change (Feb 2026)

Previously, OSA's orchestration (5-mode classifier, PACT/BMAD, strategic reasoning, request classifier) lived **inside** the BOS Go backend at `internal/orchestration/`. This duplicated ~7,950 LOC and coupled BOS tightly to OSA's internals.

**Now:** OSA is a **separate system** (Elixir/OTP at `OptimalSystemAgent/`). BOS communicates with it through the backend-owned HTTP runtime adapter in `desktop/backend-go/internal/integrations/osa`. The 5-mode classification, agent orchestration, and proactive intelligence all run inside OSA. BOS is the client.

```
┌──────────────────────────────────────────────────┐
│                 BOS (Go Backend)                  │
│                                                   │
│  User Message → chat_streaming.go                │
│       │                                           │
│       ├─ OSA_ENABLED=true?                       │
│       │   YES → HTTP Stream() + Orchestrate()    │
│       │         ↓                                │
│       │   OSA events → event mapper → SSE        │
│       │                                           │
│       └─ NO (or OSA unavailable)                 │
│           → Local agents (SmartIntentRouter,      │
│             COT, @mentions, specialist agents)    │
│                                                   │
│  SDK Client: internal/integrations/osa/           │
│  ├── client.go          (base wrapper)            │
│  ├── resilient_client.go (circuit breaker)        │
│  ├── types.go           (BOS ↔ SDK types)        │
│  └── config.go          (connection config)       │
│                                                   │
│  Event Mapper: internal/handlers/                 │
│  └── osa_event_mapper.go (SDK → BOS events)      │
└──────────────┬───────────────────────────────────┘
               │ HTTPS + JWT (HS256)
               │
┌──────────────▼───────────────────────────────────┐
│              OSA (Elixir/OTP)                     │
│                                                   │
│  5-Mode Router (BUILD/ASSIST/ANALYZE/EXECUTE/     │
│                  MAINTAIN)                         │
│  Agent Orchestration (22+ agents, wave execution) │
│  PACT/BMAD Orchestrators                          │
│  Signal Theory Classification                     │
│  SORX Skill Engine                                │
│  Fleet Management (register, poll, heartbeat)     │
│                                                   │
│  API: /api/v1/orchestrate                         │
│       /api/v1/stream/{sessionID}                  │
│       /api/v1/classify                            │
│       /api/v1/fleet/*                             │
│       /api/v1/skills/*                            │
│       /api/v1/generate                            │
│       /health                                     │
└──────────────────────────────────────────────────┘
```

---

## Chat Flow (chat_streaming.go)

When a user sends a message in BOS:

1. **OSA routing check** — If `OSA_ENABLED=true` and `h.osaClient` is initialized:
   - Generate a `sessionID` (UUID)
   - Connect to OSA's SSE stream: `h.osaClient.Stream(ctx, sessionID)`
   - Kick off orchestration async: `h.osaClient.Orchestrate(ctx, req)` in a goroutine
   - Map incoming SDK events to BOS StreamEvents via `mapOSAEventsToStreamEvents()`
   - Stream to frontend as SSE with headers `X-OSA-Routing: true`, `X-OSA-Session-Id: {id}`
   - Save assistant response to conversation history
   - **Return** — local agents are never invoked

2. **Fallback** — If OSA is disabled, unavailable, or stream connection fails:
   - Fall through to local agent routing (SmartIntentRouter → specialist agents → COT)
   - This is the same flow as before — unchanged

The circuit breaker in `ResilientClient` protects against OSA downtime. If OSA is unreachable, the breaker opens and subsequent requests immediately fall back to local agents without waiting for timeout.

---

## Event Mapping

OSA SDK emits events via SSE. The mapper (`osa_event_mapper.go`) converts them to BOS's streaming format:

| SDK Event | BOS StreamEvent | Notes |
|-----------|-----------------|-------|
| `agent.thinking` | `thinking_chunk` | Step + content from data |
| `agent.response` | `token` | Content streamed incrementally |
| `skill.started` | `tool_call` | Tool name + "calling" status |
| `skill.completed` | `tool_result` | Tool name + result |
| `skill.failed` | `tool_result` | Tool name + error (status: "error") |
| `agent.error` | `error` | Error message |
| `connected` | *(dropped)* | Internal handshake |
| `signal.classified` | *(dropped)* | Internal classification |
| `system_event` | *(dropped)* | Internal system events |
| *(stream close)* | `done` | Signals end of stream |

---

## Signal Classification (signal_classified Event)

### Overview

The **signal_classified** event is emitted as the **FIRST event** in every OSA chat stream. It provides real-time Signal Theory classification of the incoming request before orchestration begins.

### Payload

```json
{
  "mode": "EXECUTE",
  "genre": "DIRECT",
  "doc_type": "proposal",
  "weight": 0.5,
  "confidence": 0.80
}
```

| Field | Type | Description |
|-------|------|-------------|
| `mode` | string | OSA 5-mode classification: `BUILD`, `ASSIST`, `ANALYZE`, `EXECUTE`, or `MAINTAIN` |
| `genre` | string | Signal genre: `DIRECT`, `QUERY`, `REQUEST`, `CONCERN`, `EXPLORE`, etc. |
| `doc_type` | string | Document type: `proposal`, `report`, `brief`, `memo`, `specification`, etc. |
| `weight` | float (0-1) | Signal weight/importance in the context of the current session |
| `confidence` | float (0-1) | Classifier confidence (0.0 = uncertain, 1.0 = certain) |

### Event Timing

```
User sends message
    ↓
OSA classifies signal (instant)
    ↓
signal_classified event (FIRST in stream)
    ↓
agent.thinking events (orchestration in progress)
    ↓
agent.response / skill.* events
    ↓
done (stream complete)
```

### Frontend Storage

The frontend receives this event and stores classification data in two places:

#### 1. Session State (osa.ts)
```typescript
// stores/osa.ts
activeGenre: string;        // e.g., "DIRECT"
activeDocType: string;      // e.g., "proposal"
signalWeight: number;       // 0.0-1.0
signalConfidence: number;   // 0.0-1.0
```

#### 2. Signal History (signal.ts)
```typescript
// stores/signal.ts
history: SignalClassification[];  // All classifications from this session
distributions: {
  modes: Record<string, number>;      // Count by mode
  genres: Record<string, number>;     // Count by genre
  docTypes: Record<string, number>;   // Count by doc_type
};
```

### UI Components

Three components render signal classification data:

#### SignalBadge.svelte
Compact display in chat header:
```svelte
<SignalBadge mode={activeMode} genre={activeGenre} confidence={signalConfidence} />
```

Renders as: `[EXECUTE] Direct (80%)`

#### SignalDetailPanel.svelte
Expandable panel in sidebar:
```svelte
<SignalDetailPanel
  classification={currentSignal}
  history={allSignals}
/>
```

Shows:
- Current signal details (mode, genre, doc_type, weight, confidence)
- Historical distribution charts (pie/bar graphs)
- Confidence trend over conversation

#### SignalHealthWidget.svelte
Dashboard widget showing signal health metrics:
```svelte
<SignalHealthWidget
  avgConfidence={avgConfidence}
  modeDistribution={distributions.modes}
  sessionLength={history.length}
/>
```

Shows:
- Average classifier confidence across session
- Mode distribution (pie chart)
- Session signal volume

### Backend Handler

The event mapper in `osa_event_mapper.go` **drops** the signal_classified event (line 97):

```go
case "signal.classified":
    // This event is for telemetry/UI only
    // Don't emit to frontend as a StreamEvent
    continue
```

**Why dropped?** This event is captured as part of the **debug/telemetry pipeline**, not the main chat stream. The frontend receives it through a **separate SSE channel** or via direct WebSocket telemetry (implementation depends on feature flag).

---

## Signal Health Endpoint

### GET /api/v1/signal/health

Returns real-time Signal Theory health metrics for the current session.

**Endpoint:** `GET /api/v1/signal/health?session_id={sessionID}`

**Response:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-03-05T14:32:10Z",
  "signal_health": {
    "avg_confidence": 0.84,
    "total_classifications": 12,
    "mode_distribution": {
      "EXECUTE": 6,
      "ANALYZE": 3,
      "ASSIST": 2,
      "BUILD": 1,
      "MAINTAIN": 0
    },
    "genre_distribution": {
      "DIRECT": 5,
      "REQUEST": 4,
      "QUERY": 2,
      "EXPLORE": 1
    },
    "doc_type_distribution": {
      "proposal": 4,
      "report": 3,
      "brief": 2,
      "memo": 2,
      "specification": 1
    },
    "weight_stats": {
      "avg_weight": 0.68,
      "min_weight": 0.35,
      "max_weight": 0.95
    },
    "confidence_trend": [
      { "timestamp": "2026-03-05T14:30:00Z", "confidence": 0.72 },
      { "timestamp": "2026-03-05T14:31:00Z", "confidence": 0.81 },
      { "timestamp": "2026-03-05T14:32:00Z", "confidence": 0.88 }
    ]
  },
  "status": "healthy"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | UUID of the active chat session |
| `timestamp` | ISO8601 | Server timestamp |
| `signal_health.avg_confidence` | float | Average classifier confidence (0-1) across all signals in session |
| `signal_health.total_classifications` | int | Total signal classifications in this session |
| `signal_health.mode_distribution` | object | Count by 5-mode classifier result |
| `signal_health.genre_distribution` | object | Count by genre (DIRECT, QUERY, REQUEST, etc.) |
| `signal_health.doc_type_distribution` | object | Count by document type (proposal, report, etc.) |
| `signal_health.weight_stats` | object | Average, min, max signal weight in session |
| `signal_health.confidence_trend` | array | Last 10 confidence scores with timestamps |
| `status` | string | Overall health: `healthy` (avg_confidence > 0.75), `degraded` (0.5-0.75), `poor` (<0.5) |

**HTTP Status Codes:**
- `200 OK` — Signal health retrieved successfully
- `400 Bad Request` — Missing or invalid session_id
- `404 Not Found` — Session not found (likely expired)
- `503 Service Unavailable` — OSA unavailable; local agent fallback active

**Frontend Usage:**

```typescript
// src/routes/(app)/chat/+page.svelte
import { onMount } from 'svelte';
import { signalHealth, signalHistoryStore } from '$lib/stores/signal';

onMount(async () => {
  const response = await fetch(
    `/api/v1/signal/health?session_id=${sessionId}`
  );
  const data = await response.json();
  signalHealth.set(data.signal_health);
});
```

---

## SDK Methods Available

### Core (existing)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| `Orchestrate(req)` | POST `/api/v1/orchestrate` | Run full agent orchestration loop |
| `Stream(sessionID)` | GET `/api/v1/stream/{id}` | Real-time SSE event stream |
| `Classify(message, channel)` | POST `/api/v1/classify` | Signal Theory 5-tuple classification |
| `GenerateApp(req)` | POST `/api/v1/generate` | Trigger app generation pipeline |
| `GetAppStatus(appID)` | GET `/api/v1/apps/{id}/status` | Check generation progress |
| `ListSkills()` | GET `/api/v1/skills` | Available SORX skills |
| `ExecuteSkill(name, req)` | POST `/api/v1/skills/{name}/execute` | Run a skill |
| `Remember(req)` / `Recall()` | POST/GET `/api/v1/memory` | Agent memory |
| `Health()` | GET `/health` | Health check (cached 30s) |

### Fleet (new — Feb 2026)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| `RegisterFleetAgent(agentID, caps)` | POST `/api/v1/fleet/register` | Register agent with fleet coordinator |
| `PollInstructions(agentID)` | GET `/api/v1/fleet/{id}/instructions` | Get next pending instruction (nil on 204) |
| `SendHeartbeat(agentID)` | POST `/api/v1/fleet/{id}/heartbeat` | Signal agent is alive |
| `ListFleetAgents()` | GET `/api/v1/fleet/agents` | List registered fleet agents |
| `DispatchInstruction(agentID, inst)` | POST `/api/v1/fleet/{id}/dispatch` | Send instruction to specific agent |

### Swarm (new — Feb 28, 2026)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| `LaunchSwarm(req)` | POST `/api/v1/swarm/launch` | Start multi-agent swarm (pattern + task) |
| `ListSwarms()` | GET `/api/v1/swarm` | List active swarms |
| `GetSwarm(id)` | GET `/api/v1/swarm/{id}` | Get swarm status + workers |
| `CancelSwarm(id)` | DELETE `/api/v1/swarm/{id}` | Cancel running swarm |

### Tools (new — Feb 28, 2026)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| `ListTools()` | GET `/api/v1/tools` | List all available tools (built-in + MCP) |
| `ExecuteTool(name, params)` | POST `/api/v1/tools/{name}/execute` | Run a specific tool |

### Deprecated (Feb 28, 2026)
| Method | Reason | Replacement |
|--------|--------|-------------|
| `GenerateApp()` | BOS-local operation, not an OSA endpoint | Use BOS's own generation queue |
| `GetAppStatus()` | BOS-local operation | Use BOS's own status endpoint |
| `GenerateAppFromTemplate()` | BOS-local operation | Use BOS's own template system |
| `GetWorkspaces()` | BOS-local operation | Use BOS's own workspace API |
| `ListSkills()` | Renamed to match OSA | Use `ListTools()` |
| `ExecuteSkill()` | Renamed to match OSA | Use `ExecuteTool()` |

---

## Configuration

### Environment Variables

```env
# Enable/disable OSA integration
OSA_ENABLED=true

# OSA server URL
OSA_BASE_URL=http://localhost:8089

# Shared secret for JWT auth (HS256)
OSA_SHARED_SECRET=your-shared-secret-here
```

### Config Struct (`internal/config/config.go`)

```go
OSAEnabled      bool        `mapstructure:"OSA_ENABLED"`
OSABaseURL      string      `mapstructure:"OSA_BASE_URL"`
OSASharedSecret string      `mapstructure:"OSA_SHARED_SECRET"`
```

### Local Development

BOS no longer uses a local `sdk-go` replace directive. Configure the runtime
endpoint with `OSA_BASE_URL` and `OSA_SHARED_SECRET`; the Go backend calls OSA
over HTTP through its owned adapter.

---

## What Lives Where

### In BOS (Go backend)
- Specialist agents: analyst, client, document, project, task
- SmartIntentRouter, OrchestratorCOT, AgentRegistry
- RAG, tiered context, memory hierarchy
- All business modules (CRM, projects, etc.)
- SDK client wrapper with circuit breaker
- SSE streaming infrastructure
- Event mapper (SDK events → BOS events)

### In OSA (Elixir/OTP — separate repo)
- 5-mode classification (BUILD/ASSIST/ANALYZE/EXECUTE/MAINTAIN)
- 22+ agent roster with tier-aware model routing
- Wave execution (foundation → logic → presentation → review → synthesis)
- PACT/BMAD orchestrators
- Signal Theory processing
- SORX skill engine
- Fleet management
- Learning engine (SICA pattern)
- Hook pipeline (security, budget, quality)

### In the SDK (miosa-sdk-go)
- Go client for all OSA API endpoints
- JWT auth (HS256, 15-min TTL)
- SSE streaming with event parsing
- Circuit breaker + exponential backoff (optional)
- Response caching

---

## What Was Removed From BOS

The following code was deleted from `desktop/backend-go/` on 2026-02-28:

| Path | LOC | What It Did |
|------|-----|-------------|
| `internal/orchestration/osa_modes.go` | 779 | 5-mode classifier (BUILD/ASSIST/ANALYZE/EXECUTE/MAINTAIN) |
| `internal/orchestration/osa_orchestrator.go` | 1,061 | OSA orchestration engine |
| `internal/orchestration/osa_router.go` | 264 | OSA routing logic |
| `internal/orchestration/request_classifier.go` | 326 | LLM-based request classification |
| `internal/orchestration/pact.go` | 329 | PACT orchestrator |
| `internal/orchestration/bmad.go` | 319 | BMAD orchestrator |
| `internal/orchestration/strategic_reasoning.go` | 141 | Strategic reasoning engine |
| `internal/orchestration/mode_feedback.go` | 374 | Mode feedback loop |
| `internal/orchestration/classification_types.go` | 127 | Classification type definitions |
| `internal/handlers/osa_modes.go` | 79 | `/api/osa/modes` endpoint |
| + 7 test files | ~3,151 | Tests for all above |
| **Total** | **~7,950** | |

All of this functionality now lives in OSA and is accessed via the SDK.

---

**Last Updated:** 2026-02-28
