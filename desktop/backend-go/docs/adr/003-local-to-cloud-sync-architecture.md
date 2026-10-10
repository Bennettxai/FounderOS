# ADR-003: Local-to-Cloud Sync Architecture (BusinessOS + MIOSA Cloud)

## Status: Proposed

## Date: 2026-02-24

## Context

BusinessOS is designed as a self-hosted product that runs locally by default
(Docker, desktop Electron app, or bare metal). The MIOSA platform operates a
proprietary cloud at api.miosa.ai. Users who want cloud features — backup,
cross-device sync, marketplace access, team collaboration via the cloud — must
opt in explicitly by providing a MIOSA API key.

The sync system must:

1. Be entirely opt-in: a user with no MIOSA_API_KEY should have zero cloud
   surface area and no degraded experience.
2. Never move raw business data (contacts, deals, conversations, documents)
   to the cloud without an explicit "Share data" action by the user.
3. Present an identical UI and API surface regardless of mode; only the
   backing transport changes.
4. Work across three deployment targets: Docker (web), Electron (desktop), and
   bare-metal Go binary.
5. Keep the runtime boundary explicit: BusinessOS owns direct HTTP adapters for
   local OSA and MIOSA Cloud rather than routing through a checked-in SDK stub.

The OSA agent (Elixir/OTP) already exists at `internal/integrations/osa` and
is reached through `internal/integrations/osa.Client`. The sync feature uses
the same backend-owned adapter pattern for MIOSA Cloud.

## Decision

### Chosen approach: Backend-Driven Sync with Workspace Manifest

The Go backend is the single authority for all MIOSA cloud interactions. The
SvelteKit frontend never talks directly to api.miosa.ai. The Electron main
process stores the API key in the OS keychain and injects it into the backend
process on startup.

Key principles:

- The API key is stored server-side only (in the Go backend's environment or
  the Electron keychain). It is never sent to the SvelteKit frontend.
- Sync operates on a "workspace manifest": a serialized, versioned snapshot of
  configuration objects (workspace settings, agent configs, custom app
  definitions, template definitions). No row-level business data is included.
- The backend exposes two new internal endpoints: `GET /api/miosa/status` and
  `POST /api/miosa/sync`. The frontend calls only these.
- The existing `internal/integrations/osa` package gains a `Mode` field
  (`local` | `cloud`) and switches client construction accordingly.

### Sync payload definition (what crosses the boundary)

Included in the manifest:
- Workspace name, description, icon, and settings (theme, feature flags, etc.)
- Agent configurations (name, model, system prompt, tools, temperature — not
  conversation history)
- Custom app definitions (schema, layout, display config — not the app's data
  rows)
- Template definitions (structure and defaults — not documents created from
  templates)
- OSA agent connection settings (model choices, tool permissions)

Explicitly excluded:
- All rows in business tables (tasks, projects, clients, deals, contacts)
- Conversation history and AI message logs
- User credentials or OAuth tokens
- File attachments and document content
- Calendar events and email metadata
- Memory embeddings and vector store contents

### Mode switching

A new config key `OSA_MODE` (values: `local` | `cloud`) controls which
sdk-go constructor is used. When `cloud`, the config also requires
`MIOSA_API_KEY`. The `OSA_BASE_URL` and `OSA_SHARED_SECRET` remain
relevant for `local` mode only.

The two modes produce identical Go interfaces (`*osa.Client`); callers
are unaware of the difference. Only `NewClient()` in `config.go` changes
behaviour.

### Electron-specific handling

The Electron main process reads `MIOSA_API_KEY` from the OS keychain
(via `keytar`) or from the app's local `.env` at startup. It injects the
value as an environment variable into the spawned Go backend child process.
The frontend running inside Electron's BrowserWindow communicates with the
backend via the same HTTP API as the web version — there is no IPC shortcut
for the sync flow.

This keeps the web and Electron code paths identical and avoids needing
Electron preload scripts for sync.

## Consequences

### Positive

- Zero frontend changes required to add cloud sync: all routing happens in
  the Go backend based on the config.
- API key never reaches the browser; XSS cannot exfiltrate it.
- The manifest model prevents accidental raw-data sync; any future expansion
  of what syncs requires an explicit code change and ADR update.
- Adding cloud mode to OSA is a one-line change in `NewClient()`.
- The existing resilience layer (`internal/integrations/osa/resilient_client.go`)
  wraps the cloud client with the same circuit-breaker and retry logic.

### Negative

- Cloud sync is not real-time; it is a point-in-time push (user-initiated or
  periodic). Pull sync (cloud-to-local) requires a separate polling mechanism,
  not designed here.
- The manifest approach means cloud state can diverge from local state if
  the user modifies settings in the MIOSA cloud portal directly. Conflict
  resolution is out of scope for this ADR.
- Electron keychain integration (`keytar`) adds a native dependency that must
  be rebuilt per platform.

### Neutral

- The `OSA_MODE` env var is additive; existing deployments without it default
  to `local` with no behaviour change.
- The settings page gains a new "MIOSA Cloud" tab alongside existing tabs,
  consistent with the existing tab-based settings layout.

## Alternatives Considered

### Frontend-Direct Sync
The SvelteKit server-side load function calls MIOSA cloud APIs directly from
the server component, bypassing the Go backend.
Rejected because: this splits the cloud credential surface between two
servers (Go and Node), doubles the attack surface, and prevents the Go
backend's circuit-breaker resilience from applying to cloud calls.

### SDK-Sidecar (separate HTTP proxy process)
Run sdk-go as a standalone HTTP sidecar that the frontend calls directly.
Rejected because: adds a third process to manage, complicates Docker Compose
and Electron packaging, and removes the ability to use the existing
`internal/integrations/osa` abstraction layer.

### Agent-First Sync (Elixir OSA owns sync)
The OSA Elixir agent acts as the sync authority, with the Go backend
delegating to it.
Rejected because: OSA is an execution engine, not a configuration store.
Making it responsible for BusinessOS workspace manifests creates an
inappropriate dependency direction (OSA should not know about BusinessOS's
domain model).

## References

- OSA Agent: https://github.com/Miosa-osa/OSA
- Existing OSA client: `internal/integrations/osa/client.go`
- Existing OSA config: `internal/integrations/osa/config.go`
- Backend config: `internal/config/config.go`
- ADR-001: Database Isolation Strategy
- ADR-002: App Isolation Approach
