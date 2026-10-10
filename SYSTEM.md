# BusinessOS System Model

This file defines the fundamentals of BusinessOS.
It is the product constitution.
`AGENTS.md` defines how coding agents, Hermes agents, browser agents, MCP agents, workflow agents, and other agents should operate inside this system.

If `SYSTEM.md`, `AGENTS.md`, and `CLAUDE.md` conflict, `SYSTEM.md` wins on product meaning.
`AGENTS.md` wins on agent procedure.

## What BusinessOS Is

BusinessOS is a desktop-style operating system for running a business.
It is not just a web dashboard.
It is the workspace where a team manages company context, modules, apps, mini-apps, sites, packages, relationships, tasks, pipelines, files, decisions, agents, and execution.

BusinessOS is both:

- A local desktop working environment.
- A shared workspace system that can sync, deploy, and expose selected work to a team or client.

## Core Layers

BusinessOS has two durable layers.

BusinessOS owns application state.
Examples include users, sessions, workspaces, memberships, modules, desktop icons, windows, projects, tasks, sites, clients, pipelines, settings, app records, URL records, and access controls.

OptimalEngine owns knowledge and memory.
Examples include source documents, workspace context, semantic search, memory, retrieval packages, claims, facts, and assembled context.

Do not collapse these layers.
BusinessOS may call OptimalEngine for context, memory, search, and assembled knowledge.
BusinessOS should store app state, workspace records, UI state, and operational records in BusinessOS.

## Workspaces

Each company or operating context should be a workspace.
Agency MIOSA gets its own workspace.
Client companies get their own workspaces.
Personal or platform work can have separate workspaces.

Modules and records are workspace-scoped unless explicitly global.
Team access is workspace-based.
If a teammate needs to see something, it must be saved, synced, committed, registered, or deployed.

Local-only work is not team-visible.
Shared work must exist in a shared workspace module record, shared Knowledge document, committed repo change, deployed URL, or shared backend record.

## Canonical Modules

Canonical modules are the top-level operating surfaces of BusinessOS.
They are the core vocabulary of the business.
They are protected primitives.

Current canonical module map:

- Operate: Command, Agents, Knowledge, Intelligence, Inbox, Calendar.
- Business: Relationships, Projects, Tasks, Rhythm, Pipelines, Offers.
- Growth: Campaigns, Sites, Personas, Content.
- Build: Apps, Assets, Deliverables, Engines, Builders, Drive.
- Manage: Finance, Analytics, Data, Team, Connectors, Admin, Resources, Notifications, Help.

Canonical modules may appear in:

- Sidebar navigation.
- Command dashboard cards.
- Desktop icons.
- Dock or window launchers.
- Window route mappings.
- Module defaults.

Do not treat a mini-app, site, proposal, package, sandbox, or one-off tool as a canonical module.
The sidebar is for canonical modules only.

## Artifacts

Artifacts are things BusinessOS manages.
Artifacts are not canonical modules.

Artifact types include:

- Apps.
- Mini-apps.
- Sites.
- Packages.
- Proposals.
- Client portals.
- Landing pages.
- Sandboxes.
- Stable embeds.
- Durable deployments.
- Custom-domain apps.
- Downloadable files.

Ownership rules:

- Apps manages mini-apps, internal apps, public apps, app inventory, and app launch surfaces.
- Sites manages brand sites, landing pages, funnels, public pages, stable embeds, portals, deployment URLs, and custom domains.
- Deliverables manages proposals, client packages, PDFs, downloadable bundles, reply docs, and handoff documents.
- Campaigns manages campaigns, ad campaigns, launch motions, and growth pushes.
- Content manages scripts, posts, drafts, content calendars, and publishing strategy.
- Builders manages reusable build templates and creation tools.
- Engines manages sequenced workflows, multi-agent workflows, automations, and runtime capabilities.

Example:

Robert's alignment package is not a module.
It is a durable deployment that belongs in Sites and can be referenced by Deliverables.

## Desktop Model

The desktop can show canonical modules and artifacts.
A desktop icon is not the same thing as a sidebar module.

Canonical module icons open core operating surfaces.
Mini-app icons open specific tools or artifacts.
Site/package icons should still be registered in Sites or Deliverables if the team needs to find them later.

## Window Model

BusinessOS surfaces open in windows.
Module and mini-app windows must work in compact desktop contexts.
Routes loaded in windows should support `?embed=true`.

Full-browser layouts and desktop-window layouts are not the same thing.

## URL Model

BusinessOS tracks URL class and lifecycle.
Do not store naked URLs without meaning.

URL classes:

- `temporary_preview`
- `always_on_preview`
- `stable_sandbox_embed`
- `durable_deployment`
- `custom_domain`

Rules:

- Sandbox previews are for quick review or active collaboration.
- Raw sandbox previews are not final handoff links.
- Stable sandbox aliases are for embed or use-before-publish.
- Durable deployments are the normal client-facing handoff.
- Custom domains are final branded production.
- `miosa.app` is for published deployed apps.
- Sandbox previews must not default to `miosa.app`.

Every URL-producing record should carry:

```json
{
  "url": "https://3000-abc123.sandbox.miosa.ai",
  "class": "temporary_preview",
  "stable_for_embedding": false,
  "lifecycle_state": "running",
  "recommended_next_action": "create_alias_or_publish"
}
```

## Foundation Versus Current Work

Company foundation is durable operating identity.
Use foundation docs for positioning, offer architecture, target client, roles, responsibilities, core language, and stable company model.

Current work is temporary operating state.
Use Command, Rhythm, Tasks, Projects, Inbox, Pipelines, and partner sync records for current focus, weekly plans, open decisions, blockers, and next actions.

Do not bury durable foundation changes in weekly notes.
Do not rewrite durable foundation for a temporary weekly plan.

## Privacy Boundary

The public/open-source BusinessOS base must not contain private Agency MIOSA data.
Private company context belongs in private workspace data, private markdown, private backend records, environment configuration, or OptimalEngine memory.

## The Fundamental Rule

Modules are the operating system.
Apps, mini-apps, sites, packages, and deployments are things the operating system manages.
Do not confuse the two.
