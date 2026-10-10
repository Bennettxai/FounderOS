# Agent And CLI SOP

This guide defines how humans, coding agents, MCP clients, A2A agents, scripts,
and apps should use Optimal Engine.

The rule is simple:

```text
Agents and tools use engine interfaces.
They do not invent their own memory system.
They do not write final truth directly.
```

## Human Setup Flow

When a person starts from messy context, use:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
```

This does not blindly create truth. It:

```text
preserves the dump as a Source Package
  -> creates an unreviewed setup Claim
  -> applies conservative Node candidates by default
  -> proposes integration surfaces
  -> keeps integration/tool surfaces disabled until scoped
  -> writes pending topology change requests only with --review-only
```

When the person already knows the workspace structure, use:

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

Inspect the result:

```bash
mix optimal.topology --workspace default:my-workspace
```

## Agent Session Flow

An agent should start every session by understanding the workspace, then loading
governed context.

```text
1. Identify tenant/workspace/task.
2. Inspect topology.
3. Retrieve or assemble governed context.
4. Work inside the task scope.
5. Use registered tools only.
6. Record observations.
7. Create pending Claims for useful new knowledge.
8. Let review/policy promote Claims to Facts.
9. Export/render projections from engine state.
```

Minimum CLI pattern:

```bash
mix optimal.topology --workspace default:my-workspace
mix optimal.search "current project state"
mix optimal.rag "what changed this week?"
mix optimal.wiki render-tree --workspace default:my-workspace
mix optimal.reality_check
```

There are two kinds of CLI use:

```text
engine CLI
  -> mix optimal.* commands that read/write Optimal Engine state

external CLI
  -> normal terminal tools such as ls, cat, rg, grep, curl, git, docker,
     ffmpeg, or provider-specific CLIs
```

External CLI output is not automatically memory. If it contains useful evidence,
preserve it as a Source Package or observation, then create pending Claims where
appropriate.

## What The Agent May Write

| Write target | Allowed? | Correct path |
| --- | --- | --- |
| Source Package | Yes | preserve raw input, tool output, file, attachment, or observation |
| Signal/compatibility row | Yes through pipeline | classify/index input for search |
| Claim | Yes as pending/unreviewed | extract what a source appears to say |
| Fact | Only through review/policy | never direct-write as an agent guess |
| Memory Object | Through Memory Core | link to accepted Facts and evidence |
| Node | Through topology setup/change review | do not silently rewrite workspace shape |
| Markdown/wiki file | Yes as projection or draft | re-ingest edits as evidence when they change knowledge |
| Tool call result | Through governance | register, validate, execute, log, preserve |

## Integration Setup

Users may have many integration types:

```text
MCP servers
CLI tools
connector syncs
custom APIs
scripts
cron jobs
local files
third-party apps
model calls
A2A agents
```

The agent should ask:

```text
Where is this data stored?
Who is allowed to read it?
Who is allowed to write it?
Which workspace/nodes may it affect?
What credentials or scopes are required?
What actions require confirmation?
Should output become evidence, observation, or just a transient result?
Are any remote agents available through A2A Agent Cards?
```

Then the integration should be registered as disabled/draft until confirmed.

```text
integration mention
  -> proposed tool/connector definition
  -> disabled by default
  -> credential/scope review
  -> grants and partition policy
  -> governed execution
```

## Data Dump Intake

When the user talks naturally or pastes a large messy dump, the agent should not
try to force perfect structure immediately.

Correct pattern:

```text
preserve the dump
  -> extract obvious candidate structure
  -> ask missing questions
  -> propose Nodes and relationships
  -> propose integration surfaces
  -> keep proposals pending
  -> review before durable topology
```

This is how the engine handles noisy human language without corrupting the
workspace.

## Local CLI Auth Model

The local CLI is a trusted local control surface.

```text
human/agent on the same machine
  -> runs mix optimal.* or the local bin/optimal wrapper
  -> reads/writes the local configured store
  -> records actor/service metadata where the command supports it
```

That path does not require an API key because it is equivalent to a developer
or local agent running commands on the machine that owns the workspace.

Use API keys when something connects over HTTP/API, MCP, a remote script, an
external app, or a remote agent:

```text
external app / MCP server / remote agent / automation
  -> gets scoped API key
  -> sends Authorization: Bearer <key> or X-API-Key
  -> API verifies tenant, principal, scopes, and workspace scope
  -> backend applies lifecycle and audit rules
```

Create a key locally:

```bash
mix optimal.auth mint --name "Business OS" --workspace default:my-workspace
mix optimal.auth env --name "Local Agent" --workspace default:my-workspace
mix optimal.auth list
mix optimal.auth revoke <key-id>
```

The same commands can be run through the checked-in wrapper:

```bash
bin/optimal auth mint --name "Business OS" --workspace default:my-workspace
bin/optimal auth env --name "Local Agent" --workspace default:my-workspace
```

Do not put raw API keys in markdown, Source Packages, package manifests, or
Context Packages. Store them in shell environment, a local secret store, or the
deployment secret manager.

## Retrieval Pattern

Agents should request context from the engine, not scrape random files first.

```text
query/task
  -> actor identity
  -> workspace/node scope
  -> authorization envelope
  -> retrieval coordinator
  -> Context Package
  -> answer/action with source links
```

Use direct file reads only when the task explicitly concerns a file projection,
docs edit, or source inspection.

## Active Work Pattern

For meaningful tasks, use an Active Memory Pool conceptually even if the UI is
not built yet:

```text
task scope
  -> loaded Context Package
  -> humans/agents/tools involved
  -> observations
  -> pending Claims
  -> audit trail
  -> close/archive
```

The agent should report which context it used and what new observations should
be promoted.

## Tool Call Pattern

All tool calls follow the same governance shape:

```text
tool request
  -> registered tool definition
  -> privilege and partition check
  -> input schema validation
  -> execution
  -> output schema validation
  -> tool call run record
  -> Source Package or observation if useful
  -> pending Claim if knowledge-bearing
```

This applies whether the tool is MCP, HTTP API, local script, CLI command, or
scheduled job.

## A2A Pattern

A2A is for agent-to-agent collaboration. Use it when another agent is the right
worker, reviewer, negotiator, or external collaborator.

```text
agent task
  -> retrieve Context Package
  -> identify remote agent and Agent Card
  -> check workspace, actor, and delegation policy
  -> send A2A task request
  -> receive response, progress updates, or artifacts
  -> preserve useful artifacts and observations
  -> create pending Claims when knowledge-bearing
  -> audit delegation and returned work
```

Do not use A2A for ordinary files, databases, calendars, repos, or APIs. Those
are tool/data surfaces and should use CLI, MCP, connectors, APIs, or scripts.

## CLI Vs MCP Decision

Use CLI when the command directly maps to the job and the output can be captured
cleanly:

```text
local file inspection
text search
Git status/history
Docker inspection
curl against a simple endpoint
media processing
local scripts
```

Use MCP/API/connectors when the abstraction or governance is worth it:

```text
authenticated SaaS systems
per-user access control
schema-validated writes
browser-rendered pages
calendar/email/CRM actions
audit-sensitive operations
provider-specific pagination or IDs
```

Use A2A when the other side is another agent:

```text
delegate a specialist review
coordinate with a partner/supplier agent
request an external agent's artifact
stream progress from a long-running agent task
```

If an agent starts reverse-engineering a complex app response through raw CLI
commands when a rendered/browser/API tool exists, it probably chose the wrong
surface. If it loads a huge tool schema just to run a simple file search, it
probably chose the wrong surface.

## Scheduled Loop Pattern

Scheduled loops are allowed, but they must be explicit.

```text
schedule
  -> actor/service account
  -> workspace/node scope
  -> tool/connector/API/CLI command
  -> run policy
  -> captured output
  -> Source Package, observation, pending Claim, export, or audit record
```

Examples:

```text
daily review
  -> assemble Context Package
  -> render focus export
  -> record observations

connector sync
  -> preserve new payloads/assets
  -> classify Signals
  -> create pending Claims

context refresh
  -> refresh stale Context Packages
  -> mark old packages refreshed

wiki refresh
  -> rebuild stale projected pages from governed state
```

Current engine surfaces include `mix optimal.context.refresh_stale`,
`OptimalEngine.MemoryCore.ContextRefreshScheduler`, `OptimalEngine.Wiki.Scheduler`,
connector governed runs, and tool/model governance records that can represent
delegated agent work.

## Projection Pattern

Projection surfaces display or control engine state:

```text
engine state -> markdown
engine state -> wiki
engine state -> HTML
engine state -> app/API
engine state -> report/package
engine state -> agent context
```

Packages are receiver/channel bundles. If a package belongs to one Node, write
it under:

```text
nodes/<node-slug>/packages/<package-slug>/
```

Only write a workspace-level package when it intentionally spans multiple Nodes
and the package manifest lists those source Nodes.

Edits flow back in:

```text
markdown edit -> Source Package or topology change request
app form save -> owning domain service
agent output -> observation or pending Claim
```

## Verification Pattern

Before presenting work as complete:

```bash
mix compile
mix optimal.reality_check
```

For workspace/wiki/setup changes:

```bash
mix test test/wiki/service_test.exs \
  test/wiki/store_test.exs \
  test/workspace_export_test.exs \
  test/mix_tasks/optimal_setup_test.exs \
  test/workspace_initiation_test.exs \
  test/mix_tasks/optimal_initiate_test.exs \
  --seed 0
```

## Forbidden Shortcuts

```text
Do not treat generated markdown as final truth.
Do not promote Claims to Facts without review/policy.
Do not let agents call arbitrary tools outside governance.
Do not store credentials in prompts, markdown, or context packages.
Do not use vector search as the final authority.
Do not let app UI create a separate data model.
Do not skip source evidence when storing memory.
```
