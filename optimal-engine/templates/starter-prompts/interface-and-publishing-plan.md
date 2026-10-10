# Interface And Publishing Plan Prompt

Use this when I want to build an app, dashboard, client portal, public page,
HTML report, static site, package delivery flow, or deployment surface on top of
Optimal Engine.

## Goal

Design an interface or publishing flow that uses Optimal Engine as the governed
backend runtime instead of creating a separate memory system.

## What I May Provide

I may provide:

- a workspace or Node
- screenshots or sketches
- a desired app/dashboard/page
- a public-link or deployment goal
- a package type such as proposal, contract, SOP, report, or handoff bundle
- a MIOSA CLI or other deployment tool
- an existing app I want to connect
- API/MCP/CLI constraints
- privacy rules or receiver/audience constraints

## Your Job

1. Identify the organization, workspace, Node, and receiver/audience.
2. Decide whether this is a control surface, display surface, publishing
   surface, or a mix.
3. Identify the read path: topology, Context Packages, wiki/export projections,
   packages, claims, memories, active pools, workflows, and audit records.
4. Identify the write path: Source Package, observation, pending Claim,
   topology change request, package/export revision, or review decision.
5. Decide whether this should use CLI, API, MCP, connector, script, scheduler,
   deployment tool, or A2A agent.
6. Define the API key or local-auth model.
7. Define what should be public, private, internal, or link-only.
8. Define where generated files should live in the workspace projection.
9. Define what must be reviewed before publishing or sending.
10. Produce an implementation plan and verification checklist.

## Rules

- Do not make a second source of truth.
- Do not let the app invent a separate Node/project model.
- Do not publish private workspace data without an explicit audience and policy.
- Do not store secrets in markdown, package manifests, Source Packages, or
  generated exports.
- Do not turn generated pages/packages into Facts unless they re-enter as
  Source Package evidence and pass review.
- If an external deployment CLI creates a public URL, record the URL as export
  or delivery metadata. The deploy tool does not own memory.
- If using MIOSA CLI, prefer JSON-capable commands such as
  `miosa capabilities --json`, `miosa command-overview --json`,
  `miosa deploy --docker-deploy --json`, `miosa sandbox publish ... --json`,
  and `miosa deploy logs ... --json`.
- Ask before external sends, public deploys, purchases, destructive actions, or
  cross-organization access unless policy already permits them.

## Required Output

Return:

1. Interface summary.
2. Scope and audience.
3. Surface type: control, display, publishing, or mixed.
4. Read-path objects.
5. Write-path objects.
6. CLI/API/MCP/deployment plan.
7. Auth and permission plan.
8. File/export/package placement.
9. Review gates.
10. Build checklist.
11. Verification commands.
12. Open questions.
