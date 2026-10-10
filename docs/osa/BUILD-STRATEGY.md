# OSA Build Strategy — Integrated vs Separate (ARCHIVED)

> **SUPERSEDED (2026-02-28):** This decision was reversed. OSA has been extracted to a separate Elixir/OTP service. BOS orchestration code (~7,950 LOC) was removed and replaced with SDK integration.
>
> **Current source of truth:** [OSA-SDK-INTEGRATION.md](OSA-SDK-INTEGRATION.md)

> Architecture decision: Should OSA be built into BusinessOS or as a separate service?

---

## Extraction Plan: ~~OSA Stays Built-In (For Now)~~ REVERSED — OSA is now separate.

~~OSA is currently integrated inside the Go backend (`desktop/backend-go/`). This is intentional and correct for the current team size and stage.~~
**Update:** OSA was extracted on Feb 28, 2026. It runs as an Elixir/OTP service. BOS connects through the backend-owned HTTP runtime adapter.

**Current state:** OSA lives inside the monolith. The 5-mode router, orchestrator, SORX engine, and module protection are Go packages within the same binary. Zero extraction overhead.

**Future extraction path:** When the team grows beyond 10 engineers or OSA needs independent scaling, the Go interfaces already defined in this codebase (`OSARouter`, `OSAOrchestrator`, `SORXEngine`, `ModuleProtection`) allow clean extraction to a separate `osa-engine/` service. The internal interfaces would become HTTP/gRPC clients — no other changes needed.

```
Current:                          Future (10+ engineers):
desktop/backend-go/               desktop/backend-go/
  internal/orchestration/  ──→      internal/clients/osa_client.go
  internal/sorx/            ──→
                                  osa-engine/  (new repo, separate deploy)
                                    internal/orchestration/
                                    internal/sorx/
```

**Trigger conditions for extraction:** team >10 engineers, OSA p99 latency blocking other endpoints, or independent scaling requirements.

> **See also:** `docs/osa/ARCHITECTURE-SEPARATION.md` for what stays in BusinessOS (reactive, open-source) vs what moves to `osa-engine/` (proactive, premium).

---

## The Question

OSA can be built two ways:

1. **Integrated** — OSA lives inside the Go backend, shares the same process, direct function calls
2. **Separate** — OSA runs as its own service, communicates with BusinessOS via API endpoints

## Decision: Integrated Core + Separate SORX Tiers

**OSA core (modes, routing, orchestration) = INTEGRATED in the Go backend.**
**SORX Tier 3-4 (advanced reasoning) = SEPARATE via CARRIER to Elixir.**

This is a hybrid approach that gives us the benefits of both:

## Why Integrated for OSA Core

| Factor | Integrated | Separate Service |
|--------|-----------|-----------------|
| **Latency** | ~0ms (function call) | +50-200ms (HTTP/gRPC) |
| **Complexity** | 1 deploy, 1 process | 2 deploys, service discovery, health checks |
| **Data access** | Direct DB access | Needs auth tokens, API layer |
| **Cost** | Same Cloud Run instance | Additional Cloud Run instance |
| **Development speed** | Change and test immediately | Contract-first, deploy both |
| **User context** | Available in-process (session, workspace, permissions) | Must be passed via headers/tokens |

**For a 3-person team building an MVP, integrated is the right choice.** The overhead of a separate service (separate deploy, API contracts, auth tokens, service discovery) doesn't justify the architectural purity until you have 10+ engineers or need independent scaling.

### What "Integrated" Means in Practice

```
Single Go Binary (cmd/server/main.go)
├── HTTP Handlers (internal/handlers/)
│   ├── /api/v2/chat/send          → OSA orchestrator
│   ├── /api/osa/modes             → Mode classification
│   ├── /api/osa/skills            → Skill listing
│   └── /api/sorx/execute          → Skill execution
│
├── OSA Core (internal/orchestration/)
│   ├── osa_modes.go               → Mode router
│   └── osa_orchestrator.go        → Mode handlers
│
├── Services (internal/services/)
│   ├── module_protection.go       → Safety layer
│   ├── sorx.go                    → SORX service
│   └── anthropic.go               → LLM provider
│
└── SORX Engine (internal/sorx/)
    ├── engine.go                  → Tier 1-2 execution
    └── actions.go                 → 30 action handlers
```

Everything in one process. One `go build`. One Cloud Run deploy. One health check.

## Why Separate for SORX Tier 3-4

The Elixir SorxMain is different:
- It's written in a different language (Elixir, not Go)
- It has different scaling characteristics (long-running reasoning tasks)
- It uses different infrastructure (BEAM VM, not Go runtime)
- It's not needed for MVP (Tier 1-2 handle 80% of use cases)

```
Go Backend (Cloud Run)              Elixir SorxMain (Cloud Run)
┌──────────────────────┐           ┌──────────────────────┐
│ OSA Core             │           │ MCTS Engine          │
│ Tier 1-2 Skills      │──AMQP──→ │ Boardroom            │
│ Module Protection    │◀─AMQP──  │ Critic               │
│ All HTTP endpoints   │           │ (Tier 3-4 only)      │
└──────────────────────┘           └──────────────────────┘
```

**CARRIER bridge** handles the communication:
- Publish skill request to RabbitMQ queue
- Elixir consumes, processes with MCTS/Boardroom
- Publish result back to Go
- Go streams result to user

## API Design

Even though OSA is integrated, the API is clean enough to extract later if needed:

### Core OSA Endpoints

```
POST   /api/v2/chat/send              Main entry — message → OSA → response
GET    /api/osa/modes/classify         Classify a message into a mode (without executing)
GET    /api/osa/skills                 List available skills with integration status
POST   /api/sorx/execute              Execute a specific skill directly
GET    /api/sorx/executions/:id       Get execution status/results
GET    /api/osa/modules               List installed modules
POST   /api/osa/modules/install       Install a module from registry
DELETE /api/osa/modules/:id           Uninstall a module
```

### Internal Interfaces

If we ever need to extract OSA to a separate service, these Go interfaces define the boundary:

```go
// OSARouter — classifies messages into modes
type OSARouter interface {
    Classify(ctx context.Context, msg Message) ModeResult
}

// OSAOrchestrator — handles messages by mode
type OSAOrchestrator interface {
    HandleMessage(ctx context.Context, msg Message) Response
}

// SORXEngine — executes skills
type SORXEngine interface {
    ExecuteSkill(ctx context.Context, skillSlug string, params map[string]any) (*Execution, error)
    ListSkills(ctx context.Context) []Skill
}

// ModuleProtection — validates module changes
type ModuleProtection interface {
    Validate(ctx context.Context, manifest ModuleManifest) (*ValidationResult, error)
}
```

These interfaces are already used internally. Extracting to a separate service would mean implementing these interfaces as HTTP/gRPC clients instead of direct calls.

## When to Consider Extraction

Extract OSA to a separate service when:

| Signal | Threshold | Why |
|--------|-----------|-----|
| Team size | >10 engineers | Multiple teams need independent deploy cycles |
| Latency requirements | OSA p99 >5s consistently | OSA compute blocking other endpoints |
| Scaling mismatch | OSA needs 10x more instances than API | Independent scaling saves cost |
| Language change | Considering Python/Rust for OSA | Can't mix languages in one binary |

**For now (team of 3-4, pre-launch), integrated is correct.**

## Development Workflow

### Making OSA Changes

1. Modify code in `internal/orchestration/` or `internal/sorx/`
2. Run tests: `cd desktop/backend-go && go test ./internal/orchestration/ ./internal/sorx/`
3. Build: `go build ./cmd/server`
4. Test locally: `go run ./cmd/server` → hit `/api/v2/chat/send`
5. Deploy: push to main → CI/CD → Cloud Run

### Making SORX Tier 3-4 Changes (Sprint 5+)

1. Modify code in the Elixir repo (`robertohluna/sorx`)
2. Test locally: `mix test`
3. Deploy Elixir service to Cloud Run
4. Verify CARRIER bridge connectivity
5. No changes needed in Go backend (AMQP protocol is stable)

---

## Summary

| Component | Strategy | Why |
|-----------|----------|-----|
| Mode Router | Integrated (Go) | Needs <1ms latency, direct data access |
| Orchestrator | Integrated (Go) | Same process as handlers, shares context |
| Module Protection | Integrated (Go) | Must validate synchronously before install |
| SORX Tier 1-2 | Integrated (Go) | Fast skills, direct API calls |
| SORX Tier 3-4 | Separate (Elixir) | Different language, long-running tasks |
| Frontend | Separate (SvelteKit) | Already separate, deployed to Vercel/Cloud Run |

---

## Speed Optimization Plan — BUILD Mode

### Current Bottleneck: Claude Agent SDK

The app generation pipeline uses `github.com/severity1/claude-agent-sdk-go@v0.6.12`. This SDK does NOT call the Claude API directly — it **spawns the full Claude Code CLI as a subprocess** per agent.

```
SDK source (client.go:248-254):
  cliPath, err := cli.FindCLI()       // finds `claude` binary on disk
  c.transport = subprocess.New(cliPath, c.options, false, "sdk-go-client")
```

**Impact:** 4 agents = 4 CLI processes. Each process: spawn → initialize → connect → make API call → return text → exit. Massive overhead for what is ultimately a single LLM API call per agent.

**Workers only collect text:** `result.Text()` — no tool use, no file editing, no code execution. The entire power of the Claude Code CLI is wasted.

### Speed Cuts (Prioritized)

**Cut 1 (Easy, ~1 hour): Kill the planning step**

`orchestrator.go:CreatePlan()` calls Opus with 5 max turns to "create a plan," but the result is never used. The 4 agent tasks are hardcoded at `orchestrator.go:178-182`. This step burns ~1-2 minutes and an Opus call for nothing.

Action: Remove `CreatePlan()` from `Generate()` flow.

**Cut 2 (Easy, ~30 min): Reduce agent turns to 1**

Each agent runs with `sdk.WithMaxTurns(5)` but only produces text. No tool use, no iteration needed. One turn is enough for "here is the code."

Action: Change `sdk.WithMaxTurns(5)` to `sdk.WithMaxTurns(1)` in `worker.go`.

**Cut 3 (Medium, ~4 hours): Replace SDK with direct API calls**

Replace the Agent SDK subprocess calls with direct Anthropic API calls using the existing `anthropic.go` service that already works for all other modes.

Before:
```go
sdk.WithClient(ctx, func(client sdk.Client) {
    result, _ := client.SendMessage(ctx, prompt)
    output = result.Text()
}, sdk.WithModel("sonnet"), sdk.WithMaxTurns(5))
```

After:
```go
response, _ := anthropicService.SendMessage(ctx, anthropic.MessageRequest{
    Model:    "claude-sonnet-4-20250514",
    Messages: []anthropic.Message{{Role: "user", Content: prompt}},
    System:   modePrompt,
})
output = response.Content
```

Action: Rewrite `worker.go` to use `anthropicService` instead of SDK.

### Expected Results

| Metric | Before Cuts | After All 3 Cuts |
|--------|-------------|-------------------|
| LLM calls | ~21 (1 plan + 4×5 turns) | 4 (4 agents × 1 call) |
| Total time | ~5-7 min | ~45-90 sec |
| Cost per generation | ~$0.50-1.00 | ~$0.10-0.20 |
| CLI processes spawned | 5 (1 plan + 4 agents) | 0 |

### Dead Code: `internal/integrations/osa/client.go`

This 277-line file is an HTTP client that talks to `${OSA_BASE_URL}/api/orchestrate` — an external OSA service that was never deployed.

- Wired into `pact.go:265` and `bmad.go:265` for BUILD mode
- Has JWT auth, retry logic, circuit breaking — all dead
- When `osaClient` is nil (always), orchestrators skip external calls gracefully
- Referenced in `osa_orchestrator.go:21` (optional field) and `:403` (nil-guarded)

**Verdict:** Should be deprecated. Proves Nick planned for a separate service architecture but never built the other side.

See [OSA-DEFINITIVE-ARCHITECTURE.md](OSA-DEFINITIVE-ARCHITECTURE.md) for the full analysis of why integrated is the correct decision.

---

**Last Updated:** 2026-02-26
