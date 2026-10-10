# Interfaces And Publishing

Optimal Engine is the backend runtime. It does not need to own every frontend.
People should be able to use their own app, dashboard, CLI, MCP client, agent
runner, static site, or deployment tool on top of the same governed workspace
state.

The rule is:

```text
Optimal Engine owns canonical workspace state.
Interfaces display, edit, package, or publish projections.
External deploy tools ship those projections.
```

An app, public link, HTML page, dashboard, MCP client, or deployment portal
should not create a second memory system. It should call the engine, receive
scoped context or projection data, and write changes back through the correct
engine lifecycle.

## Three Interface Jobs

Most external surfaces do one of three jobs.

| Job | Examples | Engine path |
| --- | --- | --- |
| Control surface | CLI, Codex, Claude Code, Raycast, admin dashboard, workflow runner. | Commands/API/MCP call engine services and record observations or topology changes. |
| Display surface | dashboard, node page, graph, report, client portal, app view. | Read Context Packages, wiki/export projections, topology, memory, status, and package records. |
| Publishing surface | static HTML, public link, zip package, PDF/report bundle, share page. | Render export/package from governed state, record revision/delivery, publish through deployment tool. |

The same workspace can use all three.

```text
Human/agent controls work through CLI or app
  -> engine updates governed state
  -> app displays current scoped state
  -> export/publishing surface creates a shareable projection
```

## Standard Build Flow

When someone wants to build an app or interface on top of Optimal Engine, use
this order.

```text
1. Choose organization and workspace.
2. Choose the Nodes the interface is allowed to show or edit.
3. Mint a scoped API key for the app, MCP server, script, or deploy process.
4. Start the API runtime.
5. Retrieve topology, context, memory, packages, and projections through API/MCP.
6. Render the app/dashboard/site from those projections.
7. Send edits or new input back as Source Packages, observations, or topology changes.
8. Record package/export revisions when publishing externally.
```

Do not start with a custom app database unless the app needs its own unrelated
state. The workspace state should come from the engine.

## Local CLI Setup

For local work, the `mix optimal.*` tasks and `bin/optimal` wrapper are the fastest control surface:

```bash
mix deps.get
mix compile
mix optimal.reality_check
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
mix optimal.topology --workspace default:my-workspace
mix optimal.wiki render-tree --workspace default:my-workspace
```

Use the wrapper when an agent or script should use one command name:

```bash
bin/optimal reality-check
bin/optimal setup my-workspace --name "My Workspace"
bin/optimal wiki render-tree --workspace default:my-workspace
```

Local CLI commands are trusted local access to the configured store. They are
best for humans, coding agents, local scripts, and local cron jobs running on
the same machine.

## Remote App/API Setup

Use the API when an external app, web dashboard, service, MCP server, remote
agent, or deploy tool needs to connect.

Mint a scoped key:

```bash
mix optimal.auth mint --name "Workspace Dashboard" --workspace default:my-workspace
mix optimal.auth env --name "Workspace Dashboard" --workspace default:my-workspace
```

Start the API:

```bash
mix optimal.api
```

Then the app calls the engine:

```bash
curl http://localhost:4200/api/workspaces \
  -H "Authorization: Bearer $OPTIMAL_ENGINE_API_KEY"
```

Example app reads:

```text
GET  /api/workspaces
GET  /api/workspaces/:id
GET  /api/optimal/nodes
GET  /api/search?q=...
POST /api/rag
GET  /api/wiki
GET  /api/wiki/:slug
GET  /api/memory-core/claims
GET  /api/memory-core/active-pools/:id
```

Example app writes:

```text
POST  /api/memory
POST  /api/memory/remember
POST  /api/memory-core/active-pools
POST  /api/memory-core/active-pools/:id/observations
POST  /api/memory-core/claims/:id/promote
PATCH /api/workspaces/:id/config
```

Writes must preserve lifecycle rules. An app should submit new evidence,
observations, or review actions. It should not create direct Facts unless it is
calling the explicit review/promotion path.

## MCP Setup

Use MCP when an AI client should have structured tools for Optimal Engine.

```bash
cd apps/mcp
npm install
npm run build
```

The MCP server is a tool surface over the HTTP API. It needs:

```text
engine URL
scoped API key
default workspace
```

MCP is best when the agent needs authenticated, schema-described access to
workspace, memory, retrieval, wiki/export, and context tools. CLI is still
better for ordinary local commands such as `rg`, `git`, `cat`, `ffmpeg`, or
project-specific scripts.

## Publishing A Public Link

Publishing is not the same as storing truth.

```text
engine state
  -> render projection
  -> record export/package revision
  -> deploy static/public artifact
  -> optionally re-ingest sent artifact as Source Package evidence
```

Use this when the user wants:

```text
public workspace page
client portal view
HTML report
project status page
proposal package
handoff bundle
documentation site
knowledge snapshot
```

The export should declare:

```text
organization / workspace
owning Node or cross-node scope
receiver or audience
source objects
visibility policy
expiration or review cadence
generated_by
review_status
delivery target
```

## MIOSA CLI As A Publishing Surface

The public MIOSA CLI is a deployment/host/sandbox control surface. Its README
describes installing with:

```bash
npm install -g @miosa/cli
```

The relevant commands for Optimal Engine interface work are:

```text
miosa login
miosa context save/use/ls
miosa command-overview --json
miosa capabilities --json
miosa deploy --docker-deploy
miosa deploy logs
miosa deploy env set/list
miosa deploy domain add
miosa deploy redeploy
miosa deploy destroy
miosa releases / rollback
miosa sandbox create --template nextjs --publish-port 3000 --json
miosa sandbox exec ... --json
miosa sandbox publish ... --docker-deploy --json
miosa tunnel open <host> --port <n>
```

For Optimal Engine, treat MIOSA CLI as a publishing and remote execution layer:

```text
Optimal Engine renders/records the projection.
MIOSA CLI deploys the artifact and returns a URL.
Optimal Engine records the URL, revision, receiver, and delivery audit.
```

The deployment tool should not own the canonical memory. It owns the hosting or
delivery step.

### MIOSA Deployment Flow

```text
workspace / Node state in Optimal Engine
  -> app or static export reads scoped API/context
  -> build artifact or package generated
  -> miosa deploy --docker-deploy
  -> public URL / custom domain / release metadata
  -> export record updated with URL, revision, receiver, policy, and audit
```

For an agent, prefer MIOSA JSON output:

```bash
miosa command-overview --json
miosa capabilities --json
miosa deploy --docker-deploy --json
miosa deploy logs --deployment <id> --lines 200 --json
```

The agent should capture the deploy result as export metadata, not as a Fact
about the workspace unless a human/policy review accepts it.

## App Builder Contract

When building a custom app on top of Optimal Engine, the app should implement
this contract:

```text
Scope selector
  -> organization, workspace, Node, pool

Read path
  -> topology, Context Packages, wiki/export projections, packages, claims,
     memories, active pools, workflow state

Write path
  -> Source Package, observation, pending Claim, topology change request,
     package/export revision, review decision

Permission path
  -> scoped API key, membership, grants, workspace filter, audit
```

The app should not:

```text
create its own separate Node model
promote generated text directly to Fact
store API keys in markdown
publish private data without an export policy
turn package output into truth unless re-ingested as evidence
hide agent actions outside audit
```

## Interface Setup Questions

During workspace initiation or app planning, ask:

```text
Which app or interface do you want?
Who is the receiver: internal user, client, partner, public visitor, agent?
Is it read-only or can it write?
Which organization/workspace/Nodes can it access?
What should be public, private, or link-only?
Should it use CLI, API, MCP, connector, script, scheduler, or deployment tool?
Does it need a public URL?
Where should generated files live?
Should generated output be recorded as an export, package, or Source Package?
What actions require human review?
```

If any answer is unclear, keep the interface in draft and create open questions.

## Safe End-To-End Example

```text
User wants a client-facing project page
  -> choose Client or Project Node
  -> retrieve current Facts, decisions, risks, milestones, package records
  -> render private HTML preview
  -> human reviews sensitive content
  -> record export revision
  -> deploy with external CLI or static host
  -> record public URL and delivery metadata
  -> future edits re-render a new revision
```

This keeps the workflow flexible while preserving the core architecture:

```text
Backend runtime is canonical.
Interfaces are surfaces.
Published links are projections.
Sent packages can become evidence after delivery.
```
