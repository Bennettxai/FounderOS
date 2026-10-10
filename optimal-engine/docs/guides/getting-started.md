# Getting Started

This guide takes a new user from clone to a working Optimal Engine workspace.

The first workflow is intentionally simple:

```text
install
  -> run reality check
  -> initiate or set up workspace
  -> inspect topology
  -> ingest/search/render
  -> use the workspace from CLI, agent, app, or markdown
```

## 1. Prerequisites

Required:

- Erlang/OTP 26+
- Elixir `~> 1.17`
- Git
- C toolchain for the SQLite NIF
- Snappy for the RocksDB knowledge graph backend

Optional:

- Node 20+ for app/site/extension surfaces
- Ollama for local embeddings and generation
- Docker for packaged service deployment
- RocksDB runtime if you want the optional RocksDB graph/knowledge backend
- Optional multimodal tools such as document parsers, OCR, transcription,
  video, vision, or embedding adapters

RocksDB is not required for the default local engine. SQLite is the local
canonical runtime store today.

## 2. Clone And Compile

```bash
git clone https://github.com/Miosa-osa/OptimalEngine.git
cd OptimalEngine
brew install snappy
make install
```

## 3. Start The Local Engine

```bash
make bootstrap
make dev
```

`make dev` starts the HTTP engine on `http://localhost:4200`.
It creates `.optimal/connector_key` if `CONNECTOR_KEY` is not already set.
That key and the whole `.optimal/` runtime directory are local-only and ignored by git.

Verify the server:

```bash
curl http://localhost:4200/api/health
curl http://localhost:4200/api/stores
curl http://localhost:4200/api/stores/audit
```

`/api/health` proves the process and canonical store are reachable.
`/api/stores` inventories the 12 logical stores.
`/api/stores/audit` performs the deeper integrity, isolation, index, vector,
asset, backup, RLM, and cache checks and returns HTTP 503 on failure.

## 4. Run The Reality Check

```bash
mix optimal.reality_check
```

Expected current result:

```text
126 probes, 126 ok, 0 warn, 0 fail
```

This checks the runtime spine: store, topology, source evidence, memory,
retrieval, pools, workflows, tools, connectors, evaluation, wiki, compliance,
and retrieval edge cases.

## 5. Understand Signals Before Dumping Data

Optimal Engine does not treat every input as generic text. It classifies input
as a Signal:

```text
Signal = Mode + Genre + Type + Format + Structure
```

That breakdown tells the engine how to route, parse, review, package, and
retrieve the input. Read [Signal theory](../concepts/signal-theory.md) before
building serious workspaces.

## 6. Choose A Workspace Setup Path

Use initiation when the user starts with a messy dump:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
```

This creates the workspace, preserves the dump as evidence, applies
conservative Node candidates, writes markdown projections, and leaves detected
integration/tool surfaces disabled until the user scopes credentials and
permissions. For stricter environments, add `--review-only` and approve
individual topology requests later with `mix optimal.topology approve <id>
--workspace <workspace-id> --apply`.

If the user needs help creating that dump, use the starter prompts:

```text
templates/starter-prompts/workspace-initiation.md
templates/starter-prompts/company-wiki-import.md
templates/starter-prompts/package-inventory.md
templates/starter-prompts/agentic-loop-design.md
templates/starter-prompts/youtube-learning-import.md
templates/starter-prompts/interface-and-publishing-plan.md
```

Use setup when the user already knows the initial structure:

```bash
mix optimal.setup my-workspace --name "My Workspace"
```

Add explicit Nodes:

```bash
mix optimal.setup my-workspace \
  --node project:launch-plan:"Launch Plan" \
  --node person:founder:"Founder" \
  --node operational:weekly-review:"Weekly Review"
```

## 7. Understand What Was Created

The hierarchy is:

```text
Tenant / Organization
  -> Workspace
    -> Node graph
```

Projects are Nodes inside a Workspace:

```text
Workspace
  -> Project Node
  -> Person Node
  -> Product Node
  -> Operational Node
  -> Context Node
  -> Learning Node
```

Inspect topology:

```bash
mix optimal.topology --workspace default:my-workspace
```

## 8. Know Where Data Lives

Local default runtime state:

```text
.optimal/index.db
.optimal/cache/
.optimal/connector_key
```

These files are created locally and ignored by git.
They are not part of the repository and should not be copied into public setup docs, commits, Source Packages, Context Packages, or markdown workspaces.

Workspace projection files:

```text
my-workspace/
  AGENTS.md
  rhythm/
  nodes/
    first-project/
      context.md
      signal.md
      packages/
```

Important distinction:

```text
SQLite/Postgres = canonical runtime state
raw artifacts   = preserved source evidence
indexes/caches  = rebuildable acceleration
markdown/wiki   = projection and editing surface
```

See [Storage and projection map](../architecture/STORAGE-AND-PROJECTION-MAP.md)
for the full storage map.

For local, Docker, team, enterprise, and multimodal setup options, read
[Installation and deployment](installation-and-deployment.md).

## 9. Ingest Or Search

Ingest a quick text Signal:

```bash
mix optimal.ingest "Customer asked about pricing and wants a follow-up" --genre note
```

Search:

```bash
mix optimal.search "pricing"
```

Ask:

```bash
mix optimal.rag "what changed this week?"
```

## 10. Render Human-Facing Projections

Render a workspace tree:

```bash
mix optimal.wiki render-tree --workspace default:my-workspace
```

Render a Node page:

```bash
mix optimal.wiki render-node first-project --workspace default:my-workspace
```

Check a page:

```bash
mix optimal.wiki check node-first-project --workspace default:my-workspace
```

## 10. Use With Agents

For Codex, Claude Code, MCP clients, scripts, or app agents, use this sequence:

```text
inspect topology
  -> retrieve governed context
  -> work inside task scope
  -> use registered tools
  -> record observations
  -> create pending Claims
  -> let review/policy promote Facts
  -> render projections from engine state
```

Read [Agent and CLI SOP](agent-cli-sop.md) before wiring agents to tools,
connectors, scripts, or APIs.

## 11. What To Read Next

- [Engine structure](../architecture/ENGINE-STRUCTURE.md)
- [Storage and projection map](../architecture/STORAGE-AND-PROJECTION-MAP.md)
- [Installation and deployment](installation-and-deployment.md)
- [First workspace story](first-workspace-story.md)
- [Signal theory](../concepts/signal-theory.md)
- [Integrations and imports](integrations-and-imports.md)
- [Interfaces and publishing](interfaces-and-publishing.md)
- [Tool surfaces and loops](tool-surfaces-and-loops.md)
- [Agentic loops](agentic-loops.md)
- [Packages and exports](packages-and-exports.md)
- [Agent and CLI SOP](agent-cli-sop.md)
- [Mix tasks](mix-tasks.md)
- [Build goal alignment](../reference/build-goal-alignment.md)
