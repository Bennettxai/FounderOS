# Optimal Engine — Core Concepts

This file is the compact concept map for agents using Optimal Engine.

## Mental Model

```text
Tenant / Organization
  -> Workspaces
      -> Nodes
          -> Sources, Signals, Claims, Facts, Memories, Workflows, Skills
```

Projects are Nodes inside a Workspace. They are not peers of Workspace.

## Storage, Layers, And Surfaces

Use these terms precisely:

| Term | Meaning |
| --- | --- |
| Storage substrate | Where bytes or records physically live: SQLite, Postgres, raw artifact storage, indexes, caches, optional graph backends. |
| Domain layer | The owner of lifecycle and meaning: Topology, Intake, Signal, Memory Core, Retrieval, Active Pools, Workflow/Skill, Governance, Export. |
| Projection surface | How humans, agents, apps, or tools see/control state: markdown, wiki, HTML, API, MCP, app UI, package, report, public link. |

The database stores governed runtime state. Markdown/wiki/API/app views are
surfaces over that state.

```text
projection edit
  -> Source Package, observation, or topology change request
  -> review/policy
  -> governed state
  -> refreshed projection
```

## Tiers

Tiers are still useful, but they are not a substitute for layer ownership.

```text
Tier 1: preserved sources and raw artifacts
Tier 2: rebuildable indexes, summaries, chunks, embeddings, parser output
Tier 3: human-facing wiki/export pages and app views
```

Tier 3 is not canonical truth. It is useful because humans and agents can read
it quickly. Canonical truth lives in source-backed Memory Core state.

## Signal Classification

Every meaningful input becomes a Signal:

```text
Signal = Mode + Genre + Type + Format + Structure
```

| Dimension | Question | Examples |
| --- | --- | --- |
| Mode | How is it perceived? | linguistic, visual, code, data, mixed |
| Genre | What conventional form is this? | spec, transcript, proposal, report, note, SOP |
| Type | What does it do? | inform, decide, request, commit, instruct |
| Format | What container is it in? | markdown, JSON, PDF, image, audio, diff |
| Structure | What skeleton does it follow? | meeting notes, contract, proposal, checklist |

Classification is how the engine avoids treating every pasted message, file,
call transcript, proposal, or code artifact as the same kind of text.

## Current Data Flow

```text
Raw input
  -> Source Package
  -> Signal
  -> route to Workspace / Node
  -> pending Claim
  -> review / policy
  -> Fact
  -> Memory Object
  -> Context Package
  -> Active Memory Pool
  -> observation
  -> pending Claim
```

No adapter, model, parser, transcript, OCR output, generated summary, or agent
observation should become a Fact directly.

## Workspace Setup Flow

When the user starts messy:

```text
messy dump
  -> preserve as Source Package
  -> extract setup Claim
  -> propose Nodes, relationships, aliases, integrations, packages, loops
  -> apply only conservative structure or keep pending with --review-only
  -> keep tools/integrations disabled until scoped
```

Use:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
mix optimal.topology --workspace default:my-workspace
```

When the user already knows the structure:

```bash
mix optimal.setup my-workspace --name "My Workspace"
```

## Retrieval Flow

Agents should ask the engine for context instead of scraping random files first.

```text
query/task
  -> actor and workspace scope
  -> authorization envelope
  -> retrieval coordinator
  -> Context Package
  -> answer/action with source links
```

Wiki/export pages may help retrieval, but they are projection inputs, not the
truth source.

## Tool Surfaces

Optimal Engine works with many tool surfaces:

| Surface | Use when |
| --- | --- |
| CLI | Local commands directly solve the job: files, git, ffmpeg, tests, scripts. |
| MCP | Agent needs authenticated structured tools, schemas, remote resources, or safe SaaS writes. |
| API | Apps, dashboards, services, MCP servers, remote agents, or automation connect to the engine. |
| Connector | A known provider syncs data into Source Packages and assets. |
| Script/cron | A recurring local or server job runs on a schedule. |
| A2A | Another agent is the counterparty and can accept/delegate/stream tasks. |

All useful outputs return as Source Packages, observations, pending Claims,
package/export records, workflow traces, or audit events.

## Interfaces And Publishing

External apps and deployment tools are surfaces.

```text
Optimal Engine renders/records projection
  -> external app or deploy CLI publishes/displays it
  -> resulting URL/package/delivery is recorded
  -> sent artifact may re-enter as Source Package evidence
```

Do not let an app, static site, public link, or deployment tool create a second
memory system.

## Agent Rules

- Resolve organization, workspace, Node, and task scope first.
- Preserve evidence before structuring it.
- Create pending Claims, not direct Facts.
- Use explicit review/promotion paths.
- Register tools/integrations before using them for important work.
- Keep secrets out of markdown, packages, Source Packages, and Context Packages.
- Put packages under the owning Node unless they intentionally span multiple
  Nodes.
- Treat public links and generated HTML as projections with visibility policy.
