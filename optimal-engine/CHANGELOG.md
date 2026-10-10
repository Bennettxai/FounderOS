# Changelog

All notable changes to the Optimal Engine are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/).

## [0.3.0] — 2026-06-11

### Added
- **Workspace initiation** — local CLI setup path for creating a workspace from
  messy context while preserving the setup dump as evidence.
- **Workspace topology spine** — canonical organization/workspace/Node model with
  standard Node Types, relationships, memberships, aliases, lifecycle state, and
  topology review paths.
- **Source-first Memory Core** — Source Packages, Claims, Facts, Memory Objects,
  Relationship Edges, and Derivation Ledger records now form the governed truth
  lifecycle.
- **Claim promotion gate** — observations and extracted assertions become pending
  Claims first; they cannot silently become Facts without review/policy.
- **Scope envelope** — intake, assets, connectors, retrieval, and API paths carry
  tenant/workspace/Node scope instead of relying on loose defaults.
- **Governed Context Packages** — retrieval now assembles structured context with
  authorization envelopes, evidence links, memory objects, asset projections, and
  refresh metadata.
- **Active Memory Pool spine** — task-scoped working state for human/agent work,
  loaded context, observations, stale-context refresh, and auditable closure.
- **Agent observation loop** — tool/model/agent outputs are captured as evidence
  and pending Claims before promotion into durable memory.
- **Governed connector execution** — connector runs can preserve raw payloads and
  attachments while recording governance and scope.
- **Multimodal asset spine** — raw assets, adapter runs, typed extraction
  projections, and adapter-derived Claims are stored under governed scope.
- **Wiki/export projection service** — markdown/wiki/HTML/API surfaces are treated
  as projections over engine state instead of the only source of truth.
- **Backend verification harness** — reality checks now cover topology, memory,
  retrieval/context, active pools, workflow/skills, governance, connectors,
  assets, evaluation, wiki/export, and compliance probes.

### Changed
- Public documentation now frames Optimal Engine as a backend operating engine
  for workspaces, agents, memory, retrieval, and projections rather than a
  desktop-first or wiki-only product.
- Projects are documented and modeled as one Node type inside a Workspace, not a
  peer of Workspace.
- Markdown is documented as a human/agent-operable projection and editing
  surface backed by database-owned runtime state.
- The old generic memory path is demoted behind the Memory Core lifecycle so
  durable truth flows through Source Package -> Claim -> Fact -> Memory Object.

### Fixed
- Workspace isolation leaks in graph/API/retrieval paths.
- Fact promotion lineage holes that could create source-less Facts.
- Stale Context Package refresh behavior.
- Public sample names and generated/local artifacts in public-facing docs.

## [0.2.0] — 2026-04-30

### Added
- **Multi-workspace isolation** — organizations contain workspaces; each workspace gets its own filesystem, wiki, config, and scoped queries
- **Memory primitive** — versioned memories with 5 typed relations (updates / extends / derives / contradicts / cites), soft-forget with reason, content-hash dedup, citation_uri on every memory
- **Proactive surfacing** — background Surfacer GenServer with 14-category subscription model, SSE push channel, webhook callbacks
- **Memory ↔ Wiki bridge** — bidirectional integration: memories promote to wiki, wiki claims extract as memories
- **API authentication** — bearer-token API keys with bcrypt-hashed secrets, scoped permissions
- **Rate limiting** — ETS-backed token bucket per API key/IP, X-RateLimit-* headers
- **Semantic grep** — `mix optimal.grep` + `GET /api/grep` with intent/scale/modality filters
- **4-tier workspace profile** — `GET /api/profile` returns static + dynamic + curated + activity in one call
- **5 typed recall endpoints** — `GET /api/recall/{actions,who,when,where,owns}` for cued memory recovery
- **Streaming RAG** — `GET /api/rag/stream` SSE endpoint streaming pipeline stages
- **Wiki contradiction detection** — 3 policies: flag_for_review / silent_resolve / reject
- **Per-workspace YAML config** — `.optimal/config.yaml` with visualizations, profile, grep, contradictions, memory, rate_limit sections
- **TypeScript SDK** — `@optimal-engine/client` with Vercel AI SDK + OpenAI Agents adapters
- **Python SDK** — `optimal-engine` with sync/async clients, LangChain + OpenAI Agents adapters
- **MCP server** — first-party stdio server with 9 tools for Claude Desktop / Cursor / Windsurf
- **Chrome browser extension** — MV3 web clipper with popup, options, context menu
- **Raycast extension** — 3 commands: search-memory, add-memory, ask-engine
- **Claude Code Skill** — `skills/optimal-engine/` with SKILL.md + 6 references + bootstrap.sh
- **Docs site** — SvelteKit static docs at `apps/docs/` with 21 pages
- **Docker deployment** — `deploy/` with multi-stage Dockerfiles + compose + prod overlay + auto-TLS
- **Desktop UI** — Foundation tokens, Org/Workspace switcher, 12 routes including /memory, /surface, /timeline, /heatmap
- **Marketing site** — landing with onion visualization, capabilities matrix, memory-failure section, on-disk convention
- **Schema migrations 026–030** — workspaces, surfacing, memories, content_hash, api_keys

### Changed
- Module rename: `OptimalEngine.Workspace` (nodes/skills) → `OptimalEngine.Topology`
- Module rename: `OptimalEngine.Topology` (YAML routing) → `OptimalEngine.Routing`
- `OptimalEngine.Workspace` now refers to the knowledge-base layer (multi-workspace)
- CI workflow optimized: paths-ignore for non-Elixir files, concurrency groups, single matrix entry

### Fixed
- FTS search workspace_id scoping (rows without workspace_id in SELECT caused MatchError)
- Grep SQL placeholder numbering bug (?1 then ?3 skipping ?2)

## [0.1.0] — 2026-04-18

### Added
- Initial public import: pipeline, memory, connector, wiki, API, and workspace
  foundations.
