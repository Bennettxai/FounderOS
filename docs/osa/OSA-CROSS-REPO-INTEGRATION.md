# OSA Cross-Repo Integration Plan

> **Status:** ACTIVE — Integration roadmap for OSA across all MIOSA repos
> **Last Updated:** 2026-02-28

---

## Vision

OSA (Optimal System Agent) is a **standalone Elixir/OTP service** that powers AI orchestration across the entire MIOSA ecosystem. It is NOT tied to any single repo. Every OS variant — CustomOS, BusinessOS, or any future template — connects to OSA the same way: through the SDK and two access surfaces.

```
┌─────────────────────────────────────────────────────────────┐
│                    MIOSA Ecosystem                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   CustomOS    │  │  BusinessOS  │  │  [Future OS]  │      │
│  │  (open-src)   │  │  (template)  │  │  (template)   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │               │
│         ▼                  ▼                  ▼               │
│  ┌──────────────────────────────────────────────────┐       │
│  │           Go Backend (per OS instance)            │       │
│  │                                                    │       │
│  │  Two access surfaces to OSA:                      │       │
│  │                                                    │       │
│  │  1. CHAT (UI Pill / Chat Panel)                   │       │
│  │     User types → SSE stream → OSA HTTP adapter     │       │
│  │     Frontend renders: SignalBadge, ToolCallCard    │       │
│  │                                                    │       │
│  │  2. TERMINAL (Desktop Terminal Icon)              │       │
│  │     User opens terminal → xterm.js + WebSocket    │       │
│  │     OSA CLI runs inside container or local shell   │       │
│  │     Full command set: /agents, /tiers, /swarm...  │       │
│  │                                                    │       │
│  │  Runtime adapter: internal/integrations/osa       │       │
│  └──────────────────┬───────────────────────────────┘       │
│                      │ HTTPS + JWT                           │
│                      ▼                                       │
│  ┌──────────────────────────────────────────────────┐       │
│  │          OSA (Elixir/OTP — standalone)            │       │
│  │                                                    │       │
│  │  5-Mode Router · 22+ Agents · Wave Execution     │       │
│  │  PACT/BMAD · Signal Theory · SORX Skills         │       │
│  │  Fleet Management · Swarm Orchestration           │       │
│  │  18 LLM Providers · 12 Channels                   │       │
│  │  Hook Pipeline · Learning Engine                   │       │
│  │                                                    │       │
│  │  Runs as: Docker container OR local Elixir        │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

---

## Repos Involved

| Repo | What It Is | OSA Role |
|------|-----------|----------|
| `OptimalSystemAgent/` | OSA core (Elixir/OTP) | **The brain** — all AI orchestration lives here |
| `miosa-sdk-go/` | Go SDK for OSA API | **The bridge** — any Go backend talks to OSA through this |
| `CustomOS-osa/` | Open-source base OS (SvelteKit + Go + Postgres) | **Client** — uses SDK, exposes chat + terminal |
| `BOS/` | BusinessOS (CustomOS + 15 business modules) | **Client** — same integration, more modules |
| `miosa-frontend/` | MIOSA platform frontend | **Platform layer** — tenant/OS management |

---

## Two Access Surfaces

### Surface 1: Chat (UI Pill / Chat Panel)

The chat interface is the primary way users interact with OSA through the browser.

**How it works:**
1. User types in chat panel (or clicks the OSA pill)
2. Frontend sends message to Go backend (`POST /api/chat`)
3. Backend checks `OSA_ENABLED` → calls `SDK.Stream()` + `SDK.Orchestrate()`
4. OSA classifies mode (BUILD/ASSIST/ANALYZE/EXECUTE/MAINTAIN)
5. OSA dispatches agents, streams SSE events back
6. Backend maps events → frontend renders (SignalBadge, ToolCallCard, thinking blocks)

**Frontend components:**
- `chatSSEParser.ts` — parses 12 typed SSE event types
- `SignalBadge.svelte` — shows classified mode (BUILD = blue, ASSIST = green, etc.)
- `ToolCallCard.svelte` — renders tool invocations inline
- Chat `+page.svelte` — full SSE event dispatch

**Works in:** CustomOS, BusinessOS, any future OS template

### Surface 2: Terminal (Desktop Terminal Icon)

The terminal icon in the desktop dock opens a full terminal emulator where OSA runs directly.

**How it works:**
1. User clicks terminal icon in desktop dock
2. `Terminal.svelte` opens with xterm.js + WebSocket connection
3. Terminal connects to a shell session (Docker container or local shell)
4. OSA CLI is available inside the terminal (`osa` command)
5. Full command set: `/agents`, `/tiers`, `/swarm`, `/hooks`, `/learning`, 60+ commands
6. Same 5-mode routing, same agents, same Signal Theory — just CLI instead of chat UI

**Frontend components:**
- `Terminal.svelte` — xterm.js with FitAddon, WebLinksAddon, SearchAddon
- `Dock.svelte` — terminal icon in desktop dock
- Terminal page route: `/terminal`

**Works in:** CustomOS (built-in), BusinessOS (inherited), any OS with desktop mode

---

## Running OSA

OSA can run in two modes. The OS doesn't care which — it talks to the same HTTP API either way.

### Option 1: Docker (recommended for production & isolation)

```bash
# Start OSA as a Docker container
docker run -d \
  --name osa \
  -p 8089:8089 \
  -e ANTHROPIC_API_KEY=sk-ant-... \
  -e OSA_SECRET_KEY=your-secret \
  miosa/optimal-system-agent:latest

# Or with docker-compose (alongside Postgres + Redis)
docker-compose up -d
```

The terminal in CustomOS/BusinessOS can also spawn **per-user Docker containers** for sandboxed shell access. OSA runs in its own container, terminal sessions run in separate containers.

### Option 2: Local (no Docker required)

```bash
# Install Elixir/OTP (if not installed)
# macOS: brew install elixir
# Linux: asdf install erlang && asdf install elixir

cd OptimalSystemAgent
cp .env.example .env
# Edit .env — set ANTHROPIC_API_KEY (or any provider key)

mix deps.get
mix phx.server    # Starts on :8089
```

OSA auto-detects available LLM providers at boot:
1. `OSA_DEFAULT_PROVIDER` env var (if set)
2. `ANTHROPIC_API_KEY` → Claude
3. `OPENAI_API_KEY` → OpenAI
4. `GROQ_API_KEY` → Groq
5. `OPENROUTER_API_KEY` → OpenRouter
6. Fallback → Ollama (fully local, no API key needed)

### Connecting the OS to OSA

```env
# In the OS backend .env (CustomOS or BusinessOS)
OSA_ENABLED=true
OSA_BASE_URL=http://localhost:8089     # or Docker service URL
OSA_SHARED_SECRET=your-shared-secret   # JWT signing key (HS256)
```

That's it. The SDK handles auth, streaming, circuit breaking, and fallback.

---

## Integration Per Repo

### CustomOS-osa (open-source)

CustomOS is the **blank canvas**. It ships with:
- Desktop environment (windows, dock, 3D mode)
- Terminal emulator (xterm.js + WebSocket + Docker sandbox)
- Chat interface (SSE streaming)
- 9 OAuth connectors
- Module system (install/uninstall dynamic modules)

**OSA integration points:**
1. Chat panel → SDK → OSA (same as BOS)
2. Terminal icon → shell session → `osa` CLI available
3. Module install triggered by BUILD mode (OSA generates → sandbox → approve → install)
4. No business modules pre-installed — user adds what they need

**What CustomOS users get from OSA:**
- 5-mode AI routing (BUILD/ASSIST/ANALYZE/EXECUTE/MAINTAIN)
- 22+ specialized agents
- Multi-agent swarms for complex tasks
- SORX skill engine (extensible tools)
- Proactive monitoring (MAINTAIN mode)
- Fleet management for multi-agent coordination

### BusinessOS (BOS)

BusinessOS = CustomOS + 15 pre-installed business modules (CRM, Projects, Tasks, Calendar, etc.)

**Same OSA integration as CustomOS**, plus:
- Business-context agents (analyst, client, document, project, task)
- RAG pipeline with business data (pgvector semantic search)
- Tiered context system for business conversations
- Module protection (14 core modules can't be removed)

### Future OS Templates

Any new OS template (AgencyOS, DevOS, etc.) gets OSA for free:
1. Fork CustomOS
2. Add template-specific modules
3. Set `OSA_ENABLED=true` in backend config
4. OSA works immediately — chat + terminal

---

## Module System + OSA BUILD Mode

When OSA classifies a request as BUILD mode, it can generate new modules:

```
User: "Build me a CRM module"
  ↓
OSA: BUILD mode → agent orchestration (wave execution)
  ↓
Generated: Module manifest + migration SQL + routes + components
  ↓
Sandbox: Fork current state → validate → preview diff
  ↓
User: Approve or reject in UI (SandboxPreview component)
  ↓
If approved: Apply migration, register routes, install module
```

**Sandbox endpoints (Pedro building):**
```
POST   /api/sandbox/fork           Create sandbox copy
POST   /api/sandbox/:id/validate   Validate sandboxed version
GET    /api/sandbox/:id/preview    Return diff preview
POST   /api/sandbox/:id/apply      Apply changes to live
POST   /api/sandbox/:id/reject     Discard sandbox
```

**Module lifecycle (Sprint 4):**
- Install: Download manifest → run migration → register routes → add to dock
- Uninstall: Check protection → remove routes → rollback migration → remove from dock
- Customize: Fork to sandbox → edit → validate → apply

---

## Testing OSA Integration

### Local dev setup (all three terminals)

```bash
# Terminal 1 — OSA
cd ~/Desktop/MIOSA/OptimalSystemAgent
export ANTHROPIC_API_KEY=sk-ant-...
mix phx.server                          # :8089

# Terminal 2 — OS backend (BOS or CustomOS)
cd ~/Desktop/MIOSA/_ACTIVE/BOS/desktop/backend-go  # or CustomOS-osa backend
export OSA_BASE_URL=http://localhost:8089
export OSA_ENABLED=true
export OSA_SHARED_SECRET=dev-secret
go run ./cmd/server                     # :8001

# Terminal 3 — Frontend
cd ~/Desktop/MIOSA/_ACTIVE/BOS/frontend  # or CustomOS-osa frontend
npm run dev                              # :5173
```

### Verification

| Test | How | Expected |
|------|-----|----------|
| OSA health | `curl http://localhost:8089/health` | `{"status":"ok"}` |
| Chat via OSA | Send message in chat UI | Response header: `X-OSA-Routing: true` |
| Mode classification | Send "build me a CRM" | SignalBadge shows BUILD |
| Fallback | Kill OSA, send message | Local agents respond, no crash |
| Terminal | Click terminal icon, type `osa --help` | OSA CLI available |
| Swarm | `curl -X POST localhost:8001/api/osa/swarm/launch -d '{...}'` | Swarm starts |
| Tools | `curl localhost:8001/api/osa/tools` | Tool list returned |

### Stress testing (Pedro — Sprint 4)
- 50x rapid module install/uninstall cycles
- Concurrent module installs (3+ simultaneous)
- Large module manifests (100+ tables)
- Protected module removal attempts (should fail gracefully)
- OSA circuit breaker: kill/restart OSA during active chat

---

## What's Still Needed

| Item | Owner | Status |
|------|-------|--------|
| SORX tools → ToolCallCard frontend flow | Javaris | Not started |
| Proactive CARRIER events in frontend | Javaris | Not started |
| Swarm output → sandbox container connection | Roberto | Not started |
| Terminal ↔ OSA CLI wiring in Docker | Pedro | Not started |
| CustomOS-osa repo OSA SDK integration | Roberto | Planned |
| Module install/uninstall E2E flow | Pedro + Javaris | Sprint 4 |
| Per-mode prompt refinement | Roberto | Backlog |

---

**Last Updated:** 2026-02-28
