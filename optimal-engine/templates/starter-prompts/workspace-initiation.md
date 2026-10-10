# Workspace Initiation Prompt

Use this when I am starting an Optimal Engine workspace from messy context.

Do not assume the structure is obvious. Preserve what I give you first, then
propose structure for review.

## Goal

Create a setup dump and initiation plan for an Optimal Engine workspace.

The system should become a self-updating operating workspace: it organizes my
organizations, workspaces, Nodes, source evidence, Signals, Claims, Facts,
memories, integrations, workflows, packages, and exports.

## What I Am Providing

I may provide any mix of:

- markdown files
- company wiki exports
- folders
- docs
- meeting notes
- transcripts
- project lists
- CRM exports
- package examples
- SOPs
- proposals
- contracts
- API/MCP/tool lists
- command-line tools or scripts I already use
- scheduled jobs or reminders I want
- other agents or A2A Agent Cards I already use
- plain messy explanation

## Your Job

1. Identify the likely organization or tenant.
2. Identify one or more Workspaces.
3. Identify candidate Nodes inside each Workspace.
4. Identify Node types and possible aliases.
5. Identify important people, teams, companies, products, projects, operations,
   clients, partners, and context areas.
6. Identify communication channels and outside systems where context lives.
7. Identify recurring packages the organization sends to people.
8. Identify useful CLI tools, MCP servers, APIs, scripts, connectors, A2A
   agents, or scheduled loops.
9. Identify open questions where you are not sure.
10. Produce a setup dump that can be passed to:

```bash
mix optimal.initiate <workspace-slug> --name "<Workspace Name>" --dump setup.md
```

## Rules

- Do not create Facts directly from my dump.
- Treat my dump as Source Package material.
- Treat your extraction as pending Claims or proposed topology.
- Projects live inside Workspaces as Nodes.
- Packages belong under the owning Node unless they intentionally span multiple
  Nodes.
- If a name is ambiguous, list it as an open question.
- If an integration might exist, register it as proposed or disabled until I
  confirm access.
- CLI, MCP, API, script, connector, and scheduled-job surfaces are all allowed;
  choose based on the task and governance needs.
- A2A is for agent-to-agent delegation and coordination, not ordinary
  databases, files, calendars, repos, or SaaS APIs.
- Do not put credentials, API keys, secrets, or private tokens in markdown.

## Output

Return:

1. A short explanation of the proposed Workspace.
2. Candidate Nodes and Node types.
3. Candidate integrations and connection type.
4. Candidate scheduled loops or reminders.
5. Package inventory.
6. Open questions.
7. A clean `setup.md` draft.
8. The command I should run.
